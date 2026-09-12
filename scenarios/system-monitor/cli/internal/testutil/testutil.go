// Package testutil contains helpers shared by System Monitor CLI tests.
package testutil

import "testing"

// Equal fails the test with a consistent diagnostic when a value differs from
// the expected result. Keeping this assertion here makes future command tests
// use the same test-only seam without introducing helpers into production code.
func Equal[T comparable](t *testing.T, got, want T) {
	t.Helper()
	if got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}
