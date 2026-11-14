// DOC: docs/reference/operational-targets.md
// DOC: docs/concepts/ARCHITECTURE.md
// Package testutil provides shared testing utilities for API handlers.
//
// This package consolidates common test patterns into a single flat API:
//   - Eventual predicate checks (Eventually)
//   - Temp file/dir setup and assertions (WriteFile, MakeDir, AssertFileExists…)
//
// HTTP status and JSON decoding use the canonical apihttptest and
// repocontracttest companions directly at their test callsites.
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
	"testing"
	"time"
)

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
