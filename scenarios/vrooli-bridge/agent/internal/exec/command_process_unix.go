//go:build !windows

package exec

import (
	"os/exec"
	"syscall"
)

func prepareCommand(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func terminateCommand(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	// The relay contract owns the complete local command tree. Killing only the
	// direct child would leave a tool it spawned holding stdout open and would
	// violate cancellation's no-surviving-child guarantee.
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}
