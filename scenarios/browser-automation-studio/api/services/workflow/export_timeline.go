package workflow

import (
	"context"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
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
func (s *WorkflowService) GetExecutionTimeline(ctx context.Context, executionID uuid.UUID) (*ExecutionTimeline, error) {
	execution, err := s.repo.GetExecution(ctx, executionID)
	if err != nil {
		return nil, err
	}

	artifacts, err := s.repo.ListExecutionArtifacts(ctx, executionID)
	if err != nil {
		return nil, err
	}

	steps, err := s.repo.ListExecutionSteps(ctx, executionID)
	if err != nil {
		return nil, err
	}

	stepByID := make(map[uuid.UUID]*database.ExecutionStep, len(steps))
	stepByIndex := make(map[int]*database.ExecutionStep, len(steps))
	for _, step := range steps {
		if step == nil {
			continue
		}
		stepByID[step.ID] = step
		stepByIndex[step.StepIndex] = step
	}

	artifactByID := make(map[string]*database.ExecutionArtifact, len(artifacts))
	for _, artifact := range artifacts {
		if artifact == nil {
			continue
		}
		artifactByID[artifact.ID.String()] = artifact
	}

	frames := make([]TimelineFrame, 0, len(artifacts))
	for _, artifact := range artifacts {
		if artifact == nil || artifact.ArtifactType != "timeline_frame" {
			continue
		}
		frame, buildErr := s.buildTimelineFrame(artifact, artifactByID, stepByID, stepByIndex)
		if buildErr != nil {
			if s.log != nil {
				s.log.WithError(buildErr).WithField("timeline_artifact_id", artifact.ID).Warn("Failed to build timeline frame")
			}
			continue
		}
		frames = append(frames, frame)
	}

	sort.Slice(frames, func(i, j int) bool {
		if frames[i].StepIndex != frames[j].StepIndex {
			return frames[i].StepIndex < frames[j].StepIndex
		}
		return frames[i].NodeID < frames[j].NodeID
	})

	logs, err := s.repo.GetExecutionLogs(ctx, executionID)
	if err != nil {
		return nil, err
	}

	timelineLogs := make([]TimelineLog, 0, len(logs))
	for _, log := range logs {
		if log == nil {
			continue
		}
		level := strings.ToLower(strings.TrimSpace(log.Level))
		if level == "" {
			level = "info"
		}
		timelineLogs = append(timelineLogs, TimelineLog{
			ID:        log.ID.String(),
			Level:     level,
			Message:   log.Message,
			StepName:  log.StepName,
			Timestamp: log.Timestamp,
		})
	}

	return &ExecutionTimeline{
		ExecutionID: execution.ID,
		WorkflowID:  execution.WorkflowID,
		Status:      execution.Status,
		Progress:    execution.Progress,
		StartedAt:   execution.StartedAt,
		CompletedAt: execution.CompletedAt,
		Frames:      frames,
		Logs:        timelineLogs,
	}, nil
}

func (s *WorkflowService) buildTimelineFrame(
	timelineArtifact *database.ExecutionArtifact,
	artifacts map[string]*database.ExecutionArtifact,
	stepsByID map[uuid.UUID]*database.ExecutionStep,
	stepsByIndex map[int]*database.ExecutionStep,
) (TimelineFrame, error) {
	payload := map[string]any{}
	if timelineArtifact.Payload != nil {
		payload = map[string]any(timelineArtifact.Payload)
	}

	stepIndex := typeconv.ToInt(payload["stepIndex"])
	nodeID := typeconv.ToString(payload["nodeId"])
	stepType := typeconv.ToString(payload["stepType"])
	success := typeconv.ToBool(payload["success"])
	duration := typeconv.ToInt(payload["durationMs"])
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
	retryAttempt := typeconv.ToInt(payload["retryAttempt"])
	if retryAttempt == 0 {
		retryAttempt = typeconv.ToInt(payload["retry_attempt"])
	}
	retryMaxAttempts := typeconv.ToInt(payload["retryMaxAttempts"])
	if retryMaxAttempts == 0 {
		retryMaxAttempts = typeconv.ToInt(payload["retry_max_attempts"])
	}
	retryConfigured := typeconv.ToInt(payload["retryConfigured"])
	if retryConfigured == 0 {
		retryConfigured = typeconv.ToInt(payload["retry_configured"])
	}
	retryDelayMs := typeconv.ToInt(payload["retryDelayMs"])
	if retryDelayMs == 0 {
		retryDelayMs = typeconv.ToInt(payload["retry_delay_ms"])
	}
	retryBackoff := typeconv.ToFloat(payload["retryBackoffFactor"])
	if retryBackoff == 0 {
		retryBackoff = typeconv.ToFloat(payload["retry_backoff_factor"])
	}
	retryHistory := typeconv.ToRetryHistory(payload["retryHistory"])
	if len(retryHistory) == 0 {
		retryHistory = typeconv.ToRetryHistory(payload["retry_history"])
	}
	domSnapshotPreview := typeconv.ToString(payload["domSnapshotPreview"])
	if domSnapshotPreview == "" {
		domSnapshotPreview = typeconv.ToString(payload["dom_snapshot_preview"])
	}
	domSnapshotArtifactID := typeconv.ToString(payload["domSnapshotArtifactId"])
	if domSnapshotArtifactID == "" {
		domSnapshotArtifactID = typeconv.ToString(payload["dom_snapshot_artifact_id"])
	}

	startedAt := typeconv.ToTimePtr(payload["startedAt"])
	completedAt := typeconv.ToTimePtr(payload["completedAt"])

	var status string
	if success {
		status = "completed"
	} else if errorMsg != "" {
		status = "failed"
	}

	if stepID := typeconv.ToString(payload["executionStepId"]); stepID != "" {
		if parsedID, err := uuid.Parse(stepID); err == nil {
			if step := stepsByID[parsedID]; step != nil {
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
		}
	} else if step := stepsByIndex[stepIndex]; step != nil {
		if status == "" && step.Status != "" {
			status = step.Status
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

	highlightRegions := typeconv.ToHighlightRegions(payload["highlightRegions"])
	maskRegions := typeconv.ToMaskRegions(payload["maskRegions"])
	focusedElement := typeconv.ToElementFocus(payload["focusedElement"])
	elementBoundingBox := typeconv.ToBoundingBox(payload["elementBoundingBox"])
	clickPosition := typeconv.ToPoint(payload["clickPosition"])

	var screenshot *TimelineScreenshot
	if screenshotID := typeconv.ToString(payload["screenshotArtifactId"]); screenshotID != "" {
		if artifact := artifacts[screenshotID]; artifact != nil {
			screenshot = typeconv.ToTimelineScreenshot(artifact)
		}
	}

	artifactRefs := make([]TimelineArtifact, 0)
	for _, id := range typeconv.ToStringSlice(payload["artifactIds"]) {
		if artifact := artifacts[id]; artifact != nil {
			artifactRefs = append(artifactRefs, typeconv.ToTimelineArtifact(artifact))
		}
	}

	var domSnapshot *TimelineArtifact
	if domSnapshotArtifactID != "" {
		if artifact := artifacts[domSnapshotArtifactID]; artifact != nil {
			artifactCopy := typeconv.ToTimelineArtifact(artifact)
			domSnapshot = &artifactCopy
		}
	}

	frame := TimelineFrame{
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

	return frame, nil
}
