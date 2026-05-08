// HTTP handlers for the persisted health audit (Phase 3).
//
// Three endpoints over health.Store:
//
//   GET /api/v1/health/models   — current snapshot of every (runner, model)
//   GET /api/v1/health/runners  — current snapshot of every runner
//   GET /api/v1/health/audit    — paginated audit history with filters
//
// These coexist with the legacy /api/v1/runner-models/health endpoint;
// the legacy endpoint preserves the nested {runners:{model:entry}} shape
// the existing UI depends on, while the new endpoints expose flat lists
// that compose better with stats and CLI tabular output.

package handlers

import (
	"net/http"
	"strconv"
	"time"

	"agent-manager/internal/health"

	"github.com/gorilla/mux"
)

// HealthAuditHandler exposes the persisted audit endpoints.
type HealthAuditHandler struct {
	store *health.Store
}

// NewHealthAuditHandler wires a new handler over the persisted Store.
func NewHealthAuditHandler(store *health.Store) *HealthAuditHandler {
	return &HealthAuditHandler{store: store}
}

// RegisterRoutes registers the three health endpoints.
func (h *HealthAuditHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/health/models", h.GetModels).Methods("GET")
	r.HandleFunc("/api/v1/health/runners", h.GetRunners).Methods("GET")
	r.HandleFunc("/api/v1/health/audit", h.GetAudit).Methods("GET")
}

// ModelHealthListResponse is the flat-list shape returned by /health/models.
type ModelHealthListResponse struct {
	Models []ModelHealthRow `json:"models"`
}

// ModelHealthRow is one (runner, model) row in the snapshot.
type ModelHealthRow struct {
	Runner      string    `json:"runner"`
	Model       string    `json:"model"`
	Status      string    `json:"status"`
	LastChecked time.Time `json:"last_checked"`
	Reason      string    `json:"reason,omitempty"`
	Message     string    `json:"message,omitempty"`
}

// RunnerHealthListResponse is the flat-list shape returned by /health/runners.
type RunnerHealthListResponse struct {
	Runners []RunnerHealthRow `json:"runners"`
}

// RunnerHealthRow is one runner row in the snapshot.
type RunnerHealthRow struct {
	Runner      string    `json:"runner"`
	Status      string    `json:"status"`
	LastChecked time.Time `json:"last_checked"`
	Reason      string    `json:"reason,omitempty"`
	Message     string    `json:"message,omitempty"`
}

// HealthAuditResponse pages through one of the audit tables.
type HealthAuditResponse struct {
	Rows  []health.AuditRow `json:"rows"`
	Limit int               `json:"limit"`
	Scope string            `json:"scope"` // "model" or "runner"
}

// GetModels returns the flat models snapshot.
func (h *HealthAuditHandler) GetModels(w http.ResponseWriter, r *http.Request) {
	if h.store == nil {
		writeJSON(w, http.StatusOK, ModelHealthListResponse{Models: []ModelHealthRow{}})
		return
	}
	snap, err := h.store.Snapshot(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "snapshot failed: "+err.Error())
		return
	}
	rows := make([]ModelHealthRow, 0)
	for runner, models := range snap.Models {
		for model, entry := range models {
			rows = append(rows, ModelHealthRow{
				Runner:      runner,
				Model:       model,
				Status:      string(entry.Status),
				LastChecked: entry.LastChecked,
				Reason:      entry.Reason,
				Message:     entry.Message,
			})
		}
	}
	writeJSON(w, http.StatusOK, ModelHealthListResponse{Models: rows})
}

// GetRunners returns the flat runners snapshot.
func (h *HealthAuditHandler) GetRunners(w http.ResponseWriter, r *http.Request) {
	if h.store == nil {
		writeJSON(w, http.StatusOK, RunnerHealthListResponse{Runners: []RunnerHealthRow{}})
		return
	}
	snap, err := h.store.Snapshot(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "snapshot failed: "+err.Error())
		return
	}
	rows := make([]RunnerHealthRow, 0, len(snap.Runners))
	for runner, entry := range snap.Runners {
		rows = append(rows, RunnerHealthRow{
			Runner:      runner,
			Status:      string(entry.Status),
			LastChecked: entry.LastChecked,
			Reason:      entry.Reason,
			Message:     entry.Message,
		})
	}
	writeJSON(w, http.StatusOK, RunnerHealthListResponse{Runners: rows})
}

// GetAudit pages through model_health_audit (default) or runner_health_audit
// (when ?scope=runner). Filters: runner, model, since, until, status, limit.
func (h *HealthAuditHandler) GetAudit(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	scope := q.Get("scope")
	if scope == "" {
		scope = "model"
	}
	if scope != "model" && scope != "runner" {
		writeJSONError(w, http.StatusBadRequest, "scope must be 'model' or 'runner'")
		return
	}

	query := health.AuditQuery{
		RunnerType: q.Get("runner"),
		ModelID:    q.Get("model"),
		Status:     health.Status(q.Get("status")),
	}
	if v := q.Get("since"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			query.Since = t
		} else {
			writeJSONError(w, http.StatusBadRequest, "since must be RFC3339")
			return
		}
	}
	if v := q.Get("until"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			query.Until = t
		} else {
			writeJSONError(w, http.StatusBadRequest, "until must be RFC3339")
			return
		}
	}
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			query.Limit = n
		}
	}
	if query.Limit == 0 {
		query.Limit = 100
	}

	if h.store == nil {
		writeJSON(w, http.StatusOK, HealthAuditResponse{Rows: []health.AuditRow{}, Limit: query.Limit, Scope: scope})
		return
	}

	var (
		rows []health.AuditRow
		err  error
	)
	if scope == "model" {
		rows, err = h.store.QueryModelAudit(r.Context(), query)
	} else {
		rows, err = h.store.QueryRunnerAudit(r.Context(), query)
	}
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "audit query failed: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, HealthAuditResponse{Rows: rows, Limit: query.Limit, Scope: scope})
}
