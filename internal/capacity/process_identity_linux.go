//go:build linux

package capacity

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// processStartIdentity reads Linux /proc start ticks so a reused PID cannot
// impersonate the process that opened a disk job lease.
func processStartIdentity(pid int) (string, error) {
	b, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return "", err
	}
	// comm is parenthesized and may itself contain spaces or ')'. Fields after
	// its final ')' begin at proc stat field 3; starttime is field 22.
	text := string(b)
	end := strings.LastIndexByte(text, ')')
	if end < 0 || end+2 >= len(text) {
		return "", fmt.Errorf("malformed /proc/%d/stat", pid)
	}
	fields := strings.Fields(text[end+2:])
	if len(fields) <= 19 {
		return "", fmt.Errorf("/proc/%d/stat lacks start time", pid)
	}
	if _, err := strconv.ParseUint(fields[19], 10, 64); err != nil {
		return "", err
	}
	return fields[19], nil
}

func processIdentityMatches(pid int, expectedStart string) bool {
	actual, err := processStartIdentity(pid)
	return err == nil && actual == expectedStart
}
