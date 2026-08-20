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
	pageSize, err := unix.SysctlUint64("hw.pagesize")
	if err != nil || pageSize == 0 {
		pageSize = 4096
	}
	freePages, freeErr := darwinVMStatPages("vm.stats.vm.pages.free")
	inactivePages, inactiveErr := darwinVMStatPages("vm.stats.vm.pages.inactive")
	speculativePages, speculativeErr := darwinVMStatPages("vm.stats.vm.pages.speculative")
	if freeErr != nil || inactiveErr != nil || speculativeErr != nil {
		return platformMemoryReading{status: "failed", reason: "darwin host memory statistics unavailable", provenance: "darwin sysctl vm.stats.vm"}
	}
	available := (freePages + inactivePages + speculativePages) * pageSize
	return memoryReadingFromBytes(total, available, 0, inactivePages*pageSize, 0, 0, "darwin sysctl host memory statistics")
}

func darwinVMStatPages(name string) (uint64, error) {
	return unix.SysctlUint64(name)
}

func darwinSwapUsage(raw []byte) (total, free uint64, err error) {
	if len(raw) < 24 {
		return 0, 0, fmt.Errorf("vm.swapusage returned %d bytes", len(raw))
	}
	return binary.LittleEndian.Uint64(raw[0:8]), binary.LittleEndian.Uint64(raw[16:24]), nil
}
