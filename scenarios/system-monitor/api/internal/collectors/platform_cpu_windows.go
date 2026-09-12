//go:build windows

package collectors

import (
	"context"
	"fmt"
	"time"
	"unsafe"

	"github.com/vrooli/vrooli/internal/hostinventory"
	"golang.org/x/sys/windows"
)

type windowsFiletime struct {
	LowDateTime  uint32
	HighDateTime uint32
}

var procGetSystemTimes = windows.NewLazySystemDLL("kernel32.dll").NewProc("GetSystemTimes")

func collectPlatformCPU(_ context.Context, c *CPUCollector, _ hostinventory.Snapshot) platformCPUReading {
	var idle, kernel, user windowsFiletime
	ret, _, callErr := procGetSystemTimes.Call(
		uintptr(unsafe.Pointer(&idle)),
		uintptr(unsafe.Pointer(&kernel)),
		uintptr(unsafe.Pointer(&user)),
	)
	if ret == 0 {
		return platformCPUReading{status: "failed", reason: fmt.Sprintf("GetSystemTimes: %v", callErr), provenance: "Windows GetSystemTimes"}
	}
	k := filetimeValue(kernel)
	u := filetimeValue(user)
	i := filetimeValue(idle)
	// GetSystemTimes reports kernel time including idle time; derive a
	// non-negative busy-system counter for a useful mode breakdown.
	system := uint64(0)
	if k >= i {
		system = k - i
	}
	counters := []uint64{u, system, i}
	usage, modes, measured := deltaCPUValues(&c.platformState, counters, time.Now(), []string{"user", "system", "idle"}, 2)
	if !measured {
		return platformCPUReading{status: "not_yet_sampled", reason: "CPU counter delta requires a previous valid sample", provenance: "Windows GetSystemTimes", values: map[string]interface{}{"mode_breakdown_status": "not_yet_sampled"}}
	}
	return platformCPUReading{usage: usage, status: "measured", provenance: "Windows GetSystemTimes", values: map[string]interface{}{"mode_breakdown": modes, "mode_breakdown_status": "measured", "load_average_status": "unsupported", "load_average_reason": "Windows load average has no native backend"}}
}

func filetimeValue(value windowsFiletime) uint64 {
	return uint64(value.HighDateTime)<<32 | uint64(value.LowDateTime)
}
