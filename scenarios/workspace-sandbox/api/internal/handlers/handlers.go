// Package handlers provides HTTP request handlers for the sandbox API.
package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"workspace-sandbox/internal/config"
	"workspace-sandbox/internal/driver"
	"workspace-sandbox/internal/process"
	"workspace-sandbox/internal/sandbox"
	"workspace-sandbox/internal/types"
)

// StatsGetter is an interface for retrieving sandbox statistics.
type StatsGetter interface {
	GetStats(ctx context.Context) (*types.SandboxStats, error)
}

// Handlers contains dependencies for HTTP handlers.
// Dependencies are expressed as interfaces to enable testing with mocks.
type Handlers struct {
	Service        sandbox.ServiceAPI // Service interface for testability
	Driver         driver.Driver
	DB             Pinger
	Config         config.Config           // Unified configuration for accessing levers
	StatsGetter    StatsGetter             // For retrieving sandbox statistics
	ProcessTracker *process.Tracker        // For tracking sandbox processes (OT-P0-008)
	GCService      GCService               // For garbage collection operations (OT-P1-003)
}

// Version is the API version string.
// This is set at build time or defaults to "dev".
var Version = "1.0.0"

// Pinger is an interface for checking database connectivity.
type Pinger interface {
	PingContext(ctx context.Context) error
}

// --- Response Types ---

// ErrorResponse represents a standard error response with optional guidance.
type ErrorResponse struct {
	Error     string `json:"error"`
	Code      int    `json:"code"`
	Success   bool   `json:"success"`
	Hint      string `json:"hint,omitempty"`      // Actionable guidance for resolving the error
	Retryable bool   `json:"retryable,omitempty"` // Whether the operation might succeed on retry
}

// Hintable is an interface for errors that provide resolution hints.
type Hintable interface {
	Hint() string
}

// SuccessResponse wraps successful responses with metadata.
type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
}

// --- Response Helpers ---
// These helpers reduce repetition and ensure consistent response formats.

// JSONError writes a JSON error response with the given message and status code.
func (h *Handlers) JSONError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   message,
		Code:    code,
		Success: false,
	})
}

// JSONSuccess writes a JSON success response with the given data.
func (h *Handlers) JSONSuccess(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// JSONCreated writes a JSON response for newly created resources.
func (h *Handlers) JSONCreated(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(data)
}

// HandleDomainError writes an appropriate error response based on error type.
// It checks for domain errors first (which know their HTTP status),
// then falls back to internal server error for unknown errors.
// Returns true if an error was handled, false if err was nil.
func (h *Handlers) HandleDomainError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}

	// Check if error implements DomainError interface
	if domainErr, ok := err.(types.DomainError); ok {
		h.JSONDomainError(w, domainErr)
		return true
	}

	// Fallback to internal server error for unknown errors
	h.JSONError(w, err.Error(), http.StatusInternalServerError)
	return true
}

// JSONDomainError writes a JSON error response for domain errors with hints.
func (h *Handlers) JSONDomainError(w http.ResponseWriter, err types.DomainError) {
	response := ErrorResponse{
		Error:     err.Error(),
		Code:      err.HTTPStatus(),
		Success:   false,
		Retryable: err.IsRetryable(),
	}

	// Include hint if the error provides one
	if hintable, ok := err.(Hintable); ok {
		response.Hint = hintable.Hint()
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.HTTPStatus())
	json.NewEncoder(w).Encode(response)
}

// Health handles health check requests.
func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	status := "healthy"
	dbStatus := "connected"

	if err := h.DB.PingContext(r.Context()); err != nil {
		status = "unhealthy"
		dbStatus = "disconnected"
	}

	// Check driver availability using injected driver
	driverAvailable, _ := h.Driver.IsAvailable(r.Context())
	driverStatus := "available"
	if !driverAvailable {
		driverStatus = "unavailable"
	}

	response := map[string]interface{}{
		"status":    status,
		"service":   "Workspace Sandbox API",
		"version":   Version,
		"readiness": status == "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"dependencies": map[string]string{
			"database": dbStatus,
			"driver":   driverStatus,
		},
	}

	h.JSONSuccess(w, response)
}

// CreateSandbox handles sandbox creation.
func (h *Handlers) CreateSandbox(w http.ResponseWriter, r *http.Request) {
	var req types.CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	sb, err := h.Service.Create(r.Context(), &req)
	if h.HandleDomainError(w, err) {
		return
	}

	h.JSONCreated(w, sb)
}

// ListSandboxes handles sandbox listing.
func (h *Handlers) ListSandboxes(w http.ResponseWriter, r *http.Request) {
	filter := &types.ListFilter{}

	// Parse query params
	if status := r.URL.Query()["status"]; len(status) > 0 {
		for _, st := range status {
			filter.Status = append(filter.Status, types.Status(st))
		}
	}
	filter.Owner = r.URL.Query().Get("owner")
	filter.ProjectRoot = r.URL.Query().Get("projectRoot")
	filter.ScopePath = r.URL.Query().Get("scopePath")

	if limit := r.URL.Query().Get("limit"); limit != "" {
		var l int
		if _, err := parsePositiveInt(limit, &l); err == nil {
			filter.Limit = l
		}
	}
	if offset := r.URL.Query().Get("offset"); offset != "" {
		var o int
		if _, err := parsePositiveInt(offset, &o); err == nil {
			filter.Offset = o
		}
	}

	result, err := h.Service.List(r.Context(), filter)
	if h.HandleDomainError(w, err) {
		return
	}

	h.JSONSuccess(w, result)
}

// parsePositiveInt parses a string into a positive integer.
func parsePositiveInt(s string, out *int) (int, error) {
	var v int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, types.NewValidationError("", "invalid integer")
		}
		v = v*10 + int(c-'0')
	}
	*out = v
	return v, nil
}

// GetSandbox handles getting a single sandbox.
func (h *Handlers) GetSandbox(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.JSONError(w, "invalid sandbox ID", http.StatusBadRequest)
		return
	}

	sb, err := h.Service.Get(r.Context(), id)
	if h.HandleDomainError(w, err) {
		return
	}

	h.JSONSuccess(w, sb)
}

// DeleteSandbox handles sandbox deletion.
func (h *Handlers) DeleteSandbox(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.JSONError(w, "invalid sandbox ID", http.StatusBadRequest)
		return
	}

	if err := h.Service.Delete(r.Context(), id); h.HandleDomainError(w, err) {
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// StopSandbox handles stopping a sandbox.
func (h *Handlers) StopSandbox(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.JSONError(w, "invalid sandbox ID", http.StatusBadRequest)
		return
	}

	sb, err := h.Service.Stop(r.Context(), id)
	if h.HandleDomainError(w, err) {
		return
	}

	h.JSONSuccess(w, sb)
}

// GetDiff handles getting sandbox diff.
func (h *Handlers) GetDiff(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.JSONError(w, "invalid sandbox ID", http.StatusBadRequest)
		return
	}

	diff, err := h.Service.GetDiff(r.Context(), id)
	if h.HandleDomainError(w, err) {
		return
	}

	h.JSONSuccess(w, diff)
}

// Approve handles approving sandbox changes.
func (h *Handlers) Approve(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.JSONError(w, "invalid sandbox ID", http.StatusBadRequest)
		return
	}

	var req types.ApprovalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Allow empty body for approve-all
		req = types.ApprovalRequest{Mode: "all"}
	}
	req.SandboxID = id

	result, err := h.Service.Approve(r.Context(), &req)
	if h.HandleDomainError(w, err) {
		return
	}

	h.JSONSuccess(w, result)
}

// Reject handles rejecting sandbox changes.
func (h *Handlers) Reject(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.JSONError(w, "invalid sandbox ID", http.StatusBadRequest)
		return
	}

	var req struct {
		Actor string `json:"actor"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	sb, err := h.Service.Reject(r.Context(), id, req.Actor)
	if h.HandleDomainError(w, err) {
		return
	}

	h.JSONSuccess(w, sb)
}

// GetWorkspace handles getting the workspace path.
func (h *Handlers) GetWorkspace(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.JSONError(w, "invalid sandbox ID", http.StatusBadRequest)
		return
	}

	path, err := h.Service.GetWorkspacePath(r.Context(), id)
	if h.HandleDomainError(w, err) {
		return
	}

	h.JSONSuccess(w, map[string]string{"path": path})
}

// DriverInfo handles getting driver information.
func (h *Handlers) DriverInfo(w http.ResponseWriter, r *http.Request) {
	available, availErr := h.Driver.IsAvailable(r.Context())

	info := driver.Info{
		Type:        h.Driver.Type(),
		Version:     h.Driver.Version(),
		Description: "Linux overlayfs driver for copy-on-write sandboxes",
		Available:   available,
	}

	response := map[string]interface{}{
		"driver":    info,
		"available": available,
	}
	if availErr != nil {
		response["error"] = availErr.Error()
	}

	h.JSONSuccess(w, response)
}

// Stats handles getting aggregate sandbox statistics.
// This endpoint supports dashboard metrics and monitoring.
func (h *Handlers) Stats(w http.ResponseWriter, r *http.Request) {
	if h.StatsGetter == nil {
		h.JSONError(w, "stats not available", http.StatusServiceUnavailable)
		return
	}

	stats, err := h.StatsGetter.GetStats(r.Context())
	if h.HandleDomainError(w, err) {
		return
	}

	// Include timestamp for cache invalidation hints
	response := map[string]interface{}{
		"stats":     stats,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	h.JSONSuccess(w, response)
}

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

// --- Garbage Collection Endpoints [OT-P1-003] ---

// GCService is an interface for garbage collection operations.
type GCService interface {
	Run(ctx context.Context, req *types.GCRequest) (*types.GCResult, error)
}

// GC handles garbage collection requests.
// [OT-P1-003] GC/Prune Operations
func (h *Handlers) GC(w http.ResponseWriter, r *http.Request) {
	if h.GCService == nil {
		h.JSONError(w, "garbage collection service not available", http.StatusServiceUnavailable)
		return
	}

	var req types.GCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Allow empty body - use default policy
		req = types.GCRequest{}
	}

	result, err := h.GCService.Run(r.Context(), &req)
	if h.HandleDomainError(w, err) {
		return
	}

	h.JSONSuccess(w, result)
}

// GCPreview handles dry-run garbage collection preview requests.
// [OT-P1-003] GC/Prune Operations
func (h *Handlers) GCPreview(w http.ResponseWriter, r *http.Request) {
	if h.GCService == nil {
		h.JSONError(w, "garbage collection service not available", http.StatusServiceUnavailable)
		return
	}

	var req types.GCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Allow empty body - use default policy
		req = types.GCRequest{}
	}

	// Force dry run for preview
	req.DryRun = true

	result, err := h.GCService.Run(r.Context(), &req)
	if h.HandleDomainError(w, err) {
		return
	}

	h.JSONSuccess(w, result)
}

// --- Audit Log Endpoints [OT-P1-004] ---

// AuditLogGetter is an interface for retrieving audit log events.
type AuditLogGetter interface {
	GetAuditLog(ctx context.Context, sandboxID *uuid.UUID, limit, offset int) ([]*types.AuditEvent, int, error)
}

// GetAuditLog handles retrieving audit log events.
// [OT-P1-004] Audit Trail Metadata
//
// Query parameters:
//   - sandbox_id: Optional UUID to filter by sandbox
//   - limit: Maximum number of events to return (default 100, max 1000)
//   - offset: Number of events to skip for pagination
func (h *Handlers) GetAuditLog(w http.ResponseWriter, r *http.Request) {
	// Get AuditLogGetter from StatsGetter (which is the repository)
	auditGetter, ok := h.StatsGetter.(AuditLogGetter)
	if !ok {
		h.JSONError(w, "audit log not available", http.StatusServiceUnavailable)
		return
	}

	// Parse query parameters
	query := r.URL.Query()

	var sandboxID *uuid.UUID
	if idStr := query.Get("sandbox_id"); idStr != "" {
		id, err := uuid.Parse(idStr)
		if err != nil {
			h.JSONError(w, "invalid sandbox_id", http.StatusBadRequest)
			return
		}
		sandboxID = &id
	}

	limit := 100
	if limitStr := query.Get("limit"); limitStr != "" {
		if _, err := parsePositiveInt(limitStr, &limit); err != nil {
			h.JSONError(w, "invalid limit", http.StatusBadRequest)
			return
		}
	}

	offset := 0
	if offsetStr := query.Get("offset"); offsetStr != "" {
		if _, err := parsePositiveInt(offsetStr, &offset); err != nil {
			h.JSONError(w, "invalid offset", http.StatusBadRequest)
			return
		}
	}

	events, total, err := auditGetter.GetAuditLog(r.Context(), sandboxID, limit, offset)
	if err != nil {
		h.JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.JSONSuccess(w, map[string]interface{}{
		"events":     events,
		"total":      total,
		"limit":      limit,
		"offset":     offset,
		"hasMore":    offset+len(events) < total,
		"sandboxId":  sandboxID,
		"timestamp":  time.Now(),
	})
}

// GetSandboxAuditLog handles retrieving audit log for a specific sandbox.
// [OT-P1-004] Audit Trail Metadata
func (h *Handlers) GetSandboxAuditLog(w http.ResponseWriter, r *http.Request) {
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

	// Get AuditLogGetter from StatsGetter
	auditGetter, ok := h.StatsGetter.(AuditLogGetter)
	if !ok {
		h.JSONError(w, "audit log not available", http.StatusServiceUnavailable)
		return
	}

	// Parse query parameters
	query := r.URL.Query()
	limit := 100
	if limitStr := query.Get("limit"); limitStr != "" {
		if _, err := parsePositiveInt(limitStr, &limit); err != nil {
			h.JSONError(w, "invalid limit", http.StatusBadRequest)
			return
		}
	}

	offset := 0
	if offsetStr := query.Get("offset"); offsetStr != "" {
		if _, err := parsePositiveInt(offsetStr, &offset); err != nil {
			h.JSONError(w, "invalid offset", http.StatusBadRequest)
			return
		}
	}

	events, total, err := auditGetter.GetAuditLog(r.Context(), &id, limit, offset)
	if err != nil {
		h.JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.JSONSuccess(w, map[string]interface{}{
		"events":    events,
		"total":     total,
		"limit":     limit,
		"offset":    offset,
		"hasMore":   offset+len(events) < total,
		"sandboxId": id,
		"timestamp": time.Now(),
	})
}

// --- Metrics Endpoint [OT-P1-008] ---

// MetricsCollector is an interface for collecting and exporting metrics.
type MetricsCollector interface {
	ExportPrometheus() string
	Snapshot() map[string]interface{}
	SetActiveSandboxes(count int64)
	SetTotalDiskUsage(bytes int64)
	SetProcessCount(count int64)
}

// Metrics handles exporting Prometheus-format metrics.
// [OT-P1-008] Metrics/Observability
func (h *Handlers) Metrics(w http.ResponseWriter, r *http.Request, collector MetricsCollector) {
	if collector == nil {
		h.JSONError(w, "metrics not available", http.StatusServiceUnavailable)
		return
	}

	// Update gauges from current state before export
	if h.StatsGetter != nil {
		if stats, err := h.StatsGetter.GetStats(r.Context()); err == nil {
			collector.SetActiveSandboxes(stats.ActiveCount)
			collector.SetTotalDiskUsage(stats.TotalSizeBytes)
		}
	}
	if h.ProcessTracker != nil {
		stats := h.ProcessTracker.GetAllStats()
		collector.SetProcessCount(int64(stats.TotalRunning))
	}

	// Check if JSON format is requested
	if r.URL.Query().Get("format") == "json" {
		h.JSONSuccess(w, collector.Snapshot())
		return
	}

	// Default to Prometheus text format
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.Write([]byte(collector.ExportPrometheus()))
}

// --- Retry/Rebase Workflow Endpoints [OT-P2-003] ---

// CheckConflicts handles checking for conflicts between sandbox and canonical repo.
// [OT-P2-003] Retry/Rebase Workflow
//
// This endpoint checks if the canonical repo has changed since the sandbox was created
// and identifies any files that have been modified in both the sandbox and the repo.
// Use this before approving changes to determine if a rebase is needed.
func (h *Handlers) CheckConflicts(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.JSONError(w, "invalid sandbox ID", http.StatusBadRequest)
		return
	}

	result, err := h.Service.CheckConflicts(r.Context(), id)
	if h.HandleDomainError(w, err) {
		return
	}

	h.JSONSuccess(w, result)
}

// Rebase handles rebasing a sandbox against the current repo state.
// [OT-P2-003] Retry/Rebase Workflow
//
// This endpoint updates the sandbox's BaseCommitHash to the current repo state.
// After rebasing:
//   - Future conflict checks will compare against the new base commit
//   - The diff will be regenerated against the current canonical repo state
//   - If there were conflicts before, they may be resolved (or new ones detected)
//
// Request body:
//   - strategy: "regenerate" (only option currently; updates baseline without merging)
//   - actor: optional identifier for who initiated the rebase
func (h *Handlers) Rebase(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.JSONError(w, "invalid sandbox ID", http.StatusBadRequest)
		return
	}

	var req types.RebaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Allow empty body - use default strategy
		req = types.RebaseRequest{Strategy: types.RebaseStrategyRegenerate}
	}
	req.SandboxID = id

	result, err := h.Service.Rebase(r.Context(), &req)
	if h.HandleDomainError(w, err) {
		return
	}

	h.JSONSuccess(w, result)
}
