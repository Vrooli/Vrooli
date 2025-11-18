package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// probeEndpointsHandler validates that the provided UI/API URLs respond before we generate a thin client.
func (s *Server) probeEndpointsHandler(w http.ResponseWriter, r *http.Request) {
	var request struct {
		ProxyURL  string `json:"proxy_url"`
		ServerURL string `json:"server_url"`
		APIURL    string `json:"api_url"`
		TimeoutMs int    `json:"timeout_ms"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid JSON payload", http.StatusBadRequest)
		return
	}
	if request.ProxyURL != "" {
		normalized, err := normalizeProxyURL(request.ProxyURL)
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid proxy_url: %s", err.Error()), http.StatusBadRequest)
			return
		}
		request.ProxyURL = normalized
		request.ServerURL = normalized
		if request.APIURL == "" {
			request.APIURL = proxyAPIURL(normalized)
		}
	}

	if request.ServerURL == "" && request.APIURL == "" {
		http.Error(w, "provide at least a server_url or api_url to probe", http.StatusBadRequest)
		return
	}

	timeout := time.Duration(request.TimeoutMs) * time.Millisecond
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	client := &http.Client{Timeout: timeout}
	probe := func(target string) map[string]any {
		if target == "" {
			return map[string]any{"status": "skipped", "message": "no URL provided"}
		}
		if _, err := url.ParseRequestURI(target); err != nil {
			return map[string]any{"status": "error", "message": fmt.Sprintf("invalid URL: %v", err)}
		}

		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, target, nil)
		if err != nil {
			return map[string]any{"status": "error", "message": err.Error()}
		}

		resp, err := client.Do(req)
		if err != nil {
			return map[string]any{"status": "error", "message": err.Error()}
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 400 {
			return map[string]any{"status": "ok", "status_code": resp.StatusCode}
		}
		return map[string]any{
			"status":      "error",
			"status_code": resp.StatusCode,
			"message":     fmt.Sprintf("server returned %d", resp.StatusCode),
		}
	}

	response := map[string]any{
		"proxy_url": request.ProxyURL,
		"server":    probe(request.ServerURL),
		"api":       probe(request.APIURL),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
