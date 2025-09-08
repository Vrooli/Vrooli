package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type CalendarProcessor struct {
	db              *sql.DB
	config          *Config
	reminderManager *ReminderManager
}

type ReminderManager struct {
	db                     *sql.DB
	notificationServiceURL string
	notificationProfileID  string
	notificationAPIKey     string
}

type Reminder struct {
	ID            string    `json:"id"`
	EventID       string    `json:"event_id"`
	UserID        string    `json:"user_id"`
	MinutesBefore int       `json:"minutes_before"`
	Type          string    `json:"type"`
	Status        string    `json:"status"`
	SendAt        time.Time `json:"send_at"`
	SentAt        *time.Time `json:"sent_at,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

type NotificationRequest struct {
	ProfileID   string                 `json:"profile_id"`
	Recipients  []string               `json:"recipients"`
	Template    string                 `json:"template"`
	Data        map[string]interface{} `json:"data"`
	ScheduledAt *time.Time             `json:"scheduled_at,omitempty"`
}

func NewCalendarProcessor(db *sql.DB, config *Config) *CalendarProcessor {
	return &CalendarProcessor{
		db:     db,
		config: config,
		reminderManager: &ReminderManager{
			db:                     db,
			notificationServiceURL: config.NotificationServiceURL,
			notificationProfileID:  config.NotificationProfileID,
			notificationAPIKey:     config.NotificationAPIKey,
		},
	}
}

// ProcessReminders processes all pending reminders (replaces n8n calendar-reminder workflow)
func (cp *CalendarProcessor) ProcessReminders(ctx context.Context) error {
	// Get all reminders that should be sent now
	query := `
		SELECT r.id, r.event_id, r.user_id, r.minutes_before, r.type, r.send_at,
		       e.title, e.description, e.start_time, e.location, u.email, u.display_name
		FROM reminders r
		JOIN events e ON r.event_id = e.id
		JOIN users u ON r.user_id = u.id
		WHERE r.status = 'pending' 
		  AND r.send_at <= NOW()
		  AND r.sent_at IS NULL
		ORDER BY r.send_at
		LIMIT 100`

	rows, err := cp.db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to fetch pending reminders: %w", err)
	}
	defer rows.Close()

	var reminders []struct {
		ID            string
		EventID       string
		UserID        string
		MinutesBefore int
		Type          string
		SendAt        time.Time
		EventTitle    string
		EventDesc     string
		EventStart    time.Time
		EventLocation string
		UserEmail     string
		UserName      string
	}

	for rows.Next() {
		var r struct {
			ID            string
			EventID       string
			UserID        string
			MinutesBefore int
			Type          string
			SendAt        time.Time
			EventTitle    string
			EventDesc     string
			EventStart    time.Time
			EventLocation string
			UserEmail     string
			UserName      string
		}
		
		if err := rows.Scan(&r.ID, &r.EventID, &r.UserID, &r.MinutesBefore, &r.Type, 
			&r.SendAt, &r.EventTitle, &r.EventDesc, &r.EventStart, &r.EventLocation,
			&r.UserEmail, &r.UserName); err != nil {
			return fmt.Errorf("failed to scan reminder: %w", err)
		}
		reminders = append(reminders, r)
	}

	// Process each reminder
	for _, reminder := range reminders {
		if err := cp.reminderManager.SendReminder(ctx, reminder.ID, reminder.UserEmail, 
			reminder.EventTitle, reminder.EventDesc, reminder.EventStart, 
			reminder.EventLocation, reminder.MinutesBefore); err != nil {
			// Log error but continue processing other reminders
			fmt.Printf("Failed to send reminder %s: %v\n", reminder.ID, err)
			continue
		}

		// Mark reminder as sent
		updateQuery := `UPDATE reminders SET status = 'sent', sent_at = NOW() WHERE id = $1`
		if _, err := cp.db.ExecContext(ctx, updateQuery, reminder.ID); err != nil {
			fmt.Printf("Failed to update reminder status %s: %v\n", reminder.ID, err)
		}
	}

	return nil
}

// CreateRemindersForEvent creates reminders for a new event
func (cp *CalendarProcessor) CreateRemindersForEvent(ctx context.Context, eventID, userID string, 
	startTime time.Time, reminders []ReminderRequest) error {
	
	for _, reminder := range reminders {
		reminderID := uuid.New().String()
		sendAt := startTime.Add(-time.Duration(reminder.MinutesBefore) * time.Minute)
		
		query := `
			INSERT INTO reminders (id, event_id, user_id, minutes_before, type, status, send_at, created_at)
			VALUES ($1, $2, $3, $4, $5, 'pending', $6, NOW())`
		
		if _, err := cp.db.ExecContext(ctx, query, reminderID, eventID, userID, 
			reminder.MinutesBefore, reminder.NotificationType, sendAt); err != nil {
			return fmt.Errorf("failed to create reminder: %w", err)
		}
	}
	
	return nil
}

// SendReminder sends a reminder notification
func (rm *ReminderManager) SendReminder(ctx context.Context, reminderID, userEmail, 
	eventTitle, eventDesc string, eventStart time.Time, location string, minutesBefore int) error {
	
	notificationReq := NotificationRequest{
		ProfileID:  rm.notificationProfileID,
		Recipients: []string{userEmail},
		Template:   "event-reminder",
		Data: map[string]interface{}{
			"event_title":       eventTitle,
			"event_description": eventDesc,
			"event_start":       eventStart.Format(time.RFC3339),
			"event_location":    location,
			"minutes_before":    minutesBefore,
			"reminder_id":       reminderID,
		},
	}

	jsonData, err := json.Marshal(notificationReq)
	if err != nil {
		return fmt.Errorf("failed to marshal notification request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", 
		rm.notificationServiceURL+"/api/v1/notifications/send", 
		bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create notification request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if rm.notificationAPIKey != "" {
		req.Header.Set("X-API-Key", rm.notificationAPIKey)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("notification service returned status %d", resp.StatusCode)
	}

	return nil
}

// StartReminderProcessor starts a background goroutine to process reminders
func (cp *CalendarProcessor) StartReminderProcessor(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(1 * time.Minute) // Check every minute
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := cp.ProcessReminders(ctx); err != nil {
					fmt.Printf("Error processing reminders: %v\n", err)
				}
			}
		}
	}()
}

// ProcessEventAutomation handles event-triggered automation (replaces n8n event-automation workflow)
func (cp *CalendarProcessor) ProcessEventAutomation(ctx context.Context, event Event) error {
	if event.AutomationConfig == nil {
		return nil
	}

	// Check if automation is enabled
	if enabled, ok := event.AutomationConfig["enabled"].(bool); !ok || !enabled {
		return nil
	}

	// Process different automation types
	if automationType, ok := event.AutomationConfig["type"].(string); ok {
		switch automationType {
		case "webhook":
			return cp.processWebhookAutomation(ctx, event)
		case "notification":
			return cp.processNotificationAutomation(ctx, event)
		case "recurring":
			return cp.processRecurringAutomation(ctx, event)
		default:
			fmt.Printf("Unknown automation type: %s\n", automationType)
		}
	}

	return nil
}

func (cp *CalendarProcessor) processWebhookAutomation(ctx context.Context, event Event) error {
	webhookURL, ok := event.AutomationConfig["webhook_url"].(string)
	if !ok || webhookURL == "" {
		return fmt.Errorf("webhook_url not configured")
	}

	payload := map[string]interface{}{
		"event_id":    event.ID,
		"event_title": event.Title,
		"start_time":  event.StartTime,
		"end_time":    event.EndTime,
		"metadata":    event.Metadata,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

func (cp *CalendarProcessor) processNotificationAutomation(ctx context.Context, event Event) error {
	// Send automated notifications when event status changes
	notificationTemplate, ok := event.AutomationConfig["notification_template"].(string)
	if !ok {
		notificationTemplate = "event-status-change"
	}

	// Get user email
	var userEmail string
	query := `SELECT email FROM users WHERE id = $1`
	if err := cp.db.QueryRowContext(ctx, query, event.UserID).Scan(&userEmail); err != nil {
		return fmt.Errorf("failed to get user email: %w", err)
	}

	notificationReq := NotificationRequest{
		ProfileID:  cp.config.NotificationProfileID,
		Recipients: []string{userEmail},
		Template:   notificationTemplate,
		Data: map[string]interface{}{
			"event_id":     event.ID,
			"event_title":  event.Title,
			"event_status": event.Status,
			"metadata":     event.Metadata,
		},
	}

	jsonData, err := json.Marshal(notificationReq)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		cp.config.NotificationServiceURL+"/api/v1/notifications/send",
		bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create notification request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if cp.config.NotificationAPIKey != "" {
		req.Header.Set("X-API-Key", cp.config.NotificationAPIKey)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}
	defer resp.Body.Close()

	return nil
}

func (cp *CalendarProcessor) processRecurringAutomation(ctx context.Context, event Event) error {
	// Handle recurring event creation
	recurrenceConfig, ok := event.AutomationConfig["recurrence"].(map[string]interface{})
	if !ok {
		return nil
	}

	pattern, _ := recurrenceConfig["pattern"].(string)
	interval, _ := recurrenceConfig["interval"].(float64)
	
	// Calculate next occurrence
	var nextStart time.Time
	switch pattern {
	case "daily":
		nextStart = event.StartTime.AddDate(0, 0, int(interval))
	case "weekly":
		nextStart = event.StartTime.AddDate(0, 0, 7*int(interval))
	case "monthly":
		nextStart = event.StartTime.AddDate(0, int(interval), 0)
	default:
		return fmt.Errorf("unknown recurrence pattern: %s", pattern)
	}

	// Check if we should create the next occurrence
	if maxOccurrences, ok := recurrenceConfig["max_occurrences"].(float64); ok {
		var count int
		countQuery := `SELECT COUNT(*) FROM events WHERE metadata->>'recurrence_group_id' = $1`
		if err := cp.db.QueryRowContext(ctx, countQuery, event.ID).Scan(&count); err != nil {
			return fmt.Errorf("failed to count occurrences: %w", err)
		}
		if count >= int(maxOccurrences) {
			return nil // Max occurrences reached
		}
	}

	if endDate, ok := recurrenceConfig["end_date"].(string); ok {
		end, err := time.Parse(time.RFC3339, endDate)
		if err == nil && nextStart.After(end) {
			return nil // Past end date
		}
	}

	// Create next occurrence
	duration := event.EndTime.Sub(event.StartTime)
	nextEnd := nextStart.Add(duration)

	newEventID := uuid.New().String()
	newMetadata := event.Metadata
	if newMetadata == nil {
		newMetadata = make(map[string]interface{})
	}
	newMetadata["recurrence_group_id"] = event.ID
	
	metadataJSON, _ := json.Marshal(newMetadata)
	automationJSON, _ := json.Marshal(event.AutomationConfig)

	insertQuery := `
		INSERT INTO events (id, user_id, title, description, start_time, end_time, 
			timezone, location, event_type, status, metadata, automation_config, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW(), NOW())`

	if _, err := cp.db.ExecContext(ctx, insertQuery, newEventID, event.UserID, event.Title,
		event.Description, nextStart, nextEnd, event.Timezone, event.Location,
		event.EventType, "scheduled", metadataJSON, automationJSON); err != nil {
		return fmt.Errorf("failed to create recurring event: %w", err)
	}

	return nil
}