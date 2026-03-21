package procmetrics

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
)

// LinuxProcReader reads process stats from the /proc filesystem.
type LinuxProcReader struct{}

// ReadStat parses /proc/<pid>/stat and returns utime and stime in clock ticks.
// Fields 14 and 15 (0-indexed from the comm field end) are utime and stime.
func (r *LinuxProcReader) ReadStat(pid int) (utime, stime int64, err error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return 0, 0, fmt.Errorf("read /proc/%d/stat: %w", pid, err)
	}
	return parseStat(string(data))
}

// parseStat extracts utime and stime from a /proc/pid/stat line.
// The comm field (field 2) may contain spaces and parentheses, so we find
// the last ')' to skip past it before splitting the remaining fields.
func parseStat(content string) (utime, stime int64, err error) {
	// Find the end of the comm field (last closing paren)
	idx := strings.LastIndex(content, ")")
	if idx == -1 || idx+2 >= len(content) {
		return 0, 0, fmt.Errorf("malformed /proc/pid/stat: no closing paren")
	}

	// Fields after comm: state(0) ppid(1) pgrp(2) ... utime(11) stime(12) ...
	fields := strings.Fields(content[idx+2:])
	if len(fields) < 13 {
		return 0, 0, fmt.Errorf("malformed /proc/pid/stat: expected >= 13 fields after comm, got %d", len(fields))
	}

	utime, err = strconv.ParseInt(fields[11], 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("parse utime: %w", err)
	}
	stime, err = strconv.ParseInt(fields[12], 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("parse stime: %w", err)
	}
	return utime, stime, nil
}

// ReadStatus parses /proc/<pid>/status for VmRSS, VmPeak, and Threads.
func (r *LinuxProcReader) ReadStatus(pid int) (rssBytes, peakBytes int64, threads int, err error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return 0, 0, 0, fmt.Errorf("read /proc/%d/status: %w", pid, err)
	}
	return parseStatus(string(data))
}

// parseStatus extracts VmRSS, VmPeak, and Threads from /proc/pid/status content.
func parseStatus(content string) (rssBytes, peakBytes int64, threads int, err error) {
	var foundRSS, foundPeak, foundThreads bool

	for _, line := range strings.Split(content, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		switch fields[0] {
		case "VmRSS:":
			val, parseErr := strconv.ParseInt(fields[1], 10, 64)
			if parseErr != nil {
				return 0, 0, 0, fmt.Errorf("parse VmRSS: %w", parseErr)
			}
			rssBytes = val * 1024 // kB → bytes
			foundRSS = true
		case "VmPeak:":
			val, parseErr := strconv.ParseInt(fields[1], 10, 64)
			if parseErr != nil {
				return 0, 0, 0, fmt.Errorf("parse VmPeak: %w", parseErr)
			}
			peakBytes = val * 1024
			foundPeak = true
		case "Threads:":
			val, parseErr := strconv.Atoi(fields[1])
			if parseErr != nil {
				return 0, 0, 0, fmt.Errorf("parse Threads: %w", parseErr)
			}
			threads = val
			foundThreads = true
		}
		if foundRSS && foundPeak && foundThreads {
			break
		}
	}

	if !foundRSS || !foundPeak || !foundThreads {
		var missing []string
		if !foundRSS {
			missing = append(missing, "VmRSS")
		}
		if !foundPeak {
			missing = append(missing, "VmPeak")
		}
		if !foundThreads {
			missing = append(missing, "Threads")
		}
		return 0, 0, 0, fmt.Errorf("missing fields in /proc/pid/status: %s", strings.Join(missing, ", "))
	}

	return rssBytes, peakBytes, threads, nil
}

// IsAlive checks if a process exists by sending signal 0.
func (r *LinuxProcReader) IsAlive(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}
