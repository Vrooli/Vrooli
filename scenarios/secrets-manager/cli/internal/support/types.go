package support

import "time"

const CLIName = "secrets-manager"

type HealthResponse struct {
	Status       string    `json:"status"`
	Service      string    `json:"service"`
	Version      string    `json:"version"`
	Timestamp    time.Time `json:"timestamp"`
	Readiness    bool      `json:"readiness"`
	StatusNotes  []string  `json:"status_notes"`
	Dependencies struct {
		Database struct {
			Connected bool    `json:"connected"`
			LatencyMS float64 `json:"latency_ms"`
			Error     *struct {
				Code      string `json:"code"`
				Message   string `json:"message"`
				Category  string `json:"category"`
				Retryable bool   `json:"retryable"`
			} `json:"error"`
		} `json:"database"`
	} `json:"dependencies"`
}

type MissingCredential struct {
	ResourceName string `json:"resource_name"`
	SecretName   string `json:"secret_name"`
	SecretPath   string `json:"secret_path"`
	Required     bool   `json:"required"`
	Description  string `json:"description"`
}

type CredentialResourceStatus struct {
	ResourceName    string    `json:"resource_name"`
	SecretsTotal    int       `json:"secrets_total"`
	SecretsFound    int       `json:"secrets_found"`
	SecretsMissing  int       `json:"secrets_missing"`
	SecretsOptional int       `json:"secrets_optional"`
	HealthStatus    string    `json:"health_status"`
	LastChecked     time.Time `json:"last_checked"`
}

type CredentialCoverageStatus struct {
	TotalResources      int                        `json:"total_resources"`
	ConfiguredResources int                        `json:"configured_resources"`
	MissingSecrets      []MissingCredential        `json:"missing_secrets"`
	ResourceStatuses    []CredentialResourceStatus `json:"resource_statuses"`
	LastUpdated         time.Time                  `json:"last_updated"`
}

type SecretValidation struct {
	ID               string `json:"id"`
	ValidationStatus string `json:"validation_status"`
	ErrorMessage     string `json:"error_message"`
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

type ValidationResponse struct {
	ValidationID   string                `json:"validation_id"`
	TotalSecrets   int                   `json:"total_secrets"`
	ValidSecrets   int                   `json:"valid_secrets"`
	MissingSecrets []SecretValidation    `json:"missing_secrets"`
	InvalidSecrets []SecretValidation    `json:"invalid_secrets"`
	HealthSummary  []SecretHealthSummary `json:"health_summary"`
}

type ProvisionDetail struct {
	EnvKey    string `json:"env_key"`
	LogicalID string `json:"logical_id"`
	Field     string `json:"field"`
	Status    string `json:"status"`
	Error     string `json:"error"`
}

type ProvisionResponse struct {
	Success       bool              `json:"success"`
	Resource      string            `json:"resource"`
	StoredSecrets int               `json:"stored_secrets"`
	Details       []ProvisionDetail `json:"details"`
	Message       string            `json:"message"`
}

type ComplianceResponse struct {
	OverallScore             int            `json:"overall_score"`
	CredentialCoverageHealth int            `json:"credential_coverage_health"`
	VulnerabilitySummary     map[string]int `json:"vulnerability_summary"`
	RemediationProgress      struct {
		ConfiguredComponents     int `json:"configured_components"`
		CriticalIssues           int `json:"critical_issues"`
		HighIssues               int `json:"high_issues"`
		MediumIssues             int `json:"medium_issues"`
		LowIssues                int `json:"low_issues"`
		SecurityScore            int `json:"security_score"`
		CredentialCoverageHealth int `json:"credential_coverage_health"`
		OverallCompliance        int `json:"overall_compliance"`
	} `json:"remediation_progress"`
	TotalResources       int       `json:"total_resources"`
	ConfiguredResources  int       `json:"configured_resources"`
	TotalVulnerabilities int       `json:"total_vulnerabilities"`
	LastUpdated          time.Time `json:"last_updated"`
}

type SecurityVulnerability struct {
	ID             string    `json:"id"`
	ComponentType  string    `json:"component_type"`
	ComponentName  string    `json:"component_name"`
	FilePath       string    `json:"file_path"`
	LineNumber     int       `json:"line_number"`
	Severity       string    `json:"severity"`
	Type           string    `json:"type"`
	Title          string    `json:"title"`
	Description    string    `json:"description"`
	Recommendation string    `json:"recommendation"`
	CanAutoFix     bool      `json:"can_auto_fix"`
	Status         string    `json:"status"`
	Fingerprint    string    `json:"fingerprint"`
	DiscoveredAt   time.Time `json:"discovered_at"`
	LastObservedAt time.Time `json:"last_observed_at"`
}

type VulnerabilityResponse struct {
	Vulnerabilities []SecurityVulnerability `json:"vulnerabilities"`
	TotalCount      int                     `json:"total_count"`
	ScanMetadata    struct {
		ScanID       string    `json:"scan_id"`
		ScanDuration int       `json:"scan_duration"`
		RiskScore    int       `json:"risk_score"`
		Component    string    `json:"component"`
		Severity     string    `json:"severity"`
		Timestamp    time.Time `json:"timestamp"`
		Mode         string    `json:"mode"`
	} `json:"scan_metadata"`
	Recommendations []struct {
		VulnerabilityType string `json:"vulnerability_type"`
		Description       string `json:"description"`
		Priority          string `json:"priority"`
	} `json:"recommendations"`
}

type SecurityScanResponse struct {
	ScanID          string                  `json:"scan_id"`
	ComponentFilter string                  `json:"component_filter"`
	ComponentType   string                  `json:"component_type"`
	Vulnerabilities []SecurityVulnerability `json:"vulnerabilities"`
	RiskScore       int                     `json:"risk_score"`
	ScanDurationMS  int                     `json:"scan_duration_ms"`
	Recommendations []struct {
		VulnerabilityType string   `json:"vulnerability_type"`
		Priority          string   `json:"priority"`
		Description       string   `json:"description"`
		AffectedFiles     []string `json:"affected_files"`
		Count             int      `json:"count"`
	} `json:"recommendations"`
	ComponentsSummary struct {
		ResourcesScanned int `json:"resources_scanned"`
		ScenariosScanned int `json:"scenarios_scanned"`
		TotalComponents  int `json:"total_components"`
		ConfiguredCount  int `json:"configured_count"`
	} `json:"components_summary"`
	ScanMetrics struct {
		FilesScanned       int      `json:"files_scanned"`
		FilesSkipped       int      `json:"files_skipped"`
		LargeFilesSkipped  int      `json:"large_files_skipped"`
		TimeoutOccurred    bool     `json:"timeout_occurred"`
		ScanErrors         []string `json:"scan_errors"`
		ResourceScanTimeMS int      `json:"resource_scan_time_ms"`
		ScenarioScanTimeMS int      `json:"scenario_scan_time_ms"`
		TotalScanTimeMS    int      `json:"total_scan_time_ms"`
		ScanComplete       bool     `json:"scan_complete"`
	} `json:"scan_metrics"`
	GeneratedAt time.Time `json:"generated_at"`
}

type VulnerabilityStatusResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type VulnerabilityFixResponse struct {
	Status          string    `json:"status"`
	FixRequestID    string    `json:"fix_request_id"`
	Vulnerabilities int       `json:"vulnerabilities"`
	Message         string    `json:"message"`
	Timestamp       time.Time `json:"timestamp"`
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

type DeploymentManifestResponse struct {
	Scenario    string    `json:"scenario"`
	Tier        string    `json:"tier"`
	GeneratedAt time.Time `json:"generated_at"`
	Resources   []string  `json:"resources"`
	Secrets     []struct {
		ResourceName      string         `json:"resource_name"`
		SecretKey         string         `json:"secret_key"`
		SecretType        string         `json:"secret_type"`
		Required          bool           `json:"required"`
		Classification    string         `json:"classification"`
		HandlingStrategy  string         `json:"handling_strategy"`
		FallbackStrategy  string         `json:"fallback_strategy"`
		RequiresUserInput bool           `json:"requires_user_input"`
		Prompt            map[string]any `json:"prompt"`
	} `json:"secrets"`
	Summary struct {
		TotalSecrets          int               `json:"total_secrets"`
		StrategizedSecrets    int               `json:"strategized_secrets"`
		RequiresAction        int               `json:"requires_action"`
		BlockingSecrets       []string          `json:"blocking_secrets"`
		BlockingSecretDetails []map[string]any  `json:"blocking_secret_details"`
		ClassificationWeights map[string]int    `json:"classification_weights"`
		StrategyBreakdown     map[string]int    `json:"strategy_breakdown"`
		ScopeReadiness        map[string]string `json:"scope_readiness"`
	} `json:"summary"`
}

type DeploymentReadinessResponse struct {
	Scenario    string    `json:"scenario"`
	Tier        string    `json:"tier"`
	Resources   []string  `json:"resources"`
	GeneratedAt time.Time `json:"generated_at"`
	Summary     struct {
		TotalSecrets       int      `json:"total_secrets"`
		StrategizedSecrets int      `json:"strategized_secrets"`
		RequiresAction     int      `json:"requires_action"`
		BlockingSecrets    []string `json:"blocking_secrets"`
	} `json:"summary"`
}

type ScenarioSummary struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Version     string   `json:"version"`
	Status      string   `json:"status"`
	Tags        []string `json:"tags"`
	Path        string   `json:"path"`
}

type ScenarioListResponse struct {
	Scenarios []ScenarioSummary `json:"scenarios"`
	Count     int               `json:"count"`
}

type CampaignSummary struct {
	ID         string    `json:"id"`
	Scenario   string    `json:"scenario"`
	Tier       string    `json:"tier"`
	Status     string    `json:"status"`
	Progress   int       `json:"progress"`
	Blockers   int       `json:"blockers"`
	UpdatedAt  time.Time `json:"updated_at"`
	NextAction string    `json:"next_action"`
	LastStep   string    `json:"last_step"`
	Summary    *struct {
		StrategizedSecrets int `json:"strategized_secrets"`
		TotalSecrets       int `json:"total_secrets"`
		RequiresAction     int `json:"requires_action"`
	} `json:"summary"`
}

type CampaignListResponse struct {
	Campaigns []CampaignSummary `json:"campaigns"`
	Count     int               `json:"count"`
}

type ScenarioSecretOverride struct {
	ID                string    `json:"id"`
	ScenarioName      string    `json:"scenario_name"`
	ResourceSecretID  string    `json:"resource_secret_id"`
	ResourceName      string    `json:"resource_name"`
	SecretKey         string    `json:"secret_key"`
	Tier              string    `json:"tier"`
	HandlingStrategy  *string   `json:"handling_strategy"`
	FallbackStrategy  *string   `json:"fallback_strategy"`
	RequiresUserInput *bool     `json:"requires_user_input"`
	PromptLabel       *string   `json:"prompt_label"`
	PromptDescription *string   `json:"prompt_description"`
	OverrideReason    *string   `json:"override_reason"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type OverridesListResponse struct {
	Scenario  string                   `json:"scenario"`
	Tier      string                   `json:"tier"`
	Overrides []ScenarioSecretOverride `json:"overrides"`
	Count     int                      `json:"count"`
}

type EffectiveSecretStrategy struct {
	ResourceName      string   `json:"resource_name"`
	SecretKey         string   `json:"secret_key"`
	Tier              string   `json:"tier"`
	HandlingStrategy  string   `json:"handling_strategy"`
	FallbackStrategy  string   `json:"fallback_strategy"`
	RequiresUserInput bool     `json:"requires_user_input"`
	PromptLabel       string   `json:"prompt_label"`
	PromptDescription string   `json:"prompt_description"`
	IsOverridden      bool     `json:"is_overridden"`
	OverriddenFields  []string `json:"overridden_fields"`
	OverrideReason    string   `json:"override_reason"`
}

type EffectiveStrategiesResponse struct {
	Scenario   string                    `json:"scenario"`
	Tier       string                    `json:"tier"`
	Strategies []EffectiveSecretStrategy `json:"strategies"`
	Count      int                       `json:"count"`
}

type OverrideMutationResponse struct {
	Success        bool                    `json:"success"`
	Message        string                  `json:"message"`
	Copied         int                     `json:"copied"`
	Deleted        int                     `json:"deleted"`
	WouldDelete    int                     `json:"would_delete"`
	SourceTier     string                  `json:"source_tier"`
	TargetTier     string                  `json:"target_tier"`
	SourceScenario string                  `json:"source_scenario"`
	TargetScenario string                  `json:"target_scenario"`
	Tier           string                  `json:"tier"`
	Overwrite      bool                    `json:"overwrite"`
	Orphans        []OrphanOverrideSummary `json:"orphans"`
	OrphanIDs      []string                `json:"orphan_ids"`
}

type OrphanOverrideSummary struct {
	Override ScenarioSecretOverride `json:"override"`
	Reason   string                 `json:"reason"`
}

type OrphansResponse struct {
	Orphans []OrphanOverrideSummary `json:"orphans"`
	Count   int                     `json:"count"`
}
