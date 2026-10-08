package hygiene

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNoLocalStructuralRuleImplementations(t *testing.T) {
	if _, err := os.Stat(filepath.Join("..", "..", "repocontractcheck")); !os.IsNotExist(err) {
		t.Fatalf("internal/repocontractcheck must remain deleted, stat error = %v", err)
	}
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	banned := []string{
		"checkTrackedBinaries",
		"checkOrphanedProtoSurfaces",
		"checkPnpmConfig",
		"checkScenarioPnpm",
	}
	for _, file := range files {
		if filepath.Base(file) == "static_test.go" {
			continue
		}
		data, readErr := os.ReadFile(file)
		if readErr != nil {
			t.Fatal(readErr)
		}
		for _, token := range banned {
			if strings.Contains(string(data), token) {
				t.Errorf("%s contains local structural-rule token %q", file, token)
			}
		}
	}
}
