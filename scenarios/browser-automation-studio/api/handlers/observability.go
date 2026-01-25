package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/vrooli/browser-automation-studio/middleware"
	"github.com/vrooli/browser-automation-studio/services/logutil"
)

// Note: getPlaywrightDriverURL is defined in record_mode.go

// proxyWithCorrelation creates a proxied request with correlation ID propagation
func (h *Handler) proxyWithCorrelation(r *http.Request, method, targetURL string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(r.Context(), method, targetURL, body)
	if err != nil {
		return nil, err
	}
	// Propagate correlation ID to downstream service
	middleware.PropagateCorrelationIDFromContext(req)
	return req, nil
}

// DebugModeState tracks the current debug mode configuration
type DebugModeState struct {
	mu         sync.RWMutex
	enabled    bool
	components []string
	expiresAt  time.Time
}

var globalDebugMode = &DebugModeState{}

// DebugModeRequest represents a request to enable/disable debug mode
type DebugModeRequest struct {
	Enabled         bool     `json:"enabled"`
	Components      []string `json:"components,omitempty"`
	DurationMinutes int      `json:"durationMinutes,omitempty"`
}

// DebugModeResponse represents the current debug mode state
type DebugModeResponse struct {
	Enabled       bool      `json:"enabled"`
	Components    []string  `json:"components,omitempty"`
	ExpiresAt     time.Time `json:"expiresAt,omitempty"`
	RemainingMins int       `json:"remainingMins,omitempty"`
}

// GetDebugMode returns the current debug mode state
func (h *Handler) GetDebugMode(w http.ResponseWriter, r *http.Request) {
	globalDebugMode.mu.RLock()
	defer globalDebugMode.mu.RUnlock()

	response := DebugModeResponse{
		Enabled:    globalDebugMode.enabled && time.Now().Before(globalDebugMode.expiresAt),
		Components: globalDebugMode.components,
	}

	if response.Enabled {
		response.ExpiresAt = globalDebugMode.expiresAt
		response.RemainingMins = int(time.Until(globalDebugMode.expiresAt).Minutes())
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		if logger := logutil.LoggerFromContext(r.Context()); logger != nil {
			logger.WithError(err).Warn("observability: failed to encode debug mode response")
		}
	}
}

// SetDebugMode enables or disables debug mode
func (h *Handler) SetDebugMode(w http.ResponseWriter, r *http.Request) {
	var req DebugModeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	globalDebugMode.mu.Lock()
	defer globalDebugMode.mu.Unlock()

	if req.Enabled {
		duration := 30 // Default 30 minutes
		if req.DurationMinutes > 0 && req.DurationMinutes <= 120 {
			duration = req.DurationMinutes
		}

		globalDebugMode.enabled = true
		globalDebugMode.components = req.Components
		globalDebugMode.expiresAt = time.Now().Add(time.Duration(duration) * time.Minute)

		// Log that debug mode was enabled
		if logger := logutil.LoggerFromContext(r.Context()); logger != nil {
			logger.WithFields(map[string]interface{}{
				"components": req.Components,
				"duration":   duration,
			}).Info("Debug mode enabled")
		}
	} else {
		globalDebugMode.enabled = false
		globalDebugMode.components = nil
		globalDebugMode.expiresAt = time.Time{}

		if logger := logutil.LoggerFromContext(r.Context()); logger != nil {
			logger.Info("Debug mode disabled")
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(DebugModeResponse{
		Enabled:    globalDebugMode.enabled,
		Components: globalDebugMode.components,
		ExpiresAt:  globalDebugMode.expiresAt,
	}); err != nil {
		if logger := logutil.LoggerFromContext(r.Context()); logger != nil {
			logger.WithError(err).Warn("observability: failed to encode debug mode update response")
		}
	}
}

// IsDebugEnabled checks if debug mode is enabled for a component
func IsDebugEnabled(component string) bool {
	globalDebugMode.mu.RLock()
	defer globalDebugMode.mu.RUnlock()

	if !globalDebugMode.enabled || time.Now().After(globalDebugMode.expiresAt) {
		return false
	}

	// If no specific components, debug all
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

// GetObservability proxies GET /observability requests to the playwright-driver.
// Query parameters are forwarded (depth, no_cache).
func (h *Handler) GetObservability(w http.ResponseWriter, r *http.Request) {
	driverURL := getPlaywrightDriverURL()
	targetURL := driverURL + "/observability"

	// Forward query parameters
	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}

	req, err := h.proxyWithCorrelation(r, http.MethodGet, targetURL, nil)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Failed to reach playwright-driver", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		if logger := logutil.LoggerFromContext(r.Context()); logger != nil {
			logger.WithError(err).Warn("observability: failed to proxy response")
		}
	}
}

// RefreshObservability proxies POST /observability/refresh requests to the playwright-driver.
func (h *Handler) RefreshObservability(w http.ResponseWriter, r *http.Request) {
	driverURL := getPlaywrightDriverURL()
	targetURL := driverURL + "/observability/refresh"

	req, err := h.proxyWithCorrelation(r, http.MethodPost, targetURL, nil)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Failed to reach playwright-driver", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		if logger := logutil.LoggerFromContext(r.Context()); logger != nil {
			logger.WithError(err).Warn("observability: failed to proxy response")
		}
	}
}

// RunDiagnostics proxies POST /observability/diagnostics/run requests to the playwright-driver.
func (h *Handler) RunDiagnostics(w http.ResponseWriter, r *http.Request) {
	driverURL := getPlaywrightDriverURL()
	targetURL := driverURL + "/observability/diagnostics/run"

	req, err := h.proxyWithCorrelation(r, http.MethodPost, targetURL, r.Body)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	// Copy content-type header
	if ct := r.Header.Get("Content-Type"); ct != "" {
		req.Header.Set("Content-Type", ct)
	}

	client := &http.Client{Timeout: 60 * time.Second} // Longer timeout for diagnostics
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Failed to reach playwright-driver", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		if logger := logutil.LoggerFromContext(r.Context()); logger != nil {
			logger.WithError(err).Warn("observability: failed to proxy response")
		}
	}
}

// GetSessionList proxies GET /observability/sessions requests to the playwright-driver.
func (h *Handler) GetSessionList(w http.ResponseWriter, r *http.Request) {
	driverURL := getPlaywrightDriverURL()
	targetURL := driverURL + "/observability/sessions"

	req, err := h.proxyWithCorrelation(r, http.MethodGet, targetURL, nil)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Failed to reach playwright-driver", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		if logger := logutil.LoggerFromContext(r.Context()); logger != nil {
			logger.WithError(err).Warn("observability: failed to proxy response")
		}
	}
}

// RunCleanup proxies POST /observability/cleanup/run requests to the playwright-driver.
func (h *Handler) RunCleanup(w http.ResponseWriter, r *http.Request) {
	driverURL := getPlaywrightDriverURL()
	targetURL := driverURL + "/observability/cleanup/run"

	req, err := h.proxyWithCorrelation(r, http.MethodPost, targetURL, nil)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Failed to reach playwright-driver", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		if logger := logutil.LoggerFromContext(r.Context()); logger != nil {
			logger.WithError(err).Warn("observability: failed to proxy response")
		}
	}
}

// GetMetrics proxies GET /observability/metrics requests to the playwright-driver.
// Returns metrics in JSON format (parsed from Prometheus text format).
func (h *Handler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	driverURL := getPlaywrightDriverURL()
	targetURL := driverURL + "/observability/metrics"

	req, err := h.proxyWithCorrelation(r, http.MethodGet, targetURL, nil)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Failed to reach playwright-driver", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		if logger := logutil.LoggerFromContext(r.Context()); logger != nil {
			logger.WithError(err).Warn("observability: failed to proxy response")
		}
	}
}

// GetConfigRuntime proxies GET /observability/config/runtime requests to the playwright-driver.
// Returns the current state of all runtime configuration overrides.
func (h *Handler) GetConfigRuntime(w http.ResponseWriter, r *http.Request) {
	driverURL := getPlaywrightDriverURL()
	targetURL := driverURL + "/observability/config/runtime"

	req, err := h.proxyWithCorrelation(r, http.MethodGet, targetURL, nil)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Failed to reach playwright-driver", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		if logger := logutil.LoggerFromContext(r.Context()); logger != nil {
			logger.WithError(err).Warn("observability: failed to proxy response")
		}
	}
}

// UpdateConfig proxies PUT /observability/config/{env_var} requests to the playwright-driver.
// Updates a runtime configuration value.
func (h *Handler) UpdateConfig(w http.ResponseWriter, r *http.Request, envVar string) {
	driverURL := getPlaywrightDriverURL()
	targetURL := driverURL + "/observability/config/" + envVar

	req, err := h.proxyWithCorrelation(r, http.MethodPut, targetURL, r.Body)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	// Copy content-type header
	if ct := r.Header.Get("Content-Type"); ct != "" {
		req.Header.Set("Content-Type", ct)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Failed to reach playwright-driver", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		if logger := logutil.LoggerFromContext(r.Context()); logger != nil {
			logger.WithError(err).Warn("observability: failed to proxy response")
		}
	}
}

// ResetConfig proxies DELETE /observability/config/{env_var} requests to the playwright-driver.
// Resets a runtime configuration value back to its environment/default value.
func (h *Handler) ResetConfig(w http.ResponseWriter, r *http.Request, envVar string) {
	driverURL := getPlaywrightDriverURL()
	targetURL := driverURL + "/observability/config/" + envVar

	req, err := h.proxyWithCorrelation(r, http.MethodDelete, targetURL, nil)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Failed to reach playwright-driver", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		if logger := logutil.LoggerFromContext(r.Context()); logger != nil {
			logger.WithError(err).Warn("observability: failed to proxy response")
		}
	}
}

// RunPipelineTest proxies POST /observability/pipeline-test requests to the playwright-driver.
// Runs an autonomous end-to-end test of the recording pipeline.
// This test automatically creates a session if needed, runs the test, and cleans up.
func (h *Handler) RunPipelineTest(w http.ResponseWriter, r *http.Request) {
	driverURL := getPlaywrightDriverURL()
	targetURL := driverURL + "/observability/pipeline-test"

	req, err := h.proxyWithCorrelation(r, http.MethodPost, targetURL, r.Body)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	// Copy content-type header
	if ct := r.Header.Get("Content-Type"); ct != "" {
		req.Header.Set("Content-Type", ct)
	}

	// Longer timeout - the test creates a session and simulates interactions
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Failed to reach playwright-driver", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		if logger := logutil.LoggerFromContext(r.Context()); logger != nil {
			logger.WithError(err).Warn("observability: failed to proxy response")
		}
	}
}
