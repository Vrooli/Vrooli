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
func (l *Logger) Info(msg string) {
	l.Printf("INFO: %s", msg)
}

// Error logs error messages
func (l *Logger) Error(msg string, err error) {
	if err != nil {
		l.Printf("ERROR: %s: %v", msg, err)
	} else {
		l.Printf("ERROR: %s", msg)
	}
}

// Warning logs warning messages  
func (l *Logger) Warning(msg string) {
	l.Printf("WARNING: %s", msg)
}

// Debug logs debug messages (only if DEBUG env var is set)
func (l *Logger) Debug(msg string) {
	if os.Getenv("DEBUG") != "" {
		l.Printf("DEBUG: %s", msg)
	}
}