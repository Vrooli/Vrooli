package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMain(m *testing.M) {
	// Set lifecycle environment variable for tests
	os.Setenv("VROOLI_LIFECYCLE_MANAGED", "true")

	// Run tests
	code := m.Run()
	os.Exit(code)
}

// TestInitConfig tests configuration initialization
func TestInitConfig(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success_WithAllEnvVars", func(t *testing.T) {
		os.Setenv("API_PORT", "8080")
		os.Setenv("POSTGRES_URL", "postgres://user:pass@localhost:5432/testdb?sslmode=disable")
		os.Setenv("REDIS_ADDR", "localhost:6379")
		os.Setenv("REDIS_DB", "1")
		defer func() {
			os.Unsetenv("API_PORT")
			os.Unsetenv("POSTGRES_URL")
			os.Unsetenv("REDIS_ADDR")
			os.Unsetenv("REDIS_DB")
		}()

		config := initConfig()

		if config.Port != "8080" {
			t.Errorf("Expected port 8080, got %s", config.Port)
		}
		if config.RedisAddr != "localhost:6379" {
			t.Errorf("Expected Redis addr localhost:6379, got %s", config.RedisAddr)
		}
		if config.RedisDB != 1 {
			t.Errorf("Expected Redis DB 1, got %d", config.RedisDB)
		}
	})

	t.Run("Success_WithComponentEnvVars", func(t *testing.T) {
		os.Setenv("API_PORT", "9090")
		os.Setenv("POSTGRES_HOST", "dbhost")
		os.Setenv("POSTGRES_PORT", "5432")
		os.Setenv("POSTGRES_USER", "testuser")
		os.Setenv("POSTGRES_PASSWORD", "testpass")
		os.Setenv("POSTGRES_DB", "testdb")
		os.Setenv("REDIS_ADDR", "redis:6379")
		defer func() {
			os.Unsetenv("API_PORT")
			os.Unsetenv("POSTGRES_HOST")
			os.Unsetenv("POSTGRES_PORT")
			os.Unsetenv("POSTGRES_USER")
			os.Unsetenv("POSTGRES_PASSWORD")
			os.Unsetenv("POSTGRES_DB")
			os.Unsetenv("REDIS_ADDR")
		}()

		config := initConfig()

		if config.Port != "9090" {
			t.Errorf("Expected port 9090, got %s", config.Port)
		}
		expectedURL := "postgres://testuser:testpass@dbhost:5432/testdb?sslmode=disable"
		if config.PostgresURL != expectedURL {
			t.Errorf("Expected PostgresURL %s, got %s", expectedURL, config.PostgresURL)
		}
	})
}

// TestGetClientIP tests IP address extraction
func TestGetClientIP(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		expected   string
	}{
		{
			name:       "X-Forwarded-For",
			headers:    map[string]string{"X-Forwarded-For": "203.0.113.1, 198.51.100.1"},
			remoteAddr: "192.168.1.1:12345",
			expected:   "203.0.113.1",
		},
		{
			name:       "X-Real-IP",
			headers:    map[string]string{"X-Real-IP": "203.0.113.2"},
			remoteAddr: "192.168.1.1:12345",
			expected:   "203.0.113.2",
		},
		{
			name:       "RemoteAddr",
			headers:    map[string]string{},
			remoteAddr: "192.168.1.1:12345",
			expected:   "192.168.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			ip := getClientIP(req)
			if ip != tt.expected {
				t.Errorf("Expected IP %s, got %s", tt.expected, ip)
			}
		})
	}
}

// TestHealthHandler tests the health check endpoint
func TestHealthHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDatabase(t)
	if env == nil {
		t.Skip("Test database not available")
		return
	}
	defer env.Cleanup()

	t.Run("Success_AllServicesHealthy", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		healthHandler(w, req)

		health := assertHealthResponse(t, w, http.StatusOK)
		if health == nil {
			return
		}

		if health.Status != "healthy" {
			t.Errorf("Expected status 'healthy', got '%s'", health.Status)
		}
		if health.Services["postgres"] != "healthy" {
			t.Errorf("Expected postgres to be healthy, got '%s'", health.Services["postgres"])
		}
		if health.Services["redis"] != "healthy" {
			t.Errorf("Expected redis to be healthy, got '%s'", health.Services["redis"])
		}
	})

	t.Run("Error_DatabaseDown", func(t *testing.T) {
		// Close database to simulate failure
		originalDB := db
		db.Close()

		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		healthHandler(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("Expected status 503, got %d", w.Code)
		}

		// Restore database
		db = originalDB
	})
}

// TestTrackHandler tests the visitor tracking endpoint
func TestTrackHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDatabase(t)
	if env == nil {
		t.Skip("Test database not available")
		return
	}
	defer env.Cleanup()

	t.Run("Success_NewVisitor", func(t *testing.T) {
		event := TestData.TrackEventRequest(
			"fingerprint-new-visitor",
			"test-scenario",
			"page_view",
			"/test/page",
		)

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/visitor/track",
			Body:   event,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		trackHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusCreated, map[string]interface{}{
			"success": true,
		})

		if response != nil {
			if visitorID, ok := response["visitor_id"].(string); !ok || visitorID == "" {
				t.Error("Expected visitor_id in response")
			}
		}
	})

	t.Run("Success_ExistingVisitor", func(t *testing.T) {
		// Create a visitor first
		visitor := setupTestVisitor(t, "fingerprint-existing")
		defer visitor.Cleanup()

		event := TestData.TrackEventRequest(
			visitor.Visitor.Fingerprint,
			"test-scenario",
			"page_view",
			"/test/page",
		)

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/visitor/track",
			Body:   event,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		trackHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusCreated, map[string]interface{}{
			"success":    true,
			"visitor_id": visitor.Visitor.ID,
		})

		if response == nil {
			t.Error("Expected valid response")
		}
	})

	t.Run("Error_MissingFingerprint", func(t *testing.T) {
		event := map[string]interface{}{
			"scenario":   "test-scenario",
			"event_type": "page_view",
		}

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/visitor/track",
			Body:   event,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		trackHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
		})

		if response != nil {
			if msg, ok := response["message"].(string); !ok || msg == "" {
				t.Error("Expected error message in response")
			}
		}
	})

	t.Run("Error_MissingEventType", func(t *testing.T) {
		event := map[string]interface{}{
			"fingerprint": "test-fp",
			"scenario":    "test-scenario",
		}

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/visitor/track",
			Body:   event,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		trackHandler(w, httpReq)

		assertJSONResponse(t, w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
		})
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/visitor/track",
			Body:   `{"invalid": "json"`,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		trackHandler(w, httpReq)

		assertJSONResponse(t, w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
		})
	})

	t.Run("Error_InvalidMethod", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/visitor/track",
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		trackHandler(w, httpReq)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 405, got %d", w.Code)
		}
	})
}

// TestGetVisitorHandler tests the get visitor endpoint
func TestGetVisitorHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDatabase(t)
	if env == nil {
		t.Skip("Test database not available")
		return
	}
	defer env.Cleanup()

	t.Run("Success_VisitorFound", func(t *testing.T) {
		visitor := setupTestVisitor(t, "test-get-visitor")
		defer visitor.Cleanup()

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/api/v1/visitor/%s", visitor.Visitor.ID),
			URLVars: map[string]string{"id": visitor.Visitor.ID},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		getVisitorHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"id":          visitor.Visitor.ID,
			"fingerprint": visitor.Visitor.Fingerprint,
		})

		if response != nil {
			if sessionCount, ok := response["session_count"].(float64); !ok || sessionCount != float64(visitor.Visitor.SessionCount) {
				t.Errorf("Expected session_count %d, got %v", visitor.Visitor.SessionCount, sessionCount)
			}
		}
	})

	t.Run("Error_VisitorNotFound", func(t *testing.T) {
		nonExistentID := uuid.New().String()

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/api/v1/visitor/%s", nonExistentID),
			URLVars: map[string]string{"id": nonExistentID},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		getVisitorHandler(w, httpReq)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})

	// Run systematic error tests
	suite := &HandlerTestSuite{
		HandlerName: "getVisitorHandler",
		Handler:     getVisitorHandler,
		BaseURL:     "/api/v1/visitor/{id}",
	}

	patterns := NewTestScenarioBuilder().
		AddInvalidUUID("GET", "/api/v1/visitor/invalid-uuid").
		AddNonExistentVisitor("GET", "/api/v1/visitor/{id}").
		Build()

	suite.RunErrorTests(t, patterns)
}

// TestGetAnalyticsHandler tests the analytics endpoint
func TestGetAnalyticsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDatabase(t)
	if env == nil {
		t.Skip("Test database not available")
		return
	}
	defer env.Cleanup()

	t.Run("Success_WithDefaultTimeframe", func(t *testing.T) {
		// Create test visitor and sessions
		visitor := setupTestVisitor(t, "analytics-test-visitor")
		defer visitor.Cleanup()

		// Insert test sessions
		for _, session := range visitor.Sessions {
			insertQuery := `
				INSERT INTO visitor_sessions (id, visitor_id, scenario, start_time, page_views)
				VALUES ($1, $2, $3, $4, $5)
			`
			db.Exec(insertQuery, session.ID, session.VisitorID, session.Scenario,
				session.StartTime, session.PageViews)
		}

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/analytics/scenario/test-scenario",
			URLVars: map[string]string{"scenario": "test-scenario"},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		getAnalyticsHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"scenario": "test-scenario",
		})

		if response != nil {
			if _, ok := response["unique_visitors"]; !ok {
				t.Error("Expected unique_visitors in response")
			}
			if _, ok := response["total_sessions"]; !ok {
				t.Error("Expected total_sessions in response")
			}
		}
	})

	t.Run("Success_With1DayTimeframe", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:      "GET",
			Path:        "/api/v1/analytics/scenario/test-scenario",
			URLVars:     map[string]string{"scenario": "test-scenario"},
			QueryParams: map[string]string{"timeframe": "1d"},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		getAnalyticsHandler(w, httpReq)

		assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"scenario": "test-scenario",
		})
	})

	t.Run("Success_With30DayTimeframe", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:      "GET",
			Path:        "/api/v1/analytics/scenario/test-scenario",
			URLVars:     map[string]string{"scenario": "test-scenario"},
			QueryParams: map[string]string{"timeframe": "30d"},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		getAnalyticsHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response != nil {
			if bounceRate, ok := response["bounce_rate"].(float64); ok {
				if bounceRate < 0 || bounceRate > 1 {
					t.Errorf("Bounce rate should be between 0 and 1, got %f", bounceRate)
				}
			}
		}
	})

	t.Run("Success_NoDataForScenario", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/analytics/scenario/nonexistent-scenario",
			URLVars: map[string]string{"scenario": "nonexistent-scenario"},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		getAnalyticsHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"scenario":        "nonexistent-scenario",
			"unique_visitors": float64(0),
			"total_sessions":  float64(0),
		})

		if response == nil {
			t.Error("Expected valid response for scenario with no data")
		}
	})
}

// TestTrackerScriptHandler tests the tracker script serving endpoint
func TestTrackerScriptHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success_ScriptExists", func(t *testing.T) {
		// This test will only pass if the tracker.js file exists
		scriptPath := "../ui/tracker.js"

		// Check if file exists, if not skip
		if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
			t.Skip("Tracker script file not found, skipping test")
			return
		}

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/tracker.js",
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		trackerScriptHandler(w, httpReq)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "text/javascript" {
			t.Errorf("Expected Content-Type 'text/javascript', got '%s'", contentType)
		}

		cacheControl := w.Header().Get("Cache-Control")
		if cacheControl != "public, max-age=3600" {
			t.Errorf("Expected Cache-Control 'public, max-age=3600', got '%s'", cacheControl)
		}
	})
}

// TestGetOrCreateVisitor tests visitor creation and retrieval logic
func TestGetOrCreateVisitor(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDatabase(t)
	if env == nil {
		t.Skip("Test database not available")
		return
	}
	defer env.Cleanup()

	t.Run("Success_CreateNewVisitor", func(t *testing.T) {
		fingerprint := "new-visitor-fp-" + uuid.New().String()
		userAgent := "Test Browser 1.0"
		ipAddress := "192.168.1.100"

		visitor, err := getOrCreateVisitor(fingerprint, userAgent, ipAddress)
		if err != nil {
			t.Fatalf("Failed to create visitor: %v", err)
		}

		if visitor.Fingerprint != fingerprint {
			t.Errorf("Expected fingerprint %s, got %s", fingerprint, visitor.Fingerprint)
		}
		if visitor.UserAgent == nil || *visitor.UserAgent != userAgent {
			t.Errorf("Expected user agent %s, got %v", userAgent, visitor.UserAgent)
		}
		if visitor.SessionCount != 1 {
			t.Errorf("Expected session count 1, got %d", visitor.SessionCount)
		}

		// Cleanup
		db.Exec("DELETE FROM visitors WHERE id = $1", visitor.ID)
	})

	t.Run("Success_GetExistingVisitor", func(t *testing.T) {
		testVisitor := setupTestVisitor(t, "existing-visitor-fp")
		defer testVisitor.Cleanup()

		visitor, err := getOrCreateVisitor(testVisitor.Visitor.Fingerprint, "Updated Agent", "10.0.0.1")
		if err != nil {
			t.Fatalf("Failed to get visitor: %v", err)
		}

		if visitor.ID != testVisitor.Visitor.ID {
			t.Errorf("Expected visitor ID %s, got %s", testVisitor.Visitor.ID, visitor.ID)
		}
	})

	t.Run("Success_CachedVisitor", func(t *testing.T) {
		fingerprint := "cached-visitor-" + uuid.New().String()

		// Create visitor first time
		visitor1, err := getOrCreateVisitor(fingerprint, "Agent 1", "1.1.1.1")
		if err != nil {
			t.Fatalf("Failed to create visitor: %v", err)
		}
		defer db.Exec("DELETE FROM visitors WHERE id = $1", visitor1.ID)

		// Get visitor second time (should use cache)
		visitor2, err := getOrCreateVisitor(fingerprint, "Agent 2", "2.2.2.2")
		if err != nil {
			t.Fatalf("Failed to get cached visitor: %v", err)
		}

		if visitor1.ID != visitor2.ID {
			t.Error("Expected same visitor ID from cache")
		}
	})
}

// TestGetOrCreateSession tests session creation and retrieval logic
func TestGetOrCreateSession(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDatabase(t)
	if env == nil {
		t.Skip("Test database not available")
		return
	}
	defer env.Cleanup()

	t.Run("Success_CreateNewSession", func(t *testing.T) {
		visitor := setupTestVisitor(t, "session-test-visitor")
		defer visitor.Cleanup()

		properties := map[string]interface{}{
			"utm_source":   "test-source",
			"utm_medium":   "test-medium",
			"utm_campaign": "test-campaign",
			"referrer":     "https://example.com",
		}

		sessionID, err := getOrCreateSession(visitor.Visitor.ID, "test-scenario", "192.168.1.1", properties)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		if sessionID == "" {
			t.Error("Expected non-empty session ID")
		}

		// Verify session in database
		var count int
		db.QueryRow("SELECT COUNT(*) FROM visitor_sessions WHERE id = $1", sessionID).Scan(&count)
		if count != 1 {
			t.Errorf("Expected 1 session in database, got %d", count)
		}
	})

	t.Run("Success_GetActiveSession", func(t *testing.T) {
		visitor := setupTestVisitor(t, "active-session-visitor")
		defer visitor.Cleanup()

		// Create session first time
		sessionID1, err := getOrCreateSession(visitor.Visitor.ID, "test-scenario", "1.1.1.1", nil)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// Get session second time (should return same ID)
		sessionID2, err := getOrCreateSession(visitor.Visitor.ID, "test-scenario", "1.1.1.1", nil)
		if err != nil {
			t.Fatalf("Failed to get session: %v", err)
		}

		if sessionID1 != sessionID2 {
			t.Error("Expected same session ID for active session")
		}
	})
}

// TestPerformance_TrackingEndpoint tests performance of tracking endpoint
func TestPerformance_TrackingEndpoint(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDatabase(t)
	if env == nil {
		t.Skip("Test database not available")
		return
	}
	defer env.Cleanup()

	pattern := PerformanceTestPattern{
		Name:        "TrackingEndpoint",
		Description: "Test tracking endpoint performance",
		MaxDuration: 500 * time.Millisecond,
		Execute: func(t *testing.T, setupData interface{}) time.Duration {
			start := time.Now()

			event := TestData.TrackEventRequest(
				"perf-test-fp-"+uuid.New().String(),
				"perf-scenario",
				"page_view",
				"/test/page",
			)

			w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/visitor/track",
				Body:   event,
			})
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			trackHandler(w, httpReq)

			if w.Code != http.StatusCreated {
				t.Errorf("Expected status 201, got %d", w.Code)
			}

			return time.Since(start)
		},
	}

	RunPerformanceTest(t, pattern)
}

// TestPerformance_BulkTracking tests bulk tracking performance
func TestPerformance_BulkTracking(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDatabase(t)
	if env == nil {
		t.Skip("Test database not available")
		return
	}
	defer env.Cleanup()

	pattern := PerformanceTestPattern{
		Name:        "BulkTracking",
		Description: "Test bulk event tracking (100 events)",
		MaxDuration: 5 * time.Second,
		Execute: func(t *testing.T, setupData interface{}) time.Duration {
			start := time.Now()

			for i := 0; i < 100; i++ {
				event := TestData.TrackEventRequest(
					fmt.Sprintf("bulk-fp-%d", i%10), // Reuse 10 fingerprints
					"bulk-scenario",
					"page_view",
					fmt.Sprintf("/page/%d", i),
				)

				w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/visitor/track",
					Body:   event,
				})
				if err != nil {
					t.Fatalf("Failed to create request: %v", err)
				}

				trackHandler(w, httpReq)

				if w.Code != http.StatusCreated {
					t.Logf("Request %d failed with status %d", i, w.Code)
				}
			}

			return time.Since(start)
		},
	}

	RunPerformanceTest(t, pattern)
}

// TestEdgeCase_EmptyProperties tests tracking with empty properties
func TestEdgeCase_EmptyProperties(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDatabase(t)
	if env == nil {
		t.Skip("Test database not available")
		return
	}
	defer env.Cleanup()

	event := VisitorEvent{
		Fingerprint: "empty-props-fp",
		Scenario:    "test",
		EventType:   "page_view",
		PageURL:     "/test",
		Properties:  map[string]interface{}{},
	}

	w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
		Method: "POST",
		Path:   "/api/v1/visitor/track",
		Body:   event,
	})
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	trackHandler(w, httpReq)

	assertJSONResponse(t, w, http.StatusCreated, map[string]interface{}{
		"success": true,
	})
}

// TestEdgeCase_NullOptionalFields tests handling of null optional fields
func TestEdgeCase_NullOptionalFields(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDatabase(t)
	if env == nil {
		t.Skip("Test database not available")
		return
	}
	defer env.Cleanup()

	visitor := setupTestVisitor(t, "null-fields-test")
	defer visitor.Cleanup()

	w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
		Method:  "GET",
		Path:    fmt.Sprintf("/api/v1/visitor/%s", visitor.Visitor.ID),
		URLVars: map[string]string{"id": visitor.Visitor.ID},
	})
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	getVisitorHandler(w, httpReq)

	response := assertJSONResponse(t, w, http.StatusOK, nil)
	if response == nil {
		t.Error("Expected valid JSON response")
	}

	// Verify JSON marshaling handles null fields correctly
	var parsedVisitor Visitor
	if err := json.Unmarshal(w.Body.Bytes(), &parsedVisitor); err != nil {
		t.Errorf("Failed to parse visitor response: %v", err)
	}
}
