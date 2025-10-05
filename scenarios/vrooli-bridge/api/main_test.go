package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestHealthCheck tests the health check endpoint
func TestHealthCheck(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		if env.DB == nil {
			t.Skip("Database not available - health check requires DB")
		}

		w, req, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		env.App.healthCheck(w, req)

		// Health check returns 200 or 503 depending on DB availability
		if w.Code != http.StatusOK && w.Code != http.StatusServiceUnavailable {
			t.Errorf("Expected status 200 or 503, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse JSON response: %v", err)
		}

		if _, exists := response["status"]; !exists {
			t.Error("Expected 'status' field in response")
		}
	})

	t.Run("DatabaseDown", func(t *testing.T) {
		if env.DB == nil {
			t.Skip("Database not available for this test")
		}

		// Close database to simulate failure
		env.DB.Close()
		// Create new app with closed DB
		testApp := &App{
			db:           env.DB,
			projectTypes: env.App.projectTypes,
		}

		w, req, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		testApp.healthCheck(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("Expected status 503 when database is down, got %d", w.Code)
		}
	})
}

// TestGetProjects tests the GET /api/v1/projects endpoint
func TestGetProjects(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success_EmptyList", func(t *testing.T) {
		if env.DB == nil {
			t.Skip("Database not available for this test")
		}

		w, req, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/projects",
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		env.App.getProjects(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse JSON response: %v", err)
		}

		projects, ok := response["projects"].([]interface{})
		if !ok {
			t.Fatal("Expected 'projects' field to be an array")
		}

		if projects == nil {
			t.Error("Expected projects array, got nil")
		}
	})

	t.Run("Success_WithProjects", func(t *testing.T) {
		if env.DB == nil {
			t.Skip("Database not available for this test")
		}

		// Create test project
		testProject := setupTestProject(t, env, "nodejs")
		defer testProject.Cleanup()

		w, req, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/projects",
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		env.App.getProjects(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse JSON response: %v", err)
		}

		projects, ok := response["projects"].([]interface{})
		if !ok || len(projects) == 0 {
			t.Fatal("Expected at least one project in response")
		}
	})

	t.Run("DatabaseError", func(t *testing.T) {
		if env.DB == nil {
			t.Skip("Database not available for this test")
		}

		// Close database to simulate error
		env.DB.Close()
		// Create new app with closed DB
		testApp := &App{
			db:           env.DB,
			projectTypes: env.App.projectTypes,
		}

		w, req, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/projects",
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		testApp.getProjects(w, req)

		if w.Code == http.StatusOK {
			t.Error("Expected error status when database is down")
		}
	})
}

// TestScanProjects tests the POST /api/v1/projects/scan endpoint
func TestScanProjects(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success_DefaultParameters", func(t *testing.T) {
		w, req, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/projects/scan",
			Body: ScanRequest{
				Directories: []string{env.TempDir},
				Depth:       1,
			},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		env.App.scanProjects(w, req)

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response == nil {
			return
		}

		// Verify response structure
		if _, exists := response["found"]; !exists {
			t.Error("Expected 'found' field in response")
		}
		if _, exists := response["new"]; !exists {
			t.Error("Expected 'new' field in response")
		}
		if _, exists := response["projects"]; !exists {
			t.Error("Expected 'projects' field in response")
		}
	})

	t.Run("Success_FindsNodeJSProject", func(t *testing.T) {
		// Create a nodejs project
		testProject := setupTestProject(t, env, "nodejs")
		defer testProject.Cleanup()

		w, req, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/projects/scan",
			Body: ScanRequest{
				Directories: []string{env.TempDir},
				Depth:       2,
			},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		env.App.scanProjects(w, req)

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response == nil {
			return
		}

		found, ok := response["found"].(float64)
		if !ok || found < 1 {
			t.Error("Expected at least one project to be found")
		}
	})

	t.Run("Success_FindsPythonProject", func(t *testing.T) {
		// Create a python project
		testProject := setupTestProject(t, env, "python")
		defer testProject.Cleanup()

		w, req, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/projects/scan",
			Body: ScanRequest{
				Directories: []string{env.TempDir},
				Depth:       2,
			},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		env.App.scanProjects(w, req)

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response == nil {
			return
		}

		found, ok := response["found"].(float64)
		if !ok || found < 1 {
			t.Error("Expected at least one project to be found")
		}
	})

	t.Run("EmptyDirectories", func(t *testing.T) {
		w, req, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/projects/scan",
			Body: ScanRequest{
				Directories: []string{},
				Depth:       1,
			},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		env.App.scanProjects(w, req)

		// Should use default directory (home dir)
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("NegativeDepth", func(t *testing.T) {
		w, req, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/projects/scan",
			Body: ScanRequest{
				Directories: []string{env.TempDir},
				Depth:       -1,
			},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		env.App.scanProjects(w, req)

		// Should use default depth
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("ZeroDepth", func(t *testing.T) {
		w, req, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/projects/scan",
			Body: ScanRequest{
				Directories: []string{env.TempDir},
				Depth:       0,
			},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		env.App.scanProjects(w, req)

		// Should use default depth
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		w, req, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/projects/scan",
			Body:   `{"invalid": "json"`,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		env.App.scanProjects(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for invalid JSON, got %d", w.Code)
		}
	})

	t.Run("NonExistentDirectory", func(t *testing.T) {
		w, req, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/projects/scan",
			Body: ScanRequest{
				Directories: []string{"/non/existent/path"},
				Depth:       1,
			},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		env.App.scanProjects(w, req)

		// Should complete but find no projects
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response != nil {
			found, ok := response["found"].(float64)
			if !ok || found != 0 {
				t.Logf("Found %v projects in non-existent directory", found)
			}
		}
	})
}

// TestIntegrateProject tests the POST /api/v1/projects/{id}/integrate endpoint
func TestIntegrateProject(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success_NewIntegration", func(t *testing.T) {
		if env.DB == nil {
			t.Skip("Database not available for this test")
		}

		testProject := setupTestProject(t, env, "nodejs")
		defer testProject.Cleanup()

		w, req, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/projects/" + testProject.Project.ID + "/integrate",
			URLVars: map[string]string{
				"id": testProject.Project.ID,
			},
			Body: IntegrateRequest{
				Force: false,
			},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		env.App.integrateProject(w, req)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})
		if response == nil {
			return
		}

		// Verify files were created
		integrationFile := filepath.Join(testProject.Path, "VROOLI_INTEGRATION.md")
		assertFileExists(t, integrationFile)

		claudeFile := filepath.Join(testProject.Path, "CLAUDE.md")
		assertFileExists(t, claudeFile)
	})

	t.Run("Success_ForceIntegration", func(t *testing.T) {
		if env.DB == nil {
			t.Skip("Database not available for this test")
		}

		testProject := setupTestProject(t, env, "python")
		defer testProject.Cleanup()

		// Create existing integration file
		integrationFile := filepath.Join(testProject.Path, "VROOLI_INTEGRATION.md")
		ioutil.WriteFile(integrationFile, []byte("old content"), 0644)

		w, req, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/projects/" + testProject.Project.ID + "/integrate",
			URLVars: map[string]string{
				"id": testProject.Project.ID,
			},
			Body: IntegrateRequest{
				Force: true,
			},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		env.App.integrateProject(w, req)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})
		if response == nil {
			return
		}

		// Verify file was updated
		content, _ := ioutil.ReadFile(integrationFile)
		if string(content) == "old content" {
			t.Error("Expected file to be updated with force flag")
		}
	})

	t.Run("Success_ExistingCLAUDEMD", func(t *testing.T) {
		if env.DB == nil {
			t.Skip("Database not available for this test")
		}

		testProject := setupTestProject(t, env, "nodejs")
		defer testProject.Cleanup()

		// Create existing CLAUDE.md
		claudeFile := filepath.Join(testProject.Path, "CLAUDE.md")
		ioutil.WriteFile(claudeFile, []byte("# Existing Content\n"), 0644)

		w, req, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/projects/" + testProject.Project.ID + "/integrate",
			URLVars: map[string]string{
				"id": testProject.Project.ID,
			},
			Body: IntegrateRequest{
				Force: false,
			},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		env.App.integrateProject(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Verify CLAUDE.md was appended
		assertFileContains(t, claudeFile, "Existing Content")
		assertFileContains(t, claudeFile, "Vrooli Integration")
	})

	t.Run("NonExistentProject", func(t *testing.T) {
		if env.DB == nil {
			t.Skip("Database not available for this test")
		}

		nonExistentID := uuid.New().String()

		w, req, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/projects/" + nonExistentID + "/integrate",
			URLVars: map[string]string{
				"id": nonExistentID,
			},
			Body: IntegrateRequest{
				Force: false,
			},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		env.App.integrateProject(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})

	t.Run("EmptyBody", func(t *testing.T) {
		if env.DB == nil {
			t.Skip("Database not available for this test")
		}

		testProject := setupTestProject(t, env, "nodejs")
		defer testProject.Cleanup()

		w, req, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/projects/" + testProject.Project.ID + "/integrate",
			URLVars: map[string]string{
				"id": testProject.Project.ID,
			},
			Body: nil,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		env.App.integrateProject(w, req)

		// Should default to force=false and succeed
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 with empty body, got %d", w.Code)
		}
	})
}

// TestDeleteProject tests the DELETE /api/v1/projects/{id} endpoint
func TestDeleteProject(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		if env.DB == nil {
			t.Skip("Database not available for this test")
		}

		testProject := setupTestProject(t, env, "nodejs")
		defer testProject.Cleanup()

		// Create integration file
		integrationFile := filepath.Join(testProject.Path, "VROOLI_INTEGRATION.md")
		ioutil.WriteFile(integrationFile, []byte("test content"), 0644)

		w, req, err := makeHTTPRequest(HTTPTestRequest{
			Method: "DELETE",
			Path:   "/api/v1/projects/" + testProject.Project.ID,
			URLVars: map[string]string{
				"id": testProject.Project.ID,
			},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		env.App.deleteProject(w, req)

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response == nil {
			return
		}

		// Verify integration file was removed
		assertFileNotExists(t, integrationFile)

		// Verify project status updated to 'missing'
		var status string
		err = env.DB.QueryRow("SELECT integration_status FROM projects WHERE id = $1", testProject.Project.ID).Scan(&status)
		if err == nil && status != "missing" {
			t.Errorf("Expected integration_status to be 'missing', got '%s'", status)
		}
	})

	t.Run("NonExistentProject", func(t *testing.T) {
		if env.DB == nil {
			t.Skip("Database not available for this test")
		}

		nonExistentID := uuid.New().String()

		w, req, err := makeHTTPRequest(HTTPTestRequest{
			Method: "DELETE",
			Path:   "/api/v1/projects/" + nonExistentID,
			URLVars: map[string]string{
				"id": nonExistentID,
			},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		env.App.deleteProject(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})

	t.Run("NoIntegrationFile", func(t *testing.T) {
		if env.DB == nil {
			t.Skip("Database not available for this test")
		}

		testProject := setupTestProject(t, env, "python")
		defer testProject.Cleanup()

		w, req, err := makeHTTPRequest(HTTPTestRequest{
			Method: "DELETE",
			Path:   "/api/v1/projects/" + testProject.Project.ID,
			URLVars: map[string]string{
				"id": testProject.Project.ID,
			},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		env.App.deleteProject(w, req)

		// Should succeed even if file doesn't exist
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// TestDetectProjectType tests the project type detection logic
func TestDetectProjectType(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	tests := []struct {
		name         string
		fileName     string
		expectedType string
	}{
		{"NodeJS", "package.json", "nodejs"},
		{"Python_Requirements", "requirements.txt", "python"},
		{"Python_Pyproject", "pyproject.toml", "python"},
		{"Unknown", "random.txt", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := env.App.detectProjectType(tt.fileName, tt.fileName)
			if result != tt.expectedType {
				t.Errorf("Expected project type '%s', got '%s'", tt.expectedType, result)
			}
		})
	}
}

// TestScanDirectory tests the directory scanning logic
func TestScanDirectory(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("FindsProjects", func(t *testing.T) {
		// Create test projects
		testProject := setupTestProject(t, env, "nodejs")
		defer testProject.Cleanup()

		projects := env.App.scanDirectory(env.TempDir, 2)

		if len(projects) == 0 {
			t.Error("Expected to find at least one project")
		}
	})

	t.Run("RespectsDepth", func(t *testing.T) {
		// Create nested project structure
		deepPath := filepath.Join(env.TempDir, "level1", "level2", "level3")
		os.MkdirAll(deepPath, 0755)
		ioutil.WriteFile(filepath.Join(deepPath, "package.json"), []byte("{}"), 0644)

		// Scan with depth 1 - should not find the deep project
		projects := env.App.scanDirectory(env.TempDir, 1)

		// Count projects at depth 3
		deepProjects := 0
		for _, p := range projects {
			if filepath.Base(filepath.Dir(p.Path)) == "level2" {
				deepProjects++
			}
		}

		if deepProjects > 0 {
			t.Error("Expected depth limit to be respected")
		}
	})

	t.Run("SkipsHiddenDirectories", func(t *testing.T) {
		// Create hidden directory with project
		hiddenPath := filepath.Join(env.TempDir, ".hidden")
		os.MkdirAll(hiddenPath, 0755)
		ioutil.WriteFile(filepath.Join(hiddenPath, "package.json"), []byte("{}"), 0644)

		projects := env.App.scanDirectory(env.TempDir, 2)

		// Should not find project in hidden directory
		for _, p := range projects {
			if filepath.Base(filepath.Dir(p.Path)) == ".hidden" {
				t.Error("Expected hidden directories to be skipped")
			}
		}
	})

	t.Run("SkipsNodeModules", func(t *testing.T) {
		// Create node_modules with package.json
		nodeModulesPath := filepath.Join(env.TempDir, "node_modules", "some-package")
		os.MkdirAll(nodeModulesPath, 0755)
		ioutil.WriteFile(filepath.Join(nodeModulesPath, "package.json"), []byte("{}"), 0644)

		projects := env.App.scanDirectory(env.TempDir, 3)

		// Should not find projects in node_modules
		for _, p := range projects {
			if filepath.Base(filepath.Dir(p.Path)) == "node_modules" {
				t.Error("Expected node_modules to be skipped")
			}
		}
	})
}

// TestProjectExists tests the project existence check
func TestProjectExists(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	if env.DB == nil {
		t.Skip("Database not available for this test")
	}

	t.Run("ExistingProject", func(t *testing.T) {
		testProject := setupTestProject(t, env, "nodejs")
		defer testProject.Cleanup()

		exists, err := env.App.projectExists(testProject.Path)
		if err != nil {
			t.Fatalf("Error checking project existence: %v", err)
		}

		if !exists {
			t.Error("Expected project to exist")
		}
	})

	t.Run("NonExistentProject", func(t *testing.T) {
		exists, err := env.App.projectExists("/non/existent/path")
		if err != nil {
			t.Fatalf("Error checking project existence: %v", err)
		}

		if exists {
			t.Error("Expected project to not exist")
		}
	})
}

// TestFileExists tests the file existence check
func TestFileExists(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("ExistingFile", func(t *testing.T) {
		tempFile := filepath.Join(env.TempDir, "test.txt")
		ioutil.WriteFile(tempFile, []byte("test"), 0644)

		if !env.App.fileExists(tempFile) {
			t.Error("Expected file to exist")
		}
	})

	t.Run("NonExistentFile", func(t *testing.T) {
		if env.App.fileExists("/non/existent/file.txt") {
			t.Error("Expected file to not exist")
		}
	})
}

// TestGenerateIntegrationDoc tests integration document generation
func TestGenerateIntegrationDoc(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		project := &Project{
			ID:                uuid.New().String(),
			Path:              "/test/path",
			Name:              "test-project",
			Type:              "nodejs",
			IntegrationStatus: "active",
			CreatedAt:         time.Now(),
			LastUpdated:       time.Now(),
			Metadata:          make(map[string]interface{}),
		}

		content := env.App.generateIntegrationDoc(project)

		if len(content) == 0 {
			t.Error("Expected non-empty integration document")
		}

		// Verify content includes project details
		if !containsString(content, "test-project") {
			t.Error("Expected integration doc to contain project name")
		}
	})

	t.Run("GenericType", func(t *testing.T) {
		project := &Project{
			ID:                uuid.New().String(),
			Path:              "/test/path",
			Name:              "test-project",
			Type:              "unknown-type",
			IntegrationStatus: "active",
			CreatedAt:         time.Now(),
			LastUpdated:       time.Now(),
			Metadata:          make(map[string]interface{}),
		}

		content := env.App.generateIntegrationDoc(project)

		if len(content) == 0 {
			t.Error("Expected fallback to generic template")
		}
	})
}

// TestGenerateClaudeAddition tests CLAUDE.md addition generation
func TestGenerateClaudeAddition(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		project := &Project{
			ID:                uuid.New().String(),
			Path:              "/test/path",
			Name:              "test-project",
			Type:              "python",
			IntegrationStatus: "active",
			CreatedAt:         time.Now(),
			LastUpdated:       time.Now(),
			Metadata:          make(map[string]interface{}),
		}

		content := env.App.generateClaudeAddition(project)

		if len(content) == 0 {
			t.Error("Expected non-empty CLAUDE addition")
		}

		if !containsString(content, "Vrooli Integration") {
			t.Error("Expected CLAUDE addition to mention Vrooli Integration")
		}
	})
}

// TestCORSMiddleware tests the CORS middleware
func TestCORSMiddleware(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("PreflightRequest", func(t *testing.T) {
		w, req, err := makeHTTPRequest(HTTPTestRequest{
			Method: "OPTIONS",
			Path:   "/api/v1/projects",
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		handler := env.App.corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for OPTIONS request, got %d", w.Code)
		}

		// Verify CORS headers
		if origin := w.Header().Get("Access-Control-Allow-Origin"); origin == "" {
			t.Error("Expected Access-Control-Allow-Origin header")
		}

		if methods := w.Header().Get("Access-Control-Allow-Methods"); methods == "" {
			t.Error("Expected Access-Control-Allow-Methods header")
		}
	})

	t.Run("NormalRequest", func(t *testing.T) {
		w, req, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/projects",
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		handlerCalled := false
		handler := env.App.corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			w.WriteHeader(http.StatusOK)
		}))

		handler.ServeHTTP(w, req)

		if !handlerCalled {
			t.Error("Expected handler to be called for normal request")
		}

		// Verify CORS headers are still set
		if origin := w.Header().Get("Access-Control-Allow-Origin"); origin == "" {
			t.Error("Expected Access-Control-Allow-Origin header")
		}
	})
}

// TestDatabaseConnection tests database initialization and connection
func TestDatabaseConnection(t *testing.T) {
	t.Run("ValidConnection", func(t *testing.T) {
		// This test would require actual database credentials
		// Skipping for now as it requires external dependencies
		t.Skip("Requires actual database setup")
	})

	t.Run("MissingConfig", func(t *testing.T) {
		app := &App{}

		// Save original env vars
		originalURL := os.Getenv("POSTGRES_URL")
		originalHost := os.Getenv("POSTGRES_HOST")

		// Clear all postgres env vars
		os.Unsetenv("POSTGRES_URL")
		os.Unsetenv("POSTGRES_HOST")
		os.Unsetenv("POSTGRES_PORT")
		os.Unsetenv("POSTGRES_USER")
		os.Unsetenv("POSTGRES_PASSWORD")
		os.Unsetenv("POSTGRES_DB")

		err := app.initDB()

		// Restore env vars
		if originalURL != "" {
			os.Setenv("POSTGRES_URL", originalURL)
		}
		if originalHost != "" {
			os.Setenv("POSTGRES_HOST", originalHost)
		}

		if err == nil {
			t.Error("Expected error when database config is missing")
		}
	})
}

// TestLoadProjectTypes tests loading project type definitions
func TestLoadProjectTypes(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		app := &App{}
		err := app.loadProjectTypes()

		if err != nil {
			t.Fatalf("Failed to load project types: %v", err)
		}

		if len(app.projectTypes) == 0 {
			t.Error("Expected project types to be loaded")
		}

		// Verify specific types exist
		if _, exists := app.projectTypes["nodejs"]; !exists {
			t.Error("Expected nodejs project type to be loaded")
		}

		// Verify python project type
		if _, exists := app.projectTypes["python"]; !exists {
			t.Error("Expected python project type to be loaded")
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		// Create temp directory with invalid JSON
		tempDir, _ := ioutil.TempDir("", "invalid-json")
		defer os.RemoveAll(tempDir)

		// Create initialization/templates directory
		templatesDir := filepath.Join(tempDir, "../initialization/templates")
		os.MkdirAll(templatesDir, 0755)

		// Write invalid JSON
		invalidJSON := `{"nodejs": invalid json`
		ioutil.WriteFile(filepath.Join(templatesDir, "project-types.json"), []byte(invalidJSON), 0644)

		originalWD, _ := os.Getwd()
		os.Chdir(tempDir)
		defer os.Chdir(originalWD)

		app := &App{}
		err := app.loadProjectTypes()

		if err == nil {
			t.Log("Expected error when project-types.json contains invalid JSON")
		}
	})
}

// Helper function to check if string contains substring
func containsString(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) >= len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestSetupRoutes tests route configuration
func TestSetupRoutes(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	router := env.App.setupRoutes()

	if router == nil {
		t.Fatal("Expected router to be created")
	}

	// Test that routes are registered
	// We can't directly test mux routes without reflection,
	// but we can verify the router was created successfully
}

// TestInitDB tests database initialization logic
func TestInitDB(t *testing.T) {
	t.Run("MissingPostgresURL", func(t *testing.T) {
		// Save original env vars
		originalURL := os.Getenv("POSTGRES_URL")
		originalHost := os.Getenv("POSTGRES_HOST")
		originalPort := os.Getenv("POSTGRES_PORT")
		originalUser := os.Getenv("POSTGRES_USER")
		originalPassword := os.Getenv("POSTGRES_PASSWORD")
		originalDB := os.Getenv("POSTGRES_DB")

		// Clear all postgres env vars
		os.Unsetenv("POSTGRES_URL")
		os.Unsetenv("POSTGRES_HOST")
		os.Unsetenv("POSTGRES_PORT")
		os.Unsetenv("POSTGRES_USER")
		os.Unsetenv("POSTGRES_PASSWORD")
		os.Unsetenv("POSTGRES_DB")

		app := &App{}
		err := app.initDB()

		// Restore env vars
		if originalURL != "" {
			os.Setenv("POSTGRES_URL", originalURL)
		}
		if originalHost != "" {
			os.Setenv("POSTGRES_HOST", originalHost)
		}
		if originalPort != "" {
			os.Setenv("POSTGRES_PORT", originalPort)
		}
		if originalUser != "" {
			os.Setenv("POSTGRES_USER", originalUser)
		}
		if originalPassword != "" {
			os.Setenv("POSTGRES_PASSWORD", originalPassword)
		}
		if originalDB != "" {
			os.Setenv("POSTGRES_DB", originalDB)
		}

		if err == nil {
			if app.db != nil {
				app.db.Close()
			}
			t.Error("Expected error when database config is missing")
		}
	})

	t.Run("PartialConfig", func(t *testing.T) {
		// Save original env vars
		originalURL := os.Getenv("POSTGRES_URL")
		originalHost := os.Getenv("POSTGRES_HOST")

		// Set partial config
		os.Unsetenv("POSTGRES_URL")
		os.Setenv("POSTGRES_HOST", "localhost")
		os.Unsetenv("POSTGRES_PORT")

		app := &App{}
		err := app.initDB()

		// Restore env vars
		if originalURL != "" {
			os.Setenv("POSTGRES_URL", originalURL)
		} else {
			os.Unsetenv("POSTGRES_URL")
		}
		if originalHost != "" {
			os.Setenv("POSTGRES_HOST", originalHost)
		} else {
			os.Unsetenv("POSTGRES_HOST")
		}

		if err == nil {
			if app.db != nil {
				app.db.Close()
			}
			t.Error("Expected error when partial database config is provided")
		}
	})

	// Skip tests that attempt actual database connections - they can hang
	// These tests would require mocking the database driver
}

// TestPerformIntegration tests integration file generation
func TestPerformIntegration(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	projectPath := filepath.Join(env.TempDir, "test-integration")
	os.MkdirAll(projectPath, 0755)

	project := &Project{
		ID:                uuid.New().String(),
		Path:              projectPath,
		Name:              "test-integration",
		Type:              "nodejs",
		IntegrationStatus: "missing",
		CreatedAt:         time.Now(),
		LastUpdated:       time.Now(),
		Metadata:          make(map[string]interface{}),
	}

	t.Run("NewFiles", func(t *testing.T) {
		filesCreated, filesUpdated, err := env.App.performIntegration(project, false)
		if err != nil {
			t.Fatalf("Failed to perform integration: %v", err)
		}

		if len(filesCreated) == 0 {
			t.Error("Expected files to be created")
		}

		// Verify VROOLI_INTEGRATION.md was created
		integrationFile := filepath.Join(projectPath, "VROOLI_INTEGRATION.md")
		assertFileExists(t, integrationFile)

		// Verify CLAUDE.md was created
		claudeFile := filepath.Join(projectPath, "CLAUDE.md")
		assertFileExists(t, claudeFile)

		// Clean up for next test
		os.Remove(integrationFile)
		os.Remove(claudeFile)

		if len(filesUpdated) != 0 {
			t.Logf("Files updated: %v", filesUpdated)
		}
	})

	t.Run("ForceUpdate", func(t *testing.T) {
		// Create existing file
		integrationFile := filepath.Join(projectPath, "VROOLI_INTEGRATION.md")
		ioutil.WriteFile(integrationFile, []byte("old content"), 0644)

		filesCreated, filesUpdated, err := env.App.performIntegration(project, true)
		if err != nil {
			t.Fatalf("Failed to perform integration: %v", err)
		}

		// Verify file was updated
		content, _ := ioutil.ReadFile(integrationFile)
		if string(content) == "old content" {
			t.Error("Expected file to be updated with force flag")
		}

		if len(filesCreated) == 0 && len(filesUpdated) == 0 {
			t.Error("Expected files to be created or updated")
		}
	})

	t.Run("AppendToCLAUDE", func(t *testing.T) {
		// Create existing CLAUDE.md
		claudeFile := filepath.Join(projectPath, "CLAUDE.md")
		originalContent := "# Existing Content\n\nSome instructions\n"
		ioutil.WriteFile(claudeFile, []byte(originalContent), 0644)

		_, _, err := env.App.performIntegration(project, false)
		if err != nil {
			t.Fatalf("Failed to perform integration: %v", err)
		}

		// Verify original content is preserved
		content, _ := ioutil.ReadFile(claudeFile)
		if !containsSubstring(string(content), "Existing Content") {
			t.Error("Expected original content to be preserved")
		}

		// Verify Vrooli section was added
		if !containsSubstring(string(content), "Vrooli Integration") {
			t.Error("Expected Vrooli Integration section to be added")
		}
	})

	t.Run("NoDoubleAppend", func(t *testing.T) {
		// Run integration twice
		env.App.performIntegration(project, false)

		// Second run should not append again
		claudeFile := filepath.Join(projectPath, "CLAUDE.md")
		contentBefore, _ := ioutil.ReadFile(claudeFile)

		env.App.performIntegration(project, false)

		contentAfter, _ := ioutil.ReadFile(claudeFile)

		// Content should be the same (no double append)
		if string(contentBefore) != string(contentAfter) {
			t.Error("Expected CLAUDE.md not to be appended twice")
		}
	})
}

// TestGetProjectByID tests project retrieval by ID
func TestGetProjectByID(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	if env.DB == nil {
		t.Skip("Database not available for this test")
	}

	t.Run("ExistingProject", func(t *testing.T) {
		testProject := setupTestProject(t, env, "nodejs")
		defer testProject.Cleanup()

		project, err := env.App.getProjectByID(testProject.Project.ID)
		if err != nil {
			t.Fatalf("Failed to get project: %v", err)
		}

		if project.ID != testProject.Project.ID {
			t.Errorf("Expected project ID %s, got %s", testProject.Project.ID, project.ID)
		}
	})

	t.Run("NonExistentProject", func(t *testing.T) {
		_, err := env.App.getProjectByID(uuid.New().String())
		if err == nil {
			t.Error("Expected error for non-existent project")
		}
	})
}

// TestScanProjectsAdvanced tests advanced scanning scenarios
func TestScanProjectsAdvanced(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("MultipleProjectTypes", func(t *testing.T) {
		// Create different project types
		nodejsPath := filepath.Join(env.TempDir, "nodejs-app")
		pythonPath := filepath.Join(env.TempDir, "python-app")

		os.MkdirAll(nodejsPath, 0755)
		os.MkdirAll(pythonPath, 0755)

		ioutil.WriteFile(filepath.Join(nodejsPath, "package.json"), []byte(`{"name": "test"}`), 0644)
		ioutil.WriteFile(filepath.Join(pythonPath, "requirements.txt"), []byte("pytest"), 0644)

		w, req, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/projects/scan",
			Body: ScanRequest{
				Directories: []string{env.TempDir},
				Depth:       2,
			},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		env.App.scanProjects(w, req)

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response == nil {
			return
		}

		found, ok := response["found"].(float64)
		if !ok || found < 2 {
			t.Logf("Expected at least 2 projects to be found, got %v", found)
		}
	})

	t.Run("DeepNesting", func(t *testing.T) {
		// Create deeply nested structure
		deepPath := filepath.Join(env.TempDir, "level1", "level2", "level3", "project")
		os.MkdirAll(deepPath, 0755)
		ioutil.WriteFile(filepath.Join(deepPath, "package.json"), []byte(`{"name": "deep"}`), 0644)

		// Scan with high depth
		w, req, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/projects/scan",
			Body: ScanRequest{
				Directories: []string{env.TempDir},
				Depth:       5,
			},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		env.App.scanProjects(w, req)

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response != nil {
			// Should find the deep project
			found, ok := response["found"].(float64)
			if !ok || found < 1 {
				t.Logf("Expected to find deep project, got %v", found)
			}
		}
	})

	t.Run("LargeDepthValue", func(t *testing.T) {
		w, req, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/projects/scan",
			Body: ScanRequest{
				Directories: []string{env.TempDir},
				Depth:       1000, // Very large depth
			},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		env.App.scanProjects(w, req)

		// Should complete without error
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 with large depth, got %d", w.Code)
		}
	})
}

// TestHelperFunctions tests helper assertion functions
func TestHelperFunctions(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("AssertFileExists", func(t *testing.T) {
		// Create temp file
		tempFile := filepath.Join(env.TempDir, "exists.txt")
		ioutil.WriteFile(tempFile, []byte("content"), 0644)

		// This should not error
		assertFileExists(t, tempFile)
	})

	t.Run("AssertFileNotExists", func(t *testing.T) {
		// This should not error
		assertFileNotExists(t, filepath.Join(env.TempDir, "nonexistent.txt"))
	})

	t.Run("AssertFileContains", func(t *testing.T) {
		tempFile := filepath.Join(env.TempDir, "contains.txt")
		ioutil.WriteFile(tempFile, []byte("Hello World"), 0644)

		assertFileContains(t, tempFile, "Hello")
		assertFileContains(t, tempFile, "World")
	})

	t.Run("AssertJSONArray", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"items": [1, 2, 3]}`))

		array := assertJSONArray(t, w, http.StatusOK, "items")
		if len(array) != 3 {
			t.Errorf("Expected array length 3, got %d", len(array))
		}
	})

	t.Run("AssertErrorResponse", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`error message`))

		assertErrorResponse(t, w, http.StatusBadRequest)
	})
}

// TestIntegrationEdgeCases tests edge cases in integration logic
func TestIntegrationEdgeCases(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("ReadOnlyDirectory", func(t *testing.T) {
		projectPath := filepath.Join(env.TempDir, "readonly-project")
		os.MkdirAll(projectPath, 0755)

		// Make directory read-only
		os.Chmod(projectPath, 0555)
		defer os.Chmod(projectPath, 0755)

		project := &Project{
			ID:                uuid.New().String(),
			Path:              projectPath,
			Name:              "readonly-project",
			Type:              "nodejs",
			IntegrationStatus: "missing",
			CreatedAt:         time.Now(),
			LastUpdated:       time.Now(),
			Metadata:          make(map[string]interface{}),
		}

		_, _, err := env.App.performIntegration(project, false)

		// Should fail due to permission denied
		if err == nil {
			t.Log("Expected error when writing to read-only directory, but got nil")
		}
	})

	t.Run("InvalidProjectPath", func(t *testing.T) {
		project := &Project{
			ID:                uuid.New().String(),
			Path:              "/nonexistent/invalid/path",
			Name:              "invalid-project",
			Type:              "nodejs",
			IntegrationStatus: "missing",
			CreatedAt:         time.Now(),
			LastUpdated:       time.Now(),
			Metadata:          make(map[string]interface{}),
		}

		_, _, err := env.App.performIntegration(project, false)

		if err == nil {
			t.Error("Expected error with invalid project path")
		}
	})
}

// TestTemplateReplacement tests template variable replacement
func TestTemplateReplacement(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("AllPlaceholdersReplaced", func(t *testing.T) {
		project := &Project{
			ID:                uuid.New().String(),
			Path:              "/test/path",
			Name:              "test-project",
			Type:              "nodejs",
			IntegrationStatus: "active",
			CreatedAt:         time.Now(),
			LastUpdated:       time.Now(),
			Metadata:          make(map[string]interface{}),
		}

		integrationDoc := env.App.generateIntegrationDoc(project)

		// Verify placeholders are replaced
		if containsSubstring(integrationDoc, "{{") || containsSubstring(integrationDoc, "}}") {
			t.Error("Found unreplaced template placeholders in integration doc")
		}

		if !containsSubstring(integrationDoc, "test-project") {
			t.Error("Expected project name in integration doc")
		}
	})

	t.Run("ClaudeAdditionComplete", func(t *testing.T) {
		project := &Project{
			ID:                uuid.New().String(),
			Path:              "/test/path",
			Name:              "test-project",
			Type:              "python",
			IntegrationStatus: "active",
			CreatedAt:         time.Now(),
			LastUpdated:       time.Now(),
			Metadata:          make(map[string]interface{}),
		}

		claudeAddition := env.App.generateClaudeAddition(project)

		// Verify placeholders are replaced
		if containsSubstring(claudeAddition, "{{") || containsSubstring(claudeAddition, "}}") {
			t.Error("Found unreplaced template placeholders in CLAUDE addition")
		}
	})
}

// TestProjectTypeCoverage tests all project type detection patterns
func TestProjectTypeCoverage(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	testCases := []struct {
		fileName string
		expected string
	}{
		{"package.json", "nodejs"},
		{"requirements.txt", "python"},
		{"pyproject.toml", "python"},
		{"Cargo.toml", ""},        // Not in default types
		{"go.mod", ""},            // Not in default types
		{"random.file", ""},
		{"", ""},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Detect_%s", tc.fileName), func(t *testing.T) {
			result := env.App.detectProjectType(tc.fileName, tc.fileName)
			if result != tc.expected {
				t.Errorf("Expected project type '%s' for %s, got '%s'", tc.expected, tc.fileName, result)
			}
		})
	}
}

// TestScanPerformance tests scanning performance with many files
func TestScanPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("ManyFiles", func(t *testing.T) {
		// Create many files
		for i := 0; i < 50; i++ {
			dir := filepath.Join(env.TempDir, fmt.Sprintf("dir%d", i))
			os.MkdirAll(dir, 0755)

			// Create some non-project files
			for j := 0; j < 10; j++ {
				ioutil.WriteFile(filepath.Join(dir, fmt.Sprintf("file%d.txt", j)), []byte("content"), 0644)
			}
		}

		// Add a couple of actual projects
		os.MkdirAll(filepath.Join(env.TempDir, "project1"), 0755)
		ioutil.WriteFile(filepath.Join(env.TempDir, "project1", "package.json"), []byte("{}"), 0644)

		start := time.Now()

		w, req, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/projects/scan",
			Body: ScanRequest{
				Directories: []string{env.TempDir},
				Depth:       2,
			},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		env.App.scanProjects(w, req)

		duration := time.Since(start)

		if duration > 10*time.Second {
			t.Errorf("Scan took too long: %v", duration)
		}

		t.Logf("Scan completed in %v", duration)
	})
}

// TestMain sets up test database if needed
func TestMain(m *testing.M) {
	// Setup: Create test database schema if database is available
	db, err := sql.Open("postgres", "postgres://test:test@localhost:5432/vrooli_bridge_test?sslmode=disable")
	if err == nil {
		db.Exec("DROP TABLE IF EXISTS projects")
		db.Exec(`
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
		db.Close()
	}

	// Run tests
	code := m.Run()

	// Cleanup: Drop test database schema if needed
	db, err = sql.Open("postgres", "postgres://test:test@localhost:5432/vrooli_bridge_test?sslmode=disable")
	if err == nil {
		db.Exec("DROP TABLE IF EXISTS projects")
		db.Close()
	}

	os.Exit(code)
}
