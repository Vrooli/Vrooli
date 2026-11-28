package dispatch_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/scheduletest"

	localdb "vrooli-bridge/internal/database"
	"vrooli-bridge/internal/dispatch"
	"vrooli-bridge/internal/runs"
	"vrooli-bridge/internal/targetreceipt"
)

const localFixtureNode = "local-contract-agent-linux"

type localNodeReader struct{}

func (localNodeReader) GetTarget(_ context.Context, id string) (dispatch.TargetNode, error) {
	if id != localFixtureNode {
		return dispatch.TargetNode{}, dispatch.ErrNodeNotFound{ID: id}
	}
	return dispatch.TargetNode{ID: id, Kind: "agent", OS: "linux", Arch: "amd64", Scopes: []string{"vrooli:read", "vrooli-bridge:read"}}, nil
}

type localPresence struct{}

func (localPresence) IsOnline(id string) bool     { return id == localFixtureNode }
func (localPresence) Dispatchable(id string) bool { return id == localFixtureNode }

type durableRunController struct{ service runs.Service }

func (c durableRunController) CreateRun(ctx context.Context, in dispatch.CreateRunInput) (dispatch.CreateRunResult, error) {
	run, err := c.service.Create(ctx, runs.CreateInput{
		OwnerID: in.OwnerID, RequestID: in.RequestID, RequestFingerprint: in.RequestFingerprint,
		NodeID: in.NodeID, Scenario: in.Scenario, Verb: in.Verb, Args: in.Args,
		TimeoutSeconds: in.TimeoutSeconds,
	})
	return dispatch.CreateRunResult{RunID: run.ID, Replayed: run.IdempotentReplay}, err
}

func (c durableRunController) AbortRun(ctx context.Context, id, reason string) error {
	_, err := c.service.Abort(ctx, id, reason)
	return err
}

type localReachableTargetAdapter struct{ runs runs.Service }

func (a localReachableTargetAdapter) PushJob(ctx context.Context, nodeID string, job dispatch.PushedJob) (int, error) {
	if nodeID != localFixtureNode || job.Verb != "scenario status" || job.Scenario != "vrooli-bridge" {
		return 0, dispatch.ErrDeliveryFailed{NodeID: nodeID}
	}
	for seq, event := range []runs.RunEvent{
		{RunID: job.RunID, Kind: runs.EventStatus, Status: "running"},
		{RunID: job.RunID, Kind: runs.EventLog, LogChunk: "fixture target accepted read-only scenario status\n"},
		{RunID: job.RunID, Kind: runs.EventExit, ExitCode: 0},
	} {
		event.Sequence = uint64(seq + 1)
		if _, err := a.runs.AppendEvent(ctx, event); err != nil {
			return 0, err
		}
	}
	return 1, nil
}

type discardAudit struct{}

func (discardAudit) Record(context.Context, dispatch.Entry) error { return nil }

// This is a local contract fixture, not a real-node certification. It exercises
// the typed Bridge dispatch seam, reaches a terminal target result, persists it
// through the existing SQLite runs path, and reattaches a receipt that is
// explicitly ineligible for all real target lanes.
func TestLocalReachableTargetOperationReattachesTypedNonQualifyingReceipt(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := databasetest.NewSQLite(t)
	clk := scheduletest.New(time.Date(2026, 11, 19, 17, 0, 0, 0, time.UTC))
	require.NoError(t, database.EnsureSchemas(ctx, db,
		database.SchemaProviderFunc(localdb.SystemSchema),
		database.SchemaProviderFunc(runs.Schema),
	))
	runService := runs.NewService(runs.NewSQLiteRepository(db, clk), clk)
	manifest, _, err := dispatch.BuildManifest()
	require.NoError(t, err)
	service := dispatch.NewService(localNodeReader{}, localPresence{}, durableRunController{service: runService}, discardAudit{}, localReachableTargetAdapter{runs: runService}, dispatch.WithManifest(manifest))

	decision, err := service.Dispatch(ctx, dispatch.DispatchInput{Actor: "br3-local-contract-test", Job: dispatch.Job{
		NodeID: localFixtureNode, Scenario: "vrooli-bridge", Verb: "scenario status", TimeoutSeconds: 30,
	}})
	require.NoError(t, err)
	require.NotEmpty(t, decision.RunID)

	// Reconstruct from the persisted run and event history, just as a later
	// reader would after the original dispatch client has gone away.
	receipt, err := targetreceipt.ReattachLocal(ctx, runService, decision.RunID)
	require.NoError(t, err)
	require.Equal(t, targetreceipt.SchemaVersion, receipt.SchemaVersion)
	require.Equal(t, targetreceipt.LaneLocalContract, receipt.Lane)
	require.Equal(t, localFixtureNode, receipt.TargetID)
	require.Equal(t, targetreceipt.ClassLocalFixture, receipt.TargetClass)
	require.Equal(t, targetreceipt.ProvenanceLocal, receipt.Provenance)
	require.Equal(t, decision.RunID, receipt.RunID)
	require.Equal(t, "passed", receipt.Status)
	require.EqualValues(t, 0, receipt.ExitCode)
	require.Equal(t, []string{"bridge-run://" + decision.RunID}, receipt.EvidenceRefs)
	require.False(t, receipt.CapturedAt.IsZero())
	require.False(t, receipt.LaneEligible, "a local fixture must never satisfy a QEMU/VPS/native lane")
}
