package usagepace

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"agent-manager/internal/config"
	"agent-manager/internal/pricing"
	"agent-manager/internal/usageforecast"
	"github.com/gorilla/mux"
)

const APIPath = "/api/v1/usage/pace"

type settingsService interface {
	GetOrchestrationSettings(context.Context) (*config.OrchestrationSettings, error)
	UpdateOrchestrationSettings(context.Context, *config.OrchestrationSettings) error
}

// Handler exposes pace forecasts and applies a selected tier through the
// existing orchestration-settings propagation path, which updates admission
// for new runs without restarting Agent Manager.
type Handler struct {
	readings pricing.QuotaReadingSource
	settings settingsService
	now      func() time.Time
}

func NewHandler(readings pricing.QuotaReadingSource, settings settingsService) *Handler {
	return &Handler{readings: readings, settings: settings, now: time.Now}
}

func (h *Handler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc(APIPath, h.get).Methods(http.MethodGet)
	router.HandleFunc(APIPath, h.put).Methods(http.MethodPut)
}

type response struct {
	GeneratedAt            time.Time  `json:"generated_at"`
	SelectedTier           string     `json:"selected_tier,omitempty"`
	SelectedTargetRunOutAt *time.Time `json:"selected_target_run_out_at,omitempty"`
	BaselineMaxConcurrent  int        `json:"baseline_max_concurrent_runs"`
	EffectiveMaxConcurrent int        `json:"effective_max_concurrent_runs"`
	Tiers                  []Tier     `json:"tiers"`
}

type selection struct {
	SelectedTier string `json:"selected_tier"`
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	settings, err := h.settings.GetOrchestrationSettings(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	result, err := h.forecast(r.Context(), *settings)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) put(w http.ResponseWriter, r *http.Request) {
	var input selection
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "expected JSON with selected_tier"})
		return
	}
	settings, err := h.settings.GetOrchestrationSettings(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if input.SelectedTier == "baseline" {
		settings.UsagePace.SelectedTier = ""
		settings.UsagePace.TargetRunOutAt = ""
		settings.RunExecution.MaxConcurrentRuns = settings.UsagePace.BaseMaxConcurrentRuns
	} else {
		result, forecastErr := h.forecast(r.Context(), *settings)
		if forecastErr != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": forecastErr.Error()})
			return
		}
		var selected *Tier
		for i := range result.Tiers {
			if result.Tiers[i].ID == input.SelectedTier {
				selected = &result.Tiers[i]
				break
			}
		}
		if selected == nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported selected_tier"})
			return
		}
		if selected.Status != "available" || selected.EffectiveMaxConcurrent == nil {
			writeJSON(w, http.StatusConflict, map[string]string{"error": selected.Reason})
			return
		}
		settings.UsagePace.SelectedTier = selected.ID
		settings.UsagePace.TargetRunOutAt = selected.TargetRunOutAt.UTC().Format(time.RFC3339)
		settings.RunExecution.MaxConcurrentRuns = *selected.EffectiveMaxConcurrent
	}
	if err := h.settings.UpdateOrchestrationSettings(r.Context(), settings); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	result, err := h.forecast(r.Context(), *settings)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) forecast(ctx context.Context, settings config.OrchestrationSettings) (response, error) {
	now := h.now().UTC()
	if h.readings == nil {
		return response{}, errors.New("provider quota readings are unavailable")
	}
	spans, err := h.readings.QuotaReadingSpans(ctx, "", now.Add(-usageforecast.CurrentLookback))
	if err != nil {
		return response{}, err
	}
	forecasts := usageforecast.Current(spans, now)
	baseline := settings.UsagePace.BaseMaxConcurrentRuns
	if baseline < 1 {
		baseline = settings.RunExecution.MaxConcurrentRuns
	}
	tiers := Forecast(forecasts, baseline, now)
	var selectedTarget *time.Time
	if settings.UsagePace.TargetRunOutAt != "" {
		if parsed, parseErr := time.Parse(time.RFC3339, settings.UsagePace.TargetRunOutAt); parseErr == nil {
			parsed = parsed.UTC()
			selectedTarget = &parsed
		}
	}
	return response{
		GeneratedAt:            now,
		SelectedTier:           settings.UsagePace.SelectedTier,
		SelectedTargetRunOutAt: selectedTarget,
		BaselineMaxConcurrent:  baseline,
		EffectiveMaxConcurrent: settings.RunExecution.MaxConcurrentRuns,
		Tiers:                  tiers,
	}, nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
