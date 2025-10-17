// +build testing

package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// TestHealth tests the health endpoint
func TestHealth(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/health", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(Health)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, status)
		}

		var response map[string]string
		if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
			t.Errorf("Failed to decode response: %v", err)
		}

		if response["status"] != "healthy" {
			t.Errorf("Expected status 'healthy', got '%s'", response["status"])
		}

		if response["service"] != serviceName {
			t.Errorf("Expected service '%s', got '%s'", serviceName, response["service"])
		}

		if response["version"] != apiVersion {
			t.Errorf("Expected version '%s', got '%s'", apiVersion, response["version"])
		}
	})
}

// TestHTTPError tests the HTTPError function
func TestHTTPError(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ErrorResponse", func(t *testing.T) {
		rr := httptest.NewRecorder()
		HTTPError(rr, "Test error", http.StatusBadRequest, nil)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, status)
		}

		var response map[string]interface{}
		if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
			t.Errorf("Failed to decode response: %v", err)
		}

		if response["error"] != "Test error" {
			t.Errorf("Expected error 'Test error', got '%v'", response["error"])
		}

		if response["status"] != float64(http.StatusBadRequest) {
			t.Errorf("Expected status %d, got %v", http.StatusBadRequest, response["status"])
		}

		if _, ok := response["timestamp"]; !ok {
			t.Error("Expected 'timestamp' field in response")
		}
	})
}

// TestRegisterApp tests the RegisterApp endpoint
func TestRegisterApp(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	// Create mock service (no DB for now)
	service := &AppPersonalizerService{
		db:         nil,
		n8nBaseURL: "http://localhost:5678",
		minioURL:   "http://localhost:9000",
		httpClient: &http.Client{Timeout: httpTimeout},
		logger:     NewLogger(),
	}

	t.Run("Success", func(t *testing.T) {
		// Skip this test if DB is not available
		if service.db == nil {
			t.Skip("Skipping test: database not configured")
		}

		reqBody := RegisterAppRequest{
			AppName:   "test-app",
			AppPath:   env.TestAppPath,
			AppType:   "generated",
			Framework: "react",
			Version:   "1.0.0",
		}

		bodyBytes, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/apps/register", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(service.RegisterApp)
		handler.ServeHTTP(rr, req)

		// For now, expect failure due to no DB
		// This test will pass once DB is properly configured
		t.Logf("Response status: %d", rr.Code)
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/apps/register", bytes.NewBufferString("invalid-json"))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(service.RegisterApp)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, status)
		}
	})

	t.Run("MissingRequiredFields", func(t *testing.T) {
		reqBody := map[string]string{
			"app_name": "test-app",
			// Missing app_path, app_type, framework
		}

		bodyBytes, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/apps/register", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(service.RegisterApp)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, status)
		}
	})

	t.Run("NonExistentPath", func(t *testing.T) {
		reqBody := RegisterAppRequest{
			AppName:   "test-app",
			AppPath:   "/nonexistent/path",
			AppType:   "generated",
			Framework: "react",
			Version:   "1.0.0",
		}

		bodyBytes, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/apps/register", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(service.RegisterApp)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, status)
		}
	})
}

// TestListApps tests the ListApps endpoint
func TestListApps(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	_ = &AppPersonalizerService{
		db:         nil,
		n8nBaseURL: "http://localhost:5678",
		minioURL:   "http://localhost:9000",
		httpClient: &http.Client{Timeout: httpTimeout},
		logger:     NewLogger(),
	}

	t.Run("NoDatabaseConnection", func(t *testing.T) {
		t.Skip("Skipping test: database not configured")
	})
}

// TestAnalyzeApp tests the AnalyzeApp endpoint
func TestAnalyzeApp(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	service := &AppPersonalizerService{
		db:         nil,
		n8nBaseURL: "http://localhost:5678",
		minioURL:   "http://localhost:9000",
		httpClient: &http.Client{Timeout: httpTimeout},
		logger:     NewLogger(),
	}

	t.Run("InvalidJSON", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/apps/analyze", bytes.NewBufferString("invalid-json"))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(service.AnalyzeApp)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, status)
		}
	})
}

// TestAnalyzeAppStructure tests the app structure analysis functions
func TestAnalyzeAppStructure(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	service := &AppPersonalizerService{
		logger: NewLogger(),
	}

	t.Run("ReactApp", func(t *testing.T) {
		points := service.analyzeAppStructure(env.TestAppPath, "react")

		if points == nil {
			t.Fatal("Expected non-nil points")
		}

		// Check for expected categories
		expectedCategories := []string{"ui_theme", "content", "branding", "behavior", "configuration"}
		for _, cat := range expectedCategories {
			if _, ok := points[cat]; !ok {
				t.Errorf("Expected category '%s' in points", cat)
			}
		}

		// Check that ui_theme contains the files we created
		uiTheme, ok := points["ui_theme"].([]string)
		if !ok {
			t.Errorf("Expected ui_theme to be []string, got %T", points["ui_theme"])
		} else {
			hasThemeJS := false
			hasTailwind := false
			for _, file := range uiTheme {
				if strings.Contains(file, "theme.js") {
					hasThemeJS = true
				}
				if strings.Contains(file, "tailwind.config.js") {
					hasTailwind = true
				}
			}
			if !hasThemeJS {
				t.Error("Expected to find theme.js in ui_theme points")
			}
			if !hasTailwind {
				t.Error("Expected to find tailwind.config.js in ui_theme points")
			}
		}
	})

	t.Run("VueApp", func(t *testing.T) {
		points := service.analyzeAppStructure(env.TestAppPath, "vue")
		if points == nil {
			t.Fatal("Expected non-nil points")
		}
	})

	t.Run("NextApp", func(t *testing.T) {
		points := service.analyzeAppStructure(env.TestAppPath, "next.js")
		if points == nil {
			t.Fatal("Expected non-nil points")
		}
	})

	t.Run("GenericApp", func(t *testing.T) {
		points := service.analyzeAppStructure(env.TestAppPath, "unknown")
		if points == nil {
			t.Fatal("Expected non-nil points")
		}
	})
}

// TestBackupApp tests the BackupApp endpoint
func TestBackupApp(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	service := &AppPersonalizerService{
		db:         nil,
		n8nBaseURL: "http://localhost:5678",
		minioURL:   "http://localhost:9000",
		httpClient: &http.Client{Timeout: httpTimeout},
		logger:     NewLogger(),
	}

	t.Run("Success", func(t *testing.T) {
		reqBody := BackupAppRequest{
			AppPath:    env.TestAppPath,
			BackupType: "full",
		}

		bodyBytes, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/backup", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(service.BackupApp)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, status, rr.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if _, ok := response["backup_path"]; !ok {
			t.Error("Expected 'backup_path' in response")
		}

		if response["backup_type"] != "full" {
			t.Errorf("Expected backup_type 'full', got '%v'", response["backup_type"])
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/backup", bytes.NewBufferString("invalid-json"))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(service.BackupApp)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, status)
		}
	})

	t.Run("MissingAppPath", func(t *testing.T) {
		reqBody := map[string]string{
			"backup_type": "full",
		}

		bodyBytes, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/backup", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(service.BackupApp)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, status)
		}
	})
}

// TestCreateAppBackup tests the backup creation logic
func TestCreateAppBackup(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	service := &AppPersonalizerService{
		logger: NewLogger(),
	}

	t.Run("CreateFullBackup", func(t *testing.T) {
		backupPath, err := service.createAppBackup(env.TestAppPath, "full")
		if err != nil {
			t.Fatalf("Failed to create backup: %v", err)
		}

		if backupPath == "" {
			t.Error("Expected non-empty backup path")
		}

		// Verify backup file exists
		if _, err := os.Stat(backupPath); os.IsNotExist(err) {
			t.Errorf("Backup file does not exist: %s", backupPath)
		}

		// Cleanup backup file
		defer os.Remove(backupPath)
	})

	t.Run("CreateIncrementalBackup", func(t *testing.T) {
		backupPath, err := service.createAppBackup(env.TestAppPath, "incremental")
		if err != nil {
			t.Fatalf("Failed to create backup: %v", err)
		}

		if backupPath == "" {
			t.Error("Expected non-empty backup path")
		}

		defer os.Remove(backupPath)
	})
}

// TestValidateApp tests the ValidateApp endpoint
func TestValidateApp(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	service := &AppPersonalizerService{
		db:         nil,
		n8nBaseURL: "http://localhost:5678",
		minioURL:   "http://localhost:9000",
		httpClient: &http.Client{Timeout: httpTimeout},
		logger:     NewLogger(),
	}

	t.Run("Success", func(t *testing.T) {
		reqBody := ValidateAppRequest{
			AppPath: env.TestAppPath,
			Tests:   []string{"build", "lint"},
		}

		bodyBytes, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/validate", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(service.ValidateApp)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, status, rr.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if _, ok := response["validation_results"]; !ok {
			t.Error("Expected 'validation_results' in response")
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/validate", bytes.NewBufferString("invalid-json"))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(service.ValidateApp)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, status)
		}
	})

	t.Run("MissingAppPath", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"tests": []string{"build"},
		}

		bodyBytes, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/validate", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(service.ValidateApp)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, status)
		}
	})
}

// TestRunValidationTest tests the validation test runner
func TestRunValidationTest(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	service := &AppPersonalizerService{
		logger: NewLogger(),
	}

	t.Run("BuildTest", func(t *testing.T) {
		result := service.runValidationTest(env.TestAppPath, "build")
		if result == nil {
			t.Fatal("Expected non-nil result")
		}

		if _, ok := result["success"]; !ok {
			t.Error("Expected 'success' field in result")
		}

		if _, ok := result["output"]; !ok {
			t.Error("Expected 'output' field in result")
		}
	})

	t.Run("LintTest", func(t *testing.T) {
		result := service.runValidationTest(env.TestAppPath, "lint")
		if result == nil {
			t.Fatal("Expected non-nil result")
		}
	})

	t.Run("TestTest", func(t *testing.T) {
		result := service.runValidationTest(env.TestAppPath, "test")
		if result == nil {
			t.Fatal("Expected non-nil result")
		}
	})

	t.Run("StartupTest", func(t *testing.T) {
		result := service.runValidationTest(env.TestAppPath, "startup")
		if result == nil {
			t.Fatal("Expected non-nil result")
		}
	})

	t.Run("UnknownTest", func(t *testing.T) {
		result := service.runValidationTest(env.TestAppPath, "unknown")
		if result == nil {
			t.Fatal("Expected non-nil result")
		}

		errorMsg, ok := result["error"].(string)
		if !ok {
			t.Fatal("Expected error field to be string")
		}

		if !strings.Contains(errorMsg, "Unknown test type") {
			t.Errorf("Expected 'Unknown test type' error, got: %s", errorMsg)
		}
	})
}

// TestPersonalizeApp tests the PersonalizeApp endpoint
func TestPersonalizeApp(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	service := &AppPersonalizerService{
		db:         nil,
		n8nBaseURL: "http://localhost:5678",
		minioURL:   "http://localhost:9000",
		httpClient: &http.Client{Timeout: httpTimeout},
		logger:     NewLogger(),
	}

	t.Run("InvalidJSON", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/personalize", bytes.NewBufferString("invalid-json"))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(service.PersonalizeApp)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, status)
		}
	})
}

// TestTriggerPersonalizationWorkflow tests the n8n workflow trigger
func TestTriggerPersonalizationWorkflow(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("SuccessfulTrigger", func(t *testing.T) {
		// Create a test server to mock n8n
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "triggered"}`))
		}))
		defer server.Close()

		service := &AppPersonalizerService{
			n8nBaseURL: server.URL,
			httpClient: &http.Client{Timeout: httpTimeout},
			logger:     NewLogger(),
		}

		payload := map[string]interface{}{
			"app_id":               uuid.New(),
			"personalization_type": "ui_theme",
		}

		err := service.triggerPersonalizationWorkflow(payload)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("WorkflowError", func(t *testing.T) {
		// Create a test server that returns an error
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "workflow failed"}`))
		}))
		defer server.Close()

		service := &AppPersonalizerService{
			n8nBaseURL: server.URL,
			httpClient: &http.Client{Timeout: httpTimeout},
			logger:     NewLogger(),
		}

		payload := map[string]interface{}{
			"app_id": uuid.New(),
		}

		err := service.triggerPersonalizationWorkflow(payload)
		if err == nil {
			t.Error("Expected error, got nil")
		}
	})
}

// TestLoggerFunctions tests the Logger implementation
func TestLoggerFunctions(t *testing.T) {
	t.Run("NewLogger", func(t *testing.T) {
		logger := NewLogger()
		if logger == nil {
			t.Fatal("Expected non-nil logger")
		}

		if logger.Logger == nil {
			t.Error("Expected Logger to have a logger instance")
		}
	})

	t.Run("LogMethods", func(t *testing.T) {
		logger := NewLogger()

		// These should not panic
		logger.Info("test info message")
		logger.Warn("test warn message", nil)
		logger.Error("test error message", nil)
	})
}

// TestNewAppPersonalizerService tests service creation
func TestNewAppPersonalizerService(t *testing.T) {
	t.Run("CreateService", func(t *testing.T) {
		service := NewAppPersonalizerService(nil, "http://localhost:5678", "http://localhost:9000")

		if service == nil {
			t.Fatal("Expected non-nil service")
		}

		if service.n8nBaseURL != "http://localhost:5678" {
			t.Errorf("Expected n8nBaseURL 'http://localhost:5678', got '%s'", service.n8nBaseURL)
		}

		if service.minioURL != "http://localhost:9000" {
			t.Errorf("Expected minioURL 'http://localhost:9000', got '%s'", service.minioURL)
		}

		if service.httpClient == nil {
			t.Error("Expected non-nil httpClient")
		}

		if service.logger == nil {
			t.Error("Expected non-nil logger")
		}
	})
}

// TestRouterIntegration tests the full router setup
func TestRouterIntegration(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	service := NewAppPersonalizerService(nil, "http://localhost:5678", "http://localhost:9000")
	router := createTestRouter(service)

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
	}{
		{"Health", "GET", "/health", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest(tt.method, tt.path, nil)
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			t.Logf("%s %s returned status %d", tt.method, tt.path, rr.Code)
		})
	}
}
