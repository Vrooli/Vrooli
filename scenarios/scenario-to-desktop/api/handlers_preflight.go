// Package main provides preflight validation handlers for bundle deployments.
// This file contains handlers for validating and testing bundled desktop applications
// before packaging, including session management for persistent preflight environments.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	bundleruntime "scenario-to-desktop-runtime"
	runtimeapi "scenario-to-desktop-runtime/api"
	bundlemanifest "scenario-to-desktop-runtime/manifest"
)

// ============================================================================
// Types
// ============================================================================

type preflightStatusError struct {
	Status int
	Err    error
}

func (e *preflightStatusError) Error() string {
	return e.Err.Error()
}

type preflightIssue struct {
	status string
	detail string
}

type preflightSession struct {
	id         string
	manifest   *bundlemanifest.Manifest
	bundleDir  string
	appData    string
	supervisor *bundleruntime.Supervisor
	baseURL    string
	token      string
	createdAt  time.Time
	expiresAt  time.Time
}

type preflightJob struct {
	id        string
	status    string
	steps     map[string]BundlePreflightStep
	result    *BundlePreflightResponse
	err       string
	startedAt time.Time
	updatedAt time.Time
}

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

	response, err := s.runBundlePreflight(request)
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

	job := s.createPreflightJob()
	go s.runPreflightJob(job.id, request)

	writeJSONResponse(w, http.StatusOK, BundlePreflightJobStartResponse{JobID: job.id})
}

// preflightStatusHandler returns the status of an async preflight job.
func (s *Server) preflightStatusHandler(w http.ResponseWriter, r *http.Request) {
	jobID := r.URL.Query().Get("job_id")
	if jobID == "" {
		http.Error(w, "job_id is required", http.StatusBadRequest)
		return
	}

	job, ok := s.getPreflightJob(jobID)
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

	session, ok := s.getPreflightSession(sessionID)
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

// runBundlePreflight executes synchronous bundle preflight validation.
func (s *Server) runBundlePreflight(request BundlePreflightRequest) (*BundlePreflightResponse, error) {
	if strings.TrimSpace(request.BundleManifestPath) == "" {
		return nil, &preflightStatusError{Status: http.StatusBadRequest, Err: errors.New("bundle_manifest_path is required")}
	}

	if request.SessionStop && strings.TrimSpace(request.SessionID) != "" {
		if s.stopPreflightSession(request.SessionID) {
			return &BundlePreflightResponse{Status: "stopped", SessionID: request.SessionID}, nil
		}
		return nil, &preflightStatusError{Status: http.StatusNotFound, Err: fmt.Errorf("preflight session not found: %s", request.SessionID)}
	}

	if request.StatusOnly && strings.TrimSpace(request.SessionID) == "" {
		return nil, &preflightStatusError{Status: http.StatusBadRequest, Err: errors.New("session_id is required for status_only")}
	}

	manifestPath, err := filepath.Abs(request.BundleManifestPath)
	if err != nil {
		return nil, &preflightStatusError{Status: http.StatusBadRequest, Err: fmt.Errorf("resolve bundle_manifest_path: %w", err)}
	}
	if _, err := os.Stat(manifestPath); err != nil {
		return nil, &preflightStatusError{Status: http.StatusBadRequest, Err: fmt.Errorf("bundle manifest not found: %w", err)}
	}

	m, err := bundlemanifest.LoadManifest(manifestPath)
	if err != nil {
		return nil, &preflightStatusError{Status: http.StatusBadRequest, Err: fmt.Errorf("load bundle manifest: %w", err)}
	}
	if err := m.Validate(runtime.GOOS, runtime.GOARCH); err != nil {
		return nil, &preflightStatusError{Status: http.StatusBadRequest, Err: fmt.Errorf("validate bundle manifest: %w", err)}
	}

	bundleRoot := strings.TrimSpace(request.BundleRoot)
	if bundleRoot == "" {
		bundleRoot = filepath.Dir(manifestPath)
	}
	bundleRoot, err = filepath.Abs(bundleRoot)
	if err != nil {
		return nil, &preflightStatusError{Status: http.StatusBadRequest, Err: fmt.Errorf("resolve bundle_root: %w", err)}
	}

	timeout := time.Duration(request.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	if timeout > 2*time.Minute {
		timeout = 2 * time.Minute
	}

	var session *preflightSession
	if request.StatusOnly {
		var ok bool
		session, ok = s.getPreflightSession(request.SessionID)
		if !ok {
			return nil, &preflightStatusError{Status: http.StatusNotFound, Err: fmt.Errorf("preflight session not found: %s", request.SessionID)}
		}
		s.refreshPreflightSession(session, request.SessionTTLSeconds)
	} else if request.StartServices {
		if existingID := strings.TrimSpace(request.SessionID); existingID != "" {
			s.stopPreflightSession(existingID)
		}
		session, err = s.createPreflightSession(m, bundleRoot, request.SessionTTLSeconds)
		if err != nil {
			return nil, err
		}
	}

	if session == nil {
		appData, err := os.MkdirTemp("", "s2d-preflight-*")
		if err != nil {
			return nil, fmt.Errorf("create preflight app data: %w", err)
		}
		defer os.RemoveAll(appData)

		supervisor, err := bundleruntime.NewSupervisor(bundleruntime.Options{
			Manifest:   m,
			BundlePath: bundleRoot,
			AppDataDir: appData,
			DryRun:     !request.StartServices,
		})
		if err != nil {
			return nil, fmt.Errorf("init runtime: %w", err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		if err := supervisor.Start(ctx); err != nil {
			return nil, fmt.Errorf("start runtime: %w", err)
		}
		defer func() {
			_ = supervisor.Shutdown(context.Background())
		}()

		fileTimeout := timeout / 3
		if fileTimeout < 500*time.Millisecond {
			fileTimeout = 500 * time.Millisecond
		}

		tokenPath := bundlemanifest.ResolvePath(appData, m.IPC.AuthTokenRel)
		tokenBytes, err := readFileWithRetry(tokenPath, fileTimeout)
		if err != nil {
			return nil, fmt.Errorf("read auth token: %w", err)
		}
		token := strings.TrimSpace(string(tokenBytes))

		portPath := filepath.Join(appData, "runtime", "ipc_port")
		port, err := readPortFileWithRetry(portPath, fileTimeout)
		if err != nil {
			return nil, fmt.Errorf("read ipc_port: %w", err)
		}

		baseURL := fmt.Sprintf("http://%s:%d", m.IPC.Host, port)
		client := &http.Client{Timeout: 2 * time.Second}

		if err := waitForRuntimeHealth(client, baseURL, timeout); err != nil {
			return nil, err
		}

		session = &preflightSession{
			manifest:  m,
			bundleDir: bundleRoot,
			baseURL:   baseURL,
			token:     token,
			createdAt: time.Now(),
		}
	}

	client := &http.Client{Timeout: 2 * time.Second}
	baseURL := session.baseURL
	token := session.token

	runtimeStatus, statusErr := fetchRuntimeStatus(client, baseURL, token)
	if statusErr == nil && runtimeStatus != nil {
		runtimeStatus.BundleRoot = bundleRoot
	}

	fingerprints := collectServiceFingerprints(m, bundleRoot)

	if len(request.Secrets) > 0 {
		filtered := map[string]string{}
		for key, value := range request.Secrets {
			if strings.TrimSpace(value) == "" {
				continue
			}
			filtered[key] = value
		}
		payload := map[string]map[string]string{"secrets": filtered}
		if len(filtered) == 0 {
			payload = nil
		}
		if payload != nil {
			if _, err := fetchJSON(client, baseURL, token, "/secrets", http.MethodPost, payload, nil, nil); err != nil {
				return nil, fmt.Errorf("apply secrets: %w", err)
			}
		}
	}

	var validation *runtimeapi.BundleValidationResult
	if !request.StatusOnly {
		var validationValue runtimeapi.BundleValidationResult
		allowStatus := map[int]bool{http.StatusUnprocessableEntity: true}
		if _, err := fetchJSON(client, baseURL, token, "/validate", http.MethodGet, nil, &validationValue, allowStatus); err != nil {
			return nil, fmt.Errorf("validate bundle: %w", err)
		}
		validation = &validationValue
	}

	var secretsResp struct {
		Secrets []BundlePreflightSecret `json:"secrets"`
	}
	if !request.StatusOnly {
		if _, err := fetchJSON(client, baseURL, token, "/secrets", http.MethodGet, nil, &secretsResp, nil); err != nil {
			return nil, fmt.Errorf("fetch secrets: %w", err)
		}
	}

	ready, waitedSeconds, err := fetchReadyWithPolling(client, baseURL, token, request, timeout, m)
	if err != nil {
		return nil, fmt.Errorf("fetch readiness: %w", err)
	}
	ready.SnapshotAt = time.Now().Format(time.RFC3339)
	ready.WaitedSeconds = waitedSeconds

	var portsResp struct {
		Services map[string]map[string]int `json:"services"`
	}
	if _, err := fetchJSON(client, baseURL, token, "/ports", http.MethodGet, nil, &portsResp, nil); err != nil {
		return nil, fmt.Errorf("fetch ports: %w", err)
	}

	var telemetryResp BundlePreflightTelemetry
	if _, err := fetchJSON(client, baseURL, token, "/telemetry", http.MethodGet, nil, &telemetryResp, nil); err != nil {
		return nil, fmt.Errorf("fetch telemetry: %w", err)
	}

	logTails := collectLogTails(client, baseURL, token, m, request)
	checks := buildPreflightChecks(m, validation, &ready, secretsResp.Secrets, portsResp.Services, &telemetryResp, logTails, request)

	sessionID := ""
	expiresAt := ""
	if session != nil && session.id != "" {
		sessionID = session.id
		if !session.expiresAt.IsZero() {
			expiresAt = session.expiresAt.Format(time.RFC3339)
		}
	}

	return &BundlePreflightResponse{
		Status:       "ok",
		Validation:   validation,
		Ready:        &ready,
		Secrets:      secretsResp.Secrets,
		Ports:        portsResp.Services,
		Telemetry:    &telemetryResp,
		LogTails:     logTails,
		Checks:       checks,
		Runtime:      runtimeStatus,
		Fingerprints: fingerprints,
		Errors: func() []string {
			if statusErr != nil {
				return []string{fmt.Sprintf("runtime status: %v", statusErr)}
			}
			return nil
		}(),
		SessionID: sessionID,
		ExpiresAt: expiresAt,
	}, nil
}

// runPreflightJob runs an async preflight job with step-by-step progress.
func (s *Server) runPreflightJob(jobID string, request BundlePreflightRequest) {
	fail := func(stepID string, err error) {
		s.setPreflightJobStep(jobID, stepID, "fail", err.Error())
		s.finishPreflightJob(jobID, "failed", err.Error())
	}

	if strings.TrimSpace(request.BundleManifestPath) == "" {
		fail("validation", errors.New("bundle_manifest_path is required"))
		return
	}

	s.setPreflightJobStep(jobID, "validation", "running", "loading manifest")

	manifestPath, err := filepath.Abs(request.BundleManifestPath)
	if err != nil {
		fail("validation", fmt.Errorf("resolve bundle_manifest_path: %w", err))
		return
	}
	if _, err := os.Stat(manifestPath); err != nil {
		fail("validation", fmt.Errorf("bundle manifest not found: %w", err))
		return
	}

	m, err := bundlemanifest.LoadManifest(manifestPath)
	if err != nil {
		fail("validation", fmt.Errorf("load bundle manifest: %w", err))
		return
	}
	if err := m.Validate(runtime.GOOS, runtime.GOARCH); err != nil {
		fail("validation", fmt.Errorf("validate bundle manifest: %w", err))
		return
	}

	bundleRoot := strings.TrimSpace(request.BundleRoot)
	if bundleRoot == "" {
		bundleRoot = filepath.Dir(manifestPath)
	}
	bundleRoot, err = filepath.Abs(bundleRoot)
	if err != nil {
		fail("validation", fmt.Errorf("resolve bundle_root: %w", err))
		return
	}

	timeout := time.Duration(request.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	if timeout > 2*time.Minute {
		timeout = 2 * time.Minute
	}

	s.setPreflightJobStep(jobID, "runtime", "running", "starting runtime control API")

	var session *preflightSession
	if request.StartServices {
		session, err = s.createPreflightSession(m, bundleRoot, request.SessionTTLSeconds)
		if err != nil {
			fail("runtime", err)
			return
		}
	} else {
		appData, err := os.MkdirTemp("", "s2d-preflight-*")
		if err != nil {
			fail("runtime", fmt.Errorf("create preflight app data: %w", err))
			return
		}
		defer os.RemoveAll(appData)

		supervisor, err := bundleruntime.NewSupervisor(bundleruntime.Options{
			Manifest:   m,
			BundlePath: bundleRoot,
			AppDataDir: appData,
			DryRun:     true,
		})
		if err != nil {
			fail("runtime", fmt.Errorf("init runtime: %w", err))
			return
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		if err := supervisor.Start(ctx); err != nil {
			fail("runtime", fmt.Errorf("start runtime: %w", err))
			return
		}
		defer func() {
			_ = supervisor.Shutdown(context.Background())
		}()

		fileTimeout := timeout / 3
		if fileTimeout < 500*time.Millisecond {
			fileTimeout = 500 * time.Millisecond
		}

		tokenPath := bundlemanifest.ResolvePath(appData, m.IPC.AuthTokenRel)
		tokenBytes, err := readFileWithRetry(tokenPath, fileTimeout)
		if err != nil {
			fail("runtime", fmt.Errorf("read auth token: %w", err))
			return
		}
		token := strings.TrimSpace(string(tokenBytes))

		portPath := filepath.Join(appData, "runtime", "ipc_port")
		port, err := readPortFileWithRetry(portPath, fileTimeout)
		if err != nil {
			fail("runtime", fmt.Errorf("read ipc_port: %w", err))
			return
		}

		baseURL := fmt.Sprintf("http://%s:%d", m.IPC.Host, port)
		client := &http.Client{Timeout: 2 * time.Second}
		if err := waitForRuntimeHealth(client, baseURL, timeout); err != nil {
			fail("runtime", err)
			return
		}

		session = &preflightSession{
			manifest:  m,
			bundleDir: bundleRoot,
			baseURL:   baseURL,
			token:     token,
			createdAt: time.Now(),
		}
	}

	s.setPreflightJobStep(jobID, "runtime", "pass", "control API online")

	client := &http.Client{Timeout: 2 * time.Second}
	baseURL := session.baseURL
	token := session.token

	runtimeStatus, statusErr := fetchRuntimeStatus(client, baseURL, token)
	if statusErr == nil {
		if runtimeStatus != nil {
			runtimeStatus.BundleRoot = bundleRoot
		}
		s.setPreflightJobResult(jobID, func(prev *BundlePreflightResponse) *BundlePreflightResponse {
			return updatePreflightResult(prev, func(next *BundlePreflightResponse) {
				next.Runtime = runtimeStatus
			})
		})
	} else {
		s.setPreflightJobResult(jobID, func(prev *BundlePreflightResponse) *BundlePreflightResponse {
			return updatePreflightResult(prev, func(next *BundlePreflightResponse) {
				next.Errors = append(next.Errors, fmt.Sprintf("runtime status: %v", statusErr))
			})
		})
	}

	fingerprints := collectServiceFingerprints(m, bundleRoot)
	if len(fingerprints) > 0 {
		s.setPreflightJobResult(jobID, func(prev *BundlePreflightResponse) *BundlePreflightResponse {
			return updatePreflightResult(prev, func(next *BundlePreflightResponse) {
				next.Fingerprints = fingerprints
			})
		})
	}

	s.setPreflightJobStep(jobID, "secrets", "running", "applying secrets")
	if len(request.Secrets) > 0 {
		filtered := map[string]string{}
		for key, value := range request.Secrets {
			if strings.TrimSpace(value) == "" {
				continue
			}
			filtered[key] = value
		}
		payload := map[string]map[string]string{"secrets": filtered}
		if len(filtered) == 0 {
			payload = nil
		}
		if payload != nil {
			if _, err := fetchJSON(client, baseURL, token, "/secrets", http.MethodPost, payload, nil, nil); err != nil {
				fail("secrets", fmt.Errorf("apply secrets: %w", err))
				return
			}
		}
	}

	s.setPreflightJobStep(jobID, "validation", "running", "validating bundle")
	var validation *runtimeapi.BundleValidationResult
	var validationValue runtimeapi.BundleValidationResult
	allowStatus := map[int]bool{http.StatusUnprocessableEntity: true}
	if _, err := fetchJSON(client, baseURL, token, "/validate", http.MethodGet, nil, &validationValue, allowStatus); err != nil {
		fail("validation", fmt.Errorf("validate bundle: %w", err))
		return
	}
	validation = &validationValue
	s.setPreflightJobResult(jobID, func(prev *BundlePreflightResponse) *BundlePreflightResponse {
		return updatePreflightResult(prev, func(next *BundlePreflightResponse) {
			next.Validation = validation
		})
	})
	s.setPreflightJobStep(jobID, "validation", validationStepState(validation), "")

	var secretsResp struct {
		Secrets []BundlePreflightSecret `json:"secrets"`
	}
	if _, err := fetchJSON(client, baseURL, token, "/secrets", http.MethodGet, nil, &secretsResp, nil); err != nil {
		fail("secrets", fmt.Errorf("fetch secrets: %w", err))
		return
	}
	s.setPreflightJobResult(jobID, func(prev *BundlePreflightResponse) *BundlePreflightResponse {
		return updatePreflightResult(prev, func(next *BundlePreflightResponse) {
			next.Secrets = secretsResp.Secrets
		})
	})
	s.setPreflightJobStep(jobID, "secrets", secretsStepState(secretsResp.Secrets), "")

	s.setPreflightJobStep(jobID, "services", "running", "checking readiness")
	ready, waitedSeconds, err := fetchReadyWithPolling(client, baseURL, token, request, timeout, m)
	if err != nil {
		fail("services", fmt.Errorf("fetch readiness: %w", err))
		return
	}
	ready.SnapshotAt = time.Now().Format(time.RFC3339)
	ready.WaitedSeconds = waitedSeconds
	s.setPreflightJobResult(jobID, func(prev *BundlePreflightResponse) *BundlePreflightResponse {
		return updatePreflightResult(prev, func(next *BundlePreflightResponse) {
			next.Ready = &ready
		})
	})
	s.setPreflightJobStep(jobID, "services", readinessStepState(&ready, request), "")

	s.setPreflightJobStep(jobID, "diagnostics", "running", "collecting diagnostics")
	var portsResp struct {
		Services map[string]map[string]int `json:"services"`
	}
	if _, err := fetchJSON(client, baseURL, token, "/ports", http.MethodGet, nil, &portsResp, nil); err != nil {
		fail("diagnostics", fmt.Errorf("fetch ports: %w", err))
		return
	}

	var telemetryResp BundlePreflightTelemetry
	if _, err := fetchJSON(client, baseURL, token, "/telemetry", http.MethodGet, nil, &telemetryResp, nil); err != nil {
		fail("diagnostics", fmt.Errorf("fetch telemetry: %w", err))
		return
	}

	logTails := collectLogTails(client, baseURL, token, m, request)
	checks := buildPreflightChecks(m, validation, &ready, secretsResp.Secrets, portsResp.Services, &telemetryResp, logTails, request)

	s.setPreflightJobResult(jobID, func(prev *BundlePreflightResponse) *BundlePreflightResponse {
		return updatePreflightResult(prev, func(next *BundlePreflightResponse) {
			next.Ports = portsResp.Services
			next.Telemetry = &telemetryResp
			next.LogTails = logTails
			next.Checks = checks
		})
	})
	s.setPreflightJobStep(jobID, "diagnostics", diagnosticsStepState(portsResp.Services, &telemetryResp, logTails, request), "")

	sessionID := ""
	expiresAt := ""
	if session != nil && session.id != "" {
		sessionID = session.id
		if !session.expiresAt.IsZero() {
			expiresAt = session.expiresAt.Format(time.RFC3339)
		}
	}

	s.setPreflightJobResult(jobID, func(prev *BundlePreflightResponse) *BundlePreflightResponse {
		return updatePreflightResult(prev, func(next *BundlePreflightResponse) {
			next.Status = "ok"
			next.SessionID = sessionID
			next.ExpiresAt = expiresAt
		})
	})

	s.finishPreflightJob(jobID, "completed", "")
}

// ============================================================================
// Session Management
// ============================================================================

// createPreflightSession creates a new persistent preflight session with running services.
func (s *Server) createPreflightSession(manifest *bundlemanifest.Manifest, bundleRoot string, ttlSeconds int) (*preflightSession, error) {
	return s.preflightSessions.Create(manifest, bundleRoot, ttlSeconds)
}

// getPreflightSession retrieves a preflight session by ID, checking expiry.
func (s *Server) getPreflightSession(id string) (*preflightSession, bool) {
	return s.preflightSessions.Get(id)
}

// refreshPreflightSession extends the TTL of an existing session.
func (s *Server) refreshPreflightSession(session *preflightSession, ttlSeconds int) {
	s.preflightSessions.Refresh(session, ttlSeconds)
}

// stopPreflightSession stops and cleans up a preflight session.
func (s *Server) stopPreflightSession(id string) bool {
	return s.preflightSessions.Stop(id)
}

// ============================================================================
// Job Management
// ============================================================================

// createPreflightJob creates a new async preflight job.
func (s *Server) createPreflightJob() *preflightJob {
	return s.preflightJobs.Create()
}

// getPreflightJob retrieves a preflight job by ID.
func (s *Server) getPreflightJob(id string) (*preflightJob, bool) {
	return s.preflightJobs.Get(id)
}

// updatePreflightJob updates a preflight job with a modifier function.
// setPreflightJobStep updates the state of a specific step in a job.
func (s *Server) setPreflightJobStep(id, stepID, state, detail string) {
	s.preflightJobs.SetStep(id, stepID, state, detail)
}

// setPreflightJobResult updates the result of a preflight job.
func (s *Server) setPreflightJobResult(id string, updater func(prev *BundlePreflightResponse) *BundlePreflightResponse) {
	s.preflightJobs.SetResult(id, updater)
}

// finishPreflightJob marks a job as complete with final status.
func (s *Server) finishPreflightJob(id string, status, errMsg string) {
	s.preflightJobs.Finish(id, status, errMsg)
}

// startPreflightJanitor starts the background cleanup goroutine.
func (s *Server) startPreflightJanitor() {
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			s.preflightSessions.Cleanup()
			s.preflightJobs.Cleanup()
		}
	}()
}

// ============================================================================
// Helper Functions
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

func buildPreflightChecks(manifest *bundlemanifest.Manifest, validation *runtimeapi.BundleValidationResult, ready *BundlePreflightReady, secrets []BundlePreflightSecret, ports map[string]map[string]int, telemetry *BundlePreflightTelemetry, logTails []BundlePreflightLogTail, request BundlePreflightRequest) []BundlePreflightCheck {
	checks := []BundlePreflightCheck{}
	if manifest == nil {
		return checks
	}

	addCheck := func(id, step, name, status, detail string) {
		checks = append(checks, BundlePreflightCheck{
			ID:     id,
			Step:   step,
			Name:   name,
			Status: status,
			Detail: detail,
		})
	}

	statusOnly := request.StatusOnly

	manifestStatus := "pass"
	manifestDetail := fmt.Sprintf("validated for %s/%s", runtime.GOOS, runtime.GOARCH)
	if statusOnly {
		manifestStatus = "skipped"
		manifestDetail = "status-only refresh"
	} else if validation == nil {
		manifestStatus = "fail"
		manifestDetail = "validation result missing"
	} else {
		for _, err := range validation.Errors {
			if err.Code == "manifest_invalid" {
				manifestStatus = "fail"
				manifestDetail = err.Message
				break
			}
		}
	}
	addCheck("validation.manifest", "validation", "Manifest schema", manifestStatus, manifestDetail)

	binaryErrors := map[string][]string{}
	missingBinaries := map[string]runtimeapi.MissingBinary{}
	assetIssues := map[string]preflightIssue{}
	if validation != nil && !statusOnly {
		for _, err := range validation.Errors {
			if err.Service == "" {
				continue
			}
			if strings.HasPrefix(err.Code, "binary_") {
				binaryErrors[err.Service] = append(binaryErrors[err.Service], err.Message)
			}
			if err.Path != "" && (strings.HasPrefix(err.Code, "asset_") || err.Code == "checksum_mismatch") {
				key := err.Service + "|" + err.Path
				assetIssues[key] = preflightIssue{status: "fail", detail: err.Message}
			}
		}
		for _, missing := range validation.MissingBinaries {
			missingBinaries[missing.ServiceID] = missing
		}
		for _, missing := range validation.MissingAssets {
			key := missing.ServiceID + "|" + missing.Path
			assetIssues[key] = preflightIssue{status: "fail", detail: "missing asset"}
		}
		for _, invalid := range validation.InvalidChecksums {
			key := invalid.ServiceID + "|" + invalid.Path
			assetIssues[key] = preflightIssue{
				status: "fail",
				detail: fmt.Sprintf("checksum mismatch (expected %s)", invalid.Expected),
			}
		}
		for _, warn := range validation.Warnings {
			if warn.Service == "" || warn.Path == "" {
				continue
			}
			if warn.Code == "asset_size_warning" {
				key := warn.Service + "|" + warn.Path
				if existing, ok := assetIssues[key]; !ok || existing.status != "fail" {
					assetIssues[key] = preflightIssue{status: "warning", detail: warn.Message}
				}
			}
		}
	}

	if !statusOnly {
		for _, svc := range manifest.Services {
			status := "pass"
			detail := ""
			if missing, ok := missingBinaries[svc.ID]; ok {
				status = "fail"
				detail = fmt.Sprintf("missing binary: %s (%s)", missing.Path, missing.Platform)
			}
			if errs, ok := binaryErrors[svc.ID]; ok {
				status = "fail"
				if detail == "" {
					detail = strings.Join(errs, "; ")
				} else {
					detail = detail + "; " + strings.Join(errs, "; ")
				}
			}
			addCheck("validation.binary."+svc.ID, "validation", "Binary present: "+svc.ID, status, detail)

			for _, asset := range svc.Assets {
				key := svc.ID + "|" + asset.Path
				issue, ok := assetIssues[key]
				assetStatus := "pass"
				assetDetail := ""
				if ok {
					assetStatus = issue.status
					assetDetail = issue.detail
				}
				addCheck("validation.asset."+svc.ID+"."+asset.Path, "validation", "Asset "+asset.Path+" ("+svc.ID+")", assetStatus, assetDetail)
			}
		}

		for _, secret := range secrets {
			status := "pass"
			detail := ""
			if secret.Required {
				if secret.HasValue {
					detail = "required secret provided"
				} else {
					status = "fail"
					detail = "required secret missing"
				}
			} else if secret.HasValue {
				detail = "optional secret provided"
			} else {
				status = "skipped"
				detail = "optional secret not provided"
			}
			addCheck("secrets."+secret.ID, "secrets", "Secret "+secret.ID, status, detail)
		}

		addCheck("runtime.control_api", "runtime", "Runtime control API online", "pass", "control API responded")
	} else {
		addCheck("runtime.control_api", "runtime", "Runtime control API online", "pass", "status-only refresh")
	}

	if !request.StartServices {
		addCheck("services.start", "services", "Start services", "skipped", "start_services=false")
	} else if ready == nil {
		addCheck("services.ready", "services", "Overall readiness", "warning", "readiness not reported")
	} else {
		overall := "pass"
		if !ready.Ready {
			overall = "fail"
		}
		overallDetail := ""
		if ready.WaitedSeconds > 0 {
			overallDetail = fmt.Sprintf("waited %ds for readiness", ready.WaitedSeconds)
		}
		addCheck("services.ready", "services", "Overall readiness", overall, overallDetail)
		for serviceID, status := range ready.Details {
			checkStatus := "pass"
			detail := status.Message
			if status.Skipped {
				checkStatus = "skipped"
				if detail == "" {
					detail = "service skipped"
				}
			} else if !status.Ready {
				checkStatus = "fail"
				if detail == "" {
					detail = "service not ready"
				}
			}
			if status.ExitCode != nil {
				if detail != "" {
					detail = detail + "; "
				}
				detail = fmt.Sprintf("%sexit code %d", detail, *status.ExitCode)
			}
			addCheck("services."+serviceID, "services", "Service readiness: "+serviceID, checkStatus, detail)
		}
	}

	if len(ports) > 0 {
		addCheck("diagnostics.ports", "diagnostics", "Ports reported", "pass", "")
	} else {
		addCheck("diagnostics.ports", "diagnostics", "Ports reported", "warning", "no ports reported")
	}

	if telemetry != nil && telemetry.Path != "" {
		addCheck("diagnostics.telemetry", "diagnostics", "Telemetry configured", "pass", telemetry.Path)
	} else {
		addCheck("diagnostics.telemetry", "diagnostics", "Telemetry configured", "warning", "no telemetry path")
	}

	if request.StartServices {
		if len(logTails) == 0 {
			addCheck("diagnostics.log_tails", "diagnostics", "Log tails captured", "warning", "no log tails captured")
		} else {
			for _, tail := range logTails {
				status := "pass"
				detail := ""
				if tail.Error != "" {
					status = "warning"
					detail = tail.Error
				}
				addCheck("diagnostics.log_tails."+tail.ServiceID, "diagnostics", "Log tail: "+tail.ServiceID, status, detail)
			}
		}
	} else {
		addCheck("diagnostics.log_tails", "diagnostics", "Log tails captured", "skipped", "start_services=false")
	}

	return checks
}

func updatePreflightResult(prev *BundlePreflightResponse, update func(next *BundlePreflightResponse)) *BundlePreflightResponse {
	var next BundlePreflightResponse
	if prev != nil {
		next = *prev
	} else {
		next = BundlePreflightResponse{Status: "running"}
	}
	update(&next)
	return &next
}

func validationStepState(validation *runtimeapi.BundleValidationResult) string {
	if validation == nil {
		return "warning"
	}
	if validation.Valid {
		return "pass"
	}
	return "fail"
}

func secretsStepState(secrets []BundlePreflightSecret) string {
	for _, secret := range secrets {
		if secret.Required && !secret.HasValue {
			return "warning"
		}
	}
	return "pass"
}

func readinessStepState(ready *BundlePreflightReady, request BundlePreflightRequest) string {
	if !request.StartServices {
		return "skipped"
	}
	if ready == nil {
		return "warning"
	}
	if ready.Ready {
		return "pass"
	}
	return "warning"
}

func diagnosticsStepState(ports map[string]map[string]int, telemetry *BundlePreflightTelemetry, logTails []BundlePreflightLogTail, request BundlePreflightRequest) string {
	if !request.StartServices {
		return "skipped"
	}
	if len(ports) > 0 || (telemetry != nil && telemetry.Path != "") || len(logTails) > 0 {
		return "pass"
	}
	return "warning"
}

func fetchReadyWithPolling(client *http.Client, baseURL, token string, request BundlePreflightRequest, timeout time.Duration, manifest *bundlemanifest.Manifest) (BundlePreflightReady, int, error) {
	var ready BundlePreflightReady
	if _, err := fetchJSON(client, baseURL, token, "/readyz", http.MethodGet, nil, &ready, nil); err != nil {
		return ready, 0, err
	}
	if !request.StartServices || request.StatusOnly {
		return ready, 0, nil
	}
	waitBudget := maxReadinessTimeout(manifest)
	if waitBudget <= 0 {
		waitBudget = 30 * time.Second
	}
	if timeout > 0 && waitBudget > timeout {
		waitBudget = timeout
	}
	if waitBudget < 2*time.Second {
		waitBudget = 2 * time.Second
	}
	start := time.Now()
	deadline := start.Add(waitBudget)
	for time.Now().Before(deadline) {
		if ready.Ready {
			break
		}
		time.Sleep(1 * time.Second)
		if _, err := fetchJSON(client, baseURL, token, "/readyz", http.MethodGet, nil, &ready, nil); err != nil {
			return ready, int(time.Since(start).Seconds()), err
		}
	}
	return ready, int(time.Since(start).Seconds()), nil
}

func fetchRuntimeStatus(client *http.Client, baseURL, token string) (*BundlePreflightRuntime, error) {
	var status BundlePreflightRuntime
	if _, err := fetchJSON(client, baseURL, token, "/status", http.MethodGet, nil, &status, nil); err != nil {
		return nil, err
	}
	return &status, nil
}

func collectServiceFingerprints(manifest *bundlemanifest.Manifest, bundleRoot string) []BundlePreflightServiceFingerprint {
	if manifest == nil {
		return nil
	}
	platform := bundlemanifest.PlatformKey(runtime.GOOS, runtime.GOARCH)
	results := make([]BundlePreflightServiceFingerprint, 0, len(manifest.Services))
	for _, svc := range manifest.Services {
		fp := BundlePreflightServiceFingerprint{
			ServiceID: svc.ID,
			Platform:  platform,
		}
		bin, ok := manifest.ResolveBinary(svc)
		if !ok || strings.TrimSpace(bin.Path) == "" {
			fp.Error = "binary not resolved for platform"
			results = append(results, fp)
			continue
		}
		fp.BinaryPath = bin.Path
		resolved := bundlemanifest.ResolvePath(bundleRoot, bin.Path)
		fp.BinaryResolvedPath = resolved
		info, err := os.Stat(resolved)
		if err != nil {
			fp.Error = fmt.Sprintf("stat binary: %v", err)
			results = append(results, fp)
			continue
		}
		if info.IsDir() {
			fp.Error = "binary path is a directory"
			results = append(results, fp)
			continue
		}
		fp.BinarySizeBytes = info.Size()
		fp.BinaryMtime = info.ModTime().Format(time.RFC3339)
		hash, err := sha256File(resolved)
		if err != nil {
			fp.Error = fmt.Sprintf("hash binary: %v", err)
			results = append(results, fp)
			continue
		}
		fp.BinarySHA256 = hash
		results = append(results, fp)
	}
	return results
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func maxReadinessTimeout(manifest *bundlemanifest.Manifest) time.Duration {
	if manifest == nil {
		return 0
	}
	maxTimeout := time.Duration(0)
	for _, svc := range manifest.Services {
		timeout := time.Duration(svc.Readiness.TimeoutMs) * time.Millisecond
		if timeout <= 0 {
			timeout = 30 * time.Second
		}
		if timeout > maxTimeout {
			maxTimeout = timeout
		}
	}
	return maxTimeout
}

func readFileWithRetry(path string, timeout time.Duration) ([]byte, error) {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(path)
		if err == nil {
			return data, nil
		}
		lastErr = err
		time.Sleep(50 * time.Millisecond)
	}
	return nil, lastErr
}

func readPortFileWithRetry(path string, timeout time.Duration) (int, error) {
	data, err := readFileWithRetry(path, timeout)
	if err != nil {
		return 0, err
	}
	port, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, err
	}
	return port, nil
}

func waitForRuntimeHealth(client *http.Client, baseURL string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		req, _ := http.NewRequest(http.MethodGet, baseURL+"/healthz", nil)
		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return nil
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("runtime control API not responding within %s", timeout)
}

func fetchJSON(client *http.Client, baseURL, token, path, method string, payload interface{}, out interface{}, allow map[int]bool) (int, error) {
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return 0, err
		}
		body = bytes.NewReader(data)
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

func fetchText(client *http.Client, baseURL, token, path string) (string, int, error) {
	req, err := http.NewRequest(http.MethodGet, baseURL+path, nil)
	if err != nil {
		return "", 0, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", resp.StatusCode, err
	}
	bodyText := strings.TrimSpace(string(bodyBytes))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if bodyText == "" {
			bodyText = resp.Status
		}
		return bodyText, resp.StatusCode, fmt.Errorf("status %d: %s", resp.StatusCode, bodyText)
	}

	return bodyText, resp.StatusCode, nil
}

func collectLogTails(client *http.Client, baseURL, token string, manifest *bundlemanifest.Manifest, request BundlePreflightRequest) []BundlePreflightLogTail {
	if request.LogTailLines <= 0 {
		return nil
	}

	lines := request.LogTailLines
	if lines > 200 {
		lines = 200
	}

	serviceIDs := request.LogTailServices
	if len(serviceIDs) == 0 {
		for _, svc := range manifest.Services {
			if strings.TrimSpace(svc.LogDir) != "" {
				serviceIDs = append(serviceIDs, svc.ID)
			}
		}
	}

	if len(serviceIDs) == 0 {
		return nil
	}

	seen := map[string]bool{}
	var tails []BundlePreflightLogTail
	for _, serviceID := range serviceIDs {
		id := strings.TrimSpace(serviceID)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true

		path := fmt.Sprintf("/logs/tail?serviceId=%s&lines=%d", url.QueryEscape(id), lines)
		content, _, err := fetchText(client, baseURL, token, path)
		tail := BundlePreflightLogTail{
			ServiceID: id,
			Lines:     lines,
		}
		if err != nil {
			tail.Error = err.Error()
		} else {
			tail.Content = content
		}
		if tail.Content == "" && tail.Error == "" {
			continue
		}
		tails = append(tails, tail)
	}

	return tails
}
