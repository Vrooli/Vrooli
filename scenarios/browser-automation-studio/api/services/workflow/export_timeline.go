package workflow

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/google/uuid"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	executionwriter "github.com/vrooli/browser-automation-studio/automation/execution-writer"
	"github.com/vrooli/browser-automation-studio/internal/enums"
	"github.com/vrooli/browser-automation-studio/internal/typeconv"
	"github.com/vrooli/browser-automation-studio/services/export"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	"google.golang.org/protobuf/encoding/protojson"
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

// GetExecutionTimelineProto reads the on-disk proto timeline (preferred) for a given execution.
// Falls back to a minimal proto timeline if no file exists yet.
func (s *WorkflowService) GetExecutionTimelineProto(ctx context.Context, executionID uuid.UUID) (*bastimeline.ExecutionTimeline, error) {
	execution, err := s.repo.GetExecution(ctx, executionID)
	if err != nil {
		return nil, err
	}

	pb := &bastimeline.ExecutionTimeline{
		ExecutionId: execution.ID.String(),
		WorkflowId:  execution.WorkflowID.String(),
		Status:      enums.StringToExecutionStatus(execution.Status),
		Progress:    0,
		StartedAt:   autocontracts.TimeToTimestamp(execution.StartedAt),
	}
	if execution.CompletedAt != nil {
		pb.CompletedAt = autocontracts.TimePtrToTimestamp(execution.CompletedAt)
	}

	if strings.TrimSpace(execution.ResultPath) == "" {
		return pb, nil
	}

	timelinePath := filepath.Join(filepath.Dir(execution.ResultPath), "timeline.proto.json")
	raw, err := os.ReadFile(timelinePath)
	if err != nil {
		if os.IsNotExist(err) {
			return pb, nil
		}
		return nil, fmt.Errorf("read proto timeline: %w", err)
	}
	var parsed bastimeline.ExecutionTimeline
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("parse proto timeline: %w", err)
	}

	// Ensure key fields reflect current index data.
	if strings.TrimSpace(parsed.ExecutionId) == "" {
		parsed.ExecutionId = pb.ExecutionId
	}
	if strings.TrimSpace(parsed.WorkflowId) == "" {
		parsed.WorkflowId = pb.WorkflowId
	}
	parsed.Status = pb.Status
	parsed.StartedAt = pb.StartedAt
	parsed.CompletedAt = pb.CompletedAt
	return &parsed, nil
}

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
	stepByIndex := make(map[int]*executionwriter.StepResultData, len(resultData.Steps))
	for i := range resultData.Steps {
		step := &resultData.Steps[i]
		stepByIndex[step.StepIndex] = step
	}

	artifactByID := make(map[string]*executionwriter.ArtifactData, len(resultData.Artifacts))
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
		message := strings.TrimSpace(telemetry.Data.Note)
		level := "info"
		switch telemetry.Data.Kind {
		case "console":
			level = "info"
			if len(telemetry.Data.Console) > 0 && strings.TrimSpace(message) == "" {
				first := telemetry.Data.Console[0]
				message = strings.TrimSpace(first.Text)
				if t := strings.TrimSpace(first.Type); t != "" {
					level = strings.ToLower(t)
				}
			}
		case "network":
			level = "debug"
			if len(telemetry.Data.Network) > 0 && strings.TrimSpace(message) == "" {
				first := telemetry.Data.Network[0]
				message = strings.TrimSpace(first.URL)
			}
		case "retry":
			level = "warn"
		case "progress":
			level = "info"
		case "heartbeat":
			level = "info"
			if telemetry.Data.Heartbeat != nil && strings.TrimSpace(message) == "" {
				message = strings.TrimSpace(telemetry.Data.Heartbeat.Message)
			}
		}

		if message == "" {
			continue
		}

		timelineLogs = append(timelineLogs, TimelineLog{
			ID:        fmt.Sprintf("log-%d", i),
			Level:     level,
			Message:   message,
			StepName:  fmt.Sprintf("step-%d", telemetry.StepIndex),
			Timestamp: telemetry.Timestamp,
		})
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
	frameData *executionwriter.TimelineFrameData,
	artifacts map[string]*executionwriter.ArtifactData,
	stepsByIndex map[int]*executionwriter.StepResultData,
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
