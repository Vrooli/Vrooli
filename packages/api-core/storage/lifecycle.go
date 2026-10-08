package storage

import (
	"path/filepath"
	"strings"
)

// LifecycleDataDirName is the scenario-relative directory the lifecycle
// assigns a live instance as SCENARIO_DATA_DIR. internal/ports injects it,
// SQLitePath honours it for the launched instance, and storage measurement
// reads it through ResolveOwnerStorageLocations. One name keeps the three
// from drifting apart again.
const LifecycleDataDirName = "data"

// LifecycleDataDir returns the data directory the lifecycle assigns a live
// instance of the scenario rooted at scenarioDir.
func LifecycleDataDir(scenarioDir string) string {
	return filepath.Join(scenarioDir, LifecycleDataDirName)
}

// ResolveOwnerStorageLocations returns every physical location that can hold
// an entry's bytes. The first element is always ResolveOwnerStoragePath's
// answer, which remains the only target any pruner may act on.
//
// A scenario's class-data entry has a second location. A lifecycle-launched
// live instance opens its SQLite database under SCENARIO_DATA_DIR
// (<scenario>/data, see SQLitePath), while code that resolves ClassData
// directly writes the class root (~/.vrooli/data/vrooli/<scenario>). Both are
// that scenario's data. Measuring only the class root left live databases
// unmeasured: on 2026-09-14 agent-manager's 274 GiB database and
// experience-manager's 17 GB database sat outside every budget while
// storage-manager measured a stale 5.6 GB copy.
//
// Additional locations are measurement-only. They exist so a budget can alarm
// on the bytes an owner really holds; they never widen what a pruner deletes.
func ResolveOwnerStorageLocations(repoRoot string, owner OwnerManifest, entry StorageEntry, requested Platform, seams PlatformSeams) ([]string, error) {
	primary, err := ResolveOwnerStoragePath(repoRoot, owner, entry, requested, seams)
	if err != nil {
		return nil, err
	}
	locations := []string{primary}
	if owner.Kind != OwnerScenario || entry.Class != ClassData || !isClassDeclaration(entry) || NormalizePlatform(string(requested)) != HostPlatform() {
		return locations, nil
	}
	scenarioDir := filepath.Dir(filepath.Dir(owner.ManifestPath))
	if !filepath.IsAbs(scenarioDir) {
		if strings.TrimSpace(repoRoot) == "" {
			return locations, nil
		}
		scenarioDir = filepath.Join(repoRoot, scenarioDir)
	}
	lifecycle, err := cleanJoin(LifecycleDataDir(scenarioDir), entry.Subpath)
	if err != nil {
		return locations, nil
	}
	if filepath.Clean(lifecycle) != filepath.Clean(primary) {
		locations = append(locations, lifecycle)
	}
	return locations, nil
}
