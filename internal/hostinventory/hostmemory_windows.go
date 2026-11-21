//go:build windows

package hostinventory

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

type memoryStatusEx struct {
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

var globalMemoryStatusEx = windows.NewLazySystemDLL("kernel32.dll").NewProc("GlobalMemoryStatusEx")

func hostMemoryFacts() (HostMemory, error) {
	status := memoryStatusEx{Length: uint32(unsafe.Sizeof(memoryStatusEx{}))}
	if result, _, err := globalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&status))); result == 0 {
		return HostMemory{}, fmt.Errorf("GlobalMemoryStatusEx: %w", err)
	}
	return HostMemory{TotalBytes: status.TotalPhys, AvailableBytes: status.AvailPhys, Trustworthy: status.TotalPhys > 0 && status.AvailPhys > 0}, nil
}
