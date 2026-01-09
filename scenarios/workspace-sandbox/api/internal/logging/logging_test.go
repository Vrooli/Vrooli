package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	t.Run("creates logger with defaults", func(t *testing.T) {
		l := New("test-service")
		if l.service != "test-service" {
			t.Errorf("expected service 'test-service', got %s", l.service)
		}
		if l.level != LevelInfo {
			t.Errorf("expected default level Info, got %s", l.level)
		}
		if l.out == nil {
			t.Error("expected non-nil output writer")
		}
	})

	t.Run("applies WithLevel option", func(t *testing.T) {
		l := New("test", WithLevel(LevelDebug))
		if l.level != LevelDebug {
			t.Errorf("expected level Debug, got %s", l.level)
		}
	})

	t.Run("applies WithOutput option", func(t *testing.T) {
		buf := &bytes.Buffer{}
		l := New("test", WithOutput(buf))
		l.Info("test.event", "test message", nil)
		if buf.Len() == 0 {
			t.Error("expected output to custom writer")
		}
	})

	t.Run("applies multiple options", func(t *testing.T) {
		buf := &bytes.Buffer{}
		l := New("test", WithOutput(buf), WithLevel(LevelError))
		if l.level != LevelError {
			t.Errorf("expected level Error, got %s", l.level)
		}
		l.Info("test.event", "test message", nil)
		if buf.Len() != 0 {
			t.Error("expected no output for info when level is error")
		}
	})
}

func TestShouldLog(t *testing.T) {
	tests := []struct {
		configuredLevel Level
		logLevel        Level
		shouldLog       bool
	}{
		{LevelDebug, LevelDebug, true},
		{LevelDebug, LevelInfo, true},
		{LevelDebug, LevelWarn, true},
		{LevelDebug, LevelError, true},
		{LevelInfo, LevelDebug, false},
		{LevelInfo, LevelInfo, true},
		{LevelInfo, LevelWarn, true},
		{LevelInfo, LevelError, true},
		{LevelWarn, LevelDebug, false},
		{LevelWarn, LevelInfo, false},
		{LevelWarn, LevelWarn, true},
		{LevelWarn, LevelError, true},
		{LevelError, LevelDebug, false},
		{LevelError, LevelInfo, false},
		{LevelError, LevelWarn, false},
		{LevelError, LevelError, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.configuredLevel)+"_allows_"+string(tt.logLevel), func(t *testing.T) {
			l := New("test", WithLevel(tt.configuredLevel))
			if got := l.shouldLog(tt.logLevel); got != tt.shouldLog {
				t.Errorf("shouldLog(%s) with level %s = %v, want %v",
					tt.logLevel, tt.configuredLevel, got, tt.shouldLog)
			}
		})
	}
}

func TestLogOutput(t *testing.T) {
	t.Run("outputs valid JSON", func(t *testing.T) {
		buf := &bytes.Buffer{}
		l := New("test-service", WithOutput(buf))
		l.Info("test.event", "test message", nil)

		var entry Entry
		if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
			t.Fatalf("failed to parse JSON: %v\nOutput: %s", err, buf.String())
		}
	})

	t.Run("includes required fields", func(t *testing.T) {
		buf := &bytes.Buffer{}
		l := New("test-service", WithOutput(buf))
		l.Info("test.event", "test message", nil)

		var entry Entry
		if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		if entry.Level != LevelInfo {
			t.Errorf("expected level info, got %s", entry.Level)
		}
		if entry.Event != "test.event" {
			t.Errorf("expected event 'test.event', got %s", entry.Event)
		}
		if entry.Message != "test message" {
			t.Errorf("expected message 'test message', got %s", entry.Message)
		}
		if entry.Service != "test-service" {
			t.Errorf("expected service 'test-service', got %s", entry.Service)
		}
		if entry.Time == "" {
			t.Error("expected non-empty time")
		}
	})

	t.Run("extracts special fields", func(t *testing.T) {
		buf := &bytes.Buffer{}
		l := New("test-service", WithOutput(buf))
		l.Info("test.event", "test", map[string]interface{}{
			"sandboxId":  "sb-123",
			"actor":      "user-456",
			"durationMs": 42.5,
			"error":      "something went wrong",
			"extra":      "kept in fields",
		})

		var entry Entry
		if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		if entry.SandboxID != "sb-123" {
			t.Errorf("expected sandboxId 'sb-123', got %s", entry.SandboxID)
		}
		if entry.Actor != "user-456" {
			t.Errorf("expected actor 'user-456', got %s", entry.Actor)
		}
		if entry.Duration != 42.5 {
			t.Errorf("expected durationMs 42.5, got %f", entry.Duration)
		}
		if entry.Error != "something went wrong" {
			t.Errorf("expected error message, got %s", entry.Error)
		}
		if entry.Fields["extra"] != "kept in fields" {
			t.Errorf("expected extra field to be kept, got %v", entry.Fields)
		}
	})

	t.Run("appends newline", func(t *testing.T) {
		buf := &bytes.Buffer{}
		l := New("test", WithOutput(buf))
		l.Info("test", "test", nil)
		if !strings.HasSuffix(buf.String(), "\n") {
			t.Error("expected output to end with newline")
		}
	})
}

func TestLogLevels(t *testing.T) {
	t.Run("Debug", func(t *testing.T) {
		buf := &bytes.Buffer{}
		l := New("test", WithOutput(buf), WithLevel(LevelDebug))
		l.Debug("test.debug", "debug message", nil)

		var entry Entry
		if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
			t.Fatalf("failed to parse: %v", err)
		}
		if entry.Level != LevelDebug {
			t.Errorf("expected level debug, got %s", entry.Level)
		}
	})

	t.Run("Info", func(t *testing.T) {
		buf := &bytes.Buffer{}
		l := New("test", WithOutput(buf))
		l.Info("test.info", "info message", nil)

		var entry Entry
		if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
			t.Fatalf("failed to parse: %v", err)
		}
		if entry.Level != LevelInfo {
			t.Errorf("expected level info, got %s", entry.Level)
		}
	})

	t.Run("Warn", func(t *testing.T) {
		buf := &bytes.Buffer{}
		l := New("test", WithOutput(buf))
		l.Warn("test.warn", "warn message", nil)

		var entry Entry
		if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
			t.Fatalf("failed to parse: %v", err)
		}
		if entry.Level != LevelWarn {
			t.Errorf("expected level warn, got %s", entry.Level)
		}
	})

	t.Run("Error", func(t *testing.T) {
		buf := &bytes.Buffer{}
		l := New("test", WithOutput(buf))
		l.Error("test.error", "error message", nil)

		var entry Entry
		if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
			t.Fatalf("failed to parse: %v", err)
		}
		if entry.Level != LevelError {
			t.Errorf("expected level error, got %s", entry.Level)
		}
	})
}

func TestConvenienceMethods(t *testing.T) {
	t.Run("SandboxCreated", func(t *testing.T) {
		buf := &bytes.Buffer{}
		l := New("test", WithOutput(buf))
		l.SandboxCreated("sb-123", "/project/src", "user-1", 150.5)

		var entry Entry
		if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
			t.Fatalf("failed to parse: %v", err)
		}
		if entry.Event != "sandbox.created" {
			t.Errorf("expected event 'sandbox.created', got %s", entry.Event)
		}
		if entry.SandboxID != "sb-123" {
			t.Errorf("expected sandboxId 'sb-123', got %s", entry.SandboxID)
		}
		if entry.Actor != "user-1" {
			t.Errorf("expected actor 'user-1', got %s", entry.Actor)
		}
		if entry.Duration != 150.5 {
			t.Errorf("expected duration 150.5, got %f", entry.Duration)
		}
	})

	t.Run("SandboxMounted", func(t *testing.T) {
		buf := &bytes.Buffer{}
		l := New("test", WithOutput(buf))
		l.SandboxMounted("sb-123", "/merged/dir", 50.0)

		var entry Entry
		if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
			t.Fatalf("failed to parse: %v", err)
		}
		if entry.Event != "sandbox.mounted" {
			t.Errorf("expected event 'sandbox.mounted', got %s", entry.Event)
		}
	})

	t.Run("SandboxStopped", func(t *testing.T) {
		buf := &bytes.Buffer{}
		l := New("test", WithOutput(buf))
		l.SandboxStopped("sb-123", 25.0)

		var entry Entry
		if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
			t.Fatalf("failed to parse: %v", err)
		}
		if entry.Event != "sandbox.stopped" {
			t.Errorf("expected event 'sandbox.stopped', got %s", entry.Event)
		}
	})

	t.Run("SandboxApproved", func(t *testing.T) {
		buf := &bytes.Buffer{}
		l := New("test", WithOutput(buf))
		l.SandboxApproved("sb-123", "reviewer-1", 5, "abc123")

		var entry Entry
		if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
			t.Fatalf("failed to parse: %v", err)
		}
		if entry.Event != "sandbox.approved" {
			t.Errorf("expected event 'sandbox.approved', got %s", entry.Event)
		}
		if entry.Actor != "reviewer-1" {
			t.Errorf("expected actor 'reviewer-1', got %s", entry.Actor)
		}
	})

	t.Run("SandboxRejected", func(t *testing.T) {
		buf := &bytes.Buffer{}
		l := New("test", WithOutput(buf))
		l.SandboxRejected("sb-123", "reviewer-1")

		var entry Entry
		if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
			t.Fatalf("failed to parse: %v", err)
		}
		if entry.Event != "sandbox.rejected" {
			t.Errorf("expected event 'sandbox.rejected', got %s", entry.Event)
		}
	})

	t.Run("SandboxDeleted", func(t *testing.T) {
		buf := &bytes.Buffer{}
		l := New("test", WithOutput(buf))
		l.SandboxDeleted("sb-123", 10.0)

		var entry Entry
		if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
			t.Fatalf("failed to parse: %v", err)
		}
		if entry.Event != "sandbox.deleted" {
			t.Errorf("expected event 'sandbox.deleted', got %s", entry.Event)
		}
	})

	t.Run("SandboxError", func(t *testing.T) {
		buf := &bytes.Buffer{}
		l := New("test", WithOutput(buf))
		l.SandboxError("sb-123", "mount", errors.New("mount failed"))

		var entry Entry
		if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
			t.Fatalf("failed to parse: %v", err)
		}
		if entry.Event != "sandbox.error" {
			t.Errorf("expected event 'sandbox.error', got %s", entry.Event)
		}
		if entry.Level != LevelError {
			t.Errorf("expected level error, got %s", entry.Level)
		}
		if entry.Error != "mount failed" {
			t.Errorf("expected error 'mount failed', got %s", entry.Error)
		}
	})

	t.Run("DriverError", func(t *testing.T) {
		buf := &bytes.Buffer{}
		l := New("test", WithOutput(buf))
		l.DriverError("unmount", errors.New("device busy"), map[string]interface{}{
			"path": "/some/path",
		})

		var entry Entry
		if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
			t.Fatalf("failed to parse: %v", err)
		}
		if entry.Event != "driver.error" {
			t.Errorf("expected event 'driver.error', got %s", entry.Event)
		}
		if entry.Error != "device busy" {
			t.Errorf("expected error 'device busy', got %s", entry.Error)
		}
	})

	t.Run("APIRequest", func(t *testing.T) {
		buf := &bytes.Buffer{}
		l := New("test", WithOutput(buf))
		l.APIRequest("GET", "/api/v1/sandboxes", 200, 15.5)

		var entry Entry
		if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
			t.Fatalf("failed to parse: %v", err)
		}
		if entry.Event != "api.request" {
			t.Errorf("expected event 'api.request', got %s", entry.Event)
		}
		if entry.Duration != 15.5 {
			t.Errorf("expected duration 15.5, got %f", entry.Duration)
		}
	})

	t.Run("PolicyValidation passed", func(t *testing.T) {
		buf := &bytes.Buffer{}
		l := New("test", WithOutput(buf))
		l.PolicyValidation("auto_approve", "sb-123", true, "within limits")

		var entry Entry
		if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
			t.Fatalf("failed to parse: %v", err)
		}
		if entry.Event != "policy.passed" {
			t.Errorf("expected event 'policy.passed', got %s", entry.Event)
		}
	})

	t.Run("PolicyValidation failed", func(t *testing.T) {
		buf := &bytes.Buffer{}
		l := New("test", WithOutput(buf))
		l.PolicyValidation("auto_approve", "sb-123", false, "too many files")

		var entry Entry
		if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
			t.Fatalf("failed to parse: %v", err)
		}
		if entry.Event != "policy.failed" {
			t.Errorf("expected event 'policy.failed', got %s", entry.Event)
		}
	})
}

func TestContextOperations(t *testing.T) {
	t.Run("WithLogger stores logger in context", func(t *testing.T) {
		l := New("test-service")
		ctx := WithLogger(context.Background(), l)
		if ctx == nil {
			t.Fatal("expected non-nil context")
		}
	})

	t.Run("FromContext retrieves stored logger", func(t *testing.T) {
		buf := &bytes.Buffer{}
		l := New("specific-service", WithOutput(buf))
		ctx := WithLogger(context.Background(), l)

		retrieved := FromContext(ctx)
		retrieved.Info("test", "test", nil)

		var entry Entry
		if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
			t.Fatalf("failed to parse: %v", err)
		}
		if entry.Service != "specific-service" {
			t.Errorf("expected service 'specific-service', got %s", entry.Service)
		}
	})

	t.Run("FromContext returns default logger when not set", func(t *testing.T) {
		retrieved := FromContext(context.Background())
		if retrieved == nil {
			t.Fatal("expected non-nil logger")
		}
		if retrieved.service != "workspace-sandbox" {
			t.Errorf("expected default service 'workspace-sandbox', got %s", retrieved.service)
		}
	})
}

func TestConcurrentLogging(t *testing.T) {
	buf := &bytes.Buffer{}
	l := New("test", WithOutput(buf))

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(n int) {
			for j := 0; j < 100; j++ {
				l.Info("test.concurrent", "message", map[string]interface{}{
					"goroutine": n,
					"iteration": j,
				})
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify we can parse each line
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 1000 {
		t.Errorf("expected 1000 log lines, got %d", len(lines))
	}

	for i, line := range lines {
		var entry Entry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Errorf("line %d failed to parse: %v", i, err)
		}
	}
}

func TestNilFields(t *testing.T) {
	t.Run("handles nil fields map", func(t *testing.T) {
		buf := &bytes.Buffer{}
		l := New("test", WithOutput(buf))
		l.Info("test", "test", nil)

		var entry Entry
		if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
			t.Fatalf("failed to parse: %v", err)
		}
	})

	t.Run("handles empty fields map", func(t *testing.T) {
		buf := &bytes.Buffer{}
		l := New("test", WithOutput(buf))
		l.Info("test", "test", map[string]interface{}{})

		var entry Entry
		if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
			t.Fatalf("failed to parse: %v", err)
		}
	})
}

func TestLevelConstants(t *testing.T) {
	if LevelDebug != "debug" {
		t.Errorf("LevelDebug = %s, want 'debug'", LevelDebug)
	}
	if LevelInfo != "info" {
		t.Errorf("LevelInfo = %s, want 'info'", LevelInfo)
	}
	if LevelWarn != "warn" {
		t.Errorf("LevelWarn = %s, want 'warn'", LevelWarn)
	}
	if LevelError != "error" {
		t.Errorf("LevelError = %s, want 'error'", LevelError)
	}
}
