package findings_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"web-search/internal/clock"
	localdb "web-search/internal/database"
	"web-search/internal/findings"
	testdb "web-search/internal/testutil/db"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
)

func newRepo(t *testing.T) (findings.Repository, *sql.DB) {
	t.Helper()
	d := testdb.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(findings.Schema),
	))
	return findings.NewSQLiteRepository(d, clock.System{}), d
}

func auditRows(t *testing.T, d *sql.DB, findingID string) []string {
	t.Helper()
	rows, err := d.QueryContext(context.Background(),
		`SELECT mutation_type FROM finding_audit WHERE finding_id = ? ORDER BY created_at ASC, id ASC`, findingID)
	require.NoError(t, err)
	defer rows.Close()
	var out []string
	for rows.Next() {
		var m string
		require.NoError(t, rows.Scan(&m))
		out = append(out, m)
	}
	require.NoError(t, rows.Err())
	return out
}

func TestAddAndGet(t *testing.T) {
	repo, d := newRepo(t)
	ctx := context.Background()

	added, err := repo.Add(ctx, findings.NewFinding{
		Claim:      "Anthropic released Claude Opus 4.8",
		Confidence: 0.9,
		Query:      "claude opus release",
		Source:     findings.SourceManual,
		Citations:  []findings.NewCitation{{URL: "https://anthropic.com", Title: "Anthropic"}},
	}, "operator")
	require.NoError(t, err)
	require.NotEmpty(t, added.ID)
	require.Equal(t, findings.StatusActive, added.Status)
	require.Len(t, added.Citations, 1)

	got, err := repo.Get(ctx, added.ID)
	require.NoError(t, err)
	require.Equal(t, added.Claim, got.Claim)
	require.Equal(t, 0.9, got.Confidence)
	require.Len(t, got.Citations, 1)
	require.Equal(t, "Anthropic", got.Citations[0].Title)

	require.Equal(t, []string{findings.MutationCreate}, auditRows(t, d, added.ID))
}

func TestGetNotFound(t *testing.T) {
	repo, _ := newRepo(t)
	_, err := repo.Get(context.Background(), "nope")
	require.ErrorAs(t, err, &findings.ErrFindingNotFound{})
}

func TestListExcludesSupersededByDefault(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()

	a, err := repo.Add(ctx, findings.NewFinding{Claim: "old claim", Source: findings.SourceManual}, "operator")
	require.NoError(t, err)
	b, err := repo.Add(ctx, findings.NewFinding{Claim: "new claim", Source: findings.SourceManual}, "operator")
	require.NoError(t, err)

	_, err = repo.Supersede(ctx, a.ID, b.ID, "outdated", "operator")
	require.NoError(t, err)

	def, err := repo.List(ctx, findings.ListFilter{})
	require.NoError(t, err)
	require.Len(t, def, 1)
	require.Equal(t, b.ID, def[0].ID)

	all, err := repo.List(ctx, findings.ListFilter{IncludeArchived: true})
	require.NoError(t, err)
	require.Len(t, all, 2)

	onlySuperseded, err := repo.List(ctx, findings.ListFilter{Status: findings.StatusSuperseded})
	require.NoError(t, err)
	require.Len(t, onlySuperseded, 1)
	require.Equal(t, a.ID, onlySuperseded[0].ID)
}

func TestSupersedeNeverDeletes(t *testing.T) {
	repo, d := newRepo(t)
	ctx := context.Background()

	a, _ := repo.Add(ctx, findings.NewFinding{Claim: "claim a", Source: findings.SourceManual}, "operator")
	b, _ := repo.Add(ctx, findings.NewFinding{Claim: "claim b", Source: findings.SourceManual}, "operator")

	superseded, err := repo.Supersede(ctx, a.ID, b.ID, "replaced", "agent")
	require.NoError(t, err)
	require.Equal(t, findings.StatusSuperseded, superseded.Status)
	require.Equal(t, b.ID, superseded.SupersededBy)

	// The row still exists and is retrievable.
	got, err := repo.Get(ctx, a.ID)
	require.NoError(t, err)
	require.Equal(t, findings.StatusSuperseded, got.Status)

	require.Equal(t, []string{findings.MutationCreate, findings.MutationSupersede}, auditRows(t, d, a.ID))
}

func TestSupersedeUnknownReplacementRejected(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	a, _ := repo.Add(ctx, findings.NewFinding{Claim: "claim a", Source: findings.SourceManual}, "operator")
	_, err := repo.Supersede(ctx, a.ID, "ghost", "replaced", "operator")
	require.ErrorAs(t, err, &findings.ErrInvalidFinding{})
}

func TestFlagSetsDisputed(t *testing.T) {
	repo, d := newRepo(t)
	ctx := context.Background()
	a, _ := repo.Add(ctx, findings.NewFinding{Claim: "contested", Source: findings.SourceManual}, "operator")

	flagged, err := repo.Flag(ctx, a.ID, "sources conflict", "agent")
	require.NoError(t, err)
	require.Equal(t, findings.StatusDisputed, flagged.Status)
	require.Equal(t, "sources conflict", flagged.DisputeNote)
	require.Equal(t, []string{findings.MutationCreate, findings.MutationFlag}, auditRows(t, d, a.ID))
}

func TestAuditOnEveryMutation(t *testing.T) {
	repo, d := newRepo(t)
	ctx := context.Background()
	a, _ := repo.Add(ctx, findings.NewFinding{Claim: "v1", Source: findings.SourceManual}, "operator")
	_, err := repo.Edit(ctx, a.ID, findings.EditInput{Claim: "v2", Confidence: 0.5}, "operator")
	require.NoError(t, err)
	_, err = repo.Flag(ctx, a.ID, "disputed", "operator")
	require.NoError(t, err)

	require.Equal(t, []string{
		findings.MutationCreate,
		findings.MutationEdit,
		findings.MutationFlag,
	}, auditRows(t, d, a.ID))
}

func TestPruneRemovesSupersededOnly(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	a, _ := repo.Add(ctx, findings.NewFinding{Claim: "old", Source: findings.SourceManual}, "operator")
	b, _ := repo.Add(ctx, findings.NewFinding{Claim: "new", Source: findings.SourceManual}, "operator")
	_, err := repo.Supersede(ctx, a.ID, b.ID, "outdated", "operator")
	require.NoError(t, err)

	// Dry-run reports without deleting.
	ids, err := repo.Prune(ctx, true, "operator")
	require.NoError(t, err)
	require.Equal(t, []string{a.ID}, ids)
	_, err = repo.Get(ctx, a.ID)
	require.NoError(t, err, "dry-run must not delete")

	// Real prune deletes the superseded finding, leaves the active one.
	ids, err = repo.Prune(ctx, false, "operator")
	require.NoError(t, err)
	require.Equal(t, []string{a.ID}, ids)
	_, err = repo.Get(ctx, a.ID)
	require.ErrorAs(t, err, &findings.ErrFindingNotFound{})
	_, err = repo.Get(ctx, b.ID)
	require.NoError(t, err)
}

func TestLoadIndexableExcludesSuperseded(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	active, _ := repo.Add(ctx, findings.NewFinding{Claim: "active", Source: findings.SourceManual}, "operator")
	disputed, _ := repo.Add(ctx, findings.NewFinding{Claim: "disputed", Source: findings.SourceManual}, "operator")
	gone, _ := repo.Add(ctx, findings.NewFinding{Claim: "gone", Source: findings.SourceManual}, "operator")
	_, _ = repo.Flag(ctx, disputed.ID, "conflict", "operator")
	_, _ = repo.Supersede(ctx, gone.ID, active.ID, "outdated", "operator")

	idx, err := repo.LoadIndexable(ctx)
	require.NoError(t, err)
	ids := map[string]bool{}
	for _, f := range idx {
		ids[f.ID] = true
	}
	require.True(t, ids[active.ID])
	require.True(t, ids[disputed.ID])
	require.False(t, ids[gone.ID], "superseded must not be indexable")
}

func TestCountInWindow(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	_, _ = repo.Add(ctx, findings.NewFinding{Claim: "one", Source: findings.SourceManual}, "operator")
	_, _ = repo.Add(ctx, findings.NewFinding{Claim: "two", Source: findings.SourceManual}, "operator")

	now := time.Now().UTC()
	n, err := repo.Count(ctx, now.Add(-time.Hour), now.Add(time.Hour))
	require.NoError(t, err)
	require.Equal(t, 2, n)

	n, err = repo.Count(ctx, now.Add(-48*time.Hour), now.Add(-24*time.Hour))
	require.NoError(t, err)
	require.Equal(t, 0, n)
}
