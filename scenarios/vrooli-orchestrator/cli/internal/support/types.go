package support

import (
	"time"
)

// Profile mirrors the API's Profile shape returned by /api/v1/profiles and
// /api/v1/profiles/{name}.
type Profile struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	DisplayName     string                 `json:"display_name"`
	Description     string                 `json:"description,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	Resources       []string               `json:"resources"`
	Scenarios       []string               `json:"scenarios"`
	AutoBrowser     []string               `json:"auto_browser,omitempty"`
	EnvironmentVars map[string]string      `json:"environment_vars,omitempty"`
	IdleShutdown    *int                   `json:"idle_shutdown_minutes,omitempty"`
	Dependencies    []string               `json:"dependencies,omitempty"`
	Status          string                 `json:"status"`
	CreatedAt       *time.Time             `json:"created_at,omitempty"`
	UpdatedAt       *time.Time             `json:"updated_at,omitempty"`
}

// ProfileListResponse is the envelope returned by GET /api/v1/profiles.
type ProfileListResponse struct {
	Profiles []Profile `json:"profiles"`
	Count    int       `json:"count"`
}

// ActivationResult mirrors the API's ActivationResult.
type ActivationResult struct {
	ProfileName     string                 `json:"profile_name"`
	Success         bool                   `json:"success"`
	ResourcesStatus map[string]interface{} `json:"resources_status,omitempty"`
	ScenariosStatus map[string]interface{} `json:"scenarios_status,omitempty"`
	BrowserActions  []string               `json:"browser_actions,omitempty"`
	Message         string                 `json:"message,omitempty"`
	Error           string                 `json:"error,omitempty"`
}

// DeactivationResult mirrors the API's DeactivationResult.
type DeactivationResult struct {
	Success         bool                   `json:"success"`
	ResourcesStatus map[string]interface{} `json:"resources_status,omitempty"`
	ScenariosStatus map[string]interface{} `json:"scenarios_status,omitempty"`
	Message         string                 `json:"message,omitempty"`
	Error           string                 `json:"error,omitempty"`
}

// StatusResponse mirrors the /api/v1/status payload.
type StatusResponse struct {
	Service       string     `json:"service,omitempty"`
	Version       string     `json:"version,omitempty"`
	Status        string     `json:"status,omitempty"`
	Timestamp     *time.Time `json:"timestamp,omitempty"`
	ActiveProfile *Profile   `json:"active_profile,omitempty"`
	ResourceCount int        `json:"resource_count"`
	ScenarioCount int        `json:"scenario_count"`
}
