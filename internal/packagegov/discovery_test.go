package packagegov

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestReadGoModClassifiesUnresolvedTemplatePlaceholders(t *testing.T) {
	path := filepath.Join(t.TempDir(), "go.mod")
	if err := os.WriteFile(path, []byte("module example\n\nreplace github.com/vrooli/api-core => {{PACKAGES_REL_FROM_API}}/api-core\n"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := readGoMod(path)
	if !errors.Is(err, errTemplatedGoMod) {
		t.Fatalf("error = %v, want templated go.mod marker", err)
	}
}

func TestTemplateGoModWithUnresolvedPlaceholdersIsSkippedDuringInventory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "go.mod")
	if err := os.WriteFile(path, []byte("module example\n\nreplace github.com/vrooli/api-core => {{PACKAGES_REL_FROM_API}}/api-core\n"), 0600); err != nil {
		t.Fatal(err)
	}
	inv := dependencyInventory{reports: map[string]DiscoveryReport{}}
	if err := inv.scanGoMod(path, scopeTemplate, nil); err != nil {
		t.Fatalf("scanGoMod returned an error for a source template: %v", err)
	}
}

func TestRealGoModParseErrorsRemainErrors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "go.mod")
	if err := os.WriteFile(path, []byte("module example\n\nreplace github.com/vrooli/api-core =>\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readGoMod(path); err == nil {
		t.Fatal("expected malformed non-template go.mod to remain an error")
	}
}

func TestDiscoveryDoesNotSkipScenarioNamesContainingBackup(t *testing.T) {
	if shouldSkipDiscoveryDir("data-backup-manager") {
		t.Fatal("scenario directory containing backup should be discovered")
	}
	if !shouldSkipDiscoveryDir("backups") || !shouldSkipDiscoveryDir(".backup") {
		t.Fatal("backup artifact directories should remain excluded")
	}
}
