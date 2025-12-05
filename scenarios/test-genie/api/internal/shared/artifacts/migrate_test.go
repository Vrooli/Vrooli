package artifacts

import (
	"bytes"
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

func TestMigrate_PhaseResultsFromTestCoverage(t *testing.T) {
	tempDir := t.TempDir()
	scenarioDir := filepath.Join(tempDir, "scenario")

	// Create legacy location with a file
	legacyDir := filepath.Join(scenarioDir, "test", "coverage", "phase-results")
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatalf("failed to create legacy dir: %v", err)
	}
	testFile := filepath.Join(legacyDir, "unit.json")
	if err := os.WriteFile(testFile, []byte(`{"phase": "unit"}`), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	var log bytes.Buffer
	result, err := Migrate(scenarioDir, MigrationOptions{
		Logger: &log,
	})
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	// Verify migration occurred
	if result.FilesMoved != 1 {
		t.Errorf("expected 1 file moved, got %d", result.FilesMoved)
	}
	if len(result.Actions) != 1 {
		t.Errorf("expected 1 action, got %d", len(result.Actions))
	}

	// Verify file is in new location
	newPath := filepath.Join(scenarioDir, PhaseResultsDir, "unit.json")
	if _, err := os.Stat(newPath); err != nil {
		t.Errorf("expected file at canonical location: %v", err)
	}

	// Verify legacy location is gone
	if _, err := os.Stat(legacyDir); !os.IsNotExist(err) {
		t.Error("expected legacy directory to be removed")
	}
}

func TestMigrate_DryRun(t *testing.T) {
	tempDir := t.TempDir()
	scenarioDir := filepath.Join(tempDir, "scenario")

	// Create legacy location
	legacyDir := filepath.Join(scenarioDir, "test", "coverage", "phase-results")
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatalf("failed to create legacy dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(legacyDir, "test.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	var log bytes.Buffer
	result, err := Migrate(scenarioDir, MigrationOptions{
		DryRun: true,
		Logger: &log,
	})
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	// Actions should be recorded but no files moved
	if len(result.Actions) == 0 {
		t.Error("expected actions to be recorded in dry run")
	}
	if result.FilesMoved != 0 {
		t.Errorf("expected 0 files moved in dry run, got %d", result.FilesMoved)
	}

	// Legacy location should still exist
	if _, err := os.Stat(legacyDir); err != nil {
		t.Error("legacy directory should still exist in dry run")
	}
}

func TestMigrate_SkipsExistingCanonical(t *testing.T) {
	tempDir := t.TempDir()
	scenarioDir := filepath.Join(tempDir, "scenario")

	// Create both legacy and canonical locations
	legacyDir := filepath.Join(scenarioDir, "test", "coverage", "phase-results")
	canonDir := filepath.Join(scenarioDir, PhaseResultsDir)

	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatalf("failed to create legacy dir: %v", err)
	}
	if err := os.MkdirAll(canonDir, 0o755); err != nil {
		t.Fatalf("failed to create canon dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(legacyDir, "old.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	var log bytes.Buffer
	result, err := Migrate(scenarioDir, MigrationOptions{
		Logger: &log,
	})
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	// Should skip because canonical already exists
	if result.FilesMoved != 0 {
		t.Errorf("expected 0 files moved when canonical exists, got %d", result.FilesMoved)
	}

	// Log should mention skip
	if !bytes.Contains(log.Bytes(), []byte("SKIP")) {
		t.Error("expected skip message in log")
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
