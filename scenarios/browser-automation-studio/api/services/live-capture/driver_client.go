package livecapture

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/vrooli/browser-automation-studio/automation/driver"
)

// DriverClient handles HTTP communication with the playwright-driver.
// This is a thin wrapper around the unified driver.Client for backward compatibility.
type DriverClient struct {
	client *driver.Client
}

// NewDriverClient creates a new driver client with the configured driver URL.
func NewDriverClient() *DriverClient {
	client, err := driver.NewRecordingClient()
	if err != nil {
		// Fall back to a client that will fail on first use
		// This maintains backward compatibility with code that doesn't check errors
		return &DriverClient{client: nil}
	}
	return &DriverClient{client: client}
}

// NewDriverClientWithTimeout creates a driver client with a custom timeout.
func NewDriverClientWithTimeout(timeout time.Duration) *DriverClient {
	client, err := driver.NewClient(driver.WithTimeout(timeout))
	if err != nil {
		return &DriverClient{client: nil}
	}
	return &DriverClient{client: client}
}

// GetDriverURL returns the configured driver URL (for logging/debugging).
func (c *DriverClient) GetDriverURL() string {
	if c.client == nil {
		return ""
	}
	return c.client.GetDriverURL()
}

// DriverError represents an error from the driver.
// This is kept for backward compatibility - new code should use driver.Error.
type DriverError = driver.Error

// Type aliases for request/response types - canonical types are in driver package.
type (
	Viewport              = driver.Viewport
	FrameStreamingConfig  = driver.FrameStreamingConfig
	CreateSessionRequest  = driver.CreateSessionRequest
	CreateSessionResponse = driver.CreateSessionResponse

	StartRecordingRequest   = driver.StartRecordingRequest
	StartRecordingResponse  = driver.StartRecordingResponse
	StopRecordingResponse   = driver.StopRecordingResponse
	RecordingStatusResponse = driver.RecordingStatusResponse
	GetActionsResponse      = driver.GetActionsResponse

	NavigateRequest  = driver.NavigateRequest
	NavigateResponse = driver.NavigateResponse

	UpdateViewportRequest  = driver.UpdateViewportRequest
	UpdateViewportResponse = driver.UpdateViewportResponse

	ValidateSelectorRequest  = driver.ValidateSelectorRequest
	ValidateSelectorResponse = driver.ValidateSelectorResponse

	ReplayPreviewRequest  = driver.ReplayPreviewRequest
	ReplayPreviewResponse = driver.ReplayPreviewResponse
	ActionResult          = driver.ActionResult

	UpdateStreamSettingsRequest  = driver.UpdateStreamSettingsRequest
	UpdateStreamSettingsResponse = driver.UpdateStreamSettingsResponse

	CaptureScreenshotRequest  = driver.CaptureScreenshotRequest
	CaptureScreenshotResponse = driver.CaptureScreenshotResponse

	GetFrameResponse = driver.GetFrameResponse
)

// CreateSession creates a new browser session for capture.
func (c *DriverClient) CreateSession(ctx context.Context, req *CreateSessionRequest) (*CreateSessionResponse, error) {
	if c.client == nil {
		return nil, fmt.Errorf("driver client not initialized")
	}
	return c.client.CreateSession(ctx, req)
}

// CloseSession closes a capture session.
func (c *DriverClient) CloseSession(ctx context.Context, sessionID string) error {
	if c.client == nil {
		return fmt.Errorf("driver client not initialized")
	}
	return c.client.CloseSession(ctx, sessionID)
}

// StartRecording starts recording user actions.
func (c *DriverClient) StartRecording(ctx context.Context, sessionID string, req *StartRecordingRequest) (*StartRecordingResponse, error) {
	if c.client == nil {
		return nil, fmt.Errorf("driver client not initialized")
	}
	return c.client.StartRecording(ctx, sessionID, req)
}

// StopRecording stops recording user actions.
func (c *DriverClient) StopRecording(ctx context.Context, sessionID string) (*StopRecordingResponse, error) {
	if c.client == nil {
		return nil, fmt.Errorf("driver client not initialized")
	}
	return c.client.StopRecording(ctx, sessionID)
}

// GetRecordingStatus gets the current recording status.
func (c *DriverClient) GetRecordingStatus(ctx context.Context, sessionID string) (*RecordingStatusResponse, error) {
	if c.client == nil {
		return nil, fmt.Errorf("driver client not initialized")
	}
	return c.client.GetRecordingStatus(ctx, sessionID)
}

// GetRecordedActions retrieves all recorded actions for a session.
func (c *DriverClient) GetRecordedActions(ctx context.Context, sessionID string, clear bool) (*GetActionsResponse, error) {
	if c.client == nil {
		return nil, fmt.Errorf("driver client not initialized")
	}
	return c.client.GetRecordedActions(ctx, sessionID, clear)
}

// Navigate navigates the session to a URL.
func (c *DriverClient) Navigate(ctx context.Context, sessionID string, req *NavigateRequest) (*NavigateResponse, error) {
	if c.client == nil {
		return nil, fmt.Errorf("driver client not initialized")
	}
	return c.client.Navigate(ctx, sessionID, req)
}

// UpdateViewport updates the viewport dimensions.
func (c *DriverClient) UpdateViewport(ctx context.Context, sessionID string, req *UpdateViewportRequest) (*UpdateViewportResponse, error) {
	if c.client == nil {
		return nil, fmt.Errorf("driver client not initialized")
	}
	return c.client.UpdateViewport(ctx, sessionID, req)
}

// ValidateSelector validates a selector on the current page.
func (c *DriverClient) ValidateSelector(ctx context.Context, sessionID string, req *ValidateSelectorRequest) (*ValidateSelectorResponse, error) {
	if c.client == nil {
		return nil, fmt.Errorf("driver client not initialized")
	}
	return c.client.ValidateSelector(ctx, sessionID, req)
}

// ReplayPreview replays recorded actions for testing.
func (c *DriverClient) ReplayPreview(ctx context.Context, sessionID string, req *ReplayPreviewRequest) (*ReplayPreviewResponse, error) {
	if c.client == nil {
		return nil, fmt.Errorf("driver client not initialized")
	}
	return c.client.ReplayPreview(ctx, sessionID, req)
}

// UpdateStreamSettings updates the stream settings for a session.
func (c *DriverClient) UpdateStreamSettings(ctx context.Context, sessionID string, req *UpdateStreamSettingsRequest) (*UpdateStreamSettingsResponse, error) {
	if c.client == nil {
		return nil, fmt.Errorf("driver client not initialized")
	}
	return c.client.UpdateStreamSettings(ctx, sessionID, req)
}

// CaptureScreenshot captures a screenshot from the current page.
func (c *DriverClient) CaptureScreenshot(ctx context.Context, sessionID string, req *CaptureScreenshotRequest) (*CaptureScreenshotResponse, error) {
	if c.client == nil {
		return nil, fmt.Errorf("driver client not initialized")
	}
	return c.client.CaptureScreenshot(ctx, sessionID, req)
}

// GetFrame retrieves the current frame from the session.
func (c *DriverClient) GetFrame(ctx context.Context, sessionID string, queryParams string) (*GetFrameResponse, error) {
	if c.client == nil {
		return nil, fmt.Errorf("driver client not initialized")
	}
	return c.client.GetFrame(ctx, sessionID, queryParams)
}

// ForwardInput forwards pointer/keyboard/wheel events to the driver.
func (c *DriverClient) ForwardInput(ctx context.Context, sessionID string, body []byte) error {
	if c.client == nil {
		return fmt.Errorf("driver client not initialized")
	}
	return c.client.ForwardInput(ctx, sessionID, body)
}

// GetStorageState retrieves the browser storage state for session persistence.
func (c *DriverClient) GetStorageState(ctx context.Context, sessionID string) (json.RawMessage, error) {
	if c.client == nil {
		return nil, fmt.Errorf("driver client not initialized")
	}
	return c.client.GetStorageState(ctx, sessionID)
}

// RunInstruction runs instructions in the session (used for initial navigation).
func (c *DriverClient) RunInstruction(ctx context.Context, sessionID string, instructions []map[string]interface{}) error {
	if c.client == nil {
		return fmt.Errorf("driver client not initialized")
	}
	return c.client.RunInstructions(ctx, sessionID, instructions)
}
