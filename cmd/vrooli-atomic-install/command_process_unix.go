//go:build darwin || linux

package main

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func startManagedCommand(cmd *exec.Cmd, lock *os.File, _, _ string) (func() error, func(os.Signal) error, func(), error) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if lock != nil {
		// Keep a second reference in the child (fd 3). If this manager is
		// forcibly killed, a still-running command or shell descendant retains
		// the kernel lock until it exits.
		cmd.ExtraFiles = append(cmd.ExtraFiles, lock)
	}
	if err := cmd.Start(); err != nil {
		return nil, nil, nil, err
	}
	return cmd.Wait, func(sig os.Signal) error { return signalManagedCommand(cmd, sig) }, func() {}, nil
}

func managedCommandSignals() []os.Signal {
	return []os.Signal{os.Interrupt, syscall.SIGTERM, syscall.SIGHUP}
}

func signalManagedCommand(cmd *exec.Cmd, sig os.Signal) error {
	if cmd.Process == nil {
		return nil
	}
	if native, ok := sig.(syscall.Signal); ok {
		return syscall.Kill(-cmd.Process.Pid, native)
	}
	return cmd.Process.Signal(sig)
}

func waitManagedCommand(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	for {
		err := syscall.Kill(-cmd.Process.Pid, 0)
		if errors.Is(err, syscall.ESRCH) {
			return nil
		}
		if err != nil && !errors.Is(err, syscall.EPERM) {
			return err
		}
		time.Sleep(10 * time.Millisecond)
	}
}
