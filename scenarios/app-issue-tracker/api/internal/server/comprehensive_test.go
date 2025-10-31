package server

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

// TestGetIssueHandler_Comprehensive tests all scenarios for getIssueHandler
func TestGetIssueHandler_Comprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	suite := NewHandlerTestSuite(env.Server, env.Server.getIssueHandler, "getIssueHandler")

	t.Run("Success", func(t *testing.T) {
		issue := createTestIssue("issue-get-test", "Test Issue", "bug", "high", "test-app")
		if _, err := env.Server.saveIssue(issue, "open"); err != nil {
			t.Fatalf("Failed to create test issue: %v", err)
		}

		req := HTTPTestRequest{
			Method:  http.MethodGet,
			Path:    "/api/v1/issues/issue-get-test",
			URLVars: map[string]string{"id": "issue-get-test"},
		}

		w := suite.TestSuccess(t, req, http.StatusOK)
		resp := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})

		if resp == nil {
			return
		}

		data, ok := resp["data"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected data field in response")
		}

		issueData, ok := data["issue"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected issue field in data")
		}

		if issueData["id"] != "issue-get-test" {
			t.Errorf("Expected issue ID 'issue-get-test', got %v", issueData["id"])
		}
	})

	t.Run("ErrorCases", func(t *testing.T) {
		scenarios := GetHandlerErrorScenarios("/api/v1/issues/{id}")
		suite.TestError(t, scenarios)
	})
}

// TestCreateIssueHandler_Comprehensive tests all scenarios for createIssueHandler
func TestCreateIssueHandler_Comprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	suite := NewHandlerTestSuite(env.Server, env.Server.createIssueHandler, "createIssueHandler")

	t.Run("Success_MinimalFields", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: http.MethodPost,
			Path:   "/api/v1/issues",
			Body: map[string]interface{}{
				"title": "New Issue",
				"targets": []map[string]interface{}{
					{"type": "scenario", "id": "test-app"},
				},
			},
		}

		w := suite.TestSuccess(t, req, http.StatusOK)
		resp := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})

		if resp == nil {
			return
		}

		data, ok := resp["data"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected data field in response")
		}

		issueData, ok := data["issue"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected issue field in data")
		}

		if issueData["title"] != "New Issue" {
			t.Errorf("Expected title 'New Issue', got %v", issueData["title"])
		}

		// Verify issue was saved
		issueID, ok := issueData["id"].(string)
		if !ok || issueID == "" {
			t.Fatal("Expected issue ID in response")
		}

		assertIssueExists(t, env.Server, issueID, "open")
	})

	t.Run("Success_AllFields", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: http.MethodPost,
			Path:   "/api/v1/issues",
			Body: map[string]interface{}{
				"title":       "Comprehensive Issue",
				"description": "Full description",
				"targets": []map[string]interface{}{
					{"type": "scenario", "id": "test-app"},
				},
				"type":     "feature",
				"priority": "medium",
				"tags":     []string{"ui", "frontend"},
				"metadata_extra": map[string]string{
					"source": "test-suite",
				},
			},
		}

		w := suite.TestSuccess(t, req, http.StatusOK)
		resp := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})

		if resp == nil {
			return
		}

		data, ok := resp["data"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected data field in response")
		}

		issueData, ok := data["issue"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected issue field in data")
		}

		if issueData["type"] != "feature" {
			t.Errorf("Expected type 'feature', got %v", issueData["type"])
		}
		if issueData["priority"] != "medium" {
			t.Errorf("Expected priority 'medium', got %v", issueData["priority"])
		}
	})

	t.Run("ErrorCases", func(t *testing.T) {
		scenarios := append(
			CreateHandlerErrorScenarios("/api/v1/issues"),
			TestScenario{
				Name:   "Missing Title",
				Method: http.MethodPost,
				Path:   "/api/v1/issues",
				Body: map[string]interface{}{
					"targets": []map[string]interface{}{
					{"type": "scenario", "id": "test-app"},
				},
				},
				ExpectedStatus: http.StatusBadRequest,
			},
		)
		suite.TestError(t, scenarios)
	})
}

// TestUpdateIssueHandler_Comprehensive tests all scenarios for updateIssueHandler
func TestUpdateIssueHandler_Comprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	suite := NewHandlerTestSuite(env.Server, env.Server.updateIssueHandler, "updateIssueHandler")

	t.Run("Success_UpdateTitle", func(t *testing.T) {
		issue := createTestIssue("issue-update-1", "Original Title", "bug", "high", "test-app")
		if _, err := env.Server.saveIssue(issue, "open"); err != nil {
			t.Fatalf("Failed to create test issue: %v", err)
		}

		req := HTTPTestRequest{
			Method:  http.MethodPut,
			Path:    "/api/v1/issues/issue-update-1",
			URLVars: map[string]string{"id": "issue-update-1"},
			Body: map[string]interface{}{
				"title": "Updated Title",
			},
		}

		suite.TestSuccess(t, req, http.StatusOK)

		updated := assertIssueExists(t, env.Server, "issue-update-1", "open")
		if updated.Title != "Updated Title" {
			t.Errorf("Expected title 'Updated Title', got %s", updated.Title)
		}
	})

	t.Run("Success_UpdateStatus", func(t *testing.T) {
		issue := createTestIssue("issue-update-2", "Status Test", "bug", "high", "test-app")
		if _, err := env.Server.saveIssue(issue, "open"); err != nil {
			t.Fatalf("Failed to create test issue: %v", err)
		}

		req := HTTPTestRequest{
			Method:  http.MethodPut,
			Path:    "/api/v1/issues/issue-update-2",
			URLVars: map[string]string{"id": "issue-update-2"},
			Body: map[string]interface{}{
				"status": "active",
			},
		}

		suite.TestSuccess(t, req, http.StatusOK)

		// Verify issue moved to active folder
		assertIssueExists(t, env.Server, "issue-update-2", "active")
	})

	t.Run("Success_UpdateMultipleFields", func(t *testing.T) {
		issue := createTestIssue("issue-update-3", "Multi Test", "bug", "low", "test-app")
		if _, err := env.Server.saveIssue(issue, "open"); err != nil {
			t.Fatalf("Failed to create test issue: %v", err)
		}

		req := HTTPTestRequest{
			Method:  http.MethodPut,
			Path:    "/api/v1/issues/issue-update-3",
			URLVars: map[string]string{"id": "issue-update-3"},
			Body: map[string]interface{}{
				"title":    "Updated Multi",
				"priority": "high",
				"tags":     []string{"urgent", "critical"},
			},
		}

		suite.TestSuccess(t, req, http.StatusOK)

		updated := assertIssueExists(t, env.Server, "issue-update-3", "open")
		if updated.Title != "Updated Multi" {
			t.Errorf("Expected title 'Updated Multi', got %s", updated.Title)
		}
		if updated.Priority != "high" {
			t.Errorf("Expected priority 'high', got %s", updated.Priority)
		}
	})

	t.Run("ErrorCases", func(t *testing.T) {
		scenarios := UpdateHandlerErrorScenarios("/api/v1/issues/{id}")
		suite.TestError(t, scenarios)
	})
}

// TestDeleteIssueHandler_Comprehensive tests all scenarios for deleteIssueHandler
func TestDeleteIssueHandler_Comprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	suite := NewHandlerTestSuite(env.Server, env.Server.deleteIssueHandler, "deleteIssueHandler")

	t.Run("Success", func(t *testing.T) {
		issue := createTestIssue("issue-delete-1", "To Delete", "bug", "low", "test-app")
		if _, err := env.Server.saveIssue(issue, "open"); err != nil {
			t.Fatalf("Failed to create test issue: %v", err)
		}

		req := HTTPTestRequest{
			Method:  http.MethodDelete,
			Path:    "/api/v1/issues/issue-delete-1",
			URLVars: map[string]string{"id": "issue-delete-1"},
		}

		suite.TestSuccess(t, req, http.StatusOK)

		// Verify issue no longer exists
		assertIssueNotExists(t, env.Server, "issue-delete-1")
	})

	t.Run("Conflict_WhenAgentRunning", func(t *testing.T) {
		issue := createTestIssue("issue-delete-running", "Running", "bug", "medium", "test-app")
		if _, err := env.Server.saveIssue(issue, "open"); err != nil {
			t.Fatalf("Failed to create test issue: %v", err)
		}

		startTime := time.Now().UTC().Format(time.RFC3339)
		env.Server.processor.RegisterRunningProcess(issue.ID, "agent-test", startTime, nil)
		t.Cleanup(func() {
			env.Server.processor.UnregisterRunningProcess(issue.ID)
		})

		req := HTTPTestRequest{
			Method:  http.MethodDelete,
			Path:    "/api/v1/issues/issue-delete-running",
			URLVars: map[string]string{"id": "issue-delete-running"},
		}

		w := makeHTTPRequest(env.Server.deleteIssueHandler, req)
		assertErrorResponse(t, w, http.StatusConflict, "Cannot delete issue while an agent is running")

		assertIssueExists(t, env.Server, "issue-delete-running", "open")
	})

	t.Run("ErrorCases", func(t *testing.T) {
		scenarios := DeleteHandlerErrorScenarios("/api/v1/issues/{id}")
		suite.TestError(t, scenarios)
	})
}

// TestGetIssuesHandler_Comprehensive tests list endpoint with filtering
func TestGetIssuesHandler_Comprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	// Create test issues in different folders
	for i := 1; i <= 3; i++ {
		issue := createTestIssue(fmt.Sprintf("issue-list-%d", i), fmt.Sprintf("Test %d", i), "bug", "medium", "test-app")
		folder := "open"
		if i == 2 {
			folder = "active"
		} else if i == 3 {
			folder = "completed"
		}
		if _, err := env.Server.saveIssue(issue, folder); err != nil {
			t.Fatalf("Failed to create test issue: %v", err)
		}
	}

	t.Run("Success_All", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: http.MethodGet,
			Path:   "/api/v1/issues",
		}

		w := makeHTTPRequest(env.Server.getIssuesHandler, req)
		resp := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})

		if resp == nil {
			return
		}

		data, ok := resp["data"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected data field in response")
		}

		issues, ok := data["issues"].([]interface{})
		if !ok {
			t.Fatal("Expected issues array in data")
		}

		if len(issues) < 3 {
			t.Errorf("Expected at least 3 issues, got %d", len(issues))
		}
	})

	t.Run("Success_FilterByStatus", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:      http.MethodGet,
			Path:        "/api/v1/issues",
			QueryParams: map[string]string{"status": "active"},
		}

		w := makeHTTPRequest(env.Server.getIssuesHandler, req)
		resp := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})

		if resp == nil {
			return
		}

		data, ok := resp["data"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected data field in response")
		}

		issues, ok := data["issues"].([]interface{})
		if !ok {
			t.Fatal("Expected issues array in data")
		}

		// Verify all returned issues have active status
		for _, issueInterface := range issues {
			issue, ok := issueInterface.(map[string]interface{})
			if !ok {
				continue
			}
			if issue["status"] != "active" {
				t.Errorf("Expected all issues to have status 'active', got %v", issue["status"])
			}
		}
	})

	t.Run("Error_InvalidStatusFilter", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:      http.MethodGet,
			Path:        "/api/v1/issues",
			QueryParams: map[string]string{"status": "not-a-status"},
		}

		w := makeHTTPRequest(env.Server.getIssuesHandler, req)
		assertErrorResponse(t, w, http.StatusBadRequest, "Invalid status filter")
	})

	t.Run("Success_FilterByAppID", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:      http.MethodGet,
			Path:        "/api/v1/issues",
			QueryParams: map[string]string{"target_id": "test-app"},
		}

		w := makeHTTPRequest(env.Server.getIssuesHandler, req)
		assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})
	})
}

// TestHealthHandler tests the health check endpoint
func TestHealthHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	req := HTTPTestRequest{
		Method: http.MethodGet,
		Path:   "/health",
	}

	w := makeHTTPRequest(env.Server.healthHandler, req)
	resp := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
		"success": true,
	})

	if resp == nil {
		return
	}

	if _, ok := resp["data"]; !ok {
		t.Error("Expected data field in health response")
	}
}

// TestGetStatsHandler tests statistics endpoint
func TestGetStatsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	// Create issues in different statuses
	statuses := []string{"open", "active", "completed"}
	for i, status := range statuses {
		issue := createTestIssue(fmt.Sprintf("issue-stats-%d", i), "Stats Test", "bug", "medium", "test-app")
		if _, err := env.Server.saveIssue(issue, status); err != nil {
			t.Fatalf("Failed to create test issue: %v", err)
		}
	}

	req := HTTPTestRequest{
		Method: http.MethodGet,
		Path:   "/api/v1/stats",
	}

	w := makeHTTPRequest(env.Server.getStatsHandler, req)
	resp := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
		"success": true,
	})

	if resp == nil {
		return
	}

	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected data field in response")
	}

	stats, ok := data["stats"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected stats field in data")
	}

	if _, ok := stats["total_issues"]; !ok {
		t.Error("Expected total_issues in stats")
	}

	if _, ok := stats["open_issues"]; !ok {
		t.Error("Expected open_issues in stats")
	}
}

// TestGetProcessorHandler tests processor state retrieval
func TestGetProcessorHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	req := HTTPTestRequest{
		Method: http.MethodGet,
		Path:   "/api/v1/automation/processor",
	}

	w := makeHTTPRequest(env.Server.getProcessorHandler, req)
	resp := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
		"success": true,
	})

	if resp == nil {
		return
	}

	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected data field in response")
	}

	processor, ok := data["processor"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected processor field in data")
	}

	if _, ok := processor["active"]; !ok {
		t.Error("Expected active field in processor")
	}
	if _, ok := processor["concurrent_slots"]; !ok {
		t.Error("Expected concurrent_slots field in processor")
	}
}

// TestUpdateProcessorHandler tests processor configuration updates
func TestUpdateProcessorHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("Success_ActivateProcessor", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: http.MethodPatch,
			Path:   "/api/v1/automation/processor",
			Body: map[string]interface{}{
				"active": true,
			},
		}

		w := makeHTTPRequest(env.Server.updateProcessorHandler, req)
		assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})

		// Verify state was updated
		state := env.Server.currentProcessorState()
		if !state.Active {
			t.Error("Expected processor to be active")
		}
	})

	t.Run("Success_UpdateSlots", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: http.MethodPatch,
			Path:   "/api/v1/automation/processor",
			Body: map[string]interface{}{
				"concurrent_slots": 5,
			},
		}

		w := makeHTTPRequest(env.Server.updateProcessorHandler, req)
		assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})

		state := env.Server.currentProcessorState()
		if state.ConcurrentSlots != 5 {
			t.Errorf("Expected 5 concurrent slots, got %d", state.ConcurrentSlots)
		}
	})

	t.Run("ErrorCases", func(t *testing.T) {
		scenarios := []TestScenario{
			{
				Name:           "Invalid JSON",
				Method:         http.MethodPatch,
				Path:           "/api/v1/automation/processor",
				Body:           "{invalid",
				ExpectedStatus: http.StatusBadRequest,
			},
		}

		pattern := NewErrorTestPattern(env.Server.updateProcessorHandler, env.Server)
		pattern.RunScenarios(t, scenarios)
	})
}
