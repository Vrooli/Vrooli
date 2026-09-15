package session

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"

	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/driver"
)

// Session wraps a playwright driver session with mode-aware behavior.
type Session struct {
	id             string
	executionID    string
	leaseID        string
	mode           Mode
	client         *driver.Client
	mu             sync.RWMutex
	closed         bool
	closeArtifacts *driver.CloseSessionResponse
	// onTerminal is owned by Manager. It removes this session from the
	// manager's live index only after a close or lease release succeeds.
	// Executors hold Session directly, so cleanup cannot rely on callers going
	// back through Manager.Close.
	onTerminal func()

	// Multi-page tracking for recording sessions
	pages *PageTracker

	// ActualViewport is the viewport Playwright is actually using (may differ from requested)
	// Includes source attribution for debugging (e.g., "fingerprint", "requested", "default")
	actualViewport *driver.ActualViewport

	// Recording callbacks for unified action capture.
	// When set, all actions (manual, AI, or playback) are reported through these callbacks.
	recording *RecordingCallbacks
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

// GetServiceWorkers retrieves the service workers for this session.
func (s *Session) GetServiceWorkers(ctx context.Context) (*driver.GetServiceWorkersResponse, error) {
	if s.isClosed() {
		return nil, errors.New("session closed")
	}
	return s.client.GetServiceWorkers(ctx, s.id)
}

// UnregisterAllServiceWorkers unregisters all service workers for this session.
func (s *Session) UnregisterAllServiceWorkers(ctx context.Context) (*driver.UnregisterServiceWorkersResponse, error) {
	if s.isClosed() {
		return nil, errors.New("session closed")
	}
	return s.client.UnregisterAllServiceWorkers(ctx, s.id)
}

// UnregisterServiceWorker unregisters a specific service worker by scope URL.
func (s *Session) UnregisterServiceWorker(ctx context.Context, scopeURL string) (*driver.UnregisterServiceWorkerResponse, error) {
	if s.isClosed() {
		return nil, errors.New("session closed")
	}
	return s.client.UnregisterServiceWorker(ctx, s.id, scopeURL)
}

// DownloadArtifact streams a driver-managed artifact by path.
// This intentionally allows closed sessions to fetch artifacts captured at teardown.
func (s *Session) DownloadArtifact(ctx context.Context, path string) (*driver.ArtifactDownload, error) {
	if s == nil || s.client == nil {
		return nil, errors.New("session client unavailable")
	}
	if path == "" {
		return nil, errors.New("artifact path required")
	}
	return s.client.DownloadArtifact(ctx, path)
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

// SetActivePage switches the active page for execution.
// The driverPageID is the Playwright driver's internal identifier for the page.
// This is used during multi-page playback to execute actions on the correct page.
func (s *Session) SetActivePage(ctx context.Context, driverPageID string) error {
	if s.isClosed() {
		return errors.New("session closed")
	}
	return s.client.SetActivePage(ctx, s.id, driverPageID)
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

// Release relinquishes this execution's lease but deliberately keeps the
// browser resource open. A released Session is terminal for this owner: only a
// subsequent execution can acquire a new lease for the resource.
func (s *Session) Release(ctx context.Context) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	s.mu.Unlock()

	if err := s.client.ReleaseSessionLease(ctx, s.id, s.executionID, s.leaseID); err != nil && !isAbsentSessionError(err) {
		s.mu.Lock()
		s.closed = false
		s.mu.Unlock()
		return err
	}
	s.notifyTerminal()
	return nil
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

	artifacts, err := s.client.CloseSessionWithLease(ctx, s.id, s.executionID, s.leaseID)
	if err != nil {
		if isAbsentSessionError(err) {
			// A 404 is terminal for this leased resource: the driver has already
			// removed it, so retaining local ownership would leak a manager permit
			// and permanently block future executions. Treat close as idempotent;
			// there are simply no teardown artifacts to return.
			s.notifyTerminal()
			return nil, nil
		}
		// Rollback on failure - session is still open on driver
		s.mu.Lock()
		s.closed = false
		s.mu.Unlock()
		return nil, err
	}
	s.mu.Lock()
	s.closeArtifacts = artifacts
	s.mu.Unlock()
	s.notifyTerminal()
	return artifacts, nil
}

func isAbsentSessionError(err error) bool {
	var driverErr *driver.Error
	return errors.As(err, &driverErr) && driverErr.Status == http.StatusNotFound
}

func (s *Session) notifyTerminal() {
	if s.onTerminal != nil {
		s.onTerminal()
	}
}

// --- Accessors ---

// ID returns the session ID.
func (s *Session) ID() string { return s.id }

// Mode returns the session mode.
func (s *Session) Mode() Mode { return s.mode }

// Pages returns the page tracker for this session.
// Returns nil if page tracking is not initialized.
func (s *Session) Pages() *PageTracker { return s.pages }

// ActualViewport returns the viewport Playwright is actually using.
// May differ from requested dimensions due to browser profile fingerprint settings.
// Includes source attribution (e.g., "fingerprint", "requested", "default") and reason.
func (s *Session) ActualViewport() *driver.ActualViewport { return s.actualViewport }

// InitializePageTracking sets up page tracking for recording sessions.
// This should be called after session creation with the initial URL.
func (s *Session) InitializePageTracking(initialURL string) {
	s.pages = NewPageTracker(s.id, initialURL)
}

func (s *Session) isClosed() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.closed
}

// Recording returns the recording callbacks if configured.
func (s *Session) Recording() *RecordingCallbacks { return s.recording }

// ReportAction reports an action to the recording callbacks if configured.
// This should be called for all browser actions regardless of source.
func (s *Session) ReportAction(action *RecordedActionInfo) {
	if s.recording != nil && s.recording.OnAction != nil {
		s.recording.OnAction(s.id, action)
	}
}

// ReportPageEvent reports a page event to the recording callbacks if configured.
func (s *Session) ReportPageEvent(event *PageEventInfo) {
	if s.recording != nil && s.recording.OnPageEvent != nil {
		s.recording.OnPageEvent(s.id, event)
	}
}

// ReportFrame reports a frame to the recording callbacks if configured.
func (s *Session) ReportFrame(frame *FrameInfo) {
	if s.recording != nil && s.recording.OnFrame != nil {
		s.recording.OnFrame(s.id, frame)
	}
}
