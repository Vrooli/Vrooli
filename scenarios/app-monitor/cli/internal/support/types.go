package support

import (
	"encoding/json"
	"time"
)

// App mirrors the repository.App shape exposed via /api/v1/apps.
type App struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	ScenarioName string          `json:"scenario_name"`
	Status       string          `json:"status"`
	Type         string          `json:"type,omitempty"`
	URL          string          `json:"url,omitempty"`
	Port         json.RawMessage `json:"port,omitempty"`
	Uptime       string          `json:"uptime,omitempty"`
	MemoryUsage  float64         `json:"memory_usage,omitempty"`
	CPUUsage     float64         `json:"cpu_usage,omitempty"`
	LastSeenAt   *time.Time      `json:"last_seen_at,omitempty"`
	CreatedAt    *time.Time      `json:"created_at,omitempty"`
	UpdatedAt    *time.Time      `json:"updated_at,omitempty"`
}

// Preset mirrors the repository.WorkspacePreset shape returned by /api/v1/workspace/presets.
type Preset struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Layout      json.RawMessage `json:"layout,omitempty"`
	CreatedAt   *time.Time      `json:"created_at,omitempty"`
	UpdatedAt   *time.Time      `json:"updated_at,omitempty"`
}

// Resource is the shape returned by /api/v1/resources.
type Resource struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Type        string          `json:"type,omitempty"`
	Status      string          `json:"status,omitempty"`
	Description string          `json:"description,omitempty"`
	Extra       json.RawMessage `json:"-"`
}

// Rule is one interop rule entry returned by /api/v1/rules.
type Rule struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Severity       string `json:"severity"`
	Slot           string `json:"slot,omitempty"`
	SlotFile       string `json:"slot_file,omitempty"`
	Recommendation string `json:"recommendation,omitempty"`
	Description    string `json:"description,omitempty"`
	Priority       int    `json:"priority,omitempty"`
}

// RulesResponse is the envelope wrapping a rules query.
type RulesResponse struct {
	ScenarioName  string   `json:"scenario_name,omitempty"`
	TechStack     []string `json:"tech_stack,omitempty"`
	TotalCount    int      `json:"total_count"`
	FilteredCount int      `json:"filtered_count"`
	Rules         []Rule   `json:"rules"`
}

// ToolManifest describes a tool in the tool discovery manifest.
type ToolManifest struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Category    string          `json:"category,omitempty"`
	InputSchema json.RawMessage `json:"input_schema,omitempty"`
}

// ToolsManifest is the response from /api/v1/tools.
type ToolsManifest struct {
	Scenario    string         `json:"scenario,omitempty"`
	Version     string         `json:"version,omitempty"`
	Description string         `json:"description,omitempty"`
	Tools       []ToolManifest `json:"tools"`
}

// LighthouseMissingConfig is an entry from /api/v1/lighthouse/missing-configs.
type LighthouseMissingConfig struct {
	Scenario string `json:"scenario"`
	Reason   string `json:"reason,omitempty"`
}

// LighthouseHistoryEntry is one item in the lighthouse history response.
type LighthouseHistoryEntry struct {
	ReportID  string             `json:"report_id"`
	CreatedAt *time.Time         `json:"created_at,omitempty"`
	Scores    map[string]float64 `json:"scores,omitempty"`
}

// LogEntry is one structured log line.
type LogEntry struct {
	Timestamp string `json:"timestamp,omitempty"`
	Level     string `json:"level,omitempty"`
	Message   string `json:"message,omitempty"`
	Source    string `json:"source,omitempty"`
}

// MetricEntry is one data point in /api/v1/apps/:id/metrics.
type MetricEntry struct {
	Timestamp   string  `json:"timestamp,omitempty"`
	Status      string  `json:"status,omitempty"`
	CPUUsage    float64 `json:"cpu_usage,omitempty"`
	MemoryUsage float64 `json:"memory_usage,omitempty"`
}
