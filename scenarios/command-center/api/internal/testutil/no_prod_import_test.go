package testutil

import "testing"

// This package is intentionally reserved for deterministic API fakes and
// fixtures. Keeping the guard in the required testutil root prevents helpers
// from becoming production dependencies.
func testutilPackageMarker() bool { return true }

func TestPackageMarker(t *testing.T) {
	if !testutilPackageMarker() {
		t.Fatal("test utility package marker must remain available")
	}
}
