//go:build !windows

package lifecycle

import (
	"os"
	"syscall"
)

func backgroundProcessAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}

func signalProcessGroup(pgid int, force bool) error {
	if pgid <= 0 {
		return nil
	}
	signal := syscall.SIGTERM
	if force {
		signal = syscall.SIGKILL
	}
	return syscall.Kill(-pgid, signal)
}

func signalPID(pid int, force bool) error {
	if pid <= 0 {
		return nil
	}
	signal := syscall.SIGTERM
	if force {
		signal = syscall.SIGKILL
	}
	return syscall.Kill(pid, signal)
}

func reraiseSignal(signal os.Signal) error {
	s, ok := signal.(syscall.Signal)
	if !ok {
		return nil
	}
	return syscall.Kill(os.Getpid(), s)
}
