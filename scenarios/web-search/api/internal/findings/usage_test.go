package findings_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	localdb "web-search/internal/database"
	"web-search/internal/findings"

	testdb "github.com/vrooli/api-core/databasetest"

	"github.com/vrooli/api-core/schedule"
	"github.com/vrooli/api-core/scheduletest"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
)

// newRepoAtClock builds a findings repository over a fresh SQLite DB whose
// timestamps are driven by clk, so usage/age behavior is deterministic.
func newRepoAtClock(t *testing.T, clk schedule.Clock) (findings.Repository, *sql.DB) {
	t.Helper()
	d := testdb.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(findings.Schema),
	))
	return findings.NewSQLiteRepository(d, clk), d
}

// TestRecordSurfacedAndGetUsage proves surfacing increments persist and a
// never-surfaced finding has no usage row (zero Usage).
func TestRecordSurfacedAndGetUsage(t *testing.T) {
	ctx := context.Background()
	start := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	clk := scheduletest.New(start)
	repo, _ := newRepoAtClock(t, clk)

	a, err := repo.Add(ctx, findings.NewFinding{Claim: "claim a"}, "agent")
	require.NoError(t, err)
	b, err := repo.Add(ctx, findings.NewFinding{Claim: "claim b"}, "agent")
	require.NoError(t, err)

	// Surface a twice, b once.
	require.NoError(t, repo.RecordSurfaced(ctx, []string{a.ID, b.ID}))
	clk.Advance(time.Hour)
	require.NoError(t, repo.RecordSurfaced(ctx, []string{a.ID}))

	usage, err := repo.GetUsage(ctx, []string{a.ID, b.ID})
	require.NoError(t, err)
	require.Equal(t, 2, usage[a.ID].SurfacedCount)
	require.Equal(t, 1, usage[b.ID].SurfacedCount)
	require.Equal(t, start.Add(time.Hour), usage[a.ID].LastSurfacedAt.UTC(), "last_surfaced_at advances with the clock")
	require.Equal(t, start, usage[b.ID].LastSurfacedAt.UTC())

	// A finding never surfaced has no row.
	c, err := repo.Add(ctx, findings.NewFinding{Claim: "never surfaced"}, "agent")
	require.NoError(t, err)
	usage, err = repo.GetUsage(ctx, []string{c.ID})
	require.NoError(t, err)
	_, ok := usage[c.ID]
	require.False(t, ok, "no usage row until surfaced")
}

// TestRecordUsedValidatesTarget proves explicit usage increments persist and a
// bogus id is reported as not-found (never creates an orphan counter).
func TestRecordUsedValidatesTarget(t *testing.T) {
	ctx := context.Background()
	repo, _ := newRepoAtClock(t, schedule.System())

	f, err := repo.Add(ctx, findings.NewFinding{Claim: "used claim"}, "operator")
	require.NoError(t, err)

	require.NoError(t, repo.RecordUsed(ctx, f.ID))
	require.NoError(t, repo.RecordUsed(ctx, f.ID))
	usage, err := repo.GetUsage(ctx, []string{f.ID})
	require.NoError(t, err)
	require.Equal(t, 2, usage[f.ID].UsedCount)
	require.Equal(t, 0, usage[f.ID].SurfacedCount)

	var notFound findings.ErrFindingNotFound
	require.ErrorAs(t, repo.RecordUsed(ctx, "does-not-exist"), &notFound)
}

// TestUsageDoesNotMutateFinding proves the surfacing/usage telemetry never
// touches the finding row, its confidence, or its audit trail.
func TestUsageDoesNotMutateFinding(t *testing.T) {
	ctx := context.Background()
	repo, d := newRepoAtClock(t, schedule.System())

	f, err := repo.Add(ctx, findings.NewFinding{Claim: "immutable", Confidence: 0.7}, "agent")
	require.NoError(t, err)

	require.NoError(t, repo.RecordSurfaced(ctx, []string{f.ID}))
	require.NoError(t, repo.RecordUsed(ctx, f.ID))

	got, err := repo.Get(ctx, f.ID)
	require.NoError(t, err)
	require.Equal(t, f.Confidence, got.Confidence)
	require.Equal(t, f.Status, got.Status)
	require.Equal(t, f.UpdatedAt, got.UpdatedAt, "updated_at is untouched by usage telemetry")

	// Only the create audit row exists — no surfaced/used audit rows.
	require.Equal(t, []string{"create"}, auditRows(t, d, f.ID))
}

// TestUsageFactorSignal pins the OT-P2-001 usage factor: surfaced ⇒ 1.0;
// never-surfaced within grace ⇒ 1.0; never-surfaced past grace ⇒ decays toward
// the floor, never below it.
func TestUsageFactorSignal(t *testing.T) {
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	// Ever surfaced -> 1.0 even when ancient.
	ancient := findings.Finding{Confidence: 0.9, RetrievalDate: now.Add(-5 * findings.DecayHalfLife)}
	require.InDelta(t, 1.0, findings.UsageFactor(findings.Usage{SurfacedCount: 1}, ancient, now), 1e-9)

	// Never surfaced, within grace -> 1.0.
	fresh := findings.Finding{Confidence: 0.9, RetrievalDate: now.Add(-findings.UsageGracePeriod / 2)}
	require.InDelta(t, 1.0, findings.UsageFactor(findings.Usage{}, fresh, now), 1e-9)

	// Never surfaced, one usage-half-life past grace -> ~0.5 (the floor).
	old := findings.Finding{Confidence: 0.9, RetrievalDate: now.Add(-(findings.UsageGracePeriod + findings.UsageHalfLife))}
	require.InDelta(t, findings.UsageFloor, findings.UsageFactor(findings.Usage{}, old, now), 1e-6)

	// Never floors below UsageFloor however ancient.
	veryOld := findings.Finding{Confidence: 0.9, RetrievalDate: now.Add(-50 * findings.UsageHalfLife)}
	require.GreaterOrEqual(t, findings.UsageFactor(findings.Usage{}, veryOld, now), findings.UsageFloor)
}

// TestEffectiveScoreBlendsDecayAndUsage proves the blended score multiplies the
// age-decayed confidence by the usage factor.
func TestEffectiveScoreBlendsDecayAndUsage(t *testing.T) {
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	// One decay-half-life old AND never-surfaced one usage-half-life past grace.
	f := findings.Finding{Confidence: 0.8, RetrievalDate: now.Add(-findings.DecayHalfLife)}
	// Make it old enough to also trip the usage penalty fully to the floor.
	f.RetrievalDate = now.Add(-(findings.UsageGracePeriod + findings.UsageHalfLife))

	decay := findings.EffectiveConfidence(f, now)
	usage := findings.UsageFactor(findings.Usage{}, f, now)
	require.InDelta(t, decay*usage, findings.EffectiveScore(f, findings.Usage{}, now), 1e-9)

	// A surfaced finding scores strictly higher than the same finding unsurfaced.
	surfaced := findings.EffectiveScore(f, findings.Usage{SurfacedCount: 3}, now)
	unsurfaced := findings.EffectiveScore(f, findings.Usage{}, now)
	require.Greater(t, surfaced, unsurfaced)
}

// TestListDecayCandidates proves the GC eligibility query returns only active,
// never-surfaced, older-than-minAge findings, oldest-first.
func TestListDecayCandidates(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	clk := scheduletest.New(now.Add(-300 * 24 * time.Hour)) // findings created ~300d ago
	repo, _ := newRepoAtClock(t, clk)

	oldUnsurfaced, err := repo.Add(ctx, findings.NewFinding{Claim: "old, never surfaced"}, "agent")
	require.NoError(t, err)
	oldSurfaced, err := repo.Add(ctx, findings.NewFinding{Claim: "old, but surfaced"}, "agent")
	require.NoError(t, err)
	require.NoError(t, repo.RecordSurfaced(ctx, []string{oldSurfaced.ID}))

	// A recent never-surfaced finding (created "now") is within minAge.
	clk.SetNow(now)
	recent, err := repo.Add(ctx, findings.NewFinding{Claim: "recent, never surfaced"}, "agent")
	require.NoError(t, err)

	// minAge = 180d: oldUnsurfaced qualifies; oldSurfaced excluded (surfaced);
	// recent excluded (too new).
	cands, err := repo.ListDecayCandidates(ctx, 180*24*time.Hour, 50)
	require.NoError(t, err)
	ids := make([]string, 0, len(cands))
	for _, c := range cands {
		ids = append(ids, c.ID)
	}
	require.Contains(t, ids, oldUnsurfaced.ID)
	require.NotContains(t, ids, oldSurfaced.ID, "surfaced findings are not GC candidates")
	require.NotContains(t, ids, recent.ID, "findings younger than minAge are not candidates")
}
