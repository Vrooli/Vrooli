package database

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/invocationreadmodel"
	"agent-manager/internal/repository"
	"agent-manager/internal/runsignal"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func TestStatsRepositoryAggregatesDurableRunEvidence(t *testing.T) {
	ctx := context.Background()
	db, cleanup := setupTestDB(t)
	t.Cleanup(cleanup)
	stats := NewRepositories(db, logrus.New()).Stats
	now := time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC)
	taskID, completeID, failedID := uuid.New(), uuid.New(), uuid.New()
	if _, err := db.ExecContext(ctx, `INSERT INTO tasks (id, title, scope_path, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`, taskID, "analytics task", ".", "queued", SQLiteTime(now), SQLiteTime(now)); err != nil {
		t.Fatalf("insert task: %v", err)
	}
	profile := &domain.AgentProfile{ID: uuid.New(), Name: "analytics profile", ProfileKey: "analytics-profile", RoleRef: "code.default", MaxTurns: 10, Timeout: time.Minute, CreatedBy: "test"}
	if err := NewRepositories(db, logrus.New()).Profiles.Create(ctx, profile); err != nil {
		t.Fatalf("insert profile: %v", err)
	}
	insertRun := func(id uuid.UUID, status, tag, runnerType, model string, started, ended time.Time) {
		t.Helper()
		config := `{"runnerType":"` + runnerType + `","model":"` + model + `"}`
		if _, err := db.ExecContext(ctx, `INSERT INTO runs (id, task_id, tag, status, phase, resolved_config, started_at, ended_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, id, taskID, tag, status, "completed", config, SQLiteTime(started), SQLiteTime(ended), SQLiteTime(now), SQLiteTime(now)); err != nil {
			t.Fatalf("insert run: %v", err)
		}
	}
	insertRun(completeID, "complete", "analytics-complete", "codex", "gpt-test", now, now.Add(2*time.Second))
	insertRun(failedID, "failed", "analytics-failed", "claude-code", "claude-test", now.Add(time.Minute), now.Add(time.Minute+4*time.Second))
	if _, err := db.ExecContext(ctx, `UPDATE runs SET agent_profile_id = ? WHERE id = ?`, profile.ID, completeID); err != nil {
		t.Fatalf("associate profile: %v", err)
	}
	insertEvent := func(id uuid.UUID, sequence int, eventType, data string) {
		t.Helper()
		if _, err := db.ExecContext(ctx, `INSERT INTO run_events (id, run_id, sequence, event_type, timestamp, schema_version, data) VALUES (?, ?, ?, ?, ?, 1, ?)`, uuid.New(), id, sequence, eventType, SQLiteTime(now.Add(time.Duration(sequence)*time.Second)), data); err != nil {
			t.Fatalf("insert event: %v", err)
		}
	}
	insertEvent(completeID, 1, "metric", `{"totalCostUsd":1.5,"costSource":"runner_reported","inputCostUsd":0.5,"outputCostUsd":1,"inputTokens":10,"outputTokens":20}`)
	insertEvent(completeID, 2, "tool_call", `{"toolName":"read_file"}`)
	insertEvent(completeID, 3, "tool_result", `{"toolName":"read_file","success":true}`)
	insertEvent(failedID, 1, "metric", `{"totalCostUsd":2,"costSource":"pricing_table_estimate","inputTokens":5,"outputTokens":5}`)
	insertEvent(failedID, 2, "tool_call", `{"toolName":""}`)
	insertEvent(failedID, 3, "tool_result", `{"toolName":"","success":false}`)
	insertEvent(failedID, 4, "error", `{"code":"runner_failed"}`)
	readModel := NewRepositories(db, logrus.New()).InvocationReadModel.(invocationreadmodel.RunStore)
	for _, fact := range []invocationreadmodel.RunFact{
		{RunID: completeID.String(), OccurredAt: now, CreatedAt: now, DurationMS: 2000, Status: "complete", ProfileID: profile.ID.String(), RunnerType: "codex", Model: "gpt-test", Tag: "analytics-complete", TotalCostUSD: 1.5, InputCostUSD: .5, OutputCostUSD: 1, TotalTokens: 30, InputTokens: 10, OutputTokens: 20, ProjectedAt: now},
		{RunID: failedID.String(), OccurredAt: now.Add(time.Minute), CreatedAt: now, DurationMS: 4000, Status: "failed", RunnerType: "claude-code", Model: "claude-test", Tag: "analytics-failed", TotalCostUSD: 2, TotalTokens: 10, InputTokens: 5, OutputTokens: 5, ProjectedAt: now},
	} {
		if err := readModel.ReplaceRun(ctx, fact); err != nil {
			t.Fatalf("project run fact: %v", err)
		}
	}
	if err := NewRepositories(db, logrus.New()).InvocationReadModel.Replace(ctx, []invocationreadmodel.Fact{
		{InvocationFact: runsignal.InvocationFact{ToolName: "read_file", Outcome: "success"}, RunID: completeID.String(), OccurredAt: now, TimeBasis: "event", RunnerType: "codex", Model: "gpt-test", Tag: "analytics-complete", RunStatus: "complete"},
		{InvocationFact: runsignal.InvocationFact{ToolName: "", Outcome: "failure"}, RunID: failedID.String(), OccurredAt: now.Add(time.Minute), TimeBasis: "event", RunnerType: "claude-code", Model: "claude-test", Tag: "analytics-failed", RunStatus: "failed"},
	}, invocationreadmodel.Watermark{RunID: completeID.String(), LastEventID: "tool-1", LastEventAt: now, ClassifierVersion: runsignal.InvocationFactVersion, ProjectedAt: now}); err != nil {
		t.Fatalf("project invocation facts: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO invocation_read_model_errors (run_id,event_id,occurred_at,time_basis,error_code,profile_id,runner_type,model,tag) VALUES (?,?,?,?,?,?,?,?,?)`, failedID.String(), "error-1", SQLiteTime(now.Add(time.Minute)), "event", "runner_failed", "", "claude-code", "claude-test", "analytics-failed"); err != nil {
		t.Fatalf("project error fact: %v", err)
	}

	filter := repository.StatsFilter{Window: repository.StatsTimeWindow{Start: now.Add(-time.Hour), End: now.Add(time.Hour)}}
	counts, err := stats.GetRunStatusCounts(ctx, filter)
	if err != nil || counts.Total != 2 || counts.Complete != 1 || counts.Failed != 1 {
		t.Fatalf("status counts=%+v err=%v", counts, err)
	}
	if rate, err := stats.GetSuccessRate(ctx, filter); err != nil || rate != 0.5 {
		t.Fatalf("success rate=%v err=%v", rate, err)
	}
	if duration, err := stats.GetDurationStats(ctx, filter); err != nil || duration.Count != 2 || duration.AvgMs <= 0 {
		t.Fatalf("duration stats=%+v err=%v", duration, err)
	}
	cost, err := stats.GetCostStats(ctx, filter)
	if err != nil || cost.TotalCostUSD != 3.5 || cost.TotalTokens != 40 {
		t.Fatalf("cost stats=%+v err=%v", cost, err)
	}
	if runners, err := stats.GetRunnerBreakdown(ctx, filter); err != nil || len(runners) != 2 {
		t.Fatalf("runner breakdown=%+v err=%v", runners, err)
	}
	if profiles, err := stats.GetProfileBreakdown(ctx, filter, 10); err != nil || len(profiles) != 1 || profiles[0].ProfileID != profile.ID || profiles[0].ProfileName != profile.Name {
		t.Fatalf("profile breakdown=%+v err=%v", profiles, err)
	}
	if models, err := stats.GetModelBreakdown(ctx, filter, 10); err != nil || len(models) != 2 {
		t.Fatalf("model breakdown=%+v err=%v", models, err)
	}
	if usage, err := stats.GetToolUsageStats(ctx, filter, 10); err != nil || len(usage) != 2 {
		t.Fatalf("tool usage=%+v err=%v", usage, err)
	}
	if usage, err := stats.GetModelRunUsage(ctx, filter, "gpt-test", 10); err != nil || len(usage) != 1 || usage[0].RunID != completeID {
		t.Fatalf("model run usage=%+v err=%v", usage, err)
	}
	if usage, err := stats.GetToolRunUsage(ctx, filter, "read_file", 10); err != nil || len(usage) != 1 || usage[0].SuccessCount != 1 {
		t.Fatalf("named tool run usage=%+v err=%v", usage, err)
	}
	if usage, err := stats.GetToolRunUsage(ctx, filter, "unknown", 10); err != nil || len(usage) != 1 || usage[0].FailedCount != 1 {
		t.Fatalf("unknown tool run usage=%+v err=%v", usage, err)
	}
	if usage, err := stats.GetToolUsageByModel(ctx, filter, "unknown", 10); err != nil || len(usage) != 1 || usage[0].Model != "claude-test" {
		t.Fatalf("unknown tool by model=%+v err=%v", usage, err)
	}
	if usage, err := stats.GetToolUsageByModel(ctx, filter, "read_file", 10); err != nil || len(usage) != 1 || usage[0].Model != "gpt-test" || usage[0].SuccessCount != 1 {
		t.Fatalf("named tool by model=%+v err=%v", usage, err)
	}
	if patterns, err := stats.GetErrorPatterns(ctx, filter, 10); err != nil || len(patterns) != 1 || patterns[0].ErrorCode != "runner_failed" || patterns[0].SampleRunID != failedID {
		t.Fatalf("error patterns=%+v err=%v", patterns, err)
	}
	if buckets, err := stats.GetTimeSeries(ctx, filter, time.Hour); err != nil || len(buckets) != 1 || buckets[0].RunsStarted != 2 {
		t.Fatalf("time series=%+v err=%v", buckets, err)
	}
	if popular, err := stats.GetPopularModels(ctx, now.Add(-time.Hour), 10); err != nil || len(popular) != 2 {
		t.Fatalf("popular models=%+v err=%v", popular, err)
	}
	if recent, err := stats.GetRecentModels(ctx, 10); err != nil || len(recent) != 2 {
		t.Fatalf("recent models=%+v err=%v", recent, err)
	}

	filtered, err := stats.GetRunStatusCounts(ctx, repository.StatsFilter{Window: filter.Window, RunnerTypes: []domain.RunnerType{domain.RunnerTypeCodex}, TagPrefix: "analytics-complete"})
	if err != nil || filtered.Total != 1 || filtered.Complete != 1 {
		t.Fatalf("filtered counts=%+v err=%v", filtered, err)
	}
	filteredStats := repository.StatsFilter{Window: filter.Window, TagPrefix: "analytics-complete"}
	if rate, err := stats.GetSuccessRate(ctx, filteredStats); err != nil || rate != 1 {
		t.Fatalf("filtered success rate=%v err=%v", rate, err)
	}
	if cost, err := stats.GetCostStats(ctx, filteredStats); err != nil || cost.TotalCostUSD != 1.5 {
		t.Fatalf("filtered cost=%+v err=%v", cost, err)
	}
	if runners, err := stats.GetRunnerBreakdown(ctx, filteredStats); err != nil || len(runners) != 1 || runners[0].RunnerType != domain.RunnerTypeCodex {
		t.Fatalf("filtered runners=%+v err=%v", runners, err)
	}
	if models, err := stats.GetModelBreakdown(ctx, filteredStats, 10); err != nil || len(models) != 1 || models[0].Model != "gpt-test" {
		t.Fatalf("filtered models=%+v err=%v", models, err)
	}
	if tools, err := stats.GetToolUsageStats(ctx, filteredStats, 10); err != nil || len(tools) != 1 || tools[0].ToolName != "read_file" {
		t.Fatalf("filtered tools=%+v err=%v", tools, err)
	}
	for _, bucket := range []time.Duration{24 * time.Hour, 6 * time.Hour, time.Hour, time.Minute} {
		if getBucketSQL(bucket) == "" {
			t.Fatalf("bucket SQL empty for %s", bucket)
		}
	}
}
