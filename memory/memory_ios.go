//go:build ios

package memory

import (
	"math"
	"runtime"
	"runtime/debug"
	"sync"
	"time"
)

// HexVPN policy (see hexvpn-api/docs/reality-vision-plan.md §0, §3.2),
// scoped to the time an Xray instance runs: this Go runtime is shared with
// wireguard-go (HexGuard), which must keep the default GOMAXPROCS / GC
// behaviour when it is the live engine.
//
// Upstream libXray ships GOGC=10 / SetMemoryLimit(30 MiB) / FreeOSMemory
// every second, forever. Field data (BodoVPN, Oct 2026) shows that policy
// driving ~266 GC cycles/s under load and halving throughput. The relaxed
// settings below doubled throughput with the same headroom under the 50 MiB
// iOS packet-tunnel jetsam limit. SetMemoryLimit is a soft GC target, not
// an RSS cap — the extension logs os_proc_available_memory() to verify.
const (
	freeInterval = 2 * time.Second
	memoryLimit  = 36 << 20
	gcPercent    = 100
	maxProcs     = 2
)

var (
	mu       sync.Mutex
	stopFree chan struct{}
	prevGC   int
	prevLim  int64
	prevProc int
)

// Apply installs the policy. Idempotent while applied.
func Apply() {
	mu.Lock()
	defer mu.Unlock()
	if stopFree != nil {
		return
	}
	prevProc = runtime.GOMAXPROCS(maxProcs)
	prevGC = debug.SetGCPercent(gcPercent)
	prevLim = debug.SetMemoryLimit(memoryLimit)
	stop := make(chan struct{})
	stopFree = stop
	go func() {
		ticker := time.NewTicker(freeInterval)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				debug.FreeOSMemory()
			}
		}
	}()
}

// Reset restores what Apply replaced (the runtime defaults for a process
// that never ran Xray). No-op if not applied.
func Reset() {
	mu.Lock()
	defer mu.Unlock()
	if stopFree == nil {
		return
	}
	close(stopFree)
	stopFree = nil
	if prevProc > 0 {
		runtime.GOMAXPROCS(prevProc)
	}
	debug.SetGCPercent(prevGC)
	if prevLim <= 0 {
		prevLim = math.MaxInt64
	}
	debug.SetMemoryLimit(prevLim)
	debug.FreeOSMemory()
}

// InitForceFree is the upstream name; kept so nothing else breaks.
func InitForceFree() { Apply() }
