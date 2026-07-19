//go:build !windows

package lifecycle

import (
	"errors"
	"syscall"
)

const (
	scenarioLockExclusiveNonblock = syscall.LOCK_EX | syscall.LOCK_NB
	scenarioLockUnlock            = syscall.LOCK_UN
)

var lockFileFlockFn = func(fd int, how int) error { return syscall.Flock(fd, how) }

func isScenarioLockContention(err error) bool {
	return errors.Is(err, syscall.EWOULDBLOCK)
}
