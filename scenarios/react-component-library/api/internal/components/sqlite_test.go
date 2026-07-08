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
		Slot:        "ui-primitive",
		SourcePath:  "components/Button.tsx",
		Version:     "1.0.0",
		Tags:        []string{"form", "interactive"},
		Headers:     map[string]string{"libraryId": "react-component-library:Button", "version": "1.0.0", "warning": "DO NOT REMOVE"},
	})
	require.NoError(t, err)
	require.NotEmpty(t, c1.ID)
	require.False(t, c1.IndexedAt.IsZero())
	require.Equal(t, c1.IndexedAt, c1.UpdatedAt)
	require.Equal(t, []string{"form", "interactive"}, c1.Tags)
	require.Equal(t, "ui-primitive", c1.Slot)
	require.Equal(t, "DO NOT REMOVE", c1.Headers["warning"])
	require.NotContains(t, c1.Headers, "libraryId")
	require.NotContains(t, c1.Headers, "version")

	c2, err := repo.Upsert(ctx, components.UpsertInput{
		LibraryID:   "react-component-library:Button",
		DisplayName: "Button (renamed)",
		Slot:        "ui-pattern",
		SourcePath:  "components/Button.tsx",
		Version:     "1.1.0",
		Headers:     map[string]string{"category": "controls"},
	})
	require.NoError(t, err)
	require.Equal(t, c1.ID, c2.ID, "upsert by libraryId must reuse the existing primary key")
	require.Equal(t, "Button (renamed)", c2.DisplayName)
	require.Equal(t, "ui-pattern", c2.Slot)
	require.Equal(t, "controls", c2.Headers["category"])
	require.NotContains(t, c2.Headers, "warning")
	require.Equal(t, c1.IndexedAt, c2.IndexedAt, "IndexedAt is sticky")
}

func TestSQLiteRepository_UpsertManifestPersistsLatestHeadersForCategoryFacet(t *testing.T) {
	repo, _ := newComponentsDB(t)
	ctx := context.Background()

	c, err := repo.UpsertManifest(ctx, components.IndexManifestInput{
		Manifest: components.ComponentManifest{
			LibraryID:     "react-component-library:Button",
			Slug:          "Button",
			DisplayName:   "Button",
			Slot:          "ui-primitive",
			LatestVersion: "1.1.0",
			Tags:          []string{"form", "interactive"},
		},
		Versions: []components.ComponentVersion{
			{Version: "1.0.0", Status: components.VersionStatusReleased, SourcePath: "components/Button/versions/1.0.0/Button.tsx", ContentSHA256: "old"},
			{Version: "1.1.0", Status: components.VersionStatusReleased, SourcePath: "components/Button/versions/1.1.0/Button.tsx", ContentSHA256: "new"},
		},
		Headers: map[string]string{
			"libraryId": "react-component-library:Button",
			"version":   "1.1.0",
			"deps":      `{"react":"^18"}`,
			"category":  "controls",
			"warning":   "DO NOT REMOVE",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "controls", c.Headers["category"])
	require.Equal(t, "DO NOT REMOVE", c.Headers["warning"])
	require.NotContains(t, c.Headers, "libraryId")
	require.NotContains(t, c.Headers, "version")
	require.NotContains(t, c.Headers, "deps")

	got, err := repo.List(ctx, components.SearchQuery{Category: "controls", Limit: 10})
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "react-component-library:Button", got[0].LibraryID)
	require.Nil(t, got[0].Headers, "List omits headers; Get carries them")

	fetched, err := repo.Get(ctx, c.ID)
	require.NoError(t, err)
	require.Equal(t, "controls", fetched.Headers["category"])
}

func TestSQLiteRepository_UpsertManifestPersistsDesignAffinitiesAndFilters(t *testing.T) {
	repo, _ := newComponentsDB(t)
	ctx := context.Background()

	button, err := repo.UpsertManifest(ctx, components.IndexManifestInput{
		Manifest: components.ComponentManifest{
			LibraryID:     "react-component-library:Button",
			Slug:          "Button",
			DisplayName:   "Button",
			Slot:          "ui-primitive",
			LatestVersion: "1.0.0",
			DesignStyles: []components.ComponentDesignAffinity{
				{StyleID: "vrooli-default", Affinity: components.DesignAffinityNative},
				{StyleID: "vrooli-conversion-landing", Affinity: components.DesignAffinityCompatible},
			},
		},
		Versions: []components.ComponentVersion{
			{Version: "1.0.0", Status: components.VersionStatusReleased, SourcePath: "components/Button/versions/1.0.0/Button.tsx", ContentSHA256: "button"},
		},
	})
	require.NoError(t, err)
	require.Equal(t, []components.ComponentDesignAffinity{
		{StyleID: "vrooli-conversion-landing", Affinity: components.DesignAffinityCompatible},
		{StyleID: "vrooli-default", Affinity: components.DesignAffinityNative},
	}, button.DesignStyles)

	_, err = repo.UpsertManifest(ctx, components.IndexManifestInput{
		Manifest: components.ComponentManifest{
			LibraryID:     "react-component-library:DataTable",
			Slug:          "DataTable",
			DisplayName:   "Data Table",
			Slot:          "ui-pattern",
			LatestVersion: "1.0.0",
			DesignStyles: []components.ComponentDesignAffinity{
				{StyleID: "vrooli-default", Affinity: components.DesignAffinityNative},
				{StyleID: "vrooli-conversion-landing", Affinity: components.DesignAffinityDiscouraged},
			},
		},
		Versions: []components.ComponentVersion{
			{Version: "1.0.0", Status: components.VersionStatusReleased, SourcePath: "components/DataTable/versions/1.0.0/DataTable.tsx", ContentSHA256: "table"},
		},
	})
	require.NoError(t, err)

	got, err := repo.List(ctx, components.SearchQuery{StyleID: "vrooli-conversion-landing", Limit: 10})
	require.NoError(t, err)
	require.Len(t, got, 2)

	got, err = repo.List(ctx, components.SearchQuery{StyleID: "vrooli-conversion-landing", Affinity: "discouraged", Limit: 10})
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "react-component-library:DataTable", got[0].LibraryID)
	require.Equal(t, components.DesignAffinityDiscouraged, got[0].DesignStyles[0].Affinity)
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
		{LibraryID: "lib:Input", DisplayName: "Input", Description: "text input field", Slot: "ui-primitive", Tags: []string{"form", "input"}},
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

	// Match against slot.
	got, err = repo.List(ctx, components.SearchQuery{Match: "primitive", Limit: 10})
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

func TestSQLiteRepository_ListMultiTagAndCategory(t *testing.T) {
	repo, _ := newComponentsDB(t)
	ctx := context.Background()

	seed := []components.UpsertInput{
		{
			LibraryID:   "lib:Button",
			DisplayName: "Button",
			Tags:        []string{"form", "interactive"},
			Headers:     map[string]string{"category": "controls"},
		},
		{
			LibraryID:   "lib:Card",
			DisplayName: "Card",
			Tags:        []string{"layout"},
			Headers:     map[string]string{"category": "containers"},
		},
		{
			LibraryID:   "lib:Input",
			DisplayName: "Input",
			Tags:        []string{"form", "input"},
			Headers:     map[string]string{"category": "controls"},
		},
		{
			LibraryID:   "lib:Banner",
			DisplayName: "Banner",
			Tags:        []string{"layout", "feedback"},
			Headers:     map[string]string{"category": "containers"},
		},
	}
	for _, in := range seed {
		_, err := repo.Upsert(ctx, in)
		require.NoError(t, err)
	}

	// Multi-tag OR (any-of).
	got, err := repo.List(ctx, components.SearchQuery{Tags: []string{"form", "layout"}, Limit: 10})
	require.NoError(t, err)
	require.Len(t, got, 4, "form OR layout should match all four seeded rows")

	got, err = repo.List(ctx, components.SearchQuery{Tags: []string{"input"}, Limit: 10})
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "lib:Input", got[0].LibraryID)

	// Whitespace-only entries are ignored.
	got, err = repo.List(ctx, components.SearchQuery{Tags: []string{"", "  ", "feedback"}, Limit: 10})
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "lib:Banner", got[0].LibraryID)

	// Category AND.
	got, err = repo.List(ctx, components.SearchQuery{Category: "controls", Limit: 10})
	require.NoError(t, err)
	require.Len(t, got, 2)
	gotIDs := []string{got[0].LibraryID, got[1].LibraryID}
	require.Contains(t, gotIDs, "lib:Button")
	require.Contains(t, gotIDs, "lib:Input")

	// Category + multi-tag AND.
	got, err = repo.List(ctx, components.SearchQuery{
		Category: "containers",
		Tags:     []string{"feedback"},
		Limit:    10,
	})
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "lib:Banner", got[0].LibraryID)

	// Match + category + tag — full AND composition. Match against name.
	got, err = repo.List(ctx, components.SearchQuery{
		Match:    "button",
		Category: "controls",
		Tags:     []string{"form"},
		Limit:    10,
	})
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "lib:Button", got[0].LibraryID)

	// Empty match returns full list (limit-capped).
	got, err = repo.List(ctx, components.SearchQuery{Limit: 10})
	require.NoError(t, err)
	require.Len(t, got, 4)
}

func TestSQLiteRepository_ListMatchOrdersByDisplayNameCaseInsensitive(t *testing.T) {
	repo, _ := newComponentsDB(t)
	ctx := context.Background()

	for _, in := range []components.UpsertInput{
		{LibraryID: "lib:zebra", DisplayName: "zebra"},
		{LibraryID: "lib:Apple", DisplayName: "Apple"},
		{LibraryID: "lib:mango", DisplayName: "mango"},
		{LibraryID: "lib:Banana", DisplayName: "Banana"},
	} {
		_, err := repo.Upsert(ctx, in)
		require.NoError(t, err)
	}

	// Match "" filters nothing — but req SF-001 only specifies match-path
	// ordering; we keep newest-first for unfiltered lists. Test the
	// match-path explicitly.
	got, err := repo.List(ctx, components.SearchQuery{Match: "a", Limit: 10})
	require.NoError(t, err)
	require.Len(t, got, 4)
	require.Equal(t, []string{"Apple", "Banana", "mango", "zebra"},
		[]string{got[0].DisplayName, got[1].DisplayName, got[2].DisplayName, got[3].DisplayName},
		"match-mode order is display_name COLLATE NOCASE")
}

func TestSQLiteRepository_ListSearchP95UnderBudget(t *testing.T) {
	// Req SF-003 — 1000 synthetic components, p95 < 100ms across a
	// representative query mix on an in-memory SQLite.
	if testing.Short() {
		t.Skip("skipping benchmark under -short")
	}
	repo, _ := newComponentsDB(t)
	ctx := context.Background()

	for i := 0; i < 1000; i++ {
		cat := "controls"
		if i%3 == 0 {
			cat = "containers"
		} else if i%5 == 0 {
			cat = "feedback"
		}
		tag := "form"
		if i%2 == 0 {
			tag = "layout"
		}
		_, err := repo.Upsert(ctx, components.UpsertInput{
			LibraryID:   "lib:Comp" + itoa(i),
			DisplayName: "Comp " + itoa(i),
			Description: "synthetic seed row #" + itoa(i),
			Tags:        []string{tag, "synthetic"},
			Headers:     map[string]string{"category": cat},
		})
		require.NoError(t, err)
	}

	queries := []components.SearchQuery{
		{Match: "synthetic", Limit: 200},
		{Tags: []string{"form"}, Limit: 200},
		{Category: "controls", Limit: 200},
		{Match: "Comp 5", Category: "controls", Tags: []string{"form", "layout"}, Limit: 200},
	}

	const samples = 50
	latencies := make([]time.Duration, 0, samples*len(queries))
	for i := 0; i < samples; i++ {
		for _, q := range queries {
			start := time.Now()
			_, err := repo.List(ctx, q)
			require.NoError(t, err)
			latencies = append(latencies, time.Since(start))
		}
	}

	// Percentile via insertion sort — N=200 stays cheap and avoids an
	// import for a one-shot benchmark.
	for i := 1; i < len(latencies); i++ {
		for j := i; j > 0 && latencies[j-1] > latencies[j]; j-- {
			latencies[j-1], latencies[j] = latencies[j], latencies[j-1]
		}
	}
	p95 := latencies[(len(latencies)*95)/100]
	t.Logf("search p95 over %d samples × %d queries: %v", samples, len(queries), p95)
	require.Less(t, p95, 100*time.Millisecond, "req SF-003 p95 budget")
}

// itoa avoids strconv import noise inside the bench loop.
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
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
