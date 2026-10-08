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

// EnsureCoverageStructure creates all standard coverage directories.
func EnsureCoverageStructure(scenarioDir string) error {
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
