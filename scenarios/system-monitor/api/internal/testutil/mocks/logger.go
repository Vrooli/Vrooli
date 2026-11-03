package mocks

import (
	"sync"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/logx"
)

type LogEntry struct {
	Level   string
	Message string
	Args    []any
}

// Logger records structured diagnostics for deterministic service tests.
type Logger struct {
	mu      sync.Mutex
	Entries []LogEntry
}

var _ logx.Logger = (*Logger)(nil)

func (l *Logger) Debug(message string, args ...any) { l.record("debug", message, args) }
func (l *Logger) Info(message string, args ...any)  { l.record("info", message, args) }
func (l *Logger) Warn(message string, args ...any)  { l.record("warn", message, args) }
func (l *Logger) Error(message string, args ...any) { l.record("error", message, args) }

func (l *Logger) record(level, message string, args []any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.Entries = append(l.Entries, LogEntry{Level: level, Message: message, Args: append([]any(nil), args...)})
}

func (l *Logger) Contains(level, message string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, entry := range l.Entries {
		if entry.Level == level && entry.Message == message {
			return true
		}
	}
	return false
}
