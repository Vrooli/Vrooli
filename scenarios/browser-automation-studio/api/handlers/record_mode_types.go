package handlers

import (
	"github.com/google/uuid"
	livecapture "github.com/vrooli/browser-automation-studio/services/live-capture"
)

// =============================================================================
// Recording Session Types
// =============================================================================

// CreateRecordingSessionRequest is the request body for creating a browser session for recording.
type CreateRecordingSessionRequest struct {
	// Viewport dimensions (optional, defaults to 1280x720)
	ViewportWidth  int `json:"viewport_width,omitempty"`
	ViewportHeight int `json:"viewport_height,omitempty"`
	// Initial URL to navigate to (optional)
	InitialURL string `json:"initial_url,omitempty"`
	// Optional persisted session profile to load cookies/storage from
	SessionProfileID string `json:"session_profile_id,omitempty"`
	// Stream quality settings (optional)
	// Quality: JPEG quality 0-100 (default 55)
	StreamQuality *int `json:"stream_quality,omitempty"`
	// FPS: Frames per second 1-60 (default 6)
	StreamFPS *int `json:"stream_fps,omitempty"`
	// Scale: "css" for 1x scale, "device" for device pixel ratio (default "css")
	StreamScale string `json:"stream_scale,omitempty"`
}

// CreateRecordingSessionResponse is the response after creating a recording session.
type CreateRecordingSessionResponse struct {
	SessionID          string `json:"session_id"`
	CreatedAt          string `json:"created_at"`
	SessionProfileID   string `json:"session_profile_id,omitempty"`
	SessionProfileName string `json:"session_profile_name,omitempty"`
	LastUsedAt         string `json:"last_used_at,omitempty"`
}

// CloseRecordingSessionRequest is the request body for closing a recording session.
type CloseRecordingSessionRequest struct {
	SessionID string `json:"session_id"`
}

// =============================================================================
// Recording Control Types
// =============================================================================

// StartRecordingRequest is the request body for starting a live recording.
type StartRecordingRequest struct {
	SessionID   string `json:"session_id"`
	CallbackURL string `json:"callback_url,omitempty"`
}

// StartRecordingResponse is the response after starting recording.
type StartRecordingResponse struct {
	RecordingID string `json:"recording_id"`
	SessionID   string `json:"session_id"`
	StartedAt   string `json:"started_at"`
}

// StopRecordingResponse is the response after stopping recording.
type StopRecordingResponse struct {
	RecordingID string `json:"recording_id"`
	SessionID   string `json:"session_id"`
	ActionCount int    `json:"action_count"`
	StoppedAt   string `json:"stopped_at"`
}

// RecordingStatusResponse is the response for recording status.
type RecordingStatusResponse struct {
	SessionID   string `json:"session_id"`
	IsRecording bool   `json:"is_recording"`
	RecordingID string `json:"recording_id,omitempty"`
	ActionCount int    `json:"action_count"`
	FrameCount  int    `json:"frame_count,omitempty"`
	StartedAt   string `json:"started_at,omitempty"`
}

// =============================================================================
// Recorded Actions Types
// =============================================================================

// GetActionsResponse is the response for getting recorded actions.
type GetActionsResponse struct {
	SessionID   string                       `json:"session_id"`
	IsRecording bool                         `json:"is_recording,omitempty"`
	Actions     []livecapture.RecordedAction `json:"actions"`
	Count       int                          `json:"count"`
}

// GenerateWorkflowRequest is the request body for generating a workflow from recording.
type GenerateWorkflowRequest struct {
	SessionID   string     `json:"session_id"`
	Name        string     `json:"name"`
	ProjectID   *uuid.UUID `json:"project_id,omitempty"`
	ProjectName string     `json:"project_name,omitempty"`
	ActionRange *struct {
		Start int `json:"start"`
		End   int `json:"end"`
	} `json:"action_range,omitempty"`
	// Actions can be provided directly (with edits) instead of fetching from driver
	Actions []livecapture.RecordedAction `json:"actions,omitempty"`
}

// GenerateWorkflowResponse is the response after generating a workflow.
type GenerateWorkflowResponse struct {
	WorkflowID  uuid.UUID `json:"workflow_id"`
	ProjectID   uuid.UUID `json:"project_id"`
	Name        string    `json:"name"`
	NodeCount   int       `json:"node_count"`
	ActionCount int       `json:"action_count"`
}

// =============================================================================
// Replay Preview Types
// =============================================================================

// ReplayPreviewRequest is the request body for testing recorded actions.
type ReplayPreviewRequest struct {
	SessionID     string                       `json:"session_id"`
	Actions       []livecapture.RecordedAction `json:"actions"`
	Limit         *int                         `json:"limit,omitempty"`
	StopOnFailure *bool                        `json:"stop_on_failure,omitempty"`
	ActionTimeout *int                         `json:"action_timeout,omitempty"`
}

// ActionReplayError contains error details for a failed action.
type ActionReplayError struct {
	Message    string `json:"message"`
	Code       string `json:"code"`
	MatchCount *int   `json:"match_count,omitempty"`
	Selector   string `json:"selector,omitempty"`
}

// ActionReplayResult is the result of replaying a single action.
type ActionReplayResult struct {
	ActionID          string             `json:"action_id"`
	SequenceNum       int                `json:"sequence_num"`
	ActionType        string             `json:"action_type"`
	Success           bool               `json:"success"`
	DurationMs        int                `json:"duration_ms"`
	Error             *ActionReplayError `json:"error,omitempty"`
	ScreenshotOnError string             `json:"screenshot_on_error,omitempty"`
}

// ReplayPreviewResponse is the response from replay preview.
type ReplayPreviewResponse struct {
	Success         bool                 `json:"success"`
	TotalActions    int                  `json:"total_actions"`
	PassedActions   int                  `json:"passed_actions"`
	FailedActions   int                  `json:"failed_actions"`
	Results         []ActionReplayResult `json:"results"`
	TotalDurationMs int                  `json:"total_duration_ms"`
	StoppedEarly    bool                 `json:"stopped_early"`
}

// =============================================================================
// Navigation & Screenshot Types
// =============================================================================

// NavigateRecordingRequest is the request body for navigating the recording session.
type NavigateRecordingRequest struct {
	URL       string `json:"url"`
	WaitUntil string `json:"wait_until,omitempty"`
	TimeoutMs int    `json:"timeout_ms,omitempty"`
	Capture   bool   `json:"capture,omitempty"`
}

// NavigateRecordingResponse is the response from navigating the recording session.
type NavigateRecordingResponse struct {
	SessionID  string `json:"session_id"`
	URL        string `json:"url"`
	Title      string `json:"title,omitempty"`
	StatusCode int    `json:"status_code,omitempty"`
	Screenshot string `json:"screenshot,omitempty"`
}

// RecordingScreenshotResponse is the response for capturing a screenshot from the recording session.
type RecordingScreenshotResponse struct {
	SessionID  string `json:"session_id"`
	Screenshot string `json:"screenshot"`
}

// =============================================================================
// Frame Streaming Types
// =============================================================================

// RecordingFrameResponse is the response for lightweight frame previews.
// Uses WebP format for ~25-30% better compression than JPEG at same quality.
type RecordingFrameResponse struct {
	SessionID   string `json:"session_id"`
	Mime        string `json:"mime"` // "image/webp" or "image/jpeg"
	Image       string `json:"image"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	CapturedAt  string `json:"captured_at"`
	ContentHash string `json:"content_hash"`         // MD5 hash of raw frame buffer for reliable ETag
	PageTitle   string `json:"page_title,omitempty"` // Current page title (document.title)
	PageURL     string `json:"page_url,omitempty"`   // Current page URL
}

// RecordingViewportRequest updates viewport dimensions.
type RecordingViewportRequest struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// UpdateStreamSettingsRequest is the request body for updating stream settings mid-session.
type UpdateStreamSettingsRequest struct {
	// Quality: JPEG quality 1-100
	Quality *int `json:"quality,omitempty"`
	// FPS: Target frames per second 1-60
	FPS *int `json:"fps,omitempty"`
	// Scale: "css" for 1x, "device" for devicePixelRatio (cannot change mid-session)
	Scale string `json:"scale,omitempty"`
	// PerfMode: Enable/disable debug performance mode for this session
	PerfMode *bool `json:"perfMode,omitempty"`
}

// UpdateStreamSettingsResponse is the response after updating stream settings.
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
