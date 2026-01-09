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
	"github.com/vrooli/browser-automation-studio/config"
	"github.com/vrooli/browser-automation-studio/domain"
	archiveingestion "github.com/vrooli/browser-automation-studio/services/archive-ingestion"
)

// Service provides high-level operations for live capture mode.
// It orchestrates the session manager, workflow generator, and timeline service.
type Service struct {
	sessions  *session.Manager
	generator *WorkflowGenerator
	timeline  *TimelineService
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
			timeline:  NewTimelineService(),
			log:       log,
		}
	}
	return &Service{
		sessions:  mgr,
		generator: NewWorkflowGenerator(),
		timeline:  NewTimelineService(),
		log:       log,
	}
}

// NewServiceWithManager creates a service with a custom session manager (for testing).
func NewServiceWithManager(mgr *session.Manager, log *logrus.Logger) *Service {
	return &Service{
		sessions:  mgr,
		generator: NewWorkflowGenerator(),
		timeline:  NewTimelineService(),
		log:       log,
	}
}

// NewServiceWithClient creates a service with a custom driver client (for testing).
// Deprecated: Use NewServiceWithManager instead.
func NewServiceWithClient(client *driver.Client, log *logrus.Logger) *Service {
	return &Service{
		sessions:  session.NewManagerWithClient(client, session.WithLogger(log)),
		generator: NewWorkflowGenerator(),
		timeline:  NewTimelineService(),
		log:       log,
	}
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
	BrowserProfile *archiveingestion.BrowserProfile // Anti-detection and behavior settings
}

// SessionResult is the result of creating a session.
type SessionResult struct {
	SessionID      string
	CreatedAt      time.Time
	ActualViewport *ViewportDimensions // Actual viewport from Playwright (may differ due to profile)
}

// ViewportDimensions represents width and height of a viewport with source attribution.
type ViewportDimensions struct {
	Width  int
	Height int
	Source string // "requested", "fingerprint", "fingerprint_partial", or "default"
	Reason string // Human-readable explanation of what determined the dimensions
}

// CreateSession creates a new browser session for live capture.
func (s *Service) CreateSession(ctx context.Context, cfg *SessionConfig) (*SessionResult, error) {
	if s.sessions == nil {
		return nil, fmt.Errorf("session manager not initialized")
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

	sess, err := s.sessions.Create(ctx, spec)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	// Initialize page tracking for multi-tab support
	sess.InitializePageTracking(cfg.InitialURL)

	// Navigate to initial URL if provided
	if cfg.InitialURL != "" {
		if _, err := sess.Navigate(ctx, cfg.InitialURL); err != nil {
			s.log.WithError(err).Warn("Failed to navigate to initial URL")
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
		SessionID:      sess.ID(),
		CreatedAt:      time.Now().UTC(),
		ActualViewport: actualViewport,
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
		FrameQuality:     frameQuality,
		FrameFPS:         frameFPS,
	}

	return s.sessions.Client().StartRecording(ctx, sessionID, req)
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
	FlowDefinition map[string]interface{}
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

	return &PageListResult{
		Pages:        pages.ListPages(),
		ActivePageID: pages.GetActivePageID().String(),
	}, nil
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

	return pages.ListOpenPages(), pages.GetActivePageID(), nil
}

// ActivatePage switches the active page for a session.
func (s *Service) ActivatePage(ctx context.Context, sessionID string, pageID uuid.UUID) error {
	sess, ok := s.sessions.Get(sessionID)
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

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
	if err := s.sessions.Client().SetActivePage(ctx, sessionID, driverPageID); err != nil {
		return fmt.Errorf("failed to switch page in driver: %w", err)
	}

	// Update session state
	return pages.SetActivePage(pageID)
}

// CreatePage creates a new page (tab) in the browser session.
func (s *Service) CreatePage(ctx context.Context, sessionID string, url string) (*driver.CreatePageResponse, error) {
	if _, ok := s.sessions.Get(sessionID); !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	// Call driver to create the page
	return s.sessions.Client().CreatePage(ctx, sessionID, url)
}

// RestoredTab contains info about a tab that was restored.
type RestoredTab struct {
	PageID   string
	URL      string
	IsActive bool
}

// TabRestorationResult contains the results of tab restoration.
type TabRestorationResult struct {
	// InitialURL is the URL the initial tab was navigated to (first tab in the list).
	// Empty if no navigation was performed (e.g., first tab was about:blank).
	InitialURL string
	// Tabs contains info about additional tabs that were created (not including the initial tab).
	Tabs []RestoredTab
}

// RestoreTabs creates tabs from saved tab state.
// Returns info about the tabs that were restored, including the initial URL.
func (s *Service) RestoreTabs(ctx context.Context, sessionID string, tabs []archiveingestion.TabState) (*TabRestorationResult, error) {
	if _, ok := s.sessions.Get(sessionID); !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	if len(tabs) == 0 {
		return nil, nil
	}

	s.log.WithFields(map[string]interface{}{
		"session_id": sessionID,
		"tab_count":  len(tabs),
	}).Info("RestoreTabs: starting tab restoration")

	result := &TabRestorationResult{
		Tabs: make([]RestoredTab, 0, len(tabs)),
	}
	var activeDriverPageID string

	// Create tabs in order, skipping the first one since the session already has an initial page
	for i, tab := range tabs {
		if i == 0 {
			// Navigate the initial page to the first tab's URL instead of creating a new page
			if tab.URL != "" && tab.URL != "about:blank" {
				s.log.WithFields(map[string]interface{}{
					"session_id": sessionID,
					"url":        tab.URL,
				}).Info("RestoreTabs: navigating initial page to first tab URL")
				navReq := &driver.NavigateRequest{URL: tab.URL}
				resp, err := s.sessions.Client().Navigate(ctx, sessionID, navReq)
				if err != nil {
					s.log.WithError(err).WithField("url", tab.URL).Warn("RestoreTabs: failed to navigate initial page to saved URL")
				} else {
					s.log.WithFields(map[string]interface{}{
						"session_id":    sessionID,
						"navigated_url": resp.URL,
						"title":         resp.Title,
					}).Info("RestoreTabs: initial page navigation successful")
					// Store the initial URL for the response
					result.InitialURL = resp.URL
				}
			} else {
				s.log.WithFields(map[string]interface{}{
					"session_id": sessionID,
					"url":        tab.URL,
				}).Info("RestoreTabs: skipping first tab navigation (empty or about:blank)")
			}
			// We don't know the initial page's ID here, so we skip adding to restored for the first tab
			// The client will get the pages via WebSocket events or GetPages
			if tab.IsActive {
				// Mark that the first tab (initial page) should be active
				// Since it's the only page so far, it's already active
			}
			continue
		}

		// Create additional tabs (skip about:blank tabs as they're not useful)
		if tab.URL == "" || tab.URL == "about:blank" {
			s.log.WithFields(map[string]interface{}{
				"session_id": sessionID,
				"index":      i,
				"url":        tab.URL,
			}).Info("RestoreTabs: skipping about:blank tab")
			continue
		}

		s.log.WithFields(map[string]interface{}{
			"session_id": sessionID,
			"index":      i,
			"url":        tab.URL,
		}).Info("RestoreTabs: creating additional tab")

		resp, err := s.sessions.Client().CreatePage(ctx, sessionID, tab.URL)
		if err != nil {
			s.log.WithError(err).WithFields(map[string]interface{}{
				"url":   tab.URL,
				"order": tab.Order,
			}).Warn("RestoreTabs: failed to restore tab")
			continue
		}

		s.log.WithFields(map[string]interface{}{
			"session_id":     sessionID,
			"driver_page_id": resp.DriverPageID,
			"url":            tab.URL,
		}).Info("RestoreTabs: additional tab created successfully")

		result.Tabs = append(result.Tabs, RestoredTab{
			PageID:   resp.DriverPageID,
			URL:      tab.URL,
			IsActive: tab.IsActive,
		})

		if tab.IsActive {
			activeDriverPageID = resp.DriverPageID
		}
	}

	// If there was an active tab that's not the first one, switch to it
	if activeDriverPageID != "" {
		s.log.WithFields(map[string]interface{}{
			"session_id":     sessionID,
			"active_page_id": activeDriverPageID,
		}).Info("RestoreTabs: switching to active page")
		if err := s.sessions.Client().SetActivePage(ctx, sessionID, activeDriverPageID); err != nil {
			s.log.WithError(err).WithField("page_id", activeDriverPageID).Warn("RestoreTabs: failed to set active page after tab restoration")
		}
	}

	s.log.WithFields(map[string]interface{}{
		"session_id":     sessionID,
		"restored_count": len(result.Tabs),
		"initial_url":    result.InitialURL,
	}).Info("RestoreTabs: tab restoration complete")

	return result, nil
}

// AddTimelineAction adds a recorded action to the timeline.
func (s *Service) AddTimelineAction(sessionID string, action *driver.RecordedAction, pageID uuid.UUID) {
	s.timeline.AddAction(sessionID, action, pageID)
}

// AddTimelinePageEvent adds a page event to the timeline.
func (s *Service) AddTimelinePageEvent(sessionID string, event *domain.PageEvent) {
	s.timeline.AddPageEvent(sessionID, event)
}

// GetTimeline returns the unified timeline for a session.
func (s *Service) GetTimeline(sessionID string, pageID *uuid.UUID, limit int) (*domain.TimelineResponse, error) {
	if _, ok := s.sessions.Get(sessionID); !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	entries, hasMore := s.timeline.GetTimelinePaginated(sessionID, pageID, nil, limit)

	return &domain.TimelineResponse{
		Entries:      entries,
		HasMore:      hasMore,
		TotalEntries: s.timeline.GetTimelineCount(sessionID),
	}, nil
}

// ClearTimeline clears the timeline for a session.
func (s *Service) ClearTimeline(sessionID string) {
	s.timeline.ClearSession(sessionID)
}
