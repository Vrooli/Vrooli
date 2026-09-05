//go:build windows

package collectors

import (
	"context"
	"fmt"
	"unsafe"

	"github.com/vrooli/vrooli/internal/hostinventory"
	"golang.org/x/sys/windows"
)

type windowsMemoryStatusEx struct {
	Length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

var procGlobalMemoryStatusEx = windows.NewLazySystemDLL("kernel32.dll").NewProc("GlobalMemoryStatusEx")

func collectPlatformMemory(context.Context, *MemoryCollector, hostinventory.Snapshot) platformMemoryReading {
	status := windowsMemoryStatusEx{Length: uint32(unsafe.Sizeof(windowsMemoryStatusEx{}))}
	ret, _, callErr := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&status)))
	if ret == 0 {
		return platformMemoryReading{status: "failed", reason: fmt.Sprintf("GlobalMemoryStatusEx: %v", callErr), provenance: "Windows GlobalMemoryStatusEx"}
	}
	return memoryReadingFromBytes(status.TotalPhys, status.AvailPhys, 0, 0, status.TotalPageFile, status.AvailPageFile, "Windows GlobalMemoryStatusEx")
}
