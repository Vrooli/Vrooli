package ontology

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	ontologyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/ontology"
	ontologyconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/ontology/ontology_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	cliapptest "github.com/vrooli/cli-core/cliapptest"

	clitest "tech-tree-designer/cli/internal/testutil"
)

// fakeOntology embeds the generated Unimplemented handler so it satisfies
// the full service surface while individual tests override only the
// methods they exercise. Each override records the inbound request so the
// test can assert the CLI translated flags/positionals into the wire shape.
type fakeOntology struct {
	ontologyconnect.UnimplementedOntologyServiceHandler

	listCapabilitiesReq     *ontologyv1.ListCapabilitiesRequest
	getCapabilityReq        *ontologyv1.GetCapabilityRequest
	upsertCapabilityReq     *ontologyv1.UpsertCapabilityRequest
	deleteCapabilityReq     *ontologyv1.DeleteCapabilityRequest
	upsertEdgeReq           *ontologyv1.UpsertCapabilityEdgeRequest
	deleteEdgeReq           *ontologyv1.DeleteCapabilityEdgeRequest
	importTopologyReq       *ontologyv1.ImportTopologyRequest
	linkFulfillmentReq      *ontologyv1.LinkFulfillmentRequest
	unlinkFulfillmentReq    *ontologyv1.UnlinkFulfillmentRequest
	listFulfillmentsReq     *ontologyv1.ListFulfillmentsRequest
	getCoverageReq          *ontologyv1.GetCoverageRequest
	listFocusReq            *ontologyv1.ListFocusRequest
	capabilityScenariosReq  *ontologyv1.GetCapabilityScenariosRequest
	scenarioCapabilitiesReq *ontologyv1.GetScenarioCapabilitiesRequest
	overlayReq              *ontologyv1.DescribeOverlayGraphRequest
}

func (f *fakeOntology) ListCapabilities(_ context.Context, req *connect.Request[ontologyv1.ListCapabilitiesRequest]) (*connect.Response[ontologyv1.ListCapabilitiesResponse], error) {
	f.listCapabilitiesReq = req.Msg
	return connect.NewResponse(&ontologyv1.ListCapabilitiesResponse{
		Capabilities: []*ontologyv1.Capability{
			{Slug: "compute", Kind: ontologyv1.CapabilityKind_CAPABILITY_KIND_SECTOR},
			{Slug: "scheduler", Kind: ontologyv1.CapabilityKind_CAPABILITY_KIND_CAPABILITY, ParentId: "compute"},
		},
	}), nil
}

func (f *fakeOntology) GetCapability(_ context.Context, req *connect.Request[ontologyv1.GetCapabilityRequest]) (*connect.Response[ontologyv1.Capability], error) {
	f.getCapabilityReq = req.Msg
	return connect.NewResponse(&ontologyv1.Capability{Slug: req.Msg.GetSlug(), Kind: ontologyv1.CapabilityKind_CAPABILITY_KIND_CAPABILITY}), nil
}

func (f *fakeOntology) UpsertCapability(_ context.Context, req *connect.Request[ontologyv1.UpsertCapabilityRequest]) (*connect.Response[ontologyv1.Capability], error) {
	f.upsertCapabilityReq = req.Msg
	return connect.NewResponse(req.Msg.GetCapability()), nil
}

func (f *fakeOntology) DeleteCapability(_ context.Context, req *connect.Request[ontologyv1.DeleteCapabilityRequest]) (*connect.Response[ontologyv1.DeleteCapabilityResponse], error) {
	f.deleteCapabilityReq = req.Msg
	return connect.NewResponse(&ontologyv1.DeleteCapabilityResponse{Deleted: true}), nil
}

func (f *fakeOntology) UpsertCapabilityEdge(_ context.Context, req *connect.Request[ontologyv1.UpsertCapabilityEdgeRequest]) (*connect.Response[ontologyv1.CapabilityEdge], error) {
	f.upsertEdgeReq = req.Msg
	return connect.NewResponse(req.Msg.GetEdge()), nil
}

func (f *fakeOntology) DeleteCapabilityEdge(_ context.Context, req *connect.Request[ontologyv1.DeleteCapabilityEdgeRequest]) (*connect.Response[ontologyv1.DeleteCapabilityEdgeResponse], error) {
	f.deleteEdgeReq = req.Msg
	return connect.NewResponse(&ontologyv1.DeleteCapabilityEdgeResponse{Deleted: true}), nil
}

func (f *fakeOntology) ImportTopology(_ context.Context, req *connect.Request[ontologyv1.ImportTopologyRequest]) (*connect.Response[ontologyv1.ImportTopologyResponse], error) {
	f.importTopologyReq = req.Msg
	return connect.NewResponse(&ontologyv1.ImportTopologyResponse{
		SectorsTotal: 2, CapabilitiesTotal: 5, EdgesTotal: 4,
		SectorsImported: 2, CapabilitiesImported: 5, EdgesImported: 4,
	}), nil
}

func (f *fakeOntology) LinkFulfillment(_ context.Context, req *connect.Request[ontologyv1.LinkFulfillmentRequest]) (*connect.Response[ontologyv1.Fulfillment], error) {
	f.linkFulfillmentReq = req.Msg
	return connect.NewResponse(req.Msg.GetFulfillment()), nil
}

func (f *fakeOntology) UnlinkFulfillment(_ context.Context, req *connect.Request[ontologyv1.UnlinkFulfillmentRequest]) (*connect.Response[ontologyv1.UnlinkFulfillmentResponse], error) {
	f.unlinkFulfillmentReq = req.Msg
	return connect.NewResponse(&ontologyv1.UnlinkFulfillmentResponse{Deleted: true}), nil
}

func (f *fakeOntology) ListFulfillments(_ context.Context, req *connect.Request[ontologyv1.ListFulfillmentsRequest]) (*connect.Response[ontologyv1.ListFulfillmentsResponse], error) {
	f.listFulfillmentsReq = req.Msg
	return connect.NewResponse(&ontologyv1.ListFulfillmentsResponse{
		Fulfillments: []*ontologyv1.Fulfillment{{CapabilityId: "scheduler", ScenarioSlug: "swarm-manager"}},
	}), nil
}

func (f *fakeOntology) GetCoverage(_ context.Context, req *connect.Request[ontologyv1.GetCoverageRequest]) (*connect.Response[ontologyv1.CoverageSummary], error) {
	f.getCoverageReq = req.Msg
	return connect.NewResponse(&ontologyv1.CoverageSummary{
		TotalCapabilities:          10,
		TotalScenarios:             4,
		BuiltCapabilities:          6,
		InflightCapabilities:       2,
		GapCapabilities:            2,
		UnmappedScenarios:          1,
		OntologyCompleteness:       0.8,
		ImplementationSituatedness: 0.6,
		Sectors: []*ontologyv1.SectorCoverage{
			{SectorSlug: "compute", BuiltCapabilities: 3, InflightCapabilities: 1, GapCapabilities: 1, TotalCapabilities: 5},
		},
	}), nil
}

func (f *fakeOntology) ListFocus(_ context.Context, req *connect.Request[ontologyv1.ListFocusRequest]) (*connect.Response[ontologyv1.ListFocusResponse], error) {
	f.listFocusReq = req.Msg
	return connect.NewResponse(&ontologyv1.ListFocusResponse{
		Items: []*ontologyv1.FocusItem{
			{CapabilitySlug: "scheduler", Score: 0.9, DownstreamDependents: 3},
		},
	}), nil
}

func (f *fakeOntology) GetCapabilityScenarios(_ context.Context, req *connect.Request[ontologyv1.GetCapabilityScenariosRequest]) (*connect.Response[ontologyv1.CapabilityScenarios], error) {
	f.capabilityScenariosReq = req.Msg
	return connect.NewResponse(&ontologyv1.CapabilityScenarios{
		CapabilitySlug:   req.Msg.GetCapabilitySlug(),
		BuiltScenarios:   []string{"swarm-manager"},
		PlannedScenarios: []string{"tech-tree-designer"},
		Fulfillments:     []*ontologyv1.Fulfillment{{CapabilityId: req.Msg.GetCapabilitySlug(), ScenarioSlug: "swarm-manager"}},
	}), nil
}

func (f *fakeOntology) GetScenarioCapabilities(_ context.Context, req *connect.Request[ontologyv1.GetScenarioCapabilitiesRequest]) (*connect.Response[ontologyv1.ScenarioCapabilities], error) {
	f.scenarioCapabilitiesReq = req.Msg
	return connect.NewResponse(&ontologyv1.ScenarioCapabilities{
		ScenarioSlug: req.Msg.GetScenarioSlug(),
		Capabilities: []*ontologyv1.Capability{{Slug: "scheduler", Kind: ontologyv1.CapabilityKind_CAPABILITY_KIND_CAPABILITY}},
	}), nil
}

func (f *fakeOntology) DescribeOverlayGraph(_ context.Context, req *connect.Request[ontologyv1.DescribeOverlayGraphRequest]) (*connect.Response[ontologyv1.DescribeOverlayGraphResponse], error) {
	f.overlayReq = req.Msg
	return connect.NewResponse(&ontologyv1.DescribeOverlayGraphResponse{
		Graph: &ontologyv1.OverlayGraph{
			Nodes: []*ontologyv1.OverlayNode{{Scenario: "ui"}, {Scenario: "api"}},
			Edges: []*ontologyv1.OverlayEdge{{FromScenario: "ui", ToScenario: "api"}},
		},
	}), nil
}

func ontologyAPI(t *testing.T, svc *fakeOntology) http.Handler {
	t.Helper()
	path, handler := ontologyconnect.NewOntologyServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	return mux
}

func newOntologyHandlers(t *testing.T, svc *fakeOntology) *handlers {
	t.Helper()
	core := clitest.NewTestApp(t, ontologyAPI(t, svc))
	return newHandlers(core)
}

func ontologyCtx(t *testing.T, svc *fakeOntology, schema cliapp.ArgSchema, opts cliapptest.TestRunContextOptions) (*handlers, cliapp.RunContext, func() string) {
	t.Helper()
	core := clitest.NewTestApp(t, ontologyAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, schema, opts)
	return h, ctx, out.String
}

func TestListCapabilitiesTranslatesFlags(t *testing.T) {
	svc := &fakeOntology{}
	h, ctx, out := ontologyCtx(t, svc, cliapp.ArgSchema{
		Flags: []cliapp.Flag{{Name: "kind"}, {Name: "parent"}, {Name: "descendants"}},
	}, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"kind": "sector", "parent": "root", "descendants": "true"},
	})
	require.NoError(t, h.listCapabilities(ctx))
	require.Equal(t, ontologyv1.CapabilityKind_CAPABILITY_KIND_SECTOR, svc.listCapabilitiesReq.GetKind())
	require.Equal(t, "root", svc.listCapabilitiesReq.GetParentId())
	require.True(t, svc.listCapabilitiesReq.GetIncludeDescendants())
	require.Contains(t, out(), "2 capability node(s).")
	require.Contains(t, out(), "scheduler")
}

func TestListCapabilitiesRejectsUnknownKind(t *testing.T) {
	h, ctx, _ := ontologyCtx(t, &fakeOntology{}, cliapp.ArgSchema{
		Flags: []cliapp.Flag{{Name: "kind"}},
	}, cliapptest.TestRunContextOptions{Flags: map[string]string{"kind": "bogus"}})
	require.Error(t, h.listCapabilities(ctx))
}

func TestGetCapabilityPassesSlug(t *testing.T) {
	svc := &fakeOntology{}
	h, ctx, out := ontologyCtx(t, svc, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "slug", Required: true}},
	}, cliapptest.TestRunContextOptions{Positionals: map[string]string{"slug": "scheduler"}})
	require.NoError(t, h.getCapability(ctx))
	require.Equal(t, "scheduler", svc.getCapabilityReq.GetSlug())
	require.Contains(t, out(), "scheduler")
}

func TestUpsertCapabilityBuildsCapability(t *testing.T) {
	svc := &fakeOntology{}
	h, ctx, out := ontologyCtx(t, svc, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "slug", Required: true}},
		Flags:       []cliapp.Flag{{Name: "name"}, {Name: "description"}, {Name: "kind"}, {Name: "parent"}, {Name: "importance"}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"slug": "scheduler"},
		Flags: map[string]string{
			"name": "Scheduler", "description": "schedules", "kind": "capability",
			"parent": "compute", "importance": "0.75",
		},
	})
	require.NoError(t, h.upsertCapability(ctx))
	require.Equal(t, "scheduler", svc.upsertCapabilityReq.GetCapability().GetSlug())
	require.Equal(t, "compute", svc.upsertCapabilityReq.GetCapability().GetParentId())
	require.InDelta(t, 0.75, svc.upsertCapabilityReq.GetCapability().GetImportance(), 0.0001)
	require.Contains(t, out(), "Capability scheduler saved.")
}

func TestUpsertCapabilityRejectsUnknownKind(t *testing.T) {
	h, ctx, _ := ontologyCtx(t, &fakeOntology{}, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "slug", Required: true}},
		Flags:       []cliapp.Flag{{Name: "kind"}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"slug": "x"},
		Flags:       map[string]string{"kind": "nope"},
	})
	require.Error(t, h.upsertCapability(ctx))
}

func TestRemoveCapabilityReportsDeleted(t *testing.T) {
	svc := &fakeOntology{}
	h, ctx, out := ontologyCtx(t, svc, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "slug", Required: true}},
	}, cliapptest.TestRunContextOptions{Positionals: map[string]string{"slug": "scheduler"}})
	require.NoError(t, h.removeCapability(ctx))
	require.Equal(t, "scheduler", svc.deleteCapabilityReq.GetSlug())
	require.Contains(t, out(), "Deleted: true.")
}

func TestAddEdgeParsesType(t *testing.T) {
	svc := &fakeOntology{}
	h, ctx, out := ontologyCtx(t, svc, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "from", Required: true}, {Name: "to", Required: true}},
		Flags:       []cliapp.Flag{{Name: "type"}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"from": "compute", "to": "scheduler"},
		Flags:       map[string]string{"type": "requires"},
	})
	require.NoError(t, h.addEdge(ctx))
	require.Equal(t, "compute", svc.upsertEdgeReq.GetEdge().GetFromId())
	require.Equal(t, ontologyv1.CapabilityEdgeType_CAPABILITY_EDGE_TYPE_REQUIRES, svc.upsertEdgeReq.GetEdge().GetType())
	require.Contains(t, out(), "Capability edge saved.")
}

func TestAddEdgeRejectsUnknownType(t *testing.T) {
	h, ctx, _ := ontologyCtx(t, &fakeOntology{}, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "from", Required: true}, {Name: "to", Required: true}},
		Flags:       []cliapp.Flag{{Name: "type"}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"from": "a", "to": "b"},
		Flags:       map[string]string{"type": "weird"},
	})
	require.Error(t, h.addEdge(ctx))
}

func TestRemoveEdgeReportsDeleted(t *testing.T) {
	svc := &fakeOntology{}
	h, ctx, out := ontologyCtx(t, svc, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "from", Required: true}, {Name: "to", Required: true}},
		Flags:       []cliapp.Flag{{Name: "type"}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"from": "compute", "to": "scheduler"},
	})
	require.NoError(t, h.removeEdge(ctx))
	require.Equal(t, "compute", svc.deleteEdgeReq.GetEdge().GetFromId())
	require.Contains(t, out(), "Deleted: true.")
}

func TestRemoveEdgeRejectsUnknownType(t *testing.T) {
	h, ctx, _ := ontologyCtx(t, &fakeOntology{}, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "from", Required: true}, {Name: "to", Required: true}},
		Flags:       []cliapp.Flag{{Name: "type"}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"from": "a", "to": "b"},
		Flags:       map[string]string{"type": "weird"},
	})
	require.Error(t, h.removeEdge(ctx))
}

func TestImportTopologyReadsFileAndReports(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "topology.json")
	require.NoError(t, os.WriteFile(file, []byte(`{"sectors":[]}`), 0o600))
	svc := &fakeOntology{}
	h, ctx, out := ontologyCtx(t, svc, cliapp.ArgSchema{
		Flags: []cliapp.Flag{{Name: "from-file"}},
	}, cliapptest.TestRunContextOptions{Flags: map[string]string{"from-file": file}})
	require.NoError(t, h.importTopology(ctx))
	require.Equal(t, `{"sectors":[]}`, svc.importTopologyReq.GetJson())
	require.Contains(t, out(), "Imported sectors=2 capabilities=5 edges=4.")
}

func TestImportTopologyRequiresFile(t *testing.T) {
	h, ctx, _ := ontologyCtx(t, &fakeOntology{}, cliapp.ArgSchema{
		Flags: []cliapp.Flag{{Name: "from-file"}},
	}, cliapptest.TestRunContextOptions{})
	require.Error(t, h.importTopology(ctx))
}

func TestFulfillLinksFulfillment(t *testing.T) {
	svc := &fakeOntology{}
	h, ctx, out := ontologyCtx(t, svc, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "capability", Required: true}, {Name: "scenario", Required: true}},
		Flags:       []cliapp.Flag{{Name: "note"}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"capability": "scheduler", "scenario": "swarm-manager"},
		Flags:       map[string]string{"note": "core dependency"},
	})
	require.NoError(t, h.fulfill(ctx))
	require.Equal(t, "scheduler", svc.linkFulfillmentReq.GetFulfillment().GetCapabilityId())
	require.Equal(t, "core dependency", svc.linkFulfillmentReq.GetFulfillment().GetNote())
	require.Contains(t, out(), "Fulfillment linked.")
}

func TestUnfulfillReportsDeleted(t *testing.T) {
	svc := &fakeOntology{}
	h, ctx, out := ontologyCtx(t, svc, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "capability", Required: true}, {Name: "scenario", Required: true}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"capability": "scheduler", "scenario": "swarm-manager"},
	})
	require.NoError(t, h.unfulfill(ctx))
	require.Equal(t, "scheduler", svc.unlinkFulfillmentReq.GetCapabilityId())
	require.Contains(t, out(), "Deleted: true.")
}

func TestListFulfillmentsRenders(t *testing.T) {
	svc := &fakeOntology{}
	h, ctx, out := ontologyCtx(t, svc, cliapp.ArgSchema{
		Flags: []cliapp.Flag{{Name: "capability"}, {Name: "scenario"}},
	}, cliapptest.TestRunContextOptions{Flags: map[string]string{"capability": "scheduler"}})
	require.NoError(t, h.listFulfillments(ctx))
	require.Equal(t, "scheduler", svc.listFulfillmentsReq.GetCapabilityId())
	require.Contains(t, out(), "1 fulfillment link(s).")
	require.Contains(t, out(), "scheduler <- swarm-manager")
}

func TestCoverageRendersSummary(t *testing.T) {
	svc := &fakeOntology{}
	h, ctx, out := ontologyCtx(t, svc, cliapp.ArgSchema{
		Flags: []cliapp.Flag{{Name: "subtree"}},
	}, cliapptest.TestRunContextOptions{Flags: map[string]string{"subtree": "true"}})
	require.NoError(t, h.coverage(ctx))
	require.True(t, svc.getCoverageReq.GetIncludeSubtreeRollup())
	require.Contains(t, out(), "built=6 inflight=2 gap=2 unmapped=1")
	require.Contains(t, out(), "compute built=3")
}

func TestCoverageJSONIsProtoWireShape(t *testing.T) {
	svc := &fakeOntology{}
	h, ctx, out := ontologyCtx(t, svc, cliapp.ArgSchema{
		Flags: []cliapp.Flag{{Name: "subtree"}},
	}, cliapptest.TestRunContextOptions{JSON: true})
	require.NoError(t, h.coverage(ctx))
	var got ontologyv1.CoverageSummary
	require.NoError(t, protojson.Unmarshal([]byte(out()), &got))
	require.EqualValues(t, 10, got.GetTotalCapabilities())
}

func TestFocusParsesLimit(t *testing.T) {
	svc := &fakeOntology{}
	h, ctx, out := ontologyCtx(t, svc, cliapp.ArgSchema{
		Flags: []cliapp.Flag{{Name: "limit"}},
	}, cliapptest.TestRunContextOptions{Flags: map[string]string{"limit": "5"}})
	require.NoError(t, h.focus(ctx))
	require.EqualValues(t, 5, svc.listFocusReq.GetLimit())
	require.Contains(t, out(), "1 focus item(s).")
	require.Contains(t, out(), "scheduler")
}

func TestFocusRejectsBadLimit(t *testing.T) {
	h, ctx, _ := ontologyCtx(t, &fakeOntology{}, cliapp.ArgSchema{
		Flags: []cliapp.Flag{{Name: "limit"}},
	}, cliapptest.TestRunContextOptions{Flags: map[string]string{"limit": "not-a-number"}})
	require.Error(t, h.focus(ctx))
}

func TestCapabilityScenariosRenders(t *testing.T) {
	svc := &fakeOntology{}
	h, ctx, out := ontologyCtx(t, svc, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "slug", Required: true}},
		Flags:       []cliapp.Flag{{Name: "descendants"}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"slug": "scheduler"},
		Flags:       map[string]string{"descendants": "yes"},
	})
	require.NoError(t, h.capabilityScenarios(ctx))
	require.Equal(t, "scheduler", svc.capabilityScenariosReq.GetCapabilitySlug())
	require.True(t, svc.capabilityScenariosReq.GetIncludeDescendants())
	require.Contains(t, out(), "built: swarm-manager")
	require.Contains(t, out(), "planned: tech-tree-designer")
}

func TestScenarioCapabilitiesRenders(t *testing.T) {
	svc := &fakeOntology{}
	h, ctx, out := ontologyCtx(t, svc, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "slug", Required: true}},
	}, cliapptest.TestRunContextOptions{Positionals: map[string]string{"slug": "swarm-manager"}})
	require.NoError(t, h.scenarioCapabilities(ctx))
	require.Equal(t, "swarm-manager", svc.scenarioCapabilitiesReq.GetScenarioSlug())
	require.Contains(t, out(), "swarm-manager fulfills 1 capability node(s).")
	require.Contains(t, out(), "scheduler")
}

func TestOverlayTranslatesFlagsAndRenders(t *testing.T) {
	svc := &fakeOntology{}
	h, ctx, out := ontologyCtx(t, svc, cliapp.ArgSchema{
		Flags: []cliapp.Flag{{Name: "implementation"}, {Name: "ontology"}, {Name: "fulfillment"}},
	}, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"implementation": "true", "ontology": "true", "fulfillment": "false"},
	})
	require.NoError(t, h.overlay(ctx))
	require.True(t, svc.overlayReq.GetIncludeImplementation())
	require.True(t, svc.overlayReq.GetIncludeOntology())
	require.False(t, svc.overlayReq.GetIncludeFulfillment())
	require.Contains(t, out(), "Overlay graph: 2 node(s), 1 edge(s).")
	require.Contains(t, out(), "ui -> api")
}

func TestCapabilityKindParsing(t *testing.T) {
	cases := map[string]ontologyv1.CapabilityKind{
		"":           ontologyv1.CapabilityKind_CAPABILITY_KIND_UNSPECIFIED,
		"sector":     ontologyv1.CapabilityKind_CAPABILITY_KIND_SECTOR,
		"capability": ontologyv1.CapabilityKind_CAPABILITY_KIND_CAPABILITY,
		"component":  ontologyv1.CapabilityKind_CAPABILITY_KIND_COMPONENT,
		"capstone":   ontologyv1.CapabilityKind_CAPABILITY_KIND_CAPSTONE,
		"simulation": ontologyv1.CapabilityKind_CAPABILITY_KIND_SIMULATION,
		"  SECTOR  ": ontologyv1.CapabilityKind_CAPABILITY_KIND_SECTOR,
	}
	for in, want := range cases {
		got, err := capabilityKind(in)
		require.NoError(t, err, in)
		require.Equal(t, want, got, in)
	}
	_, err := capabilityKind("nonsense")
	require.Error(t, err)
}

func TestEdgeTypeParsing(t *testing.T) {
	cases := map[string]ontologyv1.CapabilityEdgeType{
		"":            ontologyv1.CapabilityEdgeType_CAPABILITY_EDGE_TYPE_PROGRESSION,
		"progression": ontologyv1.CapabilityEdgeType_CAPABILITY_EDGE_TYPE_PROGRESSION,
		"decomposes":  ontologyv1.CapabilityEdgeType_CAPABILITY_EDGE_TYPE_DECOMPOSES,
		"requires":    ontologyv1.CapabilityEdgeType_CAPABILITY_EDGE_TYPE_REQUIRES,
	}
	for in, want := range cases {
		got, err := edgeType(in)
		require.NoError(t, err, in)
		require.Equal(t, want, got, in)
	}
	_, err := edgeType("nonsense")
	require.Error(t, err)
}

func TestReadFile(t *testing.T) {
	_, err := readFile("")
	require.Error(t, err)
	_, err = readFile("/no/such/path/topology.json")
	require.Error(t, err)
	dir := t.TempDir()
	file := filepath.Join(dir, "t.json")
	require.NoError(t, os.WriteFile(file, []byte("payload"), 0o600))
	got, err := readFile(file)
	require.NoError(t, err)
	require.Equal(t, "payload", got)
}

func TestFlagHelpers(t *testing.T) {
	require.True(t, flagBool("true"))
	require.True(t, flagBool("YES"))
	require.True(t, flagBool("1"))
	require.False(t, flagBool(""))
	require.False(t, flagBool("no"))

	v, err := flagInt32("")
	require.NoError(t, err)
	require.EqualValues(t, 0, v)
	v, err = flagInt32("42")
	require.NoError(t, err)
	require.EqualValues(t, 42, v)
	_, err = flagInt32("abc")
	require.Error(t, err)

	require.InDelta(t, 1.5, flagFloat("1.5"), 0.0001)
	require.InDelta(t, 0, flagFloat(""), 0.0001)
}

func TestCapabilityLineDefaultsParentToRoot(t *testing.T) {
	require.Contains(t, capabilityLine(&ontologyv1.Capability{Slug: "x"}), "parent=root")
	require.Contains(t, capabilityLine(&ontologyv1.Capability{Slug: "x", ParentId: "y"}), "parent=y")
}
