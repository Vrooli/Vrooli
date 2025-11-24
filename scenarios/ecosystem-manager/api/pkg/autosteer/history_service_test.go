package autosteer

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

func setupHistoryTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()

	connStr := "host=localhost port=5432 user=ecosystem_manager password=ecosystem_manager_pass dbname=ecosystem_manager_test sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Skipf("Skipping test: cannot connect to test database: %v", err)
		return nil, nil
	}

	if err := db.Ping(); err != nil {
		t.Skipf("Skipping test: database not available: %v", err)
		return nil, nil
	}

	setupSQL := `
		CREATE TABLE IF NOT EXISTS profile_executions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			profile_id UUID NOT NULL,
			task_id UUID NOT NULL,
			scenario_name VARCHAR(255) NOT NULL,
			start_metrics JSONB,
			end_metrics JSONB,
			phase_breakdown JSONB,
			total_iterations INTEGER,
			total_duration_ms BIGINT,
			user_rating INTEGER,
			user_comments TEXT,
			user_feedback_at TIMESTAMP,
			executed_at TIMESTAMP DEFAULT NOW()
		);
	`

	if _, err := db.Exec(setupSQL); err != nil {
		t.Fatalf("Failed to create test tables: %v", err)
	}

	cleanup := func() {
		_, _ = db.Exec("TRUNCATE profile_executions CASCADE")
		db.Close()
	}

	return db, cleanup
}

func createTestExecution(t *testing.T, db *sql.DB, profileID string, scenarioName string, withFeedback bool) string {
	t.Helper()

	taskID := uuid.New().String()

	startMetrics := MetricsSnapshot{
		Timestamp:                    time.Now().Add(-1 * time.Hour),
		PhaseLoops:                   0,
		TotalLoops:                   0,
		BuildStatus:                  1,
		OperationalTargetsTotal:      10,
		OperationalTargetsPassing:    5,
		OperationalTargetsPercentage: 50.0,
		UX: &UXMetrics{
			AccessibilityScore: 70.0,
			UITestCoverage:     40.0,
		},
	}

	endMetrics := MetricsSnapshot{
		Timestamp:                    time.Now(),
		PhaseLoops:                   5,
		TotalLoops:                   15,
		BuildStatus:                  1,
		OperationalTargetsTotal:      10,
		OperationalTargetsPassing:    9,
		OperationalTargetsPercentage: 90.0,
		UX: &UXMetrics{
			AccessibilityScore: 92.0,
			UITestCoverage:     75.0,
		},
	}

	phaseBreakdown := []PhasePerformance{
		{
			Mode:       ModeProgress,
			Iterations: 10,
			MetricDeltas: map[string]float64{
				"operational_targets_percentage": 30.0,
			},
			Duration:      600000, // 10 minutes
			Effectiveness: 3.0,
		},
		{
			Mode:       ModeUX,
			Iterations: 5,
			MetricDeltas: map[string]float64{
				"accessibility_score": 22.0,
				"ui_test_coverage":    35.0,
			},
			Duration:      300000, // 5 minutes
			Effectiveness: 11.4,
		},
	}

	startMetricsJSON, _ := json.Marshal(startMetrics)
	endMetricsJSON, _ := json.Marshal(endMetrics)
	phaseBreakdownJSON, _ := json.Marshal(phaseBreakdown)

	query := `
		INSERT INTO profile_executions (
			profile_id, task_id, scenario_name, start_metrics, end_metrics,
			phase_breakdown, total_iterations, total_duration_ms, executed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := db.Exec(query,
		profileID,
		taskID,
		scenarioName,
		startMetricsJSON,
		endMetricsJSON,
		phaseBreakdownJSON,
		15,
		900000, // 15 minutes
		time.Now(),
	)

	if err != nil {
		t.Fatalf("Failed to create test execution: %v", err)
	}

	if withFeedback {
		feedbackQuery := `
			UPDATE profile_executions
			SET user_rating = $1, user_comments = $2, user_feedback_at = $3
			WHERE task_id = $4
		`
		_, err = db.Exec(feedbackQuery, 5, "Excellent results!", time.Now(), taskID)
		if err != nil {
			t.Fatalf("Failed to add feedback: %v", err)
		}
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

	createTestExecution(t, db, profileID1, "scenario-a", true)
	createTestExecution(t, db, profileID1, "scenario-a", false)
	createTestExecution(t, db, profileID2, "scenario-b", true)

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

	t.Run("user feedback is included", func(t *testing.T) {
		history, err := service.GetHistory(HistoryFilters{})
		if err != nil {
			t.Fatalf("GetHistory() error = %v", err)
		}

		feedbackCount := 0
		for _, exec := range history {
			if exec.UserFeedback != nil {
				feedbackCount++
				if exec.UserFeedback.Rating == 0 {
					t.Error("User feedback should have rating")
				}
			}
		}

		if feedbackCount != 2 {
			t.Errorf("Expected 2 executions with feedback, got %d", feedbackCount)
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
	taskID := createTestExecution(t, db, profileID, "test-scenario", true)

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
		if exec.UserFeedback == nil {
			t.Error("Expected user feedback to be present")
		}
		if exec.UserFeedback != nil && exec.UserFeedback.Rating != 5 {
			t.Errorf("Expected rating 5, got %d", exec.UserFeedback.Rating)
		}
	})

	t.Run("get non-existent execution", func(t *testing.T) {
		_, err := service.GetExecution(uuid.New().String())
		if err == nil {
			t.Error("Expected error for non-existent execution")
		}
	})
}

func TestHistoryService_SubmitFeedback(t *testing.T) {
	db, cleanup := setupHistoryTestDB(t)
	if db == nil {
		return
	}
	defer cleanup()

	service := NewHistoryService(db)

	profileID := uuid.New().String()
	taskID := createTestExecution(t, db, profileID, "test-scenario", false)

	t.Run("submit feedback successfully", func(t *testing.T) {
		err := service.SubmitFeedback(taskID, 4, "Good but could be better")
		if err != nil {
			t.Fatalf("SubmitFeedback() error = %v", err)
		}

		// Verify feedback was saved
		exec, err := service.GetExecution(taskID)
		if err != nil {
			t.Fatalf("GetExecution() error = %v", err)
		}

		if exec.UserFeedback == nil {
			t.Fatal("Expected user feedback to be present")
		}
		if exec.UserFeedback.Rating != 4 {
			t.Errorf("Expected rating 4, got %d", exec.UserFeedback.Rating)
		}
		if exec.UserFeedback.Comments != "Good but could be better" {
			t.Errorf("Expected comments 'Good but could be better', got %s", exec.UserFeedback.Comments)
		}
	})

	t.Run("update existing feedback", func(t *testing.T) {
		err := service.SubmitFeedback(taskID, 5, "Actually it's excellent!")
		if err != nil {
			t.Fatalf("SubmitFeedback() error = %v", err)
		}

		// Verify feedback was updated
		exec, err := service.GetExecution(taskID)
		if err != nil {
			t.Fatalf("GetExecution() error = %v", err)
		}

		if exec.UserFeedback.Rating != 5 {
			t.Errorf("Expected rating 5, got %d", exec.UserFeedback.Rating)
		}
		if exec.UserFeedback.Comments != "Actually it's excellent!" {
			t.Errorf("Expected updated comments, got %s", exec.UserFeedback.Comments)
		}
	})

	t.Run("submit feedback for non-existent execution", func(t *testing.T) {
		err := service.SubmitFeedback(uuid.New().String(), 5, "Test")
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
	createTestExecution(t, db, profileID, "scenario-a", true)
	createTestExecution(t, db, profileID, "scenario-a", true)
	createTestExecution(t, db, profileID, "scenario-b", false)

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

		if analytics.AvgDuration != 900000 {
			t.Errorf("Expected avg duration 900000ms, got %d", analytics.AvgDuration)
		}

		// Should have average rating from 2 executions with feedback
		if analytics.AvgRating != 5.0 {
			t.Errorf("Expected avg rating 5.0, got %f", analytics.AvgRating)
		}

		// Verify phase statistics
		if len(analytics.PhaseStats) == 0 {
			t.Error("Expected phase statistics")
		}

		progressStats, ok := analytics.PhaseStats[ModeProgress]
		if !ok {
			t.Error("Expected statistics for Progress mode")
		} else {
			if progressStats.TotalExecutions != 3 {
				t.Errorf("Expected 3 progress phase executions, got %d", progressStats.TotalExecutions)
			}
			if progressStats.AvgIterations != 10.0 {
				t.Errorf("Expected avg 10 iterations for progress, got %f", progressStats.AvgIterations)
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
				if stats.AvgImprovement != 40.0 {
					t.Errorf("Expected avg improvement 40.0, got %f", stats.AvgImprovement)
				}
				if stats.AvgRating != 5.0 {
					t.Errorf("Expected avg rating 5.0, got %f", stats.AvgRating)
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

		if analytics.AvgRating != 0 {
			t.Errorf("Expected avg rating 0, got %f", analytics.AvgRating)
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
	createTestExecution(t, db, profileID, "test-scenario", false)

	analytics, err := service.GetProfileAnalytics(profileID)
	if err != nil {
		t.Fatalf("GetProfileAnalytics() error = %v", err)
	}

	// Check UX phase effectiveness
	uxStats, ok := analytics.PhaseStats[ModeUX]
	if !ok {
		t.Fatal("Expected UX phase statistics")
	}

	// UX phase had effectiveness of 11.4 (from test data)
	if uxStats.AvgEffectiveness != 11.4 {
		t.Errorf("Expected UX effectiveness 11.4, got %f", uxStats.AvgEffectiveness)
	}

	// UX phase had 5 iterations and 5 minutes
	if uxStats.AvgIterations != 5.0 {
		t.Errorf("Expected UX avg iterations 5.0, got %f", uxStats.AvgIterations)
	}

	if uxStats.AvgDuration != 300000 {
		t.Errorf("Expected UX avg duration 300000ms, got %d", uxStats.AvgDuration)
	}
}
