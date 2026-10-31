package validationrun

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	db "github.com/vrooli/api-core/databasetest"
	core "github.com/vrooli/api-core/validationrun"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

func TestRepositoryPersistsReplaySafeRunAndTerminalEvidence(t *testing.T) {
	database := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(Schema)))
	repo := Repository{DB: database}
	created := Record{Run: core.Run{ID: "run-1", Target: core.Target{Scenario: "demo", Path: "/tmp/demo"}, IdempotencyKey: "request-1", ParentRunID: "parent-1", State: core.StateQueued, CreatedAt: time.Now().UTC(), Version: 1}, ETA: time.Minute, ExecutionBinding: &scenariovalidationv1.DesktopValidationBinding{TargetId: "target-1", CdpEndpoint: "http://127.0.0.1:9222", RendererId: "renderer-1", ContextId: "ctx-1", ProfileId: "normal"}}
	require.NoError(t, repo.Create(context.Background(), created))
	found, err := repo.FindByIdempotency(context.Background(), "request-1")
	require.NoError(t, err)
	require.Equal(t, created.Run.ID, found.Run.ID)
	require.Equal(t, created.Run.ParentRunID, found.Run.ParentRunID)
	require.Equal(t, "target-1", found.ExecutionBinding.GetTargetId())
	next, err := core.Transition(found.Run, core.EventClaim, time.Now().UTC())
	require.NoError(t, err)
	next.Version = found.Run.Version + 1
	found.Run = next
	require.NoError(t, repo.Update(context.Background(), found, 1))
	require.Error(t, repo.Update(context.Background(), found, 1), "stale commits must not overwrite recovery state")
}

func TestRepositoryLifecycleFencePersistsAcrossReopenAndRejectsStaleRevision(t *testing.T) {
	database := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(Schema)))
	repo := Repository{DB: database}

	open, err := repo.GetLifecycleFence(context.Background())
	require.NoError(t, err)
	require.Equal(t, LifecycleFence{Revision: 0}, open)
	closed := LifecycleFence{Closed: true, Operation: "restart-1", Revision: 1}
	require.NoError(t, repo.UpdateLifecycleFence(context.Background(), open.Revision, closed))

	reopened := Repository{DB: database}
	retained, err := reopened.GetLifecycleFence(context.Background())
	require.NoError(t, err)
	require.Equal(t, closed, retained)
	require.ErrorContains(t, reopened.UpdateLifecycleFence(context.Background(), 0, LifecycleFence{Revision: 1}), "stale lifecycle fence")

	resumed := LifecycleFence{Revision: 2}
	require.NoError(t, reopened.UpdateLifecycleFence(context.Background(), retained.Revision, resumed))
	got, err := repo.GetLifecycleFence(context.Background())
	require.NoError(t, err)
	require.Equal(t, resumed, got)
}

func TestRepositoryListsOnlyQueuedAndRunningWorkForDrain(t *testing.T) {
	database := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(Schema)))
	repo := Repository{DB: database}
	for _, record := range []Record{
		{Run: core.Run{ID: "queued", Target: core.Target{Scenario: "demo"}, IdempotencyKey: "queued", State: core.StateQueued, CreatedAt: time.Now().UTC(), Version: 1}},
		{Run: core.Run{ID: "running", Target: core.Target{Scenario: "demo"}, IdempotencyKey: "running", State: core.StateQueued, CreatedAt: time.Now().UTC(), Version: 1}},
		{Run: core.Run{ID: "done", Target: core.Target{Scenario: "demo"}, IdempotencyKey: "done", State: core.StateQueued, CreatedAt: time.Now().UTC(), Version: 1}},
	} {
		require.NoError(t, repo.Create(context.Background(), record))
	}
	for _, id := range []string{"running", "done"} {
		record, err := repo.Get(context.Background(), id)
		require.NoError(t, err)
		next, err := core.Transition(record.Run, core.EventClaim, time.Now().UTC())
		require.NoError(t, err)
		next.Version++
		record.Run = next
		require.NoError(t, repo.Update(context.Background(), record, 1))
	}
	done, err := repo.Get(context.Background(), "done")
	require.NoError(t, err)
	terminal, err := core.Transition(done.Run, core.EventSucceed, time.Now().UTC())
	require.NoError(t, err)
	terminal.Version++
	done.Run = terminal
	require.NoError(t, repo.Update(context.Background(), done, 2))

	active, err := repo.ListActive(context.Background())
	require.NoError(t, err)
	require.Equal(t, []ActiveRun{{ID: "queued", State: core.StateQueued}, {ID: "running", State: core.StateRunning}}, active)
}

func TestRepositoryFreezesBrowserSelectionAcrossCASAndReopen(t *testing.T) {
	database := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(Schema)))
	repo := Repository{DB: database}
	record := Record{Run: core.Run{ID: "selected", Target: core.Target{Scenario: "sample"}, IdempotencyKey: "selected-key", State: core.StateQueued, CreatedAt: time.Now().UTC(), Version: 1}, BrowserCasePaths: []string{"bas/cases/calendar/stale-revision-conflict.json", " ./bas/cases/calendar/duplicate-submit-once.json ", "bas/cases/calendar/uncertain-save-recovery.json"}}
	require.NoError(t, repo.Create(context.Background(), record))
	reopened := Repository{DB: database}
	found, err := reopened.FindByIdempotency(context.Background(), "selected-key")
	require.NoError(t, err)
	require.Equal(t, []string{"bas/cases/calendar/duplicate-submit-once.json", "bas/cases/calendar/stale-revision-conflict.json", "bas/cases/calendar/uncertain-save-recovery.json"}, found.BrowserCasePaths)
	found.Run.State = core.StateRunning
	found.Run.Version = 2
	require.NoError(t, reopened.Update(context.Background(), found, 1))
	found.BrowserCasePaths = []string{"bas/cases/calendar/duplicate-submit-once.json"}
	found.Run.Version = 3
	require.ErrorContains(t, reopened.Update(context.Background(), found, 2), "immutable")
	retained, err := reopened.Get(context.Background(), "selected")
	require.NoError(t, err)
	require.Len(t, retained.BrowserCasePaths, 3)
	require.Equal(t, int64(2), retained.Run.Version)
	interrupted, err := reopened.ListInterrupted(context.Background())
	require.NoError(t, err)
	require.Len(t, interrupted, 1)
	require.Equal(t, retained.BrowserCasePaths, interrupted[0].BrowserCasePaths)
}

func TestRepositoryOldRowsRemainDefaultAndEmptySelectionNeverWrites(t *testing.T) {
	database := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(Schema)))
	repo := Repository{DB: database}
	record := Record{Run: core.Run{ID: "old", Target: core.Target{Scenario: "sample"}, IdempotencyKey: "old-key", State: core.StateQueued, CreatedAt: time.Now().UTC(), Version: 1}}
	require.NoError(t, repo.Create(context.Background(), record))
	found, err := repo.Get(context.Background(), "old")
	require.NoError(t, err)
	require.Nil(t, found.BrowserCasePaths)
	record.Run.ID = "empty"
	record.Run.IdempotencyKey = "empty-key"
	record.BrowserCasePaths = []string{}
	require.Error(t, repo.Create(context.Background(), record))
	_, err = repo.FindByIdempotency(context.Background(), "empty-key")
	require.True(t, core.IsCode(err, core.ErrorNotFound))
}

func TestRepositoryPreSelectorSchemaMigratesAbsentRowsCompatibly(t *testing.T) {
	database := db.NewSQLite(t)
	oldSchema := strings.Replace(Schema(), "  browser_case_paths BLOB,\n", "", 1)
	_, err := database.ExecContext(context.Background(), oldSchema)
	require.NoError(t, err)
	_, err = database.ExecContext(context.Background(), `INSERT INTO validation_runs (id,scenario,target_path,idempotency_key,state,created_at,preliminary_result) VALUES (?,?,?,?,?,?,?)`, "pre-selector", "sample", "", "before-selector", "queued", stamp(time.Now().UTC()), []byte{})
	require.NoError(t, err)
	// Only the disposable fixture is migrated. This does not migrate live storage.
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(Schema)))
	found, err := (Repository{DB: database}).Get(context.Background(), "pre-selector")
	require.NoError(t, err)
	require.Nil(t, found.BrowserCasePaths)
}
