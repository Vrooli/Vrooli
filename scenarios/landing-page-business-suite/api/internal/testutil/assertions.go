// Package testutil provides reusable test-only assertions for the API suite.
package testutil

import (
	"net/http/httptest"
	"testing"
)

// RequireNoError fails the current test immediately when err is non-nil.
func RequireNoError(t testing.TB, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// RequireHTTPStatus records an assertion failure when a handler returns an
// unexpected status and includes its body so failures remain actionable.
func RequireHTTPStatus(t testing.TB, recorder *httptest.ResponseRecorder, want int) {
	t.Helper()
	if got := recorder.Code; got != want {
		t.Errorf("unexpected HTTP status: got %d, want %d; body: %s", got, want, recorder.Body.String())
	}
}

// RequireHTTPStatusFatal stops the current test when a handler returns an
// unexpected status and includes its body so failures remain actionable.
func RequireHTTPStatusFatal(t testing.TB, recorder *httptest.ResponseRecorder, want int) {
	t.Helper()
	if got := recorder.Code; got != want {
		t.Fatalf("unexpected HTTP status: got %d, want %d; body: %s", got, want, recorder.Body.String())
	}
}
