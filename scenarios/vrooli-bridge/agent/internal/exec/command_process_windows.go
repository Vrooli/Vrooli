//go:build windows

package exec

import "os/exec"

func prepareCommand(*exec.Cmd) {}

func terminateCommand(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}
