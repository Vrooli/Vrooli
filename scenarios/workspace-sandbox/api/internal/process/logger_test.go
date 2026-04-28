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

// withPair is a small helper that creates a pending log pair, finalises it
// against a synthetic PID, and returns the registered writers + tracked
// paths so individual tests don't repeat the boilerplate.
func withPair(t *testing.T, logger *Logger, sandboxID uuid.UUID, pid int) (*PendingLogPair, string, string) {
	t.Helper()
	pending, err := logger.CreatePendingLogPair(sandboxID)
	if err != nil {
		t.Fatalf("CreatePendingLogPair: %v", err)
	}
	stdoutPath, stderrPath, err := logger.FinalizePair(pending, pid)
	if err != nil {
		t.Fatalf("FinalizePair: %v", err)
	}
	return pending, stdoutPath, stderrPath
}

// TestLogger_CreateAndReadStreams verifies that stdout and stderr are
// kept on separate disk files and that ReadLog can return either.
func TestLogger_CreateAndReadStreams(t *testing.T) {
	tmpDir := t.TempDir()
	logger := NewLogger(LogConfig{BaseDir: tmpDir})

	sandboxID := uuid.New()
	pid := 12345

	pending, stdoutPath, stderrPath := withPair(t, logger, sandboxID, pid)

	if _, err := pending.Stdout.Write([]byte("stdout content\n")); err != nil {
		t.Fatalf("write stdout: %v", err)
	}
	if _, err := pending.Stderr.Write([]byte("stderr content\n")); err != nil {
		t.Fatalf("write stderr: %v", err)
	}

	if _, err := os.Stat(stdoutPath); err != nil {
		t.Errorf("stdout file missing: %v", err)
	}
	if _, err := os.Stat(stderrPath); err != nil {
		t.Errorf("stderr file missing: %v", err)
	}
	if stdoutPath == stderrPath {
		t.Errorf("expected separate paths, got same: %s", stdoutPath)
	}

	stdoutContent, err := logger.ReadLog(sandboxID, pid, StreamStdout, 0, 0)
	if err != nil {
		t.Fatalf("ReadLog stdout: %v", err)
	}
	if !strings.Contains(string(stdoutContent), "stdout content") {
		t.Errorf("stdout content missing")
	}
	if strings.Contains(string(stdoutContent), "stderr content") {
		t.Errorf("stdout file contains stderr text — streams not separated")
	}

	stderrContent, err := logger.ReadLog(sandboxID, pid, StreamStderr, 0, 0)
	if err != nil {
		t.Fatalf("ReadLog stderr: %v", err)
	}
	if !strings.Contains(string(stderrContent), "stderr content") {
		t.Errorf("stderr content missing")
	}
}

// TestLogger_RejectsInvalidStream verifies the Stream type guard.
func TestLogger_RejectsInvalidStream(t *testing.T) {
	tmpDir := t.TempDir()
	logger := NewLogger(LogConfig{BaseDir: tmpDir})

	sandboxID := uuid.New()
	if _, err := logger.GetLog(sandboxID, 1, "stdoutXXX"); err == nil {
		t.Error("GetLog with invalid stream should fail")
	}
	if _, err := logger.ReadLog(sandboxID, 1, "merged", 0, 0); err == nil {
		t.Error("ReadLog with invalid stream should fail")
	}
}

// TestLogger_GetLogReportsBothStreams confirms ListLogs returns both
// stdout and stderr per process.
func TestLogger_GetLogReportsBothStreams(t *testing.T) {
	tmpDir := t.TempDir()
	logger := NewLogger(LogConfig{BaseDir: tmpDir})

	sandboxID := uuid.New()
	pid := 54321
	pending, _, _ := withPair(t, logger, sandboxID, pid)
	_, _ = pending.Stdout.Write([]byte("o"))
	_, _ = pending.Stderr.Write([]byte("e"))

	logs, err := logger.ListLogs(sandboxID)
	if err != nil {
		t.Fatalf("ListLogs: %v", err)
	}
	if len(logs) != 2 {
		t.Fatalf("expected 2 logs (stdout + stderr); got %d", len(logs))
	}
	streams := map[Stream]bool{}
	for _, l := range logs {
		streams[l.Stream] = true
	}
	if !streams[StreamStdout] || !streams[StreamStderr] {
		t.Errorf("missing stream in ListLogs output: %v", streams)
	}
}

// TestLogger_LogPathSeparatesStreams pins the on-disk naming convention.
func TestLogger_LogPathSeparatesStreams(t *testing.T) {
	tmpDir := t.TempDir()
	logger := NewLogger(LogConfig{BaseDir: tmpDir})

	sandboxID := uuid.New()
	want := filepath.Join(tmpDir, sandboxID.String(), "logs", "123.stdout.log")
	if got := logger.LogPath(sandboxID, 123, StreamStdout); got != want {
		t.Errorf("LogPath stdout = %q; want %q", got, want)
	}
	want = filepath.Join(tmpDir, sandboxID.String(), "logs", "123.stderr.log")
	if got := logger.LogPath(sandboxID, 123, StreamStderr); got != want {
		t.Errorf("LogPath stderr = %q; want %q", got, want)
	}
}

// TestLogger_ReadTail returns the last N lines from a single stream.
func TestLogger_ReadTail(t *testing.T) {
	tmpDir := t.TempDir()
	logger := NewLogger(LogConfig{BaseDir: tmpDir})

	sandboxID := uuid.New()
	pid := 999
	pending, _, _ := withPair(t, logger, sandboxID, pid)

	for _, line := range []string{"line 1", "line 2", "line 3", "line 4", "line 5"} {
		if _, err := pending.Stdout.Write([]byte(line + "\n")); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	if err := pending.Stdout.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	content, err := logger.ReadLog(sandboxID, pid, StreamStdout, 2, 0)
	if err != nil {
		t.Fatalf("ReadLog tail: %v", err)
	}
	got := string(content)
	if !strings.Contains(got, "line 4") || !strings.Contains(got, "line 5") {
		t.Errorf("tail missing expected lines, got: %s", got)
	}
}

// TestLogger_CloseLogPair writes the exit footer to both streams.
func TestLogger_CloseLogPair(t *testing.T) {
	tmpDir := t.TempDir()
	logger := NewLogger(LogConfig{BaseDir: tmpDir})

	sandboxID := uuid.New()
	pid := 777
	withPair(t, logger, sandboxID, pid)

	if err := logger.CloseLogPair(sandboxID, pid, ExitInfo{ExitCode: 42, Signal: 9, OOMKilled: true}); err != nil {
		t.Fatalf("CloseLogPair: %v", err)
	}

	for _, stream := range []Stream{StreamStdout, StreamStderr} {
		content, err := logger.ReadLog(sandboxID, pid, stream, 0, 0)
		if err != nil {
			t.Fatalf("ReadLog %s: %v", stream, err)
		}
		got := string(content)
		if !strings.Contains(got, "code 42") || !strings.Contains(got, "signal 9") || !strings.Contains(got, "oom true") {
			t.Errorf("%s log missing exit info, got: %s", stream, got)
		}
	}
}

// TestLogger_CleanupSandboxLogs removes all stream files.
func TestLogger_CleanupSandboxLogs(t *testing.T) {
	tmpDir := t.TempDir()
	logger := NewLogger(LogConfig{BaseDir: tmpDir})

	sandboxID := uuid.New()
	for _, pid := range []int{1, 2, 3} {
		pending, _, _ := withPair(t, logger, sandboxID, pid)
		_ = pending.Stdout.Close()
		_ = pending.Stderr.Close()
	}

	logDir := logger.LogDir(sandboxID)
	if _, err := os.Stat(logDir); err != nil {
		t.Fatalf("log dir should exist: %v", err)
	}

	if err := logger.CleanupSandboxLogs(sandboxID); err != nil {
		t.Fatalf("CleanupSandboxLogs: %v", err)
	}
	if _, err := os.Stat(logDir); !os.IsNotExist(err) {
		t.Errorf("log dir should be removed after cleanup")
	}
}

// TestLogger_ConcurrentWrites verifies the writer's mutex protects against
// torn writes from multiple goroutines.
func TestLogger_ConcurrentWrites(t *testing.T) {
	tmpDir := t.TempDir()
	logger := NewLogger(LogConfig{BaseDir: tmpDir})

	sandboxID := uuid.New()
	pid := 888
	pending, _, _ := withPair(t, logger, sandboxID, pid)

	var wg sync.WaitGroup
	errCh := make(chan error, 1)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				if _, err := pending.Stdout.Write([]byte("write from goroutine\n")); err != nil {
					select {
					case errCh <- err:
					default:
					}
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errCh)
	if err := <-errCh; err != nil {
		t.Fatalf("concurrent write: %v", err)
	}
	if err := pending.Stdout.Close(); err != nil {
		t.Fatalf("close stdout: %v", err)
	}

	logInfo, err := logger.GetLog(sandboxID, pid, StreamStdout)
	if err != nil {
		t.Fatalf("GetLog: %v", err)
	}
	if logInfo.SizeBytes < 10000 {
		t.Errorf("stdout too small for concurrent writes: %d", logInfo.SizeBytes)
	}
}

// TestLogger_StreamLogReceivesPushChunks verifies that bytes appended to
// the writer reach a Subscribe channel without polling.
func TestLogger_StreamLogReceivesPushChunks(t *testing.T) {
	tmpDir := t.TempDir()
	logger := NewLogger(LogConfig{BaseDir: tmpDir})

	sandboxID := uuid.New()
	pid := 666
	pending, _, _ := withPair(t, logger, sandboxID, pid)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	chunks := make(chan string, 16)
	done := make(chan struct{})
	var streamErr error
	go func() {
		streamErr = logger.StreamLog(ctx, sandboxID, pid, StreamStdout, func(chunk []byte) {
			chunks <- string(chunk)
		})
		close(done)
	}()

	// Give StreamLog a tick to subscribe.
	time.Sleep(20 * time.Millisecond)

	if _, err := pending.Stdout.Write([]byte("alpha\n")); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := pending.Stdout.Write([]byte("beta\n")); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Collect at least the two new chunks (replay may include header).
	gotChunk := func(want string) bool {
		deadline := time.Now().Add(500 * time.Millisecond)
		var buf strings.Builder
		for time.Now().Before(deadline) {
			select {
			case c := <-chunks:
				buf.WriteString(c)
				if strings.Contains(buf.String(), want) {
					return true
				}
			case <-time.After(50 * time.Millisecond):
			}
		}
		return false
	}
	if !gotChunk("alpha") {
		t.Errorf("did not receive alpha within 500ms (push not working)")
	}
	if !gotChunk("beta") {
		t.Errorf("did not receive beta")
	}

	// Closing the pair signals EOF to subscribers.
	if err := logger.CloseLogPair(sandboxID, pid, ExitInfo{ExitCode: 0}); err != nil {
		t.Fatalf("CloseLogPair: %v", err)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("StreamLog did not return after CloseLogPair")
	}
	if streamErr != nil && streamErr != context.Canceled && streamErr != context.DeadlineExceeded {
		t.Errorf("StreamLog err = %v", streamErr)
	}
}

// TestLogger_NonExistentLog returns an error when the file is missing.
func TestLogger_NonExistentLog(t *testing.T) {
	tmpDir := t.TempDir()
	logger := NewLogger(LogConfig{BaseDir: tmpDir})

	sandboxID := uuid.New()
	if _, err := logger.GetLog(sandboxID, 99999, StreamStdout); err == nil {
		t.Error("expected error for non-existent log")
	}
	if _, err := logger.ReadLog(sandboxID, 99999, StreamStdout, 0, 0); err == nil {
		t.Error("expected error reading non-existent log")
	}
}

// TestLogger_EmptySandbox returns no logs.
func TestLogger_EmptySandbox(t *testing.T) {
	tmpDir := t.TempDir()
	logger := NewLogger(LogConfig{BaseDir: tmpDir})

	logs, err := logger.ListLogs(uuid.New())
	if err != nil {
		t.Fatalf("ListLogs: %v", err)
	}
	if len(logs) != 0 {
		t.Errorf("expected 0 logs, got %d", len(logs))
	}
}

// TestPendingLogPair_AbortRemovesBothFiles ensures AbortPair cleans both
// pending stream files.
func TestPendingLogPair_AbortRemovesBothFiles(t *testing.T) {
	tmpDir := t.TempDir()
	logger := NewLogger(LogConfig{BaseDir: tmpDir})

	pending, err := logger.CreatePendingLogPair(uuid.New())
	if err != nil {
		t.Fatalf("CreatePendingLogPair: %v", err)
	}
	stdoutPath := pending.Stdout.path
	stderrPath := pending.Stderr.path

	if _, err := os.Stat(stdoutPath); err != nil {
		t.Fatalf("stdout pending file missing: %v", err)
	}
	if _, err := os.Stat(stderrPath); err != nil {
		t.Fatalf("stderr pending file missing: %v", err)
	}

	if err := logger.AbortPair(pending); err != nil {
		t.Fatalf("AbortPair: %v", err)
	}
	if _, err := os.Stat(stdoutPath); !os.IsNotExist(err) {
		t.Errorf("stdout pending file should be removed")
	}
	if _, err := os.Stat(stderrPath); !os.IsNotExist(err) {
		t.Errorf("stderr pending file should be removed")
	}
}

// TestPendingLogPair_FinalizeRenamesAndPreservesContent verifies the
// two-phase create→finalize path.
func TestPendingLogPair_FinalizeRenamesAndPreservesContent(t *testing.T) {
	tmpDir := t.TempDir()
	logger := NewLogger(LogConfig{BaseDir: tmpDir})

	sandboxID := uuid.New()
	pid := 12345

	pending, err := logger.CreatePendingLogPair(sandboxID)
	if err != nil {
		t.Fatalf("CreatePendingLogPair: %v", err)
	}
	if _, err := pending.Stdout.Write([]byte("early stdout output\n")); err != nil {
		t.Fatalf("write stdout: %v", err)
	}

	stdoutPath, _, err := logger.FinalizePair(pending, pid)
	if err != nil {
		t.Fatalf("FinalizePair: %v", err)
	}
	if want := logger.LogPath(sandboxID, pid, StreamStdout); stdoutPath != want {
		t.Errorf("stdout path = %q; want %q", stdoutPath, want)
	}

	if _, err := pending.Stdout.Write([]byte("late stdout output\n")); err != nil {
		t.Fatalf("write stdout (post-finalize): %v", err)
	}

	content, err := logger.ReadLog(sandboxID, pid, StreamStdout, 0, 0)
	if err != nil {
		t.Fatalf("ReadLog: %v", err)
	}
	if !strings.Contains(string(content), "early stdout output") {
		t.Errorf("missing early content")
	}
	if !strings.Contains(string(content), "late stdout output") {
		t.Errorf("missing late content")
	}
}
