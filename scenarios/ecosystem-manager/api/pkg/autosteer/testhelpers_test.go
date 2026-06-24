package autosteer

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ecosystem-manager/api/pkg/internal/testdb"
)

// testDB holds a temp SQLite database with the controller's schema applied.
type testDB struct {
	db *sql.DB
}

// SetupTestDatabase opens a temp SQLite database with the controller's tables
// applied. The second return value is a no-op cleanup retained for caller
// symmetry — t.Cleanup already closes the DB. It never returns nil, so callers'
// Docker-skip guards are simply dead under SQLite.
func SetupTestDatabase(t *testing.T) (*testDB, func()) {
	t.Helper()
	db := testdb.NewSQLite(t, Schema())
	return &testDB{db: db}, func() {}
}

// CreateTestProfile creates a simple objective-function test profile. The mode
// becomes the single allowed skill; the objective weights one valid dimension.
func CreateTestProfile(t *testing.T, name string, mode SteerMode, maxIterations int) *AutoSteerProfile {
	t.Helper()

	return &AutoSteerProfile{
		Name:        name,
		Description: fmt.Sprintf("Test profile for %s mode", mode),
		Objective: Objective{
			DimensionWeights: map[string]float64{"standards": 1.0},
			Targets:          ObjectiveTargets{MaxOpenSeverity: "warning"},
		},
		AllowedSkills: []string{string(mode)},
		Budget: Budget{
			MaxIterations:           maxIterations,
			DiminishingReturnsFloor: 0.02,
			ReauditCadence:          0,
		},
		AuditPreset: "comprehensive",
		Tags:        []string{"test"},
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

	if err := os.MkdirAll(requirementsDir, 0o755); err != nil {
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
	if err := os.WriteFile(requirementsPath, []byte(requirementsJSON), 0o644); err != nil {
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

	if err := os.WriteFile(requirementsPath, []byte(requirementsJSON), 0o644); err != nil {
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
