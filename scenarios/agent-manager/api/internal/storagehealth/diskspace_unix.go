//go:build unix

package storagehealth

import (
	"os"
	"syscall"
)

func freeBytes(dir string) (uint64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(dir, &stat); err != nil {
		return 0, err
	}
	return uint64(stat.Bavail) * uint64(stat.Bsize), nil
}

func sameDevice(a, b string) bool {
	left, errA := os.Stat(a)
	right, errB := os.Stat(b)
	if errA != nil || errB != nil {
		return true
	}
	leftStat, okA := left.Sys().(*syscall.Stat_t)
	rightStat, okB := right.Sys().(*syscall.Stat_t)
	if !okA || !okB {
		return true
	}
	return leftStat.Dev == rightStat.Dev
}
