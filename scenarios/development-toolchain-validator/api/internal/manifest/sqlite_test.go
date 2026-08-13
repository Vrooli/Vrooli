package manifest_test

import (
	"context"
	"errors"
	"testing"
	"time"

	manifest "development-toolchain-validator/internal/manifest"
	"development-toolchain-validator/internal/testutil/db"

	"github.com/vrooli/api-core/scheduletest"

	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"

	localdb "development-toolchain-validator/internal/database"
)

func newRepo(t *testing.T) (manifest.Repository, *scheduletest.FakeClock) {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(manifest.Schema),
	))
	clk := scheduletest.New(time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC))
	return manifest.NewSQLiteRepository(d, clk), clk
}

func sample(skillID, goldenSlug string) manifest.Manifest {
	return manifest.Manifest{
		SkillID:               skillID,
		GoldenSlug:            goldenSlug,
		AllowedPaths:          []string{"src/**", "docs/**"},
		ContentRules:          []manifest.ContentRule{{PathGlob: "**/*.go", MustContain: []string{"package main"}}},
		WildcardAllowed:       false,
		ConvergenceTarget:     manifest.ConvergenceTargetEmptyDiff,
		TemplateVersionPinned: "1.0.0",
		SkillVersionPinned:    "2026-05-01T00:00:00Z",
	}
}

func TestUpsert_InsertsAndRoundTrips(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	got, err := repo.Upsert(ctx, sample("implementation-plan-authoring", "reference-react-vite"))
	require.NoError(t, err)
	require.Equal(t, "implementation-plan-authoring", got.SkillID)
	require.Equal(t, "reference-react-vite", got.GoldenSlug)
	require.Equal(t, []string{"src/**", "docs/**"}, got.AllowedPaths)
	require.Len(t, got.ContentRules, 1)
	require.Equal(t, "**/*.go", got.ContentRules[0].PathGlob)
	require.Equal(t, manifest.ConvergenceTargetEmptyDiff, got.ConvergenceTarget)
	require.False(t, got.UpdatedAt.IsZero())
}

func TestUpsert_ReplacesExisting(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	_, err := repo.Upsert(ctx, sample("implementation-plan-authoring", "reference-react-vite"))
	require.NoError(t, err)

	updated := sample("implementation-plan-authoring", "reference-react-vite")
	updated.WildcardAllowed = true
	updated.AllowedPaths = nil
	updated.ContentRules = nil
	got, err := repo.Upsert(ctx, updated)
	require.NoError(t, err)
	require.True(t, got.WildcardAllowed)
	require.Empty(t, got.AllowedPaths)
	require.Empty(t, got.ContentRules)
}

func TestGet_NotFoundReturnsSentinel(t *testing.T) {
	repo, _ := newRepo(t)
	_, err := repo.Get(context.Background(), "missing", "also-missing")
	require.Error(t, err)
	var notFound manifest.ErrManifestNotFound
	require.True(t, errors.As(err, &notFound), "expected ErrManifestNotFound, got %T: %v", err, err)
}

func TestList_OrderedBySkillThenGolden(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	for _, tup := range [][2]string{
		{"z-skill", "a-golden"},
		{"a-skill", "z-golden"},
		{"a-skill", "a-golden"},
	} {
		_, err := repo.Upsert(ctx, sample(tup[0], tup[1]))
		require.NoError(t, err)
	}
	got, err := repo.List(ctx)
	require.NoError(t, err)
	require.Len(t, got, 3)
	require.Equal(t, "a-skill", got[0].SkillID)
	require.Equal(t, "a-golden", got[0].GoldenSlug)
	require.Equal(t, "a-skill", got[1].SkillID)
	require.Equal(t, "z-golden", got[1].GoldenSlug)
	require.Equal(t, "z-skill", got[2].SkillID)
}

func TestClearStaleOverride_RoundTrips(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	now := time.Date(2026, 5, 18, 13, 0, 0, 0, time.UTC)
	require.NoError(t, repo.ClearStaleOverride(ctx, "s", "g", now))
	got, err := repo.GetStaleOverride(ctx, "s", "g")
	require.NoError(t, err)
	require.WithinDuration(t, now, got, time.Millisecond)
}

func TestGetStaleOverride_AbsentReturnsZero(t *testing.T) {
	repo, _ := newRepo(t)
	got, err := repo.GetStaleOverride(context.Background(), "nope", "nope")
	require.NoError(t, err)
	require.True(t, got.IsZero())
}

func TestClearStaleOverride_OverwritesExisting(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	first := time.Date(2026, 5, 18, 1, 0, 0, 0, time.UTC)
	second := time.Date(2026, 5, 18, 2, 0, 0, 0, time.UTC)
	require.NoError(t, repo.ClearStaleOverride(ctx, "s", "g", first))
	require.NoError(t, repo.ClearStaleOverride(ctx, "s", "g", second))
	got, err := repo.GetStaleOverride(ctx, "s", "g")
	require.NoError(t, err)
	require.WithinDuration(t, second, got, time.Millisecond)
}
