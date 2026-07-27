// Package testutil provides reusable test-only assertions for the API suite.
package testutil

import "testing"

// RequireNoError fails the current test immediately when err is non-nil.
func RequireNoError(t testing.TB, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
