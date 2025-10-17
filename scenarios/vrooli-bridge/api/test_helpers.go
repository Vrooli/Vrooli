package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalOutput *os.File
	cleanup        func()
}

// setupTestLogger initializes test logging with suppression
func setupTestLogger() func() {
	// Suppress log output during tests
	log.SetOutput(ioutil.Discard)
	return func() {
		log.SetOutput(os.Stderr)
	}
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	TempDir       string
	OriginalWD    string
	DB            *sql.DB
	App           *App
	Cleanup       func()
}

// setupTestEnvironment creates an isolated test environment with database
func setupTestEnvironment(t *testing.T) *TestEnvironment {
	tempDir, err := ioutil.TempDir("", "vrooli-bridge-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Create initialization/templates directory structure
	templatesDir := filepath.Join(tempDir, "../initialization/templates")
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create templates directory: %v", err)
	}

	// Create project types JSON file
	projectTypes := map[string]ProjectType{
		"nodejs": {
			Name:                 "Node.js",
			Detection:            []string{"package.json"},
			PreferredScenarios:   []string{"code-review", "dependency-audit"},
			SpecificCapabilities: "Node.js package management",
			Commands:             []string{"npm test", "npm build"},
		},
		"python": {
			Name:                 "Python",
			Detection:            []string{"requirements.txt", "pyproject.toml"},
			PreferredScenarios:   []string{"code-review", "testing"},
			SpecificCapabilities: "Python package management",
			Commands:             []string{"pytest", "python -m unittest"},
		},
		"generic": {
			Name:                 "Generic Project",
			Detection:            []string{},
			PreferredScenarios:   []string{"code-review"},
			SpecificCapabilities: "General project support",
			Commands:             []string{},
		},
	}

	projectTypesJSON, _ := json.MarshalIndent(projectTypes, "", "  ")
	projectTypesFile := filepath.Join(templatesDir, "project-types.json")
	if err := ioutil.WriteFile(projectTypesFile, projectTypesJSON, 0644); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to write project types: %v", err)
	}

	// Create template files
	integrationTemplate := `# Vrooli Integration

Date: {{date}}
Vrooli Version: {{vrooli_version}}
Project Type: {{project_type}}
Project Name: {{project_name}}
Project Path: {{project_path}}
Bridge Version: {{bridge_version}}

## Project-Specific Capabilities
{{project_type_specific}}

## Preferred Scenarios
{{preferred_scenarios}}
`
	integrationTemplateFile := filepath.Join(templatesDir, "VROOLI_INTEGRATION.md.template")
	if err := ioutil.WriteFile(integrationTemplateFile, []byte(integrationTemplate), 0644); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to write integration template: %v", err)
	}

	claudeTemplate := `## Vrooli Integration

This project is integrated with Vrooli.

- Date: {{date}}
- Version: {{vrooli_version}}
- Project Type: {{project_type}}
- Capabilities: {{project_specific_capabilities}}
`
	claudeTemplateFile := filepath.Join(templatesDir, "CLAUDE_ADDITIONS.md.template")
	if err := ioutil.WriteFile(claudeTemplateFile, []byte(claudeTemplate), 0644); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to write claude template: %v", err)
	}

	if err := os.Chdir(tempDir); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	// Try to setup test database - if it fails, db will be nil
	var db *sql.DB
	testDB, err := sql.Open("postgres", "postgres://test:test@localhost:5432/vrooli_bridge_test?sslmode=disable")
	if err == nil {
		// Try to ping the database
		if pingErr := testDB.Ping(); pingErr == nil {
			// Database is available - create tables
			_, execErr := testDB.Exec(`
				CREATE TABLE IF NOT EXISTS projects (
					id TEXT PRIMARY KEY,
					path TEXT UNIQUE NOT NULL,
					name TEXT NOT NULL,
					type TEXT NOT NULL,
					vrooli_version TEXT,
					bridge_version TEXT,
					integration_status TEXT NOT NULL,
					last_updated TIMESTAMP NOT NULL,
					created_at TIMESTAMP NOT NULL,
					metadata JSONB
				)
			`)
			if execErr == nil {
				db = testDB
			} else {
				testDB.Close()
			}
		} else {
			testDB.Close()
		}
	}

	app := &App{
		db:           db,
		projectTypes: projectTypes,
	}

	return &TestEnvironment{
		TempDir:    tempDir,
		OriginalWD: originalWD,
		DB:         db,
		App:        app,
		Cleanup: func() {
			if db != nil {
				db.Exec("DROP TABLE IF EXISTS projects")
				db.Close()
			}
			os.Chdir(originalWD)
			os.RemoveAll(tempDir)
		},
	}
}

// TestProject provides a pre-configured project for testing
type TestProject struct {
	Project  *Project
	Path     string
	Cleanup  func()
}

// setupTestProject creates a test project with sample data
func setupTestProject(t *testing.T, env *TestEnvironment, projectType string) *TestProject {
	projectPath := filepath.Join(env.TempDir, "test-project")
	if err := os.MkdirAll(projectPath, 0755); err != nil {
		t.Fatalf("Failed to create project directory: %v", err)
	}

	// Create project indicator file
	switch projectType {
	case "nodejs":
		packageJSON := `{"name": "test-project", "version": "1.0.0"}`
		if err := ioutil.WriteFile(filepath.Join(projectPath, "package.json"), []byte(packageJSON), 0644); err != nil {
			t.Fatalf("Failed to create package.json: %v", err)
		}
	case "python":
		requirements := "pytest==7.0.0\n"
		if err := ioutil.WriteFile(filepath.Join(projectPath, "requirements.txt"), []byte(requirements), 0644); err != nil {
			t.Fatalf("Failed to create requirements.txt: %v", err)
		}
	}

	project := &Project{
		ID:                uuid.New().String(),
		Path:              projectPath,
		Name:              "test-project",
		Type:              projectType,
		IntegrationStatus: "missing",
		CreatedAt:         time.Now(),
		LastUpdated:       time.Now(),
		Metadata:          make(map[string]interface{}),
	}

	// Insert into database if available
	if env.DB != nil {
		metadataJSON, _ := json.Marshal(project.Metadata)
		_, err := env.DB.Exec(`
			INSERT INTO projects (id, path, name, type, integration_status, created_at, last_updated, metadata)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, project.ID, project.Path, project.Name, project.Type, project.IntegrationStatus,
			project.CreatedAt, project.LastUpdated, metadataJSON)
		if err != nil {
			t.Logf("Warning: Failed to insert test project into database: %v", err)
		}
	}

	return &TestProject{
		Project: project,
		Path:    projectPath,
		Cleanup: func() {
			os.RemoveAll(projectPath)
			if env.DB != nil {
				env.DB.Exec("DELETE FROM projects WHERE id = $1", project.ID)
			}
		},
	}
}

// HTTPTestRequest represents an HTTP test request
type HTTPTestRequest struct {
	Method      string
	Path        string
	Body        interface{}
	URLVars     map[string]string
	QueryParams map[string]string
	Headers     map[string]string
}

// makeHTTPRequest creates and executes an HTTP test request
func makeHTTPRequest(req HTTPTestRequest) (*httptest.ResponseRecorder, *http.Request, error) {
	var bodyReader *bytes.Reader

	if req.Body != nil {
		var bodyBytes []byte
		var err error

		switch v := req.Body.(type) {
		case string:
			bodyBytes = []byte(v)
		case []byte:
			bodyBytes = v
		default:
			bodyBytes, err = json.Marshal(v)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to marshal request body: %v", err)
			}
		}
		bodyReader = bytes.NewReader(bodyBytes)
	} else {
		bodyReader = bytes.NewReader([]byte{})
	}

	httpReq := httptest.NewRequest(req.Method, req.Path, bodyReader)

	// Set headers
	if req.Headers != nil {
		for key, value := range req.Headers {
			httpReq.Header.Set(key, value)
		}
	}

	// Set default content type for POST/PUT requests with body
	if req.Body != nil && httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// Set URL variables (for mux)
	if req.URLVars != nil {
		httpReq = mux.SetURLVars(httpReq, req.URLVars)
	}

	// Set query parameters
	if req.QueryParams != nil {
		q := httpReq.URL.Query()
		for key, value := range req.QueryParams {
			q.Set(key, value)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	w := httptest.NewRecorder()
	return w, httpReq, nil
}

// assertJSONResponse validates JSON response structure and content
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedFields map[string]interface{}) map[string]interface{} {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
		return nil
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v. Response: %s", err, w.Body.String())
		return nil
	}

	// Validate expected fields if provided
	if expectedFields != nil {
		for key, expectedValue := range expectedFields {
			actualValue, exists := response[key]
			if !exists {
				t.Errorf("Expected field '%s' not found in response", key)
				continue
			}

			if expectedValue != nil && actualValue != expectedValue {
				t.Errorf("Expected field '%s' to be %v, got %v", key, expectedValue, actualValue)
			}
		}
	}

	return response
}

// assertJSONArray validates that response contains an array and returns it
func assertJSONArray(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, arrayField string) []interface{} {
	response := assertJSONResponse(t, w, expectedStatus, nil)
	if response == nil {
		return nil
	}

	array, ok := response[arrayField].([]interface{})
	if !ok {
		t.Errorf("Expected field '%s' to be an array, got %T", arrayField, response[arrayField])
		return nil
	}

	return array
}

// assertErrorResponse validates error responses
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
		return
	}

	// Check if response contains an error indication
	body := w.Body.String()
	if body == "" {
		t.Error("Expected error response body, got empty")
	}
}

// assertFileExists validates that a file exists
func assertFileExists(t *testing.T, filePath string) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Errorf("Expected file to exist: %s", filePath)
	}
}

// assertFileNotExists validates that a file does not exist
func assertFileNotExists(t *testing.T, filePath string) {
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Errorf("Expected file to not exist: %s", filePath)
	}
}

// assertFileContains validates that a file contains specific content
func assertFileContains(t *testing.T, filePath string, expectedContent string) {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", filePath, err)
	}

	if !strings.Contains(string(content), expectedContent) {
		t.Errorf("Expected file %s to contain '%s', but it doesn't", filePath, expectedContent)
	}
}

// TestDataGenerator provides utilities for generating test data
type TestDataGenerator struct{}

// ScanRequest creates a test scan request
func (g *TestDataGenerator) ScanRequest(directories []string, depth int) ScanRequest {
	return ScanRequest{
		Directories: directories,
		Depth:       depth,
	}
}

// IntegrateRequest creates a test integration request
func (g *TestDataGenerator) IntegrateRequest(force bool) IntegrateRequest {
	return IntegrateRequest{
		Force: force,
	}
}

// Global test data generator instance
var TestData = &TestDataGenerator{}
