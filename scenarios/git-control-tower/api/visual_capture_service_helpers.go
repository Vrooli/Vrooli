package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// prepareForCapture handles snapshot cleanup before saving a new capture.
//
// Baseline mode: additive — keeps every existing snapshot. Baseline-role
// snapshots are pinned by BaselineManifests (one per baseline), so multiple
// must coexist and routine captures must never disturb them; they are released
// only when their owning baseline is deleted (Service.Delete → SurfaceReleaser).
// Capture mode: replaces the single loose capture-role snapshot, leaving every
// baseline-pinned snapshot intact.
func prepareForCapture(storage *VisualCaptureStorage, repoID int64, scenarioSlug, mode string) error {
	switch mode {
	case CaptureModeCapture:
		return storage.DeleteSnapshotsByRole(repoID, scenarioSlug, SnapshotRoleCapture)
	default:
		// Baseline (and any future additive) mode: keep all existing snapshots.
		return nil
	}
}

// CheckCaptureStaleness determines if the most recent capture for a scenario
// is outdated by comparing its creation time against file modification times
// in the scenario's source directory.
func CheckCaptureStaleness(repoDir, scenarioSlug string, captureTime time.Time) *SnapshotStalenessInfo {
	scenarioDir, err := resolveScenarioPath(repoDir, scenarioSlug)
	if err != nil {
		return &SnapshotStalenessInfo{IsStale: false}
	}

	// Directories to skip during the walk
	skipDirs := map[string]bool{
		"node_modules": true, ".git": true, "dist": true, "build": true,
		".next": true, "__pycache__": true, ".cache": true,
	}

	var latestMod time.Time
	_ = filepath.Walk(scenarioDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if skipDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if info.ModTime().After(latestMod) {
			latestMod = info.ModTime()
		}
		return nil
	})

	if latestMod.IsZero() {
		return &SnapshotStalenessInfo{IsStale: false}
	}

	isStale := latestMod.After(captureTime)
	return &SnapshotStalenessInfo{
		IsStale:          isStale,
		LastFileChange:   &latestMod,
		CaptureCreatedAt: &captureTime,
	}
}

// discoverPages reads lighthouse.json from the scenario directory.
func discoverPages(fs FileIO, repoDir, scenarioSlug string) []LighthousePage {
	pages, _ := discoverPagesWithMethod(fs, repoDir, scenarioSlug)
	return pages
}

// discoverPagesWithMethod reads lighthouse.json and returns pages + discovery method.
func discoverPagesWithMethod(fs FileIO, repoDir, scenarioSlug string) ([]LighthousePage, string) {
	searchPaths := []string{}
	if scenarioDir, err := resolveScenarioPath(repoDir, scenarioSlug); err == nil {
		searchPaths = append(searchPaths, filepath.Join(scenarioDir, ".vrooli", "lighthouse.json"))
	}

	for _, path := range searchPaths {
		data, err := fs.ReadFile(path)
		if err != nil {
			continue
		}
		var cfg LighthouseConfig
		if err := json.Unmarshal(data, &cfg); err != nil {
			continue
		}
		if cfg.Enabled && len(cfg.Pages) > 0 {
			return cfg.Pages, "lighthouse"
		}
	}

	return []LighthousePage{{Path: "/", Label: "Home"}}, "fallback"
}

// presetSuffix returns a filename-safe suffix like "@1440x900_light" for a capture preset.
func presetSuffix(p CapturePreset) string {
	return fmt.Sprintf("@%dx%d_%s", p.Width, p.Height, p.Theme)
}

// sanitizeFilename converts page path "/" → "_root_", "/workflow/new" → "_workflow_new_"
func sanitizeFilename(pagePath string) string {
	if pagePath == "/" || pagePath == "" {
		return "_root_"
	}
	s := strings.TrimPrefix(pagePath, "/")
	s = strings.TrimSuffix(s, "/")
	s = strings.ReplaceAll(s, "/", "_")
	return "_" + s + "_"
}
