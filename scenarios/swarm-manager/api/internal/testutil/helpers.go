// DOC: docs/reference/operational-targets.md
// DOC: docs/concepts/ARCHITECTURE.md
// Package testutil provides shared testing utilities for API handlers.
//
// This package consolidates common test patterns:
//   - Temp directory setup (using t.TempDir for automatic cleanup)
//   - JSON file creation and assertion helpers
//   - Common assertion functions
//
// Design Goals:
//   - Reduce boilerplate in test files
//   - Encourage consistent testing patterns
//   - Use Go's built-in t.TempDir() for automatic cleanup (no manual defer needed)
package testutil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// WriteJSONFile creates a JSON file at the specified path.
// Creates parent directories if they don't exist.
func WriteJSONFile(t *testing.T, path string, data any) {
	t.Helper()
	parentDir := filepath.Dir(path)
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		t.Fatalf("Failed to create parent directory %s: %v", parentDir, err)
	}
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal JSON for %s: %v", path, err)
	}
	if err := os.WriteFile(path, bytes, 0o644); err != nil {
		t.Fatalf("Failed to write file %s: %v", path, err)
	}
}

// WriteFile creates a file at the specified path with the given content.
// Creates parent directories if they don't exist.
func WriteFile(t *testing.T, path string, content string) {
	t.Helper()
	parentDir := filepath.Dir(path)
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		t.Fatalf("Failed to create parent directory %s: %v", parentDir, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write file %s: %v", path, err)
	}
}

// MakeDir creates a directory at the specified path.
func MakeDir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("Failed to create directory %s: %v", path, err)
	}
}

// AssertStatus checks that the response has the expected HTTP status code.
func AssertStatus(t *testing.T, rec *httptest.ResponseRecorder, expected int) {
	t.Helper()
	if rec.Code != expected {
		t.Errorf("Expected status %d, got %d: %s", expected, rec.Code, rec.Body.String())
	}
}

// AssertStatusOK checks that the response has HTTP 200 status.
func AssertStatusOK(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	AssertStatus(t, rec, http.StatusOK)
}

// AssertStatusCreated checks that the response has HTTP 201 status.
func AssertStatusCreated(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	AssertStatus(t, rec, http.StatusCreated)
}

// AssertStatusNotFound checks that the response has HTTP 404 status.
func AssertStatusNotFound(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	AssertStatus(t, rec, http.StatusNotFound)
}

// AssertStatusBadRequest checks that the response has HTTP 400 status.
func AssertStatusBadRequest(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	AssertStatus(t, rec, http.StatusBadRequest)
}

// DecodeJSON decodes the response body as JSON into the provided value.
func DecodeJSON[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var result T
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response JSON: %v", err)
	}
	return result
}

// DecodeProtoJSON decodes the response body as proto JSON into the provided message.
func DecodeProtoJSON[T proto.Message](t *testing.T, rec *httptest.ResponseRecorder, msg T) T {
	t.Helper()
	opts := protojson.UnmarshalOptions{DiscardUnknown: true}
	if err := opts.Unmarshal(rec.Body.Bytes(), msg); err != nil {
		t.Fatalf("Failed to decode proto JSON: %v", err)
	}
	return msg
}

// ReadJSONFile reads and decodes a JSON file into the provided value.
func ReadJSONFile[T any](t *testing.T, path string) T {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", path, err)
	}
	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to decode JSON from %s: %v", path, err)
	}
	return result
}

// AssertFileExists checks that a file exists at the specified path.
func AssertFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("Expected file to exist: %s", path)
	}
}

// AssertFileNotExists checks that a file does not exist at the specified path.
func AssertFileNotExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("Expected file to not exist: %s", path)
	}
}
