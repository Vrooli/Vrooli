package validation_record_test

import (
	"context"
	"errors"
	"testing"
	"time"

	vr "development-toolchain-validator/internal/validation_record"

	db "github.com/vrooli/api-core/databasetest"

	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"

	localdb "development-toolchain-validator/internal/database"
)

func newRepo(t *testing.T) vr.Repository {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(vr.Schema),
	))
	return vr.NewSQLiteRepository(d)
}

func sampleRecord(id, golden, subject string, endedAt time.Time, kind vr.TupleKind, v vr.Verdict) vr.Record {
	return vr.Record{
		ID: id, TupleKind: kind, SubjectID: subject, GoldenSlug: golden,
		StartedAt: endedAt.Add(-time.Second), EndedAt: endedAt,
		DurationMS: 1000,
		Verdict:    v,
	}
}

func TestAppend_RoundTrips(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	r := sampleRecord("r1", "g", "s1", now, vr.TupleKindSkill, vr.VerdictPass)
	r.DiffHash = "deadbeef"
	r.DiffPathCount = 3
	r.AgentManagerRunID = "amr-1"
	require.NoError(t, repo.Append(ctx, r))

	got, err := repo.Get(ctx, "r1")
	require.NoError(t, err)
	require.Equal(t, "g", got.GoldenSlug)
	require.Equal(t, vr.VerdictPass, got.Verdict)
	require.Equal(t, "deadbeef", got.DiffHash)
	require.Equal(t, int32(3), got.DiffPathCount)
	require.WithinDuration(t, now, got.EndedAt, time.Millisecond)
}

func TestAppend_ToolFieldsRoundTrip(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()
	now := time.Date(2026, 5, 29, 12, 0, 0, 0, time.UTC)
	r := sampleRecord("rt", "reference-react-vite", "test-genie", now, vr.TupleKindTool, vr.VerdictToolFailure)
	r.ToolDetail = "2 phase(s) failed: smoke, unit"
	r.ToolRawOutput = `{"success":false,"phases":[{"name":"smoke","status":"failed"}]}`
	require.NoError(t, repo.Append(ctx, r))

	got, err := repo.Get(ctx, "rt")
	require.NoError(t, err)
	require.Equal(t, vr.TupleKindTool, got.TupleKind)
	require.Equal(t, vr.VerdictToolFailure, got.Verdict)
	require.Equal(t, "2 phase(s) failed: smoke, unit", got.ToolDetail)
	require.Equal(t, r.ToolRawOutput, got.ToolRawOutput)
}

// TestEnsureColumns_AdditiveAndIdempotent proves the migration adds the
// tool columns to a legacy table (created without them) without losing
// data, and is a no-op when re-applied.
func TestEnsureColumns_AdditiveAndIdempotent(t *testing.T) {
	d := db.NewSQLite(t)
	ctx := context.Background()

	// Legacy table: the original schema before tool_detail/tool_raw_output.
	legacy := `CREATE TABLE validation_records (
	  id TEXT PRIMARY KEY, tuple_kind INTEGER NOT NULL, subject_id TEXT NOT NULL,
	  golden_slug TEXT NOT NULL, started_at TEXT NOT NULL, ended_at TEXT NOT NULL,
	  duration_ms INTEGER NOT NULL, tokens_used INTEGER NOT NULL, cost_usd_micro INTEGER NOT NULL,
	  verdict INTEGER NOT NULL, diff_hash TEXT NOT NULL DEFAULT '', diff_path_count INTEGER NOT NULL DEFAULT 0,
	  agent_manager_run_id TEXT NOT NULL DEFAULT '', manifest_template_version_at_run TEXT NOT NULL DEFAULT '',
	  manifest_skill_version_at_run TEXT NOT NULL DEFAULT '', error_message TEXT NOT NULL DEFAULT '')`
	_, err := d.ExecContext(ctx, legacy)
	require.NoError(t, err)
	_, err = d.ExecContext(ctx, `INSERT INTO validation_records
	  (id, tuple_kind, subject_id, golden_slug, started_at, ended_at, duration_ms, tokens_used, cost_usd_micro, verdict)
	  VALUES ('legacy', 1, 's', 'g', '2026-05-29T00:00:00Z', '2026-05-29T00:00:01Z', 1000, 0, 0, 1)`)
	require.NoError(t, err)

	// First migration adds the columns; existing row survives.
	require.NoError(t, vr.EnsureColumns(ctx, d))
	// Re-applying is a no-op (no "duplicate column" error).
	require.NoError(t, vr.EnsureColumns(ctx, d))

	repo := vr.NewSQLiteRepository(d)
	got, err := repo.Get(ctx, "legacy")
	require.NoError(t, err)
	require.Equal(t, "", got.ToolDetail)
	require.Equal(t, "", got.ToolRawOutput)

	// New rows can write the new columns.
	now := time.Date(2026, 5, 29, 12, 0, 0, 0, time.UTC)
	r := sampleRecord("new", "g", "test-genie", now, vr.TupleKindTool, vr.VerdictPass)
	r.ToolDetail = "all 14 phase(s) passed"
	require.NoError(t, repo.Append(ctx, r))
	got, err = repo.Get(ctx, "new")
	require.NoError(t, err)
	require.Equal(t, "all 14 phase(s) passed", got.ToolDetail)
}

func TestGet_NotFoundReturnsSentinel(t *testing.T) {
	repo := newRepo(t)
	_, err := repo.Get(context.Background(), "missing")
	var nf vr.ErrRecordNotFound
	require.True(t, errors.As(err, &nf))
}

func TestList_FilterAndOrder(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()
	base := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	require.NoError(t, repo.Append(ctx, sampleRecord("oldest", "g1", "s1", base.Add(-2*time.Hour), vr.TupleKindSkill, vr.VerdictPass)))
	require.NoError(t, repo.Append(ctx, sampleRecord("middle", "g1", "s1", base.Add(-1*time.Hour), vr.TupleKindSkill, vr.VerdictPass)))
	require.NoError(t, repo.Append(ctx, sampleRecord("newest", "g2", "s2", base, vr.TupleKindSkill, vr.VerdictPass)))

	res, err := repo.List(ctx, vr.ListFilter{}, 0, "")
	require.NoError(t, err)
	require.Len(t, res.Records, 3)
	require.Equal(t, "newest", res.Records[0].ID)
	require.Equal(t, "oldest", res.Records[2].ID)

	res, err = repo.List(ctx, vr.ListFilter{GoldenSlug: "g1"}, 0, "")
	require.NoError(t, err)
	require.Len(t, res.Records, 2)
}

func TestList_Paginates(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()
	base := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 5; i++ {
		id := "r" + string(rune('1'+i))
		require.NoError(t, repo.Append(ctx, sampleRecord(id, "g", "s", base.Add(time.Duration(i)*time.Minute), vr.TupleKindSkill, vr.VerdictPass)))
	}
	first, err := repo.List(ctx, vr.ListFilter{}, 2, "")
	require.NoError(t, err)
	require.Len(t, first.Records, 2)
	require.NotEmpty(t, first.NextPageToken)

	second, err := repo.List(ctx, vr.ListFilter{}, 2, first.NextPageToken)
	require.NoError(t, err)
	require.Len(t, second.Records, 2)
	require.NotEqual(t, first.Records[0].ID, second.Records[0].ID)

	third, err := repo.List(ctx, vr.ListFilter{}, 2, second.NextPageToken)
	require.NoError(t, err)
	require.Len(t, third.Records, 1)
	require.Empty(t, third.NextPageToken)
}

func TestList_RejectsBadCursor(t *testing.T) {
	repo := newRepo(t)
	_, err := repo.List(context.Background(), vr.ListFilter{}, 0, "!!!notbase64!!!")
	var invalid vr.ErrInvalidRecord
	require.True(t, errors.As(err, &invalid))
}
