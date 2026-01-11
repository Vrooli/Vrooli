package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/internal/httputil"
	"scenario-to-cloud/ssh"
	"scenario-to-cloud/tlsinfo"
	"scenario-to-cloud/vps"
)

// Re-export types from vps package for backward compatibility with tests.
type (
	KillProcessRequest     = vps.KillProcessRequest
	RestartRequest         = vps.RestartRequest
	ProcessControlRequest  = vps.ProcessControlRequest
	ProcessControlResponse = vps.ProcessControlResponse
)

// handleGetLiveState fetches comprehensive live state from the VPS.
// GET /api/v1/deployments/{id}/live-state
func (s *Server) handleGetLiveState(w http.ResponseWriter, r *http.Request) {
	dc := s.FetchDeploymentContext(w, r)
	if dc == nil {
		return // Error already written
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	result := vps.RunLiveStateInspection(ctx, dc.Manifest, s.sshRunner)
	s.enrichCaddyTLS(ctx, &result)

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"result":    result,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// handleGetFiles lists directory contents on VPS.
// GET /api/v1/deployments/{id}/files?path=...
func (s *Server) handleGetFiles(w http.ResponseWriter, r *http.Request) {
	dc := s.FetchDeploymentContext(w, r)
	if dc == nil {
		return // Error already written
	}

	requestedPath := r.URL.Query().Get("path")
	if requestedPath == "" {
		requestedPath = dc.Workdir
	}

	// Security: Ensure path is within workdir to prevent directory traversal
	if !vps.IsPathWithinWorkdir(requestedPath, dc.Workdir) {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Code:    "path_not_allowed",
			Message: "Path must be within the deployment workdir",
			Hint:    "Requested path: " + requestedPath,
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// Execute ls -la
	cmd := "ls -la " + ssh.QuoteSingle(requestedPath)
	result, err := s.sshRunner.Run(ctx, dc.SSHConfig, cmd)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{
			Code:    "ssh_failed",
			Message: "Failed to list directory",
			Hint:    result.Stderr,
		})
		return
	}

	// Parse ls output
	entries := vps.ParseLsOutput(result.Stdout)

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"ok":        true,
		"path":      requestedPath,
		"entries":   entries,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// handleGetFileContent reads file contents from VPS.
// GET /api/v1/deployments/{id}/files/content?path=...
func (s *Server) handleGetFileContent(w http.ResponseWriter, r *http.Request) {
	requestedPath := r.URL.Query().Get("path")
	if requestedPath == "" {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Code:    "missing_path",
			Message: "Path query parameter is required",
		})
		return
	}

	dc := s.FetchDeploymentContext(w, r)
	if dc == nil {
		return // Error already written
	}

	// Handle relative paths by prepending workdir
	if !strings.HasPrefix(requestedPath, "/") {
		requestedPath = dc.Workdir + "/" + requestedPath
	}

	// Security: Ensure path is within workdir (with exception for common config files)
	allowedPaths := []string{"/etc/caddy/Caddyfile"}
	pathAllowed := vps.IsPathWithinWorkdir(requestedPath, dc.Workdir)
	for _, allowed := range allowedPaths {
		if requestedPath == allowed {
			pathAllowed = true
			break
		}
	}

	if !pathAllowed {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Code:    "path_not_allowed",
			Message: "Path must be within the deployment workdir",
			Hint:    "Requested path: " + requestedPath + ", Workdir: " + dc.Workdir,
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// Get file size first
	sizeCmd := "stat -c %s " + ssh.QuoteSingle(requestedPath) + " 2>/dev/null || echo -1"
	sizeResult, _ := s.sshRunner.Run(ctx, dc.SSHConfig, sizeCmd)
	fileSize := -1
	if sizeResult.Stdout != "" {
		fileSize, _ = parseInt(sizeResult.Stdout)
	}

	// Limit file size (1MB max)
	const maxFileSize = 1024 * 1024
	truncated := false
	readCmd := "cat " + ssh.QuoteSingle(requestedPath)
	if fileSize > maxFileSize {
		readCmd = "head -c " + intToStr(maxFileSize) + " " + ssh.QuoteSingle(requestedPath)
		truncated = true
	}

	result, err := s.sshRunner.Run(ctx, dc.SSHConfig, readCmd)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{
			Code:    "ssh_failed",
			Message: "Failed to read file",
			Hint:    result.Stderr,
		})
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"ok":         true,
		"path":       requestedPath,
		"size_bytes": fileSize,
		"content":    result.Stdout,
		"truncated":  truncated,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
	})
}

// handleGetDrift compares manifest expectations vs actual state.
// GET /api/v1/deployments/{id}/drift
func (s *Server) handleGetDrift(w http.ResponseWriter, r *http.Request) {
	dc := s.FetchDeploymentContext(w, r)
	if dc == nil {
		return // Error already written
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	liveState := vps.RunLiveStateInspection(ctx, dc.Manifest, s.sshRunner)
	s.enrichCaddyTLS(ctx, &liveState)
	if !liveState.OK {
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{
			Code:    "live_state_failed",
			Message: "Failed to fetch live state for drift detection",
			Hint:    liveState.Error,
		})
		return
	}

	driftReport := vps.ComputeDrift(dc.Manifest, liveState)

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"result":    driftReport,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) enrichCaddyTLS(ctx context.Context, result *domain.LiveStateResult) {
	if result == nil || result.Caddy == nil {
		return
	}
	domainName := strings.TrimSpace(result.Caddy.Domain)
	if domainName == "" {
		return
	}
	probeCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	snapshot, err := tlsinfo.RunSnapshot(probeCtx, domainName, s.tlsService, s.tlsALPNRunner)
	if err != nil {
		result.Caddy.TLS = domain.TLSInfo{
			Valid: false,
			Error: fmt.Sprintf("TLS probe failed: %v", err),
		}
		return
	}

	result.Caddy.TLS = domain.TLSInfo{
		Valid:         snapshot.Probe.Valid,
		Validation:    "time_only",
		Issuer:        snapshot.Probe.Issuer,
		Expires:       snapshot.Probe.NotAfter,
		DaysRemaining: snapshot.Probe.DaysRemaining,
		ALPN:          convertALPN(snapshot.ALPN),
	}
}

func convertALPN(check tlsinfo.ALPNCheck) *domain.ALPNCheck {
	return &domain.ALPNCheck{
		Status:   string(check.Status),
		Message:  check.Message,
		Hint:     check.Hint,
		Protocol: check.Protocol,
		Error:    check.Error,
	}
}

// handleKillProcess kills a specific process on VPS.
// POST /api/v1/deployments/{id}/actions/kill
func (s *Server) handleKillProcess(w http.ResponseWriter, r *http.Request) {
	req, err := httputil.DecodeJSON[KillProcessRequest](r.Body, 1<<20)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Code:    "invalid_json",
			Message: "Request body must be valid JSON",
			Hint:    err.Error(),
		})
		return
	}

	if req.PID <= 0 {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Code:    "invalid_pid",
			Message: "PID must be a positive integer",
		})
		return
	}

	dc := s.FetchDeploymentContext(w, r)
	if dc == nil {
		return // Error already written
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	signal := req.Signal
	if signal == "" {
		signal = "TERM"
	}

	cmd := "kill -" + signal + " " + intToStr(req.PID)
	result, err := s.sshRunner.Run(ctx, dc.SSHConfig, cmd)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{
			Code:    "kill_failed",
			Message: "Failed to kill process",
			Hint:    result.Stderr,
		})
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"ok":        true,
		"pid":       req.PID,
		"signal":    signal,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// handleRestartProcess restarts a scenario or resource on VPS.
// POST /api/v1/deployments/{id}/actions/restart
func (s *Server) handleRestartProcess(w http.ResponseWriter, r *http.Request) {
	req, err := httputil.DecodeJSON[RestartRequest](r.Body, 1<<20)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Code:    "invalid_json",
			Message: "Request body must be valid JSON",
			Hint:    err.Error(),
		})
		return
	}

	if req.Type != "scenario" && req.Type != "resource" {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Code:    "invalid_type",
			Message: "Type must be 'scenario' or 'resource'",
		})
		return
	}

	if req.ID == "" {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Code:    "missing_id",
			Message: "ID is required",
		})
		return
	}

	dc := s.FetchDeploymentContext(w, r)
	if dc == nil {
		return // Error already written
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	cmd := ssh.VrooliCommand(dc.Workdir, "vrooli "+req.Type+" restart "+ssh.QuoteSingle(req.ID))

	result, err := s.sshRunner.Run(ctx, dc.SSHConfig, cmd)
	if err != nil {
		s.appendHistoryEvent(ctx, dc.ID, domain.HistoryEvent{
			Type:      domain.EventRestarted,
			Timestamp: time.Now().UTC(),
			Message:   fmt.Sprintf("Restart failed: %s %s", req.Type, req.ID),
			Details:   result.Stderr,
			Success:   boolPtr(false),
		})
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{
			Code:    "restart_failed",
			Message: "Failed to restart " + req.Type,
			Hint:    result.Stderr,
		})
		return
	}

	s.appendHistoryEvent(ctx, dc.ID, domain.HistoryEvent{
		Type:      domain.EventRestarted,
		Timestamp: time.Now().UTC(),
		Message:   fmt.Sprintf("Restarted %s %s", req.Type, req.ID),
		Details:   strings.TrimSpace(result.Stdout),
		Success:   boolPtr(true),
	})

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"ok":        true,
		"type":      req.Type,
		"id":        req.ID,
		"output":    result.Stdout,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// handleProcessControl handles start/stop/restart/setup for scenarios and resources.
// POST /api/v1/deployments/{id}/actions/process
func (s *Server) handleProcessControl(w http.ResponseWriter, r *http.Request) {
	req, err := httputil.DecodeJSON[ProcessControlRequest](r.Body, 1<<20)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Code:    "invalid_json",
			Message: "Request body must be valid JSON",
			Hint:    err.Error(),
		})
		return
	}

	// Validate request using early returns for clarity
	if apiErr := validateProcessControlRequest(req); apiErr != nil {
		httputil.WriteAPIError(w, http.StatusBadRequest, *apiErr)
		return
	}

	dc := s.FetchDeploymentContext(w, r)
	if dc == nil {
		return // Error already written
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	cmd := ssh.VrooliCommand(dc.Workdir, "vrooli "+req.Type+" "+req.Action+" "+ssh.QuoteSingle(req.ID))
	result, err := s.sshRunner.Run(ctx, dc.SSHConfig, cmd)

	response := ProcessControlResponse{
		Action:    req.Action,
		Type:      req.Type,
		ID:        req.ID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	if err != nil {
		response.OK = false
		response.Message = "Failed to " + req.Action + " " + req.Type
		response.Output = result.Stderr
		if req.Action == "restart" || req.Action == "stop" {
			eventType := domain.EventRestarted
			if req.Action == "stop" {
				eventType = domain.EventStopped
			}
			label := actionLabel(req.Action)
			s.appendHistoryEvent(ctx, dc.ID, domain.HistoryEvent{
				Type:      eventType,
				Timestamp: time.Now().UTC(),
				Message:   fmt.Sprintf("%s failed: %s %s", label, req.Type, req.ID),
				Details:   result.Stderr,
				Success:   boolPtr(false),
				StepName:  req.Action,
			})
		}
		httputil.WriteJSON(w, http.StatusInternalServerError, response)
		return
	}

	response.OK = true
	response.Message = "Successfully " + req.Action + "ed " + req.Type + " " + req.ID
	response.Output = result.Stdout
	if req.Action == "restart" || req.Action == "stop" {
		eventType := domain.EventRestarted
		if req.Action == "stop" {
			eventType = domain.EventStopped
		}
		label := actionLabel(req.Action)
		s.appendHistoryEvent(ctx, dc.ID, domain.HistoryEvent{
			Type:      eventType,
			Timestamp: time.Now().UTC(),
			Message:   fmt.Sprintf("%s %s %s", label, req.Type, req.ID),
			Details:   strings.TrimSpace(result.Stdout),
			Success:   boolPtr(true),
			StepName:  req.Action,
		})
	}
	httputil.WriteJSON(w, http.StatusOK, response)
}

func actionLabel(action string) string {
	switch action {
	case "restart":
		return "Restart"
	case "stop":
		return "Stop"
	default:
		return action
	}
}

// validateProcessControlRequest validates the ProcessControlRequest fields.
// Returns nil if valid, or an httputil.APIError pointer if validation fails.
func validateProcessControlRequest(req ProcessControlRequest) *httputil.APIError {
	validActions := map[string]bool{"start": true, "stop": true, "restart": true, "setup": true}
	if !validActions[req.Action] {
		return &httputil.APIError{
			Code:    "invalid_action",
			Message: "Action must be 'start', 'stop', 'restart', or 'setup'",
		}
	}

	if req.Type != "scenario" && req.Type != "resource" {
		return &httputil.APIError{
			Code:    "invalid_type",
			Message: "Type must be 'scenario' or 'resource'",
		}
	}

	if req.Action == "setup" && req.Type != "resource" {
		return &httputil.APIError{
			Code:    "invalid_action",
			Message: "Setup action is only valid for resources",
		}
	}

	if req.ID == "" {
		return &httputil.APIError{
			Code:    "missing_id",
			Message: "ID is required",
		}
	}

	return nil
}

// Helper functions used by handlers

// parseInt parses a string to int.
func parseInt(s string) (int, error) {
	s = strings.TrimSpace(s)
	return strconv.Atoi(s)
}

// intToStr converts int to string.
func intToStr(i int) string {
	return strconv.Itoa(i)
}
