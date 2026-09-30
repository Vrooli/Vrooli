package versions_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"

	"react-component-library/internal/components"
	localdb "react-component-library/internal/database"
	"react-component-library/internal/versions"

	db "github.com/vrooli/api-core/databasetest"

	"github.com/vrooli/api-core/scheduletest"
)

func newVersionsDB(t *testing.T) (versions.Repository, *scheduletest.FakeClock) {
	t.Helper()
	d := db.NewSQLite(t)
	// component_versions is owned by the components domain schema; the
	// versions domain is a read/append consumer of that table.
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(components.Schema),
	))
	clk := scheduletest.New(time.Date(2026, 5, 13, 10, 0, 0, 0, time.UTC))
	return versions.NewSQLiteRepository(d, clk), clk
}

func TestSQLiteRepository_InsertGetList(t *testing.T) {
	repo, clk := newVersionsDB(t)
	ctx := context.Background()

	first, err := repo.Insert(ctx, versions.Version{
		ComponentID:   "cmp-1",
		LibraryID:     "react-component-library:Button",
		Version:       "1.0.0",
		Content:       "body v1",
		ContentSHA256: "abc",
		ChangelogMD:   "first",
	})
	require.NoError(t, err)
	require.NotEmpty(t, first.ID)
	require.False(t, first.RecordedAt.IsZero())

	clk.Advance(2 * time.Second)
	second, err := repo.Insert(ctx, versions.Version{
		ComponentID:   "cmp-1",
		Version:       "1.0.1",
		Content:       "body v2",
		ContentSHA256: "def",
	})
	require.NoError(t, err)
	require.True(t, second.RecordedAt.After(first.RecordedAt))

	got, err := repo.Get(ctx, "cmp-1", "1.0.0")
	require.NoError(t, err)
	require.Equal(t, "body v1", got.Content)
	require.Equal(t, "first", got.ChangelogMD)

	got, err = repo.Get(ctx, "react-component-library:Button", "1.0.0")
	require.NoError(t, err)
	require.Equal(t, "cmp-1", got.ComponentID)

	rows, err := repo.List(ctx, versions.ListQuery{ComponentID: "cmp-1", Limit: 10})
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.Equal(t, "1.0.1", rows[0].Version) // newest first

	rows, err = repo.List(ctx, versions.ListQuery{ComponentID: "react-component-library:Button", Limit: 10})
	require.NoError(t, err)
	require.Len(t, rows, 1, "library-id lookup should select only versions carrying that library id")
}

func TestSQLiteRepository_Latest_EmptyAndPopulated(t *testing.T) {
	repo, _ := newVersionsDB(t)
	ctx := context.Background()

	got, err := repo.Latest(ctx, "cmp-none")
	require.NoError(t, err)
	require.Empty(t, got.ID, "Latest of empty must return zero Version")

	_, err = repo.Insert(ctx, versions.Version{
		ComponentID: "cmp-1", Version: "1.0.0", Content: "x", ContentSHA256: "h",
	})
	require.NoError(t, err)

	got, err = repo.Latest(ctx, "cmp-1")
	require.NoError(t, err)
	require.Equal(t, "1.0.0", got.Version)
}

func TestSQLiteRepository_GetMissing(t *testing.T) {
	repo, _ := newVersionsDB(t)
	_, err := repo.Get(context.Background(), "cmp-1", "1.0.0")
	var nf versions.ErrVersionNotFound
	require.True(t, errors.As(err, &nf), "expected ErrVersionNotFound, got %v", err)
}

func TestSQLiteRepository_List_RespectsLimitAndComponent(t *testing.T) {
	repo, clk := newVersionsDB(t)
	ctx := context.Background()

	for i, cid := range []string{"cmp-1", "cmp-2", "cmp-1", "cmp-1"} {
		clk.Advance(time.Second)
		_, err := repo.Insert(ctx, versions.Version{
			ComponentID:   cid,
			Version:       string(rune('a' + i)),
			Content:       "x",
			ContentSHA256: "h",
		})
		require.NoError(t, err)
	}
	rows, err := repo.List(ctx, versions.ListQuery{ComponentID: "cmp-1", Limit: 2})
	require.NoError(t, err)
	require.Len(t, rows, 2, "limit must cap rows")
	for _, r := range rows {
		require.Equal(t, "cmp-1", r.ComponentID)
	}
}

func TestServiceRecordReturnsTypedCollisionForContentChangeWithoutVersionBump(t *testing.T) {
	repo, _ := newVersionsDB(t)
	svc := versions.NewService(repo, nil)
	ctx := context.Background()
	_, _, err := svc.Record(ctx, versions.RecordInput{ComponentID: "cmp-1", Content: "// @version 1.0.0\nfirst"})
	require.NoError(t, err)

	_, _, err = svc.Record(ctx, versions.RecordInput{ComponentID: "cmp-1", Content: "// @version 1.0.0\nsecond"})
	var collision versions.ErrVersionExists
	require.ErrorAs(t, err, &collision)
	require.Equal(t, "1.0.0", collision.Version)
}
