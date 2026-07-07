package stats

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"swarm-manager/internal/eventlog"
	"swarm-manager/internal/operatingmode"

	_ "modernc.org/sqlite"
)

func setupEngine(t *testing.T) (*Engine, *eventlog.SQLiteRepository) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	repo := eventlog.NewSQLiteRepository(db)
	if err := repo.InitSchema(context.Background()); err != nil {
		t.Fatalf("init schema: %v", err)
	}
	engine := NewEngine(repo)
	return engine, repo
}

func appendEvent(t *testing.T, repo *eventlog.SQLiteRepository, ts time.Time, entityType eventlog.EntityType, entityID string, eventType eventlog.EventType, payload any) {
	t.Helper()
	var meta json.RawMessage
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		meta = data
	}
	_, err := repo.Append(context.Background(), eventlog.Event{
		Timestamp:  ts,
		EntityType: entityType,
		EntityID:   entityID,
		EventType:  eventType,
		Metadata:   meta,
	})
	if err != nil {
		t.Fatalf("append: %v", err)
	}
}

func TestEmptyEngine(t *testing.T) {
	engine, _ := setupEngine(t)

	if err := engine.Rebuild(context.Background()); err != nil {
		t.Fatalf("rebuild: %v", err)
	}

	stats := engine.GetStats()
	if stats.EventCount != 0 {
		t.Errorf("expected 0 events, got %d", stats.EventCount)
	}
	if stats.Dashboard.TotalBacklogSize != 0 {
		t.Errorf("expected 0 backlog, got %d", stats.Dashboard.TotalBacklogSize)
	}

	data, err := json.Marshal(stats)
	if err != nil {
		t.Fatalf("marshal stats: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("unmarshal stats: %v", err)
	}
	assertJSONArrayField(t, payload, "blocking", "top_reasons")
	assertJSONArrayField(t, payload, "scope", "initiatives")
	assertJSONArrayField(t, payload, "dashboard", "velocity_trend")
}

func assertJSONArrayField(t *testing.T, payload map[string]any, objectKey string, fieldKey string) {
	t.Helper()
	object, ok := payload[objectKey].(map[string]any)
	if !ok {
		t.Fatalf("%s object missing or invalid: %#v", objectKey, payload[objectKey])
	}
	if object[fieldKey] == nil {
		t.Fatalf("%s.%s marshaled as null, want JSON array", objectKey, fieldKey)
	}
	if _, ok := object[fieldKey].([]any); !ok {
		t.Fatalf("%s.%s = %#v, want JSON array", objectKey, fieldKey, object[fieldKey])
	}
}

func TestModeStats(t *testing.T) {
	engine, repo := setupEngine(t)
	ctx := context.Background()
	now := time.Now().UTC()

	appendEvent(t, repo, now, eventlog.EntityInitiative, "default-init",
		eventlog.EventInitiativeCreated, nil)
	appendEvent(t, repo, now, eventlog.EntityInitiative, "holistic-init",
		eventlog.EventInitiativeCreated, nil)
	appendEvent(t, repo, now, eventlog.EntityInitiative, "holistic-init",
		eventlog.EventInitiativeModeChanged, eventlog.InitiativeModeChangePayload{From: "item-level", To: "holistic-loop"})

	if err := engine.Rebuild(ctx); err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	stats := engine.GetStats()

	if stats.Mode.UsageByMode["item-level"] != 1 {
		t.Errorf("item-level usage = %d, want 1", stats.Mode.UsageByMode["item-level"])
	}
	if stats.Mode.UsageByMode["holistic-loop"] != 1 {
		t.Errorf("holistic-loop usage = %d, want 1", stats.Mode.UsageByMode["holistic-loop"])
	}
	if stats.Mode.ModeSwitchCount != 1 {
		t.Errorf("mode switch count = %d, want 1", stats.Mode.ModeSwitchCount)
	}
}

func TestOperatingModePhaseStats(t *testing.T) {
	engine, repo := setupEngine(t)
	ctx := context.Background()
	base := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)

	startPayload := eventlog.OperatingModePhasePayload{
		Mode:            "holistic-loop",
		ScopeKind:       "initiative",
		ScopeID:         "init-a",
		InitiativeName:  "init-a",
		Phase:           "execute",
		RunStrategy:     "operator_gated_loop",
		AgentProfileKey: "swarm-manager/deep-work",
		RoundNumber:     2,
		RunID:           "run-123",
	}
	appendEvent(t, repo, base, eventlog.EntityInitiative, "init-a",
		eventlog.EventOperatingModePhaseStarted, startPayload)
	completedPayload := startPayload
	completedPayload.DurationSeconds = 90
	completedPayload.Status = "completed"
	completedPayload.ReplanNeeded = true
	appendEvent(t, repo, base.Add(90*time.Second), eventlog.EntityInitiative, "init-a",
		eventlog.EventOperatingModePhaseCompleted, completedPayload)
	appendEvent(t, repo, base.Add(91*time.Second), eventlog.EntityInitiative, "init-a",
		eventlog.EventOperatingModeBacklogSynced, eventlog.OperatingModeBacklogSyncPayload{
			Mode:                  "holistic-loop",
			ScopeKind:             "initiative",
			ScopeID:               "init-a",
			InitiativeName:        "init-a",
			Phase:                 "execute",
			BacklogItemsCompleted: 2,
			BacklogItemsUpdated:   1,
		})

	if err := engine.Rebuild(ctx); err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	stats := engine.GetStats()

	if stats.Mode.PhaseRunsByMode["holistic-loop"]["execute"] != 1 {
		t.Fatalf("phase runs = %+v", stats.Mode.PhaseRunsByMode)
	}
	if stats.Mode.CompletedByMode["holistic-loop"] != 1 {
		t.Fatalf("completed by mode = %+v", stats.Mode.CompletedByMode)
	}
	if got := stats.Mode.ReplanRateByMode["holistic-loop"]; got.SampleSize != 1 || got.Rate != 1 {
		t.Fatalf("replan rate = %+v, want n=1 rate=1", got)
	}
	if stats.Mode.AvgPhaseDurationSeconds["holistic-loop"]["execute"] != 90 {
		t.Fatalf("avg duration = %+v", stats.Mode.AvgPhaseDurationSeconds)
	}
	if stats.Mode.UsageByProfile["swarm-manager/deep-work"] != 1 {
		t.Fatalf("profile usage = %+v", stats.Mode.UsageByProfile)
	}
	if stats.Mode.PhaseRunsByProfile["swarm-manager/deep-work"]["execute"] != 1 {
		t.Fatalf("profile phase runs = %+v", stats.Mode.PhaseRunsByProfile)
	}
	if got := stats.Mode.BacklogSyncByMode["holistic-loop"]; got.Events != 1 || got.ItemsCompleted != 2 || got.ItemsUpdated != 1 {
		t.Fatalf("backlog sync = %+v", got)
	}
}

func TestPhaseRunsByLane_AggregatesPerPhaseKind(t *testing.T) {
	engine, repo := setupEngine(t)
	ctx := context.Background()
	base := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		phase     string
		phaseKind string
	}{
		{"investigate", "investigate"},
		{"plan", "investigate"},
		{"execute", "execute"},
		{"review", "review"},
		{"reconcile", "reconcile"},
		{"reconcile", "reconcile"},
	}
	for i, c := range cases {
		appendEvent(t, repo, base.Add(time.Duration(i)*time.Second), eventlog.EntityInitiative, "init-lane",
			eventlog.EventOperatingModePhaseStarted, eventlog.OperatingModePhasePayload{
				Mode:        "holistic-loop",
				ScopeKind:   "initiative",
				ScopeID:     "init-lane",
				Phase:       c.phase,
				PhaseKind:   c.phaseKind,
				RunStrategy: "operator_gated_loop",
				RoundNumber: i + 1,
			})
	}

	if err := engine.Rebuild(ctx); err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	got := engine.GetStats().Mode.PhaseRunsByLane

	want := map[string]int{
		"investigate": 2,
		"execute":     1,
		"review":      1,
		"reconcile":   2,
	}
	for lane, count := range want {
		if got[lane] != count {
			t.Errorf("lane %q = %d, want %d", lane, got[lane], count)
		}
	}
}

func TestPhaseRunsByLane_LegacyEventsKeepEmptyKey(t *testing.T) {
	engine, repo := setupEngine(t)
	ctx := context.Background()
	base := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)

	// Legacy event written before P2 wired phase_kind onto the payload.
	appendEvent(t, repo, base, eventlog.EntityInitiative, "init-legacy",
		eventlog.EventOperatingModePhaseStarted, eventlog.OperatingModePhasePayload{
			Mode:        "holistic-loop",
			ScopeKind:   "initiative",
			ScopeID:     "init-legacy",
			Phase:       "investigate",
			PhaseKind:   "", // legacy
			RunStrategy: "operator_gated_loop",
			RoundNumber: 1,
		})

	if err := engine.Rebuild(ctx); err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	got := engine.GetStats().Mode.PhaseRunsByLane
	if got[""] != 1 {
		t.Errorf("empty-lane bucket = %d, want 1 (legacy events visible)", got[""])
	}
	if got["investigate"] != 0 {
		t.Errorf("investigate bucket leaked legacy event: got %d, want 0", got["investigate"])
	}
}

func TestOperatingModeReplanRateCountsCompletedExecutePayloadOnly(t *testing.T) {
	engine, repo := setupEngine(t)
	ctx := context.Background()
	base := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)

	payload := eventlog.OperatingModePhasePayload{
		Mode:         "holistic-loop",
		ScopeKind:    "initiative",
		ScopeID:      "init-a",
		Phase:        "execute",
		RunStrategy:  "operator_gated_loop",
		RoundNumber:  3,
		RunID:        "run-replan",
		Status:       "completed",
		ReplanNeeded: true,
	}
	appendEvent(t, repo, base, eventlog.EntityInitiative, "init-a",
		eventlog.EventOperatingModePhaseCompleted, payload)
	appendEvent(t, repo, base.Add(time.Second), eventlog.EntityInitiative, "init-a",
		eventlog.EventOperatingModeReplanNeeded, payload)

	if err := engine.Rebuild(ctx); err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	got := engine.GetStats().Mode.ReplanRateByMode["holistic-loop"]
	if got.SampleSize != 1 || got.Rate != 1 {
		t.Fatalf("replan rate = %+v, want n=1 rate=1", got)
	}
}

func TestOperatingModeReplanRateFalseExecutePayload(t *testing.T) {
	engine, repo := setupEngine(t)
	ctx := context.Background()
	base := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)

	appendEvent(t, repo, base, eventlog.EntityInitiative, "init-a",
		eventlog.EventOperatingModePhaseCompleted, eventlog.OperatingModePhasePayload{
			Mode:         "holistic-loop",
			ScopeKind:    "initiative",
			ScopeID:      "init-a",
			Phase:        "execute",
			RunStrategy:  "operator_gated_loop",
			RoundNumber:  3,
			RunID:        "run-no-replan",
			Status:       "completed",
			ReplanNeeded: false,
		})

	if err := engine.Rebuild(ctx); err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	got := engine.GetStats().Mode.ReplanRateByMode["holistic-loop"]
	if got.SampleSize != 1 || got.Rate != 0 {
		t.Fatalf("replan rate = %+v, want n=1 rate=0", got)
	}
}

func TestOperatingModeAcceptanceStats(t *testing.T) {
	engine, repo := setupEngine(t)
	ctx := context.Background()
	now := time.Now().UTC()

	appendEvent(t, repo, now, eventlog.EntityInitiative, "init-a",
		eventlog.EventOperatingModePhaseCompleted, eventlog.OperatingModePhasePayload{
			Mode:        "phased-plan-drain",
			ScopeKind:   "initiative",
			ScopeID:     "init-a",
			Phase:       "review",
			RunStrategy: "sequential_handoff",
			Verdict:     "accept",
		})
	appendEvent(t, repo, now, eventlog.EntityInitiative, "init-b",
		eventlog.EventOperatingModePhaseCompleted, eventlog.OperatingModePhasePayload{
			Mode:        "phased-plan-drain",
			ScopeKind:   "initiative",
			ScopeID:     "init-b",
			Phase:       "review",
			RunStrategy: "sequential_handoff",
			Verdict:     "request_changes",
		})

	if err := engine.Rebuild(ctx); err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	got := engine.GetStats().Mode.AcceptanceRateByMode["phased-plan-drain"]
	if got.SampleSize != 2 || got.Rate != 0.5 {
		t.Fatalf("acceptance rate = %+v, want n=2 rate=.5", got)
	}
}

func TestOperatingModeMetricsUseRegistryPolicy(t *testing.T) {
	state := newAggregateState()
	policy := operatingmode.MetricsPolicy{
		EventSource:            "synthetic-mode",
		ReplanSamplePhases:     []operatingmode.Phase{"ship"},
		AcceptanceSamplePhases: []operatingmode.Phase{"judge"},
		AcceptedVerdicts:       []string{"yes"},
	}

	state.recordOperatingModePolicyMetrics(eventlog.OperatingModePhasePayload{
		Mode:         "synthetic-mode",
		Phase:        "ship",
		ReplanNeeded: true,
		Verdict:      "yes",
	}, policy)
	state.recordOperatingModePolicyMetrics(eventlog.OperatingModePhasePayload{
		Mode:         "synthetic-mode",
		Phase:        "execute",
		ReplanNeeded: true,
		Verdict:      "yes",
	}, policy)
	state.recordOperatingModePolicyMetrics(eventlog.OperatingModePhasePayload{
		Mode:    "synthetic-mode",
		Phase:   "judge",
		Verdict: "YES",
	}, policy)
	state.recordOperatingModePolicyMetrics(eventlog.OperatingModePhasePayload{
		Mode:    "synthetic-mode",
		Phase:   "review",
		Verdict: "yes",
	}, policy)

	modeStats := state.buildMode()
	if got := modeStats.ReplanRateByMode["synthetic-mode"]; got.SampleSize != 1 || got.Rate != 1 {
		t.Fatalf("replan rate = %+v, want n=1 rate=1", got)
	}
	if got := modeStats.AcceptanceRateByMode["synthetic-mode"]; got.SampleSize != 1 || got.Rate != 1 {
		t.Fatalf("acceptance rate = %+v, want n=1 rate=1", got)
	}
}

func TestThroughput(t *testing.T) {
	engine, repo := setupEngine(t)
	ctx := context.Background()
	now := time.Now().UTC()

	// Create 3 items in last 7 days.
	for i := range 3 {
		appendEvent(t, repo, now.Add(-time.Duration(i)*24*time.Hour), eventlog.EntityBacklogItem,
			"execute/item-"+string(rune('a'+i)), eventlog.EventBacklogCreated,
			eventlog.BacklogCreatedPayload{Kind: "execute", Status: "backlog", Priority: 5})
	}
	// Create 2 more items 15 days ago.
	for i := range 2 {
		appendEvent(t, repo, now.Add(-15*24*time.Hour), eventlog.EntityBacklogItem,
			"fix/old-"+string(rune('a'+i)), eventlog.EventBacklogCreated,
			eventlog.BacklogCreatedPayload{Kind: "fix", Status: "backlog", Priority: 3})
	}
	// Complete 2 items in last 7 days.
	for i := range 2 {
		appendEvent(t, repo, now.Add(-time.Duration(i)*24*time.Hour), eventlog.EntityBacklogItem,
			"execute/item-"+string(rune('a'+i)), eventlog.EventBacklogStatusChanged,
			eventlog.StatusChangePayload{From: "backlog", To: "completed"})
	}

	if err := engine.Rebuild(ctx); err != nil {
		t.Fatalf("rebuild: %v", err)
	}

	stats := engine.GetStats()
	if stats.Throughput.CreatedLast7Days != 3 {
		t.Errorf("created 7d: got %d, want 3", stats.Throughput.CreatedLast7Days)
	}
	if stats.Throughput.CreatedLast30Days != 5 {
		t.Errorf("created 30d: got %d, want 5", stats.Throughput.CreatedLast30Days)
	}
	if stats.Throughput.CompletedLast7Days != 2 {
		t.Errorf("completed 7d: got %d, want 2", stats.Throughput.CompletedLast7Days)
	}
	if stats.Throughput.NetDelta7Days != 1 {
		t.Errorf("net delta 7d: got %d, want 1", stats.Throughput.NetDelta7Days)
	}
}

func TestCycleAndLeadTime(t *testing.T) {
	engine, repo := setupEngine(t)
	ctx := context.Background()

	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Create item at T=0.
	appendEvent(t, repo, base, eventlog.EntityBacklogItem, "execute/a",
		eventlog.EventBacklogCreated, eventlog.BacklogCreatedPayload{Kind: "execute", Status: "backlog", Priority: 5})
	// Move to in_progress at T=10h.
	appendEvent(t, repo, base.Add(10*time.Hour), eventlog.EntityBacklogItem, "execute/a",
		eventlog.EventBacklogStatusChanged, eventlog.StatusChangePayload{From: "backlog", To: "in_progress"})
	// Complete at T=16h.
	appendEvent(t, repo, base.Add(16*time.Hour), eventlog.EntityBacklogItem, "execute/a",
		eventlog.EventBacklogStatusChanged, eventlog.StatusChangePayload{From: "in_progress", To: "completed"})

	if err := engine.Rebuild(ctx); err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	stats := engine.GetStats()

	// Lead time: 16h - 0h = 16h.
	if stats.Timing.AvgLeadTimeHours != 16.0 {
		t.Errorf("lead time: got %.1f, want 16.0", stats.Timing.AvgLeadTimeHours)
	}
	if stats.Timing.LeadTimeSampleSize != 1 {
		t.Errorf("lead time sample size: got %d, want 1", stats.Timing.LeadTimeSampleSize)
	}
}

func TestBlocking(t *testing.T) {
	engine, repo := setupEngine(t)
	ctx := context.Background()
	now := time.Now().UTC()

	// Create 3 items.
	for i := range 3 {
		appendEvent(t, repo, now, eventlog.EntityBacklogItem,
			"execute/item-"+string(rune('a'+i)), eventlog.EventBacklogCreated,
			eventlog.BacklogCreatedPayload{Kind: "execute", Status: "backlog", Priority: 5})
	}
	// Block 2.
	appendEvent(t, repo, now, eventlog.EntityBacklogItem, "execute/item-a",
		eventlog.EventBacklogBlocked, eventlog.BlockPayload{Reason: "waiting on design"})
	appendEvent(t, repo, now, eventlog.EntityBacklogItem, "execute/item-b",
		eventlog.EventBacklogBlocked, eventlog.BlockPayload{Reason: "waiting on design"})

	if err := engine.Rebuild(ctx); err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	stats := engine.GetStats()

	if stats.Blocking.CurrentlyBlocked != 2 {
		t.Errorf("blocked: got %d, want 2", stats.Blocking.CurrentlyBlocked)
	}
	// 2 out of 3 items are blocked.
	if stats.Blocking.BlockedRatio < 0.6 || stats.Blocking.BlockedRatio > 0.7 {
		t.Errorf("ratio: got %.2f, want ~0.67", stats.Blocking.BlockedRatio)
	}
	if len(stats.Blocking.TopReasons) != 1 || stats.Blocking.TopReasons[0].Count != 2 {
		t.Errorf("reasons: %+v", stats.Blocking.TopReasons)
	}

	// Unblock one.
	appendEvent(t, repo, now.Add(2*time.Hour), eventlog.EntityBacklogItem, "execute/item-a",
		eventlog.EventBacklogUnblocked, eventlog.BlockPayload{Reason: "design done"})

	if err := engine.Refresh(ctx); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	stats = engine.GetStats()
	if stats.Blocking.CurrentlyBlocked != 1 {
		t.Errorf("blocked after unblock: got %d, want 1", stats.Blocking.CurrentlyBlocked)
	}
	if stats.Blocking.AvgBlockHours != 2.0 {
		t.Errorf("avg block hours: got %.1f, want 2.0", stats.Blocking.AvgBlockHours)
	}
}

func TestAgentMetrics(t *testing.T) {
	engine, repo := setupEngine(t)
	ctx := context.Background()
	now := time.Now().UTC()

	// Create 3 executions: 2 complete (1 with fixup), 1 fails.
	appendEvent(t, repo, now, eventlog.EntityExecution, "exec-1",
		eventlog.EventExecutionCreated, eventlog.ExecutionCreatedPayload{BacklogKind: "execute", BacklogName: "a", Mode: "yolo"})
	appendEvent(t, repo, now, eventlog.EntityExecution, "exec-2",
		eventlog.EventExecutionCreated, eventlog.ExecutionCreatedPayload{BacklogKind: "execute", BacklogName: "b", Mode: "yolo"})
	appendEvent(t, repo, now, eventlog.EntityExecution, "exec-3",
		eventlog.EventExecutionCreated, eventlog.ExecutionCreatedPayload{BacklogKind: "fix", BacklogName: "c", Mode: "manual"})

	appendEvent(t, repo, now, eventlog.EntityExecution, "exec-1",
		eventlog.EventExecutionCompleted, eventlog.ExecutionCompletedPayload{DurationSeconds: 120, HadFixups: false})
	appendEvent(t, repo, now, eventlog.EntityExecution, "exec-2",
		eventlog.EventExecutionCompleted, eventlog.ExecutionCompletedPayload{DurationSeconds: 300, HadFixups: true})
	appendEvent(t, repo, now, eventlog.EntityExecution, "exec-3",
		eventlog.EventExecutionFailed, eventlog.ExecutionFailedPayload{Reason: "test failure", DurationSeconds: 60})

	if err := engine.Rebuild(ctx); err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	stats := engine.GetStats()

	if stats.Agent.TotalExecutions != 3 {
		t.Errorf("total: got %d, want 3", stats.Agent.TotalExecutions)
	}
	// 2 completed out of 3 finished (2 completed + 1 failed).
	wantSuccess := 2.0 / 3.0
	if diff := stats.Agent.SuccessRate - wantSuccess; diff > 0.01 || diff < -0.01 {
		t.Errorf("success rate: got %.3f, want %.3f", stats.Agent.SuccessRate, wantSuccess)
	}
	// 1 out of 2 completed had fixups.
	if stats.Agent.FollowUpRate != 0.5 {
		t.Errorf("followup rate: got %.2f, want 0.5", stats.Agent.FollowUpRate)
	}
	// Avg duration: (120/60 + 300/60 + 60/60) / 3 = (2 + 5 + 1) / 3 = 2.67 min.
	if stats.Agent.AvgExecutionMinutes < 2.6 || stats.Agent.AvgExecutionMinutes > 2.7 {
		t.Errorf("avg duration: got %.2f, want ~2.67", stats.Agent.AvgExecutionMinutes)
	}
}

func TestIncrementalRefresh(t *testing.T) {
	engine, repo := setupEngine(t)
	ctx := context.Background()
	now := time.Now().UTC()

	// Rebuild with 1 event.
	appendEvent(t, repo, now, eventlog.EntityBacklogItem, "execute/a",
		eventlog.EventBacklogCreated, eventlog.BacklogCreatedPayload{Kind: "execute", Status: "backlog", Priority: 5})

	if err := engine.Rebuild(ctx); err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	stats := engine.GetStats()
	if stats.EventCount != 1 {
		t.Errorf("after rebuild: got %d events, want 1", stats.EventCount)
	}
	if stats.Dashboard.TotalBacklogSize != 1 {
		t.Errorf("backlog: got %d, want 1", stats.Dashboard.TotalBacklogSize)
	}

	// Add more events and refresh incrementally.
	appendEvent(t, repo, now, eventlog.EntityBacklogItem, "execute/b",
		eventlog.EventBacklogCreated, eventlog.BacklogCreatedPayload{Kind: "execute", Status: "backlog", Priority: 3})

	if err := engine.Refresh(ctx); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	stats = engine.GetStats()
	if stats.EventCount != 2 {
		t.Errorf("after refresh: got %d events, want 2", stats.EventCount)
	}
	if stats.Dashboard.TotalBacklogSize != 2 {
		t.Errorf("backlog: got %d, want 2", stats.Dashboard.TotalBacklogSize)
	}
}

func TestDashboardCompletedAllTime(t *testing.T) {
	engine, repo := setupEngine(t)
	ctx := context.Background()
	now := time.Now().UTC()

	appendEvent(t, repo, now, eventlog.EntityBacklogItem, "execute/a",
		eventlog.EventBacklogCreated, eventlog.BacklogCreatedPayload{Kind: "execute", Status: "backlog", Priority: 5})
	appendEvent(t, repo, now, eventlog.EntityBacklogItem, "execute/a",
		eventlog.EventBacklogStatusChanged, eventlog.StatusChangePayload{From: "backlog", To: "completed"})

	if err := engine.Rebuild(ctx); err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	stats := engine.GetStats()

	if stats.Dashboard.TotalCompletedAllTime != 1 {
		t.Errorf("completed all time: got %d, want 1", stats.Dashboard.TotalCompletedAllTime)
	}
	if stats.Dashboard.TotalBacklogSize != 0 {
		t.Errorf("backlog should be 0 after completion, got %d", stats.Dashboard.TotalBacklogSize)
	}
}
