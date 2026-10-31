package componenttests

import (
	"context"
	"database/sql"
	"testing"

	"react-component-library/internal/components"
	localdb "react-component-library/internal/database"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
)

type readbackRepository struct {
	reports map[string]Report
	missing bool
}

func (r *readbackRepository) Save(_ context.Context, report Report) error {
	r.reports[report.ID] = report
	return nil
}

func (r *readbackRepository) Get(_ context.Context, id string) (Report, error) {
	if r.missing {
		return Report{}, sql.ErrNoRows
	}
	report, ok := r.reports[id]
	if !ok {
		return Report{}, sql.ErrNoRows
	}
	return report, nil
}

func (*readbackRepository) List(context.Context, string, string, int) ([]Report, error) {
	return nil, nil
}

func TestServiceReadsBackReportByExactKeyBeforeSuccess(t *testing.T) {
	root := components.Component{ID: "root", CatalogID: "controls.root", LibraryID: "rcl:root", Slug: "root", AssetKind: components.AssetKindComponent}
	reader := assets{root: root, versions: map[string]components.ComponentVersion{"root@1.0.0": {ComponentID: root.ID, Version: "1.0.0", Content: "export const Root = () => null", ContentSHA256: "root"}}}
	runs := 0
	runner := Runner{Assets: reader, Stories: stories{"root@1.0.0": componentStory("default")}, Executor: countingExecutor{runs: &runs}, Revision: func(context.Context, string, string) (string, error) { return "revision-1", nil }}
	reports := &readbackRepository{reports: map[string]Report{}}
	report, reused, err := NewService(runner, reports).RunWithReuse(context.Background(), Request{ComponentID: root.ID, Version: "1.0.0", IncludeClosure: true})
	require.NoError(t, err)
	require.False(t, reused)
	keyed, ok := reports.reports[report.ID]
	require.True(t, ok, "the returned ID must be the key used for durable storage")
	require.Equal(t, report.ID, keyed.ID)
	require.Equal(t, report.RootLibraryID, keyed.RootLibraryID)
	require.Equal(t, report.RootVersion, keyed.RootVersion)
	require.Equal(t, report.IncludeClosure, keyed.IncludeClosure)
	require.Equal(t, report.SourceRevision, keyed.SourceRevision)
	require.Equal(t, report.Verdict, keyed.Verdict)
}

func TestServiceDoesNotReturnSuccessWhenSavedReportIsMissingOnReadback(t *testing.T) {
	root := components.Component{ID: "root", CatalogID: "controls.root", LibraryID: "rcl:root", Slug: "root", AssetKind: components.AssetKindComponent}
	reader := assets{root: root, versions: map[string]components.ComponentVersion{"root@1.0.0": {ComponentID: root.ID, Version: "1.0.0", Content: "export const Root = () => null", ContentSHA256: "root"}}}
	runs := 0
	runner := Runner{Assets: reader, Stories: stories{"root@1.0.0": componentStory("default")}, Executor: countingExecutor{runs: &runs}, Revision: func(context.Context, string, string) (string, error) { return "revision-1", nil }}
	reports := &readbackRepository{reports: map[string]Report{}, missing: true}
	report, reused, err := NewService(runner, reports).RunWithReuse(context.Background(), Request{ComponentID: root.ID, Version: "1.0.0", IncludeClosure: true})
	require.Error(t, err)
	require.NotErrorIs(t, err, sql.ErrNoRows, "readback absence must be surfaced as an internal postcondition failure")
	require.Contains(t, err.Error(), "could not be read back")
	require.Empty(t, report.ID, "a passing execution must not escape as success without exact-ID readback")
	require.False(t, reused)
	require.Equal(t, 1, runs)
}

type countingExecutor struct{ runs *int }

type changingFreshnessExecutor struct {
	runs      *int
	freshness *string
}

func (e changingFreshnessExecutor) ReportFreshnessKey() string { return *e.freshness }

func (e countingExecutor) ExecuteStory(context.Context, string, string, string) (StoryExecution, error) {
	(*e.runs)++
	return StoryExecution{Passed: true, AccessibilityJSON: `{"contract":"bas-accessibility-snapshot/v1"}`}, nil
}

func (e changingFreshnessExecutor) ExecuteStory(context.Context, string, string, string) (StoryExecution, error) {
	(*e.runs)++
	return StoryExecution{Passed: true, AccessibilityJSON: `{"contract":"bas-accessibility-snapshot/v1"}`, Artifacts: []Artifact{{Reference: "test://capture"}}}, nil
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

func TestServiceDoesNotReuseFailedReportAfterExecutorFreshnessChanges(t *testing.T) {
	database := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(components.Schema)))
	root := components.Component{ID: "root", CatalogID: "controls.root", LibraryID: "rcl:root", Slug: "root", AssetKind: components.AssetKindComponent}
	reader := assets{root: root, versions: map[string]components.ComponentVersion{"root@1.0.0": {ComponentID: root.ID, Version: "1.0.0", Content: "export const Root = () => null", ContentSHA256: "root"}}}
	freshness := "executor-policy-1"
	runs := 0
	runner := Runner{
		Assets: reader, Stories: stories{"root@1.0.0": componentStory("default")},
		Executor: changingFreshnessExecutor{runs: &runs, freshness: &freshness},
		Revision: func(context.Context, string, string) (string, error) { return "folded-revision-1", nil },
	}
	reports := NewSQLiteRepository(database)
	old := Report{
		ID:              withExecutorFreshness(reportID(Report{RootLibraryID: root.LibraryID, RootVersion: "1.0.0", IncludeClosure: true, SourceRevision: "folded-revision-1"}), freshness),
		RootComponentID: root.ID, RootLibraryID: root.LibraryID, RootVersion: "1.0.0", IncludeClosure: true,
		SourceRevision: "folded-revision-1", Verdict: VerdictFailed,
		Results: []Result{{Stage: StageEvidence, Verdict: VerdictFailed, Message: "stale executor evidence"}},
	}
	require.NoError(t, reports.Save(context.Background(), old))

	freshness = "executor-policy-2"
	service := NewService(runner, reports)
	fresh, reused, err := service.RunWithReuse(context.Background(), Request{ComponentID: root.ID, Version: "1.0.0", IncludeClosure: true})
	require.NoError(t, err)
	require.False(t, reused, "a failed report from an earlier executor policy cannot stand as current evidence")
	require.Equal(t, 1, runs)
	require.Equal(t, VerdictPassed, fresh.Verdict)
	require.Equal(t, root.LibraryID, fresh.RootLibraryID)
	require.Equal(t, "1.0.0", fresh.RootVersion)
	require.True(t, fresh.IncludeClosure)
	require.Equal(t, "folded-revision-1", fresh.SourceRevision)
	require.NotEqual(t, old.ID, fresh.ID)

	_, reused, err = service.RunWithReuse(context.Background(), Request{ComponentID: root.ID, Version: "1.0.0", IncludeClosure: true})
	require.NoError(t, err)
	require.True(t, reused, "the same executor policy and exact source/closure identity remain reusable")
	require.Equal(t, 1, runs)
}

func TestServiceRerunReportIsReusableWithCurrentExecutorFreshness(t *testing.T) {
	database := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(components.Schema)))
	root := components.Component{ID: "root", CatalogID: "controls.root", LibraryID: "rcl:root", Slug: "root", AssetKind: components.AssetKindComponent}
	reader := assets{root: root, versions: map[string]components.ComponentVersion{"root@1.0.0": {ComponentID: root.ID, Version: "1.0.0", Content: "export const Root = () => null", ContentSHA256: "root"}}}
	freshness := "executor-policy-1"
	runs := 0
	runner := Runner{
		Assets: reader, Stories: stories{"root@1.0.0": componentStory("default")},
		Executor: changingFreshnessExecutor{runs: &runs, freshness: &freshness},
		Revision: func(context.Context, string, string) (string, error) { return "folded-revision-1", nil },
	}
	service := NewService(runner, NewSQLiteRepository(database))
	request := Request{ComponentID: root.ID, Version: "1.0.0", IncludeClosure: true}

	original, reused, err := service.RunWithReuse(context.Background(), request)
	require.NoError(t, err)
	require.False(t, reused)
	require.Equal(t, 1, runs)

	rerun, err := service.Rerun(context.Background(), original.ID)
	require.NoError(t, err)
	require.Equal(t, 2, runs, "explicit rerun must execute the story again")
	require.Equal(t, original.ID, rerun.ID, "rerun output must retain the same source, root, version, closure, and executor identity")
	require.Equal(t, root.ID, rerun.RootComponentID)
	require.Equal(t, root.LibraryID, rerun.RootLibraryID)
	require.Equal(t, "1.0.0", rerun.RootVersion)
	require.True(t, rerun.IncludeClosure)
	require.Equal(t, "folded-revision-1", rerun.SourceRevision)

	stored, err := service.Get(context.Background(), rerun.ID)
	require.NoError(t, err)
	require.Equal(t, rerun.ID, stored.ID)
	require.Equal(t, root.ID, stored.RootComponentID)
	require.Equal(t, root.LibraryID, stored.RootLibraryID)
	require.Equal(t, "1.0.0", stored.RootVersion)
	require.True(t, stored.IncludeClosure)
	require.Equal(t, "folded-revision-1", stored.SourceRevision)

	reusedReport, reused, err := service.RunWithReuse(context.Background(), request)
	require.NoError(t, err)
	require.True(t, reused, "rerun output with the current executor key and unchanged source binding must be reusable")
	require.Equal(t, rerun.ID, reusedReport.ID)
	require.Equal(t, root.ID, reusedReport.RootComponentID)
	require.Equal(t, root.LibraryID, reusedReport.RootLibraryID)
	require.Equal(t, "1.0.0", reusedReport.RootVersion)
	require.True(t, reusedReport.IncludeClosure)
	require.Equal(t, "folded-revision-1", reusedReport.SourceRevision)
	require.Equal(t, 2, runs, "reuse after rerun must not execute the story again")
}
