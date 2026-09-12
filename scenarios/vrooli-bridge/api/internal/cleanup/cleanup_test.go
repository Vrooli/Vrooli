package cleanup_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/scheduletest"

	"vrooli-bridge/internal/cleanup"
	localdb "vrooli-bridge/internal/database"
)

type fakeNodes struct{}

func (fakeNodes) GetTarget(context.Context, string) (cleanup.TargetNode, error) {
	return cleanup.TargetNode{ID: "node-1", SealingPublicKey: []byte("sealed-node-key"), Capabilities: []string{"runtime", "provision"}, Scopes: []string{"presence.read"}}, nil
}

type revocableNodes struct{ revoked bool }

func (n *revocableNodes) GetTarget(context.Context, string) (cleanup.TargetNode, error) {
	return cleanup.TargetNode{ID: "node-1", Revoked: n.revoked, SealingPublicKey: []byte("sealed-node-key")}, nil
}

type fakePresence bool

func (p fakePresence) IsOnline(string) bool { return bool(p) }

type fakePusher struct{ commands []cleanup.Command }

func (p *fakePusher) PushCleanup(_ context.Context, _ string, command cleanup.Command) (int, error) {
	p.commands = append(p.commands, command)
	return 1, nil
}

type fakeAudit struct{}

func (fakeAudit) Record(context.Context, cleanup.AuditEntry) error { return nil }

func setup(t *testing.T) (*scheduletest.FakeClock, cleanup.Repository) {
	t.Helper()
	clk := scheduletest.New(time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC))
	d := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), d,
		database.SchemaProviderFunc(localdb.SystemSchema), database.SchemaProviderFunc(cleanup.Schema)))
	return clk, cleanup.NewSQLiteRepository(d, clk)
}

func TestLeaseNamesExistingOperation(t *testing.T) {
	clk, repo := setup(t)
	pusher := &fakePusher{}
	svc := cleanup.NewService(repo, fakeNodes{}, fakePresence(true), pusher, fakeAudit{}, clk)
	ctx := context.Background()
	first, err := svc.Start(ctx, cleanup.StartInput{MachineID: "machine-1", NodeID: "node-1", Target: "minimouse", Scope: "all"}, "operator-1")
	require.NoError(t, err)
	second, err := svc.Start(ctx, cleanup.StartInput{MachineID: "machine-1", NodeID: "node-1", Target: "minimouse", Scope: "all"}, "operator-1")
	var inFlight cleanup.ErrInFlight
	require.ErrorAs(t, err, &inFlight)
	require.Equal(t, first.ID, inFlight.OperationID)
	require.Empty(t, second.ID)
}

func TestPrepareAndProvisionBreakGlassUseOneNodeBoundOperation(t *testing.T) {
	clk, repo := setup(t)
	pusher := &fakePusher{}
	svc := cleanup.NewService(repo, fakeNodes{}, fakePresence(true), pusher, fakeAudit{}, clk)
	ctx := context.Background()
	target, err := svc.Prepare(ctx, cleanup.StartInput{MachineID: "machine-protect", NodeID: "node-1", Target: "minimouse", Scope: "all"}, "operator-1")
	require.NoError(t, err)
	require.NotEmpty(t, target.OperationID)
	require.Equal(t, "operator-1", target.OperatorID)
	require.Empty(t, pusher.commands, "preparation is read-only and must not dispatch")

	op, err := svc.ProvisionBreakGlass(ctx, cleanup.ProvisionInput{MachineID: target.MachineID, NodeID: target.NodeID, Target: target.Target, Scope: target.Scope, OperationID: target.OperationID, OperatorID: target.OperatorID, SealedPassphrase: []byte("opaque-envelope")})
	require.NoError(t, err)
	require.Equal(t, target.OperationID, op.ID)
	require.Len(t, pusher.commands, 1)
	require.Equal(t, "provision_break_glass", pusher.commands[0].Operation)
	require.Equal(t, []byte("opaque-envelope"), pusher.commands[0].SealedPassphrase)
	require.Equal(t, "operator-1", pusher.commands[0].OperatorID)
	require.Equal(t, []string{"runtime", "provision"}, target.Capabilities)
	require.Equal(t, []string{"presence.read"}, target.ApprovedScopes)

	retried, err := svc.ProvisionBreakGlass(ctx, cleanup.ProvisionInput{MachineID: target.MachineID, NodeID: target.NodeID, Target: target.Target, Scope: target.Scope, OperationID: target.OperationID, OperatorID: target.OperatorID, SealedPassphrase: []byte("opaque-envelope")})
	require.NoError(t, err)
	require.Equal(t, op.ID, retried.ID)
	require.Len(t, pusher.commands, 1, "retrying the same operation must not dispatch a duplicate helper call")
}

func TestResetBreakGlassRetiresOnlyThroughExplicitTypedOperation(t *testing.T) {
	clk, repo := setup(t)
	pusher := &fakePusher{}
	svc := cleanup.NewService(repo, fakeNodes{}, fakePresence(true), pusher, fakeAudit{}, clk)
	op, err := svc.ResetBreakGlass(context.Background(), cleanup.ResetInput{
		MachineID: "machine-reset", NodeID: "node-1", Target: "minimouse", Scope: "all",
	}, "operator-1")
	require.NoError(t, err)
	require.Equal(t, cleanup.StatusApplying, op.Status)
	require.Len(t, pusher.commands, 1)
	require.Equal(t, "reset_break_glass", pusher.commands[0].Operation)
	require.True(t, pusher.commands[0].OperatorConfirmed)
	require.Empty(t, pusher.commands[0].SealedPassphrase)
	require.Empty(t, pusher.commands[0].Capability)
}

func TestFrozenPlanAndResumableAuthorization(t *testing.T) {
	clk, repo := setup(t)
	pusher := &fakePusher{}
	svc := cleanup.NewService(repo, fakeNodes{}, fakePresence(true), pusher, fakeAudit{}, clk)
	ctx := context.Background()
	op, err := svc.Start(ctx, cleanup.StartInput{MachineID: "machine-2", NodeID: "node-1", Target: "minimouse", Scope: "all"}, "operator-1")
	require.NoError(t, err)
	plan := []byte(`{"remove":[{"path":"/managed/a"}],"keep":[],"cannot_attribute":[]}`)
	parsed, err := cleanup.ParseFrozenPlan(plan)
	require.NoError(t, err)
	accepted, err := svc.AppendEvent(ctx, cleanup.Event{OperationID: op.ID, Kind: cleanup.EventPlan, Sequence: 1, PlanJSON: plan, Status: "planned"})
	require.NoError(t, err)
	require.True(t, accepted)
	planned, err := svc.Plan(ctx, op.ID)
	require.NoError(t, err)
	require.Equal(t, parsed.PlanHash, planned.PlanHash)
	require.Equal(t, plan, planned.PlanJSON)

	confirmed, err := svc.Confirm(ctx, cleanup.ConfirmInput{ID: op.ID, Target: "minimouse", PlanHash: parsed.PlanHash, SealedPassphrase: []byte("opaque-envelope"), Capability: []byte("opaque-capability"), OperatorID: "operator-1"})
	require.NoError(t, err)
	require.Equal(t, cleanup.StatusApplying, confirmed.Status)
	require.Len(t, pusher.commands, 2, "start and confirmation each dispatch exactly one typed command")
	require.Equal(t, "apply_frozen_plan", pusher.commands[1].Operation)

	resumed, err := svc.Apply(ctx, op.ID)
	require.NoError(t, err)
	require.Equal(t, cleanup.StatusApplying, resumed.Status)
	require.Equal(t, []byte("opaque-envelope"), pusher.commands[2].SealedPassphrase)
	require.Equal(t, []byte("opaque-capability"), pusher.commands[2].Capability)

	unchanged, _, err := svc.Get(ctx, op.ID)
	require.NoError(t, err)
	require.Equal(t, plan, unchanged.PlanJSON, "resume must not rediscover or replace the frozen plan")
}

func TestSuccessfulPlanExitKeepsOperationPlannedForConfirmation(t *testing.T) {
	clk, repo := setup(t)
	pusher := &fakePusher{}
	svc := cleanup.NewService(repo, fakeNodes{}, fakePresence(true), pusher, fakeAudit{}, clk)
	op, err := svc.Start(context.Background(), cleanup.StartInput{MachineID: "machine-plan-exit", NodeID: "node-1", Target: "minimouse", Scope: "all"}, "operator-1")
	require.NoError(t, err)
	plan := []byte(`{"remove":[],"keep":[],"cannot_attribute":[]}`)
	_, err = svc.AppendEvent(context.Background(), cleanup.Event{OperationID: op.ID, Kind: cleanup.EventPlan, Sequence: 1, PlanJSON: plan, Status: "planned"})
	require.NoError(t, err)
	_, err = svc.AppendEvent(context.Background(), cleanup.Event{OperationID: op.ID, Kind: cleanup.EventExit, Sequence: 2, ExitCode: 0})
	require.NoError(t, err)
	got, _, err := svc.Get(context.Background(), op.ID)
	require.NoError(t, err)
	require.Equal(t, cleanup.StatusPlanned, got.Status)
}

func TestEventSequencesAreReconciledAcrossTypedCommands(t *testing.T) {
	clk, repo := setup(t)
	pusher := &fakePusher{}
	svc := cleanup.NewService(repo, fakeNodes{}, fakePresence(true), pusher, fakeAudit{}, clk)
	ctx := context.Background()
	op, err := svc.Start(ctx, cleanup.StartInput{MachineID: "machine-sequences", NodeID: "node-1", Target: "minimouse", Scope: "agent"}, "operator-1")
	require.NoError(t, err)
	plan := []byte(`{"remove":[],"keep":[],"cannot_attribute":[]}`)
	parsed, err := cleanup.ParseFrozenPlan(plan)
	require.NoError(t, err)
	planEvent := cleanup.Event{OperationID: op.ID, Kind: cleanup.EventPlan, Sequence: 1, PlanJSON: plan, Status: "planned"}
	accepted, err := svc.AppendEvent(ctx, planEvent)
	require.NoError(t, err)
	require.True(t, accepted)

	accepted, err = svc.AppendEvent(ctx, planEvent)
	require.NoError(t, err)
	require.False(t, accepted, "identical transport replay should be idempotent")

	_, err = svc.AppendEvent(ctx, cleanup.Event{OperationID: op.ID, Kind: cleanup.EventExit, Sequence: 2, ExitCode: 0})
	require.NoError(t, err)
	receipt, err := json.Marshal(map[string]any{
		"plan_id": op.ID, "plan_hash": parsed.PlanHash, "target": op.Target, "scope": op.Scope,
		"authorizing_user": "operator-1", "removed": []any{}, "preserved": []any{},
		"cannot_attribute": []any{}, "attempts": []any{map[string]any{"applied": []any{}}},
	})
	require.NoError(t, err)
	accepted, err = svc.AppendEvent(ctx, cleanup.Event{OperationID: op.ID, Kind: cleanup.EventReceipt, Sequence: 1, ReceiptJSON: receipt})
	require.NoError(t, err)
	require.True(t, accepted)
	accepted, err = svc.AppendEvent(ctx, cleanup.Event{OperationID: op.ID, Kind: cleanup.EventStatus, Sequence: 2, Status: "completed"})
	require.NoError(t, err)
	require.True(t, accepted)

	got, events, err := svc.Get(ctx, op.ID)
	require.NoError(t, err)
	require.Equal(t, cleanup.StatusCompleted, got.Status)
	require.Equal(t, receipt, got.ReceiptJSON)
	require.Len(t, events, 4)
	require.Equal(t, []uint64{1, 2, 3, 4}, []uint64{events[0].Sequence, events[1].Sequence, events[2].Sequence, events[3].Sequence})
}

func TestVerifyDoesNotDowngradeTerminalReceiptWhenAgentIsGone(t *testing.T) {
	clk, repo := setup(t)
	pusher := &fakePusher{}
	svc := cleanup.NewService(repo, fakeNodes{}, fakePresence(true), pusher, fakeAudit{}, clk)
	ctx := context.Background()
	op, err := svc.Start(ctx, cleanup.StartInput{MachineID: "machine-verify", NodeID: "node-1", Target: "minimouse", Scope: "agent"}, "operator-1")
	require.NoError(t, err)
	_, err = svc.AppendEvent(ctx, cleanup.Event{OperationID: op.ID, Kind: cleanup.EventExit, Sequence: 1, ExitCode: 1})
	require.NoError(t, err)
	before, _, err := svc.Get(ctx, op.ID)
	require.NoError(t, err)
	verified, err := svc.Verify(ctx, op.ID)
	require.NoError(t, err)
	require.Equal(t, before.Status, verified.Status)
	require.Equal(t, cleanup.StatusFailed, verified.Status)
	require.Len(t, pusher.commands, 1, "terminal verification must not dispatch to an absent agent")
}

func TestOwnerSessionWithoutSealedPassphraseCannotConfirm(t *testing.T) {
	clk, repo := setup(t)
	pusher := &fakePusher{}
	svc := cleanup.NewService(repo, fakeNodes{}, fakePresence(true), pusher, fakeAudit{}, clk)
	ctx := context.Background()
	op, err := svc.Start(ctx, cleanup.StartInput{MachineID: "machine-owner", NodeID: "node-1", Target: "minimouse", Scope: "all"}, "operator-1")
	require.NoError(t, err)
	plan := []byte(`{"remove":[],"keep":[],"cannot_attribute":[]}`)
	parsed, err := cleanup.ParseFrozenPlan(plan)
	require.NoError(t, err)
	_, err = svc.AppendEvent(ctx, cleanup.Event{OperationID: op.ID, Kind: cleanup.EventPlan, Sequence: 1, PlanJSON: plan, Status: "planned"})
	require.NoError(t, err)
	_, err = svc.Confirm(ctx, cleanup.ConfirmInput{ID: op.ID, Target: op.Target, PlanHash: parsed.PlanHash, OperatorID: "operator-1"})
	var invalid cleanup.ErrInvalid
	require.ErrorAs(t, err, &invalid)
	require.Equal(t, "sealed_passphrase", invalid.Field)
	require.Len(t, pusher.commands, 1, "owner authentication alone must not dispatch apply")
}

func TestRevokedNodeStopsPendingApply(t *testing.T) {
	clk, repo := setup(t)
	nodes := &revocableNodes{}
	pusher := &fakePusher{}
	svc := cleanup.NewService(repo, nodes, fakePresence(true), pusher, fakeAudit{}, clk)
	ctx := context.Background()
	op, err := svc.Start(ctx, cleanup.StartInput{MachineID: "machine-revoked", NodeID: "node-1", Target: "minimouse", Scope: "all"}, "operator-1")
	require.NoError(t, err)
	plan := []byte(`{"remove":[],"keep":[],"cannot_attribute":[]}`)
	parsed, err := cleanup.ParseFrozenPlan(plan)
	require.NoError(t, err)
	_, err = svc.AppendEvent(ctx, cleanup.Event{OperationID: op.ID, Kind: cleanup.EventPlan, Sequence: 1, PlanJSON: plan, Status: "planned"})
	require.NoError(t, err)
	_, err = svc.Confirm(ctx, cleanup.ConfirmInput{ID: op.ID, Target: op.Target, PlanHash: parsed.PlanHash, SealedPassphrase: []byte("opaque"), OperatorID: "operator-1"})
	require.NoError(t, err)
	nodes.revoked = true
	_, err = svc.Apply(ctx, op.ID)
	var blocked cleanup.ErrBlocked
	require.ErrorAs(t, err, &blocked)
	require.Equal(t, "node", blocked.Field)
	require.Len(t, pusher.commands, 2, "revocation blocks the resume dispatch")
}

func TestReceiptShapeAndFailedApplyResume(t *testing.T) {
	clk, repo := setup(t)
	pusher := &fakePusher{}
	svc := cleanup.NewService(repo, fakeNodes{}, fakePresence(true), pusher, fakeAudit{}, clk)
	ctx := context.Background()
	op, err := svc.Start(ctx, cleanup.StartInput{MachineID: "machine-receipt", NodeID: "node-1", Target: "minimouse", Scope: "all"}, "operator-1")
	require.NoError(t, err)
	plan := []byte(`{"remove":[],"keep":[],"cannot_attribute":[]}`)
	parsed, err := cleanup.ParseFrozenPlan(plan)
	require.NoError(t, err)
	_, err = svc.AppendEvent(ctx, cleanup.Event{OperationID: op.ID, Kind: cleanup.EventPlan, Sequence: 1, PlanJSON: plan, Status: "planned"})
	require.NoError(t, err)
	_, err = svc.Confirm(ctx, cleanup.ConfirmInput{ID: op.ID, Target: op.Target, PlanHash: parsed.PlanHash, SealedPassphrase: []byte("opaque"), Capability: []byte("cap"), OperatorID: "operator-1"})
	require.NoError(t, err)

	receipt, err := json.Marshal(map[string]any{
		"plan_id": op.ID, "plan_hash": parsed.PlanHash, "target": op.Target, "scope": op.Scope,
		"authorizing_user": "operator-1", "removed": []any{}, "preserved": []any{},
		"cannot_attribute": []any{}, "attempts": []any{map[string]any{"applied": []any{}}, map[string]any{"applied": []any{}}},
	})
	require.NoError(t, err)
	accepted, err := svc.AppendEvent(ctx, cleanup.Event{OperationID: op.ID, Kind: cleanup.EventReceipt, Sequence: 2, ReceiptJSON: receipt})
	require.NoError(t, err)
	require.True(t, accepted)

	_, err = svc.AppendEvent(ctx, cleanup.Event{OperationID: op.ID, Kind: cleanup.EventExit, Sequence: 3, ExitCode: 1, Status: "failed"})
	require.NoError(t, err)
	resumed, err := svc.Apply(ctx, op.ID)
	require.NoError(t, err)
	require.Equal(t, cleanup.StatusApplying, resumed.Status)
	require.Len(t, pusher.commands, 3, "start, confirm, and resume each dispatch one typed command")
	got, _, err := svc.Get(ctx, op.ID)
	require.NoError(t, err)
	require.Equal(t, receipt, got.ReceiptJSON)
}

func TestWaitWakesFromSharedOperationCoordinator(t *testing.T) {
	clk, repo := setup(t)
	pusher := &fakePusher{}
	svc := cleanup.NewService(repo, fakeNodes{}, fakePresence(true), pusher, fakeAudit{}, clk)
	ctx := context.Background()
	op, err := svc.Start(ctx, cleanup.StartInput{MachineID: "machine-wait", NodeID: "node-1", Target: "minimouse", Scope: "all"}, "operator-1")
	require.NoError(t, err)
	result := make(chan cleanup.Operation, 1)
	go func() {
		finished, timedOut, waitErr := svc.Wait(ctx, op.ID, time.Second)
		if waitErr != nil || timedOut {
			result <- cleanup.Operation{}
			return
		}
		result <- finished
	}()
	_, err = svc.AppendEvent(ctx, cleanup.Event{OperationID: op.ID, Kind: cleanup.EventExit, Sequence: 1, ExitCode: 0})
	require.NoError(t, err)
	require.Equal(t, cleanup.StatusCompleted, (<-result).Status)
}
