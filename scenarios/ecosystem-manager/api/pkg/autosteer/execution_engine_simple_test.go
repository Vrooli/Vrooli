package autosteer

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// Simplified execution engine tests focusing on core functionality

func setupSimpleExecutionTest(t *testing.T) (*sql.DB, *ProfileService, *MetricsCollector, string, func()) {
	t.Helper()

	// Setup database
	connStr := "host=localhost port=5432 user=ecosystem_manager password=ecosystem_manager_pass dbname=ecosystem_manager_test sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Skipf("Skipping test: cannot connect to test database: %v", err)
		return nil, nil, nil, "", nil
	}

	if err := db.Ping(); err != nil {
		t.Skipf("Skipping test: database not available: %v", err)
		return nil, nil, nil, "", nil
	}

	// Create tables
	setupSQL := `
		CREATE TABLE IF NOT EXISTS auto_steer_profiles (
			id UUID PRIMARY KEY,
			name VARCHAR(255) NOT NULL UNIQUE,
			description TEXT,
			config JSONB NOT NULL,
			tags TEXT[],
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS profile_execution_state (
			task_id UUID PRIMARY KEY,
			profile_id UUID NOT NULL,
			current_phase_index INTEGER NOT NULL,
			current_phase_iteration INTEGER NOT NULL,
			phase_history JSONB,
			metrics JSONB,
			phase_start_metrics JSONB,
			started_at TIMESTAMP DEFAULT NOW(),
			last_updated TIMESTAMP DEFAULT NOW()
		);

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

	// Setup metrics collector with temp dir
	tmpDir, err := os.MkdirTemp("", "exec-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	scenarioDir := filepath.Join(tmpDir, "scenarios", "test-scenario", "requirements")
	if err := os.MkdirAll(scenarioDir, 0755); err != nil {
		t.Fatalf("Failed to create scenario dir: %v", err)
	}

	requirementsJSON := `{
		"modules": [{
			"id": "module-1",
			"operationalTargets": [
				{"id": "target-1", "status": "passing"},
				{"id": "target-2", "status": "passing"}
			]
		}]
	}`

	requirementsPath := filepath.Join(scenarioDir, "index.json")
	if err := os.WriteFile(requirementsPath, []byte(requirementsJSON), 0644); err != nil {
		t.Fatalf("Failed to write requirements file: %v", err)
	}

	profileService := NewProfileService(db)
	metricsCollector := NewMetricsCollector(tmpDir)

	cleanup := func() {
		_, _ = db.Exec("TRUNCATE auto_steer_profiles, profile_execution_state, profile_executions CASCADE")
		db.Close()
		os.RemoveAll(tmpDir)
	}

	return db, profileService, metricsCollector, "test-scenario", cleanup
}

func TestExecutionEngineSimple_StartAndGetState(t *testing.T) {
	db, profileService, metricsCollector, scenarioName, cleanup := setupSimpleExecutionTest(t)
	if db == nil {
		return
	}
	defer cleanup()

	// Create test profile
	profile := &AutoSteerProfile{
		Name: "Test Profile",
		Phases: []SteerPhase{
			{
				ID:            "phase-1",
				Mode:          ModeProgress,
				MaxIterations: 10,
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
	}

	if err := profileService.CreateProfile(profile); err != nil {
		t.Fatalf("Failed to create profile: %v", err)
	}

	engine := NewExecutionEngine(db, profileService, metricsCollector)
	taskID := uuid.New().String()

	// Start execution
	state, err := engine.StartExecution(taskID, profile.ID, scenarioName)
	if err != nil {
		t.Fatalf("StartExecution() error = %v", err)
	}

	if state.TaskID != taskID {
		t.Errorf("Expected task ID %s, got %s", taskID, state.TaskID)
	}
	if state.CurrentPhaseIndex != 0 {
		t.Errorf("Expected phase index 0, got %d", state.CurrentPhaseIndex)
	}

	// Get state
	retrieved, err := engine.GetExecutionState(taskID)
	if err != nil {
		t.Fatalf("GetExecutionState() error = %v", err)
	}

	if retrieved.TaskID != taskID {
		t.Errorf("Expected retrieved task ID %s, got %s", taskID, retrieved.TaskID)
	}
}

func TestExecutionEngineSimple_IsAutoSteerActive(t *testing.T) {
	db, profileService, metricsCollector, scenarioName, cleanup := setupSimpleExecutionTest(t)
	if db == nil {
		return
	}
	defer cleanup()

	profile := &AutoSteerProfile{
		Name: "Test Profile",
		Phases: []SteerPhase{
			{
				ID:            "phase-1",
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
		},
	}

	if err := profileService.CreateProfile(profile); err != nil {
		t.Fatalf("Failed to create profile: %v", err)
	}

	engine := NewExecutionEngine(db, profileService, metricsCollector)
	taskID := uuid.New().String()

	// Should not be active before start
	active, err := engine.IsAutoSteerActive(taskID)
	if err != nil {
		t.Fatalf("IsAutoSteerActive() error = %v", err)
	}
	if active {
		t.Error("Expected auto steer to not be active")
	}

	// Start execution
	_, err = engine.StartExecution(taskID, profile.ID, scenarioName)
	if err != nil {
		t.Fatalf("StartExecution() error = %v", err)
	}

	// Should be active after start
	active, err = engine.IsAutoSteerActive(taskID)
	if err != nil {
		t.Fatalf("IsAutoSteerActive() error = %v", err)
	}
	if !active {
		t.Error("Expected auto steer to be active")
	}
}

func TestExecutionEngineSimple_GetCurrentMode(t *testing.T) {
	db, profileService, metricsCollector, scenarioName, cleanup := setupSimpleExecutionTest(t)
	if db == nil {
		return
	}
	defer cleanup()

	profile := &AutoSteerProfile{
		Name: "Test Profile",
		Phases: []SteerPhase{
			{
				ID:            "phase-1",
				Mode:          ModeUX,
				MaxIterations: 5,
				StopConditions: []StopCondition{
					{
						Type:            ConditionTypeSimple,
						Metric:          "accessibility_score",
						CompareOperator: OpGreaterThan,
						Value:           90,
					},
				},
			},
		},
	}

	if err := profileService.CreateProfile(profile); err != nil {
		t.Fatalf("Failed to create profile: %v", err)
	}

	engine := NewExecutionEngine(db, profileService, metricsCollector)
	taskID := uuid.New().String()

	// Start execution
	_, err := engine.StartExecution(taskID, profile.ID, scenarioName)
	if err != nil {
		t.Fatalf("StartExecution() error = %v", err)
	}

	// Get current mode
	mode, err := engine.GetCurrentMode(taskID)
	if err != nil {
		t.Fatalf("GetCurrentMode() error = %v", err)
	}

	if mode != ModeUX {
		t.Errorf("Expected mode %s, got %s", ModeUX, mode)
	}
}

func TestCalculateMetricDeltasAndEffectiveness(t *testing.T) {
	start := MetricsSnapshot{
		OperationalTargetsPercentage: 60.0,
		UX: &UXMetrics{
			AccessibilityScore: 70.0,
		},
	}

	end := MetricsSnapshot{
		OperationalTargetsPercentage: 85.0,
		UX: &UXMetrics{
			AccessibilityScore: 92.0,
		},
	}

	deltas := calculateMetricDeltas(start, end)

	if deltas["operational_targets_percentage"] != 25.0 {
		t.Errorf("Expected op targets delta 25.0, got %f", deltas["operational_targets_percentage"])
	}

	if deltas["accessibility_score"] != 22.0 {
		t.Errorf("Expected accessibility delta 22.0, got %f", deltas["accessibility_score"])
	}

	// Test effectiveness
	phase := PhaseExecution{
		Iterations:   5,
		StartMetrics: start,
		EndMetrics:   end,
	}

	effectiveness := calculateEffectiveness(phase)

	// Total improvement: 25 + 22 = 47
	// Effectiveness: 47 / 5 = 9.4
	expected := 9.4

	if effectiveness != expected {
		t.Errorf("Expected effectiveness %f, got %f", expected, effectiveness)
	}
}
