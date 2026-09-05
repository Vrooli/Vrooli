//go:build windows

package collectors

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	kernel32               = syscall.NewLazyDLL("kernel32.dll")
	procGetDiskFreeSpaceEx = kernel32.NewProc("GetDiskFreeSpaceExW")
)

// statfsBytes reports filesystem capacity for the volume containing path.
//
// GetDiskFreeSpaceExW preserves the distinction the Unix path relies on:
// lpFreeBytesAvailableToCaller is quota-aware (available) while
// lpTotalNumberOfFreeBytes is not (free).
func statfsBytes(path string) (total, free, available int64, err error) {
	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid path %q: %w", path, err)
	}

	var freeToCaller, totalBytes, totalFree uint64
	ret, _, callErr := procGetDiskFreeSpaceEx.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		uintptr(unsafe.Pointer(&freeToCaller)),
		uintptr(unsafe.Pointer(&totalBytes)),
		uintptr(unsafe.Pointer(&totalFree)),
	)
	if ret == 0 {
		return 0, 0, 0, fmt.Errorf("GetDiskFreeSpaceEx %q: %w", path, callErr)
	}
	return int64(totalBytes), int64(totalFree), int64(freeToCaller), nil
}
