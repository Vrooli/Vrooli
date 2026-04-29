// Package process — process logging.
//
// Two log streams per process: stdout and stderr, written to separate
// files ({pid}.stdout.log and {pid}.stderr.log). Both files share a
// per-process subscription fan-out so that SSE consumers receive each
// write as it lands, with no server-side polling. Greenfield: the older
// merged-log API is gone, callers must specify a stream.
package process

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"workspace-sandbox/internal/clock"
)

// Stream identifies one of the two log streams for a process.
type Stream string

const (
	StreamStdout Stream = "stdout"
	StreamStderr Stream = "stderr"
)

// Validate returns an error if s is not stdout or stderr.
func (s Stream) Validate() error {
	switch s {
	case StreamStdout, StreamStderr:
		return nil
	default:
		return fmt.Errorf("invalid stream %q (want stdout or stderr)", s)
	}
}

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
		RetainLogs: 0,                // Keep until sandbox deletion
	}
}

// ProcessLog represents the metadata for a single log stream.
type ProcessLog struct {
	PID       int       `json:"pid"`
	SandboxID uuid.UUID `json:"sandboxId"`
	Stream    Stream    `json:"stream"`
	Path      string    `json:"path"`
	StartedAt time.Time `json:"startedAt"`
	SizeBytes int64     `json:"sizeBytes"`
	IsActive  bool      `json:"isActive"`
}

// Logger manages log files for sandbox processes.
type Logger struct {
	mu      sync.RWMutex
	config  LogConfig
	writers map[string]*logWriter // key: "{sandboxID}/{pid}/{stream}"
	clock   clock.Clock
}

// logWriter handles writing to a single stream of a process log file. It
// also maintains a fan-out list of subscribers that receive each Write.
type logWriter struct {
	mu        sync.Mutex
	file      *os.File
	path      string
	pid       int
	sandboxID uuid.UUID
	stream    Stream
	sizeBytes int64
	startedAt time.Time

	subsMu      sync.Mutex
	subscribers []chan []byte
}

// LogPair pairs writers for the two streams of a single process.
type LogPair struct {
	Stdout io.WriteCloser
	Stderr io.WriteCloser
}

// PendingLogPair represents log files created before the process PID is
// known. Use FinalizePair to rename them to the actual PID.
type PendingLogPair struct {
	Stdout    *logWriter
	Stderr    *logWriter
	TempID    string
	SandboxID uuid.UUID
}

// AsLogPair returns the pending writers as a LogPair (so callers can wire
// them as cmd.Stdout / cmd.Stderr without having to know the underlying
// type).
func (p *PendingLogPair) AsLogPair() LogPair {
	if p == nil {
		return LogPair{}
	}
	return LogPair{Stdout: p.Stdout, Stderr: p.Stderr}
}

// NewLogger creates a new process logger. clk is required (the log
// header and exit-trailer timestamps go through it so tests can assert
// the exact wording produced for a given exit info).
func NewLogger(cfg LogConfig, clk clock.Clock) *Logger {
	if clk == nil {
		panic("process.NewLogger: clock is required")
	}
	return &Logger{
		config:  cfg,
		writers: make(map[string]*logWriter),
		clock:   clk,
	}
}

// LogDir returns the log directory for a sandbox.
func (l *Logger) LogDir(sandboxID uuid.UUID) string {
	return filepath.Join(l.config.BaseDir, sandboxID.String(), "logs")
}

// LogPath returns the file path for a specific stream of a process log.
func (l *Logger) LogPath(sandboxID uuid.UUID, pid int, stream Stream) string {
	return filepath.Join(l.LogDir(sandboxID), fmt.Sprintf("%d.%s.log", pid, stream))
}

// CreatePendingLogPair creates two log files (stdout + stderr) before the
// process PID is known. Use FinalizePair after the process starts.
func (l *Logger) CreatePendingLogPair(sandboxID uuid.UUID) (*PendingLogPair, error) {
	logDir := l.LogDir(sandboxID)
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	tempID := uuid.New().String()[:8]

	stdoutLW, err := openPendingStream(logDir, tempID, sandboxID, StreamStdout, l.clock)
	if err != nil {
		return nil, err
	}
	stderrLW, err := openPendingStream(logDir, tempID, sandboxID, StreamStderr, l.clock)
	if err != nil {
		_ = stdoutLW.Close()
		_ = os.Remove(stdoutLW.path)
		return nil, err
	}

	return &PendingLogPair{
		Stdout:    stdoutLW,
		Stderr:    stderrLW,
		TempID:    tempID,
		SandboxID: sandboxID,
	}, nil
}

func openPendingStream(logDir, tempID string, sandboxID uuid.UUID, stream Stream, clk clock.Clock) (*logWriter, error) {
	tempPath := filepath.Join(logDir, fmt.Sprintf("pending_%s.%s.log", tempID, stream))
	file, err := os.OpenFile(tempPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to create pending log file: %w", err)
	}
	now := clk.Now()
	header := fmt.Sprintf("=== Process Log (%s) | Sandbox %s | Started %s ===\n\n",
		stream, sandboxID.String(), now.Format(time.RFC3339))
	if _, err := file.WriteString(header); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("failed to write pending log header: %w", err)
	}
	return &logWriter{
		file:      file,
		path:      tempPath,
		pid:       0,
		sandboxID: sandboxID,
		stream:    stream,
		sizeBytes: int64(len(header)),
		startedAt: now,
	}, nil
}

// FinalizePair renames both pending log files to use the actual PID and
// registers them in the logger's writer map. Returns the resolved paths.
func (l *Logger) FinalizePair(pending *PendingLogPair, pid int) (stdoutPath, stderrPath string, err error) {
	if pending == nil || pending.Stdout == nil || pending.Stderr == nil {
		return "", "", fmt.Errorf("invalid pending log pair")
	}

	stdoutPath = l.LogPath(pending.SandboxID, pid, StreamStdout)
	stderrPath = l.LogPath(pending.SandboxID, pid, StreamStderr)

	if renameErr := os.Rename(pending.Stdout.path, stdoutPath); renameErr != nil {
		stdoutPath = pending.Stdout.path
	}
	if renameErr := os.Rename(pending.Stderr.path, stderrPath); renameErr != nil {
		stderrPath = pending.Stderr.path
	}

	pending.Stdout.mu.Lock()
	pending.Stdout.path = stdoutPath
	pending.Stdout.pid = pid
	pending.Stdout.mu.Unlock()

	pending.Stderr.mu.Lock()
	pending.Stderr.path = stderrPath
	pending.Stderr.pid = pid
	pending.Stderr.mu.Unlock()

	l.mu.Lock()
	l.writers[l.writerKey(pending.SandboxID, pid, StreamStdout)] = pending.Stdout
	l.writers[l.writerKey(pending.SandboxID, pid, StreamStderr)] = pending.Stderr
	l.mu.Unlock()

	return stdoutPath, stderrPath, nil
}

// AbortPair closes and removes both pending log files (e.g., when the
// process failed to start).
func (l *Logger) AbortPair(pending *PendingLogPair) error {
	if pending == nil {
		return nil
	}
	var firstErr error
	for _, lw := range []*logWriter{pending.Stdout, pending.Stderr} {
		if lw == nil {
			continue
		}
		path := lw.path
		if err := lw.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		if err := os.Remove(path); err != nil && firstErr == nil && !os.IsNotExist(err) {
			firstErr = err
		}
	}
	return firstErr
}

// CloseLogPair closes both stream writers for a process and writes a
// shared trailer. Idempotent: subsequent calls return nil.
func (l *Logger) CloseLogPair(sandboxID uuid.UUID, pid int, info ExitInfo) error {
	var firstErr error
	for _, stream := range []Stream{StreamStdout, StreamStderr} {
		key := l.writerKey(sandboxID, pid, stream)
		l.mu.Lock()
		lw, ok := l.writers[key]
		if ok {
			delete(l.writers, key)
		}
		l.mu.Unlock()

		if !ok {
			continue
		}

		footer := fmt.Sprintf("\n=== Process Exited: code %d signal %d oom %v | %s ===\n",
			info.ExitCode, info.Signal, info.OOMKilled, l.clock.Now().Format(time.RFC3339))
		if _, err := lw.Write([]byte(footer)); err != nil && firstErr == nil {
			firstErr = err
		}
		if err := lw.closeAndNotify(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (l *Logger) writerKey(sandboxID uuid.UUID, pid int, stream Stream) string {
	return fmt.Sprintf("%s/%d/%s", sandboxID.String(), pid, stream)
}

// GetLog returns metadata about one stream of a process log.
func (l *Logger) GetLog(sandboxID uuid.UUID, pid int, stream Stream) (*ProcessLog, error) {
	if err := stream.Validate(); err != nil {
		return nil, err
	}
	logPath := l.LogPath(sandboxID, pid, stream)

	info, err := os.Stat(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("log not found for PID %d stream %s", pid, stream)
		}
		return nil, err
	}

	key := l.writerKey(sandboxID, pid, stream)
	l.mu.RLock()
	_, isActive := l.writers[key]
	l.mu.RUnlock()

	return &ProcessLog{
		PID:       pid,
		SandboxID: sandboxID,
		Stream:    stream,
		Path:      logPath,
		StartedAt: info.ModTime(),
		SizeBytes: info.Size(),
		IsActive:  isActive,
	}, nil
}

// ListLogs returns the metadata for every (pid, stream) pair this sandbox
// has on disk.
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
		// Filenames look like "12345.stdout.log" or "12345.stderr.log".
		// Skip pending files (prefixed "pending_").
		name := entry.Name()
		if strings.HasPrefix(name, "pending_") {
			continue
		}
		var pid int
		var streamStr string
		if _, err := fmt.Sscanf(name, "%d.%s", &pid, &streamStr); err != nil {
			continue
		}
		// streamStr arrives as "stdout.log" or "stderr.log"; chop the .log.
		streamStr = strings.TrimSuffix(streamStr, ".log")
		stream := Stream(streamStr)
		if err := stream.Validate(); err != nil {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		key := l.writerKey(sandboxID, pid, stream)
		l.mu.RLock()
		_, isActive := l.writers[key]
		l.mu.RUnlock()

		logs = append(logs, &ProcessLog{
			PID:       pid,
			SandboxID: sandboxID,
			Stream:    stream,
			Path:      filepath.Join(logDir, name),
			StartedAt: info.ModTime(),
			SizeBytes: info.Size(),
			IsActive:  isActive,
		})
	}

	return logs, nil
}

// ReadLog reads the content of one stream of a process log.
// If tail > 0, returns only the last 'tail' lines.
// If offset > 0, skips the first 'offset' bytes.
func (l *Logger) ReadLog(sandboxID uuid.UUID, pid int, stream Stream, tail int, offset int64) ([]byte, error) {
	if err := stream.Validate(); err != nil {
		return nil, err
	}
	logPath := l.LogPath(sandboxID, pid, stream)

	file, err := os.Open(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("log not found for PID %d stream %s", pid, stream)
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

	bufSize := int64(8192)
	if bufSize > info.Size() {
		bufSize = info.Size()
	}

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

// Subscribe returns a channel that receives each new chunk written to the
// given stream, plus an unsubscribe function to detach the subscriber.
//
// The channel is buffered (32) so a slow subscriber does not block the
// hot Write path; if it fills up, chunks are dropped for that subscriber
// and a sentinel `nil` is sent on the next successful enqueue. Subscribers
// SHOULD drain quickly.
//
// Returns ok=false when the stream's writer is not registered (e.g.,
// process has already exited and the writer was closed). Callers should
// fall back to ReadLog in that case.
func (l *Logger) Subscribe(sandboxID uuid.UUID, pid int, stream Stream) (ch <-chan []byte, unsubscribe func(), ok bool) {
	if err := stream.Validate(); err != nil {
		return nil, nil, false
	}
	key := l.writerKey(sandboxID, pid, stream)
	l.mu.RLock()
	lw := l.writers[key]
	l.mu.RUnlock()
	if lw == nil {
		return nil, nil, false
	}

	c := make(chan []byte, 32)
	lw.subsMu.Lock()
	lw.subscribers = append(lw.subscribers, c)
	lw.subsMu.Unlock()

	unsubscribe = func() {
		lw.subsMu.Lock()
		defer lw.subsMu.Unlock()
		for i, sub := range lw.subscribers {
			if sub == c {
				lw.subscribers = append(lw.subscribers[:i], lw.subscribers[i+1:]...)
				close(c)
				return
			}
		}
	}
	return c, unsubscribe, true
}

// CleanupSandboxLogs removes all logs for a sandbox.
// Call when the sandbox is deleted.
func (l *Logger) CleanupSandboxLogs(sandboxID uuid.UUID) error {
	logDir := l.LogDir(sandboxID)

	var firstErr error
	l.mu.Lock()
	for key, lw := range l.writers {
		if lw.sandboxID == sandboxID {
			if err := lw.closeAndNotify(); err != nil && firstErr == nil {
				firstErr = err
			}
			delete(l.writers, key)
		}
	}
	l.mu.Unlock()

	if err := os.RemoveAll(logDir); err != nil && firstErr == nil {
		firstErr = err
	}
	return firstErr
}

// MostRecentLog returns the most recently created log entry for a sandbox
// (any stream).
func (l *Logger) MostRecentLog(sandboxID uuid.UUID) (*ProcessLog, error) {
	logs, err := l.ListLogs(sandboxID)
	if err != nil {
		return nil, err
	}
	if len(logs) == 0 {
		return nil, fmt.Errorf("no logs found for sandbox %s", sandboxID)
	}

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

	// Fan out to subscribers (non-blocking).
	if n > 0 {
		// Copy bytes so subscriber can hold onto them after we return.
		chunk := make([]byte, n)
		copy(chunk, p[:n])
		lw.broadcast(chunk)
	}
	return n, err
}

func (lw *logWriter) broadcast(chunk []byte) {
	lw.subsMu.Lock()
	defer lw.subsMu.Unlock()
	for _, sub := range lw.subscribers {
		select {
		case sub <- chunk:
		default:
			// Subscriber's buffer is full. Drop this chunk for them.
			// The subscriber is responsible for falling back to ReadLog
			// at offset to recover any gap.
		}
	}
}

// Close closes the underlying file. Use closeAndNotify when the writer is
// being retired so subscribers also see EOF.
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

// closeAndNotify closes the file and signals EOF to subscribers by closing
// their channels.
func (lw *logWriter) closeAndNotify() error {
	err := lw.Close()

	lw.subsMu.Lock()
	for _, sub := range lw.subscribers {
		// Drain may panic on already-closed channel; protect.
		func(c chan []byte) {
			defer func() { _ = recover() }()
			close(c)
		}(sub)
	}
	lw.subscribers = nil
	lw.subsMu.Unlock()

	return err
}

// StreamLog is retained as a convenience for callers that want to consume
// a stream synchronously without managing a subscriber channel themselves.
// It returns when ctx is cancelled or the writer has been closed (process
// exited).
func (l *Logger) StreamLog(ctx context.Context, sandboxID uuid.UUID, pid int, stream Stream, callback func([]byte)) error {
	if err := stream.Validate(); err != nil {
		return err
	}

	// Replay any bytes already on disk so consumers don't miss content
	// that was written between process start and subscription.
	if existing, err := l.ReadLog(sandboxID, pid, stream, 0, 0); err == nil && len(existing) > 0 {
		callback(existing)
	}

	ch, unsubscribe, ok := l.Subscribe(sandboxID, pid, stream)
	if !ok {
		// Writer not registered: process probably exited already; we've
		// already replayed disk contents above.
		return nil
	}
	defer unsubscribe()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case chunk, open := <-ch:
			if !open {
				return nil
			}
			callback(chunk)
		}
	}
}
