package database

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/invocationreadmodel"
	"agent-manager/internal/runsignal"
	"agent-manager/internal/tokenaccounting"

	"github.com/google/uuid"
)

func TestInvocationReadModelReplaceIsAtomicAndIdempotent(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := &invocationReadModelRepository{db: db}
	now := time.Now().UTC()
	watermark := invocationreadmodel.Watermark{RunID: "run-1", LastEventID: "event-1", LastEventAt: now, ClassifierVersion: runsignal.InvocationFactVersion, ProjectedAt: now}
	facts := []invocationreadmodel.Fact{{InvocationFact: runsignal.InvocationFact{Version: runsignal.InvocationFactVersion, CallEventID: "call-1", ToolName: "shell", Ownership: "unknown", Outcome: "success", PairingBasis: "ordinal", Fingerprint: "fp", Availability: "available"}, RunID: "run-1", OccurredAt: now, TimeBasis: "call_event", ProfileID: "unknown", RunnerType: "unknown", Model: "unknown", Tag: "unknown", RunStatus: "complete"}}
	if err := repo.Replace(context.Background(), facts, watermark); err != nil {
		t.Fatal(err)
	}
	if err := repo.Replace(context.Background(), facts, watermark); err != nil {
		t.Fatal(err)
	}
	got, err := repo.Facts(context.Background(), "run-1")
	if err != nil || len(got) != 1 || got[0].CallEventID != "call-1" || got[0].PairingBasis != "ordinal" {
		t.Fatalf("facts=%+v err=%v", got, err)
	}
	gotWatermark, err := repo.Watermark(context.Background(), "run-1")
	if err != nil || gotWatermark == nil || gotWatermark.LastEventID != "event-1" {
		t.Fatalf("watermark=%+v err=%v", gotWatermark, err)
	}
}

func TestInvocationReadModelFactRoundTripsTokenFactors(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := &invocationReadModelRepository{db: db}
	now := time.Now().UTC()
	fact := invocationreadmodel.Fact{
		InvocationFact: runsignal.InvocationFact{
			Version: runsignal.InvocationFactVersion, CallEventID: "token-call", ToolName: "shell",
			Ownership: "resolved", Outcome: "success", Fingerprint: "token-fingerprint",
			Availability: "available", ArgTokens: 11, ResultTokens: 29,
			TokenBasis: tokenaccounting.BasisEstimated, ResidencyRequests: 3, ResidencySegment: 2, OutputEventID: "token-output", OutputBytes: 812,
			IncurredInputTokens: 11, IncurredOutputTokens: 13, IncurredCacheReadTokens: 3, IncurredCacheCreationTokens: 2,
		},
		RunID: "token-run", OccurredAt: now, TimeBasis: "call_event", ProfileID: "profile",
		RunnerType: "claude-code", Model: "model", Tag: "token", RunStatus: "complete",
	}
	watermark := invocationreadmodel.Watermark{RunID: "token-run", LastEventID: "token-call", LastEventAt: now, ClassifierVersion: runsignal.InvocationFactVersion, ProjectedAt: now}
	if err := repo.Replace(context.Background(), []invocationreadmodel.Fact{fact}, watermark); err != nil {
		t.Fatal(err)
	}
	got, err := repo.Facts(context.Background(), "token-run")
	if err != nil || len(got) != 1 {
		t.Fatalf("facts=%+v err=%v", got, err)
	}
	if got[0].ArgTokens != 11 || got[0].ResultTokens != 29 || got[0].TokenBasis != tokenaccounting.BasisEstimated || got[0].ResidencyRequests != 3 || got[0].OutputEventID != "token-output" || got[0].OutputBytes != 812 || got[0].ResidencySegment != 2 || got[0].IncurredInputTokens != 11 || got[0].IncurredOutputTokens != 13 || got[0].IncurredCacheReadTokens != 3 || got[0].IncurredCacheCreationTokens != 2 {
		t.Fatalf("token factors did not round-trip: %+v", got[0].InvocationFact)
	}
}

func TestTokenAttributionGroupsEveryDimensionAndView(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := &invocationReadModelRepository{db: db}
	now := time.Now().UTC()
	facts := []invocationreadmodel.Fact{
		{InvocationFact: runsignal.InvocationFact{Version: runsignal.InvocationFactVersion, CallEventID: "receipt-1", ToolName: "read", Capability: "filesystem", Executable: "cat", CommandPath: "read", Ownership: "project", Outcome: "success", Fingerprint: "one", Availability: "available", ArgTokens: 2, ResultTokens: 8, TokenBasis: tokenaccounting.BasisEstimated, ResidencyRequests: 2, IncurredInputTokens: 2, IncurredOutputTokens: 2}, RunID: "token-attribution", OccurredAt: now, TimeBasis: "call_event", RunStatus: "complete"},
		{InvocationFact: runsignal.InvocationFact{Version: runsignal.InvocationFactVersion, CallEventID: "call-2", ToolName: "shell", Capability: "process", Executable: "sh", CommandPath: "run", Ownership: "external", Outcome: "failure", Fingerprint: "two", Availability: "available", ArgTokens: 1, ResultTokens: 5, TokenBasis: tokenaccounting.BasisMeasured, ResidencyRequests: 1, IncurredInputTokens: 3, IncurredOutputTokens: 4}, RunID: "token-attribution", OccurredAt: now.Add(time.Second), TimeBasis: "call_event", RunStatus: "complete"},
	}
	if err := repo.Replace(context.Background(), facts, invocationreadmodel.Watermark{RunID: "token-attribution", LastEventID: "call-2", LastEventAt: now, ClassifierVersion: runsignal.InvocationFactVersion, ProjectedAt: now}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO investigation_cross_scenario_calls (run_id,receipt_event_id,occurred_at,target_scenario,operation,outcome,status_code,duration_ms,verified,ledger_availability) VALUES (?,?,?,?,?,?,?,?,?,?)`, "token-attribution", "receipt-1", SQLiteTime(now), "scenario-a", "op", "success", 200, 1, 3, "available"); err != nil {
		t.Fatal(err)
	}
	for _, groupBy := range []string{"capability", "executable", "command_path", "target_scenario_operation"} {
		rows, err := repo.TokenAttribution(context.Background(), invocationreadmodel.Filter{}, groupBy, "footprint", 20)
		if err != nil || len(rows) == 0 {
			t.Fatalf("group_by=%s rows=%+v err=%v", groupBy, rows, err)
		}
		for _, row := range rows {
			if row.GroupBy != groupBy || row.TotalTokens == 0 || row.EstimatedTokenShare < 0 || row.EstimatedTokenShare > 1 {
				t.Fatalf("group_by=%s row=%+v", groupBy, row)
			}
		}
	}
	for _, view := range []string{"footprint", "residency", "incurred"} {
		rows, err := repo.TokenAttribution(context.Background(), invocationreadmodel.Filter{}, "capability", view, 20)
		if err != nil || len(rows) != 2 {
			t.Fatalf("view=%s rows=%+v err=%v", view, rows, err)
		}
	}
}

func TestInvocationReadModelProjectionRollsBackEveryArtifactWhenSpanWriteFails(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()
	repo := &invocationReadModelRepository{db: db}
	now := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	project := func(eventID string) error {
		fact := invocationreadmodel.Fact{InvocationFact: runsignal.InvocationFact{Version: "facts.v1", CallEventID: eventID, ToolName: "shell", Ownership: "project", Outcome: "failure", Fingerprint: eventID, Availability: "available"}, RunID: "atomic-run", OccurredAt: now, TimeBasis: "call_event", ProfileID: "profile-a", RunnerType: "codex", Model: "test", Tag: "atomic", RunStatus: "complete"}
		span := invocationreadmodel.SelfReportSpan{SelfReportSpan: runsignal.SelfReportSpan{ClassifierVersion: runsignal.SelfReportClassifierVersion, EventID: eventID, RuleID: "blocked", CauseScope: "run-execution", StartOffset: 0, EndOffset: 7, Text: "blocked"}, RunID: "atomic-run", OccurredAt: now, TimeBasis: "message_event", ProfileID: "profile-a", RunnerType: "codex", Model: "test", Tag: "atomic", RunStatus: "complete"}
		watermark := invocationreadmodel.Watermark{RunID: "atomic-run", LastEventID: eventID, LastEventAt: now, ClassifierVersion: "facts.v1", SelfReportClassifierVersion: runsignal.SelfReportClassifierVersion, ProjectedAt: now}
		return repo.ReplaceProjection(ctx, []invocationreadmodel.Fact{fact}, []invocationreadmodel.SelfReportSpan{span}, watermark, invocationreadmodel.RunFact{RunID: "atomic-run", OccurredAt: now, CreatedAt: now, Status: "complete", ProfileID: "profile-a", RunnerType: "codex", Model: "test", Tag: "atomic", CostTimeBasis: "ended_at", ProjectedAt: now})
	}
	if err := project("before"); err != nil {
		t.Fatal(err)
	}
	var projectionComplete int
	if err := db.Get(&projectionComplete, `SELECT projection_complete FROM invocation_read_model_watermarks WHERE run_id='atomic-run'`); err != nil || projectionComplete != 1 {
		t.Fatalf("projection completion marker = %d, %v", projectionComplete, err)
	}
	if _, err := db.Exec(`CREATE TRIGGER fail_projection_span BEFORE INSERT ON invocation_read_model_self_report_spans WHEN NEW.event_id = 'after' BEGIN SELECT RAISE(ABORT, 'span failure'); END`); err != nil {
		t.Fatal(err)
	}
	if err := project("after"); err == nil {
		t.Fatal("expected span write failure")
	}
	facts, err := repo.Facts(ctx, "atomic-run")
	if err != nil || len(facts) != 1 || facts[0].CallEventID != "before" {
		t.Fatalf("facts=%+v err=%v", facts, err)
	}
	spans, err := repo.SelfReportSpansForRun(ctx, "atomic-run")
	if err != nil || len(spans) != 1 || spans[0].EventID != "before" {
		t.Fatalf("spans=%+v err=%v", spans, err)
	}
	watermark, err := repo.Watermark(ctx, "atomic-run")
	if err != nil || watermark == nil || watermark.LastEventID != "before" {
		t.Fatalf("watermark=%+v err=%v", watermark, err)
	}
}

func TestInvocationReadModelReplaceRunUpsertsTerminalSummary(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := &invocationReadModelRepository{db: db}
	created := time.Date(2026, 7, 29, 18, 0, 0, 0, time.UTC)
	started, ended := created.Add(time.Minute), created.Add(3*time.Minute)
	fact := invocationreadmodel.RunFact{RunID: "run-throughput", OccurredAt: ended, CreatedAt: created, StartedAt: &started, EndedAt: &ended, DurationMS: 120000, Status: "complete", ProfileID: "profile-1", RunnerType: "codex", Model: "gpt-test", Tag: "analytics", TotalCostUSD: 1.25, TotalTokens: 42, TimeAccounting: runsignal.TimeAccounting{ModelGeneratingMS: 60000, ToolExecutingMS: 30000, IdleWaitingMS: 20000, AwaitingHumanMS: 10000, ModelTokens: 42}, CostTimeBasis: "ended_at", ProjectedAt: ended.Add(time.Minute)}
	if err := repo.ReplaceRun(context.Background(), fact); err != nil {
		t.Fatal(err)
	}
	fact.TotalCostUSD, fact.TotalTokens = 2.5, 99
	if err := repo.ReplaceRun(context.Background(), fact); err != nil {
		t.Fatal(err)
	}
	var got struct {
		Cost   float64 `db:"total_cost_usd"`
		Tokens int64   `db:"total_tokens"`
		Basis  string  `db:"cost_time_basis"`
	}
	if err := db.Get(&got, `SELECT total_cost_usd,total_tokens,cost_time_basis FROM invocation_read_model_runs WHERE run_id='run-throughput'`); err != nil {
		t.Fatal(err)
	}
	if got.Cost != 2.5 || got.Tokens != 99 || got.Basis != "ended_at" {
		t.Fatalf("stored terminal summary=%+v", got)
	}
	accounting, present, err := repo.TimeAccountingForRun(context.Background(), "run-throughput")
	if err != nil || !present || accounting.DurationMS() != 120000 || accounting.ModelTokens != 42 {
		t.Fatalf("stored time accounting=%+v present=%v err=%v", accounting, present, err)
	}
}

func TestInvocationReadModelRunMetricsUseDurableTerminalFactsAndSharedFilters(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()
	repo := &invocationReadModelRepository{db: db}
	now := time.Date(2026, 7, 29, 20, 0, 0, 0, time.UTC)
	seed := func(id, status, profile string, offset time.Duration, cost float64, tokens int64) {
		started, ended := now.Add(offset), now.Add(offset+2*time.Minute)
		if err := repo.ReplaceProjection(ctx, []invocationreadmodel.Fact{{InvocationFact: runsignal.InvocationFact{Version: "v1", CallEventID: id + "-call", ToolName: "shell", Ownership: map[string]string{"run-a": "external", "run-b": "project"}[id], Outcome: "success", Fingerprint: id, Availability: "available"}, RunID: id, OccurredAt: started, TimeBasis: "call_event", ProfileID: profile, RunnerType: "codex", Model: "gpt-test", Tag: "analytics", RunStatus: status}}, nil, invocationreadmodel.Watermark{RunID: id, LastEventID: id + "-call", LastEventAt: started, ClassifierVersion: "v1", ProjectedAt: now}, invocationreadmodel.RunFact{RunID: id, OccurredAt: ended, CreatedAt: started.Add(-time.Minute), StartedAt: &started, EndedAt: &ended, DurationMS: 120000, Status: status, ProfileID: profile, RunnerType: "codex", Model: "gpt-test", Tag: "analytics", TotalCostUSD: cost, InputCostUSD: cost / 2, OutputCostUSD: cost / 4, CacheReadCostUSD: cost / 8, CacheCreationCostUSD: cost / 8, TotalTokens: tokens, InputTokens: tokens / 2, OutputTokens: tokens / 4, CacheReadTokens: tokens / 8, CacheCreationTokens: tokens / 8, CostTimeBasis: "ended_at", ProjectedAt: now}); err != nil {
			t.Fatal(err)
		}
	}
	seed("run-a", "complete", "profile-a", 0, 1.5, 10)
	seed("run-b", "failed", "profile-b", time.Minute, 0.5, 20)
	metrics, err := repo.RunMetrics(ctx, invocationreadmodel.Filter{From: &now, To: ptrTime(now.Add(time.Hour))})
	if err != nil || metrics.TotalRuns != 2 || metrics.TerminalRuns != 2 || metrics.SuccessfulRuns != 1 || metrics.SuccessRate != 0.5 || metrics.AverageDurationMS != 120000 {
		t.Fatalf("metrics=%+v err=%v", metrics, err)
	}
	durationStats, err := repo.RunDurationStatistics(ctx, invocationreadmodel.Filter{From: &now, To: ptrTime(now.Add(time.Hour))})
	if err != nil || durationStats.AverageDurationMS != 120000 || durationStats.P50DurationMS != 120000 || durationStats.P95DurationMS != 120000 || durationStats.P99DurationMS != 120000 || durationStats.MinDurationMS != 120000 || durationStats.MaxDurationMS != 120000 || durationStats.Count != 2 {
		t.Fatalf("duration stats=%+v err=%v", durationStats, err)
	}
	statusRows, err := repo.RunStatusCounts(ctx, invocationreadmodel.Filter{From: &now, To: ptrTime(now.Add(time.Hour))})
	if err != nil || len(statusRows) != 2 || statusRows[0].Status != "complete" || statusRows[0].Count != 1 || statusRows[1].Status != "failed" || statusRows[1].Count != 1 {
		t.Fatalf("status rows=%+v err=%v", statusRows, err)
	}
	runnerRows, err := repo.RunBreakdown(ctx, invocationreadmodel.Filter{From: &now, To: ptrTime(now.Add(time.Hour))}, "runner", 10)
	if err != nil || len(runnerRows) != 1 || runnerRows[0].Key != "codex" || runnerRows[0].Value != "codex" || runnerRows[0].RunCount != 2 || runnerRows[0].SuccessCount != 1 || runnerRows[0].FailedCount != 1 {
		t.Fatalf("runner rows=%+v err=%v", runnerRows, err)
	}
	trendRows, err := repo.RunTimeSeries(ctx, invocationreadmodel.Filter{From: &now, To: ptrTime(now.Add(time.Hour))}, time.Hour)
	if err != nil || len(trendRows) != 1 || !trendRows[0].Bucket.Equal(now) || trendRows[0].TerminalRuns != 2 || trendRows[0].CompletedRuns != 1 || trendRows[0].FailedRuns != 1 || trendRows[0].CancelledRuns != 0 || trendRows[0].AvgDurationMS != 120000 {
		t.Fatalf("trend rows=%+v err=%v", trendRows, err)
	}
	toolRows, err := repo.ToolUsage(ctx, invocationreadmodel.Filter{From: &now, To: ptrTime(now.Add(time.Hour))}, 10)
	if err != nil || len(toolRows) != 1 || toolRows[0].ToolName != "shell" || toolRows[0].CallCount != 2 || toolRows[0].SuccessCount != 2 || toolRows[0].FailedCount != 0 {
		t.Fatalf("tool rows=%+v err=%v", toolRows, err)
	}
	filtered, err := repo.RunMetrics(ctx, invocationreadmodel.Filter{From: &now, To: ptrTime(now.Add(time.Hour)), Ownership: "external"})
	if err != nil || filtered.TotalRuns != 1 || filtered.SuccessfulRuns != 1 {
		t.Fatalf("ownership-filtered metrics=%+v err=%v", filtered, err)
	}
	empty, err := repo.RunMetrics(ctx, invocationreadmodel.Filter{From: ptrTime(now.Add(2 * time.Hour)), To: ptrTime(now.Add(3 * time.Hour))})
	if err != nil || empty.TotalRuns != 0 || empty.AverageDurationMS != 0 {
		t.Fatalf("empty metrics=%+v err=%v", empty, err)
	}
}

func ptrTime(value time.Time) *time.Time { return &value }

func TestInvocationReadModelCohortFiltersByFactDimensions(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()
	repo := &invocationReadModelRepository{db: db}
	now := time.Now().UTC()
	facts := []invocationreadmodel.Fact{
		{InvocationFact: runsignal.InvocationFact{Version: "v1", CallEventID: "a", ToolName: "shell", Ownership: "external", Outcome: "failure", Fingerprint: "fp-a", Availability: "available"}, RunID: "run-a", OccurredAt: now, TimeBasis: "call_event", ProfileID: "p", RunnerType: "codex", Model: "m", Tag: "keep", RunStatus: "complete"},
		{InvocationFact: runsignal.InvocationFact{Version: "v1", CallEventID: "b", ToolName: "shell", Ownership: "project", Outcome: "success", Fingerprint: "fp-b", Availability: "available"}, RunID: "run-b", OccurredAt: now, TimeBasis: "call_event", ProfileID: "p", RunnerType: "codex", Model: "m", Tag: "skip", RunStatus: "complete"},
	}
	for _, fact := range facts {
		if err := repo.Replace(ctx, []invocationreadmodel.Fact{fact}, invocationreadmodel.Watermark{RunID: fact.RunID, LastEventID: fact.CallEventID, LastEventAt: now, ClassifierVersion: "v1", ProjectedAt: now}); err != nil {
			t.Fatal(err)
		}
	}
	filter := invocationreadmodel.Filter{Ownership: "external"}
	cohort, err := repo.Cohort(ctx, filter, 10)
	if err != nil || len(cohort.RunIDs) != 1 || cohort.RunIDs[0] != "run-a" || cohort.Truncated {
		t.Fatalf("cohort=%+v err=%v", cohort, err)
	}
}

func TestReceiptEvidenceRepositoryRebuildKeepsOnlyVerifiedIDs(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := &receiptEvidenceRepository{db: db}
	runID := uuid.New()

	if err := repo.ReplaceReceiptEvidence(context.Background(), runID, "available", []string{"event-b", "event-a"}); err != nil {
		t.Fatal(err)
	}
	if err := repo.ReplaceReceiptEvidence(context.Background(), runID, "unobserved", nil); err != nil {
		t.Fatal(err)
	}
	got, err := repo.ReceiptEvidence(context.Background(), runID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("receipt evidence=%v, want no stale identifiers", got)
	}
}

func TestSelfReportSpansForRunIgnoresRetiredClassifierVersions(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()
	repo := &invocationReadModelRepository{db: db}
	now := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	span := func(version, event string) invocationreadmodel.SelfReportSpan {
		return invocationreadmodel.SelfReportSpan{SelfReportSpan: runsignal.SelfReportSpan{ClassifierVersion: version, EventID: event, RuleID: "blocked", Text: "blocked"}, RunID: "r", OccurredAt: now, TimeBasis: "message_event", RunStatus: "complete"}
	}
	watermark := invocationreadmodel.Watermark{RunID: "r", LastEventID: "e", LastEventAt: now, ProjectedAt: now}
	err := repo.ReplaceProjection(ctx, nil, []invocationreadmodel.SelfReportSpan{span("self-report.v0-retired", "stale"), span(runsignal.SelfReportClassifierVersion, "current")}, watermark, invocationreadmodel.RunFact{RunID: "r", OccurredAt: now, CreatedAt: now, Status: "complete", CostTimeBasis: "ended_at", ProjectedAt: now})
	if err != nil {
		t.Fatal(err)
	}
	spans, err := repo.SelfReportSpansForRun(ctx, "r")
	if err != nil || len(spans) != 1 || spans[0].EventID != "current" {
		t.Fatalf("spans=%+v err=%v; only current-classifier spans may be served", spans, err)
	}
}
