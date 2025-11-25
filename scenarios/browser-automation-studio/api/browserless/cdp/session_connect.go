package cdp

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// resolveBrowserlessWebSocketURL resolves a browserless HTTP URL to a WebSocket debugger URL.
// Supports both browserless v1 and v2 APIs with optional fallback for legacy
// installations. V2 is required by default; set BROWSERLESS_ALLOW_V1_FALLBACK=1
// to re-enable the legacy path while migrating older deployments.
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
	if wsURL, v2Err := tryBrowserlessV2(ctx, base); v2Err == nil {
		return rewriteAdvertisedWebSocketURL(ctx, wsURL, base)
	} else if !allowLegacyV1Fallback() {
		return "", fmt.Errorf(
			"browserless v2 required (set BROWSERLESS_ALLOW_V1_FALLBACK=1 to allow v1): %w",
			v2Err,
		)
	}

	// Fall back to V1 API (legacy compatibility) when explicitly enabled.
	if wsURL, err := tryBrowserlessV1(ctx, base); err == nil {
		return rewriteAdvertisedWebSocketURL(ctx, wsURL, base)
	}

	return "", fmt.Errorf("failed to connect to browserless at %s: v2 unavailable and v1 fallback disabled or failing", base.String())
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
		return "", fmt.Errorf("version endpoint returned HTTP %d", resp.StatusCode)
	}

	var versionResp struct {
		WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&versionResp); err != nil {
		return "", fmt.Errorf("failed to decode version response: %w", err)
	}

	if versionResp.WebSocketDebuggerURL == "" {
		return "", fmt.Errorf("version endpoint did not return webSocketDebuggerUrl")
	}

	return versionResp.WebSocketDebuggerURL, nil
}

func rewriteAdvertisedWebSocketURL(ctx context.Context, advertised string, fallback *url.URL) (string, error) {
	if strings.TrimSpace(advertised) == "" {
		return "", fmt.Errorf("websocket URL is empty")
	}
	wsURL, err := url.Parse(advertised)
	if err != nil {
		return "", fmt.Errorf("invalid websocket URL: %w", err)
	}

	// Some deployments return ws://0.0.0.0:<port>/devtools/browser/<id>
	// Replace 0.0.0.0 with the actual host from the HTTP endpoint.
	host := wsURL.Hostname()
	port := wsURL.Port()

	if host == "" || host == "0.0.0.0" || host == "::" {
		host = fallback.Hostname()
		if host == "" {
			host = fallback.Host
		}
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

func allowLegacyV1Fallback() bool {
	val := strings.ToLower(strings.TrimSpace(os.Getenv("BROWSERLESS_ALLOW_V1_FALLBACK")))
	switch val {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
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
