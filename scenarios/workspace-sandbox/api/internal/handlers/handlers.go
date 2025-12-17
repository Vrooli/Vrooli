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
	"workspace-sandbox/internal/sandbox"
	"workspace-sandbox/internal/types"
)

// Handlers contains dependencies for HTTP handlers.
// Dependencies are expressed as interfaces to enable testing with mocks.
type Handlers struct {
	Service sandbox.ServiceAPI // Service interface for testability
	Driver  driver.Driver
	DB      Pinger
	Config  config.Config // Unified configuration for accessing levers
}

// Version is the API version string.
// This is set at build time or defaults to "dev".
var Version = "1.0.0"

// Pinger is an interface for checking database connectivity.
type Pinger interface {
	PingContext(ctx context.Context) error
}

// --- Response Types ---

// ErrorResponse represents a standard error response.
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Success bool   `json:"success"`
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
		h.JSONError(w, domainErr.Error(), domainErr.HTTPStatus())
		return true
	}

	// Fallback to internal server error for unknown errors
	h.JSONError(w, err.Error(), http.StatusInternalServerError)
	return true
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
