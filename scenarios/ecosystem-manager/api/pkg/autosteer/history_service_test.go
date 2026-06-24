package autosteer

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ecosystem-manager/api/pkg/internal/testdb"
)

// setupHistoryTestDB opens a temp SQLite database with the autosteer schema
// (the same DDL the API applies at boot via database.EnsureSchemas).
func setupHistoryTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	return testdb.NewSQLite(t, Schema()), func() {}
}

func createTestExecution(t *testing.T, db *sql.DB, profileID string, scenarioName string) string {
	t.Helper()

	taskID := uuid.New().String()

	phaseBreakdown := []SkillPerformance{
		{
			SkillName:     "progress",
			Iterations:    10,
			WeightedDelta: 30.0,
		},
		{
			SkillName:     "ux",
			Iterations:    5,
			WeightedDelta: 11.4,
		},
	}

	phaseBreakdownJSON, _ := json.Marshal(phaseBreakdown)

	query := `
		INSERT INTO profile_executions (
			id, profile_id, task_id, scenario_name,
			phase_breakdown, total_iterations, total_duration_ms, executed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := db.Exec(query,
		uuid.NewString(),
		profileID,
		taskID,
		scenarioName,
		phaseBreakdownJSON,
		15,
		900000, // 15 minutes
		time.Now().UTC(),
	)
	if err != nil {
		t.Fatalf("Failed to create test execution: %v", err)
	}

	return taskID
}

func TestHistoryService_GetHistory(t *testing.T) {
	db, cleanup := setupHistoryTestDB(t)
	if db == nil {
		return
	}
	defer cleanup()

	service := NewHistoryService(db)

	// Create test data
	profileID1 := uuid.New().String()
	profileID2 := uuid.New().String()

	createTestExecution(t, db, profileID1, "scenario-a")
	createTestExecution(t, db, profileID1, "scenario-a")
	createTestExecution(t, db, profileID2, "scenario-b")

	t.Run("get all history", func(t *testing.T) {
		history, err := service.GetHistory(HistoryFilters{})
		if err != nil {
			t.Fatalf("GetHistory() error = %v", err)
		}

		if len(history) != 3 {
			t.Errorf("Expected 3 executions, got %d", len(history))
		}

		// Verify executions have required fields
		for i, exec := range history {
			if exec.ID == "" {
				t.Errorf("Execution %d missing ID", i)
			}
			if exec.ProfileID == "" {
				t.Errorf("Execution %d missing ProfileID", i)
			}
			if exec.ScenarioName == "" {
				t.Errorf("Execution %d missing ScenarioName", i)
			}
			if exec.TotalIterations == 0 {
				t.Errorf("Execution %d has zero iterations", i)
			}
			if len(exec.PhaseBreakdown) == 0 {
				t.Errorf("Execution %d has no phase breakdown", i)
			}
		}
	})

	t.Run("filter by profile ID", func(t *testing.T) {
		history, err := service.GetHistory(HistoryFilters{
			ProfileID: profileID1,
		})
		if err != nil {
			t.Fatalf("GetHistory() error = %v", err)
		}

		if len(history) != 2 {
			t.Errorf("Expected 2 executions for profile 1, got %d", len(history))
		}

		for _, exec := range history {
			if exec.ProfileID != profileID1 {
				t.Errorf("Expected profile ID %s, got %s", profileID1, exec.ProfileID)
			}
		}
	})

	t.Run("filter by scenario name", func(t *testing.T) {
		history, err := service.GetHistory(HistoryFilters{
			ScenarioName: "scenario-a",
		})
		if err != nil {
			t.Fatalf("GetHistory() error = %v", err)
		}

		if len(history) != 2 {
			t.Errorf("Expected 2 executions for scenario-a, got %d", len(history))
		}

		for _, exec := range history {
			if exec.ScenarioName != "scenario-a" {
				t.Errorf("Expected scenario 'scenario-a', got %s", exec.ScenarioName)
			}
		}
	})

	t.Run("filter by date range", func(t *testing.T) {
		startDate := time.Now().Add(-2 * time.Hour)
		endDate := time.Now().Add(1 * time.Hour)

		history, err := service.GetHistory(HistoryFilters{
			StartDate: &startDate,
			EndDate:   &endDate,
		})
		if err != nil {
			t.Fatalf("GetHistory() error = %v", err)
		}

		if len(history) != 3 {
			t.Errorf("Expected 3 executions in date range, got %d", len(history))
		}

		// Test with future start date (should get nothing)
		futureDate := time.Now().Add(24 * time.Hour)
		history, err = service.GetHistory(HistoryFilters{
			StartDate: &futureDate,
		})
		if err != nil {
			t.Fatalf("GetHistory() error = %v", err)
		}

		if len(history) != 0 {
			t.Errorf("Expected 0 executions with future start date, got %d", len(history))
		}
	})

	t.Run("combine multiple filters", func(t *testing.T) {
		history, err := service.GetHistory(HistoryFilters{
			ProfileID:    profileID1,
			ScenarioName: "scenario-a",
		})
		if err != nil {
			t.Fatalf("GetHistory() error = %v", err)
		}

		if len(history) != 2 {
			t.Errorf("Expected 2 executions matching filters, got %d", len(history))
		}
	})
}

func TestHistoryService_GetExecution(t *testing.T) {
	db, cleanup := setupHistoryTestDB(t)
	if db == nil {
		return
	}
	defer cleanup()

	service := NewHistoryService(db)

	profileID := uuid.New().String()
	taskID := createTestExecution(t, db, profileID, "test-scenario")

	t.Run("get existing execution", func(t *testing.T) {
		exec, err := service.GetExecution(taskID)
		if err != nil {
			t.Fatalf("GetExecution() error = %v", err)
		}

		if exec.ExecutionID != taskID {
			t.Errorf("Expected execution ID %s, got %s", taskID, exec.ExecutionID)
		}
		if exec.ProfileID != profileID {
			t.Errorf("Expected profile ID %s, got %s", profileID, exec.ProfileID)
		}
		if exec.ScenarioName != "test-scenario" {
			t.Errorf("Expected scenario 'test-scenario', got %s", exec.ScenarioName)
		}
		if exec.TotalIterations != 15 {
			t.Errorf("Expected 15 iterations, got %d", exec.TotalIterations)
		}
		if len(exec.PhaseBreakdown) != 2 {
			t.Errorf("Expected 2 phases, got %d", len(exec.PhaseBreakdown))
		}
	})

	t.Run("get non-existent execution", func(t *testing.T) {
		_, err := service.GetExecution(uuid.New().String())
		if err == nil {
			t.Error("Expected error for non-existent execution")
		}
	})
}

func TestHistoryService_GetProfileAnalytics(t *testing.T) {
	db, cleanup := setupHistoryTestDB(t)
	if db == nil {
		return
	}
	defer cleanup()

	service := NewHistoryService(db)

	profileID := uuid.New().String()

	// Create multiple executions with different scenarios
	createTestExecution(t, db, profileID, "scenario-a")
	createTestExecution(t, db, profileID, "scenario-a")
	createTestExecution(t, db, profileID, "scenario-b")

	t.Run("get analytics for profile with executions", func(t *testing.T) {
		analytics, err := service.GetProfileAnalytics(profileID)
		if err != nil {
			t.Fatalf("GetProfileAnalytics() error = %v", err)
		}

		if analytics.ProfileID != profileID {
			t.Errorf("Expected profile ID %s, got %s", profileID, analytics.ProfileID)
		}

		if analytics.TotalExecutions != 3 {
			t.Errorf("Expected 3 total executions, got %d", analytics.TotalExecutions)
		}

		if analytics.AvgIterations != 15.0 {
			t.Errorf("Expected avg iterations 15.0, got %f", analytics.AvgIterations)
		}

		// Verify phase statistics
		if len(analytics.PhaseStats) == 0 {
			t.Error("Expected phase statistics")
		}

		progressStats, ok := analytics.PhaseStats["progress"]
		if !ok {
			t.Error("Expected statistics for Progress mode")
		} else {
			if progressStats.TotalExecutions != 3 {
				t.Errorf("Expected 3 progress phase executions, got %d", progressStats.TotalExecutions)
			}
			if progressStats.AvgIterations != 10.0 {
				t.Errorf("Expected avg 10 iterations for progress, got %f", progressStats.AvgIterations)
			}
			if progressStats.AvgWeightedDelta != 30.0 {
				t.Errorf("Expected avg weighted delta 30.0 for progress, got %f", progressStats.AvgWeightedDelta)
			}
		}

		// Verify scenario statistics
		if len(analytics.ScenarioStats) != 2 {
			t.Errorf("Expected 2 scenarios, got %d", len(analytics.ScenarioStats))
		}

		for _, stats := range analytics.ScenarioStats {
			if stats.ScenarioName == "scenario-a" {
				if stats.ExecutionCount != 2 {
					t.Errorf("Expected 2 executions for scenario-a, got %d", stats.ExecutionCount)
				}
			}
		}
	})

	t.Run("get analytics for profile with no executions", func(t *testing.T) {
		analytics, err := service.GetProfileAnalytics(uuid.New().String())
		if err != nil {
			t.Fatalf("GetProfileAnalytics() error = %v", err)
		}

		if analytics.TotalExecutions != 0 {
			t.Errorf("Expected 0 total executions, got %d", analytics.TotalExecutions)
		}

		if len(analytics.PhaseStats) != 0 {
			t.Errorf("Expected no phase stats, got %d", len(analytics.PhaseStats))
		}

		if len(analytics.ScenarioStats) != 0 {
			t.Errorf("Expected no scenario stats, got %d", len(analytics.ScenarioStats))
		}
	})
}

func TestHistoryService_PhaseEffectiveness(t *testing.T) {
	db, cleanup := setupHistoryTestDB(t)
	if db == nil {
		return
	}
	defer cleanup()

	service := NewHistoryService(db)
	profileID := uuid.New().String()
	createTestExecution(t, db, profileID, "test-scenario")

	analytics, err := service.GetProfileAnalytics(profileID)
	if err != nil {
		t.Fatalf("GetProfileAnalytics() error = %v", err)
	}

	// Check UX phase effectiveness
	uxStats, ok := analytics.PhaseStats["ux"]
	if !ok {
		t.Fatal("Expected UX phase statistics")
	}

	// UX skill had a realized weighted delta of 11.4 (from test data)
	if uxStats.AvgWeightedDelta != 11.4 {
		t.Errorf("Expected UX avg weighted delta 11.4, got %f", uxStats.AvgWeightedDelta)
	}

	// UX skill had 5 iterations
	if uxStats.AvgIterations != 5.0 {
		t.Errorf("Expected UX avg iterations 5.0, got %f", uxStats.AvgIterations)
	}
}
