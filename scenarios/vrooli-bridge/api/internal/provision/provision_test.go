package provision_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"vrooli-bridge/internal/clock"
	"vrooli-bridge/internal/provision"
	"vrooli-bridge/internal/provision/mocks"

	"github.com/stretchr/testify/require"
)

// newService wires the provision service against the in-memory fakes with the
// node online and a deliverable channel by default.
func newService(t *testing.T) (provision.Service, *mocks.FakeRepository, *mocks.FakeNodeReader, *mocks.FakePresence, *mocks.FakeAuditSink, *mocks.FakeCommandPusher) {
	t.Helper()
	repo := mocks.NewFakeRepository()
	nodes := &mocks.FakeNodeReader{Nodes: map[string]provision.TargetNode{
		"n1": {ID: "n1"},
	}}
	pres := &mocks.FakePresence{Online: map[string]bool{"n1": true}}
	audit := &mocks.FakeAuditSink{}
	pusher := &mocks.FakeCommandPusher{Delivered: 1}
	svc := provision.NewService(repo, nodes, pres, audit, pusher, clock.System{})
	return svc, repo, nodes, pres, audit, pusher
}

// [REQ:BRG-P0-006] A valid SyncToRevision creates a durable op, audits it
// (accepted), and pushes the privileged ProvisionCommand to the node.
func TestSync_CreatesOpAuditsAndPushes(t *testing.T) {
	svc, _, _, _, audit, pusher := newService(t)

	dec, err := svc.Sync(context.Background(), provision.SyncInput{
		Actor: "owner", NodeID: "n1", TargetRevision: "rev-B",
	})
	require.NoError(t, err)
	require.NotEmpty(t, dec.OpID)
	require.False(t, dec.DryRun)
	require.Equal(t, "rev-B", dec.TargetRevision)

	pushed := pusher.PushedCommands()
	require.Len(t, pushed, 1)
	require.Equal(t, dec.OpID, pushed[0].OpID)
	require.Equal(t, "rev-B", pushed[0].TargetRevision)

	recorded := audit.Recorded()
	require.Len(t, recorded, 1)
	require.True(t, recorded[0].Accepted)
	require.Equal(t, dec.OpID, recorded[0].OpID)
}

// [REQ:BRG-P0-006] When no explicit rollback is given, the control plane uses
// the node's last recorded version so a failed setup returns the node to where
// it was.
func TestSync_ResolvesRollbackFromNodeVersion(t *testing.T) {
	svc, repo, _, _, _, pusher := newService(t)
	repo.SeedVersion(provision.NodeVersion{NodeID: "n1", Revision: "rev-A"})

	dec, err := svc.Sync(context.Background(), provision.SyncInput{
		Actor: "owner", NodeID: "n1", TargetRevision: "rev-B",
	})
	require.NoError(t, err)
	require.Equal(t, "rev-A", dec.RollbackRevision)
	require.Equal(t, "rev-A", pusher.PushedCommands()[0].RollbackRevision)
}

// [REQ:BRG-P0-006] A dry-run validates and short-circuits BEFORE any side
// effect: no op created, nothing audited, nothing pushed.
func TestSync_DryRunShortCircuits(t *testing.T) {
	svc, repo, _, _, audit, pusher := newService(t)

	dec, err := svc.Sync(context.Background(), provision.SyncInput{
		Actor: "owner", NodeID: "n1", TargetRevision: "rev-B", DryRun: true,
	})
	require.NoError(t, err)
	require.True(t, dec.DryRun)
	require.Empty(t, dec.OpID)

	ops, _ := repo.List(context.Background(), provision.ListFilter{})
	require.Empty(t, ops, "no op created on a dry-run")
	require.Empty(t, audit.Recorded(), "nothing audited on a dry-run")
	require.Empty(t, pusher.PushedCommands(), "nothing pushed on a dry-run")
}

// [REQ:BRG-P0-006] An unknown node is rejected before any op is created.
func TestSync_UnknownNodeRejected(t *testing.T) {
	svc, _, _, _, _, _ := newService(t)
	_, err := svc.Sync(context.Background(), provision.SyncInput{
		Actor: "owner", NodeID: "ghost", TargetRevision: "rev-B",
	})
	var notFound provision.ErrNodeNotFound
	require.ErrorAs(t, err, &notFound)
}

// [REQ:BRG-P0-006] A revoked node can be provisioned no further; the rejection
// is audited.
func TestSync_RevokedNodeRejectedAndAudited(t *testing.T) {
	svc, _, nodes, _, audit, _ := newService(t)
	nodes.Nodes["n1"] = provision.TargetNode{ID: "n1", Revoked: true}

	_, err := svc.Sync(context.Background(), provision.SyncInput{
		Actor: "owner", NodeID: "n1", TargetRevision: "rev-B",
	})
	var revoked provision.ErrNodeRevoked
	require.ErrorAs(t, err, &revoked)
	recorded := audit.Recorded()
	require.Len(t, recorded, 1)
	require.False(t, recorded[0].Accepted)
}

// [REQ:BRG-P0-006] An offline node cannot receive the privileged push; the
// request is rejected (and audited) and no command is pushed.
func TestSync_OfflineNodeRejected(t *testing.T) {
	svc, _, _, pres, audit, pusher := newService(t)
	pres.Online["n1"] = false

	_, err := svc.Sync(context.Background(), provision.SyncInput{
		Actor: "owner", NodeID: "n1", TargetRevision: "rev-B",
	})
	var offline provision.ErrNodeOffline
	require.ErrorAs(t, err, &offline)
	require.Empty(t, pusher.PushedCommands())
	require.Len(t, audit.Recorded(), 1)
}

// [REQ:BRG-P0-006] Audit is FAIL-CLOSED: if the accepted-op audit write fails,
// the op is marked failed and the request errors rather than provisioning
// un-audited.
func TestSync_AuditFailureMarksOpFailed(t *testing.T) {
	svc, repo, _, _, audit, pusher := newService(t)
	audit.RecordErr = errors.New("audit down")

	_, err := svc.Sync(context.Background(), provision.SyncInput{
		Actor: "owner", NodeID: "n1", TargetRevision: "rev-B",
	})
	require.Error(t, err)
	require.Empty(t, pusher.PushedCommands(), "nothing is pushed when audit fails closed")

	ops, _ := repo.List(context.Background(), provision.ListFilter{})
	require.Len(t, ops, 1)
	require.Equal(t, provision.StatusFailed, ops[0].Status)
}

// [REQ:BRG-P0-006] A delivery failure (the node dropped between the online
// check and the push) marks the op failed and fails the request.
func TestSync_DeliveryFailureMarksOpFailed(t *testing.T) {
	svc, repo, _, _, _, pusher := newService(t)
	pusher.Delivered = 0 // no live connection received the frame

	_, err := svc.Sync(context.Background(), provision.SyncInput{
		Actor: "owner", NodeID: "n1", TargetRevision: "rev-B",
	})
	var delivery provision.ErrDeliveryFailed
	require.ErrorAs(t, err, &delivery)
	ops, _ := repo.List(context.Background(), provision.ListFilter{})
	require.Len(t, ops, 1)
	require.Equal(t, provision.StatusFailed, ops[0].Status)
}

// [REQ:BRG-P0-006] The node-event lifecycle: STATUS drives RUNNING, VERSION
// records the node's version, and a clean EXIT(0) marks COMPLETED — reporting
// the node's resulting version.
func TestAppendEvent_SuccessLifecycle(t *testing.T) {
	svc, _, _, _, _, _ := newService(t)
	ctx := context.Background()
	dec, err := svc.Sync(ctx, provision.SyncInput{Actor: "owner", NodeID: "n1", TargetRevision: "rev-B"})
	require.NoError(t, err)
	id := dec.OpID

	accepted, err := svc.AppendEvent(ctx, provision.ProvisionEvent{OpID: id, Kind: provision.EventStatus, Sequence: 1, Status: "fetching"})
	require.NoError(t, err)
	require.True(t, accepted)
	op, _, _ := svc.GetOp(ctx, id)
	require.Equal(t, provision.StatusRunning, op.Status)

	_, err = svc.AppendEvent(ctx, provision.ProvisionEvent{OpID: id, Kind: provision.EventVersion, Sequence: 2, Revision: "rev-B"})
	require.NoError(t, err)

	_, err = svc.AppendEvent(ctx, provision.ProvisionEvent{OpID: id, Kind: provision.EventExit, Sequence: 3, ExitCode: 0})
	require.NoError(t, err)

	op, _, _ = svc.GetOp(ctx, id)
	require.Equal(t, provision.StatusCompleted, op.Status)
	require.Equal(t, "rev-B", op.ResultingRevision)

	// The node's recorded version is now rev-B.
	ver, err := svc.GetNodeVersion(ctx, "n1")
	require.NoError(t, err)
	require.Equal(t, "rev-B", ver.Revision)
	require.Equal(t, id, ver.OpID)
}

// [REQ:BRG-P0-006] Rollback outcome: a failing setup that lands the node back on
// the rollback revision is the SAFE failure (ROLLED_BACK), distinct from a
// degraded FAILED. The node's recorded version is the rollback revision.
func TestAppendEvent_RollbackOutcome(t *testing.T) {
	svc, repo, _, _, _, _ := newService(t)
	ctx := context.Background()
	repo.SeedVersion(provision.NodeVersion{NodeID: "n1", Revision: "rev-A"})
	dec, err := svc.Sync(ctx, provision.SyncInput{Actor: "owner", NodeID: "n1", TargetRevision: "rev-B"})
	require.NoError(t, err)
	id := dec.OpID

	// helper: running → setup fails → rolls back to rev-A → reports VERSION rev-A
	// → EXIT non-zero (the original setup failure code).
	_, _ = svc.AppendEvent(ctx, provision.ProvisionEvent{OpID: id, Kind: provision.EventStatus, Sequence: 1, Status: "running setup"})
	_, _ = svc.AppendEvent(ctx, provision.ProvisionEvent{OpID: id, Kind: provision.EventStatus, Sequence: 2, Status: "rolling back"})
	_, _ = svc.AppendEvent(ctx, provision.ProvisionEvent{OpID: id, Kind: provision.EventVersion, Sequence: 3, Revision: "rev-A"})
	_, err = svc.AppendEvent(ctx, provision.ProvisionEvent{OpID: id, Kind: provision.EventExit, Sequence: 4, ExitCode: 1})
	require.NoError(t, err)

	op, _, _ := svc.GetOp(ctx, id)
	require.Equal(t, provision.StatusRolledBack, op.Status)
	require.Equal(t, "rev-A", op.ResultingRevision)

	ver, _ := svc.GetNodeVersion(ctx, "n1")
	require.Equal(t, "rev-A", ver.Revision, "the node is back on its prior revision")
}

// [REQ:BRG-P0-006] A non-zero exit with NO successful rollback is FAILED
// (degraded) — distinct from the safe ROLLED_BACK outcome.
func TestAppendEvent_DegradedFailure(t *testing.T) {
	svc, _, _, _, _, _ := newService(t)
	ctx := context.Background()
	// First provision (no rollback target): a failure cannot roll back.
	dec, err := svc.Sync(ctx, provision.SyncInput{Actor: "owner", NodeID: "n1", TargetRevision: "rev-B"})
	require.NoError(t, err)
	id := dec.OpID

	_, _ = svc.AppendEvent(ctx, provision.ProvisionEvent{OpID: id, Kind: provision.EventStatus, Sequence: 1, Status: "running setup"})
	_, err = svc.AppendEvent(ctx, provision.ProvisionEvent{OpID: id, Kind: provision.EventExit, Sequence: 2, ExitCode: 1})
	require.NoError(t, err)

	op, _, _ := svc.GetOp(ctx, id)
	require.Equal(t, provision.StatusFailed, op.Status)
}

// [REQ:BRG-P0-006] An event for an already-terminal op is acknowledged
// (accepted=false) without error — a re-sending node never spins, and the
// terminal status is not disturbed.
func TestAppendEvent_StaleAfterTerminal(t *testing.T) {
	svc, _, _, _, _, _ := newService(t)
	ctx := context.Background()
	dec, _ := svc.Sync(ctx, provision.SyncInput{Actor: "owner", NodeID: "n1", TargetRevision: "rev-B"})
	id := dec.OpID
	_, _ = svc.AppendEvent(ctx, provision.ProvisionEvent{OpID: id, Kind: provision.EventExit, Sequence: 1, ExitCode: 0})

	accepted, err := svc.AppendEvent(ctx, provision.ProvisionEvent{OpID: id, Kind: provision.EventLog, Sequence: 2, LogChunk: "late"})
	require.NoError(t, err)
	require.False(t, accepted)
	op, _, _ := svc.GetOp(ctx, id)
	require.Equal(t, provision.StatusCompleted, op.Status)
}

// [REQ:BRG-P0-006] An event for an unknown op is acknowledged without error.
func TestAppendEvent_UnknownOp(t *testing.T) {
	svc, _, _, _, _, _ := newService(t)
	accepted, err := svc.AppendEvent(context.Background(), provision.ProvisionEvent{OpID: "ghost", Kind: provision.EventLog, Sequence: 1})
	require.NoError(t, err)
	require.False(t, accepted)
}

// [REQ:BRG-P0-006] Wait blocks once and returns the terminal op when a terminal
// EXIT arrives — no polling.
func TestWait_BlockOnceUntilTerminal(t *testing.T) {
	svc, _, _, _, _, _ := newService(t)
	ctx := context.Background()
	dec, _ := svc.Sync(ctx, provision.SyncInput{Actor: "owner", NodeID: "n1", TargetRevision: "rev-B"})
	id := dec.OpID

	done := make(chan struct{})
	var got provision.ProvisioningOp
	var timedOut bool
	go func() {
		got, timedOut, _ = svc.Wait(ctx, id, 5*time.Second)
		close(done)
	}()

	// Give the waiter a beat to register, then drive the op terminal.
	time.Sleep(20 * time.Millisecond)
	_, err := svc.AppendEvent(ctx, provision.ProvisionEvent{OpID: id, Kind: provision.EventExit, Sequence: 1, ExitCode: 0})
	require.NoError(t, err)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Wait did not return after terminal event")
	}
	require.False(t, timedOut)
	require.Equal(t, provision.StatusCompleted, got.Status)
}

// [REQ:BRG-P0-006] Wait returns timed_out=true when the deadline elapses before
// the op is terminal.
func TestWait_TimesOut(t *testing.T) {
	svc, _, _, _, _, _ := newService(t)
	ctx := context.Background()
	dec, _ := svc.Sync(ctx, provision.SyncInput{Actor: "owner", NodeID: "n1", TargetRevision: "rev-B"})

	op, timedOut, err := svc.Wait(ctx, dec.OpID, 30*time.Millisecond)
	require.NoError(t, err)
	require.True(t, timedOut)
	require.False(t, op.Status.Terminal())
}

// [REQ:BRG-P0-006] Idempotent re-provision: provisioning a node already at the
// target revision is safe and yields the same end state — the node's recorded
// version remains the target. (The helper's filesystem idempotency is proven in
// the node-agent privsep test; this asserts the control-plane's view converges.)
func TestSync_IdempotentReprovisionConverges(t *testing.T) {
	svc, _, _, _, _, _ := newService(t)
	ctx := context.Background()

	run := func(target string) {
		dec, err := svc.Sync(ctx, provision.SyncInput{Actor: "owner", NodeID: "n1", TargetRevision: target})
		require.NoError(t, err)
		_, _ = svc.AppendEvent(ctx, provision.ProvisionEvent{OpID: dec.OpID, Kind: provision.EventStatus, Sequence: 1, Status: "running"})
		_, _ = svc.AppendEvent(ctx, provision.ProvisionEvent{OpID: dec.OpID, Kind: provision.EventVersion, Sequence: 2, Revision: target})
		_, err = svc.AppendEvent(ctx, provision.ProvisionEvent{OpID: dec.OpID, Kind: provision.EventExit, Sequence: 3, ExitCode: 0})
		require.NoError(t, err)
		op, _, _ := svc.GetOp(ctx, dec.OpID)
		require.Equal(t, provision.StatusCompleted, op.Status)
	}

	run("rev-B")
	run("rev-B") // re-run: same target, same converged end state

	ver, err := svc.GetNodeVersion(ctx, "n1")
	require.NoError(t, err)
	require.Equal(t, "rev-B", ver.Revision)
}

// [REQ:BRG-P0-006] GetNodeVersion reports the node's resulting version; a
// never-provisioned node returns ErrNoNodeVersion.
func TestGetNodeVersion_NeverProvisioned(t *testing.T) {
	svc, _, _, _, _, _ := newService(t)
	_, err := svc.GetNodeVersion(context.Background(), "n1")
	var none provision.ErrNoNodeVersion
	require.ErrorAs(t, err, &none)
}
