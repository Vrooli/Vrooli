package database

import (
	"context"
	"fmt"
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
			TokenBasis: tokenaccounting.BasisEstimated, ResidencyTurns: 3, ResidencySegment: 2,
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
	if got[0].ArgTokens != 11 || got[0].ResultTokens != 29 || got[0].TokenBasis != tokenaccounting.BasisEstimated || got[0].ResidencyTurns != 3 || got[0].ResidencySegment != 2 || got[0].IncurredInputTokens != 11 || got[0].IncurredOutputTokens != 13 || got[0].IncurredCacheReadTokens != 3 || got[0].IncurredCacheCreationTokens != 2 {
		t.Fatalf("token factors did not round-trip: %+v", got[0].InvocationFact)
	}
}

func TestTokenAttributionGroupsEveryDimensionAndView(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := &invocationReadModelRepository{db: db}
	now := time.Now().UTC()
	facts := []invocationreadmodel.Fact{
		{InvocationFact: runsignal.InvocationFact{Version: runsignal.InvocationFactVersion, CallEventID: "receipt-1", ToolName: "read", Capability: "filesystem", Executable: "cat", CommandPath: "read", Ownership: "project", Outcome: "success", Fingerprint: "one", Availability: "available", ArgTokens: 2, ResultTokens: 8, TokenBasis: tokenaccounting.BasisEstimated, ResidencyTurns: 2, IncurredInputTokens: 2, IncurredOutputTokens: 2}, RunID: "token-attribution", OccurredAt: now, TimeBasis: "call_event", RunStatus: "complete"},
		{InvocationFact: runsignal.InvocationFact{Version: runsignal.InvocationFactVersion, CallEventID: "call-2", ToolName: "shell", Capability: "process", Executable: "sh", CommandPath: "run", Ownership: "external", Outcome: "failure", Fingerprint: "two", Availability: "available", ArgTokens: 1, ResultTokens: 5, TokenBasis: tokenaccounting.BasisMeasured, ResidencyTurns: 1, IncurredInputTokens: 3, IncurredOutputTokens: 4}, RunID: "token-attribution", OccurredAt: now.Add(time.Second), TimeBasis: "call_event", RunStatus: "complete"},
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
		episode := invocationreadmodel.Episode{FrictionEpisode: runsignal.FrictionEpisode{EpisodeID: "episode-" + eventID, ClassifierVersion: "episodes.v1", Pattern: "command-failure", CauseScope: "toolchain", Severity: "high", HonestyFlags: []string{}, StartEventID: eventID, EndEventID: eventID, EvidenceEventIDs: []string{eventID}, OwnerConfidence: "heuristic", Fingerprint: eventID}, RunID: "atomic-run", OccurredAt: now, TimeBasis: "start_event", ProfileID: "profile-a", RunnerType: "codex", Model: "test", Tag: "atomic", RunStatus: "complete"}
		span := invocationreadmodel.SelfReportSpan{SelfReportSpan: runsignal.SelfReportSpan{ClassifierVersion: "spans.v1", EventID: eventID, RuleID: "blocked", CauseScope: "run-execution", StartOffset: 0, EndOffset: 7, Text: "blocked"}, RunID: "atomic-run", OccurredAt: now, TimeBasis: "message_event", ProfileID: "profile-a", RunnerType: "codex", Model: "test", Tag: "atomic", RunStatus: "complete"}
		watermark := invocationreadmodel.Watermark{RunID: "atomic-run", LastEventID: eventID, LastEventAt: now, ClassifierVersion: "facts.v1", EpisodeClassifierVersion: "episodes.v1", SelfReportClassifierVersion: "spans.v1", ProjectedAt: now}
		return repo.ReplaceProjection(ctx, []invocationreadmodel.Fact{fact}, nil, []invocationreadmodel.Episode{episode}, []invocationreadmodel.SelfReportSpan{span}, watermark, invocationreadmodel.RunFact{RunID: "atomic-run", OccurredAt: now, CreatedAt: now, Status: "complete", ProfileID: "profile-a", RunnerType: "codex", Model: "test", Tag: "atomic", CostTimeBasis: "ended_at", ProjectedAt: now})
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
	episodes, err := repo.EpisodesForRun(ctx, "atomic-run")
	if err != nil || len(episodes) != 1 || episodes[0].EpisodeID != "episode-before" {
		t.Fatalf("episodes=%+v err=%v", episodes, err)
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

func TestInvocationReadModelEpisodesUseSharedDimensionsAndEpisodeFilter(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := &invocationReadModelRepository{db: db}
	now := time.Date(2026, 7, 30, 13, 0, 0, 0, time.UTC)
	episode := invocationreadmodel.Episode{FrictionEpisode: runsignal.FrictionEpisode{EpisodeID: "episode-1", ClassifierVersion: "episodes.v1", Pattern: "command-failure", CauseScope: "run-execution", Severity: "medium", HonestyFlags: []string{}, StartEventID: "event-1", EndEventID: "event-1", EvidenceEventIDs: []string{"event-1"}, CycleCount: 2, RepeatedElement: "redacted-element", OwnerConfidence: "heuristic", Fingerprint: "episode-fp"}, RunID: "episode-run", OccurredAt: now, TimeBasis: "start_event", ProfileID: "profile-1", RunnerType: "codex", Model: "gpt-test", Tag: "cohort-a", RunStatus: "complete"}
	if err := repo.ReplaceProjection(context.Background(), nil, nil, []invocationreadmodel.Episode{episode}, nil, invocationreadmodel.Watermark{RunID: "episode-run", LastEventID: "event-1", LastEventAt: now, ClassifierVersion: "facts.v1", EpisodeClassifierVersion: "episodes.v1", ProjectedAt: now}, invocationreadmodel.RunFact{RunID: "episode-run", OccurredAt: now, CreatedAt: now, Status: "complete", ProfileID: "profile-1", RunnerType: "codex", Model: "gpt-test", Tag: "cohort-a", CostTimeBasis: "ended_at", ProjectedAt: now}); err != nil {
		t.Fatal(err)
	}
	got, err := repo.Episodes(context.Background(), invocationreadmodel.Filter{From: &now, To: ptrTime(now.Add(time.Hour)), ProfileID: "profile-1", RunnerType: "codex", Model: "gpt-test", TagPrefix: "cohort", EpisodePattern: "command-failure", EpisodeCauseScope: "run-execution", EpisodeFingerprint: "episode-fp"}, 10)
	if err != nil || len(got) != 1 || got[0].EpisodeID != "episode-1" || got[0].CycleCount != 2 || got[0].RepeatedElement != "redacted-element" {
		t.Fatalf("episodes=%+v err=%v", got, err)
	}
}

func TestInvocationReadModelReplaceRunUpsertsTerminalSummary(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := &invocationReadModelRepository{db: db}
	created := time.Date(2026, 7, 29, 18, 0, 0, 0, time.UTC)
	started, ended := created.Add(time.Minute), created.Add(3*time.Minute)
	fact := invocationreadmodel.RunFact{RunID: "run-throughput", OccurredAt: ended, CreatedAt: created, StartedAt: &started, EndedAt: &ended, DurationMS: 120000, Status: "complete", ProfileID: "profile-1", RunnerType: "codex", Model: "gpt-test", Tag: "analytics", TotalCostUSD: 1.25, TotalTokens: 42, ReadCalls: 4, FileRereads: 1, TimeAccounting: runsignal.TimeAccounting{ModelGeneratingMS: 60000, ToolExecutingMS: 30000, IdleWaitingMS: 20000, AwaitingHumanMS: 10000, ModelTokens: 42}, CostTimeBasis: "ended_at", ProjectedAt: ended.Add(time.Minute)}
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
	var signals struct{ Reads, Rereads int64 }
	if err := db.Get(&signals, `SELECT read_calls AS reads,files_read_more_than_once AS rereads FROM invocation_read_model_run_signals WHERE run_id='run-throughput'`); err != nil {
		t.Fatal(err)
	}
	if signals.Reads != 4 || signals.Rereads != 1 {
		t.Fatalf("stored terminal signals=%+v", signals)
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
		errors := []invocationreadmodel.ErrorFact{}
		if id == "run-b" {
			errors = append(errors, invocationreadmodel.ErrorFact{RunID: id, EventID: id + "-error", OccurredAt: ended, TimeBasis: "error_event", ErrorCode: "runner_failed", ProfileID: profile, RunnerType: "codex", Model: "gpt-test", Tag: "analytics"})
		}
		if err := repo.ReplaceProjection(ctx, []invocationreadmodel.Fact{{InvocationFact: runsignal.InvocationFact{Version: "v1", CallEventID: id + "-call", ToolName: "shell", Ownership: map[string]string{"run-a": "external", "run-b": "project"}[id], Outcome: "success", Fingerprint: id, Availability: "available"}, RunID: id, OccurredAt: started, TimeBasis: "call_event", ProfileID: profile, RunnerType: "codex", Model: "gpt-test", Tag: "analytics", RunStatus: status}}, errors, nil, nil, invocationreadmodel.Watermark{RunID: id, LastEventID: id + "-call", LastEventAt: started, ClassifierVersion: "v1", ProjectedAt: now}, invocationreadmodel.RunFact{RunID: id, OccurredAt: ended, CreatedAt: started.Add(-time.Minute), StartedAt: &started, EndedAt: &ended, DurationMS: 120000, Status: status, ProfileID: profile, RunnerType: "codex", Model: "gpt-test", Tag: "analytics", TotalCostUSD: cost, InputCostUSD: cost / 2, OutputCostUSD: cost / 4, CacheReadCostUSD: cost / 8, CacheCreationCostUSD: cost / 8, TotalTokens: tokens, InputTokens: tokens / 2, OutputTokens: tokens / 4, CacheReadTokens: tokens / 8, CacheCreationTokens: tokens / 8, ReadCalls: map[string]int64{"run-a": 2, "run-b": 1}[id], FileRereads: map[string]int64{"run-a": 1, "run-b": 0}[id], CostTimeBasis: "ended_at", ProjectedAt: now}); err != nil {
			t.Fatal(err)
		}
	}
	seed("run-a", "complete", "profile-a", 0, 1.5, 10)
	seed("run-b", "failed", "profile-b", time.Minute, 0.5, 20)
	metrics, err := repo.RunMetrics(ctx, invocationreadmodel.Filter{From: &now, To: ptrTime(now.Add(time.Hour))})
	if err != nil || metrics.TotalRuns != 2 || metrics.TerminalRuns != 2 || metrics.SuccessfulRuns != 1 || metrics.SuccessRate != 0.5 || metrics.AverageDurationMS != 120000 || metrics.ReadCalls != 3 || metrics.FileRereads != 1 || metrics.FileRereadRate != 1.0/3.0 {
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
	errorRows, err := repo.ErrorPatterns(ctx, invocationreadmodel.Filter{From: &now, To: ptrTime(now.Add(time.Hour))}, 10)
	if err != nil || len(errorRows) != 1 || errorRows[0].ErrorCode != "runner_failed" || errorRows[0].Count != 1 || errorRows[0].SampleRunID != "run-b" {
		t.Fatalf("error rows=%+v err=%v", errorRows, err)
	}
	filtered, err := repo.RunMetrics(ctx, invocationreadmodel.Filter{From: &now, To: ptrTime(now.Add(time.Hour)), Ownership: "external"})
	if err != nil || filtered.TotalRuns != 1 || filtered.SuccessfulRuns != 1 {
		t.Fatalf("ownership-filtered metrics=%+v err=%v", filtered, err)
	}
	empty, err := repo.RunMetrics(ctx, invocationreadmodel.Filter{From: ptrTime(now.Add(2 * time.Hour)), To: ptrTime(now.Add(3 * time.Hour))})
	if err != nil || empty.TotalRuns != 0 || empty.AverageDurationMS != 0 {
		t.Fatalf("empty metrics=%+v err=%v", empty, err)
	}
	for _, row := range []struct{ runID, fingerprint string }{{"run-a", "repeat"}, {"run-b", "repeat"}, {"run-b", "unique"}} {
		if _, err := db.ExecContext(ctx, `INSERT INTO run_findings (id,run_id,investigation_run_id,category,severity,recommendation_text,evidence,target_path,fingerprint,created_at) VALUES (?,?,?,?,?,?,?,?,?,?)`, uuid.New().String(), row.runID, row.runID, "efficiency", "warning", "fixture", "", "", row.fingerprint, SQLiteTime(now.Add(5*time.Minute))); err != nil {
			t.Fatal(err)
		}
	}
	findingMetrics, err := repo.FindingMetrics(ctx, invocationreadmodel.Filter{From: &now, To: ptrTime(now.Add(time.Hour))})
	if err != nil || findingMetrics.TotalFindings != 3 || findingMetrics.RecurringFindings != 2 || findingMetrics.RecurringFingerprints != 1 || findingMetrics.RecurrenceRate != 2.0/3.0 {
		t.Fatalf("finding metrics=%+v err=%v", findingMetrics, err)
	}
	filteredFindings, err := repo.FindingMetrics(ctx, invocationreadmodel.Filter{From: &now, To: ptrTime(now.Add(time.Hour)), Ownership: "external"})
	if err != nil || filteredFindings.TotalFindings != 1 || filteredFindings.RecurringFindings != 0 {
		t.Fatalf("filtered finding metrics=%+v err=%v", filteredFindings, err)
	}
}

func ptrTime(value time.Time) *time.Time { return &value }

func TestInvocationReadModelCohortAndMetricsShareFilter(t *testing.T) {
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
	metrics, err := repo.Metrics(ctx, invocationreadmodel.Filter{})
	if err != nil || metrics.TotalCalls != 2 || metrics.ResolvedCalls != 2 || metrics.ExternalCalls != 1 || metrics.FailedCalls != 1 || metrics.ExternalToolShare != 0.5 {
		t.Fatalf("metrics=%+v err=%v", metrics, err)
	}
}

func TestInvocationReadModelExternalShareDoesNotTreatUnknownAsExternal(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()
	repo := &invocationReadModelRepository{db: db}
	now := time.Now().UTC()
	fact := invocationreadmodel.Fact{
		InvocationFact: runsignal.InvocationFact{Version: "v1", CallEventID: "unknown-call", ToolName: "unknown", Ownership: "unknown", Outcome: "unknown", Fingerprint: "unknown", Availability: "unknown"},
		RunID:          "unknown-run",
		OccurredAt:     now,
		TimeBasis:      "call_event",
		ProfileID:      "unknown",
		RunnerType:     "unknown",
		Model:          "unknown",
		Tag:            "unknown",
		RunStatus:      "complete",
	}
	if err := repo.Replace(ctx, []invocationreadmodel.Fact{fact}, invocationreadmodel.Watermark{RunID: fact.RunID, LastEventID: fact.CallEventID, LastEventAt: now, ClassifierVersion: "v1", ProjectedAt: now}); err != nil {
		t.Fatal(err)
	}
	metrics, err := repo.Metrics(ctx, invocationreadmodel.Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if metrics.TotalCalls != 1 || metrics.ResolvedCalls != 0 || metrics.UnknownCalls != 1 || metrics.ExternalCalls != 0 || metrics.ExternalToolShare != 0 {
		t.Fatalf("unknown ownership changed external denominator: %+v", metrics)
	}
}

func TestInvocationReadModelDoesNotCallGenuineNonCommandUnclassified(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()
	repo := &invocationReadModelRepository{db: db}
	now := time.Now().UTC()
	fact := invocationreadmodel.Fact{
		InvocationFact: runsignal.InvocationFact{Version: "v1", CallEventID: "non-command", ToolName: "Read", Ownership: runsignal.OwnershipNotACommand, Outcome: "unknown", Availability: "available"},
		RunID:          "non-command-run", OccurredAt: now, TimeBasis: "call_event", ProfileID: "unknown", RunnerType: "codex", Model: "unknown", Tag: "tag", RunStatus: "complete",
	}
	if err := repo.Replace(ctx, []invocationreadmodel.Fact{fact}, invocationreadmodel.Watermark{RunID: fact.RunID, LastEventID: fact.CallEventID, LastEventAt: now, ClassifierVersion: "v1", ProjectedAt: now}); err != nil {
		t.Fatal(err)
	}
	metrics, err := repo.Metrics(ctx, invocationreadmodel.Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if metrics.NotACommandCalls != 1 || metrics.UnclassifiedCalls != 0 || metrics.UnknownCalls != 0 || metrics.UnclassifiedShare != 0 {
		t.Fatalf("metrics=%+v", metrics)
	}
}

func TestInvocationReadModelFrictionRatesExcludeUnclassifiedFacts(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()
	repo := &invocationReadModelRepository{db: db}
	now := time.Now().UTC()
	facts := []invocationreadmodel.Fact{
		{InvocationFact: runsignal.InvocationFact{Version: "v1", CallEventID: "classified", ToolName: "shell", Ownership: "project", Outcome: "success", Fingerprint: "same", Availability: "available"}, RunID: "run", OccurredAt: now, TimeBasis: "call_event", ProfileID: "p", RunnerType: "codex", Model: "m", Tag: "tag", RunStatus: "complete"},
		{InvocationFact: runsignal.InvocationFact{Version: "v1", CallEventID: "classified-retry", ToolName: "shell", Ownership: "project", Outcome: "success", RetryOfCallEventID: "classified", Fingerprint: "retry", Availability: "available"}, RunID: "run", OccurredAt: now.Add(time.Second), TimeBasis: "call_event", ProfileID: "p", RunnerType: "codex", Model: "m", Tag: "tag", RunStatus: "complete"},
		{InvocationFact: runsignal.InvocationFact{Version: "v1", CallEventID: "unknown", ToolName: "unknown", Ownership: "unknown", Outcome: "unknown", RetryOfCallEventID: "classified", Fingerprint: "same", Availability: "unknown"}, RunID: "run", OccurredAt: now.Add(2 * time.Second), TimeBasis: "call_event", ProfileID: "p", RunnerType: "codex", Model: "m", Tag: "tag", RunStatus: "complete"},
	}
	if err := repo.Replace(ctx, facts, invocationreadmodel.Watermark{RunID: "run", LastEventID: "unknown", LastEventAt: now.Add(2 * time.Second), ClassifierVersion: "v1", ProjectedAt: now}); err != nil {
		t.Fatal(err)
	}
	metrics, err := repo.Metrics(ctx, invocationreadmodel.Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if metrics.ClassifiedBase != 2 || metrics.RetryCalls != 1 || metrics.RepeatedCalls != 1 || metrics.RetryRate != 0.5 || metrics.RepeatedWorkRate != 0.5 {
		t.Fatalf("friction rates included unclassified evidence: %+v", metrics)
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

func TestInvocationReadModelEpisodeReadsUseOnlyShownPatternsBeforeLimit(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := &invocationReadModelRepository{db: db}
	ctx := context.Background()
	now := time.Date(2026, 7, 30, 13, 0, 0, 0, time.UTC)
	var episodes []invocationreadmodel.Episode
	// Five quarantined rows sort before the one shown row, so a LIMIT applied
	// before the shown filter would return nothing.
	for i, pattern := range []string{"stall", "stall", "stall", "stall", "stall", "command-failure"} {
		id := fmt.Sprintf("ep-%d", i)
		episodes = append(episodes, invocationreadmodel.Episode{FrictionEpisode: runsignal.FrictionEpisode{EpisodeID: id, ClassifierVersion: "episodes.v1", Pattern: pattern, CauseScope: "run-execution", Severity: "medium", HonestyFlags: []string{}, StartEventID: id, EndEventID: id, EvidenceEventIDs: []string{id}, OwnerConfidence: "heuristic", Fingerprint: "fp-" + pattern}, RunID: "run-" + id, OccurredAt: now.Add(time.Duration(i) * time.Minute), TimeBasis: "start_event", RunStatus: "complete"})
	}
	for _, episode := range episodes {
		if err := repo.ReplaceProjection(ctx, nil, nil, []invocationreadmodel.Episode{episode}, nil, invocationreadmodel.Watermark{RunID: episode.RunID, LastEventID: episode.StartEventID, LastEventAt: now, ClassifierVersion: "facts.v1", EpisodeClassifierVersion: "episodes.v1", ProjectedAt: now}, invocationreadmodel.RunFact{RunID: episode.RunID, OccurredAt: now, CreatedAt: now, Status: "complete", CostTimeBasis: "ended_at", ProjectedAt: now}); err != nil {
			t.Fatal(err)
		}
	}
	window := invocationreadmodel.Filter{From: &now, To: ptrTime(now.Add(time.Hour))}
	got, err := repo.Episodes(ctx, window, 1)
	if err != nil || len(got) != 1 || got[0].Pattern != "command-failure" {
		t.Fatalf("shown episodes=%+v err=%v", got, err)
	}
	if hidden, err := repo.HiddenEpisodes(ctx, window); err != nil || hidden != 5 {
		t.Fatalf("hidden=%d err=%v want 5", hidden, err)
	}
	cohort, err := repo.Cohort(ctx, invocationreadmodel.Filter{From: window.From, To: window.To, EpisodePattern: "command-failure"}, 10)
	if err != nil || len(cohort.RunIDs) != 1 || cohort.HiddenEpisodes != 0 {
		t.Fatalf("shown cohort=%+v err=%v", cohort, err)
	}
	cohort, err = repo.Cohort(ctx, invocationreadmodel.Filter{From: window.From, To: window.To, EpisodePattern: "stall"}, 10)
	if err != nil || len(cohort.RunIDs) != 0 || cohort.MatchedRuns != 0 || cohort.HiddenEpisodes != 5 {
		t.Fatalf("quarantined cohort=%+v err=%v want no runs and 5 hidden", cohort, err)
	}
}

// A run-scoped episode read returns that run's episodes only, so one watched
// run's friction never comes from another run.
func TestInvocationReadModelEpisodeReadsHonourTheRun(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := &invocationReadModelRepository{db: db}
	ctx := context.Background()
	now := time.Date(2026, 7, 30, 13, 0, 0, 0, time.UTC)
	for i, runID := range []string{"run-a", "run-b", "run-b"} {
		id := fmt.Sprintf("ep-%d", i)
		episode := invocationreadmodel.Episode{FrictionEpisode: runsignal.FrictionEpisode{EpisodeID: id, ClassifierVersion: "episodes.v1", Pattern: "command-failure", CauseScope: "toolchain", Severity: "one-off", HonestyFlags: []string{}, StartEventID: id, EndEventID: id, EvidenceEventIDs: []string{id}, OwnerConfidence: "heuristic", Fingerprint: "fp-" + id}, RunID: runID, OccurredAt: now.Add(time.Duration(i) * time.Minute), TimeBasis: "start_event", RunStatus: "complete"}
		if err := repo.ReplaceProjection(ctx, nil, nil, []invocationreadmodel.Episode{episode}, nil, invocationreadmodel.Watermark{RunID: runID, LastEventID: id, LastEventAt: now, ClassifierVersion: "facts.v1", EpisodeClassifierVersion: "episodes.v1", ProjectedAt: now}, invocationreadmodel.RunFact{RunID: runID, OccurredAt: now, CreatedAt: now, Status: "complete", CostTimeBasis: "ended_at", ProjectedAt: now}); err != nil {
			t.Fatal(err)
		}
	}
	got, err := repo.Episodes(ctx, invocationreadmodel.Filter{RunID: "run-a", From: &now, ShownEpisodesOnly: true}, 10)
	if err != nil || len(got) != 1 || got[0].RunID != "run-a" {
		t.Fatalf("run-a episodes=%+v err=%v", got, err)
	}
}
