package export

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/internal/typeconv"
)

// ExecutionTimeline represents the replay-friendly view of an execution.
type ExecutionTimeline struct {
	ExecutionID uuid.UUID       `json:"execution_id"`
	WorkflowID  uuid.UUID       `json:"workflow_id"`
	Status      string          `json:"status"`
	Progress    int             `json:"progress"`
	StartedAt   time.Time       `json:"started_at"`
	CompletedAt *time.Time      `json:"completed_at,omitempty"`
	Frames      []TimelineFrame `json:"frames"`
	Logs        []TimelineLog   `json:"logs"`
}

// TimelineFrame captures a single step in the execution timeline.
type TimelineFrame struct {
	StepIndex            int                             `json:"step_index"`
	NodeID               string                          `json:"node_id"`
	StepType             string                          `json:"step_type"`
	Status               string                          `json:"status"`
	Success              bool                            `json:"success"`
	DurationMs           int                             `json:"duration_ms,omitempty"`
	TotalDurationMs      int                             `json:"total_duration_ms,omitempty"`
	Progress             int                             `json:"progress,omitempty"`
	StartedAt            *time.Time                      `json:"started_at,omitempty"`
	CompletedAt          *time.Time                      `json:"completed_at,omitempty"`
	FinalURL             string                          `json:"final_url,omitempty"`
	Error                string                          `json:"error,omitempty"`
	ConsoleLogCount      int                             `json:"console_log_count,omitempty"`
	NetworkEventCount    int                             `json:"network_event_count,omitempty"`
	ExtractedDataPreview any                             `json:"extracted_data_preview,omitempty"`
	HighlightRegions     []autocontracts.HighlightRegion `json:"highlight_regions,omitempty"`
	MaskRegions          []autocontracts.MaskRegion      `json:"mask_regions,omitempty"`
	FocusedElement       *autocontracts.ElementFocus     `json:"focused_element,omitempty"`
	ElementBoundingBox   *autocontracts.BoundingBox      `json:"element_bounding_box,omitempty"`
	ClickPosition        *autocontracts.Point            `json:"click_position,omitempty"`
	CursorTrail          []autocontracts.Point           `json:"cursor_trail,omitempty"`
	ZoomFactor           float64                         `json:"zoom_factor,omitempty"`
	Screenshot           *TimelineScreenshot             `json:"screenshot,omitempty"`
	Artifacts            []TimelineArtifact              `json:"artifacts,omitempty"`
	Assertion            *autocontracts.AssertionOutcome `json:"assertion,omitempty"`
	RetryAttempt         int                             `json:"retry_attempt,omitempty"`
	RetryMaxAttempts     int                             `json:"retry_max_attempts,omitempty"`
	RetryConfigured      int                             `json:"retry_configured,omitempty"`
	RetryDelayMs         int                             `json:"retry_delay_ms,omitempty"`
	RetryBackoffFactor   float64                         `json:"retry_backoff_factor,omitempty"`
	RetryHistory         []typeconv.RetryHistoryEntry    `json:"retry_history,omitempty"`
	DomSnapshotPreview   string                          `json:"dom_snapshot_preview,omitempty"`
	DomSnapshot          *typeconv.TimelineArtifact      `json:"dom_snapshot,omitempty"`
}

// Type aliases for backward compatibility with existing code.
type (
	RetryHistoryEntry  = typeconv.RetryHistoryEntry
	TimelineScreenshot = typeconv.TimelineScreenshot
	TimelineArtifact   = typeconv.TimelineArtifact
)

// TimelineLog captures execution log output for replay consumers.
type TimelineLog struct {
	ID        string    `json:"id"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	StepName  string    `json:"step_name,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

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
