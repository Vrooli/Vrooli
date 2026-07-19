//go:build windows

package buildinfo

import (
	"fmt"
	"os"
	"os/exec"

	"golang.org/x/sys/windows"
)

const (
	windowsLockExclusive = 1
	windowsLockUnlock    = 2
)

var (
	execFn = func(argv0 string, argv []string, envv []string) error {
		args := []string(nil)
		if len(argv) > 1 {
			args = argv[1:]
		}
		cmd := exec.Command(argv0, args...)
		cmd.Env = envv
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	// flockFn retains the same injectable seam as Unix so the shared locking
	// tests compile and exercise failure handling on every host.
	flockFn = func(fd int, how int) error {
		overlapped := &windows.Overlapped{}
		if how == windowsLockUnlock {
			return windows.UnlockFileEx(windows.Handle(fd), 0, 1, 0, overlapped)
		}
		return windows.LockFileEx(windows.Handle(fd), windows.LOCKFILE_EXCLUSIVE_LOCK, 0, 1, 0, overlapped)
	}
)

func acquireRebuildLock(executable string) (func(), error) {
	lockPath := executable + ".lock"
	f, err := openFileFn(lockPath, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open lock %s: %w", lockPath, err)
	}
	if err := flockFn(int(f.Fd()), windowsLockExclusive); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("lock %s: %w", lockPath, err)
	}
	return func() {
		_ = flockFn(int(f.Fd()), windowsLockUnlock)
		_ = f.Close()
	}, nil
}
