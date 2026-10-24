//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

var (
	kernel32           = syscall.NewLazyDLL("kernel32.dll")
	getDiskFreeSpaceEx = kernel32.NewProc("GetDiskFreeSpaceExW")
)

func diskSpace() (availableMB int64, usedPercent float64, err error) {
	path, err := syscall.UTF16PtrFromString(watchMount())
	if err != nil {
		return 0, 0, err
	}
	var free, total uint64
	_, _, callErr := getDiskFreeSpaceEx.Call(uintptr(unsafe.Pointer(path)), uintptr(unsafe.Pointer(&free)), uintptr(unsafe.Pointer(&total)), 0)
	if callErr != syscall.Errno(0) {
		return 0, 0, callErr
	}
	if total > 0 {
		usedPercent = float64(total-free) * 100 / float64(total)
	}
	return int64(free / (1024 * 1024)), usedPercent, nil
}
