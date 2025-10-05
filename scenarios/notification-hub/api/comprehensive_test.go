// +build testing

package main

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestComprehensiveProfileManagement tests all profile operations comprehensively
func TestComprehensiveProfileManagement(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("ProfileCRUD", func(t *testing.T) {
		// Create
		createReq := map[string]interface{}{
			"name": "Comprehensive Test Profile",
			"slug": "comp-test-profile",
			"plan": "enterprise",
			"settings": map[string]interface{}{
				"max_notifications": 5000,
				"retention_days":    90,
			},
		}

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/admin/profiles",
			Body:   createReq,
		})
		if err != nil {
			t.Fatalf("Failed to create profile: %v", err)
		}

		resp := assertJSONResponse(t, w, http.StatusCreated, []string{"id", "api_key"})
		profileID := resp["id"].(string)
		apiKey := resp["api_key"].(string)

		// Verify creation
		assertProfileExists(t, env.DB, uuid.MustParse(profileID))

		// Read
		w, err = makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   fmt.Sprintf("/api/v1/admin/profiles/%s", profileID),
		})
		if err != nil {
			t.Fatalf("Failed to get profile: %v", err)
		}
		assertJSONResponse(t, w, http.StatusOK, []string{"id", "name", "plan"})

		// Update
		updateReq := map[string]interface{}{
			"name": "Updated Profile Name",
			"settings": map[string]interface{}{
				"max_notifications": 10000,
			},
		}

		w, err = makeHTTPRequest(env, HTTPTestRequest{
			Method: "PUT",
			Path:   fmt.Sprintf("/api/v1/admin/profiles/%s", profileID),
			Body:   updateReq,
		})
		if err != nil {
			t.Fatalf("Failed to update profile: %v", err)
		}

		updateResp := assertJSONResponse(t, w, http.StatusOK, []string{"id", "name"})
		if updateResp["name"] != "Updated Profile Name" {
			t.Errorf("Profile name not updated correctly")
		}

		// Test API key authentication
		w, err = makeHTTPRequest(env, HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/api/v1/profiles/%s/notifications", profileID),
			Headers: map[string]string{"X-API-Key": apiKey},
		})
		if err != nil {
			t.Fatalf("Failed to authenticate with API key: %v", err)
		}
		if w.Code != http.StatusOK {
			t.Errorf("Expected successful authentication, got status %d", w.Code)
		}
	})

	t.Run("ProfileErrorCases", func(t *testing.T) {
		// Missing required fields
		patterns := NewTestScenarioBuilder().
			AddMissingRequiredField("/api/v1/admin/profiles", "POST", "name").
			AddNonExistentProfile("/api/v1/admin/profiles/%s", "GET").
			Build()

		RunErrorTests(t, env, patterns)
	})
}

// TestComprehensiveContactManagement tests all contact operations
func TestComprehensiveContactManagement(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("ContactCRUD", func(t *testing.T) {
		// Create contact
		contact, err := createTestContact(env.DB, env.TestProfile.ID)
		if err != nil {
			t.Fatalf("Failed to create test contact: %v", err)
		}

		// Verify creation
		assertContactExists(t, env.DB, contact.ID)

		// List contacts
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/api/v1/profiles/%s/contacts", env.TestProfile.ID),
			Headers: map[string]string{"X-API-Key": env.TestAPIKey},
		})
		if err != nil {
			t.Fatalf("Failed to list contacts: %v", err)
		}
		assertJSONResponse(t, w, http.StatusOK, []string{"contacts"})
	})

	t.Run("ContactPreferences", func(t *testing.T) {
		contact, err := createTestContact(env.DB, env.TestProfile.ID)
		if err != nil {
			t.Fatalf("Failed to create test contact: %v", err)
		}

		// Update preferences via API (when implemented)
		// For now, test that preferences exist in contact
		if contact.Preferences == nil {
			t.Error("Contact should have preferences")
		}
	})
}

// TestComprehensiveNotificationWorkflow tests the complete notification workflow
func TestComprehensiveNotificationWorkflow(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("EndToEndNotificationFlow", func(t *testing.T) {
		// 1. Create contact
		contact, err := createTestContact(env.DB, env.TestProfile.ID)
		if err != nil {
			t.Fatalf("Failed to create contact: %v", err)
		}

		// 2. Create template (optional)
		template, err := createTestTemplate(env.DB, env.TestProfile.ID)
		if err != nil {
			t.Fatalf("Failed to create template: %v", err)
		}

		// 3. Send notification using template
		sendReq := map[string]interface{}{
			"template_id": template.ID.String(),
			"recipients": []map[string]interface{}{
				{
					"contact_id": contact.ID.String(),
					"variables": map[string]interface{}{
						"name":    "John Doe",
						"message": "Welcome to our platform!",
					},
				},
			},
			"channels": []string{"email"},
			"priority": "high",
		}

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method:  "POST",
			Path:    fmt.Sprintf("/api/v1/profiles/%s/notifications/send", env.TestProfile.ID),
			Body:    sendReq,
			Headers: map[string]string{"X-API-Key": env.TestAPIKey},
		})
		if err != nil {
			t.Fatalf("Failed to send notification: %v", err)
		}

		resp := assertJSONResponse(t, w, http.StatusCreated, []string{"notifications"})
		notifications := resp["notifications"].([]interface{})
		if len(notifications) == 0 {
			t.Fatal("Expected at least one notification to be created")
		}

		notificationID := uuid.MustParse(notifications[0].(string))

		// 4. Verify notification status
		count, err := getNotificationCount(env.DB, env.TestProfile.ID, "pending")
		if err != nil {
			t.Fatalf("Failed to get notification count: %v", err)
		}
		if count == 0 {
			t.Error("Expected pending notification to exist")
		}

		// 5. Wait for processing (simulated)
		waitForCondition(t, func() bool {
			status, err := getNotificationCount(env.DB, env.TestProfile.ID, "processing")
			return err == nil && status > 0
		}, 5*time.Second, "notification to be processed")
	})

	t.Run("BulkNotifications", func(t *testing.T) {
		// Create multiple contacts
		contacts := make([]*Contact, 10)
		for i := 0; i < 10; i++ {
			contact, err := createTestContact(env.DB, env.TestProfile.ID)
			if err != nil {
				t.Fatalf("Failed to create contact %d: %v", i, err)
			}
			contacts[i] = contact
		}

		// Send bulk notifications
		recipients := make([]map[string]interface{}, len(contacts))
		for i, contact := range contacts {
			recipients[i] = map[string]interface{}{
				"contact_id": contact.ID.String(),
				"variables": map[string]interface{}{
					"name": fmt.Sprintf("User %d", i+1),
				},
			}
		}

		sendReq := map[string]interface{}{
			"recipients": recipients,
			"subject":    "Bulk Test Notification",
			"content": map[string]interface{}{
				"text": "Hello {{name}}!",
			},
			"channels": []string{"email"},
		}

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method:  "POST",
			Path:    fmt.Sprintf("/api/v1/profiles/%s/notifications/send", env.TestProfile.ID),
			Body:    sendReq,
			Headers: map[string]string{"X-API-Key": env.TestAPIKey},
		})
		if err != nil {
			t.Fatalf("Failed to send bulk notifications: %v", err)
		}

		resp := assertJSONResponse(t, w, http.StatusCreated, []string{"notifications"})
		notifs := resp["notifications"].([]interface{})
		if len(notifs) != 10 {
			t.Errorf("Expected 10 notifications, got %d", len(notifs))
		}
	})

	t.Run("ScheduledNotifications", func(t *testing.T) {
		contact, err := createTestContact(env.DB, env.TestProfile.ID)
		if err != nil {
			t.Fatalf("Failed to create contact: %v", err)
		}

		// Schedule notification for future
		futureTime := time.Now().Add(1 * time.Hour)
		sendReq := map[string]interface{}{
			"recipients": []map[string]interface{}{
				{
					"contact_id": contact.ID.String(),
					"variables":  map[string]interface{}{"name": "Future User"},
				},
			},
			"subject":      "Scheduled Notification",
			"content":      map[string]interface{}{"text": "This is scheduled"},
			"channels":     []string{"email"},
			"scheduled_at": futureTime.Format(time.RFC3339),
		}

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method:  "POST",
			Path:    fmt.Sprintf("/api/v1/profiles/%s/notifications/send", env.TestProfile.ID),
			Body:    sendReq,
			Headers: map[string]string{"X-API-Key": env.TestAPIKey},
		})
		if err != nil {
			t.Fatalf("Failed to schedule notification: %v", err)
		}

		assertJSONResponse(t, w, http.StatusCreated, []string{"notifications"})
	})
}

// TestUnsubscribeWorkflow tests unsubscribe functionality
func TestUnsubscribeWorkflow(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("UnsubscribeAndVerify", func(t *testing.T) {
		// Create contact
		contact, err := createTestContact(env.DB, env.TestProfile.ID)
		if err != nil {
			t.Fatalf("Failed to create contact: %v", err)
		}

		// Create unsubscribe record
		err = createUnsubscribe(env.DB, env.TestProfile.ID, contact.ID, "email")
		if err != nil {
			t.Fatalf("Failed to create unsubscribe: %v", err)
		}

		// Verify unsubscribe affects notification processing
		notification, err := createTestNotification(env.DB, env.TestProfile.ID, contact.ID)
		if err != nil {
			t.Fatalf("Failed to create notification: %v", err)
		}

		// Process notification (should skip due to unsubscribe)
		processor := env.Server.processor
		job := NotificationJob{
			NotificationID: notification.ID,
			ProfileID:      env.TestProfile.ID,
			ContactID:      contact.ID,
			Channel:        "email",
			Contact:        *contact,
		}

		processor.processNotification(job)

		// Verify delivery log shows unsubscribed
		logCount, err := getDeliveryLogCount(env.DB, notification.ID)
		if err != nil {
			t.Fatalf("Failed to get delivery log count: %v", err)
		}
		if logCount == 0 {
			t.Error("Expected delivery log to be created for unsubscribed contact")
		}
	})
}

// TestTemplateManagement tests template CRUD operations
func TestTemplateManagement(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("TemplateCreation", func(t *testing.T) {
		template, err := createTestTemplate(env.DB, env.TestProfile.ID)
		if err != nil {
			t.Fatalf("Failed to create template: %v", err)
		}

		// Verify template
		if template.ID == uuid.Nil {
			t.Error("Template ID should not be nil")
		}

		// List templates
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/api/v1/profiles/%s/templates", env.TestProfile.ID),
			Headers: map[string]string{"X-API-Key": env.TestAPIKey},
		})
		if err != nil {
			t.Fatalf("Failed to list templates: %v", err)
		}

		assertJSONResponse(t, w, http.StatusOK, []string{"templates"})
	})
}

// TestAnalyticsEndpoints tests analytics functionality
func TestAnalyticsEndpoints(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("DeliveryStatistics", func(t *testing.T) {
		// Create some test data
		contact, _ := createTestContact(env.DB, env.TestProfile.ID)
		notification, _ := createTestNotification(env.DB, env.TestProfile.ID, contact.ID)

		// Get delivery stats
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/api/v1/profiles/%s/analytics/delivery-stats", env.TestProfile.ID),
			Headers: map[string]string{"X-API-Key": env.TestAPIKey},
		})
		if err != nil {
			t.Fatalf("Failed to get delivery stats: %v", err)
		}

		assertJSONResponse(t, w, http.StatusOK, []string{"stats"})

		// Cleanup
		_ = notification
	})

	t.Run("DailyStatistics", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/api/v1/profiles/%s/analytics/daily-stats", env.TestProfile.ID),
			Headers: map[string]string{"X-API-Key": env.TestAPIKey},
		})
		if err != nil {
			t.Fatalf("Failed to get daily stats: %v", err)
		}

		assertJSONResponse(t, w, http.StatusOK, []string{"daily_stats"})
	})
}

// TestEdgeCases tests various edge cases
func TestEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("EdgeCaseScenarios", func(t *testing.T) {
		patterns := NewEdgeCaseTestBuilder().
			AddNullValueTest("notification_content",
				fmt.Sprintf("/api/v1/profiles/%s/notifications/send", env.TestProfile.ID),
				"POST",
				func() map[string]interface{} {
					contact, _ := createTestContact(env.DB, env.TestProfile.ID)
					return map[string]interface{}{
						"recipients": []map[string]interface{}{
							{"contact_id": contact.ID.String()},
						},
						"content": nil, // Null content
						"channels": []string{"email"},
					}
				}).
			AddBoundaryValueTest("large_recipient_list", func(t *testing.T, env *TestEnvironment) error {
				// Test with maximum allowed recipients
				recipients := make([]map[string]interface{}, 100)
				for i := 0; i < 100; i++ {
					contact, _ := createTestContact(env.DB, env.TestProfile.ID)
					recipients[i] = map[string]interface{}{
						"contact_id": contact.ID.String(),
						"variables":  map[string]interface{}{"name": fmt.Sprintf("User%d", i)},
					}
				}

				_, err := makeHTTPRequest(env, HTTPTestRequest{
					Method: "POST",
					Path:   fmt.Sprintf("/api/v1/profiles/%s/notifications/send", env.TestProfile.ID),
					Body: map[string]interface{}{
						"recipients": recipients,
						"subject":    "Bulk Test",
						"content":    map[string]interface{}{"text": "Test"},
						"channels":   []string{"email"},
					},
					Headers: map[string]string{"X-API-Key": env.TestAPIKey},
				})
				return err
			}).
			Build()

		RunEdgeCaseTests(t, env, patterns)
	})
}
