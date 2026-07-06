package goals

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"

	"github.com/gorilla/mux"
)

// Handler exposes HTTP endpoints for goal operations.
type Handler struct {
	service *Service
}

// NewHandler creates a goals Handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers goal API routes. Target routes precede the {name}
// catch-all so gorilla/mux matches them first.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/goals", h.List).Methods("GET")
	r.HandleFunc("/api/v1/goals", h.Create).Methods("POST")
	r.HandleFunc("/api/v1/goals/{name}/targets", h.AddTargets).Methods("POST")
	r.HandleFunc("/api/v1/goals/{name}/targets", h.RemoveTargets).Methods("DELETE")
	r.HandleFunc("/api/v1/goals/{name}/archive-item", h.Archive).Methods("PATCH")
	r.HandleFunc("/api/v1/goals/{name}", h.Get).Methods("GET")
	r.HandleFunc("/api/v1/goals/{name}", h.Update).Methods("PUT")
	r.HandleFunc("/api/v1/goals/{name}", h.Delete).Methods("DELETE")
}

func (h *Handler) List(w http.ResponseWriter, _ *http.Request) {
	items, err := h.service.List()
	if err != nil {
		apierr.MapError(w, "[goals] list", apierr.Internal("failed to list goals"))
		return
	}
	writeJSON(w, "[goals] list", map[string]any{"items": items})
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierr.MapError(w, "[goals] create", apierr.BadRequest("invalid request body"))
		return
	}
	result, err := h.service.Create(req)
	if err != nil {
		mapServiceError(w, "[goals] create", err)
		return
	}
	if err := httputil.JSONWithStatus(w, http.StatusCreated, result); err != nil {
		apierr.MapError(w, "[goals] create", apierr.Internal("failed to encode response"))
	}
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.Get(nameVar(r))
	if err != nil {
		mapServiceError(w, "[goals] get", err)
		return
	}
	writeJSON(w, "[goals] get", result)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierr.MapError(w, "[goals] update", apierr.BadRequest("invalid request body"))
		return
	}
	result, err := h.service.Update(nameVar(r), req)
	if err != nil {
		mapServiceError(w, "[goals] update", err)
		return
	}
	writeJSON(w, "[goals] update", result)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Delete(nameVar(r)); err != nil {
		mapServiceError(w, "[goals] delete", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Archive(w http.ResponseWriter, r *http.Request) {
	g, err := h.service.Archive(nameVar(r))
	if err != nil {
		mapServiceError(w, "[goals] archive", err)
		return
	}
	writeJSON(w, "[goals] archive", g)
}

type targetsRequest struct {
	Targets []string `json:"targets"`
}

func (h *Handler) AddTargets(w http.ResponseWriter, r *http.Request) {
	var req targetsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierr.MapError(w, "[goals] add-targets", apierr.BadRequest("invalid request body"))
		return
	}
	result, err := h.service.AddTargets(nameVar(r), req.Targets)
	if err != nil {
		mapServiceError(w, "[goals] add-targets", err)
		return
	}
	writeJSON(w, "[goals] add-targets", result)
}

func (h *Handler) RemoveTargets(w http.ResponseWriter, r *http.Request) {
	var req targetsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierr.MapError(w, "[goals] remove-targets", apierr.BadRequest("invalid request body"))
		return
	}
	result, err := h.service.RemoveTargets(nameVar(r), req.Targets)
	if err != nil {
		mapServiceError(w, "[goals] remove-targets", err)
		return
	}
	writeJSON(w, "[goals] remove-targets", result)
}

func nameVar(r *http.Request) string {
	return strings.TrimSpace(mux.Vars(r)["name"])
}

func writeJSON(w http.ResponseWriter, ctx string, payload any) {
	if err := httputil.JSON(w, payload); err != nil {
		apierr.MapError(w, ctx, apierr.Internal("failed to encode response"))
	}
}

// mapServiceError maps a service error to the right HTTP status: validation
// errors become 400, not-found becomes 404, everything else 500.
func mapServiceError(w http.ResponseWriter, ctx string, err error) {
	switch {
	case errors.Is(err, ErrValidation):
		apierr.MapError(w, ctx, apierr.BadRequest("%s", strings.TrimPrefix(err.Error(), "goal validation error: ")))
	case strings.Contains(err.Error(), "not found"):
		apierr.MapError(w, ctx, apierr.NotFound("%s", err.Error()))
	default:
		slog.Error("goals handler error", "ctx", ctx, "err", err)
		apierr.MapError(w, ctx, apierr.Internal("goal operation failed"))
	}
}
