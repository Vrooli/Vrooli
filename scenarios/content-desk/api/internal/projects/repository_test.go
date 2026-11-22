package projects_test

import (
	"context"
	"testing"

	"content-desk/internal/projects"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	db "github.com/vrooli/api-core/databasetest"
)

func projectRepo(t *testing.T) projects.Repository {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), d, database.SchemaProviderFunc(projects.Schema)))
	return projects.NewSQLiteRepository(d)
}

func TestCreateMakesStableActiveProjectAndListReturnsIt(t *testing.T) {
	repo := projectRepo(t)
	p, err := repo.Create(context.Background(), "  Release notes  ")
	require.NoError(t, err)
	require.NotEmpty(t, p.ID)
	require.NotEqual(t, projects.UnassignedID, p.ID)
	require.Equal(t, "Release notes", p.Name)
	require.Equal(t, projects.StatusActive, p.Status)

	items, err := repo.ListActive(context.Background())
	require.NoError(t, err)
	require.Equal(t, []projects.Project{p}, items)
}

func TestUnassignedIsReservedAndCannotBeCreated(t *testing.T) {
	repo := projectRepo(t)
	for _, name := range []string{"", "  ", "unassigned", "UNASSIGNED"} {
		_, err := repo.Create(context.Background(), name)
		require.ErrorIs(t, err, projects.ErrInvalidName)
	}
	items, err := repo.ListActive(context.Background())
	require.NoError(t, err)
	require.Empty(t, items)
}
