// Package testutil contains CLI-only test helpers. Production CLI packages
// must not depend on this package.
package testutil

import "testing"

func RequireNonEmpty(t *testing.T, value, label string) {
	t.Helper()
	if value == "" {
		t.Fatalf("%s must not be empty", label)
	}
}
