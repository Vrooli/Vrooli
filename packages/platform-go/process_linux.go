//go:build linux

package platform

import (
	"errors"
	"fmt"
	"os"
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
	if err := syscall.Kill(pid, 0); err != nil && !errors.Is(err, syscall.EPERM) {
		return false
	}
	state, ok := readProcessState(pid)
	return !ok || state != 'Z'
}

func readProcessEnvironment(pid int) (map[string]string, error) {
	data, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/environ")
	if err != nil {
		return nil, err
	}
	return parseEnvironmentEntries(data), nil
}

// processCommandLine reads /proc/<pid>/cmdline, whose arguments are NUL
// separated with a trailing NUL. A kernel thread has an empty cmdline; report
// that as an error rather than an empty command so callers can tell "no
// command" from "not recorded".
func processCommandLine(pid int) (string, error) {
	if pid <= 0 {
		return "", fmt.Errorf("platform: invalid pid %d", pid)
	}
	data, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/cmdline")
	if err != nil {
		return "", err
	}
	fields := strings.FieldsFunc(string(data), func(r rune) bool { return r == 0 })
	if len(fields) == 0 {
		return "", fmt.Errorf("platform: pid %d exposes no command line", pid)
	}
	return strings.Join(fields, " "), nil
}

func processWorkingDir(pid int) (string, error) {
	if pid <= 0 {
		return "", fmt.Errorf("platform: invalid pid %d", pid)
	}
	path, err := os.Readlink("/proc/" + strconv.Itoa(pid) + "/cwd")
	if err != nil {
		return "", err
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("platform: pid %d exposes no working directory", pid)
	}
	return path, nil
}

func processHasChildren(pid int) (bool, error) {
	if pid <= 0 {
		return false, fmt.Errorf("platform: invalid pid %d", pid)
	}
	data, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/task/" + strconv.Itoa(pid) + "/children")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(data)) != "", nil
}

// processScope returns the unified-hierarchy cgroup path for pid. The v2 line
// is "0::<path>"; a v1-only host has no single answer, so it reports empty
// rather than picking a controller arbitrarily.
func processScope(pid int) (string, error) {
	if pid <= 0 {
		return "", fmt.Errorf("platform: invalid pid %d", pid)
	}
	data, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/cgroup")
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		if path, ok := strings.CutPrefix(strings.TrimSpace(line), "0::"); ok {
			return path, nil
		}
	}
	return "", nil
}

func readProcessState(pid int) (byte, bool) {
	data, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return 0, false
	}
	return parseProcessState(data)
}

func parseProcessState(stat []byte) (byte, bool) {
	segment := strings.TrimSpace(string(stat))
	if segment == "" {
		return 0, false
	}
	closing := strings.LastIndex(segment, ")")
	if closing < 0 || closing+2 >= len(segment) || segment[closing+1] != ' ' {
		return 0, false
	}
	return segment[closing+2], true
}

func parseEnvironmentEntries(data []byte) map[string]string {
	values := make(map[string]string)
	for _, entry := range strings.Split(string(data), "\x00") {
		key, value, ok := strings.Cut(entry, "=")
		if ok && key != "" {
			values[key] = value
		}
	}
	return values
}
