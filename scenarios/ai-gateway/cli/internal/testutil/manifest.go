package testutil

import (
	"os"
	"testing"
)

// ManifestBytes loads the scenario CLI manifest for domain-level contract tests.
// Domain tests run one directory below cli/, so the shared fixture keeps the
// manifest path and failure reporting consistent across all domains.
func ManifestBytes(t testing.TB) []byte {
	t.Helper()
	const path = "../../manifest.json"
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
