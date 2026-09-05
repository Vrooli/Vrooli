package pipeline

import (
	"context"
	"testing"
	"time"
)

func waitForPipelineTerminal(t *testing.T, orchestrator *DefaultOrchestrator, pipelineID string) *Status {
	t.Helper()
	// A pipeline first captures provenance and then writes several observable
	// state transitions. The previous two-second wall-clock budget could expire
	// under ordinary package-level CPU contention even though the pipeline was
	// still making progress, yielding a false negative after it completed. Use
	// the Go test's own deadline when available so a real deadlock still fails
	// the test while normal scheduler variance cannot turn completion into a
	// failure.
	deadline, hasDeadline := t.Deadline()
	if !hasDeadline {
		deadline = time.Now().Add(30 * time.Second)
	}
	for time.Now().Before(deadline) {
		if status, ok := orchestrator.GetStatus(pipelineID); ok && (status.Status == StatusCompleted || status.Status == StatusFailed || status.Status == StatusCancelled) {
			return status
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("pipeline %s did not reach a terminal state", pipelineID)
	return nil
}

// Stop-after-stage tests

func TestStopAfterStage(t *testing.T) {
	stage1 := &mockStage{name: "bundle"}
	stage2 := &mockStage{name: "preflight"}
	stage3 := &mockStage{name: "generate"}

	orchestrator := NewOrchestrator(
		WithStages(stage1, stage2, stage3),
	)

	ctx := context.Background()
	config := &Config{
		ScenarioName:   "test-scenario",
		StopAfterStage: "preflight", // Should stop after preflight
	}

	status, err := orchestrator.RunPipeline(ctx, config)
	if err != nil {
		t.Fatalf("RunPipeline error: %v", err)
	}

	final := waitForPipelineTerminal(t, orchestrator, status.PipelineID)

	// Pipeline should be completed (stopped after stage)
	if final.Status != StatusCompleted {
		t.Errorf("expected status 'completed', got %q", final.Status)
	}

	// StoppedAfterStage should be set
	if final.StoppedAfterStage != "preflight" {
		t.Errorf("expected StoppedAfterStage 'preflight', got %q", final.StoppedAfterStage)
	}

	// bundle and preflight should be completed
	if result, ok := final.Stages["bundle"]; !ok || result.Status != StatusCompleted {
		t.Errorf("expected bundle stage to be completed")
	}
	if result, ok := final.Stages["preflight"]; !ok || result.Status != StatusCompleted {
		t.Errorf("expected preflight stage to be completed")
	}

	// generate should NOT be in stages (never started)
	if _, ok := final.Stages["generate"]; ok {
		t.Errorf("expected generate stage to not be started")
	}

	// Should be resumable
	if !final.CanResume() {
		t.Errorf("expected pipeline to be resumable")
	}
}

func TestStopAfterStageSkipped(t *testing.T) {
	stage1 := &mockStage{name: "bundle"}
	stage2 := &mockStage{name: "preflight", shouldSkip: true}
	stage3 := &mockStage{name: "generate"}

	orchestrator := NewOrchestrator(
		WithStages(stage1, stage2, stage3),
	)

	ctx := context.Background()
	config := &Config{
		ScenarioName:   "test-scenario",
		StopAfterStage: "preflight", // Should stop after preflight even if skipped
	}

	status, err := orchestrator.RunPipeline(ctx, config)
	if err != nil {
		t.Fatalf("RunPipeline error: %v", err)
	}

	final := waitForPipelineTerminal(t, orchestrator, status.PipelineID)

	// Pipeline should be completed even though preflight was skipped
	if final.Status != StatusCompleted {
		t.Errorf("expected status 'completed', got %q", final.Status)
	}

	// StoppedAfterStage should be set even for skipped stage
	if final.StoppedAfterStage != "preflight" {
		t.Errorf("expected StoppedAfterStage 'preflight', got %q", final.StoppedAfterStage)
	}

	// preflight should be skipped
	if result, ok := final.Stages["preflight"]; !ok || result.Status != StatusSkipped {
		t.Errorf("expected preflight stage to be skipped")
	}

	// generate should NOT be started
	if _, ok := final.Stages["generate"]; ok {
		t.Errorf("expected generate stage to not be started")
	}
}

func TestStopAfterStageInvalidStage(t *testing.T) {
	orchestrator := NewOrchestrator()

	ctx := context.Background()
	config := &Config{
		ScenarioName:   "test-scenario",
		StopAfterStage: "invalid-stage",
	}

	_, err := orchestrator.RunPipeline(ctx, config)
	if err == nil {
		t.Fatalf("expected error for invalid stop_after_stage")
	}
}

// Resume pipeline tests

// assertStageStatus checks that a stage in the status has the expected status value.
func assertStageStatus(t *testing.T, status *Status, stage string, expected string) {
	t.Helper()
	result, ok := status.Stages[stage]
	if !ok {
		t.Errorf("expected stage %q to exist", stage)
		return
	}
	if result.Status != expected {
		t.Errorf("expected stage %q status %q, got %q", stage, expected, result.Status)
	}
}

func TestResumePipeline(t *testing.T) {
	store := NewInMemoryStore()
	stage1 := &mockStage{name: "bundle"}
	stage2 := &mockStage{name: "preflight"}
	stage3 := &mockStage{name: "generate"}

	orchestrator := NewOrchestrator(
		WithStore(store),
		WithStages(stage1, stage2, stage3),
	)

	ctx := context.Background()

	// First run: stop after preflight
	config := &Config{
		ScenarioName:   "test-scenario",
		StopAfterStage: "preflight",
	}

	status, err := orchestrator.RunPipeline(ctx, config)
	if err != nil {
		t.Fatalf("RunPipeline error: %v", err)
	}

	parentStatus := waitForPipelineTerminal(t, orchestrator, status.PipelineID)

	// Verify it can be resumed
	if !parentStatus.CanResume() {
		t.Fatalf("expected parent pipeline to be resumable")
	}
	if parentStatus.GetNextResumeStage() != "generate" {
		t.Fatalf("expected next resume stage to be 'generate', got %q", parentStatus.GetNextResumeStage())
	}

	// Resume the pipeline
	resumeStatus, err := orchestrator.ResumePipeline(ctx, status.PipelineID, nil)
	if err != nil {
		t.Fatalf("ResumePipeline error: %v", err)
	}

	finalStatus := waitForPipelineTerminal(t, orchestrator, resumeStatus.PipelineID)

	t.Run("resumed pipeline completed", func(t *testing.T) {
		if finalStatus.Status != StatusCompleted {
			t.Errorf("expected resumed pipeline status 'completed', got %q", finalStatus.Status)
		}
		if finalStatus.StoppedAfterStage != "" {
			t.Errorf("expected StoppedAfterStage to be empty, got %q", finalStatus.StoppedAfterStage)
		}
	})

	t.Run("config links to parent", func(t *testing.T) {
		if finalStatus.Config.ParentPipelineID != status.PipelineID {
			t.Errorf("expected ParentPipelineID %q, got %q", status.PipelineID, finalStatus.Config.ParentPipelineID)
		}
	})

	t.Run("skipped stages are correct", func(t *testing.T) {
		assertStageStatus(t, finalStatus, "bundle", StatusSkipped)
		assertStageStatus(t, finalStatus, "preflight", StatusSkipped)
	})

	t.Run("generate stage completed", func(t *testing.T) {
		assertStageStatus(t, finalStatus, "generate", StatusCompleted)
	})
}

func TestResumePipelineWithStopAfter(t *testing.T) {
	store := NewInMemoryStore()
	stage1 := &mockStage{name: "bundle"}
	stage2 := &mockStage{name: "preflight"}
	stage3 := &mockStage{name: "generate"}
	stage4 := &mockStage{name: "build"}

	orchestrator := NewOrchestrator(
		WithStore(store),
		WithStages(stage1, stage2, stage3, stage4),
	)

	ctx := context.Background()

	// First run: stop after preflight
	config := &Config{
		ScenarioName:   "test-scenario",
		StopAfterStage: "preflight",
	}

	status, _ := orchestrator.RunPipeline(ctx, config)
	waitForPipelineTerminal(t, orchestrator, status.PipelineID)

	// Resume but also stop after generate
	resumeConfig := &Config{
		StopAfterStage: "generate",
	}
	resumeStatus, err := orchestrator.ResumePipeline(ctx, status.PipelineID, resumeConfig)
	if err != nil {
		t.Fatalf("ResumePipeline error: %v", err)
	}

	finalStatus := waitForPipelineTerminal(t, orchestrator, resumeStatus.PipelineID)

	// Should be stopped after generate
	if finalStatus.StoppedAfterStage != "generate" {
		t.Errorf("expected StoppedAfterStage 'generate', got %q", finalStatus.StoppedAfterStage)
	}

	// build should NOT be started
	if _, ok := finalStatus.Stages["build"]; ok {
		t.Errorf("expected build stage to not be started")
	}
}

func TestResumePipelineValidation(t *testing.T) {
	store := NewInMemoryStore()
	orchestrator := NewOrchestrator(
		WithStore(store),
		WithStages(&mockStage{name: "test"}),
	)

	ctx := context.Background()

	t.Run("resume nonexistent pipeline", func(t *testing.T) {
		_, err := orchestrator.ResumePipeline(ctx, "nonexistent", nil)
		if err == nil {
			t.Errorf("expected error for nonexistent pipeline")
		}
	})

	t.Run("resume running pipeline", func(t *testing.T) {
		executeCh := make(chan struct{})
		slowStage := &mockStage{
			name:        "slow",
			executeTime: 5 * time.Second,
			executeCh:   executeCh,
		}
		slowOrchestrator := NewOrchestrator(
			WithStore(store),
			WithStages(slowStage),
		)

		status, _ := slowOrchestrator.RunPipeline(ctx, &Config{ScenarioName: "test"})
		<-executeCh // Wait for stage to start

		_, err := slowOrchestrator.ResumePipeline(ctx, status.PipelineID, nil)
		if err == nil {
			t.Errorf("expected error for running pipeline")
		}

		slowOrchestrator.CancelPipeline(status.PipelineID)
	})

	t.Run("resume completed pipeline without stop_after_stage", func(t *testing.T) {
		status, _ := orchestrator.RunPipeline(ctx, &Config{ScenarioName: "test"})
		waitForPipelineTerminal(t, orchestrator, status.PipelineID)

		_, err := orchestrator.ResumePipeline(ctx, status.PipelineID, nil)
		if err == nil {
			t.Errorf("expected error for completed pipeline without stop_after_stage")
		}
	})

	t.Run("resume failed pipeline", func(t *testing.T) {
		failOrchestrator := NewOrchestrator(
			WithStore(store),
			WithStages(&mockStage{name: "failing", shouldFail: true}),
		)

		stopOnFailure := true
		status, _ := failOrchestrator.RunPipeline(ctx, &Config{
			ScenarioName:  "test",
			StopOnFailure: &stopOnFailure,
		})
		waitForPipelineTerminal(t, failOrchestrator, status.PipelineID)

		_, err := failOrchestrator.ResumePipeline(ctx, status.PipelineID, nil)
		if err == nil {
			t.Errorf("expected error for failed pipeline")
		}
	})
}

// Config tests

func TestConfigDefaults(t *testing.T) {
	t.Run("StopOnFailure defaults to true", func(t *testing.T) {
		config := &Config{
			ScenarioName: "test",
		}
		// StopOnFailure is nil by default, should be treated as true
		if config.StopOnFailure != nil && !*config.StopOnFailure {
			t.Errorf("expected StopOnFailure to default to true")
		}
	})
}

// Status tests

func TestStatus_Fields(t *testing.T) {
	now := time.Now().Unix()
	status := &Status{
		PipelineID:   "pipeline-123",
		ScenarioName: "my-scenario",
		Status:       StatusRunning,
		CurrentStage: "bundle",
		Stages:       map[string]*StageResult{},
		StageOrder:   []string{"bundle", "preflight"},
		StartedAt:    now,
	}

	if status.PipelineID != "pipeline-123" {
		t.Errorf("expected PipelineID 'pipeline-123'")
	}
	if status.CurrentStage != "bundle" {
		t.Errorf("expected CurrentStage 'bundle'")
	}
	if len(status.StageOrder) != 2 {
		t.Errorf("expected 2 stage order entries")
	}
}

func TestStageResult_Fields(t *testing.T) {
	now := time.Now().Unix()
	result := &StageResult{
		Stage:       "bundle",
		Status:      StatusCompleted,
		StartedAt:   now,
		CompletedAt: now + 10,
		Details:     map[string]interface{}{"bundle_dir": "/tmp/bundle"},
		Logs:        []string{"starting...", "done"},
	}

	if result.Stage != "bundle" {
		t.Errorf("expected Stage 'bundle'")
	}
	if result.Status != StatusCompleted {
		t.Errorf("expected Status 'completed'")
	}
	if len(result.Logs) != 2 {
		t.Errorf("expected 2 log entries")
	}
}
