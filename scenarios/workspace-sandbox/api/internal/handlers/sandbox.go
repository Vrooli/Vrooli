package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"workspace-sandbox/internal/types"
)

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

// ListSandboxesResponse wraps the list result with health info for each sandbox.
type ListSandboxesResponse struct {
	Sandboxes  []SandboxResponse `json:"sandboxes"`
	TotalCount int               `json:"totalCount"`
	Limit      int               `json:"limit"`
	Offset     int               `json:"offset"`
}

// ListSandboxes handles sandbox listing.
// Includes mount health info for active sandboxes.
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

	// Build response with health info for each sandbox
	resp := ListSandboxesResponse{
		Sandboxes:  make([]SandboxResponse, 0, len(result.Sandboxes)),
		TotalCount: result.TotalCount,
		Limit:      result.Limit,
		Offset:     result.Offset,
	}

	for _, sb := range result.Sandboxes {
		sbResp := SandboxResponse{Sandbox: sb}

		// Verify mount integrity for active sandboxes
		if sb.Status == types.StatusActive {
			health := &MountHealthInfo{Verified: true}

			if err := h.Driver().VerifyMountIntegrity(r.Context(), sb); err != nil {
				health.Healthy = false
				health.Error = err.Error()
				health.Hint = "The sandbox mount is not accessible. Stop and Start to remount it."
			} else {
				health.Healthy = true
			}

			sbResp.MountHealth = health
		}

		resp.Sandboxes = append(resp.Sandboxes, sbResp)
	}

	h.JSONSuccess(w, resp)
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

// SandboxResponse wraps a sandbox with runtime health information.
// This provides the UI with accurate status even when the mount is unhealthy.
type SandboxResponse struct {
	*types.Sandbox

	// MountHealth indicates whether the mount is actually functional.
	// A sandbox can be "active" in the database but have an unhealthy mount.
	MountHealth *MountHealthInfo `json:"mountHealth,omitempty"`
}

// MountHealthInfo provides details about mount state for the UI.
type MountHealthInfo struct {
	// Healthy is true if the mount is verified working
	Healthy bool `json:"healthy"`

	// Verified is true if we actually checked the mount (only for active sandboxes)
	Verified bool `json:"verified"`

	// Error describes what's wrong if unhealthy
	Error string `json:"error,omitempty"`

	// Hint provides guidance for fixing the issue
	Hint string `json:"hint,omitempty"`
}

// GetSandbox handles getting a single sandbox.
// For active sandboxes, it verifies mount integrity before returning.
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

	// Build response with health info
	resp := SandboxResponse{Sandbox: sb}

	// For active sandboxes, verify mount integrity
	if sb.Status == types.StatusActive {
		health := &MountHealthInfo{Verified: true}

		if err := h.Driver().VerifyMountIntegrity(r.Context(), sb); err != nil {
			health.Healthy = false
			health.Error = err.Error()
			health.Hint = "The sandbox mount is not accessible. This can happen if the API was restarted. Stop and Start to remount it."
		} else {
			health.Healthy = true
		}

		resp.MountHealth = health
	}

	h.JSONSuccess(w, resp)
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

// StartSandbox handles starting (remounting) a stopped sandbox.
func (h *Handlers) StartSandbox(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.JSONError(w, "invalid sandbox ID", http.StatusBadRequest)
		return
	}

	sb, err := h.Service.Start(r.Context(), id)
	if h.HandleDomainError(w, err) {
		return
	}

	h.JSONSuccess(w, sb)
}
