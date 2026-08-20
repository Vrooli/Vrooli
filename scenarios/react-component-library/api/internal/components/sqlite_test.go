package components_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"

	"react-component-library/internal/components"
	localdb "react-component-library/internal/database"

	db "github.com/vrooli/api-core/databasetest"

	"github.com/vrooli/api-core/scheduletest"
)

func newComponentsRawDB(t *testing.T) (*sql.DB, components.Repository, func() time.Time) {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(components.Schema),
	))
	clk := scheduletest.New(time.Date(2026, 5, 12, 10, 0, 0, 0, time.UTC))
	repo := components.NewSQLiteRepository(d, clk)
	return d, repo, clk.Now
}

func newComponentsDB(t *testing.T) (components.Repository, func() time.Time) {
	t.Helper()
	_, repo, now := newComponentsRawDB(t)
	return repo, now
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
	require.Empty(t, c1.Category)
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
	require.Equal(t, "controls", c2.Category)
	require.NotContains(t, c2.Headers, "category")
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
			"catalogId": "controls.button",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "controls", c.Category)
	require.Equal(t, "DO NOT REMOVE", c.Headers["warning"])
	require.NotContains(t, c.Headers, "libraryId")
	require.NotContains(t, c.Headers, "version")
	require.NotContains(t, c.Headers, "deps")
	require.NotContains(t, c.Headers, "category")

	got, err := repo.List(ctx, components.SearchQuery{Category: "controls", Limit: 10})
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "react-component-library:Button", got[0].LibraryID)
	require.Nil(t, got[0].Headers, "List omits the arbitrary header map; Get carries it")
	// The lean-payload rule applies to the raw map only. The typed catalog
	// projection must survive List: it is what the catalog browser groups by,
	// and loading it as a side effect of the full header map is what left
	// CatalogID empty for every listed asset.
	require.Equal(t, "controls.button", got[0].CatalogID, "List must carry the typed catalog projection")

	fetched, err := repo.Get(ctx, c.ID)
	require.NoError(t, err)
	require.Equal(t, "controls", fetched.Category)
	require.NotContains(t, fetched.Headers, "category")
}

func TestSQLiteRepository_UpsertManifestPreservesCreatedAtAndReleasedAt(t *testing.T) {
	d, repo, now := newComponentsRawDB(t)
	ctx := context.Background()
	input := components.IndexManifestInput{
		Manifest: components.ComponentManifest{
			LibraryID:     "react-component-library:Button",
			Slug:          "Button",
			LatestVersion: "1.0.0",
		},
		Versions: []components.ComponentVersion{{
			Version:       "1.0.0",
			Status:        components.VersionStatusReleased,
			SourcePath:    "components/Button/versions/1.0.0/Button.tsx",
			Content:       "export const Button = () => null;",
			ContentSHA256: "first",
		}},
	}
	first, err := repo.UpsertManifest(ctx, input)
	require.NoError(t, err)
	firstVersion, err := repo.GetVersion(ctx, first.ID, "1.0.0")
	require.NoError(t, err)
	require.False(t, firstVersion.CreatedAt.IsZero())
	require.False(t, firstVersion.ReleasedAt.IsZero())

	_ = now()
	input.Versions[0].Content = "export const Button = () => true;"
	input.Versions[0].ContentSHA256 = "second"
	_, err = repo.UpsertManifest(ctx, input)
	var immutable components.ErrReleasedVersionMutated
	require.ErrorAs(t, err, &immutable)

	var count int
	require.NoError(t, d.QueryRowContext(ctx, `SELECT COUNT(*) FROM component_versions WHERE component_id = ?`, first.ID).Scan(&count))
	require.Equal(t, 1, count)
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
				{StyleID: "vrooli-default", Affinity: components.DesignAffinityNative, Reason: "token-native baseline"},
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
		{StyleID: "vrooli-default", Affinity: components.DesignAffinityNative, Reason: "token-native baseline"},
	}, button.DesignStyles)

	fetchedButton, err := repo.Get(ctx, button.ID)
	require.NoError(t, err)
	require.Equal(t, []components.ComponentDesignAffinity{
		{StyleID: "vrooli-conversion-landing", Affinity: components.DesignAffinityCompatible},
		{StyleID: "vrooli-default", Affinity: components.DesignAffinityNative, Reason: "token-native baseline"},
	}, fetchedButton.DesignStyles)

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

	got, err = repo.List(ctx, components.SearchQuery{Affinity: "native", Limit: 10})
	require.NoError(t, err)
	require.Len(t, got, 2)
	gotIDs := []string{got[0].LibraryID, got[1].LibraryID}
	require.Contains(t, gotIDs, "react-component-library:Button")
	require.Contains(t, gotIDs, "react-component-library:DataTable")
}

func TestSQLiteRepository_UpsertManifestPersistsStories(t *testing.T) {
	repo, _ := newComponentsDB(t)
	ctx := context.Background()

	c, err := repo.UpsertManifest(ctx, components.IndexManifestInput{
		Manifest: components.ComponentManifest{
			LibraryID:     "react-component-library:Button",
			Slug:          "Button",
			DisplayName:   "Button",
			Slot:          "ui-primitive",
			LatestVersion: "1.0.0",
		},
		Versions: []components.ComponentVersion{
			{Version: "1.0.0", Status: components.VersionStatusReleased, SourcePath: "components/Button/versions/1.0.0/Button.tsx", ContentSHA256: "button"},
		},
		Stories: []components.ComponentStory{{
			Version: "1.0.0", SchemaVersion: 1, Kind: components.StoryKindComponent,
			ArgsJSON:        `{"fields":[{"path":"tone","kind":"enum"}]}`,
			EnvironmentJSON: `{"fixtures":[]}`,
			StoriesJSON:     `[{"id":"primary","name":"Primary","args":{"tone":"primary"}}]`,
			ContractJSON:    `{"schemaVersion":1,"kind":"component"}`,
			SourcePath:      "components/Button/versions/1.0.0/story.json",
		}},
	})
	require.NoError(t, err)

	stories, err := repo.ListStories(ctx, components.StoryQuery{ComponentID: c.ID, Version: "1.0.0"})
	require.NoError(t, err)
	require.Len(t, stories, 1)
	require.Equal(t, components.StoryKindComponent, stories[0].Kind)
	require.JSONEq(t, `{"fixtures":[]}`, stories[0].EnvironmentJSON)

	_, err = repo.UpsertManifest(ctx, components.IndexManifestInput{
		Manifest: components.ComponentManifest{
			LibraryID:     "react-component-library:Button",
			Slug:          "Button",
			DisplayName:   "Button",
			Slot:          "ui-primitive",
			LatestVersion: "1.0.0",
		},
		Versions: []components.ComponentVersion{
			{Version: "1.0.0", Status: components.VersionStatusReleased, SourcePath: "components/Button/versions/1.0.0/Button.tsx", ContentSHA256: "button"},
		},
	})
	require.NoError(t, err)

	stories, err = repo.ListStories(ctx, components.StoryQuery{ComponentID: c.ID, Version: "1.0.0"})
	require.NoError(t, err)
	require.Empty(t, stories, "stories are rebuilt from disk and removed when story.json disappears")
}

func TestSQLiteRepository_AdditiveCategoryAndReasonMigrationPreservesRows(t *testing.T) {
	d := db.NewSQLite(t)
	ctx := context.Background()
	require.NoError(t, apidb.EnsureSchemas(ctx, d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(func() string {
			return `
CREATE TABLE IF NOT EXISTS components (
  id TEXT PRIMARY KEY,
  library_id TEXT NOT NULL UNIQUE,
  slug TEXT NOT NULL UNIQUE,
  display_name TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  slot TEXT NOT NULL DEFAULT '',
  source_path TEXT NOT NULL,
  version TEXT NOT NULL DEFAULT '',
  latest_version TEXT NOT NULL DEFAULT '',
  draft_version TEXT NOT NULL DEFAULT '',
  manifest_path TEXT NOT NULL DEFAULT '',
  tags TEXT NOT NULL DEFAULT '',
  indexed_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS component_design_affinities (
  component_id TEXT NOT NULL,
  style_id TEXT NOT NULL,
  affinity TEXT NOT NULL,
  PRIMARY KEY (component_id, style_id)
);`
		}),
	))
	_, err := d.ExecContext(ctx, `
INSERT INTO components (id, library_id, slug, display_name, slot, source_path, version, latest_version, manifest_path, tags, indexed_at, updated_at)
VALUES ('component-1', 'lib:Button', 'Button', 'Button', 'ui-primitive', 'Button.tsx', '1.0.0', '1.0.0', 'component.json', '', '2026-05-12T10:00:00Z', '2026-05-12T10:00:00Z');
INSERT INTO component_design_affinities (component_id, style_id, affinity)
VALUES ('component-1', 'vrooli-default', 'native');`)
	require.NoError(t, err)

	require.NoError(t, components.EnsureSchemaMigrations(ctx, d))
	require.NoError(t, components.EnsureSchemaMigrations(ctx, d), "migration is boot-idempotent")

	require.NoError(t, apidb.EnsureSchemas(ctx, d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(components.Schema),
	))
	repo := components.NewSQLiteRepository(d, scheduletest.New(time.Date(2026, 5, 12, 11, 0, 0, 0, time.UTC)))
	got, err := repo.GetByLibraryID(ctx, "lib:Button")
	require.NoError(t, err)
	require.Equal(t, "lib:Button", got.LibraryID)
	require.Empty(t, got.Category)
	require.Equal(t, []components.ComponentDesignAffinity{
		{StyleID: "vrooli-default", Affinity: components.DesignAffinityNative},
	}, got.DesignStyles)
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

// seedOrphanVersion inserts a component_versions row (plus one child
// file row) whose component_id has no owning components registry row —
// the soft-FK cruft a re-slug or withdrawal leaves behind.
func seedOrphanVersion(t *testing.T, d *sql.DB, versionID, componentID, libraryID, version, sourcePath string) {
	t.Helper()
	_, err := d.ExecContext(context.Background(), `
INSERT INTO component_versions
  (id, component_id, library_id, version, status, source_path, content, content_sha256, changelog_md, indexed_at, released_at)
VALUES (?, ?, ?, ?, 'released', ?, '', 'sha', '', '2026-05-12T10:00:00Z', '')`,
		versionID, componentID, libraryID, version, sourcePath)
	require.NoError(t, err)
	_, err = d.ExecContext(context.Background(),
		`INSERT INTO component_version_files (version_id, path, content, content_sha256, is_entry) VALUES (?, ?, '', 'sha', 1)`,
		versionID, sourcePath)
	require.NoError(t, err)
}

func countRows(t *testing.T, d *sql.DB, query string, args ...any) int {
	t.Helper()
	var n int
	require.NoError(t, d.QueryRowContext(context.Background(), query, args...).Scan(&n))
	return n
}

func TestSQLiteRepository_SweepOrphans_RemovesRegistryOrphansKeepsParented(t *testing.T) {
	d, repo, _ := newComponentsRawDB(t)
	ctx := context.Background()

	live, err := repo.UpsertManifest(ctx, components.IndexManifestInput{
		Manifest: components.ComponentManifest{
			LibraryID:     "react-component-library:Button",
			Slug:          "Button",
			DisplayName:   "Button",
			ManifestPath:  "components/Button/component.json",
			LatestVersion: "1.0.0",
		},
		Versions: []components.ComponentVersion{{
			LibraryID:     "react-component-library:Button",
			Version:       "1.0.0",
			Status:        components.VersionStatusReleased,
			SourcePath:    "components/Button/versions/1.0.0/Button.tsx",
			Content:       "export const Button = () => null;",
			ContentSHA256: "sha-button",
		}},
	})
	require.NoError(t, err)

	seedOrphanVersion(t, d, "orphan-1", "cmp-gone", "react-component-library:tab-bar", "0.1.0",
		"components/tab-bar/versions/0.1.0/tab-bar.tsx")
	seedOrphanVersion(t, d, "orphan-2", "cmp-gone", "react-component-library:tab-bar", "0.1.0-draft.1",
		"components/tab-bar/versions/0.1.0-draft.1/tab-bar.tsx")

	orphans, err := repo.SweepOrphans(ctx)
	require.NoError(t, err)
	require.Len(t, orphans, 2, "both registry-orphaned rows must be reported")
	// Ordered by library_id then version.
	require.Equal(t, "react-component-library:tab-bar", orphans[0].LibraryID)
	require.Equal(t, "0.1.0", orphans[0].Version)
	require.Equal(t, "cmp-gone", orphans[0].ComponentID)
	require.Equal(t, "0.1.0-draft.1", orphans[1].Version)

	// Orphan version + child file rows are gone.
	require.Zero(t, countRows(t, d,
		`SELECT count(*) FROM component_versions WHERE component_id NOT IN (SELECT id FROM components)`))
	require.Zero(t, countRows(t, d,
		`SELECT count(*) FROM component_version_files WHERE version_id IN ('orphan-1','orphan-2')`))

	// The registry-parented version is untouched.
	kept, err := repo.ListVersions(ctx, live.ID, 10)
	require.NoError(t, err)
	require.Len(t, kept, 1)

	// Idempotent: a second sweep finds nothing.
	again, err := repo.SweepOrphans(ctx)
	require.NoError(t, err)
	require.Empty(t, again)
}
