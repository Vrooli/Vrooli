package bundleruntime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateCatalogRequirementsNamesMissingCatalog(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "catalog", "scenarios"), 0o755); err != nil {
		t.Fatal(err)
	}
	errors := validateCatalogRequirements(root, RealFileSystem{}, []string{
		"catalog/scenarios", "catalog/internal/tools",
	})
	if len(errors) != 1 || errors[0].Code != "catalog_missing" || errors[0].Path != "catalog/internal/tools" {
		t.Fatalf("catalog errors = %#v", errors)
	}
	if !strings.Contains(errors[0].Message, "catalog/internal/tools") {
		t.Fatalf("error = %q, want missing path", errors[0].Message)
	}
}
