// DOC: docs/internal/UNIT_TEST_ARCHITECTURE.md
// DOC: docs/internal/ERROR_SEMANTICS.md
// Package testutil provides testing utilities and helpers for unit tests.
// It centralizes common test setup patterns, assertions, and fixtures.
package testutil

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Standard test errors for error injection testing.
var (
	// ErrDatabase simulates a database connection error for testing internal error paths.
	ErrDatabase = errors.New("database connection error")

	// ErrTimeout simulates a timeout error for testing retry logic.
	ErrTimeout = errors.New("operation timed out")
)

// AssertStatus checks that the response has the expected status code.
func AssertStatus(t *testing.T, rec *httptest.ResponseRecorder, expected int) {
	t.Helper()
	if rec.Code != expected {
		t.Fatalf("expected status %d, got %d: %s", expected, rec.Code, rec.Body.String())
	}
}

// AssertJSON checks that the response body is valid JSON and unmarshals it into v.
func AssertJSON(t *testing.T, rec *httptest.ResponseRecorder, v interface{}) {
	t.Helper()
	if err := json.NewDecoder(rec.Body).Decode(v); err != nil {
		t.Fatalf("failed to decode JSON response: %v, body: %s", err, rec.Body.String())
	}
}

// AssertContentType checks that the response has the expected Content-Type header.
func AssertContentType(t *testing.T, rec *httptest.ResponseRecorder, expected string) {
	t.Helper()
	ct := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, expected) {
		t.Fatalf("expected Content-Type %q, got %q", expected, ct)
	}
}

// AssertError checks that the response contains an error message matching expected.
func AssertError(t *testing.T, rec *httptest.ResponseRecorder, expectedMessage string) {
	t.Helper()
	var errResp struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v, body: %s", err, rec.Body.String())
	}
	// Check both error and message fields since API uses "error" field
	msg := errResp.Error
	if msg == "" {
		msg = errResp.Message
	}
	if !strings.Contains(msg, expectedMessage) {
		t.Fatalf("expected error containing %q, got %q", expectedMessage, msg)
	}
}

// MakeRequest creates an HTTP request for testing.
// Method defaults to GET if empty.
func MakeRequest(t *testing.T, method, path string, body io.Reader) *http.Request {
	t.Helper()
	if method == "" {
		method = http.MethodGet
	}
	req := httptest.NewRequest(method, path, body)
	req.Header.Set("Content-Type", "application/json")
	return req
}

// MakeJSONRequest creates an HTTP request with a JSON body.
func MakeJSONRequest(t *testing.T, method, path string, body interface{}) *http.Request {
	t.Helper()
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("failed to marshal request body: %v", err)
		}
		bodyReader = strings.NewReader(string(data))
	}
	return MakeRequest(t, method, path, bodyReader)
}

// DecodeJSONResponse decodes the response body into the given type.
func DecodeJSONResponse[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var result T
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode JSON: %v, body: %s", err, rec.Body.String())
	}
	return result
}

// MustParseJSON parses JSON string into the given type, failing the test on error.
func MustParseJSON[T any](t *testing.T, jsonStr string) T {
	t.Helper()
	var result T
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	return result
}

// StringPtr returns a pointer to the given string (useful for UpdateInput fields).
func StringPtr(s string) *string {
	return &s
}

// IntPtr returns a pointer to the given int.
func IntPtr(i int) *int {
	return &i
}
