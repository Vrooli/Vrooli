package componenttests

import (
	"context"
	"testing"

	"react-component-library/internal/components"
	localdb "react-component-library/internal/database"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
)

type countingExecutor struct{ runs *int }

func (e countingExecutor) ExecuteStory(context.Context, string, string, string) (StoryExecution, error) {
	(*e.runs)++
	return StoryExecution{Passed: true, AccessibilityJSON: `{"contract":"bas-accessibility-snapshot/v1"}`}, nil
}

func TestServiceReusesReportForUnchangedFoldedRevision(t *testing.T) {
	database := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(components.Schema)))
	root := components.Component{ID: "root", CatalogID: "controls.root", LibraryID: "rcl:root", Slug: "root", AssetKind: components.AssetKindComponent}
	reader := assets{root: root, versions: map[string]components.ComponentVersion{"root@1.0.0": {ComponentID: root.ID, Version: "1.0.0", Content: "export const Root = () => null", ContentSHA256: "root"}}}
	runs := 0
	runner := Runner{
		Assets:   reader,
		Stories:  stories{"root@1.0.0": componentStory("default")},
		Executor: countingExecutor{runs: &runs},
		Revision: func(context.Context, string, string) (string, error) { return "folded-revision-1", nil },
	}
	service := NewService(runner, NewSQLiteRepository(database))
	first, reused, err := service.RunWithReuse(context.Background(), Request{ComponentID: "root", Version: "1.0.0"})
	require.NoError(t, err)
	require.False(t, reused)
	require.Equal(t, 1, runs)
	second, reused, err := service.RunWithReuse(context.Background(), Request{ComponentID: "root", Version: "1.0.0"})
	require.NoError(t, err)
	require.True(t, reused)
	require.Equal(t, first.ID, second.ID)
	require.Equal(t, "folded-revision-1", second.SourceRevision)
	require.Equal(t, 1, runs, "unchanged revision must not launch the story executor again")
}

func TestServiceRetriesBlockedReportForUnchangedFoldedRevision(t *testing.T) {
	database := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(components.Schema)))
	root := components.Component{ID: "root", CatalogID: "controls.root", LibraryID: "rcl:root", Slug: "root", AssetKind: components.AssetKindComponent}
	reader := assets{root: root, versions: map[string]components.ComponentVersion{"root@1.0.0": {ComponentID: root.ID, Version: "1.0.0", Content: "export const Root = () => null", ContentSHA256: "root"}}}
	runs := 0
	runner := Runner{
		Assets:   reader,
		Stories:  stories{"root@1.0.0": componentStory("default")},
		Executor: countingExecutor{runs: &runs},
		Revision: func(context.Context, string, string) (string, error) { return "folded-revision-1", nil },
	}
	reports := NewSQLiteRepository(database)
	blocked := Report{ID: reportID(Report{RootLibraryID: root.LibraryID, RootVersion: "1.0.0", SourceRevision: "folded-revision-1"}), RootLibraryID: root.LibraryID, RootVersion: "1.0.0", SourceRevision: "folded-revision-1", Verdict: VerdictBlocked, Results: []Result{{Stage: StageDeclared, Verdict: VerdictBlocked, Message: "story executor is unavailable: Chrome not found"}}}
	require.NoError(t, reports.Save(context.Background(), blocked))
	service := NewService(runner, reports)
	_, reused, err := service.RunWithReuse(context.Background(), Request{ComponentID: "root", Version: "1.0.0"})
	require.NoError(t, err)
	require.False(t, reused, "an environment-blocked report must be retried")
	require.Equal(t, 1, runs)
}

func TestServiceDoesNotReuseReportAfterFoldedRevisionChanges(t *testing.T) {
	database := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(components.Schema)))
	root := components.Component{ID: "root", CatalogID: "controls.root", LibraryID: "rcl:root", Slug: "root", AssetKind: components.AssetKindComponent}
	reader := assets{root: root, versions: map[string]components.ComponentVersion{"root@1.0.0": {ComponentID: root.ID, Version: "1.0.0", Content: "export const Root = () => null", ContentSHA256: "root"}}}
	revision := "folded-revision-1"
	runs := 0
	runner := Runner{
		Assets:   reader,
		Stories:  stories{"root@1.0.0": componentStory("default")},
		Executor: countingExecutor{runs: &runs},
		Revision: func(context.Context, string, string) (string, error) { return revision, nil },
	}
	service := NewService(runner, NewSQLiteRepository(database))
	_, reused, err := service.RunWithReuse(context.Background(), Request{ComponentID: "root", Version: "1.0.0"})
	require.NoError(t, err)
	require.False(t, reused)

	revision = "folded-revision-2"
	_, reused, err = service.RunWithReuse(context.Background(), Request{ComponentID: "root", Version: "1.0.0"})
	require.NoError(t, err)
	require.False(t, reused, "a changed folded revision must launch a fresh report")
	require.Equal(t, 2, runs)
}
