package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

type NLPProcessor struct {
	ollamaURL string
	db        *sql.DB
	config    *Config
}

type OllamaRequest struct {
	Model    string `json:"model"`
	Messages []OllamaMessage `json:"messages"`
	Stream   bool   `json:"stream"`
	Options  map[string]interface{} `json:"options,omitempty"`
}

type OllamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OllamaResponse struct {
	Message OllamaMessage `json:"message"`
	Done    bool          `json:"done"`
}

type ParsedScheduleIntent struct {
	Action      string                 `json:"action"`       // create, update, delete, query
	EventTitle  string                 `json:"event_title"`
	Description string                 `json:"description"`
	StartTime   *time.Time             `json:"start_time"`
	EndTime     *time.Time             `json:"end_time"`
	Location    string                 `json:"location"`
	Attendees   []string               `json:"attendees"`
	Recurrence  map[string]interface{} `json:"recurrence"`
	Reminders   []ReminderRequest      `json:"reminders"`
	Confidence  float64                `json:"confidence"`
}

func NewNLPProcessor(ollamaURL string, db *sql.DB, config *Config) *NLPProcessor {
	if ollamaURL == "" {
		log.Fatal("OLLAMA_URL environment variable is required")
	}
	return &NLPProcessor{
		ollamaURL: ollamaURL,
		db:        db,
		config:    config,
	}
}

// ProcessSchedulingRequest processes natural language scheduling requests (replaces n8n ollama.json workflow)
func (nlp *NLPProcessor) ProcessSchedulingRequest(ctx context.Context, userID, message string) (*ChatResponse, error) {
	// First, try to parse the intent using Ollama if available
	intent, err := nlp.parseSchedulingIntent(ctx, message)
	if err != nil {
		// Fallback to rule-based parsing if Ollama is not available
		intent = nlp.fallbackParsing(message)
	}

	// Execute the parsed intent
	switch intent.Action {
	case "create":
		return nlp.handleCreateEvent(ctx, userID, intent)
	case "update":
		return nlp.handleUpdateEvent(ctx, userID, intent)
	case "delete":
		return nlp.handleDeleteEvent(ctx, userID, intent)
	case "query":
		return nlp.handleQueryEvents(ctx, userID, intent)
	default:
		return nlp.handleAmbiguous(ctx, intent, message)
	}
}

func (nlp *NLPProcessor) parseSchedulingIntent(ctx context.Context, message string) (*ParsedScheduleIntent, error) {
	// Create system prompt for Ollama
	systemPrompt := `You are a calendar scheduling assistant. Parse the user's message and extract scheduling information.
Return a JSON object with these fields:
- action: one of "create", "update", "delete", "query"
- event_title: the title of the event
- description: event description if provided
- start_time: ISO 8601 format datetime
- end_time: ISO 8601 format datetime
- location: event location if mentioned
- attendees: array of email addresses or names
- recurrence: object with pattern (daily/weekly/monthly), interval, and end_date
- reminders: array of {minutes_before: number, type: "email"/"push"}
- confidence: 0-1 score of parsing confidence

Example: "Schedule a meeting with John tomorrow at 3pm"
Response: {"action":"create","event_title":"Meeting with John","start_time":"2024-01-15T15:00:00Z","end_time":"2024-01-15T16:00:00Z","attendees":["John"],"confidence":0.9}`

	// Prepare Ollama request
	ollamaReq := OllamaRequest{
		Model: "llama3.2", // Use a lightweight model for fast processing
		Messages: []OllamaMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: fmt.Sprintf("Current time: %s\nUser message: %s", time.Now().Format(time.RFC3339), message)},
		},
		Stream: false,
		Options: map[string]interface{}{
			"temperature": 0.3, // Lower temperature for more consistent parsing
			"top_p":       0.9,
		},
	}

	jsonData, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Ollama request: %w", err)
	}

	// Call Ollama API
	req, err := http.NewRequestWithContext(ctx, "POST", nlp.ollamaURL+"/api/chat", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create Ollama request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Ollama returned status %d", resp.StatusCode)
	}

	// Parse Ollama response
	var ollamaResp OllamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to decode Ollama response: %w", err)
	}

	// Parse the JSON from Ollama's message content
	var intent ParsedScheduleIntent
	// Extract JSON from the response (Ollama might include explanation text)
	jsonStart := strings.Index(ollamaResp.Message.Content, "{")
	jsonEnd := strings.LastIndex(ollamaResp.Message.Content, "}")
	if jsonStart >= 0 && jsonEnd >= jsonStart {
		jsonStr := ollamaResp.Message.Content[jsonStart : jsonEnd+1]
		if err := json.Unmarshal([]byte(jsonStr), &intent); err != nil {
			return nil, fmt.Errorf("failed to parse Ollama JSON response: %w", err)
		}
	} else {
		return nil, fmt.Errorf("no JSON found in Ollama response")
	}

	return &intent, nil
}

func (nlp *NLPProcessor) fallbackParsing(message string) *ParsedScheduleIntent {
	intent := &ParsedScheduleIntent{
		Confidence: 0.5,
	}

	msgLower := strings.ToLower(message)

	// Detect action
	if strings.Contains(msgLower, "schedule") || strings.Contains(msgLower, "create") || 
	   strings.Contains(msgLower, "add") || strings.Contains(msgLower, "book") {
		intent.Action = "create"
	} else if strings.Contains(msgLower, "update") || strings.Contains(msgLower, "change") || 
	          strings.Contains(msgLower, "modify") || strings.Contains(msgLower, "reschedule") {
		intent.Action = "update"
	} else if strings.Contains(msgLower, "delete") || strings.Contains(msgLower, "cancel") || 
	          strings.Contains(msgLower, "remove") {
		intent.Action = "delete"
	} else if strings.Contains(msgLower, "show") || strings.Contains(msgLower, "list") || 
	          strings.Contains(msgLower, "what") || strings.Contains(msgLower, "when") {
		intent.Action = "query"
	} else {
		intent.Action = "create" // Default to create
	}

	// Extract time references
	now := time.Now()
	if strings.Contains(msgLower, "tomorrow") {
		tomorrow := now.AddDate(0, 0, 1)
		startTime := nlp.extractTimeFromMessage(msgLower, tomorrow)
		intent.StartTime = &startTime
		endTime := startTime.Add(1 * time.Hour)
		intent.EndTime = &endTime
	} else if strings.Contains(msgLower, "today") {
		startTime := nlp.extractTimeFromMessage(msgLower, now)
		intent.StartTime = &startTime
		endTime := startTime.Add(1 * time.Hour)
		intent.EndTime = &endTime
	} else if strings.Contains(msgLower, "next week") {
		nextWeek := now.AddDate(0, 0, 7)
		intent.StartTime = &nextWeek
		endTime := nextWeek.Add(1 * time.Hour)
		intent.EndTime = &endTime
	}

	// Extract meeting/event title
	// Look for quoted text first
	quoteRegex := regexp.MustCompile(`"([^"]+)"`)
	if matches := quoteRegex.FindStringSubmatch(message); len(matches) > 1 {
		intent.EventTitle = matches[1]
	} else {
		// Try to extract meaningful title from common patterns
		patterns := []string{
			`(?i)(?:schedule|book|create|add)\s+(?:a\s+)?(\w+(?:\s+\w+){0,3})`,
			`(?i)(\w+(?:\s+\w+){0,3})\s+(?:meeting|appointment|event|call)`,
		}
		for _, pattern := range patterns {
			re := regexp.MustCompile(pattern)
			if matches := re.FindStringSubmatch(message); len(matches) > 1 {
				intent.EventTitle = strings.TrimSpace(matches[1])
				break
			}
		}
	}

	// Extract location
	if idx := strings.Index(msgLower, " at "); idx > 0 && idx < len(msgLower)-4 {
		// Look for location after "at"
		afterAt := message[idx+4:]
		words := strings.Fields(afterAt)
		if len(words) > 0 {
			// Take up to 3 words as location
			locationWords := words[:min(3, len(words))]
			intent.Location = strings.Join(locationWords, " ")
		}
	}

	// Extract attendees (look for names or "with")
	if idx := strings.Index(msgLower, " with "); idx > 0 {
		afterWith := message[idx+6:]
		// Split by common separators
		attendeeStr := strings.FieldsFunc(afterWith, func(r rune) bool {
			return r == ',' || r == ';' || r == '&'
		})
		for _, attendee := range attendeeStr {
			cleaned := strings.TrimSpace(attendee)
			if cleaned != "" && cleaned != "and" {
				intent.Attendees = append(intent.Attendees, cleaned)
			}
		}
	}

	// Set default reminders
	if intent.Action == "create" && intent.StartTime != nil {
		intent.Reminders = []ReminderRequest{
			{MinutesBefore: 15, NotificationType: "email"},
		}
	}

	return intent
}

func (nlp *NLPProcessor) extractTimeFromMessage(message string, date time.Time) time.Time {
	// Common time patterns
	timePatterns := []struct {
		pattern *regexp.Regexp
		format  string
	}{
		{regexp.MustCompile(`(\d{1,2}):(\d{2})\s*(am|pm)`), "time12"},
		{regexp.MustCompile(`(\d{1,2})\s*(am|pm)`), "hour12"},
		{regexp.MustCompile(`(\d{1,2}):(\d{2})`), "time24"},
	}

	for _, tp := range timePatterns {
		if matches := tp.pattern.FindStringSubmatch(message); len(matches) > 0 {
			switch tp.format {
			case "time12":
				hour, _ := parseIntSafe(matches[1])
				minute, _ := parseIntSafe(matches[2])
				isPM := strings.ToLower(matches[3]) == "pm"
				if isPM && hour != 12 {
					hour += 12
				} else if !isPM && hour == 12 {
					hour = 0
				}
				return time.Date(date.Year(), date.Month(), date.Day(), hour, minute, 0, 0, date.Location())
			case "hour12":
				hour, _ := parseIntSafe(matches[1])
				isPM := strings.ToLower(matches[2]) == "pm"
				if isPM && hour != 12 {
					hour += 12
				} else if !isPM && hour == 12 {
					hour = 0
				}
				return time.Date(date.Year(), date.Month(), date.Day(), hour, 0, 0, 0, date.Location())
			case "time24":
				hour, _ := parseIntSafe(matches[1])
				minute, _ := parseIntSafe(matches[2])
				return time.Date(date.Year(), date.Month(), date.Day(), hour, minute, 0, 0, date.Location())
			}
		}
	}

	// Default to 9 AM if no time found
	return time.Date(date.Year(), date.Month(), date.Day(), 9, 0, 0, 0, date.Location())
}

func (nlp *NLPProcessor) handleCreateEvent(ctx context.Context, userID string, intent *ParsedScheduleIntent) (*ChatResponse, error) {
	// Validate required fields
	if intent.EventTitle == "" {
		return &ChatResponse{
			Response: "I need a title for the event. What would you like to call it?",
			SuggestedActions: []SuggestedAction{
				{Action: "provide_title", Confidence: 1.0},
			},
		}, nil
	}

	if intent.StartTime == nil {
		return &ChatResponse{
			Response: "When would you like to schedule this event?",
			SuggestedActions: []SuggestedAction{
				{Action: "provide_time", Confidence: 1.0},
			},
		}, nil
	}

	// Create the event
	eventID := uuid.New().String()
	endTime := intent.EndTime
	if endTime == nil {
		// Default to 1 hour duration
		defaultEnd := intent.StartTime.Add(1 * time.Hour)
		endTime = &defaultEnd
	}

	// Prepare metadata
	metadata := map[string]interface{}{
		"created_via": "chat",
		"nlp_confidence": intent.Confidence,
	}
	if len(intent.Attendees) > 0 {
		metadata["attendees"] = intent.Attendees
	}
	metadataJSON, _ := json.Marshal(metadata)

	// Insert event into database
	query := `
		INSERT INTO events (id, user_id, title, description, start_time, end_time, 
			timezone, location, event_type, status, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())`

	_, err := nlp.db.ExecContext(ctx, query, eventID, userID, intent.EventTitle,
		intent.Description, intent.StartTime, endTime, "UTC", intent.Location,
		"meeting", "scheduled", metadataJSON)
	
	if err != nil {
		return nil, fmt.Errorf("failed to create event: %w", err)
	}

	// Create reminders if specified
	if len(intent.Reminders) > 0 {
		processor := NewCalendarProcessor(nlp.db, nlp.config)
		if err := processor.CreateRemindersForEvent(ctx, eventID, userID, *intent.StartTime, intent.Reminders); err != nil {
			// Log error but don't fail the event creation
			fmt.Printf("Failed to create reminders: %v\n", err)
		}
	}

	// Format response
	response := fmt.Sprintf("I've scheduled '%s' for %s",
		intent.EventTitle,
		intent.StartTime.Format("Monday, January 2 at 3:04 PM"))
	
	if intent.Location != "" {
		response += fmt.Sprintf(" at %s", intent.Location)
	}
	
	if len(intent.Attendees) > 0 {
		response += fmt.Sprintf(" with %s", strings.Join(intent.Attendees, ", "))
	}

	return &ChatResponse{
		Response: response,
		SuggestedActions: []SuggestedAction{
			{
				Action:     "view_event",
				Parameters: map[string]interface{}{"event_id": eventID},
				Confidence: 1.0,
			},
			{
				Action:     "add_reminder",
				Parameters: map[string]interface{}{"event_id": eventID},
				Confidence: 0.8,
			},
		},
	}, nil
}

func (nlp *NLPProcessor) handleUpdateEvent(ctx context.Context, userID string, intent *ParsedScheduleIntent) (*ChatResponse, error) {
	// Find the event to update
	// This is simplified - in production, you'd use more sophisticated matching
	query := `
		SELECT id, title FROM events 
		WHERE user_id = $1 AND status != 'cancelled'
		AND (title ILIKE $2 OR $2 = '')
		ORDER BY start_time DESC LIMIT 1`
	
	var eventID, eventTitle string
	searchTitle := "%" + intent.EventTitle + "%"
	err := nlp.db.QueryRowContext(ctx, query, userID, searchTitle).Scan(&eventID, &eventTitle)
	if err != nil {
		return &ChatResponse{
			Response: "I couldn't find an event matching that description. Can you be more specific?",
			SuggestedActions: []SuggestedAction{
				{Action: "list_events", Confidence: 0.9},
			},
		}, nil
	}

	// Build update query
	updates := []string{}
	params := []interface{}{eventID}
	paramCount := 1

	if intent.StartTime != nil {
		paramCount++
		updates = append(updates, fmt.Sprintf("start_time = $%d", paramCount))
		params = append(params, *intent.StartTime)
	}
	if intent.EndTime != nil {
		paramCount++
		updates = append(updates, fmt.Sprintf("end_time = $%d", paramCount))
		params = append(params, *intent.EndTime)
	}
	if intent.Location != "" {
		paramCount++
		updates = append(updates, fmt.Sprintf("location = $%d", paramCount))
		params = append(params, intent.Location)
	}
	if intent.Description != "" {
		paramCount++
		updates = append(updates, fmt.Sprintf("description = $%d", paramCount))
		params = append(params, intent.Description)
	}

	if len(updates) == 0 {
		return &ChatResponse{
			Response: "What would you like to change about this event?",
			SuggestedActions: []SuggestedAction{
				{Action: "specify_changes", Confidence: 1.0},
			},
		}, nil
	}

	updateQuery := fmt.Sprintf("UPDATE events SET %s, updated_at = NOW() WHERE id = $1",
		strings.Join(updates, ", "))
	
	if _, err := nlp.db.ExecContext(ctx, updateQuery, params...); err != nil {
		return nil, fmt.Errorf("failed to update event: %w", err)
	}

	return &ChatResponse{
		Response: fmt.Sprintf("I've updated '%s' with the new details.", eventTitle),
		SuggestedActions: []SuggestedAction{
			{
				Action:     "view_event",
				Parameters: map[string]interface{}{"event_id": eventID},
				Confidence: 1.0,
			},
		},
	}, nil
}

func (nlp *NLPProcessor) handleDeleteEvent(ctx context.Context, userID string, intent *ParsedScheduleIntent) (*ChatResponse, error) {
	// Find and cancel the event
	query := `
		UPDATE events SET status = 'cancelled', updated_at = NOW()
		WHERE user_id = $1 AND status != 'cancelled'
		AND (title ILIKE $2 OR $2 = '')
		RETURNING id, title`
	
	var eventID, eventTitle string
	searchTitle := "%" + intent.EventTitle + "%"
	err := nlp.db.QueryRowContext(ctx, query, userID, searchTitle).Scan(&eventID, &eventTitle)
	if err != nil {
		return &ChatResponse{
			Response: "I couldn't find an event to cancel. Can you be more specific?",
			SuggestedActions: []SuggestedAction{
				{Action: "list_events", Confidence: 0.9},
			},
		}, nil
	}

	// Cancel associated reminders
	reminderQuery := `UPDATE reminders SET status = 'cancelled' WHERE event_id = $1`
	nlp.db.ExecContext(ctx, reminderQuery, eventID)

	return &ChatResponse{
		Response: fmt.Sprintf("I've cancelled '%s'.", eventTitle),
		SuggestedActions: []SuggestedAction{
			{
				Action:     "undo_cancel",
				Parameters: map[string]interface{}{"event_id": eventID},
				Confidence: 0.7,
			},
		},
	}, nil
}

func (nlp *NLPProcessor) handleQueryEvents(ctx context.Context, userID string, intent *ParsedScheduleIntent) (*ChatResponse, error) {
	// Build query based on intent
	query := `
		SELECT id, title, start_time, end_time, location 
		FROM events 
		WHERE user_id = $1 AND status != 'cancelled'`
	
	params := []interface{}{userID}
	
	// Add time filters if specified
	if intent.StartTime != nil {
		query += " AND start_time >= $2"
		params = append(params, *intent.StartTime)
		if intent.EndTime != nil {
			query += " AND start_time <= $3"
			params = append(params, *intent.EndTime)
		} else {
			// Default to end of day
			endOfDay := intent.StartTime.Add(24 * time.Hour)
			query += " AND start_time <= $3"
			params = append(params, endOfDay)
		}
	}
	
	query += " ORDER BY start_time LIMIT 10"
	
	rows, err := nlp.db.QueryContext(ctx, query, params...)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var events []string
	hasEvents := false
	for rows.Next() {
		var id, title, location string
		var startTime, endTime time.Time
		if err := rows.Scan(&id, &title, &startTime, &endTime, &location); err != nil {
			continue
		}
		hasEvents = true
		
		eventStr := fmt.Sprintf("• %s on %s",
			title,
			startTime.Format("Mon Jan 2 at 3:04 PM"))
		if location != "" {
			eventStr += fmt.Sprintf(" at %s", location)
		}
		events = append(events, eventStr)
	}

	if !hasEvents {
		return &ChatResponse{
			Response: "You don't have any upcoming events scheduled.",
			SuggestedActions: []SuggestedAction{
				{Action: "create_event", Confidence: 0.8},
			},
		}, nil
	}

	return &ChatResponse{
		Response: fmt.Sprintf("Here are your upcoming events:\n%s", strings.Join(events, "\n")),
		SuggestedActions: []SuggestedAction{
			{Action: "create_event", Confidence: 0.7},
			{Action: "show_calendar", Confidence: 0.9},
		},
	}, nil
}

func (nlp *NLPProcessor) handleAmbiguous(ctx context.Context, intent *ParsedScheduleIntent, originalMessage string) (*ChatResponse, error) {
	// When intent is unclear, ask for clarification
	suggestions := []SuggestedAction{}
	
	if strings.Contains(strings.ToLower(originalMessage), "free") ||
	   strings.Contains(strings.ToLower(originalMessage), "available") {
		suggestions = append(suggestions, SuggestedAction{
			Action:     "find_free_time",
			Confidence: 0.8,
		})
	}
	
	suggestions = append(suggestions, []SuggestedAction{
		{Action: "create_event", Confidence: 0.6},
		{Action: "list_events", Confidence: 0.6},
		{Action: "help", Confidence: 0.5},
	}...)

	return &ChatResponse{
		Response: "I'm not sure what you'd like to do. You can:\n• Schedule a new event\n• View your calendar\n• Update or cancel an existing event\n• Find free time in your schedule",
		SuggestedActions: suggestions,
	}, nil
}

func parseIntSafe(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}