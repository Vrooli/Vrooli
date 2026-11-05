//go:build darwin

package collectors

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/vrooli/vrooli/internal/hostinventory"
	"golang.org/x/sys/unix"
)

func collectPlatformMemory(context.Context, *MemoryCollector, hostinventory.Snapshot) platformMemoryReading {
	total, err := unix.SysctlUint64("hw.memsize")
	if err != nil {
		return platformMemoryReading{status: "failed", reason: fmt.Sprintf("hw.memsize: %v", err), provenance: "darwin sysctl hw.memsize"}
	}
	rawSwap, swapErr := unix.SysctlRaw("vm.swapusage")
	if swapErr != nil {
		return platformMemoryReading{status: "unsupported", reason: "vm.swapusage: " + swapErr.Error(), provenance: "darwin host_statistics64"}
	}
	swapTotal, swapFree, parseErr := darwinSwapUsage(rawSwap)
	if parseErr != nil {
		return platformMemoryReading{status: "failed", reason: parseErr.Error(), provenance: "darwin vm.swapusage"}
	}
	// The legacy BSD page MIB is not a macOS API. Until the host_statistics64
	// binding is available in the native package, explicitly
	// degrade instead of returning fabricated available-memory values.
	_ = total
	return platformMemoryReading{status: "unsupported", reason: "host_statistics64 memory pressure binding unavailable", provenance: "darwin host_statistics64", swap: map[string]interface{}{"total": swapTotal, "free": swapFree}}
}

func darwinSwapUsage(raw []byte) (total, free uint64, err error) {
	if len(raw) < 24 {
		return 0, 0, fmt.Errorf("vm.swapusage returned %d bytes", len(raw))
	}
	return binary.LittleEndian.Uint64(raw[0:8]), binary.LittleEndian.Uint64(raw[16:24]), nil
}
