package graph_test

import (
	"context"
	"testing"

	graphh "architecture-cartographer/handlers/graph"
	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/graph/mocks"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/graph"
	"github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/graph/graph_v1connect"
)

func sampleSnap() graph.GraphSnapshot {
	return graph.GraphSnapshot{
		ID:          "snap:demo:h1",
		Scenario:    "demo",
		ContentHash: "h1",
		Languages:   []graph.Language{graph.LanguageGo},
		Files: []graph.FileNode{
			{ID: "f1", Path: "a.go", PackageID: "p1", Language: graph.LanguageGo},
		},
		Packages: []graph.PackageNode{{ID: "p1", ImportPath: "demo/a", Language: graph.LanguageGo}},
		Imports:  []graph.ImportEdge{{From: "p1", ToPackageID: "p2"}},
	}
}

func TestHandler_ExtractGraph_HappyPath(t *testing.T) {
	svc := &mocks.FakeService{Snapshots: []graph.GraphSnapshot{sampleSnap()}, FromCache: true}
	h := graphh.NewHandler(svc)

	resp, err := h.ExtractGraph(context.Background(), connect.NewRequest(&graphv1.ExtractGraphRequest{
		Scenario:  "demo",
		Languages: []graphv1.Language{graphv1.Language_LANGUAGE_GO},
	}))
	require.NoError(t, err)
	require.Equal(t, "snap:demo:h1", resp.Msg.GetSnapshot().GetId())
	require.True(t, resp.Msg.GetFromCache())
	require.Equal(t, int64(1), svc.ExtractCalls.Load())
}

func TestHandler_ExtractGraph_RejectsMissingScenario(t *testing.T) {
	h := graphh.NewHandler(&mocks.FakeService{})
	_, err := h.ExtractGraph(context.Background(), connect.NewRequest(&graphv1.ExtractGraphRequest{}))
	require.Error(t, err)
	var ce *connect.Error
	require.ErrorAs(t, err, &ce)
	require.Equal(t, connect.CodeInvalidArgument, ce.Code())
}

func TestHandler_GetSnapshot_NotFound(t *testing.T) {
	h := graphh.NewHandler(&mocks.FakeService{})
	_, err := h.GetGraphSnapshot(context.Background(), connect.NewRequest(&graphv1.GetGraphSnapshotRequest{Id: "missing"}))
	require.Error(t, err)
	var ce *connect.Error
	require.ErrorAs(t, err, &ce)
	require.Equal(t, connect.CodeNotFound, ce.Code())
}

func TestHandler_ClearGraphSnapshots_DryRunHonoured(t *testing.T) {
	svc := &mocks.FakeService{Snapshots: []graph.GraphSnapshot{sampleSnap()}}
	h := graphh.NewHandler(svc)

	resp, err := h.ClearGraphSnapshots(context.Background(), connect.NewRequest(&graphv1.ClearGraphSnapshotsRequest{
		Scenario: "demo",
		DryRun:   true,
	}))
	require.NoError(t, err)
	require.True(t, resp.Msg.GetDryRun())
	require.Equal(t, int32(1), resp.Msg.GetDeleted())
	require.Len(t, svc.Snapshots, 1, "dry-run must not delete")
}

func TestHandler_ClearGraphSnapshots_HeaderDryRun(t *testing.T) {
	svc := &mocks.FakeService{Snapshots: []graph.GraphSnapshot{sampleSnap()}}
	h := graphh.NewHandler(svc)
	req := connect.NewRequest(&graphv1.ClearGraphSnapshotsRequest{Scenario: "demo"})
	req.Header().Set("X-Dry-Run", "true")
	resp, err := h.ClearGraphSnapshots(context.Background(), req)
	require.NoError(t, err)
	require.True(t, resp.Msg.GetDryRun(), "X-Dry-Run header must trigger dry-run")
	require.Len(t, svc.Snapshots, 1)
}

func TestHandler_ListGraphSnapshots(t *testing.T) {
	svc := &mocks.FakeService{Snapshots: []graph.GraphSnapshot{sampleSnap()}}
	h := graphh.NewHandler(svc)

	resp, err := h.ListGraphSnapshots(context.Background(), connect.NewRequest(&graphv1.ListGraphSnapshotsRequest{Scenario: "demo"}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.GetSnapshots(), 1)
}

func TestHandler_ExportGraph(t *testing.T) {
	svc := &mocks.FakeService{Snapshots: []graph.GraphSnapshot{sampleSnap()}}
	h := graphh.NewHandler(svc)
	resp, err := h.ExportGraph(context.Background(), connect.NewRequest(&graphv1.ExportGraphRequest{Id: "snap:demo:h1"}))
	require.NoError(t, err)
	require.Equal(t, "application/json", resp.Msg.GetContentType())
	require.NotEmpty(t, resp.Msg.GetPayload())
}

func TestHandler_InterfaceSatisfied(t *testing.T) {
	var _ graph_v1connect.GraphServiceHandler = (*graphh.Handler)(nil)
}
