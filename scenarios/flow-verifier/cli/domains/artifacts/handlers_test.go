package artifacts

import (
	"context"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"github.com/vrooli/cli-core/cliapp"
	cliapptest "github.com/vrooli/cli-core/cliapptest"

	clitest "github.com/vrooli/cli-core/cliapptest"

	artifactsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/artifacts"
	artifactsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/artifacts/artifacts_v1connect"
	scenariosv1 "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/scenarios"
	scenariosconnect "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/scenarios/scenarios_v1connect"
)

type fakeArtifacts struct {
	report *artifactsv1.ArtifactReport
}

func (f *fakeArtifacts) GetArtifactStatus(context.Context, *connect.Request[artifactsv1.GetArtifactStatusRequest]) (*connect.Response[artifactsv1.GetArtifactStatusResponse], error) {
	return connect.NewResponse(&artifactsv1.GetArtifactStatusResponse{Report: f.report}), nil
}

func (f *fakeArtifacts) GenerateArtifacts(context.Context, *connect.Request[artifactsv1.GenerateArtifactsRequest]) (*connect.Response[artifactsv1.GenerateArtifactsResponse], error) {
	return connect.NewResponse(&artifactsv1.GenerateArtifactsResponse{Report: f.report}), nil
}

func (f *fakeArtifacts) ClearArtifacts(context.Context, *connect.Request[artifactsv1.ClearArtifactsRequest]) (*connect.Response[artifactsv1.ClearArtifactsResponse], error) {
	return connect.NewResponse(&artifactsv1.ClearArtifactsResponse{FlowId: "f1", Removed: []string{"a.go"}}), nil
}

type fakeScenarios struct {
	stream []*scenariosv1.GenerateScenarioArtifactsResponse
}

func (f *fakeScenarios) ListScenarios(context.Context, *connect.Request[scenariosv1.ListScenariosRequest]) (*connect.Response[scenariosv1.ListScenariosResponse], error) {
	return connect.NewResponse(&scenariosv1.ListScenariosResponse{}), nil
}

func (f *fakeScenarios) GetScenario(context.Context, *connect.Request[scenariosv1.GetScenarioRequest]) (*connect.Response[scenariosv1.GetScenarioResponse], error) {
	return connect.NewResponse(&scenariosv1.GetScenarioResponse{}), nil
}

func (f *fakeScenarios) GenerateScenarioArtifacts(_ context.Context, _ *connect.Request[scenariosv1.GenerateScenarioArtifactsRequest], stream *connect.ServerStream[scenariosv1.GenerateScenarioArtifactsResponse]) error {
	for _, msg := range f.stream {
		if err := stream.Send(msg); err != nil {
			return err
		}
	}
	return nil
}

func (f *fakeScenarios) ClearScenarioArtifacts(context.Context, *connect.Request[scenariosv1.ClearScenarioArtifactsRequest]) (*connect.Response[scenariosv1.ClearScenarioArtifactsResponse], error) {
	return connect.NewResponse(&scenariosv1.ClearScenarioArtifactsResponse{}), nil
}

func connectAPI(t *testing.T, a *fakeArtifacts, s *fakeScenarios) http.Handler {
	t.Helper()
	mux := http.NewServeMux()
	pathA, hA := artifactsconnect.NewArtifactsServiceHandler(a)
	mux.Handle(pathA, hA)
	pathS, hS := scenariosconnect.NewScenariosServiceHandler(s)
	mux.Handle(pathS, hS)
	return mux
}

func TestArtifactsStatus_RendersReport(t *testing.T) {
	report := &artifactsv1.ArtifactReport{FlowId: "f1", Status: artifactsv1.ArtifactStatus_ARTIFACT_STATUS_FRESH}
	core := clitest.NewTestApp(t, connectAPI(t, &fakeArtifacts{report: report}, &fakeScenarios{}))
	h := newHandlers(core)
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "root"}, {Name: "flow"}, {Name: "scenario"}, {Name: "all"}}}
	ctx, out := cliapptest.NewCapturedRunContext(core, schema, cliapptest.TestRunContextOptions{Flags: map[string]string{"flow": "f1", "root": "."}})

	require.NoError(t, h.status(ctx))
	require.Contains(t, out.String(), "ARTIFACT_STATUS_FRESH")
}

func TestArtifactsGenerate_Scenario_StreamsProgress(t *testing.T) {
	stream := []*scenariosv1.GenerateScenarioArtifactsResponse{
		{FlowId: "f1", Report: &artifactsv1.ArtifactReport{Status: artifactsv1.ArtifactStatus_ARTIFACT_STATUS_FRESH}},
		{FlowId: "f2", ErrorMessage: "boom"},
	}
	core := clitest.NewTestApp(t, connectAPI(t, &fakeArtifacts{}, &fakeScenarios{stream: stream}))
	h := newHandlers(core)
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "root"}, {Name: "flow"}, {Name: "scenario"}, {Name: "all"}}}
	ctx, out := cliapptest.NewCapturedRunContext(core, schema, cliapptest.TestRunContextOptions{Flags: map[string]string{"scenario": "s1"}})

	require.NoError(t, h.generate(ctx))
	require.Contains(t, out.String(), "f1")
	require.Contains(t, out.String(), "f2")
	require.Contains(t, out.String(), "boom")
	require.Contains(t, out.String(), "Generated artifacts for 2 flow(s)")
}

func TestArtifactsClear_MutuallyExclusiveScopes(t *testing.T) {
	core := clitest.NewTestApp(t, connectAPI(t, &fakeArtifacts{}, &fakeScenarios{}))
	h := newHandlers(core)
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "root"}, {Name: "flow"}, {Name: "scenario"}, {Name: "all"}, {Name: "yes"}}}
	ctx, _ := cliapptest.NewCapturedRunContext(core, schema, cliapptest.TestRunContextOptions{Flags: map[string]string{"flow": "f1", "scenario": "s1"}})

	err := h.clear(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "mutually exclusive")
}
