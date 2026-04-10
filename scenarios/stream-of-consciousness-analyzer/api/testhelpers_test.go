package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

// fakeUUID generates a deterministic UUID-like string for testing.
func fakeUUID(n int) string {
	return "00000000-0000-0000-0000-" + padInt(n)
}

func padInt(n int) string {
	s := ""
	for i := 0; i < 12; i++ {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

// decodeJSON decodes the recorder body into T or fails the test.
func decodeJSON[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var result T
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	return result
}

// assertStatus checks the response status code or reports an error.
func assertStatus(t *testing.T, rec *httptest.ResponseRecorder, expected int) {
	t.Helper()
	if rec.Code != expected {
		t.Errorf("expected status %d, got %d; body: %s", expected, rec.Code, rec.Body.String())
	}
}

// jsonBody creates a reader from a JSON string for use in test requests.
func jsonBody(s string) *bytes.Buffer {
	return bytes.NewBufferString(s)
}

// setURLVars sets a single gorilla/mux URL variable on a request.
func setURLVars(r *http.Request, key, value string) *http.Request {
	return mux.SetURLVars(r, map[string]string{key: value})
}

// strPtr returns a pointer to the given string.
func strPtr(s string) *string {
	return &s
}
