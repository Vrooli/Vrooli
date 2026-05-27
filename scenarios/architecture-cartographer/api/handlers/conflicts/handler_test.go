package conflicts_test

import (
	"context"
	"testing"

	conflictsh "architecture-cartographer/handlers/conflicts"
	"architecture-cartographer/internal/conflicts"
	conflictmocks "architecture-cartographer/internal/conflicts/mocks"
	"architecture-cartographer/internal/domains"
	domainmocks "architecture-cartographer/internal/domains/mocks"
	"architecture-cartographer/internal/graph"
	graphmocks "architecture-cartographer/internal/graph/mocks"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	conflictsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/conflicts"
	"github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/conflicts/conflicts_v1connect"
)

func newHandler(t *testing.T) (*conflictsh.Handler, *conflictmocks.FakeService, *graphmocks.FakeService, *domainmocks.FakeService) {
	t.Helper()
	cs := &conflictmocks.FakeService{}
	gs := &graphmocks.FakeService{Snapshots: []graph.GraphSnapshot{{ID: "snap:demo:h1", Scenario: "demo"}}}
	ds := &domainmocks.FakeService{Map: domains.DerivedDomainMap{Scenario: "demo"}}
	h := conflictsh.NewHandler(conflictsh.Deps{Conflicts: cs, Graph: gs, Domains: ds})
	return h, cs, gs, ds
}

func TestHandler_DetectConflicts_FetchesSnapshotAndDomainMap(t *testing.T) {
	h, cs, gs, ds := newHandler(t)
	cs.Conflicts = []conflicts.Conflict{{ID: "c-1", Type: "cycle", Severity: conflicts.SeverityError}}

	resp, err := h.DetectConflicts(context.Background(), connect.NewRequest(&conflictsv1.DetectConflictsRequest{Scenario: "demo"}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.GetConflicts(), 1)
	require.Equal(t, int64(1), gs.ExtractCalls.Load(), "Graph.Extract should be called when no snapshot_id")
	require.Equal(t, int64(1), ds.GetCalls.Load(), "Domains.GetDomainMap should be called")
	require.Equal(t, int64(1), cs.DetectCalls.Load())
}

func TestHandler_DetectConflicts_UsesSnapshotIdWhenProvided(t *testing.T) {
	h, _, gs, _ := newHandler(t)
	_, err := h.DetectConflicts(context.Background(), connect.NewRequest(&conflictsv1.DetectConflictsRequest{
		Scenario:   "demo",
		SnapshotId: "snap:demo:h1",
	}))
	require.NoError(t, err)
	require.Equal(t, int64(0), gs.ExtractCalls.Load(), "Extract must not be called when snapshot_id supplied")
	require.Equal(t, int64(1), gs.GetCalls.Load(), "Get must be called with the supplied id")
}

func TestHandler_DetectConflicts_RejectsMissingScenario(t *testing.T) {
	h, _, _, _ := newHandler(t)
	_, err := h.DetectConflicts(context.Background(), connect.NewRequest(&conflictsv1.DetectConflictsRequest{}))
	require.Error(t, err)
	var ce *connect.Error
	require.ErrorAs(t, err, &ce)
	require.Equal(t, connect.CodeInvalidArgument, ce.Code())
}

func TestHandler_ListConflicts_PassesFilter(t *testing.T) {
	h, cs, _, _ := newHandler(t)
	cs.Conflicts = []conflicts.Conflict{{ID: "c-1", Type: "cycle"}}
	resp, err := h.ListConflicts(context.Background(), connect.NewRequest(&conflictsv1.ListConflictsRequest{
		Scenario: "demo",
		Statuses: []conflictsv1.ResolutionStatus{conflictsv1.ResolutionStatus_RESOLUTION_STATUS_DETECTED},
	}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.GetConflicts(), 1)
}

func TestHandler_AssignConflict_DryRunHonored(t *testing.T) {
	h, cs, _, _ := newHandler(t)
	resp, err := h.AssignConflict(context.Background(), connect.NewRequest(&conflictsv1.AssignConflictRequest{
		Id: "c-1", Domain: "graph", DryRun: true,
	}))
	require.NoError(t, err)
	require.True(t, resp.Msg.GetDryRun())
	require.Equal(t, int64(1), cs.AssignCalls.Load())
}

func TestHandler_ResolveConflict_ForceRoundTrips(t *testing.T) {
	h, _, _, _ := newHandler(t)
	resp, err := h.ResolveConflict(context.Background(), connect.NewRequest(&conflictsv1.ResolveConflictRequest{
		Id: "c-1", Force: true,
	}))
	require.NoError(t, err)
	require.Equal(t, conflictsv1.ResolutionStatus_RESOLUTION_STATUS_FORCE_RESOLVED, resp.Msg.GetConflict().GetStatus())
}

func TestHandler_ValidateConflicts_RejectsMissingScenario(t *testing.T) {
	h, _, _, _ := newHandler(t)
	_, err := h.ValidateConflicts(context.Background(), connect.NewRequest(&conflictsv1.ValidateConflictsRequest{}))
	require.Error(t, err)
}

func TestHandler_ValidateConflicts_ReturnsCleanWhenEmpty(t *testing.T) {
	h, _, _, _ := newHandler(t)
	resp, err := h.ValidateConflicts(context.Background(), connect.NewRequest(&conflictsv1.ValidateConflictsRequest{Scenario: "demo"}))
	require.NoError(t, err)
	require.True(t, resp.Msg.GetClean())
	require.Empty(t, resp.Msg.GetConflicts())
}

func TestHandler_ListDetectorsAndResolvers(t *testing.T) {
	h, cs, _, _ := newHandler(t)
	cs.Detectors = []conflicts.DetectorDescriptor{{Name: "cycle"}}
	cs.Resolvers = []conflicts.ResolverDescriptor{{Name: "mislocated_file", RequiresApply: true}}

	d, err := h.ListDetectors(context.Background(), connect.NewRequest(&conflictsv1.ListDetectorsRequest{}))
	require.NoError(t, err)
	require.Len(t, d.Msg.GetDetectors(), 1)

	r, err := h.ListResolvers(context.Background(), connect.NewRequest(&conflictsv1.ListResolversRequest{}))
	require.NoError(t, err)
	require.True(t, r.Msg.GetResolvers()[0].GetRequiresApply())
}

func TestHandler_InterfaceSatisfied(t *testing.T) {
	var _ conflicts_v1connect.ConflictsServiceHandler = (*conflictsh.Handler)(nil)
}
