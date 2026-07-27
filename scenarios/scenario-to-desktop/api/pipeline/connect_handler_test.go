package pipeline

import (
	"context"
	"scenario-to-desktop-api/build"
	"scenario-to-desktop-api/bundle"
	"scenario-to-desktop-api/generation"
	"scenario-to-desktop-api/preflight"
	"scenario-to-desktop-api/smoketest"
	"testing"

	"connectrpc.com/connect"
	pipelinev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/pipeline"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/shared"
)

func TestConnectServiceGetPreservesPipelineInformation(t *testing.T) {
	status := &Status{
		PipelineID:      "pipeline-1",
		ScenarioName:    "example",
		Status:          StatusRunning,
		CurrentStage:    StageResolveDeployment,
		CurrentState:    PipelineStateGateBlocked,
		ProgressPercent: 14,
		ProgressMessage: "Resolving deployment",
		StartedAt:       1710000000,
		FinalArtifacts:  map[string]string{"linux": "/tmp/app"},
		Config:          &Config{ScenarioName: "example", Framework: FrameworkElectron, Platforms: []string{"linux"}},
		IdempotencyKey:  "request-1",
		Stages: map[string]*StageResult{
			"bundle": {
				Stage:       StageBundle,
				Status:      StatusCompleted,
				StartedAt:   1710000000,
				CompletedAt: 1710000010,
				Logs:        []string{"bundle complete"},
				Details:     &bundle.PackageResult{BundleDir: "/tmp/bundle"},
			},
		},
	}
	service := NewConnectService(NewHandler(WithOrchestrator(&mockOrchestrator{getResult: status, getFound: true})))

	response, err := service.Get(context.Background(), connect.NewRequest(&pipelinev1.PipelineGetRequest{PipelineId: "pipeline-1"}))
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got := response.Msg.GetCurrentStage(); got != sharedv1.StageName_STAGE_NAME_RESOLVE_DEPLOYMENT {
		t.Fatalf("current stage = %v, want resolve deployment", got)
	}
	if got := response.Msg.GetCurrentState(); got != string(PipelineStateGateBlocked) {
		t.Fatalf("current state = %q, want %q", got, PipelineStateGateBlocked)
	}
	if got := response.Msg.GetConfig().GetPlatforms(); len(got) != 1 || got[0] != sharedv1.Platform_PLATFORM_LINUX {
		t.Fatalf("platforms = %v, want linux", got)
	}
	if got := response.Msg.GetFinalArtifacts()["linux"]; got != "/tmp/app" {
		t.Fatalf("final artifact = %q, want /tmp/app", got)
	}
	stage := response.Msg.GetStages()["bundle"]
	if stage == nil || stage.GetStatus() != sharedv1.StageStatus_STAGE_STATUS_COMPLETED || stage.GetDetails().GetBundle().GetBundleDir() != "/tmp/bundle" {
		t.Fatalf("bundle stage was not preserved: %#v", stage)
	}
}

func TestConnectServiceGetReleaseGateUsesDedicatedContract(t *testing.T) {
	status := &Status{
		PipelineID:      "pipeline-gate-1",
		ScenarioName:    "example",
		Status:          StatusRunning,
		CurrentStage:    StageDeploy,
		CurrentState:    PipelineStateGateBlocked,
		ProgressMessage: "Waiting for approval",
	}
	service := NewConnectService(NewHandler(WithOrchestrator(&mockOrchestrator{getResult: status, getFound: true})))

	response, err := service.GetReleaseGate(context.Background(), connect.NewRequest(&pipelinev1.PipelineGetRequest{PipelineId: status.PipelineID}))
	if err != nil {
		t.Fatalf("GetReleaseGate() error = %v", err)
	}
	if got := response.Msg.GetCurrentState(); got != string(PipelineStateGateBlocked) {
		t.Fatalf("current state = %q, want %q", got, PipelineStateGateBlocked)
	}
}

func TestStageDetailsToProtoPreservesEveryPipelineResultKind(t *testing.T) {
	tests := []struct {
		name  string
		input any
		check func(*pipelinev1.StageDetails) bool
	}{
		{
			name:  "resolve deployment",
			input: &ResourceDeploymentPlan{SchemaVersion: "v1", Resources: []ResourceDeploymentPlanItem{{Resource: "redis", Service: &ResourceDeploymentService{Artifact: "redis"}}}},
			check: func(value *pipelinev1.StageDetails) bool {
				return value.GetResolveDeployment().GetSchemaVersion() == "v1" && value.GetResolveDeployment().GetResources()[0].GetService().GetArtifact() == "redis"
			},
		},
		{
			name:  "bundle",
			input: &bundle.PackageResult{BundleDir: "/tmp/bundle", RuntimeBinaries: map[string]string{"linux": "runtime"}},
			check: func(value *pipelinev1.StageDetails) bool {
				return value.GetBundle().GetBundleDir() == "/tmp/bundle" && value.GetBundle().GetRuntimeBinaries()["linux"] == "runtime"
			},
		},
		{
			name:  "preflight",
			input: &preflight.Response{Status: "passed"},
			check: func(value *pipelinev1.StageDetails) bool {
				return value.GetPreflight().GetStatus().String() == "PREFLIGHT_STATUS_PASSED"
			},
		},
		{
			name:  "generate",
			input: &generation.GenerateResponse{PipelineID: "pipeline-1", DesktopPath: "/tmp/desktop", DetectedMetadata: &generation.ScenarioMetadata{Name: "example", HasUI: true}},
			check: func(value *pipelinev1.StageDetails) bool {
				return value.GetGenerate().GetPipelineId() == "pipeline-1" && value.GetGenerate().GetDetectedMetadata().GetHasUi()
			},
		},
		{
			name:  "build",
			input: &build.Status{BuildID: "build-1", Status: "ready", RequestedPlatforms: []string{"linux"}},
			check: func(value *pipelinev1.StageDetails) bool {
				return value.GetBuild().GetBuildId() == "build-1" && value.GetBuild().GetRequestedPlatforms()[0] == sharedv1.Platform_PLATFORM_LINUX
			},
		},
		{
			name:  "smoke test",
			input: &smoketest.Status{SmokeTestID: "smoke-1", Platform: "linux", Status: "passed"},
			check: func(value *pipelinev1.StageDetails) bool {
				return value.GetSmokeTest().GetSmokeTestId() == "smoke-1" && value.GetSmokeTest().GetStatus().String() == "SMOKE_TEST_STATUS_PASSED"
			},
		},
		{
			name:  "deploy",
			input: &DeployResult{UpdateURL: "https://updates.example.test", Artifacts: []DeployArtifactResult{{ArtifactID: 7, Platform: "linux"}}},
			check: func(value *pipelinev1.StageDetails) bool {
				return value.GetDeploy().GetUpdateUrl() == "https://updates.example.test" && value.GetDeploy().GetArtifacts()[0].GetPlatform() == sharedv1.Platform_PLATFORM_LINUX
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := stageDetailsToProto(test.input)
			if result == nil || !test.check(result) {
				t.Fatalf("stage details were not preserved: %#v", result)
			}
		})
	}
}

func TestConnectServiceRunRejectsMissingScenario(t *testing.T) {
	service := NewConnectService(NewHandler(WithOrchestrator(&mockOrchestrator{})))
	_, err := service.Run(context.Background(), connect.NewRequest(&pipelinev1.PipelineRunRequest{Config: &pipelinev1.PipelineConfig{}}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("error code = %v, want invalid argument (err=%v)", connect.CodeOf(err), err)
	}
}

func TestConnectServiceRunRejectsUnknownFrameworkEnum(t *testing.T) {
	service := NewConnectService(NewHandler(WithOrchestrator(&mockOrchestrator{})))
	_, err := service.Run(context.Background(), connect.NewRequest(&pipelinev1.PipelineRunRequest{Config: &pipelinev1.PipelineConfig{
		ScenarioName: "example",
		Framework:    sharedv1.Framework(99),
	}}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("error code = %v, want invalid argument (err=%v)", connect.CodeOf(err), err)
	}
}

func TestConnectServiceListFiltersScenario(t *testing.T) {
	service := NewConnectService(NewHandler(WithOrchestrator(&mockOrchestrator{pipelines: []*Status{
		{PipelineID: "one", ScenarioName: "wanted", Status: StatusIdle, StartedAt: 1710000000},
		{PipelineID: "two", ScenarioName: "other", Status: StatusCompleted, StartedAt: 1710000001},
	}})))

	response, err := service.List(context.Background(), connect.NewRequest(&pipelinev1.PipelineListRequest{ScenarioName: stringPtr("wanted")}))
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if got := response.Msg.GetPipelines(); len(got) != 1 || got[0].GetPipelineId() != "one" || got[0].GetStatus() != sharedv1.StageStatus_STAGE_STATUS_IDLE {
		t.Fatalf("unexpected list response: %#v", got)
	}
}

func TestConnectServiceGetActiveReturnsExistingPipelineWithoutCreating(t *testing.T) {
	status := &Status{PipelineID: "active-1", ScenarioName: "example", Status: StatusIdle}
	indexStore, err := NewScenarioIndexStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewScenarioIndexStore() error = %v", err)
	}
	if err := indexStore.SetActivePipeline("example", status.PipelineID); err != nil {
		t.Fatalf("SetActivePipeline() error = %v", err)
	}
	manager := NewManager(
		WithManagerOrchestrator(&mockOrchestrator{getResult: status, getFound: true}),
		WithManagerIndexStore(indexStore),
	)
	service := NewConnectService(NewHandler(WithManager(manager)))

	response, err := service.GetActive(context.Background(), connect.NewRequest(&pipelinev1.GetActivePipelineRequest{
		ScenarioName: "example",
		AutoCreate:   false,
	}))
	if err != nil {
		t.Fatalf("GetActive() error = %v", err)
	}
	if got := response.Msg.GetPipeline().GetPipelineId(); got != "active-1" {
		t.Fatalf("active pipeline id = %q, want active-1", got)
	}
	if response.Msg.GetCreated() {
		t.Fatal("GetActive(auto_create=false) unexpectedly reported a creation")
	}
}

func TestConnectServiceCleanBundleRejectsMissingStagingPipelineID(t *testing.T) {
	service := NewConnectService(NewHandler())
	_, err := service.CleanBundle(context.Background(), connect.NewRequest(&pipelinev1.BundleCleanRequest{
		ScenarioName: "example",
		LocationMode: stringPtr("staging"),
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("error code = %v, want invalid argument (err=%v)", connect.CodeOf(err), err)
	}
}
