//go:build windows

package health

import (
	"syscall"
	"unsafe"
)

// diskFreeBytes returns the bytes available to the caller on the volume
// containing path, via kernel32!GetDiskFreeSpaceExW. Uses a LazyDLL syscall
// rather than golang.org/x/sys so the agent needs no extra dependency and still
// builds CGO_ENABLED=0 for windows/{amd64,arm64}.
func diskFreeBytes(path string) (uint64, error) {
	free, _, err := diskUsage(path)
	return free, err
}

// diskUsage returns the bytes available to the caller and the volume's total
// size.
func diskUsage(path string) (free, total uint64, err error) {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	proc := kernel32.NewProc("GetDiskFreeSpaceExW")

	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, 0, err
	}

	var freeBytesAvailable, totalBytes uint64
	r, _, callErr := proc.Call(
		uintptr(unsafe.Pointer(p)),
		uintptr(unsafe.Pointer(&freeBytesAvailable)),
		uintptr(unsafe.Pointer(&totalBytes)),
		0,
	)
	if r == 0 {
		return 0, 0, callErr
	}
	return freeBytesAvailable, totalBytes, nil
}
