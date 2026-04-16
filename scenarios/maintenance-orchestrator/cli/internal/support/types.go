package support

import (
	"encoding/json"
)

// Scenario mirrors the shape returned by /api/v1/scenarios — each entry is a
// maintenance scenario discovered by the orchestrator.
type Scenario struct {
	ID            string             `json:"id"`
	Name          string             `json:"name"`
	DisplayName   string             `json:"displayName,omitempty"`
	Description   string             `json:"description,omitempty"`
	IsActive      bool               `json:"isActive"`
	Endpoint      string             `json:"endpoint,omitempty"`
	Port          json.RawMessage    `json:"port,omitempty"`
	Tags          []string           `json:"tags,omitempty"`
	LastActive    string             `json:"lastActive,omitempty"`
	ResourceUsage map[string]float64 `json:"resourceUsage,omitempty"`
}

// ScenariosResponse wraps the scenario list endpoint.
type ScenariosResponse struct {
	Scenarios []Scenario `json:"scenarios"`
}

// AllScenarioInfo mirrors the shape in /api/v1/all-scenarios. Fields are
// sourced from the underlying `vrooli scenario list --json` output and are
// intentionally untyped beyond a few common fields.
type AllScenarioInfo struct {
	Name        string          `json:"name"`
	DisplayName string          `json:"displayName,omitempty"`
	Description string          `json:"description,omitempty"`
	Tags        json.RawMessage `json:"tags,omitempty"`
}

// AllScenariosResponse wraps the list endpoint for all scenarios on the host.
type AllScenariosResponse struct {
	Scenarios []AllScenarioInfo `json:"scenarios"`
}

// ScenarioStatus is one entry in /api/v1/scenario-statuses. The map key is the
// scenario name.
type ScenarioStatus struct {
	Status       string `json:"status"`
	ProcessCount int    `json:"processCount"`
}

// ScenarioStatusesResponse wraps the statuses endpoint.
type ScenarioStatusesResponse struct {
	Statuses map[string]ScenarioStatus `json:"statuses"`
}

// Preset mirrors the preset shape returned by /api/v1/presets.
type Preset struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	States      map[string]bool `json:"states,omitempty"`
	Tags        []string        `json:"tags,omitempty"`
	Pattern     string          `json:"pattern,omitempty"`
	IsDefault   bool            `json:"isDefault,omitempty"`
	IsActive    bool            `json:"isActive,omitempty"`
}

// PresetsResponse wraps the presets list endpoint.
type PresetsResponse struct {
	Presets []Preset `json:"presets"`
}

// ActivePresetsResponse wraps the active presets endpoint.
type ActivePresetsResponse struct {
	ActivePresets []Preset `json:"activePresets"`
}

// ApplyPresetResponse is the response body of POST /api/v1/presets/{id}/apply.
type ApplyPresetResponse struct {
	Success     bool     `json:"success"`
	Preset      string   `json:"preset"`
	Activated   []string `json:"activated"`
	Deactivated []string `json:"deactivated"`
}

// StopAllResponse is the response body of POST /api/v1/stop-all.
type StopAllResponse struct {
	Success     bool     `json:"success"`
	Deactivated []string `json:"deactivated"`
}

// ActivityEntry mirrors the orchestrator's recent activity log entries.
type ActivityEntry struct {
	Timestamp string `json:"timestamp,omitempty"`
	Action    string `json:"action,omitempty"`
	Scenario  string `json:"scenario,omitempty"`
	Preset    string `json:"preset,omitempty"`
	Success   bool   `json:"success,omitempty"`
	Message   string `json:"message,omitempty"`
}

// StatusResponse is the shape returned by GET /api/v1/status.
type StatusResponse struct {
	Health            string          `json:"health"`
	MaintenanceState  string          `json:"maintenanceState"`
	TotalScenarios    int             `json:"totalScenarios"`
	ActiveScenarios   int             `json:"activeScenarios"`
	InactiveScenarios int             `json:"inactiveScenarios"`
	RecentActivity    []ActivityEntry `json:"recentActivity,omitempty"`
	Uptime            float64         `json:"uptime"`
}

// PortResponse wraps GET /api/v1/scenarios/{name}/port.
type PortResponse struct {
	Port json.RawMessage `json:"port"`
	Type string          `json:"type,omitempty"`
}
