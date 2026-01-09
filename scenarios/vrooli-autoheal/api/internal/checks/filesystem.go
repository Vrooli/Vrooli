// Package checks provides filesystem abstractions for testability
// [REQ:TEST-SEAM-001] Filesystem testing seams for system checks
package checks

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
)

// StatfsResult contains filesystem statistics
type StatfsResult struct {
	Blocks  uint64 // Total blocks
	Bfree   uint64 // Free blocks
	Bavail  uint64 // Available blocks (non-root)
	Files   uint64 // Total inodes
	Ffree   uint64 // Free inodes
	Bsize   int64  // Block size
	Namemax uint64 // Max filename length
}

// FileSystemReader abstracts filesystem operations for testability.
// This interface allows system checks to be unit tested without
// actually accessing the filesystem.
type FileSystemReader interface {
	// Statfs returns filesystem statistics for the given path
	Statfs(path string) (*StatfsResult, error)
}

// RealFileSystemReader is the production implementation of FileSystemReader.
type RealFileSystemReader struct{}

// Statfs returns real filesystem statistics using syscall.
func (r *RealFileSystemReader) Statfs(path string) (*StatfsResult, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return nil, err
	}
	return &StatfsResult{
		Blocks:  stat.Blocks,
		Bfree:   stat.Bfree,
		Bavail:  stat.Bavail,
		Files:   stat.Files,
		Ffree:   stat.Ffree,
		Bsize:   stat.Bsize,
		Namemax: uint64(stat.Namelen),
	}, nil
}

// DefaultFileSystemReader is the global filesystem reader used when none is injected.
var DefaultFileSystemReader FileSystemReader = &RealFileSystemReader{}

// MemInfo contains memory information from /proc/meminfo
type MemInfo struct {
	MemTotal     uint64 // Total RAM in KB
	MemFree      uint64 // Free RAM in KB
	MemAvailable uint64 // Available RAM in KB (accounts for buffers/cache)
	Buffers      uint64 // Buffer memory in KB
	Cached       uint64 // Cached memory in KB
	SwapTotal    uint64 // Total swap in KB
	SwapFree     uint64 // Free swap in KB
}

// ProcessInfo contains information about a process
type ProcessInfo struct {
	PID       int
	State     string // R=running, S=sleeping, Z=zombie, etc.
	PPid      int    // Parent PID
	Comm      string // Command name
	StartTime uint64 // Process start time in clock ticks since boot
	Cmdline   string // Full command line from /proc/[pid]/cmdline
	Environ   map[string]string // Environment variables from /proc/[pid]/environ (lazy loaded)
}

// ProcReader abstracts /proc filesystem access for testability.
// This interface allows system checks to be unit tested without
// actually accessing /proc.
type ProcReader interface {
	// ReadMeminfo returns memory information from /proc/meminfo
	ReadMeminfo() (*MemInfo, error)
	// ListProcesses returns information about all processes
	ListProcesses() ([]ProcessInfo, error)
	// ReadProcessEnviron reads environment variables for a specific PID
	ReadProcessEnviron(pid int) (map[string]string, error)
	// ReadProcessCmdline reads the full command line for a specific PID
	ReadProcessCmdline(pid int) (string, error)
}

// RealProcReader is the production implementation of ProcReader.
type RealProcReader struct{}

// ReadMeminfo reads memory and swap information from /proc/meminfo.
func (r *RealProcReader) ReadMeminfo() (*MemInfo, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	info := &MemInfo{}
	scanner := bufio.NewScanner(file)
	var parseErrors []string

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		var val uint64
		var err error
		fieldName := fields[0]

		switch fieldName {
		case "MemTotal:":
			val, err = strconv.ParseUint(fields[1], 10, 64)
			if err == nil {
				info.MemTotal = val
			}
		case "MemFree:":
			val, err = strconv.ParseUint(fields[1], 10, 64)
			if err == nil {
				info.MemFree = val
			}
		case "MemAvailable:":
			val, err = strconv.ParseUint(fields[1], 10, 64)
			if err == nil {
				info.MemAvailable = val
			}
		case "Buffers:":
			val, err = strconv.ParseUint(fields[1], 10, 64)
			if err == nil {
				info.Buffers = val
			}
		case "Cached:":
			val, err = strconv.ParseUint(fields[1], 10, 64)
			if err == nil {
				info.Cached = val
			}
		case "SwapTotal:":
			val, err = strconv.ParseUint(fields[1], 10, 64)
			if err == nil {
				info.SwapTotal = val
			}
		case "SwapFree:":
			val, err = strconv.ParseUint(fields[1], 10, 64)
			if err == nil {
				info.SwapFree = val
			}
		default:
			continue
		}

		if err != nil {
			parseErrors = append(parseErrors, fieldName+" "+err.Error())
		}
	}

	if scanErr := scanner.Err(); scanErr != nil {
		return info, scanErr
	}

	// Return parse errors if any critical fields failed
	if info.MemTotal == 0 && len(parseErrors) > 0 {
		return info, fmt.Errorf("failed to parse meminfo: %s", strings.Join(parseErrors, "; "))
	}

	return info, nil
}

// ListProcesses reads process information from /proc.
func (r *RealProcReader) ListProcesses() ([]ProcessInfo, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}

	var processes []ProcessInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue // Not a PID directory
		}

		info, err := readProcessStat(pid)
		if err != nil {
			continue // Process may have exited
		}
		processes = append(processes, info)
	}

	return processes, nil
}

// readProcessStat reads /proc/[pid]/stat for a single process
// Format: pid (comm) state ppid pgrp session tty_nr tpgid flags minflt cminflt majflt cmajflt
//         utime stime cutime cstime priority nice num_threads itrealvalue starttime ...
// Fields are 1-indexed in documentation, but 0-indexed in the rest slice after we extract pid and comm.
// starttime is field 22 (1-indexed), which is rest[19] (state=rest[0], ppid=rest[1], ..., starttime=rest[19])
func readProcessStat(pid int) (ProcessInfo, error) {
	statPath := "/proc/" + strconv.Itoa(pid) + "/stat"
	data, err := os.ReadFile(statPath)
	if err != nil {
		return ProcessInfo{}, err
	}

	// Parse stat file: pid (comm) state ppid ...
	content := string(data)

	// Find comm (enclosed in parentheses, may contain spaces)
	start := strings.Index(content, "(")
	end := strings.LastIndex(content, ")")
	if start == -1 || end == -1 || end <= start {
		return ProcessInfo{PID: pid}, nil
	}

	comm := content[start+1 : end]
	rest := strings.Fields(content[end+2:]) // Skip ") "

	info := ProcessInfo{
		PID:  pid,
		Comm: comm,
	}

	if len(rest) >= 1 {
		info.State = rest[0]
	}
	if len(rest) >= 2 {
		ppid, err := strconv.Atoi(rest[1])
		if err == nil {
			info.PPid = ppid
		}
		// PPid parse failure is non-fatal - leave as 0
	}
	// starttime is at index 19 in rest (field 22 overall, minus pid, comm, then 0-indexed)
	if len(rest) >= 20 {
		startTime, err := strconv.ParseUint(rest[19], 10, 64)
		if err == nil {
			info.StartTime = startTime
		}
		// StartTime parse failure is non-fatal - leave as 0
	}

	return info, nil
}

// DefaultProcReader is the global proc reader used when none is injected.
var DefaultProcReader ProcReader = &RealProcReader{}

// ReadProcessEnviron reads environment variables from /proc/[pid]/environ.
// Environment variables are stored as null-separated KEY=VALUE pairs.
// Returns an error if the environ file cannot be read (e.g., permission denied).
func (r *RealProcReader) ReadProcessEnviron(pid int) (map[string]string, error) {
	environPath := "/proc/" + strconv.Itoa(pid) + "/environ"
	data, err := os.ReadFile(environPath)
	if err != nil {
		return nil, err
	}

	env := make(map[string]string)
	// Environment is null-separated
	for _, pair := range strings.Split(string(data), "\x00") {
		if pair == "" {
			continue
		}
		idx := strings.Index(pair, "=")
		if idx > 0 {
			key := pair[:idx]
			value := pair[idx+1:]
			env[key] = value
		}
	}

	return env, nil
}

// ReadProcessCmdline reads the full command line from /proc/[pid]/cmdline.
// Arguments are stored as null-separated strings.
func (r *RealProcReader) ReadProcessCmdline(pid int) (string, error) {
	cmdlinePath := "/proc/" + strconv.Itoa(pid) + "/cmdline"
	data, err := os.ReadFile(cmdlinePath)
	if err != nil {
		return "", err
	}

	// Replace null bytes with spaces for readability
	return strings.ReplaceAll(strings.TrimRight(string(data), "\x00"), "\x00", " "), nil
}

// IsVrooliManagedProcess checks if a process was started by Vrooli's lifecycle system
// by looking for the VROOLI_LIFECYCLE_MANAGED environment variable.
// This is the primary method for orphan detection.
func IsVrooliManagedProcess(pid int) bool {
	reader := DefaultProcReader
	env, err := reader.ReadProcessEnviron(pid)
	if err != nil {
		return false
	}
	// Check for the marker set by lifecycle.sh
	return env["VROOLI_LIFECYCLE_MANAGED"] == "true"
}

// GetVrooliProcessInfo returns Vrooli-specific info for a managed process.
// Returns nil if the process is not Vrooli-managed.
func GetVrooliProcessInfo(pid int) map[string]string {
	reader := DefaultProcReader
	env, err := reader.ReadProcessEnviron(pid)
	if err != nil {
		return nil
	}
	if env["VROOLI_LIFECYCLE_MANAGED"] != "true" {
		return nil
	}

	info := make(map[string]string)
	for _, key := range []string{"VROOLI_PROCESS_ID", "VROOLI_PHASE", "VROOLI_SCENARIO", "VROOLI_STEP"} {
		if val, ok := env[key]; ok {
			info[key] = val
		}
	}
	return info
}

// getBootTime reads the system boot time from /proc/stat
// Returns boot time as Unix timestamp in seconds
func getBootTime() (uint64, error) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0, err
	}

	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "btime ") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				return strconv.ParseUint(fields[1], 10, 64)
			}
		}
	}
	return 0, fmt.Errorf("btime not found in /proc/stat")
}

// getClockTicksPerSecond returns the system's clock ticks per second (usually 100)
func getClockTicksPerSecond() uint64 {
	// On Linux, this is typically 100 (USER_HZ)
	// Could use syscall.Sysconf(_SC_CLK_TCK) but that requires cgo
	return 100
}

// ProcessAge calculates the age of a process in seconds.
// Returns 0 if the age cannot be determined.
func ProcessAge(startTime uint64) float64 {
	if startTime == 0 {
		return 0
	}

	bootTime, err := getBootTime()
	if err != nil {
		return 0
	}

	clockTicks := getClockTicksPerSecond()
	// startTime is in clock ticks since boot
	startTimeSecs := float64(startTime) / float64(clockTicks)
	// Convert to Unix timestamp
	processStartUnix := float64(bootTime) + startTimeSecs

	// Current time
	now := float64(unixNow())

	age := now - processStartUnix
	if age < 0 {
		return 0
	}
	return age
}

// unixNow returns the current Unix timestamp in seconds.
// Extracted for potential testability.
func unixNow() int64 {
	var t syscall.Timeval
	_ = syscall.Gettimeofday(&t)
	return t.Sec
}

// PortInfo contains information about port usage
type PortInfo struct {
	UsedPorts   int
	TotalPorts  int
	UsedPercent int
	TimeWait    int // Ports in TIME_WAIT state
}

// PortReader abstracts port statistics reading for testability.
type PortReader interface {
	// ReadPortStats returns port usage statistics
	ReadPortStats() (*PortInfo, error)
}

// RealPortReader is the production implementation of PortReader.
type RealPortReader struct{}

// ReadPortStats reads port usage from /proc/sys/net/ipv4/ip_local_port_range
// and counts used ports.
func (r *RealPortReader) ReadPortStats() (*PortInfo, error) {
	// Read ephemeral port range
	rangeData, err := os.ReadFile("/proc/sys/net/ipv4/ip_local_port_range")
	if err != nil {
		return nil, err
	}

	fields := strings.Fields(string(rangeData))
	if len(fields) < 2 {
		return &PortInfo{TotalPorts: 28232}, nil // Default range
	}

	lowPort, err := strconv.Atoi(fields[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse low port %q: %w", fields[0], err)
	}
	highPort, err := strconv.Atoi(fields[1])
	if err != nil {
		return nil, fmt.Errorf("failed to parse high port %q: %w", fields[1], err)
	}
	if highPort < lowPort {
		return nil, fmt.Errorf("invalid port range: high port %d < low port %d", highPort, lowPort)
	}
	totalPorts := highPort - lowPort + 1

	// Count used ports from /proc/net/tcp and /proc/net/tcp6
	usedPorts := 0
	timeWait := 0

	for _, path := range []string{"/proc/net/tcp", "/proc/net/tcp6"} {
		used, tw := countPortsFromFile(path)
		usedPorts += used
		timeWait += tw
	}

	usedPercent := 0
	if totalPorts > 0 {
		usedPercent = (usedPorts * 100) / totalPorts
	}

	return &PortInfo{
		UsedPorts:   usedPorts,
		TotalPorts:  totalPorts,
		UsedPercent: usedPercent,
		TimeWait:    timeWait,
	}, nil
}

// countPortsFromFile counts ports from a /proc/net/tcp* file
func countPortsFromFile(path string) (used, timeWait int) {
	file, err := os.Open(path)
	if err != nil {
		return 0, 0
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Scan() // Skip header

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 4 {
			continue
		}
		used++
		// State 06 = TIME_WAIT
		if fields[3] == "06" {
			timeWait++
		}
	}

	return used, timeWait
}

// DefaultPortReader is the global port reader used when none is injected.
var DefaultPortReader PortReader = &RealPortReader{}

// CacheChecker abstracts cache directory checking for testability.
type CacheChecker interface {
	// GetCacheSize returns the size of the cache directory in bytes
	GetCacheSize(path string) (uint64, error)
	// CleanCache removes files from the cache directory
	CleanCache(path string, maxAge int) (int, uint64, error)
}

// RealCacheChecker is the production implementation of CacheChecker.
type RealCacheChecker struct{}

// GetCacheSize returns the total size of files in the directory.
func (c *RealCacheChecker) GetCacheSize(path string) (uint64, error) {
	var size uint64

	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		size += uint64(info.Size())
	}

	return size, nil
}

// CleanCache removes files older than maxAge days.
// Returns count of files removed and bytes freed.
func (c *RealCacheChecker) CleanCache(path string, maxAgeDays int) (int, uint64, error) {
	// This is a placeholder - actual implementation would use os.Remove
	// But we don't want auto-cleanup without explicit user action
	return 0, 0, nil
}

// DefaultCacheChecker is the global cache checker used when none is injected.
var DefaultCacheChecker CacheChecker = &RealCacheChecker{}
