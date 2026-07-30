package database

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/invocationreadmodel"
	"agent-manager/internal/runreport"
	"github.com/google/uuid"
)

func TestInvocationFactRepositoryRebuildIsIdempotent(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := &invocationFactRepository{db: db}
	runID := uuid.New()
	facts := []runreport.InvocationFact{{Version: "invocation-fact.v1", CallEventID: "call-a", ToolName: "shell", Ownership: "unknown", Outcome: "failure", Fingerprint: "fp", Availability: "available"}}
	if err := repo.ReplaceInvocationFacts(context.Background(), runID, facts); err != nil {
		t.Fatal(err)
	}
	if err := repo.ReplaceInvocationFacts(context.Background(), runID, facts); err != nil {
		t.Fatal(err)
	}
	got, err := repo.InvocationFacts(context.Background(), runID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].CallEventID != "call-a" || got[0].Version != facts[0].Version {
		t.Fatalf("facts=%+v", got)
	}
}

func TestInvocationReadModelReplaceIsAtomicAndIdempotent(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := &invocationReadModelRepository{db: db}
	now := time.Now().UTC()
	watermark := invocationreadmodel.Watermark{RunID: "run-1", LastEventID: "event-1", LastEventAt: now, ClassifierVersion: runreport.InvocationFactVersion, ProjectedAt: now}
	facts := []invocationreadmodel.Fact{{InvocationFact: runreport.InvocationFact{Version: runreport.InvocationFactVersion, CallEventID: "call-1", ToolName: "shell", Ownership: "unknown", Outcome: "success", Fingerprint: "fp", Availability: "available"}, RunID: "run-1", OccurredAt: now, TimeBasis: "call_event", ProfileID: "unknown", RunnerType: "unknown", Model: "unknown", Tag: "unknown", RunStatus: "complete"}}
	if err := repo.Replace(context.Background(), facts, watermark); err != nil {
		t.Fatal(err)
	}
	if err := repo.Replace(context.Background(), facts, watermark); err != nil {
		t.Fatal(err)
	}
	got, err := repo.Facts(context.Background(), "run-1")
	if err != nil || len(got) != 1 || got[0].CallEventID != "call-1" {
		t.Fatalf("facts=%+v err=%v", got, err)
	}
	gotWatermark, err := repo.Watermark(context.Background(), "run-1")
	if err != nil || gotWatermark == nil || gotWatermark.LastEventID != "event-1" {
		t.Fatalf("watermark=%+v err=%v", gotWatermark, err)
	}
}

func TestInvocationReadModelReplaceRunUpsertsTerminalSummary(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := &invocationReadModelRepository{db: db}
	created := time.Date(2026, 7, 29, 18, 0, 0, 0, time.UTC)
	started, ended := created.Add(time.Minute), created.Add(3*time.Minute)
	fact := invocationreadmodel.RunFact{RunID: "run-throughput", OccurredAt: ended, CreatedAt: created, StartedAt: &started, EndedAt: &ended, DurationMS: 120000, Status: "complete", ProfileID: "profile-1", RunnerType: "codex", Model: "gpt-test", Tag: "analytics", TotalCostUSD: 1.25, TotalTokens: 42, ReadCalls: 4, FileRereads: 1, CostTimeBasis: "ended_at", ProjectedAt: ended.Add(time.Minute)}
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
		if err := repo.ReplaceProjection(ctx, []invocationreadmodel.Fact{{InvocationFact: runreport.InvocationFact{Version: "v1", CallEventID: id + "-call", ToolName: "shell", Ownership: map[string]string{"run-a": "external", "run-b": "project"}[id], Outcome: "success", Fingerprint: id, Availability: "available"}, RunID: id, OccurredAt: started, TimeBasis: "call_event", ProfileID: profile, RunnerType: "codex", Model: "gpt-test", Tag: "analytics", RunStatus: status}}, errors, invocationreadmodel.Watermark{RunID: id, LastEventID: id + "-call", LastEventAt: started, ClassifierVersion: "v1", ProjectedAt: now}, invocationreadmodel.RunFact{RunID: id, OccurredAt: ended, CreatedAt: started.Add(-time.Minute), StartedAt: &started, EndedAt: &ended, DurationMS: 120000, Status: status, ProfileID: profile, RunnerType: "codex", Model: "gpt-test", Tag: "analytics", TotalCostUSD: cost, UnknownCostUSD: cost, InputCostUSD: cost / 2, OutputCostUSD: cost / 4, CacheReadCostUSD: cost / 8, CacheCreationCostUSD: cost / 8, TotalTokens: tokens, InputTokens: tokens / 2, OutputTokens: tokens / 4, CacheReadTokens: tokens / 8, CacheCreationTokens: tokens / 8, ReadCalls: map[string]int64{"run-a": 2, "run-b": 1}[id], FileRereads: map[string]int64{"run-a": 1, "run-b": 0}[id], CostTimeBasis: "ended_at", ProjectedAt: now}); err != nil {
			t.Fatal(err)
		}
	}
	seed("run-a", "complete", "profile-a", 0, 1.5, 10)
	seed("run-b", "failed", "profile-b", time.Minute, 0.5, 20)
	metrics, err := repo.RunMetrics(ctx, invocationreadmodel.Filter{From: &now, To: ptrTime(now.Add(time.Hour))})
	if err != nil || metrics.TotalRuns != 2 || metrics.TerminalRuns != 2 || metrics.SuccessfulRuns != 1 || metrics.SuccessRate != 0.5 || metrics.AverageDurationMS != 120000 || metrics.TotalCostUSD != 2 || metrics.UnknownCostUSD != 2 || metrics.AuthoritativeCostUSD != 0 || metrics.EstimatedCostUSD != 0 || metrics.TotalTokens != 30 || metrics.InputTokens != 15 || metrics.OutputTokens != 7 || metrics.CacheReadTokens != 3 || metrics.CacheCreationTokens != 3 || metrics.InputCostUSD != 1 || metrics.OutputCostUSD != 0.5 || metrics.CacheReadCostUSD != 0.25 || metrics.CacheCreationCostUSD != 0.25 || metrics.ReadCalls != 3 || metrics.FileRereads != 1 || metrics.FileRereadRate != 1.0/3.0 {
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
	if err != nil || len(runnerRows) != 1 || runnerRows[0].Key != "codex" || runnerRows[0].Value != "codex" || runnerRows[0].RunCount != 2 || runnerRows[0].SuccessCount != 1 || runnerRows[0].FailedCount != 1 || runnerRows[0].TotalCostUSD != 2 || runnerRows[0].TotalTokens != 30 {
		t.Fatalf("runner rows=%+v err=%v", runnerRows, err)
	}
	trendRows, err := repo.RunTimeSeries(ctx, invocationreadmodel.Filter{From: &now, To: ptrTime(now.Add(time.Hour))}, time.Hour)
	if err != nil || len(trendRows) != 1 || !trendRows[0].Bucket.Equal(now) || trendRows[0].TerminalRuns != 2 || trendRows[0].CompletedRuns != 1 || trendRows[0].FailedRuns != 1 || trendRows[0].CancelledRuns != 0 || trendRows[0].TotalCostUSD != 2 || trendRows[0].AvgDurationMS != 120000 {
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
	if err != nil || filtered.TotalRuns != 1 || filtered.TotalCostUSD != 1.5 || filtered.SuccessfulRuns != 1 {
		t.Fatalf("ownership-filtered metrics=%+v err=%v", filtered, err)
	}
	empty, err := repo.RunMetrics(ctx, invocationreadmodel.Filter{From: ptrTime(now.Add(2 * time.Hour)), To: ptrTime(now.Add(3 * time.Hour))})
	if err != nil || empty.TotalRuns != 0 || empty.AverageDurationMS != 0 || empty.TotalCostUSD != 0 {
		t.Fatalf("empty metrics=%+v err=%v", empty, err)
	}
	for _, row := range []struct{ runID, fingerprint string }{{"run-a", "repeat"}, {"run-b", "repeat"}, {"run-b", "unique"}} {
		if _, err := db.ExecContext(ctx, `INSERT INTO run_findings (id,run_id,investigation_run_id,category,severity,recommendation_text,evidence,target_path,fingerprint,operator_decision,created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?)`, uuid.New().String(), row.runID, row.runID, "efficiency", "warning", "fixture", "", "", row.fingerprint, "", SQLiteTime(now.Add(5*time.Minute))); err != nil {
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

func TestInvocationReadModelMigratesLegacyFactsWithoutReinterpretation(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()
	now := time.Now().UTC()
	if _, err := db.ExecContext(ctx, `INSERT INTO tasks (id,title,scope_path) VALUES ('task-1','task','scope')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO runs (id,task_id,status,ended_at,tag,actual_model,resolved_config) VALUES ('run-legacy','task-1','complete',?,'legacy-tag','legacy-model','{"runnerType":"codex"}')`, SQLiteTime(now)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO investigation_invocation_facts (run_id,call_event_id,tool_name,ownership,outcome,fingerprint,availability,classifier_version) VALUES ('run-legacy','pruned-call','shell','external','success','fp','available','invocation-fact.legacy')`); err != nil {
		t.Fatal(err)
	}
	retainedAt := now.Add(-time.Minute)
	if _, err := db.ExecContext(ctx, `INSERT INTO runs (id,task_id,status,ended_at,tag,actual_model,resolved_config) VALUES ('run-retained','task-1','complete',?,'retained-tag','retained-model','{"runnerType":"codex"}')`, SQLiteTime(now)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO run_events (id,run_id,sequence,event_type,timestamp,schema_version,data) VALUES ('retained-call','run-retained',0,'tool_call',?,1,'{}')`, SQLiteTime(retainedAt)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO investigation_invocation_facts (run_id,call_event_id,tool_name,ownership,outcome,fingerprint,availability,classifier_version) VALUES ('run-retained','retained-call','shell','project','success','retained-fp','available','invocation-fact.retained')`); err != nil {
		t.Fatal(err)
	}
	repo := &invocationReadModelRepository{db: db}
	if count, err := repo.MigrateLegacyInvocationFacts(ctx); err != nil || count != 2 {
		t.Fatalf("first migration count=%d err=%v", count, err)
	}
	if count, err := repo.MigrateLegacyInvocationFacts(ctx); err != nil || count != 0 {
		t.Fatalf("second migration count=%d err=%v", count, err)
	}
	facts, err := repo.Facts(ctx, "run-legacy")
	if err != nil || len(facts) != 1 {
		t.Fatalf("facts=%+v err=%v", facts, err)
	}
	if facts[0].Version != "invocation-fact.legacy" || facts[0].TimeBasis != "run_end_derived" || facts[0].Model != "legacy-model" || facts[0].RunnerType != "codex" {
		t.Fatalf("migrated fact=%+v", facts[0])
	}
	retained, err := repo.Facts(ctx, "run-retained")
	if err != nil || len(retained) != 1 || retained[0].TimeBasis != "call_event" || !retained[0].OccurredAt.Equal(retainedAt) || retained[0].Version != "invocation-fact.retained" {
		t.Fatalf("retained migrated fact=%+v err=%v", retained, err)
	}
}

func TestInvocationReadModelAggregateAndCohortShareFilter(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()
	repo := &invocationReadModelRepository{db: db}
	now := time.Now().UTC()
	facts := []invocationreadmodel.Fact{
		{InvocationFact: runreport.InvocationFact{Version: "v1", CallEventID: "a", ToolName: "shell", Ownership: "external", Outcome: "failure", Fingerprint: "fp-a", Availability: "available"}, RunID: "run-a", OccurredAt: now, TimeBasis: "call_event", ProfileID: "p", RunnerType: "codex", Model: "m", Tag: "keep", RunStatus: "complete"},
		{InvocationFact: runreport.InvocationFact{Version: "v1", CallEventID: "b", ToolName: "shell", Ownership: "project", Outcome: "success", Fingerprint: "fp-b", Availability: "available"}, RunID: "run-b", OccurredAt: now, TimeBasis: "call_event", ProfileID: "p", RunnerType: "codex", Model: "m", Tag: "skip", RunStatus: "complete"},
	}
	for _, fact := range facts {
		if err := repo.Replace(ctx, []invocationreadmodel.Fact{fact}, invocationreadmodel.Watermark{RunID: fact.RunID, LastEventID: fact.CallEventID, LastEventAt: now, ClassifierVersion: "v1", ProjectedAt: now}); err != nil {
			t.Fatal(err)
		}
	}
	filter := invocationreadmodel.Filter{Ownership: "external"}
	rows, err := repo.Aggregate(ctx, filter, "outcome", 10)
	if err != nil || len(rows) != 1 || rows[0].Value != "failure" || rows[0].Count != 1 {
		t.Fatalf("aggregate=%+v err=%v", rows, err)
	}
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
		InvocationFact: runreport.InvocationFact{Version: "v1", CallEventID: "unknown-call", ToolName: "unknown", Ownership: "unknown", Outcome: "unknown", Fingerprint: "unknown", Availability: "unknown"},
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

func TestReceiptEvidenceRepositoryRebuildKeepsOnlyVerifiedIDs(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := &invocationFactRepository{db: db}
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
