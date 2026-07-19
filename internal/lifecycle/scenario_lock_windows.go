//go:build windows

package lifecycle

import (
	"errors"

	"golang.org/x/sys/windows"
)

const (
	scenarioLockExclusiveNonblock = 1
	scenarioLockUnlock            = 2
)

var lockFileFlockFn = func(fd int, how int) error {
	overlapped := &windows.Overlapped{}
	if how == scenarioLockUnlock {
		return windows.UnlockFileEx(windows.Handle(fd), 0, 1, 0, overlapped)
	}
	return windows.LockFileEx(
		windows.Handle(fd),
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
		0, 1, 0, overlapped,
	)
}

func isScenarioLockContention(err error) bool {
	return errors.Is(err, windows.ERROR_LOCK_VIOLATION) || errors.Is(err, windows.ERROR_IO_PENDING)
}
