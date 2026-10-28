//go:build windows

package target

import (
	"os"
	"os/exec"
)

func configureElectronCommand(_ *exec.Cmd) {}

func signalElectronCommand(cmd *exec.Cmd, signal os.Signal) error {
	if cmd.Process == nil {
		return nil
	}
	return cmd.Process.Signal(signal)
}

func killElectronCommand(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	return cmd.Process.Kill()
}
