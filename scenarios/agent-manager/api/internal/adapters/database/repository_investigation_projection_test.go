package database

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/invocationreadmodel"
	"agent-manager/internal/runsignal"
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
	episode := invocationreadmodel.Episode{FrictionEpisode: runsignal.FrictionEpisode{EpisodeID: "episode-1", ClassifierVersion: "episodes.v1", Pattern: "stall", CauseScope: "run-execution", Severity: "medium", HonestyFlags: []string{}, StartEventID: "event-1", EndEventID: "event-1", EvidenceEventIDs: []string{"event-1"}, CycleCount: 2, RepeatedElement: "redacted-element", OwnerConfidence: "heuristic", Fingerprint: "episode-fp"}, RunID: "episode-run", OccurredAt: now, TimeBasis: "start_event", ProfileID: "profile-1", RunnerType: "codex", Model: "gpt-test", Tag: "cohort-a", RunStatus: "complete"}
	if err := repo.ReplaceProjection(context.Background(), nil, nil, []invocationreadmodel.Episode{episode}, nil, invocationreadmodel.Watermark{RunID: "episode-run", LastEventID: "event-1", LastEventAt: now, ClassifierVersion: "facts.v1", EpisodeClassifierVersion: "episodes.v1", ProjectedAt: now}, invocationreadmodel.RunFact{RunID: "episode-run", OccurredAt: now, CreatedAt: now, Status: "complete", ProfileID: "profile-1", RunnerType: "codex", Model: "gpt-test", Tag: "cohort-a", CostTimeBasis: "ended_at", ProjectedAt: now}); err != nil {
		t.Fatal(err)
	}
	got, err := repo.Episodes(context.Background(), invocationreadmodel.Filter{From: &now, To: ptrTime(now.Add(time.Hour)), ProfileID: "profile-1", RunnerType: "codex", Model: "gpt-test", TagPrefix: "cohort", EpisodePattern: "stall", EpisodeCauseScope: "run-execution", EpisodeFingerprint: "episode-fp"}, 10)
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
	if err != nil || metrics.TotalRuns != 2 || metrics.TerminalRuns != 2 || metrics.SuccessfulRuns != 1 || metrics.SuccessRate != 0.5 || metrics.AverageDurationMS != 120000 || metrics.TotalCostUSD != 2 || metrics.TotalTokens != 30 || metrics.InputTokens != 15 || metrics.OutputTokens != 7 || metrics.CacheReadTokens != 3 || metrics.CacheCreationTokens != 3 || metrics.InputCostUSD != 1 || metrics.OutputCostUSD != 0.5 || metrics.CacheReadCostUSD != 0.25 || metrics.CacheCreationCostUSD != 0.25 || metrics.ReadCalls != 3 || metrics.FileRereads != 1 || metrics.FileRereadRate != 1.0/3.0 {
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

func TestLegacyInvestigationProjectionMigrationCopiesThenDropsRetiredTables(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()
	now := time.Now().UTC()
	if _, err := db.ExecContext(ctx, `CREATE TABLE investigation_invocation_facts (run_id TEXT NOT NULL, call_event_id TEXT NOT NULL, result_event_id TEXT, tool_call_id TEXT, tool_name TEXT NOT NULL, executable TEXT, command_path TEXT, ownership TEXT NOT NULL, catalog_snapshot TEXT, outcome TEXT NOT NULL, retry_of_call_event_id TEXT, help_recovery INTEGER NOT NULL DEFAULT 0, fingerprint TEXT NOT NULL, availability TEXT NOT NULL, classifier_version TEXT NOT NULL, PRIMARY KEY (run_id,call_event_id))`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `CREATE TABLE investigation_friction_episodes (run_id TEXT NOT NULL, episode_id TEXT NOT NULL, classifier_version TEXT NOT NULL, pattern TEXT NOT NULL, cause_scope TEXT NOT NULL, severity TEXT NOT NULL, honesty_flags TEXT NOT NULL DEFAULT '[]', start_event_id TEXT NOT NULL, end_event_id TEXT NOT NULL, evidence_event_ids TEXT NOT NULL DEFAULT '[]', turns INTEGER NOT NULL, tokens INTEGER NOT NULL, wall_clock_ms INTEGER NOT NULL, suspected_owner_scenario TEXT NOT NULL DEFAULT '', suspected_owner_command TEXT NOT NULL DEFAULT '', owner_confidence TEXT NOT NULL, failed_joined_calls INTEGER NOT NULL DEFAULT 0, fingerprint TEXT NOT NULL, PRIMARY KEY(run_id,episode_id))`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `CREATE TABLE investigation_self_report_spans (run_id TEXT NOT NULL, event_id TEXT NOT NULL, rule_id TEXT NOT NULL, classifier_version TEXT NOT NULL, cause_scope TEXT NOT NULL, start_offset INTEGER NOT NULL, end_offset INTEGER NOT NULL, text TEXT NOT NULL, PRIMARY KEY(run_id,event_id,rule_id,start_offset))`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO tasks (id,title,scope_path) VALUES ('task-1','task','scope')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO runs (id,task_id,status,ended_at,tag,actual_model,resolved_config) VALUES ('run-legacy','task-1','complete',?,'legacy-tag','legacy-model','{"runnerType":"codex"}')`, SQLiteTime(now)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO investigation_invocation_facts (run_id,call_event_id,tool_name,ownership,outcome,fingerprint,availability,classifier_version) VALUES ('run-legacy','pruned-call','shell','external','success','fp','available','invocation-fact.legacy')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO investigation_friction_episodes (run_id,episode_id,classifier_version,pattern,cause_scope,severity,honesty_flags,start_event_id,end_event_id,evidence_event_ids,turns,tokens,wall_clock_ms,owner_confidence,fingerprint) VALUES ('run-legacy','episode-1','friction-episode.v2','stall','run-execution','medium','[]','pruned-call','pruned-call','["pruned-call"]',1,0,0,'heuristic','episode-fp')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO investigation_self_report_spans (run_id,event_id,rule_id,classifier_version,cause_scope,start_offset,end_offset,text) VALUES ('run-legacy','pruned-call','blocked','self-report.v1','run-execution',0,7,'blocked')`); err != nil {
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
	if err := db.consolidateLegacyInvestigationProjections(ctx); err != nil {
		t.Fatal(err)
	}
	repo := &invocationReadModelRepository{db: db}
	facts, err := repo.Facts(ctx, "run-legacy")
	if err != nil || len(facts) != 1 {
		t.Fatalf("facts=%+v err=%v", facts, err)
	}
	if facts[0].Version != "invocation-fact.legacy" || facts[0].TimeBasis != "legacy_run_time" || facts[0].Model != "legacy-model" || facts[0].RunnerType != "codex" {
		t.Fatalf("migrated fact=%+v", facts[0])
	}
	episodes, err := repo.EpisodesForRun(ctx, "run-legacy")
	if err != nil || len(episodes) != 1 || episodes[0].EpisodeID != "episode-1" {
		t.Fatalf("migrated episodes=%+v err=%v", episodes, err)
	}
	spans, err := repo.SelfReportSpansForRun(ctx, "run-legacy")
	if err != nil || len(spans) != 1 || spans[0].RuleID != "blocked" {
		t.Fatalf("migrated spans=%+v err=%v", spans, err)
	}
	watermark, err := repo.Watermark(ctx, "run-legacy")
	if err != nil || watermark == nil || watermark.EpisodeClassifierVersion != "friction-episode.v2" || watermark.SelfReportClassifierVersion != "self-report.v1" {
		t.Fatalf("migrated watermark=%+v err=%v", watermark, err)
	}
	retained, err := repo.Facts(ctx, "run-retained")
	if err != nil || len(retained) != 1 || retained[0].TimeBasis != "call_event" || !retained[0].OccurredAt.Equal(retainedAt) || retained[0].Version != "invocation-fact.retained" {
		t.Fatalf("retained migrated fact=%+v err=%v", retained, err)
	}
	for _, table := range []string{"investigation_invocation_facts", "investigation_friction_episodes", "investigation_self_report_spans"} {
		var count int
		if err := db.Get(&count, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, table); err != nil || count != 0 {
			t.Fatalf("retired table %s remains: count=%d err=%v", table, count, err)
		}
	}
}

func TestInvocationReadModelAggregateAndCohortShareFilter(t *testing.T) {
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
