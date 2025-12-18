// Package process provides process tracking and logging for sandboxes.
package process

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

// LogConfig configures process logging behavior.
type LogConfig struct {
	// BaseDir is the root directory for all sandbox data.
	// Logs are stored in {BaseDir}/{sandboxID}/logs/
	BaseDir string

	// MaxLogSize is the maximum size of a single log file in bytes.
	// When exceeded, the log is rotated. 0 = no limit.
	MaxLogSize int64

	// RetainLogs is how long to keep log files after process exits.
	// 0 = keep indefinitely (until sandbox is deleted).
	RetainLogs time.Duration
}

// DefaultLogConfig returns sensible defaults for process logging.
func DefaultLogConfig(baseDir string) LogConfig {
	return LogConfig{
		BaseDir:    baseDir,
		MaxLogSize: 50 * 1024 * 1024, // 50 MB per log
		RetainLogs: 0,                 // Keep until sandbox deletion
	}
}

// ProcessLog represents a log file for a single process.
type ProcessLog struct {
	// PID is the process ID this log belongs to.
	PID int `json:"pid"`

	// SandboxID is the sandbox this process belongs to.
	SandboxID uuid.UUID `json:"sandboxId"`

	// Path is the absolute path to the log file.
	Path string `json:"path"`

	// StartedAt is when the process started.
	StartedAt time.Time `json:"startedAt"`

	// SizeBytes is the current size of the log file.
	SizeBytes int64 `json:"sizeBytes"`

	// IsActive indicates if the process is still running and writing.
	IsActive bool `json:"isActive"`
}

// Logger manages log files for sandbox processes.
type Logger struct {
	mu      sync.RWMutex
	config  LogConfig
	writers map[string]*logWriter // key: "{sandboxID}/{pid}"
}

// logWriter handles writing to a process log file.
type logWriter struct {
	mu        sync.Mutex
	file      *os.File
	path      string
	pid       int
	sandboxID uuid.UUID
	sizeBytes int64
	startedAt time.Time
}

// PendingLog represents a log file created before the process PID is known.
// Use FinalizeLog() to rename it to the actual PID after process starts.
type PendingLog struct {
	Writer    *logWriter
	TempID    string // Temporary identifier used for the file name
	SandboxID uuid.UUID
}

// NewLogger creates a new process logger.
func NewLogger(cfg LogConfig) *Logger {
	return &Logger{
		config:  cfg,
		writers: make(map[string]*logWriter),
	}
}

// LogDir returns the log directory for a sandbox.
func (l *Logger) LogDir(sandboxID uuid.UUID) string {
	return filepath.Join(l.config.BaseDir, sandboxID.String(), "logs")
}

// LogPath returns the log file path for a specific process.
func (l *Logger) LogPath(sandboxID uuid.UUID, pid int) string {
	return filepath.Join(l.LogDir(sandboxID), fmt.Sprintf("%d.log", pid))
}

// CreateLog creates a new log file for a process and returns a writer.
// The writer should be used as stdout/stderr for the process.
func (l *Logger) CreateLog(sandboxID uuid.UUID, pid int) (io.WriteCloser, error) {
	logDir := l.LogDir(sandboxID)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	logPath := l.LogPath(sandboxID, pid)
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}

	// Write header
	header := fmt.Sprintf("=== Process Log: PID %d | Sandbox %s | Started %s ===\n\n",
		pid, sandboxID.String(), time.Now().Format(time.RFC3339))
	file.WriteString(header)

	lw := &logWriter{
		file:      file,
		path:      logPath,
		pid:       pid,
		sandboxID: sandboxID,
		sizeBytes: int64(len(header)),
		startedAt: time.Now(),
	}

	key := l.writerKey(sandboxID, pid)
	l.mu.Lock()
	l.writers[key] = lw
	l.mu.Unlock()

	return lw, nil
}

// writerKey returns the map key for a sandbox/pid pair.
func (l *Logger) writerKey(sandboxID uuid.UUID, pid int) string {
	return fmt.Sprintf("%s/%d", sandboxID.String(), pid)
}

// CreatePendingLog creates a log file before the process PID is known.
// Returns a PendingLog that should be finalized with FinalizeLog() after
// the process starts and the PID is known.
//
// The Writer in the returned PendingLog can be used as stdout/stderr
// for the process via BwrapConfig.LogWriter.
func (l *Logger) CreatePendingLog(sandboxID uuid.UUID) (*PendingLog, error) {
	logDir := l.LogDir(sandboxID)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Generate a temporary ID for the log file
	tempID := uuid.New().String()[:8] // Short UUID prefix for temp name
	tempPath := filepath.Join(logDir, fmt.Sprintf("pending_%s.log", tempID))

	file, err := os.OpenFile(tempPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create pending log file: %w", err)
	}

	// Write initial header (will be updated when finalized)
	header := fmt.Sprintf("=== Process Log | Sandbox %s | Started %s ===\n\n",
		sandboxID.String(), time.Now().Format(time.RFC3339))
	file.WriteString(header)

	lw := &logWriter{
		file:      file,
		path:      tempPath,
		pid:       0, // Unknown until finalized
		sandboxID: sandboxID,
		sizeBytes: int64(len(header)),
		startedAt: time.Now(),
	}

	return &PendingLog{
		Writer:    lw,
		TempID:    tempID,
		SandboxID: sandboxID,
	}, nil
}

// FinalizeLog renames a pending log file to use the actual PID and
// registers it in the logger's writer map.
// Call this after the process has started and the PID is known.
func (l *Logger) FinalizeLog(pending *PendingLog, pid int) (string, error) {
	if pending == nil || pending.Writer == nil {
		return "", fmt.Errorf("invalid pending log")
	}

	lw := pending.Writer

	// Rename the file to use the actual PID
	newPath := l.LogPath(pending.SandboxID, pid)
	if err := os.Rename(lw.path, newPath); err != nil {
		// If rename fails, just keep using the temp path
		// This might happen if file is being written to
		newPath = lw.path
	}

	// Update the writer's state
	lw.mu.Lock()
	lw.path = newPath
	lw.pid = pid
	lw.mu.Unlock()

	// Register in the logger's writer map
	key := l.writerKey(pending.SandboxID, pid)
	l.mu.Lock()
	l.writers[key] = lw
	l.mu.Unlock()

	return newPath, nil
}

// AbortPendingLog closes and removes a pending log file if the process
// failed to start. Call this if CreatePendingLog succeeds but the
// process fails to start.
func (l *Logger) AbortPendingLog(pending *PendingLog) error {
	if pending == nil || pending.Writer == nil {
		return nil
	}

	lw := pending.Writer
	path := lw.path
	lw.Close()

	return os.Remove(path)
}

// CloseLog closes the log writer for a process.
// Call this when the process exits.
func (l *Logger) CloseLog(sandboxID uuid.UUID, pid int, exitCode int) error {
	key := l.writerKey(sandboxID, pid)

	l.mu.Lock()
	lw, ok := l.writers[key]
	if ok {
		delete(l.writers, key)
	}
	l.mu.Unlock()

	if !ok {
		return nil // Already closed or never created
	}

	// Write footer
	footer := fmt.Sprintf("\n=== Process Exited: code %d | %s ===\n",
		exitCode, time.Now().Format(time.RFC3339))
	lw.Write([]byte(footer))

	return lw.Close()
}

// GetLog returns metadata about a process log.
func (l *Logger) GetLog(sandboxID uuid.UUID, pid int) (*ProcessLog, error) {
	logPath := l.LogPath(sandboxID, pid)

	info, err := os.Stat(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("log not found for PID %d", pid)
		}
		return nil, err
	}

	// Check if writer is still active
	key := l.writerKey(sandboxID, pid)
	l.mu.RLock()
	_, isActive := l.writers[key]
	l.mu.RUnlock()

	return &ProcessLog{
		PID:       pid,
		SandboxID: sandboxID,
		Path:      logPath,
		StartedAt: info.ModTime(), // Approximation
		SizeBytes: info.Size(),
		IsActive:  isActive,
	}, nil
}

// ListLogs returns all log files for a sandbox.
func (l *Logger) ListLogs(sandboxID uuid.UUID) ([]*ProcessLog, error) {
	logDir := l.LogDir(sandboxID)

	entries, err := os.ReadDir(logDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*ProcessLog{}, nil
		}
		return nil, err
	}

	var logs []*ProcessLog
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Parse PID from filename (e.g., "12345.log")
		var pid int
		if _, err := fmt.Sscanf(entry.Name(), "%d.log", &pid); err != nil {
			continue // Skip non-matching files
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		key := l.writerKey(sandboxID, pid)
		l.mu.RLock()
		_, isActive := l.writers[key]
		l.mu.RUnlock()

		logs = append(logs, &ProcessLog{
			PID:       pid,
			SandboxID: sandboxID,
			Path:      filepath.Join(logDir, entry.Name()),
			StartedAt: info.ModTime(),
			SizeBytes: info.Size(),
			IsActive:  isActive,
		})
	}

	return logs, nil
}

// ReadLog reads the contents of a log file.
// If tail > 0, returns only the last 'tail' lines.
// If offset > 0, skips the first 'offset' bytes.
func (l *Logger) ReadLog(sandboxID uuid.UUID, pid int, tail int, offset int64) ([]byte, error) {
	logPath := l.LogPath(sandboxID, pid)

	file, err := os.Open(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("log not found for PID %d", pid)
		}
		return nil, err
	}
	defer file.Close()

	if tail > 0 {
		return l.readTail(file, tail)
	}

	if offset > 0 {
		if _, err := file.Seek(offset, io.SeekStart); err != nil {
			return nil, err
		}
	}

	return io.ReadAll(file)
}

// readTail reads the last n lines from a file.
func (l *Logger) readTail(file *os.File, lines int) ([]byte, error) {
	// Get file size
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}

	// For small files, just read everything
	if info.Size() < 8192 {
		content, err := io.ReadAll(file)
		if err != nil {
			return nil, err
		}
		return l.lastNLines(content, lines), nil
	}

	// For larger files, read from the end
	bufSize := int64(8192)
	if bufSize > info.Size() {
		bufSize = info.Size()
	}

	// Read last chunk
	buf := make([]byte, bufSize)
	_, err = file.ReadAt(buf, info.Size()-bufSize)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return l.lastNLines(buf, lines), nil
}

// lastNLines returns the last n lines from content.
func (l *Logger) lastNLines(content []byte, n int) []byte {
	if n <= 0 {
		return content
	}

	lines := make([][]byte, 0, n)
	scanner := bufio.NewScanner(bufio.NewReader(io.NopCloser(
		&bytesReader{content: content},
	)))

	for scanner.Scan() {
		lines = append(lines, append([]byte{}, scanner.Bytes()...))
		if len(lines) > n {
			lines = lines[1:]
		}
	}

	// Reassemble
	result := make([]byte, 0)
	for i, line := range lines {
		result = append(result, line...)
		if i < len(lines)-1 {
			result = append(result, '\n')
		}
	}
	return result
}

// bytesReader wraps a byte slice for use with Scanner.
type bytesReader struct {
	content []byte
	offset  int
}

func (r *bytesReader) Read(p []byte) (n int, err error) {
	if r.offset >= len(r.content) {
		return 0, io.EOF
	}
	n = copy(p, r.content[r.offset:])
	r.offset += n
	return n, nil
}

func (r *bytesReader) Close() error {
	return nil
}

// StreamLog streams log contents as they're written.
// The callback is called for each new chunk of data.
// Returns when the context is canceled or the process exits.
func (l *Logger) StreamLog(ctx context.Context, sandboxID uuid.UUID, pid int, callback func([]byte)) error {
	logPath := l.LogPath(sandboxID, pid)

	file, err := os.Open(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("log not found for PID %d", pid)
		}
		return err
	}
	defer file.Close()

	// Seek to end for live streaming
	currentPos, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}

	buf := make([]byte, 4096)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-ticker.C:
			// Check if more data is available
			info, err := file.Stat()
			if err != nil {
				return err
			}

			if info.Size() > currentPos {
				// Read new data
				n, err := file.Read(buf)
				if err != nil && err != io.EOF {
					return err
				}
				if n > 0 {
					callback(buf[:n])
					currentPos += int64(n)
				}
			}

			// Check if process is still active
			key := l.writerKey(sandboxID, pid)
			l.mu.RLock()
			_, isActive := l.writers[key]
			l.mu.RUnlock()

			if !isActive && info.Size() == currentPos {
				// Process ended and we've read all data
				return nil
			}
		}
	}
}

// CleanupSandboxLogs removes all logs for a sandbox.
// Call when the sandbox is deleted.
func (l *Logger) CleanupSandboxLogs(sandboxID uuid.UUID) error {
	logDir := l.LogDir(sandboxID)

	// Close any active writers first
	l.mu.Lock()
	for key, lw := range l.writers {
		if lw.sandboxID == sandboxID {
			lw.Close()
			delete(l.writers, key)
		}
	}
	l.mu.Unlock()

	return os.RemoveAll(logDir)
}

// MostRecentLog returns the most recently created log for a sandbox.
func (l *Logger) MostRecentLog(sandboxID uuid.UUID) (*ProcessLog, error) {
	logs, err := l.ListLogs(sandboxID)
	if err != nil {
		return nil, err
	}

	if len(logs) == 0 {
		return nil, fmt.Errorf("no logs found for sandbox %s", sandboxID)
	}

	// Find most recent by started time
	var mostRecent *ProcessLog
	for _, log := range logs {
		if mostRecent == nil || log.StartedAt.After(mostRecent.StartedAt) {
			mostRecent = log
		}
	}

	return mostRecent, nil
}

// --- logWriter implementation ---

func (lw *logWriter) Write(p []byte) (n int, err error) {
	lw.mu.Lock()
	defer lw.mu.Unlock()

	if lw.file == nil {
		return 0, fmt.Errorf("log file closed")
	}

	n, err = lw.file.Write(p)
	lw.sizeBytes += int64(n)
	return n, err
}

func (lw *logWriter) Close() error {
	lw.mu.Lock()
	defer lw.mu.Unlock()

	if lw.file == nil {
		return nil
	}

	err := lw.file.Close()
	lw.file = nil
	return err
}
