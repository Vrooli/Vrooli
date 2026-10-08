package graph_test

import (
	"context"
	"testing"

	graphh "architecture-cartographer/handlers/graph"
	intdomains "architecture-cartographer/internal/domains"
	domainmocks "architecture-cartographer/internal/domains/mocks"
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
			{ID: "f1", Path: "api/internal/billing/service.go", PackageID: "p1", Language: graph.LanguageGo},
		},
		Packages: []graph.PackageNode{{ID: "p1", ImportPath: "demo/billing", RepoPath: "api/internal/billing", Language: graph.LanguageGo}},
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

func TestHandler_GetZoneMap(t *testing.T) {
	snap := sampleSnap()
	snap.Packages = append(snap.Packages, graph.PackageNode{ID: "p2", ImportPath: "demo/orders", RepoPath: "api/internal/orders", Language: graph.LanguageGo})
	svc := &mocks.FakeService{Snapshots: []graph.GraphSnapshot{snap}}
	domainsSvc := &domainmocks.FakeService{Map: intdomains.DerivedDomainMap{
		Scenario: "demo",
		Domains: []intdomains.DerivedDomain{
			{Name: "billing", Paths: []string{"api/internal/billing/**"}, Archetypes: intdomains.DeclaredArchetypes("validation")},
			{Name: "orders", Paths: []string{"api/internal/orders/**"}, Archetypes: intdomains.DeclaredArchetypes("validation")},
		},
	}}
	h := graphh.NewHandler(svc, domainsSvc)

	resp, err := h.GetZoneMap(context.Background(), connect.NewRequest(&graphv1.GetZoneMapRequest{Scenario: "demo"}))
	require.NoError(t, err)
	require.Equal(t, "snap:demo:h1", resp.Msg.GetZoneMap().GetSnapshotId())
	require.Len(t, resp.Msg.GetZoneMap().GetPackages(), 2)
	require.Equal(t, "domain", resp.Msg.GetZoneMap().GetPackages()[0].GetZone())
	require.Equal(t, "billing", resp.Msg.GetZoneMap().GetPackages()[0].GetDomain())
	require.Len(t, resp.Msg.GetZoneMap().GetViolations(), 1)
	require.Equal(t, "domain-imports-sibling-domain", resp.Msg.GetZoneMap().GetViolations()[0].GetSubtype())
	require.Equal(t, int64(1), svc.ListCalls.Load())
	require.Equal(t, int64(1), domainsSvc.GetCalls.Load())
}

func TestHandler_GetSlice(t *testing.T) {
	snap := sampleSnap()
	snap.Packages = append(snap.Packages,
		graph.PackageNode{ID: "p-proto", ImportPath: "github.com/vrooli/vrooli/packages/proto/gen/go/demo/v1/billing"},
		graph.PackageNode{ID: "p-cli", ImportPath: "demo/cli/domains/billing", RepoPath: "cli/domains/billing"},
	)
	snap.Imports = append(snap.Imports, graph.ImportEdge{From: "p1", ToPackageID: "p-proto"})
	svc := &mocks.FakeService{Snapshots: []graph.GraphSnapshot{snap}}
	domainsSvc := &domainmocks.FakeService{Map: intdomains.DerivedDomainMap{
		Scenario: "demo",
		Domains: []intdomains.DerivedDomain{{
			Name:       "billing",
			Paths:      []string{"api/internal/billing/**", "cli/domains/billing/**"},
			Archetypes: intdomains.DeclaredArchetypes("service"),
		}},
	}}
	h := graphh.NewHandler(svc, domainsSvc)

	resp, err := h.GetSlice(context.Background(), connect.NewRequest(&graphv1.GetSliceRequest{Scenario: "demo", Domain: "billing"}))
	require.NoError(t, err)
	require.Equal(t, "snap:demo:h1", resp.Msg.GetSlice().GetSnapshotId())
	require.Equal(t, "billing", resp.Msg.GetSlice().GetDomain())
	require.Contains(t, resp.Msg.GetSlice().GetSurfaces(), "cli")
	require.Len(t, resp.Msg.GetSlice().GetRungs(), 5)
	require.True(t, resp.Msg.GetSlice().GetRungs()[0].GetPresent(), "proto rung should be present from generated proto import")
	require.Equal(t, int64(1), svc.ListCalls.Load())
	require.Equal(t, int64(1), domainsSvc.GetCalls.Load())
}

func TestHandler_GetSlice_RejectsUnknownDomain(t *testing.T) {
	svc := &mocks.FakeService{Snapshots: []graph.GraphSnapshot{sampleSnap()}}
	domainsSvc := &domainmocks.FakeService{Map: intdomains.DerivedDomainMap{Scenario: "demo"}}
	h := graphh.NewHandler(svc, domainsSvc)

	_, err := h.GetSlice(context.Background(), connect.NewRequest(&graphv1.GetSliceRequest{Scenario: "demo", Domain: "missing"}))
	require.Error(t, err)
	var ce *connect.Error
	require.ErrorAs(t, err, &ce)
	require.Equal(t, connect.CodeNotFound, ce.Code())
}

func TestHandler_InterfaceSatisfied(t *testing.T) {
	var _ graph_v1connect.GraphServiceHandler = (*graphh.Handler)(nil)
}
