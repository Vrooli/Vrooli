//go:build !windows

package dbmaintenance

import (
	"os"
	"syscall"
)

// sameFilesystem reports whether two directories share a device. An
// unmeasurable answer is treated as shared, which demands the larger margin.
func sameFilesystem(a, b string) bool {
	infoA, errA := os.Stat(a)
	infoB, errB := os.Stat(b)
	if errA != nil || errB != nil {
		return true
	}
	statA, okA := infoA.Sys().(*syscall.Stat_t)
	statB, okB := infoB.Sys().(*syscall.Stat_t)
	return !okA || !okB || statA.Dev == statB.Dev
}
