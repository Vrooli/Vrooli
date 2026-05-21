package manifest

import (
	"strings"
	"testing"
)

func TestSchema_NotEmpty(t *testing.T) {
	if Schema() == "" {
		t.Fatal("manifest.Schema() returned empty; check go:embed wiring")
	}
}

func TestSchema_ContainsManifestsTable(t *testing.T) {
	if !strings.Contains(Schema(), "CREATE TABLE IF NOT EXISTS manifests") {
		t.Fatalf("manifest.Schema() missing manifests table")
	}
}
