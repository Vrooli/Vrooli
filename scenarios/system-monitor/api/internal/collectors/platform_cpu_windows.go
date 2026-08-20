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
	counters := []uint64{filetimeValue(kernel) + filetimeValue(user), filetimeValue(idle)}
	usage, measured := deltaCPUUsage(&c.platformState, counters, time.Now(), 1)
	if !measured {
		return platformCPUReading{status: "failed", reason: "not_yet_sampled: CPU delta requires a previous sample", provenance: "Windows GetSystemTimes"}
	}
	return platformCPUReading{usage: usage, status: "measured", provenance: "Windows GetSystemTimes"}
}

func filetimeValue(value windowsFiletime) uint64 {
	return uint64(value.HighDateTime)<<32 | uint64(value.LowDateTime)
}
