//go:build !windows

package main

import (
	"syscall"
)

func diskSpace() (availableMB int64, usedPercent float64, err error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(watchMount(), &stat); err != nil {
		return 0, 0, err
	}
	if stat.Bsize == 0 {
		return 0, 0, syscall.EINVAL
	}
	available := uint64(stat.Bavail) * uint64(stat.Bsize)
	total := uint64(stat.Blocks) * uint64(stat.Bsize)
	used := total - uint64(stat.Bfree)*uint64(stat.Bsize)
	if total > 0 {
		usedPercent = float64(used) * 100 / float64(total)
	}
	return int64(available / (1024 * 1024)), usedPercent, nil
}
