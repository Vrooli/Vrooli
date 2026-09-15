package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
)

var (
	logSensitiveAssignment = regexp.MustCompile(`(?i)(\b(?:password|passphrase|secret|token|api[_-]?key|access[_-]?token|refresh[_-]?token|authorization|cookie|set-cookie|private[_-]?key|credential|plaintext|ciphertext|request[_-]?body|response[_-]?body)\b\s*[:=]\s*)(["']?)([^"'\s,}&]+)(["']?)`)
	logSensitiveBearer     = regexp.MustCompile(`(?i)(\b(?:authorization|token|access[_-]?token|credential)\b\s*[:=]\s*)(["']?)(bearer\s+[^"'\s,}&]+)(["']?)`)
	logSensitiveQuery      = regexp.MustCompile(`(?i)([?&](?:password|passphrase|secret|token|api[_-]?key|access[_-]?token|refresh[_-]?token|authorization|credential)=)([^&\s]+)`)
	logBearerToken         = regexp.MustCompile(`(?i)(\bbearer\s+)[^\s,]+`)
	formatLogMessage       = fmt.Sprintf
)

// RedactSensitiveText is the last logging boundary for values that should not
// enter generic process logs. Callers should still avoid passing secret values
// to the logger; this guard handles accidental key/value and bearer-token
// formatting without pretending it can identify arbitrary plaintext.
func RedactSensitiveText(value string) string {
	value = logSensitiveBearer.ReplaceAllString(value, `${1}${2}[REDACTED]${4}`)
	value = logSensitiveAssignment.ReplaceAllString(value, `${1}${2}[REDACTED]${4}`)
	value = logSensitiveQuery.ReplaceAllString(value, `${1}[REDACTED]`)
	return logBearerToken.ReplaceAllString(value, `${1}[REDACTED]`)
}

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
func (l *Logger) Info(message string, args ...interface{}) {
	if len(args) == 0 {
		l.Printf("INFO: %s", RedactSensitiveText(message))
	} else {
		l.Printf("INFO: %s", RedactSensitiveText(formatLogMessage(message, args...)))
	}
}

// Error logs error messages
func (l *Logger) Error(message string, args ...interface{}) {
	if len(args) == 0 {
		l.Printf("ERROR: %s", RedactSensitiveText(message))
	} else {
		l.Printf("ERROR: %s", RedactSensitiveText(formatLogMessage(message, args...)))
	}
}

// Warning logs warning messages
func (l *Logger) Warning(message string, args ...interface{}) {
	if len(args) == 0 {
		l.Printf("WARNING: %s", RedactSensitiveText(message))
	} else {
		l.Printf("WARNING: %s", RedactSensitiveText(formatLogMessage(message, args...)))
	}
}

// Debug logs debug messages (only if DEBUG env var is set)
func (l *Logger) Debug(message string, args ...interface{}) {
	if os.Getenv("DEBUG") != "" {
		if len(args) == 0 {
			l.Printf("DEBUG: %s", RedactSensitiveText(message))
		} else {
			l.Printf("DEBUG: %s", RedactSensitiveText(formatLogMessage(message, args...)))
		}
	}
}
