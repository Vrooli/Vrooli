//go:build unix || !windows

package infra

import (
	"os"
	"os/exec"
	"syscall"
)

func configureProcessCommand(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func gracefulStopProcess(process *os.Process) error {
	// The negative PID addresses the process group created at launch, so child
	// processes cannot survive a supervisor shutdown.
	return syscall.Kill(-process.Pid, syscall.SIGTERM)
}
