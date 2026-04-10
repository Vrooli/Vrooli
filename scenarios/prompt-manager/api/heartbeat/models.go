// Package heartbeat provides cron-based autonomous execution for team members.
//
// DOC: docs/concepts/HEARTBEATS.md
// DOC: docs/reference/heartbeat-api.md
package heartbeat

import "prompt-manager/store"

// HeartbeatConfigResponse is the API response for a heartbeat configuration
type HeartbeatConfigResponse struct {
	TeamID         string                  `json:"teamId"`
	AgentID        string                  `json:"agentId"`
	Enabled        bool                    `json:"enabled"`
	Schedule       string                  `json:"schedule"`
	ProfileKey     string                  `json:"profileKey,omitempty"`
	TimeoutSeconds int                     `json:"timeoutSeconds,omitempty"`
	LastExecution  *HeartbeatExecResultDTO `json:"lastExecution,omitempty"`
	NextExecution  string                  `json:"nextExecution,omitempty"`
	NextExecutions []string                `json:"nextExecutions,omitempty"`
	CreatedAt      string                  `json:"createdAt"`
	UpdatedAt      string                  `json:"updatedAt"`
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
	Schedule       string `json:"schedule"`                 // Cron expression (required)
	ProfileKey     string `json:"profileKey,omitempty"`     // Optional profile key override
	Enabled        *bool  `json:"enabled,omitempty"`        // Defaults to false
	TimeoutSeconds int    `json:"timeoutSeconds,omitempty"` // 0 = use default (45 min)
}

// UpdateHeartbeatRequest is the request body for updating a heartbeat config
type UpdateHeartbeatRequest struct {
	Schedule       *string `json:"schedule,omitempty"`
	ProfileKey     *string `json:"profileKey,omitempty"`
	Enabled        *bool   `json:"enabled,omitempty"`
	TimeoutSeconds *int    `json:"timeoutSeconds,omitempty"`
}

// TriggerHeartbeatRequest is the request body for manually triggering a heartbeat
type TriggerHeartbeatRequest struct {
	// No fields needed for now - could add prompt override later
}

// TriggerHeartbeatResponse is the response for manual trigger
type TriggerHeartbeatResponse struct {
	TeamID   string `json:"teamId"`
	AgentID  string `json:"agentId"`
	RunID    string `json:"runId"`
	Status   string `json:"status"`
	Position int    `json:"position,omitempty"`
	LogPath  string `json:"logPath,omitempty"`
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

// TeamLogEntry represents a log entry aggregated across team members
type TeamLogEntry struct {
	AgentID          string `json:"agentId"`
	AgentDisplayName string `json:"agentDisplayName"`
	Filename         string `json:"filename"`
	Timestamp        string `json:"timestamp"`
	Status           string `json:"status,omitempty"`
}

// TeamLogListResponse is the response for listing team-wide heartbeat logs
type TeamLogListResponse struct {
	TeamID  string         `json:"teamId"`
	Logs    []TeamLogEntry `json:"logs"`
	Total   int            `json:"total"`
	HasMore bool           `json:"hasMore"`
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

// RunningAgentEntry represents a running agent in the list response.
type RunningAgentEntry struct {
	TeamID    string `json:"teamId"`
	AgentID   string `json:"agentId"`
	AgentName string `json:"agentName"`
	TeamName  string `json:"teamName"`
	RunID     string `json:"runId"`
	StartedAt string `json:"startedAt"`
	Duration  string `json:"duration"`
}

// RunningAgentsResponse is the API response for listing running agents.
type RunningAgentsResponse struct {
	Count  int                 `json:"count"`
	Agents []RunningAgentEntry `json:"agents"`
}

// StopAgentResponse is the API response for stopping a running agent.
type StopAgentResponse struct {
	TeamID  string `json:"teamId"`
	AgentID string `json:"agentId"`
	RunID   string `json:"runId"`
	Status  string `json:"status"`
}

// TriggerTeamResponse is the response for team-level trigger.
type TriggerTeamResponse struct {
	TeamID              string                     `json:"teamId"`
	RuntimeMode         string                     `json:"runtimeMode"`
	CoordinationPattern string                     `json:"coordinationPattern"`
	QueuePolicy         string                     `json:"queuePolicy"`
	Triggers            []TriggerHeartbeatResponse `json:"triggers"`
}

// --- Handoff API models ---

// HandoffResponse is the API response for a single handoff.
type HandoffResponse struct {
	TeamID  string `json:"teamId"`
	AgentID string `json:"agentId"`
	Content string `json:"content"`
}

// HandoffHistoryResponse is the API response for handoff history.
type HandoffHistoryResponse struct {
	TeamID  string               `json:"teamId"`
	Entries []store.HandoffEntry `json:"entries"`
}

// --- Task Board API models ---

// AddTaskRequest is the request body for adding a task.
type AddTaskRequest struct {
	Title    string `json:"title"`
	Assignee string `json:"assignee,omitempty"`
	Priority string `json:"priority,omitempty"`
	From     string `json:"from"` // createdBy agent ID
}

// UpdateTaskRequest is the request body for updating a task.
type UpdateTaskRequest struct {
	Title    *string `json:"title,omitempty"`
	Status   *string `json:"status,omitempty"`
	Assignee *string `json:"assignee,omitempty"`
	Priority *string `json:"priority,omitempty"`
	Note     *string `json:"note,omitempty"` // appends a TaskNote
}

// TaskBoardResponse is the API response for the task board.
type TaskBoardResponse struct {
	TeamID string           `json:"teamId"`
	Tasks  []store.TeamTask `json:"tasks"`
	Total  int              `json:"total"`
	Limit  int              `json:"limit"`
	Offset int              `json:"offset"`
}

// --- Decision API models ---

// AddDecisionRequest is the request body for adding a decision.
type AddDecisionRequest struct {
	By          string                 `json:"by"`
	Decision    string                 `json:"decision"`
	Rationale   string                 `json:"rationale"`
	Context     string                 `json:"context,omitempty"`
	Supersedes  string                 `json:"supersedes,omitempty"`
	Topic       string                 `json:"topic,omitempty"`
	Description string                 `json:"description,omitempty"`
	Options     []store.DecisionOption `json:"options,omitempty"`
}

// UpdateDecisionRequest is the request body for updating a decision.
type UpdateDecisionRequest struct {
	Decision    *string                 `json:"decision,omitempty"`
	Rationale   *string                 `json:"rationale,omitempty"`
	Context     *string                 `json:"context,omitempty"`
	Status      *string                 `json:"status,omitempty"` // "pending", "accepted", "rejected"
	Supersedes  *string                 `json:"supersedes,omitempty"`
	Topic       *string                 `json:"topic,omitempty"`
	Description *string                 `json:"description,omitempty"`
	Options     *[]store.DecisionOption `json:"options,omitempty"`
	Selected    *string                 `json:"selected,omitempty"`
	Freeform    *string                 `json:"freeform,omitempty"`
	Notes       *string                 `json:"notes,omitempty"`
}

// --- Knowledge API models ---

// AddKnowledgeRequest is the request body for adding a knowledge entry.
type AddKnowledgeRequest struct {
	By         string `json:"by"`
	Topic      string `json:"topic"`
	Content    string `json:"content"`
	Source     string `json:"source,omitempty"`
	Supersedes string `json:"supersedes,omitempty"`
}

// UpdateKnowledgeRequest is the request body for updating a knowledge entry.
type UpdateKnowledgeRequest struct {
	Topic      *string `json:"topic,omitempty"`
	Content    *string `json:"content,omitempty"`
	Source     *string `json:"source,omitempty"`
	Supersedes *string `json:"supersedes,omitempty"`
}

// KnowledgeListResponse is the API response for listing knowledge entries.
type KnowledgeListResponse struct {
	TeamID  string                 `json:"teamId"`
	Entries []store.KnowledgeEntry `json:"entries"`
}

// DecisionListResponse is the API response for listing decisions.
type DecisionListResponse struct {
	TeamID  string                `json:"teamId"`
	Entries []store.DecisionEntry `json:"entries"`
	Total   int                   `json:"total"`
	Last    int                   `json:"last"`
}

// PendingDecisionTeamGroup groups pending decisions by team.
type PendingDecisionTeamGroup struct {
	TeamID   string                `json:"teamId"`
	TeamName string                `json:"teamName"`
	Entries  []store.DecisionEntry `json:"entries"`
}

// AllPendingDecisionsResponse is the response for the aggregate pending decisions endpoint.
type AllPendingDecisionsResponse struct {
	Teams      []PendingDecisionTeamGroup `json:"teams"`
	TotalCount int                        `json:"totalCount"`
}

// MemberContextResponse is the response for the member context endpoint.
type MemberContextResponse struct {
	TeamID  string `json:"teamId"`
	AgentID string `json:"agentId"`
	Prompt  string `json:"prompt"`
}

// PromptPreviewRequest is the request body for previewing a built prompt.
type PromptPreviewRequest struct {
	AgentID string `json:"agentId"`
	TeamID  string `json:"teamId,omitempty"`
}

// PromptPreviewResponse returns the assembled prompt text.
type PromptPreviewResponse struct {
	AgentID string `json:"agentId"`
	TeamID  string `json:"teamId,omitempty"`
	Prompt  string `json:"prompt"`
}

// PromptSection represents a single logical section of an assembled prompt.
type PromptSection struct {
	Kind       string `json:"kind"`
	Label      string `json:"label"`
	SourcePath string `json:"sourcePath,omitempty"`
	Content    string `json:"content"`
}

// StructuredPromptPreviewResponse returns the assembled prompt as individual sections.
type StructuredPromptPreviewResponse struct {
	AgentID  string          `json:"agentId"`
	TeamID   string          `json:"teamId,omitempty"`
	Sections []PromptSection `json:"sections"`
}

// TeamPromptMatrixEntry holds the structured prompt for a single team member.
type TeamPromptMatrixEntry struct {
	AgentID     string          `json:"agentId"`
	DisplayName string          `json:"displayName"`
	Sections    []PromptSection `json:"sections"`
	Error       string          `json:"error,omitempty"`
}

// TeamPromptMatrixResponse is the API response for the team prompt matrix endpoint.
type TeamPromptMatrixResponse struct {
	TeamID  string                  `json:"teamId"`
	Entries []TeamPromptMatrixEntry `json:"entries"`
}
