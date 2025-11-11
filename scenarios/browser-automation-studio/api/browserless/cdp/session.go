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
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/cdp"
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
	tabs               map[target.ID]*tabRecord
	tabOrder           []target.ID
	tabMu              sync.RWMutex
	pointerX           float64
	pointerY           float64
	pointerInitialized bool
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

	wsURL, err := resolveBrowserlessWebSocketURL(ctx, browserlessURL)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve browserless websocket URL: %w", err)
	}
	s.wsURL = wsURL

	if log != nil {
		log.Infof("Connecting to browserless CDP at: %s", wsURL)
	}

	allocCtx, cancelAllocator := chromedp.NewRemoteAllocator(ctx, wsURL, chromedp.NoModifyURL)
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
		return nil, fmt.Errorf("failed to initialize CDP session: %w", err)
	}

	if err := s.captureInitialTarget(browserCtx, browserCancel); err != nil {
		browserCancel()
		cancelAllocator()
		return nil, err
	}

	s.cancel = func() {
		s.closeAllTargets()
		cancelAllocator()
	}

	return s, nil
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
	return chromedp.Run(ctx,
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
	)
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
	ctx, err := s.ensureTargetContext(targetID)
	if err != nil {
		return err
	}
	var previous target.ID
	s.ctxMu.Lock()
	previous = s.currentTargetID
	s.ctx = ctx
	s.currentTargetID = targetID
	s.ctxMu.Unlock()
	if err := s.activateTargetContext(ctx, targetID); err != nil {
		s.log.WithError(err).Warn("failed to activate target")
	}
	if closeOld && previous != "" && previous != targetID {
		go s.closeTarget(previous)
	}
	return nil
}

func (s *Session) closeTarget(targetID target.ID) {
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
		cancel()
	}
	s.removeTab(targetID)
	_ = chromedp.Run(s.ctx, chromedp.ActionFunc(func(runCtx context.Context) error {
		executor := cdp.WithExecutor(runCtx, chromedp.FromContext(runCtx).Browser)
		return target.CloseTarget(targetID).Do(executor)
	}))
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

// resolveBrowserlessWebSocketURL resolves a browserless HTTP URL to a WebSocket debugger URL.
// Supports both browserless v1 and v2 APIs with automatic fallback.
//
// The function tries v2 API first (modern, preferred), then falls back to v1 API (legacy),
// and provides actionable error messages if both fail.
func resolveBrowserlessWebSocketURL(ctx context.Context, rawURL string) (string, error) {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return "", fmt.Errorf("browserless URL is required")
	}

	// Normalize URL (add scheme if missing)
	if !strings.Contains(trimmed, "://") {
		trimmed = "http://" + trimmed
	}

	base, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("invalid browserless URL: %w", err)
	}

	// If user provided a WebSocket URL with /devtools/ path, use it directly
	if (base.Scheme == "ws" || base.Scheme == "wss") && strings.Contains(base.Path, "/devtools/") {
		return rewriteAdvertisedWebSocketURL(ctx, base.String(), base)
	}

	// Try V2 API first (modern, preferred)
	if wsURL, err := tryBrowserlessV2(ctx, base); err == nil {
		return rewriteAdvertisedWebSocketURL(ctx, wsURL, base)
	}

	// Fall back to V1 API (legacy compatibility)
	if wsURL, err := tryBrowserlessV1(ctx, base); err == nil {
		// Note: Consider adding metrics here for production monitoring
		return rewriteAdvertisedWebSocketURL(ctx, wsURL, base)
	}

	// Both APIs failed - provide actionable error
	return "", fmt.Errorf(
		"failed to connect to browserless at %s: "+
			"both v2 (/json/new) and v1 (/json/version) endpoints failed. "+
			"Ensure browserless is running and accessible. "+
			"Supported versions: browserless v1.x (browserless/chrome:1.60+) or v2.x (ghcr.io/browserless/chrome:latest)",
		base.String(),
	)
}

// tryBrowserlessV2 attempts to create a CDP session using browserless v2 API.
//
// Browserless v2 requires explicit session creation before connecting. This function:
//  1. Calls PUT /json/new to create a new browser session
//  2. Parses the response to get the webSocketDebuggerUrl
//  3. Validates the URL contains the /devtools/ path
//
// Returns the WebSocket debugger URL on success, or an error explaining the failure.
//
// See: https://docs.browserless.io/baas/libraries/chromedp
func tryBrowserlessV2(ctx context.Context, base *url.URL) (string, error) {
	if base == nil {
		return "", fmt.Errorf("browserless URL is required")
	}

	// Build session creation URL, preserving query parameters (e.g., auth tokens)
	sessionURL := base.ResolveReference(&url.URL{
		Path:     "/json/new",
		RawQuery: base.RawQuery,
	}).String()

	// Create new session
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, sessionURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to build v2 session request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("v2 endpoint unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("v2 endpoint returned HTTP %d", resp.StatusCode)
	}

	// Parse response
	var session struct {
		ID                   string `json:"id"`
		WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
		URL                  string `json:"url"`
		Type                 string `json:"type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return "", fmt.Errorf("failed to decode v2 session response: %w", err)
	}

	if session.WebSocketDebuggerURL == "" {
		return "", fmt.Errorf("v2 endpoint did not return webSocketDebuggerUrl")
	}

	// V2 should return URL with /devtools/page/<id> or /devtools/browser/<id> path
	if !strings.Contains(session.WebSocketDebuggerURL, "/devtools/") {
		return "", fmt.Errorf("v2 endpoint returned incomplete URL: %s", session.WebSocketDebuggerURL)
	}

	return session.WebSocketDebuggerURL, nil
}

// tryBrowserlessV1 attempts to get WebSocket URL using browserless v1 API.
// V1 returns the WebSocket URL directly from /json/version.
func tryBrowserlessV1(ctx context.Context, base *url.URL) (string, error) {
	versionURL, err := buildBrowserlessVersionURL(base)
	if err != nil {
		return "", err
	}

	debuggerURL, err := fetchWebSocketDebuggerURL(ctx, versionURL)
	if err != nil {
		return "", fmt.Errorf("v1 endpoint failed: %w", err)
	}

	// V1 compatibility check: URL must have /devtools/ path
	if !strings.Contains(debuggerURL, "/devtools/") {
		return "", fmt.Errorf("v1 endpoint returned incomplete URL: %s", debuggerURL)
	}

	return debuggerURL, nil
}

func buildBrowserlessVersionURL(base *url.URL) (string, error) {
	if base == nil {
		return "", fmt.Errorf("browserless URL is required")
	}

	version := *base
	switch version.Scheme {
	case "", "http", "https":
		if version.Scheme == "" {
			version.Scheme = "http"
		}
	case "ws":
		version.Scheme = "http"
	case "wss":
		version.Scheme = "https"
	default:
		return "", fmt.Errorf("unsupported browserless scheme: %s", version.Scheme)
	}

	version.Fragment = ""
	version.Path = joinPath(version.Path, "/json/version")
	// Preserve query parameters such as ?token=...
	// Browserless requires the same token for metadata endpoints.
	version.RawQuery = base.RawQuery

	return version.String(), nil
}

func fetchWebSocketDebuggerURL(ctx context.Context, versionURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, versionURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to build version request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to query browserless version endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("browserless version endpoint returned %d", resp.StatusCode)
	}

	var payload struct {
		WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("failed to decode browserless version response: %w", err)
	}

	if payload.WebSocketDebuggerURL == "" {
		return "", fmt.Errorf("browserless did not include webSocketDebuggerUrl")
	}

	return payload.WebSocketDebuggerURL, nil
}

func rewriteAdvertisedWebSocketURL(ctx context.Context, advertised string, fallback *url.URL) (string, error) {
	if fallback == nil {
		return "", fmt.Errorf("fallback browserless URL is required")
	}

	wsURL, err := url.Parse(advertised)
	if err != nil {
		return "", fmt.Errorf("invalid websocket URL: %w", err)
	}

	host := wsURL.Hostname()
	port := wsURL.Port()
	if host == "" || host == "0.0.0.0" || host == "::" || host == "[::]" {
		host = fallback.Hostname()
	}
	if host == "" {
		host = "localhost"
	}
	if port == "" {
		port = fallback.Port()
		if port == "" {
			port = defaultPortForScheme(fallback.Scheme)
		}
	}

	resolvedHost := resolveHostToDialable(ctx, host)
	if port != "" {
		wsURL.Host = net.JoinHostPort(resolvedHost, port)
	} else {
		wsURL.Host = resolvedHost
	}

	if wsURL.Scheme == "" {
		wsURL.Scheme = pickWebSocketScheme(fallback.Scheme)
	}

	if wsURL.RawQuery == "" && fallback.RawQuery != "" {
		wsURL.RawQuery = fallback.RawQuery
	}

	return wsURL.String(), nil
}

func joinPath(basePath, suffix string) string {
	basePath = strings.TrimSuffix(basePath, "/")
	suffix = strings.TrimPrefix(suffix, "/")
	if basePath == "" {
		return "/" + suffix
	}
	return basePath + "/" + suffix
}

func defaultPortForScheme(scheme string) string {
	switch strings.ToLower(scheme) {
	case "https", "wss":
		return "443"
	case "http", "ws":
		return "80"
	default:
		return ""
	}
}

func pickWebSocketScheme(baseScheme string) string {
	switch strings.ToLower(baseScheme) {
	case "https", "wss":
		return "wss"
	default:
		return "ws"
	}
}

func resolveHostToDialable(ctx context.Context, host string) string {
	if host == "" || host == "localhost" {
		return host
	}
	if ip := net.ParseIP(host); ip != nil {
		return host
	}
	lookupCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	addrs, err := net.DefaultResolver.LookupIPAddr(lookupCtx, host)
	if err != nil || len(addrs) == 0 {
		return host
	}
	return addrs[0].IP.String()
}
