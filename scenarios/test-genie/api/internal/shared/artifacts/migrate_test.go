package artifacts

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureCoverageStructure(t *testing.T) {
	tempDir := t.TempDir()
	scenarioDir := filepath.Join(tempDir, "scenario")

	err := EnsureCoverageStructure(scenarioDir)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	// Verify scenario-global directories exist (per-run dirs are lazy).
	dirs := []string{
		LogsDir,
		LatestDir,
		RunsDir,
		SyncDir,
	}

	for _, dir := range dirs {
		fullPath := filepath.Join(scenarioDir, dir)
		if _, err := os.Stat(fullPath); err != nil {
			t.Errorf("expected directory %s to exist: %v", dir, err)
		}
	}
}

func TestRemoveLegacyArtifactDirs(t *testing.T) {
	tempDir := t.TempDir()
	scenarioDir := filepath.Join(tempDir, "scenario")

	// Seed legacy flat directories with a file each.
	for _, dir := range LegacyArtifactDirs(scenarioDir) {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("failed to seed legacy dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "old.json"), []byte("{}"), 0o644); err != nil {
			t.Fatalf("failed to seed legacy file: %v", err)
		}
	}

	// EnsureCoverageStructure performs the one-shot greenfield cleanup.
	if err := EnsureCoverageStructure(scenarioDir); err != nil {
		t.Fatalf("ensure structure: %v", err)
	}

	for _, dir := range LegacyArtifactDirs(scenarioDir) {
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			t.Errorf("expected legacy dir %s to be removed", dir)
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

	phaseDir := RunPhaseResultsDir(scenarioDir, "20251208-151044-deadbeef")
	if err := os.MkdirAll(phaseDir, 0o755); err != nil {
		t.Fatalf("failed to create run phase dir: %v", err)
	}
	testFile := filepath.Join(phaseDir, "test.json")
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
