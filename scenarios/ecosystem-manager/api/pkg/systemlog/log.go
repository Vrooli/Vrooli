package systemlog

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ecosystem-manager/api/pkg/internal/timeutil"
)

type Level string

const (
	LevelDebug Level = "DEBUG"
	LevelInfo  Level = "INFO"
	LevelWarn  Level = "WARN"
	LevelError Level = "ERROR"

	// Performance tuning constants
	syncInterval     = 100        // Sync every 100 writes
	maxRecentEntries = 5000       // Keep up to 5000 entries in memory
	tailReadSize     = 256 * 1024 // Read last 256KB for recent entries
	minTailReadSize  = 4 * 1024   // Minimum tail read of 4KB
)

type Entry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     Level     `json:"level"`
	Message   string    `json:"message"`
}

var (
	logDir        string
	logFile       *os.File
	logWriter     *bufio.Writer
	mu            sync.Mutex
	initOnce      sync.Once
	writeCounter  int
	recentEntries []Entry // Ring buffer of recent entries
	recentMu      sync.RWMutex
)

// Init initializes logging using the current working directory as the base.
// Prefer InitWithBaseDir when the scenario root is known to ensure deterministic log placement.
func Init() {
	InitWithBaseDir("")
}

// InitWithBaseDir initializes logging using the provided base directory (scenario root).
// Logs will be written to <baseDir>/logs.
func InitWithBaseDir(baseDir string) {
	initOnce.Do(func() {
		logDir = resolveLogDir(baseDir)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "failed to create log dir: %v\n", err)
			return
		}
		filePath := filepath.Join(logDir, time.Now().Format("2006-01-02_150405")+".log")
		f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to open log file: %v\n", err)
			return
		}
		logFile = f
		logWriter = bufio.NewWriterSize(f, 64*1024) // 64KB write buffer
		recentEntries = make([]Entry, 0, maxRecentEntries)
	})
}

func resolveLogDir(baseDir string) string {
	if baseDir != "" {
		return filepath.Join(baseDir, "logs")
	}

	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	return filepath.Join(cwd, "logs")
}

func write(level Level, msg string) {
	if logWriter == nil {
		return
	}

	timestamp := timeutil.NowRFC3339()
	entry := fmt.Sprintf("%s [%s] %s\n", timestamp, level, msg)

	mu.Lock()
	_, _ = logWriter.WriteString(entry)
	writeCounter++

	// Periodic sync for durability (every 100 writes)
	if writeCounter%syncInterval == 0 {
		_ = logWriter.Flush()
		_ = logFile.Sync()
	}
	mu.Unlock()

	// Update in-memory cache of recent entries (separate lock to avoid contention)
	ts, _ := time.Parse(time.RFC3339, timestamp)
	recentMu.Lock()
	recentEntries = append(recentEntries, Entry{
		Timestamp: ts,
		Level:     level,
		Message:   msg,
	})

	// Trim to max size (ring buffer behavior)
	if len(recentEntries) > maxRecentEntries {
		copy(recentEntries, recentEntries[len(recentEntries)-maxRecentEntries:])
		recentEntries = recentEntries[:maxRecentEntries]
	}
	recentMu.Unlock()
}

func Debugf(format string, args ...any) { write(LevelDebug, fmt.Sprintf(format, args...)) }
func Info(msg string)                   { write(LevelInfo, msg) }
func Infof(format string, args ...any)  { write(LevelInfo, fmt.Sprintf(format, args...)) }
func Warnf(format string, args ...any)  { write(LevelWarn, fmt.Sprintf(format, args...)) }
func Errorf(format string, args ...any) { write(LevelError, fmt.Sprintf(format, args...)) }

func RecentEntries(limit int) ([]Entry, error) {
	if logFile == nil {
		return nil, fmt.Errorf("log file not initialized")
	}

	// Fast path: serve from in-memory cache if sufficient
	recentMu.RLock()
	cacheSize := len(recentEntries)
	if limit <= 0 || cacheSize >= limit {
		// We have enough in cache
		result := make([]Entry, cacheSize)
		copy(result, recentEntries)
		recentMu.RUnlock()

		if limit > 0 && len(result) > limit {
			result = result[len(result)-limit:]
		}
		return result, nil
	}
	recentMu.RUnlock()

	// Slow path: read from file using efficient tail reading
	return readRecentEntriesFromFile(limit)
}

// readRecentEntriesFromFile efficiently reads recent entries from the log file
// by reading only the tail of the file instead of the entire file
func readRecentEntriesFromFile(limit int) ([]Entry, error) {
	mu.Lock()
	defer mu.Unlock()

	// Flush pending writes before reading
	if logWriter != nil {
		_ = logWriter.Flush()
	}

	// Get file size
	fileInfo, err := logFile.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat log file: %w", err)
	}
	fileSize := fileInfo.Size()

	// Calculate how much to read from the tail
	readSize := int64(tailReadSize)
	if fileSize < readSize {
		readSize = fileSize
	}
	if readSize < minTailReadSize && fileSize >= minTailReadSize {
		readSize = minTailReadSize
	}

	// Seek to tail position
	offset := fileSize - readSize
	if offset < 0 {
		offset = 0
	}

	if _, err := logFile.Seek(offset, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek to tail: %w", err)
	}

	// Read the tail
	scanner := bufio.NewScanner(io.LimitReader(logFile, readSize))
	scanner.Buffer(make([]byte, 4096), 512*1024) // 512KB max line size

	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		// Return to end for appending
		_, _ = logFile.Seek(0, io.SeekEnd)
		return nil, fmt.Errorf("failed to scan log file: %w", err)
	}

	// If we read from an offset, the first line might be partial - skip it
	if offset > 0 && len(lines) > 0 {
		lines = lines[1:]
	}

	// Apply limit
	if limit > 0 && len(lines) > limit {
		lines = lines[len(lines)-limit:]
	}

	// Parse lines into entries
	var entries []Entry
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 3)
		if len(parts) < 3 {
			continue
		}
		ts, err := time.Parse(time.RFC3339, parts[0])
		if err != nil {
			continue
		}
		level := Level(strings.Trim(parts[1], "[]"))
		msg := parts[2]
		entries = append(entries, Entry{Timestamp: ts, Level: level, Message: msg})
	}

	// Return to end for appending
	if _, err := logFile.Seek(0, io.SeekEnd); err != nil {
		return entries, fmt.Errorf("failed to seek to end: %w", err)
	}

	return entries, nil
}

func Close() {
	mu.Lock()
	defer mu.Unlock()

	if logWriter != nil {
		_ = logWriter.Flush()
		logWriter = nil
	}

	if logFile != nil {
		_ = logFile.Sync()
		_ = logFile.Close()
		logFile = nil
	}
}
