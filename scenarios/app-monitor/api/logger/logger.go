package logger

import (
	"log"
	"os"
)

// Logger provides structured logging with different levels
type Logger struct {
	prefix string
}

// New creates a new logger instance with an optional prefix
func New(prefix string) *Logger {
	return &Logger{prefix: prefix}
}

// Default logger instance
var defaultLogger = New("")

// Info logs informational messages
func Info(msg string, args ...interface{}) {
	defaultLogger.Info(msg, args...)
}

// Warn logs warning messages
func Warn(msg string, args ...interface{}) {
	defaultLogger.Warn(msg, args...)
}

// Error logs error messages
func Error(msg string, args ...interface{}) {
	defaultLogger.Error(msg, args...)
}

// Fatal logs a fatal error and exits
func Fatal(msg string, args ...interface{}) {
	defaultLogger.Fatal(msg, args...)
}

// Info logs informational messages
func (l *Logger) Info(msg string, args ...interface{}) {
	l.logWithLevel("INFO", msg, args...)
}

// Warn logs warning messages
func (l *Logger) Warn(msg string, args ...interface{}) {
	l.logWithLevel("WARN", msg, args...)
}

// Error logs error messages
func (l *Logger) Error(msg string, args ...interface{}) {
	l.logWithLevel("ERROR", msg, args...)
}

// Fatal logs a fatal error and exits
func (l *Logger) Fatal(msg string, args ...interface{}) {
	l.logWithLevel("FATAL", msg, args...)
	os.Exit(1)
}

// logWithLevel is the internal logging implementation
func (l *Logger) logWithLevel(level string, msg string, args ...interface{}) {
	prefix := ""
	if l.prefix != "" {
		prefix = "[" + l.prefix + "] "
	}

	// Format the message with args if provided
	formatted := msg
	if len(args) > 0 {
		// Support both "%v" style and key-value pairs
		// For simplicity, we'll use Printf-style formatting
		formatted = msg
		for i := 0; i < len(args); i++ {
			if err, ok := args[i].(error); ok {
				log.Printf("[%s] %s%s: %v", level, prefix, formatted, err)
				return
			}
		}
		// If no error, just append args
		if len(args) > 0 {
			log.Printf("[%s] %s%s %v", level, prefix, formatted, args)
		} else {
			log.Printf("[%s] %s%s", level, prefix, formatted)
		}
	} else {
		log.Printf("[%s] %s%s", level, prefix, formatted)
	}
}
