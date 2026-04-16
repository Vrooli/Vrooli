package support

import (
	"encoding/json"
	"time"
)

// App mirrors the repository row returned by /api/apps.
type App struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	DisplayName    string     `json:"display_name"`
	Type           string     `json:"type,omitempty"`
	TotalTasks     int        `json:"total_tasks"`
	CompletedTasks int        `json:"completed_tasks"`
	CreatedAt      *time.Time `json:"created_at,omitempty"`
	UpdatedAt      *time.Time `json:"updated_at,omitempty"`
}

// AppsResponse is the envelope returned by /api/apps.
type AppsResponse struct {
	Apps  []App `json:"apps"`
	Count int   `json:"count"`
}

// Task mirrors the row returned by /api/tasks.
type Task struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	Status      string     `json:"status"`
	Priority    string     `json:"priority"`
	Tags        []string   `json:"tags,omitempty"`
	AppID       string     `json:"app_id"`
	AppName     string     `json:"app_name,omitempty"`
	CreatedAt   *time.Time `json:"created_at,omitempty"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
}

// TasksResponse is the envelope returned by /api/tasks.
type TasksResponse struct {
	Tasks []Task `json:"tasks"`
	Count int    `json:"count"`
}

// ParseTextResponse is the envelope returned by /api/parse-text.
type ParseTextResponse struct {
	Success      bool   `json:"success"`
	SessionID    string `json:"session_id,omitempty"`
	TasksCreated int    `json:"tasks_created"`
	Tasks        []Task `json:"tasks,omitempty"`
	Error        string `json:"error,omitempty"`
}

// ResearchResponse is the envelope returned by /api/tasks/{id}/research.
type ResearchResponse struct {
	Success         bool                   `json:"success"`
	TaskID          string                 `json:"task_id,omitempty"`
	ResearchSummary string                 `json:"research_summary,omitempty"`
	Requirements    []string               `json:"requirements,omitempty"`
	Dependencies    []string               `json:"dependencies,omitempty"`
	Recommendations []string               `json:"recommendations,omitempty"`
	EstimatedHours  float64                `json:"estimated_hours,omitempty"`
	Complexity      string                 `json:"complexity,omitempty"`
	ResearchData    map[string]interface{} `json:"research_data,omitempty"`
	Error           string                 `json:"error,omitempty"`
}

// StatusChange is one entry in a task's status history.
type StatusChange struct {
	TaskID     string     `json:"task_id,omitempty"`
	FromStatus string     `json:"from_status,omitempty"`
	ToStatus   string     `json:"to_status,omitempty"`
	Reason     string     `json:"reason,omitempty"`
	Notes      string     `json:"notes,omitempty"`
	ChangedBy  string     `json:"changed_by,omitempty"`
	ChangedAt  *time.Time `json:"changed_at,omitempty"`
}

// StatusHistoryResponse is the envelope returned by /api/tasks/status-history.
type StatusHistoryResponse struct {
	TaskID  string         `json:"task_id"`
	History []StatusChange `json:"history"`
}

// StatusUpdateResponse is the envelope returned by PUT /api/tasks/status.
type StatusUpdateResponse struct {
	Success        bool            `json:"success"`
	TaskID         string          `json:"task_id,omitempty"`
	StatusChanged  bool            `json:"status_changed,omitempty"`
	PreviousStatus string          `json:"previous_status,omitempty"`
	NewStatus      string          `json:"new_status,omitempty"`
	StatusHistory  []StatusChange  `json:"status_history,omitempty"`
	NextActions    []string        `json:"next_actions,omitempty"`
	Error          string          `json:"error,omitempty"`
	Raw            json.RawMessage `json:"-"`
}
