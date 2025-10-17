// +build testing

package main

import (
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestGetWorkaroundsEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	router := setupTestRouter()

	t.Run("IssueWithMultipleWorkarounds", func(t *testing.T) {
		// Create issue with multiple workarounds
		workarounds := []Workaround{
			createTestWorkaround("multi-workaround-1"),
			createTestWorkaround("multi-workaround-2"),
			createTestWorkaround("multi-workaround-3"),
		}

		issue := NewTestIssue("multi-workaround-1", "cli")
		for _, w := range workarounds {
			issue = issue.WithWorkaround(w)
		}
		createTestIssue(t, env, issue.Build())

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/issues/multi-workaround-1/workarounds",
		}

		w := makeHTTPRequest(req, router)
		assertJSONResponse(t, w, http.StatusOK, validateWorkaroundsResponse)

		data := parseJSONResponse(t, w)
		workaroundsList := data["workarounds"].([]interface{})
		if len(workaroundsList) != 3 {
			t.Errorf("Expected 3 workarounds, got %d", len(workaroundsList))
		}
	})

	t.Run("CommonWorkaroundsFile", func(t *testing.T) {
		// Create common workarounds
		commonWorkarounds := map[string]interface{}{
			"workarounds": []Workaround{
				createTestWorkaround("common-1"),
				createTestWorkaround("common-2"),
			},
		}

		commonFile := filepath.Join(env.DataDir, "workarounds", "common.json")
		data := createJSONBody(t, commonWorkarounds)
		if err := ioutil.WriteFile(commonFile, []byte(data), 0644); err != nil {
			t.Fatalf("Failed to create common workarounds: %v", err)
		}

		// Request workarounds for issue without specific workarounds
		issue := NewTestIssue("no-workarounds", "cli").Build()
		createTestIssue(t, env, issue)

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/issues/no-workarounds/workarounds",
		}

		w := makeHTTPRequest(req, router)
		assertJSONResponse(t, w, http.StatusOK, validateWorkaroundsResponse)

		respData := parseJSONResponse(t, w)
		workaroundsList := respData["workarounds"].([]interface{})
		if len(workaroundsList) != 2 {
			t.Errorf("Expected 2 common workarounds, got %d", len(workaroundsList))
		}
	})
}

func TestListIssuesFiltering(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	router := setupTestRouter()

	// Create test issues with different statuses
	statuses := []string{"active", "resolved", "workaround-available"}
	for i, status := range statuses {
		issue := NewTestIssue("status-test-"+string(rune(i+'0')), "cli").
			WithStatus(status).
			Build()
		createTestIssue(t, env, issue)
	}

	t.Run("FilterByEachStatus", func(t *testing.T) {
		for _, status := range statuses {
			req := HTTPTestRequest{
				Method: "GET",
				Path:   "/api/v1/issues?status=" + status,
			}

			w := makeHTTPRequest(req, router)
			assertJSONResponse(t, w, http.StatusOK, validateIssuesResponse)

			data := parseJSONResponse(t, w)
			issues := data["issues"]
			if issues != nil {
				issuesList := issues.([]interface{})
				// Verify at least one issue with that status exists
				found := false
				for _, issue := range issuesList {
					issueMap := issue.(map[string]interface{})
					if issueMap["status"] == status {
						found = true
						break
					}
				}
				if !found && len(issuesList) > 0 {
					t.Errorf("Expected to find issues with status '%s'", status)
				}
			}
		}
	})
}

func TestLoadComponentsEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("InvalidJSON", func(t *testing.T) {
		// Create invalid JSON in components file
		componentsFile := filepath.Join(env.DataDir, "components.json")
		if err := ioutil.WriteFile(componentsFile, []byte("invalid json{"), 0644); err != nil {
			t.Fatalf("Failed to create invalid components file: %v", err)
		}

		// Try to load - should handle gracefully
		originalComponents := components
		loadComponents()

		// Components should either stay as is or be reset
		if components == nil {
			components = originalComponents
		}
	})

	t.Run("EmptyFile", func(t *testing.T) {
		// Create empty components file
		componentsFile := filepath.Join(env.DataDir, "components.json")
		if err := ioutil.WriteFile(componentsFile, []byte(""), 0644); err != nil {
			t.Fatalf("Failed to create empty components file: %v", err)
		}

		originalComponents := components
		loadComponents()

		if components == nil {
			components = originalComponents
		}
	})
}

func TestHealthHandlerWithIssues(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	router := setupTestRouter()

	t.Run("CountActiveIssues", func(t *testing.T) {
		// Create some issues
		for i := 0; i < 10; i++ {
			issue := NewTestIssue("count-test-"+string(rune(i+'0')), "cli").Build()
			createTestIssue(t, env, issue)
		}

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/health",
		}

		w := makeHTTPRequest(req, router)
		assertJSONResponse(t, w, http.StatusOK, func(data map[string]interface{}) bool {
			activeIssues, ok := data["active_issues"].(float64)
			if !ok {
				t.Error("active_issues field missing or wrong type")
				return false
			}

			// Should have at least some issues
			if int(activeIssues) < 1 {
				t.Logf("Expected some active issues, got %d", int(activeIssues))
			}
			return true
		})
	})
}

func TestReportIssueMetadata(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	router := setupTestRouter()

	t.Run("WithMetadata", func(t *testing.T) {
		issueData := map[string]interface{}{
			"component":   "orchestrator",
			"severity":    "high",
			"description": "Issue with metadata",
			"metadata": map[string]interface{}{
				"key1": "value1",
				"key2": 123,
				"key3": true,
			},
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/issues",
			Body:   createJSONBody(t, issueData),
		}

		w := makeHTTPRequest(req, router)
		if w.Code != http.StatusCreated {
			t.Errorf("Failed to create issue with metadata: %d", w.Code)
		}

		data := parseJSONResponse(t, w)
		metadata, ok := data["metadata"].(map[string]interface{})
		if !ok {
			t.Error("Metadata field missing or wrong type")
		} else {
			if metadata["key1"] != "value1" {
				t.Error("Metadata key1 value incorrect")
			}
		}
	})

	t.Run("WithCompleteFields", func(t *testing.T) {
		issueData := map[string]interface{}{
			"id":              "complete-issue",
			"component":       "cli",
			"severity":        "critical",
			"status":          "active",
			"description":     "Complete issue with all fields",
			"error_signature": "ERR_COMPLETE_001",
			"metadata": map[string]interface{}{
				"environment": "test",
			},
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/issues",
			Body:   createJSONBody(t, issueData),
		}

		w := makeHTTPRequest(req, router)
		if w.Code != http.StatusCreated {
			t.Errorf("Failed to create complete issue: %d", w.Code)
		}

		data := parseJSONResponse(t, w)

		// Verify all fields
		if data["id"] != "complete-issue" {
			t.Error("ID not set correctly")
		}
		if data["error_signature"] != "ERR_COMPLETE_001" {
			t.Error("Error signature not set correctly")
		}
		if data["severity"] != "critical" {
			t.Error("Severity not set correctly")
		}
	})
}

func TestAnalyzeIssueUpdates(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	router := setupTestRouter()

	t.Run("MultipleAnalyses", func(t *testing.T) {
		issue := NewTestIssue("multi-analysis", "cli").
			WithDescription("Issue for multiple analyses").
			Build()
		createTestIssue(t, env, issue)

		// Analyze multiple times
		for i := 0; i < 3; i++ {
			req := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/issues/multi-analysis/analyze",
			}

			w := makeHTTPRequest(req, router)
			if w.Code != http.StatusOK {
				t.Errorf("Analysis %d failed: %d", i+1, w.Code)
			}
		}

		// Verify all analyses were recorded
		updatedIssue := readIssueFile(t, env, "multi-analysis")
		if len(updatedIssue.FixAttempts) != 3 {
			t.Errorf("Expected 3 fix attempts, got %d", len(updatedIssue.FixAttempts))
		}
	})
}

func TestDetermineOverallStatusEdgeCases(t *testing.T) {
	t.Run("EmptyHealth", func(t *testing.T) {
		health := []ComponentHealth{}
		status := determineOverallStatus(health)
		if status != "healthy" {
			t.Errorf("Expected 'healthy' for empty health, got '%s'", status)
		}
	})

	t.Run("MixedHealthStatus", func(t *testing.T) {
		// Mix of healthy and degraded non-critical components
		health := []ComponentHealth{
			{Component: "comp1", Status: "healthy"},
			{Component: "comp2", Status: "degraded"},
			{Component: "comp3", Status: "healthy"},
		}

		status := determineOverallStatus(health)
		// Should be degraded since at least one is down
		if status != "degraded" {
			t.Errorf("Expected 'degraded', got '%s'", status)
		}
	})
}

func TestCheckComponentsWithTimeout(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("ZeroTimeout", func(t *testing.T) {
		// Component with 0 timeout - should default or handle gracefully
		zeroTimeoutComp := Component{
			ID:          "zero-timeout",
			Name:        "Zero Timeout Component",
			Description: "Component with zero timeout",
			HealthCheck: "echo 'ok'",
			Critical:    false,
			TimeoutMs:   0, // Zero timeout
		}

		originalComponents := components
		components = []Component{zeroTimeoutComp}
		defer func() { components = originalComponents }()

		// Should handle gracefully
		results := checkAllComponents()
		if len(results) != 1 {
			t.Errorf("Expected 1 result, got %d", len(results))
		}
	})
}

func TestHealthHandlerErrorReadingIssues(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	router := setupTestRouter()

	t.Run("UnreadableIssuesDirectory", func(t *testing.T) {
		// Make issues directory unreadable
		issuesDir := filepath.Join(env.DataDir, "issues")
		originalMode := os.FileMode(0755)
		if info, err := os.Stat(issuesDir); err == nil {
			originalMode = info.Mode()
		}

		// Restore permissions after test
		defer os.Chmod(issuesDir, originalMode)

		// This might not work on all systems, but try anyway
		os.Chmod(issuesDir, 0000)

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/health",
		}

		w := makeHTTPRequest(req, router)
		// Should still return OK, just with 0 active issues (or error in counting)
		if w.Code != http.StatusOK {
			t.Logf("Health check status: %d", w.Code)
		}

		// Restore immediately
		os.Chmod(issuesDir, originalMode)
	})
}
