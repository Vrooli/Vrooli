package projects_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	domain "device-studio/internal/projects"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	db "github.com/vrooli/api-core/databasetest"
)

func TestRoutedPlaybooksSchemaMatchesRuntimeSchema(t *testing.T) {
	path := filepath.Join("..", "..", "..", "bas", "seeds", "migrations", "sqlite", "001_projects.sql")
	migration, err := os.ReadFile(path)
	require.NoError(t, err)
	lines := strings.Split(string(migration), "\n")
	sqlLines := make([]string, 0, len(lines))
	for _, line := range lines {
		if !strings.HasPrefix(strings.TrimSpace(line), "--") {
			sqlLines = append(sqlLines, line)
		}
	}
	require.Equal(t, strings.TrimSpace(domain.Schema()), strings.TrimSpace(strings.Join(sqlLines, "\n")))
}

func setup(t *testing.T) (domain.Service, domain.Repository) {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(domain.Schema)))
	now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	r := domain.NewSQLiteRepository(d, func() time.Time { return now })
	return domain.NewService(r, func() time.Time { return now }), r
}

func TestCreateGetListAndPersistence(t *testing.T) {
	s, r := setup(t)
	ctx := context.Background()
	p, err := s.Create(ctx, domain.CreateInput{Name: "  Studio  "})
	require.NoError(t, err)
	require.Equal(t, "Studio", p.Name)
	require.EqualValues(t, 1, p.Revision)
	got, err := r.Get(ctx, p.ID)
	require.NoError(t, err)
	require.Equal(t, p, got)
	rows, next, err := s.List(ctx, 1, "")
	require.NoError(t, err)
	require.Empty(t, next)
	require.Equal(t, []domain.Project{p}, rows)
}

func TestRenameRequiresCurrentRevision(t *testing.T) {
	s, _ := setup(t)
	ctx := context.Background()
	p, err := s.Create(ctx, domain.CreateInput{Name: "First"})
	require.NoError(t, err)
	renamed, err := s.Rename(ctx, domain.RenameInput{ID: p.ID, Name: "Second", ExpectedRevision: p.Revision})
	require.NoError(t, err)
	require.Equal(t, "Second", renamed.Name)
	require.EqualValues(t, 2, renamed.Revision)
	_, err = s.Rename(ctx, domain.RenameInput{ID: p.ID, Name: "Lost update", ExpectedRevision: p.Revision})
	var conflict domain.ErrRevisionConflict
	require.ErrorAs(t, err, &conflict)
	require.EqualValues(t, 2, conflict.Current)
}

func TestListIsBoundedAndPaginated(t *testing.T) {
	s, _ := setup(t)
	ctx := context.Background()
	for _, name := range []string{"A", "B", "C"} {
		_, err := s.Create(ctx, domain.CreateInput{Name: name})
		require.NoError(t, err)
	}
	first, next, err := s.List(ctx, 2, "")
	require.NoError(t, err)
	require.Len(t, first, 2)
	require.NotEmpty(t, next)
	second, end, err := s.List(ctx, 2, next)
	require.NoError(t, err)
	require.Len(t, second, 1)
	require.Empty(t, end)
}

func TestCreateRejectsBlankName(t *testing.T) {
	s, _ := setup(t)
	_, err := s.Create(context.Background(), domain.CreateInput{Name: " \n "})
	var invalid domain.ErrInvalidProject
	require.True(t, errors.As(err, &invalid))
}

func TestKeysetPageRemainsStableAfterRename(t *testing.T) {
	s, _ := setup(t)
	ctx := context.Background()
	for _, name := range []string{"A", "B", "C"} {
		_, err := s.Create(ctx, domain.CreateInput{Name: name})
		require.NoError(t, err)
	}
	first, token, err := s.List(ctx, 2, "")
	require.NoError(t, err)
	require.Len(t, first, 2)
	require.NotEmpty(t, token)
	_, err = s.Rename(ctx, domain.RenameInput{ID: first[0].ID, Name: "Renamed", ExpectedRevision: first[0].Revision})
	require.NoError(t, err)
	second, end, err := s.List(ctx, 2, token)
	require.NoError(t, err)
	require.Len(t, second, 1)
	require.Empty(t, end)
	require.NotEqual(t, first[0].ID, second[0].ID)
	require.NotEqual(t, first[1].ID, second[0].ID)
}

func TestListOrdersLegacyRFC3339NanoTimestampsChronologically(t *testing.T) {
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(domain.Schema)))
	r := domain.NewSQLiteRepository(d, nil)
	for _, row := range []struct{ id, name, created string }{
		{"z-oldest", "Whole second", "2026-10-19T12:00:00Z"},
		{"y-middle", "One hundredth", "2026-10-19T12:00:00.01Z"},
		{"a-newest", "One tenth", "2026-10-19T12:00:00.1Z"},
	} {
		_, err := d.ExecContext(context.Background(), `INSERT INTO projects(id,name,created_at,updated_at,revision) VALUES(?,?,?,?,1)`, row.id, row.name, row.created, row.created)
		require.NoError(t, err)
	}
	service := domain.NewService(r, nil)
	var got []string
	token := ""
	for {
		page, next, err := service.List(context.Background(), 1, token)
		require.NoError(t, err)
		for _, project := range page {
			got = append(got, project.Name)
		}
		if next == "" {
			break
		}
		token = next
	}
	require.Equal(t, []string{"One tenth", "One hundredth", "Whole second"}, got)
}

func TestListRejectsMalformedPageToken(t *testing.T) {
	s, _ := setup(t)
	_, _, err := s.List(context.Background(), 10, "not-valid!")
	var invalid domain.ErrInvalidProject
	require.ErrorAs(t, err, &invalid)
}

func TestProjectNamesHaveLengthAndControlLimits(t *testing.T) {
	s, _ := setup(t)
	_, err := s.Create(context.Background(), domain.CreateInput{Name: string(make([]rune, 121))})
	var invalid domain.ErrInvalidProject
	require.ErrorAs(t, err, &invalid)
	_, err = s.Create(context.Background(), domain.CreateInput{Name: "bad\nname"})
	require.ErrorAs(t, err, &invalid)
}
