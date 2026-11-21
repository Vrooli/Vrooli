//go:build !windows

package pstoreobservability

import (
	"os"
	"syscall"
)

func fileInfoGroupMatches(info os.FileInfo, gid uint32) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Gid == gid
}
