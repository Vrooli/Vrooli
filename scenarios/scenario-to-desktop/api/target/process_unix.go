//go:build !windows

package target

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
)

func configureElectronCommand(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func signalElectronCommand(cmd *exec.Cmd, signal os.Signal) error {
	if cmd.Process == nil {
		return nil
	}
	if unixSignal, ok := signal.(syscall.Signal); ok {
		if err := syscall.Kill(-cmd.Process.Pid, unixSignal); err == nil {
			return nil
		}
	}
	return cmd.Process.Signal(signal)
}

func killElectronCommand(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	errs := []error{killElectronDescendants(cmd.Process.Pid)}
	if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}
