package main

import (
	"bytes"
	"log"
	"os"
	"strings"
	"testing"
)

// TestNewLogger tests logger initialization
// [REQ:SEC-OPS-001] Structured logging support
func TestNewLogger(t *testing.T) {
	tests := []struct {
		name       string
		prefix     string
		wantPrefix bool
	}{
		{
			name:       "Logger with prefix",
			prefix:     "TEST",
			wantPrefix: true,
		},
		{
			name:       "Logger without prefix",
			prefix:     "",
			wantPrefix: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLogger(tt.prefix)
			if logger == nil {
				t.Error("NewLogger() returned nil")
			}
		})
	}
}

// TestLoggerInfo tests Info logging
// [REQ:SEC-OPS-001] Structured logging support
func TestLoggerInfo(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	testLogger := &Logger{
		Logger: log.New(&buf, "[TEST] ", log.LstdFlags),
	}

	testLogger.Info("Test info message")

	output := buf.String()
	if !strings.Contains(output, "INFO") {
		t.Errorf("Info log should contain INFO, got: %s", output)
	}
	if !strings.Contains(output, "Test info message") {
		t.Errorf("Info log should contain message, got: %s", output)
	}
}

// TestLoggerError tests Error logging
// [REQ:SEC-OPS-001] Structured logging support
func TestLoggerError(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	testLogger := &Logger{
		Logger: log.New(&buf, "[TEST] ", log.LstdFlags),
	}

	testLogger.Error("Test error message", nil)

	output := buf.String()
	if !strings.Contains(output, "ERROR") {
		t.Errorf("Error log should contain ERROR, got: %s", output)
	}
	if !strings.Contains(output, "Test error message") {
		t.Errorf("Error log should contain message, got: %s", output)
	}
}

// TestLoggerWarning tests Warning logging
// [REQ:SEC-OPS-001] Structured logging support
func TestLoggerWarning(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	testLogger := &Logger{
		Logger: log.New(&buf, "[TEST] ", log.LstdFlags),
	}

	testLogger.Warning("Test warning message")

	output := buf.String()
	if !strings.Contains(output, "WARNING") {
		t.Errorf("Warning log should contain WARNING, got: %s", output)
	}
	if !strings.Contains(output, "Test warning message") {
		t.Errorf("Warning log should contain message, got: %s", output)
	}
}

// TestLoggerDebug tests Debug logging
// [REQ:SEC-OPS-001] Structured logging support
func TestLoggerDebug(t *testing.T) {
	// Save and restore DEBUG env var
	oldDebug := os.Getenv("DEBUG")
	defer func() {
		if oldDebug != "" {
			os.Setenv("DEBUG", oldDebug)
		} else {
			os.Unsetenv("DEBUG")
		}
	}()

	tests := []struct {
		name      string
		debugMode string
		wantLog   bool
	}{
		{
			name:      "Debug enabled",
			debugMode: "true",
			wantLog:   true,
		},
		{
			name:      "Debug disabled",
			debugMode: "",
			wantLog:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.debugMode != "" {
				os.Setenv("DEBUG", tt.debugMode)
			} else {
				os.Unsetenv("DEBUG")
			}

			var buf bytes.Buffer
			testLogger := &Logger{
				Logger: log.New(&buf, "[TEST] ", log.LstdFlags),
			}

			testLogger.Debug("Test debug message")

			output := buf.String()
			if tt.wantLog {
				if !strings.Contains(output, "DEBUG") {
					t.Errorf("Debug log should contain DEBUG when enabled, got: %s", output)
				}
				if !strings.Contains(output, "Test debug message") {
					t.Errorf("Debug log should contain message when enabled, got: %s", output)
				}
			} else {
				if output != "" {
					t.Errorf("Debug log should be empty when disabled, got: %s", output)
				}
			}
		})
	}
}
