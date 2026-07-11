package validation

import (
	"context"
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
			op, wasCreated, err := store.CreateOperation(context.Background(), ValidationOperation{
				ID: "candidate-" + time.Duration(i).String(), PlanID: "plan-1", PhaseID: "phase-1",
				IdempotencyKey: "same", Status: OperationQueued, QueuedAt: "2026-07-10T12:00:00Z",
			})
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
	first, created, err := store.CreateOperation(context.Background(), ValidationOperation{ID: "first", PlanID: "plan-1", PhaseID: "phase-1", Status: OperationQueued, QueuedAt: "2026-07-10T12:00:00Z"})
	require.NoError(t, err)
	require.True(t, created)
	second, created, err := store.CreateOperation(context.Background(), ValidationOperation{ID: "second", PlanID: "plan-1", PhaseID: "phase-1", Status: OperationQueued, QueuedAt: "2026-07-10T12:00:01Z"})
	require.NoError(t, err)
	require.False(t, created)
	require.Equal(t, first.ID, second.ID)
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
	_, created, err := store.CreateOperation(context.Background(), op)
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

func TestSQLiteOperationReadMigratesLegacyCommandChild(t *testing.T) { // [REQ:PM-VALID-004]
	store := newOperationStore(t)
	op := ValidationOperation{ID: "legacy", PlanID: "plan-1", Status: OperationQueued, QueuedAt: "2026-07-10T12:00:00Z", Children: []ValidationChild{{ID: "legacy:1", Command: "git-control-tower baseline diff --scenario foo --name impl --wait", Oracle: true, Status: ChildQueued}}}
	_, created, err := store.CreateOperation(context.Background(), op)
	require.NoError(t, err)
	require.True(t, created)
	got, found, err := store.GetOperation(context.Background(), op.ID)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, CurrentOperationSchemaVersion, got.SchemaVersion)
	require.Equal(t, "scenario-diff:foo:impl", got.Children[0].Check.SemanticKey)
}
