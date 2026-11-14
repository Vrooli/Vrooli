//go:build !(aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris)

package resources

import "io/fs"

// artifactFileIdentity has no inode on this platform; every file counts as
// unshared.
func artifactFileIdentity(info fs.FileInfo) (artifactInode, uint64, int64) {
	return artifactInode{}, 1, info.Size()
}
