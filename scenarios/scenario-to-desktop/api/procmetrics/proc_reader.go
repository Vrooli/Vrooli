package procmetrics

import (
	"fmt"
	"os"
	"path/filepath"
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

// statusFields holds the parsed fields from /proc/pid/status.
type statusFields struct {
	rssBytes     int64
	peakBytes    int64
	threads      int
	foundRSS     bool
	foundPeak    bool
	foundThreads bool
}

// allFound returns true when all required fields have been parsed.
func (sf *statusFields) allFound() bool {
	return sf.foundRSS && sf.foundPeak && sf.foundThreads
}

// missingNames returns the names of fields that were not found.
func (sf *statusFields) missingNames() []string {
	var missing []string
	if !sf.foundRSS {
		missing = append(missing, "VmRSS")
	}
	if !sf.foundPeak {
		missing = append(missing, "VmPeak")
	}
	if !sf.foundThreads {
		missing = append(missing, "Threads")
	}
	return missing
}

// parseStatusField parses a single /proc/pid/status field into sf.
func parseStatusField(sf *statusFields, key, value string) error {
	switch key {
	case "VmRSS:":
		val, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("parse VmRSS: %w", err)
		}
		sf.rssBytes = val * 1024 // kB -> bytes
		sf.foundRSS = true
	case "VmPeak:":
		val, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("parse VmPeak: %w", err)
		}
		sf.peakBytes = val * 1024
		sf.foundPeak = true
	case "Threads:":
		val, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("parse Threads: %w", err)
		}
		sf.threads = val
		sf.foundThreads = true
	}
	return nil
}

// parseStatus extracts VmRSS, VmPeak, and Threads from /proc/pid/status content.
func parseStatus(content string) (rssBytes, peakBytes int64, threads int, err error) {
	var sf statusFields

	for _, line := range strings.Split(content, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if err := parseStatusField(&sf, fields[0], fields[1]); err != nil {
			return 0, 0, 0, err
		}
		if sf.allFound() {
			break
		}
	}

	if !sf.allFound() {
		return 0, 0, 0, fmt.Errorf("missing fields in /proc/pid/status: %s", strings.Join(sf.missingNames(), ", "))
	}

	return sf.rssBytes, sf.peakBytes, sf.threads, nil
}

// IsAlive checks if a process exists by sending signal 0.
func (r *LinuxProcReader) IsAlive(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

// ProcessTree returns the unique descendants of rootPID that remain visible
// in /proc. A disappearing process is skipped because short-lived Electron
// helpers are expected during startup.
func (r *LinuxProcReader) ProcessTree(rootPID int) ([]ProcessInfo, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, fmt.Errorf("read /proc: %w", err)
	}
	all := make(map[int]ProcessInfo)
	for _, entry := range entries {
		pid, parseErr := strconv.Atoi(entry.Name())
		if parseErr != nil || !entry.IsDir() {
			continue
		}
		statData, readErr := os.ReadFile(filepath.Join("/proc", entry.Name(), "stat"))
		if readErr != nil {
			continue
		}
		ppid, command, utime, stime, parseErr := parseProcessStat(string(statData))
		if parseErr != nil {
			continue
		}
		statusData, readErr := os.ReadFile(filepath.Join("/proc", entry.Name(), "status"))
		if readErr != nil {
			continue
		}
		rss, peak, threads, parseErr := parseStatus(string(statusData))
		if parseErr != nil {
			continue
		}
		all[pid] = ProcessInfo{PID: pid, PPID: ppid, Command: command, Role: classifyRole(pid, rootPID, command), CPUJiffies: utime + stime, RSSBytes: rss, PeakBytes: peak, Threads: threads}
	}
	if _, ok := all[rootPID]; !ok {
		return nil, fmt.Errorf("root process %d is not visible", rootPID)
	}
	children := make(map[int][]int)
	for pid, info := range all {
		children[info.PPID] = append(children[info.PPID], pid)
	}
	result := make([]ProcessInfo, 0, len(all))
	queue := []int{rootPID}
	seen := make(map[int]bool)
	for len(queue) > 0 {
		pid := queue[0]
		queue = queue[1:]
		if seen[pid] {
			continue
		}
		seen[pid] = true
		if info, ok := all[pid]; ok {
			result = append(result, info)
		}
		queue = append(queue, children[pid]...)
	}
	return result, nil
}

func parseProcessStat(content string) (ppid int, command string, utime, stime int64, err error) {
	closing := strings.LastIndex(content, ")")
	if closing < 0 || closing+2 >= len(content) {
		return 0, "", 0, 0, fmt.Errorf("malformed /proc/pid/stat")
	}
	command = strings.TrimPrefix(content[strings.Index(content, "(")+1:closing], "")
	fields := strings.Fields(content[closing+2:])
	if len(fields) < 13 {
		return 0, "", 0, 0, fmt.Errorf("malformed /proc/pid/stat fields")
	}
	ppid, err = strconv.Atoi(fields[1])
	if err != nil {
		return 0, "", 0, 0, fmt.Errorf("parse ppid: %w", err)
	}
	utime, err = strconv.ParseInt(fields[11], 10, 64)
	if err != nil {
		return 0, "", 0, 0, fmt.Errorf("parse utime: %w", err)
	}
	stime, err = strconv.ParseInt(fields[12], 10, 64)
	if err != nil {
		return 0, "", 0, 0, fmt.Errorf("parse stime: %w", err)
	}
	return ppid, command, utime, stime, nil
}

func classifyRole(pid, rootPID int, command string) ProcessRole {
	if pid == rootPID {
		return RoleElectronMain
	}
	lower := strings.ToLower(command)
	switch {
	case strings.Contains(lower, "gpu"):
		return RoleElectronGPU
	case strings.Contains(lower, "renderer"), strings.Contains(lower, "zygote"):
		return RoleElectronRender
	case strings.Contains(lower, "runtime"), strings.Contains(lower, "node"), strings.Contains(lower, "deno"), strings.Contains(lower, "bun"):
		return RoleBundledRuntime
	case lower != "":
		return RoleScenarioService
	default:
		return RoleUnknown
	}
}
