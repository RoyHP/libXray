// HexVPN iOS tunnel bridge: ONE Go archive exporting both the WireGuardKit
// C API (api-apple.go, verbatim from wireguard-apple) and libXray's Invoke.
// Two Go runtimes cannot coexist in a process, so HexGuard (WireGuardKit)
// and HexGamma (Xray-core) share this one. Built by build/hexvpn_bridge.sh
// into libwg-go.a — the name WireGuardKit's module map links against.
package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"unsafe"

	libXray "github.com/xtls/libxray"
)

//export CGoInvoke
func CGoInvoke(requestJSON *C.char) *C.char {
	text := C.GoString(requestJSON)
	return C.CString(libXray.Invoke(text))
}

//export CGoFree
func CGoFree(value *C.char) {
	C.free(unsafe.Pointer(value))
}
