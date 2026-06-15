package app

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	apidb "github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/health"
	_ "modernc.org/sqlite"
	analysisapi "scenario-dependency-analyzer/internal/analysis"
	dependenciesapi "scenario-dependency-analyzer/internal/dependencies"
	graphdomain "scenario-dependency-analyzer/internal/graph"
	"scenario-dependency-analyzer/internal/modules"
	proposalapi "scenario-dependency-analyzer/internal/proposal"
	types "scenario-dependency-analyzer/internal/types"
)

func ensureTestEnvVars() {
	if os.Getenv("API_PORT") == "" {
		os.Setenv("API_PORT", "0")
	}
	if os.Getenv("SQLITE_PATH") == "" && os.Getenv("SQLITE_DB") == "" {
		os.Setenv("SQLITE_DB", ":memory:")
	}
}

// setupTestLogger initializes logging for testing with output suppression
func setupTestLogger() func() {
	// Suppress Gin debug output during tests
	gin.SetMode(gin.TestMode)

	// Redirect log output to discard during tests (can be enabled for debugging)
	originalOutput := log.Writer()
	log.SetOutput(io.Discard)

	return func() {
		log.SetOutput(originalOutput)
	}
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	TempDir      string
	OriginalWD   string
	Router       *gin.Engine
	TestDB       *sql.DB
	ScenariosDir string
	Cleanup      func()
}

type mockGraphService struct{}

func (mockGraphService) GenerateGraph(graphType string) (*types.DependencyGraph, error) {
	return &types.DependencyGraph{
		Type: graphType,
		Nodes: []types.GraphNode{
			{ID: "core-a", Type: "scenario", Group: "scenarios"},
			{ID: "consumer", Type: "scenario", Group: "scenarios"},
		},
		Edges: []types.GraphEdge{
			{Source: "consumer", Target: "core-a", Type: "scenario", Required: true, Weight: 2},
		},
	}, nil
}

func (mockGraphService) GraphCentrality(coreSeeds []string, scenario string) (*types.GraphCentralityReport, error) {
	graph, err := mockGraphService{}.GenerateGraph("combined")
	if err != nil {
		return nil, err
	}
	return graphdomain.CalculateCentrality(graph, coreSeeds, scenario), nil
}

// setupTestDirectory creates an isolated test environment with proper cleanup
func setupTestDirectory(t *testing.T) *TestEnvironment {
	ensureTestEnvVars()
	tempDir, err := os.MkdirTemp("", "scenario-dependency-analyzer-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Create test scenarios directory structure
	scenariosDir := filepath.Join(tempDir, "scenarios")
	if err := os.MkdirAll(scenariosDir, 0o750); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create scenarios dir: %v", err)
	}

	return &TestEnvironment{
		TempDir:      tempDir,
		OriginalWD:   originalWD,
		ScenariosDir: scenariosDir,
		Cleanup: func() {
			_ = os.Chdir(originalWD)
			os.RemoveAll(tempDir)
		},
	}
}

// setupTestDatabase creates an in-memory test database
func setupTestDatabase(t *testing.T) (*sql.DB, func()) {
	ensureTestEnvVars()
	testDB, err := sql.Open("sqlite", "file::memory:?cache=shared&_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("open test sqlite database: %v", err)
	}
	if err := testDB.Ping(); err != nil {
		testDB.Close()
		t.Fatalf("ping test sqlite database: %v", err)
	}
	if err := apidb.EnsureSchemas(context.Background(), testDB, modules.AllSchemas()...); err != nil {
		testDB.Close()
		t.Fatalf("initialize test schema: %v", err)
	}
	db = testDB
	setDefaultRuntime(NewRuntime(loadConfig(), testDB))

	cleanup := func() {
		if defaultRuntime != nil && defaultRuntime.DB() == testDB {
			setDefaultRuntime(nil)
		}
		if db == testDB {
			db = nil
		}
		testDB.Close()
	}

	return testDB, cleanup
}

// TestScenario represents a test scenario structure
type TestScenario struct {
	Name        string
	ServiceJSON map[string]interface{}
	Files       map[string]string
	Cleanup     func()
}

// createTestScenario creates a test scenario with service.json and optional files
func createTestScenario(t *testing.T, env *TestEnvironment, name string, resources map[string]types.Resource) *TestScenario {
	scenarioPath := filepath.Join(env.ScenariosDir, name)
	if err := os.MkdirAll(scenarioPath, 0o750); err != nil {
		t.Fatalf("Failed to create scenario dir: %v", err)
	}

	// Create .vrooli directory
	vrooliPath := filepath.Join(scenarioPath, ".vrooli")
	if err := os.MkdirAll(vrooliPath, 0o750); err != nil {
		t.Fatalf("Failed to create .vrooli dir: %v", err)
	}

	// Create service.json
	serviceConfig := types.ServiceConfig{
		Schema:  "https://schemas.vrooli.com/service/v2.0.0.json",
		Version: "2.0.0",
		Service: struct {
			Name        string   `json:"name"`
			DisplayName string   `json:"display_name"`
			Description string   `json:"description"`
			Version     string   `json:"version"`
			Tags        []string `json:"tags"`
		}{
			Name:        name,
			DisplayName: "Test " + name,
			Description: "Test scenario for " + name,
			Version:     "1.0.0",
			Tags:        []string{"test"},
		},
		Resources: resources,
	}

	serviceJSON, err := json.MarshalIndent(serviceConfig, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal service.json: %v", err)
	}

	serviceJSONPath := filepath.Join(vrooliPath, "service.json")
	if err := os.WriteFile(serviceJSONPath, serviceJSON, 0o600); err != nil {
		t.Fatalf("Failed to write service.json: %v", err)
	}

	return &TestScenario{
		Name: name,
		ServiceJSON: map[string]interface{}{
			"resources": resources,
		},
		Files: map[string]string{
			"service.json": serviceJSONPath,
		},
		Cleanup: func() {
			os.RemoveAll(scenarioPath)
		},
	}
}

func configureTestScenariosDir(t *testing.T, env *TestEnvironment) {
	t.Helper()
	if err := os.Setenv("VROOLI_SCENARIOS_DIR", env.ScenariosDir); err != nil {
		t.Fatalf("Failed to set scenarios dir env: %v", err)
	}
	refreshDependencyCatalogs()
}

func setEnvAndCleanup(t *testing.T, key, value string) {
	t.Helper()
	original, existed := os.LookupEnv(key)
	if err := os.Setenv(key, value); err != nil {
		t.Fatalf("failed to set env %s: %v", key, err)
	}
	t.Cleanup(func() {
		if !existed {
			os.Unsetenv(key)
			return
		}
		os.Setenv(key, original)
	})
}

func createTestResourceDirs(t *testing.T, env *TestEnvironment, names ...string) {
	t.Helper()
	base := filepath.Join(filepath.Dir(env.ScenariosDir), "resources")
	for _, name := range names {
		path := filepath.Join(base, name)
		if err := os.MkdirAll(path, 0o750); err != nil {
			t.Fatalf("Failed to create resource dir %s: %v", name, err)
		}
	}
	refreshDependencyCatalogs()
}

// makeHTTPRequest creates and sends an HTTP request for testing
func makeHTTPRequest(t *testing.T, router *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	var reqBody []byte
	var err error

	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("Failed to marshal request body: %v", err)
		}
	}

	req, err := http.NewRequest(method, path, bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	return recorder
}

// assertErrorResponse validates error response
func assertErrorResponse(t *testing.T, recorder *httptest.ResponseRecorder, expectedStatus int, errorMessageContains string) {
	if recorder.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d", expectedStatus, recorder.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal error response: %v", err)
	}

	errorMsg, exists := response["error"]
	if !exists {
		t.Errorf("Expected error field in response, got: %v", response)
		return
	}

	if errorStr, ok := errorMsg.(string); ok {
		if !strings.Contains(errorStr, errorMessageContains) {
			t.Errorf("Expected error to contain %q, got %q", errorMessageContains, errorStr)
		}
	}
}

// setupTestRouter creates a configured Gin router for testing
func setupTestRouter() *gin.Engine {
	ensureTestEnvVars()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := newHandler(nil)
	h.services.Graph = mockGraphService{}

	// Add test routes - use same health handler as server.go
	router.GET("/health", gin.WrapF(health.New("scenario-dependency-analyzer-api").Handler()))
	router.GET("/api/v1/health/analysis", h.analysisHealth)

	api := router.Group("/api/v1")
	{
		analysisapi.RegisterHTTPRoutes(api, h.analysisService(), h.scanService())
		dependenciesapi.RegisterHTTPRoutes(api, h.dependencyService())
		graphdomain.RegisterHTTPRoutes(api, h.graphService(), h.scenariosDir)
		proposalapi.RegisterHTTPRoutes(api, h.proposalService())
	}

	return router
}

// createTestDependency creates a test dependency record
func createTestDependency(scenarioName, depType, depName string, required bool) types.ScenarioDependency {
	return types.ScenarioDependency{
		ID:             uuid.New().String(),
		ScenarioName:   scenarioName,
		DependencyType: depType,
		DependencyName: depName,
		Required:       required,
		Purpose:        fmt.Sprintf("Test purpose for %s", depName),
		AccessMethod:   "direct",
		Configuration:  map[string]interface{}{"test": true},
		DiscoveredAt:   time.Now(),
		LastVerified:   time.Now(),
	}
}

// insertTestDependency inserts a test dependency into the database
func insertTestDependency(t *testing.T, testDB *sql.DB, dep types.ScenarioDependency) {
	configJSON, _ := json.Marshal(dep.Configuration)

	_, err := testDB.Exec(`
		INSERT INTO scenario_dependencies
		(id, scenario_name, dependency_type, dependency_name, required, purpose, access_method, configuration)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		dep.ID, dep.ScenarioName, dep.DependencyType, dep.DependencyName,
		dep.Required, dep.Purpose, dep.AccessMethod, string(configJSON),
	)
	if err != nil {
		t.Fatalf("Failed to insert test dependency: %v", err)
	}
}
