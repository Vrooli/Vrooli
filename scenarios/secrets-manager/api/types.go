package main

import (
	"encoding/json"
	"time"
)

var deploymentTierCatalog = []struct {
	Name  string
	Label string
}{
	{"tier-1-local", "Tier 1 · Local / Developer"},
	{"tier-2-desktop", "Tier 2 · Desktop"},
	{"tier-3-mobile", "Tier 3 · Mobile"},
	{"tier-4-saas", "Tier 4 · SaaS / Cloud"},
	{"tier-5-enterprise", "Tier 5 · Enterprise / Appliance"},
}

const defaultVulnerabilityStatus = "open"

var allowedVulnerabilityStatuses = map[string]struct{}{
	"open":        {},
	"in_progress": {},
	"resolved":    {},
	"accepted":    {},
	"regressed":   {},
}

// Data models
type ResourceSecret struct {
	ID                string    `json:"id" db:"id"`
	ResourceName      string    `json:"resource_name" db:"resource_name"`
	SecretKey         string    `json:"secret_key" db:"secret_key"`
	SecretType        string    `json:"secret_type" db:"secret_type"`
	Required          bool      `json:"required" db:"required"`
	Description       *string   `json:"description" db:"description"`
	ValidationPattern *string   `json:"validation_pattern" db:"validation_pattern"`
	DocumentationURL  *string   `json:"documentation_url" db:"documentation_url"`
	DefaultValue      *string   `json:"default_value" db:"default_value"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}

type SecretValidation struct {
	ID                  string    `json:"id" db:"id"`
	ResourceSecretID    string    `json:"resource_secret_id" db:"resource_secret_id"`
	ValidationStatus    string    `json:"validation_status" db:"validation_status"`
	ValidationMethod    string    `json:"validation_method" db:"validation_method"`
	ValidationTimestamp time.Time `json:"validation_timestamp" db:"validation_timestamp"`
	ErrorMessage        *string   `json:"error_message" db:"error_message"`
	ValidationDetails   *string   `json:"validation_details" db:"validation_details"`
}

type SecretScan struct {
	ID                string    `json:"id" db:"id"`
	ScanType          string    `json:"scan_type" db:"scan_type"`
	ResourcesScanned  []string  `json:"resources_scanned" db:"resources_scanned"`
	SecretsDiscovered int       `json:"secrets_discovered" db:"secrets_discovered"`
	ScanDurationMs    int       `json:"scan_duration_ms" db:"scan_duration_ms"`
	ScanTimestamp     time.Time `json:"scan_timestamp" db:"scan_timestamp"`
	ScanStatus        string    `json:"scan_status" db:"scan_status"`
	ErrorMessage      *string   `json:"error_message" db:"error_message"`
	ScanMetadata      *string   `json:"scan_metadata" db:"scan_metadata"`
}

type SecretHealthSummary struct {
	ResourceName           string     `json:"resource_name"`
	TotalSecrets           int        `json:"total_secrets"`
	RequiredSecrets        int        `json:"required_secrets"`
	ValidSecrets           int        `json:"valid_secrets"`
	MissingRequiredSecrets int        `json:"missing_required_secrets"`
	InvalidSecrets         int        `json:"invalid_secrets"`
	LastValidation         *time.Time `json:"last_validation"`
}

type SecretDeploymentStrategy struct {
	ResourceSecretID  string          `json:"resource_secret_id"`
	Tier              string          `json:"tier"`
	HandlingStrategy  string          `json:"handling_strategy"`
	FallbackStrategy  *string         `json:"fallback_strategy,omitempty"`
	RequiresUserInput bool            `json:"requires_user_input"`
	PromptLabel       *string         `json:"prompt_label,omitempty"`
	PromptDescription *string         `json:"prompt_description,omitempty"`
	GeneratorTemplate json.RawMessage `json:"generator_template,omitempty"`
	BundleHints       json.RawMessage `json:"bundle_hints,omitempty"`
}

// ScenarioSecretOverride represents a scenario-specific strategy override.
type ScenarioSecretOverride struct {
	ID                string          `json:"id"`
	ScenarioName      string          `json:"scenario_name"`
	ResourceSecretID  string          `json:"resource_secret_id"`
	ResourceName      string          `json:"resource_name"`       // Joined from resource_secrets
	SecretKey         string          `json:"secret_key"`          // Joined from resource_secrets
	Tier              string          `json:"tier"`
	HandlingStrategy  *string         `json:"handling_strategy,omitempty"`
	FallbackStrategy  *string         `json:"fallback_strategy,omitempty"`
	RequiresUserInput *bool           `json:"requires_user_input,omitempty"`
	PromptLabel       *string         `json:"prompt_label,omitempty"`
	PromptDescription *string         `json:"prompt_description,omitempty"`
	GeneratorTemplate json.RawMessage `json:"generator_template,omitempty"`
	BundleHints       json.RawMessage `json:"bundle_hints,omitempty"`
	OverrideReason    *string         `json:"override_reason,omitempty"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

// EffectiveSecretStrategy represents the merged strategy (resource default + scenario override).
type EffectiveSecretStrategy struct {
	ResourceName      string   `json:"resource_name"`
	SecretKey         string   `json:"secret_key"`
	Tier              string   `json:"tier"`
	HandlingStrategy  string   `json:"handling_strategy"`
	FallbackStrategy  string   `json:"fallback_strategy,omitempty"`
	RequiresUserInput bool     `json:"requires_user_input"`
	PromptLabel       string   `json:"prompt_label,omitempty"`
	PromptDescription string   `json:"prompt_description,omitempty"`
	IsOverridden      bool     `json:"is_overridden"`
	OverriddenFields  []string `json:"overridden_fields,omitempty"`
	OverrideReason    string   `json:"override_reason,omitempty"`
}

// OrphanOverride represents an override that no longer has a valid scenario or dependency.
type OrphanOverride struct {
	Override ScenarioSecretOverride `json:"override"`
	Reason   string                 `json:"reason"`
}

// SetOverrideRequest is the request body for creating/updating a scenario override.
type SetOverrideRequest struct {
	HandlingStrategy  *string                `json:"handling_strategy,omitempty"`
	FallbackStrategy  *string                `json:"fallback_strategy,omitempty"`
	RequiresUserInput *bool                  `json:"requires_user_input,omitempty"`
	PromptLabel       *string                `json:"prompt_label,omitempty"`
	PromptDescription *string                `json:"prompt_description,omitempty"`
	GeneratorTemplate map[string]interface{} `json:"generator_template,omitempty"`
	BundleHints       map[string]interface{} `json:"bundle_hints,omitempty"`
	OverrideReason    *string                `json:"override_reason,omitempty"`
}

// CopyFromTierRequest is the request body for copying overrides between tiers.
type CopyFromTierRequest struct {
	SourceTier string `json:"source_tier"`
	TargetTier string `json:"target_tier"`
	Overwrite  bool   `json:"overwrite"`
}

// CopyFromScenarioRequest is the request body for copying overrides from another scenario.
type CopyFromScenarioRequest struct {
	SourceScenario string `json:"source_scenario"`
	Tier           string `json:"tier"`
	Overwrite      bool   `json:"overwrite"`
}

// CleanupOrphansRequest is the request body for cleaning up orphan overrides.
type CleanupOrphansRequest struct {
	DryRun bool `json:"dry_run"`
}

type DeploymentManifest struct {
	Scenario            string                       `json:"scenario"`
	Tier                string                       `json:"tier"`
	GeneratedAt         time.Time                    `json:"generated_at"`
	Resources           []string                     `json:"resources"`
	Secrets             []DeploymentSecretEntry      `json:"secrets"`
	BundleSecrets       []BundleSecretPlan           `json:"bundle_secrets,omitempty"`
	Summary             DeploymentSummary            `json:"summary"`
	Dependencies        []DependencyInsight          `json:"dependencies,omitempty"`
	TierAggregates      map[string]TierAggregateView `json:"tier_aggregates,omitempty"`
	AnalyzerGeneratedAt *time.Time                   `json:"analyzer_generated_at,omitempty"`
}

type DeploymentSummary struct {
	TotalSecrets          int                    `json:"total_secrets"`
	StrategizedSecrets    int                    `json:"strategized_secrets"`
	RequiresAction        int                    `json:"requires_action"`
	BlockingSecrets       []string               `json:"blocking_secrets"`
	BlockingSecretDetails []BlockingSecretDetail `json:"blocking_secret_details,omitempty"`
	ClassificationWeights map[string]int         `json:"classification_weights"`
	StrategyBreakdown     map[string]int         `json:"strategy_breakdown"`
	ScopeReadiness        map[string]string      `json:"scope_readiness"`
}

type BlockingSecretDetail struct {
	Secret         string   `json:"secret"`
	Resource       string   `json:"resource"`
	Source         string   `json:"source"`
	DependencyPath []string `json:"dependency_path,omitempty"`
}

type DeploymentSecretEntry struct {
	ID                string                 `json:"id,omitempty"`
	ResourceName      string                 `json:"resource_name"`
	SecretKey         string                 `json:"secret_key"`
	SecretType        string                 `json:"secret_type"`
	Required          bool                   `json:"required"`
	Classification    string                 `json:"classification"`
	Description       string                 `json:"description,omitempty"`
	ValidationPattern string                 `json:"validation_pattern,omitempty"`
	OwnerTeam         string                 `json:"owner_team,omitempty"`
	OwnerContact      string                 `json:"owner_contact,omitempty"`
	HandlingStrategy  string                 `json:"handling_strategy"`
	RequiresUserInput bool                   `json:"requires_user_input"`
	Prompt            *PromptMetadata        `json:"prompt,omitempty"`
	GeneratorTemplate map[string]interface{} `json:"generator_template,omitempty"`
	BundleHints       map[string]interface{} `json:"bundle_hints,omitempty"`
	TierStrategies    map[string]string      `json:"tier_strategies,omitempty"`
	FallbackStrategy  string                 `json:"fallback_strategy,omitempty"`
}

type BundleSecretPlan struct {
	ID          string                 `json:"id"`
	Class       string                 `json:"class"`
	Required    bool                   `json:"required"`
	Description string                 `json:"description,omitempty"`
	Format      string                 `json:"format,omitempty"`
	Target      BundleSecretTarget     `json:"target"`
	Prompt      *PromptMetadata        `json:"prompt,omitempty"`
	Generator   map[string]interface{} `json:"generator,omitempty"`
}

type BundleSecretTarget struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

type DependencyInsight struct {
	Name         string                               `json:"name"`
	Kind         string                               `json:"kind"`
	ResourceType string                               `json:"resource_type,omitempty"`
	Source       string                               `json:"source,omitempty"`
	Alternatives []string                             `json:"alternatives,omitempty"`
	Requirements *DependencyRequirementSummary        `json:"requirements,omitempty"`
	TierSupport  map[string]DependencyTierSupportView `json:"tier_support,omitempty"`
}

type DependencyRequirementSummary struct {
	RAMMB      int    `json:"ram_mb,omitempty"`
	DiskMB     int    `json:"disk_mb,omitempty"`
	CPUCores   int    `json:"cpu_cores,omitempty"`
	Network    string `json:"network,omitempty"`
	Source     string `json:"source,omitempty"`
	Confidence string `json:"confidence,omitempty"`
}

type DependencyTierSupportView struct {
	Supported    bool     `json:"supported"`
	FitnessScore float64  `json:"fitness_score"`
	Notes        string   `json:"notes,omitempty"`
	Reason       string   `json:"reason,omitempty"`
	Alternatives []string `json:"alternatives,omitempty"`
}

type TierAggregateView struct {
	FitnessScore          float64                       `json:"fitness_score"`
	DependencyCount       int                           `json:"dependency_count,omitempty"`
	BlockingDependencies  []string                      `json:"blocking_dependencies,omitempty"`
	EstimatedRequirements *DependencyRequirementSummary `json:"estimated_requirements,omitempty"`
}

type PromptMetadata struct {
	Label       string `json:"label,omitempty"`
	Description string `json:"description,omitempty"`
}

// Vault-specific data structures
type VaultSecretsStatus struct {
	TotalResources      int                   `json:"total_resources"`
	ConfiguredResources int                   `json:"configured_resources"`
	MissingSecrets      []VaultMissingSecret  `json:"missing_secrets"`
	ResourceStatuses    []VaultResourceStatus `json:"resource_statuses"`
	LastUpdated         time.Time             `json:"last_updated"`
}

type VaultMissingSecret struct {
	ResourceName string `json:"resource_name"`
	SecretName   string `json:"secret_name"`
	SecretPath   string `json:"secret_path"`
	Required     bool   `json:"required"`
	Description  string `json:"description"`
}

type VaultSecret struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Configured  bool   `json:"configured"`
	SecretType  string `json:"type"` // api_key, endpoint, quota
	// Guidance fields for user assistance
	DocumentationURL  string `json:"documentation_url,omitempty"`
	AcquisitionURL    string `json:"acquisition_url,omitempty"`
	SetupInstructions string `json:"setup_instructions,omitempty"`
	Example           string `json:"example,omitempty"`
	ValidationHint    string `json:"validation_hint,omitempty"`
}

type VaultResourceStatus struct {
	ResourceName    string        `json:"resource_name"`
	SecretsTotal    int           `json:"secrets_total"`
	SecretsFound    int           `json:"secrets_found"`
	SecretsMissing  int           `json:"secrets_missing"`
	SecretsOptional int           `json:"secrets_optional"`
	HealthStatus    string        `json:"health_status"` // healthy, degraded, critical
	LastChecked     time.Time     `json:"last_checked"`
	AllSecrets      []VaultSecret `json:"all_secrets,omitempty"` // All secrets for this resource
}

type VaultValidationSummary struct {
	ConfiguredCount int                  `json:"configured_count"`
	MissingSecrets  []VaultMissingSecret `json:"missing_secrets"`
}

type secretProvisionResult struct {
	EnvKey    string `json:"env_key"`
	VaultPath string `json:"vault_path"`
	VaultKey  string `json:"vault_key"`
	Status    string `json:"status"`
	Error     string `json:"error,omitempty"`
}

// Security scanning data structures
type SecurityVulnerability struct {
	ID             string    `json:"id"`
	ComponentType  string    `json:"component_type"` // "resource" or "scenario"
	ComponentName  string    `json:"component_name"` // Name of resource or scenario
	FilePath       string    `json:"file_path"`
	LineNumber     int       `json:"line_number"`
	Severity       string    `json:"severity"` // critical, high, medium, low
	Type           string    `json:"type"`     // sql_injection, hardcoded_secret, etc.
	Title          string    `json:"title"`
	Description    string    `json:"description"`
	Code           string    `json:"code"` // Affected code snippet
	Recommendation string    `json:"recommendation"`
	CanAutoFix     bool      `json:"can_auto_fix"`
	DiscoveredAt   time.Time `json:"discovered_at"`
	Status         string    `json:"status,omitempty"`
	Fingerprint    string    `json:"fingerprint,omitempty"`
	LastObservedAt time.Time `json:"last_observed_at,omitempty"`
}

type SecurityScanRun struct {
	ID              string    `json:"id"`
	ScanID          string    `json:"scan_id"`
	ComponentFilter string    `json:"component_filter"`
	ComponentType   string    `json:"component_type"`
	SeverityFilter  string    `json:"severity_filter"`
	FilesScanned    int       `json:"files_scanned"`
	FilesSkipped    int       `json:"files_skipped"`
	Vulnerabilities int       `json:"vulnerabilities"`
	RiskScore       int       `json:"risk_score"`
	DurationMs      int       `json:"duration_ms"`
	Status          string    `json:"status"`
	ErrorMessage    string    `json:"error_message,omitempty"`
	StartedAt       time.Time `json:"started_at"`
	CompletedAt     time.Time `json:"completed_at"`
}

type ResourceSecretDetail struct {
	ID              string            `json:"id"`
	SecretKey       string            `json:"secret_key"`
	SecretType      string            `json:"secret_type"`
	Description     string            `json:"description"`
	Classification  string            `json:"classification"`
	Required        bool              `json:"required"`
	OwnerTeam       string            `json:"owner_team"`
	OwnerContact    string            `json:"owner_contact"`
	TierStrategies  map[string]string `json:"tier_strategies"`
	ValidationState string            `json:"validation_state"`
	LastValidated   *time.Time        `json:"last_validated"`
}

type ResourceDetail struct {
	ResourceName        string                  `json:"resource_name"`
	ValidSecrets        int                     `json:"valid_secrets"`
	MissingSecrets      int                     `json:"missing_secrets"`
	TotalSecrets        int                     `json:"total_secrets"`
	LastValidation      *time.Time              `json:"last_validation"`
	Secrets             []ResourceSecretDetail  `json:"secrets"`
	OpenVulnerabilities []SecurityVulnerability `json:"open_vulnerabilities"`
}

type SecurityScanResult struct {
	ScanID            string                  `json:"scan_id"`
	ComponentFilter   string                  `json:"component_filter,omitempty"` // Optional filter for specific component
	ComponentType     string                  `json:"component_type,omitempty"`   // Filter by "resource" or "scenario"
	Vulnerabilities   []SecurityVulnerability `json:"vulnerabilities"`
	RiskScore         int                     `json:"risk_score"` // 0-100
	ScanDurationMs    int                     `json:"scan_duration_ms"`
	Recommendations   []RemediationSuggestion `json:"recommendations"`
	ComponentsSummary ComponentScanSummary    `json:"components_summary"`
	ScanMetrics       ScanMetrics             `json:"scan_metrics"`
	GeneratedAt       time.Time               `json:"generated_at,omitempty"`
}

type RemediationSuggestion struct {
	VulnerabilityType string   `json:"vulnerability_type"`
	Priority          string   `json:"priority"`
	Description       string   `json:"description"`
	FixCommand        string   `json:"fix_command,omitempty"`
	Documentation     string   `json:"documentation,omitempty"`
	AffectedFiles     []string `json:"affected_files"`
	Count             int      `json:"count"`
	EstimatedEffort   string   `json:"estimated_effort"`
}

type ComponentScanSummary struct {
	ResourcesScanned int `json:"resources_scanned"`
	ScenariosScanned int `json:"scenarios_scanned"`
	TotalComponents  int `json:"total_components"`
	ConfiguredCount  int `json:"configured_count"`
}

type ScanMetrics struct {
	FilesScanned       int      `json:"files_scanned"`
	FilesSkipped       int      `json:"files_skipped"`
	LargeFilesSkipped  int      `json:"large_files_skipped"`
	TimeoutOccurred    bool     `json:"timeout_occurred"`
	ScanErrors         []string `json:"scan_errors,omitempty"`
	ResourceScanTimeMs int      `json:"resource_scan_time_ms"`
	ScenarioScanTimeMs int      `json:"scenario_scan_time_ms"`
	TotalScanTimeMs    int      `json:"total_scan_time_ms"`
	// Progressive scanning fields
	ScanComplete        bool   `json:"scan_complete"`
	EstimatedTotalFiles int    `json:"estimated_total_files"`
	BatchesProcessed    int    `json:"batches_processed"`
	LastBatchTime       string `json:"last_batch_time,omitempty"`
}

// ProgressiveScanResult represents an ongoing scan session
type ProgressiveScanResult struct {
	ScanID            string                  `json:"scan_id"`
	Status            string                  `json:"status"` // "running", "completed", "failed"
	StartTime         time.Time               `json:"start_time"`
	LastUpdate        time.Time               `json:"last_update"`
	ComponentFilter   string                  `json:"component_filter,omitempty"`
	ComponentType     string                  `json:"component_type,omitempty"`
	Vulnerabilities   []SecurityVulnerability `json:"vulnerabilities"`
	RiskScore         int                     `json:"risk_score"`
	ComponentsSummary ComponentScanSummary    `json:"components_summary"`
	ScanMetrics       ScanMetrics             `json:"scan_metrics"`
	Recommendations   []RemediationSuggestion `json:"recommendations"`
	EstimatedProgress float64                 `json:"estimated_progress"` // 0.0-1.0
}

type ComplianceMetrics struct {
	VaultSecretsHealth   int `json:"vault_secrets_health"` // 0-100
	SecurityScore        int `json:"security_score"`       // 0-100
	OverallCompliance    int `json:"overall_compliance"`   // 0-100
	ConfiguredComponents int `json:"configured_components"`
	CriticalIssues       int `json:"critical_issues"`
	HighIssues           int `json:"high_issues"`
	MediumIssues         int `json:"medium_issues"`
	LowIssues            int `json:"low_issues"`
}

// API request/response types
type ScanRequest struct {
	Resources []string `json:"resources"`
	ScanType  string   `json:"scan_type"`
}

type ScanResponse struct {
	ScanID            string           `json:"scan_id"`
	DiscoveredSecrets []ResourceSecret `json:"discovered_secrets"`
	ScanDurationMs    int              `json:"scan_duration_ms"`
	ResourcesScanned  []string         `json:"resources_scanned"`
}

type ValidationRequest struct {
	Resource string `json:"resource"`
}

type ValidationResponse struct {
	ValidationID   string                `json:"validation_id"`
	TotalSecrets   int                   `json:"total_secrets"`
	ValidSecrets   int                   `json:"valid_secrets"`
	MissingSecrets []SecretValidation    `json:"missing_secrets"`
	InvalidSecrets []SecretValidation    `json:"invalid_secrets"`
	HealthSummary  []SecretHealthSummary `json:"health_summary"`
}

type ProvisionRequest struct {
	Resource      string            `json:"resource"`
	Secrets       map[string]string `json:"secrets"`
	SecretKey     string            `json:"secret_key"`
	SecretValue   string            `json:"secret_value"`
	StorageMethod string            `json:"storage_method"`
}

type ProvisionResponse struct {
	Success       bool                    `json:"success"`
	Resource      string                  `json:"resource,omitempty"`
	StoredSecrets int                     `json:"stored_secrets"`
	VaultStored   int                     `json:"vault_stored"`
	Details       []secretProvisionResult `json:"details,omitempty"`
	Message       string                  `json:"message,omitempty"`
}

type DeploymentManifestRequest struct {
	Scenario        string   `json:"scenario"`
	Tier            string   `json:"tier"`
	Resources       []string `json:"resources"`
	IncludeOptional bool     `json:"include_optional"`
}
