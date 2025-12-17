package livecapture

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Service provides high-level operations for live capture mode.
// It orchestrates the driver client and workflow generator.
type Service struct {
	driver    *DriverClient
	generator *WorkflowGenerator
	log       *logrus.Logger
}

// NewService creates a new live capture service.
func NewService(log *logrus.Logger) *Service {
	return &Service{
		driver:    NewDriverClient(),
		generator: NewWorkflowGenerator(),
		log:       log,
	}
}

// NewServiceWithClient creates a service with a custom driver client (for testing).
func NewServiceWithClient(driver *DriverClient, log *logrus.Logger) *Service {
	return &Service{
		driver:    driver,
		generator: NewWorkflowGenerator(),
		log:       log,
	}
}

// GetDriverClient returns the underlying driver client for advanced operations.
func (s *Service) GetDriverClient() *DriverClient {
	return s.driver
}

// SessionConfig configures a new capture session.
type SessionConfig struct {
	ViewportWidth  int
	ViewportHeight int
	InitialURL     string
	StreamQuality  int
	StreamFPS      int
	StreamScale    string
	StorageState   json.RawMessage
	APIHost        string
	APIPort        string
}

// SessionResult is the result of creating a session.
type SessionResult struct {
	SessionID string
	CreatedAt time.Time
}

// CreateSession creates a new browser session for live capture.
func (s *Service) CreateSession(ctx context.Context, cfg *SessionConfig) (*SessionResult, error) {
	// Set defaults
	width := cfg.ViewportWidth
	height := cfg.ViewportHeight
	if width <= 0 {
		width = 1280
	}
	if height <= 0 {
		height = 720
	}
	quality := cfg.StreamQuality
	if quality <= 0 || quality > 100 {
		quality = 55
	}
	fps := cfg.StreamFPS
	if fps <= 0 || fps > 60 {
		fps = 6
	}
	scale := cfg.StreamScale
	if scale != "device" {
		scale = "css"
	}

	// Generate ephemeral IDs for the capture session
	executionID := uuid.New().String()
	workflowID := uuid.New().String()

	// Construct frame callback URL
	apiHost := cfg.APIHost
	if apiHost == "" {
		apiHost = "127.0.0.1"
	}
	apiPort := cfg.APIPort
	if apiPort == "" {
		apiPort = "8080"
	}
	frameCallbackURL := fmt.Sprintf("http://%s:%s/api/v1/recordings/live/placeholder/frame", apiHost, apiPort)

	req := &CreateSessionRequest{
		ExecutionID: executionID,
		WorkflowID:  workflowID,
		Viewport: Viewport{
			Width:  width,
			Height: height,
		},
		ReuseMode: "fresh",
		Labels: map[string]string{
			"purpose": "record-mode",
		},
		FrameStreaming: &FrameStreamingConfig{
			CallbackURL: frameCallbackURL,
			Quality:     quality,
			FPS:         fps,
			Scale:       scale,
		},
	}

	if len(cfg.StorageState) > 0 {
		req.StorageState = cfg.StorageState
	}

	resp, err := s.driver.CreateSession(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	// Navigate to initial URL if provided
	if cfg.InitialURL != "" {
		if err := s.driver.RunInstruction(ctx, resp.SessionID, []map[string]interface{}{
			{"op": "navigate", "url": cfg.InitialURL},
		}); err != nil {
			s.log.WithError(err).Warn("Failed to navigate to initial URL")
		}
	}

	return &SessionResult{
		SessionID: resp.SessionID,
		CreatedAt: time.Now().UTC(),
	}, nil
}

// CloseSession closes a capture session.
func (s *Service) CloseSession(ctx context.Context, sessionID string) error {
	return s.driver.CloseSession(ctx, sessionID)
}

// GetStorageState retrieves storage state before closing (for session profiles).
func (s *Service) GetStorageState(ctx context.Context, sessionID string) (json.RawMessage, error) {
	return s.driver.GetStorageState(ctx, sessionID)
}

// RecordingConfig configures recording start.
type RecordingConfig struct {
	APIHost string
	APIPort string
}

// StartRecording starts recording user actions.
func (s *Service) StartRecording(ctx context.Context, sessionID string, cfg *RecordingConfig) (*StartRecordingResponse, error) {
	apiHost := cfg.APIHost
	if apiHost == "" {
		apiHost = "127.0.0.1"
	}
	apiPort := cfg.APIPort
	if apiPort == "" {
		apiPort = "8080"
	}

	req := &StartRecordingRequest{
		CallbackURL:      fmt.Sprintf("http://%s:%s/api/v1/recordings/live/%s/action", apiHost, apiPort, sessionID),
		FrameCallbackURL: fmt.Sprintf("http://%s:%s/api/v1/recordings/live/%s/frame", apiHost, apiPort, sessionID),
		FrameQuality:     65,
		FrameFPS:         6,
	}

	return s.driver.StartRecording(ctx, sessionID, req)
}

// StopRecording stops recording user actions.
func (s *Service) StopRecording(ctx context.Context, sessionID string) (*StopRecordingResponse, error) {
	return s.driver.StopRecording(ctx, sessionID)
}

// GetRecordingStatus gets the current recording status.
func (s *Service) GetRecordingStatus(ctx context.Context, sessionID string) (*RecordingStatusResponse, error) {
	return s.driver.GetRecordingStatus(ctx, sessionID)
}

// GetRecordedActions retrieves all recorded actions.
func (s *Service) GetRecordedActions(ctx context.Context, sessionID string, clear bool) (*GetActionsResponse, error) {
	return s.driver.GetRecordedActions(ctx, sessionID, clear)
}

// GenerateWorkflowConfig configures workflow generation.
type GenerateWorkflowConfig struct {
	Name        string
	Actions     []RecordedAction
	ActionRange *ActionRange
}

// ActionRange specifies a subset of actions to use.
type ActionRange struct {
	Start int
	End   int
}

// GenerateWorkflowResult is the result of workflow generation.
type GenerateWorkflowResult struct {
	FlowDefinition map[string]interface{}
	NodeCount      int
	ActionCount    int
}

// GenerateWorkflow converts recorded actions to a workflow definition.
func (s *Service) GenerateWorkflow(ctx context.Context, sessionID string, cfg *GenerateWorkflowConfig) (*GenerateWorkflowResult, error) {
	var actions []RecordedAction

	// Use provided actions or fetch from driver
	if len(cfg.Actions) > 0 {
		actions = cfg.Actions
	} else {
		resp, err := s.driver.GetRecordedActions(ctx, sessionID, false)
		if err != nil {
			return nil, fmt.Errorf("get actions: %w", err)
		}
		actions = resp.Actions
	}

	// Apply action range if specified
	if cfg.ActionRange != nil {
		actions = ApplyActionRange(actions, cfg.ActionRange.Start, cfg.ActionRange.End)
	}

	if len(actions) == 0 {
		return nil, fmt.Errorf("no actions to convert")
	}

	// Generate workflow
	flowDef := s.generator.GenerateWorkflow(actions)

	// Count nodes
	nodeCount := 0
	if nodes, ok := flowDef["nodes"].([]map[string]interface{}); ok {
		nodeCount = len(nodes)
	}

	return &GenerateWorkflowResult{
		FlowDefinition: flowDef,
		NodeCount:      nodeCount,
		ActionCount:    len(actions),
	}, nil
}

// Navigate navigates the session to a URL.
func (s *Service) Navigate(ctx context.Context, sessionID string, req *NavigateRequest) (*NavigateResponse, error) {
	return s.driver.Navigate(ctx, sessionID, req)
}

// UpdateViewport updates the viewport dimensions.
func (s *Service) UpdateViewport(ctx context.Context, sessionID string, width, height int) (*UpdateViewportResponse, error) {
	return s.driver.UpdateViewport(ctx, sessionID, &UpdateViewportRequest{
		Width:  width,
		Height: height,
	})
}

// ValidateSelector validates a selector on the current page.
func (s *Service) ValidateSelector(ctx context.Context, sessionID, selector string) (*ValidateSelectorResponse, error) {
	return s.driver.ValidateSelector(ctx, sessionID, &ValidateSelectorRequest{
		Selector: selector,
	})
}

// ReplayPreview replays recorded actions for testing.
func (s *Service) ReplayPreview(ctx context.Context, sessionID string, req *ReplayPreviewRequest) (*ReplayPreviewResponse, error) {
	return s.driver.ReplayPreview(ctx, sessionID, req)
}

// UpdateStreamSettings updates stream settings for a session.
func (s *Service) UpdateStreamSettings(ctx context.Context, sessionID string, req *UpdateStreamSettingsRequest) (*UpdateStreamSettingsResponse, error) {
	return s.driver.UpdateStreamSettings(ctx, sessionID, req)
}

// CaptureScreenshot captures a screenshot from the current page.
func (s *Service) CaptureScreenshot(ctx context.Context, sessionID string, req *CaptureScreenshotRequest) (*CaptureScreenshotResponse, error) {
	return s.driver.CaptureScreenshot(ctx, sessionID, req)
}

// GetFrame retrieves the current frame from the session.
func (s *Service) GetFrame(ctx context.Context, sessionID, queryParams string) (*GetFrameResponse, error) {
	return s.driver.GetFrame(ctx, sessionID, queryParams)
}

// ForwardInput forwards pointer/keyboard/wheel events to the driver.
func (s *Service) ForwardInput(ctx context.Context, sessionID string, body []byte) error {
	return s.driver.ForwardInput(ctx, sessionID, body)
}
