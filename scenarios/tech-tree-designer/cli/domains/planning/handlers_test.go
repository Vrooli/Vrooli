package planning

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
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
	createReq      *planningv1.CreatePlannedScenarioRequest
	validateReq    *planningv1.ValidatePlannedScenarioRequest
	listReq        *planningv1.ListPlannedScenariosRequest
	putReq         *planningv1.PutPlannedProtoFileRequest
	removeReq      *planningv1.DeletePlannedProtoFileRequest
	materializeReq *planningv1.MaterializePlannedScenarioRequest
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

func (s *planningService) ListPlannedScenarios(_ context.Context, req *connect.Request[planningv1.ListPlannedScenariosRequest]) (*connect.Response[planningv1.ListPlannedScenariosResponse], error) {
	s.listReq = req.Msg
	return connect.NewResponse(&planningv1.ListPlannedScenariosResponse{Scenarios: []*planningv1.PlannedScenario{{Slug: "planned-demo", Sector: "engineering", Tier: "foundation"}}}), nil
}

func (s *planningService) GetPlannedScenario(context.Context, *connect.Request[planningv1.GetPlannedScenarioRequest]) (*connect.Response[planningv1.PlannedScenario], error) {
	return connect.NewResponse(&planningv1.PlannedScenario{
		Slug:  "planned-demo",
		Files: []*planningv1.PlannedProtoFile{{Path: "planned-demo/v1/api/service.proto", Text: "syntax = \"proto3\";\n"}},
	}), nil
}

func (s *planningService) PutPlannedProtoFile(_ context.Context, req *connect.Request[planningv1.PutPlannedProtoFileRequest]) (*connect.Response[planningv1.PlannedProtoFile], error) {
	s.putReq = req.Msg
	return connect.NewResponse(&planningv1.PlannedProtoFile{Path: req.Msg.GetPath()}), nil
}

func (s *planningService) DeletePlannedProtoFile(_ context.Context, req *connect.Request[planningv1.DeletePlannedProtoFileRequest]) (*connect.Response[planningv1.DeletePlannedProtoFileResponse], error) {
	s.removeReq = req.Msg
	return connect.NewResponse(&planningv1.DeletePlannedProtoFileResponse{Deleted: true}), nil
}

func (s *planningService) ValidatePlannedScenario(_ context.Context, req *connect.Request[planningv1.ValidatePlannedScenarioRequest]) (*connect.Response[planningv1.ValidatePlannedScenarioResponse], error) {
	s.validateReq = req.Msg
	return connect.NewResponse(&planningv1.ValidatePlannedScenarioResponse{Slug: req.Msg.GetSlug(), Passed: true}), nil
}

func (s *planningService) MaterializePlannedScenario(_ context.Context, req *connect.Request[planningv1.MaterializePlannedScenarioRequest]) (*connect.Response[planningv1.MaterializePlannedScenarioResponse], error) {
	s.materializeReq = req.Msg
	return connect.NewResponse(&planningv1.MaterializePlannedScenarioResponse{
		Slug:         req.Msg.GetSlug(),
		Generated:    true,
		WrittenPaths: []string{"scenarios/planned-demo/.vrooli/service.json"},
	}), nil
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

func TestListTranslatesFiltersAndRenders(t *testing.T) {
	svc := &planningService{}
	core := clitest.NewTestApp(t, planningAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Flags: []cliapp.Flag{{Name: "sector"}, {Name: "tier"}},
	}, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"sector": "engineering", "tier": "foundation"},
	})

	require.NoError(t, h.list(ctx))
	require.Equal(t, "engineering", svc.listReq.GetSector())
	require.Equal(t, "foundation", svc.listReq.GetTier())
	require.Contains(t, out.String(), "1 planned scenario(s).")
	require.Contains(t, out.String(), "planned-demo")
}

func TestGetListsFilesWhenNoPathPositional(t *testing.T) {
	core := clitest.NewTestApp(t, planningAPI(t, &planningService{}))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "slug", Required: true}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"slug": "planned-demo"},
		RawArgs:     []string{"planned-demo"},
	})

	require.NoError(t, h.get(ctx))
	require.Contains(t, out.String(), "planned-demo/v1/api/service.proto")
}

func TestGetPrintsFileTextWhenPathPositionalMatches(t *testing.T) {
	core := clitest.NewTestApp(t, planningAPI(t, &planningService{}))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "slug", Required: true}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"slug": "planned-demo"},
		RawArgs:     []string{"planned-demo", "planned-demo/v1/api/service.proto"},
	})

	require.NoError(t, h.get(ctx))
	require.Equal(t, "syntax = \"proto3\";\n", out.String())
}

func TestGetErrorsWhenPathPositionalMissing(t *testing.T) {
	core := clitest.NewTestApp(t, planningAPI(t, &planningService{}))
	h := newHandlers(core)
	ctx, _ := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "slug", Required: true}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"slug": "planned-demo"},
		RawArgs:     []string{"planned-demo", "planned-demo/v1/api/missing.proto"},
	})

	require.Error(t, h.get(ctx))
}

func TestPutReadsFileAndStores(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "service.proto")
	require.NoError(t, os.WriteFile(file, []byte("syntax = \"proto3\";\n"), 0o600))
	svc := &planningService{}
	core := clitest.NewTestApp(t, planningAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "slug", Required: true}, {Name: "path", Required: true}},
		Flags:       []cliapp.Flag{{Name: "from-file"}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"slug": "planned-demo", "path": "planned-demo/v1/api/service.proto"},
		Flags:       map[string]string{"from-file": file},
	})

	require.NoError(t, h.put(ctx))
	require.Equal(t, "planned-demo", svc.putReq.GetSlug())
	require.Equal(t, "planned-demo/v1/api/service.proto", svc.putReq.GetPath())
	require.Equal(t, "syntax = \"proto3\";\n", svc.putReq.GetText())
	require.Contains(t, out.String(), "Stored planned-demo/v1/api/service.proto.")
}

func TestPutErrorsWhenFileMissing(t *testing.T) {
	svc := &planningService{}
	core := clitest.NewTestApp(t, planningAPI(t, svc))
	h := newHandlers(core)
	ctx, _ := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "slug", Required: true}, {Name: "path", Required: true}},
		Flags:       []cliapp.Flag{{Name: "from-file"}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"slug": "planned-demo", "path": "p"},
		Flags:       map[string]string{"from-file": "/no/such/file.proto"},
	})

	require.Error(t, h.put(ctx))
}

func TestRemoveReportsDeleted(t *testing.T) {
	svc := &planningService{}
	core := clitest.NewTestApp(t, planningAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "slug", Required: true}, {Name: "path", Required: true}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"slug": "planned-demo", "path": "planned-demo/v1/api/service.proto"},
	})

	require.NoError(t, h.remove(ctx))
	require.Equal(t, "planned-demo", svc.removeReq.GetSlug())
	require.Equal(t, "planned-demo/v1/api/service.proto", svc.removeReq.GetPath())
	require.Contains(t, out.String(), "Deleted: true.")
}

func TestMaterializeRendersWrittenPaths(t *testing.T) {
	svc := &planningService{}
	core := clitest.NewTestApp(t, planningAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "slug", Required: true}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"slug": "planned-demo"},
	})

	require.NoError(t, h.materialize(ctx))
	require.Equal(t, "planned-demo", svc.materializeReq.GetSlug())
	require.Contains(t, out.String(), "Materialized planned-demo; generated=true.")
	require.Contains(t, out.String(), "scenarios/planned-demo/.vrooli/service.json")
}

func TestReadProtoTextFromFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "x.proto")
	require.NoError(t, os.WriteFile(file, []byte("body"), 0o600))
	got, err := readProtoText(file)
	require.NoError(t, err)
	require.Equal(t, "body", got)

	_, err = readProtoText("/no/such/file.proto")
	require.Error(t, err)
}

func TestScenarioLine(t *testing.T) {
	line := scenarioLine(&planningv1.PlannedScenario{
		Slug:   "planned-demo",
		Sector: "engineering",
		Tier:   "foundation",
		Files:  []*planningv1.PlannedProtoFile{{Path: "a"}, {Path: "b"}},
	})
	require.Equal(t, "planned-demo [engineering foundation] files=2", line)
}
