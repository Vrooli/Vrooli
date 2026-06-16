//go:build unix

package executor

import (
	"os/exec"
	"syscall"
)

// setProcessGroup puts the child in its own process group so the whole tree
// (e.g. pnpm -> node -> workers) can be killed together.
func setProcessGroup(c *exec.Cmd) {
	if c.SysProcAttr == nil {
		c.SysProcAttr = &syscall.SysProcAttr{}
	}
	c.SysProcAttr.Setpgid = true
}

// killGroup best-effort terminates the child's whole process group.
func killGroup(c *exec.Cmd) {
	if c.Process == nil {
		return
	}
	// Negative pid targets the process group led by the child.
	_ = syscall.Kill(-c.Process.Pid, syscall.SIGKILL)
}
