package dispatch_test

import (
	"context"
	"errors"
	"testing"

	"vrooli-bridge/internal/dispatch"
	"vrooli-bridge/internal/dispatch/mocks"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
)

func newSvc(t *testing.T) (dispatch.Service, *mocks.FakeNodeReader, *mocks.FakePresence, *mocks.FakeRunController, *mocks.FakeAuditSink, *mocks.FakeJobPusher) {
	t.Helper()
	nodes := &mocks.FakeNodeReader{Nodes: map[string]dispatch.TargetNode{
		"n1": {ID: "n1", OS: "linux", Arch: "amd64", Scopes: []string{"scenario test*"}},
	}}
	presence := &mocks.FakePresence{Online: map[string]bool{"n1": true}}
	runsCtl := &mocks.FakeRunController{NextRunID: "run-1"}
	sink := &mocks.FakeAuditSink{}
	pusher := &mocks.FakeJobPusher{}
	svc := dispatch.NewService(nodes, presence, runsCtl, sink, pusher)
	return svc, nodes, presence, runsCtl, sink, pusher
}

func job() dispatch.Job {
	return dispatch.Job{NodeID: "n1", Scenario: "web-search", Verb: "scenario test", Args: []string{"web-search"}}
}

// [REQ:BRG-P0-004] The happy path: a valid in-scope job creates a durable run,
// audits the dispatch as accepted, and pushes the typed job to the node.
func TestDispatch_HappyPath(t *testing.T) {
	svc, _, _, runsCtl, sink, pusher := newSvc(t)

	dec, err := svc.Dispatch(context.Background(), dispatch.DispatchInput{Actor: "owner-1", Job: job()})
	require.NoError(t, err)
	require.Equal(t, "run-1", dec.RunID)
	require.False(t, dec.DryRun)

	require.Len(t, runsCtl.Created, 1, "a durable run is created")
	require.Equal(t, "scenario test", runsCtl.Created[0].Verb)

	recorded := sink.Recorded()
	require.Len(t, recorded, 1)
	require.True(t, recorded[0].Accepted, "the dispatch is audited as accepted")
	require.Equal(t, "run-1", recorded[0].RunID)
	require.Equal(t, "owner-1", recorded[0].Actor)

	pushed := pusher.PushedJobs()
	require.Len(t, pushed, 1, "the typed job is pushed to the node")
	require.Equal(t, "run-1", pushed[0].RunID)
	require.Equal(t, []string{"web-search"}, pushed[0].Args)
}

// [REQ:BRG-P1-001] A node that is online but whose agent protocol version is
// flagged (needs-update / incompatible) is EXCLUDED from dispatch: the job is
// rejected (FailedPrecondition) and audited as rejected before any run is
// created or anything is pushed. Provisioning is exempt; only work is gated.
func TestDispatch_ProtocolIncompatibleNode_Excluded(t *testing.T) {
	svc, _, presence, runsCtl, sink, pusher := newSvc(t)
	presence.Flagged = map[string]bool{"n1": true} // online but needs update

	_, err := svc.Dispatch(context.Background(), dispatch.DispatchInput{Actor: "owner-1", Job: job()})
	require.Error(t, err)
	var needsUpdate dispatch.ErrNodeNeedsUpdate
	require.ErrorAs(t, err, &needsUpdate)
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(dispatch.ToConnectError(err)))

	require.Empty(t, runsCtl.Created, "no run created for a flagged node")
	require.Empty(t, pusher.PushedJobs(), "nothing pushed to a flagged node")
	recorded := sink.Recorded()
	require.Len(t, recorded, 1)
	require.False(t, recorded[0].Accepted, "the exclusion is audited as rejected")
}

// [REQ:BRG-P0-004] An out-of-scope verb is rejected (PermissionDenied) and
// audited as rejected BEFORE any run is created or anything is pushed.
func TestDispatch_OutOfScope_RejectedAndAudited(t *testing.T) {
	svc, nodes, _, runsCtl, sink, pusher := newSvc(t)
	nodes.Nodes["n1"] = dispatch.TargetNode{ID: "n1", Scopes: []string{"scenario status*"}}

	_, err := svc.Dispatch(context.Background(), dispatch.DispatchInput{Actor: "owner-1", Job: job()})
	require.Error(t, err)
	var outOfScope dispatch.ErrVerbOutOfScope
	require.True(t, errors.As(err, &outOfScope))

	require.Empty(t, runsCtl.Created, "no run is created on rejection")
	require.Empty(t, pusher.PushedJobs(), "nothing is pushed on rejection")
	recorded := sink.Recorded()
	require.Len(t, recorded, 1)
	require.False(t, recorded[0].Accepted, "the denial is audited")
	require.Contains(t, recorded[0].Detail, "outside this node's granted scopes")
}

// [REQ:BRG-P0-004] A revoked node can run nothing; the attempt is audited.
func TestDispatch_RevokedNode_Rejected(t *testing.T) {
	svc, nodes, _, runsCtl, sink, _ := newSvc(t)
	nodes.Nodes["n1"] = dispatch.TargetNode{ID: "n1", Scopes: []string{"scenario test*"}, Revoked: true}

	_, err := svc.Dispatch(context.Background(), dispatch.DispatchInput{Actor: "o", Job: job()})
	var revoked dispatch.ErrNodeRevoked
	require.True(t, errors.As(err, &revoked))
	require.Empty(t, runsCtl.Created)
	require.Len(t, sink.Recorded(), 1)
	require.False(t, sink.Recorded()[0].Accepted)
}

// [REQ:BRG-P0-004] An unknown node is a NotFound; no audit, no run.
func TestDispatch_UnknownNode(t *testing.T) {
	svc, _, _, runsCtl, sink, _ := newSvc(t)
	j := job()
	j.NodeID = "ghost"
	_, err := svc.Dispatch(context.Background(), dispatch.DispatchInput{Actor: "o", Job: j})
	var notFound dispatch.ErrNodeNotFound
	require.True(t, errors.As(err, &notFound))
	require.Empty(t, runsCtl.Created)
	require.Empty(t, sink.Recorded())
}

// [REQ:BRG-P0-004] An offline node is a precondition failure; audited, no run.
func TestDispatch_OfflineNode_Rejected(t *testing.T) {
	svc, _, presence, runsCtl, sink, _ := newSvc(t)
	presence.Online["n1"] = false

	_, err := svc.Dispatch(context.Background(), dispatch.DispatchInput{Actor: "o", Job: job()})
	var offline dispatch.ErrNodeOffline
	require.True(t, errors.As(err, &offline))
	require.Empty(t, runsCtl.Created)
	require.Len(t, sink.Recorded(), 1)
	require.Contains(t, sink.Recorded()[0].Detail, "offline")
}

// [REQ:BRG-P0-004] A dry-run validates the job and target but creates no run,
// writes no audit, and pushes nothing.
func TestDispatch_DryRun_NoSideEffects(t *testing.T) {
	svc, _, _, runsCtl, sink, pusher := newSvc(t)

	dec, err := svc.Dispatch(context.Background(), dispatch.DispatchInput{Actor: "o", Job: job(), DryRun: true})
	require.NoError(t, err)
	require.True(t, dec.DryRun)
	require.Empty(t, dec.RunID)
	require.Empty(t, runsCtl.Created)
	require.Empty(t, sink.Recorded())
	require.Empty(t, pusher.PushedJobs())
}

// [REQ:BRG-P0-004] A dry-run of an out-of-scope verb still fails validation.
func TestDispatch_DryRun_RejectsInvalid(t *testing.T) {
	svc, nodes, _, _, _, _ := newSvc(t)
	nodes.Nodes["n1"] = dispatch.TargetNode{ID: "n1", Scopes: []string{"scenario status*"}}
	_, err := svc.Dispatch(context.Background(), dispatch.DispatchInput{Actor: "o", Job: job(), DryRun: true})
	require.Error(t, err)
}

// [REQ:BRG-P0-004] If the job cannot be delivered (node dropped between the
// online check and the push), the created run is aborted and the dispatch fails.
func TestDispatch_DeliveryFailure_AbortsRun(t *testing.T) {
	svc, _, _, runsCtl, _, pusher := newSvc(t)
	pusher.DeliveredSet = true
	pusher.Delivered = 0 // the push reaches no live connection

	_, err := svc.Dispatch(context.Background(), dispatch.DispatchInput{Actor: "o", Job: job()})
	var delivery dispatch.ErrDeliveryFailed
	require.True(t, errors.As(err, &delivery))
	require.Equal(t, []string{"run-1"}, runsCtl.Aborted, "the orphaned run is aborted")
}

// [REQ:BRG-P0-004][REQ:BRG-P0-008] Audit is fail-closed on the accepted path: if
// the dispatch cannot be recorded, the run is aborted and the dispatch refuses
// rather than running un-audited.
func TestDispatch_AuditFailClosed(t *testing.T) {
	svc, _, _, runsCtl, sink, pusher := newSvc(t)
	sink.RecordErr = errors.New("substrate down")

	_, err := svc.Dispatch(context.Background(), dispatch.DispatchInput{Actor: "o", Job: job()})
	require.Error(t, err, "dispatch refuses when it cannot audit")
	require.Equal(t, []string{"run-1"}, runsCtl.Aborted, "the un-auditable run is aborted")
	require.Empty(t, pusher.PushedJobs(), "nothing is pushed when audit fails closed")
}
