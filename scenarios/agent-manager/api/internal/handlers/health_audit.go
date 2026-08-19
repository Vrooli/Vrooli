// HTTP handlers for the persisted health audit (Phase 3).
//
// Three endpoints over health.Store:
//
//   GET /api/v1/health/models   — current snapshot of every (runner, model)
//   GET /api/v1/health/runners  — current snapshot of every runner
//   GET /api/v1/health/audit    — paginated audit history with filters
//
// These are the sole health-audit HTTP surface; portable desired role state
// lives under /api/v1/role-policy/catalog.

package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"agent-manager/internal/health"
	"agent-manager/internal/modelpolicydrift"
	"github.com/vrooli/cli-core/agentcatalog"

	"github.com/gorilla/mux"
)

// HealthAuditHandler exposes the persisted audit endpoints.
type HealthAuditHandler struct {
	store            *health.Store
	catalogFreshness func(context.Context) []agentcatalog.CatalogFreshness
	modelPolicyDrift *modelpolicydrift.Scheduler
}

// NewHealthAuditHandler wires a new handler over the persisted Store.
func NewHealthAuditHandler(store *health.Store) *HealthAuditHandler {
	return &HealthAuditHandler{store: store}
}

// WithCatalogFreshness adds resource-owned catalog age to the runner health
// surface. The callback keeps filesystem/source policy out of the audit store.
func (h *HealthAuditHandler) WithCatalogFreshness(reader func(context.Context) []agentcatalog.CatalogFreshness) *HealthAuditHandler {
	h.catalogFreshness = reader
	return h
}

func (h *HealthAuditHandler) WithModelPolicyDrift(scheduler *modelpolicydrift.Scheduler) *HealthAuditHandler {
	h.modelPolicyDrift = scheduler
	return h
}

// RegisterRoutes registers the three health endpoints.
func (h *HealthAuditHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/health/models", h.GetModels).Methods("GET")
	r.HandleFunc("/api/v1/health/runners", h.GetRunners).Methods("GET")
	r.HandleFunc("/api/v1/health/audit", h.GetAudit).Methods("GET")
	r.HandleFunc("/api/v1/health/model-policy-drift", h.GetModelPolicyDrift).Methods("GET")
}

func (h *HealthAuditHandler) GetModelPolicyDrift(w http.ResponseWriter, r *http.Request) {
	if h.modelPolicyDrift == nil {
		writeJSON(w, http.StatusOK, modelpolicydrift.Snapshot{Status: "not_measured", Total: 4})
		return
	}
	writeJSON(w, http.StatusOK, h.modelPolicyDrift.Snapshot())
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
	Runner      string                         `json:"runner"`
	Status      string                         `json:"status"`
	LastChecked time.Time                      `json:"last_checked"`
	Reason      string                         `json:"reason,omitempty"`
	Message     string                         `json:"message,omitempty"`
	Catalog     *agentcatalog.CatalogFreshness `json:"catalog,omitempty"`
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
	catalogs := make(map[string]agentcatalog.CatalogFreshness)
	if h.catalogFreshness != nil {
		for _, catalog := range h.catalogFreshness(r.Context()) {
			catalogs[catalog.Runner] = catalog
		}
	}
	for runner, entry := range snap.Runners {
		row := RunnerHealthRow{
			Runner:      runner,
			Status:      string(entry.Status),
			LastChecked: entry.LastChecked,
			Reason:      entry.Reason,
			Message:     entry.Message,
		}
		if catalog, ok := catalogs[runner]; ok {
			row.Catalog = &catalog
		}
		rows = append(rows, row)
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
