package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"workspace-sandbox/internal/types"
)

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
