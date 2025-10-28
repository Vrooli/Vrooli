package queue

import (
	"os/exec"
	"syscall"
)

// SetProcessGroup sets the process to run in its own process group
// This allows us to kill the entire process tree when needed
func SetProcessGroup(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}

// KillProcessGroup terminates an entire process group
func KillProcessGroup(pid int) error {
	// Send SIGTERM to the entire process group
	pgid, err := syscall.Getpgid(pid)
	if err != nil {
		return err
	}

	// Negative pgid means kill the entire process group
	if err := syscall.Kill(-pgid, syscall.SIGTERM); err != nil {
		return err
	}

	return nil
}
