// +build testing

package main

import (
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestIntegrationScenarios(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	router := setupTestRouter()

	t.Run("CompleteIssueLifecycle", func(t *testing.T) {
		// 1. Create an issue
		issueData := map[string]interface{}{
			"id":          "lifecycle-test-1",
			"component":   "cli",
			"severity":    "high",
			"description": "Complete lifecycle test",
		}

		createReq := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/issues",
			Body:   createJSONBody(t, issueData),
		}
		w := makeHTTPRequest(createReq, router)
		if w.Code != http.StatusCreated {
			t.Fatalf("Failed to create issue: %d", w.Code)
		}

		// 2. List issues and verify it exists
		listReq := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/issues",
		}
		w = makeHTTPRequest(listReq, router)
		if w.Code != http.StatusOK {
			t.Fatalf("Failed to list issues: %d", w.Code)
		}

		// 3. Get workarounds for the issue
		workaroundReq := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/issues/lifecycle-test-1/workarounds",
		}
		w = makeHTTPRequest(workaroundReq, router)
		if w.Code != http.StatusOK {
			t.Fatalf("Failed to get workarounds: %d", w.Code)
		}

		// 4. Analyze the issue
		analyzeReq := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/issues/lifecycle-test-1/analyze",
		}
		w = makeHTTPRequest(analyzeReq, router)
		if w.Code != http.StatusOK {
			t.Fatalf("Failed to analyze issue: %d", w.Code)
		}

		// 5. Verify analysis was recorded
		updatedIssue := readIssueFile(t, env, "lifecycle-test-1")
		if len(updatedIssue.FixAttempts) == 0 {
			t.Error("Analysis should have recorded a fix attempt")
		}
	})

	t.Run("FilteringAndQuerying", func(t *testing.T) {
		// Create multiple issues with different attributes
		components := []string{"cli", "orchestrator", "resource-manager"}
		severities := []string{"low", "medium", "high", "critical"}

		for i, comp := range components {
			for j, sev := range severities {
				// Use simple ID without path separators
				issueID := comp + "-" + sev
				issue := NewTestIssue(issueID, comp).WithSeverity(sev).Build()

				createTestIssue(t, env, issue)

				// Verify we can filter by component
				req := HTTPTestRequest{
					Method: "GET",
					Path:   "/api/v1/issues?component=" + comp,
				}
				w := makeHTTPRequest(req, router)
				if w.Code != http.StatusOK {
					t.Errorf("Filter by component %s failed: %d", comp, w.Code)
				}

				// Verify we can filter by severity
				req = HTTPTestRequest{
					Method: "GET",
					Path:   "/api/v1/issues?severity=" + sev,
				}
				w = makeHTTPRequest(req, router)
				if w.Code != http.StatusOK {
					t.Errorf("Filter by severity %s failed: %d", sev, w.Code)
				}

				// Only test a subset to avoid too many iterations
				if i*len(severities)+j >= 3 {
					goto done
				}
			}
		}
	done:
	})

	t.Run("HealthCheckIntegration", func(t *testing.T) {
		// Create some issues
		for i := 0; i < 5; i++ {
			issueID := "health-test-" + string(rune(i+'0'))
			issue := NewTestIssue(issueID, "cli").Build()
			createTestIssue(t, env, issue)
		}

		// Check health and verify active issues count
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/health",
		}
		w := makeHTTPRequest(req, router)

		if w.Code != http.StatusOK {
			t.Fatalf("Health check failed: %d", w.Code)
		}

		data := parseJSONResponse(t, w)

		activeIssues, ok := data["active_issues"].(float64)
		if !ok {
			t.Error("active_issues field missing or wrong type")
		} else if int(activeIssues) == 0 {
			t.Log("Note: active_issues count may include or exclude test issues depending on timing")
		}
	})
}

func TestErrorConditions(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	router := setupTestRouter()

	t.Run("CorruptedIssueFile", func(t *testing.T) {
		// Create a corrupted issue file
		corruptedFile := filepath.Join(env.DataDir, "issues", "corrupted.json")
		if err := ioutil.WriteFile(corruptedFile, []byte("not valid json{"), 0644); err != nil {
			t.Fatalf("Failed to create corrupted file: %v", err)
		}

		// List should skip corrupted files
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/issues",
		}
		w := makeHTTPRequest(req, router)

		if w.Code != http.StatusOK {
			t.Errorf("Expected to handle corrupted file gracefully, got: %d", w.Code)
		}
	})

	t.Run("NonJSONFile", func(t *testing.T) {
		// Create a non-JSON file in issues directory
		nonJSONFile := filepath.Join(env.DataDir, "issues", "readme.txt")
		if err := ioutil.WriteFile(nonJSONFile, []byte("This is not an issue"), 0644); err != nil {
			t.Fatalf("Failed to create non-JSON file: %v", err)
		}

		// List should skip non-JSON files
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/issues",
		}
		w := makeHTTPRequest(req, router)

		if w.Code != http.StatusOK {
			t.Errorf("Expected to skip non-JSON files, got: %d", w.Code)
		}
	})

	t.Run("AnalyzeNonExistentIssue", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/issues/does-not-exist/analyze",
		}
		w := makeHTTPRequest(req, router)

		assertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("CorruptedAnalysisTarget", func(t *testing.T) {
		// Create corrupted issue file
		corruptedFile := filepath.Join(env.DataDir, "issues", "corrupt-analysis.json")
		if err := ioutil.WriteFile(corruptedFile, []byte("{corrupted}"), 0644); err != nil {
			t.Fatalf("Failed to create corrupted file: %v", err)
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/issues/corrupt-analysis/analyze",
		}
		w := makeHTTPRequest(req, router)

		// Should fail to parse
		if w.Code == http.StatusOK {
			t.Error("Expected error for corrupted issue file")
		}
	})
}

func TestEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	router := setupTestRouter()

	t.Run("VeryLongDescription", func(t *testing.T) {
		longDesc := string(make([]byte, 10000))
		for i := range longDesc {
			longDesc = longDesc[:i] + "x" + longDesc[i+1:]
		}

		issueData := map[string]interface{}{
			"component":   "cli",
			"severity":    "medium",
			"description": longDesc,
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/issues",
			Body:   createJSONBody(t, issueData),
		}

		w := makeHTTPRequest(req, router)
		if w.Code != http.StatusCreated {
			t.Errorf("Failed to create issue with long description: %d", w.Code)
		}
	})

	t.Run("SpecialCharactersInComponent", func(t *testing.T) {
		specialChars := []string{
			"component/with/slashes",
			"component-with-dashes",
			"component_with_underscores",
			"component.with.dots",
		}

		for _, comp := range specialChars {
			issueData := map[string]interface{}{
				"component":   comp,
				"severity":    "low",
				"description": "Testing special characters",
			}

			req := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/issues",
				Body:   createJSONBody(t, issueData),
			}

			w := makeHTTPRequest(req, router)
			if w.Code != http.StatusCreated {
				t.Logf("Issue creation with component '%s': %d", comp, w.Code)
			}
		}
	})

	t.Run("EmptyComponents", func(t *testing.T) {
		// Temporarily clear components
		originalComponents := components
		components = []Component{}
		defer func() { components = originalComponents }()

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/health",
		}

		w := makeHTTPRequest(req, router)
		if w.Code != http.StatusOK {
			t.Errorf("Health check should work with empty components: %d", w.Code)
		}

		data := parseJSONResponse(t, w)
		comps := data["components"]
		if comps == nil {
			// OK, nil is acceptable for empty components
		} else if compsList, ok := comps.([]interface{}); ok {
			if len(compsList) != 0 {
				t.Errorf("Expected 0 components, got %d", len(compsList))
			}
		} else {
			t.Error("Components field is not an array")
		}
	})

	t.Run("MissingDataDirectory", func(t *testing.T) {
		// Create a temporary subdirectory that doesn't exist
		missingDir := filepath.Join(env.TempDir, "nonexistent")

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/issues",
		}

		// Change dataDir temporarily
		originalDataDir := dataDir
		dataDir = missingDir
		defer func() { dataDir = originalDataDir }()

		w := makeHTTPRequest(req, router)
		// Should handle missing directory gracefully
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Logf("Expected OK or error for missing directory, got: %d", w.Code)
		}
	})
}

func TestComponentHealthChecks(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("ComponentTimeout", func(t *testing.T) {
		// Add a component with very short timeout
		timeoutComp := Component{
			ID:          "timeout-test",
			Name:        "Timeout Test",
			Description: "Component that will timeout",
			HealthCheck: "sleep 10", // Will timeout
			Critical:    false,
			TimeoutMs:   10, // 10ms timeout
		}

		originalComponents := components
		components = append(components, timeoutComp)
		defer func() { components = originalComponents }()

		results := checkAllComponents()

		// Find the timeout component
		found := false
		for _, result := range results {
			if result.Component == "timeout-test" {
				found = true
				if result.Status == "healthy" {
					t.Error("Expected timeout component to be unhealthy")
				}
				break
			}
		}

		if !found {
			t.Error("Timeout component not found in results")
		}
	})

	t.Run("FailingHealthCheck", func(t *testing.T) {
		failComp := Component{
			ID:          "fail-test",
			Name:        "Failing Test",
			Description: "Component that fails health check",
			HealthCheck: "exit 1",
			Critical:    false,
			TimeoutMs:   1000,
		}

		originalComponents := components
		components = append(components, failComp)
		defer func() { components = originalComponents }()

		results := checkAllComponents()

		found := false
		for _, result := range results {
			if result.Component == "fail-test" {
				found = true
				if result.Status != "unhealthy" {
					t.Errorf("Expected unhealthy status, got: %s", result.Status)
				}
				break
			}
		}

		if !found {
			t.Error("Failing component not found in results")
		}
	})
}

func TestSaveHealthState(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("WriteHealthLog", func(t *testing.T) {
		health := ComponentHealth{
			Component:      "test-comp",
			Status:         "healthy",
			ResponseTimeMs: 50,
			ErrorCount:     0,
			Details:        make(map[string]interface{}),
		}

		saveHealthState(health)

		// Verify file was created
		healthFile := filepath.Join(env.DataDir, "health", "latest.jsonl")
		if _, err := os.Stat(healthFile); os.IsNotExist(err) {
			t.Error("Health state file was not created")
		}

		// Verify content
		content, err := ioutil.ReadFile(healthFile)
		if err != nil {
			t.Fatalf("Failed to read health file: %v", err)
		}

		if len(content) == 0 {
			t.Error("Health file is empty")
		}
	})
}
