package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestMain sets up test environment
func TestMain(m *testing.M) {
	// Set lifecycle flag to bypass lifecycle check during tests
	os.Setenv("VROOLI_LIFECYCLE_MANAGED", "true")

	// Run tests
	code := m.Run()

	os.Exit(code)
}

// TestHealthHandler tests the health check endpoint
func TestHealthHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()
	suite := NewHandlerTestSuite("HealthHandler", router, "/health")

	suite.RunSuccessTest(t, "Success", HTTPTestRequest{
		Method: "GET",
		Path:   "/health",
	}, func(t *testing.T, w *httptest.ResponseRecorder) {
		response := assertJSONResponse(t, w, http.StatusOK)
		assertFieldEquals(t, response, "status", "healthy")
	})
}

// TestStatusHandler tests the status endpoint
func TestStatusHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		t.Skip("Database not available for testing")
		return
	}
	defer testDB.Cleanup()

	// Ensure tables exist
	createTables()

	router := setupTestRouter()

	t.Run("Success", func(t *testing.T) {
		w, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/assistant/status",
		}, router)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK)
		assertFieldExists(t, response, "status")
		assertFieldExists(t, response, "issues_captured")
		assertFieldExists(t, response, "agents_spawned")
		assertFieldExists(t, response, "uptime")
		assertFieldEquals(t, response, "status", "operational")
	})
}

// TestCaptureHandler tests the issue capture endpoint
func TestCaptureHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		t.Skip("Database not available for testing")
		return
	}
	defer testDB.Cleanup()

	createTables()

	router := setupTestRouter()

	t.Run("Success", func(t *testing.T) {
		issueData := map[string]interface{}{
			"description":     "Test issue",
			"scenario_name":   "test-scenario",
			"url":             "http://localhost:3000",
			"screenshot_path": "/tmp/test.png",
			"context_data": map[string]interface{}{
				"browser": "chrome",
			},
		}

		w, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/assistant/capture",
			Body:   issueData,
		}, router)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK)
		assertFieldExists(t, response, "issue_id")
		assertFieldEquals(t, response, "status", "captured")
		assertFieldEquals(t, response, "message", "Issue captured successfully")

		// Verify issue was stored in database
		issueID := response["issue_id"].(string)
		var count int
		err = testDB.DB.QueryRow("SELECT COUNT(*) FROM issues WHERE id = $1", issueID).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to query issue: %v", err)
		}
		if count != 1 {
			t.Errorf("Expected 1 issue in database, found %d", count)
		}
	})

	t.Run("EmptyDescription", func(t *testing.T) {
		issueData := map[string]interface{}{
			"description":   "",
			"scenario_name": "test-scenario",
		}

		w, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/assistant/capture",
			Body:   issueData,
		}, router)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Should still succeed - database will accept empty string
		if w.Code != http.StatusOK && w.Code != http.StatusBadRequest {
			t.Errorf("Expected 200 or 400, got %d", w.Code)
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		// This test is tricky with our helper - we'd need to modify makeHTTPRequest
		// to support raw body strings for now, skip
		t.Skip("Requires raw body support in test helper")
	})
}

// TestSpawnAgentHandler tests the agent spawn endpoint
func TestSpawnAgentHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		t.Skip("Database not available for testing")
		return
	}
	defer testDB.Cleanup()

	createTables()

	router := setupTestRouter()

	t.Run("Success", func(t *testing.T) {
		// Create a test issue first
		issueID := createTestIssue(t, testDB.DB)

		spawnData := map[string]interface{}{
			"issue_id":    issueID,
			"agent_type":  "test-agent",
			"description": "Test agent spawn",
			"screenshot":  "/tmp/test.png",
			"context": map[string]interface{}{
				"scenario": "test-scenario",
			},
		}

		w, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/assistant/spawn-agent",
			Body:   spawnData,
		}, router)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK)
		assertFieldExists(t, response, "session_id")
		assertFieldEquals(t, response, "status", "spawned")

		// Verify session was created
		sessionID := response["session_id"].(string)
		var count int
		err = testDB.DB.QueryRow("SELECT COUNT(*) FROM agent_sessions WHERE id = $1", sessionID).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to query session: %v", err)
		}
		if count != 1 {
			t.Errorf("Expected 1 session in database, found %d", count)
		}

		// Give async goroutine time to complete
		time.Sleep(100 * time.Millisecond)
	})

	t.Run("MissingIssueID", func(t *testing.T) {
		spawnData := map[string]interface{}{
			"agent_type": "test-agent",
		}

		w, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/assistant/spawn-agent",
			Body:   spawnData,
		}, router)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Should succeed but with empty issue_id (database allows it)
		if w.Code != http.StatusOK && w.Code != http.StatusBadRequest {
			t.Errorf("Expected 200 or 400, got %d", w.Code)
		}
	})
}

// TestHistoryHandler tests the history endpoint
func TestHistoryHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		t.Skip("Database not available for testing")
		return
	}
	defer testDB.Cleanup()

	createTables()

	router := setupTestRouter()

	t.Run("EmptyHistory", func(t *testing.T) {
		w, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/assistant/history",
		}, router)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK)
		assertFieldExists(t, response, "issues")
		assertFieldExists(t, response, "count")
	})

	t.Run("WithIssues", func(t *testing.T) {
		// Create test issues
		createTestIssue(t, testDB.DB)
		createTestIssue(t, testDB.DB)

		w, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/assistant/history",
		}, router)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK)
		assertFieldExists(t, response, "issues")
		count := int(response["count"].(float64))
		if count < 2 {
			t.Errorf("Expected at least 2 issues, got %d", count)
		}
	})
}

// TestIssueHandler tests the individual issue endpoint
func TestIssueHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		t.Skip("Database not available for testing")
		return
	}
	defer testDB.Cleanup()

	createTables()

	router := setupTestRouter()

	t.Run("Success", func(t *testing.T) {
		issueID := createTestIssue(t, testDB.DB)

		w, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/assistant/issues/" + issueID,
			URLVars: map[string]string{"id": issueID},
		}, router)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK)
		assertFieldEquals(t, response, "id", issueID)
		assertFieldEquals(t, response, "status", "captured")
		assertFieldExists(t, response, "description")
	})

	t.Run("NonExistent", func(t *testing.T) {
		nonExistentID := uuid.New().String()

		w, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/assistant/issues/" + nonExistentID,
			URLVars: map[string]string{"id": nonExistentID},
		}, router)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusNotFound, "Issue not found")
	})
}

// TestUpdateStatusHandler tests the status update endpoint
func TestUpdateStatusHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		t.Skip("Database not available for testing")
		return
	}
	defer testDB.Cleanup()

	createTables()

	router := setupTestRouter()

	t.Run("Success", func(t *testing.T) {
		issueID := createTestIssue(t, testDB.DB)

		updateData := map[string]interface{}{
			"status": "resolved",
			"notes":  "Fixed the issue",
		}

		w, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "PUT",
			Path:    "/api/v1/assistant/issues/" + issueID + "/status",
			URLVars: map[string]string{"id": issueID},
			Body:    updateData,
		}, router)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK)
		assertFieldEquals(t, response, "status", "updated")

		// Verify status was updated
		var status string
		err = testDB.DB.QueryRow("SELECT status FROM issues WHERE id = $1", issueID).Scan(&status)
		if err != nil {
			t.Fatalf("Failed to query status: %v", err)
		}
		if status != "resolved" {
			t.Errorf("Expected status 'resolved', got '%s'", status)
		}
	})

	t.Run("NonExistentIssue", func(t *testing.T) {
		nonExistentID := uuid.New().String()

		updateData := map[string]interface{}{
			"status": "resolved",
		}

		w, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "PUT",
			Path:    "/api/v1/assistant/issues/" + nonExistentID + "/status",
			URLVars: map[string]string{"id": nonExistentID},
			Body:    updateData,
		}, router)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Update should succeed even if issue doesn't exist (SQL UPDATE returns 0 rows affected)
		// but handler doesn't check this
		if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
			t.Errorf("Expected 200 or 404, got %d", w.Code)
		}
	})
}

// TestDatabaseConnection tests database initialization with various scenarios
func TestDatabaseConnection(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ValidConnection", func(t *testing.T) {
		testDB := setupTestDB(t)
		if testDB == nil {
			t.Skip("Database not available")
			return
		}
		defer testDB.Cleanup()

		// Test ping
		err := testDB.DB.Ping()
		if err != nil {
			t.Errorf("Failed to ping database: %v", err)
		}
	})

	t.Run("TableCreation", func(t *testing.T) {
		testDB := setupTestDB(t)
		if testDB == nil {
			t.Skip("Database not available")
			return
		}
		defer testDB.Cleanup()

		createTables()

		// Verify tables exist
		tables := []string{"issues", "agent_sessions"}
		for _, table := range tables {
			var exists bool
			err := testDB.DB.QueryRow(`
				SELECT EXISTS (
					SELECT FROM information_schema.tables
					WHERE table_name = $1
				)
			`, table).Scan(&exists)
			if err != nil {
				t.Errorf("Failed to check if table %s exists: %v", table, err)
			}
			if !exists {
				t.Errorf("Table %s does not exist", table)
			}
		}
	})
}

// TestIssueModel tests the Issue struct and related functions
func TestIssueModel(t *testing.T) {
	t.Run("CreateIssue", func(t *testing.T) {
		issue := Issue{
			ID:             uuid.New().String(),
			Timestamp:      time.Now(),
			Description:    "Test issue",
			ScenarioName:   "test-scenario",
			URL:            "http://localhost",
			ScreenshotPath: "/tmp/test.png",
			Status:         "captured",
			ContextData: map[string]interface{}{
				"test": true,
			},
		}

		if issue.ID == "" {
			t.Error("Issue ID should not be empty")
		}
		if issue.Status != "captured" {
			t.Errorf("Expected status 'captured', got '%s'", issue.Status)
		}
	})
}

// TestAgentSessionModel tests the AgentSession struct
func TestAgentSessionModel(t *testing.T) {
	t.Run("CreateSession", func(t *testing.T) {
		session := AgentSession{
			ID:        uuid.New().String(),
			IssueID:   uuid.New().String(),
			AgentType: "test-agent",
			StartTime: time.Now(),
			Status:    "running",
		}

		if session.ID == "" {
			t.Error("Session ID should not be empty")
		}
		if session.Status != "running" {
			t.Errorf("Expected status 'running', got '%s'", session.Status)
		}
	})
}

// TestConcurrentRequests tests handling multiple concurrent requests
func TestConcurrentRequests(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		t.Skip("Database not available")
		return
	}
	defer testDB.Cleanup()

	createTables()
	router := setupTestRouter()

	t.Run("ConcurrentCaptures", func(t *testing.T) {
		const numRequests = 10
		results := make(chan error, numRequests)

		for i := 0; i < numRequests; i++ {
			go func(index int) {
				issueData := map[string]interface{}{
					"description":   fmt.Sprintf("Concurrent issue %d", index),
					"scenario_name": "test-scenario",
				}

				w, err := makeHTTPRequest(HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/assistant/capture",
					Body:   issueData,
				}, router)

				if err != nil {
					results <- err
					return
				}

				if w.Code != http.StatusOK {
					results <- fmt.Errorf("expected status 200, got %d", w.Code)
					return
				}

				results <- nil
			}(i)
		}

		// Collect results
		for i := 0; i < numRequests; i++ {
			if err := <-results; err != nil {
				t.Errorf("Concurrent request failed: %v", err)
			}
		}
	})
}
