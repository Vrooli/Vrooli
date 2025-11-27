package workflow

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/services/export"
)

// DescribeExecutionExport returns the current replay export status for an execution.
func (s *WorkflowService) DescribeExecutionExport(ctx context.Context, executionID uuid.UUID) (*ExecutionExportPreview, error) {
	execution, err := s.repo.GetExecution(ctx, executionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, database.ErrNotFound) {
			return nil, database.ErrNotFound
		}
		return nil, err
	}

	var workflow *database.Workflow
	if wf, wfErr := s.repo.GetWorkflow(ctx, execution.WorkflowID); wfErr == nil {
		workflow = wf
	} else if !errors.Is(wfErr, database.ErrNotFound) {
		return nil, wfErr
	}

	timeline, err := s.GetExecutionTimeline(ctx, executionID)
	if err != nil {
		return nil, err
	}

	capturedFrames := len(timeline.Frames)
	assetCount := 0
	for _, frame := range timeline.Frames {
		if frame.Screenshot != nil {
			assetCount++
		}
		assetCount += len(frame.Artifacts)
	}
	totalDurationMs := 0
	if timeline != nil {
		for _, frame := range timeline.Frames {
			if frame.TotalDurationMs > 0 {
				totalDurationMs += frame.TotalDurationMs
			} else if frame.DurationMs > 0 {
				totalDurationMs += frame.DurationMs
			}
		}
	}
	specID := execution.ID.String()

	if capturedFrames == 0 {
		status := strings.ToLower(strings.TrimSpace(execution.Status))
		previewStatus := "pending"
		message := "Replay export pending – timeline frames not captured yet"
		if status == "failed" {
			previewStatus = "unavailable"
			message = "Replay export unavailable – execution failed before capturing any steps"
		} else if status == "completed" {
			previewStatus = "unavailable"
			message = "Replay export unavailable – workflow finished without timeline frames"
		}
		preview := &ExecutionExportPreview{
			ExecutionID:         execution.ID,
			SpecID:              specID,
			Status:              previewStatus,
			Message:             message,
			CapturedFrameCount:  capturedFrames,
			AvailableAssetCount: assetCount,
			TotalDurationMs:     totalDurationMs,
			Package:             nil,
		}
		if s.log != nil {
			s.log.WithFields(logrus.Fields{
				"execution_id":      execution.ID,
				"workflow_id":       execution.WorkflowID,
				"export_status":     previewStatus,
				"captured_frames":   capturedFrames,
				"available_assets":  assetCount,
				"timeline_total_ms": totalDurationMs,
			}).Debug("DescribeExecutionExport returning preview")
		}
		return preview, nil
	}

	exportPackage, err := export.BuildReplayMovieSpec(execution, workflow, timeline)
	if err != nil {
		return nil, err
	}

	frameCount := exportPackage.Summary.FrameCount
	if frameCount == 0 {
		frameCount = len(timeline.Frames)
	}
	message := fmt.Sprintf("Replay export ready (%d frames, %dms)", frameCount, exportPackage.Summary.TotalDurationMs)
	assetCount = len(exportPackage.Assets)
	if exportPackage.Summary.TotalDurationMs > 0 {
		totalDurationMs = exportPackage.Summary.TotalDurationMs
	}
	if exportPackage.Execution.ExecutionID != uuid.Nil {
		specID = exportPackage.Execution.ExecutionID.String()
	}

	preview := &ExecutionExportPreview{
		ExecutionID:         execution.ID,
		SpecID:              specID,
		Status:              "ready",
		Message:             message,
		CapturedFrameCount:  frameCount,
		AvailableAssetCount: assetCount,
		TotalDurationMs:     totalDurationMs,
		Package:             exportPackage,
	}

	if s.log != nil {
		s.log.WithFields(logrus.Fields{
			"execution_id":      execution.ID,
			"workflow_id":       execution.WorkflowID,
			"export_status":     "ready",
			"captured_frames":   frameCount,
			"available_assets":  assetCount,
			"timeline_total_ms": totalDurationMs,
		}).Debug("DescribeExecutionExport returning preview")
	}

	return preview, nil
}
