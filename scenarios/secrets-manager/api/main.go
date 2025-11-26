package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/lib/pq"
)

// Package-level logger
var logger *Logger

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

const scanCacheTTL = 60 * time.Second

type cachedSecurityScan struct {
	key     string
	result  *SecurityScanResult
	expires time.Time
}

var (
	securityScanCache     = map[string]cachedSecurityScan{}
	securityScanCacheMu   sync.Mutex
	scanRefreshInFlight   = map[string]bool{}
	scanRefreshInFlightMu sync.Mutex
)

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

type DeploymentManifest struct {
	Scenario            string                       `json:"scenario"`
	Tier                string                       `json:"tier"`
	GeneratedAt         time.Time                    `json:"generated_at"`
	Resources           []string                     `json:"resources"`
	Secrets             []DeploymentSecretEntry      `json:"secrets"`
	Summary             DeploymentSummary            `json:"summary"`
	Dependencies        []DependencyInsight          `json:"dependencies,omitempty"`
	TierAggregates      map[string]TierAggregateView `json:"tier_aggregates,omitempty"`
	AnalyzerGeneratedAt *time.Time                   `json:"analyzer_generated_at,omitempty"`
}

type DeploymentSummary struct {
	TotalSecrets          int               `json:"total_secrets"`
	StrategizedSecrets    int               `json:"strategized_secrets"`
	RequiresAction        int               `json:"requires_action"`
	BlockingSecrets       []string          `json:"blocking_secrets"`
	ClassificationWeights map[string]int    `json:"classification_weights"`
	StrategyBreakdown     map[string]int    `json:"strategy_breakdown"`
	ScopeReadiness        map[string]string `json:"scope_readiness"`
}

type DeploymentSecretEntry struct {
	ResourceName      string                 `json:"resource_name"`
	SecretKey         string                 `json:"secret_key"`
	SecretType        string                 `json:"secret_type"`
	Required          bool                   `json:"required"`
	Classification    string                 `json:"classification"`
	Description       string                 `json:"description,omitempty"`
	OwnerTeam         string                 `json:"owner_team,omitempty"`
	OwnerContact      string                 `json:"owner_contact,omitempty"`
	HandlingStrategy  string                 `json:"handling_strategy"`
	FallbackStrategy  string                 `json:"fallback_strategy,omitempty"`
	RequiresUserInput bool                   `json:"requires_user_input"`
	Prompt            *PromptMetadata        `json:"prompt,omitempty"`
	GeneratorTemplate map[string]interface{} `json:"generator_template,omitempty"`
	BundleHints       map[string]interface{} `json:"bundle_hints,omitempty"`
	TierStrategies    map[string]string      `json:"tier_strategies,omitempty"`
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

type OrientationSummary struct {
	HeroStats             HeroStats                `json:"hero_stats"`
	Journeys              []JourneyCard            `json:"journeys"`
	TierReadiness         []TierReadiness          `json:"tier_readiness"`
	ResourceInsights      []ResourceInsight        `json:"resource_insights"`
	VulnerabilityInsights []VulnerabilityHighlight `json:"vulnerability_insights"`
	UpdatedAt             time.Time                `json:"updated_at"`
}

type HeroStats struct {
	VaultConfigured int     `json:"vault_configured"`
	VaultTotal      int     `json:"vault_total"`
	MissingSecrets  int     `json:"missing_secrets"`
	RiskScore       int     `json:"risk_score"`
	OverallScore    int     `json:"overall_score"`
	LastScan        string  `json:"last_scan"`
	ReadinessLabel  string  `json:"readiness_label"`
	Confidence      float64 `json:"confidence"`
}

type JourneyCard struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Status      string   `json:"status"`
	CtaLabel    string   `json:"cta_label"`
	CtaAction   string   `json:"cta_action"`
	Primers     []string `json:"primers"`
	Badge       string   `json:"badge"`
}

type TierReadiness struct {
	Tier                 string   `json:"tier"`
	Label                string   `json:"label"`
	Strategized          int      `json:"strategized"`
	Total                int      `json:"total"`
	ReadyPercent         int      `json:"ready_percent"`
	BlockingSecretSample []string `json:"blocking_secret_sample"`
}

type ResourceInsight struct {
	ResourceName   string                  `json:"resource_name"`
	TotalSecrets   int                     `json:"total_secrets"`
	ValidSecrets   int                     `json:"valid_secrets"`
	MissingSecrets int                     `json:"missing_secrets"`
	InvalidSecrets int                     `json:"invalid_secrets"`
	LastValidation *time.Time              `json:"last_validation"`
	Secrets        []ResourceSecretInsight `json:"secrets"`
}

type ResourceSecretInsight struct {
	SecretKey      string            `json:"secret_key"`
	SecretType     string            `json:"secret_type"`
	Classification string            `json:"classification"`
	Required       bool              `json:"required"`
	TierStrategies map[string]string `json:"tier_strategies"`
}

type VulnerabilityHighlight struct {
	Severity string `json:"severity"`
	Count    int    `json:"count"`
	Message  string `json:"message"`
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

// Database connection
var db *sql.DB
var scanner *SecretScanner
var validator *SecretValidator

// Progressive scan management
var activeScansMutex sync.RWMutex
var activeScans = make(map[string]*ProgressiveScanResult)

func initDB() {
	var err error

	// Database configuration - support both POSTGRES_URL and individual components
	connStr := os.Getenv("POSTGRES_URL")
	if connStr == "" {
		// Try to build from individual components - REQUIRED, no defaults
		dbHost := os.Getenv("POSTGRES_HOST")
		dbPort := os.Getenv("POSTGRES_PORT")
		dbUser := os.Getenv("POSTGRES_USER")
		dbPassword := os.Getenv("POSTGRES_PASSWORD")
		dbName := os.Getenv("POSTGRES_DB")

		if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
			log.Fatal("❌ Database configuration missing. Provide POSTGRES_URL or all required database connection parameters (HOST, PORT, USER, PASSWORD, DB)")
		}

		connStr = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			dbHost, dbPort, dbUser, dbPassword, dbName)
	}

	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Implement exponential backoff for database connection
	maxRetries := 10
	baseDelay := 1 * time.Second
	maxDelay := 30 * time.Second

	logger.Info("🔄 Attempting database connection with exponential backoff...")
	logger.Info("📆 Database URL configured")

	var pingErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		pingErr = db.Ping()
		if pingErr == nil {
			logger.Info("✅ Database connected successfully on attempt %d", attempt+1)
			break
		}

		// Calculate exponential backoff delay
		delay := time.Duration(math.Min(
			float64(baseDelay)*math.Pow(2, float64(attempt)),
			float64(maxDelay),
		))

		// Add random jitter to prevent thundering herd
		jitter := time.Duration(rand.Float64() * float64(delay) * 0.25)
		actualDelay := delay + jitter

		logger.Warning("Connection attempt %d/%d failed: %v", attempt+1, maxRetries, pingErr)
		logger.Info("⏳ Waiting %v before next attempt", actualDelay)

		// Provide detailed status every few attempts
		if attempt > 0 && attempt%3 == 0 {
			logger.Info("📈 Retry progress:")
			logger.Info("   - Attempts made: %d/%d", attempt+1, maxRetries)
			logger.Info("   - Total wait time: ~%v", time.Duration(attempt*2)*baseDelay)
			logger.Info("   - Current delay: %v", actualDelay)
		}

		time.Sleep(actualDelay)
	}

	if pingErr != nil {
		log.Fatalf("❌ Database connection failed after %d attempts: %v", maxRetries, pingErr)
	}

	logger.Info("🎉 Database connection pool established successfully!")

	// Initialize scanner and validator components for downstream handlers
	scanner = NewSecretScanner(db)
	validator = NewSecretValidator(db)
}

// getEnvOrDefault removed to prevent hardcoded defaults

// Vault integration - uses resource-vault CLI commands to get secrets status
func getVaultSecretsStatus(resourceFilter string) (*VaultSecretsStatus, error) {
	// First try using resource-vault CLI directly
	status, err := getVaultSecretsStatusFromCLI(resourceFilter)
	if err != nil {
		logger.Info("resource-vault CLI failed: %v, using fallback implementation", err)
		// Fallback to direct scanning
		return getVaultSecretsStatusFallback(resourceFilter)
	}
	return status, nil
}

// getVaultSecretsStatusFromCLI uses resource-vault CLI commands
func getVaultSecretsStatusFromCLI(resourceFilter string) (*VaultSecretsStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Use resource-vault secrets validate to get status
	var cmd *exec.Cmd
	if resourceFilter != "" {
		cmd = exec.CommandContext(ctx, "resource-vault", "secrets", "check", resourceFilter)
	} else {
		cmd = exec.CommandContext(ctx, "resource-vault", "secrets", "validate")
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("resource-vault command failed: %w", err)
	}

	// Parse the output from resource-vault
	status := parseVaultCLIOutput(string(output), resourceFilter)
	status.LastUpdated = time.Now()

	return status, nil
}

// parseVaultCLIOutput parses resource-vault CLI output into structured data
func parseVaultCLIOutput(output, resourceFilter string) *VaultSecretsStatus {
	status := &VaultSecretsStatus{
		MissingSecrets:   []VaultMissingSecret{},
		ResourceStatuses: []VaultResourceStatus{},
	}

	lines := strings.Split(output, "\n")
	var currentResource string
	resourceCount := 0
	configuredCount := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse test format: "Resource: postgres" or "Status: Configured"
		if strings.HasPrefix(line, "Resource:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				currentResource = strings.TrimSpace(parts[1])
				resourceCount++
			}
		}

		if strings.HasPrefix(line, "Status:") && currentResource != "" {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				statusVal := strings.TrimSpace(parts[1])
				if strings.EqualFold(statusVal, "Configured") {
					configuredCount++
				}
			}
		}

		// Parse test format: "- DATABASE_URL (configured)" or "- OPENAI_API_KEY (required)"
		if strings.HasPrefix(line, "-") && currentResource != "" {
			// Check if it's a missing secret (contains "required" or "optional" but not "configured")
			if strings.Contains(line, "MISSING") || (strings.Contains(line, "(required)") || strings.Contains(line, "(optional)")) && !strings.Contains(line, "(configured)") {
				parts := strings.Split(line, "(")
				if len(parts) >= 1 {
					secretName := strings.TrimSpace(strings.TrimPrefix(parts[0], "-"))
					missing := VaultMissingSecret{
						ResourceName: currentResource,
						SecretName:   secretName,
						SecretPath:   fmt.Sprintf("secret/%s/%s", currentResource, secretName),
						Required:     strings.Contains(line, "(required)"),
						Description:  fmt.Sprintf("Missing required secret for %s", currentResource),
					}
					status.MissingSecrets = append(status.MissingSecrets, missing)
				}
			}
		}

		// Parse production format: "✓ postgres: 3 secrets defined"
		if strings.Contains(line, "✓") && strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) >= 2 {
				resourceName := strings.TrimSpace(strings.Replace(parts[0], "✓", "", 1))
				currentResource = resourceName
				resourceCount++

				// Check if all secrets are configured (no missing indicators)
				if !strings.Contains(parts[1], "MISSING") {
					configuredCount++
				}
			}
		}

		// Look for missing secrets like "✗ POSTGRES_PASSWORD: MISSING"
		if strings.Contains(line, "✗") && strings.Contains(line, "MISSING") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 && currentResource != "" {
				secretName := strings.TrimSpace(strings.Replace(parts[0], "✗", "", 1))
				missing := VaultMissingSecret{
					ResourceName: currentResource,
					SecretName:   secretName,
					SecretPath:   fmt.Sprintf("secret/%s/%s", currentResource, secretName),
					Required:     true,
					Description:  fmt.Sprintf("Missing required secret for %s", currentResource),
				}
				status.MissingSecrets = append(status.MissingSecrets, missing)
			}
		}

		// Parse test format: "Total Resources: 10" and "Configured: 7"
		if strings.HasPrefix(line, "Total Resources:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				if count, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
					status.TotalResources = count
				}
			}
		}
		if strings.HasPrefix(line, "Configured:") && !strings.Contains(line, "Fully") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				if count, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
					status.ConfiguredResources = count
				}
			}
		}

		// Look for "Fully configured: X" summary
		if strings.Contains(line, "Fully configured:") {
			fields := strings.Fields(line)
			for i, field := range fields {
				if field == "configured:" && i+1 < len(fields) {
					if count, err := strconv.Atoi(fields[i+1]); err == nil {
						configuredCount = count
					}
				}
			}
		}
	}

	// Only override totals if not already set from "Total Resources:" line
	if status.TotalResources == 0 {
		status.TotalResources = resourceCount
	}
	if status.ConfiguredResources == 0 {
		status.ConfiguredResources = configuredCount
	}

	return status
}

// Parse vault scan output to extract resource names
func parseVaultScanOutput(output string) []string {
	var resources []string
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Parse test format: "Found: postgres"
		if strings.HasPrefix(line, "Found:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				resourceName := strings.TrimSpace(parts[1])
				if resourceName != "" {
					resources = append(resources, resourceName)
				}
			}
		}

		// Parse production format: "✓ openrouter: 3 secrets defined"
		if strings.Contains(line, "✓") && strings.Contains(line, ":") && strings.Contains(line, "secrets defined") {
			parts := strings.Split(line, ":")
			if len(parts) > 0 {
				resourceName := strings.TrimSpace(strings.Replace(parts[0], "✓", "", -1))
				if resourceName != "" {
					resources = append(resources, resourceName)
				}
			}
		}
	}

	return resources
}

// Parse vault validation output
func parseVaultValidationOutput(output string) VaultValidationSummary {
	var summary VaultValidationSummary
	lines := strings.Split(output, "\n")

	configuredCount := 0
	var missingSecrets []VaultMissingSecret

	for _, line := range lines {
		// Look for "Fully configured: X"
		if strings.Contains(line, "Fully configured:") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				if count := parts[2]; count != "" {
					fmt.Sscanf(count, "%d", &configuredCount)
				}
			}
		}

		// Look for missing secret indicators (this would need more sophisticated parsing)
		if strings.Contains(line, "✗") && strings.Contains(line, "MISSING") {
			// Parse missing secret details - simplified for now
			missingSecrets = append(missingSecrets, VaultMissingSecret{
				ResourceName: "unknown", // Would need better parsing
				SecretName:   "unknown",
				Required:     true,
				Description:  strings.TrimSpace(line),
			})
		}
	}

	summary.ConfiguredCount = configuredCount
	summary.MissingSecrets = missingSecrets

	return summary
}

// Parse vault resource check output
func parseVaultResourceCheck(resourceName, output string) VaultResourceStatus {
	status := VaultResourceStatus{
		ResourceName: resourceName,
		LastChecked:  time.Now(),
	}

	lines := strings.Split(output, "\n")

	for _, line := range lines {
		if strings.Contains(line, "Found:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				fmt.Sscanf(parts[1], "%d", &status.SecretsFound)
			}
		}
		if strings.Contains(line, "Missing (required):") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				fmt.Sscanf(parts[2], "%d", &status.SecretsMissing)
			}
		}
		if strings.Contains(line, "Not set (optional):") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				fmt.Sscanf(parts[3], "%d", &status.SecretsOptional)
			}
		}
	}

	status.SecretsTotal = status.SecretsFound + status.SecretsMissing + status.SecretsOptional

	// Determine health status
	if status.SecretsMissing == 0 {
		status.HealthStatus = "healthy"
	} else if status.SecretsMissing <= 2 {
		status.HealthStatus = "degraded"
	} else {
		status.HealthStatus = "critical"
	}

	return status
}

// Progressive security scanner - returns immediate results and continues in background
func startProgressiveScan(componentFilter, componentTypeFilter, severityFilter string) (*ProgressiveScanResult, error) {
	scanID := uuid.New().String()
	startTime := time.Now()

	// Initialize progressive scan
	progressiveScan := &ProgressiveScanResult{
		ScanID:          scanID,
		Status:          "running",
		StartTime:       startTime,
		LastUpdate:      startTime,
		ComponentFilter: componentFilter,
		ComponentType:   componentTypeFilter,
		Vulnerabilities: []SecurityVulnerability{},
		ScanMetrics: ScanMetrics{
			ScanErrors:       []string{},
			ScanComplete:     false,
			BatchesProcessed: 0,
			LastBatchTime:    startTime.Format(time.RFC3339),
		},
		EstimatedProgress: 0.0,
	}

	// Store in active scans
	activeScansMutex.Lock()
	activeScans[scanID] = progressiveScan
	activeScansMutex.Unlock()

	// Start background scanning
	go performProgressiveScan(progressiveScan, componentFilter, componentTypeFilter, severityFilter)

	// Return immediate partial results (empty initially)
	return progressiveScan, nil
}

// Background progressive scanning with batching
func performProgressiveScan(scan *ProgressiveScanResult, componentFilter, componentTypeFilter, severityFilter string) {
	defer func() {
		// Mark scan as complete
		activeScansMutex.Lock()
		scan.Status = "completed"
		scan.ScanMetrics.ScanComplete = true
		scan.LastUpdate = time.Now()
		scan.EstimatedProgress = 1.0
		activeScansMutex.Unlock()

		logger.Info("Progressive scan %s completed: %d vulnerabilities found", scan.ScanID, len(scan.Vulnerabilities))
	}()

	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			scan.Status = "failed"
			scan.ScanMetrics.ScanErrors = append(scan.ScanMetrics.ScanErrors, fmt.Sprintf("Failed to get user home directory: %v", err))
			return
		}
		vrooliRoot = filepath.Join(home, "Vrooli")
	}

	scenariosPath := filepath.Join(vrooliRoot, "scenarios")
	resourcesPath := filepath.Join(vrooliRoot, "resources")

	// Estimate total files for progress tracking
	estimatedFiles := estimateFileCount(scenariosPath, resourcesPath, componentTypeFilter)
	scan.ScanMetrics.EstimatedTotalFiles = estimatedFiles

	const batchSize = 150 // Process files in batches
	var allVulnerabilities []SecurityVulnerability
	var resourcesScanned, scenariosScanned int

	// TODO: Progressive scanning implementation - for now use existing method
	result, err := scanComponentsForVulnerabilities(componentFilter, componentTypeFilter, severityFilter)
	if err != nil {
		scan.Status = "failed"
		scan.ScanMetrics.ScanErrors = append(scan.ScanMetrics.ScanErrors, fmt.Sprintf("Scan failed: %v", err))
		return
	}

	allVulnerabilities = result.Vulnerabilities
	resourcesScanned = result.ComponentsSummary.ResourcesScanned
	scenariosScanned = result.ComponentsSummary.ScenariosScanned

	// Final update
	activeScansMutex.Lock()
	scan.Vulnerabilities = allVulnerabilities
	scan.RiskScore = calculateRiskScore(allVulnerabilities)
	scan.Recommendations = generateRemediationSuggestions(allVulnerabilities)
	scan.ComponentsSummary = ComponentScanSummary{
		ResourcesScanned: resourcesScanned,
		ScenariosScanned: scenariosScanned,
		TotalComponents:  resourcesScanned + scenariosScanned,
	}
	scan.ScanMetrics.TotalScanTimeMs = int(time.Since(scan.StartTime).Milliseconds())
	activeScansMutex.Unlock()
}

// Original function modified to work with the new progressive system
func scanComponentsForVulnerabilities(componentFilter, componentTypeFilter, severityFilter string) (*SecurityScanResult, error) {
	cacheKey := buildScanCacheKey(componentFilter, componentTypeFilter, severityFilter)
	if cached := getCachedSecurityScan(cacheKey); cached != nil {
		return cached, nil
	}

	if os.Getenv("SECRETS_MANAGER_TEST_MODE") == "true" {
		result := buildEmptySecurityScan(componentFilter, componentTypeFilter, severityFilter, true)
		storeCachedSecurityScan(cacheKey, result)
		return result, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if stored, err := loadPersistedSecurityScan(ctx, componentFilter, componentTypeFilter, severityFilter); err == nil && stored != nil {
		storeCachedSecurityScan(cacheKey, stored)
		scheduleSecurityScanRefresh(componentFilter, componentTypeFilter, severityFilter, cacheKey)
		return stored, nil
	} else if err != nil {
		logger.Warning("failed to load persisted security scan: %v", err)
	}

	scheduleSecurityScanRefresh(componentFilter, componentTypeFilter, severityFilter, cacheKey)
	logger.Info("🌀 Vulnerability scan warming for key %s", cacheKey)
	return buildEmptySecurityScan(componentFilter, componentTypeFilter, severityFilter, false), nil
}

func performSecurityScan(componentFilter, componentTypeFilter, severityFilter string) (*SecurityScanResult, error) {
	startTime := time.Now()
	scanID := uuid.New().String()

	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user home directory: %w", err)
		}
		vrooliRoot = filepath.Join(home, "Vrooli")
	}

	scenariosPath := filepath.Join(vrooliRoot, "scenarios")
	resourcesPath := filepath.Join(vrooliRoot, "resources")
	var vulnerabilities []SecurityVulnerability
	var resourcesScanned, scenariosScanned int
	seenResources := make(map[string]bool)
	seenScenarios := make(map[string]bool)

	// Initialize scan metrics
	metrics := ScanMetrics{
		ScanErrors: []string{},
	}

	// Scan scenarios if not filtering for resources only
	if componentTypeFilter == "" || componentTypeFilter == "scenario" {
		scenarioStartTime := time.Now()
		// Create timeout context for scenario scanning
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second) // Increased timeout for large projects
		defer cancel()

		scenarioFilesScanned := 0
		maxScenarioFiles := 25000                // Significantly increased limit for large projects
		filesPerScenario := make(map[string]int) // Track files per scenario to balance scanning

		err := filepath.WalkDir(scenariosPath, func(path string, d os.DirEntry, err error) error {
			// Check timeout
			select {
			case <-ctx.Done():
				logger.Info("Scenario scanning timed out after 120 seconds")
				metrics.TimeoutOccurred = true
				metrics.ScanErrors = append(metrics.ScanErrors, "Scenario scanning timeout after 120 seconds")
				return filepath.SkipAll
			default:
			}

			if err != nil {
				metrics.FilesSkipped++
				return nil // Skip files we can't read
			}

			// Only scan source files in scenarios
			if d.IsDir() {
				return nil
			}

			// Extract scenario name first to apply per-scenario limits
			relPath, _ := filepath.Rel(scenariosPath, path)
			pathParts := strings.Split(relPath, string(filepath.Separator))
			if len(pathParts) == 0 {
				return nil
			}
			scenarioName := pathParts[0]

			// Limit files scanned
			if scenarioFilesScanned >= maxScenarioFiles {
				logger.Info("Scenario scanning stopped after %d files (limit reached)", maxScenarioFiles)
				metrics.ScanErrors = append(metrics.ScanErrors, fmt.Sprintf("Scenario file limit reached (%d files)", maxScenarioFiles))
				return filepath.SkipAll
			}

			// Limit files per scenario to ensure we scan more scenarios
			const maxFilesPerScenario = 200 // Increased limit per scenario for large projects
			if filesPerScenario[scenarioName] >= maxFilesPerScenario {
				return nil // Skip additional files from this scenario
			}

			// Check file size
			info, err := d.Info()
			if err == nil && info.Size() > 500*1024 { // Skip files larger than 500KB for scenarios
				metrics.LargeFilesSkipped++
				return nil
			}

			// Check if it's a source file we should scan
			ext := strings.ToLower(filepath.Ext(path))
			if ext != ".go" && ext != ".js" && ext != ".ts" && ext != ".py" && ext != ".sh" {
				return nil
			}

			// Skip if filtering by specific component
			if componentFilter != "" && componentFilter != scenarioName {
				return nil
			}

			// Track unique scenarios scanned
			if strings.Contains(path, "/"+scenarioName+"/") {
				// Only count each scenario once per scan
				if !seenScenarios[scenarioName] {
					scenariosScanned++
					seenScenarios[scenarioName] = true
				}
			}

			// Increment counters
			scenarioFilesScanned++
			filesPerScenario[scenarioName]++
			metrics.FilesScanned++

			// Scan file for vulnerabilities
			fileVulns, err := scanFileForVulnerabilities(path, "scenario", scenarioName)
			if err != nil {
				logger.Warning(" failed to scan scenario file %s: %v", path, err)
				metrics.ScanErrors = append(metrics.ScanErrors, fmt.Sprintf("Failed to scan %s: %v", filepath.Base(path), err))
				return nil
			}

			// Filter by severity if specified
			for _, vuln := range fileVulns {
				if severityFilter == "" || vuln.Severity == severityFilter {
					vulnerabilities = append(vulnerabilities, vuln)
				}
			}

			return nil
		})

		metrics.ScenarioScanTimeMs = int(time.Since(scenarioStartTime).Milliseconds())

		if err != nil {
			logger.Warning(" failed to walk scenarios directory: %v", err)
			metrics.ScanErrors = append(metrics.ScanErrors, fmt.Sprintf("Failed to walk scenarios directory: %v", err))
		}
	}

	// Scan resources if not filtering for scenarios only
	if componentTypeFilter == "" || componentTypeFilter == "resource" {
		resourceStartTime := time.Now()
		// Create timeout context for resource scanning
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second) // Increased timeout for large projects
		defer cancel()

		resourceFilesScanned := 0
		maxFiles := 15000                        // Significantly increased limit for large projects
		filesPerResource := make(map[string]int) // Track files per resource to balance scanning

		err := filepath.WalkDir(resourcesPath, func(path string, d os.DirEntry, err error) error {
			// Check timeout
			select {
			case <-ctx.Done():
				logger.Info("Resource scanning timed out after 90 seconds")
				metrics.TimeoutOccurred = true
				metrics.ScanErrors = append(metrics.ScanErrors, "Resource scanning timeout after 90 seconds")
				return filepath.SkipAll
			default:
			}

			if err != nil {
				metrics.FilesSkipped++
				return nil // Skip files we can't read
			}

			// Only scan config files in resources
			if d.IsDir() {
				return nil
			}

			// Extract resource name first to apply per-resource limits
			relPath, _ := filepath.Rel(resourcesPath, path)
			pathParts := strings.Split(relPath, string(filepath.Separator))
			if len(pathParts) == 0 {
				return nil
			}
			resourceName := pathParts[0]

			// Limit number of files scanned to prevent runaway scanning
			if resourceFilesScanned >= maxFiles {
				logger.Info("Resource scanning stopped after %d files (limit reached)", maxFiles)
				metrics.ScanErrors = append(metrics.ScanErrors, fmt.Sprintf("Resource file limit reached (%d files)", maxFiles))
				return filepath.SkipAll
			}

			// Limit files per resource to ensure we scan more resources
			const maxFilesPerResource = 100 // Increased limit per resource for large projects
			if filesPerResource[resourceName] >= maxFilesPerResource {
				return nil // Skip additional files from this resource
			}

			// Check file size to prevent scanning huge files
			info, err := d.Info()
			if err == nil && info.Size() > 200*1024 { // Skip files larger than 200KB for resources
				metrics.LargeFilesSkipped++
				return nil
			}

			// Check if it's a config file we should scan
			ext := strings.ToLower(filepath.Ext(path))
			if ext != ".sh" && ext != ".yaml" && ext != ".yml" && ext != ".json" && ext != ".env" {
				return nil
			}

			// Skip if filtering by specific component
			if componentFilter != "" && componentFilter != resourceName {
				return nil
			}

			// Track unique resources scanned
			if strings.Contains(path, "/"+resourceName+"/") {
				// Only count each resource once per scan
				if !seenResources[resourceName] {
					resourcesScanned++
					seenResources[resourceName] = true
				}
			}

			// Increment counters
			resourceFilesScanned++
			filesPerResource[resourceName]++
			metrics.FilesScanned++

			// Use optimized resource scanning with timeout
			fileVulns, err := scanResourceFileForVulnerabilities(ctx, path, "resource", resourceName)
			if err != nil {
				logger.Warning(" failed to scan resource file %s: %v", path, err)
				metrics.ScanErrors = append(metrics.ScanErrors, fmt.Sprintf("Failed to scan resource %s: %v", filepath.Base(path), err))
				return nil
			}

			// Filter by severity if specified
			for _, vuln := range fileVulns {
				if severityFilter == "" || vuln.Severity == severityFilter {
					vulnerabilities = append(vulnerabilities, vuln)
				}
			}

			return nil
		})

		metrics.ResourceScanTimeMs = int(time.Since(resourceStartTime).Milliseconds())

		if err != nil {
			logger.Warning(" failed to walk resources directory: %v", err)
			metrics.ScanErrors = append(metrics.ScanErrors, fmt.Sprintf("Failed to walk resources directory: %v", err))
		}
	}

	// Calculate risk score based on vulnerabilities
	riskScore := calculateRiskScore(vulnerabilities)

	// Generate remediation suggestions
	recommendations := generateRemediationSuggestions(vulnerabilities)

	// Finalize scan metrics
	totalScanTime := int(time.Since(startTime).Milliseconds())
	metrics.TotalScanTimeMs = totalScanTime

	// Log scan summary
	logger.Info("Vulnerability scan completed:")
	logger.Info("  📊 Total scan time: %dms", totalScanTime)
	logger.Info("  📁 Files scanned: %d", metrics.FilesScanned)
	logger.Info("  ⏭️  Files skipped: %d", metrics.FilesSkipped)
	logger.Info("  🔍 Vulnerabilities found: %d", len(vulnerabilities))

	completedAt := time.Now()
	if _, err := persistSecurityScan(context.Background(), scanID, componentFilter, componentTypeFilter, severityFilter, metrics, riskScore, vulnerabilities); err != nil {
		logger.Info("failed to persist scan run: %v", err)
	}
	logger.Info("  ⚠️  Scan errors: %d", len(metrics.ScanErrors))
	if metrics.TimeoutOccurred {
		logger.Info("  ⏰ Timeout occurred during scanning")
	}

	return &SecurityScanResult{
		ScanID:          scanID,
		ComponentFilter: componentFilter,
		ComponentType:   componentTypeFilter,
		Vulnerabilities: vulnerabilities,
		RiskScore:       riskScore,
		ScanDurationMs:  totalScanTime,
		Recommendations: recommendations,
		ComponentsSummary: ComponentScanSummary{
			ResourcesScanned: resourcesScanned,
			ScenariosScanned: scenariosScanned,
			TotalComponents:  resourcesScanned + scenariosScanned,
			ConfiguredCount:  0, // TODO: Calculate from vault status
		},
		ScanMetrics: metrics,
		GeneratedAt: completedAt,
	}, nil
}

func buildEmptySecurityScan(componentFilter, componentTypeFilter, severityFilter string, completed bool) *SecurityScanResult {
	metrics := ScanMetrics{
		ScanComplete: completed,
	}
	return &SecurityScanResult{
		ScanID:            uuid.New().String(),
		ComponentFilter:   componentFilter,
		ComponentType:     componentTypeFilter,
		Vulnerabilities:   []SecurityVulnerability{},
		RiskScore:         0,
		ScanDurationMs:    0,
		Recommendations:   generateRemediationSuggestions([]SecurityVulnerability{}),
		ComponentsSummary: ComponentScanSummary{},
		ScanMetrics:       metrics,
		GeneratedAt:       time.Now(),
	}
}

// Optimized resource scanning function with timeout protection
func scanResourceFileForVulnerabilities(ctx context.Context, filePath, componentType, componentName string) ([]SecurityVulnerability, error) {
	var vulnerabilities []SecurityVulnerability

	// Check timeout before starting
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("scanning timeout reached")
	default:
	}

	// Read file content with size limit (50KB max)
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	if len(content) > 50*1024 {
		return nil, fmt.Errorf("file too large: %d bytes", len(content))
	}

	contentStr := string(content)
	lines := strings.Split(contentStr, "\n")

	// Resource-specific vulnerability patterns (simplified, no AST parsing)
	resourcePatterns := []VulnerabilityPattern{
		{
			Type:           "hardcoded_secret_resource",
			Severity:       "critical",
			Pattern:        `(PASSWORD|SECRET|TOKEN|KEY|API_KEY)\s*=\s*[\"'](?!.*\$|.*env|.*getenv)[^\"']{8,}[\"']`,
			Description:    "Hardcoded secret found in resource configuration",
			Title:          "Hardcoded Secret in Resource",
			Recommendation: "Move secret to vault using resource-vault CLI",
			CanAutoFix:     false,
		},
		{
			Type:           "database_url_hardcoded",
			Severity:       "critical",
			Pattern:        `(DATABASE_URL|DB_URL|POSTGRES_URL)\s*=\s*[\"'](?!.*\$)[^\"']*://[^\"']*:[^\"']*@[^\"']+`,
			Description:    "Database URL with hardcoded credentials",
			Title:          "Hardcoded Database Credentials",
			Recommendation: "Use environment variables or vault for database credentials",
			CanAutoFix:     false,
		},
		{
			Type:           "missing_env_var_validation",
			Severity:       "medium",
			Pattern:        `\$\{?([A-Z_]+[A-Z0-9_]*)\}?(?!\s*:-|\s*\|\|)`,
			Description:    "Environment variable used without fallback or validation",
			Title:          "Missing Environment Variable Validation",
			Recommendation: "Add fallback values or validation for environment variables",
			CanAutoFix:     true,
		},
		{
			Type:           "weak_permissions",
			Severity:       "medium",
			Pattern:        `chmod\s+(777|666|755)`,
			Description:    "Potentially insecure file permissions",
			Title:          "Weak File Permissions",
			Recommendation: "Use more restrictive file permissions (644, 600, etc.)",
			CanAutoFix:     true,
		},
	}

	// Pattern-based scanning for resources
	for _, pattern := range resourcePatterns {
		// Check timeout during each pattern
		select {
		case <-ctx.Done():
			return vulnerabilities, fmt.Errorf("scanning timeout during pattern matching")
		default:
		}

		regex, err := regexp.Compile(pattern.Pattern)
		if err != nil {
			continue
		}

		matches := regex.FindAllStringIndex(contentStr, -1)
		for _, match := range matches {
			// Find line number
			lineNum := findLineNumber(contentStr, match[0])

			// Extract code snippet
			codeSnippet := extractCodeSnippet(lines, lineNum-1, 2)

			vulnerability := SecurityVulnerability{
				ID:             uuid.New().String(),
				ComponentType:  componentType,
				ComponentName:  componentName,
				FilePath:       filePath,
				LineNumber:     lineNum,
				Severity:       pattern.Severity,
				Type:           pattern.Type,
				Title:          pattern.Title,
				Description:    pattern.Description,
				Code:           codeSnippet,
				Recommendation: pattern.Recommendation,
				CanAutoFix:     pattern.CanAutoFix,
				DiscoveredAt:   time.Now(),
			}

			vulnerabilities = append(vulnerabilities, vulnerability)
		}
	}

	return vulnerabilities, nil
}

// Helper functions for progressive scanning

// estimateFileCount provides a quick count of scannable files
func estimateFileCount(scenariosPath, resourcesPath, componentTypeFilter string) int {
	count := 0

	if componentTypeFilter == "" || componentTypeFilter == "scenario" {
		filepath.WalkDir(scenariosPath, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(path))
			if ext == ".go" || ext == ".js" || ext == ".ts" || ext == ".py" || ext == ".sh" {
				count++
			}
			return nil
		})
	}

	if componentTypeFilter == "" || componentTypeFilter == "resource" {
		filepath.WalkDir(resourcesPath, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(path))
			if ext == ".go" || ext == ".js" || ext == ".ts" || ext == ".py" || ext == ".sh" || ext == ".yml" || ext == ".yaml" || ext == ".json" {
				count++
			}
			return nil
		})
	}

	return count
}

func scanResourceDirectory(resourceName, resourceDir string) ([]ResourceSecret, error) {
	var secrets []ResourceSecret

	// Patterns to look for environment variables and credentials
	envVarPatterns := []*regexp.Regexp{
		regexp.MustCompile(`\$\{([A-Z_]+[A-Z0-9_]*)\}`),           // ${VAR_NAME}
		regexp.MustCompile(`\$([A-Z_]+[A-Z0-9_]*)`),               // $VAR_NAME
		regexp.MustCompile(`([A-Z_]+[A-Z0-9_]*)=`),                // VAR_NAME=
		regexp.MustCompile(`env\.([A-Z_]+[A-Z0-9_]*)`),            // env.VAR_NAME
		regexp.MustCompile(`getenv\("([A-Z_]+[A-Z0-9_]*)"\)`),     // getenv("VAR_NAME")
		regexp.MustCompile(`os\.Getenv\("([A-Z_]+[A-Z0-9_]*)"\)`), // os.Getenv("VAR_NAME")
	}

	// Walk through resource directory
	err := filepath.WalkDir(resourceDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip files we can't read
		}

		// Skip directories and non-text files
		if d.IsDir() || !isTextFile(path) {
			return nil
		}

		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			return nil // Skip files we can't read
		}

		// Search for environment variables
		foundVars := make(map[string]bool)
		for _, pattern := range envVarPatterns {
			matches := pattern.FindAllStringSubmatch(string(content), -1)
			for _, match := range matches {
				if len(match) > 1 {
					varName := match[1]
					if !foundVars[varName] && isLikelySecret(varName) {
						foundVars[varName] = true

						secret := ResourceSecret{
							ID:                uuid.New().String(),
							ResourceName:      resourceName,
							SecretKey:         varName,
							SecretType:        classifySecretType(varName),
							Required:          isLikelyRequired(varName),
							Description:       stringPtr(fmt.Sprintf("Environment variable found in %s", filepath.Base(path))),
							ValidationPattern: nil,
							DocumentationURL:  nil,
							DefaultValue:      nil,
							CreatedAt:         time.Now(),
							UpdatedAt:         time.Now(),
						}
						secrets = append(secrets, secret)
					}
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return secrets, nil
}

func isTextFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	textExtensions := []string{".sh", ".bash", ".yml", ".yaml", ".json", ".env", ".conf", ".config", ".md", ".txt", ".go", ".js", ".py", ".dockerfile", ".sql"}

	for _, textExt := range textExtensions {
		if ext == textExt {
			return true
		}
	}

	return false
}

func isLikelySecret(varName string) bool {
	secretKeywords := []string{
		"PASSWORD", "PASS", "PWD", "SECRET", "KEY", "TOKEN", "AUTH", "API", "CREDENTIAL", "CERT", "TLS", "SSL", "PRIVATE",
	}

	upperVar := strings.ToUpper(varName)
	for _, keyword := range secretKeywords {
		if strings.Contains(upperVar, keyword) {
			return true
		}
	}

	// Also include common configuration variables that might be sensitive
	configKeywords := []string{"HOST", "PORT", "URL", "ADDR", "DATABASE", "DB", "USER", "NAMESPACE"}
	for _, keyword := range configKeywords {
		if strings.Contains(upperVar, keyword) {
			return true
		}
	}

	return false
}

func classifySecretType(varName string) string {
	upperVar := strings.ToUpper(varName)

	if strings.Contains(upperVar, "PASSWORD") || strings.Contains(upperVar, "PASS") || strings.Contains(upperVar, "PWD") {
		return "password"
	}
	if strings.Contains(upperVar, "TOKEN") {
		return "token"
	}
	if strings.Contains(upperVar, "KEY") && (strings.Contains(upperVar, "API") || strings.Contains(upperVar, "ACCESS")) {
		return "api_key"
	}
	if strings.Contains(upperVar, "SECRET") || strings.Contains(upperVar, "CREDENTIAL") {
		return "credential"
	}
	if strings.Contains(upperVar, "CERT") || strings.Contains(upperVar, "TLS") || strings.Contains(upperVar, "SSL") {
		return "certificate"
	}

	return "env_var"
}

func isLikelyRequired(varName string) bool {
	requiredKeywords := []string{"PASSWORD", "SECRET", "TOKEN", "KEY", "DATABASE", "DB", "HOST"}
	upperVar := strings.ToUpper(varName)

	for _, keyword := range requiredKeywords {
		if strings.Contains(upperVar, keyword) {
			return true
		}
	}

	return false
}

// Validation functions
func validateSecrets(resource string) (ValidationResponse, error) {
	if db == nil {
		return ValidationResponse{}, fmt.Errorf("database not initialized")
	}

	// Get all secrets for the resource (or all if empty)
	query := `
		SELECT rs.id, rs.resource_name, rs.secret_key, rs.secret_type, rs.required,
		       rs.description, rs.validation_pattern
		FROM resource_secrets rs
	`
	args := []interface{}{}

	if resource != "" {
		query += " WHERE rs.resource_name = $1"
		args = append(args, resource)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return ValidationResponse{}, err
	}
	defer rows.Close()

	var secrets []ResourceSecret
	for rows.Next() {
		var secret ResourceSecret
		err := rows.Scan(&secret.ID, &secret.ResourceName, &secret.SecretKey,
			&secret.SecretType, &secret.Required, &secret.Description, &secret.ValidationPattern)
		if err != nil {
			continue
		}
		secrets = append(secrets, secret)
	}

	// Validate each secret
	var validSecrets, totalSecrets int
	var missingSecrets, invalidSecrets []SecretValidation

	for _, secret := range secrets {
		totalSecrets++
		validation := validateSingleSecret(secret)

		// Store validation result
		storeValidationResult(validation)

		switch validation.ValidationStatus {
		case "valid":
			validSecrets++
		case "missing":
			missingSecrets = append(missingSecrets, validation)
		case "invalid":
			invalidSecrets = append(invalidSecrets, validation)
		}
	}

	// Get health summary
	healthSummary, _ := getHealthSummary()

	return ValidationResponse{
		ValidationID:   uuid.New().String(),
		TotalSecrets:   totalSecrets,
		ValidSecrets:   validSecrets,
		MissingSecrets: missingSecrets,
		InvalidSecrets: invalidSecrets,
		HealthSummary:  healthSummary,
	}, nil
}

func validateSingleSecret(secret ResourceSecret) SecretValidation {
	validation := SecretValidation{
		ID:                  uuid.New().String(),
		ResourceSecretID:    secret.ID,
		ValidationTimestamp: time.Now(),
	}

	// Check environment variable
	envValue := os.Getenv(secret.SecretKey)
	if envValue != "" {
		validation.ValidationMethod = "env"

		// Validate against pattern if provided
		if secret.ValidationPattern != nil {
			if matched, _ := regexp.MatchString(*secret.ValidationPattern, envValue); matched {
				validation.ValidationStatus = "valid"
			} else {
				validation.ValidationStatus = "invalid"
				validation.ErrorMessage = stringPtr("Value does not match required pattern")
			}
		} else {
			validation.ValidationStatus = "valid"
		}
	} else {
		// Check vault if available
		vaultValue, err := getVaultSecret(secret.SecretKey)
		if err == nil && vaultValue != "" {
			validation.ValidationMethod = "vault"
			validation.ValidationStatus = "valid"
		} else {
			validation.ValidationStatus = "missing"
			validation.ValidationMethod = "env"
			validation.ErrorMessage = stringPtr("Environment variable not set and not found in vault")
		}
	}

	return validation
}

func getVaultSecret(key string) (string, error) {
	// Use resource-vault CLI to get secret value
	cmd := exec.Command("resource-vault", "secrets", "get", key)

	// Set timeout for vault command
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd = exec.CommandContext(ctx, "resource-vault", "secrets", "get", key)

	output, err := cmd.Output()
	if err != nil {
		// Check if it's a timeout or command not found
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("vault command timeout")
		}

		// If exit error, likely secret not found or vault unavailable
		if exitErr, ok := err.(*exec.ExitError); ok {
			// resource-vault typically returns exit code 1 for not found
			if exitErr.ExitCode() == 1 {
				return "", fmt.Errorf("secret not found in vault")
			}
			return "", fmt.Errorf("vault command failed: %v", err)
		}

		return "", fmt.Errorf("failed to execute resource-vault command: %v", err)
	}

	// Clean up the output (remove trailing whitespace/newlines)
	secretValue := strings.TrimSpace(string(output))

	if secretValue == "" {
		return "", fmt.Errorf("empty secret value returned from vault")
	}

	return secretValue, nil
}

func getLocalSecretsPath() (string, error) {
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		vrooliRoot = filepath.Join(home, "Vrooli")
	}

	secretsDir := filepath.Join(vrooliRoot, ".vrooli")
	if err := os.MkdirAll(secretsDir, 0o755); err != nil {
		return "", err
	}

	return filepath.Join(secretsDir, "secrets.json"), nil
}

func loadLocalSecretsFile() (map[string]interface{}, error) {
	path, err := getLocalSecretsPath()
	if err != nil {
		return nil, err
	}

	store := map[string]interface{}{}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			store["_metadata"] = map[string]interface{}{
				"environment":  "development",
				"last_updated": time.Now().Format(time.RFC3339),
			}
			return store, nil
		}
		return nil, err
	}

	if len(data) == 0 {
		store["_metadata"] = map[string]interface{}{
			"environment":  "development",
			"last_updated": time.Now().Format(time.RFC3339),
		}
		return store, nil
	}

	if err := json.Unmarshal(data, &store); err != nil {
		return nil, err
	}

	return store, nil
}

func saveSecretsToLocalStore(secrets map[string]string) (int, error) {
	if len(secrets) == 0 {
		return 0, nil
	}

	store, err := loadLocalSecretsFile()
	if err != nil {
		return 0, err
	}

	for key, value := range secrets {
		store[key] = value
	}

	meta, ok := store["_metadata"].(map[string]interface{})
	if !ok || meta == nil {
		meta = map[string]interface{}{}
	}
	if _, exists := meta["environment"]; !exists {
		meta["environment"] = "development"
	}
	meta["last_updated"] = time.Now().Format(time.RFC3339)
	store["_metadata"] = meta

	path, err := getLocalSecretsPath()
	if err != nil {
		return 0, err
	}

	encoded, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return 0, err
	}

	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		return 0, err
	}

	return len(secrets), nil
}

func storeValidationResult(validation SecretValidation) {
	query := `
		INSERT INTO secret_validations (id, resource_secret_id, validation_status, 
			validation_method, validation_timestamp, error_message)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := db.Exec(query, validation.ID, validation.ResourceSecretID,
		validation.ValidationStatus, validation.ValidationMethod,
		validation.ValidationTimestamp, validation.ErrorMessage)
	if err != nil {
		logger.Info("Failed to store validation result: %v", err)
	}
}

func getHealthSummary() ([]SecretHealthSummary, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `SELECT * FROM secret_health_summary ORDER BY resource_name`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []SecretHealthSummary
	for rows.Next() {
		var summary SecretHealthSummary
		err := rows.Scan(&summary.ResourceName, &summary.TotalSecrets,
			&summary.RequiredSecrets, &summary.ValidSecrets,
			&summary.MissingRequiredSecrets, &summary.InvalidSecrets,
			&summary.LastValidation)
		if err != nil {
			continue
		}
		summaries = append(summaries, summary)
	}

	return summaries, nil
}

// HTTP Handlers
func healthHandler(w http.ResponseWriter, r *http.Request) {
	// Check database connectivity
	dbConnected := false
	var dbLatency float64
	var dbError map[string]interface{}

	if db != nil {
		start := time.Now()
		err := db.Ping()
		dbLatency = float64(time.Since(start).Milliseconds())

		if err == nil {
			dbConnected = true
		} else {
			dbError = map[string]interface{}{
				"code":      "CONNECTION_FAILED",
				"message":   err.Error(),
				"category":  "resource",
				"retryable": true,
			}
		}
	} else {
		dbError = map[string]interface{}{
			"code":      "NOT_INITIALIZED",
			"message":   "Database connection not initialized",
			"category":  "configuration",
			"retryable": false,
		}
	}

	// Determine overall status
	status := "healthy"
	readiness := true
	var statusNotes []string

	if !dbConnected {
		status = "degraded"
		statusNotes = append(statusNotes, "Database connectivity issues")
	}

	// Build response compliant with health-api.schema.json
	response := map[string]interface{}{
		"status":    status,
		"service":   "secrets-manager-api",
		"timestamp": time.Now().Format(time.RFC3339),
		"readiness": readiness,
		"version":   "1.0.0",
		"dependencies": map[string]interface{}{
			"database": map[string]interface{}{
				"connected":  dbConnected,
				"latency_ms": dbLatency,
				"error":      dbError,
			},
		},
	}

	if len(statusNotes) > 0 {
		response["status_notes"] = statusNotes
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Vault secrets status handler
func vaultSecretsStatusHandler(w http.ResponseWriter, r *http.Request) {
	resourceFilter := r.URL.Query().Get("resource")

	status, err := getVaultSecretsStatus(resourceFilter)
	if err != nil {
		logger.Info("Error getting vault status: %v, using mock data", err)
		// Use mock data as ultimate fallback
		status = getMockVaultStatus()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func orientationSummaryHandler(w http.ResponseWriter, r *http.Request) {
	summary, err := buildOrientationSummary(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to build orientation summary: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// Security scan handler
func securityScanHandler(w http.ResponseWriter, r *http.Request) {
	componentFilter := r.URL.Query().Get("component")
	componentTypeFilter := r.URL.Query().Get("component_type")
	severityFilter := r.URL.Query().Get("severity")

	// Support legacy scenario parameter
	if scenarioFilter := r.URL.Query().Get("scenario"); scenarioFilter != "" && componentFilter == "" {
		componentFilter = scenarioFilter
		componentTypeFilter = "scenario"
	}

	result, err := scanComponentsForVulnerabilities(componentFilter, componentTypeFilter, severityFilter)
	if err != nil {
		http.Error(w, fmt.Sprintf("Security scan failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// Vault provision handler
func vaultProvisionHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ResourceName string            `json:"resource_name"`
		Secrets      map[string]string `json:"secrets"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.ResourceName == "" {
		http.Error(w, "resource_name is required", http.StatusBadRequest)
		return
	}
	updates := normalizeProvisionSecrets(req.Secrets)
	if len(updates) == 0 {
		http.Error(w, "no secrets provided", http.StatusBadRequest)
		return
	}

	result, err := performSecretProvision(r.Context(), req.ResourceName, updates)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func normalizeProvisionSecrets(raw map[string]string) map[string]string {
	updates := map[string]string{}
	for key, value := range raw {
		envName := strings.TrimSpace(key)
		if envName == "" || strings.EqualFold(envName, "default") {
			continue
		}

		lower := strings.ToLower(envName)
		if strings.HasPrefix(lower, "resource:") {
			continue
		}
		trimmedValue := strings.TrimSpace(value)
		if trimmedValue == "" {
			continue
		}
		envName = strings.ToUpper(strings.ReplaceAll(envName, " ", "_"))
		updates[envName] = trimmedValue
	}
	return updates
}

func performSecretProvision(ctx context.Context, resource string, secrets map[string]string) (*ProvisionResponse, error) {
	if len(secrets) == 0 {
		return nil, fmt.Errorf("no secrets provided")
	}

	saved, err := saveSecretsToLocalStore(secrets)
	if err != nil {
		return nil, fmt.Errorf("failed to update local secrets store: %w", err)
	}

	response := &ProvisionResponse{
		Resource:      resource,
		StoredSecrets: saved,
	}

	if strings.TrimSpace(resource) == "" {
		response.Success = true
		response.Message = "Secrets stored locally. Provide a resource to sync with Vault."
		return response, nil
	}

	results, provisionErr := provisionSecretsToVault(ctx, resource, secrets)
	response.Details = results
	for _, result := range results {
		if strings.EqualFold(result.Status, "stored") {
			response.VaultStored++
		}
	}

	if provisionErr != nil && response.VaultStored == 0 {
		return response, fmt.Errorf("failed to store secrets in vault: %w", provisionErr)
	}

	response.Success = provisionErr == nil || response.VaultStored > 0
	if provisionErr != nil {
		response.Message = provisionErr.Error()
	}
	return response, nil
}

type secretMapping struct {
	Path     string
	VaultKey string
}

func provisionSecretsToVault(ctx context.Context, resourceName string, secrets map[string]string) ([]secretProvisionResult, error) {
	results := []secretProvisionResult{}
	if len(secrets) == 0 {
		return results, fmt.Errorf("no secrets provided")
	}
	mappings := buildSecretMappings(resourceName)
	errs := []string{}
	for rawKey, rawValue := range secrets {
		envKey := strings.ToUpper(strings.TrimSpace(rawKey))
		value := strings.TrimSpace(rawValue)
		if envKey == "" || value == "" {
			continue
		}
		mapping := mappings[envKey]
		if mapping.Path == "" {
			mapping.Path = fmt.Sprintf("secret/resources/%s/%s", resourceName, strings.ToLower(envKey))
		}
		if mapping.VaultKey == "" {
			mapping.VaultKey = "value"
		}
		result := secretProvisionResult{EnvKey: envKey, VaultPath: mapping.Path, VaultKey: mapping.VaultKey}
		if err := putSecretInVault(mapping.Path, mapping.VaultKey, value); err != nil {
			result.Status = "failed"
			result.Error = err.Error()
			errs = append(errs, fmt.Sprintf("%s: %v", envKey, err))
		} else {
			result.Status = "stored"
			recordSecretProvision(ctx, resourceName, envKey, mapping.Path)
		}
		results = append(results, result)
	}
	if len(errs) > 0 {
		return results, fmt.Errorf(strings.Join(errs, "; "))
	}
	return results, nil
}

func buildSecretMappings(resourceName string) map[string]secretMapping {
	mappings := map[string]secretMapping{}
	config, err := loadResourceSecrets(resourceName)
	if err != nil || config == nil {
		return mappings
	}
	replacer := strings.NewReplacer("{resource}", resourceName)
	for _, definitions := range config.Secrets {
		for _, def := range definitions {
			path := strings.TrimSpace(def.Path)
			if path == "" {
				continue
			}
			path = replacer.Replace(path)
			if len(def.Fields) > 0 {
				for _, field := range def.Fields {
					for keyName, env := range field {
						envVar := strings.ToUpper(strings.TrimSpace(env))
						if envVar == "" {
							continue
						}
						mappings[envVar] = secretMapping{Path: path, VaultKey: keyName}
					}
				}
			}
			defaultEnv := strings.ToUpper(strings.TrimSpace(def.DefaultEnv))
			if defaultEnv != "" {
				mappings[defaultEnv] = secretMapping{Path: path, VaultKey: "value"}
			}
			nameAlias := strings.ToUpper(strings.TrimSpace(def.Name))
			if nameAlias != "" {
				alias := fmt.Sprintf("%s_%s", strings.ToUpper(resourceName), strings.ReplaceAll(nameAlias, " ", "_"))
				mappings[alias] = secretMapping{Path: path, VaultKey: "value"}
			}
		}
	}
	return mappings
}

func putSecretInVault(path, vaultKey, value string) error {
	args := []string{"content", "put-secret", "--path", path, "--value", value}
	if vaultKey != "" && vaultKey != "value" {
		args = append(args, "--key", vaultKey)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "resource-vault", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return fmt.Errorf("%w: %s", err, msg)
		}
		return err
	}
	return nil
}

func recordSecretProvision(ctx context.Context, resourceName, envKey, vaultPath string) {
	if db == nil {
		return
	}
	secretID, err := getResourceSecretID(ctx, resourceName, envKey)
	if err != nil {
		return
	}
	_, err = db.ExecContext(ctx, `
		INSERT INTO secret_provisions (resource_secret_id, storage_method, storage_location, provisioned_at, provisioned_by, provision_status)
		VALUES ($1, 'vault', $2, CURRENT_TIMESTAMP, $3, 'active')
	`, secretID, vaultPath, os.Getenv("USER"))
	if err != nil {
		logger.Info("failed to record secret provision for %s: %v", envKey, err)
	}
}

// Compliance dashboard handler
func complianceHandler(w http.ResponseWriter, r *http.Request) {
	// Get vault secrets status
	vaultStatus, err := getVaultSecretsStatus("")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get vault status: %v", err), http.StatusInternalServerError)
		return
	}

	// Get security scan results for all components
	securityResults, err := scanComponentsForVulnerabilities("", "", "")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to scan for vulnerabilities: %v", err), http.StatusInternalServerError)
		return
	}

	// Calculate compliance metrics
	vaultHealth := 0
	if vaultStatus.TotalResources > 0 {
		vaultHealth = (vaultStatus.ConfiguredResources * 100) / vaultStatus.TotalResources
	}

	securityScore := 100 - securityResults.RiskScore
	if securityScore < 0 {
		securityScore = 0
	}

	overallCompliance := (vaultHealth + securityScore) / 2

	// Count vulnerabilities by severity
	criticalCount := 0
	highCount := 0
	mediumCount := 0
	lowCount := 0

	for _, vuln := range securityResults.Vulnerabilities {
		switch vuln.Severity {
		case "critical":
			criticalCount++
		case "high":
			highCount++
		case "medium":
			mediumCount++
		case "low":
			lowCount++
		}
	}

	// Calculate configured components from vault status and scenarios
	configuredComponents := vaultStatus.ConfiguredResources
	if securityResults.ComponentsSummary.TotalComponents > 0 {
		configuredComponents += securityResults.ComponentsSummary.ConfiguredCount
	}

	compliance := ComplianceMetrics{
		VaultSecretsHealth:   vaultHealth,
		SecurityScore:        securityScore,
		OverallCompliance:    overallCompliance,
		ConfiguredComponents: configuredComponents,
		CriticalIssues:       criticalCount,
		HighIssues:           highCount,
		MediumIssues:         mediumCount,
		LowIssues:            lowCount,
	}

	response := map[string]interface{}{
		"overall_score":        compliance.OverallCompliance,
		"vault_secrets_health": compliance.VaultSecretsHealth,
		"vulnerability_summary": map[string]int{
			"critical": compliance.CriticalIssues,
			"high":     compliance.HighIssues,
			"medium":   compliance.MediumIssues,
			"low":      compliance.LowIssues,
		},
		"remediation_progress":  compliance,
		"total_resources":       vaultStatus.TotalResources,
		"configured_resources":  vaultStatus.ConfiguredResources,
		"configured_components": compliance.ConfiguredComponents,
		"total_components":      securityResults.ComponentsSummary.TotalComponents,
		"components_summary":    securityResults.ComponentsSummary,
		"total_vulnerabilities": len(securityResults.Vulnerabilities),
		"last_updated":          time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func vulnerabilitiesHandler(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	severity := r.URL.Query().Get("severity")
	component := r.URL.Query().Get("component")
	componentType := r.URL.Query().Get("component_type")
	quickMode := r.URL.Query().Get("quick") == "true"

	// Support legacy scenario parameter
	if scenario := r.URL.Query().Get("scenario"); scenario != "" && component == "" {
		component = scenario
		componentType = "scenario"
	}

	// In quick mode or test mode, return minimal results
	if quickMode || os.Getenv("SECRETS_MANAGER_TEST_MODE") == "true" {
		quickResults := &SecurityScanResult{
			ScanID:          uuid.New().String(),
			Vulnerabilities: []SecurityVulnerability{},
			RiskScore:       0,
			ScanDurationMs:  1,
			Recommendations: generateRemediationSuggestions([]SecurityVulnerability{}),
		}
		response := map[string]interface{}{
			"vulnerabilities": quickResults.Vulnerabilities,
			"total_count":     0,
			"scan_metadata": map[string]interface{}{
				"scan_id":       quickResults.ScanID,
				"scan_duration": quickResults.ScanDurationMs,
				"risk_score":    quickResults.RiskScore,
				"component":     component,
				"severity":      severity,
				"timestamp":     time.Now().Format(time.RFC3339),
				"mode":          "quick",
			},
			"recommendations": quickResults.Recommendations,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get security scan results from real filesystem scan
	securityResults, err := scanComponentsForVulnerabilities(component, componentType, severity)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to scan for vulnerabilities: %v", err), http.StatusInternalServerError)
		return
	}

	// [REQ:SEC-SCAN-002] Vulnerability listing endpoint
	response := map[string]interface{}{
		"vulnerabilities": securityResults.Vulnerabilities,
		"total_count":     len(securityResults.Vulnerabilities),
		"scan_metadata": map[string]interface{}{
			"scan_id":       securityResults.ScanID,
			"scan_duration": securityResults.ScanDurationMs,
			"risk_score":    securityResults.RiskScore,
			"component":     component,
			"severity":      severity,
			"timestamp":     time.Now().Format(time.RFC3339),
		},
		"recommendations": securityResults.Recommendations,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func validateHandler(w http.ResponseWriter, r *http.Request) {
	var req ValidationRequest
	if r.Method == "POST" {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
	}
	if validator == nil {
		http.Error(w, "validator not ready (database unavailable)", http.StatusServiceUnavailable)
		return
	}

	response, err := validator.ValidateSecrets(req.Resource)
	if err != nil {
		http.Error(w, fmt.Sprintf("Validation failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func provisionHandler(w http.ResponseWriter, r *http.Request) {
	var req ProvisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	secrets := normalizeProvisionSecrets(req.Secrets)
	if req.SecretKey != "" && strings.TrimSpace(req.SecretValue) != "" {
		if secrets == nil {
			secrets = map[string]string{}
		}
		key := strings.ToUpper(strings.TrimSpace(req.SecretKey))
		if key != "" {
			secrets[key] = strings.TrimSpace(req.SecretValue)
		}
	}

	if len(secrets) == 0 {
		http.Error(w, "no secrets provided", http.StatusBadRequest)
		return
	}

	result, err := performSecretProvision(r.Context(), strings.TrimSpace(req.Resource), secrets)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// deploymentSecretsHandler generates deployment manifests for specific tiers
func deploymentSecretsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req DeploymentManifestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	manifest, err := generateDeploymentManifest(r.Context(), req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate manifest: %v", err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(manifest)
}

// deploymentSecretsGetHandler exposes a simple GET form for bundle consumers (scenario-to-*, deployment-manager).
// Example: GET /api/v1/deployment/secrets/picker-wheel?tier=tier-2-desktop&resources=postgres,redis&include_optional=false
func deploymentSecretsGetHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenario := strings.TrimSpace(vars["scenario"])
	if scenario == "" {
		http.Error(w, "scenario is required", http.StatusBadRequest)
		return
	}

	query := r.URL.Query()
	tier := strings.TrimSpace(query.Get("tier"))
	if tier == "" {
		tier = "tier-2-desktop"
	}
	includeOptional := false
	if raw := query.Get("include_optional"); raw != "" {
		val, err := strconv.ParseBool(raw)
		if err != nil {
			http.Error(w, "include_optional must be a boolean", http.StatusBadRequest)
			return
		}
		includeOptional = val
	}
	resources := []string{}
	if rawResources := query.Get("resources"); rawResources != "" {
		for _, r := range strings.Split(rawResources, ",") {
			r = strings.TrimSpace(r)
			if r != "" {
				resources = append(resources, r)
			}
		}
	}

	req := DeploymentManifestRequest{
		Scenario:        scenario,
		Tier:            tier,
		Resources:       resources,
		IncludeOptional: includeOptional,
	}

	manifest, err := generateDeploymentManifest(r.Context(), req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate manifest: %v", err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(manifest)
}

func resourceDetailHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	resource := vars["resource"]
	detail, err := fetchResourceDetail(r.Context(), resource)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to load resource detail: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(detail)
}

type resourceSecretUpdateRequest struct {
	Classification *string `json:"classification"`
	Description    *string `json:"description"`
	Required       *bool   `json:"required"`
	OwnerTeam      *string `json:"owner_team"`
	OwnerContact   *string `json:"owner_contact"`
}

func resourceSecretUpdateHandler(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		http.Error(w, "database not ready", http.StatusServiceUnavailable)
		return
	}
	vars := mux.Vars(r)
	resource := vars["resource"]
	secretKey := vars["secret"]
	var req resourceSecretUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	updates := []string{}
	args := []interface{}{}
	idx := 1
	if req.Classification != nil {
		value := strings.TrimSpace(*req.Classification)
		if value == "" {
			http.Error(w, "classification cannot be empty", http.StatusBadRequest)
			return
		}
		allowed := map[string]struct{}{"infrastructure": {}, "service": {}, "user": {}}
		if _, ok := allowed[value]; !ok {
			http.Error(w, "invalid classification", http.StatusBadRequest)
			return
		}
		updates = append(updates, fmt.Sprintf("classification = $%d", idx))
		args = append(args, value)
		idx++
	}
	if req.Description != nil {
		updates = append(updates, fmt.Sprintf("description = $%d", idx))
		args = append(args, strings.TrimSpace(*req.Description))
		idx++
	}
	if req.Required != nil {
		updates = append(updates, fmt.Sprintf("required = $%d", idx))
		args = append(args, *req.Required)
		idx++
	}
	if req.OwnerTeam != nil {
		updates = append(updates, fmt.Sprintf("owner_team = $%d", idx))
		args = append(args, strings.TrimSpace(*req.OwnerTeam))
		idx++
	}
	if req.OwnerContact != nil {
		updates = append(updates, fmt.Sprintf("owner_contact = $%d", idx))
		args = append(args, strings.TrimSpace(*req.OwnerContact))
		idx++
	}
	if len(updates) == 0 {
		http.Error(w, "no updates provided", http.StatusBadRequest)
		return
	}
	query := fmt.Sprintf("UPDATE resource_secrets SET %s, updated_at = CURRENT_TIMESTAMP WHERE resource_name = $%d AND secret_key = $%d RETURNING id", strings.Join(updates, ", "), idx, idx+1)
	args = append(args, resource, secretKey)
	var secretID string
	if err := db.QueryRowContext(r.Context(), query, args...).Scan(&secretID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "secret not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("failed to update secret: %v", err), http.StatusInternalServerError)
		return
	}
	_ = secretID
	secret, err := fetchSingleSecretDetail(r.Context(), resource, secretKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to fetch secret detail: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(secret)
}

type secretStrategyRequest struct {
	Tier              string                 `json:"tier"`
	HandlingStrategy  string                 `json:"handling_strategy"`
	FallbackStrategy  string                 `json:"fallback_strategy"`
	RequiresUserInput bool                   `json:"requires_user_input"`
	PromptLabel       string                 `json:"prompt_label"`
	PromptDescription string                 `json:"prompt_description"`
	GeneratorTemplate map[string]interface{} `json:"generator_template"`
	BundleHints       map[string]interface{} `json:"bundle_hints"`
}

func secretStrategyHandler(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		http.Error(w, "database not ready", http.StatusServiceUnavailable)
		return
	}
	vars := mux.Vars(r)
	resource := vars["resource"]
	secretKey := vars["secret"]
	var req secretStrategyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	tier := strings.TrimSpace(req.Tier)
	strategy := strings.TrimSpace(req.HandlingStrategy)
	if tier == "" || strategy == "" {
		http.Error(w, "tier and handling_strategy are required", http.StatusBadRequest)
		return
	}
	allowedStrategies := map[string]struct{}{"strip": {}, "generate": {}, "prompt": {}, "delegate": {}}
	if _, ok := allowedStrategies[strategy]; !ok {
		http.Error(w, "invalid handling strategy", http.StatusBadRequest)
		return
	}
	secretID, err := getResourceSecretID(r.Context(), resource, secretKey)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "secret not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("failed to load secret: %v", err), http.StatusInternalServerError)
		return
	}
	generatorJSON, _ := json.Marshal(req.GeneratorTemplate)
	bundleJSON, _ := json.Marshal(req.BundleHints)
	if string(generatorJSON) == "null" {
		generatorJSON = nil
	}
	if string(bundleJSON) == "null" {
		bundleJSON = nil
	}
	_, err = db.ExecContext(r.Context(), `
		INSERT INTO secret_deployment_strategies (
			resource_secret_id, tier, handling_strategy, fallback_strategy,
			requires_user_input, prompt_label, prompt_description,
			generator_template, bundle_hints
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT (resource_secret_id, tier)
		DO UPDATE SET
			handling_strategy = EXCLUDED.handling_strategy,
			fallback_strategy = EXCLUDED.fallback_strategy,
			requires_user_input = EXCLUDED.requires_user_input,
			prompt_label = EXCLUDED.prompt_label,
			prompt_description = EXCLUDED.prompt_description,
			generator_template = EXCLUDED.generator_template,
			bundle_hints = EXCLUDED.bundle_hints,
			updated_at = CURRENT_TIMESTAMP
	`, secretID, tier, strategy, nullString(req.FallbackStrategy), req.RequiresUserInput, req.PromptLabel, req.PromptDescription, nullBytes(generatorJSON), nullBytes(bundleJSON))
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to persist strategy: %v", err), http.StatusInternalServerError)
		return
	}
	secret, err := fetchSingleSecretDetail(r.Context(), resource, secretKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to fetch secret detail: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(secret)
}

type vulnerabilityStatusRequest struct {
	Status     string `json:"status"`
	AssignedTo string `json:"assigned_to"`
}

func vulnerabilityStatusHandler(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		http.Error(w, "database not ready", http.StatusServiceUnavailable)
		return
	}
	vars := mux.Vars(r)
	vulnID := vars["id"]
	var req vulnerabilityStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	status := strings.TrimSpace(req.Status)
	if status == "" {
		http.Error(w, "status is required", http.StatusBadRequest)
		return
	}
	if _, ok := allowedVulnerabilityStatuses[status]; !ok {
		http.Error(w, "invalid status", http.StatusBadRequest)
		return
	}
	assigned := strings.TrimSpace(req.AssignedTo)
	query := `
		UPDATE security_vulnerabilities
		SET status = $1,
		    assigned_to = CASE WHEN $2 = '' THEN assigned_to ELSE $2 END,
		    resolved_at = CASE WHEN $1 = 'resolved' THEN CURRENT_TIMESTAMP ELSE resolved_at END
		WHERE id = $3
		RETURNING id
	`
	var updatedID string
	if err := db.QueryRowContext(r.Context(), query, status, assigned, vulnID).Scan(&updatedID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "vulnerability not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("failed to update vulnerability: %v", err), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"id": updatedID, "status": status})
}

func storeDiscoveredSecret(secret ResourceSecret) {
	query := `
		INSERT INTO resource_secrets (id, resource_name, secret_key, secret_type, 
			required, description, validation_pattern, documentation_url, default_value,
			created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (resource_name, secret_key) 
		DO UPDATE SET 
			secret_type = EXCLUDED.secret_type,
			required = EXCLUDED.required,
			description = EXCLUDED.description,
			updated_at = CURRENT_TIMESTAMP
	`

	_, err := db.Exec(query, secret.ID, secret.ResourceName, secret.SecretKey,
		secret.SecretType, secret.Required, secret.Description,
		secret.ValidationPattern, secret.DocumentationURL, secret.DefaultValue,
		secret.CreatedAt, secret.UpdatedAt)
	if err != nil {
		logger.Info("Failed to store discovered secret: %v", err)
	}
}

func storeScanRecord(scan SecretScan) {
	// TODO: Implement scan record storage
	logger.Info("Scan completed: %d secrets discovered in %dms", scan.SecretsDiscovered, scan.ScanDurationMs)
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

type analyzerDeploymentReport struct {
	Scenario      string                       `json:"scenario"`
	ReportVersion int                          `json:"report_version"`
	GeneratedAt   time.Time                    `json:"generated_at"`
	Dependencies  []analyzerDependency         `json:"dependencies"`
	Aggregates    map[string]analyzerAggregate `json:"aggregates"`
}

type analyzerDependency struct {
	Name         string                         `json:"name"`
	Type         string                         `json:"type"`
	ResourceType string                         `json:"resource_type"`
	Requirements analyzerRequirement            `json:"requirements"`
	TierSupport  map[string]analyzerTierSupport `json:"tier_support"`
	Alternatives []string                       `json:"alternatives"`
	Source       string                         `json:"source"`
}

type analyzerRequirement struct {
	RAMMB      int    `json:"ram_mb"`
	DiskMB     int    `json:"disk_mb"`
	CPUCores   int    `json:"cpu_cores"`
	Network    string `json:"network"`
	Source     string `json:"source"`
	Confidence string `json:"confidence"`
}

type analyzerTierSupport struct {
	Supported    bool     `json:"supported"`
	FitnessScore float64  `json:"fitness_score"`
	Notes        string   `json:"notes"`
	Reason       string   `json:"reason"`
	Alternatives []string `json:"alternatives"`
}

type analyzerAggregate struct {
	FitnessScore          float64                   `json:"fitness_score"`
	DependencyCount       int                       `json:"dependency_count"`
	BlockingDependencies  []string                  `json:"blocking_dependencies"`
	EstimatedRequirements analyzerRequirementTotals `json:"estimated_requirements"`
}

type analyzerRequirementTotals struct {
	RAMMB    int `json:"ram_mb"`
	DiskMB   int `json:"disk_mb"`
	CPUCores int `json:"cpu_cores"`
}

func generateDeploymentManifest(ctx context.Context, req DeploymentManifestRequest) (*DeploymentManifest, error) {
	scenario := strings.TrimSpace(req.Scenario)
	tier := strings.TrimSpace(strings.ToLower(req.Tier))
	if scenario == "" || tier == "" {
		return nil, fmt.Errorf("scenario and tier are required")
	}

	resources := dedupeStrings(req.Resources)
	scenarioResources := resolveScenarioResources(scenario)
	effectiveResources := resources
	if len(scenarioResources) > 0 {
		if len(effectiveResources) == 0 {
			effectiveResources = scenarioResources
		} else {
			if intersected := intersectStrings(effectiveResources, scenarioResources); len(intersected) > 0 {
				effectiveResources = intersected
			} else {
				effectiveResources = scenarioResources
			}
		}
	}
	analyzerReport := ensureAnalyzerDeploymentReport(ctx, scenario)

	if db == nil {
		return buildFallbackManifest(scenario, tier, effectiveResources), nil
	}

	query := `
		SELECT 
			rs.id,
			rs.resource_name,
			rs.secret_key,
			rs.secret_type,
			rs.required,
			COALESCE(rs.description, '') as description,
			COALESCE(rs.classification, 'service') as classification,
			COALESCE(rs.owner_team, '') as owner_team,
			COALESCE(rs.owner_contact, '') as owner_contact,
			COALESCE(sds.handling_strategy, '') as handling_strategy,
			COALESCE(sds.fallback_strategy, '') as fallback_strategy,
			COALESCE(sds.requires_user_input, false) as requires_user_input,
			COALESCE(sds.prompt_label, '') as prompt_label,
			COALESCE(sds.prompt_description, '') as prompt_description,
			sds.generator_template,
			sds.bundle_hints,
			COALESCE(tiers.tier_map, '{}'::jsonb) as tier_map
		FROM resource_secrets rs
		LEFT JOIN secret_deployment_strategies sds
			ON sds.resource_secret_id = rs.id AND sds.tier = $1
		LEFT JOIN (
			SELECT resource_secret_id, jsonb_object_agg(tier, handling_strategy) AS tier_map
			FROM secret_deployment_strategies
			GROUP BY resource_secret_id
		) tiers ON tiers.resource_secret_id = rs.id
	`

	args := []interface{}{tier}
	filters := []string{}
	argPos := 2
	if len(effectiveResources) > 0 {
		filters = append(filters, fmt.Sprintf("rs.resource_name = ANY($%d)", argPos))
		args = append(args, pq.Array(effectiveResources))
		argPos++
	}
	if !req.IncludeOptional {
		filters = append(filters, "rs.required = TRUE")
	}
	if len(filters) > 0 {
		query = fmt.Sprintf("%s WHERE %s", query, strings.Join(filters, " AND "))
	}
	query = query + " ORDER BY rs.resource_name, rs.secret_key"

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := []DeploymentSecretEntry{}
	resourceSet := map[string]struct{}{}

	for rows.Next() {
		var (
			secretID         string
			resourceName     string
			secretKey        string
			secretType       string
			required         bool
			description      string
			classification   string
			ownerTeam        string
			ownerContact     string
			handlingStrategy string
			fallbackStrategy string
			requiresUser     bool
			promptLabel      string
			promptDesc       string
			generatorJSON    []byte
			bundleJSON       []byte
			tierMapJSON      []byte
		)

		if err := rows.Scan(&secretID, &resourceName, &secretKey, &secretType, &required, &description, &classification,
			&ownerTeam, &ownerContact, &handlingStrategy, &fallbackStrategy, &requiresUser,
			&promptLabel, &promptDesc, &generatorJSON, &bundleJSON, &tierMapJSON); err != nil {
			return nil, err
		}

		entry := DeploymentSecretEntry{
			ResourceName:      resourceName,
			SecretKey:         secretKey,
			SecretType:        secretType,
			Required:          required,
			Classification:    classification,
			Description:       description,
			OwnerTeam:         ownerTeam,
			OwnerContact:      ownerContact,
			HandlingStrategy:  handlingStrategy,
			FallbackStrategy:  fallbackStrategy,
			RequiresUserInput: requiresUser,
			GeneratorTemplate: decodeJSONMap(generatorJSON),
			BundleHints:       decodeJSONMap(bundleJSON),
			TierStrategies:    decodeStringMap(tierMapJSON),
		}

		if entry.HandlingStrategy == "" {
			entry.HandlingStrategy = "unspecified"
		}
		if entry.FallbackStrategy == "" {
			entry.FallbackStrategy = ""
		}
		if promptLabel != "" || promptDesc != "" {
			entry.Prompt = &PromptMetadata{Label: promptLabel, Description: promptDesc}
		}

		entries = append(entries, entry)
		resourceSet[resourceName] = struct{}{}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("no secrets discovered for manifest request")
	}

	resourcesList := make([]string, 0, len(resourceSet))
	for resource := range resourceSet {
		resourcesList = append(resourcesList, resource)
	}
	if analyzerReport != nil {
		resourcesList = mergeResourceLists(resourcesList, analyzerResourceNames(analyzerReport))
	}
	sort.Strings(resourcesList)

	classificationTotals := map[string]int{}
	classificationReady := map[string]int{}
	strategyBreakdown := map[string]int{}
	blockingSecrets := []string{}

	for _, entry := range entries {
		classificationTotals[entry.Classification]++
		if entry.HandlingStrategy != "unspecified" {
			classificationReady[entry.Classification]++
			strategyBreakdown[entry.HandlingStrategy]++
		} else {
			blockingSecrets = append(blockingSecrets, fmt.Sprintf("%s:%s", entry.ResourceName, entry.SecretKey))
		}
	}

	if len(blockingSecrets) > 10 {
		blockingSecrets = blockingSecrets[:10]
	}

	summary := DeploymentSummary{
		TotalSecrets:          len(entries),
		StrategizedSecrets:    len(entries) - len(blockingSecrets),
		RequiresAction:        len(blockingSecrets),
		BlockingSecrets:       blockingSecrets,
		ClassificationWeights: classificationTotals,
		StrategyBreakdown:     strategyBreakdown,
		ScopeReadiness:        map[string]string{},
	}

	for class, total := range classificationTotals {
		ready := classificationReady[class]
		summary.ScopeReadiness[class] = fmt.Sprintf("%d/%d", ready, total)
	}

	manifest := &DeploymentManifest{
		Scenario:    scenario,
		Tier:        tier,
		GeneratedAt: time.Now(),
		Resources:   resourcesList,
		Secrets:     entries,
		Summary:     summary,
	}

	if analyzerReport != nil {
		manifest.Dependencies = convertAnalyzerDependencies(analyzerReport)
		manifest.TierAggregates = convertAnalyzerAggregates(analyzerReport)
		reportTime := analyzerReport.GeneratedAt
		manifest.AnalyzerGeneratedAt = &reportTime
	}

	if payload, err := json.Marshal(manifest); err == nil {
		if _, insertErr := db.ExecContext(ctx, `INSERT INTO deployment_manifests (scenario_name, tier, manifest) VALUES ($1, $2, $3)`, scenario, tier, payload); insertErr != nil {
			logger.Info("failed to persist deployment manifest telemetry: %v", insertErr)
		}
	} else {
		logger.Info("failed to marshal deployment manifest for telemetry: %v", err)
	}

	return manifest, nil
}

func buildFallbackManifest(scenario, tier string, resources []string) *DeploymentManifest {
	defaultResources := resources
	if len(defaultResources) == 0 {
		defaultResources = []string{"core-platform"}
	}
	entries := make([]DeploymentSecretEntry, 0, len(defaultResources))
	for idx, resource := range defaultResources {
		entries = append(entries, DeploymentSecretEntry{
			ResourceName:      resource,
			SecretKey:         fmt.Sprintf("PLACEHOLDER_SECRET_%d", idx+1),
			SecretType:        "token",
			Required:          true,
			Classification:    "service",
			Description:       "Fallback manifest entry (no database connection)",
			HandlingStrategy:  "prompt",
			RequiresUserInput: true,
			Prompt: &PromptMetadata{
				Label:       "Provide secret",
				Description: "Enter the secure value manually",
			},
			TierStrategies: map[string]string{
				tier: "prompt",
			},
		})
	}
	summary := DeploymentSummary{
		TotalSecrets:          len(entries),
		StrategizedSecrets:    len(entries),
		RequiresAction:        0,
		BlockingSecrets:       []string{},
		ClassificationWeights: map[string]int{"service": len(entries)},
		StrategyBreakdown:     map[string]int{"prompt": len(entries)},
		ScopeReadiness:        map[string]string{"service": fmt.Sprintf("%d/%d", len(entries), len(entries))},
	}
	manifest := &DeploymentManifest{
		Scenario:    scenario,
		Tier:        tier,
		GeneratedAt: time.Now(),
		Resources:   defaultResources,
		Secrets:     entries,
		Summary:     summary,
	}
	return manifest
}

func loadAnalyzerDeploymentReport(scenario string) (*analyzerDeploymentReport, error) {
	if scenario == "" {
		return nil, fmt.Errorf("scenario required")
	}
	root := os.Getenv("VROOLI_ROOT")
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		root = filepath.Join(home, "Vrooli")
	}
	reportPath := filepath.Join(root, "scenarios", scenario, ".vrooli", "deployment", "deployment-report.json")
	data, err := os.ReadFile(reportPath)
	if err != nil {
		return nil, err
	}
	var report analyzerDeploymentReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, err
	}
	return &report, nil
}

func persistAnalyzerDeploymentReport(scenario string, report *analyzerDeploymentReport) error {
	if scenario == "" || report == nil {
		return fmt.Errorf("scenario and report required")
	}
	root := os.Getenv("VROOLI_ROOT")
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		root = filepath.Join(home, "Vrooli")
	}
	reportDir := filepath.Join(root, "scenarios", scenario, ".vrooli", "deployment")
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(reportDir, "deployment-report.json")
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, encoded, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func ensureAnalyzerDeploymentReport(ctx context.Context, scenario string) *analyzerDeploymentReport {
	if scenario == "" {
		return nil
	}
	report, err := loadAnalyzerDeploymentReport(scenario)
	if err == nil && report != nil {
		// Check if report is stale (> 24 hours old)
		age := time.Since(report.GeneratedAt)
		if age > 24*time.Hour {
			logger.Info("analyzer report for %s is %v old, attempting refresh", scenario, age.Round(time.Hour))
			// Try to refresh but don't fail if unavailable
			if freshReport, fetchErr := fetchAnalyzerReportViaService(ctx, scenario); fetchErr == nil && freshReport != nil {
				if persistErr := persistAnalyzerDeploymentReport(scenario, freshReport); persistErr == nil {
					return freshReport
				}
			}
			// Return stale report if refresh fails
			logger.Info("using stale analyzer report for %s (refresh failed)", scenario)
		}
		return report
	}
	remoteReport, fetchErr := fetchAnalyzerReportViaService(ctx, scenario)
	if fetchErr != nil {
		logger.Info("deployment analyzer fallback failed for %s: %v", scenario, fetchErr)
		return nil
	}
	if remoteReport == nil {
		return nil
	}
	if persistErr := persistAnalyzerDeploymentReport(scenario, remoteReport); persistErr != nil {
		logger.Info("failed to persist analyzer report for %s: %v", scenario, persistErr)
	}
	return remoteReport
}

func fetchAnalyzerReportViaService(ctx context.Context, scenario string) (*analyzerDeploymentReport, error) {
	port, err := discoverAnalyzerPort(ctx)
	if err != nil {
		return nil, err
	}
	requestCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	url := fmt.Sprintf("http://localhost:%s/api/v1/scenarios/%s/deployment", port, scenario)
	req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("analyzer responded with status %d: %s", resp.StatusCode, string(body))
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var report analyzerDeploymentReport
	if err := json.Unmarshal(body, &report); err != nil {
		return nil, err
	}
	return &report, nil
}

func discoverAnalyzerPort(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "vrooli", "scenario", "port", "scenario-dependency-analyzer", "API_PORT")
	cmd.Env = os.Environ()
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to discover analyzer port: %w", err)
	}
	port := strings.TrimSpace(string(output))
	if port == "" {
		return "", fmt.Errorf("analyzer API port not available")
	}
	return port, nil
}

func analyzerResourceNames(report *analyzerDeploymentReport) []string {
	if report == nil {
		return nil
	}
	names := []string{}
	for _, dep := range report.Dependencies {
		if dep.Type == "resource" && dep.Name != "" {
			names = append(names, dep.Name)
		}
	}
	return names
}

func mergeResourceLists(base []string, extras []string) []string {
	set := map[string]struct{}{}
	for _, item := range base {
		if item == "" {
			continue
		}
		set[item] = struct{}{}
	}
	for _, item := range extras {
		if item == "" {
			continue
		}
		set[item] = struct{}{}
	}
	merged := make([]string, 0, len(set))
	for key := range set {
		merged = append(merged, key)
	}
	return merged
}

func convertAnalyzerDependencies(report *analyzerDeploymentReport) []DependencyInsight {
	if report == nil {
		return nil
	}
	insights := make([]DependencyInsight, 0, len(report.Dependencies))
	for _, dep := range report.Dependencies {
		insight := DependencyInsight{
			Name:         dep.Name,
			Kind:         dep.Type,
			ResourceType: dep.ResourceType,
			Source:       dep.Source,
			Alternatives: dep.Alternatives,
		}
		if dep.Requirements.RAMMB != 0 || dep.Requirements.DiskMB != 0 || dep.Requirements.CPUCores != 0 || dep.Requirements.Network != "" {
			insight.Requirements = &DependencyRequirementSummary{
				RAMMB:      dep.Requirements.RAMMB,
				DiskMB:     dep.Requirements.DiskMB,
				CPUCores:   dep.Requirements.CPUCores,
				Network:    dep.Requirements.Network,
				Source:     dep.Requirements.Source,
				Confidence: dep.Requirements.Confidence,
			}
		}
		if len(dep.TierSupport) > 0 {
			insight.TierSupport = map[string]DependencyTierSupportView{}
			for tier, support := range dep.TierSupport {
				insight.TierSupport[tier] = DependencyTierSupportView{
					Supported:    support.Supported,
					FitnessScore: support.FitnessScore,
					Notes:        support.Notes,
					Reason:       support.Reason,
					Alternatives: support.Alternatives,
				}
			}
		}
		insights = append(insights, insight)
	}
	return insights
}

func convertAnalyzerAggregates(report *analyzerDeploymentReport) map[string]TierAggregateView {
	if report == nil || len(report.Aggregates) == 0 {
		return nil
	}
	result := make(map[string]TierAggregateView, len(report.Aggregates))
	for tier, aggregate := range report.Aggregates {
		view := TierAggregateView{
			FitnessScore:         aggregate.FitnessScore,
			DependencyCount:      aggregate.DependencyCount,
			BlockingDependencies: aggregate.BlockingDependencies,
		}
		if aggregate.EstimatedRequirements.RAMMB != 0 || aggregate.EstimatedRequirements.DiskMB != 0 || aggregate.EstimatedRequirements.CPUCores != 0 {
			view.EstimatedRequirements = &DependencyRequirementSummary{
				RAMMB:    aggregate.EstimatedRequirements.RAMMB,
				DiskMB:   aggregate.EstimatedRequirements.DiskMB,
				CPUCores: aggregate.EstimatedRequirements.CPUCores,
			}
		}
		result[tier] = view
	}
	return result
}

func resolveScenarioResources(scenario string) []string {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return nil
	}
	root := os.Getenv("VROOLI_ROOT")
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil
		}
		root = filepath.Join(home, "Vrooli")
	}
	servicePath := filepath.Join(root, "scenarios", scenario, ".vrooli", "service.json")
	data, err := os.ReadFile(servicePath)
	if err != nil {
		return nil
	}
	var doc struct {
		Service struct {
			Dependencies struct {
				Resources map[string]json.RawMessage `json:"resources"`
			} `json:"dependencies"`
		} `json:"service"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil
	}
	if len(doc.Service.Dependencies.Resources) == 0 {
		return nil
	}
	resources := make([]string, 0, len(doc.Service.Dependencies.Resources))
	for name := range doc.Service.Dependencies.Resources {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		resources = append(resources, name)
	}
	return dedupeStrings(resources)
}

func buildOrientationSummary(ctx context.Context) (*OrientationSummary, error) {
	var (
		vaultStatus    *VaultSecretsStatus
		securityResult *SecurityScanResult
		err            error
	)

	vaultStatus, err = getVaultSecretsStatus("")
	if err != nil {
		logger.Info("orientation vault status fallback: %v", err)
		vaultStatus = &VaultSecretsStatus{}
	}

	securityResult, err = scanComponentsForVulnerabilities("", "", "")
	if err != nil {
		logger.Info("orientation scan fallback: %v", err)
		securityResult = &SecurityScanResult{
			ScanID:          uuid.New().String(),
			Vulnerabilities: []SecurityVulnerability{},
			RiskScore:       0,
		}
	}

	tierReadiness, err := fetchTierReadiness(ctx)
	if err != nil {
		return nil, err
	}

	resourceInsights, err := fetchResourceInsights(ctx)
	if err != nil {
		return nil, err
	}

	orientation := &OrientationSummary{
		HeroStats:             buildHeroStats(vaultStatus, securityResult, tierReadiness),
		Journeys:              buildJourneyCards(vaultStatus, securityResult, tierReadiness),
		TierReadiness:         tierReadiness,
		ResourceInsights:      resourceInsights,
		VulnerabilityInsights: buildVulnerabilityHighlights(securityResult),
		UpdatedAt:             time.Now(),
	}

	return orientation, nil
}

func fetchTierReadiness(ctx context.Context) ([]TierReadiness, error) {
	var totalRequired int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM resource_secrets WHERE required = TRUE`).Scan(&totalRequired); err != nil {
		return nil, err
	}

	readiness := make([]TierReadiness, 0, len(deploymentTierCatalog))

	for _, tier := range deploymentTierCatalog {
		var strategized int
		if err := db.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM resource_secrets rs
			WHERE rs.required = TRUE AND EXISTS (
				SELECT 1 FROM secret_deployment_strategies sds
				WHERE sds.resource_secret_id = rs.id AND sds.tier = $1
			)
		`, tier.Name).Scan(&strategized); err != nil {
			return nil, err
		}

		missingRows, err := db.QueryContext(ctx, `
			SELECT rs.resource_name, rs.secret_key
			FROM resource_secrets rs
			WHERE rs.required = TRUE AND NOT EXISTS (
				SELECT 1 FROM secret_deployment_strategies sds
				WHERE sds.resource_secret_id = rs.id AND sds.tier = $1
			)
			ORDER BY rs.resource_name, rs.secret_key
			LIMIT 5
		`, tier.Name)
		if err != nil {
			return nil, err
		}
		var blockers []string
		for missingRows.Next() {
			var rName, sKey string
			if err := missingRows.Scan(&rName, &sKey); err != nil {
				missingRows.Close()
				return nil, err
			}
			blockers = append(blockers, fmt.Sprintf("%s:%s", rName, sKey))
		}
		missingRows.Close()

		readyPercent := 0
		if totalRequired == 0 {
			readyPercent = 100
		} else {
			readyPercent = int(math.Round((float64(strategized) / float64(totalRequired)) * 100))
		}

		readiness = append(readiness, TierReadiness{
			Tier:                 tier.Name,
			Label:                tier.Label,
			Strategized:          strategized,
			Total:                totalRequired,
			ReadyPercent:         readyPercent,
			BlockingSecretSample: blockers,
		})
	}

	return readiness, nil
}

func fetchResourceInsights(ctx context.Context) ([]ResourceInsight, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT resource_name, total_secrets, required_secrets, valid_secrets, missing_required_secrets, invalid_secrets, last_validation
		FROM secret_health_summary
		ORDER BY resource_name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	summaries := []SecretHealthSummary{}
	for rows.Next() {
		var summary SecretHealthSummary
		if err := rows.Scan(&summary.ResourceName, &summary.TotalSecrets, &summary.RequiredSecrets,
			&summary.ValidSecrets, &summary.MissingRequiredSecrets, &summary.InvalidSecrets, &summary.LastValidation); err != nil {
			return nil, err
		}
		summaries = append(summaries, summary)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	secretRows, err := db.QueryContext(ctx, `
		SELECT rs.resource_name,
		       rs.secret_key,
		       rs.secret_type,
		       COALESCE(rs.classification, 'service') as classification,
		       rs.required,
		       COALESCE(tiers.tier_map, '{}'::jsonb) as tier_map
		FROM resource_secrets rs
		LEFT JOIN (
			SELECT resource_secret_id, jsonb_object_agg(tier, handling_strategy) AS tier_map
			FROM secret_deployment_strategies
			GROUP BY resource_secret_id
		) tiers ON tiers.resource_secret_id = rs.id
		ORDER BY rs.resource_name, rs.secret_key
	`)
	if err != nil {
		return nil, err
	}
	defer secretRows.Close()

	resourceSecretMap := map[string][]ResourceSecretInsight{}
	for secretRows.Next() {
		var (
			resourceName   string
			secretKey      string
			secretType     string
			classification string
			required       bool
			tierMapJSON    []byte
		)
		if err := secretRows.Scan(&resourceName, &secretKey, &secretType, &classification, &required, &tierMapJSON); err != nil {
			return nil, err
		}
		insight := ResourceSecretInsight{
			SecretKey:      secretKey,
			SecretType:     secretType,
			Classification: classification,
			Required:       required,
			TierStrategies: decodeStringMap(tierMapJSON),
		}
		resourceSecretMap[resourceName] = append(resourceSecretMap[resourceName], insight)
	}
	if err := secretRows.Err(); err != nil {
		return nil, err
	}

	sort.Slice(summaries, func(i, j int) bool {
		if summaries[i].MissingRequiredSecrets == summaries[j].MissingRequiredSecrets {
			return summaries[i].ResourceName < summaries[j].ResourceName
		}
		return summaries[i].MissingRequiredSecrets > summaries[j].MissingRequiredSecrets
	})

	limit := 5
	if len(summaries) < limit {
		limit = len(summaries)
	}
	insights := make([]ResourceInsight, 0, limit)
	for idx := 0; idx < limit; idx++ {
		summary := summaries[idx]
		secrets := resourceSecretMap[summary.ResourceName]
		if len(secrets) > 6 {
			secrets = secrets[:6]
		}
		insights = append(insights, ResourceInsight{
			ResourceName:   summary.ResourceName,
			TotalSecrets:   summary.TotalSecrets,
			ValidSecrets:   summary.ValidSecrets,
			MissingSecrets: summary.MissingRequiredSecrets,
			InvalidSecrets: summary.InvalidSecrets,
			LastValidation: summary.LastValidation,
			Secrets:        secrets,
		})
	}

	return insights, nil
}

func buildHeroStats(vaultStatus *VaultSecretsStatus, security *SecurityScanResult, readiness []TierReadiness) HeroStats {
	hero := HeroStats{}
	if vaultStatus != nil {
		hero.VaultConfigured = vaultStatus.ConfiguredResources
		hero.VaultTotal = vaultStatus.TotalResources
		hero.MissingSecrets = len(vaultStatus.MissingSecrets)
		hero.LastScan = vaultStatus.LastUpdated.Format(time.RFC3339)
	}
	if security != nil {
		hero.RiskScore = security.RiskScore
		if hero.LastScan == "" {
			hero.LastScan = time.Now().Format(time.RFC3339)
		}
	}
	vaultHealth := 0
	if hero.VaultTotal > 0 {
		vaultHealth = (hero.VaultConfigured * 100) / hero.VaultTotal
	}
	securityScore := 100 - hero.RiskScore
	if securityScore < 0 {
		securityScore = 0
	}
	hero.OverallScore = (vaultHealth + securityScore) / 2
	bestReadiness := 0
	for _, tier := range readiness {
		if tier.ReadyPercent > bestReadiness {
			bestReadiness = tier.ReadyPercent
		}
	}
	hero.Confidence = math.Round((float64(bestReadiness)/100.0)*100) / 100
	switch {
	case hero.OverallScore >= 80:
		hero.ReadinessLabel = "Operational"
	case hero.OverallScore >= 50:
		hero.ReadinessLabel = "Stabilize"
	default:
		hero.ReadinessLabel = "Blocked"
	}
	return hero
}

func buildJourneyCards(vaultStatus *VaultSecretsStatus, security *SecurityScanResult, readiness []TierReadiness) []JourneyCard {
	missing := 0
	if vaultStatus != nil {
		missing = len(vaultStatus.MissingSecrets)
	}
	risk := 0
	if security != nil {
		risk = security.RiskScore
	}
	deploymentCoverage := 0
	for _, tier := range readiness {
		if tier.Tier == "tier-2-desktop" {
			deploymentCoverage = tier.ReadyPercent
			break
		}
	}
	journeys := []JourneyCard{
		{
			ID:          "orientation",
			Title:       "Orientation",
			Description: "Get familiar with your security posture and available workflows.",
			Status:      "steady",
			CtaLabel:    "Start Tour",
			CtaAction:   "open-orientation-flow",
			Primers:     []string{"Getting started"},
			Badge:       "Start",
		},
		{
			ID:          "configure-secrets",
			Title:       "Configure Secrets",
			Description: "Audit vault coverage and close the gap per resource with guided provisioning.",
			Status:      map[bool]string{true: "attention", false: "steady"}[missing > 0],
			CtaLabel:    "Start Coverage Flow",
			CtaAction:   "open-configure-flow",
			Primers:     []string{fmt.Sprintf("%d resources missing secrets", missing)},
			Badge:       "P0",
		},
		{
			ID:          "fix-vulnerabilities",
			Title:       "Address Vulnerabilities",
			Description: "Triage findings, review recommended fixes, and trigger agents for remediation.",
			Status:      map[bool]string{true: "attention", false: "steady"}[risk > 40],
			CtaLabel:    "Launch Security Flow",
			CtaAction:   "open-security-flow",
			Primers:     []string{fmt.Sprintf("Risk score %d", risk)},
			Badge:       "Security",
		},
		{
			ID:          "prep-deployment",
			Title:       "Prep Deployment",
			Description: "Select target tiers, review tier strategies, and emit bundle manifests.",
			Status:      map[bool]string{true: "attention", false: "steady"}[deploymentCoverage < 60],
			CtaLabel:    "Open Deployment Wizard",
			CtaAction:   "open-deployment-flow",
			Primers:     []string{fmt.Sprintf("Tier 2 coverage %d%%", deploymentCoverage)},
			Badge:       "Deploy",
		},
	}
	return journeys
}

func buildVulnerabilityHighlights(results *SecurityScanResult) []VulnerabilityHighlight {
	if results == nil {
		return []VulnerabilityHighlight{}
	}
	counts := map[string]int{}
	for _, vuln := range results.Vulnerabilities {
		counts[vuln.Severity]++
	}
	levels := []string{"critical", "high", "medium", "low"}
	highlights := []VulnerabilityHighlight{}
	for _, level := range levels {
		if counts[level] == 0 {
			continue
		}
		message := fmt.Sprintf("%d %s issues", counts[level], level)
		highlights = append(highlights, VulnerabilityHighlight{Severity: level, Count: counts[level], Message: message})
	}
	return highlights
}

func dedupeStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	set := map[string]struct{}{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		set[trimmed] = struct{}{}
	}
	if len(set) == 0 {
		return nil
	}
	result := make([]string, 0, len(set))
	for value := range set {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func intersectStrings(a, b []string) []string {
	if len(a) == 0 || len(b) == 0 {
		return nil
	}
	set := map[string]struct{}{}
	for _, value := range b {
		set[value] = struct{}{}
	}
	result := []string{}
	seen := map[string]struct{}{}
	for _, candidate := range a {
		if _, ok := set[candidate]; ok {
			if _, dup := seen[candidate]; dup {
				continue
			}
			seen[candidate] = struct{}{}
			result = append(result, candidate)
		}
	}
	if len(result) == 0 {
		return nil
	}
	sort.Strings(result)
	return result
}

func decodeJSONMap(payload []byte) map[string]interface{} {
	if len(payload) == 0 || string(payload) == "null" {
		return nil
	}
	var result map[string]interface{}
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil
	}
	return result
}

func decodeStringMap(payload []byte) map[string]string {
	if len(payload) == 0 || string(payload) == "null" {
		return map[string]string{}
	}
	var result map[string]string
	if err := json.Unmarshal(payload, &result); err != nil {
		return map[string]string{}
	}
	return result
}

func persistSecurityScan(ctx context.Context, scanID, componentFilter, componentTypeFilter, severityFilter string, metrics ScanMetrics, riskScore int, vulnerabilities []SecurityVulnerability) (*SecurityScanRun, error) {
	if db == nil {
		return nil, nil
	}
	metadata, err := json.Marshal(metrics)
	if err != nil {
		metadata = []byte("{}")
	}
	runID := uuid.New().String()
	now := time.Now()
	if _, err := db.ExecContext(ctx, `
		INSERT INTO security_scan_runs (
			id, scan_id, component_filter, component_type, severity_filter,
			files_scanned, files_skipped, vulnerabilities_found, risk_score, duration_ms,
			status, metadata, started_at, completed_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,'completed',$11,$12,$12)
	`, runID, scanID, nullString(componentFilter), nullString(componentTypeFilter), nullString(severityFilter), metrics.FilesScanned, metrics.FilesSkipped, len(vulnerabilities), riskScore, metrics.TotalScanTimeMs, metadata, now); err != nil {
		return nil, err
	}
	run := &SecurityScanRun{
		ID:              runID,
		ScanID:          scanID,
		ComponentFilter: componentFilter,
		ComponentType:   componentTypeFilter,
		SeverityFilter:  severityFilter,
		FilesScanned:    metrics.FilesScanned,
		FilesSkipped:    metrics.FilesSkipped,
		Vulnerabilities: len(vulnerabilities),
		RiskScore:       riskScore,
		DurationMs:      metrics.TotalScanTimeMs,
		Status:          "completed",
		StartedAt:       now,
		CompletedAt:     now,
	}
	for idx := range vulnerabilities {
		fingerprint := computeVulnerabilityFingerprint(vulnerabilities[idx])
		vulnerabilities[idx].Fingerprint = fingerprint
		vulnerabilities[idx].LastObservedAt = now
		vulnerabilities[idx].Status = defaultVulnerabilityStatus
		codeSnippet := vulnerabilities[idx].Code
		_, err := db.ExecContext(ctx, `
			INSERT INTO security_vulnerabilities (
				scan_run_id, fingerprint, component_type, component_name, file_path, line_number,
				severity, vulnerability_type, title, description, recommendation, code_snippet,
				can_auto_fix, status, first_observed_at, last_observed_at
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,'open',$14,$14)
			ON CONFLICT (fingerprint)
			DO UPDATE SET
				scan_run_id = EXCLUDED.scan_run_id,
				severity = EXCLUDED.severity,
				title = EXCLUDED.title,
				description = EXCLUDED.description,
				recommendation = EXCLUDED.recommendation,
				code_snippet = EXCLUDED.code_snippet,
				can_auto_fix = EXCLUDED.can_auto_fix,
				last_observed_at = EXCLUDED.last_observed_at,
				status = CASE
					WHEN security_vulnerabilities.status IN ('resolved','accepted') THEN 'regressed'
					ELSE security_vulnerabilities.status
				END
		`, runID, fingerprint, vulnerabilities[idx].ComponentType, vulnerabilities[idx].ComponentName, vulnerabilities[idx].FilePath, vulnerabilities[idx].LineNumber, vulnerabilities[idx].Severity, vulnerabilities[idx].Type, vulnerabilities[idx].Title, vulnerabilities[idx].Description, vulnerabilities[idx].Recommendation, codeSnippet, vulnerabilities[idx].CanAutoFix, now)
		if err != nil {
			logger.Info("failed to persist vulnerability %s: %v", fingerprint, err)
		}
	}
	return run, nil
}

func computeVulnerabilityFingerprint(v SecurityVulnerability) string {
	parts := []string{
		strings.ToLower(v.ComponentType),
		strings.ToLower(v.ComponentName),
		strings.ToLower(v.FilePath),
		fmt.Sprintf("%d", v.LineNumber),
		strings.ToLower(v.Type),
	}
	return strings.Join(parts, "|")
}

func buildScanCacheKey(componentFilter, componentTypeFilter, severityFilter string) string {
	return strings.Join([]string{componentFilter, componentTypeFilter, severityFilter}, "|")
}

func getCachedSecurityScan(key string) *SecurityScanResult {
	if key == "" {
		return nil
	}
	securityScanCacheMu.Lock()
	defer securityScanCacheMu.Unlock()
	entry, ok := securityScanCache[key]
	if !ok || time.Now().After(entry.expires) || entry.result == nil {
		return nil
	}
	return cloneSecurityScanResult(entry.result)
}

func storeCachedSecurityScan(key string, result *SecurityScanResult) {
	if key == "" || result == nil {
		return
	}
	securityScanCacheMu.Lock()
	securityScanCache[key] = cachedSecurityScan{
		key:     key,
		result:  cloneSecurityScanResult(result),
		expires: time.Now().Add(scanCacheTTL),
	}
	securityScanCacheMu.Unlock()
}

func cloneSecurityScanResult(result *SecurityScanResult) *SecurityScanResult {
	if result == nil {
		return nil
	}
	data, err := json.Marshal(result)
	if err != nil {
		return result
	}
	var clone SecurityScanResult
	if err := json.Unmarshal(data, &clone); err != nil {
		return result
	}
	return &clone
}

func warmSecurityScanCache() {
	if db == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cacheKey := buildScanCacheKey("", "", "")
	result, err := loadPersistedSecurityScan(ctx, "", "", "")
	if err != nil {
		logger.Info("No persisted vulnerability scan to warm cache: %v", err)
		return
	}
	if result == nil {
		logger.Info("No historical vulnerability scans found; cache will warm on demand")
		return
	}
	storeCachedSecurityScan(cacheKey, result)
	logger.Info("♻️  Loaded cached vulnerability scan from %s", result.GeneratedAt.Format(time.RFC3339))
	go scheduleSecurityScanRefresh("", "", "", cacheKey)
}

func scheduleSecurityScanRefresh(componentFilter, componentTypeFilter, severityFilter, cacheKey string) {
	if cacheKey == "" {
		cacheKey = buildScanCacheKey(componentFilter, componentTypeFilter, severityFilter)
	}
	scanRefreshInFlightMu.Lock()
	if scanRefreshInFlight[cacheKey] {
		scanRefreshInFlightMu.Unlock()
		return
	}
	scanRefreshInFlight[cacheKey] = true
	scanRefreshInFlightMu.Unlock()

	go func() {
		defer func() {
			scanRefreshInFlightMu.Lock()
			delete(scanRefreshInFlight, cacheKey)
			scanRefreshInFlightMu.Unlock()
		}()
		result, err := performSecurityScan(componentFilter, componentTypeFilter, severityFilter)
		if err != nil {
			logger.Warning("background vulnerability scan failed: %v", err)
			return
		}
		storeCachedSecurityScan(cacheKey, result)
	}()
}

func loadPersistedSecurityScan(ctx context.Context, componentFilter, componentTypeFilter, severityFilter string) (*SecurityScanResult, error) {
	if db == nil {
		return nil, nil
	}
	query := `
		SELECT id, scan_id, component_filter, component_type, severity_filter,
		       files_scanned, files_skipped, vulnerabilities_found, risk_score, duration_ms,
		       metadata, completed_at
		FROM security_scan_runs
		WHERE component_filter IS NOT DISTINCT FROM $1
		  AND component_type IS NOT DISTINCT FROM $2
		  AND severity_filter IS NOT DISTINCT FROM $3
		ORDER BY completed_at DESC
		LIMIT 1`
	var (
		runID             string
		scanID            string
		dbComponentFilter sql.NullString
		dbComponentType   sql.NullString
		dbSeverityFilter  sql.NullString
		filesScanned      sql.NullInt64
		filesSkipped      sql.NullInt64
		vulnCount         sql.NullInt64
		riskScore         sql.NullInt64
		durationMs        sql.NullInt64
		metadataBytes     []byte
		completedAt       time.Time
	)
	row := db.QueryRowContext(ctx, query, nullString(componentFilter), nullString(componentTypeFilter), nullString(severityFilter))
	if err := row.Scan(&runID, &scanID, &dbComponentFilter, &dbComponentType, &dbSeverityFilter, &filesScanned, &filesSkipped, &vulnCount, &riskScore, &durationMs, &metadataBytes, &completedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	metrics := ScanMetrics{}
	if len(metadataBytes) > 0 {
		_ = json.Unmarshal(metadataBytes, &metrics)
	}
	if metrics.TotalScanTimeMs == 0 && durationMs.Valid {
		metrics.TotalScanTimeMs = int(durationMs.Int64)
	}
	metrics.ScanComplete = true

	rows, err := db.QueryContext(ctx, `
		SELECT id, component_type, component_name, file_path, line_number, severity,
		       vulnerability_type, title, description, recommendation, code_snippet,
		       can_auto_fix, status, fingerprint, first_observed_at, last_observed_at
		FROM security_vulnerabilities
		WHERE scan_run_id = $1
		ORDER BY severity DESC, component_name, file_path
	`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var (
		vulnerabilities []SecurityVulnerability
		resourceSet     = map[string]struct{}{}
		scenarioSet     = map[string]struct{}{}
	)
	for rows.Next() {
		var (
			id             string
			compType       string
			compName       string
			filePath       string
			lineNumber     sql.NullInt64
			severityLevel  string
			vulnType       string
			title          string
			description    sql.NullString
			recommendation sql.NullString
			codeSnippet    sql.NullString
			canAutoFix     bool
			status         string
			fingerprint    string
			firstObserved  time.Time
			lastObserved   time.Time
		)
		if err := rows.Scan(&id, &compType, &compName, &filePath, &lineNumber, &severityLevel, &vulnType, &title, &description, &recommendation, &codeSnippet, &canAutoFix, &status, &fingerprint, &firstObserved, &lastObserved); err != nil {
			return nil, err
		}
		if compType == "resource" {
			resourceSet[compName] = struct{}{}
		} else if compType == "scenario" {
			scenarioSet[compName] = struct{}{}
		}
		vulnerabilities = append(vulnerabilities, SecurityVulnerability{
			ID:             id,
			ComponentType:  compType,
			ComponentName:  compName,
			FilePath:       filePath,
			LineNumber:     int(lineNumber.Int64),
			Severity:       severityLevel,
			Type:           vulnType,
			Title:          title,
			Description:    description.String,
			Recommendation: recommendation.String,
			Code:           codeSnippet.String,
			CanAutoFix:     canAutoFix,
			Status:         status,
			Fingerprint:    fingerprint,
			DiscoveredAt:   firstObserved,
			LastObservedAt: lastObserved,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	result := &SecurityScanResult{
		ScanID:            scanID,
		ComponentFilter:   componentFilter,
		ComponentType:     componentTypeFilter,
		Vulnerabilities:   vulnerabilities,
		RiskScore:         int(riskScore.Int64),
		ScanDurationMs:    metrics.TotalScanTimeMs,
		Recommendations:   generateRemediationSuggestions(vulnerabilities),
		ComponentsSummary: ComponentScanSummary{ResourcesScanned: len(resourceSet), ScenariosScanned: len(scenarioSet), TotalComponents: len(resourceSet) + len(scenarioSet)},
		ScanMetrics:       metrics,
		GeneratedAt:       completedAt,
	}
	return result, nil
}

func fetchResourceDetail(ctx context.Context, resourceName string) (*ResourceDetail, error) {
	resourceName = strings.TrimSpace(resourceName)
	if resourceName == "" {
		return nil, fmt.Errorf("resource name is required")
	}
	if db == nil {
		return &ResourceDetail{ResourceName: resourceName, Secrets: []ResourceSecretDetail{}}, nil
	}
	detail := &ResourceDetail{ResourceName: resourceName, Secrets: []ResourceSecretDetail{}}
	var (
		totalSecrets   sql.NullInt64
		missingSecrets sql.NullInt64
		validSecrets   sql.NullInt64
		lastValidation sql.NullTime
	)
	if err := db.QueryRowContext(ctx, `
		SELECT total_secrets, missing_required_secrets, valid_secrets, last_validation
		FROM secret_health_summary WHERE resource_name = $1
	`, resourceName).Scan(&totalSecrets, &missingSecrets, &validSecrets, &lastValidation); err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if totalSecrets.Valid {
		detail.TotalSecrets = int(totalSecrets.Int64)
	}
	if missingSecrets.Valid {
		detail.MissingSecrets = int(missingSecrets.Int64)
	}
	if validSecrets.Valid {
		detail.ValidSecrets = int(validSecrets.Int64)
	}
	if lastValidation.Valid {
		detail.LastValidation = &lastValidation.Time
	}
	secretRows, err := db.QueryContext(ctx, `
		SELECT rs.id, rs.secret_key, rs.secret_type, COALESCE(rs.description,''),
		       COALESCE(rs.classification,'service'), rs.required,
		       COALESCE(rs.owner_team,''), COALESCE(rs.owner_contact,''),
		       COALESCE(tiers.tier_map, '{}'::jsonb),
		       COALESCE(v.validation_status,'unknown') as validation_status,
		       v.validation_timestamp
		FROM resource_secrets rs
		LEFT JOIN (
			SELECT DISTINCT ON (resource_secret_id)
				resource_secret_id, validation_status, validation_timestamp
			FROM secret_validations
			ORDER BY resource_secret_id, validation_timestamp DESC
		) v ON v.resource_secret_id = rs.id
		LEFT JOIN (
			SELECT resource_secret_id, jsonb_object_agg(tier, handling_strategy) AS tier_map
			FROM secret_deployment_strategies
			GROUP BY resource_secret_id
		) tiers ON tiers.resource_secret_id = rs.id
		WHERE rs.resource_name = $1
		ORDER BY rs.secret_key
	`, resourceName)
	if err != nil {
		return nil, err
	}
	defer secretRows.Close()
	for secretRows.Next() {
		var (
			id             string
			secretKey      string
			secretType     string
			description    string
			classification string
			required       bool
			ownerTeam      string
			ownerContact   string
			tierJSON       []byte
			validation     string
			validationTime sql.NullTime
		)
		if err := secretRows.Scan(&id, &secretKey, &secretType, &description, &classification, &required, &ownerTeam, &ownerContact, &tierJSON, &validation, &validationTime); err != nil {
			return nil, err
		}
		secret := ResourceSecretDetail{
			ID:              id,
			SecretKey:       secretKey,
			SecretType:      secretType,
			Description:     description,
			Classification:  classification,
			Required:        required,
			OwnerTeam:       ownerTeam,
			OwnerContact:    ownerContact,
			TierStrategies:  decodeStringMap(tierJSON),
			ValidationState: validation,
		}
		if validationTime.Valid {
			secret.LastValidated = &validationTime.Time
		}
		detail.Secrets = append(detail.Secrets, secret)
	}
	if err := secretRows.Err(); err != nil {
		return nil, err
	}
	vulnRows, err := db.QueryContext(ctx, `
		SELECT id, fingerprint, file_path, line_number, severity, vulnerability_type,
		       title, description, recommendation, code_snippet, can_auto_fix,
		       status, first_observed_at, last_observed_at
		FROM security_vulnerabilities
		WHERE component_type = 'resource' AND component_name = $1 AND status <> 'resolved'
		ORDER BY severity DESC, last_observed_at DESC
	`, resourceName)
	if err == nil {
		defer vulnRows.Close()
		for vulnRows.Next() {
			var (
				id             string
				fingerprint    string
				filePath       string
				lineNumber     sql.NullInt64
				severity       string
				typeName       string
				title          string
				description    string
				recommendation string
				codeSnippet    string
				canAutoFix     bool
				status         string
				firstSeen      time.Time
				lastSeen       time.Time
			)
			if err := vulnRows.Scan(&id, &fingerprint, &filePath, &lineNumber, &severity, &typeName, &title, &description, &recommendation, &codeSnippet, &canAutoFix, &status, &firstSeen, &lastSeen); err != nil {
				return nil, err
			}
			vuln := SecurityVulnerability{
				ID:             id,
				ComponentType:  "resource",
				ComponentName:  resourceName,
				FilePath:       filePath,
				Severity:       severity,
				Type:           typeName,
				Title:          title,
				Description:    description,
				Code:           codeSnippet,
				Recommendation: recommendation,
				CanAutoFix:     canAutoFix,
				DiscoveredAt:   firstSeen,
				Status:         status,
				Fingerprint:    fingerprint,
				LastObservedAt: lastSeen,
			}
			if lineNumber.Valid {
				vuln.LineNumber = int(lineNumber.Int64)
			}
			detail.OpenVulnerabilities = append(detail.OpenVulnerabilities, vuln)
		}
		if err := vulnRows.Err(); err != nil {
			return nil, err
		}
	}
	return detail, nil
}

func nullString(value string) interface{} {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func nullBytes(value []byte) interface{} {
	if len(value) == 0 {
		return nil
	}
	return value
}

func fetchSingleSecretDetail(ctx context.Context, resourceName, secretKey string) (*ResourceSecretDetail, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	row := db.QueryRowContext(ctx, `
		SELECT rs.id, rs.secret_key, rs.secret_type, COALESCE(rs.description,''),
		       COALESCE(rs.classification,'service'), rs.required,
		       COALESCE(rs.owner_team,''), COALESCE(rs.owner_contact,''),
		       COALESCE(tiers.tier_map, '{}'::jsonb),
		       COALESCE(v.validation_status,'unknown'), v.validation_timestamp
		FROM resource_secrets rs
		LEFT JOIN (
			SELECT DISTINCT ON (resource_secret_id)
				resource_secret_id, validation_status, validation_timestamp
			FROM secret_validations
			ORDER BY resource_secret_id, validation_timestamp DESC
		) v ON v.resource_secret_id = rs.id
		LEFT JOIN (
			SELECT resource_secret_id, jsonb_object_agg(tier, handling_strategy) AS tier_map
			FROM secret_deployment_strategies
			GROUP BY resource_secret_id
		) tiers ON tiers.resource_secret_id = rs.id
		WHERE rs.resource_name = $1 AND rs.secret_key = $2
	`, resourceName, secretKey)
	var (
		id             string
		secretType     string
		description    string
		classification string
		required       bool
		ownerTeam      string
		ownerContact   string
		tierJSON       []byte
		validation     string
		validationTime sql.NullTime
	)
	if err := row.Scan(&id, &secretKey, &secretType, &description, &classification, &required, &ownerTeam, &ownerContact, &tierJSON, &validation, &validationTime); err != nil {
		return nil, err
	}
	secret := &ResourceSecretDetail{
		ID:              id,
		SecretKey:       secretKey,
		SecretType:      secretType,
		Description:     description,
		Classification:  classification,
		Required:        required,
		OwnerTeam:       ownerTeam,
		OwnerContact:    ownerContact,
		TierStrategies:  decodeStringMap(tierJSON),
		ValidationState: validation,
	}
	if validationTime.Valid {
		secret.LastValidated = &validationTime.Time
	}
	return secret, nil
}

func getResourceSecretID(ctx context.Context, resourceName, secretKey string) (string, error) {
	if db == nil {
		return "", fmt.Errorf("database not initialized")
	}
	var id string
	if err := db.QueryRowContext(ctx, `SELECT id FROM resource_secrets WHERE resource_name = $1 AND secret_key = $2`, resourceName, secretKey).Scan(&id); err != nil {
		return "", err
	}
	return id, nil
}

// Fix vulnerabilities handler - spawns claude-code agent
func fixVulnerabilitiesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Vulnerabilities []SecurityVulnerability `json:"vulnerabilities"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if len(req.Vulnerabilities) == 0 {
		http.Error(w, "No vulnerabilities provided", http.StatusBadRequest)
		return
	}

	fixRequestID := uuid.New().String()
	logger.Info("Starting vulnerability fix request %s for %d vulnerabilities", fixRequestID, len(req.Vulnerabilities))

	// Spawn claude-code agent to fix vulnerabilities
	go func() {
		err := spawnVulnerabilityFixerAgent(fixRequestID, req.Vulnerabilities)
		if err != nil {
			logger.Info("Failed to spawn vulnerability fixer agent: %v", err)
		}
	}()

	response := map[string]interface{}{
		"status":          "accepted",
		"fix_request_id":  fixRequestID,
		"vulnerabilities": len(req.Vulnerabilities),
		"message":         "Claude Code vulnerability fixer agent has been spawned",
		"timestamp":       time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// fixProgressHandler handles progress updates from the claude-code agent
func fixProgressHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		FixRequestID    string   `json:"fix_request_id"`
		VulnerabilityID string   `json:"vulnerability_id"`
		Status          string   `json:"status"` // completed, failed, skipped
		Message         string   `json:"message"`
		FilesModified   []string `json:"files_modified,omitempty"`
		VaultPath       string   `json:"vault_path,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.FixRequestID == "" || req.VulnerabilityID == "" || req.Status == "" {
		http.Error(w, "Missing required fields: fix_request_id, vulnerability_id, status", http.StatusBadRequest)
		return
	}

	// Log the progress update
	logger.Info("Fix progress [%s]: %s - %s (%s)", req.FixRequestID, req.VulnerabilityID, req.Status, req.Message)

	if len(req.FilesModified) > 0 {
		logger.Info("  Files modified: %v", req.FilesModified)
	}

	if req.VaultPath != "" {
		logger.Info("  Vault path: %s", req.VaultPath)
	}

	response := map[string]interface{}{
		"status":    "acknowledged",
		"timestamp": time.Now().Format(time.RFC3339),
		"message":   "Progress update received",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// fileContentHandler provides secure access to file content for vulnerability analysis
func fileContentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		http.Error(w, "Missing required parameter: path", http.StatusBadRequest)
		return
	}

	// Security: Resolve to absolute path and ensure it's within allowed directories
	vrooli_root := os.Getenv("VROOLI_ROOT")
	if vrooli_root == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			http.Error(w, "Cannot determine home directory", http.StatusInternalServerError)
			return
		}
		vrooli_root = filepath.Join(homeDir, "Vrooli")
	}

	// Clean and resolve the file path
	cleanPath := filepath.Clean(filePath)
	var fullPath string

	// Handle both absolute and relative paths
	if filepath.IsAbs(cleanPath) {
		fullPath = cleanPath
	} else {
		fullPath = filepath.Join(vrooli_root, cleanPath)
	}

	// Security check: ensure the resolved path is within the Vrooli directory
	resolvedPath, err := filepath.Abs(fullPath)
	if err != nil {
		http.Error(w, "Invalid file path", http.StatusBadRequest)
		return
	}

	allowedRoot, err := filepath.Abs(vrooli_root)
	if err != nil {
		http.Error(w, "Cannot resolve Vrooli root path", http.StatusInternalServerError)
		return
	}

	if !strings.HasPrefix(resolvedPath, allowedRoot) {
		http.Error(w, "Access denied: path outside allowed directory", http.StatusForbidden)
		return
	}

	// Read the file content
	content, err := os.ReadFile(resolvedPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "File not found", http.StatusNotFound)
		} else {
			http.Error(w, "Error reading file", http.StatusInternalServerError)
		}
		return
	}

	// Determine language based on file extension
	language := getLanguageFromPath(resolvedPath)

	// Create response
	response := map[string]interface{}{
		"path":     filePath,
		"content":  string(content),
		"language": language,
		"size":     len(content),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getLanguageFromPath determines the programming language based on file extension
func getLanguageFromPath(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))

	// Map extensions to Prism.js language identifiers
	languageMap := map[string]string{
		".js":         "javascript",
		".ts":         "typescript",
		".go":         "go",
		".py":         "python",
		".sh":         "bash",
		".bash":       "bash",
		".zsh":        "bash",
		".fish":       "bash",
		".json":       "json",
		".yaml":       "yaml",
		".yml":        "yaml",
		".toml":       "toml",
		".sql":        "sql",
		".md":         "markdown",
		".html":       "html",
		".css":        "css",
		".scss":       "scss",
		".sass":       "sass",
		".java":       "java",
		".cpp":        "cpp",
		".c":          "c",
		".rs":         "rust",
		".php":        "php",
		".rb":         "ruby",
		".swift":      "swift",
		".kt":         "kotlin",
		".dart":       "dart",
		".r":          "r",
		".dockerfile": "dockerfile",
		".xml":        "xml",
		".ini":        "ini",
		".cfg":        "ini",
		".conf":       "nginx", // Common for config files
	}

	if lang, exists := languageMap[ext]; exists {
		return lang
	}

	// Check for files without extensions but with known names
	filename := strings.ToLower(filepath.Base(filePath))
	if strings.Contains(filename, "dockerfile") {
		return "dockerfile"
	}
	if strings.Contains(filename, "makefile") {
		return "makefile"
	}

	return "text" // Default to plain text
}

// validateEnvironmentForFixes checks that the environment is ready for vulnerability fixing
func validateEnvironmentForFixes() error {
	// Check if vault is accessible
	cmd := exec.Command("resource-vault", "status")
	if err := cmd.Run(); err != nil {
		logger.Warning("Vault status check failed: %v", err)
		logger.Info("    The agent will attempt to work without vault access")
		// Don't fail completely - agent can still do code cleanup without vault
	}

	// Check if resource-claude-code is available
	cmd = exec.Command("resource-claude-code", "--help")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("resource-claude-code not available: %w", err)
	}

	logger.Info("✅ Environment validation passed - ready to spawn vulnerability fixer agent")
	return nil
}

// spawnVulnerabilityFixerAgent spawns a claude-code agent to fix vulnerabilities
func spawnVulnerabilityFixerAgent(fixRequestID string, vulnerabilities []SecurityVulnerability) error {
	// Get VROOLI_ROOT
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %w", err)
		}
		vrooliRoot = filepath.Join(home, "Vrooli")
	}

	// Validate environment before proceeding
	if err := validateEnvironmentForFixes(); err != nil {
		return fmt.Errorf("environment validation failed: %w", err)
	}

	// Load prompt template
	promptPath := filepath.Join(vrooliRoot, "scenarios", "secrets-manager", "initialization", "claude-code", "vulnerability-fixer.md")
	promptTemplate, err := os.ReadFile(promptPath)
	if err != nil {
		return fmt.Errorf("failed to read prompt template: %w", err)
	}

	// Build vulnerabilities JSON
	vulnJSON, err := json.MarshalIndent(vulnerabilities, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal vulnerabilities: %w", err)
	}

	// Replace template variables
	prompt := string(promptTemplate)
	prompt = strings.ReplaceAll(prompt, "{{FIX_REQUEST_ID}}", fixRequestID)
	prompt = strings.ReplaceAll(prompt, "{{TIMESTAMP}}", time.Now().Format(time.RFC3339))
	prompt = strings.ReplaceAll(prompt, "{{VULNERABILITIES_COUNT}}", strconv.Itoa(len(vulnerabilities)))
	prompt = strings.ReplaceAll(prompt, "{{SELECTED_VULNERABILITIES}}", string(vulnJSON))

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Use resource-claude-code directly (following system-monitor pattern)
	cmd := exec.CommandContext(ctx, "resource-claude-code", "run", "-")
	cmd.Dir = vrooliRoot

	// Apply Claude execution settings and context via environment variables
	cmd.Env = append(os.Environ(),
		"MAX_TURNS=75",
		"ANTHROPIC_BETA=computer-use-2024-10-22",
		fmt.Sprintf("SCENARIO_CONTEXT=%s", "secrets-manager"),
		fmt.Sprintf("WORKING_DIR=%s", vrooliRoot),
		fmt.Sprintf("FIX_REQUEST_ID=%s", fixRequestID),
	)

	// Set up stdin with the prompt
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	// Log the agent startup details
	logger.Info("🤖 Spawning vulnerability fixer agent:")
	logger.Info("   📁 Working directory: %s", vrooliRoot)
	logger.Info("   🎯 Vulnerabilities to fix: %d", len(vulnerabilities))
	logger.Info("   ⏱️  Timeout: 10 minutes")
	logger.Info("   📄 Prompt template: %s", promptPath)

	// Start the command
	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start claude-code command: %w", err)
	}

	logger.Info("✅ Claude Code agent started successfully (PID: %d)", cmd.Process.Pid)

	// Write prompt to stdin and close
	go func() {
		defer stdin.Close()
		stdin.Write([]byte(prompt))
	}()

	// Wait for completion (non-blocking)
	go func() {
		err := cmd.Wait()
		if err != nil {
			logger.Info("Claude Code vulnerability fixer completed with error: %v", err)
		} else {
			logger.Info("Claude Code vulnerability fixer completed successfully for request %s", fixRequestID)
		}
	}()

	logger.Info("Successfully spawned vulnerability fixer agent for request %s", fixRequestID)
	return nil
}

func main() {
	// Protect against direct execution - must be run through lifecycle system
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start secrets-manager

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	// Initialize structured logger
	logger = NewLogger("secrets-manager")

	skipDB := strings.EqualFold(os.Getenv("SECRETS_MANAGER_SKIP_DB"), "true")
	if skipDB {
		logger.Info("⚠️ Skipping database initialization (SECRETS_MANAGER_SKIP_DB=true)")
	} else {
		initDB()
		defer db.Close()
		warmSecurityScanCache()
	}
	logger.Info("🚀 Starting Secrets Manager API (database optional)")

	// Create router
	r := mux.NewRouter()

	// Health check (root path for backward compatibility)
	r.HandleFunc("/health", healthHandler).Methods("GET")

	// API routes
	api := r.PathPrefix("/api/v1").Subrouter()

	// Health check (API path for UI integration)
	api.HandleFunc("/health", healthHandler).Methods("GET")

	// Vault secrets integration routes
	api.HandleFunc("/vault/secrets/status", vaultSecretsStatusHandler).Methods("GET")
	api.HandleFunc("/vault/secrets/provision", vaultProvisionHandler).Methods("POST")

	// Security scanning routes
	api.HandleFunc("/security/scan", securityScanHandler).Methods("GET")
	api.HandleFunc("/security/compliance", complianceHandler).Methods("GET")
	api.HandleFunc("/security/vulnerabilities", vulnerabilitiesHandler).Methods("GET")
	api.HandleFunc("/vulnerabilities", vulnerabilitiesHandler).Methods("GET") // Legacy route for backward compatibility
	api.HandleFunc("/vulnerabilities/fix", fixVulnerabilitiesHandler).Methods("POST")
	api.HandleFunc("/vulnerabilities/fix/progress", fixProgressHandler).Methods("POST")
	api.HandleFunc("/vulnerabilities/{id}/status", vulnerabilityStatusHandler).Methods("POST")
	api.HandleFunc("/files/content", fileContentHandler).Methods("GET")
	api.HandleFunc("/orientation/summary", orientationSummaryHandler).Methods("GET")
	api.HandleFunc("/resources/{resource}", resourceDetailHandler).Methods("GET")
	api.HandleFunc("/resources/{resource}/secrets/{secret}", resourceSecretUpdateHandler).Methods("PATCH")
	api.HandleFunc("/resources/{resource}/secrets/{secret}/strategy", secretStrategyHandler).Methods("POST")

	// Legacy routes (keep for backward compatibility)
	api.HandleFunc("/secrets/scan", vaultSecretsStatusHandler).Methods("GET", "POST")
	api.HandleFunc("/secrets/validate", validateHandler).Methods("GET", "POST")
	api.HandleFunc("/secrets/provision", provisionHandler).Methods("POST")

	// Deployment routes (experimental/planned features)
	api.HandleFunc("/deployment/secrets", deploymentSecretsHandler).Methods("POST")
	api.HandleFunc("/deployment/secrets/{scenario}", deploymentSecretsGetHandler).Methods("GET")

	// CORS headers
	corsHeaders := handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Authorization"})
	corsMethods := handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})
	corsOrigins := handlers.AllowedOrigins([]string{"*"})

	// Get port from environment - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		port = os.Getenv("PORT") // Fallback to PORT
		if port == "" {
			log.Fatal("❌ API_PORT or PORT environment variable is required")
		}
	}

	logger.Info("🔐 Secrets Manager API starting on port %s", port)
	logger.Info("   📊 Health check: http://localhost:%s/health", port)
	logger.Info("   🔍 Scan endpoint: http://localhost:%s/api/v1/secrets/scan", port)
	logger.Info("   ✅ Validate endpoint: http://localhost:%s/api/v1/secrets/validate", port)

	// Start server
	server := &http.Server{
		Addr:    ":" + port,
		Handler: handlers.CORS(corsHeaders, corsMethods, corsOrigins)(r),
	}

	log.Fatal(server.ListenAndServe())
}
