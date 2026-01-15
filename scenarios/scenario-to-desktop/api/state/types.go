// Package state provides server-side persistence for desktop generator scenario state.
// It enables full persistence of form state, pipeline stage results, compressed logs,
// and build artifacts with hash-based invalidation.
package state

import (
	"encoding/json"
	"time"
)

// SchemaVersion is incremented when breaking changes are made to the state format.
const SchemaVersion = 1

// Stage constants define the pipeline stages.
const (
	StageBundle    = "bundle"
	StagePreflight = "preflight"
	StageGenerate  = "generate"
	StageBuild     = "build"
	StageSmokeTest = "smoketest"
)

// StageOrder defines the execution order for pipeline stages.
var StageOrder = []string{StageBundle, StagePreflight, StageGenerate, StageBuild, StageSmokeTest}

// Status constants for stage validation status.
const (
	StatusValid   = "valid"
	StatusStale   = "stale"
	StatusInvalid = "invalid"
	StatusNone    = "none"
)

// ScenarioState represents the complete persisted state for a scenario's
// desktop generator configuration and pipeline results.
type ScenarioState struct {
	ScenarioName   string                `json:"scenario_name"`
	SchemaVersion  int                   `json:"schema_version"`
	CreatedAt      time.Time             `json:"created_at"`
	UpdatedAt      time.Time             `json:"updated_at"`
	Hash           string                `json:"hash,omitempty"`
	FormState      FormState             `json:"form_state"`
	Stages         map[string]StageState `json:"stages,omitempty"`
	CompressedLogs []CompressedLog       `json:"compressed_logs,omitempty"`
	BuildArtifacts []BuildArtifact       `json:"build_artifacts,omitempty"`
}

// FormState captures user-entered generator configuration.
// This mirrors the TypeScript GeneratorDraftState from the UI.
type FormState struct {
	// Template and framework selection
	SelectedTemplate string `json:"selected_template,omitempty"`
	Framework        string `json:"framework,omitempty"`

	// App metadata (user-editable)
	AppDisplayName    string `json:"app_display_name,omitempty"`
	AppDescription    string `json:"app_description,omitempty"`
	IconPath          string `json:"icon_path,omitempty"`
	DisplayNameEdited bool   `json:"display_name_edited,omitempty"`
	DescriptionEdited bool   `json:"description_edited,omitempty"`
	IconPathEdited    bool   `json:"icon_path_edited,omitempty"`

	// Server configuration
	ServerType       string `json:"server_type,omitempty"`
	DeploymentMode   string `json:"deployment_mode,omitempty"`
	ProxyURL         string `json:"proxy_url,omitempty"`
	ServerPort       int    `json:"server_port,omitempty"`
	LocalServerPath  string `json:"local_server_path,omitempty"`
	LocalAPIEndpoint string `json:"local_api_endpoint,omitempty"`
	AutoManageTier1  bool   `json:"auto_manage_tier1,omitempty"`
	VrooliBinaryPath string `json:"vrooli_binary_path,omitempty"`

	// Bundle configuration
	BundleManifestPath string `json:"bundle_manifest_path,omitempty"`

	// Platform configuration
	Platforms    PlatformSelection `json:"platforms,omitempty"`
	LocationMode string            `json:"location_mode,omitempty"`
	OutputPath   string            `json:"output_path,omitempty"`

	// Connection state (cached, not secrets)
	ConnectionResult json.RawMessage `json:"connection_result,omitempty"`
	ConnectionError  *string         `json:"connection_error,omitempty"`

	// Preflight state (cached result)
	PreflightResult        json.RawMessage   `json:"preflight_result,omitempty"`
	PreflightError         *string           `json:"preflight_error,omitempty"`
	PreflightOverride      bool              `json:"preflight_override,omitempty"`
	PreflightSecrets       map[string]string `json:"preflight_secrets,omitempty"`
	PreflightStartServices bool              `json:"preflight_start_services,omitempty"`
	PreflightAutoRefresh   bool              `json:"preflight_auto_refresh,omitempty"`
	PreflightSessionID     *string           `json:"preflight_session_id,omitempty"`
	PreflightSessionExpiry *string           `json:"preflight_session_expires_at,omitempty"`
	PreflightSessionTTL    int               `json:"preflight_session_ttl,omitempty"`

	// Deployment
	DeploymentManagerURL *string `json:"deployment_manager_url,omitempty"`

	// Signing
	SigningEnabledForBuild bool `json:"signing_enabled_for_build,omitempty"`

	// Bundle result (cached auto-build status for restoration on page load)
	BundleResult json.RawMessage `json:"bundle_result,omitempty"`

	// Smoke test state (persisted for restoration on page load)
	SmokeTestID                *string  `json:"smoke_test_id,omitempty"`
	SmokeTestPlatform          *string  `json:"smoke_test_platform,omitempty"`
	SmokeTestStatus            *string  `json:"smoke_test_status,omitempty"`
	SmokeTestStartedAt         *string  `json:"smoke_test_started_at,omitempty"`
	SmokeTestCompletedAt       *string  `json:"smoke_test_completed_at,omitempty"`
	SmokeTestLogs              []string `json:"smoke_test_logs,omitempty"`
	SmokeTestError             *string  `json:"smoke_test_error,omitempty"`
	SmokeTestTelemetryUploaded bool     `json:"smoke_test_telemetry_uploaded,omitempty"`

	// Wrapper build state (persisted for restoration on page load)
	// This tracks the Electron wrapper generation, not platform installers
	WrapperBuildID     *string `json:"wrapper_build_id,omitempty"`
	WrapperBuildStatus *string `json:"wrapper_build_status,omitempty"` // building, ready, failed
	WrapperOutputPath  *string `json:"wrapper_output_path,omitempty"`
}

// PlatformSelection mirrors the UI platform toggle state.
type PlatformSelection struct {
	Win   bool `json:"win,omitempty"`
	Mac   bool `json:"mac,omitempty"`
	Linux bool `json:"linux,omitempty"`
}

// StageState holds the validated state for a single pipeline stage.
type StageState struct {
	Stage            string           `json:"stage"`
	Status           string           `json:"status"`
	InputFingerprint InputFingerprint `json:"input_fingerprint"`
	OutputHash       string           `json:"output_hash,omitempty"`
	ValidatedAt      time.Time        `json:"validated_at"`
	Result           json.RawMessage  `json:"result,omitempty"`
	StalenessReason  string           `json:"staleness_reason,omitempty"`
}

// InputFingerprint captures all inputs that affect a stage's validity.
type InputFingerprint struct {
	// Bundle stage inputs
	ManifestPath  string `json:"manifest_path,omitempty"`
	ManifestHash  string `json:"manifest_hash,omitempty"`
	ManifestMtime int64  `json:"manifest_mtime,omitempty"`

	// Preflight stage inputs
	PreflightSecretKeys []string `json:"preflight_secret_keys,omitempty"`
	PreflightTimeout    int      `json:"preflight_timeout,omitempty"`
	StartServices       bool     `json:"start_services,omitempty"`

	// Generate stage inputs
	TemplateType   string `json:"template_type,omitempty"`
	Framework      string `json:"framework,omitempty"`
	DeploymentMode string `json:"deployment_mode,omitempty"`
	AppDisplayName string `json:"app_display_name,omitempty"`
	AppDescription string `json:"app_description,omitempty"`
	IconPath       string `json:"icon_path,omitempty"`

	// Build stage inputs
	Platforms         []string `json:"platforms,omitempty"`
	SigningEnabled    bool     `json:"signing_enabled,omitempty"`
	SigningConfigHash string   `json:"signing_config_hash,omitempty"`
	OutputLocation    string   `json:"output_location,omitempty"`

	// Smoke test inputs
	SmokeTestPlatform string `json:"smoke_test_platform,omitempty"`
}

// CompressedLog stores gzip-compressed log content from preflight.
type CompressedLog struct {
	ServiceID      string `json:"service_id"`
	CompressedData string `json:"compressed_data"` // Base64-encoded gzip
	OriginalLines  int    `json:"original_lines"`
	CompressedSize int    `json:"compressed_size"`
	CapturedAt     string `json:"captured_at"`
}

// BuildArtifact tracks a built installer for a platform.
type BuildArtifact struct {
	Platform     string     `json:"platform"`
	Status       string     `json:"status"` // pending, building, ready, failed
	FilePath     string     `json:"file_path,omitempty"`
	FileName     string     `json:"file_name,omitempty"`
	FileSize     int64      `json:"file_size,omitempty"`
	BuildID      string     `json:"build_id,omitempty"`
	BuiltAt      *time.Time `json:"built_at,omitempty"`
	ErrorMessage string     `json:"error_message,omitempty"`
}

// StateChange represents a detected change requiring invalidation.
type StateChange struct {
	ChangeType    string `json:"change_type"`
	AffectedStage string `json:"affected_stage"`
	Reason        string `json:"reason"`
	OldValue      string `json:"old_value,omitempty"`
	NewValue      string `json:"new_value,omitempty"`
}

// ValidationStatus provides UI-friendly state information.
type ValidationStatus struct {
	ScenarioName   string                 `json:"scenario_name"`
	OverallStatus  string                 `json:"overall_status"` // valid, partial, stale, none
	Stages         map[string]StageStatus `json:"stages"`
	PendingChanges []StateChange          `json:"pending_changes,omitempty"`
	LastValidated  *time.Time             `json:"last_validated,omitempty"`
}

// StageStatus is the UI-facing stage state.
type StageStatus struct {
	Stage           string `json:"stage"`
	Status          string `json:"status"`
	LastRun         string `json:"last_run,omitempty"`
	StalenessReason string `json:"staleness_reason,omitempty"`
	CanReuse        bool   `json:"can_reuse"`
}

// --- API Request/Response Types ---

// LoadStateRequest options for loading scenario state.
type LoadStateRequest struct {
	IncludeLogs      bool   `json:"include_logs,omitempty"`
	ValidateManifest bool   `json:"validate_manifest,omitempty"`
	ManifestPath     string `json:"manifest_path,omitempty"`
}

// LoadStateResponse returns scenario state with optional validation info.
type LoadStateResponse struct {
	State           *ScenarioState `json:"state,omitempty"`
	Found           bool           `json:"found"`
	ManifestChanged bool           `json:"manifest_changed,omitempty"`
	CurrentHash     string         `json:"current_hash,omitempty"`
	StoredHash      string         `json:"stored_hash,omitempty"`
}

// SaveStateRequest provides state update data.
type SaveStateRequest struct {
	FormState      FormState                  `json:"form_state"`
	ManifestPath   string                     `json:"manifest_path,omitempty"`
	ComputeHash    bool                       `json:"compute_hash,omitempty"`
	LogTails       []LogTailInput             `json:"log_tails,omitempty"`
	BuildArtifacts []BuildArtifact            `json:"build_artifacts,omitempty"`
	StageResults   map[string]json.RawMessage `json:"stage_results,omitempty"`
	ExpectedHash   string                     `json:"expected_hash,omitempty"`
}

// LogTailInput is uncompressed log data to be compressed on save.
type LogTailInput struct {
	ServiceID string `json:"service_id"`
	Content   string `json:"content"`
	Lines     int    `json:"lines"`
}

// SaveStateResponse confirms state was saved.
type SaveStateResponse struct {
	Success     bool           `json:"success"`
	UpdatedAt   time.Time      `json:"updated_at"`
	Hash        string         `json:"hash,omitempty"`
	Conflict    bool           `json:"conflict,omitempty"`
	ServerState *ScenarioState `json:"server_state,omitempty"`
}

// ClearStateResponse confirms state was deleted.
type ClearStateResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// CheckStalenessRequest provides current config for comparison.
type CheckStalenessRequest struct {
	CurrentConfig InputFingerprint `json:"current_config"`
}

// CheckStalenessResponse reports detected changes.
type CheckStalenessResponse struct {
	Valid          bool              `json:"valid"`
	CurrentHash    string            `json:"current_hash,omitempty"`
	StoredHash     string            `json:"stored_hash,omitempty"`
	Changed        bool              `json:"changed"`
	PendingChanges []StateChange     `json:"pending_changes,omitempty"`
	AffectedStages []string          `json:"affected_stages,omitempty"`
	Status         *ValidationStatus `json:"status,omitempty"`
}

// GetLogsResponse returns decompressed logs for a service.
type GetLogsResponse struct {
	ServiceID  string `json:"service_id"`
	Content    string `json:"content"`
	Lines      int    `json:"lines"`
	CapturedAt string `json:"captured_at"`
}
