// Package cdp provides Chrome DevTools Protocol session management for browserless.
//
// Browserless Compatibility:
//
// This package supports both browserless v1 and v2 APIs:
//   - V1 (browserless/chrome:1.60+): Uses /json/version endpoint
//   - V2 (ghcr.io/browserless/chrome:latest): Uses /json/new endpoint
//
// The connection resolver automatically detects the API version and falls back
// to v1 if v2 is unavailable, ensuring compatibility with both versions.
package cdp

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
	"github.com/sirupsen/logrus"
)

// Session manages a persistent CDP connection to browserless
type Session struct {
	browserlessURL     string
	wsURL              string
	allocCtx           context.Context
	ctx                context.Context
	cancel             context.CancelFunc
	logFunc            func(string, ...interface{})
	viewportWidth      int
	viewportHeight     int
	telemetry          *Telemetry
	log                *logrus.Entry
	mu                 sync.Mutex
	ctxMu              sync.RWMutex
	targetContexts     map[target.ID]context.Context
	targetCancels      map[target.ID]context.CancelFunc
	currentTargetID    target.ID
	activeOperations   int        // Count of active operations using current context
	operationCond      *sync.Cond // Condition variable for waiting on operations to complete
	tabs               map[target.ID]*tabRecord
	tabOrder           []target.ID
	tabMu              sync.RWMutex
	pointerX           float64
	pointerY           float64
	pointerInitialized bool
	frameMu            sync.RWMutex
	frameStack         []*frameScope
	networkMockMu      sync.RWMutex
	networkMocks       []*networkMockRule
}

// Telemetry collects console logs and network events
type Telemetry struct {
	consoleLogs   []ConsoleLog
	networkEvents []NetworkEvent
	mu            sync.Mutex
}

// ConsoleLog represents a browser console message
type ConsoleLog struct {
	Type      string    `json:"type"`
	Text      string    `json:"text"`
	Timestamp time.Time `json:"timestamp"`
}

// NetworkEvent represents a network request/response
type NetworkEvent struct {
	Type         string    `json:"type"` // request, response, requestfailed
	URL          string    `json:"url"`
	Method       string    `json:"method,omitempty"`
	Status       int64     `json:"status,omitempty"`
	ResourceType string    `json:"resourceType,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
}

type tabRecord struct {
	ID        target.ID
	Title     string
	URL       string
	Attached  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewSession creates a new CDP session connected to browserless
func NewSession(ctx context.Context, browserlessURL string, viewportWidth, viewportHeight int, log *logrus.Entry) (*Session, error) {
	if viewportWidth <= 0 {
		viewportWidth = 1920
	}
	if viewportHeight <= 0 {
		viewportHeight = 1080
	}

	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		s := &Session{
			browserlessURL: browserlessURL,
			viewportWidth:  viewportWidth,
			viewportHeight: viewportHeight,
			telemetry:      &Telemetry{},
			log:            log,
			targetContexts: make(map[target.ID]context.Context),
			targetCancels:  make(map[target.ID]context.CancelFunc),
			tabs:           make(map[target.ID]*tabRecord),
		}
		s.operationCond = sync.NewCond(&s.ctxMu)

		wsURL, err := resolveBrowserlessWebSocketURL(ctx, browserlessURL)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve browserless websocket URL: %w", err)
		}
		s.wsURL = wsURL

		if log != nil {
			log.Infof("Connecting to browserless CDP at: %s (attempt %d)", wsURL, attempt)
		}

		allocCtx, cancelAllocator := chromedp.NewRemoteAllocator(context.Background(), wsURL, chromedp.NoModifyURL)
		s.allocCtx = allocCtx

		logf := func(string, ...interface{}) {}
		if log != nil {
			logf = log.Debugf
		}
		s.logFunc = logf
		browserCtx, browserCancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(logf))
		s.ctx = browserCtx

		if err := s.initialize(); err != nil {
			browserCancel()
			cancelAllocator()
			lastErr = fmt.Errorf("attempt %d: failed to initialize CDP session: %w", attempt, err)
			time.Sleep(time.Duration(attempt) * 200 * time.Millisecond)
			continue
		}

		if err := s.captureInitialTarget(browserCtx, browserCancel); err != nil {
			browserCancel()
			cancelAllocator()
			lastErr = fmt.Errorf("attempt %d: %w", attempt, err)
			time.Sleep(time.Duration(attempt) * 200 * time.Millisecond)
			continue
		}

		s.cancel = func() {
			s.closeAllTargets()
			cancelAllocator()
		}

		return s, nil
	}

	return nil, fmt.Errorf("failed to initialize CDP session after retries: %w", lastErr)
}

// initialize sets up the browser context and telemetry listeners
func (s *Session) initialize() error {
	s.log.Info("Initializing CDP session")

	s.installBrowserListeners(s.ctx)

	return s.configureContext(s.ctx)
}

func (s *Session) captureInitialTarget(ctx context.Context, cancel context.CancelFunc) error {
	chromedpCtx := chromedp.FromContext(ctx)
	if chromedpCtx == nil || chromedpCtx.Target == nil {
		return fmt.Errorf("chromedp context has no active target")
	}
	if chromedpCtx.Target.TargetID == "" {
		return fmt.Errorf("chromedp context returned empty target id")
	}

	s.ctxMu.Lock()
	s.targetContexts[chromedpCtx.Target.TargetID] = ctx
	s.targetCancels[chromedpCtx.Target.TargetID] = cancel
	s.currentTargetID = chromedpCtx.Target.TargetID
	s.ctxMu.Unlock()

	s.recordTabInfo(&target.Info{
		TargetID: chromedpCtx.Target.TargetID,
		Type:     "page",
		Attached: true,
	})

	if err := s.refreshTabInventory(ctx); err != nil {
		s.log.WithError(err).Debug("failed to seed tab inventory")
	}

	return nil
}

func (s *Session) closeAllTargets() {
	s.ctxMu.Lock()
	cancels := make([]context.CancelFunc, 0, len(s.targetCancels))
	for id, cancel := range s.targetCancels {
		cancels = append(cancels, cancel)
		delete(s.targetCancels, id)
		delete(s.targetContexts, id)
	}
	s.ctxMu.Unlock()
	for _, cancel := range cancels {
		cancel()
	}
}

func (s *Session) configureContext(ctx context.Context) error {
	if err := chromedp.Run(ctx,
		chromedp.EmulateViewport(int64(s.viewportWidth), int64(s.viewportHeight)),
		chromedp.ActionFunc(func(runCtx context.Context) error {
			if err := runtime.Enable().Do(runCtx); err != nil {
				return fmt.Errorf("failed to enable runtime: %w", err)
			}
			if err := network.Enable().Do(runCtx); err != nil {
				return fmt.Errorf("failed to enable network: %w", err)
			}
			if err := page.Enable().Do(runCtx); err != nil {
				return fmt.Errorf("failed to enable page: %w", err)
			}
			s.attachTelemetry(runCtx)
			return nil
		}),
	); err != nil {
		return err
	}
	s.ensureNetworkMocksForContext(ctx)
	return nil
}

func (s *Session) installBrowserListeners(ctx context.Context) {
	chromedp.ListenBrowser(ctx, func(ev interface{}) {
		switch ev := ev.(type) {
		case *target.EventTargetCreated:
			s.recordTabInfo(ev.TargetInfo)
		case *target.EventTargetInfoChanged:
			s.recordTabInfo(ev.TargetInfo)
		case *target.EventTargetDestroyed:
			s.removeTab(ev.TargetID)
		}
	})
}

func (s *Session) attachTelemetry(ctx context.Context) {
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		switch ev := ev.(type) {
		case *runtime.EventConsoleAPICalled:
			s.handleConsoleLog(ev)
		case *network.EventRequestWillBeSent:
			s.handleNetworkRequest(ev)
		case *network.EventResponseReceived:
			s.handleNetworkResponse(ev)
		case *network.EventLoadingFailed:
			s.handleNetworkFailed(ev)
		case *fetch.EventRequestPaused:
			go s.processFetchIntercept(ctx, ev)
		}
	})
}

func (s *Session) recordTabInfo(info *target.Info) {
	if info == nil || info.TargetID == "" || info.Type != "page" {
		return
	}
	s.tabMu.Lock()
	defer s.tabMu.Unlock()
	record, exists := s.tabs[info.TargetID]
	if !exists {
		record = &tabRecord{
			ID:        info.TargetID,
			CreatedAt: time.Now(),
		}
		s.tabs[info.TargetID] = record
		if !s.idInOrder(info.TargetID) {
			s.tabOrder = append(s.tabOrder, info.TargetID)
		}
	}
	record.Title = info.Title
	record.URL = info.URL
	record.Attached = info.Attached
	record.UpdatedAt = time.Now()
}

func (s *Session) idInOrder(targetID target.ID) bool {
	for _, existing := range s.tabOrder {
		if existing == targetID {
			return true
		}
	}
	return false
}

func (s *Session) removeTab(targetID target.ID) {
	if targetID == "" {
		return
	}
	s.tabMu.Lock()
	defer s.tabMu.Unlock()
	delete(s.tabs, targetID)
	if len(s.tabOrder) == 0 {
		return
	}
	filtered := make([]target.ID, 0, len(s.tabOrder))
	for _, id := range s.tabOrder {
		if id != targetID {
			filtered = append(filtered, id)
		}
	}
	s.tabOrder = filtered
}

func (s *Session) refreshTabInventory(ctx context.Context) error {
	var infos []*target.Info
	action := chromedp.ActionFunc(func(runCtx context.Context) error {
		executor := cdp.WithExecutor(runCtx, chromedp.FromContext(runCtx).Browser)
		list, err := target.GetTargets().Do(executor)
		if err != nil {
			return err
		}
		infos = list
		return nil
	})
	if err := chromedp.Run(ctx, action); err != nil {
		return err
	}
	seen := make(map[target.ID]struct{}, len(infos))
	for _, info := range infos {
		if info == nil || info.Type != "page" {
			continue
		}
		seen[info.TargetID] = struct{}{}
		s.recordTabInfo(info)
	}
	s.tabMu.Lock()
	if len(seen) == 0 && len(s.tabs) == 0 {
		s.tabMu.Unlock()
		return nil
	}
	filteredOrder := make([]target.ID, 0, len(s.tabOrder))
	for _, id := range s.tabOrder {
		if _, ok := seen[id]; ok {
			filteredOrder = append(filteredOrder, id)
		} else {
			delete(s.tabs, id)
		}
	}
	s.tabOrder = filteredOrder
	for id := range s.tabs {
		if _, ok := seen[id]; !ok {
			delete(s.tabs, id)
		}
	}
	s.tabMu.Unlock()
	return nil
}

func (s *Session) snapshotTabs() []tabRecord {
	s.tabMu.RLock()
	defer s.tabMu.RUnlock()
	result := make([]tabRecord, 0, len(s.tabOrder))
	for _, id := range s.tabOrder {
		if rec, ok := s.tabs[id]; ok {
			result = append(result, *rec)
		}
	}
	return result
}

func (s *Session) snapshotTargetContexts() []context.Context {
	s.ctxMu.RLock()
	defer s.ctxMu.RUnlock()
	contexts := make([]context.Context, 0, len(s.targetContexts))
	for _, ctx := range s.targetContexts {
		contexts = append(contexts, ctx)
	}
	return contexts
}

func (s *Session) currentTabSet() map[target.ID]struct{} {
	s.tabMu.RLock()
	defer s.tabMu.RUnlock()
	ids := make(map[target.ID]struct{}, len(s.tabs))
	for id := range s.tabs {
		ids[id] = struct{}{}
	}
	return ids
}

func (s *Session) waitForNewTab(ctx context.Context, known map[target.ID]struct{}, timeout time.Duration) (target.ID, error) {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(150 * time.Millisecond)
	defer ticker.Stop()
	for {
		if err := s.refreshTabInventory(ctx); err != nil {
			s.log.WithError(err).Warn("tab inventory refresh failed while waiting for new tab")
		}
		s.tabMu.RLock()
		for _, id := range s.tabOrder {
			if _, ok := known[id]; !ok {
				s.tabMu.RUnlock()
				return id, nil
			}
		}
		s.tabMu.RUnlock()
		if time.Now().After(deadline) {
			return "", fmt.Errorf("timed out waiting for new tab")
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-ticker.C:
		}
	}
}

func (s *Session) getCurrentTargetID() target.ID {
	s.ctxMu.RLock()
	defer s.ctxMu.RUnlock()
	return s.currentTargetID
}

func (s *Session) ensureTargetContext(targetID target.ID) (context.Context, error) {
	s.ctxMu.RLock()
	if ctx, ok := s.targetContexts[targetID]; ok {
		s.ctxMu.RUnlock()
		return ctx, nil
	}
	s.ctxMu.RUnlock()

	ctx, cancel := chromedp.NewContext(s.allocCtx, chromedp.WithTargetID(targetID), chromedp.WithLogf(s.logFunc))
	if err := s.configureContext(ctx); err != nil {
		cancel()
		return nil, err
	}
	s.ctxMu.Lock()
	s.targetContexts[targetID] = ctx
	s.targetCancels[targetID] = cancel
	s.ctxMu.Unlock()
	return ctx, nil
}

func (s *Session) activateTargetContext(ctx context.Context, targetID target.ID) error {
	return chromedp.Run(ctx, chromedp.ActionFunc(func(runCtx context.Context) error {
		executor := cdp.WithExecutor(runCtx, chromedp.FromContext(runCtx).Browser)
		return target.ActivateTarget(targetID).Do(executor)
	}))
}

func (s *Session) switchToTarget(targetID target.ID, closeOld bool) error {
	s.log.WithFields(map[string]interface{}{
		"targetID": targetID,
		"closeOld": closeOld,
	}).Info("[TARGET] switchToTarget called")

	ctx, err := s.ensureTargetContext(targetID)
	if err != nil {
		return err
	}
	var previous target.ID
	s.ctxMu.Lock()
	// Wait for all active operations to complete before switching contexts
	for s.activeOperations > 0 {
		s.log.WithField("active_ops", s.activeOperations).Info("[TARGET] Waiting for operations to complete before context switch")
		s.operationCond.Wait()
	}
	previous = s.currentTargetID
	s.log.WithFields(map[string]interface{}{
		"previous": previous,
		"new":      targetID,
	}).Info("[TARGET] Switching context")
	s.ctx = ctx
	s.currentTargetID = targetID
	s.ctxMu.Unlock()
	if err := s.activateTargetContext(ctx, targetID); err != nil {
		s.log.WithError(err).Warn("failed to activate target")
	}
	if closeOld && previous != "" && previous != targetID {
		s.log.WithField("previous", previous).Info("[TARGET] Closing old target")
		go s.closeTarget(previous)
	}
	return nil
}

func (s *Session) closeTarget(targetID target.ID) {
	s.log.WithField("targetID", targetID).Info("[TARGET] closeTarget called")
	s.ctxMu.Lock()
	cancel, ok := s.targetCancels[targetID]
	if ok {
		delete(s.targetCancels, targetID)
	}
	if _, exists := s.targetContexts[targetID]; exists {
		delete(s.targetContexts, targetID)
	}
	s.ctxMu.Unlock()
	if ok {
		s.log.WithField("targetID", targetID).Info("[TARGET] Cancelling target context")
		cancel()
	}
	s.removeTab(targetID)
	_ = chromedp.Run(s.ctx, chromedp.ActionFunc(func(runCtx context.Context) error {
		executor := cdp.WithExecutor(runCtx, chromedp.FromContext(runCtx).Browser)
		return target.CloseTarget(targetID).Do(executor)
	}))
	s.log.WithField("targetID", targetID).Info("[TARGET] Target closed")
}

// handleConsoleLog processes console.log events
func (s *Session) handleConsoleLog(ev *runtime.EventConsoleAPICalled) {
	s.telemetry.mu.Lock()
	defer s.telemetry.mu.Unlock()

	// Build log text from arguments
	var text string
	for i, arg := range ev.Args {
		if i > 0 {
			text += " "
		}
		if arg.Value != nil {
			text += fmt.Sprintf("%v", arg.Value)
		} else if arg.Description != "" {
			text += arg.Description
		}
	}

	s.telemetry.consoleLogs = append(s.telemetry.consoleLogs, ConsoleLog{
		Type:      ev.Type.String(),
		Text:      text,
		Timestamp: time.Now(),
	})
}

// handleNetworkRequest processes network request events
func (s *Session) handleNetworkRequest(ev *network.EventRequestWillBeSent) {
	s.telemetry.mu.Lock()
	defer s.telemetry.mu.Unlock()

	s.telemetry.networkEvents = append(s.telemetry.networkEvents, NetworkEvent{
		Type:         "request",
		URL:          ev.Request.URL,
		Method:       ev.Request.Method,
		ResourceType: ev.Type.String(),
		Timestamp:    time.Now(),
	})
}

// handleNetworkResponse processes network response events
func (s *Session) handleNetworkResponse(ev *network.EventResponseReceived) {
	s.telemetry.mu.Lock()
	defer s.telemetry.mu.Unlock()

	s.telemetry.networkEvents = append(s.telemetry.networkEvents, NetworkEvent{
		Type:      "response",
		URL:       ev.Response.URL,
		Status:    ev.Response.Status,
		Timestamp: time.Now(),
	})
}

// handleNetworkFailed processes network failure events
func (s *Session) handleNetworkFailed(ev *network.EventLoadingFailed) {
	s.telemetry.mu.Lock()
	defer s.telemetry.mu.Unlock()

	s.telemetry.networkEvents = append(s.telemetry.networkEvents, NetworkEvent{
		Type:      "requestfailed",
		URL:       ev.Timestamp.Time().String(), // URL stored in timestamp for failed requests
		Timestamp: time.Now(),
	})
}

// GetTelemetry returns a snapshot of current telemetry
func (s *Session) GetTelemetry() ([]ConsoleLog, []NetworkEvent) {
	s.telemetry.mu.Lock()
	defer s.telemetry.mu.Unlock()

	// Return copies
	logs := make([]ConsoleLog, len(s.telemetry.consoleLogs))
	copy(logs, s.telemetry.consoleLogs)

	events := make([]NetworkEvent, len(s.telemetry.networkEvents))
	copy(events, s.telemetry.networkEvents)

	return logs, events
}

// ClearTelemetry resets telemetry counters
func (s *Session) ClearTelemetry() {
	s.telemetry.mu.Lock()
	defer s.telemetry.mu.Unlock()

	s.telemetry.consoleLogs = nil
	s.telemetry.networkEvents = nil
}

// CleanBrowserState clears browser storage and cookies without closing the session
// This is useful for test isolation while keeping the browser process alive
func (s *Session) CleanBrowserState() error {
	// For now, do NOTHING - just clear internal state
	// Session reuse with no cleanup to maximize speed

	// Clear telemetry
	s.telemetry.mu.Lock()
	s.telemetry.consoleLogs = nil
	s.telemetry.networkEvents = nil
	s.telemetry.mu.Unlock()

	// Reset pointer state
	s.mu.Lock()
	s.pointerX = 0
	s.pointerY = 0
	s.pointerInitialized = false
	s.mu.Unlock()

	// Clear frame stack
	s.frameMu.Lock()
	s.frameStack = nil
	s.frameMu.Unlock()

	// Clear network mocks
	s.networkMockMu.Lock()
	s.networkMocks = nil
	s.networkMockMu.Unlock()

	return nil
}

// Close terminates the CDP session
func (s *Session) Close() error {
	s.log.Info("Closing CDP session")
	s.cancel()
	return nil
}

// Context returns the browser context for executing actions
func (s *Session) Context() context.Context {
	return s.ctx
}

// BeginOperation marks the start of an operation, incrementing the active operation
// counter and returning the current context. This prevents the context from being
// cancelled/switched while the operation is active.
//
// IMPORTANT: Must be paired with EndOperation() using defer.
func (s *Session) BeginOperation() context.Context {
	s.ctxMu.Lock()
	defer s.ctxMu.Unlock()
	s.activeOperations++
	s.log.WithFields(map[string]interface{}{
		"active_ops":       s.activeOperations,
		"current_ctx_err":  s.ctx.Err(),
		"current_targetID": s.currentTargetID,
	}).Info("[OPERATION] Operation started")
	return s.ctx
}

// EndOperation marks the end of an operation, decrementing the active operation
// counter and signaling any waiting context switches.
func (s *Session) EndOperation() {
	s.ctxMu.Lock()
	defer s.ctxMu.Unlock()
	s.activeOperations--
	s.log.WithFields(map[string]interface{}{
		"active_ops": s.activeOperations,
		"ctx_err":    s.ctx.Err(),
	}).Info("[OPERATION] Operation ended")
	if s.activeOperations == 0 {
		s.log.Info("[OPERATION] All operations complete, broadcasting")
		s.operationCond.Broadcast()
	}
}

// GetCurrentContext returns a thread-safe snapshot of the current browser context.
// This method should be used when deriving child contexts (e.g., with WithTimeout)
// to prevent race conditions when s.ctx is updated by concurrent operations like
// tab switching or target changes.
//
// Using this method ensures that derived contexts remain valid for the duration
// of an operation, even if the session switches to a different target mid-operation.
//
// DEPRECATED: Use BeginOperation/EndOperation instead for operation-level protection.
func (s *Session) GetCurrentContext() context.Context {
	s.ctxMu.RLock()
	defer s.ctxMu.RUnlock()
	return s.ctx
}
