package services

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/browserless/runtime"
	"github.com/vrooli/browser-automation-studio/database"
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
	StepIndex            int                       `json:"step_index"`
	NodeID               string                    `json:"node_id"`
	StepType             string                    `json:"step_type"`
	Status               string                    `json:"status"`
	Success              bool                      `json:"success"`
	DurationMs           int                       `json:"duration_ms,omitempty"`
	TotalDurationMs      int                       `json:"total_duration_ms,omitempty"`
	Progress             int                       `json:"progress,omitempty"`
	StartedAt            *time.Time                `json:"started_at,omitempty"`
	CompletedAt          *time.Time                `json:"completed_at,omitempty"`
	FinalURL             string                    `json:"final_url,omitempty"`
	Error                string                    `json:"error,omitempty"`
	ConsoleLogCount      int                       `json:"console_log_count,omitempty"`
	NetworkEventCount    int                       `json:"network_event_count,omitempty"`
	ExtractedDataPreview any                       `json:"extracted_data_preview,omitempty"`
	HighlightRegions     []runtime.HighlightRegion `json:"highlight_regions,omitempty"`
	MaskRegions          []runtime.MaskRegion      `json:"mask_regions,omitempty"`
	FocusedElement       *runtime.ElementFocus     `json:"focused_element,omitempty"`
	ElementBoundingBox   *runtime.BoundingBox      `json:"element_bounding_box,omitempty"`
	ClickPosition        *runtime.Point            `json:"click_position,omitempty"`
	CursorTrail          []runtime.Point           `json:"cursor_trail,omitempty"`
	ZoomFactor           float64                   `json:"zoom_factor,omitempty"`
	Screenshot           *TimelineScreenshot       `json:"screenshot,omitempty"`
	Artifacts            []TimelineArtifact        `json:"artifacts,omitempty"`
	Assertion            *runtime.AssertionResult  `json:"assertion,omitempty"`
	RetryAttempt         int                       `json:"retry_attempt,omitempty"`
	RetryMaxAttempts     int                       `json:"retry_max_attempts,omitempty"`
	RetryConfigured      int                       `json:"retry_configured,omitempty"`
	RetryDelayMs         int                       `json:"retry_delay_ms,omitempty"`
	RetryBackoffFactor   float64                   `json:"retry_backoff_factor,omitempty"`
	RetryHistory         []RetryHistoryEntry       `json:"retry_history,omitempty"`
	DomSnapshotPreview   string                    `json:"dom_snapshot_preview,omitempty"`
	DomSnapshot          *TimelineArtifact         `json:"dom_snapshot,omitempty"`
}

// RetryHistoryEntry captures the outcome of a single retry attempt for a step.
type RetryHistoryEntry struct {
	Attempt        int    `json:"attempt"`
	Success        bool   `json:"success"`
	DurationMs     int    `json:"duration_ms,omitempty"`
	CallDurationMs int    `json:"call_duration_ms,omitempty"`
	Error          string `json:"error,omitempty"`
}

// TimelineScreenshot describes screenshot metadata associated with a frame.
type TimelineScreenshot struct {
	ArtifactID   string `json:"artifact_id"`
	URL          string `json:"url,omitempty"`
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
	Width        int    `json:"width,omitempty"`
	Height       int    `json:"height,omitempty"`
	ContentType  string `json:"content_type,omitempty"`
	SizeBytes    *int64 `json:"size_bytes,omitempty"`
}

// TimelineArtifact exposes related artifacts such as console or network logs.
type TimelineArtifact struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"`
	Label        string         `json:"label,omitempty"`
	StorageURL   string         `json:"storage_url,omitempty"`
	ThumbnailURL string         `json:"thumbnail_url,omitempty"`
	ContentType  string         `json:"content_type,omitempty"`
	SizeBytes    *int64         `json:"size_bytes,omitempty"`
	StepIndex    *int           `json:"step_index,omitempty"`
	Payload      map[string]any `json:"payload,omitempty"`
}

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

	stepIndex := toInt(payload["stepIndex"])
	nodeID := toString(payload["nodeId"])
	stepType := toString(payload["stepType"])
	success := toBool(payload["success"])
	duration := toInt(payload["durationMs"])
	progress := toInt(payload["progress"])
	finalURL := toString(payload["finalUrl"])
	errorMsg := toString(payload["error"])
	consoleCount := toInt(payload["consoleLogCount"])
	networkCount := toInt(payload["networkEventCount"])
	zoomFactor := toFloat(payload["zoomFactor"])
	extractedPreview := payload["extractedDataPreview"]
	cursorTrail := toPointSlice(payload["cursorTrail"])
	assertion := toAssertionResult(payload["assertion"])
	totalDuration := toInt(payload["totalDurationMs"])
	retryAttempt := toInt(payload["retryAttempt"])
	if retryAttempt == 0 {
		retryAttempt = toInt(payload["retry_attempt"])
	}
	retryMaxAttempts := toInt(payload["retryMaxAttempts"])
	if retryMaxAttempts == 0 {
		retryMaxAttempts = toInt(payload["retry_max_attempts"])
	}
	retryConfigured := toInt(payload["retryConfigured"])
	if retryConfigured == 0 {
		retryConfigured = toInt(payload["retry_configured"])
	}
	retryDelayMs := toInt(payload["retryDelayMs"])
	if retryDelayMs == 0 {
		retryDelayMs = toInt(payload["retry_delay_ms"])
	}
	retryBackoff := toFloat(payload["retryBackoffFactor"])
	if retryBackoff == 0 {
		retryBackoff = toFloat(payload["retry_backoff_factor"])
	}
	retryHistory := toRetryHistory(payload["retryHistory"])
	if len(retryHistory) == 0 {
		retryHistory = toRetryHistory(payload["retry_history"])
	}
	domSnapshotPreview := toString(payload["domSnapshotPreview"])
	if domSnapshotPreview == "" {
		domSnapshotPreview = toString(payload["dom_snapshot_preview"])
	}
	domSnapshotArtifactID := toString(payload["domSnapshotArtifactId"])
	if domSnapshotArtifactID == "" {
		domSnapshotArtifactID = toString(payload["dom_snapshot_artifact_id"])
	}

	startedAt := toTimePtr(payload["startedAt"])
	completedAt := toTimePtr(payload["completedAt"])

	var status string
	if success {
		status = "completed"
	} else if errorMsg != "" {
		status = "failed"
	}

	if stepID := toString(payload["executionStepId"]); stepID != "" {
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

	highlightRegions := toHighlightRegions(payload["highlightRegions"])
	maskRegions := toMaskRegions(payload["maskRegions"])
	focusedElement := toElementFocus(payload["focusedElement"])
	elementBoundingBox := toBoundingBox(payload["elementBoundingBox"])
	clickPosition := toPoint(payload["clickPosition"])

	var screenshot *TimelineScreenshot
	if screenshotID := toString(payload["screenshotArtifactId"]); screenshotID != "" {
		if artifact := artifacts[screenshotID]; artifact != nil {
			screenshot = toTimelineScreenshot(artifact)
		}
	}

	artifactRefs := make([]TimelineArtifact, 0)
	for _, id := range toStringSlice(payload["artifactIds"]) {
		if artifact := artifacts[id]; artifact != nil {
			artifactRefs = append(artifactRefs, toTimelineArtifact(artifact))
		}
	}

	var domSnapshot *TimelineArtifact
	if domSnapshotArtifactID != "" {
		if artifact := artifacts[domSnapshotArtifactID]; artifact != nil {
			artifactCopy := toTimelineArtifact(artifact)
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

func toString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	case []byte:
		return string(v)
	default:
		return ""
	}
}

func toInt(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return int(i)
		}
	case string:
		if parsed, err := strconv.Atoi(v); err == nil {
			return parsed
		}
	}
	return 0
}

func toBool(value any) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		b, err := strconv.ParseBool(v)
		if err == nil {
			return b
		}
	}
	return false
}

func toFloat(value any) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	case json.Number:
		f, _ := v.Float64()
		return f
	case string:
		if parsed, err := strconv.ParseFloat(v, 64); err == nil {
			return parsed
		}
	}
	return 0
}

func toTimePtr(value any) *time.Time {
	switch v := value.(type) {
	case time.Time:
		return &v
	case *time.Time:
		return v
	case string:
		if v == "" {
			return nil
		}
		if ts, err := time.Parse(time.RFC3339Nano, v); err == nil {
			return &ts
		}
		if ts, err := time.Parse(time.RFC3339, v); err == nil {
			return &ts
		}
	}
	return nil
}

func toAssertionResult(value any) *runtime.AssertionResult {
	switch v := value.(type) {
	case *runtime.AssertionResult:
		return v
	case runtime.AssertionResult:
		result := v
		return &result
	case map[string]any:
		result := &runtime.AssertionResult{
			Mode:          toString(v["mode"]),
			Selector:      toString(v["selector"]),
			Message:       toString(v["message"]),
			Success:       toBool(v["success"]),
			Negated:       toBool(v["negated"]),
			CaseSensitive: toBool(v["caseSensitive"]),
		}
		if expected, ok := v["expected"]; ok {
			result.Expected = expected
		}
		if actual, ok := v["actual"]; ok {
			result.Actual = actual
		}
		return result
	default:
		return nil
	}
}

func toHighlightRegions(value any) []runtime.HighlightRegion {
	regions := make([]runtime.HighlightRegion, 0)
	switch v := value.(type) {
	case []runtime.HighlightRegion:
		return v
	case []map[string]any:
		for _, item := range v {
			if region := toHighlightRegion(item); region != nil {
				regions = append(regions, *region)
			}
		}
	case []any:
		for _, item := range v {
			if region := toHighlightRegion(item); region != nil {
				regions = append(regions, *region)
			}
		}
	}
	return regions
}

func toHighlightRegion(value any) *runtime.HighlightRegion {
	m, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	region := runtime.HighlightRegion{
		Selector: toString(m["selector"]),
		Padding:  toInt(m["padding"]),
		Color:    toString(m["color"]),
	}
	if bbox := toBoundingBox(m["boundingBox"]); bbox != nil {
		region.BoundingBox = bbox
	}
	if region.Selector == "" && region.BoundingBox == nil {
		return nil
	}
	return &region
}

func toMaskRegions(value any) []runtime.MaskRegion {
	regions := make([]runtime.MaskRegion, 0)
	switch v := value.(type) {
	case []runtime.MaskRegion:
		return v
	case []map[string]any:
		for _, item := range v {
			if region := toMaskRegion(item); region != nil {
				regions = append(regions, *region)
			}
		}
	case []any:
		for _, item := range v {
			if region := toMaskRegion(item); region != nil {
				regions = append(regions, *region)
			}
		}
	}
	return regions
}

func toMaskRegion(value any) *runtime.MaskRegion {
	m, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	region := runtime.MaskRegion{
		Selector: toString(m["selector"]),
		Opacity:  toFloat(m["opacity"]),
	}
	if bbox := toBoundingBox(m["boundingBox"]); bbox != nil {
		region.BoundingBox = bbox
	}
	if region.Selector == "" && region.BoundingBox == nil {
		return nil
	}
	return &region
}

func toElementFocus(value any) *runtime.ElementFocus {
	m, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	focus := runtime.ElementFocus{
		Selector: toString(m["selector"]),
	}
	if bbox := toBoundingBox(m["boundingBox"]); bbox != nil {
		focus.BoundingBox = bbox
	}
	if focus.Selector == "" && focus.BoundingBox == nil {
		return nil
	}
	return &focus
}

func toBoundingBox(value any) *runtime.BoundingBox {
	m, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	if len(m) == 0 {
		return nil
	}
	return &runtime.BoundingBox{
		X:      toFloat(m["x"]),
		Y:      toFloat(m["y"]),
		Width:  toFloat(m["width"]),
		Height: toFloat(m["height"]),
	}
}

func toPoint(value any) *runtime.Point {
	m, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	return &runtime.Point{
		X: toFloat(m["x"]),
		Y: toFloat(m["y"]),
	}
}

func toPointSlice(value any) []runtime.Point {
	points := make([]runtime.Point, 0)
	switch v := value.(type) {
	case []runtime.Point:
		return v
	case []*runtime.Point:
		for _, item := range v {
			if item != nil {
				points = append(points, *item)
			}
		}
	case []map[string]any:
		for _, item := range v {
			if pt := toPoint(item); pt != nil {
				points = append(points, *pt)
			}
		}
	case []any:
		for _, item := range v {
			if pt := toPoint(item); pt != nil {
				points = append(points, *pt)
			}
		}
	case map[string]any:
		if pt := toPoint(v); pt != nil {
			points = append(points, *pt)
		}
	}
	return points
}

func toStringSlice(value any) []string {
	result := make([]string, 0)
	switch v := value.(type) {
	case []string:
		return v
	case []any:
		for _, item := range v {
			if str := toString(item); str != "" {
				result = append(result, str)
			}
		}
	case string:
		if v != "" {
			result = append(result, v)
		}
	}
	return result
}

func toTimelineScreenshot(artifact *database.ExecutionArtifact) *TimelineScreenshot {
	if artifact == nil {
		return nil
	}
	shot := &TimelineScreenshot{
		ArtifactID:   artifact.ID.String(),
		URL:          artifact.StorageURL,
		ThumbnailURL: artifact.ThumbnailURL,
		ContentType:  artifact.ContentType,
		SizeBytes:    artifact.SizeBytes,
	}
	if artifact.Payload != nil {
		payload := map[string]any(artifact.Payload)
		shot.Width = toInt(payload["width"])
		shot.Height = toInt(payload["height"])
	}
	return shot
}

func toTimelineArtifact(artifact *database.ExecutionArtifact) TimelineArtifact {
	payload := map[string]any{}
	if artifact.Payload != nil {
		payload = map[string]any(artifact.Payload)
	}
	return TimelineArtifact{
		ID:           artifact.ID.String(),
		Type:         artifact.ArtifactType,
		Label:        artifact.Label,
		StorageURL:   artifact.StorageURL,
		ThumbnailURL: artifact.ThumbnailURL,
		ContentType:  artifact.ContentType,
		SizeBytes:    artifact.SizeBytes,
		StepIndex:    artifact.StepIndex,
		Payload:      payload,
	}
}

func toRetryHistory(value any) []RetryHistoryEntry {
	entries := make([]RetryHistoryEntry, 0)
	switch v := value.(type) {
	case []RetryHistoryEntry:
		return v
	case []map[string]any:
		for _, item := range v {
			if entry := toRetryHistoryEntry(item); entry != nil {
				entries = append(entries, *entry)
			}
		}
	case []any:
		for _, item := range v {
			if entry := toRetryHistoryEntry(item); entry != nil {
				entries = append(entries, *entry)
			}
		}
	}
	return entries
}

func toRetryHistoryEntry(value any) *RetryHistoryEntry {
	m, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	entry := RetryHistoryEntry{
		Attempt:        toInt(m["attempt"]),
		Success:        toBool(m["success"]),
		DurationMs:     toInt(m["durationMs"]),
		CallDurationMs: toInt(m["callDurationMs"]),
		Error:          toString(m["error"]),
	}
	if entry.DurationMs == 0 {
		entry.DurationMs = toInt(m["duration_ms"])
	}
	if entry.CallDurationMs == 0 {
		entry.CallDurationMs = toInt(m["call_duration_ms"])
	}
	if entry.Error == "" {
		entry.Error = toString(m["error_message"])
	}
	if entry.Attempt == 0 && entry.DurationMs == 0 && entry.CallDurationMs == 0 && entry.Error == "" && entry.Success {
		// treat empty entry as invalid unless it represents a successful attempt with no data
		entry.Success = true
	}
	if entry.Attempt == 0 && entry.Error == "" && entry.DurationMs == 0 && entry.CallDurationMs == 0 {
		return nil
	}
	return &entry
}
