package operations

import (
	"context"
	"errors"
	"testing"
	"time"

	"swarm-manager/internal/initiatives"
	"swarm-manager/internal/overview"
)

// initWithRollup is a small builder for InitiativeWithRollup test fixtures.
func initWithRollup(name string, priority int, status string, dependsOn, items []string, rollup initiatives.RollupStatus) initiatives.InitiativeWithRollup {
	return initiatives.InitiativeWithRollup{
		Initiative: initiatives.Initiative{
			Name:      name,
			Title:     name + " title",
			Status:    status,
			Priority:  priority,
			DependsOn: dependsOn,
			Items:     items,
		},
		Rollup: rollup,
	}
}

func mustSnapshotBuilder(t *testing.T, cfg SnapshotBuilderConfig) *SnapshotBuilder {
	t.Helper()
	b, err := NewSnapshotBuilder(cfg)
	if err != nil {
		t.Fatalf("NewSnapshotBuilder: %v", err)
	}
	return b
}

func rankedByName(snap *OperationsSnapshot) map[string]RankedInitiative {
	out := make(map[string]RankedInitiative, len(snap.Initiatives))
	for _, ri := range snap.Initiatives {
		out[ri.Name] = ri
	}
	return out
}

func TestSnapshotRanksPrioritizedAscendingThenUnprioritized(t *testing.T) {
	resp := &overview.OverviewResponse{
		Initiatives: []initiatives.InitiativeWithRollup{
			initWithRollup("unprioritized", 0, initiatives.InitiativeStatusActive, nil, nil, initiatives.RollupStatus{}),
			initWithRollup("p10", 10, initiatives.InitiativeStatusActive, nil, nil, initiatives.RollupStatus{}),
			initWithRollup("p1", 1, initiatives.InitiativeStatusActive, nil, nil, initiatives.RollupStatus{}),
		},
		Summary: overview.OverviewSummary{ActiveInitiatives: 3, TotalItems: 0},
	}
	b := mustSnapshotBuilder(t, SnapshotBuilderConfig{Overview: fakeOverviewReader{resp: resp}, Now: fixedNow()})

	snap, err := b.GetSnapshot(context.Background())
	if err != nil {
		t.Fatalf("GetSnapshot: %v", err)
	}

	gotOrder := []string{}
	for _, ri := range snap.Initiatives {
		gotOrder = append(gotOrder, ri.Name)
	}
	want := []string{"p1", "p10", "unprioritized"}
	if len(gotOrder) != len(want) {
		t.Fatalf("expected %d initiatives, got %v", len(want), gotOrder)
	}
	for i := range want {
		if gotOrder[i] != want[i] {
			t.Fatalf("ranking order = %v, want %v", gotOrder, want)
		}
	}
}

func TestSnapshotClassifiesReadiness(t *testing.T) {
	resp := &overview.OverviewResponse{
		Initiatives: []initiatives.InitiativeWithRollup{
			// ready: active, no deps, no in-progress work, no blocked members.
			initWithRollup("ready-one", 1, initiatives.InitiativeStatusActive, nil, []string{"execute/clear"}, initiatives.RollupStatus{Pending: 2}),
			// complete: status completed regardless of anything else.
			initWithRollup("done-one", 2, initiatives.InitiativeStatusCompleted, nil, nil, initiatives.RollupStatus{Completed: 3}),
			// in_progress: has active member work.
			initWithRollup("working-one", 3, initiatives.InitiativeStatusActive, nil, nil, initiatives.RollupStatus{InProgress: 1}),
			// blocked: depends on an incomplete initiative.
			initWithRollup("blocked-one", 4, initiatives.InitiativeStatusActive, []string{"ready-one"}, nil, initiatives.RollupStatus{Pending: 1}),
		},
		// blocked member item drives a separate blocked classification below.
		DependencyGraph: overview.DependencyGraph{Blocked: []string{"execute/stuck"}},
	}
	// Add an initiative whose only blocker is a blocked member item.
	resp.Initiatives = append(resp.Initiatives,
		initWithRollup("member-blocked", 5, initiatives.InitiativeStatusActive, nil, []string{"execute/stuck"}, initiatives.RollupStatus{Pending: 1}))

	b := mustSnapshotBuilder(t, SnapshotBuilderConfig{Overview: fakeOverviewReader{resp: resp}, Now: fixedNow()})
	snap, err := b.GetSnapshot(context.Background())
	if err != nil {
		t.Fatalf("GetSnapshot: %v", err)
	}

	byName := rankedByName(snap)
	cases := map[string]string{
		"ready-one":      ReadinessReady,
		"done-one":       ReadinessComplete,
		"working-one":    ReadinessInProgress,
		"blocked-one":    ReadinessBlocked,
		"member-blocked": ReadinessBlocked,
	}
	for name, wantReadiness := range cases {
		ri, ok := byName[name]
		if !ok {
			t.Fatalf("initiative %q missing from snapshot", name)
		}
		if ri.Readiness != wantReadiness {
			t.Errorf("%s readiness = %q, want %q", name, ri.Readiness, wantReadiness)
		}
	}

	if got := byName["blocked-one"].IncompleteDeps; len(got) != 1 || got[0] != "ready-one" {
		t.Errorf("blocked-one IncompleteDeps = %v, want [ready-one]", got)
	}
	if got := byName["member-blocked"].BlockedMemberItems; len(got) != 1 || got[0] != "execute/stuck" {
		t.Errorf("member-blocked BlockedMemberItems = %v, want [execute/stuck]", got)
	}

	if snap.Summary.BlockedInitiatives != 2 {
		t.Errorf("BlockedInitiatives = %d, want 2", snap.Summary.BlockedInitiatives)
	}
	if snap.Summary.ReadyInitiatives != 1 {
		t.Errorf("ReadyInitiatives = %d, want 1", snap.Summary.ReadyInitiatives)
	}
}

func TestSnapshotDownstreamUnblocksCountAndTiebreak(t *testing.T) {
	// foundation has no priority; two not-complete initiatives depend on it,
	// and one completed initiative depends on it (must NOT count). A second
	// unprioritized initiative "lonely" has zero downstreams, so foundation
	// must sort ahead of it on the downstream-unblocks tiebreak.
	resp := &overview.OverviewResponse{
		Initiatives: []initiatives.InitiativeWithRollup{
			initWithRollup("foundation", 0, initiatives.InitiativeStatusActive, nil, nil, initiatives.RollupStatus{}),
			initWithRollup("lonely", 0, initiatives.InitiativeStatusActive, nil, nil, initiatives.RollupStatus{}),
			initWithRollup("dep-a", 0, initiatives.InitiativeStatusActive, []string{"foundation"}, nil, initiatives.RollupStatus{}),
			initWithRollup("dep-b", 0, initiatives.InitiativeStatusActive, []string{"foundation"}, nil, initiatives.RollupStatus{}),
			initWithRollup("dep-done", 0, initiatives.InitiativeStatusCompleted, []string{"foundation"}, nil, initiatives.RollupStatus{}),
		},
	}
	b := mustSnapshotBuilder(t, SnapshotBuilderConfig{Overview: fakeOverviewReader{resp: resp}, Now: fixedNow()})
	snap, err := b.GetSnapshot(context.Background())
	if err != nil {
		t.Fatalf("GetSnapshot: %v", err)
	}

	byName := rankedByName(snap)
	if got := byName["foundation"].DownstreamUnblocks; got != 2 {
		t.Errorf("foundation DownstreamUnblocks = %d, want 2 (completed downstream excluded)", got)
	}
	if got := byName["lonely"].DownstreamUnblocks; got != 0 {
		t.Errorf("lonely DownstreamUnblocks = %d, want 0", got)
	}

	// foundation (2 downstreams, ready) must outrank lonely (0 downstreams,
	// ready) among equal-priority unprioritized initiatives.
	foundationIdx, lonelyIdx := -1, -1
	for i, ri := range snap.Initiatives {
		switch ri.Name {
		case "foundation":
			foundationIdx = i
		case "lonely":
			lonelyIdx = i
		}
	}
	if foundationIdx > lonelyIdx {
		t.Errorf("expected foundation (idx %d) to outrank lonely (idx %d) on downstream-unblocks tiebreak", foundationIdx, lonelyIdx)
	}
}

func TestSnapshotCacheHitThenExpiry(t *testing.T) {
	clock := &mutableClock{t: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	counting := &countingOverviewReader{resp: &overview.OverviewResponse{
		Initiatives: []initiatives.InitiativeWithRollup{
			initWithRollup("only", 1, initiatives.InitiativeStatusActive, nil, nil, initiatives.RollupStatus{}),
		},
	}}
	b := mustSnapshotBuilder(t, SnapshotBuilderConfig{Overview: counting, TTL: 120 * time.Second, Now: clock.Now})

	if _, err := b.GetSnapshot(context.Background()); err != nil {
		t.Fatalf("first GetSnapshot: %v", err)
	}
	// Second call within TTL: served from cache, no new overview load.
	clock.advance(60 * time.Second)
	if _, err := b.GetSnapshot(context.Background()); err != nil {
		t.Fatalf("second GetSnapshot: %v", err)
	}
	if counting.calls != 1 {
		t.Fatalf("expected 1 overview load within TTL, got %d", counting.calls)
	}

	// Past TTL: rebuild triggers a fresh overview load.
	clock.advance(61 * time.Second)
	if _, err := b.GetSnapshot(context.Background()); err != nil {
		t.Fatalf("third GetSnapshot: %v", err)
	}
	if counting.calls != 2 {
		t.Fatalf("expected 2 overview loads after TTL expiry, got %d", counting.calls)
	}

	// Invalidate forces a rebuild even within TTL.
	b.Invalidate()
	if _, err := b.GetSnapshot(context.Background()); err != nil {
		t.Fatalf("post-invalidate GetSnapshot: %v", err)
	}
	if counting.calls != 3 {
		t.Fatalf("expected 3 overview loads after Invalidate, got %d", counting.calls)
	}
}

func TestSnapshotPropagatesOverviewError(t *testing.T) {
	b := mustSnapshotBuilder(t, SnapshotBuilderConfig{
		Overview: fakeOverviewReader{err: errors.New("disk gone")},
		Now:      fixedNow(),
	})
	if _, err := b.GetSnapshot(context.Background()); err == nil {
		t.Fatal("expected error when overview load fails")
	}
}

// mutableClock is a test clock whose Now advances only when the test calls
// advance — so cache TTL behavior is deterministic.
type mutableClock struct {
	t time.Time
}

func (c *mutableClock) Now() time.Time          { return c.t }
func (c *mutableClock) advance(d time.Duration) { c.t = c.t.Add(d) }

// countingOverviewReader records how many times GetOverview was called so a
// test can assert cache hits vs rebuilds.
type countingOverviewReader struct {
	resp  *overview.OverviewResponse
	calls int
}

func (c *countingOverviewReader) GetOverview() (*overview.OverviewResponse, error) {
	c.calls++
	return c.resp, nil
}
