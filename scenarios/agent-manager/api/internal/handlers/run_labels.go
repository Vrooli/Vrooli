package handlers

import (
	"net/http"

	"agent-manager/internal/orchestration"
)

// BackfillImportedRunLabels is a narrow maintenance endpoint. The optional
// service assertion keeps label repair separate from normal run mutation.
func (h *Handler) BackfillImportedRunLabels(w http.ResponseWriter, r *http.Request) {
	service, ok := h.svc.RunService.(orchestration.RunLabelMaintenanceService)
	if !ok {
		writeSimpleError(w, r, "service", "run label backfill is unavailable")
		return
	}
	result, err := service.BackfillImportedRunLabels(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) BackfillRunSubjects(w http.ResponseWriter, r *http.Request) {
	service, ok := h.svc.RunService.(orchestration.RunSubjectMaintenanceService)
	if !ok {
		writeSimpleError(w, r, "service", "run subject backfill is unavailable")
		return
	}
	result, err := service.BackfillRunSubjects(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
