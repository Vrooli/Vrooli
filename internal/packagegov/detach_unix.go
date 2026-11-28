//go:build unix

package packagegov

import (
	"io/fs"
	"syscall"
)

// linkCount is the number of directory entries sharing info's inode.
func linkCount(info fs.FileInfo) uint64 {
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		return uint64(stat.Nlink)
	}
	return 1
}
