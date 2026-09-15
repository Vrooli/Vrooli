package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"agent-manager/internal/invocationreadmodel"

	"github.com/gorilla/mux"
)

// EfficiencyHandler exposes one bounded, read-only view over the durable
// invocation projection. It deliberately contains no inference or policy
// decisions: callers use this report as evidence for a separate judgment.
type EfficiencyHandler struct {
	store invocationreadmodel.Store
}

func NewEfficiencyHandler(store invocationreadmodel.Store) *EfficiencyHandler {
	return &EfficiencyHandler{store: store}
}

type EfficiencyResponse struct {
	Status       string                         `json:"status"`
	Window       EfficiencyWindow               `json:"window"`
	Filter       EfficiencyFilter               `json:"filter"`
	Metrics      invocationreadmodel.RunMetrics `json:"metrics"`
	Durations    EfficiencyDurations            `json:"durations"`
	Breakdown    []EfficiencyBreakdown          `json:"breakdown,omitempty"`
	Tools        []EfficiencyTool               `json:"tools,omitempty"`
	Dispositions []EfficiencyDisposition        `json:"dispositions,omitempty"`
	Baseline     *EfficiencyBaseline            `json:"baseline,omitempty"`
	Freshness    EfficiencyFreshness            `json:"freshness"`
	Unknown      []string                       `json:"unknown,omitempty"`
	Evidence     EfficiencyEvidence             `json:"evidence"`
}

type EfficiencyFreshness struct {
	LatestProjectionAt string `json:"latest_projection_at,omitempty"`
	AgeMS              int64  `json:"age_ms,omitempty"`
	Available          bool   `json:"available"`
	Stale              bool   `json:"stale"`
	StaleReason        string `json:"stale_reason,omitempty"`
}

type EfficiencyDisposition struct {
	Disposition string `json:"disposition"`
	Count       int64  `json:"count"`
	Evidence    string `json:"evidence"`
}

type EfficiencyBaseline struct {
	Window    EfficiencyWindow               `json:"window"`
	Metrics   invocationreadmodel.RunMetrics `json:"metrics"`
	Durations EfficiencyDurations            `json:"durations"`
	Delta     EfficiencyDelta                `json:"delta"`
}

type EfficiencyDelta struct {
	TotalRuns         int64   `json:"total_runs"`
	TotalTokens       int64   `json:"total_tokens"`
	SuccessfulRuns    int64   `json:"successful_runs"`
	SuccessRate       float64 `json:"success_rate"`
	AverageDurationMS float64 `json:"average_duration_ms"`
}

type EfficiencyWindow struct {
	From string `json:"from,omitempty"`
	To   string `json:"to,omitempty"`
}

type EfficiencyFilter struct {
	TagPrefix string `json:"tag_prefix,omitempty"`
	GroupBy   string `json:"group_by"`
	Limit     int    `json:"limit"`
}

type EfficiencyEvidence struct {
	SourceProjection string `json:"source_projection"`
	InferenceCalls   int    `json:"inference_calls"`
	ReadOnly         bool   `json:"read_only"`
}

type EfficiencyDurations struct {
	AverageMS float64 `json:"average_ms"`
	P50MS     float64 `json:"p50_ms"`
	P95MS     float64 `json:"p95_ms"`
	P99MS     float64 `json:"p99_ms"`
	MinMS     int64   `json:"min_ms"`
	MaxMS     int64   `json:"max_ms"`
	Count     int64   `json:"count"`
}

type EfficiencyBreakdown struct {
	Key                                string  `json:"key"`
	Value                              string  `json:"value"`
	RunCount                           int64   `json:"run_count"`
	SuccessCount                       int64   `json:"success_count"`
	FailedCount                        int64   `json:"failed_count"`
	TotalCostUSD                       float64 `json:"total_cost_usd"`
	TotalTokens                        int64   `json:"total_tokens"`
	AverageDurationMS                  float64 `json:"average_duration_ms"`
	ConsumptionPerSuccessfulCompletion float64 `json:"consumption_per_successful_completion"`
	CompletionRate                     float64 `json:"completion_rate"`
}

type EfficiencyTool struct {
	ToolName            string  `json:"tool_name"`
	CallCount           int64   `json:"call_count"`
	SuccessCount        int64   `json:"success_count"`
	FailedCount         int64   `json:"failed_count"`
	TotalTokens         int64   `json:"total_tokens"`
	EstimatedTokenShare float64 `json:"estimated_token_share"`
}

func (h *EfficiencyHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/stats/efficiency", h.Get).Methods(http.MethodGet)
}

func (h *EfficiencyHandler) Get(w http.ResponseWriter, r *http.Request) {
	if h.store == nil {
		writeJSON(w, http.StatusOK, EfficiencyResponse{
			Status:   "unavailable",
			Unknown:  []string{"invocation_read_model_unavailable"},
			Evidence: EfficiencyEvidence{SourceProjection: "invocation_read_model_runs", ReadOnly: true},
		})
		return
	}

	filter, window, responseFilter, compare, err := efficiencyFilter(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	limit := responseFilter.Limit
	ctx := r.Context()
	var (
		metrics                                                       invocationreadmodel.RunMetrics
		durations                                                     invocationreadmodel.RunDurationStatistics
		statuses                                                      []invocationreadmodel.RunStatusCount
		breakdown                                                     []invocationreadmodel.RunBreakdownRow
		tools                                                         []invocationreadmodel.ToolUsageRow
		metricsErr, durationsErr, statusesErr, breakdownErr, toolsErr error
	)
	var wg sync.WaitGroup
	wg.Add(5)
	go func() { defer wg.Done(); metrics, metricsErr = h.store.RunMetrics(ctx, filter) }()
	go func() { defer wg.Done(); durations, durationsErr = h.store.RunDurationStatistics(ctx, filter) }()
	go func() { defer wg.Done(); statuses, statusesErr = h.store.RunStatusCounts(ctx, filter) }()
	go func() {
		defer wg.Done()
		breakdown, breakdownErr = h.store.RunBreakdown(ctx, filter, responseFilter.GroupBy, limit)
	}()
	go func() { defer wg.Done(); tools, toolsErr = h.store.ToolUsage(ctx, filter, limit) }()
	wg.Wait()

	unknown := make([]string, 0, 5)
	if metricsErr != nil {
		unknown = append(unknown, "metrics:"+metricsErr.Error())
	}
	if durationsErr != nil {
		unknown = append(unknown, "durations:"+durationsErr.Error())
	}
	if statusesErr != nil {
		unknown = append(unknown, "dispositions:"+statusesErr.Error())
	}
	if breakdownErr != nil {
		unknown = append(unknown, "breakdown:"+breakdownErr.Error())
	}
	if toolsErr != nil {
		unknown = append(unknown, "tools:"+toolsErr.Error())
	}
	var baseline *EfficiencyBaseline
	if compare {
		baselineFilter, baselineWindow := previousEfficiencyWindow(filter)
		baselineMetrics, baselineDurations, baselineUnknown := h.collectBaseline(ctx, baselineFilter)
		unknown = append(unknown, baselineUnknown...)
		baseline = &EfficiencyBaseline{Window: baselineWindow, Metrics: baselineMetrics, Durations: efficiencyDurationDTO(baselineDurations), Delta: efficiencyDelta(metrics, baselineMetrics, durations, baselineDurations)}
	}
	freshness := EfficiencyFreshness{}
	if provider, ok := h.store.(invocationreadmodel.FreshnessStore); ok {
		if projectedAt, freshnessErr := provider.LatestProjection(ctx); freshnessErr == nil && !projectedAt.IsZero() {
			freshness.Available = true
			freshness.LatestProjectionAt = projectedAt.UTC().Format(time.RFC3339)
			freshness.AgeMS = maxInt64(0, time.Since(projectedAt).Milliseconds())
			if filter.To != nil && projectedAt.Before(*filter.To) {
				freshness.Stale = true
				freshness.StaleReason = "projection_before_window_end"
				unknown = append(unknown, "projection_stale")
			}
		} else if freshnessErr != nil {
			unknown = append(unknown, "freshness:"+freshnessErr.Error())
		}
	}
	status := "ok"
	if len(unknown) > 0 {
		status = "partial"
	}

	writeJSON(w, http.StatusOK, EfficiencyResponse{
		Status: status, Window: window, Filter: responseFilter,
		Metrics: metrics, Durations: efficiencyDurationDTO(durations), Breakdown: efficiencyBreakdownDTO(breakdown), Tools: efficiencyToolsDTO(tools), Dispositions: efficiencyDispositionDTO(statuses), Baseline: baseline, Freshness: freshness,
		Unknown:  unknown,
		Evidence: EfficiencyEvidence{SourceProjection: "invocation_read_model_runs", InferenceCalls: 0, ReadOnly: true},
	})
}

func efficiencyFilter(r *http.Request) (invocationreadmodel.Filter, EfficiencyWindow, EfficiencyFilter, bool, error) {
	q := r.URL.Query()
	now := time.Now().UTC()
	from, to := now.Add(-24*time.Hour), now
	if preset := q.Get("preset"); preset != "" {
		hours := map[string]time.Duration{"6h": 6, "12h": 12, "24h": 24, "7d": 24 * 7, "30d": 24 * 30}[preset]
		if hours == 0 {
			return invocationreadmodel.Filter{}, EfficiencyWindow{}, EfficiencyFilter{}, false, invalidEfficiency("preset", "accepted values: 6h, 12h, 24h, 7d, 30d")
		}
		from = now.Add(-hours * time.Hour)
	}
	if value := q.Get("start"); value != "" {
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return invocationreadmodel.Filter{}, EfficiencyWindow{}, EfficiencyFilter{}, false, invalidEfficiency("start", "must be RFC3339")
		}
		from = parsed
	}
	if value := q.Get("end"); value != "" {
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return invocationreadmodel.Filter{}, EfficiencyWindow{}, EfficiencyFilter{}, false, invalidEfficiency("end", "must be RFC3339")
		}
		to = parsed
	}
	if !from.Before(to) {
		return invocationreadmodel.Filter{}, EfficiencyWindow{}, EfficiencyFilter{}, false, invalidEfficiency("window", "start must be before end")
	}
	groupBy := q.Get("group_by")
	if groupBy == "" {
		groupBy = "profile"
	}
	accepted := map[string]bool{"runner": true, "model": true, "profile": true, "workload": true, "workload_kind": true}
	if !accepted[groupBy] {
		return invocationreadmodel.Filter{}, EfficiencyWindow{}, EfficiencyFilter{}, false, invalidEfficiency("group_by", "accepted values: runner, model, profile, workload, workload_kind")
	}
	limit := 20
	if value := q.Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 100 {
			return invocationreadmodel.Filter{}, EfficiencyWindow{}, EfficiencyFilter{}, false, invalidEfficiency("limit", "must be between 1 and 100")
		}
		limit = parsed
	}
	tagPrefix := strings.TrimSpace(q.Get("tag_prefix"))
	compare := q.Get("compare") == "previous"
	if q.Get("compare") != "" && !compare {
		return invocationreadmodel.Filter{}, EfficiencyWindow{}, EfficiencyFilter{}, false, invalidEfficiency("compare", "accepted value: previous")
	}
	return invocationreadmodel.Filter{From: &from, To: &to, TagPrefix: tagPrefix, ExcludedWorkloadKinds: []string{"imported", "interactive"}}, EfficiencyWindow{From: from.Format(time.RFC3339), To: to.Format(time.RFC3339)}, EfficiencyFilter{TagPrefix: tagPrefix, GroupBy: groupBy, Limit: limit}, compare, nil
}

func previousEfficiencyWindow(filter invocationreadmodel.Filter) (invocationreadmodel.Filter, EfficiencyWindow) {
	from, to := filter.From.Add(-filter.To.Sub(*filter.From)), *filter.From
	return invocationreadmodel.Filter{From: &from, To: &to, TagPrefix: filter.TagPrefix, ExcludedWorkloadKinds: filter.ExcludedWorkloadKinds}, EfficiencyWindow{From: from.Format(time.RFC3339), To: to.Format(time.RFC3339)}
}

func (h *EfficiencyHandler) collectBaseline(ctx context.Context, filter invocationreadmodel.Filter) (invocationreadmodel.RunMetrics, invocationreadmodel.RunDurationStatistics, []string) {
	metrics, metricsErr := h.store.RunMetrics(ctx, filter)
	durations, durationsErr := h.store.RunDurationStatistics(ctx, filter)
	unknown := []string{}
	if metricsErr != nil {
		unknown = append(unknown, "baseline_metrics:"+metricsErr.Error())
	}
	if durationsErr != nil {
		unknown = append(unknown, "baseline_durations:"+durationsErr.Error())
	}
	return metrics, durations, unknown
}

func efficiencyDelta(current, baseline invocationreadmodel.RunMetrics, currentDuration, baselineDuration invocationreadmodel.RunDurationStatistics) EfficiencyDelta {
	return EfficiencyDelta{TotalRuns: current.TotalRuns - baseline.TotalRuns, TotalTokens: current.TotalTokens - baseline.TotalTokens, SuccessfulRuns: current.SuccessfulRuns - baseline.SuccessfulRuns, SuccessRate: current.SuccessRate - baseline.SuccessRate, AverageDurationMS: currentDuration.AverageDurationMS - baselineDuration.AverageDurationMS}
}

func efficiencyDispositionDTO(rows []invocationreadmodel.RunStatusCount) []EfficiencyDisposition {
	out := make([]EfficiencyDisposition, 0, len(rows))
	for _, row := range rows {
		out = append(out, EfficiencyDisposition{Disposition: row.Status, Count: row.Count, Evidence: "terminal_status_only; usefulness_not_inferred"})
	}
	return out
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

type efficiencyValidationError struct{ field, detail string }

func (e efficiencyValidationError) Error() string { return e.field + " " + e.detail }
func invalidEfficiency(field, detail string) error {
	return efficiencyValidationError{field: field, detail: detail}
}

func efficiencyDurationDTO(value invocationreadmodel.RunDurationStatistics) EfficiencyDurations {
	return EfficiencyDurations{AverageMS: value.AverageDurationMS, P50MS: value.P50DurationMS, P95MS: value.P95DurationMS, P99MS: value.P99DurationMS, MinMS: value.MinDurationMS, MaxMS: value.MaxDurationMS, Count: value.Count}
}

func efficiencyBreakdownDTO(rows []invocationreadmodel.RunBreakdownRow) []EfficiencyBreakdown {
	out := make([]EfficiencyBreakdown, 0, len(rows))
	for _, row := range rows {
		out = append(out, EfficiencyBreakdown{Key: row.Key, Value: row.Value, RunCount: row.RunCount, SuccessCount: row.SuccessCount, FailedCount: row.FailedCount, TotalCostUSD: row.TotalCostUSD, TotalTokens: row.TotalTokens, AverageDurationMS: row.AvgDurationMS, ConsumptionPerSuccessfulCompletion: row.ConsumptionPerSuccessfulCompletion, CompletionRate: row.CompletionRate})
	}
	return out
}

func efficiencyToolsDTO(rows []invocationreadmodel.ToolUsageRow) []EfficiencyTool {
	out := make([]EfficiencyTool, 0, len(rows))
	for _, row := range rows {
		out = append(out, EfficiencyTool{ToolName: row.ToolName, CallCount: row.CallCount, SuccessCount: row.SuccessCount, FailedCount: row.FailedCount, TotalTokens: row.TotalTokens, EstimatedTokenShare: row.EstimatedTokenShare})
	}
	return out
}
