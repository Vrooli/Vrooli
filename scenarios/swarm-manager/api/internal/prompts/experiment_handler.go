package prompts

import (
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/experiment"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/promptmanager"
)

// ExperimentHandler serves experiment analysis endpoints.
type ExperimentHandler struct {
	client promptmanager.ExperimentClient
}

// NewExperimentHandler creates a handler for experiment results.
func NewExperimentHandler(client promptmanager.ExperimentClient) *ExperimentHandler {
	return &ExperimentHandler{client: client}
}

// RegisterRoutes registers experiment-related routes on the given router.
func (h *ExperimentHandler) RegisterRoutes(r *mux.Router) {
	if h == nil || h.client == nil {
		return
	}
	r.HandleFunc("/api/v1/prompts/experiments/{id}/results", h.GetResults).Methods(http.MethodGet)
}

// GetResults fetches raw outcomes from prompt-manager, analyzes them, and returns results.
func (h *ExperimentHandler) GetResults(w http.ResponseWriter, r *http.Request) {
	experimentID := strings.TrimSpace(mux.Vars(r)["id"])
	if experimentID == "" {
		apierr.MapError(w, "[experiments] results", apierr.BadRequest("experiment id is required"))
		return
	}

	outcomes, err := h.client.ListExperimentOutcomes(r.Context(), experimentID)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "status 404") {
			apierr.MapError(w, "[experiments] results", apierr.NotFound("experiment not found"))
			return
		}
		apierr.MapError(w, "[experiments] results", apierr.Internal("failed to fetch experiment outcomes"))
		return
	}

	results, err := experiment.Analyze(outcomes)
	if err != nil {
		apierr.MapError(w, "[experiments] results", apierr.Internal("failed to analyze outcomes"))
		return
	}
	results.ExperimentID = experimentID

	if err := httputil.JSON(w, results); err != nil {
		apierr.MapError(w, "[experiments] results", apierr.Internal("failed to encode response"))
	}
}
