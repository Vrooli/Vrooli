package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// VisitedTrackerHandlers provides proxy endpoints to the visited-tracker scenario
type VisitedTrackerHandlers struct {
	projectRoot string
}

// NewVisitedTrackerHandlers creates a new instance
func NewVisitedTrackerHandlers(projectRoot string) *VisitedTrackerHandlers {
	return &VisitedTrackerHandlers{
		projectRoot: projectRoot,
	}
}

// getVisitedTrackerPort resolves the visited-tracker API port dynamically
func (h *VisitedTrackerHandlers) getVisitedTrackerPort() (string, error) {
	cmd := exec.Command("vrooli", "scenario", "port", "visited-tracker", "API_PORT")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to resolve visited-tracker port: %w (output: %s)", err, string(output))
	}

	port := strings.TrimSpace(string(output))
	if port == "" {
		return "", fmt.Errorf("visited-tracker port returned empty")
	}

	return port, nil
}

// proxyToVisitedTracker forwards requests to visited-tracker with proper error handling
func (h *VisitedTrackerHandlers) proxyToVisitedTracker(w http.ResponseWriter, r *http.Request, path string) {
	port, err := h.getVisitedTrackerPort()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to resolve visited-tracker port: %v", err), http.StatusServiceUnavailable)
		return
	}

	targetURL := fmt.Sprintf("http://localhost:%s%s", port, path)

	// Add query parameters if present
	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}

	// Create proxy request
	proxyReq, err := http.NewRequest(r.Method, targetURL, r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create proxy request: %v", err), http.StatusInternalServerError)
		return
	}

	// Copy headers (except Host)
	for key, values := range r.Header {
		if key == "Host" {
			continue
		}
		for _, value := range values {
			proxyReq.Header.Add(key, value)
		}
	}

	// Execute request with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Do(proxyReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to proxy request to visited-tracker: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Set status code
	w.WriteHeader(resp.StatusCode)

	// Stream response body
	io.Copy(w, resp.Body)
}

// ListCampaignsHandler proxies GET /api/v1/campaigns?target=<path>
func (h *VisitedTrackerHandlers) ListCampaignsHandler(w http.ResponseWriter, r *http.Request) {
	h.proxyToVisitedTracker(w, r, "/api/v1/campaigns")
}

// GetCampaignHandler proxies GET /api/v1/campaigns/{id}
func (h *VisitedTrackerHandlers) GetCampaignHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	h.proxyToVisitedTracker(w, r, fmt.Sprintf("/api/v1/campaigns/%s", url.PathEscape(id)))
}

// DeleteCampaignHandler proxies DELETE /api/v1/campaigns/{id}
func (h *VisitedTrackerHandlers) DeleteCampaignHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	h.proxyToVisitedTracker(w, r, fmt.Sprintf("/api/v1/campaigns/%s", url.PathEscape(id)))
}

// ResetCampaignHandler proxies POST /api/v1/campaigns/{id}/reset
func (h *VisitedTrackerHandlers) ResetCampaignHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	h.proxyToVisitedTracker(w, r, fmt.Sprintf("/api/v1/campaigns/%s/reset", url.PathEscape(id)))
}

// GetVisitedTrackerUIPortHandler returns the UI port for visited-tracker
func (h *VisitedTrackerHandlers) GetVisitedTrackerUIPortHandler(w http.ResponseWriter, r *http.Request) {
	cmd := exec.Command("vrooli", "scenario", "port", "visited-tracker", "UI_PORT")
	output, err := cmd.CombinedOutput()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to resolve visited-tracker UI port: %v", err), http.StatusServiceUnavailable)
		return
	}

	port := strings.TrimSpace(string(output))
	if port == "" {
		http.Error(w, "visited-tracker UI port returned empty", http.StatusServiceUnavailable)
		return
	}

	response := map[string]string{
		"port": port,
		"url":  fmt.Sprintf("http://localhost:%s", port),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetCampaignsForTargetHandler returns campaigns filtered by target location
func (h *VisitedTrackerHandlers) GetCampaignsForTargetHandler(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Query().Get("target")
	if target == "" {
		http.Error(w, "target query parameter required", http.StatusBadRequest)
		return
	}

	port, err := h.getVisitedTrackerPort()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to resolve visited-tracker port: %v", err), http.StatusServiceUnavailable)
		return
	}

	// Query campaigns from visited-tracker
	targetURL := fmt.Sprintf("http://localhost:%s/api/v1/campaigns", port)
	resp, err := http.Get(targetURL)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch campaigns from visited-tracker: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		http.Error(w, fmt.Sprintf("Visited-tracker returned error: %s", string(body)), resp.StatusCode)
		return
	}

	// Parse response - visited-tracker may return array or object with campaigns field
	var campaigns []map[string]interface{}

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read campaigns response: %v", err), http.StatusInternalServerError)
		return
	}

	// Try to unmarshal as array first
	if err := json.Unmarshal(bodyBytes, &campaigns); err != nil {
		// If that fails, try as object with campaigns field
		var responseObj map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &responseObj); err != nil {
			http.Error(w, fmt.Sprintf("Failed to decode campaigns response: %v", err), http.StatusInternalServerError)
			return
		}

		// Extract campaigns array from object
		if campaignsField, ok := responseObj["campaigns"]; ok {
			if campaignsArray, ok := campaignsField.([]interface{}); ok {
				campaigns = make([]map[string]interface{}, 0, len(campaignsArray))
				for _, item := range campaignsArray {
					if campaign, ok := item.(map[string]interface{}); ok {
						campaigns = append(campaigns, campaign)
					}
				}
			}
		}
	}

	// Filter campaigns by target location
	filtered := make([]map[string]interface{}, 0)
	for _, campaign := range campaigns {
		location, ok := campaign["location"].(string)
		if ok && strings.Contains(location, target) {
			filtered = append(filtered, campaign)
		}
	}

	// Ensure we always return an array (even if empty)
	w.Header().Set("Content-Type", "application/json")
	if filtered == nil {
		filtered = make([]map[string]interface{}, 0)
	}
	json.NewEncoder(w).Encode(filtered)
}
