//go:build !windows

package buildinfo

import (
	"fmt"
	"os"
	"syscall"
)

var (
	execFn = func(argv0 string, argv []string, envv []string) error {
		return syscall.Exec(argv0, argv, envv)
	}
	// flockFn is injectable so tests can simulate contention without taking
	// real kernel locks.
	flockFn = func(fd int, how int) error { return syscall.Flock(fd, how) }
)

func acquireRebuildLock(executable string) (func(), error) {
	lockPath := executable + ".lock"
	f, err := openFileFn(lockPath, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open lock %s: %w", lockPath, err)
	}
	if err := flockFn(int(f.Fd()), syscall.LOCK_EX); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("flock %s: %w", lockPath, err)
	}
	return func() {
		_ = flockFn(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
	}, nil
}
