package corpus_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"

	"audio-tools/internal/corpus"
	localdb "audio-tools/internal/database"
	db "github.com/vrooli/api-core/databasetest"

	"github.com/vrooli/api-core/scheduletest"
)

func newSchemaDB(t *testing.T) *sql.DB {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(corpus.Schema),
	))
	return d
}

func newRepo(t *testing.T) (corpus.Repository, *scheduletest.FakeClock) {
	t.Helper()
	clk := scheduletest.New(time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC))
	return corpus.NewSQLiteRepository(newSchemaDB(t), clk), clk
}

func TestRepository_CreateGetList(t *testing.T) {
	ctx := context.Background()
	repo, _ := newRepo(t)

	saved, err := repo.Create(ctx, corpus.Clip{
		ReferenceText: "the quick brown fox",
		Tags:          []string{"news", "clean"},
		DurationMs:    1500,
		SampleRateHz:  16000,
		Format:        "pcm_s16le",
		BlobKey:       "clips/2026-06/abc.pcm",
		Source:        corpus.SourceScripted,
	})
	require.NoError(t, err)
	require.NotEmpty(t, saved.ID, "id is generated")
	require.False(t, saved.CreatedAt.IsZero(), "created_at is stamped from the clock")

	got, err := repo.Get(ctx, saved.ID)
	require.NoError(t, err)
	require.Equal(t, "the quick brown fox", got.ReferenceText)
	require.Equal(t, []string{"news", "clean"}, got.Tags)
	require.Equal(t, int64(1500), got.DurationMs)
	require.Equal(t, 16000, got.SampleRateHz)
	require.Equal(t, "pcm_s16le", got.Format)
	require.Equal(t, corpus.SourceScripted, got.Source)

	list, err := repo.List(ctx, corpus.ListFilter{})
	require.NoError(t, err)
	require.Len(t, list, 1)
}

func TestRepository_ListTagFilterAndOrder(t *testing.T) {
	ctx := context.Background()
	repo, clk := newRepo(t)

	mk := func(ref, tag string) {
		_, err := repo.Create(ctx, corpus.Clip{ReferenceText: ref, Tags: []string{tag}, BlobKey: "clips/k/" + ref + ".pcm"})
		require.NoError(t, err)
		clk.Advance(time.Second) // distinct created_at so ordering is stable
	}
	mk("first", "alpha")
	mk("second", "beta")
	mk("third", "alpha")

	all, err := repo.List(ctx, corpus.ListFilter{})
	require.NoError(t, err)
	require.Len(t, all, 3)
	require.Equal(t, "third", all[0].ReferenceText, "newest first (created_at DESC)")

	alpha, err := repo.List(ctx, corpus.ListFilter{TagContains: "alpha"})
	require.NoError(t, err)
	require.Len(t, alpha, 2, "tag filter narrows to matching clips")
}

func TestRepository_DeleteAndNotFound(t *testing.T) {
	ctx := context.Background()
	repo, _ := newRepo(t)

	_, err := repo.Get(ctx, "missing")
	require.ErrorAs(t, err, &corpus.ErrClipNotFound{})

	saved, err := repo.Create(ctx, corpus.Clip{ReferenceText: "x", BlobKey: "clips/k/x.pcm"})
	require.NoError(t, err)
	require.NoError(t, repo.Delete(ctx, saved.ID))
	require.ErrorAs(t, repo.Delete(ctx, saved.ID), &corpus.ErrClipNotFound{}, "second delete reports not-found")
}

func TestRepository_CreateRequiresBlobKey(t *testing.T) {
	repo, _ := newRepo(t)
	_, err := repo.Create(context.Background(), corpus.Clip{ReferenceText: "no blob"})
	require.Error(t, err, "a clip without a blob key is rejected")
}
