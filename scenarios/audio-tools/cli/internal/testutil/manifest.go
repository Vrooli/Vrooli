package testutil

import (
	"os"
	"testing"
)

// ReadManifest loads the scenario CLI manifest from a domain test package.
// Domain tests run one directory below cli/, so this keeps the fixture path in
// one test-only helper rather than duplicating file reads across domains.
func ReadManifest(t testing.TB) []byte {
	t.Helper()
	const path = "../../manifest.json"
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
