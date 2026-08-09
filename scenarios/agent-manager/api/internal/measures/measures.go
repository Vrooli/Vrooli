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
	"agent-manager/internal/invocationreadmodel"
	"agent-manager/internal/runreport"

	"connectrpc.com/connect"
	measurelib "github.com/vrooli/measures-go"
	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
	measureconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures/measures_v1connect"
	sharedmeasurepb "github.com/vrooli/vrooli/packages/proto/gen/go/measures/v1"
)

const (
	ExternalToolShare     = "friction.external_tool_share"
	RetryRate             = "friction.retry_rate"
	HelpRecoveryRate      = "friction.help_recovery_rate"
	RepeatedWorkRate      = "friction.repeated_work_rate"
	ToolFailureRate       = "friction.tool_failure_rate"
	RunSuccessRate        = "throughput.run_success_rate"
	RunCycleTime          = "throughput.run_cycle_time"
	RunCost               = "throughput.run_cost"
	RunVolume             = "throughput.run_volume"
	RunStatusDistribution = "throughput.run_status_distribution"
	RunnerBreakdown       = "throughput.runner_breakdown"
	ModelBreakdown        = "throughput.model_breakdown"
	ProfileBreakdown      = "throughput.profile_breakdown"
	WorkloadBreakdown     = "throughput.workload_breakdown"
	WorkloadEfficiency    = "throughput.workload_efficiency"
	TerminalRunTrend      = "throughput.terminal_run_trend"
	ToolUsage             = "friction.tool_usage"
	TokenAttribution      = "friction.token_attribution"
	ErrorPatterns         = "friction.error_patterns"
	FileRereadRate        = "friction.file_reread_rate"
	FindingRecurrenceRate = "friction.finding_recurrence_rate"
)

// Store is the narrow durable analytical substrate. The database read-model
// repository satisfies it; no measure reads raw run_events JSON.
type Store interface {
	Metrics(context.Context, invocationreadmodel.Filter) (invocationreadmodel.Metrics, error)
	RunMetrics(context.Context, invocationreadmodel.Filter) (invocationreadmodel.RunMetrics, error)
	RunDurationStatistics(context.Context, invocationreadmodel.Filter) (invocationreadmodel.RunDurationStatistics, error)
	RunStatusCounts(context.Context, invocationreadmodel.Filter) ([]invocationreadmodel.RunStatusCount, error)
	RunBreakdown(context.Context, invocationreadmodel.Filter, string, int) ([]invocationreadmodel.RunBreakdownRow, error)
	RunTimeSeries(context.Context, invocationreadmodel.Filter, time.Duration) ([]invocationreadmodel.RunTimeSeriesBucket, error)
	ToolUsage(context.Context, invocationreadmodel.Filter, int) ([]invocationreadmodel.ToolUsageRow, error)
	ErrorPatterns(context.Context, invocationreadmodel.Filter, int) ([]invocationreadmodel.ErrorPattern, error)
	FindingMetrics(context.Context, invocationreadmodel.Filter) (invocationreadmodel.FindingMetrics, error)
}

type tokenAttributionStore interface {
	TokenAttribution(context.Context, invocationreadmodel.Filter, string, string, int) ([]invocationreadmodel.TokenAttributionRow, error)
}

var tokenAttributionGroupByValues = []string{"capability", "executable", "command_path", "target_scenario_operation"}
var tokenAttributionViewValues = []string{"footprint", "residency", "incurred"}

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
	externalToolShare     metricKind = "external_tool_share"
	retryRate             metricKind = "retry_rate"
	helpRecoveryRate      metricKind = "help_recovery_rate"
	repeatedWorkRate      metricKind = "repeated_work_rate"
	toolFailureRate       metricKind = "tool_failure_rate"
	runSuccessRate        metricKind = "run_success_rate"
	runCycleTime          metricKind = "run_cycle_time"
	runCost               metricKind = "run_cost"
	runVolume             metricKind = "run_volume"
	fileRereadRate        metricKind = "file_reread_rate"
	findingRecurrenceRate metricKind = "finding_recurrence_rate"
	tokenAttribution      metricKind = "token_attribution"
)

type metricResult struct {
	Rate       float64
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
		ExternalToolShare:                    {ID: ExternalToolShare, Counts: "classified command invocations", Numerator: "external tool calls", Denominator: "classified command calls", SourceTable: "invocation_read_model_facts", Limitation: "unclassified ownership and unparseable evidence are excluded from the denominator"},
		RetryRate:                            {ID: RetryRate, Counts: "durable tool invocations", Numerator: "retry-linked calls", Denominator: "all classified tool calls", SourceTable: "invocation_read_model_facts"},
		HelpRecoveryRate:                     {ID: HelpRecoveryRate, Counts: "durable tool invocations", Numerator: "help-recovery calls", Denominator: "all classified tool calls", SourceTable: "invocation_read_model_facts"},
		RepeatedWorkRate:                     {ID: RepeatedWorkRate, Counts: "durable tool invocations", Numerator: "repeated-work calls", Denominator: "all classified tool calls", SourceTable: "invocation_read_model_facts", Limitation: "fingerprint concentration can make this measure unreliable"},
		ToolFailureRate:                      {ID: ToolFailureRate, Counts: "durable tool invocations", Numerator: "failed calls", Denominator: "all classified tool calls", SourceTable: "invocation_read_model_facts"},
		RunSuccessRate:                       {ID: RunSuccessRate, Counts: "executed terminal runs", Numerator: "successful executed terminal runs", Denominator: "all executed terminal runs", SourceTable: "invocation_read_model_runs", Limitation: "imported and interactive runs are reported separately and excluded"},
		RunCycleTime:                         {ID: RunCycleTime, Counts: "completed executed terminal runs", Numerator: "duration milliseconds", Denominator: "completed executed terminal runs", SourceTable: "invocation_read_model_runs", Limitation: "imported and interactive runs are reported separately and excluded"},
		"throughput.run_duration_statistics": {ID: "throughput.run_duration_statistics", Counts: "completed executed terminal runs", Numerator: "duration percentiles and range", Denominator: "completed executed terminal runs", SourceTable: "invocation_read_model_runs", Limitation: "imported and interactive runs are reported separately and excluded"},
		RunCost:                              {ID: RunCost, Counts: "executed terminal runs", Numerator: "charge and token totals", Denominator: "executed terminal runs", SourceTable: "invocation_read_model_runs", Limitation: "unpriced, imported, and interactive runs are excluded from metered consumption"},
		RunVolume:                            {ID: RunVolume, Counts: "executed terminal run summaries", Numerator: "executed terminal runs", Denominator: "none", SourceTable: "invocation_read_model_runs", Limitation: "imported and interactive runs are reported separately and excluded"},
		RunStatusDistribution:                {ID: RunStatusDistribution, Counts: "executed terminal runs", Numerator: "runs in each status", Denominator: "all executed terminal runs", SourceTable: "invocation_read_model_runs", Limitation: "imported and interactive runs are reported separately and excluded"},
		RunnerBreakdown:                      {ID: RunnerBreakdown, Counts: "executed terminal runs", Numerator: "runs grouped by runner", Denominator: "all executed terminal runs", SourceTable: "invocation_read_model_runs", Limitation: "imported and interactive runs are reported separately and excluded"},
		ModelBreakdown:                       {ID: ModelBreakdown, Counts: "executed terminal runs", Numerator: "runs grouped by model", Denominator: "all executed terminal runs", SourceTable: "invocation_read_model_runs", Limitation: "imported and interactive runs are reported separately and excluded; model assignment is observational, not randomized"},
		ProfileBreakdown:                     {ID: ProfileBreakdown, Counts: "executed terminal runs", Numerator: "runs grouped by profile", Denominator: "all executed terminal runs", SourceTable: "invocation_read_model_runs", Limitation: "imported and interactive runs are reported separately and excluded"},
		WorkloadBreakdown:                    {ID: WorkloadBreakdown, Counts: "executed terminal runs", Numerator: "workload totals and completion outcomes", Denominator: "all executed terminal runs", SourceTable: "invocation_read_model_runs", Limitation: "imported and interactive runs are reported separately and excluded; model assignment is observational, not randomized"},
		WorkloadEfficiency:                   {ID: WorkloadEfficiency, Counts: "executed terminal runs for a workload", Numerator: "total tokens", Denominator: "successful completions", SourceTable: "invocation_read_model_runs", Limitation: "imported and interactive runs are reported separately and excluded; model assignment is observational, not randomized"},
		TerminalRunTrend:                     {ID: TerminalRunTrend, Counts: "executed terminal runs", Numerator: "hourly terminal outcomes", Denominator: "none", SourceTable: "invocation_read_model_runs", Limitation: "imported and interactive runs are reported separately and excluded"},
		ToolUsage:                            {ID: ToolUsage, Counts: "tool invocation facts", Numerator: "calls grouped by tool", Denominator: "none", SourceTable: "invocation_read_model_facts"},
		"friction.tool_command_breakdown":    {ID: "friction.tool_command_breakdown", Counts: "tool invocation facts", Numerator: "calls grouped by executable and command path", Denominator: "none", SourceTable: "invocation_read_model_facts", Limitation: "command detail is unavailable when the source fact did not record it"},
		TokenAttribution:                     {ID: TokenAttribution, Counts: "durable invocation token factors", Numerator: "tokens grouped by the selected dimension and view", Denominator: "none", SourceTable: "invocation_read_model_facts", Limitation: "estimated share identifies rankings that rely on payload estimates; target scenario and operation require a verified receipt join"},
		ErrorPatterns:                        {ID: ErrorPatterns, Counts: "durable error facts", Numerator: "errors grouped by code", Denominator: "none", SourceTable: "invocation_read_model_errors"},
		FileRereadRate:                       {ID: FileRereadRate, Counts: "file-read calls", Numerator: "files read more than once", Denominator: "file-read calls", SourceTable: "invocation_read_model_runs"},
		FindingRecurrenceRate:                {ID: FindingRecurrenceRate, Counts: "persisted investigation findings", Numerator: "recurring findings", Denominator: "all findings", SourceTable: "run_findings"},
		"friction.capability_usage":          {ID: "friction.capability_usage", Counts: "verified capability receipts", Numerator: "calls grouped by capability", Denominator: "none", SourceTable: "investigation_cross_scenario_calls", Limitation: "only verified receipts are included"},
		"friction.capability_efficacy":       {ID: "friction.capability_efficacy", Counts: "verified capability receipts", Numerator: "successful, fallback, and abandoned calls", Denominator: "verified capability calls", SourceTable: "investigation_cross_scenario_calls", Limitation: "only verified receipts are included"},
		"select_cohort":                      {ID: "select_cohort", Counts: "terminal runs matching the selected filter", Numerator: "matching runs", Denominator: "none", SourceTable: "invocation_read_model_runs"},
		"episode_cohort":                     {ID: "episode_cohort", Counts: "persisted friction episodes", Numerator: "episodes grouped by fingerprint", Denominator: "none", SourceTable: "invocation_read_model_episodes"},
	}
	return definitions[name]
}

func allDefinitions() []Definition {
	seen := make(map[string]struct{})
	definitions := make([]Definition, 0, 24)
	for _, name := range []string{ExternalToolShare, RetryRate, HelpRecoveryRate, RepeatedWorkRate, ToolFailureRate, RunSuccessRate, RunCycleTime, "throughput.run_duration_statistics", RunCost, RunVolume, RunStatusDistribution, RunnerBreakdown, ModelBreakdown, ProfileBreakdown, WorkloadBreakdown, WorkloadEfficiency, TerminalRunTrend, ToolUsage, TokenAttribution, "friction.tool_command_breakdown", ErrorPatterns, FileRereadRate, FindingRecurrenceRate, "friction.capability_usage", "friction.capability_efficacy", "select_cohort", "episode_cohort"} {
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
		{"ownership", filter.Ownership}, {"outcome", filter.Outcome}, {"executable", filter.Executable}, {"fingerprint", filter.Fingerprint}, {"profile_id", filter.ProfileID}, {"runner_type", filter.RunnerType}, {"model", filter.Model}, {"workload_kind", filter.WorkloadKind}, {"workload_key", filter.WorkloadKey}, {"tag_prefix", filter.TagPrefix}, {"run_status", filter.RunStatus}, {"tool_name", filter.ToolName}, {"episode_pattern", filter.EpisodePattern}, {"episode_cause_scope", filter.EpisodeCauseScope}, {"episode_fingerprint", filter.EpisodeFingerprint}, {"self_report_rule_id", filter.SelfReportRuleID}, {"self_report_cause_scope", filter.SelfReportCauseScope}, {"target_scenario", filter.TargetScenario}, {"operation", filter.Operation},
	} {
		if item.value != "" {
			provenance.AppliedFilters = append(provenance.AppliedFilters, &measurepb.MeasureFilter{Field: item.field, Value: item.value})
		}
	}
	return provenance
}

func provenanceWithQuery(filter invocationreadmodel.Filter, source string, rows int64, query string) *measurepb.MeasureProvenance {
	provenance := provenanceFor(filter, source, rows)
	provenance.ExecutedQuery = query
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
		{ExternalToolShare, "ExternalToolShare", "Share of resolved tool invocations that target external tools, with unknown ownership reported separately.", "share", "{value} external-tool share ({window})", []string{"what is the external-tool share this week", "how often do agents use tools outside Vrooli", "which fraction of tool calls are external"}, externalToolShare},
		{RetryRate, "RetryRate", "Share of durable tool invocations linked to a prior retry target.", "share", "{value} retry rate ({window})", []string{"what is the retry rate this week", "how often are agent tool calls retried", "show tool invocation retries"}, retryRate},
		{HelpRecoveryRate, "HelpRecoveryRate", "Share of tool invocations that follow a help-recovery signal.", "share", "{value} help-recovery rate ({window})", []string{"what is the help recovery rate", "how often do agents recover after asking for help", "show help recoveries this month"}, helpRecoveryRate},
		{RepeatedWorkRate, "RepeatedWorkRate", "Share of tool invocations in repeated-work fingerprints.", "share", "{value} repeated-work rate ({window})", []string{"what is the repeated work rate", "how much tool work is repeated", "show agent reread and repeated invocation rate"}, repeatedWorkRate},
		{ToolFailureRate, "ToolFailureRate", "Share of durable tool invocations whose classified outcome failed.", "share", "{value} tool failure rate ({window})", []string{"what is the tool failure rate", "how often do agent tool calls fail", "show failed tool invocation share"}, toolFailureRate},
		{RunSuccessRate, "RunSuccessRate", "Share of executed terminal runs that completed successfully; imported and interactive runs are separate.", "share", "{value} run success rate ({window})", []string{"what is the agent run success rate this week", "how often do agent runs complete successfully"}, runSuccessRate},
		{RunCycleTime, "RunCycleTime", "Average executed completed run cycle time in milliseconds.", "milliseconds", "{value} average run cycle time ({window})", []string{"what is the average agent run cycle time", "how long do completed agent runs take"}, runCycleTime},
		{RunCost, "RunCost", "Total executed terminal run cost in USD with retained token usage.", "usd", "{value} agent run cost ({window})", []string{"what did agent runs cost this week", "show total agent run cost"}, runCost},
		{RunVolume, "RunVolume", "Number of executed terminal run summaries in the window.", "runs", "{value} agent run volume ({window})", []string{"how many agent runs completed this week", "show agent run volume"}, runVolume},
		{FileRereadRate, "FileRereadRate", "Share of file-read calls that revisit a path already read in the same run.", "share", "{value} file reread rate ({window})", []string{"what is the agent file reread rate", "how often do agents reread files"}, fileRereadRate},
		{FindingRecurrenceRate, "FindingRecurrenceRate", "Share of persisted investigation findings whose fingerprint recurs in the same filtered finding corpus.", "share", "{value} finding recurrence rate ({window})", []string{"what is the recurring finding rate", "which agent findings recur"}, findingRecurrenceRate},
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
	query := fmt.Sprintf("SELECT durable invocation aggregate FROM invocation_read_model_facts WHERE occurred_at >= %q AND occurred_at < %q", filter.From.UTC().Format(time.RFC3339Nano), filter.To.UTC().Format(time.RFC3339Nano))
	if kind == findingRecurrenceRate {
		findings, err := store.FindingMetrics(ctx, filter)
		if err != nil {
			return metricResult{}, err
		}
		return metricResult{Rate: findings.RecurrenceRate, Numerator: findings.RecurringFindings, Denom: findings.TotalFindings, Secondary: findings.RecurringFingerprints, Query: fmt.Sprintf("SELECT recurrence aggregate FROM run_findings WHERE created_at >= %q AND created_at < %q", filter.From.UTC().Format(time.RFC3339Nano), filter.To.UTC().Format(time.RFC3339Nano))}, nil
	}
	if kind == runSuccessRate || kind == runCycleTime || kind == runCost || kind == runVolume || kind == fileRereadRate {
		runs, err := store.RunMetrics(ctx, filter)
		if err != nil {
			return metricResult{}, err
		}
		result := metricResult{Query: fmt.Sprintf("SELECT durable terminal aggregate FROM invocation_read_model_runs WHERE occurred_at >= %q AND occurred_at < %q", filter.From.UTC().Format(time.RFC3339Nano), filter.To.UTC().Format(time.RFC3339Nano))}
		switch kind {
		case runSuccessRate:
			result.Rate, result.Numerator, result.Denom = runs.SuccessRate, runs.SuccessfulRuns, runs.TerminalRuns
		case runCycleTime:
			result.Rate, result.Numerator = runs.AverageDurationMS, runs.CompletedDurationRuns
		case runCost:
			result.Rate, result.Numerator, result.Denom = runs.TotalCostUSD, runs.TotalRuns, int64(runs.TotalTokens)
		case runVolume:
			result.Rate, result.Numerator, result.Denom = float64(runs.TotalRuns), runs.TotalRuns, runs.TerminalRuns
		case fileRereadRate:
			result.Rate, result.Numerator, result.Denom = runs.FileRereadRate, runs.FileRereads, runs.ReadCalls
		}
		return result, nil
	}
	metrics, err := store.Metrics(ctx, filter)
	if err != nil {
		return metricResult{}, err
	}
	unclassified := metrics.UnclassifiedCalls
	if unclassified == 0 {
		unclassified = metrics.UnknownCalls
	}
	result := metricResult{Unknown: unclassified, Query: query}
	population := metrics.TotalCalls
	if population == 0 {
		// Compatibility for focused stores created before the explicit total
		// population field; production repositories always populate it.
		population = metrics.ResolvedCalls + metrics.UnknownCalls
	}
	classifiedBase := metrics.ClassifiedBase
	if classifiedBase == 0 {
		classifiedBase = metrics.ResolvedCalls
	}
	if classifiedBase == 0 {
		classifiedBase = metrics.TotalCalls - unclassified
		if classifiedBase < 0 {
			classifiedBase = 0
		}
	}
	switch kind {
	case externalToolShare:
		result.Rate, result.Numerator, result.Denom, result.Population = metrics.ExternalToolShare, metrics.ExternalCalls, classifiedBase, population
	case retryRate:
		result.Rate, result.Numerator, result.Denom = metrics.RetryRate, metrics.RetryCalls, classifiedBase
	case helpRecoveryRate:
		result.Rate, result.Numerator, result.Denom = metrics.HelpRecoveryRate, metrics.HelpRecoveries, classifiedBase
	case repeatedWorkRate:
		result.Rate, result.Numerator, result.Denom = metrics.RepeatedWorkRate, metrics.RepeatedCalls, classifiedBase
	case toolFailureRate:
		result.Rate, result.Numerator, result.Denom = metrics.FailureRate, metrics.FailedCalls, classifiedBase
	default:
		return metricResult{}, fmt.Errorf("unknown friction measure %q", kind)
	}
	return result, nil
}

func filterFromProto(input *measurepb.InvocationFilter, now time.Time) (invocationreadmodel.Filter, error) {
	if input == nil {
		input = &measurepb.InvocationFilter{}
	}
	filter := invocationreadmodel.Filter{Ownership: input.GetOwnership(), Outcome: input.GetOutcome(), Executable: input.GetExecutable(), Fingerprint: input.GetFingerprint(), ProfileID: input.GetProfileId(), RunnerType: input.GetRunnerType(), Model: input.GetModel(), TagPrefix: input.GetTagPrefix(), RunStatus: input.GetRunStatus(), ToolName: input.GetToolName(), WorkloadKey: input.GetWorkloadKey(), ErrorCode: input.GetErrorCode(), EpisodePattern: input.GetEpisodePattern(), EpisodeCauseScope: input.GetEpisodeCauseScope(), EpisodeFingerprint: input.GetEpisodeFingerprint(), SelfReportRuleID: input.GetSelfReportRuleId(), SelfReportCauseScope: input.GetSelfReportCauseScope(), TargetScenario: input.GetTargetScenario(), Operation: input.GetOperation()}
	filter.ExcludedWorkloadKinds = []string{"interactive", "imported"}
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
	store          Store
	now            func() time.Time
	episodeCohort  func(context.Context, invocationreadmodel.Filter, int) (runreport.EpisodeCohort, error)
	validityConfig ValidityConfig
}

// SetEpisodeCohort connects the episode-specific durable projection without
// widening the generic invocation metric store. The callback is supplied by
// orchestration, which owns the episode repository seam.
func (h *Handler) SetEpisodeCohort(fn func(context.Context, invocationreadmodel.Filter, int) (runreport.EpisodeCohort, error)) {
	h.episodeCohort = fn
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
	result, err := execute(ctx, h.store, kind, filter)
	if err != nil {
		return metricResult{}, connect.NewError(connect.CodeInternal, err)
	}
	if kind == externalToolShare || kind == retryRate || kind == helpRecoveryRate || kind == repeatedWorkRate || kind == toolFailureRate {
		metrics, metricErr := h.store.Metrics(ctx, filter)
		if metricErr != nil {
			return metricResult{}, connect.NewError(connect.CodeInternal, metricErr)
		}
		population := metrics.TotalCalls
		if population == 0 {
			population = metrics.ResolvedCalls + metrics.UnknownCalls
		}
		unclassified := metrics.UnclassifiedCalls
		if unclassified == 0 {
			unclassified = metrics.UnknownCalls
		}
		classifiedBase := metrics.ClassifiedBase
		if classifiedBase == 0 {
			classifiedBase = metrics.ResolvedCalls
		}
		if classifiedBase == 0 {
			classifiedBase = population - unclassified
			if classifiedBase < 0 {
				classifiedBase = 0
			}
		}
		sample := result.Denom
		result.Validity = assessValidity(sample, metrics.LargestFingerprintBucket, h.validityConfig)
		result.Validity = withDenominatorValidity(result.Validity, population, classifiedBase, unclassified, h.validityConfig)
	} else {
		result.Validity = assessValidity(result.Denom, 0, h.validityConfig)
	}
	result.Filter = filter
	return result, nil
}

func protoValidity(validity Validity) *measurepb.MeasureValidity {
	return &measurepb.MeasureValidity{State: string(validity.State), Reason: validity.Reason, SampleSize: validity.SampleSize, LargestFingerprintBucket: validity.LargestFingerprintBucket, LargestFingerprintShare: validity.LargestFingerprintShare, ClassifiedBase: validity.ClassifiedBase, UnclassifiedCount: validity.UnclassifiedCount, UnclassifiedShare: validity.UnclassifiedShare, MinimumClassifiedShare: validity.MinimumClassifiedShare}
}

func metricProvenance(name string, result metricResult) *measurepb.MeasureProvenance {
	return provenanceWithQuery(result.Filter, definitionFor(name).SourceTable, result.Validity.SampleSize, result.Query)
}

func (h *Handler) validityForSample(sample int64) *measurepb.MeasureValidity {
	return protoValidity(assessValidity(sample, 0, h.validityConfig))
}

func (h *Handler) ExternalToolShare(ctx context.Context, req *connect.Request[measurepb.ExternalToolShareRequest]) (*connect.Response[measurepb.ExternalToolShareResponse], error) {
	r, err := h.metric(ctx, externalToolShare, req.Msg.GetFilter(), req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&measurepb.ExternalToolShareResponse{Share: r.Rate, ExternalCalls: r.Numerator, ResolvedCalls: r.Denom, UnknownCalls: r.Unknown, Validity: protoValidity(r.Validity), Provenance: metricProvenance(ExternalToolShare, r), DefinitionId: definitionID(ExternalToolShare)}), nil
}

func (h *Handler) RetryRate(ctx context.Context, req *connect.Request[measurepb.RetryRateRequest]) (*connect.Response[measurepb.RetryRateResponse], error) {
	r, err := h.metric(ctx, retryRate, req.Msg.GetFilter(), req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&measurepb.RetryRateResponse{Rate: r.Rate, RetryCalls: r.Numerator, TotalCalls: r.Denom, Validity: protoValidity(r.Validity), Provenance: metricProvenance(RetryRate, r), DefinitionId: definitionID(RetryRate)}), nil
}

func (h *Handler) HelpRecoveryRate(ctx context.Context, req *connect.Request[measurepb.HelpRecoveryRateRequest]) (*connect.Response[measurepb.HelpRecoveryRateResponse], error) {
	r, err := h.metric(ctx, helpRecoveryRate, req.Msg.GetFilter(), req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&measurepb.HelpRecoveryRateResponse{Rate: r.Rate, HelpRecoveries: r.Numerator, TotalCalls: r.Denom, Validity: protoValidity(r.Validity), Provenance: metricProvenance(HelpRecoveryRate, r), DefinitionId: definitionID(HelpRecoveryRate)}), nil
}

func (h *Handler) RepeatedWorkRate(ctx context.Context, req *connect.Request[measurepb.RepeatedWorkRateRequest]) (*connect.Response[measurepb.RepeatedWorkRateResponse], error) {
	r, err := h.metric(ctx, repeatedWorkRate, req.Msg.GetFilter(), req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&measurepb.RepeatedWorkRateResponse{Rate: r.Rate, RepeatedCalls: r.Numerator, TotalCalls: r.Denom, Validity: protoValidity(r.Validity), Provenance: metricProvenance(RepeatedWorkRate, r), DefinitionId: definitionID(RepeatedWorkRate)}), nil
}

func (h *Handler) ToolFailureRate(ctx context.Context, req *connect.Request[measurepb.ToolFailureRateRequest]) (*connect.Response[measurepb.ToolFailureRateResponse], error) {
	r, err := h.metric(ctx, toolFailureRate, req.Msg.GetFilter(), req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&measurepb.ToolFailureRateResponse{Rate: r.Rate, FailedCalls: r.Numerator, TotalCalls: r.Denom, Validity: protoValidity(r.Validity), Provenance: metricProvenance(ToolFailureRate, r), DefinitionId: definitionID(ToolFailureRate)}), nil
}

func (h *Handler) RunSuccessRate(ctx context.Context, req *connect.Request[measurepb.RunSuccessRateRequest]) (*connect.Response[measurepb.RunSuccessRateResponse], error) {
	r, err := h.metric(ctx, runSuccessRate, req.Msg.GetFilter(), req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&measurepb.RunSuccessRateResponse{Rate: r.Rate, SuccessfulRuns: r.Numerator, TerminalRuns: r.Denom, Validity: protoValidity(r.Validity), Provenance: metricProvenance(RunSuccessRate, r), DefinitionId: definitionID(RunSuccessRate)}), nil
}

func (h *Handler) RunCycleTime(ctx context.Context, req *connect.Request[measurepb.RunCycleTimeRequest]) (*connect.Response[measurepb.RunCycleTimeResponse], error) {
	r, err := h.metric(ctx, runCycleTime, req.Msg.GetFilter(), req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&measurepb.RunCycleTimeResponse{AverageDurationMs: r.Rate, CompletedDurationRuns: r.Numerator, Validity: protoValidity(r.Validity), Provenance: metricProvenance(RunCycleTime, r), DefinitionId: definitionID(RunCycleTime)}), nil
}

func (h *Handler) RunCost(ctx context.Context, req *connect.Request[measurepb.RunCostRequest]) (*connect.Response[measurepb.RunCostResponse], error) {
	if err := validateSubscriptionAllocation(req.Msg.GetAllocateSubscription(), req.Msg.GetAllocationBasis()); err != nil {
		return nil, err
	}
	filter, err := filterWithWindow(req.Msg.GetFilter(), req.Msg.GetWindow(), h.now())
	if err != nil {
		return nil, err
	}
	runs, err := h.store.RunMetrics(ctx, filter)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	query := fmt.Sprintf("SELECT durable cost aggregate FROM invocation_read_model_runs WHERE occurred_at >= %q AND occurred_at < %q", filter.From.UTC().Format(time.RFC3339Nano), filter.To.UTC().Format(time.RFC3339Nano))
	validity := h.validityForSample(runs.TotalRuns)
	response := &measurepb.RunCostResponse{TotalCostUsd: runs.TotalCostUSD, AverageCostUsd: runs.AverageCostUSD, TotalRuns: runs.TotalRuns, TotalTokens: runs.TotalTokens, InputTokens: runs.InputTokens, OutputTokens: runs.OutputTokens, CacheReadTokens: runs.CacheReadTokens, CacheCreationTokens: runs.CacheCreationTokens, InputCostUsd: runs.InputCostUSD, OutputCostUsd: runs.OutputCostUSD, CacheReadCostUsd: runs.CacheReadCostUSD, CacheCreationCostUsd: runs.CacheCreationCostUSD, TotalChargeMicroUsd: runs.TotalChargeMicroUSD, UnpricedTokenCount: runs.UnpricedTokenCount, Validity: validity, Provenance: provenanceWithQuery(filter, definitionFor(RunCost).SourceTable, runs.TotalRuns, query), DefinitionId: definitionID(RunCost)}
	if chargeStore, ok := h.store.(interface {
		ChargeByBasis(context.Context, invocationreadmodel.Filter) ([]invocationreadmodel.ChargeByBasis, error)
	}); ok {
		charges, chargeErr := chargeStore.ChargeByBasis(ctx, filter)
		if chargeErr != nil {
			return nil, connect.NewError(connect.CodeInternal, chargeErr)
		}
		for _, charge := range charges {
			response.ChargeByBasis = append(response.ChargeByBasis, &measurepb.ChargeByBasis{Basis: charge.Basis, RunCount: charge.RunCount, ChargeMicroUsd: charge.ChargeMicroUSD, TokenCount: charge.TokenCount, ChargeReason: charge.ChargeReason})
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
	response := &measurepb.RunVolumeResponse{TotalRuns: r.Numerator, TerminalRuns: r.Denom, Validity: protoValidity(r.Validity), Provenance: metricProvenance(RunVolume, r), DefinitionId: definitionID(RunVolume)}
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

func (h *Handler) CapabilityUsage(ctx context.Context, req *connect.Request[measurepb.CapabilityUsageRequest]) (*connect.Response[measurepb.CapabilityUsageResponse], error) {
	filter, err := filterWithWindow(nil, req.Msg.GetWindow(), h.now())
	if req.Msg.GetFilter() != nil {
		filter, err = filterWithWindow(req.Msg.GetFilter(), req.Msg.GetWindow(), h.now())
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	capabilityStore, ok := h.store.(interface {
		CapabilityUsage(context.Context, invocationreadmodel.Filter, int) ([]invocationreadmodel.CapabilityUsageRow, error)
	})
	if !ok {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("receipt capability usage is unavailable"))
	}
	rows, err := capabilityStore.CapabilityUsage(ctx, filter, 100)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	protoRows := make([]*measurepb.CapabilityUsageRow, 0, len(rows))
	var sample int64
	for _, row := range rows {
		sample += row.CallCount
		protoRows = append(protoRows, &measurepb.CapabilityUsageRow{TargetScenario: row.TargetScenario, Operation: row.Operation, CallCount: row.CallCount, SuccessCount: row.SuccessCount, FailedCount: row.FailedCount, TotalDurationMs: row.TotalDurationMS, TotalTokens: row.TotalTokens, EstimatedTokenShare: row.EstimatedTokenShare})
	}
	validity := assessValidity(sample, 0, h.validityConfig)
	if sample == 0 {
		validity.Availability = availability.New(availability.Unavailable, "no verified receipt exists for the filtered population")
	}
	query := "SELECT target_scenario, operation, outcome, duration_ms FROM investigation_cross_scenario_calls WHERE verified = 1 AND target_scenario <> ''"
	return connect.NewResponse(&measurepb.CapabilityUsageResponse{Rows: protoRows, Validity: protoValidity(validity), Provenance: provenanceWithQuery(filter, definitionFor("friction.capability_usage").SourceTable, sample, query), DefinitionId: definitionID("friction.capability_usage")}), nil
}

func (h *Handler) CapabilityEfficacy(ctx context.Context, req *connect.Request[measurepb.CapabilityEfficacyRequest]) (*connect.Response[measurepb.CapabilityEfficacyResponse], error) {
	filter, err := filterWithWindow(req.Msg.GetFilter(), req.Msg.GetWindow(), h.now())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	efficacyStore, ok := h.store.(interface {
		CapabilityEfficacy(context.Context, invocationreadmodel.Filter, int) ([]invocationreadmodel.CapabilityEfficacyRow, error)
	})
	if !ok {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("receipt capability efficacy is unavailable"))
	}
	rows, err := efficacyStore.CapabilityEfficacy(ctx, filter, 100)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	protoRows := make([]*measurepb.CapabilityEfficacyRow, 0, len(rows))
	var sample int64
	for _, row := range rows {
		sample += row.CallCount
		protoRows = append(protoRows, &measurepb.CapabilityEfficacyRow{TargetScenario: row.TargetScenario, Operation: row.Operation, CallCount: row.CallCount, SuccessCount: row.SuccessCount, FallbackAfterCount: row.FallbackAfterCount, AbandonedCount: row.AbandonedCount})
	}
	validity := assessValidity(sample, 0, h.validityConfig)
	if sample == 0 {
		validity.Availability = availability.New(availability.Unavailable, "no verified receipt exists for the filtered population")
	}
	query := "SELECT receipt calls joined to fallback and abandoned episode projections"
	return connect.NewResponse(&measurepb.CapabilityEfficacyResponse{Rows: protoRows, Validity: protoValidity(validity), Provenance: provenanceWithQuery(filter, definitionFor("friction.capability_efficacy").SourceTable, sample, query), DefinitionId: definitionID("friction.capability_efficacy")}), nil
}

// SelectCohort preserves the aggregate-to-run drill-down without reopening
// the prunable event stream. It intentionally shares filter parsing with every
// measure, so the listed runs are the population the displayed value describes.
func (h *Handler) SelectCohort(ctx context.Context, req *connect.Request[measurepb.SelectCohortRequest]) (*connect.Response[measurepb.SelectCohortResponse], error) {
	filter, err := filterWithWindow(req.Msg.GetFilter(), req.Msg.GetWindow(), h.now())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	limit := int(req.Msg.GetLimit())
	if limit <= 0 {
		limit = 25
	}
	cohortStore, ok := h.store.(interface {
		Cohort(context.Context, invocationreadmodel.Filter, int) (invocationreadmodel.Cohort, error)
	})
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("measure store does not provide durable cohort selection"))
	}
	cohort, err := cohortStore.Cohort(ctx, filter, limit)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	rows := make([]*measurepb.CohortRun, 0, len(cohort.RunIDs))
	if rowStore, ok := h.store.(interface {
		CohortRows(context.Context, []string, string) ([]invocationreadmodel.CohortRun, error)
	}); ok {
		cohortRows, rowErr := rowStore.CohortRows(ctx, cohort.RunIDs, filter.ToolName)
		if rowErr != nil {
			return nil, connect.NewError(connect.CodeInternal, rowErr)
		}
		for _, row := range cohortRows {
			protoRow := &measurepb.CohortRun{RunId: row.RunID, TaskTitle: row.TaskTitle, ProfileId: row.ProfileID, ProfileName: row.ProfileName, Status: row.Status, CreatedAt: row.CreatedAt.UTC().Format(time.RFC3339), Model: row.Model, RunnerType: row.RunnerType, WorkloadKey: row.WorkloadKey, TotalTokens: row.TotalTokens}
			if row.TotalChargeMicroUSD != nil {
				protoRow.TotalChargeMicroUsd = row.TotalChargeMicroUSD
			}
			if row.ChargeBasis != nil {
				protoRow.ChargeBasis = row.ChargeBasis
			}
			if row.ToolCallCount != nil {
				protoRow.ToolCallCount = row.ToolCallCount
			}
			rows = append(rows, protoRow)
		}
	}
	query := fmt.Sprintf("SELECT DISTINCT run_id FROM invocation_read_model_facts WHERE occurred_at >= %q AND occurred_at < %q ORDER BY run_id LIMIT %d", filter.From.UTC().Format(time.RFC3339Nano), filter.To.UTC().Format(time.RFC3339Nano), limit)
	return connect.NewResponse(&measurepb.SelectCohortResponse{RunIds: cohort.RunIDs, Rows: rows, Truncated: cohort.Truncated, Validity: h.validityForSample(int64(len(cohort.RunIDs))), Provenance: provenanceWithQuery(filter, definitionFor("select_cohort").SourceTable, int64(len(cohort.RunIDs)), query), DefinitionId: definitionID("select_cohort")}), nil
}

// EpisodeCohort exposes the ranked friction-episode investigation projection
// through the typed measures surface. It shares SelectCohort's predicate and
// bounded limit, then folds only persisted episode records.
func (h *Handler) EpisodeCohort(ctx context.Context, req *connect.Request[measurepb.EpisodeCohortRequest]) (*connect.Response[measurepb.EpisodeCohortResponse], error) {
	if h.episodeCohort == nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("episode cohort source is not configured"))
	}
	filter, err := filterWithWindow(req.Msg.GetFilter(), req.Msg.GetWindow(), h.now())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	limit := int(req.Msg.GetLimit())
	if limit <= 0 {
		limit = 25
	}
	cohort, err := h.episodeCohort(ctx, filter, limit)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	query := fmt.Sprintf("SELECT persisted friction episodes FROM selected durable run cohort LIMIT %d", limit)
	response := &measurepb.EpisodeCohortResponse{AvailabilityState: string(cohort.Availability.State), AvailabilityReason: cohort.Availability.Reason}
	for _, signal := range cohort.Signals {
		response.Signals = append(response.Signals, &measurepb.EpisodeCohortSignal{Fingerprint: signal.Fingerprint, Occurrences: int64(signal.Occurrences), DistinctRuns: int64(signal.DistinctRuns), SummedCostMs: signal.SummedCostMS, Confidence: signal.Confidence, RepresentativeRunIds: append([]string(nil), signal.RepresentativeRunIDs...)})
	}
	response.Validity = h.validityForSample(int64(len(cohort.Signals)))
	response.Provenance = provenanceWithQuery(filter, definitionFor("episode_cohort").SourceTable, int64(len(cohort.Signals)), query)
	response.DefinitionId = definitionID("episode_cohort")
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
	response.Provenance = provenanceWithQuery(filter, definitionFor(RunStatusDistribution).SourceTable, sample, query)
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
	return connect.NewResponse(&measurepb.RunDurationStatisticsResponse{AverageDurationMs: stats.AverageDurationMS, P50DurationMs: stats.P50DurationMS, P95DurationMs: stats.P95DurationMS, P99DurationMs: stats.P99DurationMS, MinDurationMs: stats.MinDurationMS, MaxDurationMs: stats.MaxDurationMS, Count: stats.Count, Validity: h.validityForSample(stats.Count), Provenance: provenanceWithQuery(filter, definitionFor("throughput.run_duration_statistics").SourceTable, stats.Count, query), DefinitionId: definitionID("throughput.run_duration_statistics")}), nil
}

func (h *Handler) runBreakdownRows(ctx context.Context, input *measurepb.InvocationFilter, window *sharedmeasurepb.TimeWindow, dimension string) ([]invocationreadmodel.RunBreakdownRow, string, invocationreadmodel.Filter, error) {
	filter, err := filterWithWindow(input, window, h.now())
	if err != nil {
		return nil, "", invocationreadmodel.Filter{}, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if dimension == "model" && filter.WorkloadKind == "" {
		// Interactive and imported runs are valid analytical populations, but
		// their model assignment has different provenance. Keep the default
		// model measure reproducible by excluding them; callers can request
		// either class explicitly through workload_kind.
		filter.ExcludedWorkloadKinds = []string{"interactive", "imported"}
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
		out = append(out, &measurepb.RunBreakdownRow{Key: row.Key, Value: row.Value, RunCount: row.RunCount, SuccessCount: row.SuccessCount, FailedCount: row.FailedCount, TotalCostUsd: row.TotalCostUSD, TotalTokens: row.TotalTokens, AverageDurationMs: row.AvgDurationMS, TotalChargeMicroUsd: row.TotalChargeMicroUSD, ConsumptionPerSuccessfulCompletion: row.ConsumptionPerSuccessfulCompletion, CompletionRate: row.CompletionRate})
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
	return connect.NewResponse(&measurepb.RunnerBreakdownResponse{Rows: protoBreakdownRows(rows), Validity: h.validityForSample(sumBreakdownRuns(rows)), Provenance: provenanceWithQuery(filter, definitionFor(RunnerBreakdown).SourceTable, sumBreakdownRuns(rows), query), DefinitionId: definitionID(RunnerBreakdown)}), nil
}

func (h *Handler) ModelBreakdown(ctx context.Context, req *connect.Request[measurepb.ModelBreakdownRequest]) (*connect.Response[measurepb.ModelBreakdownResponse], error) {
	rows, query, filter, err := h.runBreakdownRows(ctx, req.Msg.GetFilter(), req.Msg.GetWindow(), "model")
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&measurepb.ModelBreakdownResponse{Rows: protoBreakdownRows(rows), Validity: h.validityForSample(sumBreakdownRuns(rows)), Provenance: provenanceWithQuery(filter, definitionFor(ModelBreakdown).SourceTable, sumBreakdownRuns(rows), query), DefinitionId: definitionID(ModelBreakdown)}), nil
}

func (h *Handler) ProfileBreakdown(ctx context.Context, req *connect.Request[measurepb.ProfileBreakdownRequest]) (*connect.Response[measurepb.ProfileBreakdownResponse], error) {
	rows, query, filter, err := h.runBreakdownRows(ctx, req.Msg.GetFilter(), req.Msg.GetWindow(), "profile")
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&measurepb.ProfileBreakdownResponse{Rows: protoBreakdownRows(rows), Validity: h.validityForSample(sumBreakdownRuns(rows)), Provenance: provenanceWithQuery(filter, definitionFor(ProfileBreakdown).SourceTable, sumBreakdownRuns(rows), query), DefinitionId: definitionID(ProfileBreakdown)}), nil
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
	return connect.NewResponse(&measurepb.WorkloadBreakdownResponse{Rows: protoBreakdownRows(rows), Validity: h.validityForSample(sample), Provenance: provenanceWithQuery(filter, definitionFor(WorkloadBreakdown).SourceTable, sample, query), DefinitionId: definitionID(WorkloadBreakdown)}), nil
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
	validity := assessValidity(runs.TerminalRuns, 0, h.validityConfig)
	query := fmt.Sprintf("SELECT workload efficiency from invocation_read_model_runs WHERE occurred_at >= %q AND occurred_at < %q", filter.From.UTC().Format(time.RFC3339Nano), filter.To.UTC().Format(time.RFC3339Nano))
	response := &measurepb.WorkloadEfficiencyResponse{
		ConsumptionPerSuccessfulCompletion: runs.ConsumptionPerSuccessfulCompletion,
		CompletionRate:                     runs.SuccessRate,
		TotalTokens:                        runs.TotalTokens,
		TerminalRuns:                       runs.TerminalRuns,
		SuccessfulRuns:                     runs.SuccessfulRuns,
		ObservationalLimitation:            "model assignment was observational, not randomized",
		Validity:                           protoValidity(validity),
		Provenance:                         provenanceWithQuery(filter, definitionFor(WorkloadEfficiency).SourceTable, runs.TerminalRuns, query),
		DefinitionId:                       definitionID(WorkloadEfficiency),
	}
	return connect.NewResponse(response), nil
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
		response.Rows = append(response.Rows, &measurepb.TerminalRunTrendRow{Bucket: row.Bucket.UTC().Format(time.RFC3339), TerminalRuns: row.TerminalRuns, CompletedRuns: row.CompletedRuns, FailedRuns: row.FailedRuns, CancelledRuns: row.CancelledRuns, TotalCostUsd: row.TotalCostUSD, AverageDurationMs: row.AvgDurationMS})
	}
	var sample int64
	for _, row := range rows {
		sample += row.TerminalRuns
	}
	response.Validity = h.validityForSample(sample)
	response.Provenance = provenanceWithQuery(filter, definitionFor(TerminalRunTrend).SourceTable, sample, query)
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
	response.Provenance = provenanceWithQuery(filter, definitionFor(ToolUsage).SourceTable, sample, query)
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
	response.Provenance = provenanceWithQuery(filter, definitionFor("friction.tool_command_breakdown").SourceTable, sample, query)
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
	response.Provenance = provenanceWithQuery(filter, definitionFor(TokenAttribution).SourceTable, sample, query)
	response.DefinitionId = definitionID(TokenAttribution)
	return connect.NewResponse(response), nil
}

func (h *Handler) ErrorPatterns(ctx context.Context, req *connect.Request[measurepb.ErrorPatternsRequest]) (*connect.Response[measurepb.ErrorPatternsResponse], error) {
	filter, err := filterWithWindow(req.Msg.GetFilter(), req.Msg.GetWindow(), h.now())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	rows, err := h.store.ErrorPatterns(ctx, filter, 20)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	query := fmt.Sprintf("SELECT error_code aggregate FROM invocation_read_model_errors WHERE occurred_at >= %q AND occurred_at < %q GROUP BY error_code LIMIT 20", filter.From.UTC().Format(time.RFC3339Nano), filter.To.UTC().Format(time.RFC3339Nano))
	response := &measurepb.ErrorPatternsResponse{Rows: make([]*measurepb.ErrorPatternRow, 0, len(rows))}
	for _, row := range rows {
		response.Rows = append(response.Rows, &measurepb.ErrorPatternRow{ErrorCode: row.ErrorCode, Count: row.Count, LastSeen: row.LastSeen.UTC().Format(time.RFC3339), SampleRunId: row.SampleRunID})
	}
	var sample int64
	for _, row := range rows {
		sample += row.Count
	}
	response.Validity = h.validityForSample(sample)
	response.Provenance = provenanceWithQuery(filter, definitionFor(ErrorPatterns).SourceTable, sample, query)
	response.DefinitionId = definitionID(ErrorPatterns)
	return connect.NewResponse(response), nil
}

func (h *Handler) FileRereadRate(ctx context.Context, req *connect.Request[measurepb.FileRereadRateRequest]) (*connect.Response[measurepb.FileRereadRateResponse], error) {
	r, err := h.metric(ctx, fileRereadRate, req.Msg.GetFilter(), req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&measurepb.FileRereadRateResponse{Rate: r.Rate, FilesReadMoreThanOnce: r.Numerator, ReadCalls: r.Denom, Validity: protoValidity(r.Validity), Provenance: metricProvenance(FileRereadRate, r), DefinitionId: definitionID(FileRereadRate)}), nil
}

func (h *Handler) FindingRecurrenceRate(ctx context.Context, req *connect.Request[measurepb.FindingRecurrenceRateRequest]) (*connect.Response[measurepb.FindingRecurrenceRateResponse], error) {
	r, err := h.metric(ctx, findingRecurrenceRate, req.Msg.GetFilter(), req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&measurepb.FindingRecurrenceRateResponse{Rate: r.Rate, RecurringFindings: r.Numerator, TotalFindings: r.Denom, RecurringFingerprints: r.Secondary, Validity: protoValidity(r.Validity), Provenance: metricProvenance(FindingRecurrenceRate, r), DefinitionId: definitionID(FindingRecurrenceRate)}), nil
}

func (h *Handler) Registry() (*measurelib.Registry, error) {
	registry := measurelib.NewRegistry(measurelib.WithClock(h.now))
	for _, spec := range declarations() {
		spec := spec
		if err := registry.Register(declaration(spec), func(ctx context.Context, request measurelib.MeasureRequest) (measurelib.MeasureResult, error) {
			rangeValue, err := measurelib.ResolveToken(measurelib.TimeWindowToken(request.Params["window"]), h.now(), time.UTC)
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
			result, err := execute(ctx, h.store, spec.kind, filter)
			if err != nil {
				return measurelib.MeasureResult{}, err
			}
			return measurelib.MeasureResult{Value: strconv.FormatFloat(result.Rate, 'f', -1, 64), Provenance: measurelib.Provenance{ExecutedQuery: result.Query}}, nil
		}); err != nil {
			return nil, err
		}
	}
	if err := registry.Register(measurelib.MeasureDeclaration{
		Name: RunStatusDistribution, Scenario: "agent-manager", Domain: "run", Intent: "Distribution of durable terminal run summaries by status.",
		Questions: []string{"what is the agent run status distribution this week", "how many agent runs are complete, failed, or cancelled"},
		Params:    map[string]measurelib.Param{"window": {Name: "window", Type: measurelib.ParamTypeTimeWindow, Default: string(measurelib.TokenThisWeek)}},
		Result:    measurelib.Result{Kind: measurelib.ResultTable, ValueField: "status", Unit: "runs", SummaryTemplate: "run status distribution ({window})"},
		Effect:    measurelib.EffectRead, RunEligible: true, Service: "MeasuresService", Method: "RunStatusDistribution",
	}, func(ctx context.Context, request measurelib.MeasureRequest) (measurelib.MeasureResult, error) {
		rangeValue, err := measurelib.ResolveToken(measurelib.TimeWindowToken(request.Params["window"]), h.now(), time.UTC)
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
			rangeValue, err := measurelib.ResolveToken(measurelib.TimeWindowToken(request.Params["window"]), h.now(), time.UTC)
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
		rangeValue, err := measurelib.ResolveToken(measurelib.TimeWindowToken(request.Params["window"]), h.now(), time.UTC)
		if err != nil {
			return measurelib.MeasureResult{}, err
		}
		filter := invocationreadmodel.Filter{From: &rangeValue.From, To: &rangeValue.To, WorkloadKey: request.Params["workload_key"], RunnerType: request.Params["runner_type"], Model: request.Params["model"]}
		runs, err := h.store.RunMetrics(ctx, filter)
		if err != nil {
			return measurelib.MeasureResult{}, err
		}
		validity := "reliable"
		if runs.TerminalRuns < 5 {
			validity = "unreliable"
		}
		query := fmt.Sprintf("SELECT SUM(total_tokens) / successful completions FROM invocation_read_model_runs WHERE workload_key = %q AND occurred_at >= %q AND occurred_at < %q", filter.WorkloadKey, rangeValue.From.UTC().Format(time.RFC3339Nano), rangeValue.To.UTC().Format(time.RFC3339Nano))
		return measurelib.MeasureResult{
			Value:      strconv.FormatFloat(runs.ConsumptionPerSuccessfulCompletion, 'f', -1, 64),
			Fields:     []map[string]string{{"validity": validity, "terminal_runs": strconv.FormatInt(runs.TerminalRuns, 10), "successful_runs": strconv.FormatInt(runs.SuccessfulRuns, 10), "observational_limitation": "model assignment was not randomised"}},
			Provenance: measurelib.Provenance{ExecutedQuery: query},
		}, nil
	}); err != nil {
		return nil, err
	}
	if err := registry.Register(measurelib.MeasureDeclaration{Name: TerminalRunTrend, Scenario: "agent-manager", Domain: "run", Intent: "Hourly durable terminal-run outcomes, cost, and duration trends.", Questions: []string{"show the agent terminal run trend this week", "how are agent run completions and failures trending"}, Params: map[string]measurelib.Param{"window": {Name: "window", Type: measurelib.ParamTypeTimeWindow, Default: string(measurelib.TokenThisWeek)}}, Result: measurelib.Result{Kind: measurelib.ResultTable, ValueField: "bucket", Unit: "runs", SummaryTemplate: "terminal run trend ({window})"}, Effect: measurelib.EffectRead, RunEligible: true, Service: "MeasuresService", Method: "TerminalRunTrend"}, func(ctx context.Context, request measurelib.MeasureRequest) (measurelib.MeasureResult, error) {
		rangeValue, err := measurelib.ResolveToken(measurelib.TimeWindowToken(request.Params["window"]), h.now(), time.UTC)
		if err != nil {
			return measurelib.MeasureResult{}, err
		}
		rows, err := h.store.RunTimeSeries(ctx, invocationreadmodel.Filter{From: &rangeValue.From, To: &rangeValue.To}, time.Hour)
		if err != nil {
			return measurelib.MeasureResult{}, err
		}
		fields := make([]map[string]string, 0, len(rows))
		for _, row := range rows {
			fields = append(fields, map[string]string{"bucket": row.Bucket.UTC().Format(time.RFC3339), "terminal_runs": strconv.FormatInt(row.TerminalRuns, 10), "completed_runs": strconv.FormatInt(row.CompletedRuns, 10), "failed_runs": strconv.FormatInt(row.FailedRuns, 10), "cancelled_runs": strconv.FormatInt(row.CancelledRuns, 10), "total_cost_usd": strconv.FormatFloat(row.TotalCostUSD, 'f', -1, 64), "average_duration_ms": strconv.FormatFloat(row.AvgDurationMS, 'f', -1, 64)})
		}
		query := fmt.Sprintf("SELECT terminal outcome aggregates FROM invocation_read_model_runs WHERE occurred_at >= %q AND occurred_at < %q GROUP BY hourly terminal bucket", rangeValue.From.UTC().Format(time.RFC3339Nano), rangeValue.To.UTC().Format(time.RFC3339Nano))
		return measurelib.MeasureResult{Fields: fields, Provenance: measurelib.Provenance{ExecutedQuery: query}}, nil
	}); err != nil {
		return nil, err
	}
	if err := registry.Register(measurelib.MeasureDeclaration{Name: ToolUsage, Scenario: "agent-manager", Domain: "friction", Intent: "Durable tool invocation counts and classified outcomes by tool.", Questions: []string{"which tools do agents use most", "agent tool usage this week"}, Params: map[string]measurelib.Param{"window": {Name: "window", Type: measurelib.ParamTypeTimeWindow, Default: string(measurelib.TokenThisWeek)}}, Result: measurelib.Result{Kind: measurelib.ResultTable, ValueField: "tool_name", Unit: "calls", SummaryTemplate: "tool usage ({window})"}, Effect: measurelib.EffectRead, RunEligible: true, Service: "MeasuresService", Method: "ToolUsage"}, func(ctx context.Context, request measurelib.MeasureRequest) (measurelib.MeasureResult, error) {
		rangeValue, err := measurelib.ResolveToken(measurelib.TimeWindowToken(request.Params["window"]), h.now(), time.UTC)
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
	if err := registry.Register(measurelib.MeasureDeclaration{Name: "friction.capability_usage", Scenario: "agent-manager", Domain: "friction", Intent: "Receipt-backed project capability calls, outcomes, and duration.", Questions: []string{"which project capabilities do agents use", "how effective are project-owned capability calls"}, Params: map[string]measurelib.Param{"window": {Name: "window", Type: measurelib.ParamTypeTimeWindow, Default: string(measurelib.TokenThisWeek)}}, Result: measurelib.Result{Kind: measurelib.ResultTable, ValueField: "target_scenario", Unit: "calls", SummaryTemplate: "capability usage ({window})"}, Effect: measurelib.EffectRead, RunEligible: true, Service: "MeasuresService", Method: "CapabilityUsage"}, func(ctx context.Context, request measurelib.MeasureRequest) (measurelib.MeasureResult, error) {
		rangeValue, err := measurelib.ResolveToken(measurelib.TimeWindowToken(request.Params["window"]), h.now(), time.UTC)
		if err != nil {
			return measurelib.MeasureResult{}, err
		}
		capabilityStore, ok := h.store.(interface {
			CapabilityUsage(context.Context, invocationreadmodel.Filter, int) ([]invocationreadmodel.CapabilityUsageRow, error)
		})
		if !ok {
			return measurelib.MeasureResult{}, fmt.Errorf("receipt capability usage is unavailable")
		}
		rows, err := capabilityStore.CapabilityUsage(ctx, invocationreadmodel.Filter{From: &rangeValue.From, To: &rangeValue.To}, 100)
		if err != nil {
			return measurelib.MeasureResult{}, err
		}
		fields := make([]map[string]string, 0, len(rows))
		for _, row := range rows {
			fields = append(fields, map[string]string{"target_scenario": row.TargetScenario, "operation": row.Operation, "call_count": strconv.FormatInt(row.CallCount, 10), "success_count": strconv.FormatInt(row.SuccessCount, 10), "failed_count": strconv.FormatInt(row.FailedCount, 10), "total_duration_ms": strconv.FormatUint(row.TotalDurationMS, 10)})
		}
		query := fmt.Sprintf("SELECT target_scenario, operation, outcome, duration_ms FROM investigation_cross_scenario_calls WHERE verified = 1 AND occurred_at >= %q AND occurred_at < %q GROUP BY target_scenario, operation", rangeValue.From.UTC().Format(time.RFC3339Nano), rangeValue.To.UTC().Format(time.RFC3339Nano))
		return measurelib.MeasureResult{Fields: fields, Provenance: measurelib.Provenance{ExecutedQuery: query}}, nil
	}); err != nil {
		return nil, err
	}
	if err := registry.Register(measurelib.MeasureDeclaration{Name: "friction.capability_efficacy", Scenario: "agent-manager", Domain: "friction", Intent: "Receipt-backed capability success, fallback, and abandonment counts.", Questions: []string{"which capabilities work for agents", "where do capability calls lead to fallback"}, Params: map[string]measurelib.Param{"window": {Name: "window", Type: measurelib.ParamTypeTimeWindow, Default: string(measurelib.TokenThisWeek)}}, Result: measurelib.Result{Kind: measurelib.ResultTable, ValueField: "target_scenario", Unit: "calls", SummaryTemplate: "capability efficacy ({window})"}, Effect: measurelib.EffectRead, RunEligible: true, Service: "MeasuresService", Method: "CapabilityEfficacy"}, func(ctx context.Context, request measurelib.MeasureRequest) (measurelib.MeasureResult, error) {
		rangeValue, err := measurelib.ResolveToken(measurelib.TimeWindowToken(request.Params["window"]), h.now(), time.UTC)
		if err != nil {
			return measurelib.MeasureResult{}, err
		}
		efficacyStore, ok := h.store.(interface {
			CapabilityEfficacy(context.Context, invocationreadmodel.Filter, int) ([]invocationreadmodel.CapabilityEfficacyRow, error)
		})
		if !ok {
			return measurelib.MeasureResult{}, fmt.Errorf("receipt capability efficacy is unavailable")
		}
		rows, err := efficacyStore.CapabilityEfficacy(ctx, invocationreadmodel.Filter{From: &rangeValue.From, To: &rangeValue.To}, 100)
		if err != nil {
			return measurelib.MeasureResult{}, err
		}
		fields := make([]map[string]string, 0, len(rows))
		for _, row := range rows {
			fields = append(fields, map[string]string{"target_scenario": row.TargetScenario, "operation": row.Operation, "call_count": strconv.FormatInt(row.CallCount, 10), "success_count": strconv.FormatInt(row.SuccessCount, 10), "fallback_after_count": strconv.FormatInt(row.FallbackAfterCount, 10), "abandoned_count": strconv.FormatInt(row.AbandonedCount, 10)})
		}
		return measurelib.MeasureResult{Fields: fields, Provenance: measurelib.Provenance{ExecutedQuery: "SELECT receipt calls joined to fallback and abandoned episode projections"}}, nil
	}); err != nil {
		return nil, err
	}
	if err := registry.Register(measurelib.MeasureDeclaration{Name: ErrorPatterns, Scenario: "agent-manager", Domain: "friction", Intent: "Durable error-code frequency and recent sample runs.", Questions: []string{"what agent errors recur this week", "show agent error patterns"}, Params: map[string]measurelib.Param{"window": {Name: "window", Type: measurelib.ParamTypeTimeWindow, Default: string(measurelib.TokenThisWeek)}}, Result: measurelib.Result{Kind: measurelib.ResultTable, ValueField: "error_code", Unit: "errors", SummaryTemplate: "error patterns ({window})"}, Effect: measurelib.EffectRead, RunEligible: true, Service: "MeasuresService", Method: "ErrorPatterns"}, func(ctx context.Context, request measurelib.MeasureRequest) (measurelib.MeasureResult, error) {
		rangeValue, err := measurelib.ResolveToken(measurelib.TimeWindowToken(request.Params["window"]), h.now(), time.UTC)
		if err != nil {
			return measurelib.MeasureResult{}, err
		}
		rows, err := h.store.ErrorPatterns(ctx, invocationreadmodel.Filter{From: &rangeValue.From, To: &rangeValue.To}, 20)
		if err != nil {
			return measurelib.MeasureResult{}, err
		}
		fields := make([]map[string]string, 0, len(rows))
		for _, row := range rows {
			fields = append(fields, map[string]string{"error_code": row.ErrorCode, "count": strconv.FormatInt(row.Count, 10), "last_seen": row.LastSeen.UTC().Format(time.RFC3339), "sample_run_id": row.SampleRunID})
		}
		query := fmt.Sprintf("SELECT error_code aggregate FROM invocation_read_model_errors WHERE occurred_at >= %q AND occurred_at < %q GROUP BY error_code LIMIT 20", rangeValue.From.UTC().Format(time.RFC3339Nano), rangeValue.To.UTC().Format(time.RFC3339Nano))
		return measurelib.MeasureResult{Fields: fields, Provenance: measurelib.Provenance{ExecutedQuery: query}}, nil
	}); err != nil {
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

var _ measureconnect.MeasuresServiceHandler = (*Handler)(nil)
