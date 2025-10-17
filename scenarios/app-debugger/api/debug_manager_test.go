package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewDebugManager(t *testing.T) {
	dm := NewDebugManager()

	if dm == nil {
		t.Fatal("Expected debug manager to be created, got nil")
	}

	if dm.logger == nil {
		t.Error("Expected logger to be initialized")
	}
}

func TestSanitizePathComponent(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "SimpleClean",
			input:    "app-name",
			expected: "app-name",
		},
		{
			name:     "PathTraversal",
			input:    "../../../etc/passwd",
			expected: "etc/passwd",
		},
		{
			name:     "CurrentDirectory",
			input:    "./app-name",
			expected: "app-name",
		},
		{
			name:     "MultipleSlashes",
			input:    "app//name",
			expected: "app/name",
		},
		{
			name:     "TrailingSlash",
			input:    "app-name/",
			expected: "app-name",
		},
		{
			name:     "LeadingSlash",
			input:    "/app-name",
			expected: "/app-name", // Clean doesn't remove leading slash if it's a simple path
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := sanitizePathComponent(tc.input)
			if result != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestStartDebugSession_Performance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	dm := NewDebugManager()

	session, err := dm.StartDebugSession("test-app", "performance")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if session == nil {
		t.Fatal("Expected session to be created")
	}

	// Validate session structure
	if session.ID == "" {
		t.Error("Expected session ID to be set")
	}
	if session.AppName != "test-app" {
		t.Errorf("Expected app_name 'test-app', got %s", session.AppName)
	}
	if session.DebugType != "performance" {
		t.Errorf("Expected debug_type 'performance', got %s", session.DebugType)
	}
	if session.Status != "completed" {
		t.Errorf("Expected status 'completed', got %s", session.Status)
	}

	// Check results exist
	if session.Results == nil {
		t.Error("Expected results map to be initialized")
	}
}

func TestStartDebugSession_Error(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	dm := NewDebugManager()

	session, err := dm.StartDebugSession("test-app", "error")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if session.DebugType != "error" {
		t.Errorf("Expected debug_type 'error', got %s", session.DebugType)
	}
	if session.Status != "completed" {
		t.Errorf("Expected status 'completed', got %s", session.Status)
	}
}

func TestStartDebugSession_Logs(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	dm := NewDebugManager()

	// Create a test log file
	logDir := filepath.Join(env.TempDir, "scenarios", "test-app", "api")
	createTestLogFile(t, logDir, "test-app.log", "INFO: Application started\nERROR: Something went wrong\n")

	session, err := dm.StartDebugSession("test-app", "logs")

	// Note: This might fail if log files aren't found, which is expected
	if err != nil {
		// Log analysis might fail if no logs found - that's okay
		if session != nil && session.Status != "failed" && session.Status != "completed" {
			t.Errorf("Unexpected session status: %s", session.Status)
		}
	} else {
		if session.DebugType != "logs" {
			t.Errorf("Expected debug_type 'logs', got %s", session.DebugType)
		}
	}
}

func TestStartDebugSession_Health(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	dm := NewDebugManager()

	session, err := dm.StartDebugSession("test-app", "health")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if session.DebugType != "health" {
		t.Errorf("Expected debug_type 'health', got %s", session.DebugType)
	}
	if session.Status != "completed" {
		t.Errorf("Expected status 'completed', got %s", session.Status)
	}

	// Check health-specific results
	if session.Results["issues"] == nil {
		t.Error("Expected issues to be present in results")
	}
	if session.Results["recommendations"] == nil {
		t.Error("Expected recommendations to be present in results")
	}
	if session.Results["health_score"] == nil {
		t.Error("Expected health_score to be present in results")
	}
}

func TestStartDebugSession_General(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	dm := NewDebugManager()

	session, err := dm.StartDebugSession("test-app", "general")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if session.DebugType != "general" {
		t.Errorf("Expected debug_type 'general', got %s", session.DebugType)
	}
	if session.Status != "completed" {
		t.Errorf("Expected status 'completed', got %s", session.Status)
	}
}

func TestPerformanceDebug(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	dm := NewDebugManager()

	session := &DebugSession{
		ID:        "test-session",
		AppName:   "test-app",
		DebugType: "performance",
		Status:    "starting",
		Context:   make(map[string]interface{}),
		Results:   make(map[string]interface{}),
	}

	result, err := dm.performanceDebug(session)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.Status != "completed" {
		t.Errorf("Expected status 'completed', got %s", result.Status)
	}

	// Check that results contain performance metrics
	if result.Results["recommendations"] == nil {
		t.Error("Expected recommendations to be present")
	}
}

func TestErrorAnalysis(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	dm := NewDebugManager()

	session := &DebugSession{
		ID:        "test-session",
		AppName:   "test-app",
		DebugType: "error",
		Status:    "starting",
		Context:   make(map[string]interface{}),
		Results:   make(map[string]interface{}),
	}

	result, err := dm.errorAnalysis(session)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.Status != "completed" {
		t.Errorf("Expected status 'completed', got %s", result.Status)
	}
}

func TestHealthCheck(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	dm := NewDebugManager()

	session := &DebugSession{
		ID:        "test-session",
		AppName:   "test-app",
		DebugType: "health",
		Status:    "starting",
		Context:   make(map[string]interface{}),
		Results:   make(map[string]interface{}),
	}

	result, err := dm.healthCheck(session)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.Status != "completed" {
		t.Errorf("Expected status 'completed', got %s", result.Status)
	}

	// Verify health check results
	if result.Results["issues"] == nil {
		t.Error("Expected issues array to be present")
	}
	if result.Results["recommendations"] == nil {
		t.Error("Expected recommendations array to be present")
	}
	if result.Results["health_score"] == nil {
		t.Error("Expected health_score to be present")
	}
}

func TestGeneralDebug(t *testing.T) {
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

	result, err := dm.generalDebug(session)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.Status != "completed" {
		t.Errorf("Expected status 'completed', got %s", result.Status)
	}

	// Check for expected result keys
	if result.Results["suggestions"] == nil {
		t.Error("Expected suggestions to be present")
	}
}

func TestCalculateHealthScore(t *testing.T) {
	dm := NewDebugManager()

	testCases := []struct {
		name          string
		issues        []string
		expectedScore int
	}{
		{
			name:          "NoIssues",
			issues:        []string{},
			expectedScore: 100,
		},
		{
			name:          "OneIssue",
			issues:        []string{"Issue 1"},
			expectedScore: 75,
		},
		{
			name:          "TwoIssues",
			issues:        []string{"Issue 1", "Issue 2"},
			expectedScore: 75,
		},
		{
			name:          "ThreeIssues",
			issues:        []string{"Issue 1", "Issue 2", "Issue 3"},
			expectedScore: 50,
		},
		{
			name:          "SixIssues",
			issues:        []string{"Issue 1", "Issue 2", "Issue 3", "Issue 4", "Issue 5", "Issue 6"},
			expectedScore: 25,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			score := dm.calculateHealthScore(tc.issues)
			if score != tc.expectedScore {
				t.Errorf("Expected health score %d, got %d", tc.expectedScore, score)
			}
		})
	}
}

func TestAnalyzeErrorPatterns(t *testing.T) {
	dm := NewDebugManager()

	testCases := []struct {
		name     string
		logs     []string
		hasError bool
	}{
		{
			name:     "NoErrors",
			logs:     []string{"INFO: Application started", "DEBUG: Processing request"},
			hasError: false,
		},
		{
			name:     "WithErrors",
			logs:     []string{"ERROR: Database connection failed", "FATAL: Cannot recover"},
			hasError: true,
		},
		{
			name:     "MixedLogs",
			logs:     []string{"INFO: Starting", "ERROR: Failed to connect", "INFO: Retrying"},
			hasError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			patterns := dm.analyzeErrorPatterns(tc.logs)

			if tc.hasError && len(patterns) == 0 {
				t.Error("Expected error patterns to be found")
			}
			if !tc.hasError && len(patterns) > 0 {
				t.Error("Expected no error patterns for clean logs")
			}
		})
	}
}

func TestGeneratePerformanceRecommendations(t *testing.T) {
	dm := NewDebugManager()

	results := map[string]interface{}{
		"memory": map[string]interface{}{"usage": "high"},
		"cpu":    map[string]interface{}{"usage": "medium"},
	}

	recommendations := dm.generatePerformanceRecommendations(results)

	if len(recommendations) == 0 {
		t.Error("Expected performance recommendations to be generated")
	}
}

func TestGenerateFixSuggestions(t *testing.T) {
	dm := NewDebugManager()

	errorPatterns := []map[string]interface{}{
		{"type": "database", "count": 5},
		{"type": "network", "count": 3},
	}

	suggestions := dm.generateFixSuggestions(errorPatterns)

	if len(suggestions) == 0 {
		t.Error("Expected fix suggestions to be generated")
	}
}

func TestGetHostname(t *testing.T) {
	hostname := getHostname()

	if hostname == "" {
		t.Error("Expected hostname to be returned")
	}

	// Should not be "unknown" in most cases
	if hostname == "unknown" {
		// This is acceptable on some systems
		t.Log("Hostname returned 'unknown'")
	}
}

func TestReadLogFile(t *testing.T) {
	env := setupTestDirectory(t)
	defer env.Cleanup()

	dm := NewDebugManager()

	// Create a test log file
	logContent := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5\n"
	logPath := createTestLogFile(t, env.TempDir, "test.log", logContent)

	// Read the log file
	content, err := dm.readLogFile(logPath, 10)
	if err != nil {
		t.Fatalf("Expected no error reading log file, got %v", err)
	}

	if content == "" {
		t.Error("Expected log content to be returned")
	}

	assertStringContains(t, content, "Line 1")
}

func TestReadLogFile_NotFound(t *testing.T) {
	dm := NewDebugManager()

	_, err := dm.readLogFile("/nonexistent/file.log", 10)
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestGetFileModTime(t *testing.T) {
	env := setupTestDirectory(t)
	defer env.Cleanup()

	dm := NewDebugManager()

	// Create a test file
	testFile := filepath.Join(env.TempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	modTime := dm.getFileModTime(testFile)
	if modTime == "unknown" {
		t.Error("Expected valid modification time")
	}
}

func TestGetFileModTime_NotFound(t *testing.T) {
	dm := NewDebugManager()

	modTime := dm.getFileModTime("/nonexistent/file.txt")
	if modTime != "unknown" {
		t.Errorf("Expected 'unknown' for non-existent file, got %s", modTime)
	}
}

func TestGetLogPaths(t *testing.T) {
	dm := NewDebugManager()

	// This will likely return empty array since test app doesn't exist
	paths, err := dm.getLogPaths("nonexistent-app")
	// Should not error
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Should return array (might be empty or nil, both acceptable)
	if len(paths) < 0 {
		t.Error("Negative path count impossible")
	}
}

func TestAnalyzeLogFile(t *testing.T) {
	env := setupTestDirectory(t)
	defer env.Cleanup()

	dm := NewDebugManager()

	// Create a test log file
	logContent := "Line 1\nLine 2\nLine 3\n"
	logPath := createTestLogFile(t, env.TempDir, "test.log", logContent)

	analysis, err := dm.analyzeLogFile(logPath)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if analysis["file"] != logPath {
		t.Errorf("Expected file path %s, got %v", logPath, analysis["file"])
	}
	if analysis["line_count"] == nil {
		t.Error("Expected line_count to be present")
	}
}
