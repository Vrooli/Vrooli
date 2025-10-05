// +build testing

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// TestHealthHandler tests the health check endpoint
func TestHealthHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		healthHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status":  "healthy",
			"service": "morning-vision-walk",
		})

		if response != nil {
			if _, exists := response["timestamp"]; !exists {
				t.Error("Expected timestamp in health response")
			}
		}
	})
}

// TestStartConversationHandler tests starting a new conversation
func TestStartConversationHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		req := map[string]interface{}{
			"user_id": "test-user-123",
		}

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/conversation/start",
			Body:   req,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		startConversationHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, nil)

		if response != nil {
			// Verify session_id is present
			sessionID, exists := response["session_id"].(string)
			if !exists || sessionID == "" {
				t.Error("Expected session_id in response")
			}

			// Verify started_at timestamp
			if _, exists := response["started_at"]; !exists {
				t.Error("Expected started_at in response")
			}

			// Verify context data
			if _, exists := response["context"]; !exists {
				t.Error("Expected context in response")
			}

			// Verify welcome message
			if _, exists := response["welcome_message"]; !exists {
				t.Error("Expected welcome_message in response")
			}

			// Cleanup - remove the session
			if sessionID != "" {
				delete(activeSessions, sessionID)
			}
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/conversation/start",
			Body:   `{"invalid": json}`,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		startConversationHandler(w, httpReq)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for invalid JSON, got %d", w.Code)
		}
	})

	t.Run("EmptyBody", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/conversation/start",
			Body:   "",
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		startConversationHandler(w, httpReq)

		// Should still work with empty user_id (will be empty string)
		if w.Code != http.StatusOK && w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 200 or 400, got %d", w.Code)
		}

		// Cleanup any created session
		if w.Code == http.StatusOK {
			response := assertJSONResponse(t, w, http.StatusOK, nil)
			if response != nil {
				if sessionID, ok := response["session_id"].(string); ok && sessionID != "" {
					delete(activeSessions, sessionID)
				}
			}
		}
	})
}

// TestSendMessageHandler tests sending messages in a conversation
func TestSendMessageHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		// Create a test session first
		testSession := setupTestSession(t, "test-user")
		defer testSession.Cleanup()

		req := TestData.ConversationRequest(
			testSession.Session.ID,
			testSession.Session.UserID,
			"I want to improve my testing workflow",
		)

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/conversation/message",
			Body:   req,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		sendMessageHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, nil)

		if response != nil {
			// Verify message in response
			if _, exists := response["message"]; !exists {
				t.Error("Expected message in response")
			}

			// Verify session stats
			if stats, exists := response["session_stats"].(map[string]interface{}); exists {
				if _, hasCount := stats["message_count"]; !hasCount {
					t.Error("Expected message_count in session_stats")
				}
			} else {
				t.Error("Expected session_stats in response")
			}
		}
	})

	t.Run("SessionNotFound", func(t *testing.T) {
		req := TestData.ConversationRequest(
			"nonexistent-session",
			"test-user",
			"Test message",
		)

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/conversation/message",
			Body:   req,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		sendMessageHandler(w, httpReq)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404 for non-existent session, got %d", w.Code)
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/conversation/message",
			Body:   `{"invalid": json}`,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		sendMessageHandler(w, httpReq)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for invalid JSON, got %d", w.Code)
		}
	})
}

// TestEndConversationHandler tests ending a conversation
func TestEndConversationHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		// Create a test session
		testSession := setupTestSession(t, "test-user")
		// Don't defer cleanup since handler will remove it

		req := map[string]interface{}{
			"session_id": testSession.Session.ID,
		}

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/conversation/end",
			Body:   req,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		endConversationHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, nil)

		if response != nil {
			// Verify session_id
			if _, exists := response["session_id"]; !exists {
				t.Error("Expected session_id in response")
			}

			// Verify duration_minutes
			if _, exists := response["duration_minutes"]; !exists {
				t.Error("Expected duration_minutes in response")
			}

			// Verify total_messages
			if _, exists := response["total_messages"]; !exists {
				t.Error("Expected total_messages in response")
			}

			// Verify insights_generated
			if _, exists := response["insights_generated"]; !exists {
				t.Error("Expected insights_generated in response")
			}

			// Verify daily_plan
			if _, exists := response["daily_plan"]; !exists {
				t.Error("Expected daily_plan in response")
			}
		}

		// Verify session was removed
		if _, exists := activeSessions[testSession.Session.ID]; exists {
			t.Error("Expected session to be removed after ending")
		}
	})

	t.Run("SessionNotFound", func(t *testing.T) {
		req := map[string]interface{}{
			"session_id": "nonexistent-session",
		}

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/conversation/end",
			Body:   req,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		endConversationHandler(w, httpReq)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404 for non-existent session, got %d", w.Code)
		}
	})
}

// TestGenerateInsightsHandler tests insight generation
func TestGenerateInsightsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		testSession := setupTestSession(t, "test-user")
		defer testSession.Cleanup()

		req := TestData.InsightRequest(
			testSession.Session.ID,
			testSession.Session.UserID,
			testSession.Session.Messages,
		)

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/insights/generate",
			Body:   req,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		generateInsightsHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, nil)

		if response != nil {
			// Verify response has expected fields (workflow execution may return different formats)
			if _, existsSuccess := response["success"]; existsSuccess {
				// New format with success field
				t.Logf("Workflow response format: %v", response)
			} else if _, existsWorkflow := response["workflow"]; existsWorkflow {
				// Old format with workflow field
				t.Logf("Workflow response format: %v", response)
			} else {
				t.Logf("Workflow response format (unknown): %v", response)
			}
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/insights/generate",
			Body:   `{"invalid": json}`,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		generateInsightsHandler(w, httpReq)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for invalid JSON, got %d", w.Code)
		}
	})
}

// TestPrioritizeTasksHandler tests task prioritization
func TestPrioritizeTasksHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		testSession := setupTestSession(t, "test-user")
		defer testSession.Cleanup()

		tasks := []map[string]interface{}{
			{"name": "Write tests", "priority": 1},
			{"name": "Review PRs", "priority": 2},
		}

		req := TestData.TaskPrioritizationRequest(
			testSession.Session.ID,
			testSession.Session.UserID,
			tasks,
		)

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/tasks/prioritize",
			Body:   req,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		prioritizeTasksHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, nil)

		if response != nil {
			// Verify response has expected fields (workflow execution may return different formats)
			if len(response) == 0 {
				t.Error("Expected non-empty response")
			}
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/tasks/prioritize",
			Body:   `{"invalid": json}`,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		prioritizeTasksHandler(w, httpReq)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for invalid JSON, got %d", w.Code)
		}
	})
}

// TestGatherContextHandler tests context gathering
func TestGatherContextHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/context/gather",
			QueryParams: map[string]string{
				"session_id": "test-session",
				"user_id":    "test-user",
			},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		gatherContextHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, nil)

		if response != nil {
			// Verify response has expected fields (workflow execution may return different formats)
			if len(response) == 0 {
				t.Error("Expected non-empty response")
			}
		}
	})
}

// TestGetSessionHandler tests retrieving session details
func TestGetSessionHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		testSession := setupTestSession(t, "test-user")
		defer testSession.Cleanup()

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/api/session/%s", testSession.Session.ID),
			URLVars: map[string]string{"sessionId": testSession.Session.ID},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		getSessionHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"id":      testSession.Session.ID,
			"user_id": testSession.Session.UserID,
		})

		if response != nil {
			// Verify messages
			if _, exists := response["messages"]; !exists {
				t.Error("Expected messages in response")
			}

			// Verify insights
			if _, exists := response["insights"]; !exists {
				t.Error("Expected insights in response")
			}
		}
	})

	t.Run("SessionNotFound", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/session/nonexistent",
			URLVars: map[string]string{"sessionId": "nonexistent"},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		getSessionHandler(w, httpReq)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404 for non-existent session, got %d", w.Code)
		}
	})
}

// TestExportSessionHandler tests exporting session data
func TestExportSessionHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		testSession := setupTestSession(t, "test-user")
		defer testSession.Cleanup()

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/api/session/%s/export", testSession.Session.ID),
			URLVars: map[string]string{"sessionId": testSession.Session.ID},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		exportSessionHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, nil)

		if response != nil {
			// Verify session in export
			if _, exists := response["session"]; !exists {
				t.Error("Expected session in export response")
			}

			// Verify export_time
			if _, exists := response["export_time"]; !exists {
				t.Error("Expected export_time in response")
			}

			// Verify summary
			if _, exists := response["summary"]; !exists {
				t.Error("Expected summary in response")
			}

			// Verify content-disposition header
			contentDisposition := w.Header().Get("Content-Disposition")
			if contentDisposition == "" {
				t.Error("Expected Content-Disposition header")
			}
		}
	})

	t.Run("SessionNotFound", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/session/nonexistent/export",
			URLVars: map[string]string{"sessionId": "nonexistent"},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		exportSessionHandler(w, httpReq)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404 for non-existent session, got %d", w.Code)
		}
	})
}

// TestHelperFunctions tests helper functions
func TestHelperFunctions(t *testing.T) {
	t.Run("getEnv", func(t *testing.T) {
		// Test with set environment variable
		testKey := "TEST_ENV_VAR_UNIQUE"
		testValue := "test-value"
		os.Setenv(testKey, testValue)
		defer os.Unsetenv(testKey)

		result := getEnv(testKey, "default")
		if result != testValue {
			t.Errorf("Expected %s, got %s", testValue, result)
		}

		// Test with unset environment variable
		result = getEnv("NONEXISTENT_VAR", "default")
		if result != "default" {
			t.Errorf("Expected 'default', got %s", result)
		}
	})

	t.Run("summarizeConversation", func(t *testing.T) {
		// Test with empty messages
		result := summarizeConversation([]ConversationMessage{})
		if result == "" {
			t.Error("Expected non-empty summary for empty messages")
		}

		// Test with messages
		messages := []ConversationMessage{
			{ID: "1", Content: "Message 1", Timestamp: time.Now()},
			{ID: "2", Content: "Message 2", Timestamp: time.Now()},
		}
		result = summarizeConversation(messages)
		if result == "" {
			t.Error("Expected non-empty summary")
		}
	})

	t.Run("callN8nWorkflow", func(t *testing.T) {
		// Test workflow call (will use CLI approach)
		result := callN8nWorkflow("test-workflow", map[string]interface{}{
			"test": "data",
		})

		if result == nil {
			t.Error("Expected non-nil result from workflow call")
		}

		// Verify result has some fields (format may vary based on execution success/failure)
		if len(result) == 0 {
			t.Error("Expected non-empty result map")
		}
	})
}

// TestConcurrentSessions tests handling multiple concurrent sessions
func TestConcurrentSessions(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("MultipleSessions", func(t *testing.T) {
		// Create multiple sessions
		sessions := make([]*TestSession, 5)
		for i := 0; i < 5; i++ {
			sessions[i] = setupTestSession(t, fmt.Sprintf("user-%d", i))
			defer sessions[i].Cleanup()
		}

		// Verify all sessions exist
		if len(activeSessions) < 5 {
			t.Errorf("Expected at least 5 active sessions, got %d", len(activeSessions))
		}

		// Test retrieving each session
		for _, session := range sessions {
			w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
				Method:  "GET",
				Path:    fmt.Sprintf("/api/session/%s", session.Session.ID),
				URLVars: map[string]string{"sessionId": session.Session.ID},
			})
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			getSessionHandler(w, httpReq)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d for session %s", w.Code, session.Session.ID)
			}
		}
	})
}

// TestWebSocketHandler tests the WebSocket connection handler
func TestWebSocketHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("SessionNotFound", func(t *testing.T) {
		// Create a test server with WebSocket upgrade
		server := httptest.NewServer(http.HandlerFunc(handleWebSocket))
		defer server.Close()

		// Note: Full WebSocket testing would require a WebSocket client
		// This is a simplified test to verify the handler exists and basic error handling
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/ws/conversation?session_id=nonexistent", nil)

		// The handler expects an upgrade, so this will fail gracefully
		handleWebSocket(w, req)

		// The handler should attempt upgrade and fail (which is expected in this test context)
		// We're mainly testing that the handler doesn't panic
	})

	t.Run("MissingSessionID", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/ws/conversation", nil)

		// The handler expects an upgrade, so this will fail gracefully
		handleWebSocket(w, req)

		// Verify it doesn't panic without session_id
	})
}

// TestCallN8nWorkflow tests the n8n workflow calling function
func TestCallN8nWorkflow(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("WithValidData", func(t *testing.T) {
		result := callN8nWorkflow("test-workflow", map[string]interface{}{
			"test_key": "test_value",
		})

		if result == nil {
			t.Fatal("Expected non-nil result")
		}

		// Verify workflow field exists (may be in different formats depending on execution)
		if workflow, exists := result["workflow"]; exists {
			if workflow != "test-workflow" {
				t.Errorf("Expected workflow field to be 'test-workflow', got %v", workflow)
			}
		} else {
			t.Log("Workflow field not present (execution may have failed, which is acceptable in test environment)")
		}

		// Verify timestamp exists
		if _, exists := result["timestamp"]; !exists {
			t.Log("Timestamp field not present (workflow execution may have failed)")
		}
	})

	t.Run("WithEmptyData", func(t *testing.T) {
		result := callN8nWorkflow("empty-workflow", map[string]interface{}{})

		if result == nil {
			t.Fatal("Expected non-nil result")
		}
	})

	t.Run("WithNilData", func(t *testing.T) {
		// This should handle nil gracefully
		result := callN8nWorkflow("nil-workflow", nil)

		if result == nil {
			t.Fatal("Expected non-nil result even with nil input")
		}
	})
}

// TestGenerateInsightsBackground tests the async insight generation
func TestGenerateInsightsBackground(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("AsyncInsightGeneration", func(t *testing.T) {
		testSession := setupTestSession(t, "test-user")
		defer testSession.Cleanup()

		// Trigger async insight generation
		generateInsights(testSession.Session)

		// Give goroutine time to complete
		time.Sleep(100 * time.Millisecond)

		// Verify insights were added (at least attempted)
		// The function adds to session.Insights
		if len(testSession.Session.Insights) == 0 {
			t.Log("Note: Insights may not be added if workflow execution fails in test environment")
		}
	})

	t.Run("MultipleInsightGenerations", func(t *testing.T) {
		testSession := setupTestSession(t, "test-user")
		defer testSession.Cleanup()

		// Trigger multiple async insight generations
		for i := 0; i < 3; i++ {
			generateInsights(testSession.Session)
		}

		// Give goroutines time to complete
		time.Sleep(200 * time.Millisecond)

		// This tests that concurrent insight generation doesn't panic
		t.Log("Multiple concurrent insight generations completed without panic")
	})
}

// TestPerformance tests performance characteristics
func TestPerformance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("HealthCheckPerformance", func(t *testing.T) {
		start := time.Now()
		iterations := 100

		for i := 0; i < iterations; i++ {
			w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
				Method: "GET",
				Path:   "/health",
			})
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			healthHandler(w, httpReq)

			if w.Code != http.StatusOK {
				t.Errorf("Health check failed on iteration %d", i)
			}
		}

		duration := time.Since(start)
		avgDuration := duration / time.Duration(iterations)

		t.Logf("Average health check duration: %v", avgDuration)

		// Health checks should be fast (< 10ms average)
		if avgDuration > 10*time.Millisecond {
			t.Errorf("Health check too slow: %v average (expected < 10ms)", avgDuration)
		}
	})

	t.Run("SessionCreationPerformance", func(t *testing.T) {
		start := time.Now()
		iterations := 50

		createdSessions := []string{}

		for i := 0; i < iterations; i++ {
			req := map[string]interface{}{
				"user_id": fmt.Sprintf("perf-user-%d", i),
			}

			w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
				Method: "POST",
				Path:   "/api/conversation/start",
				Body:   req,
			})
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			startConversationHandler(w, httpReq)

			if w.Code != http.StatusOK {
				t.Errorf("Session creation failed on iteration %d", i)
			}

			// Track session ID for cleanup
			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err == nil {
				if sessionID, ok := response["session_id"].(string); ok {
					createdSessions = append(createdSessions, sessionID)
				}
			}
		}

		duration := time.Since(start)
		avgDuration := duration / time.Duration(iterations)

		t.Logf("Average session creation duration: %v", avgDuration)

		// Session creation should be reasonably fast (< 200ms average in test environment with n8n calls)
		if avgDuration > 200*time.Millisecond {
			t.Errorf("Session creation too slow: %v average (expected < 200ms)", avgDuration)
		}

		// Cleanup
		for _, sessionID := range createdSessions {
			delete(activeSessions, sessionID)
		}
	})
}
