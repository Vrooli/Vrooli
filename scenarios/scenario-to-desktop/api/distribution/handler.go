package distribution

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"regexp"

	"github.com/gorilla/mux"
)

// Handler provides HTTP endpoints for distribution operations.
type Handler struct {
	service Service
	repo    Repository
	store   Store
	logger  *slog.Logger
}

// HandlerOption configures a Handler.
type HandlerOption func(*Handler)

// WithHandlerService sets the distribution service.
func WithHandlerService(svc Service) HandlerOption {
	return func(h *Handler) {
		h.service = svc
	}
}

// WithHandlerRepository sets the repository.
func WithHandlerRepository(repo Repository) HandlerOption {
	return func(h *Handler) {
		h.repo = repo
	}
}

// WithHandlerStore sets the status store.
func WithHandlerStore(store Store) HandlerOption {
	return func(h *Handler) {
		h.store = store
	}
}

// WithHandlerLogger sets the logger.
func WithHandlerLogger(logger *slog.Logger) HandlerOption {
	return func(h *Handler) {
		h.logger = logger
	}
}

// NewHandler creates a new distribution handler.
func NewHandler(opts ...HandlerOption) *Handler {
	h := &Handler{}

	for _, opt := range opts {
		opt(h)
	}

	// Set defaults
	if h.logger == nil {
		h.logger = slog.Default()
	}

	return h
}

// RegisterRoutes registers all distribution routes.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	// Target management (global config)
	r.HandleFunc("/api/v1/distribution/targets", h.ListTargets).Methods("GET")
	r.HandleFunc("/api/v1/distribution/targets", h.CreateTarget).Methods("POST")
	r.HandleFunc("/api/v1/distribution/targets/{name}", h.GetTarget).Methods("GET")
	r.HandleFunc("/api/v1/distribution/targets/{name}", h.UpdateTarget).Methods("PUT")
	r.HandleFunc("/api/v1/distribution/targets/{name}", h.DeleteTarget).Methods("DELETE")

	// Validation
	r.HandleFunc("/api/v1/distribution/targets/{name}/test", h.ValidateTarget).Methods("POST")
	r.HandleFunc("/api/v1/distribution/validate", h.ValidateAllTargets).Methods("POST")

	// Distribution operations
	r.HandleFunc("/api/v1/distribution/distribute", h.Distribute).Methods("POST")
	r.HandleFunc("/api/v1/distribution/status/{distribution_id}", h.GetDistributionStatus).Methods("GET")
	r.HandleFunc("/api/v1/distribution/cancel/{distribution_id}", h.CancelDistribution).Methods("POST")
	r.HandleFunc("/api/v1/distribution/list", h.ListDistributions).Methods("GET")

	// Config info
	r.HandleFunc("/api/v1/distribution/config-path", h.GetConfigPath).Methods("GET")
}

// ListTargets returns all distribution targets.
// GET /api/v1/distribution/targets
func (h *Handler) ListTargets(w http.ResponseWriter, r *http.Request) {
	config, err := h.repo.Get(r.Context())
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to get config: "+err.Error())
		return
	}

	// Convert map to array for easier frontend consumption
	targets := make([]*DistributionTarget, 0, len(config.Targets))
	for _, target := range config.Targets {
		targets = append(targets, target)
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"targets": targets,
		"count":   len(targets),
	})
}

// CreateTargetRequest is the request body for creating a target.
type CreateTargetRequest struct {
	Name   string              `json:"name"`
	Target *DistributionTarget `json:"target"`
}

// CreateTarget creates a new distribution target.
// POST /api/v1/distribution/targets
func (h *Handler) CreateTarget(w http.ResponseWriter, r *http.Request) {
	var req CreateTargetRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	// Allow target to be sent either as nested object or flat
	if req.Target == nil {
		// Try to decode as flat target
		r.Body.Close()
		var target DistributionTarget
		if err := json.NewDecoder(r.Body).Decode(&target); err == nil && target.Name != "" {
			req.Name = target.Name
			req.Target = &target
		}
	}

	if req.Name == "" && req.Target != nil {
		req.Name = req.Target.Name
	}

	if req.Name == "" {
		h.writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	if req.Target == nil {
		h.writeError(w, http.StatusBadRequest, "target is required")
		return
	}

	// Validate target name (alphanumeric, hyphens, underscores only)
	if !isValidTargetName(req.Name) {
		h.writeError(w, http.StatusBadRequest, "invalid target name: must be alphanumeric with hyphens and underscores only")
		return
	}

	req.Target.Name = req.Name

	// Check if target already exists
	existing, _ := h.repo.GetTarget(r.Context(), req.Name)
	if existing != nil {
		h.writeError(w, http.StatusConflict, "target already exists")
		return
	}

	if err := h.repo.SaveTarget(r.Context(), req.Name, req.Target); err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to save target: "+err.Error())
		return
	}

	h.logger.Info("distribution target created", "name", req.Name)

	h.writeJSON(w, http.StatusCreated, map[string]interface{}{
		"status": "created",
		"name":   req.Name,
		"target": req.Target,
	})
}

// GetTarget returns a specific distribution target.
// GET /api/v1/distribution/targets/{name}
func (h *Handler) GetTarget(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	target, err := h.repo.GetTarget(r.Context(), name)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to get target: "+err.Error())
		return
	}

	if target == nil {
		h.writeError(w, http.StatusNotFound, "target not found")
		return
	}

	h.writeJSON(w, http.StatusOK, target)
}

// UpdateTarget updates a distribution target.
// PUT /api/v1/distribution/targets/{name}
func (h *Handler) UpdateTarget(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	var target DistributionTarget
	if err := json.NewDecoder(r.Body).Decode(&target); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	target.Name = name

	if err := h.repo.SaveTarget(r.Context(), name, &target); err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to save target: "+err.Error())
		return
	}

	h.logger.Info("distribution target updated", "name", name)

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": "updated",
		"name":   name,
		"target": &target,
	})
}

// DeleteTarget removes a distribution target.
// DELETE /api/v1/distribution/targets/{name}
func (h *Handler) DeleteTarget(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	// Check if target exists
	existing, _ := h.repo.GetTarget(r.Context(), name)
	if existing == nil {
		h.writeError(w, http.StatusNotFound, "target not found")
		return
	}

	if err := h.repo.DeleteTarget(r.Context(), name); err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to delete target: "+err.Error())
		return
	}

	h.logger.Info("distribution target deleted", "name", name)

	h.writeJSON(w, http.StatusOK, map[string]string{
		"status": "deleted",
		"name":   name,
	})
}

// ValidateTarget validates a specific target's connectivity and permissions.
// POST /api/v1/distribution/targets/{name}/test
func (h *Handler) ValidateTarget(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	result := h.service.ValidateTargets(r.Context(), []string{name})

	h.writeJSON(w, http.StatusOK, result)
}

// ValidateAllTargets validates all targets.
// POST /api/v1/distribution/validate
func (h *Handler) ValidateAllTargets(w http.ResponseWriter, r *http.Request) {
	result := h.service.ValidateTargets(r.Context(), nil)

	h.writeJSON(w, http.StatusOK, result)
}

// Distribute starts a distribution operation.
// POST /api/v1/distribution/distribute
func (h *Handler) Distribute(w http.ResponseWriter, r *http.Request) {
	var req DistributeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if req.ScenarioName == "" {
		h.writeError(w, http.StatusBadRequest, "scenario_name is required")
		return
	}

	if len(req.Artifacts) == 0 {
		h.writeError(w, http.StatusBadRequest, "artifacts is required")
		return
	}

	resp, err := h.service.Distribute(r.Context(), &req)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to start distribution: "+err.Error())
		return
	}

	h.logger.Info("distribution started",
		"distribution_id", resp.DistributionID,
		"scenario", req.ScenarioName)

	h.writeJSON(w, http.StatusAccepted, resp)
}

// GetDistributionStatus returns the status of a distribution operation.
// GET /api/v1/distribution/status/{distribution_id}
func (h *Handler) GetDistributionStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	distributionID := vars["distribution_id"]

	status, ok := h.service.GetDistributionStatus(distributionID)
	if !ok {
		h.writeError(w, http.StatusNotFound, "distribution not found")
		return
	}

	h.writeJSON(w, http.StatusOK, status)
}

// CancelDistribution cancels an in-progress distribution.
// POST /api/v1/distribution/cancel/{distribution_id}
func (h *Handler) CancelDistribution(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	distributionID := vars["distribution_id"]

	if !h.service.CancelDistribution(distributionID) {
		h.writeError(w, http.StatusNotFound, "distribution not found or already completed")
		return
	}

	h.logger.Info("distribution cancelled", "distribution_id", distributionID)

	h.writeJSON(w, http.StatusOK, map[string]string{
		"status":          "cancelled",
		"distribution_id": distributionID,
	})
}

// ListDistributions returns all tracked distributions.
// GET /api/v1/distribution/list
func (h *Handler) ListDistributions(w http.ResponseWriter, r *http.Request) {
	distributions := h.service.ListDistributions()

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"distributions": distributions,
		"count":         len(distributions),
	})
}

// GetConfigPath returns the path to the distribution config file.
// GET /api/v1/distribution/config-path
func (h *Handler) GetConfigPath(w http.ResponseWriter, r *http.Request) {
	h.writeJSON(w, http.StatusOK, map[string]string{
		"path": h.repo.GetPath(),
	})
}

// Helper functions

func (h *Handler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("failed to encode JSON response", "error", err)
	}
}

func (h *Handler) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, map[string]string{"error": message})
}

// isValidTargetName checks if a target name is valid.
var validTargetNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

func isValidTargetName(name string) bool {
	if len(name) == 0 || len(name) > 64 {
		return false
	}
	return validTargetNameRegex.MatchString(name)
}
