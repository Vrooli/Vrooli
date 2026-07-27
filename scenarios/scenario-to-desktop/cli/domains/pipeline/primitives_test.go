package pipeline

import (
	"context"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"
	pipelinev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/pipeline"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/shared"
)

type fakePipelineRPC struct{ runConfig *pipelinev1.PipelineConfig }

func pipelineStatusFixture() *pipelinev1.PipelineStatus {
	stage := sharedv1.StageName_STAGE_NAME_BUILD
	return &pipelinev1.PipelineStatus{PipelineId: "pipe-1", ScenarioName: "calculator", Status: sharedv1.StageStatus_STAGE_STATUS_RUNNING, CurrentStage: &stage, ProgressPercent: 50}
}

func (f *fakePipelineRPC) Run(_ context.Context, r *connect.Request[pipelinev1.PipelineRunRequest]) (*connect.Response[pipelinev1.PipelineRunResponse], error) {
	f.runConfig = r.Msg.GetConfig()
	return connect.NewResponse(&pipelinev1.PipelineRunResponse{PipelineId: "pipe-1"}), nil
}

func (*fakePipelineRPC) StartActive(context.Context, *connect.Request[pipelinev1.StartActivePipelineRequest]) (*connect.Response[pipelinev1.StartActivePipelineResponse], error) {
	return connect.NewResponse(&pipelinev1.StartActivePipelineResponse{Pipeline: pipelineStatusFixture()}), nil
}

func (*fakePipelineRPC) Get(context.Context, *connect.Request[pipelinev1.PipelineGetRequest]) (*connect.Response[pipelinev1.PipelineStatus], error) {
	return connect.NewResponse(pipelineStatusFixture()), nil
}

func (*fakePipelineRPC) GetReleaseGate(context.Context, *connect.Request[pipelinev1.PipelineGetRequest]) (*connect.Response[pipelinev1.PipelineStatus], error) {
	return connect.NewResponse(pipelineStatusFixture()), nil
}

func (*fakePipelineRPC) Resume(context.Context, *connect.Request[pipelinev1.PipelineResumeRequest]) (*connect.Response[pipelinev1.PipelineResumeResponse], error) {
	return connect.NewResponse(&pipelinev1.PipelineResumeResponse{PipelineId: "pipe-1"}), nil
}

func (*fakePipelineRPC) Cancel(context.Context, *connect.Request[pipelinev1.PipelineCancelRequest]) (*connect.Response[pipelinev1.PipelineCancelResponse], error) {
	return connect.NewResponse(&pipelinev1.PipelineCancelResponse{Status: "cancelled"}), nil
}

func (*fakePipelineRPC) List(context.Context, *connect.Request[pipelinev1.PipelineListRequest]) (*connect.Response[pipelinev1.PipelineListResponse], error) {
	return connect.NewResponse(&pipelinev1.PipelineListResponse{Pipelines: []*pipelinev1.PipelineListItem{{PipelineId: "pipe-1", ScenarioName: "calculator"}}}), nil
}

func (*fakePipelineRPC) GetActive(context.Context, *connect.Request[pipelinev1.GetActivePipelineRequest]) (*connect.Response[pipelinev1.ActivePipelineResponse], error) {
	return connect.NewResponse(&pipelinev1.ActivePipelineResponse{Pipeline: pipelineStatusFixture()}), nil
}

func (*fakePipelineRPC) CreateActive(context.Context, *connect.Request[pipelinev1.CreatePipelineRequest]) (*connect.Response[pipelinev1.CreatePipelineResponse], error) {
	return connect.NewResponse(&pipelinev1.CreatePipelineResponse{Pipeline: pipelineStatusFixture()}), nil
}

func (*fakePipelineRPC) ResetActive(context.Context, *connect.Request[pipelinev1.ScenarioPipelineRequest]) (*connect.Response[pipelinev1.ResetPipelineResponse], error) {
	return connect.NewResponse(&pipelinev1.ResetPipelineResponse{Cleared: true}), nil
}

func (*fakePipelineRPC) GetHistory(context.Context, *connect.Request[pipelinev1.PipelineHistoryRequest]) (*connect.Response[pipelinev1.PipelineHistoryResponse], error) {
	return connect.NewResponse(&pipelinev1.PipelineHistoryResponse{Pipelines: []*pipelinev1.PipelineStatus{pipelineStatusFixture()}, Total: 1}), nil
}

func TestPipelineConfigFromContext_BuildsTypedReleaseRequest(t *testing.T) {
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema: pipelineRunArgs(),
		Positionals: map[string]string{
			"scenario": " calculator ",
		},
		Flags: map[string]string{
			"platforms":              "windows-x64, macos-arm64, linux",
			"stages":                 "bundle, preflight, smoketest",
			"deployment-mode":        "proxy",
			"location-mode":          "staging",
			"resource-artifact-root": "/verified/artifacts",
			"version":                "1.2.3",
		},
		BoolFlags: map[string]bool{"clean": true},
	})

	config, err := pipelineConfigFromContext(ctx)
	if err != nil {
		t.Fatalf("pipelineConfigFromContext() error: %v", err)
	}
	if config.GetScenarioName() != "calculator" {
		t.Errorf("scenarioName = %q, want calculator", config.GetScenarioName())
	}
	if got, want := config.GetPlatforms(), []sharedv1.Platform{sharedv1.Platform_PLATFORM_WIN, sharedv1.Platform_PLATFORM_MAC, sharedv1.Platform_PLATFORM_LINUX}; !equalPlatforms(got, want) {
		t.Errorf("platforms = %v, want %v", got, want)
	}
	if got, want := config.GetStages(), []sharedv1.StageName{sharedv1.StageName_STAGE_NAME_BUNDLE, sharedv1.StageName_STAGE_NAME_PREFLIGHT, sharedv1.StageName_STAGE_NAME_SMOKE_TEST}; !equalStages(got, want) {
		t.Errorf("stages = %v, want %v", got, want)
	}
	if config.GetDeploymentMode() != sharedv1.DeploymentMode_DEPLOYMENT_MODE_PROXY || config.GetLocationMode() != "staging" || config.GetResourceArtifactRoot() != "/verified/artifacts" || !config.GetClean() || config.GetVersion() != "1.2.3" {
		t.Errorf("unexpected config: %+v", config)
	}
}

func TestPipelineConfigFromContext_RejectsUnsupportedOperatorInput(t *testing.T) {
	for _, tc := range []struct {
		name  string
		flags map[string]string
		want  string
	}{
		{name: "platform", flags: map[string]string{"platforms": "ios"}, want: "unsupported platform"},
		{name: "stage", flags: map[string]string{"stages": "ship"}, want: "unsupported stage"},
		{name: "mode", flags: map[string]string{"deployment-mode": "cloud"}, want: "unsupported deployment mode"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
				Schema:      pipelineRunArgs(),
				Positionals: map[string]string{"scenario": "calculator"},
				Flags:       tc.flags,
			})
			_, err := pipelineConfigFromContext(ctx)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("pipelineConfigFromContext() error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestPipelineParsingAndReports(t *testing.T) {
	if got := splitValues(" WIN, ,macOS , linux "); strings.Join(got, ",") != "win,macos,linux" {
		t.Errorf("splitValues() = %v", got)
	}
	if _, err := positiveInt32("0", "limit"); err == nil {
		t.Fatal("positiveInt32 accepted zero")
	}
	if got, err := positiveInt32("20", "limit"); err != nil || got != 20 {
		t.Errorf("positiveInt32() = %d, %v", got, err)
	}

	run := pipelineRunReport(nil, &pipelinev1.PipelineRunResponse{PipelineId: "pipe-1"})
	if run.Result[0] != "Pipeline started: pipe-1" {
		t.Errorf("run report = %+v", run)
	}
	status := pipelineStatusReport(nil, &pipelinev1.PipelineStatus{PipelineId: "pipe-1", ScenarioName: "calculator", Status: sharedv1.StageStatus_STAGE_STATUS_RUNNING, CurrentStage: sharedv1.StageName_STAGE_NAME_BUILD.Enum(), ProgressPercent: 50})
	if status.Summary[0] != "Pipeline pipe-1 is STAGE_STATUS_RUNNING (50%)" {
		t.Errorf("status report = %+v", status)
	}
}

func equalPlatforms(got, want []sharedv1.Platform) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range want {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func equalStages(got, want []sharedv1.StageName) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range want {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func TestPipelinePrimitivesUseTypedConnectContract(t *testing.T) {
	rpc := &fakePipelineRPC{}
	commands := &Commands{rpc: rpc}
	tests := []struct {
		name    string
		handler cliapp.PrimitiveHandler
		schema  cliapp.ArgSchema
		args    []string
	}{
		{"run", commands.runPrimitive(), pipelineRunArgs(), []string{"calculator", "--platforms", "linux", "--stages", "build"}},
		{"start", commands.startPrimitive(), pipelineRunArgs(), []string{"calculator"}},
		{"status", commands.statusPrimitive(), pipelineIDArgs(), []string{"pipe-1"}},
		{"gate", commands.gatePrimitive(), pipelineIDArgs(), []string{"pipe-1"}},
		{"resume", commands.resumePrimitive(), pipelineIDArgs(), []string{"pipe-1"}},
		{"cancel", commands.cancelPrimitive(), pipelineIDArgs(), []string{"pipe-1"}},
		{"list", commands.listPrimitive(), cliapp.ArgSchema{}, nil},
		{"active", commands.activePrimitive(), pipelineScenarioArgs(), []string{"calculator"}},
		{"create", commands.createPrimitive(), pipelineScenarioArgs(), []string{"calculator"}},
		{"reset", commands.resetPrimitive(), pipelineScenarioArgs(), []string{"calculator"}},
		{"history", commands.historyPrimitive(), pipelineScenarioArgs(cliapp.Flag{Name: "limit", Default: "10"}), []string{"calculator", "--limit", "3"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			modes := cliapptest.RunPrimitiveHandlerModes(t, test.handler, test.schema, test.args, nil)
			if modes.HumanErr != nil || modes.JSONErr != nil {
				t.Fatalf("primitive errors: human=%v json=%v", modes.HumanErr, modes.JSONErr)
			}
		})
	}
	if rpc.runConfig.GetScenarioName() != "calculator" || rpc.runConfig.GetPlatforms()[0] != sharedv1.Platform_PLATFORM_LINUX || rpc.runConfig.GetStages()[0] != sharedv1.StageName_STAGE_NAME_BUILD {
		t.Fatalf("run config = %#v", rpc.runConfig)
	}
}
