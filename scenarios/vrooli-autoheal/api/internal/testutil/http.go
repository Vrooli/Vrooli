package testutil

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

// AssertStatus fails the test when the recorder status differs from expected.
func AssertStatus(t *testing.T, rec *httptest.ResponseRecorder, expected int) {
	t.Helper()
	if rec.Code != expected {
		t.Fatalf("expected status %d, got %d: %s", expected, rec.Code, rec.Body.String())
	}
}

// MustDecodeJSON decodes response JSON and fails the test on decode errors.
func MustDecodeJSON[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var result T
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}
	return result
}
