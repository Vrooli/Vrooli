//go:build !windows

package orchestrator

import (
	"syscall"
)

func measureRecoveryFreeBytes(path string) (int64, error) {
	_, available, err := measureRecoverySpace(path)
	return available, err
}

func measureRecoverySpace(path string) (float64, int64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, 0, err
	}
	total := uint64(stat.Blocks) * uint64(stat.Bsize)
	available := uint64(stat.Bavail) * uint64(stat.Bsize)
	if total == 0 {
		return 0, int64(available), nil
	}
	used := float64(total-available) / float64(total) * 100
	return used, int64(available), nil
}
