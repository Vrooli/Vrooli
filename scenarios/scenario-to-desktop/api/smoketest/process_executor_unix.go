//go:build !windows

package smoketest

import (
	"os/exec"
	"syscall"
	"time"
)

func configureProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func terminateProcessTree(cmd *exec.Cmd) {
	if pgid, err := syscall.Getpgid(cmd.Process.Pid); err == nil {
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
		return
	}
	_ = cmd.Process.Kill()
}

// cleanupProcessGroupAfterLeaderExit terminates descendants that outlive a
// successfully-exited launcher. Setpgid makes the launcher's PID the process
// group ID, so addressing -pid remains precise even after the leader is reaped.
func cleanupProcessGroupAfterLeaderExit(pid int, grace time.Duration) {
	if pid <= 0 {
		return
	}
	err := syscall.Kill(-pid, syscall.SIGTERM)
	if err == syscall.ESRCH {
		return
	}
	if err != nil {
		return
	}
	if grace > 0 {
		time.Sleep(grace)
	}
	_ = syscall.Kill(-pid, syscall.SIGKILL)
}
