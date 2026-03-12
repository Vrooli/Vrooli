package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/vrooli/api-core/storage"
)

// VisualCaptureStorage manages file-based snapshot storage with retention.
type VisualCaptureStorage struct {
	resolver *storage.Resolver
	fs       FileIO
	mu       sync.Mutex
}

// NewVisualCaptureStorage creates a new storage instance.
func NewVisualCaptureStorage(resolver *storage.Resolver, fs FileIO) *VisualCaptureStorage {
	return &VisualCaptureStorage{resolver: resolver, fs: fs}
}

func (s *VisualCaptureStorage) snapshotsDir(repoID int64, scenarioSlug string) (string, error) {
	return s.resolver.Path(
		storage.Options{ScenarioID: "git-control-tower"},
		storage.ClassData,
		fmt.Sprintf("%d/snapshots/%s", repoID, scenarioSlug),
	)
}

func (s *VisualCaptureStorage) snapshotDir(repoID int64, scenarioSlug, snapshotID string) (string, error) {
	return s.resolver.Path(
		storage.Options{ScenarioID: "git-control-tower"},
		storage.ClassData,
		fmt.Sprintf("%d/snapshots/%s/%s", repoID, scenarioSlug, snapshotID),
	)
}

func (s *VisualCaptureStorage) repoSnapshotsRoot(repoID int64) (string, error) {
	return s.resolver.Path(
		storage.Options{ScenarioID: "git-control-tower"},
		storage.ClassData,
		fmt.Sprintf("%d/snapshots", repoID),
	)
}

// SaveSnapshotSet creates a directory, writes screenshots and metadata.
func (s *VisualCaptureStorage) SaveSnapshotSet(repoID int64, meta SnapshotSetMeta, screenshots map[string][]byte, videos map[string][]byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dir, err := s.snapshotDir(repoID, meta.ScenarioSlug, meta.ID)
	if err != nil {
		return fmt.Errorf("resolve snapshot dir: %w", err)
	}

	if err := s.fs.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create snapshot dir: %w", err)
	}

	// Write screenshots
	if len(screenshots) > 0 {
		ssDir := filepath.Join(dir, "screenshots")
		if err := s.fs.MkdirAll(ssDir, 0o755); err != nil {
			return fmt.Errorf("create screenshots dir: %w", err)
		}
		for name, data := range screenshots {
			if err := s.fs.WriteFile(filepath.Join(ssDir, name), data, 0o644); err != nil {
				return fmt.Errorf("write screenshot %s: %w", name, err)
			}
		}
	}

	// Write videos
	if len(videos) > 0 {
		vidDir := filepath.Join(dir, "videos")
		if err := s.fs.MkdirAll(vidDir, 0o755); err != nil {
			return fmt.Errorf("create videos dir: %w", err)
		}
		for name, data := range videos {
			if err := s.fs.WriteFile(filepath.Join(vidDir, name), data, 0o644); err != nil {
				return fmt.Errorf("write video %s: %w", name, err)
			}
		}
	}

	// Compute total size
	var totalSize int64
	for _, data := range screenshots {
		totalSize += int64(len(data))
	}
	for _, data := range videos {
		totalSize += int64(len(data))
	}
	meta.SizeBytes = totalSize

	// Write metadata
	metaBytes, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}
	if err := s.fs.WriteFile(filepath.Join(dir, "metadata.json"), metaBytes, 0o644); err != nil {
		return fmt.Errorf("write metadata: %w", err)
	}

	// Enforce retention
	_ = s.enforceRetentionLocked(repoID, meta.ScenarioSlug, 10)

	return nil
}

// ListSnapshotSets reads all metadata.json files, sorted newest-first.
func (s *VisualCaptureStorage) ListSnapshotSets(repoID int64, scenarioSlug string) ([]SnapshotSetMeta, error) {
	dir, err := s.snapshotsDir(repoID, scenarioSlug)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []SnapshotSetMeta{}, nil
		}
		return nil, fmt.Errorf("read snapshots dir: %w", err)
	}

	var metas []SnapshotSetMeta
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		metaPath := filepath.Join(dir, entry.Name(), "metadata.json")
		data, err := s.fs.ReadFile(metaPath)
		if err != nil {
			continue
		}
		var meta SnapshotSetMeta
		if err := json.Unmarshal(data, &meta); err != nil {
			continue
		}
		metas = append(metas, meta)
	}

	sort.Slice(metas, func(i, j int) bool {
		return metas[i].CreatedAt.After(metas[j].CreatedAt)
	})

	return metas, nil
}

// GetSnapshotSet reads metadata and lists files with sizes.
func (s *VisualCaptureStorage) GetSnapshotSet(repoID int64, scenarioSlug, snapshotID string) (*SnapshotSetDetail, error) {
	dir, err := s.snapshotDir(repoID, scenarioSlug, snapshotID)
	if err != nil {
		return nil, err
	}

	metaData, err := s.fs.ReadFile(filepath.Join(dir, "metadata.json"))
	if err != nil {
		return nil, fmt.Errorf("read metadata: %w", err)
	}

	var meta SnapshotSetMeta
	if err := json.Unmarshal(metaData, &meta); err != nil {
		return nil, fmt.Errorf("parse metadata: %w", err)
	}

	detail := &SnapshotSetDetail{
		SnapshotSetMeta: meta,
		Screenshots:     []SnapshotFile{},
		Videos:          []SnapshotFile{},
	}

	// List screenshots
	ssDir := filepath.Join(dir, "screenshots")
	if entries, err := os.ReadDir(ssDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			info, err := entry.Info()
			if err != nil {
				continue
			}
			detail.Screenshots = append(detail.Screenshots, SnapshotFile{
				Filename:  entry.Name(),
				SizeBytes: info.Size(),
			})
		}
	}

	// List videos
	vidDir := filepath.Join(dir, "videos")
	if entries, err := os.ReadDir(vidDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			info, err := entry.Info()
			if err != nil {
				continue
			}
			detail.Videos = append(detail.Videos, SnapshotFile{
				Filename:  entry.Name(),
				SizeBytes: info.Size(),
			})
		}
	}

	return detail, nil
}

// GetScreenshotFile returns raw PNG bytes. Validates filename has no path separators.
func (s *VisualCaptureStorage) GetScreenshotFile(repoID int64, scenarioSlug, snapshotID, filename string) ([]byte, error) {
	if err := validateFilename(filename); err != nil {
		return nil, err
	}
	dir, err := s.snapshotDir(repoID, scenarioSlug, snapshotID)
	if err != nil {
		return nil, err
	}
	return s.fs.ReadFile(filepath.Join(dir, "screenshots", filename))
}

// GetVideoFile returns raw video bytes.
func (s *VisualCaptureStorage) GetVideoFile(repoID int64, scenarioSlug, snapshotID, filename string) ([]byte, error) {
	if err := validateFilename(filename); err != nil {
		return nil, err
	}
	dir, err := s.snapshotDir(repoID, scenarioSlug, snapshotID)
	if err != nil {
		return nil, err
	}
	return s.fs.ReadFile(filepath.Join(dir, "videos", filename))
}

// DeleteSnapshotSet removes a snapshot directory.
func (s *VisualCaptureStorage) DeleteSnapshotSet(repoID int64, scenarioSlug, snapshotID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.deleteSnapshotSetLocked(repoID, scenarioSlug, snapshotID)
}

func (s *VisualCaptureStorage) deleteSnapshotSetLocked(repoID int64, scenarioSlug, snapshotID string) error {
	dir, err := s.snapshotDir(repoID, scenarioSlug, snapshotID)
	if err != nil {
		return err
	}
	return os.RemoveAll(dir)
}

// DeleteSnapshotsByRole removes all snapshots matching the given role.
func (s *VisualCaptureStorage) DeleteSnapshotsByRole(repoID int64, scenarioSlug, role string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	metas, err := s.ListSnapshotSets(repoID, scenarioSlug)
	if err != nil {
		return err
	}
	for _, m := range metas {
		if m.EffectiveRole() == role {
			if err := s.deleteSnapshotSetLocked(repoID, scenarioSlug, m.ID); err != nil {
				return fmt.Errorf("delete snapshot %s: %w", m.ID, err)
			}
		}
	}
	return nil
}

// ClearScenarioSnapshots removes all snapshots for a specific scenario.
func (s *VisualCaptureStorage) ClearScenarioSnapshots(repoID int64, scenarioSlug string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dir, err := s.snapshotsDir(repoID, scenarioSlug)
	if err != nil {
		return err
	}
	return os.RemoveAll(dir)
}

// ClearAllSnapshots removes all snapshots for a repo.
func (s *VisualCaptureStorage) ClearAllSnapshots(repoID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dir, err := s.repoSnapshotsRoot(repoID)
	if err != nil {
		return err
	}
	return os.RemoveAll(dir)
}

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
func (s *VisualCaptureStorage) enforceRetentionLocked(repoID int64, scenarioSlug string, max int) error {
	dir, err := s.snapshotsDir(repoID, scenarioSlug)
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	type metaEntry struct {
		meta SnapshotSetMeta
		name string
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

	if len(metas) <= max {
		return nil
	}

	// Sort oldest first, but put baselines at the end so they survive eviction
	sort.Slice(metas, func(i, j int) bool {
		iBaseline := metas[i].meta.EffectiveRole() == SnapshotRoleBaseline
		jBaseline := metas[j].meta.EffectiveRole() == SnapshotRoleBaseline
		if iBaseline != jBaseline {
			return !iBaseline // non-baselines first (evicted first)
		}
		return metas[i].meta.CreatedAt.Before(metas[j].meta.CreatedAt)
	})

	toDelete := len(metas) - max
	for i := 0; i < toDelete; i++ {
		// Never evict baselines during retention
		if metas[i].meta.EffectiveRole() == SnapshotRoleBaseline {
			break
		}
		os.RemoveAll(filepath.Join(dir, metas[i].name))
	}

	return nil
}

// ============================================================================
// Workflow Capture Storage
// ============================================================================

func (s *VisualCaptureStorage) workflowCapturesDir(repoID int64, scenarioSlug string) (string, error) {
	return s.resolver.Path(
		storage.Options{ScenarioID: "git-control-tower"},
		storage.ClassData,
		fmt.Sprintf("%d/workflow-captures/%s", repoID, scenarioSlug),
	)
}

func (s *VisualCaptureStorage) workflowCaptureDir(repoID int64, scenarioSlug, captureID string) (string, error) {
	return s.resolver.Path(
		storage.Options{ScenarioID: "git-control-tower"},
		storage.ClassData,
		fmt.Sprintf("%d/workflow-captures/%s/%s", repoID, scenarioSlug, captureID),
	)
}

// SaveWorkflowCaptureResult saves a workflow capture with its video artifacts.
func (s *VisualCaptureStorage) SaveWorkflowCaptureResult(repoID int64, scenarioSlug string, result *WorkflowCaptureResult, videos map[string][]byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dir, err := s.workflowCaptureDir(repoID, scenarioSlug, result.ID)
	if err != nil {
		return fmt.Errorf("resolve workflow capture dir: %w", err)
	}

	if err := s.fs.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create workflow capture dir: %w", err)
	}

	// Write videos
	if len(videos) > 0 {
		vidDir := filepath.Join(dir, "videos")
		if err := s.fs.MkdirAll(vidDir, 0o755); err != nil {
			return fmt.Errorf("create videos dir: %w", err)
		}
		for name, data := range videos {
			if err := s.fs.WriteFile(filepath.Join(vidDir, name), data, 0o644); err != nil {
				return fmt.Errorf("write video %s: %w", name, err)
			}
		}
	}

	// Write metadata
	metaBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}
	if err := s.fs.WriteFile(filepath.Join(dir, "metadata.json"), metaBytes, 0o644); err != nil {
		return fmt.Errorf("write metadata: %w", err)
	}

	return nil
}

// ListWorkflowCaptures reads all workflow capture metadata, sorted newest-first.
func (s *VisualCaptureStorage) ListWorkflowCaptures(repoID int64, scenarioSlug string) ([]WorkflowCaptureResult, error) {
	dir, err := s.workflowCapturesDir(repoID, scenarioSlug)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []WorkflowCaptureResult{}, nil
		}
		return nil, fmt.Errorf("read workflow captures dir: %w", err)
	}

	var results []WorkflowCaptureResult
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		metaPath := filepath.Join(dir, entry.Name(), "metadata.json")
		data, err := s.fs.ReadFile(metaPath)
		if err != nil {
			continue
		}
		var result WorkflowCaptureResult
		if err := json.Unmarshal(data, &result); err != nil {
			continue
		}
		results = append(results, result)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].CreatedAt.After(results[j].CreatedAt)
	})

	return results, nil
}

// GetWorkflowCapture returns a single workflow capture with video listing.
func (s *VisualCaptureStorage) GetWorkflowCapture(repoID int64, scenarioSlug, captureID string) (*WorkflowCaptureResult, []SnapshotFile, error) {
	dir, err := s.workflowCaptureDir(repoID, scenarioSlug, captureID)
	if err != nil {
		return nil, nil, err
	}

	metaData, err := s.fs.ReadFile(filepath.Join(dir, "metadata.json"))
	if err != nil {
		return nil, nil, fmt.Errorf("read metadata: %w", err)
	}

	var result WorkflowCaptureResult
	if err := json.Unmarshal(metaData, &result); err != nil {
		return nil, nil, fmt.Errorf("parse metadata: %w", err)
	}

	var videoFiles []SnapshotFile
	vidDir := filepath.Join(dir, "videos")
	if entries, err := os.ReadDir(vidDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			info, err := entry.Info()
			if err != nil {
				continue
			}
			videoFiles = append(videoFiles, SnapshotFile{
				Filename:  entry.Name(),
				SizeBytes: info.Size(),
			})
		}
	}

	return &result, videoFiles, nil
}

// GetWorkflowVideo returns raw video bytes, validating the filename.
func (s *VisualCaptureStorage) GetWorkflowVideo(repoID int64, scenarioSlug, captureID, filename string) ([]byte, error) {
	if err := validateFilename(filename); err != nil {
		return nil, err
	}
	dir, err := s.workflowCaptureDir(repoID, scenarioSlug, captureID)
	if err != nil {
		return nil, err
	}
	return s.fs.ReadFile(filepath.Join(dir, "videos", filename))
}

// DeleteWorkflowCapture removes a single workflow capture directory.
func (s *VisualCaptureStorage) DeleteWorkflowCapture(repoID int64, scenarioSlug, captureID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dir, err := s.workflowCaptureDir(repoID, scenarioSlug, captureID)
	if err != nil {
		return err
	}
	return os.RemoveAll(dir)
}

// DeleteWorkflowCapturesByRole removes all workflow captures matching the given role.
func (s *VisualCaptureStorage) DeleteWorkflowCapturesByRole(repoID int64, scenarioSlug, role string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	results, err := s.ListWorkflowCaptures(repoID, scenarioSlug)
	if err != nil {
		return err
	}
	for _, r := range results {
		if r.Role == role {
			dir, err := s.workflowCaptureDir(repoID, scenarioSlug, r.ID)
			if err != nil {
				return err
			}
			if err := os.RemoveAll(dir); err != nil {
				return fmt.Errorf("delete workflow capture %s: %w", r.ID, err)
			}
		}
	}
	return nil
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
