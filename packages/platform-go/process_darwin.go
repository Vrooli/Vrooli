//go:build darwin

package platform

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

// darwinProcessZombie is SZOMB from <sys/proc.h>: exited, not yet reaped.
const darwinProcessZombie = 5

// pidIsAlive treats a zombie as gone, as Linux does. kill(pid, 0) succeeds for
// an exited child its parent has not reaped, so without the state check a
// teardown that already killed a process would wait on it until its deadline.
func pidIsAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	if err := syscall.Kill(pid, 0); err != nil && !errors.Is(err, syscall.EPERM) {
		return false
	}
	info, err := unix.SysctlKinfoProc("kern.proc.pid", pid)
	if err != nil || info == nil {
		// kill already proved the pid exists; an unreadable state is not death.
		return true
	}
	return info.Proc.P_stat != darwinProcessZombie
}

// readProcArgs reads the kern.procargs2 block. The kernel serves it for the
// caller's own processes (and to root); another user's process is an error.
func readProcArgs(pid int) (procArgs, error) {
	if pid <= 0 {
		return procArgs{}, fmt.Errorf("platform: invalid pid %d", pid)
	}
	data, err := unix.SysctlRaw("kern.procargs2", pid)
	if err != nil {
		return procArgs{}, fmt.Errorf("platform: read process arguments for pid %d: %w", pid, err)
	}
	return parseProcArgs2(data)
}

func readProcessEnvironment(pid int) (map[string]string, error) {
	args, err := readProcArgs(pid)
	if err != nil {
		return nil, err
	}
	return args.Env, nil
}

func processExecutablePath(pid int) (string, error) {
	args, err := readProcArgs(pid)
	if err != nil {
		return "", err
	}
	return args.Executable, nil
}

// processName asks ps for the executable name alone.
func processName(pid int) (string, error) {
	if pid <= 0 {
		return "", fmt.Errorf("platform: invalid pid %d", pid)
	}
	output, err := exec.Command("ps", "-o", "comm=", "-p", strconv.Itoa(pid)).Output()
	if err != nil {
		return "", fmt.Errorf("platform: read program name for pid %d: %w", pid, err)
	}
	name := strings.TrimSpace(string(output))
	if name == "" {
		return "", fmt.Errorf("platform: pid %d exposes no program name", pid)
	}
	return filepath.Base(name), nil
}

func processWorkingDir(pid int) (string, error) {
	if pid <= 0 {
		return "", fmt.Errorf("platform: invalid pid %d", pid)
	}
	if _, err := exec.LookPath("lsof"); err != nil {
		return "", ErrUnsupported
	}
	output, err := exec.Command("lsof", "-a", "-p", strconv.Itoa(pid), "-d", "cwd", "-Fn").Output()
	if err != nil {
		return "", fmt.Errorf("platform: read working directory for pid %d: %w", pid, err)
	}
	for _, line := range strings.Split(string(output), "\n") {
		if path, ok := strings.CutPrefix(line, "n"); ok && strings.TrimSpace(path) != "" {
			return strings.TrimSpace(path), nil
		}
	}
	return "", fmt.Errorf("platform: pid %d exposes no working directory", pid)
}

func processHasChildren(pid int) (bool, error) {
	if pid <= 0 {
		return false, fmt.Errorf("platform: invalid pid %d", pid)
	}
	if _, err := exec.LookPath("pgrep"); err != nil {
		return false, ErrUnsupported
	}
	output, err := exec.Command("pgrep", "-P", strconv.Itoa(pid)).Output()
	if err != nil {
		// pgrep uses exit status 1 for "no matching processes".
		if len(output) == 0 {
			return false, nil
		}
		return false, fmt.Errorf("platform: inspect children for pid %d: %w", pid, err)
	}
	return strings.TrimSpace(string(output)) != "", nil
}
