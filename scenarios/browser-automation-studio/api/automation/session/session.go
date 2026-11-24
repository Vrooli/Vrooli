package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/google/uuid"

	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/domain"
	"github.com/vrooli/browser-automation-studio/internal/sessionpermission"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
)

// Session wraps a playwright driver session with mode-aware behavior.
type Session struct {
	id                      string
	executionID             string
	leaseID                 string
	browserBackend          string
	mode                    Mode
	appTarget               bool
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
	activeDriverPageID  string

	// ActualViewport is the viewport Playwright is actually using (may differ from requested)
	// Includes source attribution for debugging (e.g., "fingerprint", "requested", "default")
	actualViewport      *driver.ActualViewport
	identityEvidence    *driver.IdentityEvidence
	audioDeviceEvidence *driver.AudioDeviceEvidence

	// Recording callbacks for unified action capture.
	// When set, all actions (manual, AI, or playback) are reported through these callbacks.
	recording         *RecordingCallbacks
	authorizer        sessionpermission.Authorizer
	callerResolver    sessionpermission.CallerResolver
	expectedCallerID  string
	permissionProfile *sessionpermission.Profile
}

// AudioDeviceEvidence returns a copy of the driver-measured media-device
// metadata for a specifically authorized PipeWire qualification session.
func (s *Session) AudioDeviceEvidence() *driver.AudioDeviceEvidence {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.audioDeviceEvidence == nil {
		return nil
	}
	copy := *s.audioDeviceEvidence
	return &copy
}

// IdentityEvidence returns the immutable owner observation captured for this
// lease. It is a receipt, not a caller-supplied identity assertion.
func (s *Session) IdentityEvidence() *driver.IdentityEvidence {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.identityEvidence == nil {
		return nil
	}
	raw, err := json.Marshal(s.identityEvidence)
	if err != nil {
		return nil
	}
	var clone driver.IdentityEvidence
	if json.Unmarshal(raw, &clone) != nil {
		return nil
	}
	return &clone
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
	if err := s.authorizeInstruction(ctx, instr); err != nil {
		return contracts.StepOutcome{}, err
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
	if err := s.AuthorizeAction(ctx, "input", "", ""); err != nil {
		return nil, err
	}
	return s.client.ForwardInput(ctx, s.id, s.executionID, s.leaseID, input)
}

// ReleaseInput clears held pointer buttons and modifier keys after a stream
// control receipt expires or is revoked. It is owner cleanup, not a new page
// action; callers must already have resolved this exact live session lease.
func (s *Session) ReleaseInput(ctx context.Context) error {
	if s.isClosed() {
		return errors.New("session closed")
	}
	for _, button := range []string{"left", "right", "middle", "back", "forward"} {
		body, err := json.Marshal(map[string]any{"type": "pointer", "action": "release", "button": button, "modifiers": []string{}})
		if err != nil {
			return err
		}
		if _, err = s.client.ForwardInput(ctx, s.id, s.executionID, s.leaseID, body); err != nil {
			return err
		}
	}
	return nil
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
	if err := s.AuthorizeAction(ctx, "recording_start", "", ""); err != nil {
		return nil, err
	}
	return s.client.StartRecording(ctx, s.id, s.executionID, s.leaseID, req)
}

// StopRecording returns the terminal receipt from this session's recording owner.
func (s *Session) StopRecording(ctx context.Context) (*driver.StopRecordingResponse, error) {
	if s.isClosed() {
		return nil, errors.New("session closed")
	}
	if err := s.AuthorizeAction(ctx, "recording_stop", "", ""); err != nil {
		return nil, err
	}
	return s.client.StopRecording(ctx, s.id, s.executionID, s.leaseID)
}

// GetNavigationState reads the selected browser page under this immutable lease.
func (s *Session) GetNavigationState(ctx context.Context, expectedPageID string) (*driver.NavigationStateResponse, error) {
	return s.ReadNavigationState(ctx, "observe", expectedPageID, "")
}

// ReadNavigationState returns the selected page state after one owner-policy
// decision for the logical operation that requested the read.
func (s *Session) ReadNavigationState(ctx context.Context, permissionKind, expectedPageID, approvalReceipt string) (*driver.NavigationStateResponse, error) {
	if s.isClosed() {
		return nil, errors.New("session closed")
	}
	if expectedPageID == "" {
		expectedPageID = s.currentDriverPageID()
	}
	if err := s.authorizeActionRequestForPage(ctx, sessionpermission.ActionRequest{Kind: permissionKind, ApprovalReceipt: approvalReceipt}, expectedPageID); err != nil {
		return nil, err
	}
	return s.client.GetNavigationState(ctx, s.id, s.executionID, s.leaseID, expectedPageID)
}

// CaptureObservation obtains state and pixels as one logical permissioned
// observation. The profile owner charges one observe admission; the subsequent
// state and screenshot reads are the physical parts of that admitted result.
func (s *Session) CaptureObservation(ctx context.Context, expectedPageID string) (*driver.NavigationStateResponse, *driver.CaptureScreenshotResponse, error) {
	return s.CaptureObservationWithScreenshot(ctx, expectedPageID, driver.CaptureScreenshotRequest{Format: "png"})
}

// CaptureObservationWithScreenshot obtains page state and requested screenshot
// pixels under one observe admission. TestID selects one stable DOM test target;
// it is passed to the driver as data, never interpreted as a selector.
func (s *Session) CaptureObservationWithScreenshot(ctx context.Context, expectedPageID string, screenshotRequest driver.CaptureScreenshotRequest) (*driver.NavigationStateResponse, *driver.CaptureScreenshotResponse, error) {
	if s.isClosed() {
		return nil, nil, errors.New("session closed")
	}
	if expectedPageID == "" {
		expectedPageID = s.currentDriverPageID()
	}
	state, err := s.ReadNavigationState(ctx, "observe", expectedPageID, "")
	if err != nil {
		return nil, nil, err
	}
	if screenshotRequest.Format == "" {
		screenshotRequest.Format = "png"
	}
	screenshot, err := s.client.CaptureScreenshot(ctx, s.id, s.executionID, s.leaseID, &screenshotRequest)
	if err != nil {
		return nil, nil, err
	}
	return state, screenshot, nil
}

// GetNavigationStack reads browser history under this immutable lease.
func (s *Session) GetNavigationStack(ctx context.Context, expectedPageID string) (*driver.NavigationStackResponse, error) {
	if s.isClosed() {
		return nil, errors.New("session closed")
	}
	if err := s.AuthorizeAction(ctx, "history_read", "", ""); err != nil {
		return nil, err
	}
	if expectedPageID == "" {
		expectedPageID = s.currentDriverPageID()
	}
	return s.client.GetNavigationStack(ctx, s.id, s.executionID, s.leaseID, expectedPageID)
}

// GetRecordedActions retrieves recorded actions for this session.
func (s *Session) GetRecordedActions(ctx context.Context) ([]driver.RecordedAction, error) {
	resp, err := s.GetRecordedActionsResponse(ctx)
	if err != nil {
		return nil, err
	}
	return resp.Actions, nil
}

// GetRecordedActionsResponse retrieves the complete recording receipt through this lease owner.
func (s *Session) GetRecordedActionsResponse(ctx context.Context) (*driver.GetActionsResponse, error) {
	if s.isClosed() {
		return nil, errors.New("session closed")
	}
	if err := s.AuthorizeAction(ctx, "history_read", "", ""); err != nil {
		return nil, err
	}
	return s.client.GetRecordedActions(ctx, s.id, s.executionID, s.leaseID)
}

// AcknowledgeRecordedActions acknowledges durably committed entries under this lease.
func (s *Session) AcknowledgeRecordedActions(ctx context.Context, ids []string) error {
	if s.isClosed() {
		return errors.New("session closed")
	}
	if err := s.AuthorizeAction(ctx, "history_read", "", ""); err != nil {
		return err
	}
	return s.client.AcknowledgeRecordedActions(ctx, s.id, s.executionID, s.leaseID, ids)
}

// GetRecordingStatus gets the current recording status.
func (s *Session) GetRecordingStatus(ctx context.Context) (*driver.RecordingStatusResponse, error) {
	if s.isClosed() {
		return nil, errors.New("session closed")
	}
	if err := s.AuthorizeAction(ctx, "history_read", "", ""); err != nil {
		return nil, err
	}
	return s.client.GetRecordingStatus(ctx, s.id, s.executionID, s.leaseID)
}

// CaptureScreenshotWithOptions captures a screenshot through this owned lease.
func (s *Session) CaptureScreenshotWithOptions(ctx context.Context, req *driver.CaptureScreenshotRequest) (*driver.CaptureScreenshotResponse, error) {
	if s.isClosed() {
		return nil, errors.New("session closed")
	}
	if err := s.AuthorizeAction(ctx, "screenshot", "", ""); err != nil {
		return nil, err
	}
	return s.client.CaptureScreenshot(ctx, s.id, s.executionID, s.leaseID, req)
}

// GetFrame retrieves a frame through this owned lease.
func (s *Session) GetFrame(ctx context.Context, query string) (*driver.GetFrameResponse, error) {
	if s.isClosed() {
		return nil, errors.New("session closed")
	}
	if err := s.AuthorizeAction(ctx, "view", "", ""); err != nil {
		return nil, err
	}
	return s.client.GetFrame(ctx, s.id, s.executionID, s.leaseID, query)
}

// UpdateStreamSettingsWithOptions preserves the driver's complete settings surface under this lease.
func (s *Session) UpdateStreamSettingsWithOptions(ctx context.Context, req *driver.UpdateStreamSettingsRequest) (*driver.UpdateStreamSettingsResponse, error) {
	if s.isClosed() {
		return nil, errors.New("session closed")
	}
	if err := s.AuthorizeAction(ctx, "control", "", ""); err != nil {
		return nil, err
	}
	return s.client.UpdateStreamSettings(ctx, s.id, s.executionID, s.leaseID, req)
}

// ReplayPreview runs a non-persistent preview under this session owner.
func (s *Session) ReplayPreview(ctx context.Context, req *driver.ReplayPreviewRequest) (*driver.ReplayPreviewResponse, error) {
	if s.isClosed() {
		return nil, errors.New("session closed")
	}
	if err := s.AuthorizeAction(ctx, "replay_preview", "", ""); err != nil {
		return nil, err
	}
	return s.client.ReplayPreview(ctx, s.id, s.executionID, s.leaseID, req)
}

// RouteRequest proxies an owned driver's session endpoint while retaining the
// session broker as the only API boundary to /session routes.
func (s *Session) RouteRequest(ctx context.Context, method, suffix string, body []byte, headers http.Header) (*http.Response, error) {
	if s.isClosed() {
		return nil, errors.New("session closed")
	}
	kind := ""
	switch method + " " + suffix {
	case http.MethodGet + " /record/debug":
		kind = "observe"
	case http.MethodPost + " /ai-navigate", http.MethodPost + " /ai-navigate/resume":
		kind = "ai_navigate"
	case http.MethodPost + " /ai-navigate/abort":
		kind = "close"
	case http.MethodPost + " /audio/restart":
		kind = "audio_playback_restart"
	case http.MethodPost + " /audio/stop":
		kind = "audio_playback_stop"
	case http.MethodPost + " /audio/diagnostics":
		kind = "audio_observe"
	default:
		return nil, errors.New("session route is not an authorized operation")
	}
	if err := s.AuthorizeAction(ctx, kind, "", ""); err != nil {
		return nil, err
	}
	return s.client.SessionRouteRequest(ctx, s.id, s.executionID, s.leaseID, method, suffix, body, headers)
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
	if err := s.AuthorizeAction(ctx, "navigate", url, ""); err != nil {
		return nil, err
	}
	req := &driver.NavigateRequest{URL: url}
	for _, opt := range opts {
		opt(req)
	}
	return s.client.Navigate(ctx, s.id, s.executionID, s.leaseID, req)
}

// AuthorizeAction checks an operation against the session's injected grant
// authority. The authority owns receipt validation and revocation state.
// For non-navigation operations BAS resolves the selected page URL under this
// same lease before asking the authority to make the decision.
func (s *Session) AuthorizeAction(ctx context.Context, kind, targetURL, approvalReceipt string) error {
	return s.AuthorizeActionRequest(ctx, sessionpermission.ActionRequest{Kind: kind, TargetURL: targetURL, ApprovalReceipt: approvalReceipt})
}

// AuthorizeActionRequest checks one logical operation under the immutable
// session owner. Identity and current URL are resolved here; caller-provided
// values for those fields are ignored.
func (s *Session) AuthorizeActionRequest(ctx context.Context, request sessionpermission.ActionRequest) error {
	return s.authorizeActionRequestForPage(ctx, request, "")
}

// authorizeActionRequestForPage binds observation authorization to the exact
// selected page rather than implicitly using whichever page was most recent.
func (s *Session) authorizeActionRequestForPage(ctx context.Context, request sessionpermission.ActionRequest, expectedPageID string) error {
	s.mu.RLock()
	authorizer := s.authorizer
	callerResolver := s.callerResolver
	expectedCallerID := s.expectedCallerID
	s.mu.RUnlock()
	if authorizer == nil {
		return nil
	}
	if callerResolver == nil {
		return errors.New("browser permission denied: trusted caller resolver unavailable")
	}
	callerID, err := callerResolver.BrowserCallerID(ctx)
	if err != nil || callerID == "" {
		return errors.New("browser permission denied: trusted caller identity unavailable")
	}
	if expectedCallerID == "" {
		return errors.New("browser permission denied: trusted expected caller binding unavailable")
	}
	if callerID != expectedCallerID {
		return errors.New("browser permission denied: session belongs to a different trusted caller")
	}
	currentURL := ""
	if request.Kind != "navigate" && request.Kind != "close" {
		if expectedPageID == "" {
			expectedPageID = s.currentDriverPageID()
		}
		state, err := s.client.GetNavigationState(ctx, s.id, s.executionID, s.leaseID, expectedPageID)
		if err != nil {
			return fmt.Errorf("resolve current browser URL before authorization: %w", err)
		}
		currentURL = state.URL
	}
	return authorizer.Authorize(ctx, sessionpermission.ActionRequest{
		SessionID: s.id, LeaseID: s.leaseID, OwnerID: s.executionID,
		CallerID: callerID, Kind: request.Kind,
		TargetURL: request.TargetURL, CurrentURL: currentURL,
		ApprovalReceipt: request.ApprovalReceipt, CredentialID: request.CredentialID,
		ExtensionID: request.ExtensionID, ExtensionPermission: request.ExtensionPermission,
		TabID: request.TabID, ConsentReceipt: request.ConsentReceipt,
	})
}

func (s *Session) authorizeInstruction(ctx context.Context, instr contracts.CompiledInstruction) error {
	s.mu.RLock()
	authorizer := s.authorizer
	s.mu.RUnlock()
	if authorizer == nil {
		return nil
	}
	if instr.Action == nil {
		return errors.New("browser permission denied: missing typed action")
	}
	if input := instr.Action.GetInput(); input != nil && input.GetIsSensitive() {
		return errors.New("browser permission denied: raw sensitive input is unsupported; use the credential authority reference flow")
	}
	kind := permissionActionKind(instr.Action.GetType())
	target := ""
	if nav := instr.Action.GetNavigate(); nav != nil {
		target = nav.GetUrl()
	}
	return s.AuthorizeAction(ctx, kind, target, instr.Metadata["authorization_receipt"])
}

func permissionActionKind(action basactions.ActionType) string {
	switch action {
	case basactions.ActionType_ACTION_TYPE_NAVIGATE:
		return "navigate"
	case basactions.ActionType_ACTION_TYPE_CLICK:
		return "click"
	case basactions.ActionType_ACTION_TYPE_INPUT:
		return "input"
	case basactions.ActionType_ACTION_TYPE_ASSERT:
		return "assert"
	case basactions.ActionType_ACTION_TYPE_EXTRACT:
		return "extract"
	case basactions.ActionType_ACTION_TYPE_SCREENSHOT:
		return "screenshot"
	case basactions.ActionType_ACTION_TYPE_EVALUATE:
		return "evaluate"
	default:
		return "action"
	}
}

// NavigateHistory applies reload/back/forward under this Session's immutable lease.
func (s *Session) NavigateHistory(ctx context.Context, operation driver.HistoryNavigation, req *driver.HistoryNavigationRequest) (*driver.HistoryNavigationResponse, error) {
	if s.isClosed() {
		return nil, errors.New("session closed")
	}
	if s.hasAuthorizer() {
		if req == nil {
			req = &driver.HistoryNavigationRequest{}
		}
		pageID := req.ExpectedPageID
		if pageID == "" {
			pageID = s.currentDriverPageID()
		}
		target := ""
		switch operation {
		case driver.HistoryReload:
			state, err := s.client.GetNavigationState(ctx, s.id, s.executionID, s.leaseID, pageID)
			if err != nil {
				return nil, err
			}
			target = state.URL
		case driver.HistoryBack, driver.HistoryForward:
			stack, err := s.GetNavigationStack(ctx, pageID)
			if err != nil {
				return nil, err
			}
			if operation == driver.HistoryBack && len(stack.BackStack) > 0 {
				target = stack.BackStack[0].URL
			}
			if operation == driver.HistoryForward && len(stack.ForwardStack) > 0 {
				target = stack.ForwardStack[0].URL
			}
		default:
			return nil, errors.New("unsupported history navigation")
		}
		if target == "" {
			return nil, errors.New("history destination unavailable for permission check")
		}
		if err := s.AuthorizeAction(ctx, "navigate", target, ""); err != nil {
			return nil, err
		}
	}
	return s.client.NavigateHistory(ctx, s.id, s.executionID, s.leaseID, operation, req)
}

// UpdateViewport updates the viewport dimensions.
func (s *Session) UpdateViewport(ctx context.Context, width, height int, expectedPageID string) (*driver.UpdateViewportResponse, error) {
	if s.isClosed() {
		return nil, errors.New("session closed")
	}
	if err := s.AuthorizeAction(ctx, "viewport_update", "", ""); err != nil {
		return nil, err
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
	if err := s.AuthorizeAction(ctx, "screenshot", "", ""); err != nil {
		return nil, err
	}
	resp, err := s.client.CaptureScreenshot(ctx, s.id, s.executionID, s.leaseID, &driver.CaptureScreenshotRequest{
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
	if err := s.AuthorizeAction(ctx, "user_state_export", "", ""); err != nil {
		return nil, err
	}
	return s.client.GetStorageState(ctx, s.id, s.executionID, s.leaseID)
}

// GetServiceWorkers retrieves the service workers for this session.
func (s *Session) GetServiceWorkers(ctx context.Context) (*driver.GetServiceWorkersResponse, error) {
	if s.isClosed() {
		return nil, errors.New("session closed")
	}
	if err := s.AuthorizeAction(ctx, "observe", "", ""); err != nil {
		return nil, err
	}
	return s.client.GetServiceWorkers(ctx, s.id, s.executionID, s.leaseID)
}

// UnregisterAllServiceWorkers unregisters all service workers for this session.
func (s *Session) UnregisterAllServiceWorkers(ctx context.Context) (*driver.UnregisterServiceWorkersResponse, error) {
	if s.isClosed() {
		return nil, errors.New("session closed")
	}
	workers, err := s.GetServiceWorkers(ctx)
	if err != nil {
		return nil, err
	}
	if len(workers.Workers) == 0 {
		if err := s.authorizeServiceWorkerUnregister(ctx, ""); err != nil {
			return nil, err
		}
	}
	for _, worker := range workers.Workers {
		if err := s.authorizeServiceWorkerUnregister(ctx, worker.ScopeURL); err != nil {
			return nil, err
		}
	}
	return s.client.UnregisterAllServiceWorkers(ctx, s.id, s.executionID, s.leaseID)
}

// UnregisterServiceWorker unregisters a specific service worker by scope URL.
func (s *Session) UnregisterServiceWorker(ctx context.Context, scopeURL string) (*driver.UnregisterServiceWorkerResponse, error) {
	if s.isClosed() {
		return nil, errors.New("session closed")
	}
	if err := s.authorizeServiceWorkerUnregister(ctx, scopeURL); err != nil {
		return nil, err
	}
	return s.client.UnregisterServiceWorker(ctx, s.id, scopeURL, s.executionID, s.leaseID)
}

// authorizeServiceWorkerUnregister binds deletion to both the current owned
// page and the exact registration scope. Bulk deletion calls this for every
// registered scope before dispatching a single driver mutation.
func (s *Session) authorizeServiceWorkerUnregister(ctx context.Context, scopeURL string) error {
	pageID := s.currentDriverPageID()
	kind := "service_worker_unregister"
	if scopeURL == "" {
		kind = "service_worker_unregister_all"
	}
	if err := s.authorizeActionRequestForPage(ctx, sessionpermission.ActionRequest{
		Kind: kind, TargetURL: scopeURL,
	}, pageID); err != nil {
		return err
	}
	return nil
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
	if err := s.AuthorizeAction(ctx, "observe", "", ""); err != nil {
		return nil, err
	}
	return s.client.ValidateSelector(ctx, s.id, s.executionID, s.leaseID, &driver.ValidateSelectorRequest{
		Selector: selector,
	})
}

// UpdateStreamSettings updates the frame streaming settings.
func (s *Session) UpdateStreamSettings(ctx context.Context, quality, fps *int, scale string) error {
	if s.isClosed() {
		return errors.New("session closed")
	}
	if err := s.AuthorizeAction(ctx, "control", "", ""); err != nil {
		return err
	}
	_, err := s.client.UpdateStreamSettings(ctx, s.id, s.executionID, s.leaseID, &driver.UpdateStreamSettingsRequest{
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
	if err := s.AuthorizeAction(ctx, "navigate", url, ""); err != nil {
		return nil, err
	}
	return s.client.CreatePage(ctx, s.id, s.executionID, s.leaseID, url)
}

// CreatePopupPage creates a page through an owned opener page.
func (s *Session) CreatePopupPage(ctx context.Context, url, openerPageID string) (*driver.CreatePageResponse, error) {
	if s.isClosed() {
		return nil, errors.New("session closed")
	}
	if err := s.AuthorizeAction(ctx, "navigate", url, ""); err != nil {
		return nil, err
	}
	return s.client.CreatePopupPage(ctx, s.id, s.executionID, s.leaseID, url, openerPageID)
}

// ClosePage closes a browser tab under this Session's immutable lease.
func (s *Session) ClosePage(ctx context.Context, driverPageID string) (*driver.ClosePageResponse, error) {
	if s.isClosed() {
		return nil, errors.New("session closed")
	}
	if err := s.AuthorizeAction(ctx, "close_page", "", ""); err != nil {
		return nil, err
	}
	return s.client.ClosePage(ctx, s.id, s.executionID, s.leaseID, driverPageID)
}

// InitialDriverPageID returns the browser-owned identity of the page created
// with this session. It remains stable for that session lifetime and is used
// only for selecting a restored active tab, never persisted across sessions.
func (s *Session) InitialDriverPageID() string {
	if s == nil {
		return ""
	}
	return s.initialDriverPageID
}

// SetActivePage switches the active page for execution.
// The driverPageID is the Playwright driver's internal identifier for the page.
// This is used during multi-page playback to execute actions on the correct page.
func (s *Session) SetActivePage(ctx context.Context, driverPageID string) error {
	if s.isClosed() {
		return errors.New("session closed")
	}
	if err := s.AuthorizeAction(ctx, "view", "", ""); err != nil {
		return err
	}
	if err := s.client.SetActivePage(ctx, s.id, s.executionID, s.leaseID, driverPageID); err != nil {
		return err
	}
	s.mu.Lock()
	s.activeDriverPageID = driverPageID
	s.mu.Unlock()
	return nil
}

func (s *Session) currentDriverPageID() string {
	s.mu.RLock()
	pageID := s.activeDriverPageID
	if pageID == "" {
		pageID = s.initialDriverPageID
	}
	s.mu.RUnlock()
	return pageID
}

func (s *Session) hasAuthorizer() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.authorizer != nil
}

// Reset resets the session to clean state.
func (s *Session) Reset(ctx context.Context) error {
	if s.isClosed() {
		return errors.New("session closed")
	}
	if err := s.AuthorizeAction(ctx, "reset", "", ""); err != nil {
		return err
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
	if err := s.AuthorizeAction(ctx, "handoff", "", ""); err != nil {
		return err
	}
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
	if !release {
		if err := s.AuthorizeAction(ctx, "close", "", ""); err != nil {
			return nil, err
		}
	}
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

// OwnerIdentity returns the immutable broker identity admitted for this
// session. Callers use it to bind delegated read/control capabilities to this
// exact execution lease; neither value is accepted from browser input.
func (s *Session) OwnerIdentity() (executionID, leaseID string) {
	if s == nil {
		return "", ""
	}
	return s.executionID, s.leaseID
}

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
