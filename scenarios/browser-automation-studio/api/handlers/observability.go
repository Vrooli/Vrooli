package handlers

import (
	"io"
	"net/http"
	"time"
)

// Note: getPlaywrightDriverURL is defined in record_mode.go

// GetObservability proxies GET /observability requests to the playwright-driver.
// Query parameters are forwarded (depth, no_cache).
func (h *Handler) GetObservability(w http.ResponseWriter, r *http.Request) {
	driverURL := getPlaywrightDriverURL()
	targetURL := driverURL + "/observability"

	// Forward query parameters
	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, targetURL, nil)
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
	io.Copy(w, resp.Body)
}

// RefreshObservability proxies POST /observability/refresh requests to the playwright-driver.
func (h *Handler) RefreshObservability(w http.ResponseWriter, r *http.Request) {
	driverURL := getPlaywrightDriverURL()
	targetURL := driverURL + "/observability/refresh"

	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, targetURL, nil)
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
	io.Copy(w, resp.Body)
}

// RunDiagnostics proxies POST /observability/diagnostics/run requests to the playwright-driver.
func (h *Handler) RunDiagnostics(w http.ResponseWriter, r *http.Request) {
	driverURL := getPlaywrightDriverURL()
	targetURL := driverURL + "/observability/diagnostics/run"

	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, targetURL, r.Body)
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
	io.Copy(w, resp.Body)
}

// GetSessionList proxies GET /observability/sessions requests to the playwright-driver.
func (h *Handler) GetSessionList(w http.ResponseWriter, r *http.Request) {
	driverURL := getPlaywrightDriverURL()
	targetURL := driverURL + "/observability/sessions"

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, targetURL, nil)
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
	io.Copy(w, resp.Body)
}

// RunCleanup proxies POST /observability/cleanup/run requests to the playwright-driver.
func (h *Handler) RunCleanup(w http.ResponseWriter, r *http.Request) {
	driverURL := getPlaywrightDriverURL()
	targetURL := driverURL + "/observability/cleanup/run"

	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, targetURL, nil)
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
	io.Copy(w, resp.Body)
}

// GetMetrics proxies GET /observability/metrics requests to the playwright-driver.
// Returns metrics in JSON format (parsed from Prometheus text format).
func (h *Handler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	driverURL := getPlaywrightDriverURL()
	targetURL := driverURL + "/observability/metrics"

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, targetURL, nil)
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
	io.Copy(w, resp.Body)
}
