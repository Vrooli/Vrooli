package handlers

import (
	"net/http"
	"strings"

	"agent-manager/internal/runreport"
	"github.com/google/uuid"
)

// GetCohortReport returns a compact ranked projection for explicitly selected
// runs. Selection is intentionally explicit so comparability is reproducible
// and the default response can never leak a fleet-wide transcript.
func (h *Handler) GetCohortReport(w http.ResponseWriter, r *http.Request) {
	rawIDs := strings.Split(strings.TrimSpace(r.URL.Query().Get("run_ids")), ",")
	if len(rawIDs) == 0 || rawIDs[0] == "" || len(rawIDs) > 100 {
		writeSimpleError(w, r, "run_ids", "provide between 1 and 100 comma-separated run UUIDs")
		return
	}
	reports := make([]*runreport.RunReport, 0, len(rawIDs))
	seen := map[uuid.UUID]bool{}
	for _, raw := range rawIDs {
		id, err := uuid.Parse(strings.TrimSpace(raw))
		if err != nil {
			writeSimpleError(w, r, "run_ids", "run IDs must be UUIDs")
			return
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		report, err := h.svc.BuildRunReport(r.Context(), id)
		if err != nil {
			writeError(w, r, err)
			return
		}
		reports = append(reports, report)
	}
	writeJSON(w, http.StatusOK, runreport.BuildCohort(reports))
}
