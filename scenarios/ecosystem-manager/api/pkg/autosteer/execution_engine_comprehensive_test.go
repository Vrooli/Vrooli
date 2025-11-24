package autosteer

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestExecutionEngine_FullCycle(t *testing.T) {
	pg, cleanup := SetupTestDatabase(t)
	if pg == nil {
		return
	}
	defer cleanup()

	vrooliRoot, cleanupScenario := SetupTestScenario(t, "test-scenario")
	defer cleanupScenario()

	profileService := NewProfileService(pg.db)
	metricsCollector := NewMetricsCollector(vrooliRoot)
	engine := NewExecutionEngine(pg.db, profileService, metricsCollector, testPhasePromptsDir(t))

	t.Run("start execution and verify state", func(t *testing.T) {
		// Create profile
		profile := CreateTestProfile(t, "Full Cycle Test", ModeProgress, 10)
		if err := profileService.CreateProfile(profile); err != nil {
			t.Fatalf("Failed to create profile: %v", err)
		}

		// Start execution
		taskID := uuid.New().String()
		state, err := engine.StartExecution(taskID, profile.ID, "test-scenario")
		if err != nil {
			t.Fatalf("StartExecution() error = %v", err)
		}

		// Verify state
		if state.TaskID != taskID {
			t.Errorf("Expected task ID %s, got %s", taskID, state.TaskID)
		}
		if state.ProfileID != profile.ID {
			t.Errorf("Expected profile ID %s, got %s", profile.ID, state.ProfileID)
		}
		if state.CurrentPhaseIndex != 0 {
			t.Errorf("Expected phase index 0, got %d", state.CurrentPhaseIndex)
		}

		// Verify can retrieve state
		retrievedState, err := engine.GetExecutionState(taskID)
		if err != nil {
			t.Fatalf("GetExecutionState() error = %v", err)
		}
		if retrievedState == nil {
			t.Fatal("Expected to retrieve state")
		}
		if retrievedState.TaskID != taskID {
			t.Errorf("Expected retrieved task ID %s, got %s", taskID, retrievedState.TaskID)
		}
	})

	t.Run("evaluate iteration with stop condition", func(t *testing.T) {
		// Create profile with condition: loops > 5
		profile := &AutoSteerProfile{
			Name: "Iteration Test",
			Phases: []SteerPhase{
				{
					ID:            uuid.New().String(),
					Mode:          ModeProgress,
					MaxIterations: 20,
					StopConditions: []StopCondition{
						{
							Type:            ConditionTypeSimple,
							Metric:          "loops",
							CompareOperator: OpGreaterThan,
							Value:           5,
						},
					},
				},
			},
		}
		if err := profileService.CreateProfile(profile); err != nil {
			t.Fatalf("Failed to create profile: %v", err)
		}

		taskID := uuid.New().String()
		_, err := engine.StartExecution(taskID, profile.ID, "test-scenario")
		if err != nil {
			t.Fatalf("StartExecution() error = %v", err)
		}

		// Evaluate at loop 3 (should not stop)
		result, err := engine.EvaluateIteration(taskID, "test-scenario", 3)
		if err != nil {
			t.Fatalf("EvaluateIteration() error = %v", err)
		}
		if result.ShouldStop {
			t.Error("Expected iteration to continue at loop 3")
		}

		// Evaluate at loop 6 (should stop - condition met)
		result, err = engine.EvaluateIteration(taskID, "test-scenario", 6)
		if err != nil {
			t.Fatalf("EvaluateIteration() error = %v", err)
		}
		if !result.ShouldStop {
			t.Error("Expected iteration to stop at loop 6")
		}
		if result.Reason != "condition_met" {
			t.Errorf("Expected reason 'condition_met', got %s", result.Reason)
		}
	})

	t.Run("advance through multiple phases", func(t *testing.T) {
		// Create 2-phase profile (both using loops to avoid metric availability issues)
		phases := []SteerPhase{
			{
				ID:            uuid.New().String(),
				Mode:          ModeProgress,
				MaxIterations: 5,
				StopConditions: []StopCondition{
					{
						Type:            ConditionTypeSimple,
						Metric:          "loops",
						CompareOperator: OpGreaterThan,
						Value:           3,
					},
				},
			},
			{
				ID:            uuid.New().String(),
				Mode:          ModeTest,
				MaxIterations: 5,
				StopConditions: []StopCondition{
					{
						Type:            ConditionTypeSimple,
						Metric:          "loops",
						CompareOperator: OpGreaterThan,
						Value:           7,
					},
				},
			},
		}
		profile := CreateMultiPhaseProfile(t, "Multi-Phase Test", phases)
		if err := profileService.CreateProfile(profile); err != nil {
			t.Fatalf("Failed to create profile: %v", err)
		}

		taskID := uuid.New().String()
		state, err := engine.StartExecution(taskID, profile.ID, "test-scenario")
		if err != nil {
			t.Fatalf("StartExecution() error = %v", err)
		}

		// Verify starts in phase 0
		if state.CurrentPhaseIndex != 0 {
			t.Errorf("Expected phase 0, got %d", state.CurrentPhaseIndex)
		}

		// Complete first phase iterations
		for i := 1; i <= 4; i++ {
			_, err := engine.EvaluateIteration(taskID, "test-scenario", i)
			if err != nil {
				t.Fatalf("EvaluateIteration(%d) error = %v", i, err)
			}
		}

		// Advance to next phase
		advanceResult, err := engine.AdvancePhase(taskID, "test-scenario")
		if err != nil {
			t.Fatalf("AdvancePhase() error = %v", err)
		}
		if !advanceResult.Success {
			t.Error("Expected successful phase advancement")
		}
		if advanceResult.Completed {
			t.Error("Expected more phases to remain")
		}

		// Verify now in phase 1
		state, err = engine.GetExecutionState(taskID)
		if err != nil {
			t.Fatalf("GetExecutionState() error = %v", err)
		}
		if state.CurrentPhaseIndex != 1 {
			t.Errorf("Expected phase 1, got %d", state.CurrentPhaseIndex)
		}
		if len(state.PhaseHistory) != 1 {
			t.Errorf("Expected 1 completed phase, got %d", len(state.PhaseHistory))
		}

		// Complete second phase - advance should complete execution
		for i := 1; i <= 5; i++ {
			_, err := engine.EvaluateIteration(taskID, "test-scenario", i)
			if err != nil {
				t.Fatalf("Phase 2 EvaluateIteration(%d) error = %v", i, err)
			}
		}

		advanceResult, err = engine.AdvancePhase(taskID, "test-scenario")
		if err != nil {
			t.Fatalf("Final AdvancePhase() error = %v", err)
		}
		if !advanceResult.Completed {
			t.Error("Expected execution to be completed")
		}

		// Verify state is removed after completion
		state, err = engine.GetExecutionState(taskID)
		if err != nil {
			t.Fatalf("GetExecutionState() error = %v", err)
		}
		if state != nil {
			t.Error("Expected state to be removed after completion")
		}
	})
}

func TestExecutionEngine_GetCurrentMode(t *testing.T) {
	pg, cleanup := SetupTestDatabase(t)
	if pg == nil {
		return
	}
	defer cleanup()

	vrooliRoot, cleanupScenario := SetupTestScenario(t, "test-scenario")
	defer cleanupScenario()

	profileService := NewProfileService(pg.db)
	metricsCollector := NewMetricsCollector(vrooliRoot)
	engine := NewExecutionEngine(pg.db, profileService, metricsCollector, testPhasePromptsDir(t))

	t.Run("get mode for active execution", func(t *testing.T) {
		profile := CreateTestProfile(t, "Mode Test", ModeUX, 10)
		if err := profileService.CreateProfile(profile); err != nil {
			t.Fatalf("Failed to create profile: %v", err)
		}

		taskID := uuid.New().String()
		_, err := engine.StartExecution(taskID, profile.ID, "test-scenario")
		if err != nil {
			t.Fatalf("StartExecution() error = %v", err)
		}

		mode, err := engine.GetCurrentMode(taskID)
		if err != nil {
			t.Fatalf("GetCurrentMode() error = %v", err)
		}

		if mode != ModeUX {
			t.Errorf("Expected mode %s, got %s", ModeUX, mode)
		}
	})

	t.Run("get mode for inactive task", func(t *testing.T) {
		mode, err := engine.GetCurrentMode(uuid.New().String())
		if err != nil {
			t.Fatalf("GetCurrentMode() error = %v", err)
		}

		if mode != "" {
			t.Errorf("Expected empty mode for inactive task, got %s", mode)
		}
	})
}

func TestExecutionEngine_IsAutoSteerActive(t *testing.T) {
	pg, cleanup := SetupTestDatabase(t)
	if pg == nil {
		return
	}
	defer cleanup()

	vrooliRoot, cleanupScenario := SetupTestScenario(t, "test-scenario")
	defer cleanupScenario()

	profileService := NewProfileService(pg.db)
	metricsCollector := NewMetricsCollector(vrooliRoot)
	engine := NewExecutionEngine(pg.db, profileService, metricsCollector, testPhasePromptsDir(t))

	t.Run("inactive before start", func(t *testing.T) {
		taskID := uuid.New().String()
		active, err := engine.IsAutoSteerActive(taskID)
		if err != nil {
			t.Fatalf("IsAutoSteerActive() error = %v", err)
		}
		if active {
			t.Error("Expected not active before start")
		}
	})

	t.Run("active after start", func(t *testing.T) {
		profile := CreateTestProfile(t, "Active Test", ModeProgress, 10)
		if err := profileService.CreateProfile(profile); err != nil {
			t.Fatalf("Failed to create profile: %v", err)
		}

		taskID := uuid.New().String()
		_, err := engine.StartExecution(taskID, profile.ID, "test-scenario")
		if err != nil {
			t.Fatalf("StartExecution() error = %v", err)
		}

		active, err := engine.IsAutoSteerActive(taskID)
		if err != nil {
			t.Fatalf("IsAutoSteerActive() error = %v", err)
		}
		if !active {
			t.Error("Expected active after start")
		}
	})
}

func TestExecutionEngine_GetEnhancedPrompt(t *testing.T) {
	pg, cleanup := SetupTestDatabase(t)
	if pg == nil {
		return
	}
	defer cleanup()

	vrooliRoot, cleanupScenario := SetupTestScenario(t, "test-scenario")
	defer cleanupScenario()

	profileService := NewProfileService(pg.db)
	metricsCollector := NewMetricsCollector(vrooliRoot)
	engine := NewExecutionEngine(pg.db, profileService, metricsCollector, testPhasePromptsDir(t))

	t.Run("get enhanced prompt with profile info", func(t *testing.T) {
		profile := &AutoSteerProfile{
			Name:        "Prompt Test Profile",
			Description: "Testing prompt generation",
			Phases: []SteerPhase{
				{
					ID:            uuid.New().String(),
					Mode:          ModeProgress,
					MaxIterations: 10,
					Description:   "Complete operational targets",
					StopConditions: []StopCondition{
						{
							Type:            ConditionTypeSimple,
							Metric:          "operational_targets_percentage",
							CompareOperator: OpGreaterThanEquals,
							Value:           80,
						},
					},
				},
			},
			QualityGates: []QualityGate{
				{
					Name: "build_health",
					Condition: StopCondition{
						Type:            ConditionTypeSimple,
						Metric:          "build_status",
						CompareOperator: OpEquals,
						Value:           1,
					},
					FailureAction: ActionHalt,
					Message:       "Build must pass",
				},
			},
		}
		if err := profileService.CreateProfile(profile); err != nil {
			t.Fatalf("Failed to create profile: %v", err)
		}

		taskID := uuid.New().String()
		_, err := engine.StartExecution(taskID, profile.ID, "test-scenario")
		if err != nil {
			t.Fatalf("StartExecution() error = %v", err)
		}

		// Run a few iterations
		for i := 1; i <= 3; i++ {
			_, err := engine.EvaluateIteration(taskID, "test-scenario", i)
			if err != nil {
				t.Fatalf("EvaluateIteration(%d) error = %v", i, err)
			}
		}

		// Get enhanced prompt
		prompt, err := engine.GetEnhancedPrompt(taskID)
		if err != nil {
			t.Fatalf("GetEnhancedPrompt() error = %v", err)
		}

		if prompt == "" {
			t.Error("Expected non-empty enhanced prompt")
		}

		// Verify prompt contains key information (simple substring checks)
		if !contains(prompt, profile.Name) {
			t.Error("Expected prompt to contain profile name")
		}
		if !contains(prompt, "PROGRESS") {
			t.Error("Expected prompt to contain current mode")
		}
		if !contains(prompt, "Phase 1 of 1") {
			t.Error("Expected prompt to contain phase progress")
		}

		t.Logf("Generated prompt (%d chars):\n%s", len(prompt), prompt[:min(len(prompt), 500)])
	})
}

func contains(haystack, needle string) bool {
	for i := 0; i <= len(haystack)-len(needle); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func timePtr(t time.Time) *time.Time {
	return &t
}
