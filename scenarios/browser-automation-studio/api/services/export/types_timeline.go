package export

import (
	"time"

	"github.com/google/uuid"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
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
	StepIndex            int                              `json:"step_index"`
	NodeID               string                           `json:"node_id"`
	StepType             string                           `json:"step_type"`
	Status               string                           `json:"status"`
	Success              bool                             `json:"success"`
	DurationMs           int                              `json:"duration_ms,omitempty"`
	TotalDurationMs      int                              `json:"total_duration_ms,omitempty"`
	Progress             int                              `json:"progress,omitempty"`
	StartedAt            *time.Time                       `json:"started_at,omitempty"`
	CompletedAt          *time.Time                       `json:"completed_at,omitempty"`
	FinalURL             string                           `json:"final_url,omitempty"`
	Error                string                           `json:"error,omitempty"`
	ConsoleLogCount      int                              `json:"console_log_count,omitempty"`
	NetworkEventCount    int                              `json:"network_event_count,omitempty"`
	ExtractedDataPreview any                              `json:"extracted_data_preview,omitempty"`
	HighlightRegions     []*autocontracts.HighlightRegion `json:"highlight_regions,omitempty"`
	MaskRegions          []*autocontracts.MaskRegion      `json:"mask_regions,omitempty"`
	FocusedElement       *autocontracts.ElementFocus      `json:"focused_element,omitempty"`
	ElementBoundingBox   *autocontracts.BoundingBox       `json:"element_bounding_box,omitempty"`
	ClickPosition        *autocontracts.Point             `json:"click_position,omitempty"`
	CursorTrail          []*autocontracts.Point           `json:"cursor_trail,omitempty"`
	ZoomFactor           float64                          `json:"zoom_factor,omitempty"`
	Screenshot           *TimelineScreenshot              `json:"screenshot,omitempty"`
	Artifacts            []TimelineArtifact               `json:"artifacts,omitempty"`
	Assertion            *autocontracts.AssertionOutcome  `json:"assertion,omitempty"`
	Condition            *autocontracts.ConditionOutcome  `json:"condition,omitempty"`
	RetryAttempt         int                              `json:"retry_attempt,omitempty"`
	RetryMaxAttempts     int                              `json:"retry_max_attempts,omitempty"`
	RetryConfigured      int                              `json:"retry_configured,omitempty"`
	RetryDelayMs         int                              `json:"retry_delay_ms,omitempty"`
	RetryBackoffFactor   float64                          `json:"retry_backoff_factor,omitempty"`
	RetryHistory         []RetryHistoryEntry              `json:"retry_history,omitempty"`
}

// RetryHistoryEntry captures one step retry attempt in the exported timeline.
type RetryHistoryEntry struct {
	Attempt        int    `json:"attempt"`
	Success        bool   `json:"success"`
	DurationMs     int    `json:"duration_ms,omitempty"`
	CallDurationMs int    `json:"call_duration_ms,omitempty"`
	Error          string `json:"error,omitempty"`
}

// TimelineScreenshot describes screenshot metadata attached to an exported frame.
type TimelineScreenshot struct {
	ArtifactID   string `json:"artifact_id"`
	URL          string `json:"url,omitempty"`
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
	Width        int    `json:"width,omitempty"`
	Height       int    `json:"height,omitempty"`
	ContentType  string `json:"content_type,omitempty"`
	SizeBytes    *int64 `json:"size_bytes,omitempty"`
}

// TimelineArtifact describes a non-screenshot artifact attached to a frame.
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
