//go:build windows

package hostwatchdog

import (
	"syscall"
	"unsafe"
)

var getDiskFreeSpaceEx = syscall.NewLazyDLL("kernel32.dll").NewProc("GetDiskFreeSpaceExW")

func freeSpace(path string) (uint64, float64, error) {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, 0, err
	}
	var available, total, free uint64
	_, _, callErr := getDiskFreeSpaceEx.Call(uintptr(unsafe.Pointer(p)), uintptr(unsafe.Pointer(&available)), uintptr(unsafe.Pointer(&total)), uintptr(unsafe.Pointer(&free)))
	if callErr != syscall.Errno(0) {
		return 0, 0, callErr
	}
	used := total - free
	var percent float64
	if total > 0 {
		percent = float64(used) * 100 / float64(total)
	}
	return available, percent, nil
}
