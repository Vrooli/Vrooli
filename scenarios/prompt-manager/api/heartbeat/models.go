// Package heartbeat provides cron-based autonomous execution for team members.
//
// DOC: docs/concepts/HEARTBEATS.md
// DOC: docs/reference/heartbeat-api.md
package heartbeat

// HeartbeatConfigResponse is the API response for a heartbeat configuration
type HeartbeatConfigResponse struct {
	TeamID        string                  `json:"teamId"`
	AgentID       string                  `json:"agentId"`
	Enabled       bool                    `json:"enabled"`
	Schedule      string                  `json:"schedule"`
	ProfileKey    string                  `json:"profileKey,omitempty"`
	LastExecution *HeartbeatExecResultDTO `json:"lastExecution,omitempty"`
	NextExecution string                  `json:"nextExecution,omitempty"`
	CreatedAt     string                  `json:"createdAt"`
	UpdatedAt     string                  `json:"updatedAt"`
}

// HeartbeatExecResultDTO represents execution result in API responses
type HeartbeatExecResultDTO struct {
	StartedAt string `json:"startedAt"`
	EndedAt   string `json:"endedAt,omitempty"`
	Status    string `json:"status"`
	RunID     string `json:"runId,omitempty"`
	LogPath   string `json:"logPath,omitempty"`
	Error     string `json:"error,omitempty"`
}

// CreateHeartbeatRequest is the request body for creating a heartbeat config
type CreateHeartbeatRequest struct {
	Schedule   string `json:"schedule"`             // Cron expression (required)
	ProfileKey string `json:"profileKey,omitempty"` // Optional profile key override
	Enabled    *bool  `json:"enabled,omitempty"`    // Defaults to false
}

// UpdateHeartbeatRequest is the request body for updating a heartbeat config
type UpdateHeartbeatRequest struct {
	Schedule   *string `json:"schedule,omitempty"`
	ProfileKey *string `json:"profileKey,omitempty"`
	Enabled    *bool   `json:"enabled,omitempty"`
}

// TriggerHeartbeatRequest is the request body for manually triggering a heartbeat
type TriggerHeartbeatRequest struct {
	// No fields needed for now - could add prompt override later
}

// TriggerHeartbeatResponse is the response for manual trigger
type TriggerHeartbeatResponse struct {
	TeamID  string `json:"teamId"`
	AgentID string `json:"agentId"`
	RunID   string `json:"runId"`
	Status  string `json:"status"`
	LogPath string `json:"logPath,omitempty"`
}

// LogEntry represents a log file entry in list responses
type LogEntry struct {
	Filename  string `json:"filename"`
	Timestamp string `json:"timestamp"`
	Status    string `json:"status,omitempty"`
}

// LogListResponse is the response for listing heartbeat logs
type LogListResponse struct {
	TeamID  string     `json:"teamId"`
	AgentID string     `json:"agentId"`
	Logs    []LogEntry `json:"logs"`
}

// LogContentResponse is the response for getting log content
type LogContentResponse struct {
	TeamID   string `json:"teamId"`
	AgentID  string `json:"agentId"`
	Filename string `json:"filename"`
	Content  string `json:"content"`
}

// MemberDocRequest is the request body for setting member documents
type MemberDocRequest struct {
	Content string `json:"content"`
}

// MemberDocResponse is the response for member document operations
type MemberDocResponse struct {
	TeamID  string `json:"teamId"`
	AgentID string `json:"agentId"`
	Content string `json:"content"`
}
