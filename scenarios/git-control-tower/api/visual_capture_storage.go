package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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

// dataRoot resolves the git-control-tower data class root (the parent of the
// per-repoID directories). Used by the orphaned workflow-captures cleanup.
func (s *VisualCaptureStorage) dataRoot() (string, error) {
	return s.resolver.Path(
		storage.Options{ScenarioID: "git-control-tower"},
		storage.ClassData,
		"",
	)
}

func (s *VisualCaptureStorage) repoSnapshotsRoot(repoID int64) (string, error) {
	return s.resolver.Path(
		storage.Options{ScenarioID: "git-control-tower"},
		storage.ClassData,
		fmt.Sprintf("%d/snapshots", repoID),
	)
}

// writeFilesToSubdir creates a subdirectory and writes all files into it.
func (s *VisualCaptureStorage) writeFilesToSubdir(parentDir, subdir string, files map[string][]byte, label string) error {
	if len(files) == 0 {
		return nil
	}
	dir := filepath.Join(parentDir, subdir)
	if err := s.fs.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create %s dir: %w", label, err)
	}
	for name, data := range files {
		if err := s.fs.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			return fmt.Errorf("write %s %s: %w", label, name, err)
		}
	}
	return nil
}

// totalBytesSize returns the total byte count across multiple file maps.
func totalBytesSize(fileMaps ...map[string][]byte) int64 {
	var total int64
	for _, m := range fileMaps {
		for _, data := range m {
			total += int64(len(data))
		}
	}
	return total
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

	if err := s.writeFilesToSubdir(dir, "screenshots", screenshots, "screenshot"); err != nil {
		return err
	}
	if err := s.writeFilesToSubdir(dir, "videos", videos, "video"); err != nil {
		return err
	}

	meta.SizeBytes = totalBytesSize(screenshots, videos)

	metaBytes, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}
	if err := s.fs.WriteFile(filepath.Join(dir, "metadata.json"), metaBytes, 0o644); err != nil {
		return fmt.Errorf("write metadata: %w", err)
	}

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

// listDirFiles lists non-directory entries in a directory as SnapshotFile slices.
func listDirFiles(dir string) []SnapshotFile {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var files []SnapshotFile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, SnapshotFile{
			Filename:  entry.Name(),
			SizeBytes: info.Size(),
		})
	}
	return files
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

	for _, sf := range listDirFiles(filepath.Join(dir, "screenshots")) {
		if vw, vh, theme, ok := parsePresetFromFilename(sf.Filename); ok {
			sf.ViewportWidth = vw
			sf.ViewportHeight = vh
			sf.Theme = theme
		}
		detail.Screenshots = append(detail.Screenshots, sf)
	}

	if vids := listDirFiles(filepath.Join(dir, "videos")); vids != nil {
		detail.Videos = vids
	}

	return detail, nil
}

// GetScreenshotFilePath returns the absolute filesystem path to a screenshot without reading the file.
func (s *VisualCaptureStorage) GetScreenshotFilePath(repoID int64, scenarioSlug, snapshotID, filename string) (string, error) {
	if err := validateFilename(filename); err != nil {
		return "", err
	}
	dir, err := s.snapshotDir(repoID, scenarioSlug, snapshotID)
	if err != nil {
		return "", err
	}
	fp := filepath.Join(dir, "screenshots", filename)
	if _, err := os.Stat(fp); err != nil {
		return "", fmt.Errorf("screenshot not found: %w", err)
	}
	return fp, nil
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
