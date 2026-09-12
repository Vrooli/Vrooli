//go:build unix && !linux

package platform

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
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

func signalPIDWithSignal(pid int, signal os.Signal) error {
	if pid <= 0 {
		return nil
	}
	native, ok := signal.(syscall.Signal)
	if !ok {
		return fmt.Errorf("platform: unsupported process signal %T", signal)
	}
	return syscall.Kill(pid, native)
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

func assignProcessContainment(process *os.Process) (func(), error) {
	return processGroupBoundary(process)
}

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

// processCommandLine shells ps, which is the only portable way to read another
// process's argv on BSD-family hosts. Own-process callers should prefer
// os.Args; this is for naming somebody else.
func processCommandLine(pid int) (string, error) {
	if pid <= 0 {
		return "", fmt.Errorf("platform: invalid pid %d", pid)
	}
	output, err := exec.Command("ps", "-o", "command=", "-p", strconv.Itoa(pid)).Output()
	if err != nil {
		return "", fmt.Errorf("platform: read command line for pid %d: %w", pid, err)
	}
	command := strings.TrimSpace(string(output))
	if command == "" {
		return "", fmt.Errorf("platform: pid %d exposes no command line", pid)
	}
	return command, nil
}

// processScope reports empty: these hosts have no cgroup-equivalent identity
// that outlives the process. Empty is the honest answer, not an error.
func processScope(int) (string, error) { return "", nil }
