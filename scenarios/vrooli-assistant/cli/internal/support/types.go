package support

import (
	"encoding/json"
	"time"
)

// Issue mirrors the API shape returned by /api/v1/assistant/issues/{id} and
// /api/v1/assistant/history.
type Issue struct {
	ID              string                 `json:"id"`
	Timestamp       time.Time              `json:"timestamp"`
	ScreenshotPath  string                 `json:"screenshot_path,omitempty"`
	ScenarioName    string                 `json:"scenario_name,omitempty"`
	URL             string                 `json:"url,omitempty"`
	Description     string                 `json:"description"`
	ContextData     map[string]interface{} `json:"context_data,omitempty"`
	AgentSessionID  string                 `json:"agent_session_id,omitempty"`
	Status          string                 `json:"status"`
	ResolutionNotes string                 `json:"resolution_notes,omitempty"`
	Tags            []string               `json:"tags,omitempty"`
}

// HistoryResponse mirrors /api/v1/assistant/history. It includes the list of
// recent issues and a count field.
type HistoryResponse struct {
	Issues []Issue `json:"issues"`
	Count  int     `json:"count"`
}

// StatsResponse mirrors /api/v1/assistant/status. The name "stats" reflects the
// scenario-specific counts so it does not collide with cli-core's root status
// command.
type StatsResponse struct {
	Status         string          `json:"status"`
	IssuesCaptured int             `json:"issues_captured"`
	AgentsSpawned  int             `json:"agents_spawned"`
	Uptime         string          `json:"uptime"`
	Extra          json.RawMessage `json:"-"`
}

// CaptureResponse mirrors the envelope returned by POST /api/v1/assistant/capture.
type CaptureResponse struct {
	IssueID string `json:"issue_id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// SpawnResponse mirrors the envelope returned by POST /api/v1/assistant/spawn-agent.
type SpawnResponse struct {
	SessionID string `json:"session_id"`
	Status    string `json:"status"`
	Message   string `json:"message"`
}
