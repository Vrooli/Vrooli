package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	executionwriter "github.com/vrooli/browser-automation-studio/automation/execution-writer"
	"github.com/vrooli/browser-automation-studio/database"
	bastelemetry "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/domain"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
)

func (s *WorkflowService) readExecutionResult(resultPath string) (*executionwriter.ExecutionResultData, error) {
	data, err := os.ReadFile(resultPath)
	if err != nil {
		return nil, err
	}
	var result executionwriter.ExecutionResultData
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parse result JSON: %w", err)
	}
	return &result, nil
}

// GetExecutionScreenshots reads screenshots from the execution result file and returns proto types.
func (s *WorkflowService) GetExecutionScreenshots(ctx context.Context, executionID uuid.UUID) ([]*basexecution.ExecutionScreenshot, error) {
	execution, err := s.repo.GetExecution(ctx, executionID)
	if err != nil {
		return nil, fmt.Errorf("get execution: %w", err)
	}
	if execution.ResultPath == "" {
		return []*basexecution.ExecutionScreenshot{}, nil
	}

	resultData, err := s.readExecutionResult(execution.ResultPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []*basexecution.ExecutionScreenshot{}, nil
		}
		return nil, fmt.Errorf("read result file: %w", err)
	}

	screenshots := make([]*basexecution.ExecutionScreenshot, 0)
	for _, artifact := range resultData.Artifacts {
		if artifact.ArtifactType != "screenshot" && artifact.ArtifactType != "screenshot_inline" {
			continue
		}
		screenshot := &basexecution.ExecutionScreenshot{
			Screenshot: &bastelemetry.TimelineScreenshot{
				ArtifactId:   artifact.ArtifactID,
				Url:          artifact.StorageURL,
				ThumbnailUrl: artifact.ThumbnailURL,
				ContentType:  artifact.ContentType,
			},
			NodeId: "",
		}
		if artifact.SizeBytes != nil {
			screenshot.Screenshot.SizeBytes = artifact.SizeBytes
		}
		if artifact.StepIndex != nil {
			screenshot.StepIndex = int32(*artifact.StepIndex)
		}
		if artifact.Label != "" {
			screenshot.StepLabel = &artifact.Label
		}

		// Find the step to get node ID
		for _, step := range resultData.Steps {
			if artifact.StepIndex != nil && step.StepIndex == *artifact.StepIndex {
				screenshot.NodeId = step.NodeID
				break
			}
		}
		screenshots = append(screenshots, screenshot)
	}
	return screenshots, nil
}

// ExecutionVideoArtifact captures recorded video artifact metadata.
type ExecutionVideoArtifact struct {
	ArtifactID  string         `json:"artifact_id"`
	StorageURL  string         `json:"storage_url,omitempty"`
	ContentType string         `json:"content_type,omitempty"`
	Label       string         `json:"label,omitempty"`
	SizeBytes   *int64         `json:"size_bytes,omitempty"`
	Payload     map[string]any `json:"payload,omitempty"`
}

// ExecutionFileArtifact captures trace/HAR artifact metadata.
type ExecutionFileArtifact struct {
	ArtifactID  string         `json:"artifact_id"`
	StorageURL  string         `json:"storage_url,omitempty"`
	ContentType string         `json:"content_type,omitempty"`
	Label       string         `json:"label,omitempty"`
	SizeBytes   *int64         `json:"size_bytes,omitempty"`
	Payload     map[string]any `json:"payload,omitempty"`
}

// GetExecutionVideoArtifacts reads recorded video artifacts from the execution result file.
func (s *WorkflowService) GetExecutionVideoArtifacts(ctx context.Context, executionID uuid.UUID) ([]ExecutionVideoArtifact, error) {
	execution, err := s.repo.GetExecution(ctx, executionID)
	if err != nil {
		return nil, fmt.Errorf("get execution: %w", err)
	}
	if execution.ResultPath == "" {
		return []ExecutionVideoArtifact{}, nil
	}

	resultData, err := s.readExecutionResult(execution.ResultPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []ExecutionVideoArtifact{}, nil
		}
		return nil, fmt.Errorf("read result file: %w", err)
	}

	videos := make([]ExecutionVideoArtifact, 0)
	for _, artifact := range resultData.Artifacts {
		if artifact.ArtifactType != "video_meta" && artifact.ArtifactType != "video" {
			continue
		}
		videos = append(videos, ExecutionVideoArtifact{
			ArtifactID:  artifact.ArtifactID,
			StorageURL:  artifact.StorageURL,
			ContentType: artifact.ContentType,
			Label:       artifact.Label,
			SizeBytes:   artifact.SizeBytes,
			Payload:     artifact.Payload,
		})
	}
	return videos, nil
}

// GetExecutionTraceArtifacts reads trace artifacts from the execution result file.
func (s *WorkflowService) GetExecutionTraceArtifacts(ctx context.Context, executionID uuid.UUID) ([]ExecutionFileArtifact, error) {
	return s.getExecutionFileArtifacts(ctx, executionID, map[string]bool{
		"trace":      true,
		"trace_meta": true,
	})
}

// GetExecutionHarArtifacts reads HAR artifacts from the execution result file.
func (s *WorkflowService) GetExecutionHarArtifacts(ctx context.Context, executionID uuid.UUID) ([]ExecutionFileArtifact, error) {
	return s.getExecutionFileArtifacts(ctx, executionID, map[string]bool{
		"har":      true,
		"har_meta": true,
	})
}

func (s *WorkflowService) getExecutionFileArtifacts(ctx context.Context, executionID uuid.UUID, types map[string]bool) ([]ExecutionFileArtifact, error) {
	execution, err := s.repo.GetExecution(ctx, executionID)
	if err != nil {
		return nil, fmt.Errorf("get execution: %w", err)
	}
	if execution.ResultPath == "" {
		return []ExecutionFileArtifact{}, nil
	}

	resultData, err := s.readExecutionResult(execution.ResultPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []ExecutionFileArtifact{}, nil
		}
		return nil, fmt.Errorf("read result file: %w", err)
	}

	items := make([]ExecutionFileArtifact, 0)
	for _, artifact := range resultData.Artifacts {
		if !types[strings.ToLower(strings.TrimSpace(artifact.ArtifactType))] {
			continue
		}
		storageURL := artifact.StorageURL
		if storageURL == "" {
			storageURL = s.assetURLFromPayload(executionID, artifact.Payload)
		}
		items = append(items, ExecutionFileArtifact{
			ArtifactID:  artifact.ArtifactID,
			StorageURL:  storageURL,
			ContentType: artifact.ContentType,
			Label:       artifact.Label,
			SizeBytes:   artifact.SizeBytes,
			Payload:     artifact.Payload,
		})
	}
	return items, nil
}

func (s *WorkflowService) assetURLFromPayload(executionID uuid.UUID, payload map[string]any) string {
	if payload == nil || strings.TrimSpace(s.executionDataRoot) == "" {
		return ""
	}
	rawPath, ok := payload["path"]
	if !ok {
		return ""
	}
	path, ok := rawPath.(string)
	if !ok || strings.TrimSpace(path) == "" {
		return ""
	}
	base := filepath.Join(s.executionDataRoot, executionID.String())
	rel, err := filepath.Rel(base, path)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return ""
	}
	rel = filepath.ToSlash(rel)
	return fmt.Sprintf("/api/v1/recordings/assets/%s/%s", executionID.String(), rel)
}

// UpdateExecutionResultPath updates the DB execution index with the given result file path.
// This is used by the recorder when it first writes the result file.
func (s *WorkflowService) UpdateExecutionResultPath(ctx context.Context, executionID uuid.UUID, resultPath string, updatedAt time.Time) error {
	return s.repo.UpdateExecutionResultPath(ctx, executionID, resultPath, updatedAt)
}

// Ensure our workflow service satisfies the recorder's minimal interface when used as ExecutionIndexRepository.
var _ executionwriter.ExecutionIndexRepository = (*WorkflowService)(nil)

func (s *WorkflowService) GetExecution(ctx context.Context, id uuid.UUID) (*database.ExecutionIndex, error) {
	return s.repo.GetExecution(ctx, id)
}

func (s *WorkflowService) UpdateExecutionStatus(ctx context.Context, id uuid.UUID, status string, errorMessage *string, completedAt *time.Time, updatedAt time.Time) error {
	return s.repo.UpdateExecutionStatus(ctx, id, status, errorMessage, completedAt, updatedAt)
}

func (s *WorkflowService) UpdateExecution(ctx context.Context, execution *database.ExecutionIndex) error {
	return s.repo.UpdateExecution(ctx, execution)
}
