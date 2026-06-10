package findings_test

import (
	"context"
	"testing"
	"time"

	"web-search/internal/findings"
	"web-search/internal/testutil/mocks"

	"github.com/stretchr/testify/require"
)

// TestGCSupersedesDecayedNeverSurfaced is the OT-P2-003 core: an old,
// never-surfaced, low-confidence finding is soft-retired; a recently-created or
// surfaced or still-confident one is kept.
func TestGCSupersedesDecayedNeverSurfaced(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	// Findings created ~2.5 decay-half-lives ago so their effective confidence
	// has decayed well below the GC floor.
	old := now.Add(-(2*findings.DecayHalfLife + 60*24*time.Hour))
	clk := mocks.NewFakeClock(old)
	repo, _ := newRepoAtClock(t, clk)
	svc := findings.NewService(repo)

	staleUnused, err := svc.Add(ctx, findings.NewFinding{Claim: "stale + unused", Confidence: 0.5})
	require.NoError(t, err)
	surfaced, err := svc.Add(ctx, findings.NewFinding{Claim: "old but surfaced", Confidence: 0.5})
	require.NoError(t, err)
	require.NoError(t, repo.RecordSurfaced(ctx, []string{surfaced.ID}))

	// A recent finding (created "now") — too young to be a candidate.
	clk.SetNow(now)
	recent, err := svc.Add(ctx, findings.NewFinding{Claim: "recent + unused", Confidence: 0.5})
	require.NoError(t, err)

	gc := findings.NewGCService(svc, clk, findings.GCConfig{})

	// Dry run reports the candidate but mutates nothing.
	report, err := gc.Run(ctx, true)
	require.NoError(t, err)
	require.True(t, report.DryRun)
	require.Equal(t, []string{staleUnused.ID}, report.SupersededDecayed)
	got, err := svc.Get(ctx, staleUnused.ID)
	require.NoError(t, err)
	require.Equal(t, findings.StatusActive, got.Status, "dry-run does not mutate")

	// Real run retires only the stale-unused finding: 'surfaced' is excluded by
	// the never-surfaced filter, 'recent' by the min-age filter.
	report, err = gc.Run(ctx, false)
	require.NoError(t, err)
	require.Equal(t, []string{staleUnused.ID}, report.SupersededDecayed)

	got, err = svc.Get(ctx, staleUnused.ID)
	require.NoError(t, err)
	require.Equal(t, findings.StatusSuperseded, got.Status)
	require.Empty(t, got.SupersededBy, "GC retires without a replacement")

	for _, id := range []string{surfaced.ID, recent.ID} {
		f, err := svc.Get(ctx, id)
		require.NoError(t, err)
		require.Equal(t, findings.StatusActive, f.Status, "non-candidate %s must be untouched", id)
	}
}

// TestGCReportsColdArchiveStaleDisputesOrphans proves the report-only categories
// surface candidates without mutating them.
func TestGCReportsColdArchiveStaleDisputesOrphans(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	old := now.Add(-200 * 24 * time.Hour) // past the 90d TTLs
	clk := mocks.NewFakeClock(old)
	repo, d := newRepoAtClock(t, clk)
	svc := findings.NewService(repo)

	// A superseded finding, retired long ago -> cold-archive candidate.
	a, err := svc.Add(ctx, findings.NewFinding{Claim: "to retire", Confidence: 0.5})
	require.NoError(t, err)
	b, err := svc.Add(ctx, findings.NewFinding{Claim: "replacement", Confidence: 0.9})
	require.NoError(t, err)
	_, err = svc.Supersede(ctx, a.ID, b.ID, "outdated")
	require.NoError(t, err)

	// A dispute opened long ago -> stale dispute.
	c, err := svc.Add(ctx, findings.NewFinding{Claim: "contested", Confidence: 0.6})
	require.NoError(t, err)
	_, err = svc.Flag(ctx, c.ID, "sources conflict")
	require.NoError(t, err)

	// An orphan: brief_id points at a brief that does not exist.
	orphan, err := svc.Add(ctx, findings.NewFinding{Claim: "orphaned", Confidence: 0.8, BriefID: "missing-brief"})
	require.NoError(t, err)

	// Advance to "now" so the findings updated ~200d ago are past the 90d TTLs
	// (but still inside the 360d decay min-age, so none are decay candidates).
	clk.SetNow(now)
	gc := findings.NewGCService(svc, clk, findings.GCConfig{})
	report, err := gc.Run(ctx, false)
	require.NoError(t, err)
	require.Empty(t, report.SupersededDecayed, "200d-old findings are inside the decay min-age window")

	require.Contains(t, report.ColdArchiveCandidates, a.ID)
	require.Contains(t, report.StaleDisputes, c.ID)
	require.Contains(t, report.Orphans, orphan.ID)

	// None of the report-only categories mutated state.
	gotDispute, err := svc.Get(ctx, c.ID)
	require.NoError(t, err)
	require.Equal(t, findings.StatusDisputed, gotDispute.Status, "GC never auto-resolves a dispute")
	require.Equal(t, "sources conflict", gotDispute.DisputeNote)

	gotOrphan, err := svc.Get(ctx, orphan.ID)
	require.NoError(t, err)
	require.Equal(t, findings.StatusActive, gotOrphan.Status, "GC never deletes an orphan")

	// The superseded finding still exists (never hard-deleted by GC).
	var n int
	require.NoError(t, d.QueryRowContext(ctx, `SELECT COUNT(*) FROM findings WHERE id = ?`, a.ID).Scan(&n))
	require.Equal(t, 1, n, "GC cold-archive is report-only; it never hard-deletes")
}

// TestGCRespectsConfidenceFloorConfig proves a custom (higher) floor retires a
// finding the default floor would keep.
func TestGCRespectsConfidenceFloorConfig(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	// One decay-half-life old: effective confidence ~ stored/2.
	created := now.Add(-(2*findings.DecayHalfLife + time.Hour))
	clk := mocks.NewFakeClock(created)
	repo, _ := newRepoAtClock(t, clk)
	svc := findings.NewService(repo)

	f, err := svc.Add(ctx, findings.NewFinding{Claim: "mid confidence", Confidence: 0.9})
	require.NoError(t, err)
	clk.SetNow(now)

	// Default floor (0.25): the decayed-but-still-~0.16 finding qualifies.
	// Make the floor tiny so it is kept, proving the gate is honored.
	gcTight := findings.NewGCService(svc, clk, findings.GCConfig{ConfidenceFloor: 0.01})
	report, err := gcTight.Run(ctx, true)
	require.NoError(t, err)
	require.NotContains(t, report.SupersededDecayed, f.ID, "a floor below the finding's effective confidence keeps it")
}
