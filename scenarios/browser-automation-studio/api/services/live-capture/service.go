package livecapture

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/automation/session"
)

// Service provides high-level operations for live capture mode.
// It orchestrates the session manager and workflow generator.
type Service struct {
	sessions  *session.Manager
	generator *WorkflowGenerator
	log       *logrus.Logger
}

// NewService creates a new live capture service.
func NewService(log *logrus.Logger) *Service {
	mgr, err := session.NewManager(session.WithLogger(log))
	if err != nil {
		log.WithError(err).Warn("Failed to create session manager, service will fail on first use")
		return &Service{
			sessions:  nil,
			generator: NewWorkflowGenerator(),
			log:       log,
		}
	}
	return &Service{
		sessions:  mgr,
		generator: NewWorkflowGenerator(),
		log:       log,
	}
}

// NewServiceWithManager creates a service with a custom session manager (for testing).
func NewServiceWithManager(mgr *session.Manager, log *logrus.Logger) *Service {
	return &Service{
		sessions:  mgr,
		generator: NewWorkflowGenerator(),
		log:       log,
	}
}

// NewServiceWithClient creates a service with a custom driver client (for testing).
// Deprecated: Use NewServiceWithManager instead.
func NewServiceWithClient(client *driver.Client, log *logrus.Logger) *Service {
	return &Service{
		sessions:  session.NewManagerWithClient(client, session.WithLogger(log)),
		generator: NewWorkflowGenerator(),
		log:       log,
	}
}

// GetDriverClient returns the underlying driver client for advanced operations.
func (s *Service) GetDriverClient() *driver.Client {
	if s.sessions == nil {
		return nil
	}
	return s.sessions.Client()
}

// GetSessionManager returns the session manager.
func (s *Service) GetSessionManager() *session.Manager {
	return s.sessions
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
	if s.sessions == nil {
		return nil, fmt.Errorf("session manager not initialized")
	}

	// Set defaults for frame streaming
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

	// Build session spec for recording mode
	spec := session.Spec{
		ExecutionID:    uuid.New(),
		WorkflowID:     uuid.New(),
		Mode:           session.ModeRecording,
		ViewportWidth:  cfg.ViewportWidth,
		ViewportHeight: cfg.ViewportHeight,
		ReuseMode:      "fresh",
		StorageState:   cfg.StorageState,
		FrameStreaming: &session.FrameStreamingConfig{
			Quality: quality,
			FPS:     fps,
			Scale:   scale,
		},
		Labels: map[string]string{
			"purpose": "record-mode",
		},
	}

	sess, err := s.sessions.Create(ctx, spec)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	// Navigate to initial URL if provided
	if cfg.InitialURL != "" {
		if _, err := sess.Navigate(ctx, cfg.InitialURL); err != nil {
			s.log.WithError(err).Warn("Failed to navigate to initial URL")
		}
	}

	return &SessionResult{
		SessionID: sess.ID(),
		CreatedAt: time.Now().UTC(),
	}, nil
}

// CloseSession closes a capture session.
func (s *Service) CloseSession(ctx context.Context, sessionID string) error {
	return s.sessions.Close(ctx, sessionID)
}

// GetStorageState retrieves storage state before closing (for session profiles).
func (s *Service) GetStorageState(ctx context.Context, sessionID string) (json.RawMessage, error) {
	sess, ok := s.sessions.Get(sessionID)
	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	return sess.GetStorageState(ctx)
}

// RecordingConfig configures recording start.
type RecordingConfig struct {
	APIHost string
	APIPort string
}

// StartRecording starts recording user actions.
func (s *Service) StartRecording(ctx context.Context, sessionID string, cfg *RecordingConfig) (*driver.StartRecordingResponse, error) {
	apiHost := cfg.APIHost
	if apiHost == "" {
		apiHost = "127.0.0.1"
	}
	apiPort := cfg.APIPort
	if apiPort == "" {
		apiPort = "8080"
	}

	req := &driver.StartRecordingRequest{
		CallbackURL:      fmt.Sprintf("http://%s:%s/api/v1/recordings/live/%s/action", apiHost, apiPort, sessionID),
		FrameCallbackURL: fmt.Sprintf("http://%s:%s/api/v1/recordings/live/%s/frame", apiHost, apiPort, sessionID),
		FrameQuality:     65,
		FrameFPS:         6,
	}

	return s.sessions.Client().StartRecording(ctx, sessionID, req)
}

// StopRecording stops recording user actions.
func (s *Service) StopRecording(ctx context.Context, sessionID string) (*driver.StopRecordingResponse, error) {
	return s.sessions.Client().StopRecording(ctx, sessionID)
}

// GetRecordingStatus gets the current recording status.
func (s *Service) GetRecordingStatus(ctx context.Context, sessionID string) (*driver.RecordingStatusResponse, error) {
	return s.sessions.Client().GetRecordingStatus(ctx, sessionID)
}

// GetRecordedActions retrieves all recorded actions.
func (s *Service) GetRecordedActions(ctx context.Context, sessionID string, clear bool) (*driver.GetActionsResponse, error) {
	return s.sessions.Client().GetRecordedActions(ctx, sessionID, clear)
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

	// Use provided actions or fetch from session
	if len(cfg.Actions) > 0 {
		actions = cfg.Actions
	} else {
		resp, err := s.sessions.Client().GetRecordedActions(ctx, sessionID, false)
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
func (s *Service) Navigate(ctx context.Context, sessionID string, req *driver.NavigateRequest) (*driver.NavigateResponse, error) {
	return s.sessions.Client().Navigate(ctx, sessionID, req)
}

// UpdateViewport updates the viewport dimensions.
func (s *Service) UpdateViewport(ctx context.Context, sessionID string, width, height int) (*driver.UpdateViewportResponse, error) {
	return s.sessions.Client().UpdateViewport(ctx, sessionID, &driver.UpdateViewportRequest{
		Width:  width,
		Height: height,
	})
}

// ValidateSelector validates a selector on the current page.
func (s *Service) ValidateSelector(ctx context.Context, sessionID, selector string) (*driver.ValidateSelectorResponse, error) {
	return s.sessions.Client().ValidateSelector(ctx, sessionID, &driver.ValidateSelectorRequest{
		Selector: selector,
	})
}

// ReplayPreview replays recorded actions for testing.
func (s *Service) ReplayPreview(ctx context.Context, sessionID string, req *driver.ReplayPreviewRequest) (*driver.ReplayPreviewResponse, error) {
	return s.sessions.Client().ReplayPreview(ctx, sessionID, req)
}

// UpdateStreamSettings updates stream settings for a session.
func (s *Service) UpdateStreamSettings(ctx context.Context, sessionID string, req *driver.UpdateStreamSettingsRequest) (*driver.UpdateStreamSettingsResponse, error) {
	return s.sessions.Client().UpdateStreamSettings(ctx, sessionID, req)
}

// CaptureScreenshot captures a screenshot from the current page.
func (s *Service) CaptureScreenshot(ctx context.Context, sessionID string, req *driver.CaptureScreenshotRequest) (*driver.CaptureScreenshotResponse, error) {
	return s.sessions.Client().CaptureScreenshot(ctx, sessionID, req)
}

// GetFrame retrieves the current frame from the session.
func (s *Service) GetFrame(ctx context.Context, sessionID, queryParams string) (*driver.GetFrameResponse, error) {
	return s.sessions.Client().GetFrame(ctx, sessionID, queryParams)
}

// ForwardInput forwards pointer/keyboard/wheel events to the driver.
func (s *Service) ForwardInput(ctx context.Context, sessionID string, body []byte) error {
	return s.sessions.Client().ForwardInput(ctx, sessionID, body)
}
