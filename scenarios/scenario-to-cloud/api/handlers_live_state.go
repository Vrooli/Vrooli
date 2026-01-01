package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// handleGetLiveState fetches comprehensive live state from the VPS.
// GET /api/v1/deployments/{id}/live-state
func (s *Server) handleGetLiveState(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	// Get deployment from database
	deployment, err := s.repo.GetDeployment(r.Context(), id)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "get_failed",
			Message: "Failed to get deployment",
			Hint:    err.Error(),
		})
		return
	}

	if deployment == nil {
		writeAPIError(w, http.StatusNotFound, APIError{
			Code:    "not_found",
			Message: "Deployment not found",
		})
		return
	}

	// Parse manifest
	var manifest CloudManifest
	if err := json.Unmarshal(deployment.Manifest, &manifest); err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "manifest_parse_failed",
			Message: "Failed to parse deployment manifest",
			Hint:    err.Error(),
		})
		return
	}

	// Validate and normalize manifest
	normalized, _ := ValidateAndNormalizeManifest(manifest)

	// Check if VPS target exists
	if normalized.Target.VPS == nil {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "no_vps_target",
			Message: "Deployment does not have a VPS target",
		})
		return
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	// Run live state inspection
	result := RunLiveStateInspection(ctx, normalized, ExecSSHRunner{})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"result":    result,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// handleGetFiles lists directory contents on VPS.
// GET /api/v1/deployments/{id}/files?path=...
func (s *Server) handleGetFiles(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	requestedPath := r.URL.Query().Get("path")

	// Get deployment
	deployment, err := s.repo.GetDeployment(r.Context(), id)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "get_failed",
			Message: "Failed to get deployment",
			Hint:    err.Error(),
		})
		return
	}

	if deployment == nil {
		writeAPIError(w, http.StatusNotFound, APIError{
			Code:    "not_found",
			Message: "Deployment not found",
		})
		return
	}

	// Parse manifest
	var manifest CloudManifest
	if err := json.Unmarshal(deployment.Manifest, &manifest); err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "manifest_parse_failed",
			Message: "Failed to parse deployment manifest",
			Hint:    err.Error(),
		})
		return
	}

	normalized, _ := ValidateAndNormalizeManifest(manifest)
	if normalized.Target.VPS == nil {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "no_vps_target",
			Message: "Deployment does not have a VPS target",
		})
		return
	}

	cfg := sshConfigFromManifest(normalized)
	workdir := normalized.Target.VPS.Workdir

	// Default to workdir if no path specified
	if requestedPath == "" {
		requestedPath = workdir
	}

	// Security: Ensure path is within workdir to prevent directory traversal
	if !isPathWithinWorkdir(requestedPath, workdir) {
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
	sshRunner := ExecSSHRunner{}
	result, err := sshRunner.Run(ctx, cfg, cmd)
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
	id := mux.Vars(r)["id"]
	requestedPath := r.URL.Query().Get("path")

	if requestedPath == "" {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "missing_path",
			Message: "Path query parameter is required",
		})
		return
	}

	// Get deployment
	deployment, err := s.repo.GetDeployment(r.Context(), id)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "get_failed",
			Message: "Failed to get deployment",
			Hint:    err.Error(),
		})
		return
	}

	if deployment == nil {
		writeAPIError(w, http.StatusNotFound, APIError{
			Code:    "not_found",
			Message: "Deployment not found",
		})
		return
	}

	// Parse manifest
	var manifest CloudManifest
	if err := json.Unmarshal(deployment.Manifest, &manifest); err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "manifest_parse_failed",
			Message: "Failed to parse deployment manifest",
			Hint:    err.Error(),
		})
		return
	}

	normalized, _ := ValidateAndNormalizeManifest(manifest)
	if normalized.Target.VPS == nil {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "no_vps_target",
			Message: "Deployment does not have a VPS target",
		})
		return
	}

	cfg := sshConfigFromManifest(normalized)
	workdir := normalized.Target.VPS.Workdir

	// Handle relative paths by prepending workdir
	if !strings.HasPrefix(requestedPath, "/") {
		requestedPath = workdir + "/" + requestedPath
	}

	// Security: Ensure path is within workdir (with exception for common config files)
	allowedPaths := []string{"/etc/caddy/Caddyfile"}
	pathAllowed := isPathWithinWorkdir(requestedPath, workdir)
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
			Hint:    "Requested path: " + requestedPath + ", Workdir: " + workdir,
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// Get file size first
	sizeCmd := "stat -c %s " + shellQuoteSingle(requestedPath) + " 2>/dev/null || echo -1"
	sshRunner := ExecSSHRunner{}
	sizeResult, _ := sshRunner.Run(ctx, cfg, sizeCmd)
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

	result, err := sshRunner.Run(ctx, cfg, readCmd)
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
	id := mux.Vars(r)["id"]

	// Get deployment
	deployment, err := s.repo.GetDeployment(r.Context(), id)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "get_failed",
			Message: "Failed to get deployment",
			Hint:    err.Error(),
		})
		return
	}

	if deployment == nil {
		writeAPIError(w, http.StatusNotFound, APIError{
			Code:    "not_found",
			Message: "Deployment not found",
		})
		return
	}

	// Parse manifest
	var manifest CloudManifest
	if err := json.Unmarshal(deployment.Manifest, &manifest); err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "manifest_parse_failed",
			Message: "Failed to parse deployment manifest",
			Hint:    err.Error(),
		})
		return
	}

	normalized, _ := ValidateAndNormalizeManifest(manifest)
	if normalized.Target.VPS == nil {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "no_vps_target",
			Message: "Deployment does not have a VPS target",
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	// Get live state first
	liveState := RunLiveStateInspection(ctx, normalized, ExecSSHRunner{})
	if !liveState.OK {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "live_state_failed",
			Message: "Failed to fetch live state for drift detection",
			Hint:    liveState.Error,
		})
		return
	}

	// Perform drift detection
	driftReport := computeDrift(normalized, liveState)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"result":    driftReport,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// handleKillProcess kills a specific process on VPS.
// POST /api/v1/deployments/{id}/actions/kill
func (s *Server) handleKillProcess(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

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

	// Get deployment
	deployment, err := s.repo.GetDeployment(r.Context(), id)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "get_failed",
			Message: "Failed to get deployment",
			Hint:    err.Error(),
		})
		return
	}

	if deployment == nil {
		writeAPIError(w, http.StatusNotFound, APIError{
			Code:    "not_found",
			Message: "Deployment not found",
		})
		return
	}

	// Parse manifest
	var manifest CloudManifest
	if err := json.Unmarshal(deployment.Manifest, &manifest); err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "manifest_parse_failed",
			Message: "Failed to parse deployment manifest",
			Hint:    err.Error(),
		})
		return
	}

	normalized, _ := ValidateAndNormalizeManifest(manifest)
	if normalized.Target.VPS == nil {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "no_vps_target",
			Message: "Deployment does not have a VPS target",
		})
		return
	}

	cfg := sshConfigFromManifest(normalized)

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// Determine signal
	signal := req.Signal
	if signal == "" {
		signal = "TERM"
	}

	cmd := "kill -" + signal + " " + intToStr(req.PID)
	sshRunner := ExecSSHRunner{}
	result, err := sshRunner.Run(ctx, cfg, cmd)
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
	id := mux.Vars(r)["id"]

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

	// Get deployment
	deployment, err := s.repo.GetDeployment(r.Context(), id)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "get_failed",
			Message: "Failed to get deployment",
			Hint:    err.Error(),
		})
		return
	}

	if deployment == nil {
		writeAPIError(w, http.StatusNotFound, APIError{
			Code:    "not_found",
			Message: "Deployment not found",
		})
		return
	}

	// Parse manifest
	var manifest CloudManifest
	if err := json.Unmarshal(deployment.Manifest, &manifest); err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "manifest_parse_failed",
			Message: "Failed to parse deployment manifest",
			Hint:    err.Error(),
		})
		return
	}

	normalized, _ := ValidateAndNormalizeManifest(manifest)
	if normalized.Target.VPS == nil {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "no_vps_target",
			Message: "Deployment does not have a VPS target",
		})
		return
	}

	cfg := sshConfigFromManifest(normalized)
	workdir := normalized.Target.VPS.Workdir

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	var cmd string
	if req.Type == "scenario" {
		cmd = vrooliCommand(workdir, "vrooli scenario restart "+shellQuoteSingle(req.ID))
	} else {
		cmd = vrooliCommand(workdir, "vrooli resource restart "+shellQuoteSingle(req.ID))
	}

	sshRunner := ExecSSHRunner{}
	result, err := sshRunner.Run(ctx, cfg, cmd)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "restart_failed",
			Message: "Failed to restart " + req.Type,
			Hint:    result.Stderr,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok":        true,
		"type":      req.Type,
		"id":        req.ID,
		"output":    result.Stdout,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
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

	// Check scenarios
	scenarioRunning := false
	if liveState.Processes != nil {
		for _, s := range liveState.Processes.Scenarios {
			if s.ID == manifest.Scenario.ID {
				scenarioRunning = s.Status == "running"
				break
			}
		}
	}

	if scenarioRunning {
		report.Checks = append(report.Checks, DriftCheck{
			Category: "scenarios",
			Name:     manifest.Scenario.ID,
			Status:   "pass",
			Expected: "running",
			Actual:   "running",
		})
		report.Summary.Passed++
	} else {
		report.Checks = append(report.Checks, DriftCheck{
			Category: "scenarios",
			Name:     manifest.Scenario.ID,
			Status:   "drift",
			Expected: "running",
			Actual:   "not running",
			Message:  "Main scenario is not running",
			Actions:  []string{"restart_scenario"},
		})
		report.Summary.Drifts++
	}

	// Check expected resources
	for _, resID := range manifest.Dependencies.Resources {
		resourceFound := false
		if liveState.Processes != nil {
			for _, r := range liveState.Processes.Resources {
				if r.ID == resID {
					resourceFound = r.Status == "running"
					break
				}
			}
		}

		if resourceFound {
			report.Checks = append(report.Checks, DriftCheck{
				Category: "resources",
				Name:     resID,
				Status:   "pass",
				Expected: "running",
				Actual:   "running",
			})
			report.Summary.Passed++
		} else {
			report.Checks = append(report.Checks, DriftCheck{
				Category: "resources",
				Name:     resID,
				Status:   "drift",
				Expected: "running",
				Actual:   "not running",
				Message:  "Expected resource is not running",
				Actions:  []string{"restart_resource"},
			})
			report.Summary.Drifts++
		}
	}

	// Check for unexpected resources
	if liveState.Processes != nil {
		expectedResources := make(map[string]bool)
		for _, res := range manifest.Dependencies.Resources {
			expectedResources[res] = true
		}

		for _, r := range liveState.Processes.Resources {
			if !expectedResources[r.ID] {
				report.Checks = append(report.Checks, DriftCheck{
					Category: "resources",
					Name:     r.ID,
					Status:   "warning",
					Expected: "not specified in manifest",
					Actual:   "running",
					Message:  "Resource is running but not in manifest",
					Actions:  []string{"stop", "add_to_manifest"},
				})
				report.Summary.Warnings++
			}
		}
	}

	// Check for unexpected processes with ports
	if liveState.Processes != nil {
		for _, u := range liveState.Processes.Unexpected {
			if u.Port > 0 {
				report.Checks = append(report.Checks, DriftCheck{
					Category: "ports",
					Name:     intToStr(u.Port),
					Status:   "warning",
					Expected: "no unexpected process",
					Actual:   u.Command,
					Message:  "Unexpected process listening on port",
					Actions:  []string{"kill_pid"},
				})
				report.Summary.Warnings++
			}
		}
	}

	// Check Caddy/TLS
	if liveState.Caddy != nil {
		if liveState.Caddy.Running {
			report.Checks = append(report.Checks, DriftCheck{
				Category: "edge",
				Name:     "caddy",
				Status:   "pass",
				Expected: "running",
				Actual:   "running",
			})
			report.Summary.Passed++

			if liveState.Caddy.TLS.Valid {
				report.Checks = append(report.Checks, DriftCheck{
					Category: "edge",
					Name:     "tls",
					Status:   "pass",
					Expected: "valid certificate",
					Actual:   "valid certificate",
				})
				report.Summary.Passed++
			}
		} else {
			report.Checks = append(report.Checks, DriftCheck{
				Category: "edge",
				Name:     "caddy",
				Status:   "drift",
				Expected: "running",
				Actual:   "not running",
				Message:  "Caddy reverse proxy is not running",
				Actions:  []string{"restart_caddy"},
			})
			report.Summary.Drifts++
		}
	}

	return report
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
