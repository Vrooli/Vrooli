package provision_test

import (
	"context"
	"testing"

	"vrooli-bridge/internal/compat"
	"vrooli-bridge/internal/provision"

	"github.com/stretchr/testify/require"
)

// drive feeds an op the node-reported event stream that ends a provisioning run,
// returning the terminal op. It mirrors what handlers/provision does when the
// node's privileged helper streams ProvisionEvents back.
func drive(t *testing.T, svc provision.Service, opID string, events ...provision.ProvisionEvent) provision.ProvisioningOp {
	t.Helper()
	for _, ev := range events {
		ev.OpID = opID
		accepted, err := svc.AppendEvent(context.Background(), ev)
		require.NoError(t, err)
		require.True(t, accepted)
	}
	op, _, err := svc.GetOp(context.Background(), opID)
	require.NoError(t, err)
	return op
}

// [REQ:BRG-P1-001] A failed setup during an update restores the node's prior
// revision: the op reaches ROLLED_BACK (the SAFE failure) and the recorded node
// version is the rollback revision, not the failed target.
func TestRollback_FailedSetupRestoresPriorRevision(t *testing.T) {
	svc, repo, _, _, _, _ := newService(t)
	repo.SeedVersion(provision.NodeVersion{NodeID: "n1", Revision: "rev-A"})

	dec, err := svc.Sync(context.Background(), provision.SyncInput{
		Actor: "owner", NodeID: "n1", TargetRevision: "rev-B",
	})
	require.NoError(t, err)
	require.Equal(t, "rev-A", dec.RollbackRevision, "rollback resolves to the node's prior version")

	// The node's helper: fetch+setup target fails, helper rolls back to rev-A,
	// re-validates, and exits non-zero to report the update did not take.
	op := drive(t, svc, dec.OpID,
		provision.ProvisionEvent{Kind: provision.EventStatus, Sequence: 1, Status: "running setup"},
		provision.ProvisionEvent{Kind: provision.EventStatus, Sequence: 2, Status: "setup failed; rolling back"},
		provision.ProvisionEvent{Kind: provision.EventVersion, Sequence: 3, Revision: "rev-A"},
		provision.ProvisionEvent{Kind: provision.EventExit, Sequence: 4, ExitCode: 1},
	)

	require.Equal(t, provision.StatusRolledBack, op.Status)
	require.Equal(t, "rev-A", op.ResultingRevision)

	// The node's recorded version is the restored prior revision.
	v, err := svc.GetNodeVersion(context.Background(), "n1")
	require.NoError(t, err)
	require.Equal(t, "rev-A", v.Revision)
}

// [REQ:BRG-P1-001] A clean setup reaches COMPLETED and records the new target as
// the node's version (the success contrast to rollback).
func TestRollback_CleanSetupCompletes(t *testing.T) {
	svc, repo, _, _, _, _ := newService(t)
	repo.SeedVersion(provision.NodeVersion{NodeID: "n1", Revision: "rev-A"})

	dec, err := svc.Sync(context.Background(), provision.SyncInput{
		Actor: "owner", NodeID: "n1", TargetRevision: "rev-B",
	})
	require.NoError(t, err)

	op := drive(t, svc, dec.OpID,
		provision.ProvisionEvent{Kind: provision.EventStatus, Sequence: 1, Status: "running setup"},
		provision.ProvisionEvent{Kind: provision.EventVersion, Sequence: 2, Revision: "rev-B"},
		provision.ProvisionEvent{Kind: provision.EventExit, Sequence: 3, ExitCode: 0},
	)

	require.Equal(t, provision.StatusCompleted, op.Status)
	v, err := svc.GetNodeVersion(context.Background(), "n1")
	require.NoError(t, err)
	require.Equal(t, "rev-B", v.Revision)
}

// [REQ:BRG-P1-001] A failed setup that could NOT roll back (no resulting
// rollback revision) reaches FAILED (degraded; needs operator attention) — the
// distinction the terminal disposition draws from the reported version.
func TestRollback_FailedWithoutRollbackIsDegraded(t *testing.T) {
	svc, _, _, _, _, _ := newService(t)

	// First provision (no prior version): nothing to roll back to.
	dec, err := svc.Sync(context.Background(), provision.SyncInput{
		Actor: "owner", NodeID: "n1", TargetRevision: "rev-B",
	})
	require.NoError(t, err)
	require.Empty(t, dec.RollbackRevision)

	op := drive(t, svc, dec.OpID,
		provision.ProvisionEvent{Kind: provision.EventStatus, Sequence: 1, Status: "running setup"},
		provision.ProvisionEvent{Kind: provision.EventExit, Sequence: 2, ExitCode: 1},
	)
	require.Equal(t, provision.StatusFailed, op.Status)
}

// [REQ:BRG-P1-001] Protocol-compatibility gating: a node whose agent protocol
// version is older-than-current (but within the supported window) is classified
// NEEDS_UPDATE — the verdict the live layer stores and dispatch reads to EXCLUDE
// the node from work (see dispatch's needs-update gate) rather than mis-drive
// it. Provisioning (this domain) is exempt: bringing the node to a new revision
// is precisely how the agent is updated.
func TestRollback_ProtocolIncompatibleNodeFlaggedNeedsUpdate(t *testing.T) {
	// current=3, min=2: a v2 node is drivable-for-presence-only (needs update);
	// a v1 node is below the floor (incompatible); a v3+ node is OK.
	require.Equal(t, compat.StatusNeedsUpdate, compat.EvaluateAt(2, 3, 2))
	require.Equal(t, compat.StatusIncompatible, compat.EvaluateAt(1, 3, 2))
	require.Equal(t, compat.StatusOK, compat.EvaluateAt(3, 3, 2))

	// A flagged verdict is NOT dispatchable; an OK/Unspecified one is.
	require.False(t, compat.StatusNeedsUpdate.Dispatchable())
	require.False(t, compat.StatusIncompatible.Dispatchable())
	require.True(t, compat.StatusOK.Dispatchable())
	require.False(t, compat.StatusUnspecified.Dispatchable())
}
