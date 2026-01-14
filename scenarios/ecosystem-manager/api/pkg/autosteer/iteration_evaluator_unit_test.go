package autosteer

import (
	"errors"
	"testing"
)

// TestIterationEvaluator_Evaluate_NoState tests evaluation when no state exists.
func TestIterationEvaluator_Evaluate_NoState(t *testing.T) {
	stateRepo := NewMockExecutionStateRepository()
	profileRepo := NewMockProfileRepository()
	metricsProvider := NewMockMetricsProvider()
	phaseCoord := NewMockPhaseCoordinatorAPI()

	eval := NewIterationEvaluator(stateRepo, phaseCoord, metricsProvider, profileRepo)

	result, err := eval.Evaluate("non-existent-task", "test-scenario")

	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	// When no state exists, should return non-stop result
	if result.ShouldStop {
		t.Error("expected ShouldStop to be false when no state exists")
	}
}

// TestIterationEvaluator_Evaluate_Success tests successful iteration evaluation.
func TestIterationEvaluator_Evaluate_Success(t *testing.T) {
	stateRepo := NewMockExecutionStateRepository()
	profileRepo := NewMockProfileRepository()
	metricsProvider := NewMockMetricsProvider()
	phaseCoord := NewMockPhaseCoordinatorAPI()

	// Setup profile
	profile := &AutoSteerProfile{
		ID:   "test-profile",
		Name: "Test Profile",
		Phases: []SteerPhase{
			{ID: "phase-1", Mode: ModeProgress, MaxIterations: 5},
		},
	}
	_ = profileRepo.CreateProfile(profile)

	// Setup state at iteration 2
	state := stateRepo.InitializeState("task-1", "test-profile", MetricsSnapshot{})
	state.CurrentPhaseIteration = 2
	_ = stateRepo.Save(state)

	// Configure phase coordinator to return continue
	phaseCoord.ShouldAdvancePhaseResult = PhaseAdvanceDecision{
		ShouldStop: false,
		Reason:     "continue",
	}

	eval := NewIterationEvaluator(stateRepo, phaseCoord, metricsProvider, profileRepo)

	result, err := eval.Evaluate("task-1", "test-scenario")

	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.ShouldStop {
		t.Error("expected ShouldStop to be false")
	}

	// Verify metrics were collected
	if metricsProvider.CallCount != 1 {
		t.Errorf("expected metrics provider to be called once, got %d", metricsProvider.CallCount)
	}

	// Verify phase coordinator was called
	if phaseCoord.ShouldAdvancePhaseCallCount != 1 {
		t.Errorf("expected ShouldAdvancePhase to be called once, got %d", phaseCoord.ShouldAdvancePhaseCallCount)
	}

	// Verify state was updated
	updatedState, _ := stateRepo.Get("task-1")
	if updatedState.CurrentPhaseIteration != 3 {
		t.Errorf("expected CurrentPhaseIteration to be 3, got %d", updatedState.CurrentPhaseIteration)
	}
}

// TestIterationEvaluator_Evaluate_ShouldStop tests when phase should stop.
func TestIterationEvaluator_Evaluate_ShouldStop(t *testing.T) {
	stateRepo := NewMockExecutionStateRepository()
	profileRepo := NewMockProfileRepository()
	metricsProvider := NewMockMetricsProvider()
	phaseCoord := NewMockPhaseCoordinatorAPI()

	// Setup profile
	profile := &AutoSteerProfile{
		ID:   "test-profile",
		Name: "Test Profile",
		Phases: []SteerPhase{
			{ID: "phase-1", Mode: ModeProgress, MaxIterations: 5},
		},
	}
	_ = profileRepo.CreateProfile(profile)

	// Setup state at iteration 4
	state := stateRepo.InitializeState("task-1", "test-profile", MetricsSnapshot{})
	state.CurrentPhaseIteration = 4
	_ = stateRepo.Save(state)

	// Configure phase coordinator to return stop
	phaseCoord.ShouldAdvancePhaseResult = PhaseAdvanceDecision{
		ShouldStop: true,
		Reason:     "max_iterations",
	}

	eval := NewIterationEvaluator(stateRepo, phaseCoord, metricsProvider, profileRepo)

	result, err := eval.Evaluate("task-1", "test-scenario")

	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
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
}

// TestIterationEvaluator_Evaluate_AllPhasesCompleted tests when all phases are done.
func TestIterationEvaluator_Evaluate_AllPhasesCompleted(t *testing.T) {
	stateRepo := NewMockExecutionStateRepository()
	profileRepo := NewMockProfileRepository()
	metricsProvider := NewMockMetricsProvider()
	phaseCoord := NewMockPhaseCoordinatorAPI()

	// Setup profile with 1 phase
	profile := &AutoSteerProfile{
		ID:   "test-profile",
		Name: "Test Profile",
		Phases: []SteerPhase{
			{ID: "phase-1", Mode: ModeProgress, MaxIterations: 5},
		},
	}
	_ = profileRepo.CreateProfile(profile)

	// Setup state past all phases
	state := stateRepo.InitializeState("task-1", "test-profile", MetricsSnapshot{})
	state.CurrentPhaseIndex = 1 // Past the only phase (index 0)
	_ = stateRepo.Save(state)

	eval := NewIterationEvaluator(stateRepo, phaseCoord, metricsProvider, profileRepo)

	result, err := eval.Evaluate("task-1", "test-scenario")

	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.ShouldStop {
		t.Error("expected ShouldStop to be true when all phases completed")
	}
	if result.Reason != "all_phases_completed" {
		t.Errorf("expected Reason 'all_phases_completed', got '%s'", result.Reason)
	}
}

// TestIterationEvaluator_Evaluate_MetricsError tests error handling when metrics collection fails.
func TestIterationEvaluator_Evaluate_MetricsError(t *testing.T) {
	stateRepo := NewMockExecutionStateRepository()
	profileRepo := NewMockProfileRepository()
	metricsProvider := NewMockMetricsProvider()
	phaseCoord := NewMockPhaseCoordinatorAPI()

	// Setup profile
	profile := &AutoSteerProfile{
		ID:   "test-profile",
		Name: "Test Profile",
		Phases: []SteerPhase{
			{ID: "phase-1", Mode: ModeProgress, MaxIterations: 5},
		},
	}
	_ = profileRepo.CreateProfile(profile)

	// Setup state
	state := stateRepo.InitializeState("task-1", "test-profile", MetricsSnapshot{})
	_ = stateRepo.Save(state)

	// Configure metrics error
	metricsProvider.Error = errors.New("metrics collection failed")

	eval := NewIterationEvaluator(stateRepo, phaseCoord, metricsProvider, profileRepo)

	_, err := eval.Evaluate("task-1", "test-scenario")

	if err == nil {
		t.Fatal("expected error when metrics collection fails")
	}
}

// TestIterationEvaluator_Evaluate_ProfileNotFound tests error when profile doesn't exist.
func TestIterationEvaluator_Evaluate_ProfileNotFound(t *testing.T) {
	stateRepo := NewMockExecutionStateRepository()
	profileRepo := NewMockProfileRepository()
	metricsProvider := NewMockMetricsProvider()
	phaseCoord := NewMockPhaseCoordinatorAPI()

	// Setup state with non-existent profile
	state := stateRepo.InitializeState("task-1", "non-existent-profile", MetricsSnapshot{})
	_ = stateRepo.Save(state)

	eval := NewIterationEvaluator(stateRepo, phaseCoord, metricsProvider, profileRepo)

	_, err := eval.Evaluate("task-1", "test-scenario")

	if err == nil {
		t.Fatal("expected error when profile doesn't exist")
	}
}

// TestIterationEvaluator_EvaluateWithoutMetrics_Success tests evaluation without metrics collection.
func TestIterationEvaluator_EvaluateWithoutMetrics_Success(t *testing.T) {
	stateRepo := NewMockExecutionStateRepository()
	profileRepo := NewMockProfileRepository()
	metricsProvider := NewMockMetricsProvider()
	phaseCoord := NewMockPhaseCoordinatorAPI()

	// Setup profile
	profile := &AutoSteerProfile{
		ID:   "test-profile",
		Name: "Test Profile",
		Phases: []SteerPhase{
			{ID: "phase-1", Mode: ModeProgress, MaxIterations: 5},
		},
	}
	_ = profileRepo.CreateProfile(profile)

	// Setup state with existing metrics
	state := stateRepo.InitializeState("task-1", "test-profile", MetricsSnapshot{
		OperationalTargetsPercentage: 75.0,
	})
	state.CurrentPhaseIteration = 3
	_ = stateRepo.Save(state)

	// Configure phase coordinator
	phaseCoord.ShouldAdvancePhaseResult = PhaseAdvanceDecision{
		ShouldStop: false,
		Reason:     "continue",
	}

	eval := NewIterationEvaluator(stateRepo, phaseCoord, metricsProvider, profileRepo)

	result, err := eval.EvaluateWithoutMetricsCollection("task-1")

	if err != nil {
		t.Fatalf("EvaluateWithoutMetricsCollection failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.ShouldStop {
		t.Error("expected ShouldStop to be false")
	}

	// Verify metrics were NOT collected
	if metricsProvider.CallCount != 0 {
		t.Errorf("expected metrics provider to NOT be called, got %d calls", metricsProvider.CallCount)
	}

	// Verify phase coordinator was called
	if phaseCoord.ShouldAdvancePhaseCallCount != 1 {
		t.Errorf("expected ShouldAdvancePhase to be called once, got %d", phaseCoord.ShouldAdvancePhaseCallCount)
	}
}

// TestIterationEvaluator_EvaluateWithoutMetrics_NoState tests evaluation without state.
func TestIterationEvaluator_EvaluateWithoutMetrics_NoState(t *testing.T) {
	stateRepo := NewMockExecutionStateRepository()
	profileRepo := NewMockProfileRepository()
	metricsProvider := NewMockMetricsProvider()
	phaseCoord := NewMockPhaseCoordinatorAPI()

	eval := NewIterationEvaluator(stateRepo, phaseCoord, metricsProvider, profileRepo)

	result, err := eval.EvaluateWithoutMetricsCollection("non-existent-task")

	if err != nil {
		t.Fatalf("EvaluateWithoutMetricsCollection failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.ShouldStop {
		t.Error("expected ShouldStop to be false when no state exists")
	}
}

// TestIterationEvaluator_StateUpdatedCorrectly tests that state is properly incremented.
func TestIterationEvaluator_StateUpdatedCorrectly(t *testing.T) {
	stateRepo := NewMockExecutionStateRepository()
	profileRepo := NewMockProfileRepository()
	metricsProvider := NewMockMetricsProvider()
	phaseCoord := NewMockPhaseCoordinatorAPI()

	// Setup profile
	profile := &AutoSteerProfile{
		ID:   "test-profile",
		Name: "Test Profile",
		Phases: []SteerPhase{
			{ID: "phase-1", Mode: ModeProgress, MaxIterations: 10},
		},
	}
	_ = profileRepo.CreateProfile(profile)

	// Setup state
	state := stateRepo.InitializeState("task-1", "test-profile", MetricsSnapshot{})
	state.AutoSteerIteration = 5
	state.CurrentPhaseIteration = 2
	_ = stateRepo.Save(state)

	// Configure metrics to return updated values
	metricsProvider.Metrics = &MetricsSnapshot{
		OperationalTargetsPercentage: 80.0,
		TotalLoops:                   6,
		PhaseLoops:                   3,
	}

	eval := NewIterationEvaluator(stateRepo, phaseCoord, metricsProvider, profileRepo)

	_, err := eval.Evaluate("task-1", "test-scenario")
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	// Verify state was updated
	updatedState, _ := stateRepo.Get("task-1")
	if updatedState.AutoSteerIteration != 6 {
		t.Errorf("expected AutoSteerIteration to be 6, got %d", updatedState.AutoSteerIteration)
	}
	if updatedState.CurrentPhaseIteration != 3 {
		t.Errorf("expected CurrentPhaseIteration to be 3, got %d", updatedState.CurrentPhaseIteration)
	}
	if updatedState.Metrics.OperationalTargetsPercentage != 80.0 {
		t.Errorf("expected Metrics.OperationalTargetsPercentage to be 80.0, got %f", updatedState.Metrics.OperationalTargetsPercentage)
	}
}
