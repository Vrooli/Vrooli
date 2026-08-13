package themes_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	db "github.com/vrooli/api-core/databasetest"
	"react-component-library/internal/themes"
)

func newSQLiteThemeRepo(t *testing.T) themes.Repository {
	t.Helper()
	sqlDB := db.NewSQLite(t)
	_, err := sqlDB.ExecContext(context.Background(), themes.Schema())
	require.NoError(t, err)
	return themes.NewSQLiteRepository(sqlDB)
}

func TestSQLite_UpsertAndGetBuiltin(t *testing.T) {
	repo := newSQLiteThemeRepo(t)
	ctx := context.Background()
	require.NoError(t, repo.UpsertBuiltin(ctx, themes.Theme{
		ID: "t-1", Name: "T1", Tokens: map[string]string{"--color-primary": "#fff"}, Source: "builtin",
	}))
	got, err := repo.GetBuiltin(ctx, "t-1")
	require.NoError(t, err)
	require.Equal(t, "T1", got.Name)
	require.Equal(t, "#fff", got.Tokens["--color-primary"])
	require.Equal(t, "builtin", got.Source)
}

func TestSQLite_UpsertReplaces(t *testing.T) {
	repo := newSQLiteThemeRepo(t)
	ctx := context.Background()
	require.NoError(t, repo.UpsertBuiltin(ctx, themes.Theme{ID: "t-1", Name: "v1", Tokens: map[string]string{"--a": "1"}}))
	require.NoError(t, repo.UpsertBuiltin(ctx, themes.Theme{ID: "t-1", Name: "v2", Tokens: map[string]string{"--a": "2"}}))
	got, err := repo.GetBuiltin(ctx, "t-1")
	require.NoError(t, err)
	require.Equal(t, "v2", got.Name)
	require.Equal(t, "2", got.Tokens["--a"])
}

func TestSQLite_ListAndCount(t *testing.T) {
	repo := newSQLiteThemeRepo(t)
	ctx := context.Background()
	n, err := repo.CountBuiltins(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, n)

	require.NoError(t, repo.UpsertBuiltin(ctx, themes.Theme{ID: "b", Name: "B"}))
	require.NoError(t, repo.UpsertBuiltin(ctx, themes.Theme{ID: "a", Name: "A"}))
	list, err := repo.ListBuiltins(ctx)
	require.NoError(t, err)
	require.Len(t, list, 2)
	require.Equal(t, "a", list[0].ID)
	require.Equal(t, "b", list[1].ID)
}

func TestSQLite_GetNotFound(t *testing.T) {
	repo := newSQLiteThemeRepo(t)
	_, err := repo.GetBuiltin(context.Background(), "nope")
	require.Error(t, err)
	var sentinel themes.ErrThemeNotFound
	require.ErrorAs(t, err, &sentinel)
}
