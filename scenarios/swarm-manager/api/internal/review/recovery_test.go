package review

import (
	"context"
	"testing"
	"time"

	"swarm-manager/internal/agentmanager"
)

// fixedClock returns a constant time for deterministic age math.
func fixedClock(t time.Time) func() time.Time { return func() time.Time { return t } }

// TestRefreshGatheringRounds_MaxAgeBackstop verifies that a gathering round
// whose run keeps reporting "running" past the max age is finalized as failed
// and fires onRoundTerminal so the item can leave in_review.
func TestRefreshGatheringRounds_MaxAgeBackstop(t *testing.T) {
	itemDir := t.TempDir()
	writeRound(t, itemDir, Round{
		RoundNum:    1,
		GeneratedAt: "2026-06-03T00:00:00Z",
		ExecutionID: "exec-wedged",
		Status:      RoundStatusGathering,
		RunID:       "run-wedged",
		Evidence:    []EvidenceItem{},
	})

	spawner := &capturingSpawner{
		enabled:  true,
		runState: agentmanager.RunState{RunID: "run-wedged", Status: "running"},
	}
	svc := newTestService(spawner, "")
	// Tight budget + a clock 1h after the round so it is past max age.
	svc.roundMaxAge = 30 * time.Minute
	svc.clock = fixedClock(time.Date(2026, 6, 3, 1, 0, 0, 0, time.UTC))

	var fired bool
	svc.onRoundTerminal = func(_ context.Context, kind, name string, round Round) {
		fired = true
		if round.Status != RoundStatusFailed {
			t.Errorf("onRoundTerminal round status = %q, want failed", round.Status)
		}
		if kind != "execute" || name != "wedged-item" {
			t.Errorf("onRoundTerminal got %s/%s, want execute/wedged-item", kind, name)
		}
	}
	svc.trackActiveRound("run-wedged", "execute", "wedged-item", itemDir, 1)

	svc.RefreshGatheringRounds(context.Background())

	round, err := LoadRound(itemDir, 1)
	if err != nil {
		t.Fatalf("LoadRound: %v", err)
	}
	if round.Status != RoundStatusFailed {
		t.Fatalf("round status = %q, want failed (abandoned past max age)", round.Status)
	}
	if !fired {
		t.Fatal("onRoundTerminal was not fired for the abandoned round")
	}
	svc.mu.Lock()
	remaining := len(svc.activeRounds)
	svc.mu.Unlock()
	if remaining != 0 {
		t.Fatalf("expected abandoned round to be untracked, got %d active", remaining)
	}
}

// TestRefreshGatheringRounds_DeadRunBackstop verifies that a gathering round
// whose run has become unreachable (inspector error) is finalized once it has
// aged past max age.
func TestRefreshGatheringRounds_DeadRunBackstop(t *testing.T) {
	itemDir := t.TempDir()
	writeRound(t, itemDir, Round{
		RoundNum:    1,
		GeneratedAt: "2026-06-03T00:00:00Z",
		ExecutionID: "exec-dead",
		Status:      RoundStatusGathering,
		RunID:       "run-dead",
		Evidence:    []EvidenceItem{},
	})

	spawner := &capturingSpawner{
		enabled:  true,
		stateErr: agentmanager.ErrNotAvailable,
	}
	svc := newTestService(spawner, "")
	svc.roundMaxAge = 30 * time.Minute
	svc.clock = fixedClock(time.Date(2026, 6, 3, 2, 0, 0, 0, time.UTC))
	svc.trackActiveRound("run-dead", "execute", "dead-item", itemDir, 1)

	svc.RefreshGatheringRounds(context.Background())

	round, err := LoadRound(itemDir, 1)
	if err != nil {
		t.Fatalf("LoadRound: %v", err)
	}
	if round.Status != RoundStatusFailed {
		t.Fatalf("round status = %q, want failed (dead run past max age)", round.Status)
	}
}

// TestClassifyOrphan covers the sweeper's orphan-detection matrix.
func TestClassifyOrphan(t *testing.T) {
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	old := "2026-06-03T00:00:00Z"    // 12h before now
	recent := "2026-06-03T11:59:00Z" // 1m before now

	t.Run("no rounds, stale item -> orphan", func(t *testing.T) {
		dir := t.TempDir()
		svc := newTestService(&capturingSpawner{}, "")
		svc.clock = fixedClock(now)
		svc.itemDirFn = func(_, _ string) string { return dir }
		orphan, reason := svc.ClassifyOrphan("execute", "x", old, 30*time.Minute)
		if !orphan || reason == "" {
			t.Fatalf("want orphan with reason, got %v %q", orphan, reason)
		}
	})

	t.Run("no rounds, fresh item -> not orphan", func(t *testing.T) {
		dir := t.TempDir()
		svc := newTestService(&capturingSpawner{}, "")
		svc.clock = fixedClock(now)
		svc.itemDirFn = func(_, _ string) string { return dir }
		if orphan, _ := svc.ClassifyOrphan("execute", "x", recent, 30*time.Minute); orphan {
			t.Fatal("fresh no-round item should not be an orphan yet")
		}
	})

	t.Run("terminal round but still in_review -> orphan", func(t *testing.T) {
		dir := t.TempDir()
		writeRound(t, dir, Round{RoundNum: 1, GeneratedAt: recent, Status: RoundStatusComplete, Evidence: []EvidenceItem{}})
		svc := newTestService(&capturingSpawner{}, "")
		svc.clock = fixedClock(now)
		svc.itemDirFn = func(_, _ string) string { return dir }
		if orphan, _ := svc.ClassifyOrphan("execute", "x", recent, 30*time.Minute); !orphan {
			t.Fatal("terminal round with in_review item should be an orphan")
		}
	})

	t.Run("live gathering round, fresh -> not orphan", func(t *testing.T) {
		dir := t.TempDir()
		writeRound(t, dir, Round{RoundNum: 1, GeneratedAt: recent, Status: RoundStatusGathering, RunID: "r1", Evidence: []EvidenceItem{}})
		svc := newTestService(&capturingSpawner{runState: agentmanager.RunState{RunID: "r1", Status: "running"}}, "")
		svc.clock = fixedClock(now)
		svc.itemDirFn = func(_, _ string) string { return dir }
		if orphan, _ := svc.ClassifyOrphan("execute", "x", recent, 30*time.Minute); orphan {
			t.Fatal("a live gathering round younger than max age should be healthy")
		}
	})

	t.Run("stale gathering round -> orphan", func(t *testing.T) {
		dir := t.TempDir()
		writeRound(t, dir, Round{RoundNum: 1, GeneratedAt: old, Status: RoundStatusGathering, RunID: "r1", Evidence: []EvidenceItem{}})
		svc := newTestService(&capturingSpawner{runState: agentmanager.RunState{RunID: "r1", Status: "running"}}, "")
		svc.clock = fixedClock(now)
		svc.itemDirFn = func(_, _ string) string { return dir }
		if orphan, _ := svc.ClassifyOrphan("execute", "x", old, 30*time.Minute); !orphan {
			t.Fatal("a gathering round older than max age should be an orphan")
		}
	})
}

// TestHasLiveReviewRound verifies the recover-review guard.
func TestHasLiveReviewRound(t *testing.T) {
	t.Run("live run -> true", func(t *testing.T) {
		dir := t.TempDir()
		writeRound(t, dir, Round{RoundNum: 1, GeneratedAt: "2026-06-03T00:00:00Z", Status: RoundStatusGathering, RunID: "r1", Evidence: []EvidenceItem{}})
		svc := newTestService(&capturingSpawner{runState: agentmanager.RunState{RunID: "r1", Status: "running"}}, "")
		svc.itemDirFn = func(_, _ string) string { return dir }
		if !svc.HasLiveReviewRound("execute", "x") {
			t.Fatal("want live review round = true")
		}
	})

	t.Run("terminal run -> false", func(t *testing.T) {
		dir := t.TempDir()
		writeRound(t, dir, Round{RoundNum: 1, GeneratedAt: "2026-06-03T00:00:00Z", Status: RoundStatusGathering, RunID: "r1", Evidence: []EvidenceItem{}})
		svc := newTestService(&capturingSpawner{runState: agentmanager.RunState{RunID: "r1", Status: "complete"}}, "")
		svc.itemDirFn = func(_, _ string) string { return dir }
		if svc.HasLiveReviewRound("execute", "x") {
			t.Fatal("a terminal run is not a live review round")
		}
	})

	t.Run("no rounds -> false", func(t *testing.T) {
		dir := t.TempDir()
		svc := newTestService(&capturingSpawner{}, "")
		svc.itemDirFn = func(_, _ string) string { return dir }
		if svc.HasLiveReviewRound("execute", "x") {
			t.Fatal("no rounds means no live review")
		}
	})
}

// TestRecoverActiveRounds_PopulatesKindName ensures recovered gathering rounds
// carry Kind/Name so the onRoundTerminal flip is not skipped after a restart.
func TestRecoverActiveRounds_PopulatesKindName(t *testing.T) {
	root := t.TempDir()
	itemDir := root + "/execute/restart-item"
	writeRound(t, itemDir, Round{
		RoundNum:    1,
		GeneratedAt: "2026-06-03T00:00:00Z",
		Status:      RoundStatusGathering,
		RunID:       "run-restart",
		Evidence:    []EvidenceItem{},
	})

	svc := newTestService(&capturingSpawner{}, "")
	svc.dataRoot = root
	svc.RecoverActiveRounds()

	svc.mu.Lock()
	ar, ok := svc.activeRounds["run-restart"]
	svc.mu.Unlock()
	if !ok {
		t.Fatal("expected the gathering round to be recovered into tracking")
	}
	if ar.Kind != "execute" || ar.Name != "restart-item" {
		t.Fatalf("recovered round Kind/Name = %s/%s, want execute/restart-item", ar.Kind, ar.Name)
	}
}
