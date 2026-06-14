//go:build unix && !linux

package process

import (
	"errors"
	"fmt"
	"syscall"
)

func pidIsAlive(pid int) bool {
	if err := syscall.Kill(pid, 0); err != nil {
		// EPERM means the PID exists but is owned by another user; the
		// process is alive. Only ESRCH means it is gone. There is no
		// /proc on darwin/BSD, so zombies are not distinguished here.
		return errors.Is(err, syscall.EPERM)
	}
	return true
}

func readProcessEnvironment(pid int) (map[string]string, error) {
	return nil, fmt.Errorf("process environment inspection is not supported on this platform")
}
