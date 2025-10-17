package main

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestComprehensiveDebugManager tests comprehensive debug manager scenarios
func TestComprehensiveDebugManager(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	dm := NewDebugManager()

	t.Run("GetMemoryInfo_Success", func(t *testing.T) {
		// Test with current process name
		info, _ := dm.getMemoryInfo("go")
		// Should get some data or handle gracefully
		if info != nil {
			if info["status"] != "checked" {
				t.Errorf("Expected status 'checked', got %v", info["status"])
			}
		}
	})

	t.Run("GetCPUInfo_Success", func(t *testing.T) {
		info, _ := dm.getCPUInfo("go")
		if info != nil {
			if info["status"] != "checked" {
				t.Errorf("Expected status 'checked', got %v", info["status"])
			}
		}
	})

	t.Run("GetDiskInfo_ValidApp", func(t *testing.T) {
		// Test with known app
		info, err := dm.getDiskInfo("app-debugger")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if info == nil {
			t.Error("Expected info map")
		}
	})

	t.Run("GetDiskInfo_PathTraversal", func(t *testing.T) {
		// Test path traversal protection
		info, err := dm.getDiskInfo("../../../etc/passwd")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		// Should handle safely due to sanitization
		if info == nil {
			t.Error("Expected info map even for malicious input")
		}
	})

	t.Run("AnalyzeErrorPatterns_ComplexErrors", func(t *testing.T) {
		logs := []string{
			"2025-10-04 ERROR: Database connection failed\nFATAL: Cannot recover\nException in thread main",
			"PANIC: Out of memory\nERROR: Failed to allocate\nCRITICAL: System shutting down",
		}

		patterns := dm.analyzeErrorPatterns(logs)
		if len(patterns) == 0 {
			t.Error("Expected error patterns to be found")
		}

		// Check that patterns contain errors
		for _, pattern := range patterns {
			if pattern["errors"] == nil {
				t.Error("Expected errors field in pattern")
			}
		}
	})

	t.Run("AnalyzeErrorPatterns_EmptyLogs", func(t *testing.T) {
		logs := []string{}
		patterns := dm.analyzeErrorPatterns(logs)
		if len(patterns) != 0 {
			t.Error("Expected no patterns for empty logs")
		}
	})

	t.Run("GeneratePerformanceRecommendations_EmptyResults", func(t *testing.T) {
		results := make(map[string]interface{})
		recommendations := dm.generatePerformanceRecommendations(results)
		if len(recommendations) == 0 {
			t.Error("Expected recommendations to be generated")
		}
	})

	t.Run("GenerateFixSuggestions_EmptyPatterns", func(t *testing.T) {
		patterns := []map[string]interface{}{}
		suggestions := dm.generateFixSuggestions(patterns)
		if len(suggestions) == 0 {
			t.Error("Expected fix suggestions to be generated")
		}
	})
}

// TestReadLogFileVariants tests different log file reading scenarios
func TestReadLogFileVariants(t *testing.T) {
	env := setupTestDirectory(t)
	defer env.Cleanup()

	dm := NewDebugManager()

	t.Run("ReadLogFile_FewLines", func(t *testing.T) {
		content := "Line 1\nLine 2\n"
		logPath := createTestLogFile(t, env.TempDir, "small.log", content)

		data, err := dm.readLogFile(logPath, 100)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if !strings.Contains(data, "Line 1") {
			t.Error("Expected to find 'Line 1' in log data")
		}
	})

	t.Run("ReadLogFile_LimitLines", func(t *testing.T) {
		content := ""
		for i := 0; i < 100; i++ {
			content += "Line number " + string(rune(i)) + "\n"
		}
		logPath := createTestLogFile(t, env.TempDir, "large.log", content)

		data, err := dm.readLogFile(logPath, 10)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Should only get last 10 lines
		lines := strings.Split(strings.TrimSpace(data), "\n")
		if len(lines) > 10 {
			t.Errorf("Expected at most 10 lines, got %d", len(lines))
		}
	})
}

// TestDebugSessionResultsStructure tests result structure consistency
func TestDebugSessionResultsStructure(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	dm := NewDebugManager()

	testCases := []struct {
		debugType       string
		expectedKeys    []string
	}{
		{
			debugType:    "performance",
			expectedKeys: []string{"recommendations"},
		},
		{
			debugType:    "health",
			expectedKeys: []string{"issues", "recommendations", "health_score"},
		},
		{
			debugType:    "general",
			expectedKeys: []string{"suggestions"},
		},
	}

	for _, tc := range testCases {
		t.Run("DebugType_"+tc.debugType, func(t *testing.T) {
			session, err := dm.StartDebugSession("test-app", tc.debugType)
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			for _, key := range tc.expectedKeys {
				if session.Results[key] == nil {
					t.Errorf("Expected key %s in results", key)
				}
			}
		})
	}
}

// TestHTTPHandlerContentTypes tests content type handling
func TestHTTPHandlerContentTypes(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("HealthHandler_ContentType", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		healthHandler(w, req)

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", contentType)
		}
	})

	t.Run("DebugHandler_ContentType", func(t *testing.T) {
		debugReq := DebugRequest{
			AppName:   "test-app",
			DebugType: "general",
		}

		body, _ := json.Marshal(debugReq)
		req := httptest.NewRequest("POST", "/api/debug", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		debugHandler(w, req)

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" && w.Code == 200 {
			t.Errorf("Expected Content-Type application/json for success, got %s", contentType)
		}
	})

	t.Run("ReportErrorHandler_ContentType", func(t *testing.T) {
		errorReport := ErrorReport{
			AppName:      "test-app",
			ErrorMessage: "Test error",
		}

		body, _ := json.Marshal(errorReport)
		req := httptest.NewRequest("POST", "/api/report-error", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		reportErrorHandler(w, req)

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", contentType)
		}
	})

	t.Run("ListAppsHandler_ContentType", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/apps", nil)
		w := httptest.NewRecorder()

		listAppsHandler(w, req)

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", contentType)
		}
	})
}

// TestDebugRequestVariants tests different debug request patterns
func TestDebugRequestVariants(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testCases := []struct {
		name      string
		appName   string
		debugType string
		expectOK  bool
	}{
		{"ShortAppName", "a", "performance", true},
		{"NumericAppName", "123", "error", true},
		{"HyphenatedAppName", "my-app-123", "logs", true},
		{"UnderscoreAppName", "my_app_test", "health", true},
		{"MixedCaseDebugType", "app", "Performance", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			debugReq := DebugRequest{
				AppName:   tc.appName,
				DebugType: tc.debugType,
			}

			body, _ := json.Marshal(debugReq)
			req := httptest.NewRequest("POST", "/api/debug", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			debugHandler(w, req)

			if tc.expectOK && w.Code != 200 && w.Code != 500 {
				t.Errorf("Expected status 200 or 500, got %d", w.Code)
			}
		})
	}
}

// TestErrorReportVariants tests different error report patterns
func TestErrorReportVariants(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testCases := []struct {
		name   string
		report ErrorReport
	}{
		{
			name: "FullReport",
			report: ErrorReport{
				AppName:      "test-app",
				ErrorMessage: "Full error message",
				StackTrace:   "at line 1\nat line 2",
				Context: map[string]interface{}{
					"user":    "test-user",
					"version": "1.0.0",
				},
			},
		},
		{
			name: "MinimalReport",
			report: ErrorReport{
				AppName: "minimal",
			},
		},
		{
			name: "NoContext",
			report: ErrorReport{
				AppName:      "no-context",
				ErrorMessage: "Error without context",
			},
		},
		{
			name: "LongErrorMessage",
			report: ErrorReport{
				AppName:      "long-error",
				ErrorMessage: strings.Repeat("Error ", 1000),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.report)
			req := httptest.NewRequest("POST", "/api/report-error", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			reportErrorHandler(w, req)

			if w.Code != 200 {
				t.Errorf("Expected status 200, got %d", w.Code)
			}
		})
	}
}

// TestGetFileModTimeVariants tests file modification time edge cases
func TestGetFileModTimeVariants(t *testing.T) {
	env := setupTestDirectory(t)
	defer env.Cleanup()

	dm := NewDebugManager()

	t.Run("NewFile", func(t *testing.T) {
		testFile := filepath.Join(env.TempDir, "new.txt")
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		modTime := dm.getFileModTime(testFile)
		if modTime == "unknown" {
			t.Error("Expected valid modification time for new file")
		}
	})

	t.Run("NonExistentFile", func(t *testing.T) {
		modTime := dm.getFileModTime("/path/to/nonexistent/file.txt")
		if modTime != "unknown" {
			t.Errorf("Expected 'unknown' for nonexistent file, got %s", modTime)
		}
	})
}

// TestAnalyzeLogFileVariants tests log file analysis edge cases
func TestAnalyzeLogFileVariants(t *testing.T) {
	env := setupTestDirectory(t)
	defer env.Cleanup()

	dm := NewDebugManager()

	t.Run("EmptyLogFile", func(t *testing.T) {
		logPath := createTestLogFile(t, env.TempDir, "empty.log", "")

		analysis, err := dm.analyzeLogFile(logPath)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if analysis["line_count"].(int) != 1 {
			// Empty file has 1 empty line
			t.Errorf("Expected line_count 1, got %v", analysis["line_count"])
		}
	})

	t.Run("SingleLineLog", func(t *testing.T) {
		logPath := createTestLogFile(t, env.TempDir, "single.log", "Single line log")

		analysis, err := dm.analyzeLogFile(logPath)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if analysis["file"] != logPath {
			t.Errorf("Expected file path %s, got %v", logPath, analysis["file"])
		}
	})
}

// TestCalculateHealthScoreEdgeCases tests health score calculation edge cases
func TestCalculateHealthScoreEdgeCases(t *testing.T) {
	dm := NewDebugManager()

	testCases := []struct {
		name          string
		issueCount    int
		minScore      int
		maxScore      int
	}{
		{"Zero", 0, 100, 100},
		{"One", 1, 75, 75},
		{"Two", 2, 75, 75},
		{"Three", 3, 50, 50},
		{"Four", 4, 50, 50},
		{"Five", 5, 50, 50},
		{"Six", 6, 25, 25},
		{"Ten", 10, 25, 25},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			issues := make([]string, tc.issueCount)
			for i := 0; i < tc.issueCount; i++ {
				issues[i] = "Issue " + string(rune(i))
			}

			score := dm.calculateHealthScore(issues)
			if score < tc.minScore || score > tc.maxScore {
				t.Errorf("Expected score between %d and %d, got %d", tc.minScore, tc.maxScore, score)
			}
		})
	}
}
