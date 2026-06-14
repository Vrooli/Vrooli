//go:build linux

package process

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"syscall"
)

func pidIsAlive(pid int) bool {
	if err := syscall.Kill(pid, 0); err != nil {
		// EPERM means the PID exists but is owned by another user (e.g. a
		// root-owned scenario process probed from an unprivileged CLI). The
		// process is alive; only ESRCH means it is gone.
		if !errors.Is(err, syscall.EPERM) {
			return false
		}
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
	if closing < 0 || closing+2 >= len(segment) {
		return 0, false
	}
	if segment[closing+1] != ' ' {
		return 0, false
	}
	return segment[closing+2], true
}
