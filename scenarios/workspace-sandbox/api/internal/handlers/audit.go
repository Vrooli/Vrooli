package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"workspace-sandbox/internal/types"
)

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
		"events":    events,
		"total":     total,
		"limit":     limit,
		"offset":    offset,
		"hasMore":   offset+len(events) < total,
		"sandboxId": sandboxID,
		"timestamp": time.Now(),
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
