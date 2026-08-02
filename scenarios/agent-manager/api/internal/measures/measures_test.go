package measures

import (
	"context"
	"strconv"
	"testing"
	"time"

	"agent-manager/internal/invocationreadmodel"
	"connectrpc.com/connect"
	measurelib "github.com/vrooli/measures-go"
	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
	sharedmeasurepb "github.com/vrooli/vrooli/packages/proto/gen/go/measures/v1"
)

type fakeStore struct {
	metrics         invocationreadmodel.Metrics
	runMetrics      invocationreadmodel.RunMetrics
	durationStats   invocationreadmodel.RunDurationStatistics
	statusCounts    []invocationreadmodel.RunStatusCount
	breakdowns      map[string][]invocationreadmodel.RunBreakdownRow
	runTimeSeries   []invocationreadmodel.RunTimeSeriesBucket
	toolUsage       []invocationreadmodel.ToolUsageRow
	capabilityUsage []invocationreadmodel.CapabilityUsageRow
	errorPatterns   []invocationreadmodel.ErrorPattern
	findingMetrics  invocationreadmodel.FindingMetrics
	filter          invocationreadmodel.Filter
}

func (s *fakeStore) RunMetrics(_ context.Context, filter invocationreadmodel.Filter) (invocationreadmodel.RunMetrics, error) {
	s.filter = filter
	return s.runMetrics, nil
}

func (s *fakeStore) RunDurationStatistics(_ context.Context, filter invocationreadmodel.Filter) (invocationreadmodel.RunDurationStatistics, error) {
	s.filter = filter
	return s.durationStats, nil
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
	if rpc.Msg.GetShare() != 0.4 || rpc.Msg.GetUnknownCalls() != 3 || rpc.Msg.GetExecutedQuery() == "" {
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
	if served.Value != strconv.FormatFloat(rpc.Msg.GetShare(), 'f', -1, 64) || served.Provenance.ExecutedQuery != rpc.Msg.GetExecutedQuery() {
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
	store := &fakeStore{runMetrics: invocationreadmodel.RunMetrics{
		TotalTokens: 150, TerminalRuns: 5, SuccessfulRuns: 1, ConsumptionPerSuccessfulCompletion: 150,
	}}
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
	store := &fakeStore{runMetrics: invocationreadmodel.RunMetrics{TotalRuns: 8, TerminalRuns: 6, SuccessfulRuns: 3, SuccessRate: 0.5, CompletedDurationRuns: 5, AverageDurationMS: 1234, TotalCostUSD: 4.5, TotalTokens: 99, ReadCalls: 6, FileRereads: 2, FileRereadRate: 1.0 / 3.0}}
	handler := NewHandler(store, func() time.Time { return now })
	rpc, err := handler.RunSuccessRate(context.Background(), connect.NewRequest(&measurepb.RunSuccessRateRequest{Window: token(sharedmeasurepb.TimeWindowToken_TIME_WINDOW_TOKEN_LAST_7D)}))
	if err != nil || rpc.Msg.GetRate() != 0.5 || rpc.Msg.GetSuccessfulRuns() != 3 || rpc.Msg.GetTerminalRuns() != 6 || rpc.Msg.GetExecutedQuery() == "" {
		t.Fatalf("success rpc=%+v err=%v", rpc.Msg, err)
	}
	cycle, err := handler.RunCycleTime(context.Background(), connect.NewRequest(&measurepb.RunCycleTimeRequest{}))
	if err != nil || cycle.Msg.GetAverageDurationMs() != 1234 || cycle.Msg.GetCompletedDurationRuns() != 5 {
		t.Fatalf("cycle rpc=%+v err=%v", cycle.Msg, err)
	}
	cost, err := handler.RunCost(context.Background(), connect.NewRequest(&measurepb.RunCostRequest{}))
	if err != nil || cost.Msg.GetTotalCostUsd() != 4.5 || cost.Msg.GetTotalTokens() != 99 || cost.Msg.GetTotalRuns() != 8 {
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
	if err != nil || response.Msg.GetAverageDurationMs() != 100 || response.Msg.GetMinDurationMs() != 25 || response.Msg.GetMaxDurationMs() != 200 || response.Msg.GetCount() != 3 || response.Msg.GetExecutedQuery() == "" {
		t.Fatalf("duration response=%+v err=%v", response.Msg, err)
	}
}

func TestTypedFindingRecurrenceUsesSharedComputation(t *testing.T) {
	now := time.Date(2026, time.July, 29, 12, 0, 0, 0, time.UTC)
	store := &fakeStore{findingMetrics: invocationreadmodel.FindingMetrics{TotalFindings: 3, RecurringFindings: 2, RecurringFingerprints: 1, RecurrenceRate: 2.0 / 3.0}}
	handler := NewHandler(store, func() time.Time { return now })
	response, err := handler.FindingRecurrenceRate(context.Background(), connect.NewRequest(&measurepb.FindingRecurrenceRateRequest{}))
	if err != nil || response.Msg.GetRate() != 2.0/3.0 || response.Msg.GetRecurringFindings() != 2 || response.Msg.GetTotalFindings() != 3 || response.Msg.GetRecurringFingerprints() != 1 || response.Msg.GetExecutedQuery() == "" {
		t.Fatalf("recurrence response=%+v err=%v", response.Msg, err)
	}
}

func TestRegistryAndTypedRPCShareRunStatusDistribution(t *testing.T) {
	now := time.Date(2026, time.July, 29, 12, 0, 0, 0, time.UTC)
	store := &fakeStore{statusCounts: []invocationreadmodel.RunStatusCount{{Status: "complete", Count: 5}, {Status: "failed", Count: 2}}}
	handler := NewHandler(store, func() time.Time { return now })
	rpc, err := handler.RunStatusDistribution(context.Background(), connect.NewRequest(&measurepb.RunStatusDistributionRequest{Window: token(sharedmeasurepb.TimeWindowToken_TIME_WINDOW_TOKEN_LAST_7D)}))
	if err != nil || len(rpc.Msg.GetRows()) != 2 || rpc.Msg.GetRows()[0].GetStatus() != "complete" || rpc.Msg.GetRows()[0].GetCount() != 5 || rpc.Msg.GetExecutedQuery() == "" {
		t.Fatalf("status rpc=%+v err=%v", rpc.Msg, err)
	}
	registry, err := handler.Registry()
	if err != nil {
		t.Fatal(err)
	}
	served, err := registry.Execute(context.Background(), measureRequest(RunStatusDistribution, "last_7d"))
	if err != nil || len(served.Fields) != 2 || served.Fields[1]["status"] != "failed" || served.Fields[1]["count"] != "2" || served.Provenance.ExecutedQuery != rpc.Msg.GetExecutedQuery() {
		t.Fatalf("status registry=%+v rpc=%+v err=%v", served, rpc.Msg, err)
	}
}

func TestRegistryAndTypedRPCShareTerminalRunTrend(t *testing.T) {
	now := time.Date(2026, time.July, 29, 12, 0, 0, 0, time.UTC)
	bucket := now.Add(-time.Hour)
	store := &fakeStore{runTimeSeries: []invocationreadmodel.RunTimeSeriesBucket{{Bucket: bucket, TerminalRuns: 3, CompletedRuns: 2, FailedRuns: 1, TotalCostUSD: 1.25, AvgDurationMS: 900}}}
	handler := NewHandler(store, func() time.Time { return now })
	rpc, err := handler.TerminalRunTrend(context.Background(), connect.NewRequest(&measurepb.TerminalRunTrendRequest{Window: token(sharedmeasurepb.TimeWindowToken_TIME_WINDOW_TOKEN_LAST_7D)}))
	if err != nil || len(rpc.Msg.GetRows()) != 1 || rpc.Msg.GetRows()[0].GetBucket() != bucket.Format(time.RFC3339) || rpc.Msg.GetRows()[0].GetCompletedRuns() != 2 || rpc.Msg.GetRows()[0].GetFailedRuns() != 1 || rpc.Msg.GetExecutedQuery() == "" {
		t.Fatalf("trend rpc=%+v err=%v", rpc.Msg, err)
	}
	registry, err := handler.Registry()
	if err != nil {
		t.Fatal(err)
	}
	served, err := registry.Execute(context.Background(), measureRequest(TerminalRunTrend, "last_7d"))
	if err != nil || len(served.Fields) != 1 || served.Fields[0]["bucket"] != bucket.Format(time.RFC3339) || served.Fields[0]["terminal_runs"] != "3" || served.Provenance.ExecutedQuery != rpc.Msg.GetExecutedQuery() {
		t.Fatalf("trend registry=%+v rpc=%+v err=%v", served, rpc.Msg, err)
	}
}

func TestRegistryAndTypedRPCShareErrorPatterns(t *testing.T) {
	now := time.Date(2026, time.July, 29, 12, 0, 0, 0, time.UTC)
	store := &fakeStore{errorPatterns: []invocationreadmodel.ErrorPattern{{ErrorCode: "runner_failed", Count: 2, LastSeen: now.Add(-time.Hour), SampleRunID: "run-2"}}}
	handler := NewHandler(store, func() time.Time { return now })
	rpc, err := handler.ErrorPatterns(context.Background(), connect.NewRequest(&measurepb.ErrorPatternsRequest{Window: token(sharedmeasurepb.TimeWindowToken_TIME_WINDOW_TOKEN_LAST_7D)}))
	if err != nil || len(rpc.Msg.GetRows()) != 1 || rpc.Msg.GetRows()[0].GetErrorCode() != "runner_failed" || rpc.Msg.GetRows()[0].GetCount() != 2 || rpc.Msg.GetRows()[0].GetSampleRunId() != "run-2" || rpc.Msg.GetExecutedQuery() == "" {
		t.Fatalf("errors rpc=%+v err=%v", rpc.Msg, err)
	}
	registry, err := handler.Registry()
	if err != nil {
		t.Fatal(err)
	}
	served, err := registry.Execute(context.Background(), measureRequest(ErrorPatterns, "last_7d"))
	if err != nil || len(served.Fields) != 1 || served.Fields[0]["error_code"] != "runner_failed" || served.Fields[0]["count"] != "2" || served.Provenance.ExecutedQuery != rpc.Msg.GetExecutedQuery() {
		t.Fatalf("errors registry=%+v rpc=%+v err=%v", served, rpc.Msg, err)
	}
}

func token(value sharedmeasurepb.TimeWindowToken) *sharedmeasurepb.TimeWindow {
	return &sharedmeasurepb.TimeWindow{Window: &sharedmeasurepb.TimeWindow_Token{Token: value}}
}

func measureRequest(name, window string) measurelib.MeasureRequest {
	return measurelib.MeasureRequest{Measure: name, Params: map[string]string{"window": window}}
}
