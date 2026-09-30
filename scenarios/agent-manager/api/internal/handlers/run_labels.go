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

// ImportTranscriptSweep runs the scheduled importer's sweep on demand. It is
// the same idempotent operation the scheduler runs, so an operator diagnosing a
// stale corpus never has to wait out the interval or reach for a second import
// path that could drift from the scheduled one.
func (h *Handler) ImportTranscriptSweep(w http.ResponseWriter, r *http.Request) {
	if h.transcriptImporter == nil {
		writeSimpleError(w, r, "service", "transcript import sweep is unavailable")
		return
	}
	summary, err := h.transcriptImporter.RunOnce(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}
