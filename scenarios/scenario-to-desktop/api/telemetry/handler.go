package telemetry

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// Handler provides HTTP handlers for telemetry endpoints.
type Handler struct {
	service Service
}

// NewHandler creates a new telemetry handler.
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers telemetry routes on the given router.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	// Legacy paths for backward compatibility with deployed clients
	r.HandleFunc("/api/v1/deployment/telemetry", h.IngestHandler).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/v1/deployment/telemetry/{scenario_name}/summary", h.SummaryHandler).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/deployment/telemetry/{scenario_name}/insights", h.InsightsHandler).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/deployment/telemetry/{scenario_name}/tail", h.TailHandler).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/deployment/telemetry/{scenario_name}/download", h.DownloadHandler).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/deployment/telemetry/{scenario_name}", h.DeleteHandler).Methods("DELETE", "OPTIONS")
}

// IngestHandler stores deployment telemetry emitted by generated desktop bundles.
func (h *Handler) IngestHandler(w http.ResponseWriter, r *http.Request) {
	var request IngestRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	if request.ScenarioName == "" {
		http.Error(w, "scenario_name is required", http.StatusBadRequest)
		return
	}
	if len(request.Events) == 0 {
		http.Error(w, "events array cannot be empty", http.StatusBadRequest)
		return
	}

	filePath, ingested, err := h.service.IngestEvents(
		r.Context(),
		request.ScenarioName,
		request.DeploymentMode,
		request.Source,
		request.Events,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, IngestResponse{
		Status:         "ok",
		EventsIngested: ingested,
		OutputPath:     filePath,
	})
}

// SummaryHandler returns telemetry summary for a scenario.
func (h *Handler) SummaryHandler(w http.ResponseWriter, r *http.Request) {
	scenario := mux.Vars(r)["scenario_name"]
	if scenario == "" {
		http.Error(w, "scenario_name is required", http.StatusBadRequest)
		return
	}

	result, err := h.service.GetSummary(r.Context(), scenario)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, SummaryResponse{
		ScenarioName:   scenario,
		Exists:         result.Exists,
		FilePath:       result.FilePath,
		FileSizeBytes:  result.FileSizeBytes,
		EventCount:     result.EventCount,
		LastIngestedAt: result.LastIngestedAt,
	})
}

// InsightsHandler returns analyzed telemetry insights.
func (h *Handler) InsightsHandler(w http.ResponseWriter, r *http.Request) {
	scenario := mux.Vars(r)["scenario_name"]
	if scenario == "" {
		http.Error(w, "scenario_name is required", http.StatusBadRequest)
		return
	}

	result, err := h.service.GetInsights(r.Context(), scenario)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, InsightsResponse{
		ScenarioName:  scenario,
		Exists:        result.Exists,
		LastSession:   result.LastSession,
		LastSmokeTest: result.LastSmokeTest,
		LastError:     result.LastError,
	})
}

// TailHandler returns the last N telemetry entries.
func (h *Handler) TailHandler(w http.ResponseWriter, r *http.Request) {
	scenario := mux.Vars(r)["scenario_name"]
	if scenario == "" {
		http.Error(w, "scenario_name is required", http.StatusBadRequest)
		return
	}

	limit := parseTailLimit(r)
	result, err := h.service.GetTail(r.Context(), scenario, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, TailResponse{
		ScenarioName: scenario,
		Exists:       result.Exists,
		Limit:        result.Limit,
		TotalLines:   result.TotalLines,
		Entries:      result.Entries,
	})
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

// DeleteHandler removes all telemetry for a scenario.
func (h *Handler) DeleteHandler(w http.ResponseWriter, r *http.Request) {
	scenario := mux.Vars(r)["scenario_name"]
	if scenario == "" {
		http.Error(w, "scenario_name is required", http.StatusBadRequest)
		return
	}

	err := h.service.Delete(r.Context(), scenario)
	if err != nil {
		if err.Error() == "telemetry not found" {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": "deleted",
	})
}

// parseTailLimit parses the limit query parameter.
func parseTailLimit(r *http.Request) int {
	const (
		defaultLimit = 200
		maxLimit     = 1000
	)
	raw := r.URL.Query().Get("limit")
	if raw == "" {
		return defaultLimit
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		return defaultLimit
	}
	if parsed > maxLimit {
		return maxLimit
	}
	return parsed
}

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
