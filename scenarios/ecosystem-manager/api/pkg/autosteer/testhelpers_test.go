package autosteer

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// PostgresContainer wraps a testcontainers postgres instance
type PostgresContainer struct {
	container *postgres.PostgresContainer
	db        *sql.DB
	connStr   string
}

// SetupTestDatabase creates a PostgreSQL testcontainer and returns a connection
func SetupTestDatabase(t *testing.T) (*PostgresContainer, func()) {
	t.Helper()

	ctx := context.Background()

	// Create PostgreSQL container
	pgContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:15-alpine"),
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		t.Skipf("Could not start postgres container: %v (Docker may not be available)", err)
		return nil, nil
	}

	// Get connection string
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		pgContainer.Terminate(ctx)
		t.Fatalf("Failed to get connection string: %v", err)
	}

	// Connect to database
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		pgContainer.Terminate(ctx)
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Wait for database to be ready
	if err := db.Ping(); err != nil {
		db.Close()
		pgContainer.Terminate(ctx)
		t.Fatalf("Failed to ping database: %v", err)
	}

	// Create schema
	if err := createTestSchema(db); err != nil {
		db.Close()
		pgContainer.Terminate(ctx)
		t.Fatalf("Failed to create schema: %v", err)
	}

	container := &PostgresContainer{
		container: pgContainer,
		db:        db,
		connStr:   connStr,
	}

	cleanup := func() {
		db.Close()
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate postgres container: %v", err)
		}
	}

	return container, cleanup
}

// createTestSchema creates the necessary database tables
func createTestSchema(db *sql.DB) error {
	schema := `
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
			user_rating INTEGER CHECK (user_rating >= 1 AND user_rating <= 5),
			user_comments TEXT,
			user_feedback_at TIMESTAMP,
			executed_at TIMESTAMP DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_profile_executions_profile_id ON profile_executions(profile_id);
		CREATE INDEX IF NOT EXISTS idx_profile_executions_scenario ON profile_executions(scenario_name);
		CREATE INDEX IF NOT EXISTS idx_profile_executions_executed_at ON profile_executions(executed_at DESC);
	`

	_, err := db.Exec(schema)
	return err
}

// CreateTestProfile creates a simple test profile
func CreateTestProfile(t *testing.T, name string, mode SteerMode, maxIterations int) *AutoSteerProfile {
	t.Helper()

	return &AutoSteerProfile{
		Name:        name,
		Description: fmt.Sprintf("Test profile for %s mode", mode),
		Phases: []SteerPhase{
			{
				ID:            uuid.New().String(),
				Mode:          mode,
				MaxIterations: maxIterations,
				StopConditions: []StopCondition{
					{
						Type:            ConditionTypeSimple,
						Metric:          "loops",
						CompareOperator: OpGreaterThan,
						Value:           float64(maxIterations - 2),
					},
				},
			},
		},
		Tags: []string{"test"},
	}
}

// CreateMultiPhaseProfile creates a profile with multiple phases
func CreateMultiPhaseProfile(t *testing.T, name string, phases []SteerPhase) *AutoSteerProfile {
	t.Helper()

	return &AutoSteerProfile{
		Name:        name,
		Description: "Multi-phase test profile",
		Phases:      phases,
		Tags:        []string{"test", "multi-phase"},
	}
}

// SetupTestScenario creates a temporary scenario directory structure for testing
func SetupTestScenario(t *testing.T, scenarioName string) (vrooliRoot string, cleanup func()) {
	t.Helper()

	// Create temporary Vrooli root
	tmpDir, err := os.MkdirTemp("", "autosteer-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create scenario structure
	scenarioDir := filepath.Join(tmpDir, "scenarios", scenarioName)
	requirementsDir := filepath.Join(scenarioDir, "requirements")

	if err := os.MkdirAll(requirementsDir, 0755); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create scenario dir: %v", err)
	}

	// Create default requirements file
	requirementsJSON := `{
		"modules": [{
			"id": "module-1",
			"operationalTargets": [
				{"id": "target-1", "status": "passing"},
				{"id": "target-2", "status": "passing"}
			]
		}]
	}`

	requirementsPath := filepath.Join(requirementsDir, "index.json")
	if err := os.WriteFile(requirementsPath, []byte(requirementsJSON), 0644); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to write requirements file: %v", err)
	}

	cleanup = func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

// UpdateOperationalTargets updates the operational targets in a test scenario
func UpdateOperationalTargets(t *testing.T, vrooliRoot, scenarioName string, total, passing int) {
	t.Helper()

	requirementsPath := filepath.Join(vrooliRoot, "scenarios", scenarioName, "requirements", "index.json")

	// Build targets
	targets := make([]string, total)
	for i := 0; i < total; i++ {
		status := "failing"
		if i < passing {
			status = "passing"
		}
		targets[i] = fmt.Sprintf(`{"id": "target-%d", "status": "%s"}`, i+1, status)
	}

	requirementsJSON := fmt.Sprintf(`{
		"modules": [{
			"id": "module-1",
			"operationalTargets": [%s]
		}]
	}`, joinStrings(targets, ", "))

	if err := os.WriteFile(requirementsPath, []byte(requirementsJSON), 0644); err != nil {
		t.Fatalf("Failed to update requirements: %v", err)
	}
}

func joinStrings(strs []string, sep string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}

// AssertMetricsEqual checks if two metric snapshots are equal (within tolerance for floats)
func AssertMetricsEqual(t *testing.T, expected, actual MetricsSnapshot, tolerance float64) {
	t.Helper()

	if abs(expected.OperationalTargetsPercentage-actual.OperationalTargetsPercentage) > tolerance {
		t.Errorf("Operational targets percentage mismatch: expected %.2f, got %.2f",
			expected.OperationalTargetsPercentage, actual.OperationalTargetsPercentage)
	}

	if expected.BuildStatus != actual.BuildStatus {
		t.Errorf("Build status mismatch: expected %d, got %d", expected.BuildStatus, actual.BuildStatus)
	}

	// Check UX metrics if present
	if expected.UX != nil && actual.UX != nil {
		if abs(expected.UX.AccessibilityScore-actual.UX.AccessibilityScore) > tolerance {
			t.Errorf("Accessibility score mismatch: expected %.2f, got %.2f",
				expected.UX.AccessibilityScore, actual.UX.AccessibilityScore)
		}
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// testPhasePromptsDir returns a directory containing phase prompt markdown files for tests.
func testPhasePromptsDir(t *testing.T) string {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("unable to determine caller location for prompt discovery")
	}

	// Prefer real prompts if available
	dir := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", "..", "prompts", "phases"))
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("phase prompts directory not found at %s: %v", dir, err)
	}

	return dir
}

// WaitForCondition polls until a condition is met or timeout
func WaitForCondition(t *testing.T, timeout time.Duration, condition func() bool) bool {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}
