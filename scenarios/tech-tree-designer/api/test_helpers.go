package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

// setupTestLogger initializes logging for tests
func setupTestLogger() func() {
	gin.SetMode(gin.TestMode)
	return func() {
		gin.SetMode(gin.ReleaseMode)
	}
}

func execWithBackoff(t *testing.T, db *sql.DB, query string, args ...interface{}) error {
	const maxRetries = 5
	baseDelay := 50 * time.Millisecond
	maxDelay := 500 * time.Millisecond
	randSource := rand.New(rand.NewSource(time.Now().UnixNano()))

	var execErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if _, execErr = db.Exec(query, args...); execErr == nil {
			return nil
		}

		if attempt == maxRetries-1 {
			break
		}

		delay := time.Duration(math.Min(float64(baseDelay)*math.Pow(2, float64(attempt)), float64(maxDelay)))
		jitterRange := float64(delay) * 0.25
		jitter := time.Duration(randSource.Float64() * jitterRange)
		time.Sleep(delay + jitter)
	}

	return execErr
}

// setupTestDB creates an in-memory test database
func setupTestDB(t *testing.T) (*sql.DB, func()) {
	// Create mock database connection
	testDB, err := sql.Open("postgres", "postgres://postgres:postgres@localhost:5432/vrooli_test?sslmode=disable")
	if err != nil {
		t.Skipf("Skipping test: database not available: %v", err)
		return nil, func() {}
	}

	// Configure connection pool to avoid exhausting resources during tests
	testDB.SetMaxOpenConns(5)
	testDB.SetMaxIdleConns(5)
	testDB.SetConnMaxLifetime(2 * time.Minute)

	// Test connection with exponential backoff and jitter
	maxRetries := 5
	baseDelay := 200 * time.Millisecond
	maxDelay := 3 * time.Second
	randSource := rand.New(rand.NewSource(time.Now().UnixNano()))

	var pingErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		pingErr = testDB.Ping()
		if pingErr == nil {
			break
		}

		if attempt == maxRetries-1 {
			t.Skipf("Skipping test: cannot connect to test database after retries: %v", pingErr)
			testDB.Close()
			return nil, func() {}
		}

		delay := time.Duration(math.Min(float64(baseDelay)*math.Pow(2, float64(attempt)), float64(maxDelay)))
		jitterRange := float64(delay) * 0.25
		jitter := time.Duration(randSource.Float64() * jitterRange)
		time.Sleep(delay + jitter)
	}

	// Store original db and replace with test db
	originalDB := db
	db = testDB

	cleanup := func() {
		// Cleanup test data
		testDB.Exec("DELETE FROM scenario_mappings")
		testDB.Exec("DELETE FROM stage_dependencies")
		testDB.Exec("DELETE FROM progression_stages")
		testDB.Exec("DELETE FROM sector_connections")
		testDB.Exec("DELETE FROM sectors")
		testDB.Exec("DELETE FROM strategic_milestones")
		testDB.Exec("DELETE FROM tech_trees")

		// Restore original db
		db = originalDB
		testDB.Close()
	}

	return testDB, cleanup
}

// setupTestRouter creates a test router with all routes configured
func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Add CORS middleware
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "tech-tree-designer"})
	})

	// API routes
	api := r.Group("/api/v1")
	{
		// Tech tree routes
		api.GET("/tech-tree", getTechTree)
		api.GET("/tech-tree/sectors", getSectors)
		api.GET("/tech-tree/sectors/:id", getSector)
		api.GET("/tech-tree/stages/:id", getStage)
		api.PUT("/tech-tree/graph", updateGraph)

		// Progress tracking routes
		api.GET("/progress/scenarios", getScenarioMappings)
		api.POST("/progress/scenarios", updateScenarioMapping)
		api.PUT("/progress/scenarios/:scenario", updateScenarioStatus)

		// Strategic analysis routes
		api.POST("/tech-tree/analyze", analyzeStrategicPath)
		api.GET("/milestones", getStrategicMilestones)
		api.GET("/recommendations", getRecommendations)

		// Dependencies and connections
		api.GET("/dependencies", getDependencies)
		api.GET("/connections", getCrossSectorConnections)
	}

	return r
}

// makeHTTPRequest creates and executes an HTTP request for testing
func makeHTTPRequest(t *testing.T, router *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("Failed to marshal request body: %v", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, path, reqBody)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	return w
}

// assertJSONResponse validates a successful JSON response
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) map[string]interface{} {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v. Body: %s", err, w.Body.String())
	}

	return response
}

// assertErrorResponse validates an error response
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, errorContains string) {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d", expectedStatus, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal error response: %v", err)
	}

	if errorMsg, ok := response["error"].(string); ok {
		if errorContains != "" && !contains(errorMsg, errorContains) {
			t.Errorf("Expected error to contain '%s', got '%s'", errorContains, errorMsg)
		}
	} else {
		t.Error("Response does not contain error field")
	}
}

// contains is a helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		bytes.Contains([]byte(s), []byte(substr)))
}

// createTestTechTree creates a test tech tree in the database
func createTestTechTree(t *testing.T, db *sql.DB) string {
	treeID := "00000000-0000-0000-0000-000000000001"
	if err := execWithBackoff(t, db, `
		INSERT INTO tech_trees (id, name, description, version, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		ON CONFLICT (id) DO UPDATE SET is_active = true
	`, treeID, "Test Tech Tree", "Test Description", "1.0.0", true); err != nil {
		t.Fatalf("Failed to create test tech tree: %v", err)
	}

	return treeID
}

// createTestSector creates a test sector in the database
func createTestSector(t *testing.T, db *sql.DB, treeID string) string {
	sectorID := "00000000-0000-0000-0000-000000000002"
	if err := execWithBackoff(t, db, `
		INSERT INTO sectors (id, tree_id, name, category, description, progress_percentage,
			position_x, position_y, color, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		ON CONFLICT (id) DO UPDATE SET name = $3
	`, sectorID, treeID, "Software Engineering", "software", "Core software capabilities",
		45.5, 100.0, 200.0, "#3498db"); err != nil {
		t.Fatalf("Failed to create test sector: %v", err)
	}

	return sectorID
}

// createTestStage creates a test progression stage in the database
func createTestStage(t *testing.T, db *sql.DB, sectorID string) string {
	stageID := "00000000-0000-0000-0000-000000000003"
	examples := `["Example 1", "Example 2"]`
	if err := execWithBackoff(t, db, `
		INSERT INTO progression_stages (id, sector_id, stage_type, stage_order, name, description,
			progress_percentage, examples, position_x, position_y, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
		ON CONFLICT (id) DO UPDATE SET name = $5
	`, stageID, sectorID, "foundation", 1, "Foundation Stage", "Core foundation",
		30.0, examples, 150.0, 250.0); err != nil {
		t.Fatalf("Failed to create test stage: %v", err)
	}

	return stageID
}

// createTestScenarioMapping creates a test scenario mapping in the database
func createTestScenarioMapping(t *testing.T, db *sql.DB, stageID, scenarioName string) string {
	mappingID := fmt.Sprintf("00000000-0000-0000-0000-0000000000%02d", 10)
	if err := execWithBackoff(t, db, `
		INSERT INTO scenario_mappings (id, scenario_name, stage_id, contribution_weight,
			completion_status, priority, estimated_impact, last_status_check, notes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), $8, NOW(), NOW())
		ON CONFLICT (scenario_name, stage_id) DO UPDATE SET completion_status = $5
	`, mappingID, scenarioName, stageID, 0.8, "in_progress", 1, 7.5, "Test scenario"); err != nil {
		t.Fatalf("Failed to create test scenario mapping: %v", err)
	}

	return mappingID
}

// createTestMilestone creates a test strategic milestone in the database
func createTestMilestone(t *testing.T, db *sql.DB, treeID string) string {
	milestoneID := "00000000-0000-0000-0000-000000000005"
	requiredSectors := `["sector1", "sector2"]`
	requiredStages := `["stage1", "stage2"]`
	if err := execWithBackoff(t, db, `
		INSERT INTO strategic_milestones (id, tree_id, name, description, milestone_type,
			required_sectors, required_stages, completion_percentage, confidence_level,
			business_value_estimate, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
		ON CONFLICT (id) DO UPDATE SET name = $3
	`, milestoneID, treeID, "Test Milestone", "Test milestone description",
		"capability", requiredSectors, requiredStages, 45.0, 0.75, 50000); err != nil {
		t.Fatalf("Failed to create test milestone: %v", err)
	}

	return milestoneID
}

// createTestDependency creates a test stage dependency in the database
func createTestDependency(t *testing.T, db *sql.DB, dependentID, prerequisiteID string) string {
	depID := "00000000-0000-0000-0000-000000000006"
	if err := execWithBackoff(t, db, `
		INSERT INTO stage_dependencies (id, dependent_stage_id, prerequisite_stage_id,
			dependency_type, dependency_strength, description, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (id) DO UPDATE SET dependency_type = $4
	`, depID, dependentID, prerequisiteID, "requires", 0.9, "Test dependency"); err != nil {
		t.Fatalf("Failed to create test dependency: %v", err)
	}

	return depID
}

// createTestSectorConnection creates a test sector connection in the database
func createTestSectorConnection(t *testing.T, db *sql.DB, sourceID, targetID string) string {
	connID := "00000000-0000-0000-0000-000000000007"
	examples := `["Example connection"]`
	if err := execWithBackoff(t, db, `
		INSERT INTO sector_connections (id, source_sector_id, target_sector_id,
			connection_type, strength, description, examples, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		ON CONFLICT (id) DO UPDATE SET strength = $5
	`, connID, sourceID, targetID, "enables", 0.85, "Test connection", examples); err != nil {
		t.Fatalf("Failed to create test sector connection: %v", err)
	}

	return connID
}

// cleanupTestData removes all test data from the database
func cleanupTestData(db *sql.DB) {
	db.Exec("DELETE FROM scenario_mappings")
	db.Exec("DELETE FROM stage_dependencies")
	db.Exec("DELETE FROM sector_connections")
	db.Exec("DELETE FROM strategic_milestones")
	db.Exec("DELETE FROM progression_stages")
	db.Exec("DELETE FROM sectors")
	db.Exec("DELETE FROM tech_trees")
}

// assertArrayLength checks that an array field has the expected length
func assertArrayLength(t *testing.T, response map[string]interface{}, field string, expectedMin int) []interface{} {
	arr, ok := response[field].([]interface{})
	if !ok {
		t.Errorf("Expected field '%s' to be an array, got %T", field, response[field])
		return nil
	}

	if len(arr) < expectedMin {
		t.Errorf("Expected at least %d items in '%s', got %d", expectedMin, field, len(arr))
	}

	return arr
}

// assertFieldExists checks that a field exists in the response
func assertFieldExists(t *testing.T, response map[string]interface{}, field string) {
	if _, ok := response[field]; !ok {
		t.Errorf("Expected field '%s' in response", field)
	}
}
