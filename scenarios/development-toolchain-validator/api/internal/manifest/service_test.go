package manifest_test

import (
	"context"
	"errors"
	"testing"
	"time"

	manifest "development-toolchain-validator/internal/manifest"
	mmocks "development-toolchain-validator/internal/manifest/mocks"
	"development-toolchain-validator/internal/testutil/mocks"

	"github.com/stretchr/testify/require"
)

func newSvc(t *testing.T) (manifest.Service, *mmocks.FakeRepository, *mocks.FakeClock) {
	t.Helper()
	repo := mmocks.NewFakeRepository()
	clk := mocks.NewFakeClock(time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC))
	return manifest.NewService(repo, clk), repo, clk
}

func TestUpsert_HappyPath(t *testing.T) {
	svc, _, clk := newSvc(t)
	got, err := svc.Upsert(context.Background(), manifest.UpsertInput{
		SkillID:         "plan-skill-discovery",
		GoldenSlug:      "reference-react-vite",
		WildcardAllowed: true,
	})
	require.NoError(t, err)
	require.Equal(t, "plan-skill-discovery", got.SkillID)
	require.Equal(t, clk.Now(), got.UpdatedAt)
	require.Equal(t, manifest.ConvergenceTargetNone, got.ConvergenceTarget, "unspecified must default to none")
}

func TestUpsert_RejectsBadSkillID(t *testing.T) {
	svc, _, _ := newSvc(t)
	_, err := svc.Upsert(context.Background(), manifest.UpsertInput{
		SkillID:    "  ",
		GoldenSlug: "reference-react-vite",
	})
	var invalid manifest.ErrInvalidManifest
	require.True(t, errors.As(err, &invalid))
	require.Equal(t, "skill_id", invalid.Field)
}

func TestUpsert_RejectsBadGoldenSlug(t *testing.T) {
	svc, _, _ := newSvc(t)
	_, err := svc.Upsert(context.Background(), manifest.UpsertInput{
		SkillID:    "plan-skill-discovery",
		GoldenSlug: "UPPERCASE!",
	})
	var invalid manifest.ErrInvalidManifest
	require.True(t, errors.As(err, &invalid))
	require.Equal(t, "golden_slug", invalid.Field)
}

func TestUpsert_RequiresAllowedOrWildcard(t *testing.T) {
	svc, _, _ := newSvc(t)
	_, err := svc.Upsert(context.Background(), manifest.UpsertInput{
		SkillID:    "plan-skill-discovery",
		GoldenSlug: "reference-react-vite",
		// no wildcard, no allowed_paths, no content_rules
	})
	var invalid manifest.ErrInvalidManifest
	require.True(t, errors.As(err, &invalid))
}

func TestUpsert_RejectsBlankAllowedEntry(t *testing.T) {
	svc, _, _ := newSvc(t)
	_, err := svc.Upsert(context.Background(), manifest.UpsertInput{
		SkillID:      "plan-skill-discovery",
		GoldenSlug:   "reference-react-vite",
		AllowedPaths: []string{"src/**", "  "},
	})
	var invalid manifest.ErrInvalidManifest
	require.True(t, errors.As(err, &invalid))
}

func TestUpsert_RejectsBlankContentRuleGlob(t *testing.T) {
	svc, _, _ := newSvc(t)
	_, err := svc.Upsert(context.Background(), manifest.UpsertInput{
		SkillID:      "plan-skill-discovery",
		GoldenSlug:   "reference-react-vite",
		AllowedPaths: []string{"src/**"},
		ContentRules: []manifest.ContentRule{{PathGlob: " "}},
	})
	var invalid manifest.ErrInvalidManifest
	require.True(t, errors.As(err, &invalid))
}

func TestGet_NotFound(t *testing.T) {
	svc, _, _ := newSvc(t)
	_, err := svc.Get(context.Background(), "missing", "missing")
	var notFound manifest.ErrManifestNotFound
	require.True(t, errors.As(err, &notFound))
}

func TestGet_EmptyArgsRejected(t *testing.T) {
	svc, _, _ := newSvc(t)
	_, err := svc.Get(context.Background(), "", "g")
	var invalid manifest.ErrInvalidManifest
	require.True(t, errors.As(err, &invalid))
	_, err = svc.Get(context.Background(), "s", "")
	require.True(t, errors.As(err, &invalid))
}

func TestClearStale_RecordsTimestamp(t *testing.T) {
	svc, repo, clk := newSvc(t)
	at, err := svc.ClearStale(context.Background(), "plan-skill-discovery", "reference-react-vite")
	require.NoError(t, err)
	require.Equal(t, clk.Now(), at)
	stored, err := repo.GetStaleOverride(context.Background(), "plan-skill-discovery", "reference-react-vite")
	require.NoError(t, err)
	require.Equal(t, clk.Now(), stored)
}

func TestClearStale_RejectsEmpty(t *testing.T) {
	svc, _, _ := newSvc(t)
	_, err := svc.ClearStale(context.Background(), "", "g")
	var invalid manifest.ErrInvalidManifest
	require.True(t, errors.As(err, &invalid))
}

func TestList_PassThrough(t *testing.T) {
	svc, _, _ := newSvc(t)
	_, err := svc.Upsert(context.Background(), manifest.UpsertInput{
		SkillID: "a", GoldenSlug: "g", WildcardAllowed: true,
	})
	require.NoError(t, err)
	got, err := svc.List(context.Background())
	require.NoError(t, err)
	require.Len(t, got, 1)
}
