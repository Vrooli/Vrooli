// DOC: docs/reference/operational-targets.md
// DOC: docs/concepts/ARCHITECTURE.md
// Package testutil provides shared testing utilities for API handlers.
//
// This package consolidates common test patterns into a single flat API:
//   - HTTP status assertions (AssertStatus*)
//   - Temp file/dir setup and assertions (WriteFile, MakeDir, AssertFileExists…)
//   - Response body decoding (DecodeJSON, DecodeProtoJSON)
//   - Shared fakes for the dispatch/agentmanager seams (NoopInvalidator…)
//
// Design Goals:
//   - Reduce boilerplate in test files
//   - Encourage consistent testing patterns
//   - Use Go's built-in t.TempDir() for automatic cleanup (no manual defer needed)
//
// Helpers take testing.TB so they work from both tests and benchmarks. The
// package is imported only from _test.go files; the no_prod_import_test guard
// enforces that production code never depends on it.
package testutil

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// AssertStatus checks that the response has the expected HTTP status code.
func AssertStatus(tb testing.TB, rec *httptest.ResponseRecorder, expected int) {
	tb.Helper()
	if rec.Code != expected {
		tb.Errorf("expected status %d, got %d: %s", expected, rec.Code, rec.Body.String())
	}
}

// AssertStatusOK checks that the response has HTTP 200 status.
func AssertStatusOK(tb testing.TB, rec *httptest.ResponseRecorder) {
	tb.Helper()
	AssertStatus(tb, rec, http.StatusOK)
}

// AssertStatusCreated checks that the response has HTTP 201 status.
func AssertStatusCreated(tb testing.TB, rec *httptest.ResponseRecorder) {
	tb.Helper()
	AssertStatus(tb, rec, http.StatusCreated)
}

// AssertStatusNotFound checks that the response has HTTP 404 status.
func AssertStatusNotFound(tb testing.TB, rec *httptest.ResponseRecorder) {
	tb.Helper()
	AssertStatus(tb, rec, http.StatusNotFound)
}

// AssertStatusBadRequest checks that the response has HTTP 400 status.
func AssertStatusBadRequest(tb testing.TB, rec *httptest.ResponseRecorder) {
	tb.Helper()
	AssertStatus(tb, rec, http.StatusBadRequest)
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
