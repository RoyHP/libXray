// Package hexwg is HexVPN's Android bridge to wireguard-go: the HexGuard
// engine, bound next to libXray by gomobile so both engines share ONE Go
// runtime in the app process (two Go runtimes in one process is an
// unsupported configuration: the first loaded loses async preemption).
//
// It is wireguard-android's tunnel/tools/libwg-go/api-android.go without
// cgo exports: the same wireguard-go device over the VpnService's tun fd,
// exposed as plain functions gobind can bind. The Kotlin side
// (HexVpnService) establishes the tun, calls TurnOn, protects the UDP
// socket(s) it reports, and calls TurnOff on disconnect or engine switch.
package hexwg

// #cgo LDFLAGS: -llog
// #include <android/log.h>
// #include <stdlib.h>
import "C"

import (
	"fmt"
	"runtime/debug"
	"strings"
	"sync"
	"unsafe"

	"golang.org/x/sys/unix"
	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun"
)

const logTag = "HexWG"

type androidLogger struct {
	level C.int
}

func (l androidLogger) Printf(format string, args ...interface{}) {
	tag := C.CString(logTag)
	msg := C.CString(fmt.Sprintf(format, args...))
	C.__android_log_write(l.level, tag, msg)
	C.free(unsafe.Pointer(msg))
	C.free(unsafe.Pointer(tag))
}

var (
	handlesMu sync.Mutex
	handles   = map[int]*device.Device{}
	nextID    int
)

// TurnOn starts a WireGuard device on tunFd with the given UAPI settings
// (the "private_key=…\npublic_key=…" form, hex keys) and returns a handle
// (>= 0), or -1 after logging why. The device owns tunFd from here on:
// it is closed by TurnOff, or here on failure.
func TurnOn(tunFd int, settings string) int {
	logger := &device.Logger{
		Verbosef: androidLogger{level: C.ANDROID_LOG_DEBUG}.Printf,
		Errorf:   androidLogger{level: C.ANDROID_LOG_ERROR}.Printf,
	}
	tunDev, name, err := tun.CreateUnmonitoredTUNFromFD(tunFd)
	if err != nil {
		unix.Close(tunFd)
		logger.Errorf("CreateUnmonitoredTUNFromFD: %v", err)
		return -1
	}
	logger.Verbosef("Attaching to interface %v", name)
	dev := device.NewDevice(tunDev, conn.NewStdNetBind(), logger)
	if err := dev.IpcSet(settings); err != nil {
		logger.Errorf("IpcSet: %v", err)
		dev.Close()
		return -1
	}
	dev.DisableSomeRoamingForBrokenMobileSemantics()
	if err := dev.Up(); err != nil {
		logger.Errorf("Unable to bring up device: %v", err)
		dev.Close()
		return -1
	}
	logger.Verbosef("Device started")

	handlesMu.Lock()
	defer handlesMu.Unlock()
	id := nextID
	nextID++
	handles[id] = dev
	return id
}

// TurnOff stops the device behind handle and closes its tun fd. Unknown
// handles are ignored.
func TurnOff(handle int) {
	handlesMu.Lock()
	dev, ok := handles[handle]
	if ok {
		delete(handles, handle)
	}
	handlesMu.Unlock()
	if ok {
		dev.Close()
	}
}

func lookup(handle int) *device.Device {
	handlesMu.Lock()
	defer handlesMu.Unlock()
	return handles[handle]
}

// GetSocketV4 returns the fd of the device's IPv4 UDP socket, for
// VpnService.protect, or -1.
func GetSocketV4(handle int) int {
	dev := lookup(handle)
	if dev == nil {
		return -1
	}
	bind, _ := dev.Bind().(conn.PeekLookAtSocketFd)
	if bind == nil {
		return -1
	}
	fd, err := bind.PeekLookAtSocketFd4()
	if err != nil {
		return -1
	}
	return fd
}

// GetSocketV6 returns the fd of the device's IPv6 UDP socket, for
// VpnService.protect, or -1.
func GetSocketV6(handle int) int {
	dev := lookup(handle)
	if dev == nil {
		return -1
	}
	bind, _ := dev.Bind().(conn.PeekLookAtSocketFd)
	if bind == nil {
		return -1
	}
	fd, err := bind.PeekLookAtSocketFd6()
	if err != nil {
		return -1
	}
	return fd
}

// GetConfig returns the device's UAPI state (handshake times, transfer
// counters, endpoint), or "" for an unknown handle.
func GetConfig(handle int) string {
	dev := lookup(handle)
	if dev == nil {
		return ""
	}
	settings, err := dev.IpcGet()
	if err != nil {
		return ""
	}
	return settings
}

// Version is the wireguard-go revision compiled in.
func Version() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	for _, dep := range info.Deps {
		if dep.Path == "golang.zx2c4.com/wireguard" {
			parts := strings.Split(dep.Version, "-")
			if len(parts) == 3 && len(parts[2]) == 12 {
				return parts[2][:7]
			}
			return dep.Version
		}
	}
	return "unknown"
}
