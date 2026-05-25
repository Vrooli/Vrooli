package graph

import (
	"context"
	"io"
	"log"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	intgraph "typescript-code-graph/internal/graph"
	intrewrite "typescript-code-graph/internal/rewrite"
	"typescript-code-graph/internal/sidecar"
	sidecarmocks "typescript-code-graph/internal/sidecar/mocks"

	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/typescript-code-graph/v1/graph"
	"github.com/vrooli/vrooli/packages/proto/gen/go/typescript-code-graph/v1/graph/graph_v1connect"
	rewritepb "github.com/vrooli/vrooli/packages/proto/gen/go/typescript-code-graph/v1/rewrite"
)

// newTestClient mounts the Connect handler on an httptest.Server and
// returns a paired Connect client. The rewrite.Service shares the
// sidecar mock and path mutex with the graph.Service so tests can
// exercise either RPC family.
func newTestClient(t *testing.T, svc *intgraph.Service) graph_v1connect.TypeScriptCodeGraphServiceClient {
	t.Helper()
	return newTestClientWithRewrite(t, svc, nil)
}

func newTestClientWithRewrite(t *testing.T, gsvc *intgraph.Service, rsvc *intrewrite.Service) graph_v1connect.TypeScriptCodeGraphServiceClient {
	t.Helper()
	if rsvc == nil {
		// Default rewrite service backed by a fresh in-memory store
		// and a fresh fake sidecar; lets RewriteApply tests run with
		// an isolated dependency tree.
		fake := &sidecarmocks.FakeSidecarClient{StatusValue: sidecar.StatusReady}
		rsvc = intrewrite.NewService(intrewrite.NewMemoryPlanStore(), fake, intgraph.NewPathMutex())
	}
	_, h := graph_v1connect.NewTypeScriptCodeGraphServiceHandler(NewConnectHandler(Deps{
		GraphService:   gsvc,
		RewriteService: rsvc,
		Logger:         log.New(io.Discard, "", 0),
	}))
	server := httptest.NewServer(h)
	t.Cleanup(server.Close)
	return graph_v1connect.NewTypeScriptCodeGraphServiceClient(server.Client(), server.URL)
}

func TestHandlerExtractHappyPath(t *testing.T) {
	t.Parallel()
	fake := &sidecarmocks.FakeSidecarClient{
		StatusValue: sidecar.StatusReady,
		ExtractFn: func(ctx context.Context, p string) (sidecar.ExtractResult, error) {
			return sidecar.ExtractResult{
				Graph: sidecar.RawGraph{
					Nodes: []sidecar.RawNode{
						{ID: "file:src/a.ts", Kind: 1, Name: "a.ts", Path: "src/a.ts"},
						{
							ID: "ts_component:m:Btn", Kind: 201, Name: "Btn", Path: "src/Btn.tsx",
							LeadingComments: []string{"/** @vrooliWidget */"},
						},
					},
				},
				RequestID: "req-handler-1",
			}, nil
		},
	}
	svc := intgraph.NewService(fake, intgraph.NewPathMutex())
	client := newTestClient(t, svc)

	resp, err := client.Extract(context.Background(), connect.NewRequest(&graphv1.ExtractRequest{
		ScenarioPath: "/abs/proj",
	}))
	require.NoError(t, err)
	require.NotNil(t, resp.Msg.GetGraph())
	require.Len(t, resp.Msg.GetGraph().GetNodes(), 2)
	require.NotEmpty(t, resp.Msg.GetGraphHash())
	require.GreaterOrEqual(t, resp.Msg.GetExtractionMs(), int64(0))
	require.Equal(t, "req-handler-1", resp.Msg.GetSidecarRequestId(),
		"sidecar_request_id must round-trip onto the proto response")

	// Find the component node; its leading_comments must survive verbatim
	// and attributes["kind"] should report the TS-specific enum name.
	var comp *graphv1.ExtractResponse
	_ = comp
	for _, n := range resp.Msg.GetGraph().GetNodes() {
		if n.GetId() == "ts_component:m:Btn" {
			require.Equal(t, []string{"/** @vrooliWidget */"}, n.GetLeadingComments())
			require.Equal(t, "TS_NODE_KIND_COMPONENT", n.GetAttributes()["kind"])
			return
		}
	}
	t.Fatalf("component node missing from response")
}

func TestHandlerExtractInvalidArgument(t *testing.T) {
	t.Parallel()
	fake := &sidecarmocks.FakeSidecarClient{StatusValue: sidecar.StatusReady}
	svc := intgraph.NewService(fake, intgraph.NewPathMutex())
	client := newTestClient(t, svc)
	_, err := client.Extract(context.Background(), connect.NewRequest(&graphv1.ExtractRequest{ScenarioPath: ""}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestHandlerExtractUnavailableSidecar(t *testing.T) {
	t.Parallel()
	fake := &sidecarmocks.FakeSidecarClient{StatusValue: sidecar.StatusUnhealthy}
	svc := intgraph.NewService(fake, intgraph.NewPathMutex())
	client := newTestClient(t, svc)
	_, err := client.Extract(context.Background(), connect.NewRequest(&graphv1.ExtractRequest{ScenarioPath: "/abs"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
}

func TestHandlerExtractWorkspaceUnsupported(t *testing.T) {
	t.Parallel()
	fake := &sidecarmocks.FakeSidecarClient{
		StatusValue: sidecar.StatusReady,
		ExtractFn: func(ctx context.Context, p string) (sidecar.ExtractResult, error) {
			return sidecar.ExtractResult{}, &sidecar.ExtractError{Kind: "workspace_unsupported"}
		},
	}
	svc := intgraph.NewService(fake, intgraph.NewPathMutex())
	client := newTestClient(t, svc)
	_, err := client.Extract(context.Background(), connect.NewRequest(&graphv1.ExtractRequest{ScenarioPath: "/abs"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeUnimplemented, connect.CodeOf(err))
}

func TestHandlerExtractNotFound(t *testing.T) {
	t.Parallel()
	fake := &sidecarmocks.FakeSidecarClient{
		StatusValue: sidecar.StatusReady,
		ExtractFn: func(ctx context.Context, p string) (sidecar.ExtractResult, error) {
			return sidecar.ExtractResult{}, &sidecar.ExtractError{Kind: "path_unreadable"}
		},
	}
	svc := intgraph.NewService(fake, intgraph.NewPathMutex())
	client := newTestClient(t, svc)
	_, err := client.Extract(context.Background(), connect.NewRequest(&graphv1.ExtractRequest{ScenarioPath: "/abs"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

// TestHandlerRewritePlan_HappyPath confirms the Phase-5 wiring: the
// graph handler delegates to handlers/rewrite, which in turn drives
// rewrite.Service. A round-trip yields a plan_id and the normalized
// operations.
func TestHandlerRewritePlan_HappyPath(t *testing.T) {
	t.Parallel()
	fake := &sidecarmocks.FakeSidecarClient{StatusValue: sidecar.StatusReady}
	pathMu := intgraph.NewPathMutex()
	gsvc := intgraph.NewService(fake, pathMu)
	rsvc := intrewrite.NewService(intrewrite.NewMemoryPlanStore(), fake, pathMu)
	client := newTestClientWithRewrite(t, gsvc, rsvc)

	resp, err := client.RewritePlan(context.Background(), connect.NewRequest(&graphv1.RewritePlanRequest{
		ScenarioPath: "/abs/proj",
		Operations: []*rewritepb.Operation{
			{Op: &rewritepb.Operation_FileMove{FileMove: &rewritepb.FileMove{FromPath: "a.ts", ToPath: "b.ts"}}},
		},
	}))
	require.NoError(t, err)
	require.NotEmpty(t, resp.Msg.GetPlanId())
}

// TestHandlerRewritePlan_InvalidArgument confirms validation errors
// surface as InvalidArgument (empty scenario_path).
func TestHandlerRewritePlan_InvalidArgument(t *testing.T) {
	t.Parallel()
	client := newTestClient(t, intgraph.NewService(&sidecarmocks.FakeSidecarClient{StatusValue: sidecar.StatusReady}, intgraph.NewPathMutex()))
	_, err := client.RewritePlan(context.Background(), connect.NewRequest(&graphv1.RewritePlanRequest{ScenarioPath: ""}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

// TestHandlerRewriteApply_RejectsApplyFalse mirrors the proto comment:
// apply=false is rejected with InvalidArgument because dry-run is
// signaled via the X-Dry-Run header.
func TestHandlerRewriteApply_RejectsApplyFalse(t *testing.T) {
	t.Parallel()
	client := newTestClient(t, intgraph.NewService(&sidecarmocks.FakeSidecarClient{StatusValue: sidecar.StatusReady}, intgraph.NewPathMutex()))
	_, err := client.RewriteApply(context.Background(), connect.NewRequest(&graphv1.RewriteApplyRequest{ScenarioPath: "/abs", PlanId: "x", Apply: false}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

// TestHandlerRewriteApply_DryRunHeader confirms the X-Dry-Run: true
// header flows through to the service and short-circuits before any
// sidecar call.
func TestHandlerRewriteApply_DryRunHeader(t *testing.T) {
	t.Parallel()
	fake := &sidecarmocks.FakeSidecarClient{StatusValue: sidecar.StatusUnhealthy} // unhealthy on purpose
	pathMu := intgraph.NewPathMutex()
	gsvc := intgraph.NewService(fake, pathMu)
	rsvc := intrewrite.NewService(intrewrite.NewMemoryPlanStore(), fake, pathMu)
	client := newTestClientWithRewrite(t, gsvc, rsvc)

	planResp, err := client.RewritePlan(context.Background(), connect.NewRequest(&graphv1.RewritePlanRequest{
		ScenarioPath: "/abs/proj",
		Operations: []*rewritepb.Operation{
			{Op: &rewritepb.Operation_FileMove{FileMove: &rewritepb.FileMove{FromPath: "a.ts", ToPath: "b.ts"}}},
		},
	}))
	require.NoError(t, err)

	applyReq := connect.NewRequest(&graphv1.RewriteApplyRequest{
		ScenarioPath: "/abs/proj",
		PlanId:       planResp.Msg.GetPlanId(),
		Apply:        true,
	})
	applyReq.Header().Set("X-Dry-Run", "true")
	applyResp, err := client.RewriteApply(context.Background(), applyReq)
	require.NoError(t, err)
	require.True(t, applyResp.Msg.GetDryRun())
	require.Equal(t, 0, fake.RewriteApplyCalls, "dry-run must not call sidecar")
}

// TestHandlerRewriteApply_PlanNotFound confirms an unknown plan_id
// surfaces as FailedPrecondition.
func TestHandlerRewriteApply_PlanNotFound(t *testing.T) {
	t.Parallel()
	client := newTestClient(t, intgraph.NewService(&sidecarmocks.FakeSidecarClient{StatusValue: sidecar.StatusReady}, intgraph.NewPathMutex()))
	_, err := client.RewriteApply(context.Background(), connect.NewRequest(&graphv1.RewriteApplyRequest{
		ScenarioPath: "/abs/proj",
		PlanId:       "missing",
		Apply:        true,
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
}
