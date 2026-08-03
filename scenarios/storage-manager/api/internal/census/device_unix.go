//go:build aix || android || darwin || dragonfly || freebsd || hurd || illumos || linux || netbsd || openbsd || solaris

package census

import (
	"syscall"
)

func deviceCoverage(root string, measured int64, complete bool) ScanCoverage {
	coverage := ScanCoverage{MeasuredBytes: measured, Complete: complete}
	var stat syscall.Statfs_t
	if err := syscall.Statfs(root, &stat); err != nil {
		return coverage
	}
	blockSize := int64(stat.Bsize)
	if blockSize <= 0 {
		return coverage
	}
	coverage.DeviceTotalBytes = int64(stat.Blocks) * blockSize
	available := int64(stat.Bavail) * blockSize
	coverage.DeviceUsedBytes = coverage.DeviceTotalBytes - available
	coverage.MeasuredByDevice = true
	return coverage
}
