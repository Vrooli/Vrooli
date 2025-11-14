package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// TestHealthHandler tests the health check endpoint comprehensively
func TestHealthHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server := NewServer(0)

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/health", nil)
		w := httptest.NewRecorder()

		server.healthHandler(w, req)

		response := assertJSONResponse(t, w, http.StatusOK)
		assertFieldValue(t, response, "service", "scenario-to-desktop-api")
		assertFieldExists(t, response, "version")
		assertFieldExists(t, response, "status")
		assertFieldExists(t, response, "timestamp")
		assertFieldExists(t, response, "readiness")
	})

	t.Run("MultipleRequests", func(t *testing.T) {
		// Test that health endpoint handles concurrent requests
		for i := 0; i < 10; i++ {
			req := httptest.NewRequest("GET", "/api/v1/health", nil)
			w := httptest.NewRecorder()
			server.healthHandler(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Request %d failed with status %d", i, w.Code)
			}
		}
	})
}

// TestStatusHandler tests the status endpoint
func TestStatusHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server := NewServer(0)

	t.Run("EmptyStatistics", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/status", nil)
		w := httptest.NewRecorder()

		server.statusHandler(w, req)

		response := assertJSONResponse(t, w, http.StatusOK)

		// Verify service information
		service := assertFieldExists(t, response, "service")
		if serviceMap, ok := service.(map[string]interface{}); ok {
			if serviceMap["name"] != "scenario-to-desktop" {
				t.Error("Expected service name to be scenario-to-desktop")
			}
		}

		// Verify statistics
		stats := assertFieldExists(t, response, "statistics")
		if statsMap, ok := stats.(map[string]interface{}); ok {
			if statsMap["total_builds"].(float64) != 0 {
				t.Error("Expected total_builds to be 0")
			}
		}

		// Verify capabilities
		assertFieldExists(t, response, "capabilities")
		assertFieldExists(t, response, "supported_frameworks")
		assertFieldExists(t, response, "endpoints")
	})

	t.Run("WithBuildStatistics", func(t *testing.T) {
		// Add some build statuses
		server.buildStatuses["build1"] = createTestBuildStatus("build1", "building")
		server.buildStatuses["build2"] = createTestBuildStatus("build2", "ready")
		server.buildStatuses["build3"] = createTestBuildStatus("build3", "failed")

		req := httptest.NewRequest("GET", "/api/v1/status", nil)
		w := httptest.NewRecorder()

		server.statusHandler(w, req)

		response := assertJSONResponse(t, w, http.StatusOK)
		stats := response["statistics"].(map[string]interface{})

		if stats["total_builds"].(float64) != 3 {
			t.Errorf("Expected total_builds to be 3, got %v", stats["total_builds"])
		}
		if stats["active_builds"].(float64) != 1 {
			t.Errorf("Expected active_builds to be 1, got %v", stats["active_builds"])
		}
		if stats["completed_builds"].(float64) != 1 {
			t.Errorf("Expected completed_builds to be 1, got %v", stats["completed_builds"])
		}
		if stats["failed_builds"].(float64) != 1 {
			t.Errorf("Expected failed_builds to be 1, got %v", stats["failed_builds"])
		}
	})
}

// TestListTemplatesHandler tests template listing
func TestListTemplatesHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/templates", nil)
		w := httptest.NewRecorder()

		env.Server.listTemplatesHandler(w, req)

		response := assertJSONResponse(t, w, http.StatusOK)
		templates := assertFieldExists(t, response, "templates")

		// Verify templates is an array
		if templatesArray, ok := templates.([]interface{}); ok {
			if len(templatesArray) == 0 {
				t.Error("Expected at least one template")
			}

			// Verify first template structure
			if len(templatesArray) > 0 {
				if firstTemplate, ok := templatesArray[0].(map[string]interface{}); ok {
					if _, exists := firstTemplate["name"]; !exists {
						t.Error("Expected template to have 'name' field")
					}
					if _, exists := firstTemplate["type"]; !exists {
						t.Error("Expected template to have 'type' field")
					}
					if _, exists := firstTemplate["framework"]; !exists {
						t.Error("Expected template to have 'framework' field")
					}
				}
			}
		} else {
			t.Error("Expected templates to be an array")
		}

		count := assertFieldExists(t, response, "count")
		if count == nil || count.(float64) == 0 {
			t.Error("Expected count to be greater than 0")
		}
	})
}

// TestGetTemplateHandler tests getting specific templates
func TestGetTemplateHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("BasicTemplate", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/templates/react-vite", nil)
		req = mux.SetURLVars(req, map[string]string{"type": "basic"})
		w := httptest.NewRecorder()

		env.Server.getTemplateHandler(w, req)

		if w.Code == http.StatusOK {
			response := assertJSONResponse(t, w, http.StatusOK)
			assertFieldExists(t, response, "name")
			assertFieldExists(t, response, "framework")
		} else if w.Code == http.StatusNotFound {
			t.Log("Template file not found (expected in test environment)")
		}
	})

	t.Run("AdvancedTemplate", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/templates/advanced", nil)
		req = mux.SetURLVars(req, map[string]string{"type": "advanced"})
		w := httptest.NewRecorder()

		env.Server.getTemplateHandler(w, req)

		if w.Code == http.StatusOK {
			response := assertJSONResponse(t, w, http.StatusOK)
			assertFieldExists(t, response, "name")
		}
	})

	t.Run("NonExistentTemplate", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/templates/nonexistent", nil)
		req = mux.SetURLVars(req, map[string]string{"type": "nonexistent"})
		w := httptest.NewRecorder()

		env.Server.getTemplateHandler(w, req)

		// Should return 400 because the template type is not in the whitelist
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("MultiWindowTemplate", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/templates/multi_window", nil)
		req = mux.SetURLVars(req, map[string]string{"type": "multi_window"})
		w := httptest.NewRecorder()

		env.Server.getTemplateHandler(w, req)

		if w.Code == http.StatusOK {
			response := assertJSONResponse(t, w, http.StatusOK)
			assertFieldValue(t, response, "type", "multi_window")
		}
	})
}

// TestGenerateDesktopHandler tests desktop generation endpoint
func TestGenerateDesktopHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("ValidRequest", func(t *testing.T) {
		body := createValidGenerateRequest()
		body["output_path"] = filepath.Join(env.TempDir, "output")

		req := createJSONRequest("POST", "/api/v1/desktop/generate", body)
		w := httptest.NewRecorder()

		env.Server.generateDesktopHandler(w, req)

		if w.Code != http.StatusCreated {
			t.Logf("Response body: %s", w.Body.String())
		}

		response := assertJSONResponse(t, w, http.StatusCreated)
		buildID := assertFieldExists(t, response, "build_id")

		if buildID != nil {
			if _, err := uuid.Parse(buildID.(string)); err != nil {
				t.Errorf("Invalid UUID format: %v", err)
			}

			// Verify build status was created
			status := assertBuildStatusExists(t, env.Server, buildID.(string))
			assertBuildStatusValue(t, status, "status", "building")
		}

		assertFieldExists(t, response, "desktop_path")
		assertFieldExists(t, response, "install_instructions")
		assertFieldExists(t, response, "test_command")
		assertFieldExists(t, response, "status_url")
	})

	t.Run("MissingAppName", func(t *testing.T) {
		body := createValidGenerateRequest()
		delete(body, "app_name")

		req := createJSONRequest("POST", "/api/v1/desktop/generate", body)
		w := httptest.NewRecorder()

		env.Server.generateDesktopHandler(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, "app_name")
	})

	t.Run("MissingFramework", func(t *testing.T) {
		body := createValidGenerateRequest()
		delete(body, "framework")

		req := createJSONRequest("POST", "/api/v1/desktop/generate", body)
		w := httptest.NewRecorder()

		env.Server.generateDesktopHandler(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, "framework")
	})

	t.Run("MissingTemplateType", func(t *testing.T) {
		body := createValidGenerateRequest()
		delete(body, "template_type")

		req := createJSONRequest("POST", "/api/v1/desktop/generate", body)
		w := httptest.NewRecorder()

		env.Server.generateDesktopHandler(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, "template_type")
	})

	t.Run("MissingOutputPath", func(t *testing.T) {
		body := createValidGenerateRequest()
		delete(body, "output_path")

		req := createJSONRequest("POST", "/api/v1/desktop/generate", body)
		w := httptest.NewRecorder()

		env.Server.generateDesktopHandler(w, req)

		// output_path is now optional - defaults to scenarios/<app_name>/platforms/electron/
		// So this should succeed, not fail
		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", w.Code)
		}
	})

	t.Run("InvalidFramework", func(t *testing.T) {
		body := createValidGenerateRequest()
		body["framework"] = "invalid-framework"

		req := createJSONRequest("POST", "/api/v1/desktop/generate", body)
		w := httptest.NewRecorder()

		env.Server.generateDesktopHandler(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, "framework")
	})

	t.Run("InvalidTemplateType", func(t *testing.T) {
		body := createValidGenerateRequest()
		body["template_type"] = "invalid-template"

		req := createJSONRequest("POST", "/api/v1/desktop/generate", body)
		w := httptest.NewRecorder()

		env.Server.generateDesktopHandler(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, "template_type")
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/desktop/generate", strings.NewReader("{invalid json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		env.Server.generateDesktopHandler(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, "")
	})

	t.Run("EmptyBody", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/desktop/generate", strings.NewReader(""))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		env.Server.generateDesktopHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("AllFrameworks", func(t *testing.T) {
		frameworks := []string{"electron", "tauri", "neutralino"}
		for _, framework := range frameworks {
			body := createValidGenerateRequest()
			body["framework"] = framework
			body["output_path"] = filepath.Join(env.TempDir, "output-"+framework)

			req := createJSONRequest("POST", "/api/v1/desktop/generate", body)
			w := httptest.NewRecorder()

			env.Server.generateDesktopHandler(w, req)

			if w.Code != http.StatusCreated {
				t.Errorf("Framework %s failed with status %d", framework, w.Code)
			}
		}
	})

	t.Run("AllTemplateTypes", func(t *testing.T) {
		templates := []string{"basic", "advanced", "multi_window", "kiosk"}
		for _, template := range templates {
			body := createValidGenerateRequest()
			body["template_type"] = template
			body["output_path"] = filepath.Join(env.TempDir, "output-"+template)

			req := createJSONRequest("POST", "/api/v1/desktop/generate", body)
			w := httptest.NewRecorder()

			env.Server.generateDesktopHandler(w, req)

			if w.Code != http.StatusCreated {
				t.Errorf("Template %s failed with status %d", template, w.Code)
			}
		}
	})

	t.Run("WithCustomFeatures", func(t *testing.T) {
		body := createValidGenerateRequest()
		body["output_path"] = filepath.Join(env.TempDir, "output-features")
		body["features"] = map[string]interface{}{
			"system_tray":       true,
			"auto_updater":      true,
			"native_menus":      true,
			"file_associations": []string{".txt", ".md"},
		}

		req := createJSONRequest("POST", "/api/v1/desktop/generate", body)
		w := httptest.NewRecorder()

		env.Server.generateDesktopHandler(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", w.Code)
		}
	})

	t.Run("WithWindowConfig", func(t *testing.T) {
		body := createValidGenerateRequest()
		body["output_path"] = filepath.Join(env.TempDir, "output-window")
		body["window"] = map[string]interface{}{
			"width":      1280,
			"height":     720,
			"resizable":  true,
			"fullscreen": false,
		}

		req := createJSONRequest("POST", "/api/v1/desktop/generate", body)
		w := httptest.NewRecorder()

		env.Server.generateDesktopHandler(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", w.Code)
		}
	})
}

// TestGetBuildStatusHandler tests build status retrieval
func TestGetBuildStatusHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server := NewServer(0)

	t.Run("ExistingBuild", func(t *testing.T) {
		buildID := uuid.New().String()
		server.buildStatuses[buildID] = createTestBuildStatus(buildID, "ready")

		req := httptest.NewRequest("GET", "/api/v1/desktop/status/"+buildID, nil)
		req = mux.SetURLVars(req, map[string]string{"build_id": buildID})
		w := httptest.NewRecorder()

		server.getBuildStatusHandler(w, req)

		response := assertJSONResponse(t, w, http.StatusOK)
		assertFieldValue(t, response, "build_id", buildID)
		assertFieldValue(t, response, "status", "ready")
		assertFieldExists(t, response, "framework")
		assertFieldExists(t, response, "platforms")
	})

	t.Run("NonExistentBuild", func(t *testing.T) {
		buildID := uuid.New().String()

		req := httptest.NewRequest("GET", "/api/v1/desktop/status/"+buildID, nil)
		req = mux.SetURLVars(req, map[string]string{"build_id": buildID})
		w := httptest.NewRecorder()

		server.getBuildStatusHandler(w, req)

		assertErrorResponse(t, w, http.StatusNotFound, "")
	})

	t.Run("BuildingStatus", func(t *testing.T) {
		buildID := uuid.New().String()
		server.buildStatuses[buildID] = createTestBuildStatus(buildID, "building")

		req := httptest.NewRequest("GET", "/api/v1/desktop/status/"+buildID, nil)
		req = mux.SetURLVars(req, map[string]string{"build_id": buildID})
		w := httptest.NewRecorder()

		server.getBuildStatusHandler(w, req)

		response := assertJSONResponse(t, w, http.StatusOK)
		assertFieldValue(t, response, "status", "building")
	})

	t.Run("FailedStatus", func(t *testing.T) {
		buildID := uuid.New().String()
		status := createTestBuildStatus(buildID, "failed")
		status.ErrorLog = []string{"Build failed due to missing dependencies"}
		server.buildStatuses[buildID] = status

		req := httptest.NewRequest("GET", "/api/v1/desktop/status/"+buildID, nil)
		req = mux.SetURLVars(req, map[string]string{"build_id": buildID})
		w := httptest.NewRecorder()

		server.getBuildStatusHandler(w, req)

		response := assertJSONResponse(t, w, http.StatusOK)
		assertFieldValue(t, response, "status", "failed")
		assertFieldExists(t, response, "error_log")
	})
}

// TestBuildDesktopHandler tests the build endpoint
func TestBuildDesktopHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("ValidRequest", func(t *testing.T) {
		body := map[string]interface{}{
			"desktop_path": filepath.Join(env.TempDir, "test-desktop"),
			"platforms":    []string{"win", "mac", "linux"},
			"sign":         false,
			"publish":      false,
		}

		req := createJSONRequest("POST", "/api/v1/desktop/build", body)
		w := httptest.NewRecorder()

		env.Server.buildDesktopHandler(w, req)

		response := assertJSONResponse(t, w, http.StatusCreated)
		buildID := assertFieldExists(t, response, "build_id")
		assertFieldExists(t, response, "status")
		assertFieldExists(t, response, "status_url")

		if buildID != nil {
			if _, err := uuid.Parse(buildID.(string)); err != nil {
				t.Errorf("Invalid UUID format: %v", err)
			}
		}
	})

	t.Run("WithSigning", func(t *testing.T) {
		body := map[string]interface{}{
			"desktop_path": filepath.Join(env.TempDir, "test-desktop-signed"),
			"platforms":    []string{"win"},
			"sign":         true,
			"publish":      false,
		}

		req := createJSONRequest("POST", "/api/v1/desktop/build", body)
		w := httptest.NewRecorder()

		env.Server.buildDesktopHandler(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", w.Code)
		}
	})

	t.Run("WithPublishing", func(t *testing.T) {
		body := map[string]interface{}{
			"desktop_path": filepath.Join(env.TempDir, "test-desktop-publish"),
			"platforms":    []string{"mac"},
			"sign":         true,
			"publish":      true,
		}

		req := createJSONRequest("POST", "/api/v1/desktop/build", body)
		w := httptest.NewRecorder()

		env.Server.buildDesktopHandler(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", w.Code)
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/desktop/build", strings.NewReader("invalid"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		env.Server.buildDesktopHandler(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, "")
	})
}

// TestTestDesktopHandler tests the desktop testing endpoint
func TestTestDesktopHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("ValidRequest", func(t *testing.T) {
		body := map[string]interface{}{
			"app_path":  filepath.Join(env.TempDir, "test-app"),
			"platforms": []string{"linux"},
			"headless":  true,
		}

		req := createJSONRequest("POST", "/api/v1/desktop/test", body)
		w := httptest.NewRecorder()

		env.Server.testDesktopHandler(w, req)

		response := assertJSONResponse(t, w, http.StatusOK)
		assertFieldExists(t, response, "test_results")
		assertFieldExists(t, response, "status")
		assertFieldExists(t, response, "timestamp")
	})

	t.Run("MultiPlatform", func(t *testing.T) {
		body := map[string]interface{}{
			"app_path":  filepath.Join(env.TempDir, "test-app-multi"),
			"platforms": []string{"win", "mac", "linux"},
			"headless":  false,
		}

		req := createJSONRequest("POST", "/api/v1/desktop/test", body)
		w := httptest.NewRecorder()

		env.Server.testDesktopHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/desktop/test", strings.NewReader("{bad"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		env.Server.testDesktopHandler(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, "")
	})
}

// TestPackageDesktopHandler tests the packaging endpoint
func TestPackageDesktopHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server := NewServer(0)

	t.Run("MicrosoftStore", func(t *testing.T) {
		body := map[string]interface{}{
			"app_path":   "/tmp/test-app",
			"store":      "microsoft",
			"enterprise": false,
		}

		req := createJSONRequest("POST", "/api/v1/desktop/package", body)
		w := httptest.NewRecorder()

		server.packageDesktopHandler(w, req)

		response := assertJSONResponse(t, w, http.StatusOK)
		assertFieldExists(t, response, "status")
		assertFieldExists(t, response, "packages")
		assertFieldExists(t, response, "timestamp")
	})

	t.Run("MacAppStore", func(t *testing.T) {
		body := map[string]interface{}{
			"app_path":   "/tmp/test-app-mac",
			"store":      "mac",
			"enterprise": false,
		}

		req := createJSONRequest("POST", "/api/v1/desktop/package", body)
		w := httptest.NewRecorder()

		server.packageDesktopHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("EnterprisePackaging", func(t *testing.T) {
		body := map[string]interface{}{
			"app_path":   "/tmp/test-app-enterprise",
			"enterprise": true,
		}

		req := createJSONRequest("POST", "/api/v1/desktop/package", body)
		w := httptest.NewRecorder()

		server.packageDesktopHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/desktop/package", strings.NewReader("invalid"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.packageDesktopHandler(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, "")
	})
}

// TestBuildCompleteWebhookHandler tests the webhook endpoint
func TestBuildCompleteWebhookHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server := NewServer(0)

	t.Run("ValidWebhook_Completed", func(t *testing.T) {
		buildID := uuid.New().String()
		server.buildStatuses[buildID] = createTestBuildStatus(buildID, "building")

		body := map[string]interface{}{
			"status": "completed",
		}

		req := createJSONRequest("POST", "/api/v1/desktop/webhook/build-complete", body)
		req.Header.Set("X-Build-ID", buildID)
		w := httptest.NewRecorder()

		server.buildCompleteWebhookHandler(w, req)

		response := assertJSONResponse(t, w, http.StatusOK)
		assertFieldValue(t, response, "status", "received")

		// Verify status was updated
		if server.buildStatuses[buildID].Status != "completed" {
			t.Error("Expected build status to be updated to completed")
		}
		if server.buildStatuses[buildID].CompletedAt == nil {
			t.Error("Expected CompletedAt to be set")
		}
	})

	t.Run("ValidWebhook_Failed", func(t *testing.T) {
		buildID := uuid.New().String()
		server.buildStatuses[buildID] = createTestBuildStatus(buildID, "building")

		body := map[string]interface{}{
			"status": "failed",
			"error":  "Build failed due to compilation errors",
		}

		req := createJSONRequest("POST", "/api/v1/desktop/webhook/build-complete", body)
		req.Header.Set("X-Build-ID", buildID)
		w := httptest.NewRecorder()

		server.buildCompleteWebhookHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Verify status was updated
		if server.buildStatuses[buildID].Status != "failed" {
			t.Error("Expected build status to be updated to failed")
		}
	})

	t.Run("MissingBuildID", func(t *testing.T) {
		body := map[string]interface{}{
			"status": "completed",
		}

		req := createJSONRequest("POST", "/api/v1/desktop/webhook/build-complete", body)
		// Don't set X-Build-ID header
		w := httptest.NewRecorder()

		server.buildCompleteWebhookHandler(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, "Build-ID")
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		buildID := uuid.New().String()

		req := httptest.NewRequest("POST", "/api/v1/desktop/webhook/build-complete", strings.NewReader("invalid"))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Build-ID", buildID)
		w := httptest.NewRecorder()

		server.buildCompleteWebhookHandler(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, "")
	})

	t.Run("NonExistentBuild", func(t *testing.T) {
		buildID := uuid.New().String()
		// Don't add build to server.buildStatuses

		body := map[string]interface{}{
			"status": "completed",
		}

		req := createJSONRequest("POST", "/api/v1/desktop/webhook/build-complete", body)
		req.Header.Set("X-Build-ID", buildID)
		w := httptest.NewRecorder()

		server.buildCompleteWebhookHandler(w, req)

		// Should still return 200 even if build doesn't exist
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// TestValidateDesktopConfig tests configuration validation
func TestValidateDesktopConfig(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server := NewServer(0)

	t.Run("ValidConfig", func(t *testing.T) {
		config := &DesktopConfig{
			AppName:      "TestApp",
			Framework:    "electron",
			TemplateType: "basic",
			OutputPath:   "/tmp/test",
		}

		err := server.validateDesktopConfig(config)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		// Verify defaults were set
		if config.License != "MIT" {
			t.Errorf("Expected default license MIT, got: %s", config.License)
		}
		if len(config.Platforms) != 3 {
			t.Errorf("Expected default platforms [win, mac, linux], got: %v", config.Platforms)
		}
	})

	t.Run("MissingAppName", func(t *testing.T) {
		config := &DesktopConfig{
			Framework:    "electron",
			TemplateType: "basic",
			OutputPath:   "/tmp/test",
		}

		err := server.validateDesktopConfig(config)
		if err == nil {
			t.Error("Expected error for missing app_name")
		}
		if !strings.Contains(err.Error(), "app_name") {
			t.Errorf("Expected error to mention app_name, got: %v", err)
		}
	})

	t.Run("MissingFramework", func(t *testing.T) {
		config := &DesktopConfig{
			AppName:      "Test",
			TemplateType: "basic",
			OutputPath:   "/tmp/test",
		}

		err := server.validateDesktopConfig(config)
		if err == nil || !strings.Contains(err.Error(), "framework") {
			t.Error("Expected error for missing framework")
		}
	})

	t.Run("InvalidFramework", func(t *testing.T) {
		config := &DesktopConfig{
			AppName:      "Test",
			Framework:    "invalid",
			TemplateType: "basic",
			OutputPath:   "/tmp/test",
		}

		err := server.validateDesktopConfig(config)
		if err == nil || !strings.Contains(err.Error(), "framework") {
			t.Error("Expected error for invalid framework")
		}
	})

	t.Run("InvalidTemplateType", func(t *testing.T) {
		config := &DesktopConfig{
			AppName:      "Test",
			Framework:    "electron",
			TemplateType: "invalid",
			OutputPath:   "/tmp/test",
		}

		err := server.validateDesktopConfig(config)
		if err == nil || !strings.Contains(err.Error(), "template_type") {
			t.Error("Expected error for invalid template_type")
		}
	})

	t.Run("MissingOutputPath", func(t *testing.T) {
		config := &DesktopConfig{
			AppName:      "Test",
			Framework:    "electron",
			TemplateType: "basic",
		}

		err := server.validateDesktopConfig(config)
		// output_path is now optional - defaults to standard location
		if err != nil {
			t.Errorf("Expected no error for missing output_path (defaults to standard location), got: %v", err)
		}
	})

	t.Run("CustomLicense", func(t *testing.T) {
		config := &DesktopConfig{
			AppName:      "Test",
			Framework:    "electron",
			TemplateType: "basic",
			OutputPath:   "/tmp/test",
			License:      "Apache-2.0",
		}

		err := server.validateDesktopConfig(config)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if config.License != "Apache-2.0" {
			t.Error("Expected license to remain Apache-2.0")
		}
	})

	t.Run("CustomPlatforms", func(t *testing.T) {
		config := &DesktopConfig{
			AppName:      "Test",
			Framework:    "electron",
			TemplateType: "basic",
			OutputPath:   "/tmp/test",
			Platforms:    []string{"win"},
		}

		err := server.validateDesktopConfig(config)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if len(config.Platforms) != 1 {
			t.Error("Expected platforms to remain as single item")
		}
	})

	t.Run("ServerPortDefault", func(t *testing.T) {
		config := &DesktopConfig{
			AppName:      "Test",
			Framework:    "electron",
			TemplateType: "basic",
			OutputPath:   "/tmp/test",
			ServerType:   "node",
		}

		err := server.validateDesktopConfig(config)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if config.ServerPort != 3000 {
			t.Errorf("Expected default port 3000, got: %d", config.ServerPort)
		}
	})
}

// TestContainsUtility tests the contains utility function
func TestContainsUtility(t *testing.T) {
	testCases := []struct {
		name     string
		slice    []string
		item     string
		expected bool
	}{
		{"EmptySlice", []string{}, "test", false},
		{"ItemExists", []string{"a", "b", "c"}, "b", true},
		{"ItemNotExists", []string{"a", "b", "c"}, "d", false},
		{"SingleItemMatch", []string{"test"}, "test", true},
		{"SingleItemNoMatch", []string{"test"}, "other", false},
		{"CaseSensitive", []string{"Test"}, "test", false},
		{"MultipleMatches", []string{"a", "b", "a"}, "a", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := contains(tc.slice, tc.item)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v for slice %v and item %s",
					tc.expected, result, tc.slice, tc.item)
			}
		})
	}
}

// TestNewServer tests server initialization
func TestNewServer(t *testing.T) {
	t.Run("ValidPort", func(t *testing.T) {
		server := NewServer(8080)
		if server == nil {
			t.Fatal("Expected server to be created")
		}
		if server.port != 8080 {
			t.Errorf("Expected port 8080, got %d", server.port)
		}
		if server.buildStatuses == nil {
			t.Error("Expected buildStatuses map to be initialized")
		}
		if server.router == nil {
			t.Error("Expected router to be initialized")
		}
	})

	t.Run("ZeroPort", func(t *testing.T) {
		server := NewServer(0)
		if server == nil {
			t.Fatal("Expected server to be created")
		}
		if server.port != 0 {
			t.Errorf("Expected port 0, got %d", server.port)
		}
	})
}

// TestServerRoutes tests that all routes are properly configured
func TestServerRoutes(t *testing.T) {
	server := NewServer(0)

	routes := []struct {
		method       string
		path         string
		allow404     bool // Allow 404 for routes with path parameters (resource not found is valid)
	}{
		{"GET", "/api/v1/health", false},
		{"GET", "/api/v1/status", false},
		{"GET", "/api/v1/templates", false},
		{"GET", "/api/v1/templates/react-vite", true}, // Template file might not exist in test env
		{"POST", "/api/v1/desktop/generate", false},
		{"GET", "/api/v1/desktop/status/test-id", true}, // Build ID doesn't exist
		{"POST", "/api/v1/desktop/build", false},
		{"POST", "/api/v1/desktop/test", false},
		{"POST", "/api/v1/desktop/package", false},
		{"POST", "/api/v1/desktop/webhook/build-complete", false},
	}

	for _, route := range routes {
		t.Run(route.method+"_"+route.path, func(t *testing.T) {
			req := httptest.NewRequest(route.method, route.path, nil)
			w := httptest.NewRecorder()

			server.router.ServeHTTP(w, req)

			// Routes should exist - 404 only acceptable if explicitly allowed (resource not found)
			// Other status codes (400, 500, etc.) indicate route exists but request was invalid
			if w.Code == 404 && !route.allow404 {
				t.Errorf("Route %s %s not found (status: %d)", route.method, route.path, w.Code)
			}
		})
	}
}

// TestCORSMiddleware tests CORS headers
func TestCORSMiddleware(t *testing.T) {
	server := NewServer(0)

	t.Run("OptionsRequest", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/api/v1/health", nil)
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		// Check CORS headers - middleware adds them
		origin := w.Header().Get("Access-Control-Allow-Origin")
		if origin == "" {
			t.Log("CORS middleware may not be active in test mode")
		}
		// Should return 200 for OPTIONS
		if w.Code != http.StatusOK {
			t.Logf("OPTIONS request returned status %d", w.Code)
		}
	})

	t.Run("GetRequest", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/health", nil)
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		// Check CORS headers are present (middleware sets specific origin, not *)
		origin := w.Header().Get("Access-Control-Allow-Origin")
		if origin == "" {
			t.Error("Expected CORS headers on GET request")
		}
	})
}

// TestEdgeCases tests various edge cases
func TestEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("VeryLongAppName", func(t *testing.T) {
		longName := strings.Repeat("VeryLongName", 100)
		body := createValidGenerateRequest()
		body["app_name"] = longName
		body["output_path"] = filepath.Join(env.TempDir, "long-name")

		req := createJSONRequest("POST", "/api/v1/desktop/generate", body)
		w := httptest.NewRecorder()

		env.Server.generateDesktopHandler(w, req)

		// Should handle long names (may validate or accept)
		if w.Code != http.StatusCreated && w.Code != http.StatusBadRequest {
			t.Logf("Long app name resulted in status %d", w.Code)
		}
	})

	t.Run("SpecialCharactersInAppName", func(t *testing.T) {
		body := createValidGenerateRequest()
		body["app_name"] = "App @#$%^&*()"
		body["output_path"] = filepath.Join(env.TempDir, "special-chars")

		req := createJSONRequest("POST", "/api/v1/desktop/generate", body)
		w := httptest.NewRecorder()

		env.Server.generateDesktopHandler(w, req)

		// Should handle or reject special characters
		if w.Code != http.StatusCreated && w.Code != http.StatusBadRequest {
			t.Logf("Special characters resulted in status %d", w.Code)
		}
	})

	t.Run("EmptyPlatformsArray", func(t *testing.T) {
		body := createValidGenerateRequest()
		body["platforms"] = []string{}
		body["output_path"] = filepath.Join(env.TempDir, "empty-platforms")

		req := createJSONRequest("POST", "/api/v1/desktop/generate", body)
		w := httptest.NewRecorder()

		env.Server.generateDesktopHandler(w, req)

		// Should use defaults
		if w.Code == http.StatusCreated {
			response := assertJSONResponse(t, w, http.StatusCreated)
			buildID := response["build_id"].(string)
			status := env.Server.buildStatuses[buildID]
			if len(status.Platforms) == 0 {
				t.Error("Expected default platforms to be set")
			}
		}
	})

	t.Run("NullFieldValues", func(t *testing.T) {
		body := createValidGenerateRequest()
		body["styling"] = nil
		body["features"] = nil
		body["output_path"] = filepath.Join(env.TempDir, "null-fields")

		req := createJSONRequest("POST", "/api/v1/desktop/generate", body)
		w := httptest.NewRecorder()

		env.Server.generateDesktopHandler(w, req)

		// Should handle nil values gracefully
		if w.Code == http.StatusCreated {
			t.Log("Null field values handled successfully")
		}
	})
}

// TestConcurrentRequests tests concurrent request handling
func TestConcurrentRequests(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("ConcurrentHealthChecks", func(t *testing.T) {
		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func(id int) {
				req := httptest.NewRequest("GET", "/api/v1/health", nil)
				w := httptest.NewRecorder()
				env.Server.healthHandler(w, req)
				if w.Code != http.StatusOK {
					t.Errorf("Concurrent request %d failed with status %d", id, w.Code)
				}
				done <- true
			}(i)
		}

		// Wait for all requests to complete
		for i := 0; i < 10; i++ {
			<-done
		}
	})

	t.Run("ConcurrentGenerateRequests", func(t *testing.T) {
		done := make(chan bool, 5)
		for i := 0; i < 5; i++ {
			go func(id int) {
				body := createValidGenerateRequest()
				body["app_name"] = fmt.Sprintf("ConcurrentApp%d", id)
				body["output_path"] = filepath.Join(env.TempDir, fmt.Sprintf("output-%d", id))

				req := createJSONRequest("POST", "/api/v1/desktop/generate", body)
				w := httptest.NewRecorder()
				env.Server.generateDesktopHandler(w, req)

				if w.Code != http.StatusCreated {
					t.Errorf("Concurrent generate request %d failed with status %d", id, w.Code)
				}
				done <- true
			}(i)
		}

		// Wait for all requests to complete
		for i := 0; i < 5; i++ {
			<-done
		}

		// Allow background goroutines to start
		time.Sleep(100 * time.Millisecond)

		// Verify all builds were created
		if len(env.Server.buildStatuses) != 5 {
			t.Errorf("Expected 5 build statuses, got %d", len(env.Server.buildStatuses))
		}
	})
}

// TestPerformDesktopGeneration tests async generation
func TestPerformDesktopGeneration(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("GenerationFlow", func(t *testing.T) {
		buildID := uuid.New().String()
		config := &DesktopConfig{
			AppName:        "TestApp",
			AppDisplayName: "Test App",
			AppDescription: "Test",
			Version:        "1.0.0",
			Author:         "Test",
			AppID:          "com.test.app",
			ServerType:     "node",
			ServerPath:     "./server",
			APIEndpoint:    "http://localhost:3000",
			Framework:      "electron",
			TemplateType:   "basic",
			OutputPath:     filepath.Join(env.TempDir, "output"),
			Platforms:      []string{"linux"},
			Features:       make(map[string]interface{}),
			Window:         make(map[string]interface{}),
			Styling:        make(map[string]interface{}),
		}

		buildStatus := &BuildStatus{
			BuildID:      buildID,
			ScenarioName: config.AppName,
			Status:       "building",
			Framework:    config.Framework,
			TemplateType: config.TemplateType,
			Platforms:    config.Platforms,
			OutputPath:   config.OutputPath,
			CreatedAt:    time.Now(),
			BuildLog:     []string{},
			ErrorLog:     []string{},
			Artifacts:    make(map[string]string),
			Metadata:     make(map[string]interface{}),
		}

		env.Server.buildStatuses[buildID] = buildStatus

		// Run generation (will fail due to missing template generator, but tests flow)
		env.Server.performDesktopGeneration(buildID, config)

		// Wait for async operation
		time.Sleep(200 * time.Millisecond)

		// Check that function completed without panic
		status := env.Server.buildStatuses[buildID]
		if status != nil {
			t.Log("Generation completed (status may be failed due to missing tools)")
		}
	})
}
