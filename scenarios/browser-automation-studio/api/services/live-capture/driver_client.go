package livecapture

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const (
	defaultDriverURL     = "http://127.0.0.1:39400"
	defaultClientTimeout = 30 * time.Second
	playwrightDriverEnv  = "PLAYWRIGHT_DRIVER_URL"
)

// DriverClient handles HTTP communication with the playwright-driver.
// It encapsulates all driver interactions to keep handlers thin.
type DriverClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewDriverClient creates a new driver client with the configured driver URL.
func NewDriverClient() *DriverClient {
	return &DriverClient{
		baseURL:    resolveDriverURL(),
		httpClient: &http.Client{Timeout: defaultClientTimeout},
	}
}

// NewDriverClientWithTimeout creates a driver client with a custom timeout.
func NewDriverClientWithTimeout(timeout time.Duration) *DriverClient {
	return &DriverClient{
		baseURL:    resolveDriverURL(),
		httpClient: &http.Client{Timeout: timeout},
	}
}

func resolveDriverURL() string {
	url := os.Getenv(playwrightDriverEnv)
	if url == "" {
		return defaultDriverURL
	}
	return url
}

// GetDriverURL returns the configured driver URL (for logging/debugging).
func (c *DriverClient) GetDriverURL() string {
	return c.baseURL
}

// DriverError represents an error from the driver.
type DriverError struct {
	StatusCode int
	Body       string
	Operation  string
}

func (e *DriverError) Error() string {
	return fmt.Sprintf("driver %s failed with status %d: %s", e.Operation, e.StatusCode, e.Body)
}

// CreateSessionRequest is the request to create a new capture session.
type CreateSessionRequest struct {
	ExecutionID    string                `json:"execution_id"`
	WorkflowID     string                `json:"workflow_id"`
	Viewport       Viewport              `json:"viewport"`
	ReuseMode      string                `json:"reuse_mode"`
	Labels         map[string]string     `json:"labels,omitempty"`
	StorageState   json.RawMessage       `json:"storage_state,omitempty"`
	FrameStreaming *FrameStreamingConfig `json:"frame_streaming,omitempty"`
}

// Viewport defines browser viewport dimensions.
type Viewport struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// FrameStreamingConfig configures live frame streaming.
type FrameStreamingConfig struct {
	CallbackURL string `json:"callback_url"`
	Quality     int    `json:"quality"`
	FPS         int    `json:"fps"`
	Scale       string `json:"scale"`
}

// CreateSessionResponse is the response from creating a session.
type CreateSessionResponse struct {
	SessionID string `json:"session_id"`
}

// CreateSession creates a new browser session for capture.
func (c *DriverClient) CreateSession(ctx context.Context, req *CreateSessionRequest) (*CreateSessionResponse, error) {
	var resp CreateSessionResponse
	if err := c.post(ctx, "/session/start", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// CloseSession closes a capture session.
func (c *DriverClient) CloseSession(ctx context.Context, sessionID string) error {
	return c.postNoBody(ctx, fmt.Sprintf("/session/%s/close", sessionID), nil)
}

// StartRecordingRequest is the request to start recording.
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

// StartRecording starts recording user actions.
func (c *DriverClient) StartRecording(ctx context.Context, sessionID string, req *StartRecordingRequest) (*StartRecordingResponse, error) {
	var resp StartRecordingResponse
	if err := c.post(ctx, fmt.Sprintf("/session/%s/record/start", sessionID), req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// StopRecordingResponse is the response from stopping recording.
type StopRecordingResponse struct {
	SessionID   string `json:"session_id"`
	IsRecording bool   `json:"is_recording"`
	ActionCount int    `json:"action_count"`
	StoppedAt   string `json:"stopped_at"`
}

// StopRecording stops recording user actions.
func (c *DriverClient) StopRecording(ctx context.Context, sessionID string) (*StopRecordingResponse, error) {
	var resp StopRecordingResponse
	if err := c.postNoBody(ctx, fmt.Sprintf("/session/%s/record/stop", sessionID), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// RecordingStatusResponse is the response from getting recording status.
type RecordingStatusResponse struct {
	SessionID   string `json:"session_id"`
	IsRecording bool   `json:"is_recording"`
	ActionCount int    `json:"action_count"`
	StartedAt   string `json:"started_at,omitempty"`
	FrameCount  int    `json:"frame_count"`
}

// GetRecordingStatus gets the current recording status.
func (c *DriverClient) GetRecordingStatus(ctx context.Context, sessionID string) (*RecordingStatusResponse, error) {
	var resp RecordingStatusResponse
	if err := c.get(ctx, fmt.Sprintf("/session/%s/record/status", sessionID), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetActionsResponse is the response from getting recorded actions.
type GetActionsResponse struct {
	SessionID   string           `json:"session_id"`
	IsRecording bool             `json:"is_recording"`
	Actions     []RecordedAction `json:"actions"`
}

// GetRecordedActions retrieves all recorded actions for a session.
func (c *DriverClient) GetRecordedActions(ctx context.Context, sessionID string, clear bool) (*GetActionsResponse, error) {
	path := fmt.Sprintf("/session/%s/record/actions", sessionID)
	if clear {
		path += "?clear=true"
	}
	var resp GetActionsResponse
	if err := c.get(ctx, path, &resp); err != nil {
		return nil, err
	}
	if resp.Actions == nil {
		resp.Actions = []RecordedAction{}
	}
	return &resp, nil
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

// Navigate navigates the session to a URL.
func (c *DriverClient) Navigate(ctx context.Context, sessionID string, req *NavigateRequest) (*NavigateResponse, error) {
	var resp NavigateResponse
	if err := c.post(ctx, fmt.Sprintf("/session/%s/record/navigate", sessionID), req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
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

// UpdateViewport updates the viewport dimensions.
func (c *DriverClient) UpdateViewport(ctx context.Context, sessionID string, req *UpdateViewportRequest) (*UpdateViewportResponse, error) {
	var resp UpdateViewportResponse
	if err := c.post(ctx, fmt.Sprintf("/session/%s/record/viewport", sessionID), req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
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

// ValidateSelector validates a selector on the current page.
func (c *DriverClient) ValidateSelector(ctx context.Context, sessionID string, req *ValidateSelectorRequest) (*ValidateSelectorResponse, error) {
	var resp ValidateSelectorResponse
	if err := c.post(ctx, fmt.Sprintf("/session/%s/record/validate-selector", sessionID), req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
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

// ReplayPreview replays recorded actions for testing.
func (c *DriverClient) ReplayPreview(ctx context.Context, sessionID string, req *ReplayPreviewRequest) (*ReplayPreviewResponse, error) {
	var resp ReplayPreviewResponse
	if err := c.post(ctx, fmt.Sprintf("/session/%s/record/replay-preview", sessionID), req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
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

// UpdateStreamSettings updates the stream settings for a session.
func (c *DriverClient) UpdateStreamSettings(ctx context.Context, sessionID string, req *UpdateStreamSettingsRequest) (*UpdateStreamSettingsResponse, error) {
	var resp UpdateStreamSettingsResponse
	if err := c.post(ctx, fmt.Sprintf("/session/%s/record/stream-settings", sessionID), req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
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

// CaptureScreenshot captures a screenshot from the current page.
func (c *DriverClient) CaptureScreenshot(ctx context.Context, sessionID string, req *CaptureScreenshotRequest) (*CaptureScreenshotResponse, error) {
	var resp CaptureScreenshotResponse
	if err := c.post(ctx, fmt.Sprintf("/session/%s/record/screenshot", sessionID), req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
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

// GetFrame retrieves the current frame from the session.
func (c *DriverClient) GetFrame(ctx context.Context, sessionID string, queryParams string) (*GetFrameResponse, error) {
	path := fmt.Sprintf("/session/%s/record/frame", sessionID)
	if queryParams != "" {
		path += "?" + queryParams
	}
	var resp GetFrameResponse
	if err := c.get(ctx, path, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ForwardInput forwards pointer/keyboard/wheel events to the driver.
func (c *DriverClient) ForwardInput(ctx context.Context, sessionID string, body []byte) error {
	return c.postRaw(ctx, fmt.Sprintf("/session/%s/record/input", sessionID), body, nil)
}

// GetStorageState retrieves the browser storage state for session persistence.
func (c *DriverClient) GetStorageState(ctx context.Context, sessionID string) (json.RawMessage, error) {
	var resp struct {
		StorageState json.RawMessage `json:"storage_state"`
	}
	if err := c.get(ctx, fmt.Sprintf("/session/%s/storage-state", sessionID), &resp); err != nil {
		return nil, err
	}
	return resp.StorageState, nil
}

// RunInstruction runs a single instruction in the session (used for initial navigation).
func (c *DriverClient) RunInstruction(ctx context.Context, sessionID string, instructions []map[string]interface{}) error {
	req := map[string]interface{}{
		"instructions": instructions,
	}
	return c.post(ctx, fmt.Sprintf("/session/%s/run", sessionID), req, nil)
}

// HTTP helper methods

func (c *DriverClient) get(ctx context.Context, path string, response interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	return c.doRequest(req, response, "GET "+path)
}

func (c *DriverClient) post(ctx context.Context, path string, body interface{}, response interface{}) error {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	return c.doRequest(req, response, "POST "+path)
}

func (c *DriverClient) postNoBody(ctx context.Context, path string, response interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	return c.doRequest(req, response, "POST "+path)
}

func (c *DriverClient) postRaw(ctx context.Context, path string, body []byte, response interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	return c.doRequest(req, response, "POST "+path)
}

func (c *DriverClient) doRequest(req *http.Request, response interface{}, operation string) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("driver unavailable: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return &DriverError{
			StatusCode: resp.StatusCode,
			Body:       string(body),
			Operation:  operation,
		}
	}

	if response != nil && len(body) > 0 {
		if err := json.Unmarshal(body, response); err != nil {
			return fmt.Errorf("parse response: %w", err)
		}
	}

	return nil
}
