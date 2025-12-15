package workflow

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/google/uuid"
	autorecorder "github.com/vrooli/browser-automation-studio/automation/recorder"
	"github.com/vrooli/browser-automation-studio/internal/typeconv"
	"github.com/vrooli/browser-automation-studio/services/export"
)

// Type aliases for backward compatibility with existing workflow package code.
type (
	ExecutionTimeline  = export.ExecutionTimeline
	TimelineFrame      = export.TimelineFrame
	TimelineLog        = export.TimelineLog
	RetryHistoryEntry  = typeconv.RetryHistoryEntry
	TimelineScreenshot = typeconv.TimelineScreenshot
	TimelineArtifact   = typeconv.TimelineArtifact
)

// GetExecutionTimeline assembles replay-ready artifacts for a given execution.
// Reads execution data from the result JSON file stored on disk.
func (s *WorkflowService) GetExecutionTimeline(ctx context.Context, executionID uuid.UUID) (*ExecutionTimeline, error) {
	execution, err := s.repo.GetExecution(ctx, executionID)
	if err != nil {
		return nil, err
	}

	// If no result file exists yet, return a minimal timeline
	if execution.ResultPath == "" {
		return &ExecutionTimeline{
			ExecutionID: execution.ID,
			WorkflowID:  execution.WorkflowID,
			Status:      execution.Status,
			StartedAt:   execution.StartedAt,
			CompletedAt: execution.CompletedAt,
			Frames:      []TimelineFrame{},
			Logs:        []TimelineLog{},
		}, nil
	}

	// Read the execution result file
	resultData, err := s.readExecutionResult(execution.ResultPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Result file not found, return minimal timeline
			return &ExecutionTimeline{
				ExecutionID: execution.ID,
				WorkflowID:  execution.WorkflowID,
				Status:      execution.Status,
				StartedAt:   execution.StartedAt,
				CompletedAt: execution.CompletedAt,
				Frames:      []TimelineFrame{},
				Logs:        []TimelineLog{},
			}, nil
		}
		return nil, fmt.Errorf("read result file: %w", err)
	}

	// Build indexes for artifacts and steps
	stepByIndex := make(map[int]*autorecorder.StepResultData, len(resultData.Steps))
	for i := range resultData.Steps {
		step := &resultData.Steps[i]
		stepByIndex[step.StepIndex] = step
	}

	artifactByID := make(map[string]*autorecorder.ArtifactData, len(resultData.Artifacts))
	for i := range resultData.Artifacts {
		artifact := &resultData.Artifacts[i]
		artifactByID[artifact.ArtifactID] = artifact
	}

	// Build timeline frames from timeline frame data
	frames := make([]TimelineFrame, 0, len(resultData.TimelineFrame))
	for _, frameData := range resultData.TimelineFrame {
		frame := s.buildTimelineFrameFromData(&frameData, artifactByID, stepByIndex)
		frames = append(frames, frame)
	}

	sort.Slice(frames, func(i, j int) bool {
		if frames[i].StepIndex != frames[j].StepIndex {
			return frames[i].StepIndex < frames[j].StepIndex
		}
		return frames[i].NodeID < frames[j].NodeID
	})

	// Build logs from telemetry data (telemetry can contain log-level information)
	timelineLogs := make([]TimelineLog, 0)
	for i, telemetry := range resultData.Telemetry {
		// Extract log entries from telemetry if present
		if telemetry.Data.Message != "" {
			level := strings.ToLower(strings.TrimSpace(telemetry.Data.Level))
			if level == "" {
				level = "info"
			}
			timelineLogs = append(timelineLogs, TimelineLog{
				ID:        fmt.Sprintf("log-%d", i),
				Level:     level,
				Message:   telemetry.Data.Message,
				StepName:  fmt.Sprintf("step-%d", telemetry.StepIndex),
				Timestamp: telemetry.Timestamp,
			})
		}
	}

	// Calculate progress from summary
	progress := 0
	if resultData.Summary.TotalSteps > 0 {
		progress = (resultData.Summary.CompletedSteps * 100) / resultData.Summary.TotalSteps
	}

	return &ExecutionTimeline{
		ExecutionID: execution.ID,
		WorkflowID:  execution.WorkflowID,
		Status:      execution.Status,
		Progress:    progress,
		StartedAt:   execution.StartedAt,
		CompletedAt: execution.CompletedAt,
		Frames:      frames,
		Logs:        timelineLogs,
	}, nil
}

// buildTimelineFrameFromData constructs a TimelineFrame from recorder data types.
// This reads from the execution result JSON file format.
func (s *WorkflowService) buildTimelineFrameFromData(
	frameData *autorecorder.TimelineFrameData,
	artifacts map[string]*autorecorder.ArtifactData,
	stepsByIndex map[int]*autorecorder.StepResultData,
) TimelineFrame {
	payload := frameData.Payload
	if payload == nil {
		payload = map[string]any{}
	}

	// Extract data from payload (timeline frame metadata)
	stepIndex := frameData.StepIndex
	nodeID := frameData.NodeID
	stepType := frameData.StepType
	success := frameData.Success
	duration := frameData.DurationMs

	// Additional metadata from payload
	progress := typeconv.ToInt(payload["progress"])
	finalURL := typeconv.ToString(payload["finalUrl"])
	errorMsg := typeconv.ToString(payload["error"])
	consoleCount := typeconv.ToInt(payload["consoleLogCount"])
	networkCount := typeconv.ToInt(payload["networkEventCount"])
	zoomFactor := typeconv.ToFloat(payload["zoomFactor"])
	extractedPreview := payload["extractedDataPreview"]
	cursorTrail := typeconv.ToPointSlice(payload["cursorTrail"])
	assertion := typeconv.ToAssertionOutcome(payload["assertion"])
	totalDuration := typeconv.ToInt(payload["totalDurationMs"])

	// Retry information
	retryAttempt := frameData.Attempt
	if retryAttempt == 0 {
		retryAttempt = typeconv.ToInt(payload["retryAttempt"])
	}
	retryMaxAttempts := typeconv.ToInt(payload["retryMaxAttempts"])
	retryConfigured := typeconv.ToInt(payload["retryConfigured"])
	retryDelayMs := typeconv.ToInt(payload["retryDelayMs"])
	retryBackoff := typeconv.ToFloat(payload["retryBackoffFactor"])
	retryHistory := typeconv.ToRetryHistory(payload["retryHistory"])

	// DOM snapshot info
	domSnapshotPreview := typeconv.ToString(payload["domSnapshotPreview"])
	domSnapshotArtifactID := frameData.DOMSnapshotArtifactID
	if domSnapshotArtifactID == "" {
		domSnapshotArtifactID = typeconv.ToString(payload["domSnapshotArtifactId"])
	}

	// Timestamps
	startedAt := typeconv.ToTimePtr(payload["startedAt"])
	completedAt := typeconv.ToTimePtr(payload["completedAt"])

	// Determine status
	var status string
	if success {
		status = "completed"
	} else if errorMsg != "" {
		status = "failed"
	}

	// Enrich from step data if available
	if step := stepsByIndex[stepIndex]; step != nil {
		if status == "" && step.Status != "" {
			status = step.Status
		}
		if duration == 0 && step.DurationMs > 0 {
			duration = step.DurationMs
		}
		if startedAt == nil {
			startedAt = &step.StartedAt
		}
		if completedAt == nil && step.CompletedAt != nil {
			completedAt = step.CompletedAt
		}
		if errorMsg == "" && step.Error != "" {
			errorMsg = step.Error
		}
	}

	if status == "" {
		if success {
			status = "completed"
		} else if errorMsg != "" {
			status = "failed"
		} else {
			status = "unknown"
		}
	}

	// Visual metadata
	highlightRegions := typeconv.ToHighlightRegions(payload["highlightRegions"])
	maskRegions := typeconv.ToMaskRegions(payload["maskRegions"])
	focusedElement := typeconv.ToElementFocus(payload["focusedElement"])
	elementBoundingBox := typeconv.ToBoundingBox(payload["elementBoundingBox"])
	clickPosition := typeconv.ToPoint(payload["clickPosition"])

	// Screenshot
	var screenshot *TimelineScreenshot
	screenshotID := frameData.ScreenshotArtifactID
	if screenshotID == "" {
		screenshotID = typeconv.ToString(payload["screenshotArtifactId"])
	}
	if screenshotID != "" {
		if artifact := artifacts[screenshotID]; artifact != nil {
			screenshot = typeconv.ToTimelineScreenshot(artifact)
		}
	}

	// Related artifacts
	artifactRefs := make([]TimelineArtifact, 0)
	for _, id := range typeconv.ToStringSlice(payload["artifactIds"]) {
		if artifact := artifacts[id]; artifact != nil {
			artifactRefs = append(artifactRefs, typeconv.ToTimelineArtifact(artifact))
		}
	}

	// DOM snapshot artifact
	var domSnapshot *TimelineArtifact
	if domSnapshotArtifactID != "" {
		if artifact := artifacts[domSnapshotArtifactID]; artifact != nil {
			artifactCopy := typeconv.ToTimelineArtifact(artifact)
			domSnapshot = &artifactCopy
		}
	}

	return TimelineFrame{
		StepIndex:            stepIndex,
		NodeID:               nodeID,
		StepType:             stepType,
		Status:               status,
		Success:              success,
		DurationMs:           duration,
		TotalDurationMs:      totalDuration,
		Progress:             progress,
		StartedAt:            startedAt,
		CompletedAt:          completedAt,
		FinalURL:             finalURL,
		Error:                errorMsg,
		ConsoleLogCount:      consoleCount,
		NetworkEventCount:    networkCount,
		ExtractedDataPreview: extractedPreview,
		HighlightRegions:     highlightRegions,
		MaskRegions:          maskRegions,
		FocusedElement:       focusedElement,
		ElementBoundingBox:   elementBoundingBox,
		ClickPosition:        clickPosition,
		CursorTrail:          cursorTrail,
		ZoomFactor:           zoomFactor,
		Screenshot:           screenshot,
		Artifacts:            artifactRefs,
		Assertion:            assertion,
		RetryAttempt:         retryAttempt,
		RetryMaxAttempts:     retryMaxAttempts,
		RetryConfigured:      retryConfigured,
		RetryDelayMs:         retryDelayMs,
		RetryBackoffFactor:   retryBackoff,
		RetryHistory:         retryHistory,
		DomSnapshotPreview:   domSnapshotPreview,
		DomSnapshot:          domSnapshot,
	}
}
