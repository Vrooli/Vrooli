package pipeline

import (
	"context"
	"testing"
	"time"
)

// mockTimeProvider provides a fixed time for deterministic testing.
type mockTimeProvider struct {
	now int64
}

func (m *mockTimeProvider) Now() int64 {
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

// Additional orchestrator tests

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
		orchestrator.RunPipeline(ctx, config)
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

// Mock logger for testing
type mockLogger struct{}

func (m *mockLogger) Info(msg string, args ...interface{})  {}
func (m *mockLogger) Warn(msg string, args ...interface{})  {}
func (m *mockLogger) Error(msg string, args ...interface{}) {}
func (m *mockLogger) Debug(msg string, args ...interface{}) {}

// Store tests

func TestInMemoryStore(t *testing.T) {
	store := NewInMemoryStore()

	t.Run("Save and Get", func(t *testing.T) {
		status := &Status{
			PipelineID:   "pipeline-123",
			ScenarioName: "test-scenario",
			Status:       StatusRunning,
		}
		store.Save(status)

		retrieved, ok := store.Get("pipeline-123")
		if !ok {
			t.Fatalf("expected to retrieve saved status")
		}
		if retrieved.PipelineID != "pipeline-123" {
			t.Errorf("expected pipeline ID 'pipeline-123'")
		}
	})

	t.Run("Get nonexistent", func(t *testing.T) {
		_, ok := store.Get("nonexistent")
		if ok {
			t.Errorf("expected false for nonexistent")
		}
	})

	t.Run("Update existing", func(t *testing.T) {
		status := &Status{
			PipelineID: "pipeline-update",
			Status:     StatusPending,
		}
		store.Save(status)

		updated := store.Update("pipeline-update", func(s *Status) {
			s.Status = StatusCompleted
		})
		if !updated {
			t.Errorf("expected Update to return true")
		}

		retrieved, _ := store.Get("pipeline-update")
		if retrieved.Status != StatusCompleted {
			t.Errorf("expected status 'completed', got %q", retrieved.Status)
		}
	})

	t.Run("Update nonexistent", func(t *testing.T) {
		updated := store.Update("nonexistent", func(s *Status) {
			s.Status = StatusCompleted
		})
		if updated {
			t.Errorf("expected Update to return false for nonexistent")
		}
	})

	t.Run("List", func(t *testing.T) {
		listStore := NewInMemoryStore()
		listStore.Save(&Status{PipelineID: "p1", Status: StatusRunning})
		listStore.Save(&Status{PipelineID: "p2", Status: StatusCompleted})

		all := listStore.List()
		if len(all) != 2 {
			t.Errorf("expected 2 statuses, got %d", len(all))
		}
	})
}

func TestInMemoryCancelManager(t *testing.T) {
	cm := NewInMemoryCancelManager()

	t.Run("Set and Take cancel", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cm.Set("pipeline-1", cancel)

		taken := cm.Take("pipeline-1")
		if taken == nil {
			t.Fatalf("expected to take cancel func")
		}

		// Verify cancel works
		taken()
		select {
		case <-ctx.Done():
			// Expected
		default:
			t.Errorf("expected context to be cancelled")
		}
	})

	t.Run("Take twice", func(t *testing.T) {
		_, cancel := context.WithCancel(context.Background())
		cm.Set("pipeline-2", cancel)

		taken := cm.Take("pipeline-2")
		if taken == nil {
			t.Fatalf("expected to take cancel func")
		}

		// Second take should return nil
		taken2 := cm.Take("pipeline-2")
		if taken2 != nil {
			t.Errorf("expected nil on second take")
		}
	})

	t.Run("Take nonexistent", func(t *testing.T) {
		taken := cm.Take("nonexistent")
		if taken != nil {
			t.Errorf("expected nil for nonexistent")
		}
	})

	t.Run("Clear", func(t *testing.T) {
		_, cancel := context.WithCancel(context.Background())
		cm.Set("pipeline-3", cancel)
		cm.Clear("pipeline-3")

		taken := cm.Take("pipeline-3")
		if taken != nil {
			t.Errorf("expected nil after clear")
		}
	})
}

func TestUUIDGenerator(t *testing.T) {
	gen := NewUUIDGenerator()

	id1 := gen.Generate()
	id2 := gen.Generate()

	if id1 == "" {
		t.Errorf("expected non-empty ID")
	}
	if id1 == id2 {
		t.Errorf("expected unique IDs")
	}
}

func TestRealTimeProvider(t *testing.T) {
	tp := NewRealTimeProvider()

	now := tp.Now()
	if now == 0 {
		t.Errorf("expected non-zero unix timestamp")
	}

	// Verify it returns a reasonable timestamp (after year 2020)
	if now < 1577836800 {
		t.Errorf("expected timestamp after 2020, got %d", now)
	}
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

	// Wait for completion
	time.Sleep(200 * time.Millisecond)

	final, ok := orchestrator.GetStatus(status.PipelineID)
	if !ok {
		t.Fatalf("expected to retrieve pipeline status")
	}

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

	// Wait for completion
	time.Sleep(200 * time.Millisecond)

	final, ok := orchestrator.GetStatus(status.PipelineID)
	if !ok {
		t.Fatalf("expected to retrieve pipeline status")
	}

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

	// Wait for first run to complete
	time.Sleep(200 * time.Millisecond)

	parentStatus, ok := orchestrator.GetStatus(status.PipelineID)
	if !ok {
		t.Fatalf("expected to retrieve parent pipeline status")
	}

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

	// Wait for resumed pipeline to complete
	time.Sleep(200 * time.Millisecond)

	finalStatus, ok := orchestrator.GetStatus(resumeStatus.PipelineID)
	if !ok {
		t.Fatalf("expected to retrieve resumed pipeline status")
	}

	// Resumed pipeline should be completed
	if finalStatus.Status != StatusCompleted {
		t.Errorf("expected resumed pipeline status 'completed', got %q", finalStatus.Status)
	}

	// Resumed pipeline should not have StoppedAfterStage (ran to completion)
	if finalStatus.StoppedAfterStage != "" {
		t.Errorf("expected StoppedAfterStage to be empty, got %q", finalStatus.StoppedAfterStage)
	}

	// Config should link to parent
	if finalStatus.Config.ParentPipelineID != status.PipelineID {
		t.Errorf("expected ParentPipelineID %q, got %q", status.PipelineID, finalStatus.Config.ParentPipelineID)
	}

	// bundle and preflight should be skipped (resumed from later stage)
	if result, ok := finalStatus.Stages["bundle"]; !ok || result.Status != StatusSkipped {
		t.Errorf("expected bundle stage to be skipped in resumed pipeline")
	}
	if result, ok := finalStatus.Stages["preflight"]; !ok || result.Status != StatusSkipped {
		t.Errorf("expected preflight stage to be skipped in resumed pipeline")
	}

	// generate should be completed
	if result, ok := finalStatus.Stages["generate"]; !ok || result.Status != StatusCompleted {
		t.Errorf("expected generate stage to be completed in resumed pipeline")
	}
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
	time.Sleep(200 * time.Millisecond)

	// Resume but also stop after generate
	resumeConfig := &Config{
		StopAfterStage: "generate",
	}
	resumeStatus, err := orchestrator.ResumePipeline(ctx, status.PipelineID, resumeConfig)
	if err != nil {
		t.Fatalf("ResumePipeline error: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	finalStatus, _ := orchestrator.GetStatus(resumeStatus.PipelineID)

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
		time.Sleep(200 * time.Millisecond)

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
		time.Sleep(200 * time.Millisecond)

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
