// Package task provides AI task/investigation management commands for the CLI.
package task

// CreateRequest is the request for creating a new task.
type CreateRequest struct {
	Prompt string `json:"prompt"`
	Type   string `json:"type,omitempty"` // investigate, diagnose, fix, analyze
}

// CreateResponse is the response from creating a task.
type CreateResponse struct {
	Task      TaskInfo `json:"task"`
	Timestamp string   `json:"timestamp"`
}

// TaskInfo represents an AI task/investigation.
type TaskInfo struct {
	ID           string `json:"id"`
	DeploymentID string `json:"deployment_id"`
	Type         string `json:"type"`
	Prompt       string `json:"prompt"`
	Status       string `json:"status"` // pending, running, completed, failed, stopped
	Progress     string `json:"progress,omitempty"`
	Result       string `json:"result,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
	CompletedAt  string `json:"completed_at,omitempty"`
}

// ListResponse is the response from listing tasks.
type ListResponse struct {
	Tasks     []TaskInfo `json:"tasks"`
	Timestamp string     `json:"timestamp"`
}

// GetResponse is the response from getting a task.
type GetResponse struct {
	Task      TaskInfo `json:"task"`
	Timestamp string   `json:"timestamp"`
}

// StopResponse is the response from stopping a task.
type StopResponse struct {
	Success   bool   `json:"success"`
	TaskID    string `json:"task_id"`
	Message   string `json:"message,omitempty"`
	Timestamp string `json:"timestamp"`
}

// AgentStatusResponse is the response from agent status check.
type AgentStatusResponse struct {
	Available    bool     `json:"available"`
	Model        string   `json:"model,omitempty"`
	Provider     string   `json:"provider,omitempty"`
	ActiveTasks  int      `json:"active_tasks"`
	QueuedTasks  int      `json:"queued_tasks"`
	Capabilities []string `json:"capabilities,omitempty"`
	Timestamp    string   `json:"timestamp"`
}
