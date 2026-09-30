package session

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"

	"github.com/google/uuid"

	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/domain"
)

// Session wraps a playwright driver session with mode-aware behavior.
type Session struct {
	id                      string
	executionID             string
	leaseID                 string
	mode                    Mode
	client                  *driver.Client
	mu                      sync.RWMutex
	terminal                *terminalOperation
	lastInstructionSequence uint64
	// onTerminal is owned by Manager. It removes this session from the
	// manager's live index only after a close or lease release succeeds.
	// Executors hold Session directly, so cleanup cannot rely on callers going
	// back through Manager.Close.
	onTerminal func()

	// Multi-page tracking for recording sessions
	pages               *PageTracker
	initialDriverPageID string

	// ActualViewport is the viewport Playwright is actually using (may differ from requested)
	// Includes source attribution for debugging (e.g., "fingerprint", "requested", "default")
	actualViewport *driver.ActualViewport

	// Recording callbacks for unified action capture.
	// When set, all actions (manual, AI, or playback) are reported through these callbacks.
	recording *RecordingCallbacks
}

// terminalOperation owns one close/release attempt and its shared result.
// Results are published before done closes and never modified afterward.
type terminalOperation struct {
	done      chan struct{}
	artifacts *driver.CloseSessionResponse
	err       error
}

// --- Execution Mode Operations ---

// Run executes a compiled instruction and returns the step outcome.
// Only available in ModeExecution and ModeHybrid.
func (s *Session) Run(ctx context.Context, instr contracts.CompiledInstruction) (contracts.StepOutcome, error) {
	if s.mode == ModeRecording {
		return contracts.StepOutcome{}, errors.New("cannot run instructions in recording-only mode")
	}
	s.mu.Lock()
	if s.terminal != nil {
		s.mu.Unlock()
		return contracts.StepOutcome{}, errors.New("session closed")
	}
	// The wire number must remain an exact JavaScript integer.
	if s.lastInstructionSequence >= 1<<53-1 {
		s.mu.Unlock()
		return contracts.StepOutcome{}, errors.New("instruction sequence exhausted")
	}
	s.lastInstructionSequence++
	sequence := s.lastInstructionSequence
	s.mu.Unlock()
	if instr.InvocationID == "" {
		instr.InvocationID = uuid.NewString()
	}
	if instr.Attempt <= 0 {
		instr.Attempt = 1
	}
	return s.client.RunInstruction(ctx, s.id, s.executionID, s.leaseID, sequence, instr)
}

// --- Recording Mode Operations ---

// ForwardInput forwards pointer/keyboard/wheel events to the driver.
// Only available in ModeRecording and ModeHybrid.
func (s *Session) ForwardInput(ctx context.Context, input []byte) (*driver.ForwardInputResponse, error) {
	if s.mode == ModeExecution {
		return nil, errors.New("cannot forward input in execution-only mode")
	}
	if s.isClosed() {
		return nil, errors.New("session closed")
	}
	return s.client.ForwardInput(ctx, s.id, s.executionID, s.leaseID, input)
}

// StartRecording starts recording under this session's immutable lease.
// Only available in ModeRecording and ModeHybrid.
func (s *Session) StartRecording(ctx context.Context, req *driver.StartRecordingRequest) (*driver.StartRecordingResponse, error) {
	if s.mode == ModeExecution {
		return nil, errors.New("cannot start recording in execution-only mode")
	}
	if s.isClosed() {
		return nil, errors.New("session closed")
	}
	return s.client.StartRecording(ctx, s.id, s.executionID, s.leaseID, req)
}

// StopRecording returns the terminal receipt from this session's recording owner.
func (s *Session) StopRecording(ctx context.Context) (*driver.StopRecordingResponse, error) {
	if s.isClosed() {
		return nil, errors.New("session closed")
	}
	return s.client.StopRecording(ctx, s.id, s.executionID, s.leaseID)
}

// GetNavigationState reads the selected browser page under this immutable lease.
func (s *Session) GetNavigationState(ctx context.Context, expectedPageID string) (*driver.NavigationStateResponse, error) {
	if s.isClosed() {
		return nil, errors.New("session closed")
	}
	return s.client.GetNavigationState(ctx, s.id, s.executionID, s.leaseID, expectedPageID)
}

// GetNavigationStack reads browser history under this immutable lease.
func (s *Session) GetNavigationStack(ctx context.Context, expectedPageID string) (*driver.NavigationStackResponse, error) {
	if s.isClosed() {
		return nil, errors.New("session closed")
	}
	return s.client.GetNavigationStack(ctx, s.id, s.executionID, s.leaseID, expectedPageID)
}

// GetRecordedActions retrieves recorded actions for this session.
func (s *Session) GetRecordedActions(ctx context.Context) ([]driver.RecordedAction, error) {
	if s.isClosed() {
		return nil, errors.New("session closed")
	}
	resp, err := s.client.GetRecordedActions(ctx, s.id)
	if err != nil {
		return nil, err
	}
	return resp.Actions, nil
}

// AcknowledgeRecordedActions acknowledges durably committed entries under this lease.
func (s *Session) AcknowledgeRecordedActions(ctx context.Context, ids []string) error {
	if s.isClosed() {
		return errors.New("session closed")
	}
	return s.client.AcknowledgeRecordedActions(ctx, s.id, s.executionID, s.leaseID, ids)
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

// WithExpectedPage binds navigation to a previously selected driver page.
func WithExpectedPage(pageID string) NavigateOption {
	return func(r *driver.NavigateRequest) { r.ExpectedPageID = pageID }
}

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
	return s.client.Navigate(ctx, s.id, s.executionID, s.leaseID, req)
}

// NavigateHistory applies reload/back/forward under this Session's immutable lease.
func (s *Session) NavigateHistory(ctx context.Context, operation driver.HistoryNavigation, req *driver.HistoryNavigationRequest) (*driver.HistoryNavigationResponse, error) {
	if s.isClosed() {
		return nil, errors.New("session closed")
	}
	return s.client.NavigateHistory(ctx, s.id, s.executionID, s.leaseID, operation, req)
}

// UpdateViewport updates the viewport dimensions.
func (s *Session) UpdateViewport(ctx context.Context, width, height int, expectedPageID string) (*driver.UpdateViewportResponse, error) {
	if s.isClosed() {
		return nil, errors.New("session closed")
	}
	return s.client.UpdateViewport(ctx, s.id, s.executionID, s.leaseID, &driver.UpdateViewportRequest{Width: width, Height: height, ExpectedPageID: expectedPageID})
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

// CreatePage creates a browser tab under this Session's immutable lease.
func (s *Session) CreatePage(ctx context.Context, url string) (*driver.CreatePageResponse, error) {
	if s.isClosed() {
		return nil, errors.New("session closed")
	}
	return s.client.CreatePage(ctx, s.id, s.executionID, s.leaseID, url)
}

// ClosePage closes a browser tab under this Session's immutable lease.
func (s *Session) ClosePage(ctx context.Context, driverPageID string) (*driver.ClosePageResponse, error) {
	if s.isClosed() {
		return nil, errors.New("session closed")
	}
	return s.client.ClosePage(ctx, s.id, s.executionID, s.leaseID, driverPageID)
}

// SetActivePage switches the active page for execution.
// The driverPageID is the Playwright driver's internal identifier for the page.
// This is used during multi-page playback to execute actions on the correct page.
func (s *Session) SetActivePage(ctx context.Context, driverPageID string) error {
	if s.isClosed() {
		return errors.New("session closed")
	}
	return s.client.SetActivePage(ctx, s.id, s.executionID, s.leaseID, driverPageID)
}

// Reset resets the session to clean state.
func (s *Session) Reset(ctx context.Context) error {
	if s.isClosed() {
		return errors.New("session closed")
	}
	return s.client.ResetSession(ctx, s.id, s.executionID, s.leaseID)
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
	_, err := s.finalize(ctx, true)
	return err
}

// CloseWithArtifacts closes the session and returns its shared finalization result.
func (s *Session) CloseWithArtifacts(ctx context.Context) (*driver.CloseSessionResponse, error) {
	return s.finalize(ctx, false)
}

// finalize serializes close and release. A joining caller can stop waiting, but
// only the first caller's context controls the driver request.
func (s *Session) finalize(ctx context.Context, release bool) (*driver.CloseSessionResponse, error) {
	s.mu.Lock()
	if pending := s.terminal; pending != nil {
		s.mu.Unlock()
		select {
		case <-pending.done:
			return pending.artifacts, pending.err
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	pending := &terminalOperation{done: make(chan struct{})}
	s.terminal = pending
	s.mu.Unlock()

	var artifacts *driver.CloseSessionResponse
	var err error
	if release {
		err = s.client.ReleaseSessionLease(ctx, s.id, s.executionID, s.leaseID)
	} else {
		artifacts, err = s.client.CloseSessionWithLease(ctx, s.id, s.executionID, s.leaseID)
	}
	// Absence ends ownership of this lease; it does not prove capture durability.
	if isAbsentSessionError(err) {
		artifacts, err = nil, nil
	}
	if err == nil && s.onTerminal != nil {
		s.onTerminal()
	}
	s.mu.Lock()
	pending.artifacts, pending.err = artifacts, err
	if err != nil {
		// Existing waiters keep this attempt; a later explicit call may retry.
		s.terminal = nil
	}
	close(pending.done)
	s.mu.Unlock()
	return artifacts, err
}

func isAbsentSessionError(err error) bool {
	var driverErr *driver.Error
	return errors.As(err, &driverErr) && driverErr.Status == http.StatusNotFound
}

// --- Accessors ---

// ID returns the session ID.
func (s *Session) ID() string { return s.id }

// Mode returns the session mode.
func (s *Session) Mode() Mode { return s.mode }

// Pages returns the page tracker for this session.
// Returns nil if page tracking is not initialized.
func (s *Session) Pages() *PageTracker { return s.pages }

// FramePage resolves a captured source only while this lease and its active page
// still own the frame. It exposes the canonical page ID, never the lease.
func (s *Session) FramePage(source *driver.FrameSource) (uuid.UUID, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if source == nil || s.terminal != nil || s.pages == nil ||
		source.SessionID != s.id || source.ExecutionID == "" || source.ExecutionID != s.executionID ||
		source.LeaseID == "" || source.LeaseID != s.leaseID || source.PageID == "" {
		return uuid.Nil, false
	}
	page := s.pages.GetActivePage()
	if page == nil || page.Status != domain.PageStatusActive || page.DriverPageID != source.PageID {
		return uuid.Nil, false
	}
	return page.ID, true
}

// ActualViewport returns the viewport Playwright is actually using.
// May differ from requested dimensions due to browser profile fingerprint settings.
// Includes source attribution (e.g., "fingerprint", "requested", "default") and reason.
func (s *Session) ActualViewport() *driver.ActualViewport { return s.actualViewport }

// InitializePageTracking sets up page tracking for recording sessions.
// This should be called after session creation with the initial URL.
func (s *Session) InitializePageTracking(initialURL string) {
	s.pages = NewPageTracker(s.id, initialURL)
	if s.initialDriverPageID != "" {
		s.pages.SetInitialPageDriverID(s.initialDriverPageID)
	}
}

func (s *Session) isClosed() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.terminal != nil
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
