package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/vrooli/api-core/storage"
)

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
