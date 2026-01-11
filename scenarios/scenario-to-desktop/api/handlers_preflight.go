// Package main provides preflight validation handlers for bundle deployments.
// This file contains handlers for validating and testing bundled desktop applications
// before packaging, including session management for persistent preflight environments.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	bundlemanifest "scenario-to-desktop-runtime/manifest"
)

// ============================================================================
// HTTP Handlers
// ============================================================================

// preflightBundleHandler handles synchronous bundle preflight validation.
func (s *Server) preflightBundleHandler(w http.ResponseWriter, r *http.Request) {
	var request BundlePreflightRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	response, err := s.preflightService.RunBundlePreflight(request)
	if err != nil {
		var pErr *preflightStatusError
		if errors.As(err, &pErr) {
			http.Error(w, pErr.Err.Error(), pErr.Status)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// preflightStartHandler starts an async preflight job.
func (s *Server) preflightStartHandler(w http.ResponseWriter, r *http.Request) {
	var request BundlePreflightRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	job := s.preflightService.CreateJob()
	go s.preflightService.RunPreflightJob(job.id, request)

	writeJSONResponse(w, http.StatusOK, BundlePreflightJobStartResponse{JobID: job.id})
}

// preflightStatusHandler returns the status of an async preflight job.
func (s *Server) preflightStatusHandler(w http.ResponseWriter, r *http.Request) {
	jobID := r.URL.Query().Get("job_id")
	if jobID == "" {
		http.Error(w, "job_id is required", http.StatusBadRequest)
		return
	}

	job, ok := s.preflightService.GetJob(jobID)
	if !ok {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}

	steps := make([]BundlePreflightStep, 0, len(job.steps))
	order := []string{"validation", "runtime", "secrets", "services", "diagnostics"}
	for _, id := range order {
		step, exists := job.steps[id]
		if exists {
			steps = append(steps, step)
		}
	}
	for id, step := range job.steps {
		found := false
		for _, o := range order {
			if o == id {
				found = true
				break
			}
		}
		if !found {
			steps = append(steps, step)
		}
	}

	resp := BundlePreflightJobStatusResponse{
		JobID:     job.id,
		Status:    job.status,
		Steps:     steps,
		Result:    job.result,
		Error:     job.err,
		StartedAt: job.startedAt.Format(time.RFC3339),
		UpdatedAt: job.updatedAt.Format(time.RFC3339),
	}

	writeJSONResponse(w, http.StatusOK, resp)
}

// preflightHealthHandler proxies health checks to a running preflight session.
func (s *Server) preflightHealthHandler(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session_id")
	serviceID := r.URL.Query().Get("service_id")
	if sessionID == "" || serviceID == "" {
		http.Error(w, "session_id and service_id are required", http.StatusBadRequest)
		return
	}

	session, ok := s.preflightService.GetSession(sessionID)
	if !ok {
		http.Error(w, "session not found or expired", http.StatusNotFound)
		return
	}

	svc, ok := findManifestService(session.manifest, serviceID)
	if !ok {
		writeJSONResponse(w, http.StatusOK, map[string]interface{}{
			"service_id": serviceID,
			"supported":  false,
			"message":    "service not found in manifest",
		})
		return
	}

	// Check if service has HTTP health check configured
	if svc.Health.Type != "http" {
		writeJSONResponse(w, http.StatusOK, map[string]interface{}{
			"service_id":  serviceID,
			"supported":   false,
			"health_type": svc.Health.Type,
			"message":     "health proxy only supports http health checks",
		})
		return
	}
	if svc.Health.PortName == "" {
		writeJSONResponse(w, http.StatusOK, map[string]interface{}{
			"service_id": serviceID,
			"supported":  false,
			"message":    "health port_name is required",
		})
		return
	}

	healthPath := strings.TrimSpace(svc.Health.Path)
	if healthPath == "" {
		writeJSONResponse(w, http.StatusOK, map[string]interface{}{
			"service_id": serviceID,
			"supported":  false,
			"message":    "health path is required",
		})
		return
	}
	if !strings.HasPrefix(healthPath, "/") {
		healthPath = "/" + healthPath
	}

	// Get port from runtime
	timeoutMs := svc.Health.TimeoutMs
	if timeoutMs <= 0 {
		timeoutMs = 2000
	}
	client := &http.Client{Timeout: time.Duration(timeoutMs) * time.Millisecond}
	var portsResp struct {
		Services map[string]map[string]int `json:"services"`
	}
	if _, err := fetchJSON(client, session.baseURL, session.token, "/ports", http.MethodGet, nil, &portsResp, nil); err != nil {
		writeJSONResponse(w, http.StatusOK, map[string]interface{}{
			"service_id": serviceID,
			"supported":  true,
			"message":    fmt.Sprintf("failed to fetch ports: %v", err),
		})
		return
	}
	port := portsResp.Services[serviceID][svc.Health.PortName]
	if port == 0 {
		writeJSONResponse(w, http.StatusOK, map[string]interface{}{
			"service_id": serviceID,
			"supported":  true,
			"message":    fmt.Sprintf("port not found: %s", svc.Health.PortName),
		})
		return
	}

	healthURL := fmt.Sprintf("http://localhost:%d%s", port, healthPath)
	start := time.Now()
	resp, err := client.Get(healthURL)
	if err != nil {
		writeJSONResponse(w, http.StatusOK, map[string]interface{}{
			"service_id":  serviceID,
			"supported":   true,
			"health_type": svc.Health.Type,
			"url":         healthURL,
			"message":     fmt.Sprintf("health check failed: %v", err),
			"fetched_at":  time.Now().Format(time.RFC3339),
			"elapsed_ms":  time.Since(start).Milliseconds(),
		})
		return
	}
	defer resp.Body.Close()

	const maxBody = 8 * 1024
	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, maxBody+1))
	truncated := len(bodyBytes) > maxBody
	if truncated {
		bodyBytes = bodyBytes[:maxBody]
	}
	bodyText := string(bodyBytes)

	response := map[string]interface{}{
		"service_id":   serviceID,
		"supported":    true,
		"health_type":  svc.Health.Type,
		"url":          healthURL,
		"status_code":  resp.StatusCode,
		"status":       resp.Status,
		"body":         bodyText,
		"content_type": resp.Header.Get("Content-Type"),
		"truncated":    truncated,
		"fetched_at":   time.Now().Format(time.RFC3339),
		"elapsed_ms":   time.Since(start).Milliseconds(),
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// bundleManifestHandler loads and returns a bundle manifest for UI display.
func (s *Server) bundleManifestHandler(w http.ResponseWriter, r *http.Request) {
	var request BundleManifestRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(request.BundleManifestPath) == "" {
		http.Error(w, "bundle_manifest_path is required", http.StatusBadRequest)
		return
	}

	manifestPath, err := filepath.Abs(request.BundleManifestPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("resolve bundle_manifest_path: %v", err), http.StatusBadRequest)
		return
	}
	if _, err := os.Stat(manifestPath); err != nil {
		http.Error(w, fmt.Sprintf("bundle manifest not found: %v", err), http.StatusBadRequest)
		return
	}

	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("read bundle manifest: %v", err), http.StatusInternalServerError)
		return
	}

	var manifest json.RawMessage
	if err := json.Unmarshal(raw, &manifest); err != nil {
		http.Error(w, fmt.Sprintf("parse bundle manifest: %v", err), http.StatusBadRequest)
		return
	}

	writeJSONResponse(w, http.StatusOK, BundleManifestResponse{
		Path:     manifestPath,
		Manifest: manifest,
	})
}

// ============================================================================
// Preflight Core Logic
// ============================================================================

func findManifestService(manifest *bundlemanifest.Manifest, serviceID string) (*bundlemanifest.Service, bool) {
	if manifest == nil {
		return nil, false
	}
	for i := range manifest.Services {
		service := &manifest.Services[i]
		if service.ID == serviceID {
			return service, true
		}
	}
	return nil, false
}
