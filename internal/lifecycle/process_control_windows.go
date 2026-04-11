//go:build windows

package lifecycle

import (
	"os"
	"syscall"
)

func backgroundProcessAttr() *syscall.SysProcAttr {
	return nil
}

func signalProcessGroup(pgid int, force bool) error {
	return signalPID(pgid, force)
}

func signalPID(pid int, force bool) error {
	if pid <= 0 {
		return nil
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	if force {
		return process.Kill()
	}
	return process.Kill()
}
