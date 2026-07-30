package handlers

import (
	"net/http"

	"agent-manager/internal/runreport"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

func (h *Handler) GetLedger(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		writeSimpleError(w, r, "run_id", "invalid UUID format for run ID")
		return
	}
	report, err := h.svc.BuildRunReport(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	calls := append([]runreport.CrossScenarioCall(nil), report.CrossScenarioCalls...)
	if r.URL.Query().Get("with_projections") != "true" {
		for index := range calls {
			calls[index].Projection = nil
		}
	}
	writeJSON(w, http.StatusOK, struct {
		LedgerAvailability     runreport.Availability         `json:"ledgerAvailability"`
		ProjectionAvailability runreport.Availability         `json:"projectionAvailability"`
		LedgerTargetRollups    []runreport.LedgerTargetRollup `json:"ledgerTargetRollups"`
		Calls                  []runreport.CrossScenarioCall  `json:"calls"`
	}{report.LedgerAvailability, report.ProjectionAvailability, report.LedgerTargetRollups, calls})
}
