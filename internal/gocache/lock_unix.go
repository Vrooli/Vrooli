//go:build unix

package gocache

import (
	"errors"
	"os"
	"syscall"
)

// protocolSupported reports whether this platform provides the advisory
// locks the protocol depends on.
const protocolSupported = true

func flock(f *os.File, how int) error {
	for {
		err := syscall.Flock(int(f.Fd()), how)
		if !errors.Is(err, syscall.EINTR) {
			return err
		}
	}
}

func lockShared(f *os.File) error    { return flock(f, syscall.LOCK_SH) }
func lockExclusive(f *os.File) error { return flock(f, syscall.LOCK_EX) }
func unlock(f *os.File) error        { return flock(f, syscall.LOCK_UN) }

// tryLock attempts a non-blocking lock and reports whether it was granted.
func tryLock(f *os.File, exclusive bool) (bool, error) {
	how := syscall.LOCK_SH | syscall.LOCK_NB
	if exclusive {
		how = syscall.LOCK_EX | syscall.LOCK_NB
	}
	err := flock(f, how)
	if errors.Is(err, syscall.EWOULDBLOCK) {
		return false, nil
	}
	return err == nil, err
}

// allocatedBytes reports the disk space a file occupies, which is what
// reclaim frees, falling back to its length.
func allocatedBytes(info os.FileInfo) int64 {
	if stat, ok := info.Sys().(*syscall.Stat_t); ok && stat.Blocks > 0 {
		return stat.Blocks * 512
	}
	return info.Size()
}

// fileIdentity reports the device and inode that identify a file system
// object independently of the path used to reach it.
func fileIdentity(info os.FileInfo) (device, inode uint64, ok bool) {
	stat, isStat := info.Sys().(*syscall.Stat_t)
	if !isStat {
		return 0, 0, false
	}
	return uint64(stat.Dev), stat.Ino, true //nolint:unconvert // Dev is int32 on some unix platforms.
}
