//go:build ios

package memory

import (
	"runtime"
	"runtime/debug"
	"sync"
	"time"
)

// HexVPN policy (see hexvpn-api/docs/reality-vision-plan.md §0, §3.2).
//
// Upstream libXray ships GOGC=10 / SetMemoryLimit(30 MiB) / FreeOSMemory
// every second. Field data (BodoVPN, Oct 2026) shows that policy driving
// ~266 GC cycles/s under load and halving throughput. The relaxed settings
// below doubled throughput with the same headroom under the 50 MiB iOS
// packet-tunnel jetsam limit. SetMemoryLimit is a soft GC target, not an
// RSS cap — the extension logs os_proc_available_memory() to verify.
const (
	freeInterval = 2 * time.Second
	memoryLimit  = 36 << 20
	gcPercent    = 100
	maxProcs     = 2
)

var initForceFreeOnce sync.Once

func forceFree(interval time.Duration) {
	go func() {
		for {
			time.Sleep(interval)
			debug.FreeOSMemory()
		}
	}()
}

func InitForceFree() {
	initForceFreeOnce.Do(func() {
		runtime.GOMAXPROCS(maxProcs)
		debug.SetGCPercent(gcPercent)
		debug.SetMemoryLimit(memoryLimit)
		forceFree(freeInterval)
	})
}
