package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/internal/httputil"
	"scenario-to-cloud/reach"
	"scenario-to-cloud/sshidentity"
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

// Every handler in this file reaches the target through the deployment's
// bound transport (reach): live state and files are observation programs
// and typed vrooli verbs, process control is the lifecycle owner's verbs.
// There is no shell string and no raw SSH here.

// handleGetLiveState fetches comprehensive live state from the target.
// GET /api/v1/deployments/{id}/live-state
func (s *Server) handleGetLiveState(w http.ResponseWriter, r *http.Request) {
	dc := s.FetchDeploymentContext(w, r)
	if dc == nil {
		return // Error already written
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()
	identity := s.resolveCanonicalIdentity(r.Context(), dc.Deployment)

	result := vps.RunLiveStateInspection(ctx, dc.Manifest, identity, s.proberFor(dc))
	if result.OK && result.System != nil {
		verified := sshidentity.ApplyVerificationResult(
			identity,
			sshidentity.VerificationState(result.System.SSH.VerificationState),
			time.Now().UTC(),
		)
		s.persistCanonicalIdentity(ctx, dc.Deployment.ID, verified)
	}
	s.enrichCaddyTLS(ctx, &result)

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"deployment_id": dc.Deployment.ID,
		"result":        result,
		"timestamp":     time.Now().UTC().Format(time.RFC3339),
	})
}

// handleGetMetricsDebug returns raw system metric probe output plus parsed metrics.
// GET /api/v1/deployments/{id}/metrics-debug
func (s *Server) handleGetMetricsDebug(w http.ResponseWriter, r *http.Request) {
	dc := s.FetchDeploymentContext(w, r)
	if dc == nil {
		return // Error already written
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()
	identity := s.resolveCanonicalIdentity(r.Context(), dc.Deployment)

	result := vps.RunSystemMetricsDebug(ctx, identity, s.proberFor(dc))

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"deployment_id": dc.Deployment.ID,
		"result":        result,
		"timestamp":     time.Now().UTC().Format(time.RFC3339),
	})
}

// handleGetFiles lists directory contents on the target.
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

	result, err := s.proberFor(dc).Observe(ctx, "ls", "-la", "--", requestedPath)
	if err != nil {
		apierrors.Write(w, reach.APIError(err))
		return
	}
	if result.ExitCode != 0 {
		httputil.WriteAPIError(w, http.StatusBadGateway, httputil.APIError{
			Code:    "list_failed",
			Message: "Failed to list directory",
			Hint:    strings.TrimSpace(result.Stderr),
		})
		return
	}

	entries := vps.ParseLsOutput(result.Stdout)

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"ok":        true,
		"path":      requestedPath,
		"entries":   entries,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// handleGetFileContent reads file contents from the target.
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
	prober := s.proberFor(dc)

	// Get file size first
	fileSize := -1
	if sizeResult, err := prober.Observe(ctx, "stat", "-c", "%s", "--", requestedPath); err == nil && sizeResult.ExitCode == 0 {
		fileSize, _ = parseInt(sizeResult.Stdout)
	}

	// Limit file size (1MB max)
	const maxFileSize = 1024 * 1024
	truncated := false
	var result reach.Result
	var err error
	if fileSize > maxFileSize {
		result, err = prober.Observe(ctx, "head", "-c", intToStr(maxFileSize), "--", requestedPath)
		truncated = true
	} else {
		result, err = prober.Observe(ctx, "cat", "--", requestedPath)
	}
	if err != nil {
		apierrors.Write(w, reach.APIError(err))
		return
	}
	if result.ExitCode != 0 {
		httputil.WriteAPIError(w, http.StatusBadGateway, httputil.APIError{
			Code:    "read_failed",
			Message: "Failed to read file",
			Hint:    strings.TrimSpace(result.Stderr),
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
	identity := s.resolveCanonicalIdentity(r.Context(), dc.Deployment)

	liveState := vps.RunLiveStateInspection(ctx, dc.Manifest, identity, s.proberFor(dc))
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
	result.Caddy.TLS = buildDomainTLSInfo(snapshot, err)
}

// handleKillProcess is refused: killing a process by pid is not an owner
// operation. A scenario the deployment owns is stopped through the
// lifecycle owner (actions/process with action=stop); a foreign process is
// the host operator's to stop.
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
		return
	}
	apierrors.Write(w, apierrors.Newf(apierrors.CodeUnsupportedCapability, "Killing pid %d by signal has no target owner action; stop the owning scenario through the lifecycle owner or stop a foreign unit on the host", req.PID).
		WithDetail("required_owner", "privilegebroker process.stop.scoped").
		WithNextAction(apierrors.NextAction{Owner: "scenario-to-cloud", Kind: "process", Reference: "/api/v1/deployments/" + dc.ID + "/actions/process", Label: "Stop the scenario that owns the process (type=scenario, action=stop)"}))
}

// handleRestartProcess restarts a scenario or resource through the
// lifecycle owner's verb.
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

	result, err := s.proberFor(dc).Effect(ctx, req.Type+" restart", req.ID)
	if err != nil || result.ExitCode != 0 {
		detail := strings.TrimSpace(result.Stderr)
		if err != nil {
			detail = err.Error()
		}
		s.appendHistoryEvent(ctx, dc.ID, domain.HistoryEvent{
			Type:      domain.EventRestarted,
			Timestamp: time.Now().UTC(),
			Message:   fmt.Sprintf("Restart failed: %s %s", req.Type, req.ID),
			Details:   detail,
			Success:   boolPtr(false),
		})
		if err != nil {
			apierrors.Write(w, reach.APIError(err))
			return
		}
		httputil.WriteAPIError(w, http.StatusBadGateway, httputil.APIError{
			Code:    "restart_failed",
			Message: "Failed to restart " + req.Type,
			Hint:    detail,
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

// handleProcessControl handles start/stop/restart/setup for scenarios and
// resources through the lifecycle owner's verbs.
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

	result, err := s.proberFor(dc).Effect(ctx, req.Type+" "+req.Action, req.ID)

	response := ProcessControlResponse{
		Action:    req.Action,
		Type:      req.Type,
		ID:        req.ID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	if err != nil || result.ExitCode != 0 {
		response.OK = false
		response.Message = "Failed to " + req.Action + " " + req.Type
		response.Output = strings.TrimSpace(result.Stderr)
		if err != nil {
			response.Output = err.Error()
		}
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
				Details:   response.Output,
				Success:   boolPtr(false),
				StepName:  req.Action,
			})
		}
		httputil.WriteJSON(w, http.StatusBadGateway, response)
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
	if err := reach.ValidateArgs([]string{req.ID}); err != nil {
		return &httputil.APIError{
			Code:    "invalid_id",
			Message: "ID must be a plain scenario or resource identifier",
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
