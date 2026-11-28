package usagepace

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"agent-manager/internal/config"
	"agent-manager/internal/pricing"
	"agent-manager/internal/runtimeactivity"
	"agent-manager/internal/usageforecast"
	"agent-manager/internal/utilizationhistory"
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
	activity RuntimeActivityReader
	history  HistoryReader
	now      func() time.Time
}

type HistoryReader interface {
	Read(context.Context, time.Time, time.Time) ([]utilizationhistory.Sample, error)
}

type RuntimeActivityReader interface {
	Read(context.Context) runtimeactivity.Snapshot
}

func NewHandler(readings pricing.QuotaReadingSource, settings settingsService, activity RuntimeActivityReader, history ...HistoryReader) *Handler {
	h := &Handler{readings: readings, settings: settings, activity: activity, now: time.Now}
	if len(history) > 0 {
		h.history = history[0]
	}
	return h
}

func (h *Handler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc(APIPath, h.get).Methods(http.MethodGet)
	router.HandleFunc(APIPath, h.put).Methods(http.MethodPut)
}

type response struct {
	GeneratedAt            time.Time       `json:"generated_at"`
	SelectedTier           string          `json:"selected_tier,omitempty"`
	SelectedTargetRunOutAt *time.Time      `json:"selected_target_run_out_at,omitempty"`
	BaselineMaxConcurrent  int             `json:"baseline_max_concurrent_runs"`
	EffectiveMaxConcurrent int             `json:"effective_max_concurrent_runs"`
	Tiers                  []Tier          `json:"tiers"`
	UtilizationHistory     HistoryCoverage `json:"utilization_history"`
}

type HistoryCoverage struct {
	Samples            int                 `json:"sample_count"`
	Complete           int                 `json:"complete_sample_count"`
	PositivelyObserved int                 `json:"positively_observed_sample_count"`
	From               *time.Time          `json:"from,omitempty"`
	Through            *time.Time          `json:"through,omitempty"`
	ProviderRunIDs     map[string][]string `json:"latest_provider_run_ids,omitempty"`
	UnmappedRunIDs     []string            `json:"latest_unmapped_run_ids,omitempty"`
	Unknown            []string            `json:"unknown_reasons,omitempty"`
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
	activity := runtimeactivity.Snapshot{}
	if h.activity != nil {
		activity = h.activity.Read(ctx)
	}
	utilization := Utilization{ExecutingRuns: activity.ObservedExecutingRuns, ObservedAt: activity.ObservedAt, Complete: activity.Complete}
	baseline := settings.UsagePace.BaseMaxConcurrentRuns
	if baseline < 1 {
		baseline = settings.RunExecution.MaxConcurrentRuns
	}
	tiers := Forecast(forecasts, utilization, now)
	coverage := HistoryCoverage{ProviderRunIDs: map[string][]string{}, UnmappedRunIDs: []string{}, Unknown: []string{}}
	if h.history != nil {
		samples, historyErr := h.history.Read(ctx, now.Add(-usageforecast.CurrentLookback), now)
		if historyErr != nil {
			return response{}, historyErr
		}
		coverage = summarizeHistory(samples)
		quotaSource, ok := h.readings.(pricing.QuotaObservationRepository)
		if ok {
			quota, quotaErr := quotaSource.ListQuotaObservations(ctx, "", "", "", 30000)
			if quotaErr != nil {
				return response{}, quotaErr
			}
			quota = retainRecentQuota(quota, now.Add(-usageforecast.CurrentLookback))
			tiers = forecastWithHistory(forecasts, quota, samples, baseline, utilization, now)
		}
	}
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
		UtilizationHistory:     coverage,
	}, nil
}

func summarizeHistory(samples []utilizationhistory.Sample) HistoryCoverage {
	coverage := HistoryCoverage{Samples: len(samples), ProviderRunIDs: map[string][]string{}, UnmappedRunIDs: []string{}, Unknown: []string{}}
	if len(samples) == 0 {
		coverage.Unknown = append(coverage.Unknown, "no durable physical utilization samples have been recorded")
		return coverage
	}
	first, last := samples[0], samples[len(samples)-1]
	from, through := first.ObservedAt.UTC(), last.ObservedAt.UTC()
	coverage.From, coverage.Through = &from, &through
	for _, sample := range samples {
		if sample.Complete && len(sample.Unknown) == 0 {
			coverage.Complete++
		}
		if len(sample.UnmappedRunIDs) > 0 {
			coverage.PositivelyObserved++
			continue
		}
		for _, ids := range sample.ProviderRunIDs {
			if len(ids) > 0 {
				coverage.PositivelyObserved++
				break
			}
		}
	}
	for provider, ids := range last.ProviderRunIDs {
		coverage.ProviderRunIDs[provider] = append([]string(nil), ids...)
	}
	coverage.UnmappedRunIDs = append(coverage.UnmappedRunIDs, last.UnmappedRunIDs...)
	// The latest ten observations explain current collection quality without
	// returning a 70-day diagnostics archive on every pace read.
	start := 0
	if len(samples) > 10 {
		start = len(samples) - 10
	}
	seen := map[string]bool{}
	for _, sample := range samples[start:] {
		for _, reason := range sample.Unknown {
			if reason != "" && !seen[reason] && len(coverage.Unknown) < 20 {
				seen[reason] = true
				coverage.Unknown = append(coverage.Unknown, reason)
			}
		}
		if len(sample.UnmappedRunIDs) > 0 && !seen["physical runs include unmapped provider identities"] && len(coverage.Unknown) < 20 {
			seen["physical runs include unmapped provider identities"] = true
			coverage.Unknown = append(coverage.Unknown, "physical runs include unmapped provider identities")
		}
	}
	return coverage
}

func retainRecentQuota(input []pricing.QuotaObservation, since time.Time) []pricing.QuotaObservation {
	result := make([]pricing.QuotaObservation, 0, len(input))
	for _, observation := range input {
		if !observation.ObservedAt.Before(since) {
			result = append(result, observation)
		}
	}
	return result
}

func forecastWithHistory(forecasts []usageforecast.Forecast, quota []pricing.QuotaObservation, samples []utilizationhistory.Sample, baseline int, utilization Utilization, now time.Time) []Tier {
	var resetTarget time.Time
	for _, forecast := range forecasts {
		if forecast.Window != nil && forecast.Window.ResetAt.After(now) && forecast.Window.ResetAt.After(resetTarget) {
			resetTarget = forecast.Window.ResetAt
		}
	}
	targets := []utilizationhistory.Target{
		{ID: TierReset, At: resetTarget},
		{ID: TierUseByThursday, At: targetForDay(TierUseByThursday, now, usageforecast.DigestLocation())},
		{ID: TierUseByWednesday, At: targetForDay(TierUseByWednesday, now, usageforecast.DigestLocation())},
		{ID: TierExtraAllowance},
	}
	estimates := utilizationhistory.Estimate(utilizationhistory.EstimateInput{Now: now, Targets: targets, Quota: quota, Samples: samples})
	result := make([]Tier, 0, len(estimates))
	for _, estimate := range estimates {
		row := Tier{ID: estimate.ID, Label: tierLabel(estimate.ID), Status: estimate.Status, Reason: estimate.Reason, Basis: estimate.Basis, UtilizationComplete: utilization.Complete}
		for _, scope := range estimate.Unavailable {
			row.UnavailableScopes = append(row.UnavailableScopes, UnavailableScope{Provider: scope.Provider, Pool: scope.Pool, Window: scope.Window, AccountScope: scope.AccountScope, Reason: scope.Reason})
		}
		if !estimate.Target.IsZero() {
			target := estimate.Target.UTC()
			row.TargetRunOutAt = &target
			row.TargetRunOutSeconds = int64(target.Sub(now).Seconds())
		}
		if estimate.Slots != nil {
			slots := *estimate.Slots
			row.AvailableSlots = &slots
			row.EffectiveMaxConcurrent = &slots
			row.ScopeSlots = make([]ScopeSlot, 0, len(estimate.ProviderSlots))
			for _, provider := range estimate.ProviderSlots {
				scope := provider.Provider + "|" + provider.Pool + "|" + provider.Window + "|account:" + provider.AccountScope
				row.SourceScopes = append(row.SourceScopes, scope)
				resetAt := time.Time{}
				for _, forecast := range forecasts {
					if forecast.Window != nil && forecast.Window.Provider == provider.Provider && forecast.Window.Pool == provider.Pool && forecast.Window.Window == provider.Window && forecast.Window.AccountScope == provider.AccountScope {
						resetAt = forecast.Window.ResetAt
						break
					}
				}
				row.ScopeSlots = append(row.ScopeSlots, ScopeSlot{
					Scope: scope, Provider: provider.Provider, Pool: provider.Pool, Window: provider.Window, AccountScope: provider.AccountScope,
					AvailableSlots: provider.Slots, ProjectedRunOut: estimate.Target, ResetAt: resetAt,
					RunIDs: provider.RunIDs, QuotaStart: provider.QuotaStart, QuotaEnd: provider.QuotaEnd,
					SampleStart: provider.SampleStart, SampleEnd: provider.SampleEnd, SampleCount: provider.SampleCount,
					RunSeconds: provider.RunSeconds, BurnPercent: provider.BurnPercent, ScopeEvidence: provider.ScopeEvidence,
				})
				if row.SourceObservedAt == nil || provider.QuotaEnd.After(*row.SourceObservedAt) {
					observed := provider.QuotaEnd
					row.SourceObservedAt = &observed
				}
			}
		}
		if utilization.ExecutingRuns > 0 {
			observed := utilization.ExecutingRuns
			row.ObservedExecutingRuns = &observed
		}
		if !utilization.ObservedAt.IsZero() {
			observed := utilization.ObservedAt.UTC()
			row.UtilizationObservedAt = &observed
		}
		result = append(result, row)
	}
	return result
}

func tierLabel(id string) string {
	switch id {
	case TierReset:
		return "Last through the final active quota reset"
	case TierUseByThursday:
		return "Use by Thursday"
	case TierUseByWednesday:
		return "Use by Wednesday"
	case TierExtraAllowance:
		return "Use extra allowance"
	default:
		return id
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
