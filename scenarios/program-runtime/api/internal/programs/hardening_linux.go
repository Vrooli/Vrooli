//go:build linux

package programs

import (
	"fmt"
	"os/exec"
	"syscall"

	"golang.org/x/sys/unix"
)

func configureProcessGroup(cmd *exec.Cmd) error {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return nil
}

func applyProcessLimits(pid int) error {
	limits := []struct {
		resource int
		value    uint64
	}{{unix.RLIMIT_AS, 1 << 30}, {unix.RLIMIT_CPU, 125}, {unix.RLIMIT_FSIZE, 64 << 20}, {unix.RLIMIT_NOFILE, 256}}
	for _, limit := range limits {
		r := &unix.Rlimit{Cur: limit.value, Max: limit.value}
		if err := unix.Prlimit(pid, limit.resource, r, nil); err != nil {
			return fmt.Errorf("apply kernel resource limit %d: %w", limit.resource, err)
		}
	}
	return nil
}

func killProcessGroup(pid int) { _ = syscall.Kill(-pid, syscall.SIGKILL) }
