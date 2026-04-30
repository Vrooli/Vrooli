package initiatives

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"

	"github.com/gorilla/mux"
)

// Handler provides HTTP handlers for initiative operations.
type Handler struct {
	service *Service
}

// NewHandler creates an initiative Handler backed by the given service.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers initiative API routes on the given router.
// File routes are registered before entity routes so gorilla/mux matches
// /initiatives/{name}/files before /initiatives/{name}.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/initiatives", h.List).Methods("GET")
	r.HandleFunc("/api/v1/initiatives", h.Create).Methods("POST")

	// File management routes (must precede the {name} catch-all).
	r.HandleFunc("/api/v1/initiatives/{name}/files", h.ListInitiativeFiles).Methods("GET")
	r.HandleFunc("/api/v1/initiatives/{name}/files", h.UploadInitiativeFile).Methods("POST")
	r.HandleFunc("/api/v1/initiatives/{name}/files", h.OperateInitiativeFile).Methods("PATCH")
	r.HandleFunc("/api/v1/initiatives/{name}/files/{filepath:.*}", h.GetInitiativeFileContent).Methods("GET")

	// Entity routes.
	r.HandleFunc("/api/v1/initiatives/{name}/context", h.GetContext).Methods("GET")
	r.HandleFunc("/api/v1/initiatives/{name}", h.Get).Methods("GET")
	r.HandleFunc("/api/v1/initiatives/{name}", h.Update).Methods("PUT")
	r.HandleFunc("/api/v1/initiatives/{name}", h.Delete).Methods("DELETE")
	r.HandleFunc("/api/v1/initiatives/{name}/archive-item", h.Archive).Methods("PATCH")
	r.HandleFunc("/api/v1/initiatives/{name}/archive-item", h.Unarchive).Methods("DELETE")
	r.HandleFunc("/api/v1/initiatives/{name}/items", h.AddItems).Methods("POST")
	r.HandleFunc("/api/v1/initiatives/{name}/items", h.RemoveItems).Methods("DELETE")
}

// List returns all initiatives with rollup status. Optional ?scenario=csv
// filter narrows the result to initiatives whose member items target any of
// the named scenarios (derived from acceptance_allow globs).
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.List()
	if err != nil {
		slog.Error("failed to list initiatives", "error", err)
		apierr.MapError(w, "[initiatives] list", apierr.Internal("failed to list initiatives"))
		return
	}

	if scenarios := parseScenariosQuery(r); len(scenarios) > 0 {
		items = filterByTargetScenarios(items, scenarios)
	}

	resp := map[string]any{"items": items}
	if err := httputil.JSON(w, resp); err != nil {
		apierr.MapError(w, "[initiatives] list", apierr.Internal("failed to encode response"))
	}
}

// parseScenariosQuery reads the "scenario" (or "scenarios") query parameter
// as a comma-separated list. Empty entries are ignored. Returns nil when no
// filter is specified.
func parseScenariosQuery(r *http.Request) []string {
	query := r.URL.Query()
	raw := strings.TrimSpace(query.Get("scenario"))
	if raw == "" {
		raw = strings.TrimSpace(query.Get("scenarios"))
	}
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		s := strings.TrimSpace(p)
		if s != "" {
			result = append(result, s)
		}
	}
	return result
}

// filterByTargetScenarios keeps initiatives whose TargetScenarios overlap with
// any of the given scenarios. TargetScenarios is populated during List().
func filterByTargetScenarios(items []InitiativeWithRollup, scenarios []string) []InitiativeWithRollup {
	if len(scenarios) == 0 {
		return items
	}
	allow := make(map[string]bool, len(scenarios))
	for _, s := range scenarios {
		allow[s] = true
	}
	filtered := make([]InitiativeWithRollup, 0, len(items))
	for _, item := range items {
		for _, s := range item.TargetScenarios {
			if allow[s] {
				filtered = append(filtered, item)
				break
			}
		}
	}
	return filtered
}

// Create creates a new initiative.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := httputil.DecodeJSONStrict(r, &req); err != nil {
		apierr.MapError(w, "[initiatives] create", apierr.BadRequest("invalid request body"))
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		apierr.MapError(w, "[initiatives] create", apierr.BadRequest("name is required"))
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		apierr.MapError(w, "[initiatives] create", apierr.BadRequest("title is required"))
		return
	}
	if status := strings.TrimSpace(req.Status); status != "" && !ValidateStatus(status) {
		apierr.MapError(w, "[initiatives] create", apierr.BadRequest("status must be %s", UserSettableInitiativeStatusList()))
		return
	}
	if mode := strings.TrimSpace(req.Mode); mode != "" && !ValidateMode(mode) {
		apierr.MapError(w, "[initiatives] create", apierr.BadRequest("mode must be one of %s", OperatingModeList()))
		return
	}

	init, err := h.service.Create(req)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			apierr.MapError(w, "[initiatives] create", apierr.Conflict("%s", err.Error()))
			return
		}
		if errors.Is(err, ErrValidation) {
			apierr.MapError(w, "[initiatives] create", apierr.BadRequest("%s", err.Error()))
			return
		}
		slog.Error("failed to create initiative", "error", err)
		apierr.MapError(w, "[initiatives] create", apierr.Internal("failed to create initiative"))
		return
	}

	rollup, scenarios := h.service.aggregateInitiativeData(init)
	if rollup == nil {
		rollup = &RollupStatus{}
	}

	resp := InitiativeWithRollup{Initiative: *init, Rollup: *rollup, TargetScenarios: scenarios}
	if err := httputil.JSONWithStatus(w, http.StatusCreated, resp); err != nil {
		apierr.MapError(w, "[initiatives] create", apierr.Internal("failed to encode response"))
	}
}

// Get returns a single initiative with rollup status.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	if strings.TrimSpace(name) == "" {
		apierr.MapError(w, "[initiatives] get", apierr.BadRequest("name is required"))
		return
	}

	result, err := h.service.Get(name)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			apierr.MapError(w, "[initiatives] get", apierr.NotFound("initiative not found"))
			return
		}
		slog.Error("failed to get initiative", "error", err)
		apierr.MapError(w, "[initiatives] get", apierr.Internal("failed to load initiative"))
		return
	}

	if err := httputil.JSON(w, result); err != nil {
		apierr.MapError(w, "[initiatives] get", apierr.Internal("failed to encode response"))
	}
	h.service.RecordView(name)
}

// GetContext returns an initiative with its immediate neighborhood:
// members, upstream initiatives, downstream initiatives.
func (h *Handler) GetContext(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	if strings.TrimSpace(name) == "" {
		apierr.MapError(w, "[initiatives] context", apierr.BadRequest("name is required"))
		return
	}

	ctx, err := h.service.GetContext(name)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			apierr.MapError(w, "[initiatives] context", apierr.NotFound("initiative not found"))
			return
		}
		slog.Error("failed to load initiative context", "error", err)
		apierr.MapError(w, "[initiatives] context", apierr.Internal("failed to load initiative context"))
		return
	}

	if err := httputil.JSON(w, ctx); err != nil {
		apierr.MapError(w, "[initiatives] context", apierr.Internal("failed to encode response"))
	}
}

// Update modifies an existing initiative.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	if strings.TrimSpace(name) == "" {
		apierr.MapError(w, "[initiatives] update", apierr.BadRequest("name is required"))
		return
	}

	var req UpdateRequest
	if err := httputil.DecodeJSONStrict(r, &req); err != nil {
		apierr.MapError(w, "[initiatives] update", apierr.BadRequest("invalid request body"))
		return
	}

	if !req.HasChanges() {
		apierr.MapError(w, "[initiatives] update", apierr.BadRequest("at least one field must be provided"))
		return
	}
	if req.Title != nil && strings.TrimSpace(*req.Title) == "" {
		apierr.MapError(w, "[initiatives] update", apierr.BadRequest("title is required"))
		return
	}
	if req.Status != nil && !ValidateStatus(strings.TrimSpace(*req.Status)) {
		apierr.MapError(w, "[initiatives] update", apierr.BadRequest("status must be %s", UserSettableInitiativeStatusList()))
		return
	}
	if req.Mode != nil {
		apierr.MapError(w, "[initiatives] update", apierr.BadRequest("mode changes must use /api/v1/initiatives/{name}/operating-mode/switch"))
		return
	}

	init, err := h.service.Update(name, req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			apierr.MapError(w, "[initiatives] update", apierr.NotFound("initiative not found"))
			return
		}
		if errors.Is(err, ErrValidation) {
			apierr.MapError(w, "[initiatives] update", apierr.BadRequest("%s", err.Error()))
			return
		}
		slog.Error("failed to update initiative", "error", err)
		apierr.MapError(w, "[initiatives] update", apierr.Internal("failed to update initiative"))
		return
	}

	rollup, scenarios := h.service.aggregateInitiativeData(init)
	if rollup == nil {
		rollup = &RollupStatus{}
	}

	resp := InitiativeWithRollup{Initiative: *init, Rollup: *rollup, TargetScenarios: scenarios}
	if err := httputil.JSON(w, resp); err != nil {
		apierr.MapError(w, "[initiatives] update", apierr.Internal("failed to encode response"))
	}
}

// Delete removes an initiative.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	if strings.TrimSpace(name) == "" {
		apierr.MapError(w, "[initiatives] delete", apierr.BadRequest("name is required"))
		return
	}

	if err := h.service.Delete(name); err != nil {
		slog.Error("failed to delete initiative", "error", err)
		apierr.MapError(w, "[initiatives] delete", apierr.Internal("failed to delete initiative"))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// respondWithRollup computes the rollup and scenario aggregation in a single
// pass, then writes the initiative response. TargetScenarios is returned on
// every single-initiative read so the UI can surface scenario chips without
// an extra round-trip.
func (h *Handler) respondWithRollup(w http.ResponseWriter, init *Initiative, ctx string) {
	rollup, scenarios := h.service.aggregateInitiativeData(init)
	if rollup == nil {
		rollup = &RollupStatus{}
	}
	resp := InitiativeWithRollup{Initiative: *init, Rollup: *rollup, TargetScenarios: scenarios}
	if err := httputil.JSON(w, resp); err != nil {
		apierr.MapError(w, ctx, apierr.Internal("failed to encode response"))
	}
}

// loadInitiative loads an initiative by name, writing an error response on failure.
func (h *Handler) loadInitiative(w http.ResponseWriter, name, ctx string) (*Initiative, bool) {
	init, err := h.service.store.Load(name)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			apierr.MapError(w, ctx, apierr.NotFound("initiative not found"))
			return nil, false
		}
		apierr.MapError(w, ctx, apierr.Internal("failed to load initiative"))
		return nil, false
	}
	return init, true
}

// Archive sets archived_at on an initiative.
func (h *Handler) Archive(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	if strings.TrimSpace(name) == "" {
		apierr.MapError(w, "[initiatives] archive", apierr.BadRequest("name is required"))
		return
	}

	init, ok := h.loadInitiative(w, name, "[initiatives] archive")
	if !ok {
		return
	}

	if init.ArchivedAt != nil {
		h.respondWithRollup(w, init, "[initiatives] archive")
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	init.ArchivedAt = &now
	init.Updated = now

	if err := h.service.store.Save(init); err != nil {
		apierr.MapError(w, "[initiatives] archive", apierr.Internal("failed to save initiative"))
		return
	}

	if h.service.eventLogger != nil {
		h.service.eventLogger.EmitInitiativeArchived(name, init.Status, now)
	}

	h.respondWithRollup(w, init, "[initiatives] archive")
}

// Unarchive clears archived_at on an initiative.
func (h *Handler) Unarchive(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	if strings.TrimSpace(name) == "" {
		apierr.MapError(w, "[initiatives] unarchive", apierr.BadRequest("name is required"))
		return
	}

	init, ok := h.loadInitiative(w, name, "[initiatives] unarchive")
	if !ok {
		return
	}

	if init.ArchivedAt == nil {
		h.respondWithRollup(w, init, "[initiatives] unarchive")
		return
	}

	prevArchivedAt := *init.ArchivedAt
	init.ArchivedAt = nil
	init.Updated = time.Now().UTC().Format(time.RFC3339)

	if err := h.service.store.Save(init); err != nil {
		apierr.MapError(w, "[initiatives] unarchive", apierr.Internal("failed to save initiative"))
		return
	}

	if h.service.eventLogger != nil {
		h.service.eventLogger.EmitInitiativeUnarchived(name, prevArchivedAt)
	}

	h.respondWithRollup(w, init, "[initiatives] unarchive")
}

// itemsRequest is the JSON body for AddItems and RemoveItems.
type itemsRequest struct {
	Items []string `json:"items"`
}

// AddItems adds item references to an existing initiative.
func (h *Handler) AddItems(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	if strings.TrimSpace(name) == "" {
		apierr.MapError(w, "[initiatives] add-items", apierr.BadRequest("name is required"))
		return
	}

	var req itemsRequest
	if err := httputil.DecodeJSONStrict(r, &req); err != nil {
		apierr.MapError(w, "[initiatives] add-items", apierr.BadRequest("invalid request body"))
		return
	}
	if len(req.Items) == 0 {
		apierr.MapError(w, "[initiatives] add-items", apierr.BadRequest("at least one item is required"))
		return
	}

	if err := h.service.AddItems(name, req.Items); err != nil {
		if strings.Contains(err.Error(), "invalid item reference") {
			apierr.MapError(w, "[initiatives] add-items", apierr.BadRequest("%s", err.Error()))
			return
		}
		if strings.Contains(err.Error(), "not found") {
			apierr.MapError(w, "[initiatives] add-items", apierr.NotFound("initiative not found"))
			return
		}
		slog.Error("failed to add items", "error", err)
		apierr.MapError(w, "[initiatives] add-items", apierr.Internal("failed to add items"))
		return
	}

	result, err := h.service.Get(name)
	if err != nil {
		slog.Error("failed to reload initiative after adding items", "error", err)
		apierr.MapError(w, "[initiatives] add-items", apierr.Internal("items added but failed to reload initiative"))
		return
	}

	if err := httputil.JSON(w, result); err != nil {
		apierr.MapError(w, "[initiatives] add-items", apierr.Internal("failed to encode response"))
	}
}

// RemoveItems removes item references from an existing initiative.
func (h *Handler) RemoveItems(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	if strings.TrimSpace(name) == "" {
		apierr.MapError(w, "[initiatives] remove-items", apierr.BadRequest("name is required"))
		return
	}

	var req itemsRequest
	if err := httputil.DecodeJSONStrict(r, &req); err != nil {
		apierr.MapError(w, "[initiatives] remove-items", apierr.BadRequest("invalid request body"))
		return
	}
	if len(req.Items) == 0 {
		apierr.MapError(w, "[initiatives] remove-items", apierr.BadRequest("at least one item is required"))
		return
	}

	if err := h.service.RemoveItems(name, req.Items); err != nil {
		if strings.Contains(err.Error(), "not found") {
			apierr.MapError(w, "[initiatives] remove-items", apierr.NotFound("initiative not found"))
			return
		}
		slog.Error("failed to remove items", "error", err)
		apierr.MapError(w, "[initiatives] remove-items", apierr.Internal("failed to remove items"))
		return
	}

	result, err := h.service.Get(name)
	if err != nil {
		slog.Error("failed to reload initiative after removing items", "error", err)
		apierr.MapError(w, "[initiatives] remove-items", apierr.Internal("items removed but failed to reload initiative"))
		return
	}

	if err := httputil.JSON(w, result); err != nil {
		apierr.MapError(w, "[initiatives] remove-items", apierr.Internal("failed to encode response"))
	}
}
