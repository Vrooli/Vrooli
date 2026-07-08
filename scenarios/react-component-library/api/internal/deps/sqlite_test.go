package deps_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"react-component-library/internal/deps"
	"react-component-library/internal/testutil/db"
)

func newSQLiteRepo(t *testing.T) deps.Repository {
	t.Helper()
	sqlDB := db.NewSQLite(t)
	_, err := sqlDB.ExecContext(context.Background(), deps.Schema())
	require.NoError(t, err)
	return deps.NewSQLiteRepository(sqlDB)
}

func TestSQLite_SyncAndList(t *testing.T) {
	repo := newSQLiteRepo(t)
	ctx := context.Background()
	require.NoError(t, repo.SyncForComponent(ctx, deps.SyncInput{
		ComponentID: "cmp-1",
		LibraryID:   "rcl:Button",
		Declarations: []deps.DeclarationFields{
			{Version: "1.0.0", DepName: "react", VersionRange: "^18.0.0"},
			{Version: "1.0.0", DepName: "lodash", VersionRange: "^4.17.0", Kind: deps.DepKindPeer},
		},
	}))
	got, err := repo.ListForComponent(ctx, "cmp-1")
	require.NoError(t, err)
	require.Len(t, got, 2)
	// Sorted by name.
	require.Equal(t, "lodash", got[0].DepName)
	require.Equal(t, "react", got[1].DepName)
	require.Equal(t, "1.0.0", got[0].Version)
	require.Equal(t, deps.DepKindPeer, got[0].Kind)
}

func TestSQLite_SyncStoresRowsPerVersion(t *testing.T) {
	repo := newSQLiteRepo(t)
	ctx := context.Background()
	require.NoError(t, repo.SyncForComponent(ctx, deps.SyncInput{
		ComponentID: "cmp-1",
		LibraryID:   "rcl:Button",
		Declarations: []deps.DeclarationFields{
			{Version: "1.0.0", DepName: "react", VersionRange: "^17.0.0"},
			{Version: "1.1.0", DepName: "react", VersionRange: "^18.0.0"},
		},
	}))

	got, err := repo.ListForComponentVersion(ctx, "cmp-1", "1.0.0")
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "^17.0.0", got[0].VersionRange)

	got, err = repo.ListForComponentVersion(ctx, "cmp-1", "1.1.0")
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "^18.0.0", got[0].VersionRange)
}

func TestSQLite_SyncReplaces(t *testing.T) {
	repo := newSQLiteRepo(t)
	ctx := context.Background()
	require.NoError(t, repo.SyncForComponent(ctx, deps.SyncInput{
		ComponentID:  "cmp-1",
		Declarations: []deps.DeclarationFields{{DepName: "react", VersionRange: "^17"}},
	}))
	require.NoError(t, repo.SyncForComponent(ctx, deps.SyncInput{
		ComponentID:  "cmp-1",
		Declarations: []deps.DeclarationFields{{DepName: "react", VersionRange: "^18"}, {DepName: "lodash", VersionRange: "*"}},
	}))
	got, err := repo.ListForComponent(ctx, "cmp-1")
	require.NoError(t, err)
	require.Len(t, got, 2)
	for _, d := range got {
		if d.DepName == "react" {
			require.Equal(t, "^18", d.VersionRange)
		}
	}
}

func TestSQLite_SyncEmptyClears(t *testing.T) {
	repo := newSQLiteRepo(t)
	ctx := context.Background()
	require.NoError(t, repo.SyncForComponent(ctx, deps.SyncInput{
		ComponentID:  "cmp-1",
		Declarations: []deps.DeclarationFields{{DepName: "react", VersionRange: "^18"}},
	}))
	require.NoError(t, repo.SyncForComponent(ctx, deps.SyncInput{ComponentID: "cmp-1"}))
	got, err := repo.ListForComponent(ctx, "cmp-1")
	require.NoError(t, err)
	require.Empty(t, got)
}

func TestSQLite_DeleteForComponent(t *testing.T) {
	repo := newSQLiteRepo(t)
	ctx := context.Background()
	require.NoError(t, repo.SyncForComponent(ctx, deps.SyncInput{
		ComponentID:  "cmp-1",
		Declarations: []deps.DeclarationFields{{DepName: "react", VersionRange: "^18"}},
	}))
	require.NoError(t, repo.DeleteForComponent(ctx, "cmp-1"))
	got, err := repo.ListForComponent(ctx, "cmp-1")
	require.NoError(t, err)
	require.Empty(t, got)
}
