
package main

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestNotificationProcessor tests the notification processing logic
func TestNotificationProcessor(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("ProcessorInitialization", func(t *testing.T) {
		if env.Server.processor == nil {
			t.Fatal("Processor should be initialized")
		}

		if env.Server.processor.db == nil {
			t.Error("Processor database connection should not be nil")
		}

		if env.Server.processor.redisClient == nil {
			t.Error("Processor Redis client should not be nil")
		}

		if env.Server.processor.workers != 5 {
			t.Errorf("Expected 5 workers, got %d", env.Server.processor.workers)
		}
	})

	t.Run("ProcessPendingNotifications_Empty", func(t *testing.T) {
		err := env.Server.processor.ProcessPendingNotifications()
		if err != nil {
			t.Errorf("Processing empty queue should not error: %v", err)
		}
	})

	t.Run("ProcessPendingNotifications_WithData", func(t *testing.T) {
		// Create test contact
		contact, err := createTestContact(env.DB, env.TestProfile.ID)
		if err != nil {
			t.Fatalf("Failed to create test contact: %v", err)
		}

		// Insert test notification directly into database
		notificationID := uuid.New()
		query := `
			INSERT INTO notifications (
				id, profile_id, contact_id, subject, content, variables,
				channels_requested, priority, scheduled_at, status, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), 'pending', NOW(), NOW())
		`

		subject := "Test Notification"
		content := map[string]interface{}{
			"text": "Hello {{name}}",
			"html": "<p>Hello {{name}}</p>",
		}
		variables := map[string]interface{}{
			"name": "Test User",
		}
		channels := []string{"email"}

		_, err = env.DB.Exec(query, notificationID, env.TestProfile.ID, contact.ID,
			subject, content, variables, channels, "normal")
		if err != nil {
			t.Fatalf("Failed to insert test notification: %v", err)
		}

		// Process the notification
		err = env.Server.processor.ProcessPendingNotifications()
		if err != nil {
			t.Errorf("Failed to process notifications: %v", err)
		}

		// Wait for processing to complete
		time.Sleep(500 * time.Millisecond)

		// Check notification status was updated
		var status string
		err = env.DB.QueryRow("SELECT status FROM notifications WHERE id = $1", notificationID).Scan(&status)
		if err != nil {
			t.Fatalf("Failed to check notification status: %v", err)
		}

		// Status should be processing or delivered
		validStatuses := map[string]bool{"processing": true, "delivered": true, "failed": true}
		if !validStatuses[status] {
			t.Errorf("Expected status to be processing/delivered/failed, got: %s", status)
		}
	})
}

// TestRenderTemplate tests the template rendering functionality
func TestRenderTemplate(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	processor := env.Server.processor

	t.Run("RenderTemplate_SimpleVariable", func(t *testing.T) {
		template := "Hello {{name}}!"
		variables := map[string]interface{}{
			"name": "John",
		}

		result := processor.renderTemplate(template, variables)
		expected := "Hello John!"

		if result != expected {
			t.Errorf("Expected '%s', got '%s'", expected, result)
		}
	})

	t.Run("RenderTemplate_MultipleVariables", func(t *testing.T) {
		template := "Hello {{name}}, your order {{order_id}} is ready!"
		variables := map[string]interface{}{
			"name":     "Jane",
			"order_id": "12345",
		}

		result := processor.renderTemplate(template, variables)
		expected := "Hello Jane, your order 12345 is ready!"

		if result != expected {
			t.Errorf("Expected '%s', got '%s'", expected, result)
		}
	})

	t.Run("RenderTemplate_MissingVariable", func(t *testing.T) {
		template := "Hello {{name}}, {{missing}} variable!"
		variables := map[string]interface{}{
			"name": "Bob",
		}

		result := processor.renderTemplate(template, variables)

		// Missing variables should remain as placeholders
		if result == template {
			t.Log("Missing variables left as placeholders (expected behavior)")
		}
	})

	t.Run("RenderTemplate_EmptyVariables", func(t *testing.T) {
		template := "Hello world!"
		variables := map[string]interface{}{}

		result := processor.renderTemplate(template, variables)

		if result != template {
			t.Errorf("Expected '%s', got '%s'", template, result)
		}
	})
}

// TestSendEmail tests email sending functionality
func TestSendEmail(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	processor := env.Server.processor

	t.Run("SendEmail_Simulated", func(t *testing.T) {
		// Create test contact
		contact, err := createTestContact(env.DB, env.TestProfile.ID)
		if err != nil {
			t.Fatalf("Failed to create test contact: %v", err)
		}

		job := NotificationJob{
			NotificationID: uuid.New(),
			ProfileID:      env.TestProfile.ID,
			ContactID:      contact.ID,
			Channel:        "email",
			Subject:        "Test Email",
			Content: map[string]interface{}{
				"text": "This is a test email",
				"html": "<p>This is a test email</p>",
			},
			Variables: map[string]interface{}{
				"name": "Test User",
			},
			Contact:  *contact,
			Priority: "normal",
		}

		result := processor.sendEmail(job)

		// In test environment, email is simulated
		if !result.Success {
			t.Errorf("Simulated email should succeed, got error: %s", result.Error)
		}

		if result.Channel != "email" {
			t.Errorf("Expected channel 'email', got '%s'", result.Channel)
		}

		if result.Metadata == nil {
			t.Error("Expected metadata in result")
		}
	})
}

// TestSendSMS tests SMS sending functionality
func TestSendSMS(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	processor := env.Server.processor

	t.Run("SendSMS_Simulated", func(t *testing.T) {
		contact, err := createTestContact(env.DB, env.TestProfile.ID)
		if err != nil {
			t.Fatalf("Failed to create test contact: %v", err)
		}

		job := NotificationJob{
			NotificationID: uuid.New(),
			ProfileID:      env.TestProfile.ID,
			ContactID:      contact.ID,
			Channel:        "sms",
			Subject:        "Test SMS",
			Content: map[string]interface{}{
				"text": "This is a test SMS",
			},
			Variables: map[string]interface{}{},
			Contact:   *contact,
			Priority:  "normal",
		}

		result := processor.sendSMS(job)

		// In test environment, SMS is simulated
		if !result.Success {
			t.Errorf("Simulated SMS should succeed, got error: %s", result.Error)
		}

		if result.Channel != "sms" {
			t.Errorf("Expected channel 'sms', got '%s'", result.Channel)
		}
	})
}

// TestSendPushNotification tests push notification sending
func TestSendPushNotification(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	processor := env.Server.processor

	t.Run("SendPush_Simulated", func(t *testing.T) {
		contact, err := createTestContact(env.DB, env.TestProfile.ID)
		if err != nil {
			t.Fatalf("Failed to create test contact: %v", err)
		}

		job := NotificationJob{
			NotificationID: uuid.New(),
			ProfileID:      env.TestProfile.ID,
			ContactID:      contact.ID,
			Channel:        "push",
			Subject:        "Test Push",
			Content: map[string]interface{}{
				"text": "This is a test push notification",
			},
			Variables: map[string]interface{}{},
			Contact:   *contact,
			Priority:  "high",
		}

		result := processor.sendPushNotification(job)

		// In test environment, push is simulated
		if !result.Success {
			t.Errorf("Simulated push should succeed, got error: %s", result.Error)
		}

		if result.Channel != "push" {
			t.Errorf("Expected channel 'push', got '%s'", result.Channel)
		}
	})
}

// TestSendWebhook tests webhook sending
func TestSendWebhook(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	processor := env.Server.processor

	t.Run("SendWebhook_NoURL", func(t *testing.T) {
		contact, err := createTestContact(env.DB, env.TestProfile.ID)
		if err != nil {
			t.Fatalf("Failed to create test contact: %v", err)
		}

		job := NotificationJob{
			NotificationID: uuid.New(),
			ProfileID:      env.TestProfile.ID,
			ContactID:      contact.ID,
			Channel:        "webhook",
			Subject:        "Test Webhook",
			Content: map[string]interface{}{
				"text": "Test webhook content",
			},
			Variables: map[string]interface{}{},
			Contact:   *contact,
			Priority:  "normal",
		}

		result := processor.sendWebhook(job)

		// Should fail without webhook URL
		if result.Success {
			t.Error("Expected webhook to fail without URL")
		}

		if result.Error != "No webhook URL configured" {
			t.Errorf("Expected 'No webhook URL configured', got: %s", result.Error)
		}
	})
}

// TestIsUnsubscribed tests the unsubscribe checking functionality
func TestIsUnsubscribed(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	processor := env.Server.processor

	t.Run("IsUnsubscribed_NotUnsubscribed", func(t *testing.T) {
		contact, err := createTestContact(env.DB, env.TestProfile.ID)
		if err != nil {
			t.Fatalf("Failed to create test contact: %v", err)
		}

		isUnsub := processor.isUnsubscribed(contact.ID, "email")
		if isUnsub {
			t.Error("Contact should not be unsubscribed by default")
		}
	})

	t.Run("IsUnsubscribed_Unsubscribed", func(t *testing.T) {
		contact, err := createTestContact(env.DB, env.TestProfile.ID)
		if err != nil {
			t.Fatalf("Failed to create test contact: %v", err)
		}

		// Add unsubscribe record
		query := `
			INSERT INTO unsubscribes (id, contact_id, profile_id, channel, reason, active, created_at)
			VALUES ($1, $2, $3, $4, $5, true, NOW())
		`
		_, err = env.DB.Exec(query, uuid.New(), contact.ID, env.TestProfile.ID, "email", "user request")
		if err != nil {
			t.Fatalf("Failed to create unsubscribe record: %v", err)
		}

		isUnsub := processor.isUnsubscribed(contact.ID, "email")
		if !isUnsub {
			t.Error("Contact should be unsubscribed for email")
		}

		// Should still be subscribed to SMS
		isUnsubSMS := processor.isUnsubscribed(contact.ID, "sms")
		if isUnsubSMS {
			t.Error("Contact should not be unsubscribed for SMS")
		}
	})
}

// TestUpdateNotificationStatus tests notification status updates
func TestUpdateNotificationStatus(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	processor := env.Server.processor

	t.Run("UpdateNotificationStatus", func(t *testing.T) {
		contact, err := createTestContact(env.DB, env.TestProfile.ID)
		if err != nil {
			t.Fatalf("Failed to create test contact: %v", err)
		}

		// Create test notification
		notificationID := uuid.New()
		query := `
			INSERT INTO notifications (
				id, profile_id, contact_id, subject, content, variables,
				channels_requested, priority, scheduled_at, status, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), 'pending', NOW(), NOW())
		`
		_, err = env.DB.Exec(query, notificationID, env.TestProfile.ID, contact.ID,
			"Test", map[string]interface{}{}, map[string]interface{}{},
			[]string{"email"}, "normal")
		if err != nil {
			t.Fatalf("Failed to create notification: %v", err)
		}

		// Update status
		processor.updateNotificationStatus(notificationID, "processing")

		// Verify status was updated
		var status string
		err = env.DB.QueryRow("SELECT status FROM notifications WHERE id = $1", notificationID).Scan(&status)
		if err != nil {
			t.Fatalf("Failed to query notification: %v", err)
		}

		if status != "processing" {
			t.Errorf("Expected status 'processing', got '%s'", status)
		}
	})
}

// TestRecordDeliveryResult tests delivery result recording
func TestRecordDeliveryResult(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	processor := env.Server.processor

	t.Run("RecordDeliveryResult_Success", func(t *testing.T) {
		notificationID := uuid.New()

		result := DeliveryResult{
			NotificationID: notificationID,
			Channel:        "email",
			Success:        true,
			Error:          "",
			DeliveredAt:    time.Now(),
			Metadata: map[string]interface{}{
				"to":      "test@example.com",
				"subject": "Test Email",
			},
		}

		processor.recordDeliveryResult(result)

		// Wait for async operation
		time.Sleep(200 * time.Millisecond)

		// Verify record was created in database
		var count int
		err := env.DB.QueryRow("SELECT COUNT(*) FROM delivery_logs WHERE notification_id = $1", notificationID).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to query delivery logs: %v", err)
		}

		if count != 1 {
			t.Errorf("Expected 1 delivery log, got %d", count)
		}

		// Verify Redis cache
		ctx := context.Background()
		key := "delivery:" + notificationID.String() + ":email"
		exists, err := env.Redis.Exists(ctx, key).Result()
		if err != nil {
			t.Errorf("Failed to check Redis: %v", err)
		}

		if exists != 1 {
			t.Error("Expected delivery result in Redis cache")
		}
	})

	t.Run("RecordDeliveryResult_Failure", func(t *testing.T) {
		notificationID := uuid.New()

		result := DeliveryResult{
			NotificationID: notificationID,
			Channel:        "sms",
			Success:        false,
			Error:          "SMS provider unavailable",
			DeliveredAt:    time.Now(),
			Metadata:       map[string]interface{}{},
		}

		processor.recordDeliveryResult(result)

		// Wait for async operation
		time.Sleep(200 * time.Millisecond)

		// Verify record was created
		var errorMsg string
		err := env.DB.QueryRow("SELECT error FROM delivery_logs WHERE notification_id = $1 AND channel = 'sms'",
			notificationID).Scan(&errorMsg)
		if err == sql.ErrNoRows {
			t.Error("Expected delivery log record")
		} else if err != nil {
			t.Fatalf("Failed to query delivery log: %v", err)
		}

		if errorMsg != "SMS provider unavailable" {
			t.Errorf("Expected error message 'SMS provider unavailable', got '%s'", errorMsg)
		}
	})
}

// TestChannelMarking tests marking channels as delivered or failed
func TestChannelMarking(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	processor := env.Server.processor

	t.Run("MarkChannelDelivered", func(t *testing.T) {
		contact, err := createTestContact(env.DB, env.TestProfile.ID)
		if err != nil {
			t.Fatalf("Failed to create test contact: %v", err)
		}

		// Create notification with multiple channels
		notificationID := uuid.New()
		query := `
			INSERT INTO notifications (
				id, profile_id, contact_id, subject, content, variables,
				channels_requested, channels_attempted, priority, scheduled_at, status, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), 'processing', NOW(), NOW())
		`
		_, err = env.DB.Exec(query, notificationID, env.TestProfile.ID, contact.ID,
			"Test", map[string]interface{}{}, map[string]interface{}{},
			[]string{"email", "sms"}, []string{}, "normal")
		if err != nil {
			t.Fatalf("Failed to create notification: %v", err)
		}

		// Mark email as delivered
		processor.markChannelDelivered(notificationID, "email")

		// Wait for update
		time.Sleep(100 * time.Millisecond)

		// Check channels_attempted was updated
		var channelsAttempted []string
		err = env.DB.QueryRow("SELECT channels_attempted FROM notifications WHERE id = $1",
			notificationID).Scan(&channelsAttempted)
		if err != nil {
			t.Fatalf("Failed to query notification: %v", err)
		}

		if len(channelsAttempted) != 1 || channelsAttempted[0] != "email" {
			t.Errorf("Expected channels_attempted to contain 'email', got %v", channelsAttempted)
		}
	})

	t.Run("MarkChannelFailed", func(t *testing.T) {
		contact, err := createTestContact(env.DB, env.TestProfile.ID)
		if err != nil {
			t.Fatalf("Failed to create test contact: %v", err)
		}

		notificationID := uuid.New()
		query := `
			INSERT INTO notifications (
				id, profile_id, contact_id, subject, content, variables,
				channels_requested, channels_attempted, priority, scheduled_at, status, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), 'processing', NOW(), NOW())
		`
		_, err = env.DB.Exec(query, notificationID, env.TestProfile.ID, contact.ID,
			"Test", map[string]interface{}{}, map[string]interface{}{},
			[]string{"sms"}, []string{}, "normal")
		if err != nil {
			t.Fatalf("Failed to create notification: %v", err)
		}

		// Mark SMS as failed
		processor.markChannelFailed(notificationID, "sms", "Provider error")

		// Wait for update
		time.Sleep(100 * time.Millisecond)

		// Check notification status
		var status string
		var channelsAttempted []string
		err = env.DB.QueryRow("SELECT status, channels_attempted FROM notifications WHERE id = $1",
			notificationID).Scan(&status, &channelsAttempted)
		if err != nil {
			t.Fatalf("Failed to query notification: %v", err)
		}

		// Since only one channel was requested and it failed, status should be 'failed'
		if status != "failed" {
			t.Errorf("Expected status 'failed', got '%s'", status)
		}

		if len(channelsAttempted) != 1 || channelsAttempted[0] != "sms" {
			t.Errorf("Expected channels_attempted to contain 'sms', got %v", channelsAttempted)
		}
	})
}
