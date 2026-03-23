package initiatives

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"swarm-manager/internal/httputil"
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
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/initiatives", h.List).Methods("GET")
	r.HandleFunc("/api/v1/initiatives", h.Create).Methods("POST")
	r.HandleFunc("/api/v1/initiatives/{name}", h.Get).Methods("GET")
	r.HandleFunc("/api/v1/initiatives/{name}", h.Update).Methods("PUT")
	r.HandleFunc("/api/v1/initiatives/{name}", h.Delete).Methods("DELETE")
}

// List returns all initiatives with rollup status.
func (h *Handler) List(w http.ResponseWriter, _ *http.Request) {
	items, err := h.service.List()
	if err != nil {
		log.Printf("[initiatives] list: %v", err)
		httputil.InternalError(w, "[initiatives] list", "failed to list initiatives")
		return
	}

	resp := map[string]any{"items": items}
	if err := httputil.JSON(w, resp); err != nil {
		httputil.InternalError(w, "[initiatives] list", "failed to encode response")
	}
}

// Create creates a new initiative.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.BadRequest(w, "[initiatives] create", "invalid request body")
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		httputil.BadRequest(w, "[initiatives] create", "name is required")
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		httputil.BadRequest(w, "[initiatives] create", "title is required")
		return
	}

	init, err := h.service.Create(req)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			httputil.Conflict(w, "[initiatives] create", err.Error())
			return
		}
		log.Printf("[initiatives] create: %v", err)
		httputil.InternalError(w, "[initiatives] create", "failed to create initiative")
		return
	}

	rollup, _ := h.service.ComputeRollup(init)
	if rollup == nil {
		rollup = &RollupStatus{}
	}

	resp := InitiativeWithRollup{Initiative: *init, Rollup: *rollup}
	if err := httputil.JSONWithStatus(w, http.StatusCreated, resp); err != nil {
		httputil.InternalError(w, "[initiatives] create", "failed to encode response")
	}
}

// Get returns a single initiative with rollup status.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	if strings.TrimSpace(name) == "" {
		httputil.BadRequest(w, "[initiatives] get", "name is required")
		return
	}

	result, err := h.service.Get(name)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			httputil.NotFound(w, "[initiatives] get", "initiative not found")
			return
		}
		log.Printf("[initiatives] get: %v", err)
		httputil.InternalError(w, "[initiatives] get", "failed to load initiative")
		return
	}

	if err := httputil.JSON(w, result); err != nil {
		httputil.InternalError(w, "[initiatives] get", "failed to encode response")
	}
}

// Update modifies an existing initiative.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	if strings.TrimSpace(name) == "" {
		httputil.BadRequest(w, "[initiatives] update", "name is required")
		return
	}

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.BadRequest(w, "[initiatives] update", "invalid request body")
		return
	}

	if strings.TrimSpace(req.Title) == "" {
		httputil.BadRequest(w, "[initiatives] update", "title is required")
		return
	}
	if !ValidateStatus(req.Status) {
		httputil.BadRequest(w, "[initiatives] update", "status must be active, completed, or archived")
		return
	}

	init, err := h.service.Update(name, req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			httputil.NotFound(w, "[initiatives] update", "initiative not found")
			return
		}
		log.Printf("[initiatives] update: %v", err)
		httputil.InternalError(w, "[initiatives] update", "failed to update initiative")
		return
	}

	rollup, _ := h.service.ComputeRollup(init)
	if rollup == nil {
		rollup = &RollupStatus{}
	}

	resp := InitiativeWithRollup{Initiative: *init, Rollup: *rollup}
	if err := httputil.JSON(w, resp); err != nil {
		httputil.InternalError(w, "[initiatives] update", "failed to encode response")
	}
}

// Delete removes an initiative.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	if strings.TrimSpace(name) == "" {
		httputil.BadRequest(w, "[initiatives] delete", "name is required")
		return
	}

	if err := h.service.Delete(name); err != nil {
		log.Printf("[initiatives] delete: %v", err)
		httputil.InternalError(w, "[initiatives] delete", "failed to delete initiative")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
