package validation

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"

	testdb "plan-manager/internal/testutil/db"
	"plan-manager/internal/testutil/mocks"
)

func newOperationStore(t *testing.T) *sqliteResultStore {
	t.Helper()
	db := testdb.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(Schema)))
	return NewSQLiteResultStore(db, mocks.NewFakeClock(time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)))
}

func v2Operation(op ValidationOperation) ValidationOperation {
	op.SchemaVersion = CurrentOperationSchemaVersion
	for i := range op.Children {
		if op.Children[i].Check.SemanticKey == "" {
			op.Children[i].Check = ValidationCheck{
				Kind:        ValidationCheckCustom,
				SemanticKey: "test:" + op.Children[i].ID,
				Command:     op.Children[i].Command,
				Oracle:      op.Children[i].Oracle,
			}
		}
	}
	return op
}

func TestSQLiteOperationStoreConcurrentIdempotency(t *testing.T) { // [REQ:PM-VALID-004]
	store := newOperationStore(t)
	const count = 16
	ids := make(chan string, count)
	created := make(chan bool, count)
	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			op, wasCreated, err := store.CreateOperation(context.Background(), v2Operation(ValidationOperation{
				ID: "candidate-" + time.Duration(i).String(), PlanID: "plan-1", PhaseID: "phase-1",
				IdempotencyKey: "same", Status: OperationQueued, QueuedAt: "2026-07-10T12:00:00Z",
			}))
			require.NoError(t, err)
			ids <- op.ID
			created <- wasCreated
		}(i)
	}
	wg.Wait()
	close(ids)
	close(created)
	var first string
	for id := range ids {
		if first == "" {
			first = id
		}
		require.Equal(t, first, id)
	}
	winners := 0
	for value := range created {
		if value {
			winners++
		}
	}
	require.Equal(t, 1, winners)
}

func TestSQLiteOperationStoreCoalescesUnkeyedActiveStarts(t *testing.T) { // [REQ:PM-VALID-004]
	store := newOperationStore(t)
	first, created, err := store.CreateOperation(context.Background(), v2Operation(ValidationOperation{ID: "first", PlanID: "plan-1", PhaseID: "phase-1", Status: OperationQueued, QueuedAt: "2026-07-10T12:00:00Z"}))
	require.NoError(t, err)
	require.True(t, created)
	second, created, err := store.CreateOperation(context.Background(), v2Operation(ValidationOperation{ID: "second", PlanID: "plan-1", PhaseID: "phase-1", Status: OperationQueued, QueuedAt: "2026-07-10T12:00:01Z"}))
	require.NoError(t, err)
	require.False(t, created)
	require.Equal(t, first.ID, second.ID)
}

func TestSQLiteOperationStoreKeepsExplicitRequestsDistinct(t *testing.T) { // [REQ:PM-VALID-004]
	store := newOperationStore(t)
	first, created, err := store.CreateOperation(context.Background(), v2Operation(ValidationOperation{
		ID: "first", PlanID: "plan-1", PhaseID: "phase-1", ExecutionID: "execution-1", ScopeGeneration: 3,
		IdempotencyKey: "before-code-change", Status: OperationQueued, QueuedAt: "2026-07-10T12:00:00Z",
	}))
	require.NoError(t, err)
	require.True(t, created)

	second, created, err := store.CreateOperation(context.Background(), v2Operation(ValidationOperation{
		ID: "second", PlanID: "plan-1", PhaseID: "phase-1", ExecutionID: "execution-1", ScopeGeneration: 3,
		IdempotencyKey: "after-code-change", Status: OperationQueued, QueuedAt: "2026-07-10T12:00:01Z",
	}))
	require.NoError(t, err)
	require.True(t, created)
	require.NotEqual(t, first.ID, second.ID, "a distinct explicit request must create fresh evidence")

	replay, created, err := store.CreateOperation(context.Background(), v2Operation(ValidationOperation{
		ID: "replay", PlanID: "plan-1", PhaseID: "phase-1", ExecutionID: "execution-1", ScopeGeneration: 3,
		IdempotencyKey: "after-code-change", Status: OperationQueued, QueuedAt: "2026-07-10T12:00:02Z",
	}))
	require.NoError(t, err)
	require.False(t, created)
	require.Equal(t, second.ID, replay.ID)
}

func TestSQLiteOperationStoreScopesRetriesToExecutionGeneration(t *testing.T) { // [REQ:PM-VALID-004]
	store := newOperationStore(t)
	first, created, err := store.CreateOperation(context.Background(), v2Operation(ValidationOperation{
		ID: "first", PlanID: "plan-1", PhaseID: "phase-1", ExecutionID: "execution-1", ScopeGeneration: 1,
		IdempotencyKey: "phase-validation", Status: OperationQueued, QueuedAt: "2026-07-10T12:00:00Z",
	}))
	require.NoError(t, err)
	require.True(t, created)

	for _, candidate := range []ValidationOperation{
		{ID: "next-generation", PlanID: "plan-1", PhaseID: "phase-1", ExecutionID: "execution-1", ScopeGeneration: 2, IdempotencyKey: "phase-validation", Status: OperationQueued, QueuedAt: "2026-07-10T12:00:01Z"},
		{ID: "other-execution", PlanID: "plan-1", PhaseID: "phase-1", ExecutionID: "execution-2", ScopeGeneration: 1, IdempotencyKey: "phase-validation", Status: OperationQueued, QueuedAt: "2026-07-10T12:00:02Z"},
	} {
		got, candidateCreated, candidateErr := store.CreateOperation(context.Background(), v2Operation(candidate))
		require.NoError(t, candidateErr)
		require.True(t, candidateCreated)
		require.NotEqual(t, first.ID, got.ID)
	}
}

func TestSQLiteOperationStoreRoundTripsPartialChildCheckpoint(t *testing.T) { // [REQ:PM-VALID-004]
	store := newOperationStore(t)
	op := ValidationOperation{
		ID: "op-1", PlanID: "plan-1", Status: OperationRunning, Attempt: 1,
		QueuedAt: "2026-07-10T12:00:00Z", StartedAt: "2026-07-10T12:00:01Z",
		Children: []ValidationChild{
			{ID: "op-1:1", Command: "oracle", Oracle: true, Status: ChildTerminal, Verdict: VerdictPass, ExternalID: "run-1"},
			{ID: "op-1:2", Command: "informational", Status: ChildRunning, Attempt: 1},
		},
	}
	_, created, err := store.CreateOperation(context.Background(), v2Operation(op))
	require.NoError(t, err)
	require.True(t, created)
	got, found, err := store.GetOperation(context.Background(), op.ID)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "run-1", got.Children[0].ExternalID)
	require.Equal(t, ChildRunning, got.Children[1].Status)
	pending, err := store.ListNonTerminalOperations(context.Background())
	require.NoError(t, err)
	require.Len(t, pending, 1)

	got.Status = OperationTerminal
	got.TerminalAt = "2026-07-10T12:01:00Z"
	require.NoError(t, store.SaveOperation(context.Background(), got))
	pending, err = store.ListNonTerminalOperations(context.Background())
	require.NoError(t, err)
	require.Empty(t, pending)
}

func TestSQLiteTerminalResultStableIDIsReplaySafe(t *testing.T) { // [REQ:PM-VALID-004]
	store := newOperationStore(t)
	result := Result{ID: "op-1:result", PlanID: "plan-1", Verdict: VerdictPass, RanAt: "2026-07-10T12:00:00Z"}
	require.NoError(t, store.SaveResult(context.Background(), result))
	replay := result
	replay.Verdict = VerdictUnknown
	replay.Detail = "must not replace the committed terminal result"
	require.NoError(t, store.SaveResult(context.Background(), replay))
	got, found, err := store.LastResult(context.Background(), "plan-1", "")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, result.ID, got.ID)
	require.Equal(t, VerdictPass, got.Verdict)
	require.Empty(t, got.Detail)
}

// [REQ:PM-VALID-004] A terminal result is the execution gate's durable
// receipt. Every binding field must survive a repository round trip; otherwise
// a restart turns valid evidence into evidence for an "unknown" execution.
func TestSQLiteResultStoreRoundTripsExecutionBinding(t *testing.T) {
	store := newOperationStore(t)
	want := Result{
		ID: "op-1:result", PlanID: "plan-1", PhaseID: "phase-1", Verdict: VerdictPass,
		RanAt: "2026-07-10T12:00:00Z", ExecutionID: "execution-1", OperationID: "op-1",
		ScopeGeneration: 3, FullInventory: true,
		RequiredMembers: []string{"plan-manager", "react-component-library"},
		SelectedMembers: []string{"plan-manager", "react-component-library"},
	}
	require.NoError(t, store.SaveResult(context.Background(), want))

	got, found, err := store.LastResult(context.Background(), want.PlanID, want.PhaseID)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, want.ExecutionID, got.ExecutionID)
	require.Equal(t, want.OperationID, got.OperationID)
	require.Equal(t, want.ScopeGeneration, got.ScopeGeneration)
	require.Equal(t, want.FullInventory, got.FullInventory)
	require.Equal(t, want.RequiredMembers, got.RequiredMembers)
	require.Equal(t, want.SelectedMembers, got.SelectedMembers)
}

func TestEnsureMigrationsUpgradesLegacyResultsWithoutQualifyingThem(t *testing.T) { // [REQ:PM-VALID-004]
	db := testdb.NewSQLite(t)
	_, err := db.Exec(`
CREATE TABLE validation_results (
  id TEXT PRIMARY KEY, plan_id TEXT NOT NULL, phase_id TEXT NOT NULL DEFAULT '',
  verdict TEXT NOT NULL DEFAULT '', staleness TEXT NOT NULL DEFAULT '',
  commands_run TEXT NOT NULL DEFAULT '[]', detail TEXT NOT NULL DEFAULT '', ran_at TEXT NOT NULL
);
INSERT INTO validation_results (id, plan_id, phase_id, verdict, ran_at)
VALUES ('legacy-result', 'plan-1', 'phase-1', 'pass', '2026-07-10T12:00:00Z');`)
	require.NoError(t, err)

	require.NoError(t, EnsureMigrations(context.Background(), db))
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(Schema)))

	store := NewSQLiteResultStore(db, mocks.NewFakeClock(time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)))
	legacy, found, err := store.GetResult(context.Background(), "legacy-result")
	require.NoError(t, err)
	require.True(t, found)
	require.Empty(t, legacy.ExecutionID, "legacy evidence must require a fresh validation")
	require.Zero(t, legacy.ScopeGeneration)
	require.False(t, legacy.FullInventory)

	// The migration is idempotent and a newly saved receipt preserves its proof.
	require.NoError(t, EnsureMigrations(context.Background(), db))
	require.NoError(t, store.SaveResult(context.Background(), Result{
		ID: "new-result", PlanID: "plan-1", PhaseID: "phase-1", Verdict: VerdictPass,
		RanAt: "2026-07-10T12:01:00Z", ExecutionID: "execution-1", ScopeGeneration: 1,
		FullInventory: true, RequiredMembers: []string{"plan-manager"}, SelectedMembers: []string{"plan-manager"},
	}))
	current, found, err := store.GetResult(context.Background(), "new-result")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "execution-1", current.ExecutionID)
	require.Equal(t, []string{"plan-manager"}, current.SelectedMembers)
}

func TestSQLiteOperationReadRejectsLegacyCommandChild(t *testing.T) { // [REQ:PM-VALID-004]
	store := newOperationStore(t)
	op := ValidationOperation{ID: "legacy", PlanID: "plan-1", Status: OperationQueued, QueuedAt: "2026-07-10T12:00:00Z", Children: []ValidationChild{{ID: "legacy:1", Command: "git-control-tower baseline diff --scenario foo --name impl --wait", Oracle: true, Status: ChildQueued}}}
	payload, err := json.Marshal(op)
	require.NoError(t, err)
	_, err = store.db.ExecContext(context.Background(), insertOperationSQL, op.ID, op.PlanID, op.PhaseID, op.IdempotencyKey, string(op.Status), op.QueuedAt, op.QueuedAt, string(payload))
	require.NoError(t, err)
	_, found, err := store.GetOperation(context.Background(), op.ID)
	require.False(t, found)
	require.ErrorContains(t, err, "uses storage schema v0")
}
