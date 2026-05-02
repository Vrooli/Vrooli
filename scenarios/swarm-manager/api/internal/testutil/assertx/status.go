package assertx

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func Status(tb testing.TB, rec *httptest.ResponseRecorder, expected int) {
	tb.Helper()
	if rec.Code != expected {
		tb.Errorf("expected status %d, got %d: %s", expected, rec.Code, rec.Body.String())
	}
}

func StatusOK(tb testing.TB, rec *httptest.ResponseRecorder) {
	tb.Helper()
	Status(tb, rec, http.StatusOK)
}

func StatusCreated(tb testing.TB, rec *httptest.ResponseRecorder) {
	tb.Helper()
	Status(tb, rec, http.StatusCreated)
}

func StatusNotFound(tb testing.TB, rec *httptest.ResponseRecorder) {
	tb.Helper()
	Status(tb, rec, http.StatusNotFound)
}

func StatusBadRequest(tb testing.TB, rec *httptest.ResponseRecorder) {
	tb.Helper()
	Status(tb, rec, http.StatusBadRequest)
}

// Eventually polls predicate until it succeeds or the timeout expires. It is
// intended for tests that observe fire-and-forget work, where fixed sleeps
// either hide failures or make the suite slower than necessary.
func Eventually(tb testing.TB, timeout time.Duration, reason string, predicate func() bool) {
	tb.Helper()
	const interval = 10 * time.Millisecond
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if predicate() {
			return
		}
		time.Sleep(interval)
	}
	if predicate() {
		return
	}
	tb.Fatalf("timed out after %s waiting for %s", timeout, reason)
}
