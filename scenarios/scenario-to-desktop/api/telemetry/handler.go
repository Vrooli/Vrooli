package telemetry

import (
	"net/http"

	"github.com/gorilla/mux"
)

// Handler owns the raw telemetry export stream. Telemetry ingestion and all
// metadata operations are served by the generated TelemetryService contract.
type Handler struct {
	service Service
}

// NewHandler creates a new telemetry handler.
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// RegisterDownloadRoute retains the only telemetry HTTP route because it
// streams a raw JSONL export rather than a unary domain response.
func (h *Handler) RegisterDownloadRoute(r *mux.Router) {
	r.HandleFunc("/api/v1/deployment/telemetry/{scenario_name}/download", h.DownloadHandler).Methods("GET", "OPTIONS")
}

// DownloadHandler serves the raw telemetry file.
func (h *Handler) DownloadHandler(w http.ResponseWriter, r *http.Request) {
	scenario := mux.Vars(r)["scenario_name"]
	if scenario == "" {
		http.Error(w, "scenario_name is required", http.StatusBadRequest)
		return
	}

	filePath := h.service.GetFilePath(scenario)
	result, err := h.service.GetSummary(r.Context(), scenario)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !result.Exists {
		http.Error(w, "telemetry not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	http.ServeFile(w, r, filePath)
}
