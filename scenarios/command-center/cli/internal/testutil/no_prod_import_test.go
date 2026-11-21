package testutil

import "testing"

// Required test utility root for CLI-only fixtures. Production CLI packages
// must not import this package.
func testutilPackageMarker() bool { return true }

func TestPackageMarker(t *testing.T) {
	if !testutilPackageMarker() {
		t.Fatal("test utility package marker must remain available")
	}
}
