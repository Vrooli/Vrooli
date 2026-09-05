// Package testutil contains shared fixtures for the BAS CLI test suites.
package testutil

import (
	"os"
	"testing"
)

// ReadManifest loads the scenario CLI manifest from a domain test package.
// All domain tests run one directory below cli/, so keeping this fixture here
// avoids each package carrying its own file-reading copy.
func ReadManifest(t testing.TB) []byte {
	t.Helper()
	const path = "../manifest.json"
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
