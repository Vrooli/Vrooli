//go:build unix

package destinations

import (
	"os/exec"
	"syscall"
)

// configureLeaseProcess puts the lease command in its own process group. The
// `vrooli` entry point may spawn the workspace build binary as a child; killing
// only the parent leaves the child holding the exclusive volume flock, which
// blocks every later backup write on the drive until the machine reboots or an
// operator hunts the orphan. A dedicated group lets release kill both.
func configureLeaseProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	// If the parent context ends while the lease is held, kill the whole group,
	// not just the shim, or the holder child keeps the volume flock.
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}

// killLeaseProcess terminates the entire lease process group and reaps it.
func killLeaseProcess(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	_ = cmd.Wait()
}
