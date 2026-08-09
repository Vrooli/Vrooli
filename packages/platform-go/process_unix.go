//go:build unix && !linux

package platform

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

func detachedProcessAttrs() *syscall.SysProcAttr { return &syscall.SysProcAttr{Setsid: true} }

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

func signalProcessGroup(groupID int, force bool) error {
	if groupID <= 0 {
		return nil
	}
	signal := syscall.SIGTERM
	if force {
		signal = syscall.SIGKILL
	}
	return syscall.Kill(-groupID, signal)
}

func reraiseSignal(signal os.Signal) error {
	native, ok := signal.(syscall.Signal)
	if !ok {
		return nil
	}
	return syscall.Kill(os.Getpid(), native)
}

func killProcess(pid int, force bool) error { return signalPID(pid, force) }

func replaceProcess(argv0 string, argv []string, env []string) error {
	return syscall.Exec(argv0, argv, env)
}

func assignProcessContainment(*os.Process) (func(), error) { return func() {}, nil }

func gracefulStopProcess(process *os.Process) error {
	if process == nil {
		return nil
	}
	return signalProcessGroup(process.Pid, false)
}

func processGroupID(pid int) (int, error) { return syscall.Getpgid(pid) }

func terminationSignals() []os.Signal { return []os.Signal{os.Interrupt, syscall.SIGTERM} }

func pidIsAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	if err := syscall.Kill(pid, 0); err != nil {
		return errors.Is(err, syscall.EPERM)
	}
	return true
}

func readProcessEnvironment(int) (map[string]string, error) {
	return nil, fmt.Errorf("platform: process environment inspection is not supported on this platform")
}
