package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

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

// UpdateExecutionResultPath updates the DB execution index with the given result file path.
// This is used by the recorder when it first writes the result file.
func (s *WorkflowService) UpdateExecutionResultPath(ctx context.Context, executionID uuid.UUID, resultPath string) error {
	exec, err := s.repo.GetExecution(ctx, executionID)
	if err != nil {
		return err
	}
	exec.ResultPath = resultPath
	return s.repo.UpdateExecution(ctx, exec)
}

// Ensure our workflow service satisfies the recorder's minimal interface when used as ExecutionIndexRepository.
var _ executionwriter.ExecutionIndexRepository = (*WorkflowService)(nil)

func (s *WorkflowService) GetExecution(ctx context.Context, id uuid.UUID) (*database.ExecutionIndex, error) {
	return s.repo.GetExecution(ctx, id)
}

func (s *WorkflowService) UpdateExecution(ctx context.Context, execution *database.ExecutionIndex) error {
	return s.repo.UpdateExecution(ctx, execution)
}

