package flows

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"github.com/vrooli/cli-core/cliapp"
	cliapptest "github.com/vrooli/cli-core/cliapptest"

	clitest "flow-verifier/cli/internal/testutil"

	flowsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/flows"
	flowsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/flows/flows_v1connect"
)

type fakeService struct {
	listResp     *flowsv1.ListFlowsResponse
	explainResp  *flowsv1.ExplainFlowResponse
	createErr    error
	createInputs []*flowsv1.CreateFlowRequest
}

func (f *fakeService) ListFlows(context.Context, *connect.Request[flowsv1.ListFlowsRequest]) (*connect.Response[flowsv1.ListFlowsResponse], error) {
	if f.listResp == nil {
		f.listResp = &flowsv1.ListFlowsResponse{}
	}
	return connect.NewResponse(f.listResp), nil
}

func (f *fakeService) GetFlow(context.Context, *connect.Request[flowsv1.GetFlowRequest]) (*connect.Response[flowsv1.GetFlowResponse], error) {
	return connect.NewResponse(&flowsv1.GetFlowResponse{Flow: &flowsv1.FlowDetail{FlowId: "x"}}), nil
}

func (f *fakeService) CreateFlow(_ context.Context, req *connect.Request[flowsv1.CreateFlowRequest]) (*connect.Response[flowsv1.CreateFlowResponse], error) {
	f.createInputs = append(f.createInputs, req.Msg)
	if f.createErr != nil {
		return nil, f.createErr
	}
	return connect.NewResponse(&flowsv1.CreateFlowResponse{FlowDir: "scenarios/test/feature/flow"}), nil
}

func (f *fakeService) ValidateFlow(context.Context, *connect.Request[flowsv1.ValidateFlowRequest]) (*connect.Response[flowsv1.ValidateFlowResponse], error) {
	return connect.NewResponse(&flowsv1.ValidateFlowResponse{}), nil
}

func (f *fakeService) ExplainFlow(context.Context, *connect.Request[flowsv1.ExplainFlowRequest]) (*connect.Response[flowsv1.ExplainFlowResponse], error) {
	if f.explainResp == nil {
		f.explainResp = &flowsv1.ExplainFlowResponse{Report: "explain body"}
	}
	return connect.NewResponse(f.explainResp), nil
}

func (f *fakeService) CodegenFlow(context.Context, *connect.Request[flowsv1.CodegenFlowRequest]) (*connect.Response[flowsv1.CodegenFlowResponse], error) {
	return connect.NewResponse(&flowsv1.CodegenFlowResponse{}), nil
}

func (f *fakeService) ReconcileFlow(context.Context, *connect.Request[flowsv1.ReconcileFlowRequest]) (*connect.Response[flowsv1.ReconcileFlowResponse], error) {
	return connect.NewResponse(&flowsv1.ReconcileFlowResponse{Passed: true}), nil
}

func (f *fakeService) GetNavigationStudio(context.Context, *connect.Request[flowsv1.GetNavigationStudioRequest]) (*connect.Response[flowsv1.GetNavigationStudioResponse], error) {
	return connect.NewResponse(&flowsv1.GetNavigationStudioResponse{Descriptor_: &flowsv1.NavigationStudioDescriptor{Renderer: "navigation-graph"}}), nil
}

func connectAPI(t *testing.T, svc *fakeService) http.Handler {
	t.Helper()
	path, handler := flowsconnect.NewFlowsServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	return mux
}

func TestFlowsList_RendersResults(t *testing.T) {
	svc := &fakeService{listResp: &flowsv1.ListFlowsResponse{Flows: []*flowsv1.FlowSummary{
		{FlowId: "a", Language: "go", SchemaVersion: 6, ContractPath: "a/b/flow.json"},
	}}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "root"}, {Name: "kind"}}}
	ctx, out := cliapptest.NewCapturedRunContext(core, schema, cliapptest.TestRunContextOptions{Flags: map[string]string{"root": "."}})

	require.NoError(t, h.list(ctx))
	require.Contains(t, out.String(), "Found 1 flow(s)")
	require.Contains(t, out.String(), "a/b/flow.json")
}

func TestFlowsExplain_RendersReport(t *testing.T) {
	core := clitest.NewTestApp(t, connectAPI(t, &fakeService{}))
	h := newHandlers(core)
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "flow"}, {Name: "root"}}}
	ctx, out := cliapptest.NewCapturedRunContext(core, schema, cliapptest.TestRunContextOptions{Flags: map[string]string{"flow": "f1", "root": "."}})

	require.NoError(t, h.explain(ctx))
	require.Contains(t, out.String(), "explain body")
}

func TestFlowsCreate_WiresRequest(t *testing.T) {
	svc := &fakeService{}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	schema := cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "feature-dir", Required: true}},
		Flags:       []cliapp.Flag{{Name: "flow-id"}, {Name: "lang"}, {Name: "root"}, {Name: "kind"}},
	}
	ctx, _ := cliapptest.NewCapturedRunContext(core, schema, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"feature-dir": "scenarios/test/feature"},
		Flags:       map[string]string{"flow-id": "test.flow", "lang": "go", "root": "."},
	})

	require.NoError(t, h.create(ctx))
	require.Len(t, svc.createInputs, 1)
	require.Equal(t, "test.flow", svc.createInputs[0].FlowId)
	require.Equal(t, "go", svc.createInputs[0].Language)
}

func TestFlowsCreate_SurfacesAPIError(t *testing.T) {
	svc := &fakeService{createErr: errors.New("boom")}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	schema := cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "feature-dir", Required: true}},
		Flags:       []cliapp.Flag{{Name: "flow-id"}, {Name: "lang"}, {Name: "root"}, {Name: "kind"}},
	}
	ctx, _ := cliapptest.NewCapturedRunContext(core, schema, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"feature-dir": "scenarios/x/y"},
		Flags:       map[string]string{"flow-id": "x.y", "root": "."},
	})

	require.Error(t, h.create(ctx))
}
