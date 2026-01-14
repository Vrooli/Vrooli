package dto

import "time"

// TaskCreateRequest represents the fields required to create a new task.
// This is a subset of TaskItem containing only user-provided fields.
type TaskCreateRequest struct {
	Title            string   `json:"title" validate:"required,min=1,max=200"`
	Type             string   `json:"type" validate:"required,oneof=resource scenario"`
	Operation        string   `json:"operation" validate:"required,oneof=generator improver"`
	Target           string   `json:"target,omitempty"`
	Targets          []string `json:"targets,omitempty"`
	Category         string   `json:"category,omitempty"`
	Priority         string   `json:"priority,omitempty" validate:"omitempty,oneof=low medium high critical"`
	EffortEstimate   string   `json:"effort_estimate,omitempty"`
	Urgency          string   `json:"urgency,omitempty"`
	Dependencies     []string `json:"dependencies,omitempty"`
	RelatedScenarios []string `json:"related_scenarios,omitempty"`
	RelatedResources []string `json:"related_resources,omitempty"`
	Tags             []string `json:"tags,omitempty"`
	Notes            string   `json:"notes,omitempty"`
	SteerMode        string   `json:"steer_mode,omitempty"`
	AutoSteerProfile string   `json:"auto_steer_profile_id,omitempty"`
}

// TaskUpdateRequest represents fields that can be updated on an existing task.
type TaskUpdateRequest struct {
	Title            *string   `json:"title,omitempty"`
	Priority         *string   `json:"priority,omitempty"`
	Status           *string   `json:"status,omitempty"`
	Target           *string   `json:"target,omitempty"`
	Targets          *[]string `json:"targets,omitempty"`
	Tags             *[]string `json:"tags,omitempty"`
	Notes            *string   `json:"notes,omitempty"`
	SteerMode        *string   `json:"steer_mode,omitempty"`
	AutoSteerProfile *string   `json:"auto_steer_profile_id,omitempty"`
}

// TaskResponse represents the persisted state of a task.
// It excludes ephemeral/runtime fields that are only relevant during processing.
type TaskResponse struct {
	ID                   string         `json:"id"`
	Title                string         `json:"title"`
	Type                 string         `json:"type"`
	Operation            string         `json:"operation"`
	Target               string         `json:"target,omitempty"`
	Targets              []string       `json:"targets,omitempty"`
	Category             string         `json:"category"`
	Priority             string         `json:"priority"`
	EffortEstimate       string         `json:"effort_estimate,omitempty"`
	Urgency              string         `json:"urgency,omitempty"`
	Dependencies         []string       `json:"dependencies,omitempty"`
	Blocks               []string       `json:"blocks,omitempty"`
	RelatedScenarios     []string       `json:"related_scenarios,omitempty"`
	RelatedResources     []string       `json:"related_resources,omitempty"`
	Status               string         `json:"status"`
	CurrentPhase         string         `json:"current_phase,omitempty"`
	StartedAt            string         `json:"started_at,omitempty"`
	CompletedAt          string         `json:"completed_at,omitempty"`
	CooldownUntil        string         `json:"cooldown_until,omitempty"`
	CompletionCount      int            `json:"completion_count"`
	LastCompletedAt      string         `json:"last_completed_at,omitempty"`
	ValidationCriteria   []string       `json:"validation_criteria,omitempty"`
	CreatedBy            string         `json:"created_by,omitempty"`
	CreatedAt            string         `json:"created_at,omitempty"`
	UpdatedAt            string         `json:"updated_at,omitempty"`
	Tags                 []string       `json:"tags,omitempty"`
	Notes                string         `json:"notes,omitempty"`
	Results              map[string]any `json:"results,omitempty"`
	ConsecutiveFailures  int            `json:"consecutive_failures,omitempty"`
	SteerMode            string         `json:"steer_mode,omitempty"`
	AutoSteerProfileID   string         `json:"auto_steer_profile_id,omitempty"`
}

// ProcessInfo contains information about a running process.
type ProcessInfo struct {
	TaskID    string    `json:"task_id"`
	AgentTag  string    `json:"agent_tag,omitempty"`
	RunID     string    `json:"run_id,omitempty"`
	StartedAt time.Time `json:"started_at"`
	IsTimedOut bool     `json:"is_timed_out,omitempty"`
}

// TaskDetailResponse includes runtime information in addition to persisted state.
// This is used when fetching a single task with full details.
type TaskDetailResponse struct {
	TaskResponse
	CurrentProcess      *ProcessInfo `json:"current_process,omitempty"`
	AutoSteerPhaseIndex *int         `json:"auto_steer_phase_index,omitempty"`
}

// TaskListResponse represents a list of tasks with metadata.
type TaskListResponse struct {
	Tasks []TaskResponse `json:"tasks"`
	Count int            `json:"count"`
	Total int            `json:"total,omitempty"`
}

// TaskActionResponse represents the result of an action on a task.
type TaskActionResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	TaskID  string `json:"task_id,omitempty"`
	Error   string `json:"error,omitempty"`
}
