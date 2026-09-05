//go:build windows

package platform

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/sys/windows"
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
	flags := uint32(0)
	if nonBlocking {
		flags |= windows.LOCKFILE_FAIL_IMMEDIATELY
	}
	if err := windows.LockFileEx(windows.Handle(f.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK|flags, 0, 1, 0, &windows.Overlapped{}); err != nil {
		if nonBlocking && errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
			return nil, ErrLockUnavailable
		}
		return nil, err
	}
	return func() { _ = windows.UnlockFileEx(windows.Handle(f.Fd()), 0, 1, 0, &windows.Overlapped{}) }, nil
}
