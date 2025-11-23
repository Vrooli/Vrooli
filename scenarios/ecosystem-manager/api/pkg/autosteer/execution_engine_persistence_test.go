package autosteer

import (
	"testing"

	"github.com/google/uuid"
)

// TestExecutionEngine_PhaseAdvancementPersistence specifically tests that
// phase changes are properly persisted to the database
func TestExecutionEngine_PhaseAdvancementPersistence(t *testing.T) {
	pg, cleanup := SetupTestDatabase(t)
	if pg == nil {
		return
	}
	defer cleanup()

	vrooliRoot, cleanupScenario := SetupTestScenario(t, "test-scenario")
	defer cleanupScenario()

	profileService := NewProfileService(pg.db)
	metricsCollector := NewMetricsCollector(vrooliRoot)

	t.Run("phase advance persists across engine instances", func(t *testing.T) {
		// Create 2-phase profile
		phases := []SteerPhase{
			{
				ID:            uuid.New().String(),
				Mode:          ModeProgress,
				MaxIterations: 3,
				StopConditions: []StopCondition{
					{
						Type:            ConditionTypeSimple,
						Metric:          "loops",
						CompareOperator: OpGreaterThan,
						Value:           2,
					},
				},
			},
			{
				ID:            uuid.New().String(),
				Mode:          ModeTest,
				MaxIterations: 3,
				StopConditions: []StopCondition{
					{
						Type:            ConditionTypeSimple,
						Metric:          "loops",
						CompareOperator: OpGreaterThan,
						Value:           2,
					},
				},
			},
		}
		profile := CreateMultiPhaseProfile(t, "Persistence Test", phases)
		if err := profileService.CreateProfile(profile); err != nil {
			t.Fatalf("Failed to create profile: %v", err)
		}

		taskID := uuid.New().String()

		// Engine 1: Start execution
		engine1 := NewExecutionEngine(pg.db, profileService, metricsCollector)
		state, err := engine1.StartExecution(taskID, profile.ID, "test-scenario")
		if err != nil {
			t.Fatalf("StartExecution() error = %v", err)
		}
		if state.CurrentPhaseIndex != 0 {
			t.Fatalf("Expected initial phase 0, got %d", state.CurrentPhaseIndex)
		}

		// Engine 2: Verify state persisted from engine1
		engine2 := NewExecutionEngine(pg.db, profileService, metricsCollector)
		retrievedState, err := engine2.GetExecutionState(taskID)
		if err != nil {
			t.Fatalf("GetExecutionState() error = %v", err)
		}
		if retrievedState.CurrentPhaseIndex != 0 {
			t.Fatalf("Expected retrieved phase 0, got %d", retrievedState.CurrentPhaseIndex)
		}

		// Engine 1: Complete some iterations
		for i := 1; i <= 3; i++ {
			_, err := engine1.EvaluateIteration(taskID, "test-scenario", i)
			if err != nil {
				t.Fatalf("EvaluateIteration(%d) error = %v", i, err)
			}
		}

		// Engine 1: Advance phase
		t.Log("Advancing to next phase...")
		advanceResult, err := engine1.AdvancePhase(taskID, "test-scenario")
		if err != nil {
			t.Fatalf("AdvancePhase() error = %v", err)
		}
		if !advanceResult.Success {
			t.Fatalf("Expected successful advancement")
		}
		if advanceResult.Completed {
			t.Fatalf("Expected more phases")
		}

		// CRITICAL: Verify with engine1 immediately after advance
		stateAfterAdvance, err := engine1.GetExecutionState(taskID)
		if err != nil {
			t.Fatalf("GetExecutionState() after advance error = %v", err)
		}
		t.Logf("State after advance (engine1): phase_index=%d, phase_iteration=%d",
			stateAfterAdvance.CurrentPhaseIndex, stateAfterAdvance.CurrentPhaseIteration)

		if stateAfterAdvance.CurrentPhaseIndex != 1 {
			t.Errorf("IMMEDIATE CHECK FAILED: Expected phase 1 after advance, got %d", stateAfterAdvance.CurrentPhaseIndex)
		}

		// CRITICAL: Create NEW engine and verify persistence
		engine3 := NewExecutionEngine(pg.db, profileService, metricsCollector)
		persistedState, err := engine3.GetExecutionState(taskID)
		if err != nil {
			t.Fatalf("GetExecutionState() from new engine error = %v", err)
		}
		t.Logf("Persisted state (engine3): phase_index=%d, phase_iteration=%d",
			persistedState.CurrentPhaseIndex, persistedState.CurrentPhaseIteration)

		if persistedState.CurrentPhaseIndex != 1 {
			t.Errorf("PERSISTENCE CHECK FAILED: Expected persisted phase 1, got %d", persistedState.CurrentPhaseIndex)
			t.Errorf("This indicates the phase advancement was not properly saved to the database")
		}

		if persistedState.CurrentPhaseIteration != 0 {
			t.Errorf("Expected phase iteration reset to 0, got %d", persistedState.CurrentPhaseIteration)
		}

		// Verify phase history was updated
		if len(persistedState.PhaseHistory) != 1 {
			t.Errorf("Expected 1 completed phase in history, got %d", len(persistedState.PhaseHistory))
		}
	})

	t.Run("verify database row is actually updated", func(t *testing.T) {
		// Create single-phase profile
		profile := CreateTestProfile(t, "DB Update Test", ModeProgress, 5)
		if err := profileService.CreateProfile(profile); err != nil {
			t.Fatalf("Failed to create profile: %v", err)
		}

		taskID := uuid.New().String()
		engine := NewExecutionEngine(pg.db, profileService, metricsCollector)

		// Start execution
		_, err := engine.StartExecution(taskID, profile.ID, "test-scenario")
		if err != nil {
			t.Fatalf("StartExecution() error = %v", err)
		}

		// Query database directly
		var phaseIndex int
		query := `SELECT current_phase_index FROM profile_execution_state WHERE task_id = $1`
		err = pg.db.QueryRow(query, taskID).Scan(&phaseIndex)
		if err != nil {
			t.Fatalf("Direct DB query error = %v", err)
		}

		t.Logf("Direct DB query: phase_index=%d", phaseIndex)
		if phaseIndex != 0 {
			t.Errorf("Expected DB to have phase 0, got %d", phaseIndex)
		}

		// Update iteration count in memory
		state, _ := engine.GetExecutionState(taskID)
		state.CurrentPhaseIteration = 99

		// Save the state
		if err := engine.saveExecutionState(state); err != nil {
			t.Fatalf("saveExecutionState() error = %v", err)
		}

		// Query database directly again
		var updatedIteration int
		query2 := `SELECT current_phase_iteration FROM profile_execution_state WHERE task_id = $1`
		err = pg.db.QueryRow(query2, taskID).Scan(&updatedIteration)
		if err != nil {
			t.Fatalf("Direct DB query after update error = %v", err)
		}

		t.Logf("Direct DB query after update: iteration=%d", updatedIteration)
		if updatedIteration != 99 {
			t.Errorf("Expected DB to have iteration 99, got %d - saveExecutionState() not working!", updatedIteration)
		}
	})
}
