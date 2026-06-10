package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/ecosystem-manager/api/pkg/importance"
)

type ImportanceHandlers struct {
	service *importance.Service
}

func NewImportanceHandlers(service *importance.Service) *ImportanceHandlers {
	return &ImportanceHandlers{service: service}
}

func (h *ImportanceHandlers) GetImportanceHandler(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.service == nil {
		writeError(w, "Importance service unavailable", http.StatusServiceUnavailable)
		return
	}
	refresh := r.URL.Query().Get("refresh") == "true"
	report, err := h.service.Report(r.Context(), refresh)
	if err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(report); err != nil {
		writeError(w, "Failed to encode importance report", http.StatusInternalServerError)
	}
}
