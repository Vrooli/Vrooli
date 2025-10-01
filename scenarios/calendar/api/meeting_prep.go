package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// MeetingPrepManager handles meeting preparation automation
type MeetingPrepManager struct {
	db           *sql.DB
	nlpProcessor *NLPProcessor
}

// NewMeetingPrepManager creates a new meeting preparation manager
func NewMeetingPrepManager(db *sql.DB, nlp *NLPProcessor) *MeetingPrepManager {
	return &MeetingPrepManager{
		db:           db,
		nlpProcessor: nlp,
	}
}

// MeetingAgenda represents a generated meeting agenda
type MeetingAgenda struct {
	EventID     string       `json:"event_id"`
	Title       string       `json:"title"`
	Objective   string       `json:"objective"`
	Duration    string       `json:"duration"`
	Attendees   []string     `json:"attendees"`
	AgendaItems []AgendaItem `json:"agenda_items"`
	PreWork     []string     `json:"pre_work,omitempty"`
	Documents   []string     `json:"documents,omitempty"`
	ActionItems []string     `json:"action_items,omitempty"`
	Notes       string       `json:"notes,omitempty"`
	GeneratedAt time.Time    `json:"generated_at"`
}

// AgendaItem represents a single agenda item
type AgendaItem struct {
	Topic       string `json:"topic"`
	Duration    int    `json:"duration_minutes"`
	Owner       string `json:"owner,omitempty"`
	Description string `json:"description,omitempty"`
}

// GenerateAgenda creates an agenda for a meeting
func (m *MeetingPrepManager) GenerateAgenda(eventID, userID string) (*MeetingAgenda, error) {
	// Fetch event details
	var event Event
	err := m.db.QueryRow(`
		SELECT id, title, description, start_time, end_time, location, event_type, metadata
		FROM events
		WHERE id = $1 AND user_id = $2 AND status = 'active'
	`, eventID, userID).Scan(
		&event.ID, &event.Title, &event.Description,
		&event.StartTime, &event.EndTime, &event.Location,
		&event.EventType, &event.Metadata,
	)
	if err != nil {
		return nil, fmt.Errorf("event not found: %v", err)
	}

	// Calculate duration
	duration := event.EndTime.Sub(event.StartTime)
	durationMinutes := int(duration.Minutes())

	// Generate agenda based on meeting type and duration
	agenda := &MeetingAgenda{
		EventID:     eventID,
		Title:       event.Title,
		Duration:    fmt.Sprintf("%d minutes", durationMinutes),
		GeneratedAt: time.Now(),
	}

	// Set objective based on title and description
	agenda.Objective = m.generateObjective(event.Title, event.Description)

	// Generate agenda items based on meeting type and duration
	agenda.AgendaItems = m.generateAgendaItems(event.EventType, durationMinutes, event.Title)

	// Add pre-work suggestions
	agenda.PreWork = m.suggestPreWork(event.EventType, event.Title)

	// Fetch attendees if available
	agenda.Attendees = m.fetchAttendees(eventID)

	return agenda, nil
}

// generateObjective creates a meeting objective
func (m *MeetingPrepManager) generateObjective(title, description string) string {
	if description != "" {
		return description
	}

	// Generate objective based on title keywords
	titleLower := strings.ToLower(title)
	switch {
	case strings.Contains(titleLower, "standup") || strings.Contains(titleLower, "daily"):
		return "Share progress updates, identify blockers, and align on priorities for the day"
	case strings.Contains(titleLower, "review"):
		return "Review progress, gather feedback, and identify next steps"
	case strings.Contains(titleLower, "planning"):
		return "Define goals, create action plans, and assign responsibilities"
	case strings.Contains(titleLower, "retrospective"):
		return "Reflect on what went well, what could be improved, and create action items"
	case strings.Contains(titleLower, "1-on-1") || strings.Contains(titleLower, "one-on-one"):
		return "Discuss progress, provide feedback, and address any concerns or questions"
	case strings.Contains(titleLower, "kickoff"):
		return "Align on project goals, timeline, roles, and success criteria"
	case strings.Contains(titleLower, "brainstorm"):
		return "Generate ideas, explore solutions, and identify promising approaches"
	default:
		return "Discuss agenda items and reach decisions on key topics"
	}
}

// generateAgendaItems creates agenda items based on meeting type and duration
func (m *MeetingPrepManager) generateAgendaItems(eventType string, durationMinutes int, title string) []AgendaItem {
	items := []AgendaItem{}

	// Always start with intro/welcome
	items = append(items, AgendaItem{
		Topic:       "Welcome & Agenda Review",
		Duration:    5,
		Description: "Brief introduction and review of meeting objectives",
	})

	titleLower := strings.ToLower(title)
	remainingTime := durationMinutes - 10 // Account for intro and wrap-up

	// Generate items based on meeting type
	switch {
	case strings.Contains(titleLower, "standup"):
		itemTime := remainingTime / 3
		items = append(items,
			AgendaItem{
				Topic:       "Yesterday's Progress",
				Duration:    itemTime,
				Description: "What was accomplished yesterday",
			},
			AgendaItem{
				Topic:       "Today's Plan",
				Duration:    itemTime,
				Description: "What will be worked on today",
			},
			AgendaItem{
				Topic:       "Blockers & Help Needed",
				Duration:    remainingTime - (itemTime * 2),
				Description: "Any obstacles or assistance required",
			},
		)
	case strings.Contains(titleLower, "review"):
		items = append(items,
			AgendaItem{
				Topic:       "Progress Overview",
				Duration:    remainingTime / 3,
				Description: "Review of work completed and milestones achieved",
			},
			AgendaItem{
				Topic:       "Feedback & Discussion",
				Duration:    remainingTime / 3,
				Description: "Gather input and discuss improvements",
			},
			AgendaItem{
				Topic:       "Next Steps",
				Duration:    remainingTime / 3,
				Description: "Define action items and timeline",
			},
		)
	case strings.Contains(titleLower, "planning"):
		items = append(items,
			AgendaItem{
				Topic:       "Goals & Objectives",
				Duration:    remainingTime / 4,
				Description: "Define what we want to achieve",
			},
			AgendaItem{
				Topic:       "Strategy & Approach",
				Duration:    remainingTime / 3,
				Description: "How we will achieve our goals",
			},
			AgendaItem{
				Topic:       "Timeline & Milestones",
				Duration:    remainingTime / 4,
				Description: "Key dates and deliverables",
			},
			AgendaItem{
				Topic:       "Roles & Responsibilities",
				Duration:    remainingTime - (remainingTime/4 + remainingTime/3 + remainingTime/4),
				Description: "Who will do what",
			},
		)
	default:
		// Generic agenda for unspecified meeting types
		numItems := 3
		if durationMinutes > 60 {
			numItems = 4
		}
		itemTime := remainingTime / numItems

		for i := 1; i <= numItems; i++ {
			items = append(items, AgendaItem{
				Topic:       fmt.Sprintf("Discussion Item %d", i),
				Duration:    itemTime,
				Description: "Key topic for discussion",
			})
		}
	}

	// Always end with wrap-up
	items = append(items, AgendaItem{
		Topic:       "Summary & Action Items",
		Duration:    5,
		Description: "Recap decisions and next steps",
	})

	return items
}

// suggestPreWork generates pre-work suggestions
func (m *MeetingPrepManager) suggestPreWork(eventType, title string) []string {
	suggestions := []string{}
	titleLower := strings.ToLower(title)

	switch {
	case strings.Contains(titleLower, "review"):
		suggestions = append(suggestions,
			"Review relevant documents and reports",
			"Prepare questions and feedback",
			"Gather metrics and data points",
		)
	case strings.Contains(titleLower, "planning"):
		suggestions = append(suggestions,
			"Review previous plans and outcomes",
			"Research industry best practices",
			"Prepare initial ideas and proposals",
		)
	case strings.Contains(titleLower, "brainstorm"):
		suggestions = append(suggestions,
			"Research the problem space",
			"Gather inspiration and examples",
			"Prepare initial ideas to seed discussion",
		)
	case strings.Contains(titleLower, "1-on-1"):
		suggestions = append(suggestions,
			"Reflect on recent accomplishments",
			"Identify challenges and questions",
			"Consider career goals and development areas",
		)
	default:
		suggestions = append(suggestions,
			"Review meeting invite and objectives",
			"Prepare any questions or topics to discuss",
		)
	}

	return suggestions
}

// fetchAttendees gets attendees for an event
func (m *MeetingPrepManager) fetchAttendees(eventID string) []string {
	attendees := []string{}

	rows, err := m.db.Query(`
		SELECT attendee_email 
		FROM event_attendees 
		WHERE event_id = $1
	`, eventID)
	if err != nil {
		return attendees
	}
	defer rows.Close()

	for rows.Next() {
		var email string
		if err := rows.Scan(&email); err == nil {
			attendees = append(attendees, email)
		}
	}

	return attendees
}

// generateAgendaHandler handles agenda generation requests
func generateAgendaHandler(w http.ResponseWriter, r *http.Request) {
	if prepManager == nil {
		http.Error(w, "Meeting preparation not configured", http.StatusServiceUnavailable)
		return
	}

	vars := mux.Vars(r)
	eventID := vars["id"]

	userID := r.Context().Value("user_id").(string)

	agenda, err := prepManager.GenerateAgenda(eventID, userID)
	if err != nil {
		errorHandler.HandleError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agenda)
}

// updateAgendaHandler handles agenda updates
func updateAgendaHandler(w http.ResponseWriter, r *http.Request) {
	if prepManager == nil {
		http.Error(w, "Meeting preparation not configured", http.StatusServiceUnavailable)
		return
	}

	vars := mux.Vars(r)
	eventID := vars["id"]
	userID := r.Context().Value("user_id").(string)

	var agenda MeetingAgenda
	if err := json.NewDecoder(r.Body).Decode(&agenda); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Store agenda in event metadata
	agendaJSON, err := json.Marshal(agenda)
	if err != nil {
		http.Error(w, "Failed to process agenda", http.StatusInternalServerError)
		return
	}

	_, err = db.Exec(`
		UPDATE events 
		SET metadata = jsonb_set(
			COALESCE(metadata, '{}'), 
			'{agenda}', 
			$1::jsonb
		)
		WHERE id = $2 AND user_id = $3
	`, string(agendaJSON), eventID, userID)

	if err != nil {
		errorHandler.HandleError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Agenda updated successfully",
	})
}
