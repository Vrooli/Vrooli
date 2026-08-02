package handlers

import (
	"net/http"

	"agent-manager/internal/runreport"
	"agent-manager/internal/runsignal"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// GetInvocationFacts is the explicit drill-down endpoint for normalized,
// redacted invocation evidence. Default reports remain compact.
func (h *Handler) GetInvocationFacts(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		writeSimpleError(w, r, "run_id", "invalid UUID format for run ID")
		return
	}
	if _, err := h.svc.GetRun(r.Context(), id); err != nil {
		writeError(w, r, err)
		return
	}
	facts, err := h.svc.InvocationFacts(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, runreport.InvocationFactsResponse{ClassifierVersion: runsignal.InvocationFactVersion, Availability: runreport.Availability{State: runreport.AvailabilityAvailable}, Facts: facts})
}
