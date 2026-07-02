package requirements

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScenarioNameFromDirPrefersExplicitName(t *testing.T) {
	if got := scenarioNameFromDir("/tmp/example", "named"); got != "named" {
		t.Fatalf("expected explicit name override, got %q", got)
	}
	if got := scenarioNameFromDir("/tmp/example", ""); got != "example" {
		t.Fatalf("expected directory basename fallback, got %q", got)
	}
}

func TestEnsureDirValidatesScenarioDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := ensureDir(dir); err != nil {
		t.Fatalf("expected temp dir to be accepted, got %v", err)
	}

	filePath := filepath.Join(dir, "not-a-dir")
	if err := os.WriteFile(filePath, []byte("x"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	err := ensureDir(filePath)
	if err == nil || !strings.Contains(err.Error(), "not a directory") {
		t.Fatalf("expected non-directory validation error, got %v", err)
	}
}
