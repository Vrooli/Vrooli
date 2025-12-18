package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"workspace-sandbox/internal/config"
	"workspace-sandbox/internal/driver"
	"workspace-sandbox/internal/process"
	"workspace-sandbox/internal/types"
)

// --- Process Management Endpoints (OT-P0-003, OT-P0-008) ---

// applyResourceLimitDefaults applies default resource limits from ExecutionConfig
// when request values are 0, and clamps to maximum allowed values.
func applyResourceLimitDefaults(
	req driver.ResourceLimits,
	execCfg config.ExecutionConfig,
) driver.ResourceLimits {
	defaults := execCfg.DefaultResourceLimits
	maxes := execCfg.MaxResourceLimits

	result := req

	// Apply defaults for zero values
	if result.MemoryLimitMB == 0 && defaults.MemoryLimitMB > 0 {
		result.MemoryLimitMB = defaults.MemoryLimitMB
	}
	if result.CPUTimeSec == 0 && defaults.CPUTimeSec > 0 {
		result.CPUTimeSec = defaults.CPUTimeSec
	}
	if result.MaxProcesses == 0 && defaults.MaxProcesses > 0 {
		result.MaxProcesses = defaults.MaxProcesses
	}
	if result.MaxOpenFiles == 0 && defaults.MaxOpenFiles > 0 {
		result.MaxOpenFiles = defaults.MaxOpenFiles
	}
	if result.TimeoutSec == 0 && defaults.TimeoutSec > 0 {
		result.TimeoutSec = defaults.TimeoutSec
	}

	// Clamp to maximums (0 = no maximum)
	if maxes.MemoryLimitMB > 0 && result.MemoryLimitMB > maxes.MemoryLimitMB {
		result.MemoryLimitMB = maxes.MemoryLimitMB
	}
	if maxes.CPUTimeSec > 0 && result.CPUTimeSec > maxes.CPUTimeSec {
		result.CPUTimeSec = maxes.CPUTimeSec
	}
	if maxes.MaxProcesses > 0 && result.MaxProcesses > maxes.MaxProcesses {
		result.MaxProcesses = maxes.MaxProcesses
	}
	if maxes.MaxOpenFiles > 0 && result.MaxOpenFiles > maxes.MaxOpenFiles {
		result.MaxOpenFiles = maxes.MaxOpenFiles
	}
	if maxes.TimeoutSec > 0 && result.TimeoutSec > maxes.TimeoutSec {
		result.TimeoutSec = maxes.TimeoutSec
	}

	return result
}

// convertProfileToDriver converts a config.IsolationProfile to driver.IsolationProfile.
func convertProfileToDriver(p *config.IsolationProfile) *driver.IsolationProfile {
	if p == nil {
		return nil
	}
	return &driver.IsolationProfile{
		ID:             p.ID,
		Name:           p.Name,
		Description:    p.Description,
		Builtin:        p.Builtin,
		NetworkAccess:  p.NetworkAccess,
		ReadOnlyBinds:  p.ReadOnlyBinds,
		ReadWriteBinds: p.ReadWriteBinds,
		Environment:    p.Environment,
		Hostname:       p.Hostname,
	}
}

// ExecRequest represents a request to execute a command in a sandbox.
type ExecRequest struct {
	Command      string            `json:"command"`
	Args         []string          `json:"args,omitempty"`
	AllowNetwork bool              `json:"allowNetwork,omitempty"`
	Env          map[string]string `json:"env,omitempty"`
	WorkingDir   string            `json:"workingDir,omitempty"`
	SessionID    string            `json:"sessionId,omitempty"`

	// IsolationLevel controls filesystem access.
	// "full" (default): maximum isolation, only /workspace accessible.
	// "vrooli-aware": can access Vrooli CLIs, configs, and localhost APIs.
	IsolationLevel string `json:"isolationLevel,omitempty"`

	// Resource limits (0 = unlimited)
	MemoryLimitMB int `json:"memoryLimitMB,omitempty"` // Max address space in MB
	CPUTimeSec    int `json:"cpuTimeSec,omitempty"`    // Max CPU time in seconds
	TimeoutSec    int `json:"timeoutSec,omitempty"`    // Wall-clock timeout in seconds
	MaxProcesses  int `json:"maxProcesses,omitempty"`  // Max child processes
	MaxOpenFiles  int `json:"maxOpenFiles,omitempty"`  // Max open file descriptors
}

// ExecResponse represents the result of executing a command.
type ExecResponse struct {
	ExitCode int    `json:"exitCode"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	PID      int    `json:"pid,omitempty"`
	TimedOut bool   `json:"timedOut,omitempty"` // True if process was killed due to timeout
}

// Exec handles executing a command inside a sandbox with bubblewrap isolation.
// [OT-P0-003] Bubblewrap Process Isolation
func (h *Handlers) Exec(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.JSONError(w, "invalid sandbox ID", http.StatusBadRequest)
		return
	}

	var req ExecRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Command == "" {
		h.JSONError(w, "command is required", http.StatusBadRequest)
		return
	}

	// Get the sandbox
	sb, err := h.Service.Get(r.Context(), id)
	if h.HandleDomainError(w, err) {
		return
	}

	// Verify sandbox is active
	if sb.Status != types.StatusActive {
		h.JSONError(w, "sandbox must be active to execute commands", http.StatusConflict)
		return
	}

	// Build bwrap config with resource limits and isolation level
	cfg := driver.DefaultBwrapConfig()
	if req.WorkingDir != "" {
		cfg.WorkingDir = req.WorkingDir
	}
	for k, v := range req.Env {
		cfg.Env[k] = v
	}

	// Determine isolation profile to use
	isolationLevel := req.IsolationLevel
	if isolationLevel == "" {
		isolationLevel = h.Config.Execution.DefaultIsolationProfile
	}
	if isolationLevel == "" {
		isolationLevel = "full" // Ultimate fallback
	}

	// Look up and apply isolation profile
	if h.ProfileStore != nil {
		profile, profErr := h.ProfileStore.Get(isolationLevel)
		if profErr == nil {
			driver.ApplyIsolationProfile(&cfg, convertProfileToDriver(profile))
		} else if isolationLevel == "vrooli-aware" {
			// Fallback for legacy "vrooli-aware" if not found in store
			driver.ApplyVrooliAwareConfig(&cfg)
		}
	} else if isolationLevel == "vrooli-aware" {
		driver.ApplyVrooliAwareConfig(&cfg)
	}

	// Override network if explicitly requested
	if req.AllowNetwork {
		cfg.AllowNetwork = true
	}

	// Set resource limits with defaults and clamping from ExecutionConfig
	requestedLimits := driver.ResourceLimits{
		MemoryLimitMB: req.MemoryLimitMB,
		CPUTimeSec:    req.CPUTimeSec,
		MaxProcesses:  req.MaxProcesses,
		MaxOpenFiles:  req.MaxOpenFiles,
		TimeoutSec:    req.TimeoutSec,
	}
	cfg.ResourceLimits = applyResourceLimitDefaults(requestedLimits, h.Config.Execution)

	// Execute the command (all drivers implement Exec via the Driver interface)
	result, err := h.Driver().Exec(r.Context(), sb, cfg, req.Command, req.Args...)
	if err != nil {
		h.JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Track the process if tracker is available
	if h.ProcessTracker != nil && result.PID > 0 {
		h.ProcessTracker.Track(id, result.PID, req.Command, req.SessionID)
	}

	// Determine if process timed out
	timedOut := result.ExitCode == 124 && result.Error != nil

	h.JSONSuccess(w, ExecResponse{
		ExitCode: result.ExitCode,
		Stdout:   string(result.Stdout),
		Stderr:   string(result.Stderr),
		PID:      result.PID,
		TimedOut: timedOut,
	})
}

// StartProcessRequest represents a request to start a background process.
type StartProcessRequest struct {
	Command      string            `json:"command"`
	Args         []string          `json:"args,omitempty"`
	AllowNetwork bool              `json:"allowNetwork,omitempty"`
	Env          map[string]string `json:"env,omitempty"`
	WorkingDir   string            `json:"workingDir,omitempty"`
	SessionID    string            `json:"sessionId,omitempty"`
	Name         string            `json:"name,omitempty"` // Optional friendly name for the process

	// IsolationLevel controls filesystem access.
	// "full" (default): maximum isolation, only /workspace accessible.
	// "vrooli-aware": can access Vrooli CLIs, configs, and localhost APIs.
	IsolationLevel string `json:"isolationLevel,omitempty"`

	// Resource limits (0 = unlimited)
	// Note: TimeoutSec is not enforced for background processes; use manual kill.
	MemoryLimitMB int `json:"memoryLimitMB,omitempty"` // Max address space in MB
	CPUTimeSec    int `json:"cpuTimeSec,omitempty"`    // Max CPU time in seconds
	MaxProcesses  int `json:"maxProcesses,omitempty"`  // Max child processes
	MaxOpenFiles  int `json:"maxOpenFiles,omitempty"`  // Max open file descriptors
}

// StartProcess handles starting a background process in a sandbox.
// [OT-P0-003] Bubblewrap Process Isolation
// [OT-P0-008] Process/Session Tracking
func (h *Handlers) StartProcess(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.JSONError(w, "invalid sandbox ID", http.StatusBadRequest)
		return
	}

	var req StartProcessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Command == "" {
		h.JSONError(w, "command is required", http.StatusBadRequest)
		return
	}

	// Get the sandbox
	sb, err := h.Service.Get(r.Context(), id)
	if h.HandleDomainError(w, err) {
		return
	}

	// Verify sandbox is active
	if sb.Status != types.StatusActive {
		h.JSONError(w, "sandbox must be active to start processes", http.StatusConflict)
		return
	}

	// Build bwrap config with resource limits and isolation level
	cfg := driver.DefaultBwrapConfig()
	if req.WorkingDir != "" {
		cfg.WorkingDir = req.WorkingDir
	}
	for k, v := range req.Env {
		cfg.Env[k] = v
	}

	// Determine isolation profile to use
	isolationLevel := req.IsolationLevel
	if isolationLevel == "" {
		isolationLevel = h.Config.Execution.DefaultIsolationProfile
	}
	if isolationLevel == "" {
		isolationLevel = "full" // Ultimate fallback
	}

	// Look up and apply isolation profile
	if h.ProfileStore != nil {
		profile, profErr := h.ProfileStore.Get(isolationLevel)
		if profErr == nil {
			driver.ApplyIsolationProfile(&cfg, convertProfileToDriver(profile))
		} else if isolationLevel == "vrooli-aware" {
			// Fallback for legacy "vrooli-aware" if not found in store
			driver.ApplyVrooliAwareConfig(&cfg)
		}
	} else if isolationLevel == "vrooli-aware" {
		driver.ApplyVrooliAwareConfig(&cfg)
	}

	// Override network if explicitly requested
	if req.AllowNetwork {
		cfg.AllowNetwork = true
	}

	// Set resource limits with defaults and clamping from ExecutionConfig
	// Note: TimeoutSec is not used for background processes - use manual kill
	requestedLimits := driver.ResourceLimits{
		MemoryLimitMB: req.MemoryLimitMB,
		CPUTimeSec:    req.CPUTimeSec,
		MaxProcesses:  req.MaxProcesses,
		MaxOpenFiles:  req.MaxOpenFiles,
		TimeoutSec:    0, // Not applicable for background processes
	}
	cfg.ResourceLimits = applyResourceLimitDefaults(requestedLimits, h.Config.Execution)
	cfg.ResourceLimits.TimeoutSec = 0 // Ensure timeout is never applied to background processes

	// Create pending log BEFORE starting process to capture stdout/stderr
	var pendingLog *process.PendingLog
	if h.ProcessLogger != nil {
		var logErr error
		pendingLog, logErr = h.ProcessLogger.CreatePendingLog(id)
		if logErr == nil {
			// Pass the log writer to capture process output
			cfg.LogWriter = pendingLog.Writer
		}
	}

	// Start process
	pid, err := h.Driver().StartProcess(r.Context(), sb, cfg, req.Command, req.Args...)
	if err != nil {
		// Clean up pending log if process failed to start
		if pendingLog != nil {
			h.ProcessLogger.AbortPendingLog(pendingLog)
		}
		h.JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Finalize log with actual PID now that process has started
	var logPath string
	if pendingLog != nil {
		logPath, _ = h.ProcessLogger.FinalizeLog(pendingLog, pid)
	}

	// Track the process with optional name
	var trackedProc *process.TrackedProcess
	if h.ProcessTracker != nil {
		displayName := req.Command
		if req.Name != "" {
			displayName = req.Name
		}
		trackedProc, _ = h.ProcessTracker.Track(id, pid, displayName, req.SessionID)
	}

	response := map[string]interface{}{
		"pid":       pid,
		"sandboxId": id,
		"command":   req.Command,
	}
	if req.Name != "" {
		response["name"] = req.Name
	}
	if trackedProc != nil {
		response["startedAt"] = trackedProc.StartedAt
	}
	if logPath != "" {
		response["logPath"] = logPath
	}

	h.JSONCreated(w, response)
}

// ListProcesses handles listing processes for a sandbox.
// [OT-P0-008] Process/Session Tracking
func (h *Handlers) ListProcesses(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.JSONError(w, "invalid sandbox ID", http.StatusBadRequest)
		return
	}

	// Verify sandbox exists
	_, err = h.Service.Get(r.Context(), id)
	if h.HandleDomainError(w, err) {
		return
	}

	if h.ProcessTracker == nil {
		h.JSONError(w, "process tracking not available", http.StatusServiceUnavailable)
		return
	}

	// Get all processes, optionally filter by running only
	runningOnly := r.URL.Query().Get("running") == "true"
	var procs []*process.TrackedProcess
	if runningOnly {
		procs = h.ProcessTracker.GetRunningProcesses(id)
	} else {
		procs = h.ProcessTracker.GetProcesses(id)
	}

	h.JSONSuccess(w, map[string]interface{}{
		"processes": procs,
		"total":     len(procs),
		"running":   h.ProcessTracker.GetActiveCount(id),
	})
}

// KillProcess handles killing a specific process.
// [OT-P0-008] Process/Session Tracking
func (h *Handlers) KillProcess(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		h.JSONError(w, "invalid sandbox ID", http.StatusBadRequest)
		return
	}

	var pidInt int
	if _, err := parsePositiveInt(vars["pid"], &pidInt); err != nil {
		h.JSONError(w, "invalid PID", http.StatusBadRequest)
		return
	}

	// Verify sandbox exists
	_, err = h.Service.Get(r.Context(), id)
	if h.HandleDomainError(w, err) {
		return
	}

	if h.ProcessTracker == nil {
		h.JSONError(w, "process tracking not available", http.StatusServiceUnavailable)
		return
	}

	err = h.ProcessTracker.KillProcess(r.Context(), id, pidInt)
	if err != nil {
		h.JSONError(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// KillAllProcesses handles killing all processes for a sandbox.
// [OT-P0-008] Process/Session Tracking
func (h *Handlers) KillAllProcesses(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.JSONError(w, "invalid sandbox ID", http.StatusBadRequest)
		return
	}

	// Verify sandbox exists
	_, err = h.Service.Get(r.Context(), id)
	if h.HandleDomainError(w, err) {
		return
	}

	if h.ProcessTracker == nil {
		h.JSONError(w, "process tracking not available", http.StatusServiceUnavailable)
		return
	}

	killed, errs := h.ProcessTracker.KillAll(r.Context(), id)

	response := map[string]interface{}{
		"killed": killed,
	}
	if len(errs) > 0 {
		errStrings := make([]string, len(errs))
		for i, e := range errs {
			errStrings[i] = e.Error()
		}
		response["errors"] = errStrings
	}

	h.JSONSuccess(w, response)
}

// ProcessStats handles getting aggregate process statistics.
// [OT-P0-008] Process/Session Tracking
func (h *Handlers) ProcessStats(w http.ResponseWriter, r *http.Request) {
	if h.ProcessTracker == nil {
		h.JSONError(w, "process tracking not available", http.StatusServiceUnavailable)
		return
	}

	stats := h.ProcessTracker.GetAllStats()
	h.JSONSuccess(w, stats)
}

// BwrapInfo handles getting bubblewrap capabilities information.
// [OT-P0-003] Bubblewrap Process Isolation
func (h *Handlers) BwrapInfo(w http.ResponseWriter, r *http.Request) {
	info, err := driver.GetBwrapInfo(r.Context())
	if err != nil {
		h.JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.JSONSuccess(w, info)
}

// --- Process Log Endpoints (Phase 2) ---

// GetProcessLogs returns logs for a specific process.
// Query parameters:
//   - tail: number of lines from end (default: all)
//   - offset: byte offset to start reading from
func (h *Handlers) GetProcessLogs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		h.JSONError(w, "invalid sandbox ID", http.StatusBadRequest)
		return
	}

	pid, err := strconv.Atoi(vars["pid"])
	if err != nil || pid <= 0 {
		h.JSONError(w, "invalid PID", http.StatusBadRequest)
		return
	}

	// Verify sandbox exists
	_, err = h.Service.Get(r.Context(), id)
	if h.HandleDomainError(w, err) {
		return
	}

	if h.ProcessLogger == nil {
		h.JSONError(w, "process logging not available", http.StatusServiceUnavailable)
		return
	}

	// Parse query parameters
	var tail int
	var offset int64
	if tailStr := r.URL.Query().Get("tail"); tailStr != "" {
		tail, _ = strconv.Atoi(tailStr)
	}
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		offset, _ = strconv.ParseInt(offsetStr, 10, 64)
	}

	// Get log metadata
	logInfo, err := h.ProcessLogger.GetLog(id, pid)
	if err != nil {
		h.JSONError(w, fmt.Sprintf("log not found: %v", err), http.StatusNotFound)
		return
	}

	// Read log content
	content, err := h.ProcessLogger.ReadLog(id, pid, tail, offset)
	if err != nil {
		h.JSONError(w, fmt.Sprintf("failed to read log: %v", err), http.StatusInternalServerError)
		return
	}

	h.JSONSuccess(w, map[string]interface{}{
		"pid":       pid,
		"sandboxId": id,
		"path":      logInfo.Path,
		"sizeBytes": logInfo.SizeBytes,
		"isActive":  logInfo.IsActive,
		"content":   string(content),
	})
}

// StreamProcessLogs streams logs for a specific process via Server-Sent Events (SSE).
// Clients connect and receive log updates in real-time until the process exits
// or the connection is closed.
func (h *Handlers) StreamProcessLogs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		h.JSONError(w, "invalid sandbox ID", http.StatusBadRequest)
		return
	}

	pid, err := strconv.Atoi(vars["pid"])
	if err != nil || pid <= 0 {
		h.JSONError(w, "invalid PID", http.StatusBadRequest)
		return
	}

	// Verify sandbox exists
	_, err = h.Service.Get(r.Context(), id)
	if h.HandleDomainError(w, err) {
		return
	}

	if h.ProcessLogger == nil {
		h.JSONError(w, "process logging not available", http.StatusServiceUnavailable)
		return
	}

	// Verify log exists
	_, err = h.ProcessLogger.GetLog(id, pid)
	if err != nil {
		h.JSONError(w, fmt.Sprintf("log not found: %v", err), http.StatusNotFound)
		return
	}

	// Set up SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

	flusher, ok := w.(http.Flusher)
	if !ok {
		h.JSONError(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	// Stream logs until context is canceled or process ends
	ctx := r.Context()
	err = h.ProcessLogger.StreamLog(ctx, id, pid, func(chunk []byte) {
		// Send SSE data event
		fmt.Fprintf(w, "data: %s\n\n", string(chunk))
		flusher.Flush()
	})

	if err != nil && err != ctx.Err() {
		// Send error event before closing
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
		flusher.Flush()
	}

	// Send end event
	fmt.Fprintf(w, "event: end\ndata: stream closed\n\n")
	flusher.Flush()
}

// ListProcessLogs returns all log files for a sandbox.
func (h *Handlers) ListProcessLogs(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.JSONError(w, "invalid sandbox ID", http.StatusBadRequest)
		return
	}

	// Verify sandbox exists
	_, err = h.Service.Get(r.Context(), id)
	if h.HandleDomainError(w, err) {
		return
	}

	if h.ProcessLogger == nil {
		h.JSONError(w, "process logging not available", http.StatusServiceUnavailable)
		return
	}

	logs, err := h.ProcessLogger.ListLogs(id)
	if err != nil {
		h.JSONError(w, fmt.Sprintf("failed to list logs: %v", err), http.StatusInternalServerError)
		return
	}

	h.JSONSuccess(w, map[string]interface{}{
		"logs":      logs,
		"total":     len(logs),
		"sandboxId": id,
	})
}
