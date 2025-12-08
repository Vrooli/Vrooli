package artifacts

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMigrate_NoLegacyPaths(t *testing.T) {
	tempDir := t.TempDir()
	scenarioDir := filepath.Join(tempDir, "scenario")
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatalf("failed to create scenario dir: %v", err)
	}

	result, err := Migrate(scenarioDir, MigrationOptions{})
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if result.FilesMoved != 0 {
		t.Errorf("expected 0 files moved, got %d", result.FilesMoved)
	}
	if len(result.Actions) != 0 {
		t.Errorf("expected 0 actions, got %d", len(result.Actions))
	}
}

func TestEnsureCoverageStructure(t *testing.T) {
	tempDir := t.TempDir()
	scenarioDir := filepath.Join(tempDir, "scenario")

	err := EnsureCoverageStructure(scenarioDir)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	// Verify key directories exist
	dirs := []string{
		PhaseResultsDir,
		UISmokeDir,
		AutomationDir,
		LighthouseDir,
		SyncDir,
	}

	for _, dir := range dirs {
		fullPath := filepath.Join(scenarioDir, dir)
		if _, err := os.Stat(fullPath); err != nil {
			t.Errorf("expected directory %s to exist: %v", dir, err)
		}
	}
}

func TestCleanCoverageArtifacts(t *testing.T) {
	tempDir := t.TempDir()
	scenarioDir := filepath.Join(tempDir, "scenario")

	// Create coverage structure with files
	if err := EnsureCoverageStructure(scenarioDir); err != nil {
		t.Fatalf("failed to create structure: %v", err)
	}

	testFile := filepath.Join(scenarioDir, PhaseResultsDir, "test.json")
	if err := os.WriteFile(testFile, []byte("{}"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Clean
	err := CleanCoverageArtifacts(scenarioDir)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	// Verify coverage root is gone
	coverageRoot := filepath.Join(scenarioDir, CoverageRoot)
	if _, err := os.Stat(coverageRoot); !os.IsNotExist(err) {
		t.Error("expected coverage root to be removed")
	}
}

func TestCleanCoverageArtifacts_NoCoverage(t *testing.T) {
	tempDir := t.TempDir()
	scenarioDir := filepath.Join(tempDir, "scenario")
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatalf("failed to create scenario dir: %v", err)
	}

	// Should succeed even with no coverage directory
	err := CleanCoverageArtifacts(scenarioDir)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
}
