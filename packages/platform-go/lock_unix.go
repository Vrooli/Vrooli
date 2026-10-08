//go:build !windows

package platform

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

func acquireFileLock(path string) (func(), error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return nil, fmt.Errorf("platform: open lock %s: %w", path, err)
	}
	release, err := lockFile(f, false)
	if err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("platform: lock %s: %w", path, err)
	}
	return func() {
		release()
		_ = f.Close()
	}, nil
}

func lockFile(f *os.File, nonBlocking bool) (func(), error) {
	flags := syscall.LOCK_EX
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
