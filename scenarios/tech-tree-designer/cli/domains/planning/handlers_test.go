package planning

import (
	"context"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	planningv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/planning"
	planningconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/planning/planning_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	cliapptest "github.com/vrooli/cli-core/cliapptest"

	clitest "tech-tree-designer/cli/internal/testutil"
)

type planningService struct {
	createReq   *planningv1.CreatePlannedScenarioRequest
	validateReq *planningv1.ValidatePlannedScenarioRequest
}

func (s *planningService) CreatePlannedScenario(_ context.Context, req *connect.Request[planningv1.CreatePlannedScenarioRequest]) (*connect.Response[planningv1.PlannedScenario], error) {
	s.createReq = req.Msg
	return connect.NewResponse(&planningv1.PlannedScenario{
		Slug:            req.Msg.GetSlug(),
		DisplayName:     req.Msg.GetDisplayName(),
		Sector:          req.Msg.GetSector(),
		Tier:            req.Msg.GetTier(),
		TargetStability: req.Msg.GetTargetStability(),
	}), nil
}

func (s *planningService) ListPlannedScenarios(context.Context, *connect.Request[planningv1.ListPlannedScenariosRequest]) (*connect.Response[planningv1.ListPlannedScenariosResponse], error) {
	return connect.NewResponse(&planningv1.ListPlannedScenariosResponse{Scenarios: []*planningv1.PlannedScenario{{Slug: "planned-demo"}}}), nil
}

func (s *planningService) GetPlannedScenario(context.Context, *connect.Request[planningv1.GetPlannedScenarioRequest]) (*connect.Response[planningv1.PlannedScenario], error) {
	return connect.NewResponse(&planningv1.PlannedScenario{
		Slug:  "planned-demo",
		Files: []*planningv1.PlannedProtoFile{{Path: "planned-demo/v1/api/service.proto", Text: "syntax = \"proto3\";\n"}},
	}), nil
}

func (s *planningService) PutPlannedProtoFile(context.Context, *connect.Request[planningv1.PutPlannedProtoFileRequest]) (*connect.Response[planningv1.PlannedProtoFile], error) {
	return connect.NewResponse(&planningv1.PlannedProtoFile{Path: "planned-demo/v1/api/service.proto"}), nil
}

func (s *planningService) DeletePlannedProtoFile(context.Context, *connect.Request[planningv1.DeletePlannedProtoFileRequest]) (*connect.Response[planningv1.DeletePlannedProtoFileResponse], error) {
	return connect.NewResponse(&planningv1.DeletePlannedProtoFileResponse{Deleted: true}), nil
}

func (s *planningService) ValidatePlannedScenario(_ context.Context, req *connect.Request[planningv1.ValidatePlannedScenarioRequest]) (*connect.Response[planningv1.ValidatePlannedScenarioResponse], error) {
	s.validateReq = req.Msg
	return connect.NewResponse(&planningv1.ValidatePlannedScenarioResponse{Slug: req.Msg.GetSlug(), Passed: true}), nil
}

func (s *planningService) MaterializePlannedScenario(context.Context, *connect.Request[planningv1.MaterializePlannedScenarioRequest]) (*connect.Response[planningv1.MaterializePlannedScenarioResponse], error) {
	return connect.NewResponse(&planningv1.MaterializePlannedScenarioResponse{Slug: "planned-demo", Generated: true}), nil
}

func planningAPI(t *testing.T, svc *planningService) http.Handler {
	t.Helper()
	path, handler := planningconnect.NewPlanningServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	return mux
}

func TestCreateCallsConnectAndRendersMutation(t *testing.T) {
	svc := &planningService{}
	core := clitest.NewTestApp(t, planningAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "slug", Required: true}},
		Flags:       []cliapp.Flag{{Name: "display-name"}, {Name: "sector"}, {Name: "tier"}, {Name: "stability", Default: "experimental"}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"slug": "planned-demo"},
		Flags: map[string]string{
			"display-name": "Planned Demo",
			"sector":       "engineering",
			"tier":         "foundation",
			"stability":    "experimental",
		},
	})

	require.NoError(t, h.create(ctx))
	require.Equal(t, "planned-demo", svc.createReq.GetSlug())
	require.Equal(t, "engineering", svc.createReq.GetSector())
	require.Contains(t, out.String(), "Planned scenario planned-demo saved.")
}

func TestValidateJSONIsProtoWireShape(t *testing.T) {
	svc := &planningService{}
	core := clitest.NewTestApp(t, planningAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "slug", Required: true}},
	}, cliapptest.TestRunContextOptions{
		JSON:        true,
		Positionals: map[string]string{"slug": "planned-demo"},
	})

	require.NoError(t, h.validate(ctx))
	require.Equal(t, "planned-demo", svc.validateReq.GetSlug())
	var got planningv1.ValidatePlannedScenarioResponse
	require.NoError(t, protojson.Unmarshal(out.Bytes(), &got))
	require.True(t, got.GetPassed())
}
