// Package testutil contains helpers shared by Secrets Manager API tests.
package testutil

import "testing"

// RequireNoError fails the current test at its caller when err is non-nil.
func RequireNoError(t *testing.T, err error, action string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", action, err)
	}
}
