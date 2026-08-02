// Package measures exposes Agent Manager's durable friction analytics through
// one shared compute path for the measures-go registry and Connect RPCs.
package measures

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

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
	TerminalRunTrend      = "throughput.terminal_run_trend"
	ToolUsage             = "friction.tool_usage"
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
)

type metricResult struct {
	Rate      float64
	Numerator int64
	Denom     int64
	Unknown   int64
	Secondary int64
	Query     string
	Validity  Validity
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
		{RunSuccessRate, "RunSuccessRate", "Share of terminal runs that completed successfully.", "share", "{value} run success rate ({window})", []string{"what is the agent run success rate this week", "how often do agent runs complete successfully"}, runSuccessRate},
		{RunCycleTime, "RunCycleTime", "Average completed run cycle time in milliseconds.", "milliseconds", "{value} average run cycle time ({window})", []string{"what is the average agent run cycle time", "how long do completed agent runs take"}, runCycleTime},
		{RunCost, "RunCost", "Total terminal run cost in USD with retained token usage.", "usd", "{value} agent run cost ({window})", []string{"what did agent runs cost this week", "show total agent run cost"}, runCost},
		{RunVolume, "RunVolume", "Number of terminal run summaries in the window.", "runs", "{value} agent run volume ({window})", []string{"how many agent runs completed this week", "show agent run volume"}, runVolume},
		{FileRereadRate, "FileRereadRate", "Share of file-read calls that revisit a path already read in the same run.", "share", "{value} file reread rate ({window})", []string{"what is the agent file reread rate", "how often do agents reread files"}, fileRereadRate},
		{FindingRecurrenceRate, "FindingRecurrenceRate", "Share of persisted investigation findings whose fingerprint recurs in the same filtered finding corpus.", "share", "{value} finding recurrence rate ({window})", []string{"what is the recurring finding rate", "which agent findings recur"}, findingRecurrenceRate},
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
	return measurelib.MeasureDeclaration{
		Name: spec.name, Scenario: "agent-manager", Domain: domain, Intent: spec.intent, Questions: spec.questions,
		Params: map[string]measurelib.Param{"window": {Name: "window", Type: measurelib.ParamTypeTimeWindow, Default: string(measurelib.TokenThisWeek)}},
		Result: measurelib.Result{Kind: measurelib.ResultScalar, ValueField: "value", Unit: spec.unit, SummaryTemplate: spec.summary},
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
	result := metricResult{Unknown: metrics.UnknownCalls, Query: query}
	switch kind {
	case externalToolShare:
		result.Rate, result.Numerator, result.Denom = metrics.ExternalToolShare, metrics.ExternalCalls, metrics.ResolvedCalls
	case retryRate:
		result.Rate, result.Numerator, result.Denom = metrics.RetryRate, metrics.RetryCalls, metrics.TotalCalls
	case helpRecoveryRate:
		result.Rate, result.Numerator, result.Denom = metrics.HelpRecoveryRate, metrics.HelpRecoveries, metrics.TotalCalls
	case repeatedWorkRate:
		result.Rate, result.Numerator, result.Denom = metrics.RepeatedWorkRate, metrics.RepeatedCalls, metrics.TotalCalls
	case toolFailureRate:
		result.Rate, result.Numerator, result.Denom = metrics.FailureRate, metrics.FailedCalls, metrics.TotalCalls
	default:
		return metricResult{}, fmt.Errorf("unknown friction measure %q", kind)
	}
	return result, nil
}

func filterFromProto(input *measurepb.InvocationFilter, now time.Time) (invocationreadmodel.Filter, error) {
	if input == nil {
		input = &measurepb.InvocationFilter{}
	}
	filter := invocationreadmodel.Filter{Ownership: input.GetOwnership(), Outcome: input.GetOutcome(), Executable: input.GetExecutable(), Fingerprint: input.GetFingerprint(), ProfileID: input.GetProfileId(), RunnerType: input.GetRunnerType(), Model: input.GetModel(), TagPrefix: input.GetTagPrefix(), RunStatus: input.GetRunStatus(), ToolName: input.GetToolName(), EpisodePattern: input.GetEpisodePattern(), EpisodeCauseScope: input.GetEpisodeCauseScope(), EpisodeFingerprint: input.GetEpisodeFingerprint(), SelfReportRuleID: input.GetSelfReportRuleId(), SelfReportCauseScope: input.GetSelfReportCauseScope()}
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
		result.Validity = assessValidity(result.Denom, metrics.LargestFingerprintBucket, h.validityConfig)
	} else {
		result.Validity = assessValidity(result.Denom, 0, h.validityConfig)
	}
	return result, nil
}

func protoValidity(validity Validity) *measurepb.MeasureValidity {
	return &measurepb.MeasureValidity{State: string(validity.State), Reason: validity.Reason, SampleSize: validity.SampleSize, LargestFingerprintBucket: validity.LargestFingerprintBucket, LargestFingerprintShare: validity.LargestFingerprintShare}
}

func (h *Handler) validityForSample(sample int64) *measurepb.MeasureValidity {
	return protoValidity(assessValidity(sample, 0, h.validityConfig))
}

func (h *Handler) ExternalToolShare(ctx context.Context, req *connect.Request[measurepb.ExternalToolShareRequest]) (*connect.Response[measurepb.ExternalToolShareResponse], error) {
	r, err := h.metric(ctx, externalToolShare, req.Msg.GetFilter(), req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&measurepb.ExternalToolShareResponse{Share: r.Rate, ExternalCalls: r.Numerator, ResolvedCalls: r.Denom, UnknownCalls: r.Unknown, ExecutedQuery: r.Query, Validity: protoValidity(r.Validity)}), nil
}

func (h *Handler) RetryRate(ctx context.Context, req *connect.Request[measurepb.RetryRateRequest]) (*connect.Response[measurepb.RetryRateResponse], error) {
	r, err := h.metric(ctx, retryRate, req.Msg.GetFilter(), req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&measurepb.RetryRateResponse{Rate: r.Rate, RetryCalls: r.Numerator, TotalCalls: r.Denom, ExecutedQuery: r.Query, Validity: protoValidity(r.Validity)}), nil
}

func (h *Handler) HelpRecoveryRate(ctx context.Context, req *connect.Request[measurepb.HelpRecoveryRateRequest]) (*connect.Response[measurepb.HelpRecoveryRateResponse], error) {
	r, err := h.metric(ctx, helpRecoveryRate, req.Msg.GetFilter(), req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&measurepb.HelpRecoveryRateResponse{Rate: r.Rate, HelpRecoveries: r.Numerator, TotalCalls: r.Denom, ExecutedQuery: r.Query, Validity: protoValidity(r.Validity)}), nil
}

func (h *Handler) RepeatedWorkRate(ctx context.Context, req *connect.Request[measurepb.RepeatedWorkRateRequest]) (*connect.Response[measurepb.RepeatedWorkRateResponse], error) {
	r, err := h.metric(ctx, repeatedWorkRate, req.Msg.GetFilter(), req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&measurepb.RepeatedWorkRateResponse{Rate: r.Rate, RepeatedCalls: r.Numerator, TotalCalls: r.Denom, ExecutedQuery: r.Query, Validity: protoValidity(r.Validity)}), nil
}

func (h *Handler) ToolFailureRate(ctx context.Context, req *connect.Request[measurepb.ToolFailureRateRequest]) (*connect.Response[measurepb.ToolFailureRateResponse], error) {
	r, err := h.metric(ctx, toolFailureRate, req.Msg.GetFilter(), req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&measurepb.ToolFailureRateResponse{Rate: r.Rate, FailedCalls: r.Numerator, TotalCalls: r.Denom, ExecutedQuery: r.Query, Validity: protoValidity(r.Validity)}), nil
}

func (h *Handler) RunSuccessRate(ctx context.Context, req *connect.Request[measurepb.RunSuccessRateRequest]) (*connect.Response[measurepb.RunSuccessRateResponse], error) {
	r, err := h.metric(ctx, runSuccessRate, req.Msg.GetFilter(), req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&measurepb.RunSuccessRateResponse{Rate: r.Rate, SuccessfulRuns: r.Numerator, TerminalRuns: r.Denom, ExecutedQuery: r.Query, Validity: protoValidity(r.Validity)}), nil
}

func (h *Handler) RunCycleTime(ctx context.Context, req *connect.Request[measurepb.RunCycleTimeRequest]) (*connect.Response[measurepb.RunCycleTimeResponse], error) {
	r, err := h.metric(ctx, runCycleTime, req.Msg.GetFilter(), req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&measurepb.RunCycleTimeResponse{AverageDurationMs: r.Rate, CompletedDurationRuns: r.Numerator, ExecutedQuery: r.Query, Validity: protoValidity(r.Validity)}), nil
}

func (h *Handler) RunCost(ctx context.Context, req *connect.Request[measurepb.RunCostRequest]) (*connect.Response[measurepb.RunCostResponse], error) {
	filter, err := filterWithWindow(req.Msg.GetFilter(), req.Msg.GetWindow(), h.now())
	if err != nil {
		return nil, err
	}
	runs, err := h.store.RunMetrics(ctx, filter)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	query := fmt.Sprintf("SELECT durable cost aggregate FROM invocation_read_model_runs WHERE occurred_at >= %q AND occurred_at < %q", filter.From.UTC().Format(time.RFC3339Nano), filter.To.UTC().Format(time.RFC3339Nano))
	return connect.NewResponse(&measurepb.RunCostResponse{TotalCostUsd: runs.TotalCostUSD, AverageCostUsd: runs.AverageCostUSD, TotalRuns: runs.TotalRuns, TotalTokens: runs.TotalTokens, InputTokens: runs.InputTokens, OutputTokens: runs.OutputTokens, CacheReadTokens: runs.CacheReadTokens, CacheCreationTokens: runs.CacheCreationTokens, InputCostUsd: runs.InputCostUSD, OutputCostUsd: runs.OutputCostUSD, CacheReadCostUsd: runs.CacheReadCostUSD, CacheCreationCostUsd: runs.CacheCreationCostUSD, AuthoritativeCostUsd: runs.AuthoritativeCostUSD, EstimatedCostUsd: runs.EstimatedCostUSD, UnknownCostUsd: runs.UnknownCostUSD, ExecutedQuery: query, Validity: h.validityForSample(runs.TotalRuns)}), nil
}

func (h *Handler) RunVolume(ctx context.Context, req *connect.Request[measurepb.RunVolumeRequest]) (*connect.Response[measurepb.RunVolumeResponse], error) {
	r, err := h.metric(ctx, runVolume, req.Msg.GetFilter(), req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&measurepb.RunVolumeResponse{TotalRuns: r.Numerator, TerminalRuns: r.Denom, ExecutedQuery: r.Query, Validity: protoValidity(r.Validity)}), nil
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
	query := fmt.Sprintf("SELECT DISTINCT run_id FROM invocation_read_model_facts WHERE occurred_at >= %q AND occurred_at < %q ORDER BY run_id LIMIT %d", filter.From.UTC().Format(time.RFC3339Nano), filter.To.UTC().Format(time.RFC3339Nano), limit)
	return connect.NewResponse(&measurepb.SelectCohortResponse{RunIds: cohort.RunIDs, Truncated: cohort.Truncated, ExecutedQuery: query, Validity: h.validityForSample(int64(len(cohort.RunIDs)))}), nil
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
	response := &measurepb.EpisodeCohortResponse{AvailabilityState: string(cohort.Availability.State), AvailabilityReason: cohort.Availability.Reason, ExecutedQuery: fmt.Sprintf("SELECT persisted friction episodes FROM selected durable run cohort LIMIT %d", limit)}
	for _, signal := range cohort.Signals {
		response.Signals = append(response.Signals, &measurepb.EpisodeCohortSignal{Fingerprint: signal.Fingerprint, Occurrences: int64(signal.Occurrences), DistinctRuns: int64(signal.DistinctRuns), SummedCostMs: signal.SummedCostMS, Confidence: signal.Confidence, RepresentativeRunIds: append([]string(nil), signal.RepresentativeRunIDs...)})
	}
	response.Validity = h.validityForSample(int64(len(cohort.Signals)))
	return connect.NewResponse(response), nil
}

func (h *Handler) runStatusRows(ctx context.Context, input *measurepb.InvocationFilter, window *sharedmeasurepb.TimeWindow) ([]invocationreadmodel.RunStatusCount, string, error) {
	filter, err := filterWithWindow(input, window, h.now())
	if err != nil {
		return nil, "", connect.NewError(connect.CodeInvalidArgument, err)
	}
	rows, err := h.store.RunStatusCounts(ctx, filter)
	if err != nil {
		return nil, "", connect.NewError(connect.CodeInternal, err)
	}
	return rows, fmt.Sprintf("SELECT status, COUNT(*) FROM invocation_read_model_runs WHERE occurred_at >= %q AND occurred_at < %q GROUP BY status", filter.From.UTC().Format(time.RFC3339Nano), filter.To.UTC().Format(time.RFC3339Nano)), nil
}

func (h *Handler) RunStatusDistribution(ctx context.Context, req *connect.Request[measurepb.RunStatusDistributionRequest]) (*connect.Response[measurepb.RunStatusDistributionResponse], error) {
	rows, query, err := h.runStatusRows(ctx, req.Msg.GetFilter(), req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	response := &measurepb.RunStatusDistributionResponse{ExecutedQuery: query, Rows: make([]*measurepb.RunStatusCount, 0, len(rows))}
	for _, row := range rows {
		response.Rows = append(response.Rows, &measurepb.RunStatusCount{Status: row.Status, Count: row.Count})
	}
	var sample int64
	for _, row := range rows {
		sample += row.Count
	}
	response.Validity = h.validityForSample(sample)
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
	return connect.NewResponse(&measurepb.RunDurationStatisticsResponse{AverageDurationMs: stats.AverageDurationMS, P50DurationMs: stats.P50DurationMS, P95DurationMs: stats.P95DurationMS, P99DurationMs: stats.P99DurationMS, MinDurationMs: stats.MinDurationMS, MaxDurationMs: stats.MaxDurationMS, Count: stats.Count, ExecutedQuery: query, Validity: h.validityForSample(stats.Count)}), nil
}

func (h *Handler) runBreakdownRows(ctx context.Context, input *measurepb.InvocationFilter, window *sharedmeasurepb.TimeWindow, dimension string) ([]invocationreadmodel.RunBreakdownRow, string, error) {
	filter, err := filterWithWindow(input, window, h.now())
	if err != nil {
		return nil, "", connect.NewError(connect.CodeInvalidArgument, err)
	}
	rows, err := h.store.RunBreakdown(ctx, filter, dimension, 20)
	if err != nil {
		return nil, "", connect.NewError(connect.CodeInternal, err)
	}
	return rows, fmt.Sprintf("SELECT %s, terminal run aggregates FROM invocation_read_model_runs WHERE occurred_at >= %q AND occurred_at < %q GROUP BY %s LIMIT 20", dimension, filter.From.UTC().Format(time.RFC3339Nano), filter.To.UTC().Format(time.RFC3339Nano), dimension), nil
}

func protoBreakdownRows(rows []invocationreadmodel.RunBreakdownRow) []*measurepb.RunBreakdownRow {
	out := make([]*measurepb.RunBreakdownRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, &measurepb.RunBreakdownRow{Key: row.Key, Value: row.Value, RunCount: row.RunCount, SuccessCount: row.SuccessCount, FailedCount: row.FailedCount, TotalCostUsd: row.TotalCostUSD, TotalTokens: row.TotalTokens, AverageDurationMs: row.AvgDurationMS})
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
	rows, query, err := h.runBreakdownRows(ctx, req.Msg.GetFilter(), req.Msg.GetWindow(), "runner")
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&measurepb.RunnerBreakdownResponse{Rows: protoBreakdownRows(rows), ExecutedQuery: query, Validity: h.validityForSample(sumBreakdownRuns(rows))}), nil
}

func (h *Handler) ModelBreakdown(ctx context.Context, req *connect.Request[measurepb.ModelBreakdownRequest]) (*connect.Response[measurepb.ModelBreakdownResponse], error) {
	rows, query, err := h.runBreakdownRows(ctx, req.Msg.GetFilter(), req.Msg.GetWindow(), "model")
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&measurepb.ModelBreakdownResponse{Rows: protoBreakdownRows(rows), ExecutedQuery: query, Validity: h.validityForSample(sumBreakdownRuns(rows))}), nil
}

func (h *Handler) ProfileBreakdown(ctx context.Context, req *connect.Request[measurepb.ProfileBreakdownRequest]) (*connect.Response[measurepb.ProfileBreakdownResponse], error) {
	rows, query, err := h.runBreakdownRows(ctx, req.Msg.GetFilter(), req.Msg.GetWindow(), "profile")
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&measurepb.ProfileBreakdownResponse{Rows: protoBreakdownRows(rows), ExecutedQuery: query, Validity: h.validityForSample(sumBreakdownRuns(rows))}), nil
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
	response := &measurepb.TerminalRunTrendResponse{ExecutedQuery: fmt.Sprintf("SELECT terminal outcome aggregates FROM invocation_read_model_runs WHERE occurred_at >= %q AND occurred_at < %q GROUP BY hourly terminal bucket", filter.From.UTC().Format(time.RFC3339Nano), filter.To.UTC().Format(time.RFC3339Nano)), Rows: make([]*measurepb.TerminalRunTrendRow, 0, len(rows))}
	for _, row := range rows {
		response.Rows = append(response.Rows, &measurepb.TerminalRunTrendRow{Bucket: row.Bucket.UTC().Format(time.RFC3339), TerminalRuns: row.TerminalRuns, CompletedRuns: row.CompletedRuns, FailedRuns: row.FailedRuns, CancelledRuns: row.CancelledRuns, TotalCostUsd: row.TotalCostUSD, AverageDurationMs: row.AvgDurationMS})
	}
	var sample int64
	for _, row := range rows {
		sample += row.TerminalRuns
	}
	response.Validity = h.validityForSample(sample)
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
	response := &measurepb.ToolUsageResponse{ExecutedQuery: fmt.Sprintf("SELECT tool_name, classified outcomes FROM invocation_read_model_facts WHERE occurred_at >= %q AND occurred_at < %q GROUP BY tool_name LIMIT 20", filter.From.UTC().Format(time.RFC3339Nano), filter.To.UTC().Format(time.RFC3339Nano)), Rows: make([]*measurepb.ToolUsageRow, 0, len(rows))}
	for _, row := range rows {
		response.Rows = append(response.Rows, &measurepb.ToolUsageRow{ToolName: row.ToolName, CallCount: row.CallCount, SuccessCount: row.SuccessCount, FailedCount: row.FailedCount})
	}
	var sample int64
	for _, row := range rows {
		sample += row.CallCount
	}
	response.Validity = h.validityForSample(sample)
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
	response := &measurepb.ErrorPatternsResponse{ExecutedQuery: fmt.Sprintf("SELECT error_code aggregate FROM invocation_read_model_errors WHERE occurred_at >= %q AND occurred_at < %q GROUP BY error_code LIMIT 20", filter.From.UTC().Format(time.RFC3339Nano), filter.To.UTC().Format(time.RFC3339Nano)), Rows: make([]*measurepb.ErrorPatternRow, 0, len(rows))}
	for _, row := range rows {
		response.Rows = append(response.Rows, &measurepb.ErrorPatternRow{ErrorCode: row.ErrorCode, Count: row.Count, LastSeen: row.LastSeen.UTC().Format(time.RFC3339), SampleRunId: row.SampleRunID})
	}
	var sample int64
	for _, row := range rows {
		sample += row.Count
	}
	response.Validity = h.validityForSample(sample)
	return connect.NewResponse(response), nil
}

func (h *Handler) FileRereadRate(ctx context.Context, req *connect.Request[measurepb.FileRereadRateRequest]) (*connect.Response[measurepb.FileRereadRateResponse], error) {
	r, err := h.metric(ctx, fileRereadRate, req.Msg.GetFilter(), req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&measurepb.FileRereadRateResponse{Rate: r.Rate, FilesReadMoreThanOnce: r.Numerator, ReadCalls: r.Denom, ExecutedQuery: r.Query, Validity: protoValidity(r.Validity)}), nil
}

func (h *Handler) FindingRecurrenceRate(ctx context.Context, req *connect.Request[measurepb.FindingRecurrenceRateRequest]) (*connect.Response[measurepb.FindingRecurrenceRateResponse], error) {
	r, err := h.metric(ctx, findingRecurrenceRate, req.Msg.GetFilter(), req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&measurepb.FindingRecurrenceRateResponse{Rate: r.Rate, RecurringFindings: r.Numerator, TotalFindings: r.Denom, RecurringFingerprints: r.Secondary, ExecutedQuery: r.Query, Validity: protoValidity(r.Validity)}), nil
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
				fields = append(fields, map[string]string{"value": row.Value, "run_count": strconv.FormatInt(row.RunCount, 10), "success_count": strconv.FormatInt(row.SuccessCount, 10), "failed_count": strconv.FormatInt(row.FailedCount, 10), "total_cost_usd": strconv.FormatFloat(row.TotalCostUSD, 'f', -1, 64), "average_duration_ms": strconv.FormatFloat(row.AvgDurationMS, 'f', -1, 64)})
			}
			query := fmt.Sprintf("SELECT %s, terminal run aggregates FROM invocation_read_model_runs WHERE occurred_at >= %q AND occurred_at < %q GROUP BY %s LIMIT 20", spec.dimension, rangeValue.From.UTC().Format(time.RFC3339Nano), rangeValue.To.UTC().Format(time.RFC3339Nano), spec.dimension)
			return measurelib.MeasureResult{Fields: fields, Provenance: measurelib.Provenance{ExecutedQuery: query}}, nil
		}); err != nil {
			return nil, err
		}
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
