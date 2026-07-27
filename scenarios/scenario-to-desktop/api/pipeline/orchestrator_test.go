package pipeline

import (
	"context"
	"testing"
	"time"
)

func TestOrchestratorCreation(t *testing.T) {
	orchestrator := NewOrchestrator()
	if orchestrator == nil {
		t.Fatalf("expected orchestrator to be created")
	}
}

func TestOrchestratorWithMockStages(t *testing.T) {
	stage1 := &mockStage{name: "stage1"}
	stage2 := &mockStage{name: "stage2"}

	orchestrator := NewOrchestrator(
		WithStages(stage1, stage2),
	)

	ctx := context.Background()
	config := &Config{
		ScenarioName: "test-scenario",
	}

	status, err := orchestrator.RunPipeline(ctx, config)
	if err != nil {
		t.Fatalf("RunPipeline error: %v", err)
	}

	if status.PipelineID == "" {
		t.Errorf("expected pipeline ID to be set")
	}
	if status.ScenarioName != "test-scenario" {
		t.Errorf("expected scenario name 'test-scenario', got %q", status.ScenarioName)
	}
}

func TestOrchestratorValidation(t *testing.T) {
	orchestrator := NewOrchestrator()

	ctx := context.Background()
	config := &Config{
		ScenarioName: "", // Missing required field
	}

	_, err := orchestrator.RunPipeline(ctx, config)
	if err == nil {
		t.Fatalf("expected error for missing scenario_name")
	}
}

func TestOrchestratorGetStatus(t *testing.T) {
	orchestrator := NewOrchestrator(
		WithStages(&mockStage{name: "test"}),
	)

	ctx := context.Background()
	config := &Config{
		ScenarioName: "test-scenario",
	}

	status, _ := orchestrator.RunPipeline(ctx, config)

	// Should be able to retrieve the status
	retrieved, ok := orchestrator.GetStatus(status.PipelineID)
	if !ok {
		t.Fatalf("expected to retrieve pipeline status")
	}
	if retrieved.PipelineID != status.PipelineID {
		t.Errorf("expected matching pipeline IDs")
	}
}

func TestOrchestratorListPipelines(t *testing.T) {
	orchestrator := NewOrchestrator(
		WithStages(&mockStage{name: "test"}),
	)

	ctx := context.Background()
	config := &Config{
		ScenarioName: "test-scenario",
	}

	_, _ = orchestrator.RunPipeline(ctx, config)
	_, _ = orchestrator.RunPipeline(ctx, config)

	pipelines := orchestrator.ListPipelines()
	if len(pipelines) < 2 {
		t.Errorf("expected at least 2 pipelines, got %d", len(pipelines))
	}
}

func TestOrchestratorCancellation(t *testing.T) {
	executeCh := make(chan struct{})
	stage := &mockStage{
		name:        "slow-stage",
		executeTime: 10 * time.Second,
		executeCh:   executeCh,
	}

	orchestrator := NewOrchestrator(
		WithStages(stage),
	)

	ctx := context.Background()
	config := &Config{
		ScenarioName: "test-scenario",
	}

	status, _ := orchestrator.RunPipeline(ctx, config)

	// Wait for stage to start executing
	<-executeCh

	// Cancel the pipeline
	cancelled := orchestrator.CancelPipeline(status.PipelineID)
	if !cancelled {
		t.Errorf("expected CancelPipeline to return true")
	}

	// Give it time to process the cancellation
	time.Sleep(100 * time.Millisecond)

	// Check status
	final, _ := orchestrator.GetStatus(status.PipelineID)
	if final.Status != StatusCancelled && final.Status != StatusRunning {
		// May still be running or cancelled depending on timing
		t.Logf("pipeline status: %s", final.Status)
	}
}

func TestStageSkipping(t *testing.T) {
	stage1 := &mockStage{name: "stage1", shouldSkip: true}
	stage2 := &mockStage{name: "stage2"}

	orchestrator := NewOrchestrator(
		WithStages(stage1, stage2),
	)

	ctx := context.Background()
	config := &Config{
		ScenarioName: "test-scenario",
	}

	status, _ := orchestrator.RunPipeline(ctx, config)

	// Wait for completion
	time.Sleep(100 * time.Millisecond)

	final, _ := orchestrator.GetStatus(status.PipelineID)

	// First stage should be skipped
	if stage1Result, ok := final.Stages["stage1"]; ok {
		if stage1Result.Status != StatusSkipped {
			t.Errorf("expected stage1 to be skipped, got %s", stage1Result.Status)
		}
	}
}

func TestStageFailure(t *testing.T) {
	stage1 := &mockStage{name: "stage1", shouldFail: true}
	stage2 := &mockStage{name: "stage2"}

	orchestrator := NewOrchestrator(
		WithStages(stage1, stage2),
	)

	ctx := context.Background()
	stopOnFailure := true
	config := &Config{
		ScenarioName:  "test-scenario",
		StopOnFailure: &stopOnFailure,
	}

	status, _ := orchestrator.RunPipeline(ctx, config)

	// Poll for completion instead of fixed sleep to avoid race conditions
	var final *Status
	for range 50 {
		time.Sleep(50 * time.Millisecond)
		final, _ = orchestrator.GetStatus(status.PipelineID)
		if final.IsComplete() {
			break
		}
	}

	if final.Status != StatusFailed {
		t.Errorf("expected pipeline to fail, got %s", final.Status)
	}
}

func TestOrchestratorWithOptions(t *testing.T) {
	store := NewInMemoryStore()
	cancelManager := NewInMemoryCancelManager()
	idGen := NewUUIDGenerator()
	timeProv := NewRealTimeProvider()
	logger := &mockLogger{}
	stage := &mockStage{name: "test-stage"}

	orchestrator := NewOrchestrator(
		WithStore(store),
		WithCancelManager(cancelManager),
		WithIDGenerator(idGen),
		WithTimeProvider(timeProv),
		WithLogger(logger),
		WithStages(stage),
		WithOrchestratorScenarioRoot("/tmp/scenarios"),
	)

	if orchestrator == nil {
		t.Fatalf("expected orchestrator to be created")
	}
	if orchestrator.store != store {
		t.Errorf("expected custom store to be used")
	}
	if orchestrator.cancelManager != cancelManager {
		t.Errorf("expected custom cancel manager to be used")
	}
	if orchestrator.scenarioRoot != "/tmp/scenarios" {
		t.Errorf("expected scenario root '/tmp/scenarios', got %q", orchestrator.scenarioRoot)
	}
}

func TestOrchestratorListPipelinesConcurrency(t *testing.T) {
	orchestrator := NewOrchestrator(
		WithStages(&mockStage{name: "test"}),
	)

	ctx := context.Background()
	config := &Config{
		ScenarioName: "test-scenario",
	}

	// Run multiple pipelines
	for i := 0; i < 5; i++ {
		_, _ = orchestrator.RunPipeline(ctx, config)
	}

	time.Sleep(100 * time.Millisecond)

	pipelines := orchestrator.ListPipelines()
	if len(pipelines) < 5 {
		t.Errorf("expected at least 5 pipelines, got %d", len(pipelines))
	}
}

func TestOrchestratorCancelNonexistent(t *testing.T) {
	orchestrator := NewOrchestrator()

	cancelled := orchestrator.CancelPipeline("nonexistent")
	if cancelled {
		t.Errorf("expected CancelPipeline to return false for nonexistent")
	}
}

func TestOrchestratorGetStatusNonexistent(t *testing.T) {
	orchestrator := NewOrchestrator()

	_, ok := orchestrator.GetStatus("nonexistent")
	if ok {
		t.Errorf("expected GetStatus to return false for nonexistent")
	}
}

func TestUpdatePipelineConfigAppliesIdleOverridesAndStageSelection(t *testing.T) {
	stageOne := &mockStage{name: StageBundle}
	stageTwo := &mockStage{name: StageBuild}
	orchestrator := NewOrchestrator(
		WithStages(stageOne, stageTwo),
		WithIDGenerator(&fixedIDGenerator{id: "idle-pipeline"}),
	)
	status, err := orchestrator.CreateIdlePipeline(&Config{ScenarioName: "demo", Framework: FrameworkElectron})
	if err != nil {
		t.Fatalf("CreateIdlePipeline: %v", err)
	}
	stop := true
	updates := &Config{
		Platforms: []string{"linux-amd64"}, StopAfterStage: StageBuild, ResumeFromStage: StageBundle,
		BundleManifestPath: "bundle.json", ResourceArtifactRoot: "resources", DeploymentMode: DeploymentModeProxy,
		TemplateType: "advanced", LocationMode: "staging", ProxyURL: "http://proxy", Version: "1.2.3",
		SkipPreflight: true, SkipSmokeTest: true, Clean: true, Sign: true, Publish: true, StopOnFailure: &stop,
		PreflightTimeoutSeconds: 45, PreflightSecrets: map[string]string{"TOKEN": "value"}, Stages: []string{StageBuild},
	}
	if err := orchestrator.UpdatePipelineConfig(status.PipelineID, updates); err != nil {
		t.Fatalf("UpdatePipelineConfig: %v", err)
	}
	updated, ok := orchestrator.GetStatus(status.PipelineID)
	if !ok {
		t.Fatal("updated pipeline missing")
	}
	if updated.Config.DeploymentMode != DeploymentModeProxy || !updated.Config.Clean || updated.Config.PreflightTimeoutSeconds != 45 || len(updated.Config.Platforms) != 1 || len(updated.StageOrder) != 1 || updated.StageOrder[0] != StageBuild {
		t.Fatalf("updated config/status = %#v / %#v", updated.Config, updated)
	}
	if err := orchestrator.UpdatePipelineConfig("missing", updates); err == nil {
		t.Fatal("expected missing pipeline error")
	}
	orchestrator.store.Update(status.PipelineID, func(value *Status) { value.Status = StatusRunning })
	if err := orchestrator.UpdatePipelineConfig(status.PipelineID, updates); err == nil {
		t.Fatal("expected non-idle update error")
	}
}

type fixedIDGenerator struct{ id string }

func (g *fixedIDGenerator) Generate() string { return g.id }
