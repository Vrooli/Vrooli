package main

import (
	"fmt"
	"log"
	"os"
)

// Logger provides structured logging for the secrets manager
type Logger struct {
	*log.Logger
}

// NewLogger creates a new structured logger
func NewLogger(prefix string) *Logger {
	return &Logger{
		Logger: log.New(os.Stdout, fmt.Sprintf("[%s] ", prefix), log.LstdFlags|log.Lshortfile),
	}
}

// Info logs informational messages
func (l *Logger) Info(format string, args ...interface{}) {
	if len(args) == 0 {
		l.Printf("INFO: %s", format)
	} else {
		l.Printf("INFO: "+format, args...)
	}
}

// Error logs error messages
func (l *Logger) Error(format string, args ...interface{}) {
	if len(args) == 0 {
		l.Printf("ERROR: %s", format)
	} else {
		l.Printf("ERROR: "+format, args...)
	}
}

// Warning logs warning messages
func (l *Logger) Warning(format string, args ...interface{}) {
	if len(args) == 0 {
		l.Printf("WARNING: %s", format)
	} else {
		l.Printf("WARNING: "+format, args...)
	}
}

// Debug logs debug messages (only if DEBUG env var is set)
func (l *Logger) Debug(format string, args ...interface{}) {
	if os.Getenv("DEBUG") != "" {
		if len(args) == 0 {
			l.Printf("DEBUG: %s", format)
		} else {
			l.Printf("DEBUG: "+format, args...)
		}
	}
}
