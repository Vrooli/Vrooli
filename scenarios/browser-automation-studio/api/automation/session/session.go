package session

import (
	"context"
	"encoding/json"
	"errors"
	"sync"

	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/driver"
)

// Session wraps a playwright driver session with mode-aware behavior.
type Session struct {
	id     string
	mode   Mode
	client *driver.Client
	mu     sync.RWMutex
	closed bool
	closeArtifacts *driver.CloseSessionResponse
}

// --- Execution Mode Operations ---

// Run executes a compiled instruction and returns the step outcome.
// Only available in ModeExecution and ModeHybrid.
func (s *Session) Run(ctx context.Context, instr contracts.CompiledInstruction) (contracts.StepOutcome, error) {
	if s.mode == ModeRecording {
		return contracts.StepOutcome{}, errors.New("cannot run instructions in recording-only mode")
	}
	if s.isClosed() {
		return contracts.StepOutcome{}, errors.New("session closed")
	}
	return s.client.RunInstruction(ctx, s.id, instr)
}

// --- Recording Mode Operations ---

// ForwardInput forwards pointer/keyboard/wheel events to the driver.
// Only available in ModeRecording and ModeHybrid.
func (s *Session) ForwardInput(ctx context.Context, input []byte) error {
	if s.mode == ModeExecution {
		return errors.New("cannot forward input in execution-only mode")
	}
	if s.isClosed() {
		return errors.New("session closed")
	}
	return s.client.ForwardInput(ctx, s.id, input)
}

// RecordingConfig configures recording start.
type RecordingConfig struct {
	ActionCallbackURL string
	FrameCallbackURL  string
	Quality           int
	FPS               int
}

// StartRecording starts recording user actions.
// Only available in ModeRecording and ModeHybrid.
func (s *Session) StartRecording(ctx context.Context, cfg RecordingConfig) error {
	if s.mode == ModeExecution {
		return errors.New("cannot start recording in execution-only mode")
	}
	if s.isClosed() {
		return errors.New("session closed")
	}
	_, err := s.client.StartRecording(ctx, s.id, &driver.StartRecordingRequest{
		CallbackURL:      cfg.ActionCallbackURL,
		FrameCallbackURL: cfg.FrameCallbackURL,
		FrameQuality:     cfg.Quality,
		FrameFPS:         cfg.FPS,
	})
	return err
}

// StopRecording stops recording user actions.
func (s *Session) StopRecording(ctx context.Context) error {
	if s.isClosed() {
		return errors.New("session closed")
	}
	_, err := s.client.StopRecording(ctx, s.id)
	return err
}

// GetRecordedActions retrieves recorded actions for this session.
func (s *Session) GetRecordedActions(ctx context.Context, clear bool) ([]driver.RecordedAction, error) {
	if s.isClosed() {
		return nil, errors.New("session closed")
	}
	resp, err := s.client.GetRecordedActions(ctx, s.id, clear)
	if err != nil {
		return nil, err
	}
	return resp.Actions, nil
}

// GetRecordingStatus gets the current recording status.
func (s *Session) GetRecordingStatus(ctx context.Context) (*driver.RecordingStatusResponse, error) {
	if s.isClosed() {
		return nil, errors.New("session closed")
	}
	return s.client.GetRecordingStatus(ctx, s.id)
}

// --- Common Operations (All Modes) ---

// NavigateOption configures navigation.
type NavigateOption func(*driver.NavigateRequest)

// WithWaitUntil sets the wait condition for navigation.
func WithWaitUntil(waitUntil string) NavigateOption {
	return func(r *driver.NavigateRequest) { r.WaitUntil = waitUntil }
}

// WithNavigateTimeout sets the navigation timeout in milliseconds.
func WithNavigateTimeout(ms int) NavigateOption {
	return func(r *driver.NavigateRequest) { r.TimeoutMs = ms }
}

// WithCapture enables screenshot capture after navigation.
func WithCapture(capture bool) NavigateOption {
	return func(r *driver.NavigateRequest) { r.Capture = capture }
}

// Navigate navigates the session to a URL.
func (s *Session) Navigate(ctx context.Context, url string, opts ...NavigateOption) (*driver.NavigateResponse, error) {
	if s.isClosed() {
		return nil, errors.New("session closed")
	}
	req := &driver.NavigateRequest{URL: url}
	for _, opt := range opts {
		opt(req)
	}
	return s.client.Navigate(ctx, s.id, req)
}

// UpdateViewport updates the viewport dimensions.
func (s *Session) UpdateViewport(ctx context.Context, width, height int) error {
	if s.isClosed() {
		return errors.New("session closed")
	}
	_, err := s.client.UpdateViewport(ctx, s.id, &driver.UpdateViewportRequest{
		Width:  width,
		Height: height,
	})
	return err
}

// Screenshot represents a captured screenshot.
type Screenshot struct {
	Data      string
	MediaType string
	Width     int
	Height    int
}

// CaptureScreenshot captures a screenshot from the current page.
func (s *Session) CaptureScreenshot(ctx context.Context) (*Screenshot, error) {
	if s.isClosed() {
		return nil, errors.New("session closed")
	}
	resp, err := s.client.CaptureScreenshot(ctx, s.id, &driver.CaptureScreenshotRequest{
		Format:  "jpeg",
		Quality: 85,
	})
	if err != nil {
		return nil, err
	}
	return &Screenshot{
		Data:      resp.Data,
		MediaType: resp.MediaType,
		Width:     resp.Width,
		Height:    resp.Height,
	}, nil
}

// GetStorageState retrieves the browser storage state for session persistence.
func (s *Session) GetStorageState(ctx context.Context) (json.RawMessage, error) {
	if s.isClosed() {
		return nil, errors.New("session closed")
	}
	return s.client.GetStorageState(ctx, s.id)
}

// ValidateSelector validates a selector on the current page.
func (s *Session) ValidateSelector(ctx context.Context, selector string) (*driver.ValidateSelectorResponse, error) {
	if s.isClosed() {
		return nil, errors.New("session closed")
	}
	return s.client.ValidateSelector(ctx, s.id, &driver.ValidateSelectorRequest{
		Selector: selector,
	})
}

// UpdateStreamSettings updates the frame streaming settings.
func (s *Session) UpdateStreamSettings(ctx context.Context, quality, fps *int, scale string) error {
	if s.isClosed() {
		return errors.New("session closed")
	}
	_, err := s.client.UpdateStreamSettings(ctx, s.id, &driver.UpdateStreamSettingsRequest{
		Quality: quality,
		FPS:     fps,
		Scale:   scale,
	})
	return err
}

// Reset resets the session to clean state.
func (s *Session) Reset(ctx context.Context) error {
	if s.isClosed() {
		return errors.New("session closed")
	}
	return s.client.ResetSession(ctx, s.id)
}

// Close closes the session.
func (s *Session) Close(ctx context.Context) error {
	_, err := s.CloseWithArtifacts(ctx)
	return err
}

// CloseWithArtifacts closes the session and returns any artifact metadata from the driver.
func (s *Session) CloseWithArtifacts(ctx context.Context) (*driver.CloseSessionResponse, error) {
	s.mu.Lock()
	if s.closed {
		artifacts := s.closeArtifacts
		s.mu.Unlock()
		return artifacts, nil
	}
	s.closed = true
	s.mu.Unlock()

	artifacts, err := s.client.CloseSession(ctx, s.id)
	if err != nil {
		// Rollback on failure - session is still open on driver
		s.mu.Lock()
		s.closed = false
		s.mu.Unlock()
		return nil, err
	}
	s.mu.Lock()
	s.closeArtifacts = artifacts
	s.mu.Unlock()
	return artifacts, nil
}

// --- Accessors ---

// ID returns the session ID.
func (s *Session) ID() string { return s.id }

// Mode returns the session mode.
func (s *Session) Mode() Mode { return s.mode }

func (s *Session) isClosed() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.closed
}
