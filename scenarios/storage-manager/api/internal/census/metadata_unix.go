//go:build aix || android || darwin || dragonfly || freebsd || hurd || illumos || linux || netbsd || openbsd || solaris

package census

import (
	"os"
	"syscall"
)

type fileIdentity struct {
	device uint64
	inode  uint64
	valid  bool
}

type fileMetadata struct {
	identity  fileIdentity
	device    uint64
	bytes     int64
	allocated int64
}

func inspectPathHost(path string) (fileMetadata, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return fileMetadata{}, err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fileMetadata{bytes: info.Size()}, nil
	}
	bytes := info.Size()
	allocated := bytes
	// POSIX st_blocks is expressed in 512-byte units. Allocated bytes are the
	// correct quantity for reconciling a file walk with statfs usage.
	allocated = int64(stat.Blocks) * 512
	return fileMetadata{identity: fileIdentity{device: uint64(stat.Dev), inode: uint64(stat.Ino), valid: true}, device: uint64(stat.Dev), bytes: bytes, allocated: allocated}, nil
}

func (hostFileSystem) inspectMetadata(path string) (fileMetadata, error) {
	return inspectPathHost(path)
}
