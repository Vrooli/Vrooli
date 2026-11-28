package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

// Test-only helpers moved from production files by the private-function debt lens.
// assertJSONResponse validates JSON response structure and status code
func assertJSONResponse(t *testing.T, recorder *httptest.ResponseRecorder, expectedStatus int, expectedKeys []string) map[string]interface{} {
	t.Helper()

	if recorder.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, recorder.Code, recorder.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v. Body: %s", err, recorder.Body.String())
	}

	for _, key := range expectedKeys {
		if _, exists := response[key]; !exists {
			t.Errorf("Expected key '%s' not found in response: %v", key, response)
		}
	}

	return response
}

// assertErrorResponse validates error response format
func assertErrorResponse(t *testing.T, recorder *httptest.ResponseRecorder, expectedStatus int) {
	t.Helper()

	if recorder.Code != expectedStatus {
		t.Errorf("Expected error status %d, got %d. Body: %s", expectedStatus, recorder.Code, recorder.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse error response: %v. Body: %s", err, recorder.Body.String())
	}

	if _, exists := response["error"]; !exists {
		t.Errorf("Expected 'error' field in error response: %v", response)
	}
}

// createTestFunnel creates a funnel for testing
func createTestFunnel(t *testing.T, server *Server, name string) string {
	t.Helper()

	projectID := createTestProject(t, server, fmt.Sprintf("%s Project", name))

	funnelData := map[string]interface{}{
		"name":        name,
		"description": "Test funnel for automated testing",
		"project_id":  projectID,
		"steps": []map[string]interface{}{
			{
				"type":     "form",
				"position": 0,
				"title":    "Contact Info",
				"content":  json.RawMessage(`{"fields":[{"name":"email","type":"email","required":true}]}`),
			},
			{
				"type":     "quiz",
				"position": 1,
				"title":    "Quick Quiz",
				"content":  json.RawMessage(`{"question":"What is your goal?","options":["Option A","Option B"]}`),
			},
		},
	}

	req, _ := makeHTTPRequest("POST", "/api/v1/funnels", funnelData)
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("Failed to create test funnel: status %d, body: %s", recorder.Code, recorder.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse create funnel response: %v", err)
	}

	funnelID, ok := response["id"].(string)
	if !ok || funnelID == "" {
		t.Fatalf("Funnel response missing id: %v", response)
	}

	testFunnelProjects.Store(funnelID, projectID)

	t.Cleanup(func() {
		cleanupTestData(t, server, funnelID)
	})

	return funnelID
}

// createTestLead creates a lead for testing
func createTestLead(t *testing.T, server *Server, funnelID string) (string, string) {
	t.Helper()

	sessionID := uuid.New().String()

	// Get funnel to find slug
	req, _ := makeHTTPRequest("GET", "/api/v1/funnels/"+funnelID, nil)
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, req)

	var funnel map[string]interface{}
	json.Unmarshal(recorder.Body.Bytes(), &funnel)

	if funnel["slug"] == nil {
		t.Fatalf("No slug in funnel response: %+v", funnel)
	}
	slug := funnel["slug"].(string)

	// First update funnel to active status
	updateData := map[string]interface{}{
		"name":   funnel["name"],
		"status": "active",
	}
	req, _ = makeHTTPRequest("PUT", "/api/v1/funnels/"+funnelID, updateData)
	recorder = httptest.NewRecorder()
	server.router.ServeHTTP(recorder, req)

	// Execute funnel to create lead with proper headers
	req, _ = makeHTTPRequest("GET", fmt.Sprintf("/api/v1/execute/%s?session_id=%s", slug, sessionID), nil)
	req.RemoteAddr = "127.0.0.1:12345" // Set test IP address
	recorder = httptest.NewRecorder()
	server.router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("Failed to create test lead: status %d, body: %s", recorder.Code, recorder.Body.String())
	}

	return sessionID, slug
}
func cleanupProject(t *testing.T, server *Server, projectID string) {
	t.Helper()

	if projectID == "" {
		return
	}

	ctx := context.Background()
	server.db.Exec(ctx, "DELETE FROM funnel_builder.projects WHERE id = $1", projectID)
}

// assertResponseTime validates response time is within limit
func assertResponseTime(t *testing.T, start time.Time, maxDuration time.Duration, operation string) {
	t.Helper()

	elapsed := time.Since(start)
	if elapsed > maxDuration {
		t.Errorf("%s took %v, expected < %v", operation, elapsed, maxDuration)
	}
}
