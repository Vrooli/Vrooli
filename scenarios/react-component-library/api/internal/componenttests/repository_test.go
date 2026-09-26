package componenttests

import (
	"context"
	"database/sql"
	"strings"
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

func TestSQLiteRepositoryPayloadCeilingEvictsOldUnpinnedEvidence(t *testing.T) {
	database := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(internalcomponents.Schema)))
	large := strings.Repeat("x", 20)
	_, err := database.ExecContext(context.Background(), `
INSERT INTO component_test_reports (id, component_id, root_library_id, root_version, include_closure, created_at, verdict, results_json) VALUES
  ('old', 'button', 'rcl:Button', '1.0.0', 0, '2026-01-01T00:00:00Z', 'passed', ?),
  ('pinned', 'button', 'rcl:Button', '1.0.0', 0, '2026-01-02T00:00:00Z', 'passed', ?),
  ('new', 'button', 'rcl:Button', '1.0.0', 0, '2026-01-03T00:00:00Z', 'passed', ?);
INSERT INTO component_version_test_rollup (library_id, version, first_pass_report_id) VALUES ('rcl:Button', '1.0.0', 'pinned');
`, large, large, large)
	require.NoError(t, err)
	tx, err := database.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	require.NoError(t, enforceReportPayloadCeilingWithLimit(context.Background(), tx, 45))
	require.NoError(t, tx.Commit())
	var count int
	require.NoError(t, database.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM component_test_reports`).Scan(&count))
	require.Equal(t, 2, count)
	var oldCount, pinnedCount int
	require.NoError(t, database.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM component_test_reports WHERE id='old'`).Scan(&oldCount))
	require.NoError(t, database.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM component_test_reports WHERE id='pinned'`).Scan(&pinnedCount))
	require.Zero(t, oldCount)
	require.Equal(t, 1, pinnedCount)
}

func TestSQLiteSweepRepositoryResumesDurableResults(t *testing.T) {
	database := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(internalcomponents.Schema)))
	store := NewSQLiteSweepRepository(database)
	sweep, err := store.Start(context.Background(), "component-id", false, "sweep-test")
	require.NoError(t, err)
	sweep.Results["rcl:Button@1.0.0"] = string(VerdictPassed)
	require.NoError(t, store.Save(context.Background(), sweep))
	resumed, err := store.LatestOpen(context.Background(), "component-id")
	require.NoError(t, err)
	require.Equal(t, "sweep-test", resumed.ID)
	require.Equal(t, string(VerdictPassed), resumed.Results["rcl:Button@1.0.0"])
	resumed.Status = SweepComplete
	resumed.CompletedAt = time.Now().UTC()
	require.NoError(t, store.Save(context.Background(), resumed))
	_, err = store.LatestOpen(context.Background(), "component-id")
	require.ErrorIs(t, err, sql.ErrNoRows)
}
