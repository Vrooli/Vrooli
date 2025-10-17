package main

import (
	"database/sql"
	"fmt"
	"strings"
)

// EventCategory represents a category for events
type EventCategory struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Color       string `json:"color"` // Hex color code for UI display
	Icon        string `json:"icon"`  // Icon name for UI display
	Description string `json:"description"`
	IsDefault   bool   `json:"is_default"` // System-provided categories
	UserID      string `json:"user_id"`    // NULL for system categories
}

// CategoryManager handles event categorization
type CategoryManager struct {
	db *sql.DB
}

// NewCategoryManager creates a new category manager
func NewCategoryManager(db *sql.DB) *CategoryManager {
	return &CategoryManager{
		db: db,
	}
}

// GetDefaultCategories returns the system-defined categories
func (cm *CategoryManager) GetDefaultCategories() []EventCategory {
	return []EventCategory{
		{
			ID:          "meeting",
			Name:        "Meeting",
			Color:       "#4285F4", // Blue
			Icon:        "people",
			Description: "Team meetings, 1-on-1s, and conference calls",
			IsDefault:   true,
		},
		{
			ID:          "appointment",
			Name:        "Appointment",
			Color:       "#34A853", // Green
			Icon:        "event",
			Description: "Doctor visits, client meetings, and scheduled appointments",
			IsDefault:   true,
		},
		{
			ID:          "task",
			Name:        "Task",
			Color:       "#FBBC04", // Yellow
			Icon:        "task",
			Description: "Deadlines, deliverables, and work tasks",
			IsDefault:   true,
		},
		{
			ID:          "personal",
			Name:        "Personal",
			Color:       "#EA4335", // Red
			Icon:        "person",
			Description: "Personal time, breaks, and non-work activities",
			IsDefault:   true,
		},
		{
			ID:          "travel",
			Name:        "Travel",
			Color:       "#9C27B0", // Purple
			Icon:        "flight",
			Description: "Flights, commutes, and travel time",
			IsDefault:   true,
		},
		{
			ID:          "reminder",
			Name:        "Reminder",
			Color:       "#FF9800", // Orange
			Icon:        "notification",
			Description: "Reminders and notifications",
			IsDefault:   true,
		},
		{
			ID:          "focus",
			Name:        "Focus Time",
			Color:       "#00BCD4", // Cyan
			Icon:        "hourglass",
			Description: "Deep work and focused time blocks",
			IsDefault:   true,
		},
		{
			ID:          "social",
			Name:        "Social",
			Color:       "#E91E63", // Pink
			Icon:        "group",
			Description: "Social events, parties, and gatherings",
			IsDefault:   true,
		},
	}
}

// GetCategories retrieves all categories (default + user custom)
func (cm *CategoryManager) GetCategories(userID string) ([]EventCategory, error) {
	categories := cm.GetDefaultCategories()

	// Query for user-defined categories
	query := `
		SELECT id, name, color, icon, description 
		FROM event_categories 
		WHERE user_id = $1
		ORDER BY name
	`

	rows, err := cm.db.Query(query, userID)
	if err != nil {
		return categories, nil // Return default categories even if query fails
	}
	defer rows.Close()

	for rows.Next() {
		var category EventCategory
		err := rows.Scan(&category.ID, &category.Name, &category.Color,
			&category.Icon, &category.Description)
		if err != nil {
			continue
		}
		category.UserID = userID
		category.IsDefault = false
		categories = append(categories, category)
	}

	return categories, nil
}

// CreateCategory creates a custom category for a user
func (cm *CategoryManager) CreateCategory(userID string, category EventCategory) (*EventCategory, error) {
	// Validate category
	if category.Name == "" {
		return nil, fmt.Errorf("category name is required")
	}

	// Set defaults
	if category.Color == "" {
		category.Color = "#808080" // Default gray
	}
	if category.Icon == "" {
		category.Icon = "label"
	}

	// Generate ID from name if not provided
	if category.ID == "" {
		category.ID = strings.ToLower(strings.ReplaceAll(category.Name, " ", "_"))
	}

	query := `
		INSERT INTO event_categories (id, user_id, name, color, icon, description)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	err := cm.db.QueryRow(query, category.ID, userID, category.Name,
		category.Color, category.Icon, category.Description).Scan(&category.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to create category: %v", err)
	}

	category.UserID = userID
	category.IsDefault = false

	return &category, nil
}

// SuggestCategory suggests a category for an event based on its title and description
func (cm *CategoryManager) SuggestCategory(title, description string) string {
	text := strings.ToLower(title + " " + description)

	// Meeting keywords
	meetingKeywords := []string{"meeting", "sync", "standup", "1:1", "1-on-1", "call",
		"conference", "discussion", "review", "retrospective", "demo", "presentation"}
	for _, keyword := range meetingKeywords {
		if strings.Contains(text, keyword) {
			return "meeting"
		}
	}

	// Appointment keywords
	appointmentKeywords := []string{"appointment", "doctor", "dentist", "interview",
		"consultation", "session", "therapy", "checkup", "exam"}
	for _, keyword := range appointmentKeywords {
		if strings.Contains(text, keyword) {
			return "appointment"
		}
	}

	// Task keywords
	taskKeywords := []string{"deadline", "due", "submit", "complete", "finish",
		"deliver", "release", "deploy", "launch", "milestone"}
	for _, keyword := range taskKeywords {
		if strings.Contains(text, keyword) {
			return "task"
		}
	}

	// Travel keywords
	travelKeywords := []string{"flight", "travel", "trip", "commute", "drive",
		"airport", "train", "bus", "transit"}
	for _, keyword := range travelKeywords {
		if strings.Contains(text, keyword) {
			return "travel"
		}
	}

	// Focus time keywords
	focusKeywords := []string{"focus", "deep work", "coding", "writing", "research",
		"study", "analyze", "design", "planning"}
	for _, keyword := range focusKeywords {
		if strings.Contains(text, keyword) {
			return "focus"
		}
	}

	// Social keywords
	socialKeywords := []string{"party", "celebration", "birthday", "lunch", "dinner",
		"coffee", "drinks", "happy hour", "team building", "outing"}
	for _, keyword := range socialKeywords {
		if strings.Contains(text, keyword) {
			return "social"
		}
	}

	// Personal keywords
	personalKeywords := []string{"personal", "break", "gym", "workout", "exercise",
		"meditation", "walk", "errand", "shopping"}
	for _, keyword := range personalKeywords {
		if strings.Contains(text, keyword) {
			return "personal"
		}
	}

	// Default to meeting if no match
	return "meeting"
}

// EventFilter represents filtering criteria for events
type EventFilter struct {
	Categories   []string `json:"categories"`
	EventTypes   []string `json:"event_types"` // Legacy support
	Tags         []string `json:"tags"`
	MinDuration  int      `json:"min_duration_minutes"`
	MaxDuration  int      `json:"max_duration_minutes"`
	HasReminders bool     `json:"has_reminders"`
	HasLocation  bool     `json:"has_location"`
}

// BuildFilterQuery builds SQL conditions for event filtering
func (cm *CategoryManager) BuildFilterQuery(filter EventFilter, argCount int) (string, []interface{}, int) {
	conditions := []string{}
	args := []interface{}{}

	// Filter by categories
	if len(filter.Categories) > 0 {
		placeholders := []string{}
		for _, category := range filter.Categories {
			argCount++
			placeholders = append(placeholders, fmt.Sprintf("$%d", argCount))
			args = append(args, category)
		}
		conditions = append(conditions, fmt.Sprintf("event_type IN (%s)", strings.Join(placeholders, ", ")))
	}

	// Filter by tags (stored in metadata)
	if len(filter.Tags) > 0 {
		for _, tag := range filter.Tags {
			argCount++
			conditions = append(conditions, fmt.Sprintf("metadata->'tags' @> $%d", argCount))
			args = append(args, fmt.Sprintf(`["%s"]`, tag))
		}
	}

	// Filter by duration
	if filter.MinDuration > 0 {
		argCount++
		conditions = append(conditions, fmt.Sprintf("EXTRACT(EPOCH FROM (end_time - start_time))/60 >= $%d", argCount))
		args = append(args, filter.MinDuration)
	}

	if filter.MaxDuration > 0 {
		argCount++
		conditions = append(conditions, fmt.Sprintf("EXTRACT(EPOCH FROM (end_time - start_time))/60 <= $%d", argCount))
		args = append(args, filter.MaxDuration)
	}

	// Filter by reminders
	if filter.HasReminders {
		conditions = append(conditions, `EXISTS (
			SELECT 1 FROM event_reminders 
			WHERE event_reminders.event_id = events.id
		)`)
	}

	// Filter by location
	if filter.HasLocation {
		conditions = append(conditions, "location IS NOT NULL AND location != ''")
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " AND " + strings.Join(conditions, " AND ")
	}

	return whereClause, args, argCount
}

// GetEventsByCategory retrieves events grouped by category
func (cm *CategoryManager) GetEventsByCategory(userID string, startDate, endDate string) (map[string][]Event, error) {
	query := `
		SELECT id, user_id, title, description, start_time, end_time, 
		       timezone, location, event_type, status, metadata, 
		       automation_config, created_at, updated_at
		FROM events 
		WHERE user_id = $1 
		AND status = 'active'
		AND start_time >= $2 
		AND start_time <= $3
		ORDER BY start_time
	`

	rows, err := cm.db.Query(query, userID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %v", err)
	}
	defer rows.Close()

	eventsByCategory := make(map[string][]Event)

	for rows.Next() {
		var event Event
		var metadataJSON, automationJSON []byte

		err := rows.Scan(
			&event.ID, &event.UserID, &event.Title, &event.Description,
			&event.StartTime, &event.EndTime, &event.Timezone, &event.Location,
			&event.EventType, &event.Status, &metadataJSON, &automationJSON,
			&event.CreatedAt, &event.UpdatedAt,
		)
		if err != nil {
			continue
		}

		// Use event_type as category, default to "meeting" if empty
		category := event.EventType
		if category == "" {
			category = cm.SuggestCategory(event.Title, event.Description)
		}

		if _, exists := eventsByCategory[category]; !exists {
			eventsByCategory[category] = []Event{}
		}

		eventsByCategory[category] = append(eventsByCategory[category], event)
	}

	return eventsByCategory, nil
}

// GetCategoryStatistics returns statistics about event categories
func (cm *CategoryManager) GetCategoryStatistics(userID string, startDate, endDate string) (map[string]interface{}, error) {
	query := `
		SELECT 
			COALESCE(event_type, 'uncategorized') as category,
			COUNT(*) as event_count,
			SUM(EXTRACT(EPOCH FROM (end_time - start_time))/3600) as total_hours
		FROM events 
		WHERE user_id = $1 
		AND status = 'active'
		AND start_time >= $2 
		AND start_time <= $3
		GROUP BY event_type
		ORDER BY event_count DESC
	`

	rows, err := cm.db.Query(query, userID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query statistics: %v", err)
	}
	defer rows.Close()

	stats := make([]map[string]interface{}, 0)
	var totalEvents int
	var totalHours float64

	for rows.Next() {
		var category string
		var count int
		var hours float64

		err := rows.Scan(&category, &count, &hours)
		if err != nil {
			continue
		}

		totalEvents += count
		totalHours += hours

		stats = append(stats, map[string]interface{}{
			"category":   category,
			"count":      count,
			"hours":      hours,
			"percentage": 0, // Will calculate after loop
		})
	}

	// Calculate percentages
	for i := range stats {
		if totalEvents > 0 {
			stats[i]["percentage"] = float64(stats[i]["count"].(int)) / float64(totalEvents) * 100
		}
	}

	return map[string]interface{}{
		"categories":   stats,
		"total_events": totalEvents,
		"total_hours":  totalHours,
		"period": map[string]string{
			"start": startDate,
			"end":   endDate,
		},
	}, nil
}
