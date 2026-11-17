//go:build linux

package rootmetrics

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Process ancestry is an untrusted observation, not an authenticated identity.
// Read only allowlisted comm values and parent IDs, never arguments or paths.
func observeInitiator() string { return classifyAncestry(os.Getppid(), readProcessStatus) }

func readProcessStatus(pid int) (string, int, bool) {
	b, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil || len(b) > 65536 {
		return "", 0, false
	}
	var name string
	var parent int
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "Name:\t") {
			name = strings.TrimSpace(strings.TrimPrefix(line, "Name:\t"))
		}
		if strings.HasPrefix(line, "PPid:\t") {
			parent, _ = strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "PPid:\t")))
		}
	}
	return name, parent, name != "" && parent >= 0
}

func classifyAncestry(pid int, read func(int) (string, int, bool)) string {
	// Prefer a recognizable orchestrator over intermediary shells/installers.
	for i := 0; i < 16 && pid > 1; i++ {
		name, parent, ok := read(pid)
		if !ok {
			break
		}
		switch name {
		case "make", "gmake":
			return "make"
		case "codex":
			return "codex"
		case "systemd":
			return "systemd"
		}
		if parent <= 0 || parent == pid {
			break
		}
		pid = parent
	}
	return "unknown"
}
