package main

import "encoding/json"

type healthResponse struct {
	Status     string            `json:"status"`
	Service    string            `json:"service"`
	Version    string            `json:"version"`
	Readiness  bool              `json:"readiness"`
	Timestamp  string            `json:"timestamp"`
	Deps       map[string]string `json:"dependencies"`
	Error      string            `json:"error,omitempty"`
	Message    string            `json:"message,omitempty"`
	Operations map[string]any    `json:"operations,omitempty"`
}

// BacklogItem represents a tracked unit of work for the swarm.
type BacklogItem struct {
	Name           string   `json:"name"`
	Title          string   `json:"title"`
	Description    string   `json:"description"`
	Status         string   `json:"status"`
	Priority       int      `json:"priority"`
	Tags           []string `json:"tags"`
	Created        string   `json:"created"`
	Updated        string   `json:"updated"`
	Kind           string   `json:"kind"`
	ResearchTarget string   `json:"research_target,omitempty"`
}

type BacklogItemResponse struct {
	Item BacklogItem `json:"item"`
}

type ListBacklogResponse struct {
	Items []BacklogItem `json:"items"`
}

type CreateBacklogRequest struct {
	Name           string   `json:"name"`
	Title          string   `json:"title"`
	Description    string   `json:"description,omitempty"`
	Priority       int      `json:"priority,omitempty"`
	Tags           []string `json:"tags,omitempty"`
	Kind           string   `json:"kind"`
	ResearchTarget string   `json:"researchTarget,omitempty"`
}

type BacklogFile struct {
	Name     string        `json:"name"`
	Path     string        `json:"path"`
	Type     string        `json:"type"`
	Size     int64         `json:"size,omitempty"`
	Children []BacklogFile `json:"children,omitempty"`
}

type BacklogFilesResponse struct {
	Files []BacklogFile `json:"files"`
}

type BacklogFileResponse struct {
	File BacklogFile `json:"file"`
}

type QueueBacklogResponse struct {
	Item    BacklogItem `json:"item"`
	TaskID  string      `json:"task_id"`
	RunID   string      `json:"run_id"`
	BaseURL string      `json:"base_url"`
	Created string      `json:"created"`
}

type ResearchResponse struct {
	TaskID  string `json:"task_id"`
	RunID   string `json:"run_id"`
	BaseURL string `json:"base_url"`
	Created string `json:"created"`
}

// Scenario represents a scenario entry in the catalog.
type Scenario struct {
	Name              string   `json:"name"`
	DisplayName       string   `json:"display_name"`
	Description       string   `json:"description"`
	Status            string   `json:"status"`
	Priority          int      `json:"priority"`
	CompletenessScore *int     `json:"completeness_score,omitempty"`
	IsGreenfield      bool     `json:"is_greenfield"`
	Tags              []string `json:"tags"`
}

type ScenarioResponse struct {
	Scenario Scenario `json:"scenario"`
}

type ListScenariosResponse struct {
	Scenarios []Scenario `json:"scenarios"`
}

type ScenarioFile struct {
	Name     string         `json:"name"`
	Path     string         `json:"path"`
	Type     string         `json:"type"`
	Size     int64          `json:"size,omitempty"`
	Children []ScenarioFile `json:"children,omitempty"`
}

type ScenarioFilesResponse struct {
	Files []ScenarioFile `json:"files"`
}

type DeleteScenarioResponse struct {
	Name     string `json:"name"`
	Archived bool   `json:"archived"`
	Message  string `json:"message"`
}

// QueueItem represents a local queue entry.
type QueueItem struct {
	ID      string          `json:"id"`
	Kind    string          `json:"kind"`
	Payload json.RawMessage `json:"payload,omitempty"`
	Created string          `json:"created"`
}

type QueueListResponse struct {
	Items []QueueItem `json:"items"`
}

type ExecutionRecord struct {
	ExecutionID   string `json:"execution_id"`
	BacklogKind   string `json:"backlog_kind"`
	BacklogName   string `json:"backlog_name"`
	TaskID        string `json:"task_id"`
	RunID         string `json:"run_id"`
	Status        string `json:"status"`
	Mode          string `json:"mode"`
	ScheduledAt   string `json:"scheduled_at"`
	StartedAt     string `json:"started_at"`
	FinishedAt    string `json:"finished_at"`
	FailureReason string `json:"failure_reason"`
	StartedBy     string `json:"started_by"`
	Operation     string `json:"operation"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

type ExecutionListResponse struct {
	Items []ExecutionRecord `json:"items"`
}

type ExecutionItemResponse struct {
	Execution ExecutionRecord `json:"execution"`
}

type ExecutionPolicy struct {
	DefaultMode         string `json:"default_mode"`
	DefaultDelaySeconds int64  `json:"default_delay_seconds"`
}

type ExecutionPolicyResponse struct {
	Policy ExecutionPolicy `json:"policy"`
}

type AgentManagerStatusResponse struct {
	Enabled   bool    `json:"enabled"`
	Available bool    `json:"available"`
	URL       *string `json:"url,omitempty"`
	ProfileID *string `json:"profile_id,omitempty"`
}
