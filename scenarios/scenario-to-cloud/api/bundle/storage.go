package bundle

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"scenario-to-cloud/domain"
)

// GetLocalBundlesDir returns the path to the local bundles directory.
// This consolidates the repeated pattern of finding repo root and appending the bundles path.
func GetLocalBundlesDir() (string, error) {
	repoRoot, err := FindRepoRootFromCWD()
	if err != nil {
		return "", err
	}
	return filepath.Join(repoRoot, "scenarios", "scenario-to-cloud", "coverage", "bundles"), nil
}

// ListBundles lists all bundles in the given directory.
func ListBundles(bundlesDir string) ([]domain.BundleInfo, error) {
	if !dirExists(bundlesDir) {
		return []domain.BundleInfo{}, nil
	}

	entries, err := os.ReadDir(bundlesDir)
	if err != nil {
		return nil, fmt.Errorf("read bundles directory: %w", err)
	}

	var bundles []domain.BundleInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "mini-vrooli_") || !strings.HasSuffix(name, ".tar.gz") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		scenarioID, sha256Hash := ParseBundleFilename(name)
		bundles = append(bundles, domain.BundleInfo{
			Path:       filepath.Join(bundlesDir, name),
			Filename:   name,
			ScenarioID: scenarioID,
			Sha256:     sha256Hash,
			SizeBytes:  info.Size(),
			CreatedAt:  info.ModTime().UTC().Format(time.RFC3339),
		})
	}

	// Sort by creation time, newest first
	sort.Slice(bundles, func(i, j int) bool {
		return bundles[i].CreatedAt > bundles[j].CreatedAt
	})

	return bundles, nil
}

// ParseBundleFilename extracts scenario ID and SHA256 hash from a bundle filename.
func ParseBundleFilename(name string) (scenarioID, sha256Hash string) {
	// Format: mini-vrooli_<scenario-id>_<sha256>.tar.gz
	name = strings.TrimPrefix(name, "mini-vrooli_")
	name = strings.TrimSuffix(name, ".tar.gz")

	// Find the last underscore followed by a 64-char hex string (SHA256)
	lastUnderscore := strings.LastIndex(name, "_")
	if lastUnderscore == -1 || len(name)-lastUnderscore-1 != 64 {
		return name, ""
	}

	return name[:lastUnderscore], name[lastUnderscore+1:]
}

// GetBundleStats returns aggregate storage statistics for all bundles.
func GetBundleStats(bundlesDir string) (domain.BundleStats, error) {
	bundles, err := ListBundles(bundlesDir)
	if err != nil {
		return domain.BundleStats{}, err
	}

	stats := domain.BundleStats{
		ByScenario: make(map[string]domain.ScenarioStats),
	}

	for _, b := range bundles {
		stats.TotalCount++
		stats.TotalSizeBytes += b.SizeBytes

		// Track per-scenario stats
		scenStats := stats.ByScenario[b.ScenarioID]
		scenStats.Count++
		scenStats.SizeBytes += b.SizeBytes
		stats.ByScenario[b.ScenarioID] = scenStats

		// Track oldest/newest (bundles are sorted newest first)
		if stats.NewestCreatedAt == "" {
			stats.NewestCreatedAt = b.CreatedAt
		}
		stats.OldestCreatedAt = b.CreatedAt
	}

	return stats, nil
}

// DeleteBundle removes a single bundle by SHA256 hash.
// Returns the size of the deleted file in bytes, or 0 if not found.
func DeleteBundle(bundlesDir, sha256Hash string) (int64, error) {
	if sha256Hash == "" {
		return 0, fmt.Errorf("sha256 hash is required")
	}

	bundles, err := ListBundles(bundlesDir)
	if err != nil {
		return 0, err
	}

	for _, b := range bundles {
		if b.Sha256 == sha256Hash {
			if err := os.Remove(b.Path); err != nil {
				if os.IsNotExist(err) {
					return 0, nil // Already deleted, idempotent
				}
				return 0, fmt.Errorf("delete bundle %s: %w", b.Filename, err)
			}
			return b.SizeBytes, nil
		}
	}

	return 0, nil // Not found, idempotent
}

// DeleteBundlesForScenario removes bundles for a specific scenario,
// keeping the N most recent bundles (by creation time).
// Returns the list of deleted bundles and total freed bytes.
func DeleteBundlesForScenario(bundlesDir, scenarioID string, keepLatest int) ([]domain.BundleInfo, int64, error) {
	if scenarioID == "" {
		return nil, 0, fmt.Errorf("scenario ID is required")
	}
	if keepLatest < 0 {
		keepLatest = 0
	}

	bundles, err := ListBundles(bundlesDir)
	if err != nil {
		return nil, 0, err
	}

	// Filter bundles for this scenario (already sorted newest first)
	var scenarioBundles []domain.BundleInfo
	for _, b := range bundles {
		if b.ScenarioID == scenarioID {
			scenarioBundles = append(scenarioBundles, b)
		}
	}

	// Keep the newest N bundles, delete the rest
	var deleted []domain.BundleInfo
	var freedBytes int64

	for i, b := range scenarioBundles {
		if i < keepLatest {
			continue // Keep this one
		}
		if err := os.Remove(b.Path); err != nil {
			if os.IsNotExist(err) {
				continue // Already deleted
			}
			return deleted, freedBytes, fmt.Errorf("delete bundle %s: %w", b.Filename, err)
		}
		deleted = append(deleted, b)
		freedBytes += b.SizeBytes
	}

	return deleted, freedBytes, nil
}

// DeleteAllOldBundles removes bundles across all scenarios, keeping N newest per scenario.
// Returns the list of deleted bundles and total freed bytes.
func DeleteAllOldBundles(bundlesDir string, keepLatestPerScenario int) ([]domain.BundleInfo, int64, error) {
	if keepLatestPerScenario < 0 {
		keepLatestPerScenario = 0
	}

	bundles, err := ListBundles(bundlesDir)
	if err != nil {
		return nil, 0, err
	}

	// Group bundles by scenario
	byScenario := make(map[string][]domain.BundleInfo)
	for _, b := range bundles {
		byScenario[b.ScenarioID] = append(byScenario[b.ScenarioID], b)
	}

	var deleted []domain.BundleInfo
	var freedBytes int64

	// For each scenario, keep N newest and delete the rest
	for _, scenarioBundles := range byScenario {
		for i, b := range scenarioBundles {
			if i < keepLatestPerScenario {
				continue // Keep this one
			}
			if err := os.Remove(b.Path); err != nil {
				if os.IsNotExist(err) {
					continue // Already deleted
				}
				// Log but continue with other deletions
				continue
			}
			deleted = append(deleted, b)
			freedBytes += b.SizeBytes
		}
	}

	return deleted, freedBytes, nil
}

// FindRepoRootFromCWD finds the repository root starting from the current working directory.
func FindRepoRootFromCWD() (string, error) {
	if override := strings.TrimSpace(os.Getenv("SCENARIO_TO_CLOUD_REPO_ROOT")); override != "" {
		return filepath.Clean(override), nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return FindRepoRoot(cwd)
}

// FindRepoRoot finds the repository root starting from the given directory.
func FindRepoRoot(start string) (string, error) {
	dir := filepath.Clean(start)
	for i := 0; i < 20; i++ {
		// Repo root detection must not depend on a committed `go.work`.
		// Some deployments intentionally omit `go.work` to avoid workspace-mode coupling.
		if dirExists(filepath.Join(dir, ".vrooli")) && dirExists(filepath.Join(dir, "scenarios")) && dirExists(filepath.Join(dir, "resources")) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("repo root not found from %q", start)
}
