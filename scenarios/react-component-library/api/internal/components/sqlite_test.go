package components_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"

	"react-component-library/internal/components"
	localdb "react-component-library/internal/database"
	"react-component-library/internal/testutil/db"
	"react-component-library/internal/testutil/mocks"
)

func newComponentsDB(t *testing.T) (components.Repository, func() time.Time) {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(components.Schema),
	))
	clk := mocks.NewFakeClock(time.Date(2026, 5, 12, 10, 0, 0, 0, time.UTC))
	repo := components.NewSQLiteRepository(d, clk)
	return repo, clk.Now
}

func TestSQLiteRepository_UpsertInsertsThenUpdates(t *testing.T) {
	repo, _ := newComponentsDB(t)
	ctx := context.Background()

	c1, err := repo.Upsert(ctx, components.UpsertInput{
		LibraryID:   "react-component-library:Button",
		DisplayName: "Button",
		Description: "Primary CTA",
		SourcePath:  "components/Button.tsx",
		Version:     "1.0.0",
		Tags:        []string{"form", "interactive"},
		Headers:     map[string]string{"libraryId": "react-component-library:Button", "version": "1.0.0"},
	})
	require.NoError(t, err)
	require.NotEmpty(t, c1.ID)
	require.False(t, c1.IndexedAt.IsZero())
	require.Equal(t, c1.IndexedAt, c1.UpdatedAt)
	require.Equal(t, []string{"form", "interactive"}, c1.Tags)
	require.Equal(t, "1.0.0", c1.Headers["version"])

	c2, err := repo.Upsert(ctx, components.UpsertInput{
		LibraryID:   "react-component-library:Button",
		DisplayName: "Button (renamed)",
		SourcePath:  "components/Button.tsx",
		Version:     "1.1.0",
		Headers:     map[string]string{"libraryId": "react-component-library:Button", "version": "1.1.0"},
	})
	require.NoError(t, err)
	require.Equal(t, c1.ID, c2.ID, "upsert by libraryId must reuse the existing primary key")
	require.Equal(t, "Button (renamed)", c2.DisplayName)
	require.Equal(t, c1.IndexedAt, c2.IndexedAt, "IndexedAt is sticky")
}

func TestSQLiteRepository_GetByLibraryID_NotFound(t *testing.T) {
	repo, _ := newComponentsDB(t)
	_, err := repo.GetByLibraryID(context.Background(), "missing")
	var nf components.ErrComponentNotFound
	require.True(t, errors.As(err, &nf), "got %T", err)
	require.Equal(t, "missing", nf.IDOrLibraryID)
}

func TestSQLiteRepository_ListSearchAndTagFilter(t *testing.T) {
	repo, _ := newComponentsDB(t)
	ctx := context.Background()

	seed := []components.UpsertInput{
		{LibraryID: "lib:Button", DisplayName: "Button", Description: "click me", Tags: []string{"form"}},
		{LibraryID: "lib:Card", DisplayName: "Card", Description: "container", Tags: []string{"layout"}},
		{LibraryID: "lib:Input", DisplayName: "Input", Description: "text input field", Tags: []string{"form", "input"}},
	}
	for _, in := range seed {
		_, err := repo.Upsert(ctx, in)
		require.NoError(t, err)
	}

	// Match against description.
	got, err := repo.List(ctx, components.SearchQuery{Match: "input", Limit: 10})
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "lib:Input", got[0].LibraryID)

	// Tag filter.
	got, err = repo.List(ctx, components.SearchQuery{Tag: "form", Limit: 10})
	require.NoError(t, err)
	require.Len(t, got, 2)

	// Limit <= 0 returns nil.
	got, err = repo.List(ctx, components.SearchQuery{Limit: 0})
	require.NoError(t, err)
	require.Nil(t, got)
}

func TestSQLiteRepository_DeleteMissing(t *testing.T) {
	repo, _ := newComponentsDB(t)
	ctx := context.Background()

	for _, lib := range []string{"a", "b", "c"} {
		_, err := repo.Upsert(ctx, components.UpsertInput{LibraryID: lib, SourcePath: lib + ".tsx"})
		require.NoError(t, err)
	}
	deleted, err := repo.DeleteMissing(ctx, []string{"a", "c"})
	require.NoError(t, err)
	require.Equal(t, 1, deleted)

	_, err = repo.GetByLibraryID(ctx, "b")
	require.Error(t, err)
	_, err = repo.GetByLibraryID(ctx, "a")
	require.NoError(t, err)

	// Empty keep list wipes everything.
	deleted, err = repo.DeleteMissing(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, 2, deleted)
}
