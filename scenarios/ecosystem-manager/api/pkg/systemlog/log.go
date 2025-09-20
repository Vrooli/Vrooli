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
)

type Level string

const (
	LevelDebug Level = "DEBUG"
	LevelInfo  Level = "INFO"
	LevelWarn  Level = "WARN"
	LevelError Level = "ERROR"
)

type Entry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     Level     `json:"level"`
	Message   string    `json:"message"`
}

var (
	logDir   string
	logFile  *os.File
	mu       sync.Mutex
	initOnce sync.Once
)

func Init() {
	initOnce.Do(func() {
		cwd, err := os.Getwd()
		if err != nil {
			cwd = "."
		}
		logDir = filepath.Join(filepath.Dir(cwd), "logs")
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
	})
}

func write(level Level, msg string) {
	if logFile == nil {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	entry := fmt.Sprintf("%s [%s] %s\n", time.Now().Format(time.RFC3339), level, msg)
	_, _ = logFile.WriteString(entry)
}

func Debugf(format string, args ...interface{}) { write(LevelDebug, fmt.Sprintf(format, args...)) }
func Info(msg string)                           { write(LevelInfo, msg) }
func Infof(format string, args ...interface{})  { write(LevelInfo, fmt.Sprintf(format, args...)) }
func Warnf(format string, args ...interface{})  { write(LevelWarn, fmt.Sprintf(format, args...)) }
func Errorf(format string, args ...interface{}) { write(LevelError, fmt.Sprintf(format, args...)) }

func RecentEntries(limit int) ([]Entry, error) {
	if logFile == nil {
		return nil, fmt.Errorf("log file not initialized")
	}

	mu.Lock()
	defer mu.Unlock()

	if _, err := logFile.Seek(0, 0); err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(logFile)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if limit > 0 && len(lines) > limit {
		lines = lines[len(lines)-limit:]
	}

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
	if _, err := logFile.Seek(0, io.SeekEnd); err != nil {
		return entries, err
	}

	return entries, nil
}

func Close() {
	mu.Lock()
	defer mu.Unlock()
	if logFile != nil {
		logFile.Close()
		logFile = nil
	}
}
