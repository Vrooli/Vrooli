package autosteer

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

func TestAutoSteerHandlers_ExecutionFlow(t *testing.T) {
	pg, cleanup := SetupTestDatabase(t)
	if pg == nil {
		return
	}
	defer cleanup()

	vrooliRoot, cleanupScenario := SetupTestScenario(t, "test-scenario")
	defer cleanupScenario()

	// Initialize services
	profileService := NewProfileService(pg.db)
	metricsCollector := NewMetricsCollector(vrooliRoot)
	executionEngine := NewExecutionEngine(pg.db, profileService, metricsCollector)
	historyService := NewHistoryService(pg.db)

	// Create handlers
	handlers := NewAutoSteerHandlers(profileService, executionEngine, historyService)

	// Create 2-phase test profile
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
	profile := CreateMultiPhaseProfile(t, "Handler Test", phases)
	if err := profileService.CreateProfile(profile); err != nil {
		t.Fatalf("Failed to create profile: %v", err)
	}

	taskID := uuid.New().String()

	t.Run("complete execution flow through handlers", func(t *testing.T) {
		// Step 1: Start execution
		startReq := struct {
			TaskID       string `json:"task_id"`
			ProfileID    string `json:"profile_id"`
			ScenarioName string `json:"scenario_name"`
		}{
			TaskID:       taskID,
			ProfileID:    profile.ID,
			ScenarioName: "test-scenario",
		}

		body, _ := json.Marshal(startReq)
		req := httptest.NewRequest("POST", "/api/auto-steer/execution/start", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		handlers.StartExecution(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("Expected status 201, got %d: %s", w.Code, w.Body.String())
		}

		var startState ProfileExecutionState
		if err := json.NewDecoder(w.Body).Decode(&startState); err != nil {
			t.Fatalf("Failed to decode start response: %v", err)
		}

		if startState.CurrentPhaseIndex != 0 {
			t.Errorf("Expected initial phase 0, got %d", startState.CurrentPhaseIndex)
		}

		// Step 2: Run some iterations
		for i := 1; i <= 3; i++ {
			evalReq := struct {
				TaskID       string `json:"task_id"`
				ScenarioName string `json:"scenario_name"`
				Loops        int    `json:"loops"`
			}{
				TaskID:       taskID,
				ScenarioName: "test-scenario",
				Loops:        i,
			}

			body, _ := json.Marshal(evalReq)
			req := httptest.NewRequest("POST", "/api/auto-steer/execution/evaluate", bytes.NewBuffer(body))
			w := httptest.NewRecorder()

			handlers.EvaluateIteration(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("Iteration %d: expected status 200, got %d: %s", i, w.Code, w.Body.String())
			}
		}

		// Step 3: Advance to next phase
		advanceReq := struct {
			TaskID       string `json:"task_id"`
			ScenarioName string `json:"scenario_name"`
		}{
			TaskID:       taskID,
			ScenarioName: "test-scenario",
		}

		body, _ = json.Marshal(advanceReq)
		req = httptest.NewRequest("POST", "/api/auto-steer/execution/advance", bytes.NewBuffer(body))
		w = httptest.NewRecorder()

		handlers.AdvancePhase(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var advanceResult PhaseAdvanceResult
		if err := json.NewDecoder(w.Body).Decode(&advanceResult); err != nil {
			t.Fatalf("Failed to decode advance response: %v", err)
		}

		if !advanceResult.Success {
			t.Error("Expected successful phase advancement")
		}

		if advanceResult.NextPhaseIndex != 1 {
			t.Errorf("Expected next phase index 1, got %d", advanceResult.NextPhaseIndex)
		}

		// Step 4: CRITICAL - Verify phase persisted by retrieving state
		req = httptest.NewRequest("GET", "/api/auto-steer/execution/"+taskID, nil)
		req = mux.SetURLVars(req, map[string]string{"taskId": taskID})
		w = httptest.NewRecorder()

		handlers.GetExecutionState(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var retrievedState ProfileExecutionState
		if err := json.NewDecoder(w.Body).Decode(&retrievedState); err != nil {
			t.Fatalf("Failed to decode state response: %v", err)
		}

		t.Logf("Retrieved state: phase_index=%d, iteration=%d",
			retrievedState.CurrentPhaseIndex, retrievedState.CurrentPhaseIteration)

		// MAIN ASSERTION: Phase should be persisted
		if retrievedState.CurrentPhaseIndex != 1 {
			t.Errorf("PERSISTENCE FAILURE: Expected phase 1 after advance, got %d", retrievedState.CurrentPhaseIndex)
			t.Error("The phase advancement was not persisted through the HTTP handlers")
		}

		if retrievedState.CurrentPhaseIteration != 0 {
			t.Errorf("Expected iteration reset to 0, got %d", retrievedState.CurrentPhaseIteration)
		}

		if len(retrievedState.PhaseHistory) != 1 {
			t.Errorf("Expected 1 completed phase in history, got %d", len(retrievedState.PhaseHistory))
		}
	})

	t.Run("validate error handling", func(t *testing.T) {
		// Test missing fields
		invalidReq := struct {
			TaskID string `json:"task_id"`
			// Missing profile_id and scenario_name
		}{
			TaskID: uuid.New().String(),
		}

		body, _ := json.Marshal(invalidReq)
		req := httptest.NewRequest("POST", "/api/auto-steer/execution/start", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		handlers.StartExecution(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for invalid request, got %d", w.Code)
		}
	})
}
