//go:build testing
// +build testing

package server

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

// TestSearchIssuesHandler tests the search functionality
func TestSearchIssuesHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	// Create test issues
	for i := 1; i <= 5; i++ {
		var title, description string
		if i <= 2 {
			title = "Database Connection Error"
			description = "Cannot connect to database"
		} else {
			title = "UI Button Not Working"
			description = "Click event not firing"
		}

		issue := createTestIssue(
			fmt.Sprintf("issue-search-%d", i),
			title,
			"bug",
			"medium",
			"search-test",
		)
		issue.Description = description
		if _, err := env.Server.saveIssue(issue, "open"); err != nil {
			t.Fatalf("Failed to create test issue: %v", err)
		}
	}

	t.Run("SearchByTitle", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:      http.MethodGet,
			Path:        "/api/v1/issues/search",
			QueryParams: map[string]string{"q": "Database"},
		}

		w := makeHTTPRequest(env.Server.searchIssuesHandler, req)
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("SearchNoQuery", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: http.MethodGet,
			Path:   "/api/v1/issues/search",
		}

		w := makeHTTPRequest(env.Server.searchIssuesHandler, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for missing query, got %d", w.Code)
		}
	})
}

// TestExportIssuesHandler tests export functionality
func TestExportIssuesHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	// Create test issues
	for i := 1; i <= 3; i++ {
		issue := createTestIssue(
			fmt.Sprintf("issue-export-%d", i),
			fmt.Sprintf("Export Test %d", i),
			"bug",
			"medium",
			"export-test",
		)
		if _, err := env.Server.saveIssue(issue, "open"); err != nil {
			t.Fatalf("Failed to create test issue: %v", err)
		}
	}

	t.Run("ExportJSON", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:      http.MethodGet,
			Path:        "/api/v1/export",
			QueryParams: map[string]string{"format": "json"},
		}

		w := makeHTTPRequest(env.Server.exportIssuesHandler, req)
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		contentType := w.Header().Get("Content-Type")
		if !strings.Contains(contentType, "application/json") {
			t.Errorf("Expected JSON content type, got %s", contentType)
		}
	})

	t.Run("ExportCSV", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:      http.MethodGet,
			Path:        "/api/v1/export",
			QueryParams: map[string]string{"format": "csv"},
		}

		w := makeHTTPRequest(env.Server.exportIssuesHandler, req)
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		contentType := w.Header().Get("Content-Type")
		if !strings.Contains(contentType, "text/csv") {
			t.Errorf("Expected CSV content type, got %s", contentType)
		}
	})

	t.Run("ExportMarkdown", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:      http.MethodGet,
			Path:        "/api/v1/export",
			QueryParams: map[string]string{"format": "markdown"},
		}

		w := makeHTTPRequest(env.Server.exportIssuesHandler, req)
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		contentType := w.Header().Get("Content-Type")
		if !strings.Contains(contentType, "text/markdown") {
			t.Errorf("Expected Markdown content type, got %s", contentType)
		}
	})
}

// TestPreviewInvestigationPromptHandler tests prompt preview functionality
func TestPreviewInvestigationPromptHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	// Create test issue
	issue := createTestIssueWithError(
		"issue-prompt-preview",
		"Preview Test",
		"Test error message",
		"stack trace here",
		[]string{"api/main.go"},
	)
	if _, err := env.Server.saveIssue(issue, "open"); err != nil {
		t.Fatalf("Failed to create test issue: %v", err)
	}

	t.Run("PreviewWithIssueID", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: http.MethodPost,
			Path:   "/api/v1/investigate/preview",
			Body: map[string]interface{}{
				"issue_id": "issue-prompt-preview",
				"agent_id": "unified-resolver",
			},
		}

		w := makeHTTPRequest(env.Server.previewInvestigationPromptHandler, req)
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("PreviewMissingIssueID", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: http.MethodPost,
			Path:   "/api/v1/investigate/preview",
			Body:   map[string]interface{}{},
		}

		w := makeHTTPRequest(env.Server.previewInvestigationPromptHandler, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}

func TestTriggerInvestigationHandler_RespectsConcurrentSlots(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	issue := createTestIssue(
		"issue-concurrency-1",
		"Concurrent Slot Test",
		"bug",
		"high",
		"concurrency-suite",
	)
	if _, err := env.Server.saveIssue(tissue, "open"); err != nil {
		t.Fatalf("Failed to create test issue: %v", err)
	}

	slots := 1
	env.Server.processor.UpdateState(nil, &slots, nil, nil, nil)

	env.Server.processor.RegisterRunningProcess(
		"issue-concurrency-running",
		"agent-existing",
		time.Now().UTC().Format(time.RFC3339),
		nil,
	)

	req := HTTPTestRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/investigate",
		Body: map[string]interface{}{
			"issue_id": tissue.ID,
		},
	}

	w := makeHTTPRequest(env.Server.triggerInvestigationHandler, req)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("Expected status %d when concurrent slots exhausted, got %d (body: %s)", http.StatusTooManyRequests, w.Code, w.Body.String())
	}
}

// TestGetAgentsHandler tests agent listing
func TestGetAgentsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	req := HTTPTestRequest{
		Method: http.MethodGet,
		Path:   "/api/v1/agents",
	}

	w := makeHTTPRequest(env.Server.getAgentsHandler, req)
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

	agents, ok := data["agents"].([]interface{})
	if !ok {
		t.Fatal("Expected agents array in data")
	}

	if len(agents) == 0 {
		t.Error("Expected at least one agent")
	}
}

// TestGetAppsHandler tests app listing
func TestGetAppsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	// Create issues for different apps
	for i, appID := range []string{"app1", "app2", "app3"} {
		issue := createTestIssue(
			fmt.Sprintf("issue-app-%d", i),
			fmt.Sprintf("Test %d", i),
			"bug",
			"medium",
			appID,
		)
		if _, err := env.Server.saveIssue(issue, "open"); err != nil {
			t.Fatalf("Failed to create test issue: %v", err)
		}
	}

	req := HTTPTestRequest{
		Method: http.MethodGet,
		Path:   "/api/v1/apps",
	}

	w := makeHTTPRequest(env.Server.getAppsHandler, req)
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

	apps, ok := data["apps"].([]interface{})
	if !ok {
		t.Fatal("Expected apps array in data")
	}

	if len(apps) < 3 {
		t.Errorf("Expected at least 3 apps, got %d", len(apps))
	}
}

// TestAgentSettingsHandlers tests agent settings get/update
func TestAgentSettingsHandlers(t *testing.T) {
	t.Skip("Agent settings require scenario root setup - skipping in unit tests")
}

// TestResetIssueCounterHandler tests counter reset
func TestResetIssueCounterHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	// Set a counter value
	for i := 0; i < 10; i++ {
		env.Server.processor.IncrementProcessedCount()
	}

	req := HTTPTestRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/automation/processor/reset-counter",
	}

	w := makeHTTPRequest(env.Server.resetIssueCounterHandler, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify counter was reset
	count := env.Server.processor.ProcessedCount()

	if count != 0 {
		t.Errorf("Expected counter to be 0, got %d", count)
	}
}

// TestGetRunningProcessesHandler tests running process tracking
func TestGetRunningProcessesHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	req := HTTPTestRequest{
		Method: http.MethodGet,
		Path:   "/api/v1/processes/running",
	}

	w := makeHTTPRequest(env.Server.getRunningProcessesHandler, req)
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

	if _, ok := data["processes"]; !ok {
		t.Error("Expected processes field in data")
	}
}

func TestStopRunningProcessHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	issueID := "issue-stop-123"
	env.Server.processor.RegisterRunningProcess(issueID, "agent-1", time.Now().UTC().Format(time.RFC3339), nil)

	req := HTTPTestRequest{
		Method:  http.MethodDelete,
		Path:    fmt.Sprintf("/api/v1/processes/running/%s", issueID),
		URLVars: map[string]string{"id": issueID},
	}

	w := makeHTTPRequest(env.Server.stopRunningProcessHandler, req)
	resp := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
		"success": true,
	})

	if resp == nil {
		return
	}

	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected data field in response")
	}

	if id, ok := data["issue_id"].(string); !ok || id != issueID {
		t.Errorf("Expected issue_id %s in response, got %#v", issueID, data["issue_id"])
	}

	// Subsequent stop should return not found
	w = makeHTTPRequest(env.Server.stopRunningProcessHandler, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for non-running issue, got %d", w.Code)
	}
}

// TestGetRateLimitStatusHandler tests rate limit status
func TestGetRateLimitStatusHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	req := HTTPTestRequest{
		Method: http.MethodGet,
		Path:   "/api/v1/rate-limit-status",
	}

	w := makeHTTPRequest(env.Server.getRateLimitStatusHandler, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestWebSocketHandler tests WebSocket endpoint
func TestWebSocketHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	// Note: WebSocket testing requires special handling
	// This is a basic sanity check
	req := HTTPTestRequest{
		Method: http.MethodGet,
		Path:   "/api/v1/ws",
	}

	w := makeHTTPRequest(env.Server.handleWebSocket, req)
	// WebSocket upgrade will fail in test (expected)
	// We're just making sure the handler doesn't panic
	if w == nil {
		t.Error("Expected response from WebSocket handler")
	}
}
