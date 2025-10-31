package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestCompleteFunnelFlow tests end-to-end funnel execution
func TestCompleteFunnelFlow(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testServer := setupTestServer(t)
	if testServer == nil {
		return
	}
	defer testServer.Cleanup()

	t.Run("CompleteUserJourney", func(t *testing.T) {
		// Step 1: Create funnel
		funnelData := map[string]interface{}{
			"name":        "Complete Journey Test",
			"description": "Testing full user journey",
			"steps": []map[string]interface{}{
				{
					"type":     "form",
					"position": 0,
					"title":    "Contact Information",
					"content":  json.RawMessage(`{"fields":[{"name":"email","type":"email","required":true},{"name":"name","type":"text","required":true}]}`),
				},
				{
					"type":     "quiz",
					"position": 1,
					"title":    "Your Interests",
					"content":  json.RawMessage(`{"question":"What interests you most?","options":["Product A","Product B","Product C"]}`),
				},
				{
					"type":     "cta",
					"position": 2,
					"title":    "Get Started",
					"content":  json.RawMessage(`{"buttonText":"Sign Up Now","destinationUrl":"https://example.com/signup"}`),
				},
			},
		}

		req, _ := makeHTTPRequest("POST", "/api/v1/funnels", funnelData)
		recorder := httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		var createResponse map[string]interface{}
		json.Unmarshal(recorder.Body.Bytes(), &createResponse)
		funnelID := createResponse["id"].(string)
		slug := createResponse["slug"].(string)
		defer cleanupTestData(t, testServer.Server, funnelID)

		// Step 2: Update funnel to active
		updateData := map[string]interface{}{
			"name":   "Complete Journey Test",
			"status": "active",
		}

		req, _ = makeHTTPRequest("PUT", "/api/v1/funnels/"+funnelID, updateData)
		recorder = httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Fatalf("Failed to activate funnel: %d", recorder.Code)
		}

		// Step 3: Start funnel execution
		req, _ = makeHTTPRequest("GET", fmt.Sprintf("/api/v1/execute/%s", slug), nil)
		req.RemoteAddr = "127.0.0.1:12345" // Set test IP address
		recorder = httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		var execResponse map[string]interface{}
		if err := json.Unmarshal(recorder.Body.Bytes(), &execResponse); err != nil {
			t.Fatalf("Failed to parse execution response: %v. Body: %s", err, recorder.Body.String())
		}

		if recorder.Code != http.StatusOK {
			t.Fatalf("Execute funnel failed with status %d: %s", recorder.Code, recorder.Body.String())
		}

		if execResponse["session_id"] == nil {
			t.Fatalf("No session_id in response: %+v", execResponse)
		}
		sessionID := execResponse["session_id"].(string)

		if execResponse["step"] == nil {
			t.Fatalf("No step in response: %+v", execResponse)
		}
		step1 := execResponse["step"].(map[string]interface{})

		if step1["id"] == nil {
			t.Fatalf("No step ID in response: %+v", step1)
		}
		step1ID := step1["id"].(string)

		if execResponse["progress"].(float64) != 0 {
			t.Error("Expected initial progress to be 0")
		}

		// Step 4: Submit first step (contact form)
		submitData := map[string]interface{}{
			"session_id":  sessionID,
			"step_id":     step1ID,
			"response":    json.RawMessage(`{"email":"journey@example.com","name":"Test User"}`),
			"duration_ms": 15000,
		}

		req, _ = makeHTTPRequest("POST", fmt.Sprintf("/api/v1/execute/%s/submit", slug), submitData)
		recorder = httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		var submit1Response map[string]interface{}
		json.Unmarshal(recorder.Body.Bytes(), &submit1Response)

		if submit1Response["progress"].(float64) <= 0 {
			t.Error("Expected progress to increase after first step")
		}

		step2 := submit1Response["next_step"].(map[string]interface{})
		step2ID := step2["id"].(string)

		// Step 5: Submit second step (quiz)
		submitData = map[string]interface{}{
			"session_id":  sessionID,
			"step_id":     step2ID,
			"response":    json.RawMessage(`{"selection":"Product B"}`),
			"duration_ms": 8000,
		}

		req, _ = makeHTTPRequest("POST", fmt.Sprintf("/api/v1/execute/%s/submit", slug), submitData)
		recorder = httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		var submit2Response map[string]interface{}
		json.Unmarshal(recorder.Body.Bytes(), &submit2Response)

		step3 := submit2Response["next_step"].(map[string]interface{})
		step3ID := step3["id"].(string)

		// Step 6: Submit final step (CTA)
		submitData = map[string]interface{}{
			"session_id":  sessionID,
			"step_id":     step3ID,
			"response":    json.RawMessage(`{"clicked":true}`),
			"duration_ms": 2000,
		}

		req, _ = makeHTTPRequest("POST", fmt.Sprintf("/api/v1/execute/%s/submit", slug), submitData)
		recorder = httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Fatalf("Final submit failed with status %d: %s", recorder.Code, recorder.Body.String())
		}

		var finalResponse map[string]interface{}
		if err := json.Unmarshal(recorder.Body.Bytes(), &finalResponse); err != nil {
			t.Fatalf("Failed to unmarshal final response: %v. Body: %s", err, recorder.Body.String())
		}

		t.Logf("Final response: %+v", finalResponse)

		if completed, ok := finalResponse["completed"].(bool); !ok || !completed {
			t.Errorf("Expected funnel to be completed. Got response: %+v", finalResponse)
		}

		// Step 7: Verify analytics
		req, _ = makeHTTPRequest("GET", "/api/v1/funnels/"+funnelID+"/analytics", nil)
		recorder = httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		var analytics map[string]interface{}
		json.Unmarshal(recorder.Body.Bytes(), &analytics)

		if analytics["totalViews"].(float64) < 1 {
			t.Error("Expected at least 1 view in analytics")
		}

		if analytics["capturedLeads"].(float64) < 1 {
			t.Error("Expected at least 1 captured lead")
		}

		if analytics["completedLeads"].(float64) < 1 {
			t.Error("Expected at least 1 completed lead")
		}

		// Step 8: Verify lead data
		// First check directly in the database
		var dbCompleted bool
		dbQuery := `SELECT completed FROM funnel_builder.leads WHERE funnel_id = $1 AND session_id = $2`
		err := testServer.Server.db.QueryRow(context.Background(), dbQuery, funnelID, sessionID).Scan(&dbCompleted)
		if err != nil {
			t.Fatalf("Failed to query lead from database: %v", err)
		}
		t.Logf("Database shows completed=%v for session %s", dbCompleted, sessionID)

		// Now check via API
		req, _ = makeHTTPRequest("GET", "/api/v1/funnels/"+funnelID+"/leads", nil)
		recorder = httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		var leads []map[string]interface{}
		if err := json.Unmarshal(recorder.Body.Bytes(), &leads); err != nil {
			t.Fatalf("Failed to unmarshal leads: %v. Body: %s", err, recorder.Body.String())
		}

		if len(leads) == 0 {
			t.Fatal("Expected at least one lead")
		}

		lead := leads[0]
		if lead["email"] != "journey@example.com" {
			t.Errorf("Expected email 'journey@example.com', got '%v'", lead["email"])
		}

		completed, ok := lead["completed"].(bool)
		if !ok {
			t.Errorf("Expected 'completed' to be a bool, got %T: %v. Full lead: %+v", lead["completed"], lead["completed"], lead)
		} else if !completed {
			t.Errorf("Expected lead to be marked as completed. DB says: %v, API returned: %v", dbCompleted, completed)
		}
	})
}

// TestMultipleSessionsFlow tests concurrent funnel sessions
func TestMultipleSessionsFlow(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testServer := setupTestServer(t)
	if testServer == nil {
		return
	}
	defer testServer.Cleanup()

	t.Run("ConcurrentSessions", func(t *testing.T) {
		funnelID := createTestFunnel(t, testServer.Server, "Concurrent Sessions Test")
		defer cleanupTestData(t, testServer.Server, funnelID)

		// Get funnel slug
		req, _ := makeHTTPRequest("GET", "/api/v1/funnels/"+funnelID, nil)
		recorder := httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		var funnel map[string]interface{}
		json.Unmarshal(recorder.Body.Bytes(), &funnel)
		slug := funnel["slug"].(string)

		// Update to active
		updateData := map[string]interface{}{
			"name":   "Concurrent Sessions Test",
			"status": "active",
		}
		req, _ = makeHTTPRequest("PUT", "/api/v1/funnels/"+funnelID, updateData)
		recorder = httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		// Create 5 concurrent sessions
		numSessions := 5
		sessionIDs := make([]string, numSessions)

		for i := 0; i < numSessions; i++ {
			req, _ := makeHTTPRequest("GET", fmt.Sprintf("/api/v1/execute/%s", slug), nil)
			recorder := httptest.NewRecorder()
			testServer.Server.router.ServeHTTP(recorder, req)

			if recorder.Code != http.StatusOK {
				t.Fatalf("Failed to execute funnel: status %d, body: %s", recorder.Code, recorder.Body.String())
			}

			var response map[string]interface{}
			json.Unmarshal(recorder.Body.Bytes(), &response)

			sessionID, ok := response["session_id"].(string)
			if !ok || sessionID == "" {
				t.Fatalf("Expected session_id in response, got: %v", response)
			}
			sessionIDs[i] = sessionID
		}

		// Verify all sessions are unique
		uniqueSessions := make(map[string]bool)
		for _, sid := range sessionIDs {
			uniqueSessions[sid] = true
		}

		if len(uniqueSessions) != numSessions {
			t.Errorf("Expected %d unique sessions, got %d", numSessions, len(uniqueSessions))
		}

		// Verify analytics shows all sessions
		req, _ = makeHTTPRequest("GET", "/api/v1/funnels/"+funnelID+"/analytics", nil)
		recorder = httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		var analytics map[string]interface{}
		json.Unmarshal(recorder.Body.Bytes(), &analytics)

		if analytics["totalViews"].(float64) < float64(numSessions) {
			t.Errorf("Expected at least %d views, got %v", numSessions, analytics["totalViews"])
		}
	})
}

// TestFunnelLifecycle tests complete CRUD operations
func TestFunnelLifecycle(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testServer := setupTestServer(t)
	if testServer == nil {
		return
	}
	defer testServer.Cleanup()

	t.Run("FullCRUDCycle", func(t *testing.T) {
		// CREATE
		createData := map[string]interface{}{
			"name":        "Lifecycle Test Funnel",
			"description": "Testing full lifecycle",
			"steps": []map[string]interface{}{
				{
					"type":     "form",
					"position": 0,
					"title":    "Initial Step",
					"content":  json.RawMessage(`{"fields":[{"name":"email","type":"email"}]}`),
				},
			},
		}

		req, _ := makeHTTPRequest("POST", "/api/v1/funnels", createData)
		recorder := httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		var createResponse map[string]interface{}
		json.Unmarshal(recorder.Body.Bytes(), &createResponse)
		funnelID := createResponse["id"].(string)
		defer cleanupTestData(t, testServer.Server, funnelID)

		// READ - Single
		req, _ = makeHTTPRequest("GET", "/api/v1/funnels/"+funnelID, nil)
		recorder = httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		var readResponse map[string]interface{}
		json.Unmarshal(recorder.Body.Bytes(), &readResponse)

		if readResponse["name"] != "Lifecycle Test Funnel" {
			t.Errorf("Expected name 'Lifecycle Test Funnel', got '%v'", readResponse["name"])
		}

		// READ - List
		req, _ = makeHTTPRequest("GET", "/api/v1/funnels", nil)
		recorder = httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		var funnels []map[string]interface{}
		json.Unmarshal(recorder.Body.Bytes(), &funnels)

		found := false
		for _, f := range funnels {
			if f["id"] == funnelID {
				found = true
				break
			}
		}
		if !found {
			t.Error("Created funnel not found in list")
		}

		// UPDATE
		updateData := map[string]interface{}{
			"name":        "Updated Lifecycle Funnel",
			"description": "Updated description",
			"status":      "active",
		}

		req, _ = makeHTTPRequest("PUT", "/api/v1/funnels/"+funnelID, updateData)
		recorder = httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Errorf("Update failed with status %d", recorder.Code)
		}

		// Verify update
		req, _ = makeHTTPRequest("GET", "/api/v1/funnels/"+funnelID, nil)
		recorder = httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		json.Unmarshal(recorder.Body.Bytes(), &readResponse)

		if readResponse["name"] != "Updated Lifecycle Funnel" {
			t.Errorf("Expected updated name, got '%v'", readResponse["name"])
		}

		if readResponse["status"] != "active" {
			t.Errorf("Expected status 'active', got '%v'", readResponse["status"])
		}

		// DELETE
		req, _ = makeHTTPRequest("DELETE", "/api/v1/funnels/"+funnelID, nil)
		recorder = httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Errorf("Delete failed with status %d", recorder.Code)
		}

		// Verify deletion
		req, _ = makeHTTPRequest("GET", "/api/v1/funnels/"+funnelID, nil)
		recorder = httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusNotFound {
			t.Error("Expected 404 after deletion")
		}
	})
}

// TestAnalyticsTracking tests event tracking and analytics
func TestAnalyticsTracking(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testServer := setupTestServer(t)
	if testServer == nil {
		return
	}
	defer testServer.Cleanup()

	t.Run("EventTracking", func(t *testing.T) {
		funnelID := createTestFunnel(t, testServer.Server, "Analytics Tracking Test")
		defer cleanupTestData(t, testServer.Server, funnelID)

		// Get funnel
		req, _ := makeHTTPRequest("GET", "/api/v1/funnels/"+funnelID, nil)
		recorder := httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		var funnel map[string]interface{}
		json.Unmarshal(recorder.Body.Bytes(), &funnel)
		slug := funnel["slug"].(string)

		// Update to active
		updateData := map[string]interface{}{
			"name":   "Analytics Tracking Test",
			"status": "active",
		}
		req, _ = makeHTTPRequest("PUT", "/api/v1/funnels/"+funnelID, updateData)
		recorder = httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		// Start session 1 - complete it
		req, _ = makeHTTPRequest("GET", fmt.Sprintf("/api/v1/execute/%s", slug), nil)
		recorder = httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		var exec1 map[string]interface{}
		json.Unmarshal(recorder.Body.Bytes(), &exec1)
		session1 := exec1["session_id"].(string)
		step1 := exec1["step"].(map[string]interface{})
		step1ID := step1["id"].(string)

		// Complete step 1
		submitData := map[string]interface{}{
			"session_id": session1,
			"step_id":    step1ID,
			"response":   json.RawMessage(`{"email":"complete@example.com"}`),
		}

		req, _ = makeHTTPRequest("POST", fmt.Sprintf("/api/v1/execute/%s/submit", slug), submitData)
		recorder = httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		var submit1 map[string]interface{}
		json.Unmarshal(recorder.Body.Bytes(), &submit1)
		step2 := submit1["next_step"].(map[string]interface{})
		step2ID := step2["id"].(string)

		// Complete step 2
		submitData = map[string]interface{}{
			"session_id": session1,
			"step_id":    step2ID,
			"response":   json.RawMessage(`{"answer":"Option A"}`),
		}

		req, _ = makeHTTPRequest("POST", fmt.Sprintf("/api/v1/execute/%s/submit", slug), submitData)
		recorder = httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		// Start session 2 - abandon after first step
		req, _ = makeHTTPRequest("GET", fmt.Sprintf("/api/v1/execute/%s", slug), nil)
		recorder = httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		// Check analytics
		req, _ = makeHTTPRequest("GET", "/api/v1/funnels/"+funnelID+"/analytics", nil)
		recorder = httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		var analytics map[string]interface{}
		json.Unmarshal(recorder.Body.Bytes(), &analytics)

		// Should have 2 total views
		if analytics["totalViews"].(float64) < 2 {
			t.Errorf("Expected at least 2 views, got %v", analytics["totalViews"])
		}

		// Should have captured and completed leads
		if analytics["capturedLeads"].(float64) < 1 {
			t.Errorf("Expected at least 1 captured lead, got %v", analytics["capturedLeads"])
		}

		if analytics["completedLeads"].(float64) < 1 {
			t.Errorf("Expected at least 1 completed lead, got %v", analytics["completedLeads"])
		}

		// Conversion rate should be calculated
		conversionRate := analytics["conversionRate"].(float64)
		if conversionRate <= 0 || conversionRate > 100 {
			t.Errorf("Expected valid conversion rate (0-100), got %v", conversionRate)
		}

		// Drop-off points should exist
		dropOffPoints := analytics["dropOffPoints"].([]interface{})
		if len(dropOffPoints) == 0 {
			t.Error("Expected drop-off points data")
		}
	})
}

// TestLeadDataPersistence tests lead data storage and retrieval
func TestLeadDataPersistence(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testServer := setupTestServer(t)
	if testServer == nil {
		return
	}
	defer testServer.Cleanup()

	t.Run("DataPersistence", func(t *testing.T) {
		funnelID := createTestFunnel(t, testServer.Server, "Lead Persistence Test")
		defer cleanupTestData(t, testServer.Server, funnelID)

		sessionID, slug := createTestLead(t, testServer.Server, funnelID)

		// Get step and submit with detailed data
		req, _ := makeHTTPRequest("GET", fmt.Sprintf("/api/v1/execute/%s?session_id=%s", slug, sessionID), nil)
		recorder := httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		var execResponse map[string]interface{}
		json.Unmarshal(recorder.Body.Bytes(), &execResponse)
		step := execResponse["step"].(map[string]interface{})
		stepID := step["id"].(string)

		// Submit with rich data
		submitData := map[string]interface{}{
			"session_id": sessionID,
			"step_id":    stepID,
			"response": json.RawMessage(`{
				"email": "persistence@example.com",
				"name": "Persistence Test",
				"phone": "+1234567890",
				"company": "Test Corp",
				"customField": "Custom Value"
			}`),
			"duration_ms": 12000,
		}

		req, _ = makeHTTPRequest("POST", fmt.Sprintf("/api/v1/execute/%s/submit", slug), submitData)
		recorder = httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		// Retrieve leads
		req, _ = makeHTTPRequest("GET", "/api/v1/funnels/"+funnelID+"/leads", nil)
		recorder = httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		var leads []map[string]interface{}
		json.Unmarshal(recorder.Body.Bytes(), &leads)

		if len(leads) == 0 {
			t.Fatal("Expected at least one lead")
		}

		// Find our lead
		var targetLead map[string]interface{}
		for _, lead := range leads {
			if email, ok := lead["email"].(string); ok && email == "persistence@example.com" {
				targetLead = lead
				break
			}
		}

		if targetLead == nil {
			t.Fatal("Could not find lead with test email")
		}

		// Verify all data persisted
		if targetLead["name"] != "Persistence Test" {
			t.Errorf("Expected name 'Persistence Test', got '%v'", targetLead["name"])
		}

		if targetLead["phone"] != "+1234567890" {
			t.Errorf("Expected phone '+1234567890', got '%v'", targetLead["phone"])
		}

		// Verify custom data in JSONB field
		data := targetLead["data"]
		if data == nil {
			t.Error("Expected data field to be populated")
		}
	})
}

// TestResponseTimeConstraints validates performance requirements
func TestResponseTimeConstraints(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testServer := setupTestServer(t)
	if testServer == nil {
		return
	}
	defer testServer.Cleanup()

	t.Run("PerformanceRequirements", func(t *testing.T) {
		funnelID := createTestFunnel(t, testServer.Server, "Performance Test")
		defer cleanupTestData(t, testServer.Server, funnelID)

		// Test health endpoint response time
		start := time.Now()
		req, _ := makeHTTPRequest("GET", "/health", nil)
		recorder := httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)
		assertResponseTime(t, start, 100*time.Millisecond, "Health check")

		// Test funnel retrieval response time
		start = time.Now()
		req, _ = makeHTTPRequest("GET", "/api/v1/funnels/"+funnelID, nil)
		recorder = httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)
		assertResponseTime(t, start, 200*time.Millisecond, "Funnel retrieval")

		// Test funnel list response time
		start = time.Now()
		req, _ = makeHTTPRequest("GET", "/api/v1/funnels", nil)
		recorder = httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)
		assertResponseTime(t, start, 300*time.Millisecond, "Funnel list")

		// Test analytics response time
		start = time.Now()
		req, _ = makeHTTPRequest("GET", "/api/v1/funnels/"+funnelID+"/analytics", nil)
		recorder = httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)
		assertResponseTime(t, start, 500*time.Millisecond, "Analytics retrieval")
	})
}
