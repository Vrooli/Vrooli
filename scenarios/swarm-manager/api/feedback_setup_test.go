package main

import (
	"testing"
	"time"

	"swarm-manager/internal/initiativelock"
	"swarm-manager/internal/initiatives"
)

// TestSweepStaleFeedbackLocks_ClearsExpiredLocksAcrossInitiatives pins the
// boot-time stale-lock sweep: a server crash while a feedback or review
// round held the initiative lock must not wedge future feedback
// submissions behind a dead lockfile. The sweep walks every initiative on
// disk and clears locks older than MaxAge; live locks are left alone so
// rounds that survived the restart can resume polling.
func TestSweepStaleFeedbackLocks_ClearsExpiredLocksAcrossInitiatives(t *testing.T) {
	root := t.TempDir()
	initStore := initiatives.NewStore(root)

	// Seed two initiatives on disk so the sweep has something to iterate.
	for _, name := range []string{"stale-one", "live-two"} {
		init := &initiatives.Initiative{
			Name:   name,
			Title:  name,
			Status: initiatives.InitiativeStatusActive,
		}
		if err := initStore.Save(init); err != nil {
			t.Fatalf("seed %s: %v", name, err)
		}
	}

	now := time.Now().UTC()
	lock := &initiativelock.Lock{
		Dir:    initStore.InitDir,
		MaxAge: time.Hour,
	}

	// "stale-one" holds a 3-hour-old lock (well past MaxAge).
	lock.Clock = func() time.Time { return now.Add(-3 * time.Hour) }
	if err := lock.Acquire("stale-one", initiativelock.Holder{RunID: "dead-run", Purpose: "feedback"}); err != nil {
		t.Fatalf("seed stale lock: %v", err)
	}
	// "live-two" holds a fresh lock; sweep must leave it alone.
	lock.Clock = func() time.Time { return now }
	if err := lock.Acquire("live-two", initiativelock.Holder{RunID: "live-run", Purpose: "review"}); err != nil {
		t.Fatalf("seed live lock: %v", err)
	}

	// Run the boot-path sweep.
	sweepStaleFeedbackLocks(initStore, lock)

	if h, _ := lock.Inspect("stale-one"); h != nil {
		t.Errorf("expected stale lock cleared on boot, got holder %+v", h)
	}
	if h, _ := lock.Inspect("live-two"); h == nil {
		t.Errorf("expected live lock preserved, got nil")
	}
}

// TestSweepStaleFeedbackLocks_NoInitiativesIsNoop ensures the sweep
// tolerates an empty store (fresh deployment). Regression guard against a
// past bug where the caller crashed on an empty LoadAll result.
func TestSweepStaleFeedbackLocks_NoInitiativesIsNoop(t *testing.T) {
	root := t.TempDir()
	initStore := initiatives.NewStore(root)
	lock := &initiativelock.Lock{Dir: initStore.InitDir, MaxAge: time.Hour}

	// Must not panic; nothing to assert beyond survival.
	sweepStaleFeedbackLocks(initStore, lock)
}
