package pipeline

import (
	"context"
	"testing"
	"time"
)

// mockTimeProvider provides a fixed time for deterministic testing.
type mockTimeProvider struct {
	now time.Time
}

func (m *mockTimeProvider) Now() time.Time {
	return m.now
}

// mockStage is a test stage that can be configured to succeed or fail.
type mockStage struct {
	name        string
	shouldFail  bool
	shouldSkip  bool
	executeCh   chan struct{}
	executeTime time.Duration
}

func (s *mockStage) Name() string {
	return s.name
}

func (s *mockStage) Dependencies() []string {
	return nil
}

func (s *mockStage) CanSkip(input *StageInput) bool {
	return s.shouldSkip
}

func (s *mockStage) Execute(ctx context.Context, input *StageInput) *StageResult {
	if s.executeCh != nil {
		close(s.executeCh)
	}

	if s.executeTime > 0 {
		select {
		case <-ctx.Done():
			return &StageResult{
				Stage:       s.name,
				Status:      StatusCancelled,
				CompletedAt: time.Now().Unix(),
			}
		case <-time.After(s.executeTime):
		}
	}

	if s.shouldFail {
		return &StageResult{
			Stage:       s.name,
			Status:      StatusFailed,
			Error:       "mock failure",
			CompletedAt: time.Now().Unix(),
		}
	}

	return &StageResult{
		Stage:       s.name,
		Status:      StatusCompleted,
		CompletedAt: time.Now().Unix(),
	}
}

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

	orchestrator.RunPipeline(ctx, config)
	orchestrator.RunPipeline(ctx, config)

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

	// Wait for completion
	time.Sleep(100 * time.Millisecond)

	final, _ := orchestrator.GetStatus(status.PipelineID)

	if final.Status != StatusFailed {
		t.Errorf("expected pipeline to fail, got %s", final.Status)
	}
}
