package main

import "time"

// DesktopConfig represents the configuration for generating a desktop application
type DesktopConfig struct {
	// Application identity
	AppName        string `json:"app_name" validate:"required"`
	AppDisplayName string `json:"app_display_name" validate:"required"`
	AppDescription string `json:"app_description" validate:"required"`
	Version        string `json:"version" validate:"required"`
	Author         string `json:"author" validate:"required"`
	AuthorEmail    string `json:"author_email"`
	Homepage       string `json:"homepage"`
	License        string `json:"license"`
	AppID          string `json:"app_id" validate:"required"`
	AppURL         string `json:"app_url"`

	// Server configuration
	ServerType   string `json:"server_type" validate:"required,oneof=node static external executable"`
	ServerPort   int    `json:"server_port"`
	ServerPath   string `json:"server_path" validate:"required"`
	APIEndpoint  string `json:"api_endpoint" validate:"required,url"`
	ScenarioPath string `json:"scenario_dist_path"`

	// Template configuration
	Framework    string `json:"framework" validate:"required,oneof=electron tauri neutralino"`
	TemplateType string `json:"template_type" validate:"required,oneof=basic advanced multi_window kiosk"`

	// Features
	Features map[string]interface{} `json:"features"`

	// Window configuration
	Window map[string]interface{} `json:"window"`

	// Platform targets
	Platforms []string `json:"platforms"`

	// Output configuration
	OutputPath string `json:"output_path" validate:"required"`

	// Styling
	Styling map[string]interface{} `json:"styling"`
}

// PlatformBuildResult represents the result of building for a specific platform
type PlatformBuildResult struct {
	Platform    string     `json:"platform"`
	Status      string     `json:"status"` // building, ready, failed, skipped
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	ErrorLog    []string   `json:"error_log,omitempty"`
	Artifact    string     `json:"artifact,omitempty"`
	FileSize    int64      `json:"file_size,omitempty"`
	SkipReason  string     `json:"skip_reason,omitempty"` // e.g., "Wine not installed"
}

// BuildStatus represents the status of a desktop application build
type BuildStatus struct {
	BuildID            string                          `json:"build_id"`
	ScenarioName       string                          `json:"scenario_name"`
	Status             string                          `json:"status"` // building, ready, partial, failed
	Framework          string                          `json:"framework"`
	TemplateType       string                          `json:"template_type"`
	Platforms          []string                        `json:"platforms"`           // Legacy: platforms that were built
	RequestedPlatforms []string                        `json:"requested_platforms"` // NEW: platforms that were requested to build
	PlatformResults    map[string]*PlatformBuildResult `json:"platform_results,omitempty"`
	OutputPath         string                          `json:"output_path"`
	CreatedAt          time.Time                       `json:"created_at"`
	CompletedAt        *time.Time                      `json:"completed_at,omitempty"`
	ErrorLog           []string                        `json:"error_log,omitempty"`
	BuildLog           []string                        `json:"build_log,omitempty"`
	Artifacts          map[string]string               `json:"artifacts,omitempty"`
	Metadata           map[string]interface{}          `json:"metadata,omitempty"`
}

// TemplateInfo represents information about available templates
type TemplateInfo struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Type        string   `json:"type"`
	Framework   string   `json:"framework"`
	UseCases    []string `json:"use_cases"`
	Features    []string `json:"features"`
	Complexity  string   `json:"complexity"`
	Examples    []string `json:"examples"`
}

// WineInstallStatus represents the status of Wine installation
type WineInstallStatus struct {
	InstallID   string     `json:"install_id"`
	Status      string     `json:"status"` // pending, installing, completed, failed
	Method      string     `json:"method"` // flatpak, appimage, skip
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Log         []string   `json:"log,omitempty"`
	ErrorLog    []string   `json:"error_log,omitempty"`
}

// WineCheckResponse represents Wine availability status
type WineCheckResponse struct {
	Installed         bool                `json:"installed"`
	Version           string              `json:"version,omitempty"`
	Platform          string              `json:"platform"`
	RequiredFor       []string            `json:"required_for"`
	InstallMethods    []WineInstallMethod `json:"install_methods"`
	RecommendedMethod string              `json:"recommended_method,omitempty"`
}

// WineInstallMethod represents an installation method
type WineInstallMethod struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	RequiresSudo bool     `json:"requires_sudo"`
	Steps        []string `json:"steps"`
	Estimated    string   `json:"estimated_time"`
}
