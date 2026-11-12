//go:build !windows

package platform

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

func openLockFile(path string, perm os.FileMode) (*os.File, error) {
	fd, err := syscall.Open(path, syscall.O_RDWR|syscall.O_CREAT|syscall.O_NOFOLLOW|syscall.O_NONBLOCK|syscall.O_CLOEXEC, uint32(perm.Perm()))
	if err != nil {
		return nil, fmt.Errorf("platform: open lock %s: %w", path, err)
	}
	f := os.NewFile(uintptr(fd), path)
	if err := validateLockFile(f); err != nil {
		_ = f.Close()
		return nil, err
	}
	return f, nil
}

func validateLockFile(f *os.File) error {
	info, err := f.Stat()
	if err != nil {
		return err
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !info.Mode().IsRegular() || !ok || st.Nlink != 1 {
		return fmt.Errorf("platform: lock must be a regular single-link file")
	}
	return nil
}

func lockFile(f *os.File, nonBlocking bool) (func(), error) {
	return lockFileMode(f, nonBlocking, false)
}

func lockFileMode(f *os.File, nonBlocking, shared bool) (func(), error) {
	flags := syscall.LOCK_EX
	if shared {
		flags = syscall.LOCK_SH
	}
	if nonBlocking {
		flags |= syscall.LOCK_NB
	}
	if err := syscall.Flock(int(f.Fd()), flags); err != nil {
		if nonBlocking && (errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN)) {
			return nil, ErrLockUnavailable
		}
		return nil, err
	}
	return func() { _ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN) }, nil
}
