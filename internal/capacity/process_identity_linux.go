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

// ProcessStartIdentity returns the Linux start token that, together with a
// PID, identifies one process for its whole life. Owners embed it in claim
// identities so a reused PID can never be mistaken for the owner.
func ProcessStartIdentity(pid int) (string, error) { return processStartIdentity(pid) }

// processGone reports whether the process identified by pid (and, when known,
// its start token) has definitely exited. Anything uncertain - an unreadable
// /proc entry, a malformed stat line - is "not gone": callers use this to
// release capacity, and releasing a live owner's capacity is worse than
// holding a dead owner's for one more TTL.
func processGone(pid int, expectedStart string) bool {
	if pid <= 0 {
		return false
	}
	b, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return os.IsNotExist(err)
	}
	text := string(b)
	end := strings.LastIndexByte(text, ')')
	if end < 0 || end+2 >= len(text) {
		return false
	}
	fields := strings.Fields(text[end+2:])
	if len(fields) <= 19 {
		return false
	}
	// A zombie has exited; only its parent's wait is outstanding.
	if fields[0] == "Z" || fields[0] == "X" {
		return true
	}
	return expectedStart != "" && fields[19] != expectedStart
}
