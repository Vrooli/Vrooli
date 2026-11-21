// Package testutil contains helpers shared by CLI tests.
package testutil

import "testing"

// RequireEqual fails the test immediately when comparable values differ.
func RequireEqual[T comparable](t testing.TB, got, want T, name string) {
	t.Helper()
	if got != want {
		t.Fatalf("%s = %v, want %v", name, got, want)
	}
}
