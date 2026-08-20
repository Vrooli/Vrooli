package componenttests

import (
	"context"
	"testing"
	"time"

	internalcomponents "react-component-library/internal/components"
	localdb "react-component-library/internal/database"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	db "github.com/vrooli/api-core/databasetest"
)

func TestSQLiteRepositoryPersistsNormalizedReportsNewestFirst(t *testing.T) {
	database := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(internalcomponents.Schema)))
	repository := NewSQLiteRepository(database)
	older := Report{ID: "ctr_old", RootComponentID: "button", RootLibraryID: "rcl:Button", RootVersion: "1.0.0", CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Verdict: VerdictPassed, Results: []Result{{Stage: StageClosure, Verdict: VerdictPassed}}}
	newer := older
	newer.ID = "ctr_new"
	newer.CreatedAt = older.CreatedAt.Add(time.Minute)
	newer.Verdict = VerdictFailed
	newer.Results = []Result{{Stage: StageContract, Verdict: VerdictFailed, Remediation: "fix contract"}}
	newer.Artifacts = []Artifact{{Kind: "story-contract", Label: "Button story", AssetLibraryID: "rcl:Button", Version: "1.0.0", Reference: "rcl:Button@1.0.0:story.json"}}
	require.NoError(t, repository.Save(context.Background(), older))
	require.NoError(t, repository.Save(context.Background(), newer))
	got, err := repository.Get(context.Background(), "ctr_new")
	require.NoError(t, err)
	require.Equal(t, VerdictFailed, got.Verdict)
	require.Equal(t, "fix contract", got.Results[0].Remediation)
	require.Equal(t, newer.Artifacts, got.Artifacts)
	list, err := repository.List(context.Background(), "button", "", 10)
	require.NoError(t, err)
	require.Equal(t, []string{"ctr_new", "ctr_old"}, []string{list[0].ID, list[1].ID})
}

func TestSQLiteRepositoryReadsLegacyResultsArray(t *testing.T) {
	database := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(internalcomponents.Schema)))
	_, err := database.ExecContext(context.Background(), `INSERT INTO component_test_reports (id, component_id, root_library_id, root_version, include_closure, created_at, verdict, results_json) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"ctr_legacy", "button", "rcl:Button", "1.0.0", false, "2026-01-01T00:00:00Z", VerdictPassed, `[{"stage":"contract_validation","verdict":"passed"}]`)
	require.NoError(t, err)

	reports, err := NewSQLiteRepository(database).List(context.Background(), "button", "", 10)
	require.NoError(t, err)
	require.Len(t, reports, 1)
	require.Equal(t, StageContract, reports[0].Results[0].Stage)
	require.Empty(t, reports[0].Artifacts)
}
