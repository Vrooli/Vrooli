//go:build darwin

package platform

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

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
