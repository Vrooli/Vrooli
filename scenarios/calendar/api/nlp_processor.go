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
		log.Println("⚠️  NLP Processor initialized without Ollama - using rule-based fallback")
	}
	return &NLPProcessor{
		ollamaURL: ollamaURL,
		db:        db,
		config:    config,
	}
}

// ProcessSchedulingRequest processes natural language scheduling requests (replaces n8n ollama.json workflow)
func (nlp *NLPProcessor) ProcessSchedulingRequest(ctx context.Context, userID, message string) (*ChatResponse, error) {
	// Add timeout context for the entire processing pipeline
	processCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	// Input validation
	if strings.TrimSpace(message) == "" {
		return &ChatResponse{
			Response: "I didn't receive any message to process. What would you like to do with your calendar?",
			SuggestedActions: []SuggestedAction{
				{Action: "help", Confidence: 1.0},
				{Action: "create_event", Confidence: 0.8},
			},
		}, nil
	}

	if len(message) > 1000 {
		return &ChatResponse{
			Response: "That message is too long for me to process effectively. Please try a shorter request.",
			SuggestedActions: []SuggestedAction{
				{Action: "help", Confidence: 0.9},
			},
		}, nil
	}

	// First, try to parse the intent using Ollama if available
	intent, ollamaErr := nlp.parseSchedulingIntent(processCtx, message)
	if ollamaErr != nil {
		log.Printf("⚠️  Ollama parsing failed, falling back to rule-based: %v", ollamaErr)
		// Fallback to rule-based parsing if Ollama is not available
		intent = nlp.fallbackParsing(message)
		// Enhance fallback confidence based on error type
		if intent != nil {
			if strings.Contains(ollamaErr.Error(), "not configured") {
				intent.Confidence = 0.6 // Reasonable confidence for rule-based
			} else {
				intent.Confidence = 0.4 // Lower confidence due to AI failure
			}
		}
	}

	// Validate parsed intent
	if intent == nil {
		return &ChatResponse{
			Response: "I had trouble understanding your request. Could you please rephrase it?",
			SuggestedActions: []SuggestedAction{
				{Action: "help", Confidence: 1.0},
				{Action: "create_event", Confidence: 0.5},
				{Action: "list_events", Confidence: 0.5},
			},
		}, nil
	}

	// Execute the parsed intent with error context
	var result *ChatResponse
	var actionErr error

	switch intent.Action {
	case "create":
		result, actionErr = nlp.handleCreateEvent(processCtx, userID, intent)
	case "update":
		result, actionErr = nlp.handleUpdateEvent(processCtx, userID, intent)
	case "delete":
		result, actionErr = nlp.handleDeleteEvent(processCtx, userID, intent)
	case "query":
		result, actionErr = nlp.handleQueryEvents(processCtx, userID, intent)
	default:
		result, actionErr = nlp.handleAmbiguous(processCtx, intent, message)
	}

	// Enhanced error handling with user-friendly responses
	if actionErr != nil {
		log.Printf("❌ Action execution failed for user %s, action %s: %v", userID, intent.Action, actionErr)
		
		// Check for specific error types and provide appropriate responses
		if strings.Contains(actionErr.Error(), "context deadline exceeded") {
			return &ChatResponse{
				Response: "I'm taking longer than usual to process your request. Please try again in a moment.",
				SuggestedActions: []SuggestedAction{
					{Action: "retry", Confidence: 0.9},
					{Action: "help", Confidence: 0.5},
				},
			}, nil
		}
		
		if strings.Contains(actionErr.Error(), "database") || strings.Contains(actionErr.Error(), "connection") {
			return &ChatResponse{
				Response: "I'm having trouble connecting to the calendar system right now. Please try again in a few moments.",
				SuggestedActions: []SuggestedAction{
					{Action: "retry", Confidence: 0.8},
					{Action: "help", Confidence: 0.6},
				},
			}, nil
		}

		// Generic error fallback
		return &ChatResponse{
			Response: "I encountered an issue processing your request. Please try rephrasing your message or try again later.",
			SuggestedActions: []SuggestedAction{
				{Action: "help", Confidence: 0.9},
				{Action: "retry", Confidence: 0.7},
			},
		}, nil
	}

	return result, nil
}

func (nlp *NLPProcessor) parseSchedulingIntent(ctx context.Context, message string) (*ParsedScheduleIntent, error) {
	// Check if Ollama is available
	if nlp.ollamaURL == "" {
		return nil, fmt.Errorf("Ollama not configured")
	}
	
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

	// Enhanced JSON parsing from Ollama's message content
	var intent ParsedScheduleIntent
	content := strings.TrimSpace(ollamaResp.Message.Content)
	
	if content == "" {
		return nil, fmt.Errorf("received empty response from Ollama")
	}

	// Try multiple JSON extraction strategies
	var jsonStr string
	// parseErr not used - removed

	// Strategy 1: Look for JSON block markers
	if codeStart := strings.Index(content, "```json"); codeStart >= 0 {
		codeStart += 7 // Skip "```json"
		if codeEnd := strings.Index(content[codeStart:], "```"); codeEnd > 0 {
			jsonStr = strings.TrimSpace(content[codeStart : codeStart+codeEnd])
		}
	}

	// Strategy 2: Find complete JSON object (most robust)
	if jsonStr == "" {
		var braceCount int
		var startIdx = -1
		for i, char := range content {
			if char == '{' {
				if startIdx == -1 {
					startIdx = i
				}
				braceCount++
			} else if char == '}' {
				braceCount--
				if braceCount == 0 && startIdx != -1 {
					jsonStr = content[startIdx : i+1]
					break
				}
			}
		}
	}

	// Strategy 3: Fallback to simple search (legacy)
	if jsonStr == "" {
		jsonStart := strings.Index(content, "{")
		jsonEnd := strings.LastIndex(content, "}")
		if jsonStart >= 0 && jsonEnd >= jsonStart {
			jsonStr = content[jsonStart : jsonEnd+1]
		}
	}

	if jsonStr == "" {
		return nil, fmt.Errorf("no valid JSON found in Ollama response: %s", content[:min(100, len(content))])
	}

	// Validate and parse JSON with detailed error handling
	if err := json.Unmarshal([]byte(jsonStr), &intent); err != nil {
		// Try to provide helpful error context
		syntaxErr, isSyntaxErr := err.(*json.SyntaxError)
		if isSyntaxErr {
			// Calculate line and column for better debugging
			lines := strings.Split(jsonStr, "\n")
			lineNum := 1
			colNum := syntaxErr.Offset
			for _, line := range lines {
				if colNum <= int64(len(line)) {
					break
				}
				colNum -= int64(len(line) + 1) // +1 for newline
				lineNum++
			}
			return nil, fmt.Errorf("JSON syntax error at line %d, column %d: %w\nJSON excerpt: %s", 
				lineNum, colNum, err, jsonStr[:min(200, len(jsonStr))])
		}
		return nil, fmt.Errorf("failed to parse Ollama JSON response: %w\nJSON: %s", err, jsonStr[:min(200, len(jsonStr))])
	}

	// Validate parsed intent for required fields and reasonable values
	if err := nlp.validateParsedIntent(&intent); err != nil {
		log.Printf("⚠️  Intent validation warning: %v", err)
		// Don't fail completely, but log the issue for monitoring
	}

	return &intent, nil
}

// validateParsedIntent performs validation on parsed intent to catch common issues
func (nlp *NLPProcessor) validateParsedIntent(intent *ParsedScheduleIntent) error {
	if intent == nil {
		return fmt.Errorf("intent is nil")
	}

	// Validate action
	validActions := map[string]bool{"create": true, "update": true, "delete": true, "query": true}
	if !validActions[intent.Action] {
		return fmt.Errorf("invalid action: %s", intent.Action)
	}

	// Validate confidence score
	if intent.Confidence < 0.0 || intent.Confidence > 1.0 {
		return fmt.Errorf("confidence score out of range: %f", intent.Confidence)
	}

	// Validate time logic if both times are present
	if intent.StartTime != nil && intent.EndTime != nil {
		if intent.EndTime.Before(*intent.StartTime) || intent.EndTime.Equal(*intent.StartTime) {
			return fmt.Errorf("end time (%v) must be after start time (%v)", intent.EndTime, intent.StartTime)
		}
		
		// Check for unreasonably long events (more than 24 hours)
		duration := intent.EndTime.Sub(*intent.StartTime)
		if duration > 24*time.Hour {
			return fmt.Errorf("event duration too long: %v", duration)
		}
	}

	// Validate start time is not too far in the past
	if intent.StartTime != nil {
		if intent.StartTime.Before(time.Now().Add(-24 * time.Hour)) {
			return fmt.Errorf("start time is more than 24 hours in the past: %v", intent.StartTime)
		}
		
		// Warn about events scheduled too far in the future (more than 2 years)
		if intent.StartTime.After(time.Now().Add(2 * 365 * 24 * time.Hour)) {
			return fmt.Errorf("start time is more than 2 years in the future: %v", intent.StartTime)
		}
	}

	// Validate attendees format (basic email validation)
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	for _, attendee := range intent.Attendees {
		attendee = strings.TrimSpace(attendee)
		if attendee != "" && strings.Contains(attendee, "@") {
			// If it contains @, it should be a valid email
			if !emailRegex.MatchString(attendee) {
				return fmt.Errorf("invalid email format for attendee: %s", attendee)
			}
		}
	}

	// Validate reminders
	for _, reminder := range intent.Reminders {
		if reminder.MinutesBefore < 0 {
			return fmt.Errorf("reminder minutes_before cannot be negative: %d", reminder.MinutesBefore)
		}
		if reminder.MinutesBefore > 10080 { // More than a week
			return fmt.Errorf("reminder too far in advance: %d minutes", reminder.MinutesBefore)
		}
		validTypes := map[string]bool{"email": true, "push": true, "sms": true, "webhook": true}
		if !validTypes[reminder.NotificationType] {
			return fmt.Errorf("invalid reminder type: %s", reminder.NotificationType)
		}
	}

	return nil
}

func (nlp *NLPProcessor) fallbackParsing(message string) *ParsedScheduleIntent {
	// Input validation
	if strings.TrimSpace(message) == "" {
		return nil
	}

	intent := &ParsedScheduleIntent{
		Confidence: 0.5,
	}

	msgLower := strings.ToLower(message)
	
	// Enhanced action detection with scoring
	actionScores := make(map[string]int)
	
	// Create action indicators
	createWords := []string{"schedule", "create", "add", "book", "set up", "plan", "arrange"}
	for _, word := range createWords {
		if strings.Contains(msgLower, word) {
			actionScores["create"]++
		}
	}
	
	// Update action indicators
	updateWords := []string{"update", "change", "modify", "reschedule", "move", "shift", "edit"}
	for _, word := range updateWords {
		if strings.Contains(msgLower, word) {
			actionScores["update"]++
		}
	}
	
	// Delete action indicators
	deleteWords := []string{"delete", "cancel", "remove", "clear", "drop"}
	for _, word := range deleteWords {
		if strings.Contains(msgLower, word) {
			actionScores["delete"]++
		}
	}
	
	// Query action indicators
	queryWords := []string{"show", "list", "what", "when", "find", "search", "display", "view"}
	for _, word := range queryWords {
		if strings.Contains(msgLower, word) {
			actionScores["query"]++
		}
	}
	
	// Determine best action based on highest score
	maxScore := 0
	bestAction := "create" // default
	for action, score := range actionScores {
		if score > maxScore {
			maxScore = score
			bestAction = action
		}
	}
	
	intent.Action = bestAction
	
	// Adjust confidence based on action detection strength
	if maxScore > 0 {
		// Calculate confidence with proper float math
	confidenceCalc := 0.5 + float64(maxScore)*0.1
	if confidenceCalc > 0.8 {
		intent.Confidence = 0.8
	} else {
		intent.Confidence = confidenceCalc
	}
	} else {
		intent.Confidence = 0.3 // Lower confidence when no clear action indicators
	}

	// Enhanced time reference extraction
	now := time.Now()
	var baseDate *time.Time
	
	// Determine base date with improved logic
	if strings.Contains(msgLower, "tomorrow") {
		tomorrow := now.AddDate(0, 0, 1)
		baseDate = &tomorrow
	} else if strings.Contains(msgLower, "today") || 
	         strings.Contains(msgLower, "this afternoon") || 
	         strings.Contains(msgLower, "this morning") ||
	         strings.Contains(msgLower, "this evening") {
		baseDate = &now
	} else if strings.Contains(msgLower, "next week") {
		nextWeek := now.AddDate(0, 0, 7)
		baseDate = &nextWeek
	} else if strings.Contains(msgLower, "next month") {
		nextMonth := now.AddDate(0, 1, 0)
		baseDate = &nextMonth
	} else {
		// Look for specific day names
		dayNames := map[string]int{
			"monday": 1, "tuesday": 2, "wednesday": 3, "thursday": 4,
			"friday": 5, "saturday": 6, "sunday": 0,
		}
		
		for dayName, dayNum := range dayNames {
			if strings.Contains(msgLower, dayName) {
				// Find next occurrence of this day
				currentDay := int(now.Weekday())
				daysUntil := (dayNum - currentDay + 7) % 7
				if daysUntil == 0 {
					daysUntil = 7 // Next week if it's the same day
				}
				targetDate := now.AddDate(0, 0, daysUntil)
				baseDate = &targetDate
				break
			}
		}
	}
	
	// Extract time and set start/end times
	if baseDate != nil {
		startTime := nlp.extractTimeFromMessage(msgLower, *baseDate)
		intent.StartTime = &startTime
		
		// Smart duration detection
		duration := 1 * time.Hour // default
		if strings.Contains(msgLower, "lunch") {
			duration = 1 * time.Hour
		} else if strings.Contains(msgLower, "meeting") || strings.Contains(msgLower, "call") {
			duration = 1 * time.Hour
		} else if strings.Contains(msgLower, "appointment") {
			duration = 30 * time.Minute
		} else if strings.Contains(msgLower, "interview") {
			duration = 1 * time.Hour
		} else if strings.Contains(msgLower, "workshop") || strings.Contains(msgLower, "training") {
			duration = 2 * time.Hour
		}
		
		// Look for explicit duration mentions
		durationPatterns := []struct {
			pattern *regexp.Regexp
			value   time.Duration
		}{
			{regexp.MustCompile(`(\d+)\s*hours?`), 0}, // Will be calculated
			{regexp.MustCompile(`(\d+)\s*minutes?`), 0}, // Will be calculated
			{regexp.MustCompile(`half\s*hour|30\s*min`), 30 * time.Minute},
			{regexp.MustCompile(`quarter\s*hour|15\s*min`), 15 * time.Minute},
			{regexp.MustCompile(`all\s*day`), 8 * time.Hour},
		}
		
		for _, dp := range durationPatterns {
			if matches := dp.pattern.FindStringSubmatch(msgLower); len(matches) > 0 {
				if dp.value > 0 {
					duration = dp.value
				} else if len(matches) > 1 {
					if val, err := parseIntSafe(matches[1]); err == nil && val > 0 {
						if strings.Contains(dp.pattern.String(), "hour") {
							duration = time.Duration(val) * time.Hour
						} else if strings.Contains(dp.pattern.String(), "minute") {
							duration = time.Duration(val) * time.Minute
						}
					}
				}
				break
			}
		}
		
		// Validate duration is reasonable
		if duration > 12*time.Hour {
			duration = 2 * time.Hour // Cap at 2 hours for safety
		}
		
		endTime := startTime.Add(duration)
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
	// Enhanced validation with detailed feedback
	if intent.EventTitle == "" {
		return &ChatResponse{
			Response: "I need a title for the event. What would you like to call it?",
			SuggestedActions: []SuggestedAction{
				{Action: "provide_title", Confidence: 1.0},
			},
			RequiresConfirmation: true,
		}, nil
	}

	if intent.StartTime == nil {
		return &ChatResponse{
			Response: "When would you like to schedule this event?",
			SuggestedActions: []SuggestedAction{
				{Action: "provide_time", Confidence: 1.0},
			},
			RequiresConfirmation: true,
		}, nil
	}

	// Validate event time is not in the past (with small grace period)
	if intent.StartTime.Before(time.Now().Add(-10 * time.Minute)) {
		return &ChatResponse{
			Response: "That time appears to be in the past. When would you like to schedule this event?",
			SuggestedActions: []SuggestedAction{
				{Action: "provide_time", Confidence: 1.0},
			},
			RequiresConfirmation: true,
		}, nil
	}

	// Create the event with enhanced error handling
	eventID := uuid.New().String()
	endTime := intent.EndTime
	if endTime == nil {
		// Default to 1 hour duration
		defaultEnd := intent.StartTime.Add(1 * time.Hour)
		endTime = &defaultEnd
	}

	// Enhanced conflict checking
	conflictQuery := `
		SELECT id, title, start_time, end_time 
		FROM events 
		WHERE user_id = $1 
		AND status = 'active'
		AND (
			(start_time <= $2 AND end_time > $2) OR
			(start_time < $3 AND end_time >= $3) OR
			(start_time >= $2 AND end_time <= $3)
		)
		LIMIT 3`

	conflictRows, err := nlp.db.QueryContext(ctx, conflictQuery, userID, intent.StartTime, endTime)
	if err != nil {
		log.Printf("⚠️  Failed to check for conflicts: %v", err)
		// Continue with creation but log the issue
	} else {
		defer conflictRows.Close()
		var conflicts []string
		for conflictRows.Next() {
			var conflictID, conflictTitle string
			var conflictStart, conflictEnd time.Time
			if err := conflictRows.Scan(&conflictID, &conflictTitle, &conflictStart, &conflictEnd); err == nil {
				conflicts = append(conflicts, fmt.Sprintf("'%s' from %s to %s",
					conflictTitle,
					conflictStart.Format("3:04 PM"),
					conflictEnd.Format("3:04 PM")))
			}
		}
		
		if len(conflicts) > 0 {
			return &ChatResponse{
				Response: fmt.Sprintf("⚠️  You have a scheduling conflict with:\n%s\n\nWould you still like to schedule this event?",
					strings.Join(conflicts, "\n")),
				SuggestedActions: []SuggestedAction{
					{Action: "confirm_with_conflict", Confidence: 0.7, Parameters: map[string]interface{}{"conflicts": conflicts}},
					{Action: "suggest_alternative_time", Confidence: 0.9},
					{Action: "cancel", Confidence: 0.5},
				},
				RequiresConfirmation: true,
				Context: map[string]interface{}{
					"conflicts": conflicts,
					"proposed_event": intent,
				},
			}, nil
		}
	}

	// Enhanced metadata preparation
	metadata := map[string]interface{}{
		"created_via":     "chat",
		"nlp_confidence":  intent.Confidence,
		"processing_time": time.Now().Format(time.RFC3339),
	}
	if len(intent.Attendees) > 0 {
		metadata["attendees"] = intent.Attendees
	}
	if intent.Confidence < 0.7 {
		metadata["verification_recommended"] = true
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		log.Printf("⚠️  Failed to marshal metadata: %v", err)
		metadataJSON = []byte("{}") // Fallback to empty JSON
	}

	// Use transaction for atomic event creation
	tx, err := nlp.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback() // This will be a no-op if commit succeeds

	// Insert event into database with enhanced error context
	query := `
		INSERT INTO events (id, user_id, title, description, start_time, end_time, 
			timezone, location, event_type, status, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())`

	_, err = tx.ExecContext(ctx, query, eventID, userID, intent.EventTitle,
		intent.Description, intent.StartTime, endTime, "UTC", intent.Location,
		"meeting", "active", metadataJSON)

	if err != nil {
		// Provide more specific error messages
		if strings.Contains(err.Error(), "duplicate key") {
			return nil, fmt.Errorf("event ID conflict, please try again: %w", err)
		}
		if strings.Contains(err.Error(), "foreign key") {
			return nil, fmt.Errorf("user ID validation failed: %w", err)
		}
		return nil, fmt.Errorf("failed to create event in database: %w", err)
	}

	// Create reminders with better error handling
	var reminderErrors []string
	if len(intent.Reminders) > 0 {
		// processor := NewCalendarProcessor(nlp.db, nlp.config) // Not needed - inline reminder creation
		for _, reminder := range intent.Reminders {
			// Create reminder inline (method doesn't exist on CalendarProcessor)
		reminderID := uuid.New().String()
		minutesBefore := 15 // Default
		if reminder.MinutesBefore > 0 {
			minutesBefore = reminder.MinutesBefore
		}
		reminderTime := intent.StartTime.Add(time.Duration(-minutesBefore) * time.Minute)
		
		reminderQuery := `
			INSERT INTO event_reminders (id, event_id, minutes_before, notification_type, scheduled_time)
			VALUES ($1, $2, $3, $4, $5)
		`
		
		if _, err := nlp.db.Exec(reminderQuery, reminderID, eventID, minutesBefore, reminder.NotificationType, reminderTime); err != nil {
				reminderErrors = append(reminderErrors, fmt.Sprintf("%s reminder: %v", reminder.NotificationType, err))
				log.Printf("⚠️  Failed to create %s reminder for event %s: %v", reminder.NotificationType, eventID, err)
			}
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit event creation: %w", err)
	}

	// Format response with additional context
	response := fmt.Sprintf("✅ I've scheduled '%s' for %s",
		intent.EventTitle,
		intent.StartTime.Format("Monday, January 2 at 3:04 PM"))

	if intent.Location != "" {
		response += fmt.Sprintf(" at %s", intent.Location)
	}

	if len(intent.Attendees) > 0 {
		response += fmt.Sprintf(" with %s", strings.Join(intent.Attendees, ", "))
	}

	// Add reminder status information
	if len(intent.Reminders) > 0 {
		if len(reminderErrors) == 0 {
			response += fmt.Sprintf("\n🔔 %d reminder(s) have been set", len(intent.Reminders))
		} else {
			response += fmt.Sprintf("\n⚠️  Event created but some reminders failed: %s", strings.Join(reminderErrors, ", "))
		}
	}

	// Include confidence warning if low
	if intent.Confidence < 0.5 {
		response += "\n\n💡 Please verify the event details are correct."
	}

	suggestedActions := []SuggestedAction{
		{
			Action:     "view_event",
			Parameters: map[string]interface{}{"event_id": eventID},
			Confidence: 1.0,
		},
	}

	// Add conditional suggestions
	if len(intent.Reminders) == 0 {
		suggestedActions = append(suggestedActions, SuggestedAction{
			Action:     "add_reminder",
			Parameters: map[string]interface{}{"event_id": eventID},
			Confidence: 0.8,
		})
	}

	if intent.Location == "" {
		suggestedActions = append(suggestedActions, SuggestedAction{
			Action:     "add_location",
			Parameters: map[string]interface{}{"event_id": eventID},
			Confidence: 0.6,
		})
	}

	return &ChatResponse{
		Response:         response,
		SuggestedActions: suggestedActions,
		Context: map[string]interface{}{
			"created_event_id": eventID,
			"confidence":       intent.Confidence,
			"reminder_errors":  reminderErrors,
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
	reminderQuery := `UPDATE event_reminders SET status = 'cancelled' WHERE event_id = $1`
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