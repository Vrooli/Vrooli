//go:build windows

package checks

import (
	"fmt"
	"syscall"
	"unsafe"
)

// kernel32 is resolved lazily so the package still loads on systems where the
// DLL is unavailable; the error surfaces from the Statfs call instead.
var (
	kernel32               = syscall.NewLazyDLL("kernel32.dll")
	procGetDiskFreeSpaceEx = kernel32.NewProc("GetDiskFreeSpaceExW")
)

// Statfs returns filesystem statistics using GetDiskFreeSpaceExW.
//
// Windows reports bytes rather than blocks, so the result uses a block size of
// one and carries byte counts in the block fields. Callers derive everything
// from Blocks*Bsize, so the arithmetic is identical to the Unix path.
//
// The distinction this scenario depends on survives the mapping:
// lpFreeBytesAvailableToCaller is the quota-aware space an unprivileged writer
// can use (Bavail), while lpTotalNumberOfFreeBytes ignores quotas (Bfree).
// Windows has no inode concept, so Files and Ffree stay zero; the inode check
// skips Windows explicitly and every other reader guards on a non-zero total.
func (r *RealFileSystemReader) Statfs(path string) (*StatfsResult, error) {
	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, fmt.Errorf("invalid path %q: %w", path, err)
	}

	var freeToCaller, totalBytes, totalFree uint64
	ret, _, callErr := procGetDiskFreeSpaceEx.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		uintptr(unsafe.Pointer(&freeToCaller)),
		uintptr(unsafe.Pointer(&totalBytes)),
		uintptr(unsafe.Pointer(&totalFree)),
	)
	if ret == 0 {
		return nil, fmt.Errorf("GetDiskFreeSpaceEx %q: %w", path, callErr)
	}

	return &StatfsResult{
		Blocks: totalBytes,
		Bfree:  totalFree,
		Bavail: freeToCaller,
		Bsize:  1,
	}, nil
}
