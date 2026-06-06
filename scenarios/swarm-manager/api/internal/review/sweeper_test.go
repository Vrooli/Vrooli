package review

import (
	"context"
	"testing"
	"time"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/backlog"
)

// fakeLister returns a fixed set of backlog items for the sweeper.
type fakeLister struct{ items []backlog.BacklogItem }

func (f *fakeLister) LoadAll(_ []backlog.BacklogKind) ([]backlog.BacklogItem, error) {
	return f.items, nil
}

// TestSweeper_RecoversOrphans verifies the sweeper recovers a stranded
// in_review item (terminal round, flip never happened) while leaving a healthy
// live-review item and a review_pending item untouched.
func TestSweeper_RecoversOrphans(t *testing.T) {
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	recent := "2026-06-03T11:59:00Z"

	orphanDir := t.TempDir()
	writeRound(t, orphanDir, Round{RoundNum: 1, GeneratedAt: recent, Status: RoundStatusComplete, Evidence: []EvidenceItem{}})

	healthyDir := t.TempDir()
	writeRound(t, healthyDir, Round{RoundNum: 1, GeneratedAt: recent, Status: RoundStatusGathering, RunID: "live", Evidence: []EvidenceItem{}})

	dirs := map[string]string{"orphan": orphanDir, "healthy": healthyDir}

	svc := newTestService(&capturingSpawner{runState: agentmanager.RunState{RunID: "live", Status: "running"}}, "")
	svc.clock = fixedClock(now)
	svc.itemDirFn = func(_, name string) string { return dirs[name] }

	lister := &fakeLister{items: []backlog.BacklogItem{
		{Kind: backlog.KindExecute, Name: "orphan", Status: backlog.StatusInReview, Updated: recent},
		{Kind: backlog.KindExecute, Name: "healthy", Status: backlog.StatusInReview, Updated: recent},
		{Kind: backlog.KindExecute, Name: "decided", Status: backlog.StatusReviewPending, Updated: recent},
	}}

	var recovered []string
	sw := &Sweeper{
		Service: svc,
		Backlog: lister,
		Recover: func(_ context.Context, kind, name, reason string) error {
			recovered = append(recovered, kind+"/"+name)
			return nil
		},
		MaxAge:   30 * time.Minute,
		Interval: time.Minute,
	}

	n, err := sw.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if n != 1 {
		t.Fatalf("recovered count = %d, want 1", n)
	}
	if len(recovered) != 1 || recovered[0] != "execute/orphan" {
		t.Fatalf("recovered = %v, want [execute/orphan]", recovered)
	}
}

// TestSweeper_NilSafe ensures a partially-wired sweeper no-ops instead of
// panicking.
func TestSweeper_NilSafe(t *testing.T) {
	sw := &Sweeper{}
	if n, err := sw.RunOnce(context.Background()); err != nil || n != 0 {
		t.Fatalf("nil sweeper RunOnce = (%d, %v), want (0, nil)", n, err)
	}
}
