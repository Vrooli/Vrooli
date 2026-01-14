package autosteer

import (
	"errors"
	"testing"
)

// TestStartExecution_Success tests successful execution start with all mocks.
func TestStartExecution_Success(t *testing.T) {
	// Create mocks
	stateRepo := NewMockExecutionStateRepository()
	profileRepo := NewMockProfileRepository()
	metricsProvider := NewMockMetricsProvider()
	phaseCoord := NewMockPhaseCoordinatorAPI()
	iterEval := NewMockIterationEvaluatorAPI()
	promptEnhancer := NewMockPromptEnhancerAPI()

	// Setup profile
	profile := &AutoSteerProfile{
		ID:   "test-profile",
		Name: "Test Profile",
		Phases: []SteerPhase{
			{ID: "phase-1", Mode: ModeProgress, MaxIterations: 5},
		},
	}
	if err := profileRepo.CreateProfile(profile); err != nil {
		t.Fatalf("failed to create profile: %v", err)
	}

	// Create orchestrator with all mocks
	orch := NewExecutionOrchestrator(
		stateRepo,
		phaseCoord,
		iterEval,
		profileRepo,
		metricsProvider,
		promptEnhancer,
	)

	// Execute
	state, err := orch.StartExecution("task-1", "test-profile", "test-scenario")

	// Verify
	if err != nil {
		t.Fatalf("StartExecution failed: %v", err)
	}
	if state == nil {
		t.Fatal("expected non-nil state")
	}
	if state.TaskID != "task-1" {
		t.Errorf("expected TaskID 'task-1', got '%s'", state.TaskID)
	}
	if state.ProfileID != "test-profile" {
		t.Errorf("expected ProfileID 'test-profile', got '%s'", state.ProfileID)
	}
	if state.CurrentPhaseIndex != 0 {
		t.Errorf("expected CurrentPhaseIndex 0, got %d", state.CurrentPhaseIndex)
	}

	// Verify metrics provider was called
	if metricsProvider.CallCount != 1 {
		t.Errorf("expected metrics provider to be called once, got %d", metricsProvider.CallCount)
	}
}

// TestStartExecution_ProfileNotFound tests error when profile doesn't exist.
func TestStartExecution_ProfileNotFound(t *testing.T) {
	stateRepo := NewMockExecutionStateRepository()
	profileRepo := NewMockProfileRepository()
	metricsProvider := NewMockMetricsProvider()
	phaseCoord := NewMockPhaseCoordinatorAPI()
	iterEval := NewMockIterationEvaluatorAPI()
	promptEnhancer := NewMockPromptEnhancerAPI()

	orch := NewExecutionOrchestrator(
		stateRepo,
		phaseCoord,
		iterEval,
		profileRepo,
		metricsProvider,
		promptEnhancer,
	)

	// Execute with non-existent profile
	_, err := orch.StartExecution("task-1", "non-existent", "test-scenario")

	if err == nil {
		t.Fatal("expected error for non-existent profile")
	}
}

// TestStartExecution_MetricsCollectionError tests error handling when metrics collection fails.
func TestStartExecution_MetricsCollectionError(t *testing.T) {
	stateRepo := NewMockExecutionStateRepository()
	profileRepo := NewMockProfileRepository()
	metricsProvider := NewMockMetricsProvider()
	phaseCoord := NewMockPhaseCoordinatorAPI()
	iterEval := NewMockIterationEvaluatorAPI()
	promptEnhancer := NewMockPromptEnhancerAPI()

	// Setup profile
	profile := &AutoSteerProfile{
		ID:   "test-profile",
		Name: "Test Profile",
		Phases: []SteerPhase{
			{ID: "phase-1", Mode: ModeProgress, MaxIterations: 5},
		},
	}
	_ = profileRepo.CreateProfile(profile)

	// Configure metrics error
	metricsProvider.Error = errors.New("metrics collection failed")

	orch := NewExecutionOrchestrator(
		stateRepo,
		phaseCoord,
		iterEval,
		profileRepo,
		metricsProvider,
		promptEnhancer,
	)

	_, err := orch.StartExecution("task-1", "test-profile", "test-scenario")

	if err == nil {
		t.Fatal("expected error when metrics collection fails")
	}
}

// TestGetExecutionState_Success tests retrieving execution state.
func TestGetExecutionState_Success(t *testing.T) {
	stateRepo := NewMockExecutionStateRepository()
	profileRepo := NewMockProfileRepository()
	metricsProvider := NewMockMetricsProvider()
	phaseCoord := NewMockPhaseCoordinatorAPI()
	iterEval := NewMockIterationEvaluatorAPI()
	promptEnhancer := NewMockPromptEnhancerAPI()

	// Pre-populate state
	state := stateRepo.InitializeState("task-1", "profile-1", MetricsSnapshot{})
	_ = stateRepo.Save(state)

	orch := NewExecutionOrchestrator(
		stateRepo,
		phaseCoord,
		iterEval,
		profileRepo,
		metricsProvider,
		promptEnhancer,
	)

	result, err := orch.GetExecutionState("task-1")

	if err != nil {
		t.Fatalf("GetExecutionState failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil state")
	}
	if result.TaskID != "task-1" {
		t.Errorf("expected TaskID 'task-1', got '%s'", result.TaskID)
	}
}

// TestEvaluateIteration_DelegatesToEvaluator tests that EvaluateIteration delegates correctly.
func TestEvaluateIteration_DelegatesToEvaluator(t *testing.T) {
	stateRepo := NewMockExecutionStateRepository()
	profileRepo := NewMockProfileRepository()
	metricsProvider := NewMockMetricsProvider()
	phaseCoord := NewMockPhaseCoordinatorAPI()
	iterEval := NewMockIterationEvaluatorAPI()
	promptEnhancer := NewMockPromptEnhancerAPI()

	// Configure evaluator to return specific result
	iterEval.EvaluateResult = &IterationEvaluation{
		ShouldStop: true,
		Reason:     "max_iterations",
	}

	orch := NewExecutionOrchestrator(
		stateRepo,
		phaseCoord,
		iterEval,
		profileRepo,
		metricsProvider,
		promptEnhancer,
	)

	result, err := orch.EvaluateIteration("task-1", "test-scenario")

	if err != nil {
		t.Fatalf("EvaluateIteration failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.ShouldStop {
		t.Error("expected ShouldStop to be true")
	}
	if result.Reason != "max_iterations" {
		t.Errorf("expected Reason 'max_iterations', got '%s'", result.Reason)
	}

	// Verify evaluator was called with correct arguments
	if iterEval.LastTaskID != "task-1" {
		t.Errorf("expected LastTaskID 'task-1', got '%s'", iterEval.LastTaskID)
	}
	if iterEval.LastScenarioName != "test-scenario" {
		t.Errorf("expected LastScenarioName 'test-scenario', got '%s'", iterEval.LastScenarioName)
	}
}

// TestAdvancePhase_Success tests successful phase advancement.
func TestAdvancePhase_Success(t *testing.T) {
	stateRepo := NewMockExecutionStateRepository()
	profileRepo := NewMockProfileRepository()
	metricsProvider := NewMockMetricsProvider()
	phaseCoord := NewMockPhaseCoordinatorAPI()
	iterEval := NewMockIterationEvaluatorAPI()
	promptEnhancer := NewMockPromptEnhancerAPI()

	// Setup profile with 2 phases
	profile := &AutoSteerProfile{
		ID:   "test-profile",
		Name: "Test Profile",
		Phases: []SteerPhase{
			{ID: "phase-1", Mode: ModeProgress, MaxIterations: 5},
			{ID: "phase-2", Mode: ModeRefactor, MaxIterations: 3},
		},
	}
	_ = profileRepo.CreateProfile(profile)

	// Pre-populate state at phase 0
	state := stateRepo.InitializeState("task-1", "test-profile", MetricsSnapshot{})
	state.CurrentPhaseIteration = 3
	_ = stateRepo.Save(state)

	orch := NewExecutionOrchestrator(
		stateRepo,
		phaseCoord,
		iterEval,
		profileRepo,
		metricsProvider,
		promptEnhancer,
	)

	result, err := orch.AdvancePhase("task-1", "test-scenario")

	if err != nil {
		t.Fatalf("AdvancePhase failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.Success {
		t.Error("expected Success to be true")
	}
	if result.Completed {
		t.Error("expected Completed to be false (more phases remain)")
	}
	if result.NextPhaseIndex != 1 {
		t.Errorf("expected NextPhaseIndex 1, got %d", result.NextPhaseIndex)
	}
}

// TestAdvancePhase_CompleteExecution tests completion when all phases done.
func TestAdvancePhase_CompleteExecution(t *testing.T) {
	stateRepo := NewMockExecutionStateRepository()
	profileRepo := NewMockProfileRepository()
	metricsProvider := NewMockMetricsProvider()
	phaseCoord := NewMockPhaseCoordinatorAPI()
	iterEval := NewMockIterationEvaluatorAPI()
	promptEnhancer := NewMockPromptEnhancerAPI()

	// Setup profile with 1 phase
	profile := &AutoSteerProfile{
		ID:   "test-profile",
		Name: "Test Profile",
		Phases: []SteerPhase{
			{ID: "phase-1", Mode: ModeProgress, MaxIterations: 5},
		},
	}
	_ = profileRepo.CreateProfile(profile)

	// Pre-populate state at last phase
	state := stateRepo.InitializeState("task-1", "test-profile", MetricsSnapshot{})
	_ = stateRepo.Save(state)

	orch := NewExecutionOrchestrator(
		stateRepo,
		phaseCoord,
		iterEval,
		profileRepo,
		metricsProvider,
		promptEnhancer,
	)

	result, err := orch.AdvancePhase("task-1", "test-scenario")

	if err != nil {
		t.Fatalf("AdvancePhase failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.Success {
		t.Error("expected Success to be true")
	}
	if !result.Completed {
		t.Error("expected Completed to be true (all phases done)")
	}
}

// TestGetEnhancedPrompt_Success tests enhanced prompt generation with mocks.
func TestGetEnhancedPrompt_Success(t *testing.T) {
	stateRepo := NewMockExecutionStateRepository()
	profileRepo := NewMockProfileRepository()
	metricsProvider := NewMockMetricsProvider()
	phaseCoord := NewMockPhaseCoordinatorAPI()
	iterEval := NewMockIterationEvaluatorAPI()
	promptEnhancer := NewMockPromptEnhancerAPI()

	// Setup profile
	profile := &AutoSteerProfile{
		ID:   "test-profile",
		Name: "Test Profile",
		Phases: []SteerPhase{
			{ID: "phase-1", Mode: ModeProgress, MaxIterations: 5},
		},
	}
	_ = profileRepo.CreateProfile(profile)

	// Pre-populate state
	state := stateRepo.InitializeState("task-1", "test-profile", MetricsSnapshot{})
	_ = stateRepo.Save(state)

	// Configure prompt enhancer mock
	promptEnhancer.AutoSteerSectionResult = "## Test Auto Steer Section"

	orch := NewExecutionOrchestrator(
		stateRepo,
		phaseCoord,
		iterEval,
		profileRepo,
		metricsProvider,
		promptEnhancer,
	)

	result, err := orch.GetEnhancedPrompt("task-1")

	if err != nil {
		t.Fatalf("GetEnhancedPrompt failed: %v", err)
	}
	if result != "## Test Auto Steer Section" {
		t.Errorf("expected mocked prompt section, got '%s'", result)
	}
	if promptEnhancer.GenerateAutoSteerSectionCallCount != 1 {
		t.Errorf("expected GenerateAutoSteerSection to be called once, got %d", promptEnhancer.GenerateAutoSteerSectionCallCount)
	}
}

// TestGenerateModeSection_DelegatesToPromptEnhancer tests mode section generation.
func TestGenerateModeSection_DelegatesToPromptEnhancer(t *testing.T) {
	stateRepo := NewMockExecutionStateRepository()
	profileRepo := NewMockProfileRepository()
	metricsProvider := NewMockMetricsProvider()
	phaseCoord := NewMockPhaseCoordinatorAPI()
	iterEval := NewMockIterationEvaluatorAPI()
	promptEnhancer := NewMockPromptEnhancerAPI()

	promptEnhancer.ModeSectionResult = "## Progress Mode\nFocus on progress."

	orch := NewExecutionOrchestrator(
		stateRepo,
		phaseCoord,
		iterEval,
		profileRepo,
		metricsProvider,
		promptEnhancer,
	)

	result := orch.GenerateModeSection(ModeProgress)

	if result != "## Progress Mode\nFocus on progress." {
		t.Errorf("expected mocked mode section, got '%s'", result)
	}
	if promptEnhancer.LastMode != ModeProgress {
		t.Errorf("expected LastMode to be ModeProgress, got '%s'", promptEnhancer.LastMode)
	}
}

// TestDeleteExecutionState_Success tests deleting execution state.
func TestDeleteExecutionState_Success(t *testing.T) {
	stateRepo := NewMockExecutionStateRepository()
	profileRepo := NewMockProfileRepository()
	metricsProvider := NewMockMetricsProvider()
	phaseCoord := NewMockPhaseCoordinatorAPI()
	iterEval := NewMockIterationEvaluatorAPI()
	promptEnhancer := NewMockPromptEnhancerAPI()

	// Pre-populate state
	state := stateRepo.InitializeState("task-1", "profile-1", MetricsSnapshot{})
	_ = stateRepo.Save(state)

	orch := NewExecutionOrchestrator(
		stateRepo,
		phaseCoord,
		iterEval,
		profileRepo,
		metricsProvider,
		promptEnhancer,
	)

	err := orch.DeleteExecutionState("task-1")

	if err != nil {
		t.Fatalf("DeleteExecutionState failed: %v", err)
	}

	// Verify state is deleted
	result, _ := stateRepo.Get("task-1")
	if result != nil {
		t.Error("expected state to be deleted")
	}
}

// TestIsAutoSteerActive_True tests detecting active Auto Steer.
func TestIsAutoSteerActive_True(t *testing.T) {
	stateRepo := NewMockExecutionStateRepository()
	profileRepo := NewMockProfileRepository()
	metricsProvider := NewMockMetricsProvider()
	phaseCoord := NewMockPhaseCoordinatorAPI()
	iterEval := NewMockIterationEvaluatorAPI()
	promptEnhancer := NewMockPromptEnhancerAPI()

	// Pre-populate state
	state := stateRepo.InitializeState("task-1", "profile-1", MetricsSnapshot{})
	_ = stateRepo.Save(state)

	orch := NewExecutionOrchestrator(
		stateRepo,
		phaseCoord,
		iterEval,
		profileRepo,
		metricsProvider,
		promptEnhancer,
	)

	active, err := orch.IsAutoSteerActive("task-1")

	if err != nil {
		t.Fatalf("IsAutoSteerActive failed: %v", err)
	}
	if !active {
		t.Error("expected IsAutoSteerActive to be true")
	}
}

// TestIsAutoSteerActive_False tests detecting inactive Auto Steer.
func TestIsAutoSteerActive_False(t *testing.T) {
	stateRepo := NewMockExecutionStateRepository()
	profileRepo := NewMockProfileRepository()
	metricsProvider := NewMockMetricsProvider()
	phaseCoord := NewMockPhaseCoordinatorAPI()
	iterEval := NewMockIterationEvaluatorAPI()
	promptEnhancer := NewMockPromptEnhancerAPI()

	orch := NewExecutionOrchestrator(
		stateRepo,
		phaseCoord,
		iterEval,
		profileRepo,
		metricsProvider,
		promptEnhancer,
	)

	active, err := orch.IsAutoSteerActive("non-existent-task")

	if err != nil {
		t.Fatalf("IsAutoSteerActive failed: %v", err)
	}
	if active {
		t.Error("expected IsAutoSteerActive to be false")
	}
}
