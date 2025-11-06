//go:build testing
// +build testing

package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalLogger *log.Logger
	cleanup        func()
}

// setupTestLogger initializes logging for testing
func setupTestLogger() func() {
	// Redirect to dev/null for cleaner test output
	log.SetOutput(os.Discard)
	return func() {
		log.SetOutput(os.Stderr)
	}
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	TempDir    string
	OriginalWD string
	DB         *sql.DB
	Cleanup    func()
}

// setupTestDirectory creates an isolated test environment with proper cleanup
func setupTestDirectory(t *testing.T) *TestEnvironment {
	tempDir, err := os.MkdirTemp("", "saas-landing-test")
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
	if err := os.MkdirAll(scenariosDir, 0755); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create scenarios dir: %v", err)
	}

	// Setup in-memory database for testing
	testDB := setupTestDatabase(t)

	return &TestEnvironment{
		TempDir:    tempDir,
		OriginalWD: originalWD,
		DB:         testDB,
		Cleanup: func() {
			if testDB != nil {
				testDB.Close()
			}
			os.RemoveAll(tempDir)
		},
	}
}

// setupTestDatabase creates an in-memory test database
func setupTestDatabase(t *testing.T) *sql.DB {
	// For unit tests, we'll use a real PostgreSQL connection if available,
	// otherwise skip database-dependent tests
	postgresURL := os.Getenv("TEST_POSTGRES_URL")
	if postgresURL == "" {
		// Try to use a test database
		postgresURL = "postgres://postgres:postgres@localhost:5432/saas_landing_test?sslmode=disable"
	}

	testDB, err := sql.Open("postgres", postgresURL)
	if err != nil {
		t.Skipf("Skipping database tests: %v", err)
		return nil
	}

	// Ping to verify connection
	if err := testDB.Ping(); err != nil {
		testDB.Close()
		t.Skipf("Skipping database tests: database not available: %v", err)
		return nil
	}

	// Create test schema
	if err := initTestSchema(testDB); err != nil {
		testDB.Close()
		t.Fatalf("Failed to initialize test schema: %v", err)
	}

	return testDB
}

// initTestSchema creates the test database schema
func initTestSchema(db *sql.DB) error {
	schema := `
		DROP TABLE IF EXISTS ab_test_results CASCADE;
		DROP TABLE IF EXISTS landing_pages CASCADE;
		DROP TABLE IF EXISTS templates CASCADE;
		DROP TABLE IF EXISTS saas_scenarios CASCADE;

		CREATE TABLE IF NOT EXISTS saas_scenarios (
			id UUID PRIMARY KEY,
			scenario_name VARCHAR(255) UNIQUE NOT NULL,
			display_name VARCHAR(255),
			description TEXT,
			saas_type VARCHAR(100),
			industry VARCHAR(100),
			revenue_potential VARCHAR(100),
			has_landing_page BOOLEAN DEFAULT FALSE,
			landing_page_url VARCHAR(500),
			last_scan TIMESTAMP,
			confidence_score DECIMAL(3,2),
			metadata JSONB
		);

		CREATE TABLE IF NOT EXISTS templates (
			id UUID PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			category VARCHAR(100),
			saas_type VARCHAR(100),
			industry VARCHAR(100),
			html_content TEXT,
			css_content TEXT,
			js_content TEXT,
			config_schema JSONB,
			preview_url VARCHAR(500),
			usage_count INTEGER DEFAULT 0,
			rating DECIMAL(3,2),
			created_at TIMESTAMP DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS landing_pages (
			id UUID PRIMARY KEY,
			scenario_id UUID REFERENCES saas_scenarios(id),
			template_id UUID REFERENCES templates(id),
			variant VARCHAR(50),
			title VARCHAR(255),
			description TEXT,
			content JSONB,
			seo_metadata JSONB,
			performance_metrics JSONB,
			status VARCHAR(50),
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS ab_test_results (
			id UUID PRIMARY KEY,
			landing_page_id UUID REFERENCES landing_pages(id),
			variant VARCHAR(50),
			metric_name VARCHAR(100),
			metric_value DECIMAL(10,2),
			timestamp TIMESTAMP DEFAULT NOW(),
			session_id VARCHAR(255),
			user_agent TEXT
		);
	`

	_, err := db.Exec(schema)
	return err
}

// HTTPTestRequest represents a test HTTP request
type HTTPTestRequest struct {
	Method  string
	Path    string
	Body    interface{}
	URLVars map[string]string
	Headers map[string]string
}

// makeHTTPRequest creates and executes an HTTP test request
func makeHTTPRequest(req HTTPTestRequest) (*httptest.ResponseRecorder, error) {
	var bodyReader *bytes.Buffer
	if req.Body != nil {
		if bodyStr, ok := req.Body.(string); ok {
			bodyReader = bytes.NewBufferString(bodyStr)
		} else {
			bodyJSON, err := json.Marshal(req.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal body: %w", err)
			}
			bodyReader = bytes.NewBuffer(bodyJSON)
		}
	} else {
		bodyReader = bytes.NewBuffer([]byte{})
	}

	httpReq, err := http.NewRequest(req.Method, req.Path, bodyReader)
	if err != nil {
		return nil, err
	}

	// Set default content type
	httpReq.Header.Set("Content-Type", "application/json")

	// Set custom headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Set URL vars if provided
	if req.URLVars != nil {
		httpReq = mux.SetURLVars(httpReq, req.URLVars)
	}

	w := httptest.NewRecorder()
	return w, nil
}

// assertJSONResponse validates a JSON response
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedFields map[string]interface{}) map[string]interface{} {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
		return nil
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse JSON response: %v. Body: %s", err, w.Body.String())
		return nil
	}

	for key, expectedValue := range expectedFields {
		actualValue, exists := response[key]
		if !exists {
			t.Errorf("Expected field '%s' not found in response", key)
			continue
		}

		// Type-aware comparison
		if expectedValue != nil && actualValue != expectedValue {
			// For strings, do direct comparison
			if expectedStr, ok := expectedValue.(string); ok {
				if actualStr, ok := actualValue.(string); ok {
					if expectedStr != actualStr {
						t.Errorf("Field '%s': expected '%v', got '%v'", key, expectedValue, actualValue)
					}
				}
			}
		}
	}

	return response
}

// assertErrorResponse validates an error response
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectErrorMessage bool) {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
		return
	}

	if expectErrorMessage && w.Body.Len() == 0 {
		t.Error("Expected error message in response body, got empty body")
	}
}

// createTestScenario creates a test SaaS scenario in the database
func createTestScenario(t *testing.T, db *sql.DB, name string) *SaaSScenario {
	scenario := &SaaSScenario{
		ID:               uuid.New().String(),
		ScenarioName:     name,
		DisplayName:      fmt.Sprintf("Test %s", name),
		Description:      fmt.Sprintf("Test scenario: %s", name),
		SaaSType:         "b2b_tool",
		Industry:         "technology",
		RevenuePotential: "$10K-$50K",
		HasLandingPage:   false,
		LandingPageURL:   "",
		LastScan:         time.Now(),
		ConfidenceScore:  0.8,
		Metadata:         map[string]interface{}{"test": true},
	}

	dbService := NewDatabaseService(db)
	if err := dbService.CreateSaaSScenario(scenario); err != nil {
		t.Fatalf("Failed to create test scenario: %v", err)
	}

	return scenario
}

// createTestTemplate creates a test template in the database
func createTestTemplate(t *testing.T, db *sql.DB, name string) *Template {
	template := &Template{
		ID:          uuid.New().String(),
		Name:        name,
		Category:    "modern",
		SaaSType:    "b2b_tool",
		Industry:    "technology",
		HTMLContent: "<html><body>Test</body></html>",
		CSSContent:  "body { margin: 0; }",
		JSContent:   "console.log('test');",
		ConfigSchema: map[string]interface{}{
			"title":       "string",
			"description": "string",
		},
		PreviewURL: fmt.Sprintf("/preview/%s", name),
		UsageCount: 0,
		Rating:     4.5,
		CreatedAt:  time.Now(),
	}

	query := `
		INSERT INTO templates (id, name, category, saas_type, industry, html_content,
			css_content, js_content, config_schema, preview_url, usage_count, rating, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	configJSON, _ := json.Marshal(template.ConfigSchema)
	_, err := db.Exec(query, template.ID, template.Name, template.Category, template.SaaSType,
		template.Industry, template.HTMLContent, template.CSSContent, template.JSContent,
		string(configJSON), template.PreviewURL, template.UsageCount, template.Rating, template.CreatedAt)

	if err != nil {
		t.Fatalf("Failed to create test template: %v", err)
	}

	return template
}

// createTestLandingPage creates a test landing page in the database
func createTestLandingPage(t *testing.T, db *sql.DB, scenarioID, templateID string) *LandingPage {
	page := &LandingPage{
		ID:          uuid.New().String(),
		ScenarioID:  scenarioID,
		TemplateID:  templateID,
		Variant:     "control",
		Title:       "Test Landing Page",
		Description: "Test landing page description",
		Content: map[string]interface{}{
			"hero": "Welcome to our service",
		},
		SEOMetadata: map[string]interface{}{
			"title": "Test SEO Title",
		},
		PerformanceMetrics: map[string]interface{}{
			"views": 0,
		},
		Status:    "draft",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	dbService := NewDatabaseService(db)
	if err := dbService.CreateLandingPage(page); err != nil {
		t.Fatalf("Failed to create test landing page: %v", err)
	}

	return page
}

// createTestScenariosDirectory creates a test scenarios directory with sample scenarios
func createTestScenariosDirectory(t *testing.T, tempDir string) string {
	scenariosDir := filepath.Join(tempDir, "scenarios")

	// Create a sample SaaS scenario
	saasScenarioDir := filepath.Join(scenariosDir, "test-saas-app")
	if err := os.MkdirAll(filepath.Join(saasScenarioDir, ".vrooli"), 0755); err != nil {
		t.Fatalf("Failed to create scenario dir: %v", err)
	}

	// Create service.json
	serviceConfig := map[string]interface{}{
		"service": map[string]interface{}{
			"displayName": "Test SaaS App",
			"description": "A test SaaS application",
			"tags":        []string{"saas", "business-application", "multi-tenant"},
		},
		"resources": map[string]interface{}{
			"postgres": map[string]interface{}{
				"enabled": true,
			},
		},
	}

	serviceJSON, _ := json.MarshalIndent(serviceConfig, "", "  ")
	serviceFile := filepath.Join(saasScenarioDir, ".vrooli", "service.json")
	if err := os.WriteFile(serviceFile, serviceJSON, 0644); err != nil {
		t.Fatalf("Failed to write service.json: %v", err)
	}

	// Create PRD.md
	prdContent := `# Test SaaS App

## Business Value
Revenue Potential: $10K-$50K

This is a multi-tenant SaaS application with subscription pricing.
`
	prdFile := filepath.Join(saasScenarioDir, "PRD.md")
	if err := os.WriteFile(prdFile, []byte(prdContent), 0644); err != nil {
		t.Fatalf("Failed to write PRD.md: %v", err)
	}

	// Create ui and api directories
	os.MkdirAll(filepath.Join(saasScenarioDir, "ui"), 0755)
	os.MkdirAll(filepath.Join(saasScenarioDir, "api"), 0755)

	// Create a non-SaaS scenario for comparison
	nonSaasDir := filepath.Join(scenariosDir, "simple-script")
	if err := os.MkdirAll(nonSaasDir, 0755); err != nil {
		t.Fatalf("Failed to create non-SaaS dir: %v", err)
	}

	return scenariosDir
}
