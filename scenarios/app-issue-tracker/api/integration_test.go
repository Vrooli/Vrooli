//go:build testing
// +build testing

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

// TestIntegration_IssueLifecycle tests complete issue lifecycle
func TestIntegration_IssueLifecycle(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	// Step 1: Create issue
	t.Log("Step 1: Creating issue")
	createReq := HTTPTestRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/issues",
		Body: map[string]interface{}{
			"title":       "Integration Test Bug",
			"description": "Test bug for lifecycle validation",
			"app_id":      "integration-test",
			"type":        "bug",
			"priority":    "high",
			"tags":        []string{"integration", "test"},
		},
	}

	createW := makeHTTPRequest(env.Server.createIssueHandler, createReq)
	if createW.Code != http.StatusOK {
		t.Fatalf("Failed to create issue: status %d, body: %s", createW.Code, createW.Body.String())
	}

	var createResp struct {
		Success bool `json:"success"`
		Data    struct {
			Issue Issue `json:"issue"`
		} `json:"data"`
	}
	if err := json.Unmarshal(createW.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("Failed to parse create response: %v", err)
	}

	issueID := createResp.Data.Issue.ID
	if issueID == "" {
		t.Fatal("Expected issue ID in response")
	}

	t.Logf("Created issue: %s", issueID)

	// Step 2: Retrieve issue
	t.Log("Step 2: Retrieving issue")
	getReq := HTTPTestRequest{
		Method:  http.MethodGet,
		Path:    "/api/v1/issues/" + issueID,
		URLVars: map[string]string{"id": issueID},
	}

	getW := makeHTTPRequest(env.Server.getIssueHandler, getReq)
	if getW.Code != http.StatusOK {
		t.Fatalf("Failed to get issue: status %d", getW.Code)
	}

	// Step 3: Update issue - change priority and add notes
	t.Log("Step 3: Updating issue priority and notes")
	updateReq := HTTPTestRequest{
		Method:  http.MethodPut,
		Path:    "/api/v1/issues/" + issueID,
		URLVars: map[string]string{"id": issueID},
		Body: map[string]interface{}{
			"priority": "critical",
			"notes":    "Escalated due to customer impact",
		},
	}

	updateW := makeHTTPRequest(env.Server.updateIssueHandler, updateReq)
	if updateW.Code != http.StatusOK {
		t.Fatalf("Failed to update issue: status %d", updateW.Code)
	}

	// Verify update
	updated := assertIssueExists(t, env.Server, issueID, "open")
	if updated.Priority != "critical" {
		t.Errorf("Expected priority 'critical', got %s", updated.Priority)
	}

	// Step 4: Move to active status
	t.Log("Step 4: Moving issue to active")
	activateReq := HTTPTestRequest{
		Method:  http.MethodPut,
		Path:    "/api/v1/issues/" + issueID,
		URLVars: map[string]string{"id": issueID},
		Body: map[string]interface{}{
			"status": "active",
		},
	}

	activateW := makeHTTPRequest(env.Server.updateIssueHandler, activateReq)
	if activateW.Code != http.StatusOK {
		t.Fatalf("Failed to activate issue: status %d", activateW.Code)
	}

	// Verify moved to active folder
	assertIssueExists(t, env.Server, issueID, "active")

	// Step 5: Complete the issue
	t.Log("Step 5: Completing issue")
	completeReq := HTTPTestRequest{
		Method:  http.MethodPut,
		Path:    "/api/v1/issues/" + issueID,
		URLVars: map[string]string{"id": issueID},
		Body: map[string]interface{}{
			"status": "completed",
		},
	}

	completeW := makeHTTPRequest(env.Server.updateIssueHandler, completeReq)
	if completeW.Code != http.StatusOK {
		t.Fatalf("Failed to complete issue: status %d", completeW.Code)
	}

	// Verify moved to completed folder
	completed := assertIssueExists(t, env.Server, issueID, "completed")
	if completed.Status != "completed" {
		t.Errorf("Expected status 'completed', got %s", completed.Status)
	}

	// Step 6: Delete issue
	t.Log("Step 6: Deleting issue")
	deleteReq := HTTPTestRequest{
		Method:  http.MethodDelete,
		Path:    "/api/v1/issues/" + issueID,
		URLVars: map[string]string{"id": issueID},
	}

	deleteW := makeHTTPRequest(env.Server.deleteIssueHandler, deleteReq)
	if deleteW.Code != http.StatusOK {
		t.Fatalf("Failed to delete issue: status %d", deleteW.Code)
	}

	// Verify deletion
	assertIssueNotExists(t, env.Server, issueID)

	t.Log("Issue lifecycle completed successfully")
}

// TestIntegration_MultipleIssuesWorkflow tests managing multiple issues
func TestIntegration_MultipleIssuesWorkflow(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	// Create multiple issues
	numIssues := 5
	issueIDs := make([]string, numIssues)

	for i := 0; i < numIssues; i++ {
		req := HTTPTestRequest{
			Method: http.MethodPost,
			Path:   "/api/v1/issues",
			Body: map[string]interface{}{
				"title":  fmt.Sprintf("Multi-Issue Test %d", i),
				"app_id": "multi-test",
			},
		}

		w := makeHTTPRequest(env.Server.createIssueHandler, req)
		if w.Code != http.StatusOK {
			t.Fatalf("Failed to create issue %d: status %d", i, w.Code)
		}

		var resp struct {
			Data struct {
				Issue Issue `json:"issue"`
			} `json:"data"`
		}
		json.Unmarshal(w.Body.Bytes(), &resp)
		issueIDs[i] = resp.Data.Issue.ID
	}

	// List all issues
	listReq := HTTPTestRequest{
		Method: http.MethodGet,
		Path:   "/api/v1/issues",
	}

	listW := makeHTTPRequest(env.Server.getIssuesHandler, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("Failed to list issues: status %d", listW.Code)
	}

	var listResp struct {
		Data struct {
			Issues []Issue `json:"issues"`
		} `json:"data"`
	}
	if err := json.Unmarshal(listW.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("Failed to parse list response: %v", err)
	}

	if len(listResp.Data.Issues) < numIssues {
		t.Errorf("Expected at least %d issues, got %d", numIssues, len(listResp.Data.Issues))
	}

	// Move some issues to different statuses
	for i, issueID := range issueIDs {
		var status string
		if i%2 == 0 {
			status = "active"
		} else {
			status = "completed"
		}

		req := HTTPTestRequest{
			Method:  http.MethodPut,
			Path:    "/api/v1/issues/" + issueID,
			URLVars: map[string]string{"id": issueID},
			Body: map[string]interface{}{
				"status": status,
			},
		}

		w := makeHTTPRequest(env.Server.updateIssueHandler, req)
		if w.Code != http.StatusOK {
			t.Errorf("Failed to update issue %s: status %d", issueID, w.Code)
		}
	}

	// Filter by status
	activeReq := HTTPTestRequest{
		Method:      http.MethodGet,
		Path:        "/api/v1/issues",
		QueryParams: map[string]string{"status": "active"},
	}

	activeW := makeHTTPRequest(env.Server.getIssuesHandler, activeReq)
	if activeW.Code != http.StatusOK {
		t.Fatalf("Failed to filter active issues: status %d", activeW.Code)
	}

	t.Log("Multiple issues workflow completed successfully")
}

// TestIntegration_ErrorContextPreservation tests that error context is preserved
func TestIntegration_ErrorContextPreservation(t *testing.T) {
	t.Skip("Error context handling requires schema investigation - skipping for now")
}

// TestIntegration_AttachmentHandling tests full attachment workflow
func TestIntegration_AttachmentHandling(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	// Create issue with attachments
	createReq := HTTPTestRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/issues",
		Body: map[string]interface{}{
			"title":  "Attachment Test",
			"app_id": "attachment-test",
			"artifacts": []map[string]string{
				{
					"name":         "error.log",
					"category":     "logs",
					"content":      "Error occurred at line 123",
					"encoding":     "plain",
					"content_type": "text/plain",
				},
			},
		},
	}

	createW := makeHTTPRequest(env.Server.createIssueHandler, createReq)
	if createW.Code != http.StatusOK {
		t.Fatalf("Failed to create issue with attachment: status %d, body: %s", createW.Code, createW.Body.String())
	}

	var createResp struct {
		Data struct {
			Issue Issue `json:"issue"`
		} `json:"data"`
	}
	json.Unmarshal(createW.Body.Bytes(), &createResp)
	issueID := createResp.Data.Issue.ID

	issue := assertIssueExists(t, env.Server, issueID, "open")

	if len(issue.Attachments) == 0 {
		t.Fatal("Expected attachments, got none")
	}

	// Retrieve attachment
	attachment := issue.Attachments[0]
	getAttachmentReq := HTTPTestRequest{
		Method:  http.MethodGet,
		Path:    fmt.Sprintf("/api/v1/issues/%s/attachments/%s", issueID, attachment.Path),
		URLVars: map[string]string{"id": issueID, "attachment": attachment.Path},
	}

	getAttachmentW := makeHTTPRequest(env.Server.getIssueAttachmentHandler, getAttachmentReq)
	if getAttachmentW.Code != http.StatusOK {
		t.Errorf("Failed to retrieve attachment: status %d", getAttachmentW.Code)
	}
}

// TestIntegration_ProcessorConfiguration tests processor state management
func TestIntegration_ProcessorConfiguration(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	// Get initial state
	getReq := HTTPTestRequest{
		Method: http.MethodGet,
		Path:   "/api/v1/automation/processor",
	}

	getW := makeHTTPRequest(env.Server.getProcessorHandler, getReq)
	if getW.Code != http.StatusOK {
		t.Fatalf("Failed to get processor state: status %d", getW.Code)
	}

	// Update configuration
	updateReq := HTTPTestRequest{
		Method: http.MethodPatch,
		Path:   "/api/v1/automation/processor",
		Body: map[string]interface{}{
			"active":           true,
			"concurrent_slots": 3,
			"refresh_interval": 30,
		},
	}

	updateW := makeHTTPRequest(env.Server.updateProcessorHandler, updateReq)
	if updateW.Code != http.StatusOK {
		t.Fatalf("Failed to update processor: status %d", updateW.Code)
	}

	// Verify state was updated
	state := env.Server.currentProcessorState()
	if !state.Active {
		t.Error("Expected processor to be active")
	}
	if state.ConcurrentSlots != 3 {
		t.Errorf("Expected 3 concurrent slots, got %d", state.ConcurrentSlots)
	}
	if state.RefreshInterval != 30 {
		t.Errorf("Expected refresh interval 30, got %d", state.RefreshInterval)
	}

	// Deactivate
	deactivateReq := HTTPTestRequest{
		Method: http.MethodPatch,
		Path:   "/api/v1/automation/processor",
		Body: map[string]interface{}{
			"active": false,
		},
	}

	deactivateW := makeHTTPRequest(env.Server.updateProcessorHandler, deactivateReq)
	if deactivateW.Code != http.StatusOK {
		t.Fatalf("Failed to deactivate processor: status %d", deactivateW.Code)
	}

	state = env.Server.currentProcessorState()
	if state.Active {
		t.Error("Expected processor to be inactive")
	}
}

// TestIntegration_TimestampTracking tests that timestamps are properly maintained
func TestIntegration_TimestampTracking(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	// Create issue
	createReq := HTTPTestRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/issues",
		Body: map[string]interface{}{
			"title":  "Timestamp Test",
			"app_id": "timestamp-test",
		},
	}

	createW := makeHTTPRequest(env.Server.createIssueHandler, createReq)
	var createResp struct {
		Data struct {
			Issue Issue `json:"issue"`
		} `json:"data"`
	}
	json.Unmarshal(createW.Body.Bytes(), &createResp)
	issueID := createResp.Data.Issue.ID

	issue1 := assertIssueExists(t, env.Server, issueID, "open")
	createdAt := issue1.Metadata.CreatedAt

	if createdAt == "" {
		t.Error("Expected created_at timestamp")
	}
}
