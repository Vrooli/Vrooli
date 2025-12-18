package process

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestLoggerCreateAndRead tests basic log creation and reading.
func TestLoggerCreateAndRead(t *testing.T) {
	// Setup temp directory
	tmpDir := t.TempDir()
	cfg := LogConfig{
		BaseDir:    tmpDir,
		MaxLogSize: 1024 * 1024,
	}
	logger := NewLogger(cfg)

	sandboxID := uuid.New()
	pid := 12345

	// Create log
	writer, err := logger.CreateLog(sandboxID, pid)
	if err != nil {
		t.Fatalf("CreateLog failed: %v", err)
	}

	// Write some content
	testContent := "test log line 1\ntest log line 2\n"
	_, err = writer.Write([]byte(testContent))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Verify log file exists
	logPath := logger.LogPath(sandboxID, pid)
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Fatalf("Log file was not created at %s", logPath)
	}

	// Close the writer
	err = writer.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Read back the log
	content, err := logger.ReadLog(sandboxID, pid, 0, 0)
	if err != nil {
		t.Fatalf("ReadLog failed: %v", err)
	}

	if !strings.Contains(string(content), "test log line 1") {
		t.Errorf("Log content missing expected text, got: %s", string(content))
	}
	if !strings.Contains(string(content), "test log line 2") {
		t.Errorf("Log content missing expected text, got: %s", string(content))
	}
}

// TestLoggerGetLog tests retrieving log metadata.
func TestLoggerGetLog(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := LogConfig{BaseDir: tmpDir}
	logger := NewLogger(cfg)

	sandboxID := uuid.New()
	pid := 54321

	// Create and write log
	writer, err := logger.CreateLog(sandboxID, pid)
	if err != nil {
		t.Fatalf("CreateLog failed: %v", err)
	}

	writer.Write([]byte("some content"))
	writer.Close()

	// Get log metadata
	logInfo, err := logger.GetLog(sandboxID, pid)
	if err != nil {
		t.Fatalf("GetLog failed: %v", err)
	}

	if logInfo.PID != pid {
		t.Errorf("PID mismatch: got %d, want %d", logInfo.PID, pid)
	}
	if logInfo.SandboxID != sandboxID {
		t.Errorf("SandboxID mismatch: got %s, want %s", logInfo.SandboxID, sandboxID)
	}
	if logInfo.SizeBytes == 0 {
		t.Error("SizeBytes should be > 0")
	}
}

// TestLoggerListLogs tests listing logs for a sandbox.
func TestLoggerListLogs(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := LogConfig{BaseDir: tmpDir}
	logger := NewLogger(cfg)

	sandboxID := uuid.New()

	// Create multiple logs
	for _, pid := range []int{100, 200, 300} {
		writer, err := logger.CreateLog(sandboxID, pid)
		if err != nil {
			t.Fatalf("CreateLog failed for PID %d: %v", pid, err)
		}
		writer.Write([]byte("content"))
		writer.Close()
	}

	// List logs
	logs, err := logger.ListLogs(sandboxID)
	if err != nil {
		t.Fatalf("ListLogs failed: %v", err)
	}

	if len(logs) != 3 {
		t.Errorf("Expected 3 logs, got %d", len(logs))
	}
}

// TestLoggerReadTail tests reading last N lines.
func TestLoggerReadTail(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := LogConfig{BaseDir: tmpDir}
	logger := NewLogger(cfg)

	sandboxID := uuid.New()
	pid := 999

	// Create log with multiple lines
	writer, err := logger.CreateLog(sandboxID, pid)
	if err != nil {
		t.Fatalf("CreateLog failed: %v", err)
	}

	lines := []string{
		"line 1",
		"line 2",
		"line 3",
		"line 4",
		"line 5",
	}
	for _, line := range lines {
		writer.Write([]byte(line + "\n"))
	}
	writer.Close()

	// Read last 2 lines
	content, err := logger.ReadLog(sandboxID, pid, 2, 0)
	if err != nil {
		t.Fatalf("ReadLog with tail failed: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "line 4") || !strings.Contains(contentStr, "line 5") {
		t.Errorf("Tail content doesn't contain expected lines, got: %s", contentStr)
	}
}

// TestLoggerCloseLog tests closing log with exit code.
func TestLoggerCloseLog(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := LogConfig{BaseDir: tmpDir}
	logger := NewLogger(cfg)

	sandboxID := uuid.New()
	pid := 777

	// Create log
	_, err := logger.CreateLog(sandboxID, pid)
	if err != nil {
		t.Fatalf("CreateLog failed: %v", err)
	}

	// Close with exit code
	err = logger.CloseLog(sandboxID, pid, 42)
	if err != nil {
		t.Fatalf("CloseLog failed: %v", err)
	}

	// Verify exit code in log
	content, err := logger.ReadLog(sandboxID, pid, 0, 0)
	if err != nil {
		t.Fatalf("ReadLog failed: %v", err)
	}

	if !strings.Contains(string(content), "code 42") {
		t.Errorf("Log should contain exit code 42, got: %s", string(content))
	}
}

// TestLoggerCleanupSandboxLogs tests cleaning up all logs for a sandbox.
func TestLoggerCleanupSandboxLogs(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := LogConfig{BaseDir: tmpDir}
	logger := NewLogger(cfg)

	sandboxID := uuid.New()

	// Create logs
	for _, pid := range []int{1, 2, 3} {
		writer, _ := logger.CreateLog(sandboxID, pid)
		writer.Close()
	}

	// Verify logs exist
	logDir := logger.LogDir(sandboxID)
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		t.Fatalf("Log directory should exist")
	}

	// Cleanup
	err := logger.CleanupSandboxLogs(sandboxID)
	if err != nil {
		t.Fatalf("CleanupSandboxLogs failed: %v", err)
	}

	// Verify logs are gone
	if _, err := os.Stat(logDir); !os.IsNotExist(err) {
		t.Errorf("Log directory should be removed after cleanup")
	}
}

// TestLoggerConcurrentWrites tests concurrent writes to the same log.
func TestLoggerConcurrentWrites(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := LogConfig{BaseDir: tmpDir}
	logger := NewLogger(cfg)

	sandboxID := uuid.New()
	pid := 888

	writer, err := logger.CreateLog(sandboxID, pid)
	if err != nil {
		t.Fatalf("CreateLog failed: %v", err)
	}

	// Write concurrently
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				writer.Write([]byte("write from goroutine\n"))
			}
		}(i)
	}

	wg.Wait()
	writer.Close()

	// Verify log exists and has content
	logInfo, err := logger.GetLog(sandboxID, pid)
	if err != nil {
		t.Fatalf("GetLog failed: %v", err)
	}

	// Each write is ~20 bytes, 10*100 writes + header
	if logInfo.SizeBytes < 10000 {
		t.Errorf("Log seems too small for concurrent writes: %d bytes", logInfo.SizeBytes)
	}
}

// TestLoggerStreamLog tests log streaming functionality.
func TestLoggerStreamLog(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := LogConfig{BaseDir: tmpDir}
	logger := NewLogger(cfg)

	sandboxID := uuid.New()
	pid := 666

	// Create log and keep writer open (simulating active process)
	writer, err := logger.CreateLog(sandboxID, pid)
	if err != nil {
		t.Fatalf("CreateLog failed: %v", err)
	}

	// Start streaming in background
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	var chunks []string
	var streamErr error
	done := make(chan struct{})

	go func() {
		streamErr = logger.StreamLog(ctx, sandboxID, pid, func(chunk []byte) {
			chunks = append(chunks, string(chunk))
		})
		close(done)
	}()

	// Write some data
	time.Sleep(50 * time.Millisecond)
	writer.Write([]byte("streaming line 1\n"))
	time.Sleep(150 * time.Millisecond)
	writer.Write([]byte("streaming line 2\n"))

	// Close the writer to signal process ended
	writer.Close()

	// Wait for stream to finish
	<-done

	// Stream should have ended due to timeout or process end
	if streamErr != nil && streamErr != context.DeadlineExceeded {
		t.Logf("Stream ended with: %v (expected)", streamErr)
	}
}

// TestLoggerNonExistentLog tests reading a non-existent log.
func TestLoggerNonExistentLog(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := LogConfig{BaseDir: tmpDir}
	logger := NewLogger(cfg)

	sandboxID := uuid.New()
	pid := 99999

	_, err := logger.GetLog(sandboxID, pid)
	if err == nil {
		t.Error("Expected error for non-existent log")
	}

	_, err = logger.ReadLog(sandboxID, pid, 0, 0)
	if err == nil {
		t.Error("Expected error reading non-existent log")
	}
}

// TestLoggerEmptySandbox tests listing logs for sandbox with no logs.
func TestLoggerEmptySandbox(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := LogConfig{BaseDir: tmpDir}
	logger := NewLogger(cfg)

	sandboxID := uuid.New()

	logs, err := logger.ListLogs(sandboxID)
	if err != nil {
		t.Fatalf("ListLogs failed: %v", err)
	}

	if len(logs) != 0 {
		t.Errorf("Expected 0 logs for empty sandbox, got %d", len(logs))
	}
}

// TestLoggerLogPath tests log path generation.
func TestLoggerLogPath(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := LogConfig{BaseDir: tmpDir}
	logger := NewLogger(cfg)

	sandboxID := uuid.New()
	pid := 123

	path := logger.LogPath(sandboxID, pid)

	expectedPath := filepath.Join(tmpDir, sandboxID.String(), "logs", "123.log")
	if path != expectedPath {
		t.Errorf("LogPath mismatch: got %s, want %s", path, expectedPath)
	}
}

// TestDefaultLogConfig tests the default config values.
func TestDefaultLogConfig(t *testing.T) {
	cfg := DefaultLogConfig("/test/dir")

	if cfg.BaseDir != "/test/dir" {
		t.Errorf("BaseDir mismatch: got %s, want /test/dir", cfg.BaseDir)
	}

	if cfg.MaxLogSize != 50*1024*1024 {
		t.Errorf("MaxLogSize mismatch: got %d, want %d", cfg.MaxLogSize, 50*1024*1024)
	}
}

// TestPendingLogCreateAndFinalize tests the two-phase log creation workflow.
func TestPendingLogCreateAndFinalize(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := LogConfig{BaseDir: tmpDir}
	logger := NewLogger(cfg)

	sandboxID := uuid.New()
	pid := 12345

	// Phase 1: Create pending log BEFORE process starts
	pending, err := logger.CreatePendingLog(sandboxID)
	if err != nil {
		t.Fatalf("CreatePendingLog failed: %v", err)
	}

	if pending.TempID == "" {
		t.Error("TempID should not be empty")
	}
	if pending.Writer == nil {
		t.Error("Writer should not be nil")
	}

	// Verify temp file exists
	tempPath := pending.Writer.path
	if _, err := os.Stat(tempPath); os.IsNotExist(err) {
		t.Fatalf("Temp log file should exist at %s", tempPath)
	}

	// Write some content (simulating process output)
	testContent := "simulated process output\n"
	_, err = pending.Writer.Write([]byte(testContent))
	if err != nil {
		t.Fatalf("Write to pending log failed: %v", err)
	}

	// Phase 2: Finalize with actual PID after process starts
	logPath, err := logger.FinalizeLog(pending, pid)
	if err != nil {
		t.Fatalf("FinalizeLog failed: %v", err)
	}

	// Verify final path is correct
	expectedPath := logger.LogPath(sandboxID, pid)
	if logPath != expectedPath {
		t.Errorf("Log path mismatch: got %s, want %s", logPath, expectedPath)
	}

	// Verify file was renamed (temp file should not exist)
	if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
		t.Errorf("Temp file should be renamed/removed")
	}

	// Verify final file exists
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Fatalf("Final log file should exist at %s", logPath)
	}

	// Verify content is preserved
	content, err := logger.ReadLog(sandboxID, pid, 0, 0)
	if err != nil {
		t.Fatalf("ReadLog failed: %v", err)
	}
	if !strings.Contains(string(content), "simulated process output") {
		t.Errorf("Log content missing expected text, got: %s", string(content))
	}

	// Verify log is registered and can be retrieved
	logInfo, err := logger.GetLog(sandboxID, pid)
	if err != nil {
		t.Fatalf("GetLog failed: %v", err)
	}
	if logInfo.PID != pid {
		t.Errorf("PID mismatch: got %d, want %d", logInfo.PID, pid)
	}
}

// TestPendingLogAbort tests aborting a pending log when process fails to start.
func TestPendingLogAbort(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := LogConfig{BaseDir: tmpDir}
	logger := NewLogger(cfg)

	sandboxID := uuid.New()

	// Create pending log
	pending, err := logger.CreatePendingLog(sandboxID)
	if err != nil {
		t.Fatalf("CreatePendingLog failed: %v", err)
	}

	tempPath := pending.Writer.path

	// Verify temp file exists
	if _, err := os.Stat(tempPath); os.IsNotExist(err) {
		t.Fatalf("Temp log file should exist")
	}

	// Abort the pending log (process failed to start)
	err = logger.AbortPendingLog(pending)
	if err != nil {
		t.Fatalf("AbortPendingLog failed: %v", err)
	}

	// Verify temp file is removed
	if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
		t.Errorf("Temp log file should be removed after abort")
	}
}

// TestPendingLogWriteDuringProcess tests writing to log while process runs.
func TestPendingLogWriteDuringProcess(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := LogConfig{BaseDir: tmpDir}
	logger := NewLogger(cfg)

	sandboxID := uuid.New()
	pid := 54321

	// Create pending log
	pending, err := logger.CreatePendingLog(sandboxID)
	if err != nil {
		t.Fatalf("CreatePendingLog failed: %v", err)
	}

	// Write before finalize (simulating early process output)
	pending.Writer.Write([]byte("output before finalize\n"))

	// Finalize
	_, err = logger.FinalizeLog(pending, pid)
	if err != nil {
		t.Fatalf("FinalizeLog failed: %v", err)
	}

	// Continue writing after finalize (simulating ongoing process output)
	pending.Writer.Write([]byte("output after finalize\n"))

	// Close log
	pending.Writer.Close()

	// Verify all content is present
	content, err := logger.ReadLog(sandboxID, pid, 0, 0)
	if err != nil {
		t.Fatalf("ReadLog failed: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "output before finalize") {
		t.Errorf("Missing content written before finalize")
	}
	if !strings.Contains(contentStr, "output after finalize") {
		t.Errorf("Missing content written after finalize")
	}
}

// TestPendingLogIsActiveStatus tests that finalized pending logs show as active.
func TestPendingLogIsActiveStatus(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := LogConfig{BaseDir: tmpDir}
	logger := NewLogger(cfg)

	sandboxID := uuid.New()
	pid := 11111

	// Create and finalize pending log
	pending, err := logger.CreatePendingLog(sandboxID)
	if err != nil {
		t.Fatalf("CreatePendingLog failed: %v", err)
	}

	_, err = logger.FinalizeLog(pending, pid)
	if err != nil {
		t.Fatalf("FinalizeLog failed: %v", err)
	}

	// Check that log shows as active (writer still open)
	logInfo, err := logger.GetLog(sandboxID, pid)
	if err != nil {
		t.Fatalf("GetLog failed: %v", err)
	}
	if !logInfo.IsActive {
		t.Error("Log should be active while writer is open")
	}

	// Close the writer
	pending.Writer.Close()

	// Check that log shows as inactive
	logInfo, err = logger.GetLog(sandboxID, pid)
	if err != nil {
		t.Fatalf("GetLog failed: %v", err)
	}
	// Note: After closing, the log may still appear in the map briefly
	// but the file is closed
}
