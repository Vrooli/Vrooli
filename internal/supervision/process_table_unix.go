//go:build linux || darwin

package supervision

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/shell"
)

const processStartTableFieldCount = 6

func readNativeProcessTable() (map[int]ProcessInfo, error) {
	env := append(os.Environ(), "LC_ALL=C")
	output, err := shell.Output(shell.Spec{
		Name: "ps",
		Args: []string{"-axo", "pid=,lstart="},
		Env:  env,
	})
	if err != nil {
		return nil, fmt.Errorf("inspect process start times: %w", err)
	}
	return parseProcessStartTable(string(output), time.Local)
}

func parseProcessStartTable(output string, location *time.Location) (map[int]ProcessInfo, error) {
	processes := make(map[int]ProcessInfo)
	for lineNumber, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if len(fields) != processStartTableFieldCount {
			return nil, fmt.Errorf("parse process start time line %d: expected pid plus lstart, got %q", lineNumber+1, line)
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			return nil, fmt.Errorf("parse process pid on line %d: %w", lineNumber+1, err)
		}
		startedAt, err := time.ParseInLocation("Mon Jan _2 15:04:05 2006", strings.Join(fields[1:], " "), location)
		if err != nil {
			return nil, fmt.Errorf("parse process start time for pid %d: %w", pid, err)
		}
		processes[pid] = ProcessInfo{PID: pid, StartedAt: startedAt}
	}
	return processes, nil
}
