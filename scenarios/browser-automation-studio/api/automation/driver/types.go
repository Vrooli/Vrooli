package driver

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
)

// Viewport defines browser viewport dimensions.
type Viewport struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// FrameStreamingConfig configures live frame streaming for recording mode.
type FrameStreamingConfig struct {
	CallbackURL string `json:"callback_url"`
	Quality     int    `json:"quality"`
	FPS         int    `json:"fps"`
	Scale       string `json:"scale"`
}

// CapabilityRequest specifies required browser capabilities for execution.
type CapabilityRequest struct {
	Tabs       bool `json:"tabs,omitempty"`
	Iframes    bool `json:"iframes,omitempty"`
	Uploads    bool `json:"uploads,omitempty"`
	Downloads  bool `json:"downloads,omitempty"`
	HAR        bool `json:"har,omitempty"`
	Video      bool `json:"video,omitempty"`
	Tracing    bool `json:"tracing,omitempty"`
	ViewportW  int  `json:"viewport_width,omitempty"`
	ViewportH  int  `json:"viewport_height,omitempty"`
	MaxTimeout int  `json:"max_timeout_ms,omitempty"`
}

// CreateSessionRequest is the unified request to create a browser session.
// Used by both recording and execution modes.
type CreateSessionRequest struct {
	// Common fields
	ExecutionID string            `json:"execution_id"`
	WorkflowID  string            `json:"workflow_id"`
	Viewport    Viewport          `json:"viewport"`
	ReuseMode   string            `json:"reuse_mode"`
	Labels      map[string]string `json:"labels,omitempty"`

	// Recording mode - storage state as raw JSON
	StorageState json.RawMessage `json:"storage_state,omitempty"`

	// Recording mode - frame streaming configuration
	FrameStreaming *FrameStreamingConfig `json:"frame_streaming,omitempty"`

	// Execution mode - base URL for relative navigation
	BaseURL string `json:"base_url,omitempty"`

	// Execution mode - required capabilities
	RequiredCapabilities *CapabilityRequest `json:"required_capabilities,omitempty"`
}

// CreateSessionRequestFromUUID creates a session request using uuid.UUID types.
// This is a convenience constructor for execution mode which uses UUIDs internally.
func CreateSessionRequestFromUUID(executionID, workflowID uuid.UUID) *CreateSessionRequest {
	return &CreateSessionRequest{
		ExecutionID: executionID.String(),
		WorkflowID:  workflowID.String(),
	}
}

// CreateSessionResponse is the response from creating a session.
type CreateSessionResponse struct {
	SessionID string `json:"session_id"`
}

// StartRecordingRequest is the request to start recording user actions.
type StartRecordingRequest struct {
	CallbackURL      string `json:"callback_url"`
	FrameCallbackURL string `json:"frame_callback_url"`
	FrameQuality     int    `json:"frame_quality"`
	FrameFPS         int    `json:"frame_fps"`
}

// StartRecordingResponse is the response from starting recording.
type StartRecordingResponse struct {
	SessionID   string `json:"session_id"`
	IsRecording bool   `json:"is_recording"`
	StartedAt   string `json:"started_at"`
}

// StopRecordingResponse is the response from stopping recording.
type StopRecordingResponse struct {
	SessionID   string `json:"session_id"`
	IsRecording bool   `json:"is_recording"`
	ActionCount int    `json:"action_count"`
	StoppedAt   string `json:"stopped_at"`
}

// RecordingStatusResponse is the response from getting recording status.
type RecordingStatusResponse struct {
	SessionID   string `json:"session_id"`
	IsRecording bool   `json:"is_recording"`
	ActionCount int    `json:"action_count"`
	StartedAt   string `json:"started_at,omitempty"`
	FrameCount  int    `json:"frame_count"`
}

// RecordedAction represents an action captured during recording.
// This mirrors the JSON structure sent from playwright-driver.
type RecordedAction struct {
	ID          string                 `json:"id"`
	SessionID   string                 `json:"sessionId"`
	SequenceNum int                    `json:"sequenceNum"`
	Timestamp   string                 `json:"timestamp"`
	DurationMs  int                    `json:"durationMs,omitempty"`
	ActionType  string                 `json:"actionType"`
	Confidence  float64                `json:"confidence"`
	Selector    *SelectorSet           `json:"selector,omitempty"`
	ElementMeta *ElementMeta           `json:"elementMeta,omitempty"`
	BoundingBox *contracts.BoundingBox `json:"boundingBox,omitempty"`
	Payload     map[string]interface{} `json:"payload,omitempty"`
	URL         string                 `json:"url"`
	FrameID     string                 `json:"frameId,omitempty"`
	CursorPos   *contracts.Point       `json:"cursorPos,omitempty"`
}

// SelectorSet contains multiple selector strategies for resilience.
type SelectorSet struct {
	Primary    string              `json:"primary"`
	Candidates []SelectorCandidate `json:"candidates"`
}

// SelectorCandidate is a single selector with metadata.
type SelectorCandidate struct {
	Type        string  `json:"type"`
	Value       string  `json:"value"`
	Confidence  float64 `json:"confidence"`
	Specificity int     `json:"specificity"`
}

// ElementMeta captures information about the target element.
type ElementMeta struct {
	TagName    string            `json:"tagName"`
	ID         string            `json:"id,omitempty"`
	ClassName  string            `json:"className,omitempty"`
	InnerText  string            `json:"innerText,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`
	IsVisible  bool              `json:"isVisible"`
	IsEnabled  bool              `json:"isEnabled"`
	Role       string            `json:"role,omitempty"`
	AriaLabel  string            `json:"ariaLabel,omitempty"`
}

// GetActionsResponse is the response from getting recorded actions.
type GetActionsResponse struct {
	SessionID   string           `json:"session_id"`
	IsRecording bool             `json:"is_recording"`
	Actions     []RecordedAction `json:"actions"`
}

// NavigateRequest is the request to navigate the session.
type NavigateRequest struct {
	URL       string `json:"url"`
	WaitUntil string `json:"wait_until,omitempty"`
	TimeoutMs int    `json:"timeout_ms,omitempty"`
	Capture   bool   `json:"capture,omitempty"`
}

// NavigateResponse is the response from navigation.
type NavigateResponse struct {
	URL        string `json:"url"`
	Title      string `json:"title"`
	StatusCode int    `json:"status_code,omitempty"`
	Screenshot string `json:"screenshot,omitempty"`
}

// UpdateViewportRequest is the request to update viewport.
type UpdateViewportRequest struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// UpdateViewportResponse is the response from updating viewport.
type UpdateViewportResponse struct {
	SessionID string `json:"session_id"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
}

// ValidateSelectorRequest is the request to validate a selector.
type ValidateSelectorRequest struct {
	Selector string `json:"selector"`
}

// ValidateSelectorResponse is the response from selector validation.
type ValidateSelectorResponse struct {
	Valid      bool   `json:"valid"`
	MatchCount int    `json:"match_count"`
	Selector   string `json:"selector"`
	Error      string `json:"error,omitempty"`
}

// ReplayPreviewRequest is the request to replay actions.
type ReplayPreviewRequest struct {
	Actions       []RecordedAction `json:"actions"`
	Limit         *int             `json:"limit,omitempty"`
	StopOnFailure *bool            `json:"stop_on_failure,omitempty"`
	ActionTimeout *int             `json:"action_timeout,omitempty"`
}

// ActionResult represents the result of a single replayed action.
type ActionResult struct {
	Index      int    `json:"index"`
	ActionType string `json:"action_type"`
	Success    bool   `json:"success"`
	Error      string `json:"error,omitempty"`
	DurationMs int    `json:"duration_ms"`
}

// ReplayPreviewResponse is the response from replay preview.
type ReplayPreviewResponse struct {
	Success         bool           `json:"success"`
	PassedActions   int            `json:"passed_actions"`
	FailedActions   int            `json:"failed_actions"`
	TotalDurationMs int            `json:"total_duration_ms"`
	Results         []ActionResult `json:"results"`
}

// UpdateStreamSettingsRequest is the request to update stream settings.
type UpdateStreamSettingsRequest struct {
	Quality  *int   `json:"quality,omitempty"`
	FPS      *int   `json:"fps,omitempty"`
	Scale    string `json:"scale,omitempty"`
	PerfMode *bool  `json:"perfMode,omitempty"`
}

// UpdateStreamSettingsResponse is the response from updating stream settings.
type UpdateStreamSettingsResponse struct {
	SessionID    string `json:"session_id"`
	Quality      int    `json:"quality"`
	FPS          int    `json:"fps"`
	CurrentFPS   int    `json:"current_fps"`
	Scale        string `json:"scale"`
	IsStreaming  bool   `json:"is_streaming"`
	Updated      bool   `json:"updated"`
	ScaleWarning string `json:"scale_warning,omitempty"`
	PerfMode     bool   `json:"perf_mode"`
}

// CaptureScreenshotRequest is the request to capture a screenshot.
type CaptureScreenshotRequest struct {
	Format  string `json:"format,omitempty"`
	Quality int    `json:"quality,omitempty"`
}

// CaptureScreenshotResponse is the response from capturing a screenshot.
type CaptureScreenshotResponse struct {
	Data       string `json:"data"`
	MediaType  string `json:"media_type"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	CapturedAt string `json:"captured_at"`
}

// GetFrameResponse is the response from getting a frame.
type GetFrameResponse struct {
	Data        string `json:"data"`
	MediaType   string `json:"media_type"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	CapturedAt  string `json:"captured_at"`
	ContentHash string `json:"content_hash"`
	PageTitle   string `json:"page_title,omitempty"`
	PageURL     string `json:"page_url,omitempty"`
}

// RunInstructionRequest wraps an instruction for execution.
type RunInstructionRequest struct {
	Instruction contracts.CompiledInstruction `json:"instruction"`
}

// RunInstructionsRequest wraps multiple instructions (used for simple ops like navigate).
type RunInstructionsRequest struct {
	Instructions []map[string]interface{} `json:"instructions"`
}

// StepOutcomeResponse extends StepOutcome with driver-specific fields for JSON decoding.
type StepOutcomeResponse struct {
	contracts.StepOutcome

	ScreenshotBase64    string `json:"screenshot_base64,omitempty"`
	ScreenshotMediaType string `json:"screenshot_media_type,omitempty"`
	ScreenshotWidth     int    `json:"screenshot_width,omitempty"`
	ScreenshotHeight    int    `json:"screenshot_height,omitempty"`

	DOMHTML    string `json:"dom_html,omitempty"`
	DOMPreview string `json:"dom_preview,omitempty"`

	VideoPath string `json:"video_path,omitempty"`
	TracePath string `json:"trace_path,omitempty"`

	// ElementSnapshot contains target element metadata when captured by the driver.
	// The driver populates this when the action targets an element (click, input, etc.).
	// This enables execution debugging with the same element info available in recordings.
	// Uses contracts.ElementMeta (proto type) for type compatibility with StepOutcome.
	ElementSnapshot *contracts.ElementMeta `json:"element_snapshot,omitempty"`
}

// CloseSessionResponse is returned when a session is closed.
type CloseSessionResponse struct {
	Success    bool     `json:"success,omitempty"`
	VideoPaths []string `json:"video_paths,omitempty"`
}
