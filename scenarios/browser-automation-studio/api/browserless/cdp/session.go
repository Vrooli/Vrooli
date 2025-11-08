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

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/sirupsen/logrus"
)

// Session manages a persistent CDP connection to browserless
type Session struct {
	browserlessURL string
	wsURL          string
	allocCtx       context.Context
	ctx            context.Context
	cancel         context.CancelFunc
	viewportWidth  int
	viewportHeight int
	telemetry      *Telemetry
	log            *logrus.Entry
	mu             sync.Mutex
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
	browserCtx, browserCancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(logf))
	s.ctx = browserCtx
	s.cancel = func() {
		browserCancel()
		cancelAllocator()
	}

	if err := s.initialize(); err != nil {
		s.cancel()
		return nil, fmt.Errorf("failed to initialize CDP session: %w", err)
	}

	return s, nil
}

// initialize sets up the browser context and telemetry listeners
func (s *Session) initialize() error {
	s.log.Info("Initializing CDP session")

	return chromedp.Run(s.ctx,
		// Set viewport
		chromedp.EmulateViewport(int64(s.viewportWidth), int64(s.viewportHeight)),

		// Enable necessary CDP domains
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Enable console domain
			if err := runtime.Enable().Do(ctx); err != nil {
				return fmt.Errorf("failed to enable runtime: %w", err)
			}

			// Enable network domain
			if err := network.Enable().Do(ctx); err != nil {
				return fmt.Errorf("failed to enable network: %w", err)
			}

			// Enable page domain
			if err := page.Enable().Do(ctx); err != nil {
				return fmt.Errorf("failed to enable page: %w", err)
			}

			// Set up console log listener
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

			return nil
		}),
	)
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

// resolveBrowserlessWebSocketURL determines the correct CDP WebSocket endpoint for browserless.
// Browserless advertises ws://0.0.0.0 by default which is not dialable from clients, so we
// rewrite the host and preserve authentication/query parameters before handing it to chromedp.
func resolveBrowserlessWebSocketURL(ctx context.Context, rawURL string) (string, error) {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return "", fmt.Errorf("browserless URL is required")
	}

	if !strings.Contains(trimmed, "://") {
		trimmed = "http://" + trimmed
	}

	base, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("invalid browserless URL: %w", err)
	}

	// If the user already provided a DevTools websocket, normalize it and return early.
	if (base.Scheme == "ws" || base.Scheme == "wss") && strings.Contains(base.Path, "/devtools/browser/") {
		return rewriteAdvertisedWebSocketURL(ctx, base.String(), base)
	}

	versionURL, err := buildBrowserlessVersionURL(base)
	if err != nil {
		return "", err
	}

	versionCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	debuggerURL, err := fetchWebSocketDebuggerURL(versionCtx, versionURL)
	if err != nil {
		return "", err
	}

	return rewriteAdvertisedWebSocketURL(versionCtx, debuggerURL, base)
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
