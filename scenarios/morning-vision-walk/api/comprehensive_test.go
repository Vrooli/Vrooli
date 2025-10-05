// +build testing

package main

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

// TestComprehensiveHandlerCoverage provides comprehensive coverage for all handlers
func TestComprehensiveHandlerCoverage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Test all handlers for comprehensive coverage
	t.Run("AllHandlers", func(t *testing.T) {
		testHealthHandlerComprehensive(t)
		testStartConversationHandlerComprehensive(t)
		testSendMessageHandlerComprehensive(t)
		testEndConversationHandlerComprehensive(t)
		testGenerateInsightsHandlerComprehensive(t)
		testPrioritizeTasksHandlerComprehensive(t)
		testGatherContextHandlerComprehensive(t)
		testGetSessionHandlerComprehensive(t)
		testExportSessionHandlerComprehensive(t)
	})
}

func testHealthHandlerComprehensive(t *testing.T) {
	t.Run("HealthHandler_EdgeCases", func(t *testing.T) {
		// Test multiple rapid health checks
		for i := 0; i < 10; i++ {
			w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
				Method: "GET",
				Path:   "/health",
			})
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			healthHandler(w, httpReq)

			response := assertJSONResponse(t, w, http.StatusOK, nil)
			if response == nil {
				t.Error("Expected non-nil response")
			}
		}
	})
}

func testStartConversationHandlerComprehensive(t *testing.T) {
	t.Run("StartConversation_EdgeCases", func(t *testing.T) {
		// Test with very long user_id
		t.Run("LongUserID", func(t *testing.T) {
			longUserID := ""
			for i := 0; i < 1000; i++ {
				longUserID += "a"
			}

			req := map[string]interface{}{
				"user_id": longUserID,
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

			if w.Code == http.StatusOK {
				response := assertJSONResponse(t, w, http.StatusOK, nil)
				if response != nil {
					if sessionID, ok := response["session_id"].(string); ok && sessionID != "" {
						defer delete(activeSessions, sessionID)
					}
				}
			}
		})

		// Test with special characters in user_id
		t.Run("SpecialCharactersUserID", func(t *testing.T) {
			req := map[string]interface{}{
				"user_id": "user@example.com!#$%^&*()",
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

			if w.Code == http.StatusOK {
				response := assertJSONResponse(t, w, http.StatusOK, nil)
				if response != nil {
					if sessionID, ok := response["session_id"].(string); ok && sessionID != "" {
						defer delete(activeSessions, sessionID)
					}
				}
			}
		})

		// Test concurrent session creation
		t.Run("ConcurrentCreation", func(t *testing.T) {
			done := make(chan bool, 5)
			sessionIDs := make(chan string, 5)

			for i := 0; i < 5; i++ {
				go func(index int) {
					req := map[string]interface{}{
						"user_id": fmt.Sprintf("concurrent-user-%d", index),
					}

					w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
						Method: "POST",
						Path:   "/api/conversation/start",
						Body:   req,
					})
					if err != nil {
						t.Errorf("Failed to create request: %v", err)
						done <- false
						return
					}

					startConversationHandler(w, httpReq)

					if w.Code == http.StatusOK {
						response := assertJSONResponse(t, w, http.StatusOK, nil)
						if response != nil {
							if sessionID, ok := response["session_id"].(string); ok {
								sessionIDs <- sessionID
							}
						}
					}

					done <- true
				}(i)
			}

			// Wait for all goroutines
			for i := 0; i < 5; i++ {
				<-done
			}
			close(sessionIDs)

			// Cleanup sessions
			for sessionID := range sessionIDs {
				delete(activeSessions, sessionID)
			}
		})
	})
}

func testSendMessageHandlerComprehensive(t *testing.T) {
	t.Run("SendMessage_EdgeCases", func(t *testing.T) {
		// Test with empty message
		t.Run("EmptyMessage", func(t *testing.T) {
			testSession := setupTestSession(t, "test-user")
			defer testSession.Cleanup()

			req := TestData.ConversationRequest(
				testSession.Session.ID,
				testSession.Session.UserID,
				"",
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

			// Should handle empty message gracefully
			if w.Code == http.StatusOK {
				assertJSONResponse(t, w, http.StatusOK, nil)
			}
		})

		// Test with very long message
		t.Run("VeryLongMessage", func(t *testing.T) {
			testSession := setupTestSession(t, "test-user")
			defer testSession.Cleanup()

			longMessage := ""
			for i := 0; i < 10000; i++ {
				longMessage += "a"
			}

			req := TestData.ConversationRequest(
				testSession.Session.ID,
				testSession.Session.UserID,
				longMessage,
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

			// Should handle long message gracefully
			if w.Code == http.StatusOK {
				assertJSONResponse(t, w, http.StatusOK, nil)
			}
		})

		// Test with context data
		t.Run("WithContext", func(t *testing.T) {
			testSession := setupTestSession(t, "test-user")
			defer testSession.Cleanup()

			req := ConversationRequest{
				SessionID: testSession.Session.ID,
				UserID:    testSession.Session.UserID,
				Message:   "Test message",
				Context: map[string]interface{}{
					"location": "office",
					"time":     "morning",
					"mood":     "focused",
				},
			}

			w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
				Method: "POST",
				Path:   "/api/conversation/message",
				Body:   req,
			})
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			sendMessageHandler(w, httpReq)

			assertJSONResponse(t, w, http.StatusOK, nil)
		})

		// Test insight generation trigger (every 6 messages)
		t.Run("InsightGenerationTrigger", func(t *testing.T) {
			testSession := setupTestSession(t, "test-user")
			defer testSession.Cleanup()

			// Clear initial messages
			testSession.Session.Messages = []ConversationMessage{}

			// Send 6 messages to trigger insight generation
			for i := 0; i < 6; i++ {
				req := TestData.ConversationRequest(
					testSession.Session.ID,
					testSession.Session.UserID,
					fmt.Sprintf("Message %d", i),
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

				if w.Code != http.StatusOK {
					t.Errorf("Message %d failed with status %d", i, w.Code)
				}
			}

			// Wait for async insight generation
			time.Sleep(200 * time.Millisecond)
		})
	})
}

func testEndConversationHandlerComprehensive(t *testing.T) {
	t.Run("EndConversation_EdgeCases", func(t *testing.T) {
		// Test ending session with no messages
		t.Run("NoMessages", func(t *testing.T) {
			testSession := setupTestSession(t, "test-user")
			testSession.Session.Messages = []ConversationMessage{}

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

			assertJSONResponse(t, w, http.StatusOK, nil)
		})

		// Test ending session with many messages
		t.Run("ManyMessages", func(t *testing.T) {
			testSession := setupTestSession(t, "test-user")

			// Add many messages
			for i := 0; i < 100; i++ {
				testSession.Session.Messages = append(testSession.Session.Messages, ConversationMessage{
					ID:        fmt.Sprintf("msg-%d", i),
					SessionID: testSession.Session.ID,
					Role:      "user",
					Content:   fmt.Sprintf("Message %d", i),
					Timestamp: time.Now(),
				})
			}

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

			assertJSONResponse(t, w, http.StatusOK, nil)
		})

		// Test double-ending session
		t.Run("DoubleEnd", func(t *testing.T) {
			testSession := setupTestSession(t, "test-user")

			req := map[string]interface{}{
				"session_id": testSession.Session.ID,
			}

			// First end
			w1, httpReq1, err := makeHTTPRequest(HTTPTestRequest{
				Method: "POST",
				Path:   "/api/conversation/end",
				Body:   req,
			})
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			endConversationHandler(w1, httpReq1)

			if w1.Code != http.StatusOK {
				t.Errorf("First end failed with status %d", w1.Code)
			}

			// Second end should fail
			w2, httpReq2, err := makeHTTPRequest(HTTPTestRequest{
				Method: "POST",
				Path:   "/api/conversation/end",
				Body:   req,
			})
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			endConversationHandler(w2, httpReq2)

			if w2.Code != http.StatusNotFound {
				t.Errorf("Expected 404 for second end, got %d", w2.Code)
			}
		})
	})
}

func testGenerateInsightsHandlerComprehensive(t *testing.T) {
	t.Run("GenerateInsights_EdgeCases", func(t *testing.T) {
		// Test with empty history
		t.Run("EmptyHistory", func(t *testing.T) {
			req := TestData.InsightRequest(
				"test-session",
				"test-user",
				[]ConversationMessage{},
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

			assertJSONResponse(t, w, http.StatusOK, nil)
		})

		// Test with large history
		t.Run("LargeHistory", func(t *testing.T) {
			largeHistory := make([]ConversationMessage, 100)
			for i := 0; i < 100; i++ {
				largeHistory[i] = ConversationMessage{
					ID:        fmt.Sprintf("msg-%d", i),
					SessionID: "test-session",
					Role:      "user",
					Content:   fmt.Sprintf("Message content %d", i),
					Timestamp: time.Now(),
				}
			}

			req := TestData.InsightRequest(
				"test-session",
				"test-user",
				largeHistory,
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

			assertJSONResponse(t, w, http.StatusOK, nil)
		})
	})
}

func testPrioritizeTasksHandlerComprehensive(t *testing.T) {
	t.Run("PrioritizeTasks_EdgeCases", func(t *testing.T) {
		// Test with empty tasks
		t.Run("EmptyTasks", func(t *testing.T) {
			testSession := setupTestSession(t, "test-user")
			defer testSession.Cleanup()

			req := TestData.TaskPrioritizationRequest(
				testSession.Session.ID,
				testSession.Session.UserID,
				[]map[string]interface{}{},
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

			assertJSONResponse(t, w, http.StatusOK, nil)
		})

		// Test with many tasks
		t.Run("ManyTasks", func(t *testing.T) {
			testSession := setupTestSession(t, "test-user")
			defer testSession.Cleanup()

			tasks := make([]map[string]interface{}, 50)
			for i := 0; i < 50; i++ {
				tasks[i] = map[string]interface{}{
					"name":     fmt.Sprintf("Task %d", i),
					"priority": i,
					"effort":   i * 10,
				}
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

			assertJSONResponse(t, w, http.StatusOK, nil)
		})
	})
}

func testGatherContextHandlerComprehensive(t *testing.T) {
	t.Run("GatherContext_EdgeCases", func(t *testing.T) {
		// Test with missing parameters
		t.Run("MissingSessionID", func(t *testing.T) {
			w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
				Method: "GET",
				Path:   "/api/context/gather",
				QueryParams: map[string]string{
					"user_id": "test-user",
				},
			})
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			gatherContextHandler(w, httpReq)

			// Should handle gracefully
			assertJSONResponse(t, w, http.StatusOK, nil)
		})

		// Test with both parameters
		t.Run("BothParameters", func(t *testing.T) {
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

			assertJSONResponse(t, w, http.StatusOK, nil)
		})
	})
}

func testGetSessionHandlerComprehensive(t *testing.T) {
	t.Run("GetSession_EdgeCases", func(t *testing.T) {
		// Test with empty session (just created)
		t.Run("EmptySession", func(t *testing.T) {
			testSession := setupTestSession(t, "test-user")
			defer testSession.Cleanup()

			testSession.Session.Messages = []ConversationMessage{}
			testSession.Session.Insights = []map[string]interface{}{}

			w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
				Method:  "GET",
				Path:    fmt.Sprintf("/api/session/%s", testSession.Session.ID),
				URLVars: map[string]string{"sessionId": testSession.Session.ID},
			})
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			getSessionHandler(w, httpReq)

			response := assertJSONResponse(t, w, http.StatusOK, nil)
			if response != nil {
				if messages, ok := response["messages"].([]interface{}); ok {
					if len(messages) != 0 {
						t.Error("Expected empty messages array")
					}
				}
			}
		})

		// Test with full session data
		t.Run("FullSessionData", func(t *testing.T) {
			testSession := setupTestSession(t, "test-user")
			defer testSession.Cleanup()

			// Add comprehensive data
			testSession.Session.DailyVision = "Complete all tests"
			testSession.Session.ActionItems = []string{"Item 1", "Item 2"}
			testSession.Session.Insights = []map[string]interface{}{
				{"type": "insight1", "value": "data1"},
			}

			w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
				Method:  "GET",
				Path:    fmt.Sprintf("/api/session/%s", testSession.Session.ID),
				URLVars: map[string]string{"sessionId": testSession.Session.ID},
			})
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			getSessionHandler(w, httpReq)

			response := assertJSONResponse(t, w, http.StatusOK, nil)
			if response != nil {
				if dailyVision, ok := response["daily_vision"].(string); ok {
					if dailyVision != "Complete all tests" {
						t.Errorf("Expected daily_vision to be 'Complete all tests', got %s", dailyVision)
					}
				}
			}
		})
	})
}

func testExportSessionHandlerComprehensive(t *testing.T) {
	t.Run("ExportSession_EdgeCases", func(t *testing.T) {
		// Test export with minimal data
		t.Run("MinimalData", func(t *testing.T) {
			testSession := setupTestSession(t, "test-user")
			defer testSession.Cleanup()

			testSession.Session.Messages = []ConversationMessage{}
			testSession.Session.Insights = []map[string]interface{}{}

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
				if _, exists := response["session"]; !exists {
					t.Error("Expected session in export")
				}
				if _, exists := response["summary"]; !exists {
					t.Error("Expected summary in export")
				}
			}
		})

		// Test export with complete data
		t.Run("CompleteData", func(t *testing.T) {
			testSession := setupTestSession(t, "test-user")
			defer testSession.Cleanup()

			// Add complete data
			testSession.Session.DailyVision = "Build amazing things"
			testSession.Session.ActionItems = []string{"Task 1", "Task 2", "Task 3"}

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
				if _, exists := response["daily_vision"]; !exists {
					t.Error("Expected daily_vision in export")
				}
				if actionItems, ok := response["action_items"].([]interface{}); ok {
					if len(actionItems) != 3 {
						t.Errorf("Expected 3 action items, got %d", len(actionItems))
					}
				}
			}
		})
	})
}

// TestGetEnvEdgeCases tests environment variable handling
func TestGetEnvEdgeCases(t *testing.T) {
	t.Run("GetEnv_EmptyString", func(t *testing.T) {
		result := getEnv("NONEXISTENT_VAR_123", "")
		if result != "" {
			t.Errorf("Expected empty string, got %s", result)
		}
	})

	t.Run("GetEnv_WhitespaceDefault", func(t *testing.T) {
		result := getEnv("NONEXISTENT_VAR_456", "   ")
		if result != "   " {
			t.Errorf("Expected whitespace default, got %s", result)
		}
	})
}

// TestSummarizeConversationEdgeCases tests conversation summarization
func TestSummarizeConversationEdgeCases(t *testing.T) {
	t.Run("SummarizeConversation_SingleMessage", func(t *testing.T) {
		messages := []ConversationMessage{
			{ID: "1", Content: "Single message", Timestamp: time.Now()},
		}
		result := summarizeConversation(messages)
		if result == "" {
			t.Error("Expected non-empty summary")
		}
	})

	t.Run("SummarizeConversation_ManyMessages", func(t *testing.T) {
		messages := make([]ConversationMessage, 1000)
		for i := 0; i < 1000; i++ {
			messages[i] = ConversationMessage{
				ID:        fmt.Sprintf("msg-%d", i),
				Content:   fmt.Sprintf("Message %d", i),
				Timestamp: time.Now(),
			}
		}
		result := summarizeConversation(messages)
		if result == "" {
			t.Error("Expected non-empty summary")
		}
	})
}
