package main

import (
	"bytes"
	"log"
	"os"
	"strings"
	"sync"
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
		{
			name:       "Logger with special characters",
			prefix:     "TEST-123_API",
			wantPrefix: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLogger(tt.prefix)
			if logger == nil {
				t.Fatal("NewLogger() returned nil")
			}
			if logger.Logger == nil {
				t.Error("NewLogger() returned logger with nil underlying Logger")
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

// TestLoggerInfoWithFormatArgs tests Info logging with format arguments
// [REQ:SEC-OPS-001] Structured logging support
func TestLoggerInfoWithFormatArgs(t *testing.T) {
	var buf bytes.Buffer
	testLogger := &Logger{
		Logger: log.New(&buf, "[TEST] ", log.LstdFlags),
	}

	testLogger.Info("User %s logged in from %s", "admin", "192.168.1.1")

	output := buf.String()
	if !strings.Contains(output, "INFO") {
		t.Errorf("Info log should contain INFO, got: %s", output)
	}
	if !strings.Contains(output, "admin") {
		t.Errorf("Info log should contain formatted username, got: %s", output)
	}
	if !strings.Contains(output, "192.168.1.1") {
		t.Errorf("Info log should contain formatted IP, got: %s", output)
	}
}

// TestLoggerErrorWithFormatArgs tests Error logging with format arguments
// [REQ:SEC-OPS-001] Structured logging support
func TestLoggerErrorWithFormatArgs(t *testing.T) {
	var buf bytes.Buffer
	testLogger := &Logger{
		Logger: log.New(&buf, "[TEST] ", log.LstdFlags),
	}

	testLogger.Error("Failed to connect: %s (attempt %d)", "connection refused", 3)

	output := buf.String()
	if !strings.Contains(output, "ERROR") {
		t.Errorf("Error log should contain ERROR, got: %s", output)
	}
	if !strings.Contains(output, "connection refused") {
		t.Errorf("Error log should contain formatted error, got: %s", output)
	}
	if !strings.Contains(output, "3") {
		t.Errorf("Error log should contain formatted attempt count, got: %s", output)
	}
}

// TestLoggerWarningWithFormatArgs tests Warning logging with format arguments
// [REQ:SEC-OPS-001] Structured logging support
func TestLoggerWarningWithFormatArgs(t *testing.T) {
	var buf bytes.Buffer
	testLogger := &Logger{
		Logger: log.New(&buf, "[TEST] ", log.LstdFlags),
	}

	testLogger.Warning("Resource %s is at %d%% capacity", "memory", 85)

	output := buf.String()
	if !strings.Contains(output, "WARNING") {
		t.Errorf("Warning log should contain WARNING, got: %s", output)
	}
	if !strings.Contains(output, "memory") {
		t.Errorf("Warning log should contain formatted resource, got: %s", output)
	}
	if !strings.Contains(output, "85") {
		t.Errorf("Warning log should contain formatted percentage, got: %s", output)
	}
}

// TestLoggerDebugWithFormatArgs tests Debug logging with format arguments
// [REQ:SEC-OPS-001] Structured logging support
func TestLoggerDebugWithFormatArgs(t *testing.T) {
	oldDebug := os.Getenv("DEBUG")
	os.Setenv("DEBUG", "true")
	defer func() {
		if oldDebug != "" {
			os.Setenv("DEBUG", oldDebug)
		} else {
			os.Unsetenv("DEBUG")
		}
	}()

	var buf bytes.Buffer
	testLogger := &Logger{
		Logger: log.New(&buf, "[TEST] ", log.LstdFlags),
	}

	testLogger.Debug("Processing item %d of %d", 5, 10)

	output := buf.String()
	if !strings.Contains(output, "DEBUG") {
		t.Errorf("Debug log should contain DEBUG, got: %s", output)
	}
	if !strings.Contains(output, "5") {
		t.Errorf("Debug log should contain formatted current item, got: %s", output)
	}
	if !strings.Contains(output, "10") {
		t.Errorf("Debug log should contain formatted total, got: %s", output)
	}
}

// TestLoggerEmptyMessage tests logging with empty messages
// [REQ:SEC-OPS-001] Structured logging support
func TestLoggerEmptyMessage(t *testing.T) {
	var buf bytes.Buffer
	testLogger := &Logger{
		Logger: log.New(&buf, "[TEST] ", log.LstdFlags),
	}

	testLogger.Info("")
	output := buf.String()
	if !strings.Contains(output, "INFO:") {
		t.Errorf("Empty Info log should still contain INFO prefix, got: %s", output)
	}
}

// TestLoggerSpecialCharacters tests logging with special characters
// [REQ:SEC-OPS-001] Structured logging support
func TestLoggerSpecialCharacters(t *testing.T) {
	var buf bytes.Buffer
	testLogger := &Logger{
		Logger: log.New(&buf, "[TEST] ", log.LstdFlags),
	}

	specialMessage := "Secret with special chars: @#$%^&*(){}[]|\\:\";<>?,./`~"
	testLogger.Info(specialMessage)

	output := buf.String()
	if !strings.Contains(output, "INFO") {
		t.Errorf("Log should contain INFO, got: %s", output)
	}
	// Verify the message wasn't corrupted
	if !strings.Contains(output, "@#$%^&*()") {
		t.Errorf("Log should preserve special characters, got: %s", output)
	}
}

// TestLoggerConcurrentWrites tests that logger handles concurrent writes safely
// [REQ:SEC-OPS-001] Structured logging support
func TestLoggerConcurrentWrites(t *testing.T) {
	var buf bytes.Buffer
	testLogger := &Logger{
		Logger: log.New(&buf, "[TEST] ", log.LstdFlags),
	}

	var wg sync.WaitGroup
	iterations := 100

	// Launch multiple goroutines writing concurrently
	for i := 0; i < iterations; i++ {
		wg.Add(4)
		go func(n int) {
			defer wg.Done()
			testLogger.Info("Concurrent info %d", n)
		}(i)
		go func(n int) {
			defer wg.Done()
			testLogger.Error("Concurrent error %d", n)
		}(i)
		go func(n int) {
			defer wg.Done()
			testLogger.Warning("Concurrent warning %d", n)
		}(i)
		go func(n int) {
			defer wg.Done()
			testLogger.Debug("Concurrent debug %d", n)
		}(i)
	}

	wg.Wait()

	output := buf.String()
	// Should have logged Info, Error, and Warning entries (Debug only if DEBUG env is set)
	infoCount := strings.Count(output, "INFO:")
	errorCount := strings.Count(output, "ERROR:")
	warningCount := strings.Count(output, "WARNING:")

	if infoCount != iterations {
		t.Errorf("Expected %d INFO logs, got %d", iterations, infoCount)
	}
	if errorCount != iterations {
		t.Errorf("Expected %d ERROR logs, got %d", iterations, errorCount)
	}
	if warningCount != iterations {
		t.Errorf("Expected %d WARNING logs, got %d", iterations, warningCount)
	}
}

// TestLoggerMultilineMessage tests logging with multiline messages
// [REQ:SEC-OPS-001] Structured logging support
func TestLoggerMultilineMessage(t *testing.T) {
	var buf bytes.Buffer
	testLogger := &Logger{
		Logger: log.New(&buf, "[TEST] ", log.LstdFlags),
	}

	multilineMessage := "Line 1\nLine 2\nLine 3"
	testLogger.Info(multilineMessage)

	output := buf.String()
	if !strings.Contains(output, "INFO") {
		t.Errorf("Log should contain INFO, got: %s", output)
	}
	if !strings.Contains(output, "Line 1") {
		t.Errorf("Log should contain first line, got: %s", output)
	}
}
