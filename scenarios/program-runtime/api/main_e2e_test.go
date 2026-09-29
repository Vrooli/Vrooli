//go:build e2e

package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/boottest"
	apidb "github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
	bindingshandler "program-runtime/handlers/bindings"
	runtimeprograms "program-runtime/internal/programs"
	"program-runtime/internal/sessions"
)

// Verify this API entry point serves its own healthy response and terminates.
// boottest owns process setup, isolated storage, deadlines, and diagnostics.
func TestE2E_BinaryBootsAndServesHealth(t *testing.T) {
	boottest.Run(t, boottest.Config{Service: "program-runtime-api"})
}

func TestProductionDeclaredUsageReceiptBoundary(t *testing.T) {
	ctx := context.Background()
	manager := sessions.NewManager(sessions.Options{})
	makeSession := func(name string) *sessions.Session {
		session, err := manager.Create(ctx, name, "", nil)
		require.NoError(t, err)
		return session
	}
	read := declaredUsageReceipt(manager)

	dedicated := makeSession("declared-program:owner.workflow")
	zero, err := read(ctx, dedicated.ID)
	require.NoError(t, err)
	require.True(t, zero.AccountingComplete)
	require.True(t, zero.ChargeMeasured)
	require.Zero(t, zero.Tokens)
	require.Zero(t, zero.ChargeMicros)

	require.NoError(t, manager.RecordInferenceUsage(ctx, dedicated.ID, 125, 17))
	require.NoError(t, manager.SaveDelegation(ctx, &sessions.Delegation{SessionID: dedicated.ID, ExecutionID: "child-done", CreatedAt: time.Now().UTC(), LastStatus: "succeeded"}))
	unsettled, err := read(ctx, dedicated.ID)
	require.NoError(t, err)
	require.False(t, unsettled.AccountingComplete, "terminal status alone is not a usage settlement receipt")
	require.False(t, unsettled.ChargeMeasured)
	_, err = manager.SettleDelegationUsage(ctx, dedicated.ID, "child-done", "succeeded", 40, true, "priced child")
	require.NoError(t, err)
	measured, err := read(ctx, dedicated.ID)
	require.NoError(t, err)
	require.True(t, measured.AccountingComplete)
	require.True(t, measured.ChargeMeasured)
	require.EqualValues(t, 17, measured.Tokens)
	require.EqualValues(t, 165, measured.ChargeMicros)

	reused := makeSession("interactive-session")
	require.NoError(t, manager.RecordInferenceUsage(ctx, reused.ID, 999, 999))
	isolation, err := read(ctx, reused.ID)
	require.NoError(t, err)
	require.False(t, isolation.AccountingComplete)
	require.False(t, isolation.ChargeMeasured)
	require.Zero(t, isolation.Tokens)
	require.Zero(t, isolation.ChargeMicros)

	unknown := makeSession("declared-program:unpriced")
	require.NoError(t, manager.RecordInferenceUsage(ctx, unknown.ID, 0, 4))
	require.NoError(t, manager.MarkAccountingUnknown(ctx, unknown.ID))
	unpriced, err := read(ctx, unknown.ID)
	require.NoError(t, err)
	require.False(t, unpriced.AccountingComplete)
	require.False(t, unpriced.ChargeMeasured)

	delegationUnknown := makeSession("declared-program:unpriced-child")
	require.NoError(t, manager.SaveDelegation(ctx, &sessions.Delegation{SessionID: delegationUnknown.ID, ExecutionID: "unpriced-child", CreatedAt: time.Now().UTC(), LastStatus: "running"}))
	_, err = manager.SettleDelegationUsage(ctx, delegationUnknown.ID, "unpriced-child", "succeeded", 0, false, "missing per-run charge")
	require.NoError(t, err)
	unknownChild, err := read(ctx, delegationUnknown.ID)
	require.NoError(t, err)
	require.False(t, unknownChild.AccountingComplete)
	require.False(t, unknownChild.ChargeMeasured)

	pending := makeSession("declared-program:pending-child")
	require.NoError(t, manager.SaveDelegation(ctx, &sessions.Delegation{SessionID: pending.ID, ExecutionID: "child-running", CreatedAt: time.Now().UTC(), LastStatus: "running"}))
	unfinished, err := read(ctx, pending.ID)
	require.NoError(t, err)
	require.False(t, unfinished.AccountingComplete)
	require.False(t, unfinished.ChargeMeasured)
}

func TestProductionDeclaredUsageReceiptFailsClosedWhenBothInferenceWritesFail(t *testing.T) {
	ctx := context.Background()
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, db, apidb.SchemaProviderFunc(sessions.Schema), apidb.SchemaProviderFunc(runtimeprograms.Schema)))
	manager := sessions.NewManager(sessions.Options{Store: db})
	session, err := manager.Create(ctx, "declared-program:uncertain", "", nil)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `CREATE TRIGGER reject_inference_meter BEFORE UPDATE OF inference_cost_micros ON sessions BEGIN SELECT RAISE(FAIL, 'meter unavailable'); END`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `CREATE TRIGGER reject_unknown_flag BEFORE UPDATE OF inference_charge_unknown ON sessions BEGIN SELECT RAISE(FAIL, 'unknown flag unavailable'); END`)
	require.NoError(t, err)
	require.Error(t, manager.RecordInferenceUsage(ctx, session.ID, 10, 4))

	var storedUnknown int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT inference_charge_unknown FROM sessions WHERE id = ?`, session.ID).Scan(&storedUnknown))
	require.Zero(t, storedUnknown, "the test must exercise failure of both durable writes")
	receipt, err := declaredUsageReceipt(manager)(ctx, session.ID)
	require.NoError(t, err)
	require.False(t, receipt.AccountingComplete)
	require.False(t, receipt.ChargeMeasured)
	require.Zero(t, receipt.ChargeMicros)

	service := runtimeprograms.NewService(runtimeprograms.Options{
		Store:           db,
		Runner:          inferenceFailureRunner{manager: manager, sessionID: session.ID},
		UsageReceipt:    declaredUsageReceipt(manager),
		ValidateSession: func(id string) bool { _, getErr := manager.Get(ctx, id); return getErr == nil },
	})
	program, err := service.Submit(ctx, session.ID, "pass", programsv1.Provenance_PROVENANCE_AGENT, false)
	require.NoError(t, err)
	require.False(t, program.GetUsageAccountingComplete())
	require.False(t, program.GetUsageChargeMeasured())
	_, err = manager.Delete(ctx, session.ID, "completed")
	require.NoError(t, err)
	restartedProgram, err := runtimeprograms.NewRepository(db).Get(ctx, program.GetId())
	require.NoError(t, err)
	require.False(t, restartedProgram.GetUsageAccountingComplete())
	require.False(t, restartedProgram.GetUsageChargeMeasured())
}

func TestProductionDeclaredUsageReceiptFailsClosedAfterDelegationPersistenceFailures(t *testing.T) {
	for _, tc := range []struct {
		name, path, failureTrigger string
	}{
		{name: "async child save", path: "start", failureTrigger: `CREATE TRIGGER reject_child_save BEFORE INSERT ON session_delegations BEGIN SELECT RAISE(FAIL, 'child row unavailable'); END`},
		{name: "sync meter update", path: "execute", failureTrigger: `CREATE TRIGGER reject_delegation_meter BEFORE UPDATE OF delegation_cost_micros ON sessions BEGIN SELECT RAISE(FAIL, 'delegation meter unavailable'); END`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			db := dbtest.NewSQLite(t)
			require.NoError(t, apidb.EnsureSchemas(ctx, db, apidb.SchemaProviderFunc(sessions.Schema), apidb.SchemaProviderFunc(runtimeprograms.Schema)))
			manager := sessions.NewManager(sessions.Options{Store: db})
			session, err := manager.Create(ctx, "declared-program:delegation-failure", "", nil)
			require.NoError(t, err)
			_, err = db.ExecContext(ctx, tc.failureTrigger)
			require.NoError(t, err)
			_, err = db.ExecContext(ctx, `CREATE TRIGGER reject_uncertainty BEFORE UPDATE OF inference_charge_unknown, delegation_usage_observed ON sessions BEGIN SELECT RAISE(FAIL, 'uncertainty columns unavailable'); END`)
			require.NoError(t, err)

			requestBody := `{"session_id":"` + session.ID + `","owner":"owner","workflow_key":"owner/workflow","idempotency_key":"same-key"}`
			var handler http.Handler
			delegator := fixedDelegationResult{}
			if tc.path == "start" {
				handler = bindingshandler.AgentStartBridge(manager, delegator)
			} else {
				handler = bindingshandler.AgentBridge(manager, delegator)
			}
			runner := &delegationCatchRunner{handler: handler, body: requestBody}
			service := runtimeprograms.NewService(runtimeprograms.Options{
				Store: db, Runner: runner, UsageReceipt: declaredUsageReceipt(manager),
				ValidateSession: func(id string) bool { _, getErr := manager.Get(ctx, id); return getErr == nil },
			})
			program, err := service.Submit(ctx, session.ID, "pass", programsv1.Provenance_PROVENANCE_AGENT, false)
			require.NoError(t, err)
			require.Equal(t, http.StatusInternalServerError, runner.status)
			require.False(t, program.GetUsageAccountingComplete())
			require.False(t, program.GetUsageChargeMeasured())
			_, err = manager.Delete(ctx, session.ID, "completed")
			require.NoError(t, err)
			restarted, err := runtimeprograms.NewRepository(db).Get(ctx, program.GetId())
			require.NoError(t, err)
			require.False(t, restarted.GetUsageAccountingComplete())
			require.False(t, restarted.GetUsageChargeMeasured())
		})
	}
}

type inferenceFailureRunner struct {
	manager   *sessions.Manager
	sessionID string
}

func (r inferenceFailureRunner) Execute(context.Context, string, string, bool) (runtimeprograms.Result, error) {
	_ = r.manager.RecordInferenceUsage(context.Background(), r.sessionID, 10, 4) // Program catches the bridge error.
	return runtimeprograms.Result{Stdout: "caught"}, nil
}

type fixedDelegationResult struct{}

func (fixedDelegationResult) Delegate(context.Context, runtimeprograms.DelegationRequest) (map[string]any, error) {
	return map[string]any{"execution_id": "exec-shared", "status": "succeeded", "owner": "owner", "workflow_key": "owner/workflow", "idempotency_key": "same-key", "cost_micros": float64(5)}, nil
}
func (fixedDelegationResult) Start(context.Context, runtimeprograms.DelegationRequest) (map[string]any, error) {
	return map[string]any{"execution_id": "exec-shared", "status": "succeeded", "owner": "owner", "workflow_key": "owner/workflow", "idempotency_key": "same-key", "cost_micros": float64(5)}, nil
}
func (fixedDelegationResult) Collect(context.Context, string, string, int) (map[string]any, error) {
	return nil, nil
}

type delegationCatchRunner struct {
	handler http.Handler
	body    string
	status  int
}

func (r *delegationCatchRunner) Execute(context.Context, string, string, bool) (runtimeprograms.Result, error) {
	response := httptest.NewRecorder()
	r.handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(r.body)))
	r.status = response.Code
	return runtimeprograms.Result{Stdout: response.Body.String()}, nil
}
