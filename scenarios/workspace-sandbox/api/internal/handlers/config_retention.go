package handlers

// config_retention.go — diff-archive retention configuration endpoints.
//
//   GET /api/v1/config/retention   → returns the current RetentionConfig
//   PUT /api/v1/config/retention   → validates and persists a new value
//
// The handler delegates persistence to RetentionStore, which is the
// single source of truth for retention values. The reconciler reads
// from the same store on every tick, so an accepted PUT takes effect
// on the next reconciler pass without restarting the service.
//
// Field semantics:
//
//   maxArchiveAgeDays      — 0 disables age-based eviction
//   maxArchiveSizeBytes    — 0 disables global size budget
//   maxArchivesPerProject  — 0 disables per-project cap
//
// All values must be >= 0; negatives are rejected with 400.

import (
	"encoding/json"
	"net/http"

	"workspace-sandbox/internal/config"
)

// retentionResponse is the wire shape returned by GET. Matches
// config.RetentionConfig field-for-field via the JSON tags.
type retentionResponse struct {
	MaxArchiveAgeDays     int   `json:"maxArchiveAgeDays"`
	MaxArchiveSizeBytes   int64 `json:"maxArchiveSizeBytes"`
	MaxArchivesPerProject int   `json:"maxArchivesPerProject"`
}

// retentionRequest is the wire shape accepted by PUT. Pointer fields
// distinguish "absent" from "explicitly zero" so callers can change
// one lever without re-sending the others.
type retentionRequest struct {
	MaxArchiveAgeDays     *int   `json:"maxArchiveAgeDays"`
	MaxArchiveSizeBytes   *int64 `json:"maxArchiveSizeBytes"`
	MaxArchivesPerProject *int   `json:"maxArchivesPerProject"`
}

// GetRetention returns the current retention config.
// GET /api/v1/config/retention
func (h *Handlers) GetRetention(w http.ResponseWriter, r *http.Request) {
	if h.RetentionStore == nil {
		h.JSONError(w, "retention store not configured", http.StatusServiceUnavailable)
		return
	}
	cfg := h.RetentionStore.Get()
	h.JSONSuccess(w, retentionResponse{
		MaxArchiveAgeDays:     cfg.MaxArchiveAgeDays,
		MaxArchiveSizeBytes:   cfg.MaxArchiveSizeBytes,
		MaxArchivesPerProject: cfg.MaxArchivesPerProject,
	})
}

// UpdateRetention validates and persists a new retention config.
// PUT /api/v1/config/retention
//
// Partial updates are supported: omit a field to leave its current
// value unchanged. To explicitly disable a lever set it to 0.
func (h *Handlers) UpdateRetention(w http.ResponseWriter, r *http.Request) {
	if h.RetentionStore == nil {
		h.JSONError(w, "retention store not configured", http.StatusServiceUnavailable)
		return
	}

	var req retentionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.JSONError(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	cur := h.RetentionStore.Get()
	next := config.RetentionConfig{
		MaxArchiveAgeDays:     cur.MaxArchiveAgeDays,
		MaxArchiveSizeBytes:   cur.MaxArchiveSizeBytes,
		MaxArchivesPerProject: cur.MaxArchivesPerProject,
	}
	if req.MaxArchiveAgeDays != nil {
		next.MaxArchiveAgeDays = *req.MaxArchiveAgeDays
	}
	if req.MaxArchiveSizeBytes != nil {
		next.MaxArchiveSizeBytes = *req.MaxArchiveSizeBytes
	}
	if req.MaxArchivesPerProject != nil {
		next.MaxArchivesPerProject = *req.MaxArchivesPerProject
	}

	if err := h.RetentionStore.Set(next); err != nil {
		h.JSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.JSONSuccess(w, retentionResponse{
		MaxArchiveAgeDays:     next.MaxArchiveAgeDays,
		MaxArchiveSizeBytes:   next.MaxArchiveSizeBytes,
		MaxArchivesPerProject: next.MaxArchivesPerProject,
	})
}
