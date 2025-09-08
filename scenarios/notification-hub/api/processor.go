package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

// NotificationProcessor handles sending notifications through various channels
type NotificationProcessor struct {
	db          *sql.DB
	redisClient *redis.Client
	ctx         context.Context
	workers     int
	wg          sync.WaitGroup
	jobs        chan NotificationJob
}

// NotificationJob represents a notification to be processed
type NotificationJob struct {
	NotificationID uuid.UUID
	ProfileID      uuid.UUID
	ContactID      uuid.UUID
	Channel        string
	Subject        string
	Content        map[string]interface{}
	Variables      map[string]interface{}
	Contact        Contact
	Priority       string
}

// DeliveryResult represents the result of a delivery attempt
type DeliveryResult struct {
	NotificationID uuid.UUID
	Channel        string
	Success        bool
	Error          string
	DeliveredAt    time.Time
	Metadata       map[string]interface{}
}

// NewNotificationProcessor creates a new processor
func NewNotificationProcessor(db *sql.DB, redisClient *redis.Client) *NotificationProcessor {
	np := &NotificationProcessor{
		db:          db,
		redisClient: redisClient,
		ctx:         context.Background(),
		workers:     5,
		jobs:        make(chan NotificationJob, 100),
	}
	
	// Start worker pool
	for i := 0; i < np.workers; i++ {
		np.wg.Add(1)
		go np.worker()
	}
	
	return np
}

// worker processes notification jobs
func (np *NotificationProcessor) worker() {
	defer np.wg.Done()
	
	for job := range np.jobs {
		np.processNotification(job)
	}
}

// ProcessPendingNotifications checks for and processes pending notifications
func (np *NotificationProcessor) ProcessPendingNotifications() error {
	query := `
		SELECT n.id, n.profile_id, n.contact_id, n.subject, n.content, 
		       n.variables, n.channels_requested, n.priority,
		       c.identifier, c.first_name, c.last_name, c.preferences
		FROM notifications n
		JOIN contacts c ON n.contact_id = c.id
		WHERE n.status = 'pending' 
		  AND n.scheduled_at <= NOW()
		ORDER BY 
		  CASE n.priority
		    WHEN 'urgent' THEN 1
		    WHEN 'high' THEN 2
		    WHEN 'normal' THEN 3
		    WHEN 'low' THEN 4
		  END,
		  n.scheduled_at
		LIMIT 100
	`
	
	rows, err := np.db.Query(query)
	if err != nil {
		return fmt.Errorf("failed to query pending notifications: %w", err)
	}
	defer rows.Close()
	
	for rows.Next() {
		var job NotificationJob
		var channelsRequested []string
		var contactPreferences map[string]interface{}
		var firstName, lastName *string
		
		err := rows.Scan(
			&job.NotificationID, &job.ProfileID, &job.ContactID,
			&job.Subject, &job.Content, &job.Variables,
			&channelsRequested, &job.Priority,
			&job.Contact.Identifier, &firstName, &lastName,
			&contactPreferences,
		)
		if err != nil {
			log.Printf("Failed to scan notification: %v", err)
			continue
		}
		
		if firstName != nil {
			job.Contact.FirstName = firstName
		}
		if lastName != nil {
			job.Contact.LastName = lastName
		}
		job.Contact.Preferences = contactPreferences
		
		// Update status to processing
		np.updateNotificationStatus(job.NotificationID, "processing")
		
		// Queue job for each channel
		for _, channel := range channelsRequested {
			channelJob := job
			channelJob.Channel = channel
			
			select {
			case np.jobs <- channelJob:
			case <-time.After(5 * time.Second):
				log.Printf("Failed to queue job for notification %s", job.NotificationID)
			}
		}
	}
	
	return nil
}

// processNotification handles sending through a specific channel
func (np *NotificationProcessor) processNotification(job NotificationJob) {
	var result DeliveryResult
	result.NotificationID = job.NotificationID
	result.Channel = job.Channel
	result.DeliveredAt = time.Now()
	
	// Check if contact has unsubscribed from this channel
	if np.isUnsubscribed(job.ContactID, job.Channel) {
		result.Success = false
		result.Error = "Contact unsubscribed from channel"
		np.recordDeliveryResult(result)
		return
	}
	
	// Process based on channel
	switch job.Channel {
	case "email":
		result = np.sendEmail(job)
	case "sms":
		result = np.sendSMS(job)
	case "push":
		result = np.sendPushNotification(job)
	case "webhook":
		result = np.sendWebhook(job)
	default:
		result.Success = false
		result.Error = fmt.Sprintf("Unsupported channel: %s", job.Channel)
	}
	
	// Record delivery result
	np.recordDeliveryResult(result)
	
	// Update notification status
	if result.Success {
		np.markChannelDelivered(job.NotificationID, job.Channel)
	} else {
		np.markChannelFailed(job.NotificationID, job.Channel, result.Error)
	}
}

// sendEmail sends an email notification
func (np *NotificationProcessor) sendEmail(job NotificationJob) DeliveryResult {
	result := DeliveryResult{
		NotificationID: job.NotificationID,
		Channel:        "email",
		DeliveredAt:    time.Now(),
	}
	
	// Get SMTP configuration
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASSWORD")
	fromEmail := os.Getenv("SMTP_FROM_EMAIL")
	
	if smtpHost == "" {
		// For demo purposes, simulate email sending
		log.Printf("Simulating email to %s: %s", job.Contact.Identifier, job.Subject)
		result.Success = true
		result.Metadata = map[string]interface{}{
			"simulated": true,
			"to":        job.Contact.Identifier,
			"subject":   job.Subject,
		}
		return result
	}
	
	// Build email content
	htmlContent := ""
	textContent := ""
	if html, ok := job.Content["html"].(string); ok {
		htmlContent = np.renderTemplate(html, job.Variables)
	}
	if text, ok := job.Content["text"].(string); ok {
		textContent = np.renderTemplate(text, job.Variables)
	}
	
	// Create email message
	message := fmt.Sprintf("From: %s\r\n", fromEmail)
	message += fmt.Sprintf("To: %s\r\n", job.Contact.Identifier)
	message += fmt.Sprintf("Subject: %s\r\n", job.Subject)
	message += "MIME-Version: 1.0\r\n"
	message += "Content-Type: multipart/alternative; boundary=\"boundary\"\r\n\r\n"
	
	if textContent != "" {
		message += "--boundary\r\n"
		message += "Content-Type: text/plain; charset=\"UTF-8\"\r\n\r\n"
		message += textContent + "\r\n"
	}
	
	if htmlContent != "" {
		message += "--boundary\r\n"
		message += "Content-Type: text/html; charset=\"UTF-8\"\r\n\r\n"
		message += htmlContent + "\r\n"
	}
	
	message += "--boundary--"
	
	// Send email
	auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)
	err := smtp.SendMail(
		smtpHost+":"+smtpPort,
		auth,
		fromEmail,
		[]string{job.Contact.Identifier},
		[]byte(message),
	)
	
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("Failed to send email: %v", err)
	} else {
		result.Success = true
		result.Metadata = map[string]interface{}{
			"to":      job.Contact.Identifier,
			"subject": job.Subject,
		}
	}
	
	return result
}

// sendSMS sends an SMS notification
func (np *NotificationProcessor) sendSMS(job NotificationJob) DeliveryResult {
	result := DeliveryResult{
		NotificationID: job.NotificationID,
		Channel:        "sms",
		DeliveredAt:    time.Now(),
	}
	
	// Get SMS provider configuration
	smsProvider := os.Getenv("SMS_PROVIDER") // twilio, nexmo, etc.
	
	if smsProvider == "" {
		// Simulate SMS sending for demo
		log.Printf("Simulating SMS to %s: %s", job.Contact.Identifier, job.Content["text"])
		result.Success = true
		result.Metadata = map[string]interface{}{
			"simulated": true,
			"to":        job.Contact.Identifier,
		}
		return result
	}
	
	// Implement actual SMS sending based on provider
	switch smsProvider {
	case "twilio":
		result = np.sendTwilioSMS(job)
	default:
		result.Success = false
		result.Error = fmt.Sprintf("Unsupported SMS provider: %s", smsProvider)
	}
	
	return result
}

// sendPushNotification sends a push notification
func (np *NotificationProcessor) sendPushNotification(job NotificationJob) DeliveryResult {
	result := DeliveryResult{
		NotificationID: job.NotificationID,
		Channel:        "push",
		DeliveredAt:    time.Now(),
	}
	
	// Get push notification service configuration
	pushService := os.Getenv("PUSH_SERVICE") // fcm, apns, etc.
	
	if pushService == "" {
		// Simulate push notification for demo
		log.Printf("Simulating push notification to device: %s", job.Contact.Identifier)
		result.Success = true
		result.Metadata = map[string]interface{}{
			"simulated": true,
			"device":    job.Contact.Identifier,
			"title":     job.Subject,
		}
		return result
	}
	
	// Implement actual push notification based on service
	switch pushService {
	case "fcm":
		result = np.sendFCMNotification(job)
	default:
		result.Success = false
		result.Error = fmt.Sprintf("Unsupported push service: %s", pushService)
	}
	
	return result
}

// sendWebhook sends a webhook notification
func (np *NotificationProcessor) sendWebhook(job NotificationJob) DeliveryResult {
	result := DeliveryResult{
		NotificationID: job.NotificationID,
		Channel:        "webhook",
		DeliveredAt:    time.Now(),
	}
	
	// Get webhook URL from contact preferences or profile settings
	webhookURL := ""
	if prefs, ok := job.Contact.Preferences["webhook_url"].(string); ok {
		webhookURL = prefs
	}
	
	if webhookURL == "" {
		result.Success = false
		result.Error = "No webhook URL configured"
		return result
	}
	
	// Prepare webhook payload
	payload := map[string]interface{}{
		"notification_id": job.NotificationID,
		"contact_id":      job.ContactID,
		"subject":         job.Subject,
		"content":         job.Content,
		"variables":       job.Variables,
		"timestamp":       time.Now().Unix(),
	}
	
	jsonData, _ := json.Marshal(payload)
	
	// Send webhook
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(webhookURL, "application/json", bytes.NewBuffer(jsonData))
	
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("Failed to send webhook: %v", err)
	} else {
		defer resp.Body.Close()
		result.Success = resp.StatusCode >= 200 && resp.StatusCode < 300
		if !result.Success {
			result.Error = fmt.Sprintf("Webhook returned status %d", resp.StatusCode)
		}
		result.Metadata = map[string]interface{}{
			"url":         webhookURL,
			"status_code": resp.StatusCode,
		}
	}
	
	return result
}

// Helper methods

func (np *NotificationProcessor) sendTwilioSMS(job NotificationJob) DeliveryResult {
	result := DeliveryResult{
		NotificationID: job.NotificationID,
		Channel:        "sms",
		DeliveredAt:    time.Now(),
	}
	
	// Twilio implementation would go here
	result.Success = false
	result.Error = "Twilio integration not implemented"
	
	return result
}

func (np *NotificationProcessor) sendFCMNotification(job NotificationJob) DeliveryResult {
	result := DeliveryResult{
		NotificationID: job.NotificationID,
		Channel:        "push",
		DeliveredAt:    time.Now(),
	}
	
	// FCM implementation would go here
	result.Success = false
	result.Error = "FCM integration not implemented"
	
	return result
}

func (np *NotificationProcessor) renderTemplate(template string, variables map[string]interface{}) string {
	// Simple variable replacement
	result := template
	for key, value := range variables {
		placeholder := fmt.Sprintf("{{%s}}", key)
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", value))
	}
	return result
}

func (np *NotificationProcessor) isUnsubscribed(contactID uuid.UUID, channel string) bool {
	var unsubscribed bool
	query := `
		SELECT EXISTS(
			SELECT 1 FROM unsubscribes 
			WHERE contact_id = $1 AND channel = $2 AND active = true
		)
	`
	np.db.QueryRow(query, contactID, channel).Scan(&unsubscribed)
	return unsubscribed
}

func (np *NotificationProcessor) updateNotificationStatus(notificationID uuid.UUID, status string) {
	query := `UPDATE notifications SET status = $2, updated_at = NOW() WHERE id = $1`
	np.db.Exec(query, notificationID, status)
}

func (np *NotificationProcessor) markChannelDelivered(notificationID uuid.UUID, channel string) {
	// Update channels_attempted array
	query := `
		UPDATE notifications 
		SET channels_attempted = array_append(channels_attempted, $2),
		    status = CASE 
		        WHEN array_length(channels_attempted, 1) >= array_length(channels_requested, 1) - 1
		        THEN 'delivered'
		        ELSE status
		    END,
		    updated_at = NOW()
		WHERE id = $1
	`
	np.db.Exec(query, notificationID, channel)
}

func (np *NotificationProcessor) markChannelFailed(notificationID uuid.UUID, channel string, error string) {
	// Log failure and update status
	log.Printf("Delivery failed for notification %s on channel %s: %s", notificationID, channel, error)
	
	query := `
		UPDATE notifications 
		SET channels_attempted = array_append(channels_attempted, $2),
		    status = CASE 
		        WHEN array_length(channels_attempted, 1) >= array_length(channels_requested, 1) - 1
		        THEN 'failed'
		        ELSE status
		    END,
		    updated_at = NOW()
		WHERE id = $1
	`
	np.db.Exec(query, notificationID, channel)
}

func (np *NotificationProcessor) recordDeliveryResult(result DeliveryResult) {
	// Store delivery attempt in database
	query := `
		INSERT INTO delivery_logs (
			id, notification_id, channel, success, error, 
			delivered_at, metadata, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
	`
	
	metadataJSON, _ := json.Marshal(result.Metadata)
	np.db.Exec(query, uuid.New(), result.NotificationID, result.Channel,
		result.Success, result.Error, result.DeliveredAt, metadataJSON)
	
	// Also cache in Redis for quick access
	if np.redisClient != nil {
		key := fmt.Sprintf("delivery:%s:%s", result.NotificationID, result.Channel)
		data, _ := json.Marshal(result)
		np.redisClient.Set(np.ctx, key, data, 24*time.Hour)
	}
}

// Close shuts down the processor
func (np *NotificationProcessor) Close() {
	close(np.jobs)
	np.wg.Wait()
}