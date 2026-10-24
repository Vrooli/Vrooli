package build

import "time"

// Status represents the status of a desktop application build.
type Status struct {
	BuildID            string                     `json:"build_id"`
	ScenarioName       string                     `json:"scenario_name"`
	Status             string                     `json:"status"` // building, ready, partial, failed
	Framework          string                     `json:"framework"`
	TemplateType       string                     `json:"template_type"`
	Platforms          []string                   `json:"platforms"`           // Legacy: platforms that were built
	RequestedPlatforms []string                   `json:"requested_platforms"` // NEW: platforms that were requested to build
	PlatformResults    map[string]*PlatformResult `json:"platform_results,omitempty"`
	OutputPath         string                     `json:"output_path"`
	CreatedAt          time.Time                  `json:"created_at"`
	CompletedAt        *time.Time                 `json:"completed_at,omitempty"`
	ErrorLog           []string                   `json:"error_log,omitempty"`
	BuildLog           []string                   `json:"build_log,omitempty"`
	Artifacts          map[string]string          `json:"artifacts,omitempty"`
	Metadata           map[string]interface{}     `json:"metadata,omitempty"`
}

// PlatformResult represents the result of building for a specific platform.
type PlatformResult struct {
	Platform    string     `json:"platform"`
	Status      string     `json:"status"` // building, ready, failed, skipped
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	ErrorLog    []string   `json:"error_log,omitempty"`
	Artifact    string     `json:"artifact,omitempty"`
	FileSize    int64      `json:"file_size,omitempty"`
	SkipReason  string     `json:"skip_reason,omitempty"` // e.g., "Wine not installed"
}

// BuildRequest represents a request to build a desktop application.
type BuildRequest struct {
	DesktopPath string   `json:"desktop_path"`
	Platforms   []string `json:"platforms"`
	Sign        bool     `json:"sign"`
	Publish     bool     `json:"publish"`
}

// ScenarioBuildRequest represents a request to build a scenario's desktop app.
type ScenarioBuildRequest struct {
	ScenarioName string   `json:"scenario_name"`
	DesktopPath  string   `json:"desktop_path"`
	Platforms    []string `json:"platforms"`
	Clean        bool     `json:"clean"`
}

// BuildResponse represents the response from a build endpoint.
type BuildResponse struct {
	BuildID   string `json:"build_id"`
	Status    string `json:"status"`
	StatusURL string `json:"status_url,omitempty"`
}
