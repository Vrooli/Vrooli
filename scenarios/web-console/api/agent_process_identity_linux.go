//go:build linux

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// platformDiscoverAgentProcesses reads /proc to find agent processes started by
// Web Console. Everything it needs — the owning session, the working directory,
// and the start time — is available without running an external command, so the
// scan stays cheap enough to run on the transcript poller's cadence.
//
// Unreadable entries are skipped rather than reported: /proc races with process
// exit constantly, and a vanished process is not an error worth surfacing.
func platformDiscoverAgentProcesses() ([]agentProcess, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}
	bootTime, ok := procBootTime()
	found := make([]agentProcess, 0, 8)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid, convErr := strconv.Atoi(entry.Name())
		if convErr != nil {
			continue
		}
		base := filepath.Join("/proc", entry.Name())
		if !agentProcessNames[procExecutableName(base)] {
			continue
		}
		sessionID := procEnvValue(base, agentProcessSessionEnvKey)
		if sessionID == "" {
			continue
		}
		proc := agentProcess{PID: pid, SessionID: sessionID}
		if cwd, linkErr := os.Readlink(filepath.Join(base, "cwd")); linkErr == nil {
			proc.WorkingDir = cwd
		}
		if ok {
			if started, startErr := procStartTime(base, bootTime); startErr == nil {
				proc.StartedAt = started
			}
		}
		found = append(found, proc)
	}
	return found, nil
}

// procExecutableName returns the process's own name from /proc/<pid>/comm,
// which is the command actually executing rather than argv[0].
func procExecutableName(base string) string {
	data, err := os.ReadFile(filepath.Join(base, "comm"))
	if err != nil {
		return ""
	}
	return string(bytes.TrimSpace(data))
}

// procEnvValue extracts one NUL-delimited variable from a process environment.
// Reading another user's environ returns EACCES, which is skipped like any
// other unreadable entry.
func procEnvValue(base, key string) string {
	data, err := os.ReadFile(filepath.Join(base, "environ"))
	if err != nil {
		return ""
	}
	prefix := []byte(key + "=")
	for _, field := range bytes.Split(data, []byte{0}) {
		if bytes.HasPrefix(field, prefix) {
			return string(field[len(prefix):])
		}
	}
	return ""
}

// procStartTime converts field 22 of /proc/<pid>/stat (start time in clock
// ticks since boot) into wall-clock time. The process name in field 2 can
// contain spaces and parentheses, so parsing starts after the final ')'.
func procStartTime(base string, bootTime time.Time) (time.Time, error) {
	data, err := os.ReadFile(filepath.Join(base, "stat"))
	if err != nil {
		return time.Time{}, err
	}
	nameEnd := bytes.LastIndexByte(data, ')')
	if nameEnd < 0 || nameEnd+2 >= len(data) {
		return time.Time{}, os.ErrInvalid
	}
	fields := bytes.Fields(data[nameEnd+2:])
	// Field 22 overall is index 19 once the first three fields (pid, comm,
	// state) are accounted for and state becomes index 0 here.
	const startTimeIndex = 19
	if len(fields) <= startTimeIndex {
		return time.Time{}, os.ErrInvalid
	}
	ticks, convErr := strconv.ParseInt(string(fields[startTimeIndex]), 10, 64)
	if convErr != nil {
		return time.Time{}, convErr
	}
	return bootTime.Add(time.Duration(ticks) * time.Second / time.Duration(procClockTicksPerSecond)), nil
}

// procClockTicksPerSecond is USER_HZ. It is 100 on every Linux architecture Go
// supports, and is fixed at kernel build time rather than being tunable.
const procClockTicksPerSecond = 100

// procBootTime reads the btime field from /proc/stat, the epoch second the
// kernel booted, which anchors per-process start ticks to wall-clock time.
func procBootTime() (time.Time, bool) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return time.Time{}, false
	}
	for _, line := range bytes.Split(data, []byte{'\n'}) {
		if !bytes.HasPrefix(line, []byte("btime ")) {
			continue
		}
		seconds, convErr := strconv.ParseInt(string(bytes.TrimSpace(line[len("btime "):])), 10, 64)
		if convErr != nil {
			return time.Time{}, false
		}
		return time.Unix(seconds, 0), true
	}
	return time.Time{}, false
}
