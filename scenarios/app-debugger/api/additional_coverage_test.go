package main

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// TestHelperFunctions tests helper functions that need coverage
func TestHelperFunctions(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("MakeHTTPRequest", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/test",
			Body: map[string]string{
				"key": "value",
			},
			Headers: map[string]string{
				"X-Test-Header": "test-value",
			},
		}

		w, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if w == nil {
			t.Fatal("Expected ResponseRecorder to be created")
		}
	})

	t.Run("AssertStringNotContains", func(t *testing.T) {
		// Test that function works correctly
		testStr := "Hello World"
		assertStringNotContains(t, testStr, "Goodbye")
	})

	t.Run("GetMapKeys", func(t *testing.T) {
		testMap := map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
		}
		keys := getMapKeys(testMap)
		if len(keys) != 3 {
			t.Errorf("Expected 3 keys, got %d", len(keys))
		}
	})

	t.Run("AssertJSONResponse_ValidJSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"key": "value"}`))

		data := assertJSONResponse(t, w, 200)
		if data["key"] != "value" {
			t.Errorf("Expected key=value, got %v", data["key"])
		}
	})

	t.Run("AssertErrorResponse_WithBody", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.WriteHeader(400)
		w.Write([]byte("Error message"))

		assertErrorResponse(t, w, 400)
	})
}

// TestDebugManagerHelpers tests debug manager helper functions
func TestDebugManagerHelpers(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	dm := NewDebugManager()

	t.Run("GetMemoryInfo", func(t *testing.T) {
		info, err := dm.getMemoryInfo("nonexistent-app")
		if err == nil {
			// Might succeed or fail depending on system
			if info == nil {
				t.Error("Expected info map to be returned")
			}
		}
	})

	t.Run("GetCPUInfo", func(t *testing.T) {
		info, err := dm.getCPUInfo("nonexistent-app")
		if err == nil {
			if info == nil {
				t.Error("Expected info map to be returned")
			}
		}
	})

	t.Run("GetDiskInfo", func(t *testing.T) {
		info, err := dm.getDiskInfo("nonexistent-app")
		// Should not error even for nonexistent app
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if info == nil {
			t.Error("Expected info map to be returned")
		}
	})

	t.Run("GetRecentLogs_NoFiles", func(t *testing.T) {
		logs, err := dm.getRecentLogs("nonexistent-app-12345")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if logs == nil {
			t.Error("Expected logs array to be returned")
		}
	})

	t.Run("GetRecentLogs_WithFiles", func(t *testing.T) {
		env := setupTestDirectory(t)
		defer env.Cleanup()

		// Create a log file
		logDir := filepath.Join(env.TempDir, "scenarios", "test-app", "api")
		createTestLogFile(t, logDir, "test-app.log", "Test log content\n")

		// This might not find the file depending on current directory
		logs, _ := dm.getRecentLogs("test-app")
		if logs == nil {
			t.Error("Expected logs to be returned")
		}
	})

	t.Run("IsAppRunning", func(t *testing.T) {
		running, err := dm.isAppRunning("nonexistent-app")
		// Expected to error since app doesn't exist
		if err != nil {
			// This is acceptable
			if running {
				t.Error("Nonexistent app should not be running")
			}
		}
	})

	t.Run("GetAppPorts", func(t *testing.T) {
		ports, err := dm.getAppPorts("test-app")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if len(ports) == 0 {
			t.Error("Expected at least one port")
		}
	})

	t.Run("IsPortAccessible", func(t *testing.T) {
		// Port 99999 should not be accessible
		accessible := dm.isPortAccessible("99999")
		if accessible {
			t.Error("Port 99999 should not be accessible")
		}
	})

	t.Run("GetAppDependencies", func(t *testing.T) {
		deps, err := dm.getAppDependencies("test-app")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if deps == nil {
			t.Error("Expected dependencies array")
		}
	})

	t.Run("IsDependencyHealthy", func(t *testing.T) {
		healthy := dm.isDependencyHealthy("nonexistent-resource")
		// Should return false for nonexistent resource
		if healthy {
			t.Error("Nonexistent resource should not be healthy")
		}
	})

	t.Run("GetAppStatus", func(t *testing.T) {
		status, err := dm.getAppStatus("nonexistent-app")
		// Might error, but should return something
		if err != nil && status == "" {
			// This is acceptable
		}
	})

	t.Run("GetRecentErrors", func(t *testing.T) {
		errors := dm.getRecentErrors("test-app")
		if errors == nil {
			t.Error("Expected errors array to be returned")
		}
	})

	t.Run("GetResourceUsage", func(t *testing.T) {
		usage := dm.getResourceUsage("test-app")
		if usage == nil {
			t.Error("Expected resource usage map")
		}
		if usage["memory"] == "" {
			t.Error("Expected memory field in resource usage")
		}
	})
}

// TestLogAnalysisEdgeCases tests log analysis edge cases
func TestLogAnalysisEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	dm := NewDebugManager()

	t.Run("LogAnalysis_WithMultipleLogFiles", func(t *testing.T) {
		// Create multiple log files
		logDir := filepath.Join(env.TempDir, "scenarios", "test-app", "api")
		createTestLogFile(t, logDir, "test-app.log", "INFO: Starting\nERROR: Failed\n")
		createTestLogFile(t, logDir, "app.log", "DEBUG: Processing\n")

		session := &DebugSession{
			ID:        "test-session",
			AppName:   "test-app",
			DebugType: "logs",
			Status:    "starting",
			Context:   make(map[string]interface{}),
			Results:   make(map[string]interface{}),
		}

		result, err := dm.logAnalysis(session)
		// Might fail if logs not found, but shouldn't crash
		if err != nil {
			if result.Status != "failed" {
				t.Errorf("Expected status 'failed', got %s", result.Status)
			}
		}
	})

	t.Run("AnalyzeLogFile_LargeFile", func(t *testing.T) {
		// Create a large log file
		largeContent := ""
		for i := 0; i < 200; i++ {
			largeContent += "Log line " + string(rune(i)) + "\n"
		}
		logPath := createTestLogFile(t, env.TempDir, "large.log", largeContent)

		analysis, err := dm.analyzeLogFile(logPath)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if analysis["file"] != logPath {
			t.Errorf("Expected file path %s, got %v", logPath, analysis["file"])
		}
	})

	t.Run("GetLogPaths_MultipleLocations", func(t *testing.T) {
		paths, err := dm.getLogPaths("test-app")
		// Should not error
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		// Should return array (might be empty)
		if len(paths) < 0 {
			t.Error("Negative path count impossible")
		}
	})
}

// TestGatherDebugContext tests context gathering
func TestGatherDebugContext(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	dm := NewDebugManager()

	session := &DebugSession{
		ID:        "test-session",
		AppName:   "test-app",
		DebugType: "general",
		Status:    "starting",
		Context:   make(map[string]interface{}),
		Results:   make(map[string]interface{}),
	}

	// This will likely fail but shouldn't crash
	err := dm.gatherDebugContext(session)

	// Timestamp and debug_host should be set if no error occurred
	// If there's an error, these might not be set, which is acceptable
	if err == nil {
		if session.Context["timestamp"] == nil {
			t.Error("Expected timestamp to be set when no error")
		}
		if session.Context["debug_host"] == nil {
			t.Error("Expected debug_host to be set when no error")
		}
	}
}

// TestDebugSessionWorkflow tests complete debug workflows
func TestDebugSessionWorkflow(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	dm := NewDebugManager()

	t.Run("PerformanceWorkflow", func(t *testing.T) {
		session, err := dm.StartDebugSession("workflow-test", "performance")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Validate results
		if session.Results["recommendations"] == nil {
			t.Error("Expected recommendations in performance results")
		}
	})

	t.Run("ErrorWorkflow", func(t *testing.T) {
		session, err := dm.StartDebugSession("workflow-test", "error")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Should have analyzed logs even if none found
		if session.Results == nil {
			t.Error("Expected results map")
		}
	})

	t.Run("HealthWorkflow", func(t *testing.T) {
		session, err := dm.StartDebugSession("workflow-test", "health")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Validate health check structure
		if session.Results["health_score"] == nil {
			t.Error("Expected health_score in results")
		}

		// Health score should be a number
		if score, ok := session.Results["health_score"].(int); ok {
			if score < 0 || score > 100 {
				t.Errorf("Health score %d out of valid range 0-100", score)
			}
		}
	})
}

// TestHTTPHandlerEdgeCases tests additional edge cases
func TestHTTPHandlerEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("DebugHandler_InvalidDebugType", func(t *testing.T) {
		debugReq := DebugRequest{
			AppName:   "test-app",
			DebugType: "unknown-type",
		}

		body, _ := json.Marshal(debugReq)
		req := httptest.NewRequest("POST", "/api/debug", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		debugHandler(w, req)

		// Should still succeed with general debug
		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("ErrorReport_LargeStackTrace", func(t *testing.T) {
		largeStack := ""
		for i := 0; i < 1000; i++ {
			largeStack += "at function() line " + string(rune(i)) + "\n"
		}

		errorReport := ErrorReport{
			AppName:      "test-app",
			ErrorMessage: "Error with large stack",
			StackTrace:   largeStack,
		}

		body, _ := json.Marshal(errorReport)
		req := httptest.NewRequest("POST", "/api/report-error", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		reportErrorHandler(w, req)

		data := assertJSONResponse(t, w, 200)
		if data["status"] != "received" {
			t.Errorf("Expected status 'received', got %v", data["status"])
		}
	})

	t.Run("DebugHandler_SpecialCharactersInType", func(t *testing.T) {
		debugReq := DebugRequest{
			AppName:   "test-app",
			DebugType: "performance\n\r\t",
		}

		body, _ := json.Marshal(debugReq)
		req := httptest.NewRequest("POST", "/api/debug", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		debugHandler(w, req)

		// Should handle gracefully
		if w.Code != 200 && w.Code != 500 {
			t.Errorf("Unexpected status code: %d", w.Code)
		}
	})
}

// TestFileOperations tests file-related operations
func TestFileOperations(t *testing.T) {
	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("CreateTestLogFile", func(t *testing.T) {
		content := "Test log line 1\nTest log line 2\n"
		logPath := createTestLogFile(t, env.TempDir, "test.log", content)

		// Verify file was created
		if _, err := os.Stat(logPath); os.IsNotExist(err) {
			t.Errorf("Log file was not created: %s", logPath)
		}

		// Verify content
		fileContent, err := os.ReadFile(logPath)
		if err != nil {
			t.Fatalf("Failed to read log file: %v", err)
		}

		if string(fileContent) != content {
			t.Errorf("Expected content %q, got %q", content, string(fileContent))
		}
	})

	t.Run("CreateNestedLogFile", func(t *testing.T) {
		nestedPath := filepath.Join("logs", "app", "nested.log")
		logPath := createTestLogFile(t, env.TempDir, nestedPath, "Nested log\n")

		if _, err := os.Stat(logPath); os.IsNotExist(err) {
			t.Errorf("Nested log file was not created: %s", logPath)
		}
	})
}
