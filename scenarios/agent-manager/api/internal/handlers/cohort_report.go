package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"agent-manager/internal/runreport"
	"github.com/google/uuid"
)

// GetCohortReport returns a compact ranked projection for explicitly selected
// runs. Selection is intentionally explicit so comparability is reproducible
// and the default response can never leak a fleet-wide transcript.
func (h *Handler) GetCohortReport(w http.ResponseWriter, r *http.Request) {
	rawSelection := strings.TrimSpace(r.URL.Query().Get("run_ids"))
	if name := strings.TrimSpace(r.URL.Query().Get("cohort")); name != "" {
		_, cohort, err := h.svc.ShowCohort(r.Context(), name, 100)
		if err != nil {
			writeError(w, r, err)
			return
		}
		rawSelection = strings.Join(cohort.RunIDs, ",")
	}
	rawIDs := strings.Split(rawSelection, ",")
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

func (h *Handler) GetGoalCohort(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.URL.Query().Get("cohort"))
	if name == "" {
		writeSimpleError(w, r, "cohort", "cohort is required")
		return
	}
	limit := 100
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, parseErr := strconv.Atoi(raw)
		if parseErr != nil || parsed < 1 || parsed > 5000 {
			writeSimpleError(w, r, "limit", "limit must be between 1 and 5000")
			return
		}
		limit = parsed
	}
	_, cohort, err := h.svc.ShowCohort(r.Context(), name, limit)
	if err != nil {
		writeError(w, r, err)
		return
	}
	reports := make([]*runreport.RunReport, 0, len(cohort.RunIDs))
	for _, rawID := range cohort.RunIDs {
		id, parseErr := uuid.Parse(rawID)
		if parseErr != nil {
			writeError(w, r, parseErr)
			return
		}
		report, reportErr := h.svc.BuildRunReport(r.Context(), id)
		if reportErr != nil {
			writeError(w, r, reportErr)
			return
		}
		reports = append(reports, report)
	}
	writeJSON(w, http.StatusOK, runreport.BuildGoalCohort(reports))
}
