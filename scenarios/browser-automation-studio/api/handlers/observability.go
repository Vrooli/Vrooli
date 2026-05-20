package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/vrooli/browser-automation-studio/middleware"
)

// Note: getPlaywrightDriverURL is defined in record_mode.go.
//
// This file hosts the transport-agnostic helpers powering the
// ObservabilityService Connect handler in handlers/observability/. The
// legacy chi REST endpoints were deleted in the Phase 4 proto+Connect
// migration; the methods below are consumed by the Connect adapter only.

// =============================================================================
// Debug-mode toggle state
// =============================================================================

// DebugModeState tracks the current in-process debug-mode configuration.
type DebugModeState struct {
	mu         sync.RWMutex
	enabled    bool
	components []string
	expiresAt  time.Time
}

var globalDebugMode = &DebugModeState{}

// DebugModeSnapshot is the transport-agnostic readout of the debug-mode
// toggle. Both the Connect handler and downstream callers consume it
// directly; the proto layer maps it to DebugModeState.
type DebugModeSnapshot struct {
	Enabled       bool
	Components    []string
	ExpiresAt     time.Time
	RemainingMins int
}

// GetDebugModeSnapshot returns the live debug-mode state.
func GetDebugModeSnapshot() DebugModeSnapshot {
	globalDebugMode.mu.RLock()
	defer globalDebugMode.mu.RUnlock()
	enabled := globalDebugMode.enabled && time.Now().Before(globalDebugMode.expiresAt)
	out := DebugModeSnapshot{
		Enabled:    enabled,
		Components: append([]string(nil), globalDebugMode.components...),
	}
	if enabled {
		out.ExpiresAt = globalDebugMode.expiresAt
		out.RemainingMins = int(time.Until(globalDebugMode.expiresAt).Minutes())
	}
	return out
}

// SetDebugModeState toggles debug mode on or off. When enabling, duration
// is clamped to [1, 120] minutes with a default of 30. components is the
// optional whitelist of debug-eligible components.
func SetDebugModeState(enabled bool, components []string, durationMinutes int) DebugModeSnapshot {
	globalDebugMode.mu.Lock()
	defer globalDebugMode.mu.Unlock()
	if enabled {
		duration := 30
		if durationMinutes > 0 && durationMinutes <= 120 {
			duration = durationMinutes
		}
		globalDebugMode.enabled = true
		globalDebugMode.components = append([]string(nil), components...)
		globalDebugMode.expiresAt = time.Now().Add(time.Duration(duration) * time.Minute)
	} else {
		globalDebugMode.enabled = false
		globalDebugMode.components = nil
		globalDebugMode.expiresAt = time.Time{}
	}
	snap := DebugModeSnapshot{
		Enabled:    globalDebugMode.enabled,
		Components: append([]string(nil), globalDebugMode.components...),
	}
	if globalDebugMode.enabled {
		snap.ExpiresAt = globalDebugMode.expiresAt
		snap.RemainingMins = int(time.Until(globalDebugMode.expiresAt).Minutes())
	}
	return snap
}

// IsDebugEnabled checks whether debug mode is enabled for a named
// component. Empty component-list means "all".
func IsDebugEnabled(component string) bool {
	globalDebugMode.mu.RLock()
	defer globalDebugMode.mu.RUnlock()

	if !globalDebugMode.enabled || time.Now().After(globalDebugMode.expiresAt) {
		return false
	}

	if len(globalDebugMode.components) == 0 {
		return true
	}

	for _, c := range globalDebugMode.components {
		if c == component {
			return true
		}
	}
	return false
}

// =============================================================================
// Playwright-driver observability proxy
// =============================================================================

// ObservabilityProxyError is returned by the proxy helpers when the
// playwright-driver upstream is unreachable or rejects the request. The
// Connect adapter maps these to the right connect.Code.
type ObservabilityProxyError struct {
	StatusCode int
	Body       []byte
	Wrapped    error
}

func (e *ObservabilityProxyError) Error() string {
	if e.Wrapped != nil {
		return fmt.Sprintf("playwright-driver proxy: %v", e.Wrapped)
	}
	return fmt.Sprintf("playwright-driver proxy: status=%d body=%s", e.StatusCode, string(e.Body))
}

func (e *ObservabilityProxyError) Unwrap() error { return e.Wrapped }

// ErrUpstreamUnavailable is reported when the playwright-driver is
// unreachable (DNS/connect failure). The Connect layer maps this to
// CodeUnavailable.
var ErrUpstreamUnavailable = errors.New("playwright-driver upstream unavailable")

// fetchObservabilityJSON issues an HTTP request to the playwright-driver
// observability surface and decodes the response body as a JSON object.
// Non-2xx upstream responses are surfaced as ObservabilityProxyError.
func (h *Handler) fetchObservabilityJSON(
	ctx context.Context,
	method, path string,
	query url.Values,
	body io.Reader,
	timeout time.Duration,
) (map[string]any, error) {
	driverURL := getPlaywrightDriverURL()
	target := driverURL + path
	if len(query) > 0 {
		target += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, target, body)
	if err != nil {
		return nil, &ObservabilityProxyError{Wrapped: err}
	}
	middleware.PropagateCorrelationIDFromContext(req)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUpstreamUnavailable, err)
	}
	defer resp.Body.Close()

	raw, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, &ObservabilityProxyError{StatusCode: resp.StatusCode, Wrapped: readErr}
	}
	if resp.StatusCode >= 400 {
		return nil, &ObservabilityProxyError{StatusCode: resp.StatusCode, Body: raw}
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return map[string]any{}, nil
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, &ObservabilityProxyError{StatusCode: resp.StatusCode, Body: raw, Wrapped: err}
	}
	return out, nil
}

// FetchObservability proxies GET /observability with optional depth/no_cache.
func (h *Handler) FetchObservability(ctx context.Context, depth string, noCache bool) (map[string]any, error) {
	q := url.Values{}
	if depth != "" {
		q.Set("depth", depth)
	}
	if noCache {
		q.Set("no_cache", "true")
	}
	return h.fetchObservabilityJSON(ctx, http.MethodGet, "/observability", q, nil, 30*time.Second)
}

// FetchObservabilityRefresh proxies POST /observability/refresh.
func (h *Handler) FetchObservabilityRefresh(ctx context.Context) (map[string]any, error) {
	return h.fetchObservabilityJSON(ctx, http.MethodPost, "/observability/refresh", nil, nil, 10*time.Second)
}

// FetchObservabilityDiagnostics proxies POST /observability/diagnostics/run.
// options is JSON-encoded and forwarded as the request body.
func (h *Handler) FetchObservabilityDiagnostics(ctx context.Context, options map[string]any) (map[string]any, error) {
	body, err := encodeJSONBody(options)
	if err != nil {
		return nil, err
	}
	return h.fetchObservabilityJSON(ctx, http.MethodPost, "/observability/diagnostics/run", nil, body, 60*time.Second)
}

// FetchObservabilitySessions proxies GET /observability/sessions.
func (h *Handler) FetchObservabilitySessions(ctx context.Context) (map[string]any, error) {
	return h.fetchObservabilityJSON(ctx, http.MethodGet, "/observability/sessions", nil, nil, 10*time.Second)
}

// FetchObservabilityCleanup proxies POST /observability/cleanup/run.
func (h *Handler) FetchObservabilityCleanup(ctx context.Context) (map[string]any, error) {
	return h.fetchObservabilityJSON(ctx, http.MethodPost, "/observability/cleanup/run", nil, nil, 30*time.Second)
}

// FetchObservabilityMetrics proxies GET /observability/metrics.
func (h *Handler) FetchObservabilityMetrics(ctx context.Context) (map[string]any, error) {
	return h.fetchObservabilityJSON(ctx, http.MethodGet, "/observability/metrics", nil, nil, 10*time.Second)
}

// FetchObservabilityPipelineTest proxies POST /observability/pipeline-test.
func (h *Handler) FetchObservabilityPipelineTest(ctx context.Context, options map[string]any) (map[string]any, error) {
	body, err := encodeJSONBody(options)
	if err != nil {
		return nil, err
	}
	return h.fetchObservabilityJSON(ctx, http.MethodPost, "/observability/pipeline-test", nil, body, 120*time.Second)
}

// FetchObservabilityConfigRuntime proxies GET /observability/config/runtime.
func (h *Handler) FetchObservabilityConfigRuntime(ctx context.Context) (map[string]any, error) {
	return h.fetchObservabilityJSON(ctx, http.MethodGet, "/observability/config/runtime", nil, nil, 10*time.Second)
}

// UpdateObservabilityConfig proxies PUT /observability/config/{envVar}.
func (h *Handler) UpdateObservabilityConfig(ctx context.Context, envVar, value string) (map[string]any, error) {
	if envVar == "" {
		return nil, errors.New("envVar is required")
	}
	body, err := encodeJSONBody(map[string]any{"value": value})
	if err != nil {
		return nil, err
	}
	return h.fetchObservabilityJSON(ctx, http.MethodPut, "/observability/config/"+url.PathEscape(envVar), nil, body, 10*time.Second)
}

// ResetObservabilityConfig proxies DELETE /observability/config/{envVar}.
func (h *Handler) ResetObservabilityConfig(ctx context.Context, envVar string) (map[string]any, error) {
	if envVar == "" {
		return nil, errors.New("envVar is required")
	}
	return h.fetchObservabilityJSON(ctx, http.MethodDelete, "/observability/config/"+url.PathEscape(envVar), nil, nil, 10*time.Second)
}

// encodeJSONBody marshals the supplied map as JSON. A nil/empty map is
// encoded as `{}` so downstream parsers always see a valid object.
func encodeJSONBody(payload map[string]any) (io.Reader, error) {
	if payload == nil {
		payload = map[string]any{}
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(raw), nil
}
