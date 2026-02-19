package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
)

// [REQ:OBS-003] Structured logging tests

func TestInitStructuredLogging(t *testing.T) {
	// Should not panic
	InitStructuredLogging(slog.LevelInfo)
}

func TestStructuredLog_OutputsJSON(t *testing.T) {
	// Set up a buffer to capture log output
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := slog.New(handler)
	slog.SetDefault(logger)

	StructuredLog(slog.LevelInfo, "tunnel", "check_health", "ok", 42, nil)

	// Parse the JSON output
	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	// Verify required fields
	if entry["component"] != "tunnel" {
		t.Errorf("component: want 'tunnel', got %v", entry["component"])
	}
	if entry["action"] != "check_health" {
		t.Errorf("action: want 'check_health', got %v", entry["action"])
	}
	if entry["result"] != "ok" {
		t.Errorf("result: want 'ok', got %v", entry["result"])
	}
	if entry["duration_ms"] != float64(42) {
		t.Errorf("duration_ms: want 42, got %v", entry["duration_ms"])
	}
	if _, hasError := entry["error"]; hasError {
		t.Error("should not have 'error' field when err is nil")
	}
}

func TestStructuredLog_WithError(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	slog.SetDefault(slog.New(handler))

	StructuredLog(slog.LevelError, "probe", "run_probe", "failure", 500, errors.New("connection refused"))

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if entry["error"] != "connection refused" {
		t.Errorf("error: want 'connection refused', got %v", entry["error"])
	}
}
