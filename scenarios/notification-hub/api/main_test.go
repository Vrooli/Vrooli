// +build testing

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

// TestHealthCheck tests the health check endpoint
func TestHealthCheck(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("HealthCheckReturnsOK", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Health check should return 200 or 503 depending on dependencies
		if w.Code != http.StatusOK && w.Code != http.StatusServiceUnavailable {
			t.Errorf("Expected status 200 or 503, got %d", w.Code)
		}

		response := assertJSONResponse(t, w, w.Code, []string{"status", "service", "timestamp"})

		// Validate status field
		status, ok := response["status"].(string)
		if !ok {
			t.Error("Status field should be a string")
		}

		validStatuses := map[string]bool{"healthy": true, "degraded": true, "unhealthy": true}
		if !validStatuses[status] {
			t.Errorf("Invalid status: %s", status)
		}

		// Validate service field
		if service, ok := response["service"].(string); !ok || service != "notification-hub-api" {
			t.Errorf("Expected service 'notification-hub-api', got %v", response["service"])
		}

		// Validate dependencies exist
		if _, exists := response["dependencies"]; !exists {
			t.Error("Expected 'dependencies' field in health check response")
		}
	})

	t.Run("HealthCheckContainsDependencies", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		deps, ok := response["dependencies"].(map[string]interface{})
		if !ok {
			t.Fatal("Dependencies should be a map")
		}

		// Check for expected dependencies
		expectedDeps := []string{"database", "redis", "notification_processor", "profile_system", "template_system"}
		for _, depName := range expectedDeps {
			if _, exists := deps[depName]; !exists {
				t.Errorf("Expected dependency '%s' not found in health check", depName)
			}
		}
	})
}

// TestAPIDocsEndpoint tests the API documentation endpoint
func TestAPIDocsEndpoint(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("APIDocsReturnsHTML", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/docs",
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "text/html" {
			t.Errorf("Expected Content-Type 'text/html', got '%s'", contentType)
		}

		body := w.Body.String()
		if len(body) == 0 {
			t.Error("Expected non-empty response body")
		}
	})
}

// TestProfileManagement tests profile CRUD operations
func TestProfileManagement(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	var createdProfileID string

	t.Run("CreateProfile", func(t *testing.T) {
		req := map[string]interface{}{
			"name": "Test Profile API",
			"slug": "test-profile-api",
			"plan": "premium",
			"settings": map[string]interface{}{
				"notification_limit": 1000,
			},
		}

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/admin/profiles",
			Body:   req,
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusCreated, []string{"id", "name", "api_key", "status"})

		createdProfileID = response["id"].(string)

		// Validate response fields
		if response["name"] != "Test Profile API" {
			t.Errorf("Expected name 'Test Profile API', got %v", response["name"])
		}

		if response["status"] != "active" {
			t.Errorf("Expected status 'active', got %v", response["status"])
		}

		// API key should only be returned on creation
		if _, exists := response["api_key"]; !exists {
			t.Error("Expected api_key in creation response")
		}
	})

	t.Run("ListProfiles", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/admin/profiles",
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, []string{"profiles"})

		profiles, ok := response["profiles"].([]interface{})
		if !ok {
			t.Fatal("Expected profiles to be an array")
		}

		if len(profiles) == 0 {
			t.Error("Expected at least one profile in list")
		}
	})

	t.Run("GetProfile", func(t *testing.T) {
		if createdProfileID == "" {
			t.Skip("Profile creation failed, skipping get test")
		}

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   fmt.Sprintf("/api/v1/admin/profiles/%s", createdProfileID),
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, []string{"id", "name", "slug"})

		if response["id"] != createdProfileID {
			t.Errorf("Expected ID %s, got %v", createdProfileID, response["id"])
		}

		// API key should NOT be returned in get response
		if _, exists := response["api_key"]; exists {
			t.Error("API key should not be returned in get profile response")
		}
	})

	t.Run("UpdateProfile", func(t *testing.T) {
		if createdProfileID == "" {
			t.Skip("Profile creation failed, skipping update test")
		}

		updateReq := map[string]interface{}{
			"name": "Updated Profile Name",
			"plan": "enterprise",
		}

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "PUT",
			Path:   fmt.Sprintf("/api/v1/admin/profiles/%s", createdProfileID),
			Body:   updateReq,
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, []string{"id", "name", "plan"})

		if response["name"] != "Updated Profile Name" {
			t.Errorf("Expected updated name, got %v", response["name"])
		}

		if response["plan"] != "enterprise" {
			t.Errorf("Expected plan 'enterprise', got %v", response["plan"])
		}
	})

	t.Run("CreateProfile_MissingName", func(t *testing.T) {
		req := map[string]interface{}{
			"slug": "missing-name",
		}

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/admin/profiles",
			Body:   req,
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusBadRequest, "Invalid request")
	})
}

// TestNotificationSending tests notification sending functionality
func TestNotificationSending(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	// Create a test contact first
	contact, err := createTestContact(env.DB, env.TestProfile.ID)
	if err != nil {
		t.Fatalf("Failed to create test contact: %v", err)
	}

	t.Run("SendNotification_Success", func(t *testing.T) {
		req := map[string]interface{}{
			"recipients": []map[string]interface{}{
				{
					"contact_id": contact.ID.String(),
					"variables": map[string]interface{}{
						"name":    "John Doe",
						"message": "Hello from test",
					},
				},
			},
			"subject":  "Test Notification",
			"content": map[string]interface{}{
				"text": "This is a test notification for {{name}}: {{message}}",
				"html": "<p>This is a test notification for {{name}}: {{message}}</p>",
			},
			"channels": []string{"email"},
			"priority": "normal",
		}

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method:  "POST",
			Path:    fmt.Sprintf("/api/v1/profiles/%s/notifications/send", env.TestProfile.ID),
			Body:    req,
			Headers: map[string]string{"X-API-Key": env.TestAPIKey},
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusCreated, []string{"notifications", "message"})

		notifications, ok := response["notifications"].([]interface{})
		if !ok {
			t.Fatal("Expected notifications to be an array")
		}

		if len(notifications) != 1 {
			t.Errorf("Expected 1 notification, got %d", len(notifications))
		}
	})

	t.Run("SendNotification_MultipleRecipients", func(t *testing.T) {
		// Create another contact
		contact2, err := createTestContact(env.DB, env.TestProfile.ID)
		if err != nil {
			t.Fatalf("Failed to create second test contact: %v", err)
		}

		req := map[string]interface{}{
			"recipients": []map[string]interface{}{
				{
					"contact_id": contact.ID.String(),
					"variables":  map[string]interface{}{"name": "User 1"},
				},
				{
					"contact_id": contact2.ID.String(),
					"variables":  map[string]interface{}{"name": "User 2"},
				},
			},
			"subject":  "Bulk Notification",
			"content": map[string]interface{}{
				"text": "Hello {{name}}!",
			},
			"channels": []string{"email", "sms"},
		}

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method:  "POST",
			Path:    fmt.Sprintf("/api/v1/profiles/%s/notifications/send", env.TestProfile.ID),
			Body:    req,
			Headers: map[string]string{"X-API-Key": env.TestAPIKey},
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusCreated, []string{"notifications"})

		notifications, ok := response["notifications"].([]interface{})
		if !ok {
			t.Fatal("Expected notifications to be an array")
		}

		if len(notifications) != 2 {
			t.Errorf("Expected 2 notifications, got %d", len(notifications))
		}
	})

	t.Run("SendNotification_InvalidContactID", func(t *testing.T) {
		req := map[string]interface{}{
			"recipients": []map[string]interface{}{
				{
					"contact_id": "invalid-uuid",
					"variables":  map[string]interface{}{"name": "Test"},
				},
			},
			"subject":  "Test",
			"content": map[string]interface{}{"text": "Test"},
			"channels": []string{"email"},
		}

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method:  "POST",
			Path:    fmt.Sprintf("/api/v1/profiles/%s/notifications/send", env.TestProfile.ID),
			Body:    req,
			Headers: map[string]string{"X-API-Key": env.TestAPIKey},
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusBadRequest, "Invalid contact_id")
	})

	t.Run("SendNotification_MissingRecipients", func(t *testing.T) {
		req := map[string]interface{}{
			"subject":  "Test",
			"content": map[string]interface{}{"text": "Test"},
			"channels": []string{"email"},
		}

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method:  "POST",
			Path:    fmt.Sprintf("/api/v1/profiles/%s/notifications/send", env.TestProfile.ID),
			Body:    req,
			Headers: map[string]string{"X-API-Key": env.TestAPIKey},
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Should fail due to missing recipients
		if w.Code == http.StatusCreated {
			t.Error("Expected error for missing recipients, got success")
		}
	})
}

// TestAuthentication tests API key authentication
func TestAuthentication(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	basePath := fmt.Sprintf("/api/v1/profiles/%s/notifications", env.TestProfile.ID)

	t.Run("WithoutAPIKey", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method:  "GET",
			Path:    basePath,
			Headers: map[string]string{}, // No API key
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusUnauthorized, "API key required")
	})

	t.Run("WithInvalidAPIKey", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method:  "GET",
			Path:    basePath,
			Headers: map[string]string{"X-API-Key": "invalid-key-12345"},
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusUnauthorized, "Invalid API key or profile")
	})

	t.Run("WithValidAPIKey", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method:  "GET",
			Path:    basePath,
			Headers: map[string]string{"X-API-Key": env.TestAPIKey},
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Should return 200 for valid API key
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", w.Code, w.Body.String())
		}
	})

	t.Run("WithBearerToken", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method:  "GET",
			Path:    basePath,
			Headers: map[string]string{"Authorization": "Bearer " + env.TestAPIKey},
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Should accept Bearer token format
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 with Bearer token, got %d", w.Code)
		}
	})
}

// TestErrorPatterns tests systematic error conditions
func TestErrorPatterns(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	// Test notification endpoints with error patterns
	notificationPatterns := NewTestScenarioBuilder().
		AddMissingAPIKey("/api/v1/profiles/%s/notifications", "GET").
		AddInvalidAPIKey("/api/v1/profiles/%s/notifications", "GET").
		AddInvalidJSON("/api/v1/profiles/%s/notifications/send", "POST").
		Build()

	RunErrorTests(t, env, notificationPatterns)

	// Test contact endpoints with error patterns
	contactPatterns := NewTestScenarioBuilder().
		AddMissingAPIKey("/api/v1/profiles/%s/contacts", "GET").
		AddInvalidJSON("/api/v1/profiles/%s/contacts", "POST").
		Build()

	RunErrorTests(t, env, contactPatterns)
}

// TestNotificationListing tests listing notifications
func TestNotificationListing(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("ListNotifications_Empty", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/api/v1/profiles/%s/notifications", env.TestProfile.ID),
			Headers: map[string]string{"X-API-Key": env.TestAPIKey},
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, []string{"notifications"})

		notifications, ok := response["notifications"].([]interface{})
		if !ok {
			t.Fatal("Expected notifications to be an array")
		}

		// Should return empty array, not null
		if notifications == nil {
			t.Error("Expected empty array, got nil")
		}
	})
}

// TestContactManagement tests contact CRUD operations
func TestContactManagement(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("ListContacts_Empty", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/api/v1/profiles/%s/contacts", env.TestProfile.ID),
			Headers: map[string]string{"X-API-Key": env.TestAPIKey},
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, []string{"contacts"})

		contacts, ok := response["contacts"].([]interface{})
		if !ok {
			t.Fatal("Expected contacts to be an array")
		}

		if contacts == nil {
			t.Error("Expected empty array, got nil")
		}
	})
}

// TestTemplateManagement tests template CRUD operations
func TestTemplateManagement(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("ListTemplates_Empty", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/api/v1/profiles/%s/templates", env.TestProfile.ID),
			Headers: map[string]string{"X-API-Key": env.TestAPIKey},
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, []string{"templates"})

		templates, ok := response["templates"].([]interface{})
		if !ok {
			t.Fatal("Expected templates to be an array")
		}

		if templates == nil {
			t.Error("Expected empty array, got nil")
		}
	})
}

// TestAnalytics tests analytics endpoints
func TestAnalytics(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("GetDeliveryStats", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/api/v1/profiles/%s/analytics/delivery-stats", env.TestProfile.ID),
			Headers: map[string]string{"X-API-Key": env.TestAPIKey},
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, []string{"stats"})

		if response["stats"] == nil {
			t.Error("Expected stats object, got nil")
		}
	})

	t.Run("GetDailyStats", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/api/v1/profiles/%s/analytics/daily-stats", env.TestProfile.ID),
			Headers: map[string]string{"X-API-Key": env.TestAPIKey},
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, []string{"daily_stats"})

		dailyStats, ok := response["daily_stats"].([]interface{})
		if !ok {
			t.Fatal("Expected daily_stats to be an array")
		}

		if dailyStats == nil {
			t.Error("Expected empty array, got nil")
		}
	})
}

// TestUnsubscribeWebhook tests the unsubscribe webhook endpoint
func TestUnsubscribeWebhook(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("UnsubscribeWebhook", func(t *testing.T) {
		req := map[string]interface{}{
			"email":   "user@example.com",
			"reason":  "no longer interested",
			"profile": env.TestProfile.ID.String(),
		}

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/webhooks/unsubscribe",
			Body:   req,
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Unsubscribe webhook should always return 200
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", w.Code, w.Body.String())
		}
	})
}
