//go:build windows

package retention

import (
	"fmt"
	"syscall"
	"unsafe"
)

// freeSpace reports the bytes available to the calling process on the volume
// holding dir.
//
// GetDiskFreeSpaceExW's first out-parameter is the caller-available count, which
// already accounts for disk quotas; the volume-wide total is deliberately not
// used, for the same reason Bavail is preferred over Bfree on unix.
func freeSpace(dir string) (int64, error) {
	pathPtr, err := syscall.UTF16PtrFromString(dir)
	if err != nil {
		return 0, fmt.Errorf("encode path %s: %w", dir, err)
	}
	var freeToCaller, totalBytes, totalFree uint64
	proc := syscall.NewLazyDLL("kernel32.dll").NewProc("GetDiskFreeSpaceExW")
	ret, _, callErr := proc.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		uintptr(unsafe.Pointer(&freeToCaller)),
		uintptr(unsafe.Pointer(&totalBytes)),
		uintptr(unsafe.Pointer(&totalFree)),
	)
	if ret == 0 {
		return 0, fmt.Errorf("GetDiskFreeSpaceEx %s: %w", dir, callErr)
	}
	return int64(freeToCaller), nil
}
