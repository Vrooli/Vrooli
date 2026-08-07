//go:build windows

package hostinventory

import "os/exec"

func configureCommandProcessGroup(*exec.Cmd) {}

func terminateCommandProcessGroup(command *exec.Cmd) error {
	if command.Process == nil {
		return nil
	}
	return command.Process.Kill()
}
