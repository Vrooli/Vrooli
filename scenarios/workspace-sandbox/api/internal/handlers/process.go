package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"workspace-sandbox/internal/driver"
	"workspace-sandbox/internal/process"
	"workspace-sandbox/internal/types"
)

// --- Process Management Endpoints (OT-P0-003, OT-P0-008) ---

// ExecRequest represents a request to execute a command in a sandbox.
type ExecRequest struct {
	Command      string            `json:"command"`
	Args         []string          `json:"args,omitempty"`
	AllowNetwork bool              `json:"allowNetwork,omitempty"`
	Env          map[string]string `json:"env,omitempty"`
	WorkingDir   string            `json:"workingDir,omitempty"`
	SessionID    string            `json:"sessionId,omitempty"`
}

// ExecResponse represents the result of executing a command.
type ExecResponse struct {
	ExitCode int    `json:"exitCode"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	PID      int    `json:"pid,omitempty"`
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

	// Get overlayfs driver for exec
	overlayDriver, ok := h.Driver.(*driver.OverlayfsDriver)
	if !ok {
		h.JSONError(w, "exec requires overlayfs driver", http.StatusServiceUnavailable)
		return
	}

	// Build bwrap config
	cfg := driver.DefaultBwrapConfig()
	cfg.AllowNetwork = req.AllowNetwork
	if req.WorkingDir != "" {
		cfg.WorkingDir = req.WorkingDir
	}
	for k, v := range req.Env {
		cfg.Env[k] = v
	}

	// Execute the command
	result, err := overlayDriver.Exec(r.Context(), sb, cfg, req.Command, req.Args...)
	if err != nil {
		h.JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Track the process if tracker is available
	if h.ProcessTracker != nil && result.PID > 0 {
		h.ProcessTracker.Track(id, result.PID, req.Command, req.SessionID)
	}

	h.JSONSuccess(w, ExecResponse{
		ExitCode: result.ExitCode,
		Stdout:   string(result.Stdout),
		Stderr:   string(result.Stderr),
		PID:      result.PID,
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

	// Get overlayfs driver
	overlayDriver, ok := h.Driver.(*driver.OverlayfsDriver)
	if !ok {
		h.JSONError(w, "process start requires overlayfs driver", http.StatusServiceUnavailable)
		return
	}

	// Build bwrap config
	cfg := driver.DefaultBwrapConfig()
	cfg.AllowNetwork = req.AllowNetwork
	if req.WorkingDir != "" {
		cfg.WorkingDir = req.WorkingDir
	}
	for k, v := range req.Env {
		cfg.Env[k] = v
	}

	// Start the process
	pid, err := overlayDriver.StartProcess(r.Context(), sb, cfg, req.Command, req.Args...)
	if err != nil {
		h.JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Track the process
	var trackedProc *process.TrackedProcess
	if h.ProcessTracker != nil {
		trackedProc, _ = h.ProcessTracker.Track(id, pid, req.Command, req.SessionID)
	}

	response := map[string]interface{}{
		"pid":       pid,
		"sandboxId": id,
		"command":   req.Command,
	}
	if trackedProc != nil {
		response["startedAt"] = trackedProc.StartedAt
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
