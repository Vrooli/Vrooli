package stats

import (
	"context"
	"testing"
	"time"

	"swarm-manager/internal/eventlog"
)

func emitRecordCreated(t *testing.T, repo *eventlog.SQLiteRepository, ts time.Time, recordID string, p eventlog.RecordCreatedPayload) {
	t.Helper()
	appendEvent(t, repo, ts, eventlog.EntityRecord, recordID, eventlog.EventRecordCreated, p)
}

func emitRecordSuperseded(t *testing.T, repo *eventlog.SQLiteRepository, ts time.Time, successorID string, p eventlog.RecordSupersededPayload) {
	t.Helper()
	appendEvent(t, repo, ts, eventlog.EntityRecord, successorID, eventlog.EventRecordSuperseded, p)
}

func TestEngine_RecordCounters(t *testing.T) {
	engine, repo := setupEngine(t)
	now := time.Now().UTC()

	emitRecordCreated(t, repo, now.Add(-2*time.Hour), "rec-1", eventlog.RecordCreatedPayload{
		Kind: "fix", Scenario: "audio-tools", BacklogRef: "fix/voice-auto-stop", Stub: false,
	})
	emitRecordCreated(t, repo, now.Add(-3*24*time.Hour), "rec-2", eventlog.RecordCreatedPayload{
		Kind: "execute", Scenario: "swarm-manager", Stub: false,
	})
	emitRecordCreated(t, repo, now.Add(-20*24*time.Hour), "rec-3", eventlog.RecordCreatedPayload{
		Kind: "fix", Scenario: "audio-tools", BacklogRef: "fix/y", Stub: true,
	})
	// 60 days ago, outside both windows.
	emitRecordCreated(t, repo, now.Add(-60*24*time.Hour), "rec-4", eventlog.RecordCreatedPayload{
		Kind: "fix", Scenario: "audio-tools", Stub: false,
	})
	emitRecordSuperseded(t, repo, now.Add(-1*time.Hour), "rec-1", eventlog.RecordSupersededPayload{
		SupersededID: "rec-4", Reason: "regression",
	})

	if err := engine.Rebuild(context.Background()); err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	r := engine.GetStats().Records

	if r.TotalRecords != 4 {
		t.Errorf("TotalRecords = %d, want 4", r.TotalRecords)
	}
	if r.CreatedLast7Days != 2 {
		t.Errorf("CreatedLast7Days = %d, want 2", r.CreatedLast7Days)
	}
	if r.CreatedLast30Days != 3 {
		t.Errorf("CreatedLast30Days = %d, want 3", r.CreatedLast30Days)
	}
	if r.ByKind["fix"] != 3 || r.ByKind["execute"] != 1 {
		t.Errorf("ByKind = %v", r.ByKind)
	}
	if r.ByScenario["audio-tools"] != 3 || r.ByScenario["swarm-manager"] != 1 {
		t.Errorf("ByScenario = %v", r.ByScenario)
	}
	if r.WithBacklogRef != 2 || r.WithoutBacklogRef != 2 {
		t.Errorf("with/without backlog_ref = %d/%d (want 2/2)", r.WithBacklogRef, r.WithoutBacklogRef)
	}
	if r.Stubs != 1 {
		t.Errorf("Stubs = %d, want 1", r.Stubs)
	}
	if r.SupersedeCount != 1 {
		t.Errorf("SupersedeCount = %d, want 1", r.SupersedeCount)
	}
	wantRegression := 1.0 / 4.0
	if !approxEq(r.RegressionRate, wantRegression) {
		t.Errorf("RegressionRate = %v, want %v", r.RegressionRate, wantRegression)
	}
}

func TestEngine_RecordCounters_EmptyByDefault(t *testing.T) {
	engine, _ := setupEngine(t)
	if err := engine.Rebuild(context.Background()); err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	r := engine.GetStats().Records
	if r.TotalRecords != 0 || r.RegressionRate != 0 {
		t.Errorf("expected zero-value RecordStats, got %+v", r)
	}
	if r.ByKind == nil || r.ByScenario == nil {
		t.Errorf("expected non-nil maps so JSON renders {} not null: %+v", r)
	}
}

func TestEngine_RecordCounters_MapsAreCopiesNotAliases(t *testing.T) {
	engine, repo := setupEngine(t)
	emitRecordCreated(t, repo, time.Now().UTC(), "rec-1", eventlog.RecordCreatedPayload{
		Kind: "fix", Scenario: "s",
	})
	if err := engine.Rebuild(context.Background()); err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	r := engine.GetStats().Records
	// Mutating the response map must not corrupt the engine's internal counters.
	r.ByKind["fix"] = 999
	r2 := engine.GetStats().Records
	if r2.ByKind["fix"] != 1 {
		t.Errorf("response map should be a copy; engine state mutated to %v", r2.ByKind)
	}
}
