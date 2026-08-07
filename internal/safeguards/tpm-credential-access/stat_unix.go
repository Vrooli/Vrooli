//go:build !windows

package tpmcredentialaccess

import (
	"os"
	"syscall"
)

func fileInfoGroupID(info os.FileInfo) (uint32, bool) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, false
	}
	return stat.Gid, true
}
