package measures

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/conversationsearch"
	"agent-manager/internal/invocationreadmodel"
	"agent-manager/internal/runsignal"

	"connectrpc.com/connect"
	"github.com/jmoiron/sqlx"
	coredb "github.com/vrooli/api-core/database"
	measurelib "github.com/vrooli/measures-go"
	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
	sharedmeasurepb "github.com/vrooli/vrooli/packages/proto/gen/go/measures/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
	_ "modernc.org/sqlite"
)

type fakeStore struct {
	metrics          invocationreadmodel.Metrics
	runMetrics       invocationreadmodel.RunMetrics
	usage            invocationreadmodel.UsageTotals
	fallback         invocationreadmodel.FallbackInsights
	costDistribution invocationreadmodel.RunCostDistribution
	durationStats    invocationreadmodel.RunDurationStatistics
	statusCounts     []invocationreadmodel.RunStatusCount
	breakdowns       map[string][]invocationreadmodel.RunBreakdownRow
	runTimeSeries    []invocationreadmodel.RunTimeSeriesBucket
	toolUsage        []invocationreadmodel.ToolUsageRow
	capabilityUsage  []invocationreadmodel.CapabilityUsageRow
	tokenAttribution []invocationreadmodel.TokenAttributionRow
	errorPatterns    []invocationreadmodel.ErrorPattern
	findingMetrics   invocationreadmodel.FindingMetrics
	filter           invocationreadmodel.Filter
	latestProjection time.Time
}

func (s *fakeStore) WindowedUsage(_ context.Context, filter invocationreadmodel.Filter) (invocationreadmodel.UsageTotals, error) {
	s.filter = filter
	return s.usage, nil
}

func (s *fakeStore) FallbackInsights(_ context.Context, filter invocationreadmodel.Filter) (invocationreadmodel.FallbackInsights, error) {
	s.filter = filter
	return s.fallback, nil
}

// The scope selector reaches the store; leaving it unset lets the store apply
// its defaults (Agent Manager runs for run statistics, everything for tokens).
func TestScopeSelectorReachesTheStore(t *testing.T) {
	for _, tc := range []struct {
		scope measurepb.RunScope
		want  invocationreadmodel.RunScope
	}{
		{measurepb.RunScope_RUN_SCOPE_UNSPECIFIED, invocationreadmodel.ScopeUnspecified},
		{measurepb.RunScope_RUN_SCOPE_AGENT_MANAGER, invocationreadmodel.ScopeAgentManager},
		{measurepb.RunScope_RUN_SCOPE_OPERATOR_SESSIONS, invocationreadmodel.ScopeOperatorSessions},
		{measurepb.RunScope_RUN_SCOPE_ALL, invocationreadmodel.ScopeAll},
	} {
		store := &fakeStore{breakdowns: map[string][]invocationreadmodel.RunBreakdownRow{"model": {}}}
		handler := NewHandler(store, func() time.Time { return time.Date(2026, time.July, 29, 12, 0, 0, 0, time.UTC) })
		request := &measurepb.ModelBreakdownRequest{Window: token(sharedmeasurepb.TimeWindowToken_TIME_WINDOW_TOKEN_LAST_7D), Filter: &measurepb.InvocationFilter{Scope: tc.scope}}
		if _, err := handler.ModelBreakdown(context.Background(), connect.NewRequest(request)); err != nil {
			t.Fatal(err)
		}
		if store.filter.Scope != tc.want {
			t.Fatalf("scope %v reached the store as %q, want %q", tc.scope, store.filter.Scope, tc.want)
		}
	}
}

func TestFallbackInsightsReportsCountsAndReasons(t *testing.T) {
	store := &fakeStore{fallback: invocationreadmodel.FallbackInsights{
		RunnerAttempts: 3, RunnerExhausted: 1, ModelAttempts: 2,
		ModelPairs:   []invocationreadmodel.FallbackPair{{From: "a", To: "b", Reason: "rate_limit", Count: 2}},
		ModelReasons: []invocationreadmodel.CountRow{{Key: "rate_limit", Count: 2}},
	}}
	handler := NewHandler(store, func() time.Time { return time.Date(2026, time.July, 29, 12, 0, 0, 0, time.UTC) })
	response, err := handler.FallbackInsights(context.Background(), connect.NewRequest(&measurepb.FallbackInsightsRequest{Window: token(sharedmeasurepb.TimeWindowToken_TIME_WINDOW_TOKEN_LAST_7D)}))
	if err != nil {
		t.Fatal(err)
	}
	msg := response.Msg
	if msg.GetRunnerAttempts() != 3 || msg.GetRunnerExhausted() != 1 || msg.GetModelAttempts() != 2 || len(msg.GetModelPairs()) != 1 || msg.GetModelPairs()[0].GetTo() != "b" || msg.GetModelReasons()[0].GetKey() != "rate_limit" {
		t.Fatalf("fallback response = %+v", msg)
	}
	if msg.GetValidity().GetSampleSize() != 6 {
		t.Fatalf("sample size = %d, want 6 events", msg.GetValidity().GetSampleSize())
	}
	registry, err := handler.Registry()
	if err != nil {
		t.Fatal(err)
	}
	served, err := registry.Execute(context.Background(), measureRequest(FallbackInsights, "last_7d"))
	if err != nil || served.Fields[0]["runner_attempts"] != "3" {
		t.Fatalf("registry fallback = %+v err=%v", served, err)
	}
}

func TestProfileBreakdownPreservesTokenComponents(t *testing.T) {
	store := &fakeStore{breakdowns: map[string][]invocationreadmodel.RunBreakdownRow{
		"profile": {{Key: "profile-id", Value: "Profile", TotalTokens: 30, InputTokens: 15, OutputTokens: 7, CacheReadTokens: 3, CacheCreationTokens: 3}},
	}}
	handler := NewHandler(store, time.Now)
	response, err := handler.ProfileBreakdown(context.Background(), connect.NewRequest(&measurepb.ProfileBreakdownRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	rows := response.Msg.GetRows()
	if len(rows) != 1 || rows[0].GetKey() != "profile-id" || rows[0].GetTotalTokens() != 30 || rows[0].GetInputTokens() != 15 || rows[0].GetOutputTokens() != 7 || rows[0].GetCacheReadTokens() != 3 || rows[0].GetCacheCreationTokens() != 3 {
		t.Fatalf("profile breakdown = %+v", rows)
	}
}

func (s *fakeStore) RunMetrics(_ context.Context, filter invocationreadmodel.Filter) (invocationreadmodel.RunMetrics, error) {
	s.filter = filter
	return s.runMetrics, nil
}

func (s *fakeStore) LatestProjection(context.Context) (time.Time, error) {
	return s.latestProjection, nil
}

func TestMeasureProvenanceReportsProjectionFreshness(t *testing.T) {
	now := time.Date(2026, time.July, 29, 12, 0, 0, 0, time.UTC)
	store := &fakeStore{latestProjection: now.Add(-time.Hour)}
	handler := NewHandler(store, func() time.Time { return now })
	response, err := handler.RunVolume(context.Background(), connect.NewRequest(&measurepb.RunVolumeRequest{Window: &sharedmeasurepb.TimeWindow{Window: &sharedmeasurepb.TimeWindow_Custom{Custom: &sharedmeasurepb.CustomRange{From: timestamppb.New(now.Add(-24 * time.Hour)), To: timestamppb.New(now)}}}}))
	if err != nil {
		t.Fatal(err)
	}
	provenance := response.Msg.GetProvenance()
	if provenance.GetProjectionAt() != "2026-07-29T11:00:00Z" || !provenance.GetProjectionStale() || provenance.GetProjectionStaleReason() != "projection_before_window_end" {
		t.Fatalf("provenance freshness = %+v", provenance)
	}
}

func (s *fakeStore) RunDurationStatistics(_ context.Context, filter invocationreadmodel.Filter) (invocationreadmodel.RunDurationStatistics, error) {
	s.filter = filter
	return s.durationStats, nil
}

func (s *fakeStore) RunCostDistribution(_ context.Context, filter invocationreadmodel.Filter) (invocationreadmodel.RunCostDistribution, error) {
	s.filter = filter
	return s.costDistribution, nil
}

func (s *fakeStore) RunStatusCounts(_ context.Context, filter invocationreadmodel.Filter) ([]invocationreadmodel.RunStatusCount, error) {
	s.filter = filter
	return s.statusCounts, nil
}

func (s *fakeStore) RunBreakdown(_ context.Context, filter invocationreadmodel.Filter, dimension string, _ int) ([]invocationreadmodel.RunBreakdownRow, error) {
	s.filter = filter
	return s.breakdowns[dimension], nil
}

func (s *fakeStore) RunTimeSeries(_ context.Context, filter invocationreadmodel.Filter, _ time.Duration) ([]invocationreadmodel.RunTimeSeriesBucket, error) {
	s.filter = filter
	return s.runTimeSeries, nil
}

func (s *fakeStore) ToolUsage(_ context.Context, filter invocationreadmodel.Filter, _ int) ([]invocationreadmodel.ToolUsageRow, error) {
	s.filter = filter
	return s.toolUsage, nil
}

func (s *fakeStore) CapabilityUsage(_ context.Context, filter invocationreadmodel.Filter, _ int) ([]invocationreadmodel.CapabilityUsageRow, error) {
	s.filter = filter
	return s.capabilityUsage, nil
}

func (s *fakeStore) TokenAttribution(_ context.Context, filter invocationreadmodel.Filter, groupBy, _ string, _ int) ([]invocationreadmodel.TokenAttributionRow, error) {
	s.filter = filter
	for index := range s.tokenAttribution {
		s.tokenAttribution[index].GroupBy = groupBy
	}
	return s.tokenAttribution, nil
}

func TestTokenAttributionValidatesSelectionsAndReturnsEstimatedShare(t *testing.T) {
	now := time.Date(2026, time.July, 29, 12, 0, 0, 0, time.UTC)
	store := &fakeStore{tokenAttribution: []invocationreadmodel.TokenAttributionRow{{Value: "shell", TotalTokens: 100, EstimatedTokens: 25, EstimatedTokenShare: .25, P50FootprintTokens: 10, P95FootprintTokens: 10, MaxFootprintTokens: 20}}}
	handler := NewHandler(store, func() time.Time { return now })
	response, err := handler.TokenAttribution(context.Background(), connect.NewRequest(&measurepb.TokenAttributionRequest{GroupBy: "executable", View: "footprint"}))
	if err != nil || response.Msg.GetEstimatedTokenShare() != .25 || len(response.Msg.GetRows()) != 1 || response.Msg.GetRows()[0].GetEstimatedTokenShare() != .25 {
		t.Fatalf("response=%+v err=%v", response.Msg, err)
	}
	if _, err := handler.TokenAttribution(context.Background(), connect.NewRequest(&measurepb.TokenAttributionRequest{GroupBy: "bad", View: "footprint"})); err == nil || !strings.Contains(err.Error(), "capability, executable, command_path, target_scenario_operation") {
		t.Fatalf("invalid group-by error=%v", err)
	}
}

func (s *fakeStore) ErrorPatterns(_ context.Context, filter invocationreadmodel.Filter, _ int) ([]invocationreadmodel.ErrorPattern, error) {
	s.filter = filter
	return s.errorPatterns, nil
}

func (s *fakeStore) FindingMetrics(_ context.Context, filter invocationreadmodel.Filter) (invocationreadmodel.FindingMetrics, error) {
	s.filter = filter
	return s.findingMetrics, nil
}

func (s *fakeStore) Metrics(_ context.Context, filter invocationreadmodel.Filter) (invocationreadmodel.Metrics, error) {
	s.filter = filter
	return s.metrics, nil
}

func TestRegistryAndTypedRPCShareExternalToolComputation(t *testing.T) {
	now := time.Date(2026, time.July, 29, 12, 0, 0, 0, time.UTC)
	store := &fakeStore{metrics: invocationreadmodel.Metrics{ExternalToolShare: 0.4, ExternalCalls: 4, ResolvedCalls: 10, UnknownCalls: 3}}
	handler := NewHandler(store, func() time.Time { return now })

	rpc, err := handler.ExternalToolShare(context.Background(), connect.NewRequest(&measurepb.ExternalToolShareRequest{Window: token(sharedmeasurepb.TimeWindowToken_TIME_WINDOW_TOKEN_LAST_7D)}))
	if err != nil {
		t.Fatal(err)
	}
	if rpc.Msg.GetShare() != 0.4 || rpc.Msg.GetUnknownCalls() != 3 || rpc.Msg.GetProvenance().GetExecutedQuery() == "" {
		t.Fatalf("rpc response=%+v", rpc.Msg)
	}

	registry, err := handler.Registry()
	if err != nil {
		t.Fatal(err)
	}
	served, err := registry.Execute(context.Background(), measureRequest(ExternalToolShare, "last_7d"))
	if err != nil {
		t.Fatal(err)
	}
	if served.Value != strconv.FormatFloat(rpc.Msg.GetShare(), 'f', -1, 64) || served.Provenance.ExecutedQuery != rpc.Msg.GetProvenance().GetExecutedQuery() {
		t.Fatalf("registry=%+v rpc=%+v", served, rpc.Msg)
	}
	if store.filter.From == nil || store.filter.To == nil || !store.filter.To.After(*store.filter.From) {
		t.Fatalf("expected resolved half-open range, filter=%+v", store.filter)
	}
}

func TestTypedMeasuresDefaultWindowAndMapFilterDimensions(t *testing.T) {
	now := time.Date(2026, time.July, 29, 12, 0, 0, 0, time.UTC)
	store := &fakeStore{metrics: invocationreadmodel.Metrics{RetryRate: 0.25, RetryCalls: 2, TotalCalls: 8}}
	handler := NewHandler(store, func() time.Time { return now })
	response, err := handler.RetryRate(context.Background(), connect.NewRequest(&measurepb.RetryRateRequest{Filter: &measurepb.InvocationFilter{Ownership: "external", ProfileId: "profile-a", TagPrefix: "batch-"}}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.GetRate() != 0.25 || store.filter.Ownership != "external" || store.filter.ProfileID != "profile-a" || store.filter.TagPrefix != "batch-" {
		t.Fatalf("response=%+v filter=%+v", response.Msg, store.filter)
	}
	if store.filter.From == nil || store.filter.To == nil {
		t.Fatal("default window was not resolved")
	}
}

func TestWorkloadEfficiencyIncludesFailedAttemptsInNumerator(t *testing.T) {
	now := time.Date(2026, time.July, 29, 12, 0, 0, 0, time.UTC)
	store := &fakeStore{
		runMetrics: invocationreadmodel.RunMetrics{TerminalRuns: 5, SuccessfulRuns: 1},
		usage:      invocationreadmodel.UsageTotals{TotalTokens: 150},
	}
	handler := NewHandler(store, func() time.Time { return now })
	registry, err := handler.Registry()
	if err != nil {
		t.Fatal(err)
	}
	result, err := registry.Execute(context.Background(), measurelib.MeasureRequest{
		Measure: WorkloadEfficiency,
		Params:  map[string]string{"window": "last_7d", "workload_key": "wf:classify", "model": "gpt-test"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Value != "150" || result.Provenance.ExecutedQuery == "" || result.Fields[0]["observational_limitation"] == "" {
		t.Fatalf("result=%+v", result)
	}
	if store.filter.WorkloadKey != "wf:classify" || store.filter.Model != "gpt-test" {
		t.Fatalf("filter=%+v", store.filter)
	}
}

func TestRepeatedWorkRateMarksDegenerateFingerprintCorpusUnreliable(t *testing.T) {
	store := &fakeStore{metrics: invocationreadmodel.Metrics{RepeatedWorkRate: 0.991, RepeatedCalls: 109, TotalCalls: 110, LargestFingerprintBucket: 100}}
	handler := NewHandler(store, func() time.Time { return time.Date(2026, time.July, 29, 12, 0, 0, 0, time.UTC) })
	response, err := handler.RepeatedWorkRate(context.Background(), connect.NewRequest(&measurepb.RepeatedWorkRateRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if got := response.Msg.GetValidity(); got.GetState() != "unreliable" || got.GetLargestFingerprintBucket() != 100 || got.GetReason() == "" {
		t.Fatalf("validity=%+v, want unreliable concentration result", got)
	}
}

func TestRepeatedWorkRateMarksHealthyFingerprintCorpusAvailable(t *testing.T) {
	store := &fakeStore{metrics: invocationreadmodel.Metrics{RepeatedWorkRate: 0.2, RepeatedCalls: 2, TotalCalls: 10, LargestFingerprintBucket: 3}}
	handler := NewHandler(store, func() time.Time { return time.Date(2026, time.July, 29, 12, 0, 0, 0, time.UTC) })
	response, err := handler.RepeatedWorkRate(context.Background(), connect.NewRequest(&measurepb.RepeatedWorkRateRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if got := response.Msg.GetValidity(); got.GetState() != "available" || got.GetReason() != "" {
		t.Fatalf("validity=%+v, want available healthy result", got)
	}
}

func TestRateMarksSmallSampleUnreliable(t *testing.T) {
	store := &fakeStore{metrics: invocationreadmodel.Metrics{RetryRate: 1, RetryCalls: 2, TotalCalls: 2, LargestFingerprintBucket: 1}}
	handler := NewHandler(store, func() time.Time { return time.Date(2026, time.July, 29, 12, 0, 0, 0, time.UTC) })
	response, err := handler.RetryRate(context.Background(), connect.NewRequest(&measurepb.RetryRateRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if got := response.Msg.GetValidity(); got.GetState() != "unreliable" || got.GetSampleSize() != 2 {
		t.Fatalf("validity=%+v, want unreliable small sample", got)
	}
}

func TestRegistryAndTypedRPCShareRunThroughputComputation(t *testing.T) {
	now := time.Date(2026, time.July, 29, 12, 0, 0, 0, time.UTC)
	store := &fakeStore{
		runMetrics: invocationreadmodel.RunMetrics{TotalRuns: 8, TerminalRuns: 6, SuccessfulRuns: 3, SuccessRate: 0.5, CompletedDurationRuns: 5, AverageDurationMS: 1234, P50DurationMS: 1000, P90DurationMS: 2000, ReadCalls: 6, FileRereads: 2, FileRereadRate: 1.0 / 3.0},
		usage:      invocationreadmodel.UsageTotals{MeteredCostUSD: 4.5, TotalTokens: 99, Runs: 8, Requests: 20},
	}
	handler := NewHandler(store, func() time.Time { return now })
	rpc, err := handler.RunSuccessRate(context.Background(), connect.NewRequest(&measurepb.RunSuccessRateRequest{Window: token(sharedmeasurepb.TimeWindowToken_TIME_WINDOW_TOKEN_LAST_7D)}))
	if err != nil || rpc.Msg.GetRate() != 0.5 || rpc.Msg.GetSuccessfulRuns() != 3 || rpc.Msg.GetTerminalRuns() != 6 || rpc.Msg.GetProvenance().GetExecutedQuery() == "" {
		t.Fatalf("success rpc=%+v err=%v", rpc.Msg, err)
	}
	cycle, err := handler.RunCycleTime(context.Background(), connect.NewRequest(&measurepb.RunCycleTimeRequest{}))
	if err != nil || cycle.Msg.GetAverageDurationMs() != 1234 || cycle.Msg.GetP50DurationMs() != 1000 || cycle.Msg.GetP90DurationMs() != 2000 || cycle.Msg.GetCompletedDurationRuns() != 5 || cycle.Msg.GetValidity().GetSampleSize() != 5 {
		t.Fatalf("cycle rpc=%+v err=%v", cycle.Msg, err)
	}
	cost, err := handler.RunCost(context.Background(), connect.NewRequest(&measurepb.RunCostRequest{}))
	if err != nil || cost.Msg.GetTotalCostUsd() != 4.5 || cost.Msg.GetTotalTokens() != 99 || cost.Msg.GetTotalRuns() != 8 || cost.Msg.GetRequests() != 20 || cost.Msg.GetValidity().GetSampleSize() != 8 {
		t.Fatalf("cost rpc=%+v err=%v", cost.Msg, err)
	}
	registry, err := handler.Registry()
	if err != nil {
		t.Fatal(err)
	}
	served, err := registry.Execute(context.Background(), measureRequest(RunVolume, "last_7d"))
	if err != nil || served.Value != "8" || served.Provenance.ExecutedQuery == "" {
		t.Fatalf("volume registry=%+v err=%v", served, err)
	}
	rereads, err := handler.FileRereadRate(context.Background(), connect.NewRequest(&measurepb.FileRereadRateRequest{}))
	if err != nil || rereads.Msg.GetRate() != 1.0/3.0 || rereads.Msg.GetFilesReadMoreThanOnce() != 2 || rereads.Msg.GetReadCalls() != 6 {
		t.Fatalf("reread rpc=%+v err=%v", rereads.Msg, err)
	}
}

func TestRunDurationStatisticsUsesDurableStore(t *testing.T) {
	now := time.Date(2026, time.July, 29, 12, 0, 0, 0, time.UTC)
	store := &fakeStore{durationStats: invocationreadmodel.RunDurationStatistics{AverageDurationMS: 100, P50DurationMS: 100, P95DurationMS: 100, P99DurationMS: 100, MinDurationMS: 25, MaxDurationMS: 200, Count: 3}}
	handler := NewHandler(store, func() time.Time { return now })
	response, err := handler.RunDurationStatistics(context.Background(), connect.NewRequest(&measurepb.RunDurationStatisticsRequest{Window: token(sharedmeasurepb.TimeWindowToken_TIME_WINDOW_TOKEN_LAST_7D)}))
	if err != nil || response.Msg.GetAverageDurationMs() != 100 || response.Msg.GetMinDurationMs() != 25 || response.Msg.GetMaxDurationMs() != 200 || response.Msg.GetCount() != 3 || response.Msg.GetProvenance().GetExecutedQuery() == "" {
		t.Fatalf("duration response=%+v err=%v", response.Msg, err)
	}
}

func TestRunCostPopulatesDistribution(t *testing.T) {
	now := time.Date(2026, time.July, 29, 12, 0, 0, 0, time.UTC)
	store := &fakeStore{
		usage: invocationreadmodel.UsageTotals{Runs: 8, MeteredCostUSD: 4.5, TotalTokens: 99},
		costDistribution: invocationreadmodel.RunCostDistribution{
			P50Tokens: 1000, P90Tokens: 9000, P95Tokens: 9500, P99Tokens: 9900, MaxTokens: 10000,
			P50CostUSD: 0.01, P90CostUSD: 0.09, P95CostUSD: 0.095, P99CostUSD: 0.099, MaxCostUSD: 0.10,
			SampleSize: 5,
			TokenBuckets: []invocationreadmodel.RunTokenBucket{
				{Label: "<100K", MinTokens: 0, MaxTokens: 100000, RunCount: 5},
			},
		},
	}
	handler := NewHandler(store, func() time.Time { return now })
	response, err := handler.RunCost(context.Background(), connect.NewRequest(&measurepb.RunCostRequest{Window: token(sharedmeasurepb.TimeWindowToken_TIME_WINDOW_TOKEN_LAST_7D)}))
	if err != nil {
		t.Fatal(err)
	}
	if got := response.Msg.GetP50Tokens(); got != 1000 {
		t.Fatalf("p50 tokens = %d, want 1000", got)
	}
	if got := response.Msg.GetP90Tokens(); got != 9000 {
		t.Fatalf("p90 tokens = %d, want 9000", got)
	}
	if got := response.Msg.GetDistributionSampleSize(); got != 5 {
		t.Fatalf("sample size = %d, want 5", got)
	}
	if len(response.Msg.GetTokenBuckets()) != 1 || response.Msg.GetTokenBuckets()[0].GetRunCount() != 5 {
		t.Fatalf("token buckets = %+v", response.Msg.GetTokenBuckets())
	}
}

func TestTypedFindingRecurrenceUsesSharedComputation(t *testing.T) {
	now := time.Date(2026, time.July, 29, 12, 0, 0, 0, time.UTC)
	store := &fakeStore{findingMetrics: invocationreadmodel.FindingMetrics{TotalFindings: 3, RecurringFindings: 2, RecurringFingerprints: 1, RecurrenceRate: 2.0 / 3.0}}
	handler := NewHandler(store, func() time.Time { return now })
	response, err := handler.FindingRecurrenceRate(context.Background(), connect.NewRequest(&measurepb.FindingRecurrenceRateRequest{}))
	if err != nil || response.Msg.GetRate() != 2.0/3.0 || response.Msg.GetRecurringFindings() != 2 || response.Msg.GetTotalFindings() != 3 || response.Msg.GetRecurringFingerprints() != 1 || response.Msg.GetProvenance().GetExecutedQuery() == "" {
		t.Fatalf("recurrence response=%+v err=%v", response.Msg, err)
	}
}

func TestConversationSearchQualityUsesBoundedContentFreeTelemetry(t *testing.T) {
	now := time.Date(2026, time.September, 4, 20, 0, 0, 0, time.UTC)
	db, err := sqlx.Connect("sqlite", "file:measures-conversation-search?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := coredb.EnsureSchemas(context.Background(), db, coredb.SchemaProviderFunc(conversationsearch.Schema)); err != nil {
		t.Fatal(err)
	}
	repository := conversationsearch.NewSQLiteRepository(db)
	service, err := conversationsearch.NewService(repository, repository, repository, []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	for index := 0; index < 5; index++ {
		record := conversationsearch.SearchTelemetry{
			RequestID: "quality-" + strconv.Itoa(index), Mode: "hybrid", Sort: "relevance",
			Duration: time.Duration(10+index*10) * time.Millisecond, CandidateCount: 2, ResultCount: 2,
			LexicalContributed: true, SemanticContributed: index > 0, CreatedAt: now.Add(-time.Duration(index+1) * time.Minute),
		}
		if index == 0 {
			record.ResultCount = 0
			record.DegradationReasons = []string{"embedding_unavailable"}
		}
		if _, err := service.RecordSearchTelemetry(context.Background(), record); err != nil {
			t.Fatal(err)
		}
	}
	handler := NewHandler(&fakeStore{}, func() time.Time { return now })
	handler.SetConversationSearchQuality(service, nil)
	response, err := handler.ConversationSearchQuality(context.Background(), connect.NewRequest(&measurepb.ConversationSearchQualityRequest{Window: token(sharedmeasurepb.TimeWindowToken_TIME_WINDOW_TOKEN_LAST_7D)}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.GetQueries() != 5 || response.Msg.GetNoResultRate() != 0.2 || response.Msg.GetDegradationRate() != 0.2 || response.Msg.GetP50LatencyMs() != 30 || response.Msg.GetP95LatencyMs() != 50 {
		t.Fatalf("quality response=%+v", response.Msg)
	}
	if response.Msg.GetValidity().GetState() != "available" || response.Msg.GetProvenance().GetRowCount() != 5 || response.Msg.GetDefinitionId() != ConversationSearchQuality {
		t.Fatalf("quality metadata=%+v", response.Msg)
	}
}

func TestRegistryAndTypedRPCShareRunStatusDistribution(t *testing.T) {
	now := time.Date(2026, time.July, 29, 12, 0, 0, 0, time.UTC)
	store := &fakeStore{statusCounts: []invocationreadmodel.RunStatusCount{{Status: "complete", Count: 5}, {Status: "failed", Count: 2}}}
	handler := NewHandler(store, func() time.Time { return now })
	rpc, err := handler.RunStatusDistribution(context.Background(), connect.NewRequest(&measurepb.RunStatusDistributionRequest{Window: token(sharedmeasurepb.TimeWindowToken_TIME_WINDOW_TOKEN_LAST_7D)}))
	if err != nil || len(rpc.Msg.GetRows()) != 2 || rpc.Msg.GetRows()[0].GetStatus() != "complete" || rpc.Msg.GetRows()[0].GetCount() != 5 || rpc.Msg.GetProvenance().GetExecutedQuery() == "" {
		t.Fatalf("status rpc=%+v err=%v", rpc.Msg, err)
	}
	registry, err := handler.Registry()
	if err != nil {
		t.Fatal(err)
	}
	served, err := registry.Execute(context.Background(), measureRequest(RunStatusDistribution, "last_7d"))
	if err != nil || len(served.Fields) != 2 || served.Fields[1]["status"] != "failed" || served.Fields[1]["count"] != "2" || served.Provenance.ExecutedQuery != rpc.Msg.GetProvenance().GetExecutedQuery() {
		t.Fatalf("status registry=%+v rpc=%+v err=%v", served, rpc.Msg, err)
	}
}

func TestRegistryAndTypedRPCShareTerminalRunTrend(t *testing.T) {
	now := time.Date(2026, time.July, 29, 12, 0, 0, 0, time.UTC)
	bucket := now.Add(-time.Hour)
	store := &fakeStore{runTimeSeries: []invocationreadmodel.RunTimeSeriesBucket{{Bucket: bucket, TerminalRuns: 3, CompletedRuns: 2, FailedRuns: 1, AvgDurationMS: 900}}}
	handler := NewHandler(store, func() time.Time { return now })
	rpc, err := handler.TerminalRunTrend(context.Background(), connect.NewRequest(&measurepb.TerminalRunTrendRequest{Window: token(sharedmeasurepb.TimeWindowToken_TIME_WINDOW_TOKEN_LAST_7D)}))
	if err != nil || len(rpc.Msg.GetRows()) != 1 || rpc.Msg.GetRows()[0].GetBucket() != bucket.Format(time.RFC3339) || rpc.Msg.GetRows()[0].GetCompletedRuns() != 2 || rpc.Msg.GetRows()[0].GetFailedRuns() != 1 || rpc.Msg.GetProvenance().GetExecutedQuery() == "" {
		t.Fatalf("trend rpc=%+v err=%v", rpc.Msg, err)
	}
	registry, err := handler.Registry()
	if err != nil {
		t.Fatal(err)
	}
	served, err := registry.Execute(context.Background(), measureRequest(TerminalRunTrend, "last_7d"))
	if err != nil || len(served.Fields) != 1 || served.Fields[0]["bucket"] != bucket.Format(time.RFC3339) || served.Fields[0]["terminal_runs"] != "3" || served.Provenance.ExecutedQuery != rpc.Msg.GetProvenance().GetExecutedQuery() {
		t.Fatalf("trend registry=%+v rpc=%+v err=%v", served, rpc.Msg, err)
	}
}

func TestRegistryAndTypedRPCShareErrorPatterns(t *testing.T) {
	now := time.Date(2026, time.July, 29, 12, 0, 0, 0, time.UTC)
	store := &fakeStore{errorPatterns: []invocationreadmodel.ErrorPattern{{ErrorCode: "runner_failed", Count: 2, LastSeen: now.Add(-time.Hour), SampleRunID: "run-2"}}}
	handler := NewHandler(store, func() time.Time { return now })
	rpc, err := handler.ErrorPatterns(context.Background(), connect.NewRequest(&measurepb.ErrorPatternsRequest{Window: token(sharedmeasurepb.TimeWindowToken_TIME_WINDOW_TOKEN_LAST_7D)}))
	if err != nil || len(rpc.Msg.GetRows()) != 1 || rpc.Msg.GetRows()[0].GetErrorCode() != "runner_failed" || rpc.Msg.GetRows()[0].GetCount() != 2 || rpc.Msg.GetRows()[0].GetSampleRunId() != "run-2" || rpc.Msg.GetProvenance().GetExecutedQuery() == "" {
		t.Fatalf("errors rpc=%+v err=%v", rpc.Msg, err)
	}
	registry, err := handler.Registry()
	if err != nil {
		t.Fatal(err)
	}
	served, err := registry.Execute(context.Background(), measureRequest(ErrorPatterns, "last_7d"))
	if err != nil || len(served.Fields) != 1 || served.Fields[0]["error_code"] != "runner_failed" || served.Fields[0]["count"] != "2" || served.Provenance.ExecutedQuery != rpc.Msg.GetProvenance().GetExecutedQuery() {
		t.Fatalf("errors registry=%+v rpc=%+v err=%v", served, rpc.Msg, err)
	}
}

func token(value sharedmeasurepb.TimeWindowToken) *sharedmeasurepb.TimeWindow {
	return &sharedmeasurepb.TimeWindow{Window: &sharedmeasurepb.TimeWindow_Token{Token: value}}
}

func measureRequest(name, window string) measurelib.MeasureRequest {
	return measurelib.MeasureRequest{Measure: name, Params: map[string]string{"window": window}}
}

func TestCapabilityEfficacyDefinitionNamesOnlyShownCounts(t *testing.T) {
	numerator := definitionFor("friction.capability_efficacy").Numerator
	for pattern, word := range map[string]string{"fallback-after-capability": "fallback", "capability-abandoned": "abandoned"} {
		if got := strings.Contains(numerator, word); got != runsignal.Shown(pattern) {
			t.Errorf("numerator %q mentions %q = %v, pattern shown = %v", numerator, word, got, runsignal.Shown(pattern))
		}
	}
}
