//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package resources

import (
	"io/fs"
	"syscall"
)

// artifactFileIdentity reports the inode, link count and allocated bytes of a
// file so retention can tell shared (hardlinked) bytes from reclaimable ones.
func artifactFileIdentity(info fs.FileInfo) (artifactInode, uint64, int64) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return artifactInode{}, 1, info.Size()
	}
	return artifactInode{dev: uint64(stat.Dev), ino: uint64(stat.Ino)}, uint64(stat.Nlink), int64(stat.Blocks) * 512 //nolint:unconvert // field widths differ by platform
}
