package golden_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"development-toolchain-validator/internal/golden"
	db "github.com/vrooli/api-core/databasetest"

	"github.com/vrooli/api-core/scheduletest"

	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"

	localdb "development-toolchain-validator/internal/database"
)

func newRepo(t *testing.T) (golden.Repository, *scheduletest.FakeClock) {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(golden.Schema),
	))
	clk := scheduletest.New(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC))
	return golden.NewSQLiteRepository(d, clk), clk
}

func sampleGolden(slug string) golden.Golden {
	return golden.Golden{
		Slug:                  slug,
		TemplateID:            "react-vite",
		TemplateVersionPinned: "1.0.1",
		Path:                  ".vrooli/generated-goldens/" + slug,
	}
}

func TestSQLiteRepository_CreateAndGetRoundTrip(t *testing.T) {
	repo, clk := newRepo(t)
	ctx := context.Background()

	created, err := repo.Create(ctx, sampleGolden("reference-react-vite"))
	require.NoError(t, err)
	require.NotEmpty(t, created.ID)
	require.Equal(t, clk.Now(), created.CreatedAt)
	require.Equal(t, created.CreatedAt, created.LastRegeneratedAt)

	got, err := repo.Get(ctx, "reference-react-vite")
	require.NoError(t, err)
	require.Equal(t, created.ID, got.ID)
	require.Equal(t, "react-vite", got.TemplateID)
	require.Equal(t, "1.0.1", got.TemplateVersionPinned)
	require.Equal(t, ".vrooli/generated-goldens/reference-react-vite", got.Path)
	require.Equal(t, ".vrooli/generated-goldens/reference-react-vite", got.LogicalRoot)
	require.Equal(t, golden.MaterializationModeEphemeral, got.MaterializationMode)
	require.Equal(t, golden.MaterializationStatusNever, got.LastMaterializedStatus)
	require.True(t, created.CreatedAt.Equal(got.CreatedAt))
}

func TestSQLiteRepository_EnsureColumnsBackfillsLegacyRows(t *testing.T) {
	d := db.NewSQLite(t)
	ctx := context.Background()
	_, err := d.ExecContext(ctx, `
CREATE TABLE goldens (
  id TEXT PRIMARY KEY,
  slug TEXT NOT NULL UNIQUE,
  template_id TEXT NOT NULL,
  template_version_pinned TEXT NOT NULL,
  path TEXT NOT NULL,
  created_at TEXT NOT NULL,
  last_regenerated_at TEXT NOT NULL
)`)
	require.NoError(t, err)
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC).Format(time.RFC3339Nano)
	_, err = d.ExecContext(ctx, `INSERT INTO goldens (id, slug, template_id, template_version_pinned, path, created_at, last_regenerated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"id-1", "reference-react-vite", "react-vite", "1.0.1", ".vrooli/generated-goldens/reference-react-vite", now, now)
	require.NoError(t, err)
	require.NoError(t, golden.EnsureColumns(ctx, d))
	require.NoError(t, golden.EnsureColumns(ctx, d))

	repo := golden.NewSQLiteRepository(d, scheduletest.New(time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)))
	got, err := repo.Get(ctx, "reference-react-vite")
	require.NoError(t, err)
	require.Equal(t, ".vrooli/generated-goldens/reference-react-vite", got.Path)
	require.Equal(t, ".vrooli/generated-goldens/reference-react-vite", got.LogicalRoot)
	require.Equal(t, golden.MaterializationModeEphemeral, got.MaterializationMode)
	require.Equal(t, golden.MaterializationStatusNever, got.LastMaterializedStatus)
}

func TestSQLiteRepository_GetReturnsNotFound(t *testing.T) {
	repo, _ := newRepo(t)
	_, err := repo.Get(context.Background(), "missing")
	require.Error(t, err)
	var nf golden.ErrGoldenNotFound
	require.True(t, errors.As(err, &nf), "expected ErrGoldenNotFound, got %T", err)
	require.Equal(t, "missing", nf.Slug)
}

func TestSQLiteRepository_DuplicateSlugRejected(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	_, err := repo.Create(ctx, sampleGolden("dup"))
	require.NoError(t, err)
	_, err = repo.Create(ctx, sampleGolden("dup"))
	require.Error(t, err)
	var exists golden.ErrGoldenAlreadyExists
	require.True(t, errors.As(err, &exists), "expected ErrGoldenAlreadyExists, got %T: %v", err, err)
	require.Equal(t, "dup", exists.Slug)
}

func TestSQLiteRepository_ListSortedBySlug(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	for _, slug := range []string{"charlie", "alpha", "bravo"} {
		_, err := repo.Create(ctx, sampleGolden(slug))
		require.NoError(t, err)
	}
	got, err := repo.List(ctx)
	require.NoError(t, err)
	require.Len(t, got, 3)
	require.Equal(t, "alpha", got[0].Slug)
	require.Equal(t, "bravo", got[1].Slug)
	require.Equal(t, "charlie", got[2].Slug)
}

func TestSQLiteRepository_UpdateChangesFieldsAndPersistsLastRegeneratedAt(t *testing.T) {
	repo, clk := newRepo(t)
	ctx := context.Background()
	created, err := repo.Create(ctx, sampleGolden("g"))
	require.NoError(t, err)

	clk.Advance(time.Hour)
	created.Path = "scenarios/new-path"
	created.TemplateVersionPinned = "1.0.2"
	created.LastRegeneratedAt = time.Time{} // exercise the clock-fill branch

	updated, err := repo.Update(ctx, created)
	require.NoError(t, err)
	require.Equal(t, "scenarios/new-path", updated.Path)
	require.Equal(t, "1.0.2", updated.TemplateVersionPinned)
	require.Equal(t, clk.Now(), updated.LastRegeneratedAt)

	got, err := repo.Get(ctx, "g")
	require.NoError(t, err)
	require.Equal(t, "scenarios/new-path", got.Path)
	require.Equal(t, "1.0.2", got.TemplateVersionPinned)
}

func TestSQLiteRepository_UpdateMissingReturnsNotFound(t *testing.T) {
	repo, _ := newRepo(t)
	_, err := repo.Update(context.Background(), golden.Golden{Slug: "ghost", Path: "x", TemplateVersionPinned: "1.0.0"})
	require.Error(t, err)
	var nf golden.ErrGoldenNotFound
	require.True(t, errors.As(err, &nf))
}

func TestSQLiteRepository_DeleteRemovesRow(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	_, err := repo.Create(ctx, sampleGolden("doomed"))
	require.NoError(t, err)
	require.NoError(t, repo.Delete(ctx, "doomed"))

	_, err = repo.Get(ctx, "doomed")
	var nf golden.ErrGoldenNotFound
	require.True(t, errors.As(err, &nf))
}

func TestSQLiteRepository_DeleteMissingReturnsNotFound(t *testing.T) {
	repo, _ := newRepo(t)
	err := repo.Delete(context.Background(), "ghost")
	var nf golden.ErrGoldenNotFound
	require.True(t, errors.As(err, &nf), "expected ErrGoldenNotFound, got %T", err)
}
