package skill_catalog_test

import (
	"context"
	"errors"
	"testing"
	"time"

	skillcatalog "development-toolchain-validator/internal/skill_catalog"
	"development-toolchain-validator/internal/testutil/db"
	"development-toolchain-validator/internal/testutil/mocks"

	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"

	localdb "development-toolchain-validator/internal/database"
)

func newRepo(t *testing.T) (skillcatalog.Repository, *mocks.FakeClock) {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(skillcatalog.Schema),
	))
	clk := mocks.NewFakeClock(time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC))
	return skillcatalog.NewSQLiteRepository(d, clk), clk
}

func sampleSkill(id string) skillcatalog.Skill {
	return skillcatalog.Skill{
		ID:          id,
		Version:     "2026-05-01T00:00:00Z",
		ContentHash: "abc123",
	}
}

func TestUpsert_InsertsNewRow(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()

	inserted, changed, err := repo.Upsert(ctx, sampleSkill("plan-skill-discovery"))
	require.NoError(t, err)
	require.True(t, inserted, "first upsert must report insert")
	require.True(t, changed, "first upsert must report changed")
}

func TestUpsert_UpdatesExistingRow(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	_, _, err := repo.Upsert(ctx, sampleSkill("plan-skill-discovery"))
	require.NoError(t, err)

	updated := sampleSkill("plan-skill-discovery")
	updated.ContentHash = "def456"
	inserted, changed, err := repo.Upsert(ctx, updated)
	require.NoError(t, err)
	require.False(t, inserted, "second upsert must not report insert")
	require.True(t, changed, "content_hash change must be reported as changed")
}

func TestUpsert_IdempotentReturnsUnchanged(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	_, _, err := repo.Upsert(ctx, sampleSkill("plan-skill-discovery"))
	require.NoError(t, err)

	inserted, changed, err := repo.Upsert(ctx, sampleSkill("plan-skill-discovery"))
	require.NoError(t, err)
	require.False(t, inserted)
	require.False(t, changed, "repeat upsert with same fields must report unchanged")
}

func TestGet_NotFoundReturnsSentinel(t *testing.T) {
	repo, _ := newRepo(t)
	_, err := repo.Get(context.Background(), "missing")
	require.Error(t, err)
	var notFound skillcatalog.ErrSkillNotFound
	require.True(t, errors.As(err, &notFound), "expected ErrSkillNotFound, got %T: %v", err, err)
	require.Equal(t, "missing", notFound.ID)
}

func TestList_OrderedByID(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	_, _, err := repo.Upsert(ctx, sampleSkill("zoo"))
	require.NoError(t, err)
	_, _, err = repo.Upsert(ctx, sampleSkill("alpha"))
	require.NoError(t, err)
	_, _, err = repo.Upsert(ctx, sampleSkill("middle"))
	require.NoError(t, err)

	got, err := repo.List(ctx)
	require.NoError(t, err)
	require.Len(t, got, 3)
	require.Equal(t, "alpha", got[0].ID)
	require.Equal(t, "middle", got[1].ID)
	require.Equal(t, "zoo", got[2].ID)
}

func TestDeleteMissing_RemovesUnreferencedRows(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	for _, id := range []string{"keep1", "keep2", "drop1", "drop2"} {
		_, _, err := repo.Upsert(ctx, sampleSkill(id))
		require.NoError(t, err)
	}

	removed, err := repo.DeleteMissing(ctx, []string{"keep1", "keep2"})
	require.NoError(t, err)
	require.Equal(t, 2, removed)

	after, err := repo.List(ctx)
	require.NoError(t, err)
	require.Len(t, after, 2)
}

func TestDeleteMissing_EmptyKeepDeletesAll(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	_, _, err := repo.Upsert(ctx, sampleSkill("a"))
	require.NoError(t, err)
	_, _, err = repo.Upsert(ctx, sampleSkill("b"))
	require.NoError(t, err)

	removed, err := repo.DeleteMissing(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, 2, removed)
}
