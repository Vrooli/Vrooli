//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package resources

import (
	"os"
	"syscall"
)

func legacyStorageOwnerUID(info os.FileInfo) (uint32, bool) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, false
	}
	return stat.Uid, true
}
