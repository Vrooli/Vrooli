package main

import (
	"time"

	signingtypes "scenario-to-desktop-api/signing/types"
)

// DesktopConfig represents the configuration for generating a desktop application
type DesktopConfig struct {
	// Application identity
	AppName        string `json:"app_name" validate:"required"`
	AppDisplayName string `json:"app_display_name" validate:"required"`
	AppDescription string `json:"app_description" validate:"required"`
	Version        string `json:"version" validate:"required"`
	Author         string `json:"author" validate:"required"`
	AuthorEmail    string `json:"author_email"`
	Icon           string `json:"icon,omitempty"`
	Homepage       string `json:"homepage"`
	License        string `json:"license"`
	AppID          string `json:"app_id" validate:"required"`
	AppURL         string `json:"app_url"`

	// Server configuration
	ServerType        string `json:"server_type" validate:"required,oneof=node static external executable"`
	ServerPort        int    `json:"server_port"`
	ServerPath        string `json:"server_path"`
	APIEndpoint       string `json:"api_endpoint" validate:"required,url"`
	ScenarioPath      string `json:"scenario_dist_path"`
	ScenarioName      string `json:"scenario_name"`
	AutoManageVrooli  bool   `json:"auto_manage_vrooli"`
	VrooliBinaryPath  string `json:"vrooli_binary_path"`
	DeploymentMode    string `json:"deployment_mode"`
	ProxyURL          string `json:"proxy_url,omitempty"`
	ExternalServerURL string `json:"external_server_url"`
	ExternalAPIURL    string `json:"external_api_url"`

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
	OutputPath string `json:"output_path"`
	// LocationMode indicates whether the generator should place output in the canonical
	// scenario folder, a gitignored staging area, or a custom path provided by the caller.
	LocationMode string `json:"location_mode,omitempty"`

	// Styling
	Styling map[string]interface{} `json:"styling"`

	// Bundled runtime configuration (optional; required for deployment_mode=bundled)
	BundleManifestPath       string           `json:"bundle_manifest_path,omitempty"`
	BundleRuntimeRoot        string           `json:"bundle_runtime_root,omitempty"`
	BundleIPC                *BundleIPCConfig `json:"bundle_ipc,omitempty"`
	BundleUISvcID            string           `json:"bundle_ui_service_id,omitempty"`
	BundleUIPortName         string           `json:"bundle_ui_port_name,omitempty"`
	BundleTelemetryUploadURL string           `json:"bundle_telemetry_upload_url,omitempty"`

	// Code signing configuration (optional; recommended for production)
	// Uses the full signing module types for comprehensive configuration.
	CodeSigning *signingtypes.SigningConfig `json:"code_signing,omitempty"`

	// Auto-update configuration (optional; recommended for distribution)
	UpdateConfig *UpdateConfig `json:"update_config,omitempty"`
}

// UpdateConfig configures auto-updates for the desktop application.
type UpdateConfig struct {
	// Channel is the update channel: "dev", "beta", or "stable" (default: "stable")
	Channel string `json:"channel,omitempty"`
	// Provider is the update provider: "github", "generic", or "none" (default: "none")
	Provider string `json:"provider,omitempty"`
	// AutoCheck enables automatic update checks on app start (default: false)
	AutoCheck bool `json:"auto_check"`
	// GitHub configuration (when provider=github)
	GitHub *GitHubUpdateConfig `json:"github,omitempty"`
	// Generic server configuration (when provider=generic)
	Generic *GenericUpdateConfig `json:"generic,omitempty"`
}

// GitHubUpdateConfig configures GitHub Releases as the update provider.
type GitHubUpdateConfig struct {
	// Owner is the GitHub organization or user
	Owner string `json:"owner"`
	// Repo is the GitHub repository name
	Repo string `json:"repo"`
	// Private indicates if this is a private repository (requires GH_TOKEN at runtime)
	Private bool `json:"private"`
}

// GenericUpdateConfig configures a self-hosted update server.
type GenericUpdateConfig struct {
	// URL is the base URL of the update server
	URL string `json:"url"`
	// ChannelPath is the URL path pattern for channels (default: "/{channel}")
	ChannelPath string `json:"channel_path,omitempty"`
}

// BundleIPCConfig captures runtime control surface hints derived from bundle.json.
type BundleIPCConfig struct {
	Host         string `json:"host,omitempty"`
	Port         int    `json:"port,omitempty"`
	AuthTokenRel string `json:"auth_token_path,omitempty"`
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
