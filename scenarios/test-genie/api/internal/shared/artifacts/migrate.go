package artifacts

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// LatestRunID reads coverage/latest/manifest.json and returns the run_id of the
// most recent run. It returns an empty string (no error) when no run has been
// recorded yet.
func LatestRunID(scenarioDir string) (string, error) {
	data, err := os.ReadFile(LatestManifestPath(scenarioDir))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("failed to read latest manifest: %w", err)
	}
	var manifest struct {
		RunID string `json:"run_id"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return "", fmt.Errorf("failed to parse latest manifest: %w", err)
	}
	return manifest.RunID, nil
}

// LegacyArtifactDirs lists the pre-Plan-A flat artifact directories that the
// runID-keyed layout replaces. They are deleted on startup (greenfield: no
// migration of their contents into coverage/runs/<runID>/).
func LegacyArtifactDirs(scenarioDir string) []string {
	return []string{
		filepath.Join(scenarioDir, CoverageRoot, "phase-results"),
		filepath.Join(scenarioDir, CoverageRoot, "automation"),
		filepath.Join(scenarioDir, CoverageRoot, "ui-smoke"),
		filepath.Join(scenarioDir, CoverageRoot, "lighthouse"),
		filepath.Join(scenarioDir, CoverageRoot, "unit"),
	}
}

// RemoveLegacyArtifactDirs deletes the pre-Plan-A flat artifact directories.
// This is a one-shot greenfield cleanup; their contents are not re-importable
// as runs and are not migrated.
func RemoveLegacyArtifactDirs(scenarioDir string) error {
	for _, dir := range LegacyArtifactDirs(scenarioDir) {
		if err := os.RemoveAll(dir); err != nil {
			return fmt.Errorf("failed to remove legacy artifact dir %s: %w", dir, err)
		}
	}
	return nil
}

// EnsureCoverageStructure creates all standard coverage directories and clears
// any leftover legacy artifact directories from the pre-runID layout.
func EnsureCoverageStructure(scenarioDir string) error {
	if err := RemoveLegacyArtifactDirs(scenarioDir); err != nil {
		return err
	}
	for _, dir := range AllCoverageSubdirs(scenarioDir) {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create %s: %w", dir, err)
		}
	}
	return nil
}

// CleanCoverageArtifacts removes all coverage artifacts.
// This is useful for starting fresh before a test run.
func CleanCoverageArtifacts(scenarioDir string) error {
	coverageRoot := filepath.Join(scenarioDir, CoverageRoot)
	if _, err := os.Stat(coverageRoot); os.IsNotExist(err) {
		return nil // Nothing to clean
	}
	return os.RemoveAll(coverageRoot)
}
