package livecapture

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	coredb "github.com/vrooli/api-core/database"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/automation/session"
	"github.com/vrooli/browser-automation-studio/config"
	"github.com/vrooli/browser-automation-studio/domain"
	unifiedrecording "github.com/vrooli/browser-automation-studio/services/recording"
	"github.com/vrooli/browser-automation-studio/services/recording/persistence"
	sessionprofilepersistence "github.com/vrooli/browser-automation-studio/services/session-profile/persistence"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
)

// Service provides high-level operations for live capture mode.
// It orchestrates the session manager and workflow generator.
//
// DOC: docs/architecture/recording.md#service-layer
type Service struct {
	sessions  *session.Manager
	generator *WorkflowGenerator
	log       *logrus.Logger

	// Unified recording service for action and page event persistence.
	// All recorded actions (manual and AI) flow through this service.
	// DOC: docs/architecture/recording.md#unified-recording
	unifiedRecordingSvc *unifiedrecording.Service
}

// NewService creates a new live capture service.
func NewService(log *logrus.Logger, unifiedRecordingSvc *unifiedrecording.Service) *Service {
	mgr, err := session.NewManager(session.WithLogger(log))
	if err != nil {
		log.WithError(err).Warn("Failed to create session manager, service will fail on first use")
		return &Service{
			sessions:            nil,
			generator:           NewWorkflowGenerator(),
			log:                 log,
			unifiedRecordingSvc: unifiedRecordingSvc,
		}
	}
	return &Service{
		sessions:            mgr,
		generator:           NewWorkflowGenerator(),
		log:                 log,
		unifiedRecordingSvc: unifiedRecordingSvc,
	}
}

// NewServiceWithManager creates a service with a custom session manager (for testing).
func NewServiceWithManager(mgr *session.Manager, log *logrus.Logger, unifiedRecordingSvc *unifiedrecording.Service) *Service {
	return &Service{
		sessions:            mgr,
		generator:           NewWorkflowGenerator(),
		log:                 log,
		unifiedRecordingSvc: unifiedRecordingSvc,
	}
}

// NewServiceWithClient creates a service with a custom driver client (for testing).
// Deprecated: Use NewServiceWithManager instead.
func NewServiceWithClient(client *driver.Client, log *logrus.Logger, unifiedRecordingSvc *unifiedrecording.Service) *Service {
	return &Service{
		sessions:            session.NewManagerWithClient(client, session.WithLogger(log)),
		generator:           NewWorkflowGenerator(),
		log:                 log,
		unifiedRecordingSvc: unifiedRecordingSvc,
	}
}

// UnifiedRecordingService returns the unified recording service.
// DOC: docs/architecture/recording.md#unified-recording
func (s *Service) UnifiedRecordingService() *unifiedrecording.Service {
	return s.unifiedRecordingSvc
}

// DriverClient returns the underlying driver client for direct pass-through operations.
// Handlers should use this for operations that don't require service-level business logic
// (e.g., Navigate, Reload, GetFrame, ForwardInput).
func (s *Service) DriverClient() driver.ClientInterface {
	if s.sessions == nil {
		return nil
	}
	return s.sessions.Client()
}

// Sessions returns the session manager for direct access.
// Deprecated: Use DriverClient() for pass-through operations.
func (s *Service) Sessions() *session.Manager {
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
	BrowserProfile *sessionprofilepersistence.BrowserProfile // Anti-detection and behavior settings
}

// SessionResult is the result of creating a session.
type SessionResult struct {
	// Close releases this exact admission, even if its public ID is later reused.
	Close          func(context.Context) error
	SessionID      string
	CreatedAt      time.Time
	ActualViewport *ViewportDimensions // Actual viewport from Playwright (may differ due to profile)
	// InitialNavigation contains info about the initial URL navigation, if one occurred.
	// Used for capturing history entries in the handler.
	InitialNavigation *HistoryEntryInfo
}

// ViewportDimensions represents width and height of a viewport with source attribution.
type ViewportDimensions struct {
	Width  int
	Height int
	Source string // "requested", "fingerprint", "fingerprint_partial", or "default"
	Reason string // Human-readable explanation of what determined the dimensions
}

// CreateSession creates a new browser session for live capture.
func (s *Service) CreateSession(ctx context.Context, cfg *SessionConfig) (result *SessionResult, err error) {
	if s.sessions == nil {
		return nil, fmt.Errorf("session manager not initialized")
	}
	if s.unifiedRecordingSvc == nil {
		return nil, unifiedrecording.ErrRepositoryUnavailable
	}

	// Get defaults from config - this is the single source of truth for defaults
	appCfg := config.Load()

	// Set defaults for frame streaming, using config values
	quality := cfg.StreamQuality
	if quality <= 0 || quality > 100 {
		quality = appCfg.Recording.DefaultStreamQuality
		if quality <= 0 || quality > 100 {
			quality = 55 // Ultimate fallback
		}
	}
	fps := cfg.StreamFPS
	if fps <= 0 || fps > 60 {
		fps = appCfg.Recording.DefaultStreamFPS
		if fps <= 0 || fps > 60 {
			fps = 30 // Ultimate fallback
		}
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
		BrowserProfile: cfg.BrowserProfile,
	}

	// Inject recording callbacks that route to the unified recording service.
	// This ensures all browser actions (manual, AI, or playback) are captured
	// through a single recording pipeline.
	// DOC: docs/architecture/recording.md#unified-recording
	spec.Recording = s.buildRecordingCallbacks()

	sess, err := s.sessions.Create(ctx, spec)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	// Every failure after browser admission releases that exact Session. A failed
	// cleanup remains joined to the original error and retains retry ownership.
	defer func() {
		if err == nil {
			return
		}
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		err = errors.Join(err, sess.Close(cleanupCtx))
	}()

	// Register session with unified recording service for timeline persistence.
	// This ensures the recording_sessions table entry exists before any actions
	// are recorded, satisfying the foreign key constraint on timeline_entries.
	regCfg := unifiedrecording.SessionConfig{
		ViewportWidth:  cfg.ViewportWidth,
		ViewportHeight: cfg.ViewportHeight,
	}
	if err := s.unifiedRecordingSvc.RegisterSession(ctx, sess.ID(), regCfg); err != nil {
		return nil, fmt.Errorf("register recording journal: %w", err)
	}

	// Initialize page tracking for multi-tab support
	sess.InitializePageTracking(cfg.InitialURL)

	var initialNavigation *HistoryEntryInfo
	if cfg.InitialURL != "" {
		initialNavigation, err = s.navigateInitialPage(ctx, sess, cfg.InitialURL)
		if err != nil {
			return nil, fmt.Errorf("navigate initial page: %w", err)
		}
	}

	// Get actual viewport from Playwright (may differ from requested due to profile)
	var actualViewport *ViewportDimensions
	if av := sess.ActualViewport(); av != nil {
		actualViewport = &ViewportDimensions{
			Width:  av.Width,
			Height: av.Height,
			Source: string(av.Source),
			Reason: av.Reason,
		}
	}

	return &SessionResult{
		Close:             sess.Close,
		SessionID:         sess.ID(),
		CreatedAt:         time.Now().UTC(),
		ActualViewport:    actualViewport,
		InitialNavigation: initialNavigation,
	}, nil
}

// navigateInitialPage gives fresh admission and saved-tab restoration one receipt policy.
func (s *Service) navigateInitialPage(ctx context.Context, owner *session.Session, url string) (*HistoryEntryInfo, error) {
	pages := owner.Pages()
	if pages == nil {
		return nil, fmt.Errorf("initial page tracking unavailable: %s", owner.ID())
	}
	initialID := pages.GetInitialPageID()
	resp, err := owner.Navigate(ctx, url)
	if err != nil {
		return nil, err
	}
	current, ok := s.sessions.Get(owner.ID())
	if !ok || current != owner || owner.Pages() != pages {
		return nil, fmt.Errorf("session ownership changed during initial navigation: %s", owner.ID())
	}
	pageID := pages.GetPageIDByDriverID(resp.DriverPageID)
	if pageID == nil || *pageID != initialID {
		return nil, fmt.Errorf("initial navigation returned a different page: %s", resp.DriverPageID)
	}
	pages.UpdatePageInfo(initialID, resp.URL, resp.Title, resp.FaviconURL)
	return &HistoryEntryInfo{URL: resp.URL, Title: resp.Title}, nil
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

// GetServiceWorkers retrieves service workers for a session.
func (s *Service) GetServiceWorkers(ctx context.Context, sessionID string) (*driver.GetServiceWorkersResponse, error) {
	sess, ok := s.sessions.Get(sessionID)
	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	return sess.GetServiceWorkers(ctx)
}

// UnregisterAllServiceWorkers unregisters all service workers for a session.
func (s *Service) UnregisterAllServiceWorkers(ctx context.Context, sessionID string) (*driver.UnregisterServiceWorkersResponse, error) {
	sess, ok := s.sessions.Get(sessionID)
	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	return sess.UnregisterAllServiceWorkers(ctx)
}

// UnregisterServiceWorker unregisters a specific service worker by scope URL.
func (s *Service) UnregisterServiceWorker(ctx context.Context, sessionID, scopeURL string) (*driver.UnregisterServiceWorkerResponse, error) {
	sess, ok := s.sessions.Get(sessionID)
	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	return sess.UnregisterServiceWorker(ctx, scopeURL)
}

// RecordingConfig configures recording start.
type RecordingConfig struct {
	APIHost string
	APIPort string
	// FrameQuality is the JPEG quality for frame streaming (1-100).
	// Default: uses BAS_RECORDING_DEFAULT_STREAM_QUALITY from config (default 55).
	FrameQuality int
	// FrameFPS is the target frames per second for frame streaming.
	// Default: uses BAS_RECORDING_DEFAULT_STREAM_FPS from config (default 30).
	// Note: For CDP screencast (Chromium), Chrome controls actual FPS.
	// For polling strategy (Firefox/WebKit), this controls capture interval.
	FrameFPS int
}

// StartRecording starts recording user actions.
func (s *Service) StartRecording(ctx context.Context, sessionID string, cfg *RecordingConfig) (*driver.StartRecordingResponse, error) {
	owned, ok := s.sessions.Get(sessionID)
	if !ok {
		return nil, &driver.Error{Status: http.StatusNotFound, Message: "Recording session is not owned by this API"}
	}

	apiHost := cfg.APIHost
	if apiHost == "" {
		apiHost = "127.0.0.1"
	}
	apiPort := cfg.APIPort
	if apiPort == "" {
		apiPort = "8080"
	}

	// Get defaults from config - single source of truth
	appCfg := config.Load()

	// Apply defaults for optional config values, using config system
	frameQuality := cfg.FrameQuality
	if frameQuality <= 0 {
		frameQuality = appCfg.Recording.DefaultStreamQuality
		if frameQuality <= 0 {
			frameQuality = 55 // Ultimate fallback
		}
	}
	frameFPS := cfg.FrameFPS
	if frameFPS <= 0 {
		frameFPS = appCfg.Recording.DefaultStreamFPS
		if frameFPS <= 0 {
			frameFPS = 30 // Ultimate fallback
		}
	}

	req := &driver.StartRecordingRequest{
		CallbackURL:      fmt.Sprintf("http://%s:%s/api/v1/recordings/live/%s/action", apiHost, apiPort, sessionID),
		FrameCallbackURL: fmt.Sprintf("http://%s:%s/api/v1/recordings/live/%s/frame", apiHost, apiPort, sessionID),
		PageCallbackURL:  fmt.Sprintf("http://%s:%s/api/v1/recordings/live/%s/page-event", apiHost, apiPort, sessionID),
		RoutedTestMode:   coredb.IsTestMode(ctx),
		FrameQuality:     frameQuality,
		FrameFPS:         frameFPS,
	}

	return owned.StartRecording(ctx, req)
}

// StopRecording uses the same owned session as recording start.
func (s *Service) StopRecording(ctx context.Context, sessionID string) (*driver.StopRecordingResponse, error) {
	owned, ok := s.sessions.Get(sessionID)
	if !ok {
		return nil, &driver.Error{Status: http.StatusNotFound, Message: "Recording session is not owned by this API"}
	}
	return owned.StopRecording(ctx)
}

// ForwardInput shares session ownership across HTTP and WebSocket transports.
func (s *Service) ForwardInput(ctx context.Context, sessionID string, input []byte) (*driver.ForwardInputResponse, error) {
	owned, ok := s.sessions.Get(sessionID)
	if !ok {
		return nil, &driver.Error{Status: http.StatusNotFound, Message: "Recording session is not owned by this API"}
	}
	return owned.ForwardInput(ctx, input)
}

// GenerateWorkflowConfig configures workflow generation.
type GenerateWorkflowConfig struct {
	Name        string
	Actions     []driver.RecordedAction
	ActionRange *ActionRange
}

// ActionRange specifies a subset of actions to use.
type ActionRange struct {
	Start int
	End   int
}

// GenerateWorkflowResult is the result of workflow generation.
type GenerateWorkflowResult struct {
	FlowDefinition *basworkflows.WorkflowDefinitionV2
	NodeCount      int
	ActionCount    int
}

// GenerateWorkflow converts recorded actions to a workflow definition.
func (s *Service) GenerateWorkflow(ctx context.Context, sessionID string, cfg *GenerateWorkflowConfig) (*GenerateWorkflowResult, error) {
	var actions []driver.RecordedAction

	// Use provided actions or fetch from session
	if len(cfg.Actions) > 0 {
		actions = cfg.Actions
	} else {
		resp, err := s.sessions.Client().GetRecordedActions(ctx, sessionID)
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

	// Resolve logical targets from the session's existing tracker. Generator
	// callers without an owned session remain single-page and fail closed for
	// ambiguous multi-page action input.
	var pages []*domain.Page
	if s.sessions != nil {
		if sess, ok := s.sessions.Get(sessionID); ok && sess.Pages() != nil {
			pages, _ = sess.Pages().Snapshot(false)
		}
	}

	// Generate workflow
	flowDef, err := s.generator.GenerateWorkflowWithPages(actions, pages)
	if err != nil {
		return nil, fmt.Errorf("generate workflow: %w", err)
	}

	return &GenerateWorkflowResult{
		FlowDefinition: flowDef,
		NodeCount:      len(flowDef.Nodes),
		ActionCount:    len(actions),
	}, nil
}

// GetSession returns the session by ID.
func (s *Service) GetSession(sessionID string) (*session.Session, bool) {
	return s.sessions.Get(sessionID)
}

// GetPages returns all pages for a session.
func (s *Service) GetPages(sessionID string) (*PageListResult, error) {
	sess, ok := s.sessions.Get(sessionID)
	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	pages := sess.Pages()
	if pages == nil {
		return nil, fmt.Errorf("page tracking not initialized for session: %s", sessionID)
	}

	list, active := pages.Snapshot(false)
	result := &PageListResult{Pages: list}
	if active != uuid.Nil {
		result.ActivePageID = active.String()
	}
	return result, nil
}

// PageListResult contains the list of pages and active page ID.
type PageListResult struct {
	Pages        []*domain.Page
	ActivePageID string
}

// GetOpenPages returns only the open (non-closed) pages for a session.
// This is used for capturing tab state before session close.
func (s *Service) GetOpenPages(sessionID string) ([]*domain.Page, uuid.UUID, error) {
	sess, ok := s.sessions.Get(sessionID)
	if !ok {
		return nil, uuid.Nil, fmt.Errorf("session not found: %s", sessionID)
	}

	pages := sess.Pages()
	if pages == nil {
		return nil, uuid.Nil, fmt.Errorf("page tracking not initialized for session: %s", sessionID)
	}

	list, active := pages.Snapshot(true)
	return list, active, nil
}

// PageCloseResult carries the browser closure observation and resulting selection.
type PageCloseResult struct {
	Event        *domain.PageEvent
	ActivePageID string
}

// ClosePage closes the admitted browser tab before committing the API observation.
func (s *Service) ClosePage(ctx context.Context, sessionID string, pageID uuid.UUID) (*PageCloseResult, error) {
	sess, ok := s.sessions.Get(sessionID)
	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	pages := sess.Pages()
	if pages == nil {
		return nil, fmt.Errorf("page tracking not initialized for session: %s", sessionID)
	}
	page, ok := pages.GetPage(pageID)
	if !ok {
		return nil, fmt.Errorf("page %s not found", pageID)
	}
	var selected *uuid.UUID
	if page.Status != domain.PageStatusClosed {
		receipt, err := sess.ClosePage(ctx, page.DriverPageID)
		if err != nil {
			return nil, err
		}
		current, ok := s.sessions.Get(sessionID)
		if !ok || current != sess || sess.Pages() != pages {
			return nil, errors.New("session ownership changed while closing page")
		}
		if receipt.ActivePageID != "" {
			selected = pages.GetPageIDByDriverID(receipt.ActivePageID)
			if selected == nil {
				return nil, errors.New("browser close receipt selected an unknown page")
			}
		} else {
			open, _ := pages.Snapshot(true)
			for _, remaining := range open {
				if remaining.ID != pageID {
					return nil, errors.New("browser close receipt omitted the remaining selection")
				}
			}
		}
	}
	event, err := pages.ClosePage(pageID)
	if err != nil {
		return nil, err
	}
	if selected != nil {
		if err := pages.SetActivePage(*selected); err != nil {
			return nil, err
		}
	}
	result := &PageCloseResult{Event: event}
	if active := pages.GetActivePageID(); active != uuid.Nil {
		result.ActivePageID = active.String()
	}
	return result, nil
}

// ActivatePage switches the active page for a session.
func (s *Service) ActivatePage(ctx context.Context, sessionID string, pageID uuid.UUID) error {
	sess, ok := s.sessions.Get(sessionID)
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	return s.activatePage(ctx, sess, pageID)
}

// activatePage retains the transaction's original owner across the driver await.
func (s *Service) activatePage(ctx context.Context, sess *session.Session, pageID uuid.UUID) error {
	sessionID := sess.ID()
	pages := sess.Pages()
	if pages == nil {
		return fmt.Errorf("page tracking not initialized for session: %s", sessionID)
	}

	// Verify page exists and is open
	page, ok := pages.GetPage(pageID)
	if !ok {
		return fmt.Errorf("page not found: %s", pageID)
	}
	if page.Status != domain.PageStatusActive {
		return fmt.Errorf("page is closed: %s", pageID)
	}

	// Get driver page ID for switching
	driverPageID := pages.GetDriverPageID(pageID)
	if driverPageID == "" {
		return fmt.Errorf("page not registered with driver: %s", pageID)
	}

	// Tell driver to switch active page
	if err := sess.SetActivePage(ctx, driverPageID); err != nil {
		return fmt.Errorf("failed to switch page in driver: %w", err)
	}

	current, ok := s.sessions.Get(sessionID)
	if !ok || current != sess {
		return fmt.Errorf("browser page switched but session ownership changed: %s", sessionID)
	}
	return pages.SetActivePage(pageID)
}

// CreatePage creates a new page (tab) in the browser session.
func (s *Service) CreatePage(ctx context.Context, sessionID string, url string) (*domain.Page, error) {
	owner, ok := s.sessions.Get(sessionID)
	if !ok || owner.Pages() == nil {
		return nil, fmt.Errorf("recording session unavailable: %s", sessionID)
	}
	return s.createPage(ctx, owner, url)
}

// createPage shares receipt registration without re-admitting a restored transaction by ID.
func (s *Service) createPage(ctx context.Context, owner *session.Session, url string) (*domain.Page, error) {
	sessionID := owner.ID()
	pages := owner.Pages()
	current, ok := s.sessions.Get(sessionID)
	if !ok || current != owner || pages == nil || owner.Pages() != pages {
		return nil, fmt.Errorf("session ownership changed before page creation: %s", sessionID)
	}
	result, err := owner.CreatePage(ctx, url)
	if err != nil {
		return nil, err
	}
	current, ok = s.sessions.Get(sessionID)
	if !ok || current != owner || owner.Pages() != pages {
		return nil, fmt.Errorf("browser page created but session ownership changed: %s", sessionID)
	}
	page := pages.AddPage(&domain.Page{DriverPageID: result.DriverPageID, URL: result.URL, Title: result.Title})
	pages.UpdatePageInfo(page.ID, result.URL, result.Title, result.FaviconURL)
	if err := pages.SetActivePage(page.ID); err != nil {
		return nil, err
	}
	page, _ = pages.GetPage(page.ID)
	return page, nil
}

// RestoredTab contains info about a tab that was restored.
type RestoredTab struct {
	PageID   string
	URL      string
	Title    string
	IsActive bool
}

// HistoryEntryInfo contains minimal info needed to create a history entry.
// This is returned from service methods so handlers can add entries to session profiles.
type HistoryEntryInfo struct {
	URL   string
	Title string
}

// TabRestorationResult contains the results of tab restoration.
type TabRestorationResult struct {
	// InitialURL is the URL the initial tab was navigated to (first tab in the list).
	// Empty if no navigation was performed (e.g., first tab was about:blank).
	InitialURL string
	// InitialTitle is the page title from the initial navigation.
	InitialTitle string
	// Tabs contains info about additional tabs that were created (not including the initial tab).
	Tabs []RestoredTab
	// HistoryEntries contains all navigations for history capture.
	// This includes the initial navigation and all restored tabs.
	HistoryEntries []HistoryEntryInfo
}

// RestoreTabs creates tabs from saved tab state.
// Returns info about the tabs that were restored, including the initial URL.
func (s *Service) RestoreTabs(ctx context.Context, sessionID string, tabs []sessionprofilepersistence.TabState) (*TabRestorationResult, error) {
	sess, ok := s.sessions.Get(sessionID)
	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	if len(tabs) == 0 {
		return nil, nil
	}

	pages := sess.Pages()
	if pages == nil {
		return nil, fmt.Errorf("page tracking not initialized for session: %s", sessionID)
	}
	result := &TabRestorationResult{
		Tabs:           make([]RestoredTab, 0, len(tabs)),
		HistoryEntries: make([]HistoryEntryInfo, 0, len(tabs)),
	}
	var activePageID uuid.UUID
	initialID := pages.GetInitialPageID()
	if tabs[0].IsActive {
		activePageID = initialID
	}

	if url := tabs[0].URL; url != "" && url != "about:blank" {
		initial, err := s.navigateInitialPage(ctx, sess, url)
		if err != nil {
			return result, fmt.Errorf("failed to restore initial tab: %w", err)
		}
		result.InitialURL, result.InitialTitle = initial.URL, initial.Title
		result.HistoryEntries = append(result.HistoryEntries, *initial)
	}

	for _, tab := range tabs[1:] {
		// Keep the existing policy of omitting additional empty tabs.
		if tab.URL == "" || tab.URL == "about:blank" {
			continue
		}
		resp, err := s.createPage(ctx, sess, tab.URL)
		if err != nil {
			return result, fmt.Errorf("failed to restore tab at order %d: %w", tab.Order, err)
		}
		result.Tabs = append(result.Tabs, RestoredTab{
			PageID: resp.DriverPageID, URL: resp.URL, Title: resp.Title, IsActive: tab.IsActive,
		})
		result.HistoryEntries = append(result.HistoryEntries, HistoryEntryInfo{URL: resp.URL, Title: resp.Title})
		if tab.IsActive {
			activePageID = resp.ID
		}
	}

	// Every new page becomes selected. Restore the saved selection only after
	// creation, through the same owner that updates browser and API state.
	if activePageID != uuid.Nil && activePageID != pages.GetActivePageID() {
		if err := s.activatePage(ctx, sess, activePageID); err != nil {
			return result, fmt.Errorf("failed to switch to restored active page: %w", err)
		}
	}

	return result, nil
}

// AddTimelineAction adds a recorded action to the timeline via the unified recording service.
// DOC: docs/architecture/recording.md#data-flow
func (s *Service) AddTimelineAction(ctx context.Context, sessionID string, action *driver.RecordedAction, pageID uuid.UUID) error {
	if s.unifiedRecordingSvc == nil {
		return unifiedrecording.ErrRepositoryUnavailable
	}
	source := unifiedrecording.ActionSourceManual
	if action != nil && action.Payload != nil && action.Payload["source"] == "ai" {
		source = unifiedrecording.ActionSourceAI
	}
	return s.unifiedRecordingSvc.RecordAction(ctx, sessionID, action, pageID, source)
}

func (s *Service) AddTimelinePageEvent(ctx context.Context, sessionID string, event *domain.PageEvent) error {
	if s.unifiedRecordingSvc == nil {
		return unifiedrecording.ErrRepositoryUnavailable
	}
	return s.unifiedRecordingSvc.RecordPageEvent(ctx, sessionID, event)
}

// GetTimeline returns the unified timeline for a session.
func (s *Service) GetTimeline(ctx context.Context, sessionID string, pageID *uuid.UUID, limit, offset int) (*domain.TimelineResponse, error) {
	if _, ok := s.sessions.Get(sessionID); !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	if s.unifiedRecordingSvc == nil {
		return nil, unifiedrecording.ErrRepositoryUnavailable
	}

	// Build query for unified recording service
	query := persistence.TimelineQuery{
		SessionID: sessionID,
		PageID:    pageID,
		Limit:     limit,
		Offset:    offset,
	}

	resp, err := s.unifiedRecordingSvc.GetTimeline(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("get timeline: %w", err)
	}

	// Convert unified entries to domain entries
	entries := make([]domain.TimelineEntry, 0, len(resp.Entries))
	for _, e := range resp.Entries {
		entry := domain.TimelineEntry{
			ID:        e.ID,
			Type:      domain.TimelineType(e.Type),
			Timestamp: e.Timestamp,
			PageID:    e.PageID,
		}
		if e.Action != nil {
			entry.Action = &domain.RecordedActionEntry{
				ID:          e.Action.ID.String(),
				ActionType:  e.Action.ActionType,
				URL:         e.Action.URL,
				SequenceNum: e.Action.SequenceNum,
				Timestamp:   e.Action.Timestamp.Format(time.RFC3339Nano),
				Confidence:  e.Action.Confidence,
				PageTitle:   e.Action.PageTitle,
				Payload:     e.Action.Payload,
			}
			if e.Action.Selector != nil {
				entry.Action.Selector = &domain.SelectorInfo{
					Primary: e.Action.Selector.Primary,
				}
			}
		}
		if e.PageEvent != nil {
			entry.PageEvent = e.PageEvent
		}
		entries = append(entries, entry)
	}

	return &domain.TimelineResponse{
		Entries:      entries,
		HasMore:      resp.HasMore,
		TotalEntries: resp.TotalCount,
	}, nil
}

// buildRecordingCallbacks creates recording callbacks that route to the unified recording service.
// These callbacks are injected into the session spec during session creation.
// DOC: docs/architecture/recording.md#unified-recording
func (s *Service) buildRecordingCallbacks() *session.RecordingCallbacks {
	return &session.RecordingCallbacks{
		OnAction: func(sessionID string, action *session.RecordedActionInfo) {
			// Convert session action info to driver.RecordedAction
			driverAction := &driver.RecordedAction{
				ID:          action.ID,
				SessionID:   sessionID,
				SequenceNum: action.SequenceNum,
				Timestamp:   action.Timestamp,
				ActionType:  action.ActionType,
				URL:         action.URL,
				PageTitle:   action.PageTitle,
				Confidence:  action.Confidence,
				Payload:     action.Payload,
			}
			if action.Selector != "" {
				driverAction.Selector = &driver.SelectorSet{
					Primary: action.Selector,
				}
			}

			// Determine action source
			source := unifiedrecording.ActionSourceManual
			if action.Source == "ai" {
				source = unifiedrecording.ActionSourceAI
			} else if action.Source == "playback" {
				source = unifiedrecording.ActionSourceAuto
			}

			// Record to unified service
			ctx := context.Background()
			if err := s.unifiedRecordingSvc.RecordAction(ctx, sessionID, driverAction, action.PageID, source); err != nil {
				s.log.WithError(err).WithFields(logrus.Fields{
					"session_id":  sessionID,
					"action_type": action.ActionType,
					"source":      source,
				}).Warn("Failed to record action via callback")
			}
		},
		OnPageEvent: func(sessionID string, event *session.PageEventInfo) {
			// Parse timestamp
			ts, err := time.Parse(time.RFC3339Nano, event.Timestamp)
			if err != nil {
				ts = time.Now()
			}

			// Convert to domain.PageEvent
			domainEvent := &domain.PageEvent{
				ID:        uuid.New(),
				Type:      domain.PageEventType(event.Type),
				PageID:    event.PageID,
				URL:       event.URL,
				Title:     event.Title,
				OpenerID:  event.OpenerID,
				Timestamp: ts,
			}

			// Record to unified service
			ctx := context.Background()
			if err := s.unifiedRecordingSvc.RecordPageEvent(ctx, sessionID, domainEvent); err != nil {
				s.log.WithError(err).WithFields(logrus.Fields{
					"session_id": sessionID,
					"event_type": event.Type,
					"page_id":    event.PageID,
				}).Warn("Failed to record page event via callback")
			}
		},
	}
}
