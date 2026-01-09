// Package checks tests for real filesystem implementations
// [REQ:TEST-SEAM-001] Verify real filesystem abstractions work correctly
package checks

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// =============================================================================
// RealFileSystemReader Tests
// =============================================================================

// TestRealFileSystemReaderStatfs verifies Statfs on root filesystem
func TestRealFileSystemReaderStatfs(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping filesystem test on non-Linux platform")
	}

	reader := &RealFileSystemReader{}

	result, err := reader.Statfs("/")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("result should not be nil")
	}
	if result.Blocks == 0 {
		t.Error("Blocks should be > 0")
	}
	if result.Bsize == 0 {
		t.Error("Bsize should be > 0")
	}
}

// TestRealFileSystemReaderStatfsError verifies error handling
func TestRealFileSystemReaderStatfsError(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping filesystem test on non-Linux platform")
	}

	reader := &RealFileSystemReader{}

	_, err := reader.Statfs("/nonexistent/path/12345")

	if err == nil {
		t.Error("expected error for nonexistent path")
	}
}

// TestDefaultFileSystemReaderExists verifies DefaultFileSystemReader is set
func TestDefaultFileSystemReaderExists(t *testing.T) {
	if DefaultFileSystemReader == nil {
		t.Error("DefaultFileSystemReader should not be nil")
	}
}

// =============================================================================
// RealProcReader Tests
// =============================================================================

// TestRealProcReaderReadMeminfo verifies reading /proc/meminfo
func TestRealProcReaderReadMeminfo(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping /proc test on non-Linux platform")
	}

	reader := &RealProcReader{}

	info, err := reader.ReadMeminfo()

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if info == nil {
		t.Fatal("info should not be nil")
	}
	// Note: SwapTotal could be 0 on systems without swap
	t.Logf("SwapTotal: %d KB, SwapFree: %d KB", info.SwapTotal, info.SwapFree)
}

// TestRealProcReaderListProcesses verifies listing processes
func TestRealProcReaderListProcesses(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping /proc test on non-Linux platform")
	}

	reader := &RealProcReader{}

	procs, err := reader.ListProcesses()

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(procs) == 0 {
		t.Error("expected at least one process")
	}

	// Find PID 1 (init/systemd)
	found := false
	for _, p := range procs {
		if p.PID == 1 {
			found = true
			t.Logf("PID 1: comm=%q state=%s", p.Comm, p.State)
			break
		}
	}
	if !found {
		t.Log("PID 1 not found (may be in container)")
	}
}

// TestReadProcessStat verifies reading a single process stat
func TestReadProcessStat(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping /proc test on non-Linux platform")
	}

	// Read current process
	pid := os.Getpid()
	info, err := readProcessStat(pid)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if info.PID != pid {
		t.Errorf("PID = %d, want %d", info.PID, pid)
	}
	if info.State == "" {
		t.Error("State should not be empty")
	}
	t.Logf("Current process: comm=%q state=%s ppid=%d", info.Comm, info.State, info.PPid)
}

// TestReadProcessStatError verifies error handling
func TestReadProcessStatError(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping /proc test on non-Linux platform")
	}

	_, err := readProcessStat(999999999)

	if err == nil {
		t.Error("expected error for nonexistent PID")
	}
}

// TestDefaultProcReaderExists verifies DefaultProcReader is set
func TestDefaultProcReaderExists(t *testing.T) {
	if DefaultProcReader == nil {
		t.Error("DefaultProcReader should not be nil")
	}
}

// =============================================================================
// RealPortReader Tests
// =============================================================================

// TestRealPortReaderReadPortStats verifies reading port stats
func TestRealPortReaderReadPortStats(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping /proc test on non-Linux platform")
	}

	reader := &RealPortReader{}

	info, err := reader.ReadPortStats()

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if info == nil {
		t.Fatal("info should not be nil")
	}
	if info.TotalPorts == 0 {
		t.Error("TotalPorts should be > 0")
	}
	t.Logf("Ports: used=%d total=%d percent=%d%% timewait=%d",
		info.UsedPorts, info.TotalPorts, info.UsedPercent, info.TimeWait)
}

// TestCountPortsFromFile verifies counting ports from /proc/net/tcp
func TestCountPortsFromFile(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping /proc test on non-Linux platform")
	}

	used, timeWait := countPortsFromFile("/proc/net/tcp")

	// Should get at least some connections (loopback, etc.)
	t.Logf("/proc/net/tcp: used=%d timewait=%d", used, timeWait)
}

// TestCountPortsFromFileNotExist verifies handling of missing file
func TestCountPortsFromFileNotExist(t *testing.T) {
	used, timeWait := countPortsFromFile("/nonexistent/file")

	if used != 0 || timeWait != 0 {
		t.Errorf("expected 0,0 for nonexistent file, got %d,%d", used, timeWait)
	}
}

// TestDefaultPortReaderExists verifies DefaultPortReader is set
func TestDefaultPortReaderExists(t *testing.T) {
	if DefaultPortReader == nil {
		t.Error("DefaultPortReader should not be nil")
	}
}

// =============================================================================
// RealCacheChecker Tests
// =============================================================================

// TestRealCacheCheckerGetCacheSize verifies getting directory size
func TestRealCacheCheckerGetCacheSize(t *testing.T) {
	// Create temp directory with some files
	tmpDir := t.TempDir()

	// Create test files
	if err := os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("hello"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("world!!"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	checker := &RealCacheChecker{}

	size, err := checker.GetCacheSize(tmpDir)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// "hello" (5 bytes) + "world!!" (7 bytes) = 12 bytes
	if size != 12 {
		t.Errorf("size = %d, want 12", size)
	}
}

// TestRealCacheCheckerGetCacheSizeNotExist verifies handling of missing dir
func TestRealCacheCheckerGetCacheSizeNotExist(t *testing.T) {
	checker := &RealCacheChecker{}

	size, err := checker.GetCacheSize("/nonexistent/path/12345")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if size != 0 {
		t.Errorf("size = %d, want 0 for nonexistent path", size)
	}
}

// TestRealCacheCheckerGetCacheSizeWithSubdir verifies subdirs are not counted
func TestRealCacheCheckerGetCacheSizeWithSubdir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file
	if err := os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create a subdirectory (should not be counted)
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "nested.txt"), []byte("nested"), 0644); err != nil {
		t.Fatalf("failed to create nested file: %v", err)
	}

	checker := &RealCacheChecker{}

	size, err := checker.GetCacheSize(tmpDir)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// Only "test" (4 bytes), not nested file
	if size != 4 {
		t.Errorf("size = %d, want 4 (excluding subdir)", size)
	}
}

// TestRealCacheCheckerCleanCache verifies cleanup placeholder
func TestRealCacheCheckerCleanCache(t *testing.T) {
	tmpDir := t.TempDir()
	checker := &RealCacheChecker{}

	count, size, err := checker.CleanCache(tmpDir, 30)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// Placeholder returns 0, 0, nil
	if count != 0 || size != 0 {
		t.Errorf("CleanCache = (%d, %d), want (0, 0)", count, size)
	}
}

// TestDefaultCacheCheckerExists verifies DefaultCacheChecker is set
func TestDefaultCacheCheckerExists(t *testing.T) {
	if DefaultCacheChecker == nil {
		t.Error("DefaultCacheChecker should not be nil")
	}
}

// =============================================================================
// StatfsResult Tests
// =============================================================================

// TestStatfsResultFields verifies struct fields
func TestStatfsResultFields(t *testing.T) {
	result := StatfsResult{
		Blocks:  1000000,
		Bfree:   500000,
		Bavail:  400000,
		Files:   100000,
		Ffree:   80000,
		Bsize:   4096,
		Namemax: 255,
	}

	if result.Blocks != 1000000 {
		t.Errorf("Blocks = %d, want 1000000", result.Blocks)
	}
	if result.Bsize != 4096 {
		t.Errorf("Bsize = %d, want 4096", result.Bsize)
	}
}

// =============================================================================
// MemInfo Tests
// =============================================================================

// TestMemInfoFields verifies struct fields
func TestMemInfoFields(t *testing.T) {
	info := MemInfo{
		SwapTotal: 8388608,
		SwapFree:  6291456,
	}

	if info.SwapTotal != 8388608 {
		t.Errorf("SwapTotal = %d, want 8388608", info.SwapTotal)
	}
	if info.SwapFree != 6291456 {
		t.Errorf("SwapFree = %d, want 6291456", info.SwapFree)
	}
}

// =============================================================================
// ProcessInfo Tests
// =============================================================================

// TestProcessInfoFields verifies struct fields
func TestProcessInfoFields(t *testing.T) {
	info := ProcessInfo{
		PID:   1234,
		State: "S",
		PPid:  1,
		Comm:  "test",
	}

	if info.PID != 1234 {
		t.Errorf("PID = %d, want 1234", info.PID)
	}
	if info.State != "S" {
		t.Errorf("State = %q, want %q", info.State, "S")
	}
}

// =============================================================================
// PortInfo Tests
// =============================================================================

// TestPortInfoFields verifies struct fields
func TestPortInfoFields(t *testing.T) {
	info := PortInfo{
		UsedPorts:   1000,
		TotalPorts:  28232,
		UsedPercent: 3,
		TimeWait:    50,
	}

	if info.UsedPorts != 1000 {
		t.Errorf("UsedPorts = %d, want 1000", info.UsedPorts)
	}
	if info.UsedPercent != 3 {
		t.Errorf("UsedPercent = %d, want 3", info.UsedPercent)
	}
}
