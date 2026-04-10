package state

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"

	httputil "scenario-to-desktop-api/shared/http"
)

// Handler provides HTTP handlers for scenario state operations.
type Handler struct {
	service *Service
}

// NewHandler creates a new state handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers state management routes on the given router.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	// State CRUD
	r.HandleFunc("/api/v1/scenarios/{scenario}/state", h.GetStateHandler).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/scenarios/{scenario}/state", h.SaveStateHandler).Methods("PUT", "OPTIONS")
	r.HandleFunc("/api/v1/scenarios/{scenario}/state", h.DeleteStateHandler).Methods("DELETE", "OPTIONS")

	// Staleness check
	r.HandleFunc("/api/v1/scenarios/{scenario}/state/check", h.CheckStalenessHandler).Methods("POST", "OPTIONS")

	// Log retrieval
	r.HandleFunc("/api/v1/scenarios/{scenario}/state/logs/{service}", h.GetLogsHandler).Methods("GET", "OPTIONS")

	// Invalidation
	r.HandleFunc("/api/v1/scenarios/{scenario}/state/invalidate", h.InvalidateHandler).Methods("POST", "OPTIONS")
}

// GetStateHandler retrieves scenario state.
// GET /api/v1/scenarios/{scenario}/state
// Query params:
//   - include_logs: bool - Include compressed logs in response
//   - validate_manifest: bool - Check if manifest has changed
//   - manifest_path: string - Override manifest path for validation
func (h *Handler) GetStateHandler(w http.ResponseWriter, r *http.Request) {
	scenarioName := mux.Vars(r)["scenario"]
	if scenarioName == "" {
		http.Error(w, "scenario name is required", http.StatusBadRequest)
		return
	}

	req := LoadStateRequest{
		IncludeLogs:      r.URL.Query().Get("include_logs") == "true",
		ValidateManifest: r.URL.Query().Get("validate_manifest") == "true",
		ManifestPath:     r.URL.Query().Get("manifest_path"),
	}

	resp, err := h.service.LoadState(r.Context(), scenarioName, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, resp)
}

// SaveStateHandler stores or updates scenario state.
// PUT /api/v1/scenarios/{scenario}/state
func (h *Handler) SaveStateHandler(w http.ResponseWriter, r *http.Request) {
	scenarioName := mux.Vars(r)["scenario"]
	if scenarioName == "" {
		http.Error(w, "scenario name is required", http.StatusBadRequest)
		return
	}

	var req SaveStateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON format: "+err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := h.service.SaveState(r.Context(), scenarioName, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, resp)
}

// DeleteStateHandler clears scenario state.
// DELETE /api/v1/scenarios/{scenario}/state
func (h *Handler) DeleteStateHandler(w http.ResponseWriter, r *http.Request) {
	scenarioName := mux.Vars(r)["scenario"]
	if scenarioName == "" {
		http.Error(w, "scenario name is required", http.StatusBadRequest)
		return
	}

	resp, err := h.service.ClearState(r.Context(), scenarioName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, resp)
}

// CheckStalenessHandler compares current config against stored state.
// POST /api/v1/scenarios/{scenario}/state/check
func (h *Handler) CheckStalenessHandler(w http.ResponseWriter, r *http.Request) {
	scenarioName := mux.Vars(r)["scenario"]
	if scenarioName == "" {
		http.Error(w, "scenario name is required", http.StatusBadRequest)
		return
	}

	var req CheckStalenessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON format: "+err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := h.service.CheckStaleness(r.Context(), scenarioName, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, resp)
}

// GetLogsHandler retrieves and decompresses logs for a service.
// GET /api/v1/scenarios/{scenario}/state/logs/{service}
func (h *Handler) GetLogsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["scenario"]
	serviceID := vars["service"]

	if scenarioName == "" || serviceID == "" {
		http.Error(w, "scenario and service are required", http.StatusBadRequest)
		return
	}

	resp, err := h.service.GetLogs(r.Context(), scenarioName, serviceID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if resp == nil {
		http.Error(w, "logs not found", http.StatusNotFound)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, resp)
}

// InvalidateHandler forces invalidation from a specific stage.
// POST /api/v1/scenarios/{scenario}/state/invalidate
type InvalidateRequest struct {
	FromStage string `json:"from_stage"`
	Reason    string `json:"reason,omitempty"`
}

func (h *Handler) InvalidateHandler(w http.ResponseWriter, r *http.Request) {
	scenarioName := mux.Vars(r)["scenario"]
	if scenarioName == "" {
		http.Error(w, "scenario name is required", http.StatusBadRequest)
		return
	}

	var req InvalidateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON format: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.FromStage == "" {
		http.Error(w, "from_stage is required", http.StatusBadRequest)
		return
	}

	reason := req.Reason
	if reason == "" {
		reason = "Manual invalidation"
	}

	if err := h.service.InvalidateStagesFrom(r.Context(), scenarioName, req.FromStage, reason); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return updated status
	status, err := h.service.GetValidationStatus(r.Context(), scenarioName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, status)
}
