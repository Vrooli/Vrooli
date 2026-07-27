package pipeline

import (
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	pipelinev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/pipeline"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/shared"
)

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
