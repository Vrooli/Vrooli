package validation

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	baselinesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/baselines"
	baselinesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/baselines/baselines_v1connect"
	internalvalidation "plan-manager/internal/validation"
)

type fixedGCTResolver struct{ url string }

func (r fixedGCTResolver) ResolveScenarioURLDefault(context.Context, string) (string, error) {
	return r.url, nil
}

type gctCollectionStub struct {
	baselinesconnect.UnimplementedBaselinesServiceHandler
	request     *baselinesv1.StartCollectionCaptureRequest
	pending     bool
	waits       int
	pathCapture *baselinesv1.CapturePathSnapshotRequest
	pathDiff    *baselinesv1.DiffPathSnapshotsRequest
}

func (s *gctCollectionStub) GetPathSnapshot(_ context.Context, req *connect.Request[baselinesv1.GetPathSnapshotRequest]) (*connect.Response[baselinesv1.GetPathSnapshotResponse], error) {
	return connect.NewResponse(&baselinesv1.GetPathSnapshotResponse{Snapshot: &baselinesv1.PathSnapshot{Name: req.Msg.GetName(), Branch: "agi", Selections: []string{"packages/proto/**"}}}), nil
}

func (s *gctCollectionStub) CapturePathSnapshot(_ context.Context, req *connect.Request[baselinesv1.CapturePathSnapshotRequest]) (*connect.Response[baselinesv1.CapturePathSnapshotResponse], error) {
	s.pathCapture = req.Msg
	return connect.NewResponse(&baselinesv1.CapturePathSnapshotResponse{Snapshot: &baselinesv1.PathSnapshot{Name: req.Msg.GetName(), Branch: "agi"}}), nil
}

func (s *gctCollectionStub) DiffPathSnapshots(_ context.Context, req *connect.Request[baselinesv1.DiffPathSnapshotsRequest]) (*connect.Response[baselinesv1.DiffPathSnapshotsResponse], error) {
	s.pathDiff = req.Msg
	return connect.NewResponse(&baselinesv1.DiffPathSnapshotsResponse{Classification: "informational-source-evidence", Deltas: []*baselinesv1.SourceDelta{{Path: "packages/proto/a.proto", Status: "modified"}}}), nil
}

func (s *gctCollectionStub) StartCollectionCapture(_ context.Context, req *connect.Request[baselinesv1.StartCollectionCaptureRequest]) (*connect.Response[baselinesv1.StartCollectionCaptureResponse], error) {
	s.request = req.Msg
	ready := int32(2)
	if s.pending {
		ready = 0
	}
	return connect.NewResponse(&baselinesv1.StartCollectionCaptureResponse{Collection: collectionFixture(req.Msg.GetName(), ready, 2, !s.pending)}), nil
}

func (s *gctCollectionStub) GetCollectionStatus(_ context.Context, req *connect.Request[baselinesv1.GetCollectionStatusRequest]) (*connect.Response[baselinesv1.GetCollectionStatusResponse], error) {
	return connect.NewResponse(&baselinesv1.GetCollectionStatusResponse{Collection: collectionFixture(req.Msg.GetName(), 2, 0, true)}), nil
}

func collectionFixture(name string, ready, pending int32, complete bool) *baselinesv1.BaselineCollection {
	return &baselinesv1.BaselineCollection{
		Name: name, Branch: "agi", Coverage: &baselinesv1.CollectionCoverage{Required: 2, Ready: ready, Pending: pending, Complete: complete},
		Members:       []*baselinesv1.CollectionMember{{Scenario: "git-control-tower", BaselineName: name, Required: true, Status: "ready", RunId: "run-gct"}, {Scenario: "plan-manager", BaselineName: name, Required: true, Status: "ready", RunId: "run-pm"}},
		PathSnapshots: []*baselinesv1.PathSnapshotReference{{Name: "paths-before", Branch: "agi", CreatedAt: "2026-07-13T00:00:00Z"}},
	}
}

func TestGCTCollectionClientReturnsPendingCaptureWithoutWaiting(t *testing.T) {
	stub := &gctCollectionStub{pending: true}
	path, handler := baselinesconnect.NewBaselinesServiceHandler(stub)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()
	client := gctCollectionClient{resolver: fixedGCTResolver{url: server.URL}, http: server.Client()}
	result, err := client.StartCollectionCapture(context.Background(), internalvalidation.BaselineCollectionCaptureRequest{Name: "before", Scenarios: []string{"git-control-tower", "plan-manager"}})
	require.NoError(t, err)
	require.False(t, result.Complete())
	require.Equal(t, 0, stub.waits)
	require.Equal(t, "agi", result.Branch)
	require.Equal(t, "run-gct", result.Members[0].RunID)
	require.Equal(t, "paths-before", result.PathSnapshots[0].Name)
}

func TestGCTCollectionClientUsesDiscoveredTypedAPI(t *testing.T) {
	stub := &gctCollectionStub{}
	path, handler := baselinesconnect.NewBaselinesServiceHandler(stub)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()
	client := gctCollectionClient{resolver: fixedGCTResolver{url: server.URL}, http: server.Client()}
	result, err := client.StartCollectionCapture(context.Background(), internalvalidation.BaselineCollectionCaptureRequest{Name: "before", Scenarios: []string{"git-control-tower", "plan-manager"}})
	require.NoError(t, err)
	require.True(t, result.Complete())
	require.NotNil(t, stub.request)
	require.Equal(t, []string{"git-control-tower", "plan-manager"}, []string{stub.request.GetTargets()[0].GetScenario(), stub.request.GetTargets()[1].GetScenario()})
}

func TestGCTCollectionClientCapturesAndDiffsTypedPathEvidence(t *testing.T) {
	stub := &gctCollectionStub{}
	path, handler := baselinesconnect.NewBaselinesServiceHandler(stub)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()
	client := gctCollectionClient{resolver: fixedGCTResolver{url: server.URL}, http: server.Client()}
	result, err := client.DiffPathEvidence(context.Background(), internalvalidation.BaselinePathDiffRequest{BeforeName: "before", Branch: "agi", Paths: []string{"packages/proto/**"}, OperationID: "op:1"})
	require.NoError(t, err)
	require.Equal(t, 1, result.Deltas)
	require.Equal(t, []string{"packages/proto/**"}, stub.pathCapture.GetSelections())
	require.Equal(t, []string{"packages/proto/**"}, stub.pathDiff.GetSelections())
	require.Contains(t, result.AfterName, "before-after-op-1")
}
