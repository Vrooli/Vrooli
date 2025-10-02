package main

import (
	"encoding/json"
	"time"
)

// EventType represents the type of event being broadcast
type EventType string

const (
	// Issue events
	EventIssueCreated       EventType = "issue.created"
	EventIssueUpdated       EventType = "issue.updated"
	EventIssueStatusChanged EventType = "issue.status_changed"
	EventIssueDeleted       EventType = "issue.deleted"

	// Agent events
	EventAgentStarted   EventType = "agent.started"
	EventAgentCompleted EventType = "agent.completed"
	EventAgentFailed    EventType = "agent.failed"

	// Processor events
	EventProcessorStateChanged EventType = "processor.state_changed"

	// Rate limit events
	EventRateLimitChanged EventType = "rate_limit.changed"
)

// Event represents a domain event to be broadcast to clients
type Event struct {
	Type      EventType   `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// IssueEventData contains issue-related event data
type IssueEventData struct {
	Issue *Issue `json:"issue"`
}

// IssueStatusChangedData contains data for status change events
type IssueStatusChangedData struct {
	IssueID   string `json:"issue_id"`
	OldStatus string `json:"old_status"`
	NewStatus string `json:"new_status"`
}

// IssueDeletedData contains data for deletion events
type IssueDeletedData struct {
	IssueID string `json:"issue_id"`
}

// AgentStartedData contains data for agent start events
type AgentStartedData struct {
	IssueID   string    `json:"issue_id"`
	AgentID   string    `json:"agent_id"`
	StartTime time.Time `json:"start_time"`
}

// AgentCompletedData contains data for agent completion events
type AgentCompletedData struct {
	IssueID   string    `json:"issue_id"`
	AgentID   string    `json:"agent_id"`
	Success   bool      `json:"success"`
	EndTime   time.Time `json:"end_time"`
	NewStatus string    `json:"new_status,omitempty"`
}

// ProcessorStateData contains processor state information
type ProcessorStateData struct {
	Active           bool `json:"active"`
	ConcurrentSlots  int  `json:"concurrent_slots"`
	RefreshInterval  int  `json:"refresh_interval"`
	MaxIssues        int  `json:"max_issues"`
	IssuesProcessed  int  `json:"issues_processed"`
	IssuesRemaining  *int `json:"issues_remaining,omitempty"`
}

// RateLimitData contains rate limit information
type RateLimitData struct {
	RateLimited       bool   `json:"rate_limited"`
	RateLimitedCount  int    `json:"rate_limited_count"`
	ResetTime         string `json:"reset_time,omitempty"`
	SecondsUntilReset int    `json:"seconds_until_reset"`
	RateLimitAgent    string `json:"rate_limit_agent,omitempty"`
}

// NewEvent creates a new event with the current timestamp
func NewEvent(eventType EventType, data interface{}) Event {
	return Event{
		Type:      eventType,
		Timestamp: time.Now(),
		Data:      data,
	}
}

// ToJSON serializes the event to JSON
func (e Event) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}
