package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// GetStorageStats walks directory tree and accumulates sizes.
func (s *VisualCaptureStorage) GetStorageStats(repoID int64) (*VisualCaptureStorageStats, error) {
	rootDir, err := s.repoSnapshotsRoot(repoID)
	if err != nil {
		return nil, err
	}

	stats := &VisualCaptureStorageStats{
		PerScenario: []ScenarioStorageBreakdown{},
	}

	scenarioDirs, err := os.ReadDir(rootDir)
	if err != nil {
		if os.IsNotExist(err) {
			return stats, nil
		}
		return nil, err
	}

	for _, scenarioDir := range scenarioDirs {
		if !scenarioDir.IsDir() {
			continue
		}
		slug := scenarioDir.Name()
		snapshotDirs, err := os.ReadDir(filepath.Join(rootDir, slug))
		if err != nil {
			continue
		}

		breakdown := ScenarioStorageBreakdown{
			ScenarioSlug: slug,
		}

		for _, snapDir := range snapshotDirs {
			if !snapDir.IsDir() {
				continue
			}
			breakdown.SnapshotCount++
			size := dirSize(filepath.Join(rootDir, slug, snapDir.Name()))
			breakdown.SizeBytes += size
		}

		stats.PerScenario = append(stats.PerScenario, breakdown)
		stats.TotalSizeBytes += breakdown.SizeBytes
		stats.SnapshotCount += breakdown.SnapshotCount
	}

	return stats, nil
}

// enforceRetentionLocked keeps at most max snapshots per scenario, deleting
// oldest non-baseline snapshots first. Baselines are never evicted by retention.
// Caller must hold s.mu.
type metaEntry struct {
	meta SnapshotSetMeta
	name string
}

// loadSnapshotMetas reads metadata.json from each subdirectory.
func (s *VisualCaptureStorage) loadSnapshotMetas(dir string) []metaEntry {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var metas []metaEntry
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		data, err := s.fs.ReadFile(filepath.Join(dir, entry.Name(), "metadata.json"))
		if err != nil {
			continue
		}
		var meta SnapshotSetMeta
		if err := json.Unmarshal(data, &meta); err != nil {
			continue
		}
		metas = append(metas, metaEntry{meta: meta, name: entry.Name()})
	}
	return metas
}

func (s *VisualCaptureStorage) enforceRetentionLocked(repoID int64, scenarioSlug string, max int) error {
	dir, err := s.snapshotsDir(repoID, scenarioSlug)
	if err != nil {
		return err
	}

	metas := s.loadSnapshotMetas(dir)
	if len(metas) <= max {
		return nil
	}

	// Sort oldest first, but put baselines at the end so they survive eviction
	sort.Slice(metas, func(i, j int) bool {
		iBaseline := metas[i].meta.EffectiveRole() == SnapshotRoleBaseline
		jBaseline := metas[j].meta.EffectiveRole() == SnapshotRoleBaseline
		if iBaseline != jBaseline {
			return !iBaseline
		}
		return metas[i].meta.CreatedAt.Before(metas[j].meta.CreatedAt)
	})

	toDelete := len(metas) - max
	for i := 0; i < toDelete; i++ {
		if metas[i].meta.EffectiveRole() == SnapshotRoleBaseline {
			break
		}
		os.RemoveAll(filepath.Join(dir, metas[i].name))
	}

	return nil
}

var presetFilenameRe = regexp.MustCompile(`@(\d+)x(\d+)_(light|dark)\.png$`)

// parsePresetFromFilename extracts viewport dimensions and theme from a
// filename like "_root_@1440x900_light.png".
func parsePresetFromFilename(name string) (width, height int, theme string, ok bool) {
	m := presetFilenameRe.FindStringSubmatch(name)
	if m == nil {
		return 0, 0, "", false
	}
	w, _ := strconv.Atoi(m[1])
	h, _ := strconv.Atoi(m[2])
	return w, h, m[3], true
}

func validateFilename(filename string) error {
	if strings.Contains(filename, "/") || strings.Contains(filename, "\\") || strings.Contains(filename, "..") {
		return fmt.Errorf("invalid filename: path traversal not allowed")
	}
	return nil
}

func dirSize(path string) int64 {
	var size int64
	_ = filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		size += info.Size()
		return nil
	})
	return size
}
