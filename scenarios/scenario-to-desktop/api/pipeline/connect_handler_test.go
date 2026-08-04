package pipeline

import (
	"context"
	"errors"
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

func TestConnectServiceGetAndStartActiveCoverManagerLifecycle(t *testing.T) {
	t.Run("empty active slot", func(t *testing.T) {
		index, err := NewScenarioIndexStore(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		service := NewConnectService(NewHandler(WithManager(NewManager(
			WithManagerOrchestrator(&mockOrchestrator{}),
			WithManagerIndexStore(index),
		))))
		response, err := service.GetActive(context.Background(), connect.NewRequest(&pipelinev1.GetActivePipelineRequest{ScenarioName: "demo"}))
		if err != nil || response.Msg.GetPipeline() != nil || response.Msg.GetCreated() {
			t.Fatalf("empty GetActive = %#v, %v", response.Msg, err)
		}
	})

	t.Run("auto creates and starts idle pipeline", func(t *testing.T) {
		index, err := NewScenarioIndexStore(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		orchestrator := &mockOrchestrator{
			createResult: &Status{PipelineID: "idle-1", Status: StatusIdle},
			startResult:  &Status{PipelineID: "idle-1", ScenarioName: "demo", Status: StatusRunning},
		}
		manager := NewManager(WithManagerOrchestrator(orchestrator), WithManagerIndexStore(index))
		service := NewConnectService(NewHandler(WithManager(manager)))
		active, err := service.GetActive(context.Background(), connect.NewRequest(&pipelinev1.GetActivePipelineRequest{ScenarioName: "demo", AutoCreate: true}))
		if err != nil || !active.Msg.GetCreated() || active.Msg.GetPipeline().GetPipelineId() != "idle-1" {
			t.Fatalf("auto GetActive = %#v, %v", active.Msg, err)
		}
		started, err := service.StartActive(context.Background(), connect.NewRequest(&pipelinev1.StartActivePipelineRequest{
			ScenarioName:    "demo",
			ConfigOverrides: &pipelinev1.PipelineConfig{Platforms: []sharedv1.Platform{sharedv1.Platform_PLATFORM_LINUX}},
		}))
		if err != nil || started.Msg.GetPipeline().GetStatus() != sharedv1.StageStatus_STAGE_STATUS_RUNNING || started.Msg.GetMessage() != "Active pipeline started" {
			t.Fatalf("StartActive = %#v, %v", started.Msg, err)
		}
		if orchestrator.updatedConfig == nil || len(orchestrator.updatedConfig.Platforms) != 1 {
			t.Fatalf("StartActive did not pass configuration overrides: %#v", orchestrator.updatedConfig)
		}
	})
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

func TestConnectServiceCancelReportsRequestedCompletedAndUnavailableStates(t *testing.T) {
	t.Run("requested", func(t *testing.T) {
		service := NewConnectService(NewHandler(WithOrchestrator(&mockOrchestrator{cancelSuccess: true})))
		response, err := service.Cancel(context.Background(), connect.NewRequest(&pipelinev1.PipelineCancelRequest{PipelineId: "running"}))
		if err != nil || response.Msg.GetStatus() != "cancelling" {
			t.Fatalf("cancel response = %#v, %v", response, err)
		}
	})
	t.Run("already complete", func(t *testing.T) {
		service := NewConnectService(NewHandler(WithOrchestrator(&mockOrchestrator{
			getResult: &Status{PipelineID: "done", Status: StatusCompleted}, getFound: true,
		})))
		response, err := service.Cancel(context.Background(), connect.NewRequest(&pipelinev1.PipelineCancelRequest{PipelineId: "done"}))
		if err != nil || response.Msg.GetStatus() != StatusCompleted || response.Msg.GetMessage() != "Pipeline has already completed" {
			t.Fatalf("completed cancel response = %#v, %v", response, err)
		}
	})
	t.Run("not found and not cancellable", func(t *testing.T) {
		notFound := NewConnectService(NewHandler(WithOrchestrator(&mockOrchestrator{})))
		_, err := notFound.Cancel(context.Background(), connect.NewRequest(&pipelinev1.PipelineCancelRequest{PipelineId: "missing"}))
		if connect.CodeOf(err) != connect.CodeNotFound {
			t.Fatalf("not found code = %v", connect.CodeOf(err))
		}
		active := NewConnectService(NewHandler(WithOrchestrator(&mockOrchestrator{getResult: &Status{Status: StatusRunning}, getFound: true})))
		_, err = active.Cancel(context.Background(), connect.NewRequest(&pipelinev1.PipelineCancelRequest{PipelineId: "active"}))
		if connect.CodeOf(err) != connect.CodeFailedPrecondition {
			t.Fatalf("active code = %v", connect.CodeOf(err))
		}
	})
}

func TestConnectServiceRunAndResumeMapDomainErrorsToContractCodes(t *testing.T) {
	config := &pipelinev1.PipelineConfig{ScenarioName: "example", Platforms: []sharedv1.Platform{sharedv1.Platform_PLATFORM_LINUX}}
	for _, test := range []struct {
		name string
		err  error
		code connect.Code
	}{
		{"invalid", errors.New("invalid configuration"), connect.CodeInvalidArgument},
		{"missing", errors.New("pipeline not found"), connect.CodeNotFound},
		{"completed", errors.New("already completed"), connect.CodeFailedPrecondition},
		{"internal", errors.New("backend unavailable"), connect.CodeInternal},
	} {
		t.Run(test.name, func(t *testing.T) {
			service := NewConnectService(NewHandler(WithOrchestrator(&mockOrchestrator{runError: test.err, resumeError: test.err})))
			_, err := service.Run(context.Background(), connect.NewRequest(&pipelinev1.PipelineRunRequest{Config: config}))
			if connect.CodeOf(err) != test.code {
				t.Fatalf("Run code = %v, want %v", connect.CodeOf(err), test.code)
			}
			_, err = service.Resume(context.Background(), connect.NewRequest(&pipelinev1.PipelineResumeRequest{PipelineId: "prior", Config: config}))
			if connect.CodeOf(err) != test.code {
				t.Fatalf("Resume code = %v, want %v", connect.CodeOf(err), test.code)
			}
		})
	}
}

func TestPipelineConfigProtoRoundTripPreservesExplicitControls(t *testing.T) {
	stop := true
	config := &Config{
		ScenarioName: "example", Platforms: []string{"linux", "mac"}, Framework: FrameworkElectron,
		DeploymentMode: DeploymentModeProxy, TemplateType: "advanced", SkipPreflight: true, SkipSmokeTest: true,
		StopOnFailure: &stop, WebhookURL: "https://hooks.example.test", ProxyURL: "http://proxy.example.test",
		BundleManifestPath: "bundle.json", ResourceArtifactRoot: "resources", LocationMode: "staging", Clean: true,
		Sign: true, Publish: true, Version: "1.2.3", PreflightTimeoutSeconds: 45, StopAfterStage: StageBuild,
		ResumeFromStage: StagePreflight, ParentPipelineID: "parent", IdempotencyKey: "key", Stages: []string{StageBundle, StageBuild},
		UpdateConfig: &generation.UpdateConfig{Provider: "generic", Channel: "stable", AutoCheck: true, Generic: &generation.GenericUpdateConfig{URL: "http://127.0.0.1:8765/updates"}},
	}
	value, err := configFromProto(configToProto(config))
	if err != nil {
		t.Fatalf("config round trip: %v", err)
	}
	if value.DeploymentMode != DeploymentModeProxy || !value.SkipPreflight || !value.Clean || value.StopAfterStage != StageBuild || value.IdempotencyKey != "key" || len(value.Platforms) != 2 || value.UpdateConfig == nil || value.UpdateConfig.Generic == nil || value.UpdateConfig.Generic.URL != "http://127.0.0.1:8765/updates" {
		t.Fatalf("round-tripped config = %#v", value)
	}
}

func TestConfigFromProtoRejectsInsecureProductionUpdateFeed(t *testing.T) {
	provider, feedURL, trustMode := "generic", "http://127.0.0.1:8765/updates", "production"
	_, err := configFromProto(&pipelinev1.PipelineConfig{
		ScenarioName: "example", ArtifactTrustMode: &trustMode,
		UpdateConfig: &sharedv1.UpdateConfig{Provider: &provider, Generic: &sharedv1.GenericUpdateConfig{Url: feedURL}},
	})
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("configFromProto() code = %v, want invalid argument (err=%v)", connect.CodeOf(err), err)
	}
}

func TestConnectServiceActivePipelineLifecycleContracts(t *testing.T) {
	index, err := NewScenarioIndexStore(t.TempDir())
	if err != nil {
		t.Fatalf("new index: %v", err)
	}
	if err := index.SetActivePipeline("example", "old"); err != nil {
		t.Fatalf("seed active index: %v", err)
	}
	orchestrator := &mockOrchestrator{
		createResult: &Status{PipelineID: "new", ScenarioName: "example", Status: StatusIdle},
		getResult:    &Status{PipelineID: "old", ScenarioName: "example", Status: StatusCompleted},
		getFound:     true,
	}
	service := NewConnectService(NewHandler(WithManager(NewManager(
		WithManagerOrchestrator(orchestrator), WithManagerIndexStore(index),
	))))

	created, err := service.CreateActive(context.Background(), connect.NewRequest(&pipelinev1.CreatePipelineRequest{
		ScenarioName: "example", Config: &pipelinev1.PipelineConfig{TemplateType: sharedv1.TemplateType_TEMPLATE_TYPE_ADVANCED},
	}))
	if err != nil {
		t.Fatalf("CreateActive: %v", err)
	}
	if created.Msg.GetPipeline().GetPipelineId() != "new" || created.Msg.GetArchivedPipelineId() != "old" {
		t.Fatalf("create response = %#v", created.Msg)
	}

	limit := int32(10)
	history, err := service.GetHistory(context.Background(), connect.NewRequest(&pipelinev1.PipelineHistoryRequest{ScenarioName: "example", Limit: &limit}))
	if err != nil || history.Msg.GetTotal() != 1 || len(history.Msg.GetPipelines()) != 1 {
		t.Fatalf("history response = %#v, %v", history, err)
	}

	reset, err := service.ResetActive(context.Background(), connect.NewRequest(&pipelinev1.ScenarioPipelineRequest{ScenarioName: "example"}))
	if err != nil || !reset.Msg.GetCleared() || reset.Msg.GetArchivedPipelineId() != "new" {
		t.Fatalf("reset response = %#v, %v", reset, err)
	}
	if status, ok := NewManager(WithManagerOrchestrator(orchestrator), WithManagerIndexStore(index)).GetActivePipelineStatus("example"); ok || status != nil {
		t.Fatalf("active pipeline remains after reset: %#v", status)
	}
}
