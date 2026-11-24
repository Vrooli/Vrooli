package workflow

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	executionwriter "github.com/vrooli/browser-automation-studio/automation/execution-writer"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/internal/executionartifactlock"
	exportservices "github.com/vrooli/browser-automation-studio/services/export"
	basevidence "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/evidence"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
)

// GetExecutionReplayPackage loads the writer-owned, versioned evidence package.
// It is the only supported consumer-facing replay handoff; callers never
// reconstruct it from BAS result or artifact filesystem paths.
func (s *WorkflowService) GetExecutionReplayPackage(ctx context.Context, executionID uuid.UUID) (*basevidence.ReplayPackage, error) {
	return exportservices.NewTimelineLoader(s.repo).LoadReplayPackage(ctx, executionID)
}

// GetExecutionScreenshots delegates timeline reads and screenshot projection to
// the canonical export timeline loader.
func (s *WorkflowService) GetExecutionScreenshots(ctx context.Context, executionID uuid.UUID) ([]*basexecution.ExecutionScreenshot, error) {
	return exportservices.NewTimelineLoader(s.repo).LoadExecutionScreenshots(ctx, executionID)
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

// GetExecutionVideoArtifacts adapts the canonical artifact catalog to the
// workflow service's public contract.
func (s *WorkflowService) GetExecutionVideoArtifacts(ctx context.Context, executionID uuid.UUID) ([]ExecutionVideoArtifact, error) {
	files, err := exportservices.NewArtifactCatalog(s.executionDataRoot).List(ctx, executionID, "videos")
	if err != nil {
		return nil, err
	}
	videos := make([]ExecutionVideoArtifact, 0, len(files))
	for _, file := range files {
		videos = append(videos, ExecutionVideoArtifact{
			ArtifactID: file.ArtifactID, StorageURL: file.StorageURL, ContentType: file.ContentType,
			Label: file.Label, SizeBytes: &file.SizeBytes, Payload: file.Payload,
		})
	}
	return videos, nil
}

// GetExecutionTraceArtifacts adapts the canonical artifact catalog to the
// workflow service's public contract.
func (s *WorkflowService) GetExecutionTraceArtifacts(ctx context.Context, executionID uuid.UUID) ([]ExecutionFileArtifact, error) {
	return workflowFileArtifacts(exportservices.NewArtifactCatalog(s.executionDataRoot), ctx, executionID, "traces")
}

// GetExecutionHarArtifacts adapts the canonical artifact catalog to the
// workflow service's public contract.
func (s *WorkflowService) GetExecutionHarArtifacts(ctx context.Context, executionID uuid.UUID) ([]ExecutionFileArtifact, error) {
	return workflowFileArtifacts(exportservices.NewArtifactCatalog(s.executionDataRoot), ctx, executionID, "har")
}

func workflowFileArtifacts(catalog *exportservices.ArtifactCatalog, ctx context.Context, executionID uuid.UUID, kind string) ([]ExecutionFileArtifact, error) {
	files, err := catalog.List(ctx, executionID, kind)
	if err != nil {
		return nil, err
	}
	items := make([]ExecutionFileArtifact, 0, len(files))
	for _, file := range files {
		items = append(items, ExecutionFileArtifact{
			ArtifactID: file.ArtifactID, StorageURL: file.StorageURL, ContentType: file.ContentType,
			Label: file.Label, SizeBytes: &file.SizeBytes, Payload: file.Payload,
		})
	}
	return items, nil
}

// UpdateExecutionResultPath updates the DB execution index with the given result file path.
// This is used by the recorder when it first writes the result file.
func (s *WorkflowService) UpdateExecutionResultPath(ctx context.Context, executionID uuid.UUID, resultPath string, updatedAt time.Time) error {
	return executionartifactlock.WithLock(ctx, s.executionDataRootOrDefault(), executionID, func(lockCtx context.Context) error {
		return s.repo.UpdateExecutionResultPath(lockCtx, executionID, resultPath, updatedAt)
	})
}

// Ensure our workflow service satisfies the recorder's minimal interface when used as ExecutionIndexRepository.
var _ executionwriter.ExecutionIndexRepository = (*WorkflowService)(nil)

func (s *WorkflowService) GetExecution(ctx context.Context, id uuid.UUID) (*database.ExecutionIndex, error) {
	execution, err := s.repo.GetExecution(ctx, id)
	if err != nil || execution == nil || execution.CompletedAt == nil {
		return execution, err
	}
	// A persisted terminal status precedes deferred sink cleanup. Readers that
	// use completion to release routed fixtures must join that local owner too.
	// Ending an observation cancels only the read, never the durable execution.
	if value, ok := s.executionCancels.Load(id); ok {
		if control, ok := value.(executionControl); ok && control.done != nil {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-control.done:
			}
		}
	}
	return s.repo.GetExecution(ctx, id)
}

// WaitForExecution joins the local runner for one caller-bounded observation.
// Ending the wait does not cancel execution; callers that want cancellation
// must use StopExecution with this exact execution ID.
func (s *WorkflowService) WaitForExecution(ctx context.Context, id uuid.UUID) (*database.ExecutionIndex, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("workflow execution service not configured")
	}
	if value, ok := s.executionCancels.Load(id); ok {
		control, valid := value.(executionControl)
		if !valid || control.done == nil {
			return nil, errors.New("execution completion owner is invalid")
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-control.done:
		}
	} else {
		// A restarted process may still serve an already-persisted terminal
		// receipt, but it must not pretend it can join an active runner it does
		// not own.
		latest, err := s.repo.GetExecution(ctx, id)
		if err != nil {
			return nil, err
		}
		if latest == nil {
			return nil, database.ErrNotFound
		}
		if latest.CompletedAt == nil {
			return nil, fmt.Errorf("execution %s is %s but this API process has no completion owner", id, latest.Status)
		}
	}
	latest, err := s.GetExecution(ctx, id)
	if err != nil {
		return nil, err
	}
	if latest == nil || latest.CompletedAt == nil {
		return nil, fmt.Errorf("execution %s runner exited without a persisted terminal result", id)
	}
	return latest, nil
}

func (s *WorkflowService) UpdateExecutionStatus(ctx context.Context, id uuid.UUID, status string, errorMessage *string, completedAt *time.Time, updatedAt time.Time) error {
	return s.persistExecutionStatus(ctx, id, status, errorMessage, completedAt, updatedAt)
}

func (s *WorkflowService) persistExecutionStatus(ctx context.Context, id uuid.UUID, status string, errorMessage *string, completedAt *time.Time, updatedAt time.Time) error {
	return executionartifactlock.WithLock(ctx, s.executionDataRootOrDefault(), id, func(lockCtx context.Context) error {
		return s.repo.UpdateExecutionStatus(lockCtx, id, status, errorMessage, completedAt, updatedAt)
	})
}

func (s *WorkflowService) UpdateExecution(ctx context.Context, execution *database.ExecutionIndex) error {
	if execution == nil || execution.ID == uuid.Nil {
		return errors.New("execution index is nil")
	}
	return executionartifactlock.WithLock(ctx, s.executionDataRootOrDefault(), execution.ID, func(lockCtx context.Context) error {
		return s.repo.UpdateExecution(lockCtx, execution)
	})
}
