package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// LogLevel represents the severity of a log message
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

// Logger provides structured logging with consistent formatting
type Logger struct {
	component string
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     LogLevel               `json:"level"`
	Component string                 `json:"component"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// NewLogger creates a new logger for a specific component
func NewLogger(component string) *Logger {
	return &Logger{component: component}
}

// log writes a structured log entry
func (l *Logger) log(level LogLevel, message string, fields map[string]interface{}) {
	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     level,
		Component: l.component,
		Message:   message,
		Fields:    fields,
	}

	// Write as JSON for structured logging
	data, err := json.Marshal(entry)
	if err != nil {
		// Fallback to simple format if JSON marshaling fails
		fmt.Fprintf(os.Stderr, "[%s] %s [%s] %s\n", entry.Timestamp, level, l.component, message)
		return
	}

	fmt.Fprintln(os.Stdout, string(data))
}

// Debug logs a debug-level message
func (l *Logger) Debug(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(LogLevelDebug, message, f)
}

// Info logs an info-level message
func (l *Logger) Info(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(LogLevelInfo, message, f)
}

// Warn logs a warning-level message
func (l *Logger) Warn(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(LogLevelWarn, message, f)
}

// Error logs an error-level message
func (l *Logger) Error(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(LogLevelError, message, f)
}

// Global logger instances for different components
var (
	mainLogger  *Logger
	cacheLogger *Logger
	dbLogger    *Logger
	nlpLogger   *Logger
	recLogger   *Logger
)

// initLoggers initializes all component loggers
func initLoggers() {
	mainLogger = NewLogger("main")
	cacheLogger = NewLogger("cache")
	dbLogger = NewLogger("database")
	nlpLogger = NewLogger("nlp")
	recLogger = NewLogger("recommendations")
}
