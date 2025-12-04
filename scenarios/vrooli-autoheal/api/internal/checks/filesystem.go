// Package checks provides filesystem abstractions for testability
// [REQ:TEST-SEAM-001] Filesystem testing seams for system checks
package checks

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"syscall"
)

// StatfsResult contains filesystem statistics
type StatfsResult struct {
	Blocks     uint64 // Total blocks
	Bfree      uint64 // Free blocks
	Bavail     uint64 // Available blocks (non-root)
	Files      uint64 // Total inodes
	Ffree      uint64 // Free inodes
	Bsize      int64  // Block size
	Namemax    uint64 // Max filename length
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
	SwapTotal uint64 // Total swap in KB
	SwapFree  uint64 // Free swap in KB
}

// ProcessInfo contains information about a process
type ProcessInfo struct {
	PID    int
	State  string // R=running, S=sleeping, Z=zombie, etc.
	PPid   int    // Parent PID
	Comm   string // Command name
}

// ProcReader abstracts /proc filesystem access for testability.
// This interface allows system checks to be unit tested without
// actually accessing /proc.
type ProcReader interface {
	// ReadMeminfo returns memory information from /proc/meminfo
	ReadMeminfo() (*MemInfo, error)
	// ListProcesses returns information about all processes
	ListProcesses() ([]ProcessInfo, error)
}

// RealProcReader is the production implementation of ProcReader.
type RealProcReader struct{}

// ReadMeminfo reads swap information from /proc/meminfo.
func (r *RealProcReader) ReadMeminfo() (*MemInfo, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	info := &MemInfo{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		switch fields[0] {
		case "SwapTotal:":
			info.SwapTotal, _ = strconv.ParseUint(fields[1], 10, 64)
		case "SwapFree:":
			info.SwapFree, _ = strconv.ParseUint(fields[1], 10, 64)
		}
	}

	return info, scanner.Err()
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
		info.PPid, _ = strconv.Atoi(rest[1])
	}

	return info, nil
}

// DefaultProcReader is the global proc reader used when none is injected.
var DefaultProcReader ProcReader = &RealProcReader{}

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

	lowPort, _ := strconv.Atoi(fields[0])
	highPort, _ := strconv.Atoi(fields[1])
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
