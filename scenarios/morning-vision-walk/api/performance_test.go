// +build testing

package main

import (
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestPerformanceBenchmarks runs performance benchmarks for key operations
func TestPerformanceBenchmarks(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("PerformanceTests", func(t *testing.T) {
		testHealthCheckLatency(t)
		testSessionCreationThroughput(t)
		testMessageProcessingLatency(t)
		testConcurrentSessions(t)
		testMemoryUsage(t)
	})
}

func testHealthCheckLatency(t *testing.T) {
	t.Run("HealthCheck_Latency", func(t *testing.T) {
		iterations := 1000
		start := time.Now()

		for i := 0; i < iterations; i++ {
			w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
				Method: "GET",
				Path:   "/health",
			})
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			healthHandler(w, httpReq)

			if w.Code != 200 {
				t.Errorf("Health check failed on iteration %d", i)
			}
		}

		duration := time.Since(start)
		avgLatency := duration / time.Duration(iterations)

		t.Logf("Health check performance:")
		t.Logf("  Total: %v", duration)
		t.Logf("  Average: %v", avgLatency)
		t.Logf("  Throughput: %.2f req/s", float64(iterations)/duration.Seconds())

		// Health checks should be fast (< 5ms average)
		if avgLatency > 5*time.Millisecond {
			t.Logf("WARNING: Health check latency %v exceeds target of 5ms", avgLatency)
		}
	})
}

func testSessionCreationThroughput(t *testing.T) {
	t.Run("SessionCreation_Throughput", func(t *testing.T) {
		iterations := 100
		start := time.Now()
		sessionIDs := make([]string, 0, iterations)

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

			if w.Code == 200 {
				response := assertJSONResponse(t, w, 200, nil)
				if response != nil {
					if sessionID, ok := response["session_id"].(string); ok {
						sessionIDs = append(sessionIDs, sessionID)
					}
				}
			}
		}

		duration := time.Since(start)
		avgLatency := duration / time.Duration(iterations)

		t.Logf("Session creation performance:")
		t.Logf("  Total: %v", duration)
		t.Logf("  Average: %v", avgLatency)
		t.Logf("  Throughput: %.2f req/s", float64(iterations)/duration.Seconds())

		// Cleanup
		for _, sessionID := range sessionIDs {
			delete(activeSessions, sessionID)
		}

		// Session creation should complete reasonably (< 500ms average with n8n calls)
		if avgLatency > 500*time.Millisecond {
			t.Logf("WARNING: Session creation latency %v exceeds target of 500ms", avgLatency)
		}
	})
}

func testMessageProcessingLatency(t *testing.T) {
	t.Run("MessageProcessing_Latency", func(t *testing.T) {
		testSession := setupTestSession(t, "perf-user")
		defer testSession.Cleanup()

		iterations := 50
		start := time.Now()

		for i := 0; i < iterations; i++ {
			req := TestData.ConversationRequest(
				testSession.Session.ID,
				testSession.Session.UserID,
				fmt.Sprintf("Test message %d", i),
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

			if w.Code != 200 {
				t.Errorf("Message processing failed on iteration %d", i)
			}
		}

		duration := time.Since(start)
		avgLatency := duration / time.Duration(iterations)

		t.Logf("Message processing performance:")
		t.Logf("  Total: %v", duration)
		t.Logf("  Average: %v", avgLatency)
		t.Logf("  Throughput: %.2f msg/s", float64(iterations)/duration.Seconds())

		// Message processing should be reasonable (< 500ms average with n8n calls)
		if avgLatency > 500*time.Millisecond {
			t.Logf("WARNING: Message processing latency %v exceeds target of 500ms", avgLatency)
		}
	})
}

func testConcurrentSessions(t *testing.T) {
	t.Run("ConcurrentSessions_Stress", func(t *testing.T) {
		concurrency := 20
		messagesPerSession := 10

		var wg sync.WaitGroup
		start := time.Now()
		errors := make(chan error, concurrency*messagesPerSession)

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(sessionIndex int) {
				defer wg.Done()

				// Create session
				req := map[string]interface{}{
					"user_id": fmt.Sprintf("concurrent-user-%d", sessionIndex),
				}

				w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
					Method: "POST",
					Path:   "/api/conversation/start",
					Body:   req,
				})
				if err != nil {
					errors <- fmt.Errorf("session %d: failed to create request: %v", sessionIndex, err)
					return
				}

				startConversationHandler(w, httpReq)

				if w.Code != 200 {
					errors <- fmt.Errorf("session %d: failed to start (status %d)", sessionIndex, w.Code)
					return
				}

				response := assertJSONResponse(t, w, 200, nil)
				if response == nil {
					errors <- fmt.Errorf("session %d: nil response", sessionIndex)
					return
				}

				sessionID, ok := response["session_id"].(string)
				if !ok {
					errors <- fmt.Errorf("session %d: no session_id", sessionIndex)
					return
				}
				defer delete(activeSessions, sessionID)

				// Send messages
				for j := 0; j < messagesPerSession; j++ {
					msgReq := TestData.ConversationRequest(
						sessionID,
						fmt.Sprintf("concurrent-user-%d", sessionIndex),
						fmt.Sprintf("Message %d", j),
					)

					w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
						Method: "POST",
						Path:   "/api/conversation/message",
						Body:   msgReq,
					})
					if err != nil {
						errors <- fmt.Errorf("session %d message %d: %v", sessionIndex, j, err)
						continue
					}

					sendMessageHandler(w, httpReq)

					if w.Code != 200 {
						errors <- fmt.Errorf("session %d message %d: status %d", sessionIndex, j, w.Code)
					}
				}

				// End session
				endReq := map[string]interface{}{
					"session_id": sessionID,
				}

				w, httpReq, err = makeHTTPRequest(HTTPTestRequest{
					Method: "POST",
					Path:   "/api/conversation/end",
					Body:   endReq,
				})
				if err != nil {
					errors <- fmt.Errorf("session %d: failed to end: %v", sessionIndex, err)
					return
				}

				endConversationHandler(w, httpReq)

				if w.Code != 200 {
					errors <- fmt.Errorf("session %d: failed to end (status %d)", sessionIndex, w.Code)
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		duration := time.Since(start)
		totalOperations := concurrency * (1 + messagesPerSession + 1) // start + messages + end

		t.Logf("Concurrent sessions performance:")
		t.Logf("  Concurrency: %d sessions", concurrency)
		t.Logf("  Messages per session: %d", messagesPerSession)
		t.Logf("  Total operations: %d", totalOperations)
		t.Logf("  Total time: %v", duration)
		t.Logf("  Average throughput: %.2f ops/s", float64(totalOperations)/duration.Seconds())

		errorCount := 0
		for err := range errors {
			errorCount++
			if errorCount <= 10 { // Log first 10 errors
				t.Logf("  Error: %v", err)
			}
		}

		if errorCount > 0 {
			t.Logf("  Total errors: %d/%d (%.1f%%)", errorCount, totalOperations,
				float64(errorCount)/float64(totalOperations)*100)
		}

		// Allow some failures in concurrent stress test
		failureRate := float64(errorCount) / float64(totalOperations)
		if failureRate > 0.1 {
			t.Logf("WARNING: High failure rate: %.1f%%", failureRate*100)
		}
	})
}

func testMemoryUsage(t *testing.T) {
	t.Run("MemoryUsage_SessionGrowth", func(t *testing.T) {
		// Create many sessions and verify they can be cleaned up
		sessionCount := 100
		sessionIDs := make([]string, 0, sessionCount)

		// Create sessions
		for i := 0; i < sessionCount; i++ {
			req := map[string]interface{}{
				"user_id": fmt.Sprintf("memory-test-user-%d", i),
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

			if w.Code == 200 {
				response := assertJSONResponse(t, w, 200, nil)
				if response != nil {
					if sessionID, ok := response["session_id"].(string); ok {
						sessionIDs = append(sessionIDs, sessionID)

						// Add messages to some sessions
						if i%10 == 0 {
							for j := 0; j < 20; j++ {
								msgReq := TestData.ConversationRequest(
									sessionID,
									fmt.Sprintf("memory-test-user-%d", i),
									fmt.Sprintf("Test message %d", j),
								)

								w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
									Method: "POST",
									Path:   "/api/conversation/message",
									Body:   msgReq,
								})
								if err == nil {
									sendMessageHandler(w, httpReq)
								}
							}
						}
					}
				}
			}
		}

		t.Logf("Created %d sessions (%d with messages)", len(sessionIDs), sessionCount/10)
		t.Logf("Active sessions in memory: %d", len(activeSessions))

		// Cleanup all sessions
		for _, sessionID := range sessionIDs {
			delete(activeSessions, sessionID)
		}

		t.Logf("After cleanup: %d active sessions", len(activeSessions))

		if len(activeSessions) > sessionCount {
			t.Errorf("Expected at most %d active sessions, got %d", sessionCount, len(activeSessions))
		}
	})
}

// BenchmarkHealthHandler benchmarks the health endpoint
func BenchmarkHealthHandler(b *testing.B) {
	for i := 0; i < b.N; i++ {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})
		if err != nil {
			b.Fatalf("Failed to create request: %v", err)
		}

		healthHandler(w, httpReq)

		if w.Code != 200 {
			b.Errorf("Health check failed")
		}
	}
}

// BenchmarkSessionCreation benchmarks session creation
func BenchmarkSessionCreation(b *testing.B) {
	sessionIDs := make([]string, 0, b.N)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := map[string]interface{}{
			"user_id": fmt.Sprintf("bench-user-%d", i),
		}

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/conversation/start",
			Body:   req,
		})
		if err != nil {
			b.Fatalf("Failed to create request: %v", err)
		}

		startConversationHandler(w, httpReq)

		if w.Code == 200 {
			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err == nil {
				if sessionID, ok := response["session_id"].(string); ok {
					sessionIDs = append(sessionIDs, sessionID)
				}
			}
		}
	}
	b.StopTimer()

	// Cleanup
	for _, sessionID := range sessionIDs {
		delete(activeSessions, sessionID)
	}
}

// BenchmarkMessageProcessing benchmarks message processing
func BenchmarkMessageProcessing(b *testing.B) {
	// Setup test session
	sessionID := fmt.Sprintf("bench-session-%d", time.Now().UnixNano())
	session := &Session{
		ID:          sessionID,
		UserID:      "bench-user",
		StartTime:   time.Now(),
		Messages:    []ConversationMessage{},
		Insights:    []map[string]interface{}{},
		ActionItems: []string{},
	}
	activeSessions[sessionID] = session
	defer delete(activeSessions, sessionID)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := TestData.ConversationRequest(
			sessionID,
			"bench-user",
			fmt.Sprintf("Benchmark message %d", i),
		)

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/conversation/message",
			Body:   req,
		})
		if err != nil {
			b.Fatalf("Failed to create request: %v", err)
		}

		sendMessageHandler(w, httpReq)

		if w.Code != 200 {
			b.Errorf("Message processing failed")
		}
	}
}
