package findings_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	testdb "github.com/vrooli/api-core/databasetest"
	localdb "web-search/internal/database"
	"web-search/internal/findings"

	"github.com/vrooli/api-core/schedule"

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
	return findings.NewSQLiteRepository(d, schedule.System()), d
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

// TestFindingSchemaRequiredFields pins the required-field validation surface:
// a finding without a claim is rejected with a descriptive error at the
// application layer, and the schema itself refuses NULL claim / retrieval_date
// rows, so no code path can insert an incomplete finding. (Citations are
// deliberately optional — a manual finding may be uncited — and the retrieval
// date is stamped by the repository on every insert.)
func TestFindingSchemaRequiredFields(t *testing.T) {
	repo, d := newRepo(t)
	ctx := context.Background()
	svc := findings.NewService(repo)

	// Application layer: a missing claim is a descriptive validation error.
	_, err := svc.Add(ctx, findings.NewFinding{Claim: "   "})
	var invalid findings.ErrInvalidFinding
	require.ErrorAs(t, err, &invalid)
	require.Equal(t, "claim", invalid.Field)
	require.Contains(t, err.Error(), "claim")

	// Schema layer: NULL claim and NULL retrieval_date violate NOT NULL.
	_, err = d.ExecContext(ctx,
		`INSERT INTO findings (id, claim, retrieval_date, created_at, updated_at)
		 VALUES ('x1', NULL, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`)
	require.Error(t, err, "claim is required at the schema level")
	_, err = d.ExecContext(ctx,
		`INSERT INTO findings (id, claim, retrieval_date, created_at, updated_at)
		 VALUES ('x2', 'claim', NULL, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`)
	require.Error(t, err, "retrieval_date is required at the schema level")

	// Every successfully added finding carries a non-zero retrieval date.
	added, err := svc.Add(ctx, findings.NewFinding{Claim: "stamped"})
	require.NoError(t, err)
	require.False(t, added.RetrievalDate.IsZero())
}

// TestConfidenceConstrainedToUnitRange pins that an out-of-range confidence
// can never enter the store: the service clamps input into [0,1] on both add
// and edit, so every persisted confidence is a valid probability.
func TestConfidenceConstrainedToUnitRange(t *testing.T) {
	repo, _ := newRepo(t)
	svc := findings.NewService(repo)
	ctx := context.Background()

	over, err := svc.Add(ctx, findings.NewFinding{Claim: "over", Confidence: 1.7})
	require.NoError(t, err)
	require.InDelta(t, 1.0, over.Confidence, 1e-9)

	under, err := svc.Add(ctx, findings.NewFinding{Claim: "under", Confidence: -0.4})
	require.NoError(t, err)
	require.InDelta(t, 0.0, under.Confidence, 1e-9)

	edited, err := svc.Edit(ctx, over.ID, findings.EditInput{Claim: "over", Confidence: 42})
	require.NoError(t, err)
	require.InDelta(t, 1.0, edited.Confidence, 1e-9)

	for _, id := range []string{over.ID, under.ID} {
		got, err := svc.Get(ctx, id)
		require.NoError(t, err)
		require.GreaterOrEqual(t, got.Confidence, 0.0)
		require.LessOrEqual(t, got.Confidence, 1.0)
	}
}

// TestStatusEnumEnforced pins the lifecycle status enum: exactly
// active/disputed/superseded are recognized, and an unknown status filter is
// rejected with a validation error rather than silently matching nothing.
func TestStatusEnumEnforced(t *testing.T) {
	for _, valid := range []string{findings.StatusActive, findings.StatusDisputed, findings.StatusSuperseded} {
		require.True(t, findings.ValidStatus(valid), valid)
	}
	for _, unknown := range []string{"archived", "deleted", "ACTIVE", "open"} {
		require.False(t, findings.ValidStatus(unknown), unknown)
	}

	repo, _ := newRepo(t)
	svc := findings.NewService(repo)
	_, err := svc.List(context.Background(), findings.ListFilter{Status: "archived"})
	var invalid findings.ErrInvalidFinding
	require.ErrorAs(t, err, &invalid)
	require.Equal(t, "status", invalid.Field)
}

// TestBriefSchemaRequiresQueryAndRunTimestamp pins the briefs schema contract:
// a brief row cannot carry a NULL query or run_timestamp, and run_timestamp
// has no default so it must always be supplied.
func TestBriefSchemaRequiresQueryAndRunTimestamp(t *testing.T) {
	_, d := newRepo(t)
	ctx := context.Background()

	_, err := d.ExecContext(ctx,
		`INSERT INTO briefs (id, query, run_timestamp) VALUES ('b1', NULL, '2026-01-01T00:00:00Z')`)
	require.Error(t, err, "NULL query must be rejected")

	_, err = d.ExecContext(ctx,
		`INSERT INTO briefs (id, query, run_timestamp) VALUES ('b2', 'q', NULL)`)
	require.Error(t, err, "NULL run_timestamp must be rejected")

	_, err = d.ExecContext(ctx, `INSERT INTO briefs (id, query) VALUES ('b3', 'q')`)
	require.Error(t, err, "omitted run_timestamp has no default and must be rejected")

	_, err = d.ExecContext(ctx,
		`INSERT INTO briefs (id, query, run_timestamp) VALUES ('b4', 'q', '2026-01-01T00:00:00Z')`)
	require.NoError(t, err, "a brief carrying query + run_timestamp is valid")
}

// TestFindingCRUDRoundTrip walks one finding through the full lifecycle —
// create, read, update, soft-delete (supersede) — asserting the expected
// state at every step. The soft-deleted row stays readable (never deleted).
func TestFindingCRUDRoundTrip(t *testing.T) {
	repo, _ := newRepo(t)
	svc := findings.NewService(repo)
	ctx := context.Background()

	// Create.
	created, err := svc.Add(ctx, findings.NewFinding{
		Claim:      "v1 claim",
		Confidence: 0.4,
		Citations:  []findings.NewCitation{{URL: "https://src.example", Title: "Src"}},
	})
	require.NoError(t, err)

	// Read.
	got, err := svc.Get(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, "v1 claim", got.Claim)
	require.InDelta(t, 0.4, got.Confidence, 1e-9)
	require.Equal(t, findings.StatusActive, got.Status)
	require.Len(t, got.Citations, 1)

	// Update (edit fields).
	_, err = svc.Edit(ctx, created.ID, findings.EditInput{Claim: "v2 claim", Confidence: 0.9})
	require.NoError(t, err)
	got, err = svc.Get(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, "v2 claim", got.Claim)
	require.InDelta(t, 0.9, got.Confidence, 1e-9)

	// Soft-delete (supersede) and read back the archived state.
	replacement, err := svc.Add(ctx, findings.NewFinding{Claim: "v3 claim", Confidence: 0.95})
	require.NoError(t, err)
	_, err = svc.Supersede(ctx, created.ID, replacement.ID, "rewritten")
	require.NoError(t, err)
	got, err = svc.Get(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, findings.StatusSuperseded, got.Status)
	require.Equal(t, replacement.ID, got.SupersededBy)
	require.Equal(t, "v2 claim", got.Claim, "the archived claim text is preserved")
}

// TestAuditMetadataCarriesActorReasonAndBrief asserts every mutation writes
// the full audit metadata — actor (who mutated), reason (why), and
// source_brief_id (provenance) — not merely the mutation type.
func TestAuditMetadataCarriesActorReasonAndBrief(t *testing.T) {
	repo, d := newRepo(t)
	ctx := context.Background()

	a, err := repo.Add(ctx, findings.NewFinding{Claim: "audited", BriefID: "brief-1"}, "agent")
	require.NoError(t, err)
	b, err := repo.Add(ctx, findings.NewFinding{Claim: "replacement"}, "agent")
	require.NoError(t, err)
	_, err = repo.Edit(ctx, a.ID, findings.EditInput{Claim: "audited v2", Confidence: 0.5}, "operator")
	require.NoError(t, err)
	_, err = repo.Flag(ctx, a.ID, "sources disagree", "agent")
	require.NoError(t, err)
	_, err = repo.Supersede(ctx, a.ID, b.ID, "newer evidence", "operator")
	require.NoError(t, err)

	// The archived row itself records what replaced it.
	archived, err := repo.Get(ctx, a.ID)
	require.NoError(t, err)
	require.Equal(t, findings.StatusSuperseded, archived.Status)
	require.Equal(t, b.ID, archived.SupersededBy)

	type auditRow struct{ mutation, reason, brief, actor string }
	rows, err := d.QueryContext(ctx,
		`SELECT mutation_type, reason, source_brief_id, actor FROM finding_audit
		  WHERE finding_id = ? ORDER BY created_at ASC, id ASC`, a.ID)
	require.NoError(t, err)
	defer rows.Close()
	var got []auditRow
	for rows.Next() {
		var r auditRow
		require.NoError(t, rows.Scan(&r.mutation, &r.reason, &r.brief, &r.actor))
		got = append(got, r)
	}
	require.NoError(t, rows.Err())

	require.Equal(t, []auditRow{
		{mutation: findings.MutationCreate, reason: "", brief: "brief-1", actor: "agent"},
		{mutation: findings.MutationEdit, reason: "", brief: "", actor: "operator"},
		{mutation: findings.MutationFlag, reason: "sources disagree", brief: "", actor: "agent"},
		{mutation: findings.MutationSupersede, reason: "newer evidence", brief: "", actor: "operator"},
	}, got)
}

// TestFindingProvenanceComplete asserts a stored finding carries the full
// provenance a human auditor needs: source URLs + titles, retrieval date,
// originating query, originating brief id, and the capture source.
func TestFindingProvenanceComplete(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()

	added, err := repo.Add(ctx, findings.NewFinding{
		Claim:   "provenance-bearing claim",
		Query:   "original question",
		BriefID: "brief-42",
		Source:  findings.SourceL3,
		Citations: []findings.NewCitation{
			{URL: "https://one.example", Title: "One"},
			{URL: "https://two.example", Title: "Two"},
		},
	}, "agent")
	require.NoError(t, err)

	got, err := repo.Get(ctx, added.ID)
	require.NoError(t, err)
	require.Equal(t, "original question", got.Query)
	require.Equal(t, "brief-42", got.BriefID)
	require.Equal(t, findings.SourceL3, got.Source)
	require.False(t, got.RetrievalDate.IsZero(), "retrieval date is always stamped")
	require.Len(t, got.Citations, 2)
	urls := []string{got.Citations[0].URL, got.Citations[1].URL}
	require.ElementsMatch(t, []string{"https://one.example", "https://two.example"}, urls)
	for _, c := range got.Citations {
		require.NotEmpty(t, c.Title)
		require.False(t, c.RetrievedAt.IsZero())
	}
}

// TestPruneNeverDeletesActiveOrDisputed pins the no-hard-delete contract: the
// ONLY delete-capable path (prune) refuses everything except superseded rows,
// so an active or disputed finding cannot be hard-deleted through any API.
func TestPruneNeverDeletesActiveOrDisputed(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()

	active, err := repo.Add(ctx, findings.NewFinding{Claim: "active"}, "operator")
	require.NoError(t, err)
	disputed, err := repo.Add(ctx, findings.NewFinding{Claim: "disputed"}, "operator")
	require.NoError(t, err)
	_, err = repo.Flag(ctx, disputed.ID, "contested", "operator")
	require.NoError(t, err)

	ids, err := repo.Prune(ctx, false, "operator")
	require.NoError(t, err)
	require.Empty(t, ids, "nothing is superseded, so nothing may be deleted")

	for _, id := range []string{active.ID, disputed.ID} {
		_, err := repo.Get(ctx, id)
		require.NoError(t, err, "finding %s must survive prune", id)
	}
}

// TestDefaultListIncludesActiveAndDisputed asserts the default read surface
// returns both active and disputed findings (only superseded is hidden).
func TestDefaultListIncludesActiveAndDisputed(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	active, err := repo.Add(ctx, findings.NewFinding{Claim: "active"}, "operator")
	require.NoError(t, err)
	disputed, err := repo.Add(ctx, findings.NewFinding{Claim: "contested"}, "operator")
	require.NoError(t, err)
	_, err = repo.Flag(ctx, disputed.ID, "conflict", "operator")
	require.NoError(t, err)

	def, err := repo.List(ctx, findings.ListFilter{})
	require.NoError(t, err)
	ids := map[string]bool{}
	for _, f := range def {
		ids[f.ID] = true
	}
	require.True(t, ids[active.ID], "active findings are in the default view")
	require.True(t, ids[disputed.ID], "disputed findings are in the default view")
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

// TestEditSupersededRejected pins the archived-history invariant: a superseded
// finding is the audit record of what was believed and replaced, so editing it
// is rejected with a validation error and the stored row stays untouched.
func TestEditSupersededRejected(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()

	a, _ := repo.Add(ctx, findings.NewFinding{Claim: "old claim", Confidence: 0.6, Source: findings.SourceManual}, "operator")
	b, _ := repo.Add(ctx, findings.NewFinding{Claim: "new claim", Confidence: 0.9, Source: findings.SourceManual}, "operator")
	_, err := repo.Supersede(ctx, a.ID, b.ID, "replaced", "operator")
	require.NoError(t, err)

	_, err = repo.Edit(ctx, a.ID, findings.EditInput{Claim: "rewritten history", Confidence: 0.1}, "operator")
	require.ErrorAs(t, err, &findings.ErrInvalidFinding{})

	got, err := repo.Get(ctx, a.ID)
	require.NoError(t, err)
	require.Equal(t, "old claim", got.Claim, "archived claim is immutable")
	require.Equal(t, findings.StatusSuperseded, got.Status)

	// Active findings remain editable.
	edited, err := repo.Edit(ctx, b.ID, findings.EditInput{Claim: "refined claim", Confidence: 0.95}, "operator")
	require.NoError(t, err)
	require.Equal(t, "refined claim", edited.Claim)
}
