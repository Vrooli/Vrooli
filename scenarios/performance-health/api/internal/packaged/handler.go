package packaged

import (
	"encoding/json"
	"net/http"
	"strings"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) http.Handler { return &Handler{service: service} }

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.service == nil {
		http.Error(w, "packaged measurement service unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	switch {
	case r.URL.Path == "/api/v1/packaged/experiment/validate" && r.Method == http.MethodPost:
		var manifest ExperimentManifest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 256<<10)).Decode(&manifest); err != nil {
			http.Error(w, "invalid packaged-experiment.v1 manifest: "+err.Error(), http.StatusBadRequest)
			return
		}
		if err := manifest.Validate(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	case (r.URL.Path == "/api/v1/packaged/capture" || r.URL.Path == "/api/v1/packaged/capture/plan") && r.Method == http.MethodPost:
		var request CaptureRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32<<10)).Decode(&request); err != nil {
			http.Error(w, "invalid capture request: "+err.Error(), http.StatusBadRequest)
			return
		}
		if r.URL.Path == "/api/v1/packaged/capture/plan" {
			request.PlanOnly = true
		}
		samples, err := h.service.Capture(r.Context(), request)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(captureResult(request, samples))
	case r.URL.Path == "/api/v1/packaged/report" && r.Method == http.MethodGet:
		q := r.URL.Query()
		report, err := h.service.Report(r.Context(), q.Get("scenario"), q.Get("ramp"), q.Get("strategy"), q.Get("condition"), q.Get("experiment_id"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if q.Get("format") == "markdown" {
			w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
			_, _ = w.Write([]byte(RenderMarkdown(report)))
			return
		}
		_ = json.NewEncoder(w).Encode(report)
	case r.URL.Path == "/api/v1/packaged/compare" && r.Method == http.MethodPost:
		var request struct {
			A string `json:"a"`
			B string `json:"b"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10)).Decode(&request); err != nil {
			http.Error(w, "invalid comparison request: "+err.Error(), http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(request.A) == "" || strings.TrimSpace(request.B) == "" {
			http.Error(w, "a and b experiment IDs are required", http.StatusBadRequest)
			return
		}
		a, err := h.service.Report(r.Context(), "", "", "", "", request.A)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		b, err := h.service.Report(r.Context(), "", "", "", "", request.B)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"a": a, "b": b})
	default:
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodPost)
		http.Error(w, "not found", http.StatusNotFound)
	}
}

// captureResult keeps the REST and RPC projections of owner work identical.
func captureResult(request CaptureRequest, samples []Sample) map[string]any {
	experiment := request.Experiment
	if experiment == "" && len(samples) > 0 {
		experiment = samples[0].ExperimentID
	}
	return map[string]any{"experiment_id": experiment, "samples": samples, "planned_work": request.PlannedWork()}
}
