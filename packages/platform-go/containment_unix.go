//go:build unix

package platform

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

// processGroupBoundary is the Unix tree boundary: the detached process group
// a child starts in. There is no handle to retain, and the release is a
// no-op by design; the pid is checked so a caller never holds a boundary to
// a process that never started. Resource ceilings are ContainedCommand's job.
func processGroupBoundary(process *os.Process) (func(), error) {
	if process == nil || process.Pid <= 0 {
		return nil, errors.New("platform: invalid process for containment")
	}
	if err := syscall.Kill(process.Pid, 0); err != nil && !errors.Is(err, syscall.EPERM) {
		return nil, fmt.Errorf("platform: process %d is not alive: %w", process.Pid, err)
	}
	return func() {}, nil
}
