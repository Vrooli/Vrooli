package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"scenario-to-cloud/domain"
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

	result := RunLiveStateInspection(ctx, dc.Manifest, s.sshRunner)

	writeJSON(w, http.StatusOK, map[string]interface{}{
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
	if !isPathWithinWorkdir(requestedPath, dc.Workdir) {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "path_not_allowed",
			Message: "Path must be within the deployment workdir",
			Hint:    "Requested path: " + requestedPath,
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// Execute ls -la
	cmd := "ls -la " + shellQuoteSingle(requestedPath)
	result, err := s.sshRunner.Run(ctx, dc.SSHConfig, cmd)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "ssh_failed",
			Message: "Failed to list directory",
			Hint:    result.Stderr,
		})
		return
	}

	// Parse ls output
	entries := parseLsOutput(result.Stdout)

	writeJSON(w, http.StatusOK, map[string]interface{}{
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
		writeAPIError(w, http.StatusBadRequest, APIError{
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
	pathAllowed := isPathWithinWorkdir(requestedPath, dc.Workdir)
	for _, allowed := range allowedPaths {
		if requestedPath == allowed {
			pathAllowed = true
			break
		}
	}

	if !pathAllowed {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "path_not_allowed",
			Message: "Path must be within the deployment workdir",
			Hint:    "Requested path: " + requestedPath + ", Workdir: " + dc.Workdir,
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// Get file size first
	sizeCmd := "stat -c %s " + shellQuoteSingle(requestedPath) + " 2>/dev/null || echo -1"
	sizeResult, _ := s.sshRunner.Run(ctx, dc.SSHConfig, sizeCmd)
	fileSize := -1
	if sizeResult.Stdout != "" {
		fileSize, _ = parseInt(sizeResult.Stdout)
	}

	// Limit file size (1MB max)
	const maxFileSize = 1024 * 1024
	truncated := false
	readCmd := "cat " + shellQuoteSingle(requestedPath)
	if fileSize > maxFileSize {
		readCmd = "head -c " + intToStr(maxFileSize) + " " + shellQuoteSingle(requestedPath)
		truncated = true
	}

	result, err := s.sshRunner.Run(ctx, dc.SSHConfig, readCmd)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "ssh_failed",
			Message: "Failed to read file",
			Hint:    result.Stderr,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
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

	liveState := RunLiveStateInspection(ctx, dc.Manifest, s.sshRunner)
	if !liveState.OK {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "live_state_failed",
			Message: "Failed to fetch live state for drift detection",
			Hint:    liveState.Error,
		})
		return
	}

	driftReport := computeDrift(dc.Manifest, liveState)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"result":    driftReport,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// handleKillProcess kills a specific process on VPS.
// POST /api/v1/deployments/{id}/actions/kill
func (s *Server) handleKillProcess(w http.ResponseWriter, r *http.Request) {
	req, err := decodeJSON[KillProcessRequest](r.Body, 1<<20)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "invalid_json",
			Message: "Request body must be valid JSON",
			Hint:    err.Error(),
		})
		return
	}

	if req.PID <= 0 {
		writeAPIError(w, http.StatusBadRequest, APIError{
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
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "kill_failed",
			Message: "Failed to kill process",
			Hint:    result.Stderr,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok":        true,
		"pid":       req.PID,
		"signal":    signal,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// handleRestartProcess restarts a scenario or resource on VPS.
// POST /api/v1/deployments/{id}/actions/restart
func (s *Server) handleRestartProcess(w http.ResponseWriter, r *http.Request) {
	req, err := decodeJSON[RestartRequest](r.Body, 1<<20)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "invalid_json",
			Message: "Request body must be valid JSON",
			Hint:    err.Error(),
		})
		return
	}

	if req.Type != "scenario" && req.Type != "resource" {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "invalid_type",
			Message: "Type must be 'scenario' or 'resource'",
		})
		return
	}

	if req.ID == "" {
		writeAPIError(w, http.StatusBadRequest, APIError{
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

	cmd := vrooliCommand(dc.Workdir, "vrooli "+req.Type+" restart "+shellQuoteSingle(req.ID))

	result, err := s.sshRunner.Run(ctx, dc.SSHConfig, cmd)
	if err != nil {
		s.appendHistoryEvent(ctx, dc.ID, domain.HistoryEvent{
			Type:      domain.EventRestarted,
			Timestamp: time.Now().UTC(),
			Message:   fmt.Sprintf("Restart failed: %s %s", req.Type, req.ID),
			Details:   result.Stderr,
			Success:   boolPtr(false),
		})
		writeAPIError(w, http.StatusInternalServerError, APIError{
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

	writeJSON(w, http.StatusOK, map[string]interface{}{
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
	req, err := decodeJSON[ProcessControlRequest](r.Body, 1<<20)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "invalid_json",
			Message: "Request body must be valid JSON",
			Hint:    err.Error(),
		})
		return
	}

	// Validate request using early returns for clarity
	if apiErr := validateProcessControlRequest(req); apiErr != nil {
		writeAPIError(w, http.StatusBadRequest, *apiErr)
		return
	}

	dc := s.FetchDeploymentContext(w, r)
	if dc == nil {
		return // Error already written
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	cmd := vrooliCommand(dc.Workdir, "vrooli "+req.Type+" "+req.Action+" "+shellQuoteSingle(req.ID))
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
		writeJSON(w, http.StatusInternalServerError, response)
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
	writeJSON(w, http.StatusOK, response)
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
// Returns nil if valid, or an APIError pointer if validation fails.
func validateProcessControlRequest(req ProcessControlRequest) *APIError {
	validActions := map[string]bool{"start": true, "stop": true, "restart": true, "setup": true}
	if !validActions[req.Action] {
		return &APIError{
			Code:    "invalid_action",
			Message: "Action must be 'start', 'stop', 'restart', or 'setup'",
		}
	}

	if req.Type != "scenario" && req.Type != "resource" {
		return &APIError{
			Code:    "invalid_type",
			Message: "Type must be 'scenario' or 'resource'",
		}
	}

	if req.Action == "setup" && req.Type != "resource" {
		return &APIError{
			Code:    "invalid_action",
			Message: "Setup action is only valid for resources",
		}
	}

	if req.ID == "" {
		return &APIError{
			Code:    "missing_id",
			Message: "ID is required",
		}
	}

	return nil
}

// KillProcessRequest is the request body for killing a process.
type KillProcessRequest struct {
	PID    int    `json:"pid"`
	Signal string `json:"signal,omitempty"` // TERM, KILL, etc.
}

// RestartRequest is the request body for restarting a scenario or resource.
type RestartRequest struct {
	Type string `json:"type"` // "scenario" or "resource"
	ID   string `json:"id"`
}

// ProcessControlRequest is the request body for controlling a scenario or resource.
type ProcessControlRequest struct {
	Action string `json:"action"` // "start", "stop", "restart", "setup"
	Type   string `json:"type"`   // "scenario" or "resource"
	ID     string `json:"id"`
}

// ProcessControlResponse is the response for process control actions.
type ProcessControlResponse struct {
	OK        bool   `json:"ok"`
	Action    string `json:"action"`
	Type      string `json:"type"`
	ID        string `json:"id"`
	Message   string `json:"message"`
	Output    string `json:"output,omitempty"`
	Timestamp string `json:"timestamp"`
}

// DriftReport contains the drift detection results.
type DriftReport struct {
	OK        bool         `json:"ok"`
	Timestamp string       `json:"timestamp"`
	Summary   DriftSummary `json:"summary"`
	Checks    []DriftCheck `json:"checks"`
}

// DriftSummary contains aggregate drift statistics.
type DriftSummary struct {
	Passed   int `json:"passed"`
	Warnings int `json:"warnings"`
	Drifts   int `json:"drifts"`
}

// DriftCheck represents a single drift check result.
type DriftCheck struct {
	Category string   `json:"category"` // scenarios, resources, ports, edge
	Name     string   `json:"name"`
	Status   string   `json:"status"` // pass, warning, drift
	Expected string   `json:"expected"`
	Actual   string   `json:"actual"`
	Message  string   `json:"message,omitempty"`
	Actions  []string `json:"actions,omitempty"`
}

// FileEntry represents a file or directory in the file explorer.
type FileEntry struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // file, directory, symlink
	SizeBytes   int64  `json:"size_bytes"`
	Modified    string `json:"modified"`
	Permissions string `json:"permissions"`
}

// computeDrift compares manifest expectations vs actual live state.
func computeDrift(manifest CloudManifest, liveState LiveStateResult) DriftReport {
	report := DriftReport{
		OK:        true,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Checks:    []DriftCheck{},
	}

	// Check main scenario
	scenarioRunning := isScenarioRunning(liveState.Processes, manifest.Scenario.ID)
	report.addCheck(scenarioRunning, DriftCheck{
		Category: "scenarios",
		Name:     manifest.Scenario.ID,
		Expected: "running",
	}, DriftCheck{
		Message: "Main scenario is not running",
		Actions: []string{"restart_scenario"},
	})

	// Check expected resources
	for _, resID := range manifest.Dependencies.Resources {
		resourceRunning := isResourceRunning(liveState.Processes, resID)
		report.addCheck(resourceRunning, DriftCheck{
			Category: "resources",
			Name:     resID,
			Expected: "running",
		}, DriftCheck{
			Message: "Expected resource is not running",
			Actions: []string{"restart_resource"},
		})
	}

	// Check for unexpected resources
	report.checkUnexpectedResources(liveState.Processes, manifest.Dependencies.Resources)

	// Check for unexpected processes with ports
	report.checkUnexpectedPorts(liveState.Processes)

	// Check Caddy/TLS
	report.checkCaddyState(liveState.Caddy)

	return report
}

// addCheck adds a pass or drift check to the report based on the condition.
// passFields provides the base check info; driftFields provides additional info for failures.
func (r *DriftReport) addCheck(passed bool, passFields, driftFields DriftCheck) {
	check := passFields
	if passed {
		check.Status = "pass"
		check.Actual = passFields.Expected
		r.Summary.Passed++
	} else {
		check.Status = "drift"
		check.Actual = "not running"
		check.Message = driftFields.Message
		check.Actions = driftFields.Actions
		r.Summary.Drifts++
	}
	r.Checks = append(r.Checks, check)
}

// isScenarioRunning checks if a scenario is running in the process list.
func isScenarioRunning(processes *ProcessState, scenarioID string) bool {
	if processes == nil {
		return false
	}
	for _, s := range processes.Scenarios {
		if s.ID == scenarioID {
			return s.Status == "running"
		}
	}
	return false
}

// isResourceRunning checks if a resource is running in the process list.
func isResourceRunning(processes *ProcessState, resourceID string) bool {
	if processes == nil {
		return false
	}
	for _, r := range processes.Resources {
		if r.ID == resourceID {
			return r.Status == "running"
		}
	}
	return false
}

// checkUnexpectedResources adds warnings for resources running but not in manifest.
func (r *DriftReport) checkUnexpectedResources(processes *ProcessState, expectedIDs []string) {
	if processes == nil {
		return
	}
	expectedSet := make(map[string]bool)
	for _, res := range expectedIDs {
		expectedSet[res] = true
	}
	for _, res := range processes.Resources {
		if !expectedSet[res.ID] {
			r.Checks = append(r.Checks, DriftCheck{
				Category: "resources",
				Name:     res.ID,
				Status:   "warning",
				Expected: "not specified in manifest",
				Actual:   "running",
				Message:  "Resource is running but not in manifest",
				Actions:  []string{"stop", "add_to_manifest"},
			})
			r.Summary.Warnings++
		}
	}
}

// checkUnexpectedPorts adds warnings for unexpected processes listening on ports.
func (r *DriftReport) checkUnexpectedPorts(processes *ProcessState) {
	if processes == nil {
		return
	}
	for _, u := range processes.Unexpected {
		if u.Port > 0 {
			r.Checks = append(r.Checks, DriftCheck{
				Category: "ports",
				Name:     intToStr(u.Port),
				Status:   "warning",
				Expected: "no unexpected process",
				Actual:   u.Command,
				Message:  "Unexpected process listening on port",
				Actions:  []string{"kill_pid"},
			})
			r.Summary.Warnings++
		}
	}
}

// checkCaddyState adds checks for Caddy and TLS status.
func (r *DriftReport) checkCaddyState(caddy *CaddyState) {
	if caddy == nil {
		return
	}
	if caddy.Running {
		r.Checks = append(r.Checks, DriftCheck{
			Category: "edge",
			Name:     "caddy",
			Status:   "pass",
			Expected: "running",
			Actual:   "running",
		})
		r.Summary.Passed++

		if caddy.TLS.Valid {
			r.Checks = append(r.Checks, DriftCheck{
				Category: "edge",
				Name:     "tls",
				Status:   "pass",
				Expected: "valid certificate",
				Actual:   "valid certificate",
			})
			r.Summary.Passed++
		}
	} else {
		r.Checks = append(r.Checks, DriftCheck{
			Category: "edge",
			Name:     "caddy",
			Status:   "drift",
			Expected: "running",
			Actual:   "not running",
			Message:  "Caddy reverse proxy is not running",
			Actions:  []string{"restart_caddy"},
		})
		r.Summary.Drifts++
	}
}

// Helper functions

// isPathWithinWorkdir checks if a path is within the workdir to prevent directory traversal.
func isPathWithinWorkdir(path, workdir string) bool {
	// Normalize paths
	path = normalizePath(path)
	workdir = normalizePath(workdir)

	// Check if path starts with workdir
	return len(path) >= len(workdir) && path[:len(workdir)] == workdir
}

// normalizePath normalizes a path by removing trailing slashes and handling ..
func normalizePath(p string) string {
	// Remove trailing slash
	for len(p) > 1 && p[len(p)-1] == '/' {
		p = p[:len(p)-1]
	}
	return p
}

// parseLsOutput parses the output of `ls -la` into FileEntry structs.
func parseLsOutput(output string) []FileEntry {
	var entries []FileEntry
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip total line
		if strings.HasPrefix(line, "total ") {
			continue
		}

		// Skip . and ..
		if strings.HasSuffix(line, " .") || strings.HasSuffix(line, " ..") {
			continue
		}

		// Parse ls -la output
		// Format: -rw-r--r-- 1 user group size month day time/year name
		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}

		permissions := fields[0]
		sizeStr := fields[4]
		month := fields[5]
		day := fields[6]
		timeOrYear := fields[7]
		name := strings.Join(fields[8:], " ")

		// Determine type from permissions
		fileType := "file"
		if permissions[0] == 'd' {
			fileType = "directory"
		} else if permissions[0] == 'l' {
			fileType = "symlink"
		}

		size, _ := parseInt64(sizeStr)

		entries = append(entries, FileEntry{
			Name:        name,
			Type:        fileType,
			SizeBytes:   size,
			Modified:    month + " " + day + " " + timeOrYear,
			Permissions: permissions,
		})
	}

	return entries
}

// parseInt parses a string to int.
func parseInt(s string) (int, error) {
	s = strings.TrimSpace(s)
	return strconv.Atoi(s)
}

// parseInt64 parses a string to int64.
func parseInt64(s string) (int64, error) {
	s = strings.TrimSpace(s)
	return strconv.ParseInt(s, 10, 64)
}

// intToStr converts int to string.
func intToStr(i int) string {
	return strconv.Itoa(i)
}

// strings is imported at the top, using strings.Split etc.
