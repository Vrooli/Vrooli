package generation

import (
	"time"

	signingtypes "scenario-to-desktop-api/signing/types"
)

// DesktopConfig represents the configuration for generating a desktop application.
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
	OutputPath   string `json:"output_path"`
	LocationMode string `json:"location_mode,omitempty"`

	// Styling
	Styling map[string]interface{} `json:"styling"`

	// Bundled runtime configuration
	BundleManifestPath       string           `json:"bundle_manifest_path,omitempty"`
	BundleRuntimeRoot        string           `json:"bundle_runtime_root,omitempty"`
	BundleIPC                *BundleIPCConfig `json:"bundle_ipc,omitempty"`
	BundleUISvcID            string           `json:"bundle_ui_service_id,omitempty"`
	BundleUIPortName         string           `json:"bundle_ui_port_name,omitempty"`
	BundleTelemetryUploadURL string           `json:"bundle_telemetry_upload_url,omitempty"`

	// Code signing configuration
	CodeSigning *signingtypes.SigningConfig `json:"code_signing,omitempty"`

	// Auto-update configuration
	UpdateConfig *UpdateConfig `json:"update_config,omitempty"`
}

// UpdateConfig configures auto-updates for the desktop application.
type UpdateConfig struct {
	Channel   string               `json:"channel,omitempty"`
	Provider  string               `json:"provider,omitempty"`
	AutoCheck bool                 `json:"auto_check"`
	GitHub    *GitHubUpdateConfig  `json:"github,omitempty"`
	Generic   *GenericUpdateConfig `json:"generic,omitempty"`
}

// GitHubUpdateConfig configures GitHub Releases as the update provider.
type GitHubUpdateConfig struct {
	Owner   string `json:"owner"`
	Repo    string `json:"repo"`
	Private bool   `json:"private"`
}

// GenericUpdateConfig configures a self-hosted update server.
type GenericUpdateConfig struct {
	URL         string `json:"url"`
	ChannelPath string `json:"channel_path,omitempty"`
}

// BundleIPCConfig captures runtime control surface hints from bundle.json.
type BundleIPCConfig struct {
	Host         string `json:"host,omitempty"`
	Port         int    `json:"port,omitempty"`
	AuthTokenRel string `json:"auth_token_path,omitempty"`
}

// ScenarioMetadata contains extracted information about a scenario.
type ScenarioMetadata struct {
	Name            string
	DisplayName     string
	Description     string
	Version         string
	Author          string
	License         string
	AppID           string
	HasUI           bool
	UIDistPath      string
	UIPort          int
	APIPort         int
	ScenarioPath    string
	Category        string
	Tags            []string
	ServiceJSONPath string
	PackageJSONPath string
}

// BuildStatus represents the status of a desktop build.
type BuildStatus struct {
	BuildID     string
	Status      string // building, ready, failed
	OutputPath  string
	StartedAt   time.Time
	CompletedAt *time.Time
	BuildLog    []string
	ErrorLog    []string
	Artifacts   map[string]string
	Metadata    map[string]interface{}
}

// ConnectionConfig represents saved connection settings for a scenario.
type ConnectionConfig struct {
	ProxyURL           string `json:"proxy_url,omitempty"`
	ServerType         string `json:"server_type,omitempty"`
	AutoManageVrooli   bool   `json:"auto_manage_vrooli,omitempty"`
	VrooliBinaryPath   string `json:"vrooli_binary_path,omitempty"`
	DeploymentMode     string `json:"deployment_mode,omitempty"`
	BundleManifestPath string `json:"bundle_manifest_path,omitempty"`
	AppDisplayName     string `json:"app_display_name,omitempty"`
	AppDescription     string `json:"app_description,omitempty"`
	Icon               string `json:"icon,omitempty"`
}

// QuickGenerateRequest is the request for quick desktop generation.
type QuickGenerateRequest struct {
	ScenarioName     string   `json:"scenario_name"`
	TemplateType     string   `json:"template_type"`
	DeploymentMode   string   `json:"deployment_mode"`
	AutoManageVrooli *bool    `json:"auto_manage_vrooli"`
	LegacyAutoManage *bool    `json:"auto_manage_tier1"`
	ProxyURL         string   `json:"proxy_url"`
	LegacyServerURL  string   `json:"server_url"`
	LegacyAPIURL     string   `json:"api_url"`
	BundleManifest   string   `json:"bundle_manifest_path"`
	VrooliBinary     string   `json:"vrooli_binary_path"`
	Platforms        []string `json:"platforms"`
}

// GenerateResponse is the response from generation endpoints.
type GenerateResponse struct {
	BuildID             string            `json:"build_id"`
	Status              string            `json:"status"`
	ScenarioName        string            `json:"scenario_name,omitempty"`
	DesktopPath         string            `json:"desktop_path"`
	DetectedMetadata    *ScenarioMetadata `json:"detected_metadata,omitempty"`
	InstallInstructions string            `json:"install_instructions"`
	TestCommand         string            `json:"test_command"`
	StatusURL           string            `json:"status_url"`
}
