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
	"net/http/httptest"
	"testing"

	"swarm-manager/internal/testutil/assertx"
	"swarm-manager/internal/testutil/fsx"
	"swarm-manager/internal/testutil/httpx"

	"google.golang.org/protobuf/proto"
)

// WriteJSONFile creates a JSON file at the specified path.
// Creates parent directories if they don't exist.
func WriteJSONFile(t *testing.T, path string, data any) {
	t.Helper()
	fsx.WriteJSONFile(t, path, data)
}

// WriteFile creates a file at the specified path with the given content.
// Creates parent directories if they don't exist.
func WriteFile(t *testing.T, path string, content string) {
	t.Helper()
	fsx.WriteFile(t, path, content)
}

// MakeDir creates a directory at the specified path.
func MakeDir(t *testing.T, path string) {
	t.Helper()
	fsx.MakeDir(t, path)
}

// AssertStatus checks that the response has the expected HTTP status code.
func AssertStatus(t *testing.T, rec *httptest.ResponseRecorder, expected int) {
	t.Helper()
	assertx.Status(t, rec, expected)
}

// AssertStatusOK checks that the response has HTTP 200 status.
func AssertStatusOK(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	assertx.StatusOK(t, rec)
}

// AssertStatusCreated checks that the response has HTTP 201 status.
func AssertStatusCreated(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	assertx.StatusCreated(t, rec)
}

// AssertStatusNotFound checks that the response has HTTP 404 status.
func AssertStatusNotFound(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	assertx.StatusNotFound(t, rec)
}

// AssertStatusBadRequest checks that the response has HTTP 400 status.
func AssertStatusBadRequest(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	assertx.StatusBadRequest(t, rec)
}

// DecodeJSON decodes the response body as JSON into the provided value.
func DecodeJSON[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	return httpx.DecodeJSON[T](t, rec)
}

// DecodeProtoJSON decodes the response body as proto JSON into the provided message.
func DecodeProtoJSON[T proto.Message](t *testing.T, rec *httptest.ResponseRecorder, msg T) T {
	t.Helper()
	return httpx.DecodeProtoJSON(t, rec, msg)
}

// ReadJSONFile reads and decodes a JSON file into the provided value.
func ReadJSONFile[T any](t *testing.T, path string) T {
	t.Helper()
	return fsx.ReadJSONFile[T](t, path)
}

// AssertFileExists checks that a file exists at the specified path.
func AssertFileExists(t *testing.T, path string) {
	t.Helper()
	fsx.AssertFileExists(t, path)
}

// AssertFileNotExists checks that a file does not exist at the specified path.
func AssertFileNotExists(t *testing.T, path string) {
	t.Helper()
	fsx.AssertFileNotExists(t, path)
}
