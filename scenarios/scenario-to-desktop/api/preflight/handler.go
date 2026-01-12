package preflight

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

	"github.com/gorilla/mux"

	httputil "scenario-to-desktop-api/shared/http"

	bundlemanifest "scenario-to-desktop-runtime/manifest"
)

// Handler provides HTTP handlers for preflight operations.
type Handler struct {
	service Service
}

// NewHandler creates a new preflight handler.
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers preflight routes on the given router.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	// Synchronous bundle preflight validation
	r.HandleFunc("/api/v1/desktop/preflight", h.BundleHandler).Methods("POST")

	// Async preflight job management
	r.HandleFunc("/api/v1/desktop/preflight/start", h.StartHandler).Methods("POST")
	r.HandleFunc("/api/v1/desktop/preflight/status", h.StatusHandler).Methods("GET")
	r.HandleFunc("/api/v1/desktop/preflight/health", h.HealthHandler).Methods("GET")

	// Bundle manifest inspection
	r.HandleFunc("/api/v1/desktop/bundle-manifest", h.ManifestHandler).Methods("POST")
}

// BundleHandler handles synchronous bundle preflight validation.
// POST /api/v1/preflight/bundle
func (h *Handler) BundleHandler(w http.ResponseWriter, r *http.Request) {
	var request Request
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	response, err := h.service.RunBundlePreflight(request)
	if err != nil {
		var pErr *StatusError
		if errors.As(err, &pErr) {
			http.Error(w, pErr.Err.Error(), pErr.Status)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, response)
}

// StartHandler starts an async preflight job.
// POST /api/v1/preflight/start
func (h *Handler) StartHandler(w http.ResponseWriter, r *http.Request) {
	var request Request
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	job := h.service.CreateJob()
	go h.service.RunPreflightJob(job.ID, request)

	httputil.WriteJSON(w, http.StatusOK, JobStartResponse{JobID: job.ID})
}

// StatusHandler returns the status of an async preflight job.
// GET /api/v1/preflight/status?job_id=...
func (h *Handler) StatusHandler(w http.ResponseWriter, r *http.Request) {
	jobID := r.URL.Query().Get("job_id")
	if jobID == "" {
		http.Error(w, "job_id is required", http.StatusBadRequest)
		return
	}

	job, ok := h.service.GetJob(jobID)
	if !ok {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}

	steps := make([]Step, 0, len(job.Steps))
	order := []string{"validation", "runtime", "secrets", "services", "diagnostics"}
	for _, id := range order {
		step, exists := job.Steps[id]
		if exists {
			steps = append(steps, step)
		}
	}
	for id, step := range job.Steps {
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

	resp := JobStatusResponse{
		JobID:     job.ID,
		Status:    job.Status,
		Steps:     steps,
		Result:    job.Result,
		Error:     job.Err,
		StartedAt: job.StartedAt.Format(time.RFC3339),
		UpdatedAt: job.UpdatedAt.Format(time.RFC3339),
	}

	httputil.WriteJSON(w, http.StatusOK, resp)
}

// HealthHandler proxies health checks to a running preflight session.
// GET /api/v1/preflight/health?session_id=...&service_id=...
func (h *Handler) HealthHandler(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session_id")
	serviceID := r.URL.Query().Get("service_id")
	if sessionID == "" || serviceID == "" {
		http.Error(w, "session_id and service_id are required", http.StatusBadRequest)
		return
	}

	session, ok := h.service.GetSession(sessionID)
	if !ok {
		http.Error(w, "session not found or expired", http.StatusNotFound)
		return
	}

	svc, ok := findManifestService(session.Manifest, serviceID)
	if !ok {
		httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"service_id": serviceID,
			"supported":  false,
			"message":    "service not found in manifest",
		})
		return
	}

	// Check if service has HTTP health check configured
	if svc.Health.Type != "http" {
		httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"service_id":  serviceID,
			"supported":   false,
			"health_type": svc.Health.Type,
			"message":     "health proxy only supports http health checks",
		})
		return
	}
	if svc.Health.PortName == "" {
		httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"service_id": serviceID,
			"supported":  false,
			"message":    "health port_name is required",
		})
		return
	}

	healthPath := strings.TrimSpace(svc.Health.Path)
	if healthPath == "" {
		httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
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
	if _, err := fetchJSON(client, session.BaseURL, session.Token, "/ports", http.MethodGet, nil, &portsResp, nil); err != nil {
		httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"service_id": serviceID,
			"supported":  true,
			"message":    fmt.Sprintf("failed to fetch ports: %v", err),
		})
		return
	}
	port := portsResp.Services[serviceID][svc.Health.PortName]
	if port == 0 {
		httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
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
		httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
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

	httputil.WriteJSON(w, http.StatusOK, response)
}

// ManifestHandler loads and returns a bundle manifest for UI display.
// POST /api/v1/preflight/manifest
func (h *Handler) ManifestHandler(w http.ResponseWriter, r *http.Request) {
	var request ManifestRequest
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

	httputil.WriteJSON(w, http.StatusOK, ManifestResponse{
		Path:     manifestPath,
		Manifest: manifest,
	})
}

// Helper functions

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

// fetchJSON performs an HTTP request and decodes the JSON response.
// This is a local helper duplicated from runtime_client.go for handler use.
func fetchJSON(client *http.Client, baseURL, token, path, method string, payload interface{}, out interface{}, allow map[int]bool) (int, error) {
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return 0, err
		}
		body = strings.NewReader(string(data))
	}

	req, err := http.NewRequest(method, baseURL+path, body)
	if err != nil {
		return 0, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if allow != nil && allow[resp.StatusCode] {
			if out != nil {
				if decodeErr := json.NewDecoder(resp.Body).Decode(out); decodeErr != nil {
					return resp.StatusCode, decodeErr
				}
			}
			return resp.StatusCode, nil
		}
		bodyBytes, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(bodyBytes)))
	}

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return resp.StatusCode, err
		}
	}
	return resp.StatusCode, nil
}
