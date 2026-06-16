//go:build !unix

package executor

import "os/exec"

// setProcessGroup is a no-op on platforms without POSIX process groups.
func setProcessGroup(*exec.Cmd) {}

// killGroup falls back to killing just the child process.
func killGroup(c *exec.Cmd) {
	if c.Process != nil {
		_ = c.Process.Kill()
	}
}
