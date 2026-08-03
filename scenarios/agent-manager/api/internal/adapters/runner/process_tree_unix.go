//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package runner

import (
	"os/exec"
	"syscall"
	"time"
)

func killProcessTree(cmd *exec.Cmd) {
	if cmd != nil && cmd.Process != nil {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}

func signalProcessTree(cmd *exec.Cmd, grace time.Duration) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	pgid := -cmd.Process.Pid
	if err := syscall.Kill(pgid, syscall.SIGTERM); err != nil {
		_ = syscall.Kill(pgid, syscall.SIGKILL)
		return
	}
	go func() {
		time.Sleep(grace)
		_ = syscall.Kill(pgid, syscall.SIGKILL)
	}()
}
