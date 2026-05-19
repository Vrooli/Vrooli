package validation_record_test

import (
	"context"
	"errors"
	"testing"
	"time"

	vr "development-toolchain-validator/internal/validation_record"

	"development-toolchain-validator/internal/testutil/db"

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
