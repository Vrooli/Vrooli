//go:build linux

package smoketest

import (
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

func evictAndMeasureArtifactPageCache(path string) (float64, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, fmt.Errorf("open artifact: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return 0, fmt.Errorf("stat artifact: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() == 0 {
		return 0, fmt.Errorf("artifact must be a non-empty regular file")
	}
	size := int(info.Size())
	mapped, err := unix.Mmap(int(file.Fd()), 0, size, unix.PROT_READ, unix.MAP_SHARED)
	if err != nil {
		return 0, fmt.Errorf("map artifact for residency check: %w", err)
	}
	defer unix.Munmap(mapped)
	if err := file.Sync(); err != nil {
		return 0, fmt.Errorf("flush artifact before eviction: %w", err)
	}
	if err := unix.Fadvise(int(file.Fd()), 0, info.Size(), unix.FADV_DONTNEED); err != nil {
		return 0, fmt.Errorf("posix_fadvise DONTNEED: %w", err)
	}
	pages := (size + os.Getpagesize() - 1) / os.Getpagesize()
	residency := make([]byte, pages)
	_, _, errno := unix.Syscall(unix.SYS_MINCORE, uintptr(unsafe.Pointer(&mapped[0])), uintptr(size), uintptr(unsafe.Pointer(&residency[0])))
	if errno != 0 {
		return 0, fmt.Errorf("mincore artifact: %w", errno)
	}
	resident := 0
	for _, page := range residency {
		if page&1 != 0 {
			resident++
		}
	}
	return float64(resident) * 100 / float64(len(residency)), nil
}
