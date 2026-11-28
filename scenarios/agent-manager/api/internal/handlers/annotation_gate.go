package handlers

import (
	"net/http"

	"agent-manager/internal/annotationgate"
)

// Gate returns the annotation gate rows over a window: the same projection the
// usage.annotation_gate measure executes, in the measure's response shape, for
// callers that reach the API through the UI proxy (which forwards /api and the
// Connect procedures, not the registry's /measures).
func (h *AnnotationsHandler) Gate(w http.ResponseWriter, r *http.Request) {
	if !annotationWindowOK(w, r) {
		return
	}
	coverage, err := h.service.Coverage(r.Context(), r.URL.Query().Get("window"))
	if err != nil {
		writeAnnotationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"fields": annotationgate.Rows(coverage)})
}
