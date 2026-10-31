package validation

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	db "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/storage"
	core "github.com/vrooli/api-core/validationrun"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	"google.golang.org/protobuf/types/known/durationpb"
	workflowrun "workflow-health/internal/validationrun"
)

func TestLifecycleFenceDrainsAcceptedRunsAndResumesExactRevisionAfterReplacement(t *testing.T) {
	database := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(workflowrun.Schema)))
	repo := workflowrun.Repository{DB: database}
	require.NoError(t, repo.Create(context.Background(), workflowrun.Record{Run: core.Run{
		ID: "accepted-run", Target: core.Target{Scenario: "demo", Path: filepath.Join(t.TempDir(), "demo")},
		IdempotencyKey: "accepted-key", State: core.StateQueued, CreatedAt: time.Now().UTC(), Version: 1,
	}}))
	handler := testHandler(t)
	handler.deps.Ledger = repo

	prepared, err := handler.Prepare(context.Background(), connect.NewRequest(&commonv1.LifecyclePrepareRequest{
		OperationId: "restart-operation", Reason: "managed restart",
	}))
	require.NoError(t, err)
	standing := prepared.Msg.GetStanding()
	require.True(t, standing.GetAdmissionClosed())
	require.False(t, standing.GetDrained())
	require.Equal(t, int32(1), standing.GetRemaining())
	require.Equal(t, "primary/accepted-run", standing.GetWork()[0].GetId())

	// Reattachment remains available while closed, while a new reservation is refused.
	replay, err := handler.StartValidationRun(context.Background(), connect.NewRequest(&scenariovalidationv1.StartValidationRunRequest{
		Scenario: "demo", Path: filepath.Join(t.TempDir(), "different-path"), IdempotencyKey: "accepted-key",
	}))
	require.Error(t, err, "same key with a different immutable target must remain a conflict")
	require.Nil(t, replay)
	replay, err = handler.StartValidationRun(context.Background(), connect.NewRequest(&scenariovalidationv1.StartValidationRunRequest{
		Scenario: "demo", Path: repoTargetPath(t, repo, "accepted-run"), IdempotencyKey: "accepted-key",
	}))
	require.NoError(t, err)
	require.Equal(t, "accepted-run", replay.Msg.GetRun().GetRunId())
	_, err = handler.StartValidationRun(context.Background(), connect.NewRequest(&scenariovalidationv1.StartValidationRunRequest{
		Scenario: "demo", Path: filepath.Join(t.TempDir(), "new-target"), IdempotencyKey: "new-key",
	}))
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))

	_, err = handler.Drain(context.Background(), connect.NewRequest(&commonv1.LifecycleDrainRequest{
		OperationId: standing.GetOperationId(), FenceToken: standing.GetFenceToken(), Revision: standing.GetRevision(), Timeout: durationpb.New(20 * time.Millisecond),
	}))
	require.Equal(t, connect.CodeDeadlineExceeded, connect.CodeOf(err))
	stillClosed, err := repo.GetLifecycleFence(context.Background())
	require.NoError(t, err)
	require.True(t, stillClosed.Closed, "a timed-out observation must leave maintenance closed")

	active, err := repo.Get(context.Background(), "accepted-run")
	require.NoError(t, err)
	claimed, err := core.Transition(active.Run, core.EventClaim, time.Now().UTC())
	require.NoError(t, err)
	claimed.Version++
	active.Run = claimed
	require.NoError(t, repo.Update(context.Background(), active, 1))
	terminal, err := core.Transition(active.Run, core.EventSucceed, time.Now().UTC())
	require.NoError(t, err)
	terminal.Version++
	active.Run = terminal
	require.NoError(t, repo.Update(context.Background(), active, 2))
	handler.signalLifecycleChange()

	drained, err := handler.Drain(context.Background(), connect.NewRequest(&commonv1.LifecycleDrainRequest{
		OperationId: standing.GetOperationId(), FenceToken: standing.GetFenceToken(), Revision: standing.GetRevision(),
	}))
	require.NoError(t, err)
	require.True(t, drained.Msg.GetStanding().GetDrained())

	// A replacement process starts against the same durable row and accepts only
	// the exact operation/revision held by the lifecycle owner.
	replacement := testHandler(t)
	replacement.deps.Ledger = repo
	_, err = replacement.Resume(context.Background(), connect.NewRequest(&commonv1.LifecycleResumeRequest{
		OperationId: standing.GetOperationId(), FenceToken: standing.GetFenceToken(), Revision: standing.GetRevision() + 1,
	}))
	require.Equal(t, connect.CodeAborted, connect.CodeOf(err))
	open, err := replacement.Resume(context.Background(), connect.NewRequest(&commonv1.LifecycleResumeRequest{
		OperationId: standing.GetOperationId(), FenceToken: standing.GetFenceToken(), Revision: standing.GetRevision(),
	}))
	require.NoError(t, err)
	require.Equal(t, commonv1.LifecycleMaintenancePhase_LIFECYCLE_MAINTENANCE_PHASE_RESUMED, open.Msg.GetStanding().GetPhase())
	require.False(t, open.Msg.GetStanding().GetAdmissionClosed())
}

func TestLifecyclePrepareLinearizesWithInFlightAdmission(t *testing.T) {
	database := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(workflowrun.Schema)))
	handler := testHandler(t)
	handler.deps.Ledger = workflowrun.Repository{DB: database}
	release, err := handler.admitValidation(context.Background())
	require.NoError(t, err)

	prepared, err := handler.Prepare(context.Background(), connect.NewRequest(&commonv1.LifecyclePrepareRequest{OperationId: "race-operation", Reason: "test drain"}))
	require.NoError(t, err)
	require.Equal(t, int32(1), prepared.Msg.GetStanding().GetAdmitting())
	require.False(t, prepared.Msg.GetStanding().GetDrained())
	if _, err := handler.admitValidation(context.Background()); !errors.Is(err, errValidationAdmissionClosed) {
		t.Fatalf("new admission error = %v, want closed-fence refusal", err)
	}
	release()

	drained, err := handler.Drain(context.Background(), connect.NewRequest(&commonv1.LifecycleDrainRequest{
		OperationId: prepared.Msg.GetStanding().GetOperationId(), FenceToken: prepared.Msg.GetStanding().GetFenceToken(), Revision: prepared.Msg.GetStanding().GetRevision(),
	}))
	require.NoError(t, err)
	require.True(t, drained.Msg.GetStanding().GetDrained())
}

func TestLifecyclePrepareFailureRollsBackOnlyAFenceItCreated(t *testing.T) {
	database := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(workflowrun.Schema)))
	handler := testHandler(t)
	handler.deps.Ledger = workflowrun.Repository{DB: failingActiveInventoryDB{DB: database}}

	_, err := handler.Prepare(context.Background(), connect.NewRequest(&commonv1.LifecyclePrepareRequest{
		OperationId: "failed-prepare", Reason: "inventory failure",
	}))
	require.ErrorContains(t, err, "observe validation work")
	fence, err := (workflowrun.Repository{DB: database}).GetLifecycleFence(context.Background())
	require.NoError(t, err)
	require.Equal(t, workflowrun.LifecycleFence{Revision: 2}, fence, "a failed prepare must not strand a newly closed fence")

	// A human-authorized emergency prepare from an open fence also rolls back
	// when it cannot establish inventory. Emergency override may skip drain, not
	// the inventory validity required to safely claim the provider state.
	_, err = handler.Prepare(context.Background(), connect.NewRequest(&commonv1.LifecyclePrepareRequest{
		OperationId: "failed-emergency", Reason: "emergency replacement", EmergencyOverride: true, OverrideReason: "operator decision",
	}))
	require.ErrorContains(t, err, "observe validation work")
	fence, err = (workflowrun.Repository{DB: database}).GetLifecycleFence(context.Background())
	require.NoError(t, err)
	require.Equal(t, workflowrun.LifecycleFence{Revision: 4}, fence)
}

func TestLifecycleEmergencyAdoptsClosedFenceWithoutClaimingDrain(t *testing.T) {
	database := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(workflowrun.Schema)))
	repo := workflowrun.Repository{DB: database}
	require.NoError(t, repo.Create(context.Background(), workflowrun.Record{Run: core.Run{
		ID: "active-run", Target: core.Target{Scenario: "demo"}, IdempotencyKey: "active-key",
		State: core.StateQueued, CreatedAt: time.Now().UTC(), Version: 1,
	}}))
	handler := testHandler(t)
	handler.deps.Ledger = repo
	prepared, err := handler.Prepare(context.Background(), connect.NewRequest(&commonv1.LifecyclePrepareRequest{
		OperationId: "ordinary-operation", Reason: "managed restart",
	}))
	require.NoError(t, err)
	require.False(t, prepared.Msg.GetStanding().GetDrained())

	emergency, err := handler.Prepare(context.Background(), connect.NewRequest(&commonv1.LifecyclePrepareRequest{
		OperationId: "emergency-operation", Reason: "emergency replacement",
		EmergencyOverride: true, OverrideReason: "operator decision",
	}))
	require.NoError(t, err)
	require.True(t, emergency.Msg.GetStanding().GetAdmissionClosed())
	require.False(t, emergency.Msg.GetStanding().GetDrained())
	require.Equal(t, prepared.Msg.GetStanding().GetRevision(), emergency.Msg.GetStanding().GetRevision(), "adoption must not advance the durable fence or imply drain")
	retained, err := repo.GetLifecycleFence(context.Background())
	require.NoError(t, err)
	require.Equal(t, workflowrun.LifecycleFence{Closed: true, Operation: "emergency-operation", Revision: 1}, retained)
}

func TestLifecyclePrepareCancellationAfterCloseUsesBoundedRollbackContext(t *testing.T) {
	database := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(workflowrun.Schema)))
	ctx, cancel := context.WithCancel(context.Background())
	handler := testHandler(t)
	repo := workflowrun.Repository{DB: cancelAfterFenceCloseDB{DB: database, cancel: cancel}}
	handler.deps.Ledger = repo

	_, err := handler.Prepare(ctx, connect.NewRequest(&commonv1.LifecyclePrepareRequest{
		OperationId: "canceled-prepare", Reason: "caller canceled after close",
	}))
	require.Error(t, err)
	require.ErrorContains(t, err, "observe validation work")
	fence, err := (workflowrun.Repository{DB: database}).GetLifecycleFence(context.Background())
	require.NoError(t, err)
	require.Equal(t, workflowrun.LifecycleFence{Revision: 2}, fence, "cleanup must use a fresh bounded context and release only its own newly-created fence")

	priorDB := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), priorDB, apidb.SchemaProviderFunc(workflowrun.Schema)))
	priorRepo := workflowrun.Repository{DB: priorDB}
	initial, err := priorRepo.GetLifecycleFence(context.Background())
	require.NoError(t, err)
	require.NoError(t, priorRepo.UpdateLifecycleFence(context.Background(), initial.Revision, workflowrun.LifecycleFence{Closed: true, Operation: "abandoned-operation", Revision: 1}))
	priorCtx, cancelPrior := context.WithCancel(context.Background())
	priorHandler := testHandler(t)
	priorHandler.deps.Ledger = workflowrun.Repository{DB: cancelAfterFenceCloseDB{DB: priorDB, cancel: cancelPrior}}
	_, err = priorHandler.Prepare(priorCtx, connect.NewRequest(&commonv1.LifecyclePrepareRequest{
		OperationId: "emergency-adoption", Reason: "replace abandoned operation",
		EmergencyOverride: true, OverrideReason: "operator decision",
	}))
	require.Error(t, err)
	retained, err := priorRepo.GetLifecycleFence(context.Background())
	require.NoError(t, err)
	require.Equal(t, workflowrun.LifecycleFence{Closed: true, Operation: "emergency-adoption", Revision: 1}, retained,
		"failure while adopting a pre-existing fence must not reopen or advance it")
}

func TestLifecycleDrainInventoriesRoutedTestPoolAndUsesRawRunIDForExecutorEvidence(t *testing.T) {
	ctx := context.Background()
	routed, err := apidb.Open(ctx, apidb.Config{Driver: apidb.DriverSQLite, DSN: filepath.Join(t.TempDir(), "primary.db")})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, routed.Close()) })
	require.NoError(t, apidb.EnsureSchemas(ctx, routed.Primary(), apidb.SchemaProviderFunc(workflowrun.Schema)))
	routed.SetTestPoolInitializer(func(ctx context.Context, pool *sql.DB) error {
		return apidb.EnsureSchemas(ctx, pool, apidb.SchemaProviderFunc(workflowrun.Schema))
	})
	testDSN, err := storage.SQLiteDSNAt(filepath.Join(t.TempDir(), "test-pool.db"), storage.SQLiteTuning{TxLock: "immediate"})
	require.NoError(t, err)
	require.NoError(t, routed.InstallTestPool(ctx, testDSN, "lifecycle-test-pool", time.Minute))

	repo := workflowrun.Repository{DB: routed}
	record := workflowrun.Record{Run: core.Run{
		ID: "test-pool-run", Target: core.Target{Scenario: "demo"}, IdempotencyKey: "test-pool-key",
		State: core.StateQueued, CreatedAt: time.Now().UTC(), Version: 1,
	}}
	require.NoError(t, repo.Create(apidb.WithTestMode(ctx), record))
	handler := testHandler(t)
	handler.deps.Ledger = repo
	_, cancel := context.WithCancel(ctx)
	handler.mu.Lock()
	handler.cancels[record.Run.ID] = cancel
	handler.mu.Unlock()
	t.Cleanup(cancel)

	prepared, err := handler.Prepare(ctx, connect.NewRequest(&commonv1.LifecyclePrepareRequest{
		OperationId: "routed-inventory", Reason: "managed restart",
	}))
	require.NoError(t, err)
	standing := prepared.Msg.GetStanding()
	require.True(t, standing.GetAdmissionClosed())
	require.False(t, standing.GetDrained())
	require.Equal(t, int32(1), standing.GetRemaining())
	require.Equal(t, "test-pool/test-pool-run", standing.GetWork()[0].GetId())
	require.True(t, standing.GetWork()[0].GetPhysicalExecutorPresent(), "executor evidence is keyed by the persisted run ID, not the decorated inventory ID")

	primaryFence, err := repo.GetLifecycleFence(ctx)
	require.NoError(t, err)
	require.True(t, primaryFence.Closed, "maintenance fence is owned by the primary ledger")
	testFence, err := repo.GetLifecycleFence(apidb.WithTestMode(ctx))
	require.NoError(t, err)
	require.False(t, testFence.Closed, "test-mode routing must not redirect the provider's durable fence")
}

type failingActiveInventoryDB struct{ workflowrun.DB }

type cancelAfterFenceCloseDB struct {
	workflowrun.DB
	cancel context.CancelFunc
}

func (d cancelAfterFenceCloseDB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	result, err := d.DB.ExecContext(ctx, query, args...)
	if err == nil && strings.Contains(query, "UPDATE validation_lifecycle_fence") && d.cancel != nil {
		d.cancel()
	}
	return result, err
}

func (d failingActiveInventoryDB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	if strings.Contains(query, "FROM validation_runs WHERE state IN") {
		return nil, errors.New("injected durable inventory failure")
	}
	return d.DB.QueryContext(ctx, query, args...)
}

func repoTargetPath(t *testing.T, repo workflowrun.Repository, runID string) string {
	t.Helper()
	record, err := repo.Get(context.Background(), runID)
	require.NoError(t, err)
	return record.Run.Target.Path
}
