// Package measures exposes Agent Manager's durable friction analytics through
// one shared compute path for the measures-go registry and Connect RPCs.
package measures

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"agent-manager/internal/availability"
	"agent-manager/internal/conversationsearch"
	"agent-manager/internal/invocationreadmodel"
	"agent-manager/internal/outcomeledger"
	"agent-manager/internal/pricing"
	"agent-manager/internal/transcriptindex"
	"agent-manager/internal/usageattribution"
	"agent-manager/internal/usageforecast"
	"agent-manager/internal/usagemodes"

	"connectrpc.com/connect"
	measurelib "github.com/vrooli/measures-go"
	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
	measureconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures/measures_v1connect"
	sharedmeasurepb "github.com/vrooli/vrooli/packages/proto/gen/go/measures/v1"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	RunSuccessRate            = "throughput.run_success_rate"
	RunCycleTime              = "throughput.run_cycle_time"
	RunCost                   = "throughput.run_cost"
	RunVolume                 = "throughput.run_volume"
	RunStatusDistribution     = "throughput.run_status_distribution"
	RunnerBreakdown           = "throughput.runner_breakdown"
	ModelBreakdown            = "throughput.model_breakdown"
	ProfileBreakdown          = "throughput.profile_breakdown"
	WorkloadBreakdown         = "throughput.workload_breakdown"
	WorkloadEfficiency        = "throughput.workload_efficiency"
	TerminalRunTrend          = "throughput.terminal_run_trend"
	ToolUsage                 = "friction.tool_usage"
	TokenAttribution          = "friction.token_attribution"
	ToolFriction              = "friction.tool_friction"
	ToolProfiles              = "friction.tool_profiles"
	ErrorSignatures           = "friction.error_signatures"
	ConversationSearchQuality = "conversation_search.quality"
	FallbackInsights          = "health.fallback_insights"
)

// Store is the narrow durable analytical substrate. The database read-model
// repository satisfies it; no measure reads raw run_events JSON.
type Store interface {
	RunMetrics(context.Context, invocationreadmodel.Filter) (invocationreadmodel.RunMetrics, error)
	WindowedUsage(context.Context, invocationreadmodel.Filter) (invocationreadmodel.UsageTotals, error)
	RunDurationStatistics(context.Context, invocationreadmodel.Filter) (invocationreadmodel.RunDurationStatistics, error)
	RunStatusCounts(context.Context, invocationreadmodel.Filter) ([]invocationreadmodel.RunStatusCount, error)
	RunBreakdown(context.Context, invocationreadmodel.Filter, string, int) ([]invocationreadmodel.RunBreakdownRow, error)
	RunTimeSeries(context.Context, invocationreadmodel.Filter, time.Duration) ([]invocationreadmodel.RunTimeSeriesBucket, error)
	ToolUsage(context.Context, invocationreadmodel.Filter, int) ([]invocationreadmodel.ToolUsageRow, error)
	FallbackInsights(context.Context, invocationreadmodel.Filter) (invocationreadmodel.FallbackInsights, error)
}

// runCostDistributionStore is a store that can read the per-run token and
// cost distribution behind RunCost.
type runCostDistributionStore interface {
	RunCostDistribution(context.Context, invocationreadmodel.Filter) (invocationreadmodel.RunCostDistribution, error)
}

type tokenAttributionStore interface {
	TokenAttribution(context.Context, invocationreadmodel.Filter, string, string, int) ([]invocationreadmodel.TokenAttributionRow, error)
}

var (
	tokenAttributionGroupByValues = []string{"capability", "executable", "command_path", "target_scenario_operation"}
	tokenAttributionViewValues    = []string{"footprint", "residency", "incurred"}
)

func validateTokenAttributionSelection(groupBy, view string) error {
	if !containsString(tokenAttributionGroupByValues, groupBy) {
		return fmt.Errorf("invalid token attribution by %q; accepted values: %s", groupBy, strings.Join(tokenAttributionGroupByValues, ", "))
	}
	if !containsString(tokenAttributionViewValues, view) {
		return fmt.Errorf("invalid token attribution view %q; accepted values: %s", view, strings.Join(tokenAttributionViewValues, ", "))
	}
	return nil
}

func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

type metricKind string

const (
	runSuccessRate   metricKind = "run_success_rate"
	runCycleTime     metricKind = "run_cycle_time"
	runCost          metricKind = "run_cost"
	runVolume        metricKind = "run_volume"
	tokenAttribution metricKind = "token_attribution"
)

type metricResult struct {
	Rate       float64
	P50, P90   float64
	Numerator  int64
	Denom      int64
	Population int64
	Unknown    int64
	Secondary  int64
	Query      string
	Validity   Validity
	Filter     invocationreadmodel.Filter
}

// Definition is the backend-owned explanation of a measure. Keeping this
// beside the calculation prevents the UI, CLI, and registry from drifting
// apart about what a number means.
type Definition struct {
	ID          string
	Counts      string
	Numerator   string
	Denominator string
	SourceTable string
	Limitation  string
}

func definitionFor(name string) Definition {
	definitions := map[string]Definition{
		RunSuccessRate:                       {ID: RunSuccessRate, Counts: "executed terminal runs", Numerator: "executed terminal runs that completed", Denominator: "all executed terminal runs", SourceTable: "invocation_read_model_runs", Limitation: "completed is a lifecycle fact, not acceptance; imported and attached runs never count, whatever the scope. For acceptance read usage.outcomes"},
		RunCycleTime:                         {ID: RunCycleTime, Counts: "completed executed terminal runs", Numerator: "duration milliseconds", Denominator: "completed executed terminal runs", SourceTable: "invocation_read_model_runs", Limitation: "by default only runs Agent Manager executed count; choose a scope for operator sessions"},
		"throughput.run_duration_statistics": {ID: "throughput.run_duration_statistics", Counts: "completed executed terminal runs", Numerator: "duration percentiles and range", Denominator: "completed executed terminal runs", SourceTable: "invocation_read_model_runs", Limitation: "by default only runs Agent Manager executed count; choose a scope for operator sessions"},
		RunCost:                              {ID: RunCost, Counts: "transcript requests observed in the window", Numerator: "token totals; dollars for per-token billed runs only", Denominator: "runs with requests in the window", SourceTable: "transcript_usage", Limitation: "subscription work has no dollar cost; its spend is quota points. A request copied into several harness homes counts once"},
		RunVolume:                            {ID: RunVolume, Counts: "executed terminal run summaries", Numerator: "executed terminal runs", Denominator: "none", SourceTable: "invocation_read_model_runs", Limitation: "by default only runs Agent Manager executed count; choose a scope for operator sessions"},
		RunStatusDistribution:                {ID: RunStatusDistribution, Counts: "executed terminal runs", Numerator: "runs in each status", Denominator: "all executed terminal runs", SourceTable: "invocation_read_model_runs", Limitation: "by default only runs Agent Manager executed count; choose a scope for operator sessions"},
		RunnerBreakdown:                      {ID: RunnerBreakdown, Counts: "executed terminal runs", Numerator: "runs grouped by runner", Denominator: "all executed terminal runs", SourceTable: "invocation_read_model_runs", Limitation: "by default only runs Agent Manager executed count; choose a scope for operator sessions"},
		ModelBreakdown:                       {ID: ModelBreakdown, Counts: "executed terminal runs and transcript requests", Numerator: "runs and tokens grouped by model", Denominator: "all executed terminal runs", SourceTable: "invocation_read_model_runs, transcript_usage", Limitation: "run counts cover the selected scope; tokens count every request in the window unless a scope is chosen; model assignment is observational, not randomized"},
		ProfileBreakdown:                     {ID: ProfileBreakdown, Counts: "executed terminal runs", Numerator: "runs grouped by profile", Denominator: "all executed terminal runs", SourceTable: "invocation_read_model_runs", Limitation: "by default only runs Agent Manager executed count; choose a scope for operator sessions"},
		WorkloadBreakdown:                    {ID: WorkloadBreakdown, Counts: "executed terminal runs", Numerator: "workload totals and completion outcomes", Denominator: "all executed terminal runs", SourceTable: "invocation_read_model_runs", Limitation: "by default only runs Agent Manager executed count; choose a scope for operator sessions; model assignment is observational, not randomized"},
		WorkloadEfficiency:                   {ID: WorkloadEfficiency, Counts: "executed terminal runs for a workload", Numerator: "total tokens", Denominator: "successful completions", SourceTable: "invocation_read_model_runs", Limitation: "by default only runs Agent Manager executed count; choose a scope for operator sessions; model assignment is observational, not randomized"},
		TerminalRunTrend:                     {ID: TerminalRunTrend, Counts: "executed terminal runs", Numerator: "hourly terminal outcomes", Denominator: "none", SourceTable: "invocation_read_model_runs", Limitation: "by default only runs Agent Manager executed count; choose a scope for operator sessions"},
		ToolFriction:                         {ID: ToolFriction, Counts: "command calls in the window at the current fact version (code excluded)", Numerator: "failed calls, shape guesses, polled calls, bypasses, context tax and command time", Denominator: "calls whose outcome is known (failure rate); each finding's share is over its kind's base (Vrooli CLI failed calls, calls, shape guesses, polled calls or time; context tax; bypass calls)", SourceTable: "invocation_read_model_facts", Limitation: "failure rates cover calls with a known outcome; facts at an older fact version are not read and the coverage sentence counts them as awaiting re-derivation. Error text, notices and truncation are read from the first 3,000 characters of a result"},
		ToolProfiles:                         {ID: ToolProfiles, Counts: "command calls in the window grouped by command path", Numerator: "calls, failures, shape guesses, output weight, context tax, latency, polling per command path", Denominator: "the profile's calls", SourceTable: "invocation_read_model_facts", Limitation: "ranked by the profile's largest share of the window's failed calls, command time or context tax"},
		ErrorSignatures:                      {ID: ErrorSignatures, Counts: "failed and degraded Vrooli CLI calls", Numerator: "calls whose first meaningful error line normalises to one signature", Denominator: "none", SourceTable: "invocation_read_model_facts", Limitation: "ids, numbers, paths and quoted values are normalised away; a line seen across several CLIs that names no scenario is owned by the control plane"},
		ToolUsage:                            {ID: ToolUsage, Counts: "tool invocation facts", Numerator: "calls grouped by tool", Denominator: "none", SourceTable: "invocation_read_model_facts"},
		"friction.tool_command_breakdown":    {ID: "friction.tool_command_breakdown", Counts: "tool invocation facts", Numerator: "calls grouped by executable and command path", Denominator: "none", SourceTable: "invocation_read_model_facts", Limitation: "command detail is unavailable when the source fact did not record it"},
		TokenAttribution:                     {ID: TokenAttribution, Counts: "durable invocation token factors", Numerator: "tokens grouped by the selected dimension and view", Denominator: "none", SourceTable: "invocation_read_model_facts", Limitation: "estimated share identifies rankings that rely on payload estimates; target scenario and operation require a verified receipt join"},
		FallbackInsights:                     {ID: FallbackInsights, Counts: "runner and model fallback events and policy-candidate outcomes", Numerator: "attempts, exhausted walks and outcomes", Denominator: "none", SourceTable: "run_events", Limitation: "only events still retained in the window are counted"},
		ConversationSearchQuality:            {ID: ConversationSearchQuality, Counts: "privacy-safe conversation search request telemetry", Numerator: "no-result, weak-only, reformulated, selected, contributing-leg, degraded, error, and latency observations", Denominator: "search requests in the selected window", SourceTable: "conversation_search_telemetry", Limitation: "query and result content are never stored; at most the most recent 10000 requests in a window are aggregated"},
		"select_cohort":                      {ID: "select_cohort", Counts: "terminal runs matching the selected filter", Numerator: "matching runs", Denominator: "none", SourceTable: "invocation_read_model_runs"},
		"cohort_report":                      {ID: "cohort_report", Counts: "run reports of the selected cohort", Numerator: "friction signals, time accounting and goal progression", Denominator: "selected runs", SourceTable: "invocation_read_model_runs", Limitation: "at most 100 runs; each run report is built from its durable projection"},
		UsageForecast:                        {ID: UsageForecast, Counts: "provider quota readings in the current windows", Numerator: "used percent, trailing burn and projected exhaustion per window", Denominator: "none", SourceTable: "quota_observations", Limitation: "readings come only from captured sessions; a window with no reading for 6 hours is unknown"},
		UsageBurners:                         {ID: UsageBurners, Counts: "transcript usage reports (operator sessions and Agent Manager runs) in each current window", Numerator: "each consumer's share of the window's observed used-percent increase", Denominator: "the window's observed increase", SourceTable: "transcript_usage, quota_observations", Limitation: "points are estimates from a per-window calibration of API-price-weighted tokens; off-disk use is the unattributed remainder"},
		UsageRecommendations:                 {ID: UsageRecommendations, Counts: "owner-level actions from quota levers, tool friction findings, opportunities and gate-proven operator interventions", Numerator: "each action's effect in its own unit, scored in quota points a week by the stated conversions", Denominator: "none", SourceTable: "transcript_usage, invocation_read_model_runs, invocation_read_model_facts, operator_interventions, program-runtime library", Limitation: "deterministic rules; finished sessions say what they would have saved and only running ones are projected; calls and interventions are priced by stated assumptions; actions under the materiality floor are counted, not listed"},
		UsageFriction:                        {ID: UsageFriction, Counts: "owner-level tool findings from the tool profiles of the window, and operator messages the intervention classifier put in a category that passed the accuracy gate", Numerator: "each finding's cost in the unit of its harm (calls, command time, context tax); operator interventions per category and per agent-hour", Denominator: "the window's total of each kind of harm the finding holds (one named base per kind)", SourceTable: "invocation_read_model_facts, operator_interventions, transcript_usage, friction_problem_status", Limitation: "findings need three or more calls; an empty list is read beside the coverage sentence; no operator intervention category is shown until it passes the accuracy gate, and the section then states coverage and why instead of reading as no friction"},
		UsageCoverage:                        {ID: UsageCoverage, Counts: "transcript files on disk modified in the period", Numerator: "captured bytes and files", Denominator: "bytes and files on disk", SourceTable: "transcript_files", Limitation: "walks the configured roots at request time; a file still being written counts as partially indexed"},
		UsageCompare:                         {ID: UsageCompare, Counts: "runs of two selected cohorts", Numerator: "attributed quota points", Denominator: "work units with an accepted outcome estimate", SourceTable: "transcript_usage, quota_observations, run_ledger_facts", Limitation: "unreliable below five accepted units per side; unknown outcomes stay out; a completed run is not accepted; cohorts are observational, not randomized"},
		UsageParity:                          {ID: UsageParity, Counts: "runs active in the window, by start path and harness", Numerator: "runs or commands with each measure", Denominator: "runs or commands it applies to", SourceTable: "run_ledger_facts, transcript_files", Limitation: "window is by transcript activity, not import time; attached helper rows and duplicates stay out of denominators; the ledger lags changed runs by up to a few minutes"},
		UsageOutcomes:                        {ID: UsageOutcomes, Counts: "work units active in the window", Numerator: "attributed quota points of units with a known outcome", Denominator: "work units with an accepted outcome estimate", SourceTable: "run_ledger_facts, transcript_usage, quota_observations", Limitation: "outcomes are estimates with a stated source; observational unless randomized; unknown outcomes are excluded and reported; rows below 5 accepted outcomes read not enough data; weights behind break-even are usually priors"},
		UsageExperiment:                      {ID: UsageExperiment, Counts: "work units a Learn comparison assigned to an arm by chance", Numerator: "attributed quota points of units with a known outcome, per assigned arm", Denominator: "accepted outcomes per arm", SourceTable: "run_ledger_facts, transcript_usage, quota_observations, usage_mode_periods", Limitation: "analysis is by assigned arm (intention to treat), so sessions that ran at full power stay in their arm; unknown outcomes stay out and are counted; no verdict until each arm has the stated minimum of accepted outcomes; the permutation test uses a fixed seed; only large cost differences are detectable at this volume"},
		UsageInterventions:                   interventionsDefinition,
		UsageOperatorSignals:                 operatorSignalsDefinition,
		UsageOpportunities:                   opportunitiesDefinition,
		UsageForecastBacktest:                {ID: UsageForecastBacktest, Counts: "completed windows with an observed outcome", Numerator: "absolute error of the projected final percent at 25% and 50% of each window", Denominator: "scored windows", SourceTable: "quota_observations", Limitation: "windows whose newest reading is more than 6 hours before their end are not scored"},
	}
	return definitions[name]
}

func allDefinitions() []Definition {
	seen := make(map[string]struct{})
	definitions := make([]Definition, 0, 24)
	for _, name := range []string{RunSuccessRate, RunCycleTime, "throughput.run_duration_statistics", RunCost, RunVolume, RunStatusDistribution, RunnerBreakdown, ModelBreakdown, ProfileBreakdown, WorkloadBreakdown, WorkloadEfficiency, TerminalRunTrend, ToolUsage, TokenAttribution, "friction.tool_command_breakdown", ToolFriction, ToolProfiles, ErrorSignatures, ConversationSearchQuality, FallbackInsights, "select_cohort", "cohort_report", UsageForecast, UsageForecastBacktest, UsageBurners, UsageRecommendations, UsageFriction, UsageCoverage, UsageCompare, UsageParity, UsageOutcomes, UsageExperiment, UsageInterventions, UsageOperatorSignals} {
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		definitions = append(definitions, definitionFor(name))
	}
	return definitions
}

func provenanceFor(filter invocationreadmodel.Filter, source string, rows int64) *measurepb.MeasureProvenance {
	provenance := &measurepb.MeasureProvenance{SourceTable: source, RowCount: rows}
	if filter.From != nil {
		provenance.WindowStart = filter.From.UTC().Format(time.RFC3339Nano)
	}
	if filter.To != nil {
		provenance.WindowEnd = filter.To.UTC().Format(time.RFC3339Nano)
	}
	for _, item := range []struct{ field, value string }{
		{"ownership", filter.Ownership}, {"outcome", filter.Outcome}, {"executable", filter.Executable}, {"fingerprint", filter.Fingerprint}, {"profile_id", filter.ProfileID}, {"runner_type", filter.RunnerType}, {"model", filter.Model}, {"workload_kind", filter.WorkloadKind}, {"workload_key", filter.WorkloadKey}, {"tag_prefix", filter.TagPrefix}, {"run_status", filter.RunStatus}, {"tool_name", filter.ToolName}, {"target_scenario", filter.TargetScenario}, {"operation", filter.Operation},
	} {
		if item.value != "" {
			provenance.AppliedFilters = append(provenance.AppliedFilters, &measurepb.MeasureFilter{Field: item.field, Value: item.value})
		}
	}
	return provenance
}

// resolveWindow reads the request's window token, or the measure's declared
// default when the caller names none (the CLI and `measures get` send only the
// parameters the operator typed).
func (h *Handler) resolveWindow(request measurelib.MeasureRequest, fallback measurelib.TimeWindowToken) (measurelib.Range, error) {
	token := measurelib.TimeWindowToken(strings.TrimSpace(request.Params["window"]))
	if token == "" {
		token = fallback
	}
	return measurelib.ResolveToken(token, h.now(), time.UTC)
}

func (h *Handler) provenanceWithQuery(ctx context.Context, filter invocationreadmodel.Filter, source string, rows int64, query string) *measurepb.MeasureProvenance {
	provenance := provenanceFor(filter, source, rows)
	provenance.ExecutedQuery = query
	if provider, ok := h.store.(invocationreadmodel.FreshnessStore); ok {
		if projectedAt, err := provider.LatestProjection(ctx); err == nil && !projectedAt.IsZero() {
			provenance.ProjectionAt = projectedAt.UTC().Format(time.RFC3339)
			provenance.ProjectionAgeMs = h.now().UTC().Sub(projectedAt.UTC()).Milliseconds()
			if provenance.ProjectionAgeMs < 0 {
				provenance.ProjectionAgeMs = 0
			}
			if filter.To != nil && projectedAt.Before(*filter.To) {
				provenance.ProjectionStale = true
				provenance.ProjectionStaleReason = "projection_before_window_end"
			}
		}
	}
	return provenance
}

func definitionID(name string) string {
	definition := definitionFor(name)
	if definition.ID == "" {
		return name
	}
	return definition.ID
}

func declarations() []struct {
	name, method, intent, unit, summary string
	questions                           []string
	kind                                metricKind
} {
	return []struct {
		name, method, intent, unit, summary string
		questions                           []string
		kind                                metricKind
	}{
		{RunSuccessRate, "RunSuccessRate", "Share of executed terminal runs that completed. Completed is a lifecycle fact, not acceptance (see usage.outcomes); imported and attached runs never count.", "share", "{value} of runs completed ({window})", []string{"what share of agent runs complete this week", "how often do agent runs complete"}, runSuccessRate},
		{RunCycleTime, "RunCycleTime", "Average cycle time in milliseconds of completed runs Agent Manager executed.", "milliseconds", "{value} average run cycle time ({window})", []string{"what is the average agent run cycle time", "how long do completed agent runs take"}, runCycleTime},
		{RunCost, "RunCost", "Per-token billed cost in USD of the requests made in the window; subscription work is quota points, not dollars.", "usd", "{value} per-token billed cost ({window})", []string{"what did per-token billed runs cost this week", "show total agent run cost"}, runCost},
		{RunVolume, "RunVolume", "Number of executed terminal run summaries in the window.", "runs", "{value} agent run volume ({window})", []string{"how many agent runs completed this week", "show agent run volume"}, runVolume},
		{TokenAttribution, "TokenAttribution", "Durable token spend grouped by capability, executable, command path, or target scenario operation across footprint, residency, and incurred views.", "tokens", "{value} token attribution ({window})", []string{"which capabilities consume the most tokens", "which commands have the largest token footprint", "what incurred token spend follows each tool call"}, tokenAttribution},
	}
}

func declaration(spec struct {
	name, method, intent, unit, summary string
	questions                           []string
	kind                                metricKind
},
) measurelib.MeasureDeclaration {
	domain := "friction"
	if spec.kind == runSuccessRate || spec.kind == runCycleTime || spec.kind == runCost || spec.kind == runVolume {
		domain = "run"
	}
	params := map[string]measurelib.Param{"window": {Name: "window", Type: measurelib.ParamTypeTimeWindow, Default: string(measurelib.TokenThisWeek)}}
	result := measurelib.Result{Kind: measurelib.ResultScalar, ValueField: "value", Unit: spec.unit, SummaryTemplate: spec.summary}
	if spec.kind == tokenAttribution {
		params["group_by"] = measurelib.Param{Name: "group_by", Type: "string", Default: "capability", Description: "capability, executable, command_path, or target_scenario_operation"}
		params["view"] = measurelib.Param{Name: "view", Type: "string", Default: "footprint", Description: "footprint, residency, or incurred"}
		result = measurelib.Result{Kind: measurelib.ResultTable, ValueField: "value", Unit: "tokens", SummaryTemplate: spec.summary}
	}
	return measurelib.MeasureDeclaration{
		Name: spec.name, Scenario: "agent-manager", Domain: domain, Intent: spec.intent, Questions: spec.questions,
		Params: params,
		Result: result,
		Effect: measurelib.EffectRead, RunEligible: true, Service: "MeasuresService", Method: spec.method,
	}
}

func execute(ctx context.Context, store Store, kind metricKind, filter invocationreadmodel.Filter) (metricResult, error) {
	if kind == runCost {
		usage, err := store.WindowedUsage(ctx, filter)
		if err != nil {
			return metricResult{}, err
		}
		return metricResult{Rate: usage.MeteredCostUSD, Numerator: usage.Requests, Denom: usage.Runs, Query: usageQuery(filter)}, nil
	}
	runs, err := store.RunMetrics(ctx, filter)
	if err != nil {
		return metricResult{}, err
	}
	result := metricResult{Query: fmt.Sprintf("SELECT durable terminal aggregate FROM invocation_read_model_runs WHERE occurred_at >= %q AND occurred_at < %q", filter.From.UTC().Format(time.RFC3339Nano), filter.To.UTC().Format(time.RFC3339Nano))}
	switch kind {
	case runSuccessRate:
		result.Rate, result.Numerator, result.Denom = runs.SuccessRate, runs.SuccessfulRuns, runs.TerminalRuns
	case runCycleTime:
		result.Rate, result.P50, result.P90 = runs.AverageDurationMS, runs.P50DurationMS, runs.P90DurationMS
		result.Numerator, result.Denom = runs.CompletedDurationRuns, runs.CompletedDurationRuns
	case runVolume:
		result.Rate, result.Numerator, result.Denom = float64(runs.TotalRuns), runs.TotalRuns, runs.TerminalRuns
	default:
		return metricResult{}, fmt.Errorf("unknown measure %q", kind)
	}
	return result, nil
}

func filterFromProto(input *measurepb.InvocationFilter, now time.Time) (invocationreadmodel.Filter, error) {
	if input == nil {
		input = &measurepb.InvocationFilter{}
	}
	filter := invocationreadmodel.Filter{Ownership: input.GetOwnership(), Outcome: input.GetOutcome(), Executable: input.GetExecutable(), Fingerprint: input.GetFingerprint(), ProfileID: input.GetProfileId(), RunnerType: input.GetRunnerType(), Model: input.GetModel(), TagPrefix: input.GetTagPrefix(), RunStatus: input.GetRunStatus(), ToolName: input.GetToolName(), WorkloadKey: input.GetWorkloadKey(), TargetScenario: input.GetTargetScenario(), Operation: input.GetOperation()}
	filter.Scope = scopeFromProto(input.GetScope())
	window := input.GetWindow()
	if window == nil {
		window = &sharedmeasurepb.TimeWindow{Window: &sharedmeasurepb.TimeWindow_Token{Token: sharedmeasurepb.TimeWindowToken_TIME_WINDOW_TOKEN_THIS_WEEK}}
	}
	rangeValue, err := measurelib.ResolveTimeWindow(window, now, time.UTC)
	if err != nil {
		return filter, err
	}
	filter.From, filter.To = &rangeValue.From, &rangeValue.To
	return filter, nil
}

func scopeFromProto(scope measurepb.RunScope) invocationreadmodel.RunScope {
	switch scope {
	case measurepb.RunScope_RUN_SCOPE_AGENT_MANAGER:
		return invocationreadmodel.ScopeAgentManager
	case measurepb.RunScope_RUN_SCOPE_OPERATOR_SESSIONS:
		return invocationreadmodel.ScopeOperatorSessions
	case measurepb.RunScope_RUN_SCOPE_ALL:
		return invocationreadmodel.ScopeAll
	}
	return invocationreadmodel.ScopeUnspecified
}

// usageQuery is the provenance text of a windowed token total.
func usageQuery(filter invocationreadmodel.Filter) string {
	return fmt.Sprintf("SELECT SUM(input_tokens + output_tokens + cache_read_tokens + cache_creation_tokens) FROM transcript_usage WHERE observed_at >= %q AND observed_at < %q", filter.From.UTC().Format(time.RFC3339Nano), filter.To.UTC().Format(time.RFC3339Nano))
}

func filterWithWindow(input *measurepb.InvocationFilter, window *sharedmeasurepb.TimeWindow, now time.Time) (invocationreadmodel.Filter, error) {
	filter, err := filterFromProto(input, now)
	if err != nil || window == nil {
		return filter, err
	}
	rangeValue, err := measurelib.ResolveTimeWindow(window, now, time.UTC)
	if err != nil {
		return filter, err
	}
	filter.From, filter.To = &rangeValue.From, &rangeValue.To
	return filter, nil
}

// Handler is both the typed Connect surface and the owner of the shared
// compute functions used by the measures-go registry.
type Handler struct {
	store               Store
	now                 func() time.Time
	cohorts             CohortSource
	validityConfig      ValidityConfig
	conversationSearch  *conversationsearch.Service
	conversationIndexer *conversationsearch.Indexer
	quotaReadings       pricing.QuotaReadingSource
	usageActivity       usageforecast.ActivitySource
	usageStore          usageattribution.Store
	frictionStatuses    usageattribution.StatusStore
	usagePricer         *usageattribution.PriceBook
	coverage            transcriptindex.CoverageReader
	interventions       InterventionSource
	signals             SignalSource
	programs            ProgramLibrarySource
	tools               usageattribution.ToolCallStore
	usageModes          *usagemodes.Service
	ledger              outcomeledger.Reader
	runRebaser          RunRebaser
	roleProposer        RoleProposer
	signalControl       SignalControl
}

func (h *Handler) SetConversationSearchQuality(service *conversationsearch.Service, indexer *conversationsearch.Indexer) {
	h.conversationSearch = service
	h.conversationIndexer = indexer
}

func NewHandler(store Store, now func() time.Time) *Handler {
	if now == nil {
		now = time.Now
	}
	return &Handler{store: store, now: now, validityConfig: DefaultValidityConfig()}
}

// SetValidityConfig lets composition select the product threshold while tests
// can exercise both healthy and degenerate corpora deterministically.
func (h *Handler) SetValidityConfig(config ValidityConfig) { h.validityConfig = config.normalized() }

func (h *Handler) AllMeasureDefinitions(_ context.Context, _ *connect.Request[measurepb.AllMeasureDefinitionsRequest]) (*connect.Response[measurepb.AllMeasureDefinitionsResponse], error) {
	definitions := allDefinitions()
	response := &measurepb.AllMeasureDefinitionsResponse{Definitions: make([]*measurepb.MeasureDefinition, 0, len(definitions))}
	for _, definition := range definitions {
		response.Definitions = append(response.Definitions, &measurepb.MeasureDefinition{Id: definition.ID, Counts: definition.Counts, Numerator: definition.Numerator, Denominator: definition.Denominator, SourceTable: definition.SourceTable, Limitation: definition.Limitation})
	}
	return connect.NewResponse(response), nil
}

func (h *Handler) metric(ctx context.Context, kind metricKind, input *measurepb.InvocationFilter, window *sharedmeasurepb.TimeWindow) (metricResult, error) {
	filter, err := filterWithWindow(input, window, h.now())
	if err != nil {
		return metricResult{}, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return h.metricFor(ctx, kind, filter)
}

// metricFor computes one measure with its validity for a resolved filter. The
// typed RPCs and the registry share it, so both report the same reliability.
func (h *Handler) metricFor(ctx context.Context, kind metricKind, filter invocationreadmodel.Filter) (metricResult, error) {
	result, err := execute(ctx, h.store, kind, filter)
	if err != nil {
		return metricResult{}, connect.NewError(connect.CodeInternal, err)
	}
	result.Validity = assessValidity(result.Denom, 0, h.validityConfig)
	result.Filter = filter
	return result, nil
}

func protoValidity(validity Validity) *measurepb.MeasureValidity {
	return &measurepb.MeasureValidity{State: string(validity.State), Reason: validity.Reason, SampleSize: validity.SampleSize, LargestFingerprintBucket: validity.LargestFingerprintBucket, LargestFingerprintShare: validity.LargestFingerprintShare, ClassifiedBase: validity.ClassifiedBase, UnclassifiedCount: validity.UnclassifiedCount, UnclassifiedShare: validity.UnclassifiedShare, MinimumClassifiedShare: validity.MinimumClassifiedShare}
}

func (h *Handler) metricProvenance(ctx context.Context, name string, result metricResult) *measurepb.MeasureProvenance {
	return h.provenanceWithQuery(ctx, result.Filter, definitionFor(name).SourceTable, result.Validity.SampleSize, result.Query)
}

func (h *Handler) validityForSample(sample int64) *measurepb.MeasureValidity {
	return protoValidity(assessValidity(sample, 0, h.validityConfig))
}

func (h *Handler) RunSuccessRate(ctx context.Context, req *connect.Request[measurepb.RunSuccessRateRequest]) (*connect.Response[measurepb.RunSuccessRateResponse], error) {
	r, err := h.metric(ctx, runSuccessRate, req.Msg.GetFilter(), req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&measurepb.RunSuccessRateResponse{Rate: r.Rate, SuccessfulRuns: r.Numerator, TerminalRuns: r.Denom, Validity: protoValidity(r.Validity), Provenance: h.metricProvenance(ctx, RunSuccessRate, r), DefinitionId: definitionID(RunSuccessRate)}), nil
}

func (h *Handler) RunCycleTime(ctx context.Context, req *connect.Request[measurepb.RunCycleTimeRequest]) (*connect.Response[measurepb.RunCycleTimeResponse], error) {
	r, err := h.metric(ctx, runCycleTime, req.Msg.GetFilter(), req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&measurepb.RunCycleTimeResponse{AverageDurationMs: r.Rate, P50DurationMs: r.P50, P90DurationMs: r.P90, CompletedDurationRuns: r.Numerator, Validity: protoValidity(r.Validity), Provenance: h.metricProvenance(ctx, RunCycleTime, r), DefinitionId: definitionID(RunCycleTime)}), nil
}

func (h *Handler) RunCost(ctx context.Context, req *connect.Request[measurepb.RunCostRequest]) (*connect.Response[measurepb.RunCostResponse], error) {
	if err := validateSubscriptionAllocation(req.Msg.GetAllocateSubscription(), req.Msg.GetAllocationBasis()); err != nil {
		return nil, err
	}
	filter, err := filterWithWindow(req.Msg.GetFilter(), req.Msg.GetWindow(), h.now())
	if err != nil {
		return nil, err
	}
	usage, err := h.store.WindowedUsage(ctx, filter)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &measurepb.RunCostResponse{TotalCostUsd: usage.MeteredCostUSD, TotalRuns: usage.Runs, Requests: usage.Requests, TotalTokens: usage.TotalTokens, InputTokens: usage.InputTokens, OutputTokens: usage.OutputTokens, CacheReadTokens: usage.CacheReadTokens, CacheCreationTokens: usage.CacheCreationTokens, InputCostUsd: usage.InputCostUSD, OutputCostUsd: usage.OutputCostUSD, CacheReadCostUsd: usage.CacheReadCostUSD, CacheCreationCostUsd: usage.CacheCreationCostUSD, TotalChargeMicroUsd: usage.MeteredChargeMicroUSD, Validity: h.validityForSample(usage.Runs), Provenance: h.provenanceWithQuery(ctx, filter, definitionFor(RunCost).SourceTable, usage.Runs, usageQuery(filter)), DefinitionId: definitionID(RunCost)}
	if usage.Runs > 0 {
		response.AverageCostUsd = usage.MeteredCostUSD / float64(usage.Runs)
	}
	if distStore, ok := h.store.(runCostDistributionStore); ok {
		dist, distErr := distStore.RunCostDistribution(ctx, filter)
		if distErr != nil {
			return nil, connect.NewError(connect.CodeInternal, distErr)
		}
		response.P50Tokens = dist.P50Tokens
		response.P90Tokens = dist.P90Tokens
		response.P95Tokens = dist.P95Tokens
		response.P99Tokens = dist.P99Tokens
		response.MaxTokens = dist.MaxTokens
		response.P50CostUsd = dist.P50CostUSD
		response.P90CostUsd = dist.P90CostUSD
		response.P95CostUsd = dist.P95CostUSD
		response.P99CostUsd = dist.P99CostUSD
		response.MaxCostUsd = dist.MaxCostUSD
		response.DistributionSampleSize = dist.SampleSize
		for _, bucket := range dist.TokenBuckets {
			response.TokenBuckets = append(response.TokenBuckets, &measurepb.RunTokenBucket{Label: bucket.Label, MinTokens: bucket.MinTokens, MaxTokens: bucket.MaxTokens, RunCount: bucket.RunCount})
		}
	}
	return connect.NewResponse(response), nil
}

func validateSubscriptionAllocation(allocate bool, basis string) error {
	if !allocate {
		return nil
	}
	if strings.TrimSpace(basis) == "" {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("subscription allocation basis is required when allocation is enabled"))
	}
	return nil
}

func (h *Handler) RunVolume(ctx context.Context, req *connect.Request[measurepb.RunVolumeRequest]) (*connect.Response[measurepb.RunVolumeResponse], error) {
	r, err := h.metric(ctx, runVolume, req.Msg.GetFilter(), req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	response := &measurepb.RunVolumeResponse{TotalRuns: r.Numerator, TerminalRuns: r.Denom, Validity: protoValidity(r.Validity), Provenance: h.metricProvenance(ctx, RunVolume, r), DefinitionId: definitionID(RunVolume)}
	if coverageStore, ok := h.store.(interface {
		HistoryCoverage(context.Context) (time.Time, int64, error)
	}); ok {
		floor, outside, coverageErr := coverageStore.HistoryCoverage(ctx)
		if coverageErr != nil {
			return nil, connect.NewError(connect.CodeInternal, coverageErr)
		}
		if !floor.IsZero() {
			response.HistoryFloor = floor.UTC().Format(time.RFC3339)
		}
		response.OutsideHistoryRunCount = outside
	}
	return connect.NewResponse(response), nil
}

func (h *Handler) runStatusRows(ctx context.Context, input *measurepb.InvocationFilter, window *sharedmeasurepb.TimeWindow) ([]invocationreadmodel.RunStatusCount, string, invocationreadmodel.Filter, error) {
	filter, err := filterWithWindow(input, window, h.now())
	if err != nil {
		return nil, "", invocationreadmodel.Filter{}, connect.NewError(connect.CodeInvalidArgument, err)
	}
	rows, err := h.store.RunStatusCounts(ctx, filter)
	if err != nil {
		return nil, "", invocationreadmodel.Filter{}, connect.NewError(connect.CodeInternal, err)
	}
	return rows, fmt.Sprintf("SELECT status, COUNT(*) FROM invocation_read_model_runs WHERE occurred_at >= %q AND occurred_at < %q GROUP BY status", filter.From.UTC().Format(time.RFC3339Nano), filter.To.UTC().Format(time.RFC3339Nano)), filter, nil
}

func (h *Handler) RunStatusDistribution(ctx context.Context, req *connect.Request[measurepb.RunStatusDistributionRequest]) (*connect.Response[measurepb.RunStatusDistributionResponse], error) {
	rows, query, filter, err := h.runStatusRows(ctx, req.Msg.GetFilter(), req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	response := &measurepb.RunStatusDistributionResponse{Rows: make([]*measurepb.RunStatusCount, 0, len(rows))}
	for _, row := range rows {
		response.Rows = append(response.Rows, &measurepb.RunStatusCount{Status: row.Status, Count: row.Count})
	}
	var sample int64
	for _, row := range rows {
		sample += row.Count
	}
	response.Validity = h.validityForSample(sample)
	response.Provenance = h.provenanceWithQuery(ctx, filter, definitionFor(RunStatusDistribution).SourceTable, sample, query)
	response.DefinitionId = definitionID(RunStatusDistribution)
	return connect.NewResponse(response), nil
}

func (h *Handler) RunDurationStatistics(ctx context.Context, req *connect.Request[measurepb.RunDurationStatisticsRequest]) (*connect.Response[measurepb.RunDurationStatisticsResponse], error) {
	filter, err := filterWithWindow(req.Msg.GetFilter(), req.Msg.GetWindow(), h.now())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	stats, err := h.store.RunDurationStatistics(ctx, filter)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	query := fmt.Sprintf("SELECT durable duration summary FROM invocation_read_model_runs WHERE occurred_at >= %q AND occurred_at < %q", filter.From.UTC().Format(time.RFC3339Nano), filter.To.UTC().Format(time.RFC3339Nano))
	return connect.NewResponse(&measurepb.RunDurationStatisticsResponse{AverageDurationMs: stats.AverageDurationMS, P50DurationMs: stats.P50DurationMS, P90DurationMs: stats.P90DurationMS, P95DurationMs: stats.P95DurationMS, P99DurationMs: stats.P99DurationMS, MinDurationMs: stats.MinDurationMS, MaxDurationMs: stats.MaxDurationMS, Count: stats.Count, Validity: h.validityForSample(stats.Count), Provenance: h.provenanceWithQuery(ctx, filter, definitionFor("throughput.run_duration_statistics").SourceTable, stats.Count, query), DefinitionId: definitionID("throughput.run_duration_statistics")}), nil
}

func (h *Handler) runBreakdownRows(ctx context.Context, input *measurepb.InvocationFilter, window *sharedmeasurepb.TimeWindow, dimension string) ([]invocationreadmodel.RunBreakdownRow, string, invocationreadmodel.Filter, error) {
	filter, err := filterWithWindow(input, window, h.now())
	if err != nil {
		return nil, "", invocationreadmodel.Filter{}, connect.NewError(connect.CodeInvalidArgument, err)
	}
	rows, err := h.store.RunBreakdown(ctx, filter, dimension, 20)
	if err != nil {
		return nil, "", invocationreadmodel.Filter{}, connect.NewError(connect.CodeInternal, err)
	}
	return rows, fmt.Sprintf("SELECT %s, terminal run aggregates FROM invocation_read_model_runs WHERE occurred_at >= %q AND occurred_at < %q GROUP BY %s LIMIT 20", dimension, filter.From.UTC().Format(time.RFC3339Nano), filter.To.UTC().Format(time.RFC3339Nano), dimension), filter, nil
}

func protoBreakdownRows(rows []invocationreadmodel.RunBreakdownRow) []*measurepb.RunBreakdownRow {
	out := make([]*measurepb.RunBreakdownRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, &measurepb.RunBreakdownRow{Key: row.Key, Value: row.Value, RunCount: row.RunCount, SuccessCount: row.SuccessCount, FailedCount: row.FailedCount, TotalCostUsd: row.TotalCostUSD, TotalTokens: row.TotalTokens, InputTokens: row.InputTokens, OutputTokens: row.OutputTokens, CacheReadTokens: row.CacheReadTokens, CacheCreationTokens: row.CacheCreationTokens, AverageDurationMs: row.AvgDurationMS, TotalChargeMicroUsd: row.TotalChargeMicroUSD, ConsumptionPerSuccessfulCompletion: row.ConsumptionPerSuccessfulCompletion, CompletionRate: row.CompletionRate})
	}
	return out
}

func sumBreakdownRuns(rows []invocationreadmodel.RunBreakdownRow) (total int64) {
	for _, row := range rows {
		total += row.RunCount
	}
	return total
}

func (h *Handler) RunnerBreakdown(ctx context.Context, req *connect.Request[measurepb.RunnerBreakdownRequest]) (*connect.Response[measurepb.RunnerBreakdownResponse], error) {
	rows, query, filter, err := h.runBreakdownRows(ctx, req.Msg.GetFilter(), req.Msg.GetWindow(), "runner")
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&measurepb.RunnerBreakdownResponse{Rows: protoBreakdownRows(rows), Validity: h.validityForSample(sumBreakdownRuns(rows)), Provenance: h.provenanceWithQuery(ctx, filter, definitionFor(RunnerBreakdown).SourceTable, sumBreakdownRuns(rows), query), DefinitionId: definitionID(RunnerBreakdown)}), nil
}

func (h *Handler) ModelBreakdown(ctx context.Context, req *connect.Request[measurepb.ModelBreakdownRequest]) (*connect.Response[measurepb.ModelBreakdownResponse], error) {
	rows, query, filter, err := h.runBreakdownRows(ctx, req.Msg.GetFilter(), req.Msg.GetWindow(), "model")
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&measurepb.ModelBreakdownResponse{Rows: protoBreakdownRows(rows), Validity: h.validityForSample(sumBreakdownRuns(rows)), Provenance: h.provenanceWithQuery(ctx, filter, definitionFor(ModelBreakdown).SourceTable, sumBreakdownRuns(rows), query), DefinitionId: definitionID(ModelBreakdown)}), nil
}

func (h *Handler) ProfileBreakdown(ctx context.Context, req *connect.Request[measurepb.ProfileBreakdownRequest]) (*connect.Response[measurepb.ProfileBreakdownResponse], error) {
	rows, query, filter, err := h.runBreakdownRows(ctx, req.Msg.GetFilter(), req.Msg.GetWindow(), "profile")
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&measurepb.ProfileBreakdownResponse{Rows: protoBreakdownRows(rows), Validity: h.validityForSample(sumBreakdownRuns(rows)), Provenance: h.provenanceWithQuery(ctx, filter, definitionFor(ProfileBreakdown).SourceTable, sumBreakdownRuns(rows), query), DefinitionId: definitionID(ProfileBreakdown)}), nil
}

func (h *Handler) WorkloadBreakdown(ctx context.Context, req *connect.Request[measurepb.WorkloadBreakdownRequest]) (*connect.Response[measurepb.WorkloadBreakdownResponse], error) {
	if err := validateSubscriptionAllocation(req.Msg.GetAllocateSubscription(), req.Msg.GetAllocationBasis()); err != nil {
		return nil, err
	}
	rows, query, filter, err := h.runBreakdownRows(ctx, req.Msg.GetFilter(), req.Msg.GetWindow(), "workload")
	if err != nil {
		return nil, err
	}
	sample := sumBreakdownRuns(rows)
	return connect.NewResponse(&measurepb.WorkloadBreakdownResponse{Rows: protoBreakdownRows(rows), Validity: h.validityForSample(sample), Provenance: h.provenanceWithQuery(ctx, filter, definitionFor(WorkloadBreakdown).SourceTable, sample, query), DefinitionId: definitionID(WorkloadBreakdown)}), nil
}

func (h *Handler) WorkloadEfficiency(ctx context.Context, req *connect.Request[measurepb.WorkloadEfficiencyRequest]) (*connect.Response[measurepb.WorkloadEfficiencyResponse], error) {
	if err := validateSubscriptionAllocation(req.Msg.GetAllocateSubscription(), req.Msg.GetAllocationBasis()); err != nil {
		return nil, err
	}
	filter, err := filterWithWindow(req.Msg.GetFilter(), req.Msg.GetWindow(), h.now())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	runs, err := h.store.RunMetrics(ctx, filter)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	usage, err := h.store.WindowedUsage(ctx, filter)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	validity := assessValidity(runs.TerminalRuns, 0, h.validityConfig)
	query := fmt.Sprintf("SELECT workload efficiency from invocation_read_model_runs WHERE occurred_at >= %q AND occurred_at < %q", filter.From.UTC().Format(time.RFC3339Nano), filter.To.UTC().Format(time.RFC3339Nano))
	response := &measurepb.WorkloadEfficiencyResponse{
		ConsumptionPerSuccessfulCompletion: perSuccess(usage.TotalTokens, runs.SuccessfulRuns),
		CompletionRate:                     runs.SuccessRate,
		TotalTokens:                        usage.TotalTokens,
		TerminalRuns:                       runs.TerminalRuns,
		SuccessfulRuns:                     runs.SuccessfulRuns,
		ObservationalLimitation:            "model assignment was observational, not randomized",
		Validity:                           protoValidity(validity),
		Provenance:                         h.provenanceWithQuery(ctx, filter, definitionFor(WorkloadEfficiency).SourceTable, runs.TerminalRuns, query),
		DefinitionId:                       definitionID(WorkloadEfficiency),
	}
	return connect.NewResponse(response), nil
}

func perSuccess(tokens, successes int64) float64 {
	if successes == 0 {
		return 0
	}
	return float64(tokens) / float64(successes)
}

func (h *Handler) TerminalRunTrend(ctx context.Context, req *connect.Request[measurepb.TerminalRunTrendRequest]) (*connect.Response[measurepb.TerminalRunTrendResponse], error) {
	filter, err := filterWithWindow(req.Msg.GetFilter(), req.Msg.GetWindow(), h.now())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	rows, err := h.store.RunTimeSeries(ctx, filter, time.Hour)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	query := fmt.Sprintf("SELECT terminal outcome aggregates FROM invocation_read_model_runs WHERE occurred_at >= %q AND occurred_at < %q GROUP BY hourly terminal bucket", filter.From.UTC().Format(time.RFC3339Nano), filter.To.UTC().Format(time.RFC3339Nano))
	response := &measurepb.TerminalRunTrendResponse{Rows: make([]*measurepb.TerminalRunTrendRow, 0, len(rows))}
	for _, row := range rows {
		response.Rows = append(response.Rows, &measurepb.TerminalRunTrendRow{Bucket: row.Bucket.UTC().Format(time.RFC3339), TerminalRuns: row.TerminalRuns, CompletedRuns: row.CompletedRuns, FailedRuns: row.FailedRuns, CancelledRuns: row.CancelledRuns, AverageDurationMs: row.AvgDurationMS})
	}
	var sample int64
	for _, row := range rows {
		sample += row.TerminalRuns
	}
	response.Validity = h.validityForSample(sample)
	response.Provenance = h.provenanceWithQuery(ctx, filter, definitionFor(TerminalRunTrend).SourceTable, sample, query)
	response.DefinitionId = definitionID(TerminalRunTrend)
	return connect.NewResponse(response), nil
}

func (h *Handler) ToolUsage(ctx context.Context, req *connect.Request[measurepb.ToolUsageRequest]) (*connect.Response[measurepb.ToolUsageResponse], error) {
	filter, err := filterWithWindow(req.Msg.GetFilter(), req.Msg.GetWindow(), h.now())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	rows, err := h.store.ToolUsage(ctx, filter, 20)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	query := fmt.Sprintf("SELECT tool_name, classified outcomes FROM invocation_read_model_facts WHERE occurred_at >= %q AND occurred_at < %q GROUP BY tool_name LIMIT 20", filter.From.UTC().Format(time.RFC3339Nano), filter.To.UTC().Format(time.RFC3339Nano))
	response := &measurepb.ToolUsageResponse{Rows: make([]*measurepb.ToolUsageRow, 0, len(rows))}
	for _, row := range rows {
		response.Rows = append(response.Rows, &measurepb.ToolUsageRow{ToolName: row.ToolName, CallCount: row.CallCount, SuccessCount: row.SuccessCount, FailedCount: row.FailedCount, TotalTokens: row.TotalTokens, EstimatedTokenShare: row.EstimatedTokenShare})
	}
	var sample int64
	for _, row := range rows {
		sample += row.CallCount
	}
	response.Validity = h.validityForSample(sample)
	response.Provenance = h.provenanceWithQuery(ctx, filter, definitionFor(ToolUsage).SourceTable, sample, query)
	response.DefinitionId = definitionID(ToolUsage)
	return connect.NewResponse(response), nil
}

func (h *Handler) ToolCommandBreakdown(ctx context.Context, req *connect.Request[measurepb.ToolCommandBreakdownRequest]) (*connect.Response[measurepb.ToolCommandBreakdownResponse], error) {
	filter, err := filterWithWindow(req.Msg.GetFilter(), req.Msg.GetWindow(), h.now())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	limit := int(req.Msg.GetLimit())
	if limit <= 0 {
		limit = 20
	}
	commandStore, ok := h.store.(interface {
		ToolCommandBreakdown(context.Context, invocationreadmodel.Filter, int) ([]invocationreadmodel.ToolCommandRow, error)
	})
	if !ok {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("tool command detail is unavailable"))
	}
	rows, err := commandStore.ToolCommandBreakdown(ctx, filter, limit)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &measurepb.ToolCommandBreakdownResponse{Rows: make([]*measurepb.ToolCommandBreakdownRow, 0, len(rows))}
	var sample int64
	for _, row := range rows {
		sample += row.CallCount
		response.Rows = append(response.Rows, &measurepb.ToolCommandBreakdownRow{Executable: row.Executable, CommandPath: row.CommandPath, CallCount: row.CallCount, SuccessCount: row.SuccessCount, FailedCount: row.FailedCount, RunCount: row.RunCount, TotalTokens: row.TotalTokens, EstimatedTokenShare: row.EstimatedTokenShare, P50FootprintTokens: row.P50FootprintTokens, P95FootprintTokens: row.P95FootprintTokens, MaxFootprintTokens: row.MaxFootprintTokens, Truncated: row.Truncated})
	}
	query := fmt.Sprintf("SELECT executable, command_path, COUNT(*) FROM invocation_read_model_facts WHERE tool_name = %q GROUP BY executable, command_path LIMIT %d", filter.ToolName, limit)
	response.Validity = h.validityForSample(sample)
	response.Provenance = h.provenanceWithQuery(ctx, filter, definitionFor("friction.tool_command_breakdown").SourceTable, sample, query)
	response.DefinitionId = definitionID("friction.tool_command_breakdown")
	return connect.NewResponse(response), nil
}

func (h *Handler) TokenAttribution(ctx context.Context, req *connect.Request[measurepb.TokenAttributionRequest]) (*connect.Response[measurepb.TokenAttributionResponse], error) {
	groupBy, view := strings.TrimSpace(req.Msg.GetGroupBy()), strings.TrimSpace(req.Msg.GetView())
	if err := validateTokenAttributionSelection(groupBy, view); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	filter, err := filterWithWindow(req.Msg.GetFilter(), req.Msg.GetWindow(), h.now())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	store, ok := h.store.(tokenAttributionStore)
	if !ok {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("token attribution source is unavailable"))
	}
	limit := int(req.Msg.GetLimit())
	if limit <= 0 {
		limit = 20
	}
	rows, err := store.TokenAttribution(ctx, filter, groupBy, view, limit)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &measurepb.TokenAttributionResponse{GroupBy: groupBy, View: view, Rows: make([]*measurepb.TokenAttributionRow, 0, len(rows))}
	var sample, totalTokens, estimatedTokens int64
	for _, row := range rows {
		sample += row.CallCount
		totalTokens += row.TotalTokens
		estimatedTokens += row.EstimatedTokens
		response.Rows = append(response.Rows, &measurepb.TokenAttributionRow{GroupBy: row.GroupBy, Value: row.Value, CallCount: row.CallCount, TotalTokens: row.TotalTokens, EstimatedTokens: row.EstimatedTokens, EstimatedTokenShare: row.EstimatedTokenShare, P50FootprintTokens: row.P50FootprintTokens, P95FootprintTokens: row.P95FootprintTokens, MaxFootprintTokens: row.MaxFootprintTokens})
	}
	if totalTokens > 0 {
		response.EstimatedTokenShare = float64(estimatedTokens) / float64(totalTokens)
	}
	query := fmt.Sprintf("SELECT %s token aggregates FROM invocation_read_model_facts WHERE occurred_at >= %q AND occurred_at < %q GROUP BY %s ORDER BY %s DESC LIMIT %d", view, filter.From.UTC().Format(time.RFC3339Nano), filter.To.UTC().Format(time.RFC3339Nano), groupBy, view, limit)
	response.Validity = h.validityForSample(sample)
	response.Provenance = h.provenanceWithQuery(ctx, filter, definitionFor(TokenAttribution).SourceTable, sample, query)
	response.DefinitionId = definitionID(TokenAttribution)
	return connect.NewResponse(response), nil
}

func (h *Handler) ConversationSearchQuality(ctx context.Context, req *connect.Request[measurepb.ConversationSearchQualityRequest]) (*connect.Response[measurepb.ConversationSearchQualityResponse], error) {
	if h.conversationSearch == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("conversation search telemetry is unavailable"))
	}
	filter, err := filterWithWindow(nil, req.Msg.GetWindow(), h.now())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	aggregate, err := h.conversationSearch.AggregateSearchTelemetry(ctx, *filter.From, *filter.To, 10000)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	rate := func(value int64) float64 {
		if aggregate.Queries == 0 {
			return 0
		}
		return float64(value) / float64(aggregate.Queries)
	}
	validity := assessValidity(aggregate.Queries, 0, h.validityConfig)
	if aggregate.Truncated {
		validity.Availability = availability.New(availability.Unreliable, "window exceeds the bounded 10000-request aggregation sample")
	}
	response := &measurepb.ConversationSearchQualityResponse{
		Queries: aggregate.Queries, NoResultQueries: aggregate.NoResult, NoResultRate: rate(aggregate.NoResult),
		WeakOnlyQueries: aggregate.WeakOnly, WeakOnlyRate: rate(aggregate.WeakOnly),
		ReformulatedQueries: aggregate.Reformulated, ReformulationRate: rate(aggregate.Reformulated),
		SelectedQueries: aggregate.Selected, SelectedQueryRate: rate(aggregate.Selected),
		P50SelectedRank: aggregate.P50SelectedRank, P95SelectedRank: aggregate.P95SelectedRank,
		LexicalContributionRate: rate(aggregate.LexicalContributed), SemanticContributionRate: rate(aggregate.SemanticContributed),
		DegradedQueries: aggregate.Degraded, DegradationRate: rate(aggregate.Degraded),
		P50LatencyMs: aggregate.P50LatencyMS, P95LatencyMs: aggregate.P95LatencyMS,
		ErrorQueries: aggregate.Errors, ErrorRate: rate(aggregate.Errors), Truncated: aggregate.Truncated,
		Validity: protoValidity(validity), DefinitionId: definitionID(ConversationSearchQuality),
		Provenance: h.provenanceWithQuery(ctx, filter, "conversation_search_telemetry", aggregate.Queries, "SELECT bounded categorical conversation-search aggregates; no query or result content columns exist"),
	}
	if h.conversationIndexer != nil {
		if status, statusErr := h.conversationIndexer.StatusSnapshot(ctx); statusErr == nil {
			response.PendingDocuments = status.PendingChanges
			response.OrphanDocuments = status.OrphanDocuments
			if !status.LastSuccessAt.IsZero() {
				response.CurrentIndexLagMs = maxInt64Measure(0, h.now().Sub(status.LastSuccessAt).Milliseconds())
			}
		}
	}
	return connect.NewResponse(response), nil
}

func maxInt64Measure(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func (h *Handler) Registry() (*measurelib.Registry, error) {
	registry := measurelib.NewRegistry(measurelib.WithClock(h.now))
	for _, spec := range declarations() {
		spec := spec
		if err := registry.Register(declaration(spec), func(ctx context.Context, request measurelib.MeasureRequest) (measurelib.MeasureResult, error) {
			rangeValue, err := h.resolveWindow(request, measurelib.TokenThisWeek)
			if err != nil {
				return measurelib.MeasureResult{}, err
			}
			filter := invocationreadmodel.Filter{From: &rangeValue.From, To: &rangeValue.To}
			if spec.kind == tokenAttribution {
				groupBy, view := request.Params["group_by"], request.Params["view"]
				if err := validateTokenAttributionSelection(groupBy, view); err != nil {
					return measurelib.MeasureResult{}, err
				}
				store, ok := h.store.(tokenAttributionStore)
				if !ok {
					return measurelib.MeasureResult{}, fmt.Errorf("token attribution source is unavailable")
				}
				rows, err := store.TokenAttribution(ctx, filter, groupBy, view, 20)
				if err != nil {
					return measurelib.MeasureResult{}, err
				}
				fields := make([]map[string]string, 0, len(rows))
				for _, row := range rows {
					fields = append(fields, map[string]string{"group_by": row.GroupBy, "value": row.Value, "call_count": strconv.FormatInt(row.CallCount, 10), "total_tokens": strconv.FormatInt(row.TotalTokens, 10), "estimated_tokens": strconv.FormatInt(row.EstimatedTokens, 10), "estimated_token_share": strconv.FormatFloat(row.EstimatedTokenShare, 'f', -1, 64), "p50_footprint_tokens": strconv.FormatInt(row.P50FootprintTokens, 10), "p95_footprint_tokens": strconv.FormatInt(row.P95FootprintTokens, 10), "max_footprint_tokens": strconv.FormatInt(row.MaxFootprintTokens, 10)})
				}
				return measurelib.MeasureResult{Fields: fields, Provenance: measurelib.Provenance{ExecutedQuery: "SELECT token attribution aggregates from invocation_read_model_facts"}}, nil
			}
			result, err := h.metricFor(ctx, spec.kind, filter)
			if err != nil {
				return measurelib.MeasureResult{}, err
			}
			row := map[string]string{"validity": string(result.Validity.State), "sample_size": strconv.FormatInt(result.Validity.SampleSize, 10)}
			if result.Validity.Reason != "" {
				row["validity_reason"] = result.Validity.Reason
			}
			switch spec.kind {
			case runCycleTime:
				row["p50_ms"], row["p90_ms"] = strconv.FormatFloat(result.P50, 'f', -1, 64), strconv.FormatFloat(result.P90, 'f', -1, 64)
			case runCost:
				row["requests"], row["runs"] = strconv.FormatInt(result.Numerator, 10), strconv.FormatInt(result.Denom, 10)
				// The per-run distribution is the same one the RunCost RPC returns.
				if distStore, ok := h.store.(runCostDistributionStore); ok {
					dist, err := distStore.RunCostDistribution(ctx, filter)
					if err != nil {
						return measurelib.MeasureResult{}, err
					}
					for name, value := range map[string]int64{"p50_tokens": dist.P50Tokens, "p90_tokens": dist.P90Tokens, "p95_tokens": dist.P95Tokens, "p99_tokens": dist.P99Tokens, "max_tokens": dist.MaxTokens, "distribution_runs": dist.SampleSize} {
						row[name] = strconv.FormatInt(value, 10)
					}
					for name, value := range map[string]float64{"p50_cost_usd": dist.P50CostUSD, "p90_cost_usd": dist.P90CostUSD, "p95_cost_usd": dist.P95CostUSD, "p99_cost_usd": dist.P99CostUSD, "max_cost_usd": dist.MaxCostUSD} {
						row[name] = strconv.FormatFloat(value, 'f', -1, 64)
					}
				}
			}
			return measurelib.MeasureResult{Value: strconv.FormatFloat(result.Rate, 'f', -1, 64), Fields: []map[string]string{row}, Provenance: measurelib.Provenance{ExecutedQuery: result.Query}}, nil
		}); err != nil {
			return nil, err
		}
	}
	if err := registry.Register(measurelib.MeasureDeclaration{
		Name: "throughput.run_duration_statistics", Scenario: "agent-manager", Domain: "run",
		Intent:    "Duration percentiles and range for completed Agent Manager runs.",
		Questions: []string{"what are the agent run duration percentiles"},
		Params:    map[string]measurelib.Param{"window": {Name: "window", Type: measurelib.ParamTypeTimeWindow, Default: string(measurelib.TokenThisWeek)}},
		Result:    measurelib.Result{Kind: measurelib.ResultScalar, ValueField: "value", Unit: "milliseconds", SummaryTemplate: "run duration statistics ({window})"},
		Effect:    measurelib.EffectRead, RunEligible: true, Service: "MeasuresService", Method: "RunDurationStatistics",
	}, func(ctx context.Context, request measurelib.MeasureRequest) (measurelib.MeasureResult, error) {
		window, err := h.resolveWindow(request, measurelib.TokenThisWeek)
		if err != nil {
			return measurelib.MeasureResult{}, err
		}
		filter := invocationreadmodel.Filter{From: &window.From, To: &window.To}
		stats, err := h.store.RunDurationStatistics(ctx, filter)
		if err != nil {
			return measurelib.MeasureResult{}, err
		}
		validity := h.validityForSample(stats.Count)
		fields := map[string]string{"count": strconv.FormatInt(stats.Count, 10), "p50_ms": strconv.FormatFloat(stats.P50DurationMS, 'f', -1, 64), "p90_ms": strconv.FormatFloat(stats.P90DurationMS, 'f', -1, 64), "p95_ms": strconv.FormatFloat(stats.P95DurationMS, 'f', -1, 64), "p99_ms": strconv.FormatFloat(stats.P99DurationMS, 'f', -1, 64), "min_ms": strconv.FormatInt(stats.MinDurationMS, 10), "max_ms": strconv.FormatInt(stats.MaxDurationMS, 10), "validity": validity.GetState()}
		if validity.GetReason() != "" {
			fields["validity_reason"] = validity.GetReason()
		}
		return measurelib.MeasureResult{Value: strconv.FormatFloat(stats.AverageDurationMS, 'f', -1, 64), Fields: []map[string]string{fields}, Provenance: measurelib.Provenance{ExecutedQuery: fmt.Sprintf("SELECT duration statistics FROM invocation_read_model_runs WHERE occurred_at >= %q AND occurred_at < %q", window.From.UTC().Format(time.RFC3339Nano), window.To.UTC().Format(time.RFC3339Nano))}}, nil
	}); err != nil {
		return nil, err
	}
	if err := registry.Register(measurelib.MeasureDeclaration{
		Name: ConversationSearchQuality, Scenario: "agent-manager", Domain: "conversation_search",
		Intent:    "Privacy-safe conversation recall quality, interaction, degradation, latency, and index-lag evidence.",
		Questions: []string{"how reliable is conversation recall this week", "what is the conversation search no-result and degradation rate", "how fast is agent conversation search"},
		Params:    map[string]measurelib.Param{"window": {Name: "window", Type: measurelib.ParamTypeTimeWindow, Default: string(measurelib.TokenThisWeek)}},
		Result:    measurelib.Result{Kind: measurelib.ResultTable, ValueField: "queries", Unit: "queries", SummaryTemplate: "conversation search quality ({window})"},
		Effect:    measurelib.EffectRead, RunEligible: true, Service: "MeasuresService", Method: "ConversationSearchQuality",
	}, func(ctx context.Context, request measurelib.MeasureRequest) (measurelib.MeasureResult, error) {
		if h.conversationSearch == nil {
			return measurelib.MeasureResult{}, fmt.Errorf("conversation search telemetry is unavailable")
		}
		rangeValue, err := h.resolveWindow(request, measurelib.TokenThisWeek)
		if err != nil {
			return measurelib.MeasureResult{}, err
		}
		aggregate, err := h.conversationSearch.AggregateSearchTelemetry(ctx, rangeValue.From, rangeValue.To, 10000)
		if err != nil {
			return measurelib.MeasureResult{}, err
		}
		rate := func(value int64) string {
			if aggregate.Queries == 0 {
				return "0"
			}
			return strconv.FormatFloat(float64(value)/float64(aggregate.Queries), 'f', -1, 64)
		}
		fields := []map[string]string{{
			"queries": strconv.FormatInt(aggregate.Queries, 10), "no_result_rate": rate(aggregate.NoResult),
			"weak_only_rate": rate(aggregate.WeakOnly), "reformulation_rate": rate(aggregate.Reformulated),
			"selected_query_rate": rate(aggregate.Selected), "lexical_contribution_rate": rate(aggregate.LexicalContributed),
			"semantic_contribution_rate": rate(aggregate.SemanticContributed), "degradation_rate": rate(aggregate.Degraded),
			"error_rate": rate(aggregate.Errors), "p50_latency_ms": strconv.FormatFloat(aggregate.P50LatencyMS, 'f', -1, 64),
			"p95_latency_ms": strconv.FormatFloat(aggregate.P95LatencyMS, 'f', -1, 64), "truncated": strconv.FormatBool(aggregate.Truncated),
		}}
		return measurelib.MeasureResult{Fields: fields, Provenance: measurelib.Provenance{ExecutedQuery: "SELECT bounded categorical conversation-search aggregates; no query or result content columns exist"}}, nil
	}); err != nil {
		return nil, err
	}
	if err := registry.Register(measurelib.MeasureDeclaration{
		Name: RunStatusDistribution, Scenario: "agent-manager", Domain: "run", Intent: "Distribution of durable terminal run summaries by status.",
		Questions: []string{"what is the agent run status distribution this week", "how many agent runs are complete, failed, or cancelled"},
		Params:    map[string]measurelib.Param{"window": {Name: "window", Type: measurelib.ParamTypeTimeWindow, Default: string(measurelib.TokenThisWeek)}},
		Result:    measurelib.Result{Kind: measurelib.ResultTable, ValueField: "status", Unit: "runs", SummaryTemplate: "run status distribution ({window})"},
		Effect:    measurelib.EffectRead, RunEligible: true, Service: "MeasuresService", Method: "RunStatusDistribution",
	}, func(ctx context.Context, request measurelib.MeasureRequest) (measurelib.MeasureResult, error) {
		rangeValue, err := h.resolveWindow(request, measurelib.TokenThisWeek)
		if err != nil {
			return measurelib.MeasureResult{}, err
		}
		rows, err := h.store.RunStatusCounts(ctx, invocationreadmodel.Filter{From: &rangeValue.From, To: &rangeValue.To})
		if err != nil {
			return measurelib.MeasureResult{}, err
		}
		fields := make([]map[string]string, 0, len(rows))
		for _, row := range rows {
			fields = append(fields, map[string]string{"status": row.Status, "count": strconv.FormatInt(row.Count, 10)})
		}
		query := fmt.Sprintf("SELECT status, COUNT(*) FROM invocation_read_model_runs WHERE occurred_at >= %q AND occurred_at < %q GROUP BY status", rangeValue.From.UTC().Format(time.RFC3339Nano), rangeValue.To.UTC().Format(time.RFC3339Nano))
		return measurelib.MeasureResult{Fields: fields, Provenance: measurelib.Provenance{ExecutedQuery: query}}, nil
	}); err != nil {
		return nil, err
	}
	for _, spec := range []struct{ name, method, dimension, intent string }{
		{RunnerBreakdown, "RunnerBreakdown", "runner", "Durable terminal-run performance grouped by runner."},
		{ModelBreakdown, "ModelBreakdown", "model", "Durable terminal-run performance grouped by model."},
		{ProfileBreakdown, "ProfileBreakdown", "profile", "Durable terminal-run performance grouped by profile."},
		{WorkloadBreakdown, "WorkloadBreakdown", "workload", "Durable terminal-run performance grouped by declared workload key."},
	} {
		spec := spec
		if err := registry.Register(measurelib.MeasureDeclaration{
			Name: spec.name, Scenario: "agent-manager", Domain: "run", Intent: spec.intent,
			Questions: []string{"agent run " + spec.dimension + " breakdown this week"}, Params: map[string]measurelib.Param{"window": {Name: "window", Type: measurelib.ParamTypeTimeWindow, Default: string(measurelib.TokenThisWeek)}},
			Result: measurelib.Result{Kind: measurelib.ResultTable, ValueField: "value", Unit: "runs", SummaryTemplate: spec.dimension + " run breakdown ({window})"}, Effect: measurelib.EffectRead, RunEligible: true, Service: "MeasuresService", Method: spec.method,
		}, func(ctx context.Context, request measurelib.MeasureRequest) (measurelib.MeasureResult, error) {
			rangeValue, err := h.resolveWindow(request, measurelib.TokenThisWeek)
			if err != nil {
				return measurelib.MeasureResult{}, err
			}
			rows, err := h.store.RunBreakdown(ctx, invocationreadmodel.Filter{From: &rangeValue.From, To: &rangeValue.To}, spec.dimension, 20)
			if err != nil {
				return measurelib.MeasureResult{}, err
			}
			fields := make([]map[string]string, 0, len(rows))
			for _, row := range rows {
				fields = append(fields, map[string]string{"value": row.Value, "run_count": strconv.FormatInt(row.RunCount, 10), "success_count": strconv.FormatInt(row.SuccessCount, 10), "failed_count": strconv.FormatInt(row.FailedCount, 10), "total_cost_usd": strconv.FormatFloat(row.TotalCostUSD, 'f', -1, 64), "average_duration_ms": strconv.FormatFloat(row.AvgDurationMS, 'f', -1, 64), "consumption_per_successful_completion": strconv.FormatFloat(row.ConsumptionPerSuccessfulCompletion, 'f', -1, 64), "completion_rate": strconv.FormatFloat(row.CompletionRate, 'f', -1, 64), "observational_limitation": "model assignment was not randomised"})
			}
			query := fmt.Sprintf("SELECT %s, terminal run aggregates FROM invocation_read_model_runs WHERE occurred_at >= %q AND occurred_at < %q GROUP BY %s LIMIT 20", spec.dimension, rangeValue.From.UTC().Format(time.RFC3339Nano), rangeValue.To.UTC().Format(time.RFC3339Nano), spec.dimension)
			return measurelib.MeasureResult{Fields: fields, Provenance: measurelib.Provenance{ExecutedQuery: query}}, nil
		}); err != nil {
			return nil, err
		}
	}
	if err := registry.Register(measurelib.MeasureDeclaration{
		Name: WorkloadEfficiency, Scenario: "agent-manager", Domain: "run",
		Intent:    "Consumption per successful completion, including failed attempts in the numerator.",
		Questions: []string{"which workload consumes the fewest tokens per successful completion", "show workload efficiency"},
		Params: map[string]measurelib.Param{
			"window":       {Name: "window", Type: measurelib.ParamTypeTimeWindow, Default: string(measurelib.TokenThisWeek)},
			"workload_key": {Name: "workload_key", Type: "string"},
			"runner_type":  {Name: "runner_type", Type: "string"},
			"model":        {Name: "model", Type: "string"},
		},
		Result: measurelib.Result{Kind: measurelib.ResultScalar, ValueField: "value", Unit: "tokens_per_success", SummaryTemplate: "workload efficiency ({window})"},
		Effect: measurelib.EffectRead, RunEligible: true, Service: "MeasuresService", Method: "WorkloadEfficiency",
	}, func(ctx context.Context, request measurelib.MeasureRequest) (measurelib.MeasureResult, error) {
		rangeValue, err := h.resolveWindow(request, measurelib.TokenThisWeek)
		if err != nil {
			return measurelib.MeasureResult{}, err
		}
		filter := invocationreadmodel.Filter{From: &rangeValue.From, To: &rangeValue.To, WorkloadKey: request.Params["workload_key"], RunnerType: request.Params["runner_type"], Model: request.Params["model"]}
		runs, err := h.store.RunMetrics(ctx, filter)
		if err != nil {
			return measurelib.MeasureResult{}, err
		}
		usage, err := h.store.WindowedUsage(ctx, filter)
		if err != nil {
			return measurelib.MeasureResult{}, err
		}
		validity := "reliable"
		if runs.TerminalRuns < 5 {
			validity = "unreliable"
		}
		query := fmt.Sprintf("SELECT SUM(tokens requested in the window) / successful completions FROM transcript_usage JOIN invocation_read_model_runs WHERE workload_key = %q AND observed_at >= %q AND observed_at < %q", filter.WorkloadKey, rangeValue.From.UTC().Format(time.RFC3339Nano), rangeValue.To.UTC().Format(time.RFC3339Nano))
		return measurelib.MeasureResult{
			Value:      strconv.FormatFloat(perSuccess(usage.TotalTokens, runs.SuccessfulRuns), 'f', -1, 64),
			Fields:     []map[string]string{{"validity": validity, "terminal_runs": strconv.FormatInt(runs.TerminalRuns, 10), "successful_runs": strconv.FormatInt(runs.SuccessfulRuns, 10), "observational_limitation": "model assignment was not randomised"}},
			Provenance: measurelib.Provenance{ExecutedQuery: query},
		}, nil
	}); err != nil {
		return nil, err
	}
	if err := registry.Register(measurelib.MeasureDeclaration{Name: TerminalRunTrend, Scenario: "agent-manager", Domain: "run", Intent: "Hourly durable terminal-run outcomes, cost, and duration trends.", Questions: []string{"show the agent terminal run trend this week", "how are agent run completions and failures trending"}, Params: map[string]measurelib.Param{"window": {Name: "window", Type: measurelib.ParamTypeTimeWindow, Default: string(measurelib.TokenThisWeek)}}, Result: measurelib.Result{Kind: measurelib.ResultTable, ValueField: "bucket", Unit: "runs", SummaryTemplate: "terminal run trend ({window})"}, Effect: measurelib.EffectRead, RunEligible: true, Service: "MeasuresService", Method: "TerminalRunTrend"}, func(ctx context.Context, request measurelib.MeasureRequest) (measurelib.MeasureResult, error) {
		rangeValue, err := h.resolveWindow(request, measurelib.TokenThisWeek)
		if err != nil {
			return measurelib.MeasureResult{}, err
		}
		rows, err := h.store.RunTimeSeries(ctx, invocationreadmodel.Filter{From: &rangeValue.From, To: &rangeValue.To}, time.Hour)
		if err != nil {
			return measurelib.MeasureResult{}, err
		}
		fields := make([]map[string]string, 0, len(rows))
		for _, row := range rows {
			fields = append(fields, map[string]string{"bucket": row.Bucket.UTC().Format(time.RFC3339), "terminal_runs": strconv.FormatInt(row.TerminalRuns, 10), "completed_runs": strconv.FormatInt(row.CompletedRuns, 10), "failed_runs": strconv.FormatInt(row.FailedRuns, 10), "cancelled_runs": strconv.FormatInt(row.CancelledRuns, 10), "average_duration_ms": strconv.FormatFloat(row.AvgDurationMS, 'f', -1, 64)})
		}
		query := fmt.Sprintf("SELECT terminal outcome aggregates FROM invocation_read_model_runs WHERE occurred_at >= %q AND occurred_at < %q GROUP BY hourly terminal bucket", rangeValue.From.UTC().Format(time.RFC3339Nano), rangeValue.To.UTC().Format(time.RFC3339Nano))
		return measurelib.MeasureResult{Fields: fields, Provenance: measurelib.Provenance{ExecutedQuery: query}}, nil
	}); err != nil {
		return nil, err
	}
	if err := registry.Register(measurelib.MeasureDeclaration{Name: ToolUsage, Scenario: "agent-manager", Domain: "friction", Intent: "Durable tool invocation counts and classified outcomes by tool.", Questions: []string{"which tools do agents use most", "agent tool usage this week"}, Params: map[string]measurelib.Param{"window": {Name: "window", Type: measurelib.ParamTypeTimeWindow, Default: string(measurelib.TokenThisWeek)}}, Result: measurelib.Result{Kind: measurelib.ResultTable, ValueField: "tool_name", Unit: "calls", SummaryTemplate: "tool usage ({window})"}, Effect: measurelib.EffectRead, RunEligible: true, Service: "MeasuresService", Method: "ToolUsage"}, func(ctx context.Context, request measurelib.MeasureRequest) (measurelib.MeasureResult, error) {
		rangeValue, err := h.resolveWindow(request, measurelib.TokenThisWeek)
		if err != nil {
			return measurelib.MeasureResult{}, err
		}
		rows, err := h.store.ToolUsage(ctx, invocationreadmodel.Filter{From: &rangeValue.From, To: &rangeValue.To}, 20)
		if err != nil {
			return measurelib.MeasureResult{}, err
		}
		fields := make([]map[string]string, 0, len(rows))
		for _, row := range rows {
			fields = append(fields, map[string]string{"tool_name": row.ToolName, "call_count": strconv.FormatInt(row.CallCount, 10), "success_count": strconv.FormatInt(row.SuccessCount, 10), "failed_count": strconv.FormatInt(row.FailedCount, 10)})
		}
		query := fmt.Sprintf("SELECT tool_name, classified outcomes FROM invocation_read_model_facts WHERE occurred_at >= %q AND occurred_at < %q GROUP BY tool_name LIMIT 20", rangeValue.From.UTC().Format(time.RFC3339Nano), rangeValue.To.UTC().Format(time.RFC3339Nano))
		return measurelib.MeasureResult{Fields: fields, Provenance: measurelib.Provenance{ExecutedQuery: query}}, nil
	}); err != nil {
		return nil, err
	}
	if err := h.registerFallbackInsights(registry); err != nil {
		return nil, err
	}
	if err := h.registerUsageMeasures(registry); err != nil {
		return nil, err
	}
	if err := h.registerInterventions(registry); err != nil {
		return nil, err
	}
	if err := h.registerOperatorSignals(registry); err != nil {
		return nil, err
	}
	if err := h.registerOpportunities(registry); err != nil {
		return nil, err
	}
	return registry, nil
}

func (h *Handler) MeasuresHandler() (http.Handler, error) {
	registry, err := h.Registry()
	if err != nil {
		return nil, err
	}
	return registry.Handler(), nil
}

// RuntimeMeasureBinding is an explicitly governed typed RPC projection, not a
// CLI command. Its owner-produced snapshot lets Runtime resolve the existing
// measure service without depending on a running Agent Manager instance.
type RuntimeMeasureBinding struct {
	ID           string   `json:"id"`
	Service      string   `json:"service"`
	Method       string   `json:"method"`
	Effect       string   `json:"effect"`
	RunEligible  bool     `json:"run_eligible"`
	Permissions  []string `json:"permissions"`
	PrimaryField string   `json:"primary_field"`
}

type RuntimeMeasureBindings struct {
	Version       int                     `json:"version"`
	Scenario      string                  `json:"scenario"`
	SourceDigests map[string]string       `json:"source_digests"`
	Bindings      []RuntimeMeasureBinding `json:"bindings"`
}

// RuntimeBindings is a deliberately narrow owner policy. Registry membership
// alone is not permission to expose every measure RPC as a program binding.
// Adding a method here requires an explicit owner-reviewed governance choice.
func (h *Handler) RuntimeBindings() (RuntimeMeasureBindings, error) {
	registry, err := h.Registry()
	if err != nil {
		return RuntimeMeasureBindings{}, err
	}
	type policy struct{ name, method, id, primary string }
	policies := []policy{
		{ToolFriction, "ToolFriction", "tool-friction", "summary"},
		{RunSuccessRate, "RunSuccessRate", "run-success-rate", "rate"},
		{RunVolume, "RunVolume", "run-volume", "total_runs"},
		{"throughput.run_duration_statistics", "RunDurationStatistics", "run-duration-statistics", "average_duration_ms"},
		{RunCost, "RunCost", "run-cost", "total_cost_usd"},
		{RunStatusDistribution, "RunStatusDistribution", "run-status-distribution", "rows"},
		{RunnerBreakdown, "RunnerBreakdown", "runner-breakdown", "rows"},
		{ModelBreakdown, "ModelBreakdown", "model-breakdown", "rows"},
		{ProfileBreakdown, "ProfileBreakdown", "profile-breakdown", "rows"},
		{ToolUsage, "ToolUsage", "tool-usage", "rows"},
	}
	service := measurepb.File_agent_manager_v1_measures_measures_proto.Services().ByName("MeasuresService")
	if service == nil {
		return RuntimeMeasureBindings{}, fmt.Errorf("measure service descriptor is unavailable")
	}
	output := RuntimeMeasureBindings{Version: 1, Scenario: "agent-manager", Bindings: make([]RuntimeMeasureBinding, 0, len(policies))}
	for _, p := range policies {
		decl, ok := registry.Lookup(p.name)
		if !ok || decl.Scenario != output.Scenario || decl.Service != "MeasuresService" || decl.Method != p.method || decl.Effect != measurelib.EffectRead || !decl.RunEligible {
			return RuntimeMeasureBindings{}, fmt.Errorf("runtime measure policy does not match eligible registry declaration %q", p.name)
		}
		method := service.Methods().ByName(protoreflect.Name(p.method))
		if method == nil || method.Output().Fields().ByName(protoreflect.Name(p.primary)) == nil {
			return RuntimeMeasureBindings{}, fmt.Errorf("runtime measure %q has an unresolved method or primary field", p.name)
		}
		output.Bindings = append(output.Bindings, RuntimeMeasureBinding{ID: output.Scenario + "/measures/" + p.id, Service: string(service.FullName()), Method: p.method, Effect: string(decl.Effect), RunEligible: decl.RunEligible, Permissions: []string{"network:internal"}, PrimaryField: p.primary})
	}
	return output, nil
}

var _ measureconnect.MeasuresServiceHandler = (*Handler)(nil)
