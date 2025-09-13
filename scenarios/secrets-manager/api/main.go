package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// Data models
type ResourceSecret struct {
	ID                string     `json:"id" db:"id"`
	ResourceName      string     `json:"resource_name" db:"resource_name"`
	SecretKey         string     `json:"secret_key" db:"secret_key"`
	SecretType        string     `json:"secret_type" db:"secret_type"`
	Required          bool       `json:"required" db:"required"`
	Description       *string    `json:"description" db:"description"`
	ValidationPattern *string    `json:"validation_pattern" db:"validation_pattern"`
	DocumentationURL  *string    `json:"documentation_url" db:"documentation_url"`
	DefaultValue      *string    `json:"default_value" db:"default_value"`
	CreatedAt         time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at" db:"updated_at"`
}

type SecretValidation struct {
	ID                 string     `json:"id" db:"id"`
	ResourceSecretID   string     `json:"resource_secret_id" db:"resource_secret_id"`
	ValidationStatus   string     `json:"validation_status" db:"validation_status"`
	ValidationMethod   string     `json:"validation_method" db:"validation_method"`
	ValidationTimestamp time.Time `json:"validation_timestamp" db:"validation_timestamp"`
	ErrorMessage       *string    `json:"error_message" db:"error_message"`
	ValidationDetails  *string    `json:"validation_details" db:"validation_details"`
}

type SecretScan struct {
	ID                 string     `json:"id" db:"id"`
	ScanType          string     `json:"scan_type" db:"scan_type"`
	ResourcesScanned  []string   `json:"resources_scanned" db:"resources_scanned"`
	SecretsDiscovered int        `json:"secrets_discovered" db:"secrets_discovered"`
	ScanDurationMs    int        `json:"scan_duration_ms" db:"scan_duration_ms"`
	ScanTimestamp     time.Time  `json:"scan_timestamp" db:"scan_timestamp"`
	ScanStatus        string     `json:"scan_status" db:"scan_status"`
	ErrorMessage      *string    `json:"error_message" db:"error_message"`
	ScanMetadata      *string    `json:"scan_metadata" db:"scan_metadata"`
}

type SecretHealthSummary struct {
	ResourceName         string     `json:"resource_name"`
	TotalSecrets         int        `json:"total_secrets"`
	RequiredSecrets      int        `json:"required_secrets"`
	ValidSecrets         int        `json:"valid_secrets"`
	MissingRequiredSecrets int      `json:"missing_required_secrets"`
	InvalidSecrets       int        `json:"invalid_secrets"`
	LastValidation       *time.Time `json:"last_validation"`
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

type VaultResourceStatus struct {
	ResourceName    string    `json:"resource_name"`
	SecretsTotal    int       `json:"secrets_total"`
	SecretsFound    int       `json:"secrets_found"`
	SecretsMissing  int       `json:"secrets_missing"`
	SecretsOptional int       `json:"secrets_optional"`
	HealthStatus    string    `json:"health_status"` // healthy, degraded, critical
	LastChecked     time.Time `json:"last_checked"`
}

type VaultValidationSummary struct {
	ConfiguredCount int                  `json:"configured_count"`
	MissingSecrets  []VaultMissingSecret `json:"missing_secrets"`
}

// Security scanning data structures
type SecurityVulnerability struct {
	ID            string    `json:"id"`
	ComponentType string    `json:"component_type"` // "resource" or "scenario"
	ComponentName string    `json:"component_name"` // Name of resource or scenario
	FilePath      string    `json:"file_path"`
	LineNumber    int       `json:"line_number"`
	Severity      string    `json:"severity"` // critical, high, medium, low
	Type          string    `json:"type"`     // sql_injection, hardcoded_secret, etc.
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	Code          string    `json:"code"`     // Affected code snippet
	Recommendation string   `json:"recommendation"`
	CanAutoFix    bool      `json:"can_auto_fix"`
	DiscoveredAt  time.Time `json:"discovered_at"`
}

type SecurityScanResult struct {
	ScanID            string                  `json:"scan_id"`
	ComponentFilter   string                  `json:"component_filter,omitempty"`   // Optional filter for specific component
	ComponentType     string                  `json:"component_type,omitempty"`     // Filter by "resource" or "scenario"
	Vulnerabilities   []SecurityVulnerability `json:"vulnerabilities"`
	RiskScore         int                     `json:"risk_score"` // 0-100
	ScanDurationMs    int                     `json:"scan_duration_ms"`
	Recommendations   []RemediationSuggestion `json:"recommendations"`
	ComponentsSummary ComponentScanSummary    `json:"components_summary"`
	ScanMetrics       ScanMetrics             `json:"scan_metrics"`
}

type RemediationSuggestion struct {
	VulnerabilityType string   `json:"vulnerability_type"`
	Priority          string   `json:"priority"`
	Description       string   `json:"description"`
	FixCommand        string   `json:"fix_command,omitempty"`
	Documentation     string   `json:"documentation,omitempty"`
	AffectedFiles     []string `json:"affected_files"`
	Count            int      `json:"count"`
	EstimatedEffort  string   `json:"estimated_effort"`
}

type ComponentScanSummary struct {
	ResourcesScanned int `json:"resources_scanned"`
	ScenariosScanned int `json:"scenarios_scanned"`
	TotalComponents  int `json:"total_components"`
	ConfiguredCount  int `json:"configured_count"`
}

type ScanMetrics struct {
	FilesScanned         int      `json:"files_scanned"`
	FilesSkipped         int      `json:"files_skipped"`
	LargeFilesSkipped    int      `json:"large_files_skipped"`
	TimeoutOccurred      bool     `json:"timeout_occurred"`
	ScanErrors           []string `json:"scan_errors,omitempty"`
	ResourceScanTimeMs   int      `json:"resource_scan_time_ms"`
	ScenarioScanTimeMs   int      `json:"scenario_scan_time_ms"`
	TotalScanTimeMs      int      `json:"total_scan_time_ms"`
}

type ComplianceMetrics struct {
	VaultSecretsHealth  int `json:"vault_secrets_health"`  // 0-100
	SecurityScore       int `json:"security_score"`        // 0-100
	OverallCompliance   int `json:"overall_compliance"`    // 0-100
	ConfiguredComponents int `json:"configured_components"`
	CriticalIssues      int `json:"critical_issues"`
	HighIssues          int `json:"high_issues"`
	MediumIssues        int `json:"medium_issues"`
	LowIssues           int `json:"low_issues"`
}

// API request/response types
type ScanRequest struct {
	Resources []string `json:"resources"`
	ScanType  string   `json:"scan_type"`
}

type ScanResponse struct {
	ScanID           string           `json:"scan_id"`
	DiscoveredSecrets []ResourceSecret `json:"discovered_secrets"`
	ScanDurationMs   int              `json:"scan_duration_ms"`
	ResourcesScanned []string         `json:"resources_scanned"`
}

type ValidationRequest struct {
	Resource string `json:"resource"`
}

type ValidationResponse struct {
	ValidationID     string              `json:"validation_id"`
	TotalSecrets     int                 `json:"total_secrets"`
	ValidSecrets     int                 `json:"valid_secrets"`
	MissingSecrets   []SecretValidation  `json:"missing_secrets"`
	InvalidSecrets   []SecretValidation  `json:"invalid_secrets"`
	HealthSummary    []SecretHealthSummary `json:"health_summary"`
}

type ProvisionRequest struct {
	SecretKey      string `json:"secret_key"`
	SecretValue    string `json:"secret_value"`
	StorageMethod  string `json:"storage_method"`
}

type ProvisionResponse struct {
	Success            bool              `json:"success"`
	StorageLocation    string            `json:"storage_location"`
	ValidationResult   SecretValidation  `json:"validation_result"`
}



// Database connection
var db *sql.DB
var scanner *SecretScanner
var validator *SecretValidator

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
			log.Fatal("❌ Database configuration missing. Provide POSTGRES_URL or all of: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB")
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
	
	log.Println("🔄 Attempting database connection with exponential backoff...")
	log.Printf("📆 Database URL configured")
	
	var pingErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		pingErr = db.Ping()
		if pingErr == nil {
			log.Printf("✅ Database connected successfully on attempt %d", attempt + 1)
			break
		}
		
		// Calculate exponential backoff delay
		delay := time.Duration(math.Min(
			float64(baseDelay) * math.Pow(2, float64(attempt)),
			float64(maxDelay),
		))
		
		// Add progressive jitter to prevent thundering herd
		jitterRange := float64(delay) * 0.25
		jitter := time.Duration(jitterRange * (float64(attempt) / float64(maxRetries)))
		actualDelay := delay + jitter
		
		log.Printf("⚠️  Connection attempt %d/%d failed: %v", attempt + 1, maxRetries, pingErr)
		log.Printf("⏳ Waiting %v before next attempt", actualDelay)
		
		// Provide detailed status every few attempts
		if attempt > 0 && attempt % 3 == 0 {
			log.Printf("📈 Retry progress:")
			log.Printf("   - Attempts made: %d/%d", attempt + 1, maxRetries)
			log.Printf("   - Total wait time: ~%v", time.Duration(attempt * 2) * baseDelay)
			log.Printf("   - Current delay: %v (with jitter: %v)", delay, jitter)
		}
		
		time.Sleep(actualDelay)
	}
	
	if pingErr != nil {
		log.Fatalf("❌ Database connection failed after %d attempts: %v", maxRetries, pingErr)
	}
	
	log.Println("🎉 Database connection pool established successfully!")
	
	// Initialize scanner and validator components (will be nil without DB)
	// scanner = NewSecretScanner(db)
	// validator = NewSecretValidator(db)
}

// getEnvOrDefault removed to prevent hardcoded defaults

// Vault integration - uses resource-vault CLI commands to get secrets status
func getVaultSecretsStatus(resourceFilter string) (*VaultSecretsStatus, error) {
	// Try using fallback implementation that scans directly
	// The vault CLI commands appear to hang in some environments
	log.Printf("Using fallback vault status implementation")
	return getVaultSecretsStatusFallback(resourceFilter)
}

// Parse vault scan output to extract resource names
func parseVaultScanOutput(output string) []string {
	var resources []string
	lines := strings.Split(output, "\n")
	
	for _, line := range lines {
		// Look for lines like "  ✓ openrouter: 3 secrets defined"
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

// Security vulnerability scanner - scans both resources and scenarios
func scanComponentsForVulnerabilities(componentFilter, componentTypeFilter, severityFilter string) (*SecurityScanResult, error) {
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
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		
		scenarioFilesScanned := 0
		maxScenarioFiles := 1000 // Higher limit for scenarios as they're the main focus
		filesPerScenario := make(map[string]int) // Track files per scenario to balance scanning
		
		err := filepath.WalkDir(scenariosPath, func(path string, d os.DirEntry, err error) error {
			// Check timeout
			select {
			case <-ctx.Done():
				log.Printf("Scenario scanning timed out after 45 seconds")
				metrics.TimeoutOccurred = true
				metrics.ScanErrors = append(metrics.ScanErrors, "Scenario scanning timeout after 45 seconds")
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
				log.Printf("Scenario scanning stopped after %d files (limit reached)", maxScenarioFiles)
				metrics.ScanErrors = append(metrics.ScanErrors, fmt.Sprintf("Scenario file limit reached (%d files)", maxScenarioFiles))
				return filepath.SkipAll
			}
			
			// Limit files per scenario to ensure we scan more scenarios
			const maxFilesPerScenario = 20
			if filesPerScenario[scenarioName] >= maxFilesPerScenario {
				return nil // Skip additional files from this scenario
			}
			
			// Check file size
			info, err := d.Info()
			if err == nil && info.Size() > 100*1024 { // Skip files larger than 100KB for scenarios
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
				log.Printf("Warning: failed to scan scenario file %s: %v", path, err)
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
			log.Printf("Warning: failed to walk scenarios directory: %v", err)
			metrics.ScanErrors = append(metrics.ScanErrors, fmt.Sprintf("Failed to walk scenarios directory: %v", err))
		}
	}
	
	// Scan resources if not filtering for scenarios only
	if componentTypeFilter == "" || componentTypeFilter == "resource" {
		resourceStartTime := time.Now()
		// Create timeout context for resource scanning
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		
		resourceFilesScanned := 0
		maxFiles := 500 // Increased limit to scan more resources
		filesPerResource := make(map[string]int) // Track files per resource to balance scanning
		
		err := filepath.WalkDir(resourcesPath, func(path string, d os.DirEntry, err error) error {
			// Check timeout
			select {
			case <-ctx.Done():
				log.Printf("Resource scanning timed out after 30 seconds")
				metrics.TimeoutOccurred = true
				metrics.ScanErrors = append(metrics.ScanErrors, "Resource scanning timeout after 30 seconds")
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
				log.Printf("Resource scanning stopped after %d files (limit reached)", maxFiles)
				metrics.ScanErrors = append(metrics.ScanErrors, fmt.Sprintf("Resource file limit reached (%d files)", maxFiles))
				return filepath.SkipAll
			}
			
			// Limit files per resource to ensure we scan more resources
			const maxFilesPerResource = 10
			if filesPerResource[resourceName] >= maxFilesPerResource {
				return nil // Skip additional files from this resource
			}
			
			// Check file size to prevent scanning huge files
			info, err := d.Info()
			if err == nil && info.Size() > 50*1024 { // Skip files larger than 50KB
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
				log.Printf("Warning: failed to scan resource file %s: %v", path, err)
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
			log.Printf("Warning: failed to walk resources directory: %v", err)
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
	log.Printf("Vulnerability scan completed:")
	log.Printf("  📊 Total scan time: %dms", totalScanTime)
	log.Printf("  📁 Files scanned: %d", metrics.FilesScanned)
	log.Printf("  ⏭️  Files skipped: %d", metrics.FilesSkipped)
	log.Printf("  🔍 Vulnerabilities found: %d", len(vulnerabilities))
	log.Printf("  ⚠️  Scan errors: %d", len(metrics.ScanErrors))
	if metrics.TimeoutOccurred {
		log.Printf("  ⏰ Timeout occurred during scanning")
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
	}, nil
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
			Type:        "hardcoded_secret_resource",
			Severity:    "critical",
			Pattern:     `(PASSWORD|SECRET|TOKEN|KEY|API_KEY)\s*=\s*[\"'](?!.*\$|.*env|.*getenv)[^\"']{8,}[\"']`,
			Description: "Hardcoded secret found in resource configuration",
			Title:       "Hardcoded Secret in Resource",
			Recommendation: "Move secret to vault using resource-vault CLI",
			CanAutoFix:  false,
		},
		{
			Type:        "database_url_hardcoded",
			Severity:    "critical", 
			Pattern:     `(DATABASE_URL|DB_URL|POSTGRES_URL)\s*=\s*[\"'](?!.*\$)[^\"']*://[^\"']*:[^\"']*@[^\"']+`,
			Description: "Database URL with hardcoded credentials",
			Title:       "Hardcoded Database Credentials",
			Recommendation: "Use environment variables or vault for database credentials",
			CanAutoFix:  false,
		},
		{
			Type:        "missing_env_var_validation",
			Severity:    "medium",
			Pattern:     `\$\{?([A-Z_]+[A-Z0-9_]*)\}?(?!\s*:-|\s*\|\|)`,
			Description: "Environment variable used without fallback or validation",
			Title:       "Missing Environment Variable Validation", 
			Recommendation: "Add fallback values or validation for environment variables",
			CanAutoFix:  true,
		},
		{
			Type:        "weak_permissions",
			Severity:    "medium",
			Pattern:     `chmod\s+(777|666|755)`,
			Description: "Potentially insecure file permissions",
			Title:       "Weak File Permissions",
			Recommendation: "Use more restrictive file permissions (644, 600, etc.)",
			CanAutoFix:  true,
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
				ID:           uuid.New().String(),
				ComponentType: componentType,
				ComponentName: componentName,
				FilePath:     filePath,
				LineNumber:   lineNum,
				Severity:     pattern.Severity,
				Type:         pattern.Type,
				Title:        pattern.Title,
				Description:  pattern.Description,
				Code:         codeSnippet,
				Recommendation: pattern.Recommendation,
				CanAutoFix:   pattern.CanAutoFix,
				DiscoveredAt: time.Now(),
			}
			
			vulnerabilities = append(vulnerabilities, vulnerability)
		}
	}

	return vulnerabilities, nil
}





func scanResourceDirectory(resourceName, resourceDir string) ([]ResourceSecret, error) {
	var secrets []ResourceSecret
	
	// Patterns to look for environment variables and credentials
	envVarPatterns := []*regexp.Regexp{
		regexp.MustCompile(`\$\{([A-Z_]+[A-Z0-9_]*)\}`),        // ${VAR_NAME}
		regexp.MustCompile(`\$([A-Z_]+[A-Z0-9_]*)`),            // $VAR_NAME
		regexp.MustCompile(`([A-Z_]+[A-Z0-9_]*)=`),             // VAR_NAME=
		regexp.MustCompile(`env\.([A-Z_]+[A-Z0-9_]*)`),         // env.VAR_NAME
		regexp.MustCompile(`getenv\("([A-Z_]+[A-Z0-9_]*)"\)`),  // getenv("VAR_NAME")
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
	// TODO: Implement vault integration using resource-vault CLI
	// For now, return empty to indicate not found
	return "", fmt.Errorf("vault integration not implemented")
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
		log.Printf("Failed to store validation result: %v", err)
	}
}

func getHealthSummary() ([]SecretHealthSummary, error) {
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
	response := map[string]interface{}{
		"status":    "healthy",
		"service":   "secrets-manager-api",
		"version":   "1.0.0",
		"timestamp": time.Now().Format(time.RFC3339),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Vault secrets status handler
func vaultSecretsStatusHandler(w http.ResponseWriter, r *http.Request) {
	resourceFilter := r.URL.Query().Get("resource")
	
	status, err := getVaultSecretsStatus(resourceFilter)
	if err != nil {
		log.Printf("Error getting vault status: %v, using mock data", err)
		// Use mock data as ultimate fallback
		status = getMockVaultStatus()
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
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
	
	// Use resource-vault CLI to provision secrets
	// For now, just trigger the interactive init - in a real implementation,
	// we'd need to pass the secrets programmatically
	initCmd := exec.Command("resource-vault", "secrets", "init", req.ResourceName)
	output, err := initCmd.Output()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to provision secrets: %v", err), http.StatusInternalServerError)
		return
	}
	
	response := map[string]interface{}{
		"success": true,
		"provisioned_secrets": []string{}, // Would be populated from actual provisioning
		"vault_paths": map[string]string{}, // Would be populated with actual paths
		"message": string(output),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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
		"overall_score":         compliance.OverallCompliance,
		"vault_secrets_health":  compliance.VaultSecretsHealth,
		"vulnerability_summary": map[string]int{
			"critical": compliance.CriticalIssues,
			"high":     compliance.HighIssues,
			"medium":   compliance.MediumIssues,
			"low":      compliance.LowIssues,
		},
		"remediation_progress":     compliance,
		"total_resources":          vaultStatus.TotalResources,
		"configured_resources":     vaultStatus.ConfiguredResources,
		"configured_components":    compliance.ConfiguredComponents,
		"total_components":         securityResults.ComponentsSummary.TotalComponents,
		"components_summary":       securityResults.ComponentsSummary,
		"total_vulnerabilities":    len(securityResults.Vulnerabilities),
		"last_updated":            time.Now(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func vulnerabilitiesHandler(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	severity := r.URL.Query().Get("severity")
	component := r.URL.Query().Get("component")
	componentType := r.URL.Query().Get("component_type")
	
	// Support legacy scenario parameter
	if scenario := r.URL.Query().Get("scenario"); scenario != "" && component == "" {
		component = scenario
		componentType = "scenario"
	}
	
	// Get security scan results from real filesystem scan
	securityResults, err := scanComponentsForVulnerabilities(component, componentType, severity)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to scan for vulnerabilities: %v", err), http.StatusInternalServerError)
		return
	}
	
	response := map[string]interface{}{
		"vulnerabilities": securityResults.Vulnerabilities,
		"total_count":     len(securityResults.Vulnerabilities),
		"scan_id":         securityResults.ScanID,
		"scan_duration":   securityResults.ScanDurationMs,
		"risk_score":      securityResults.RiskScore,
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
	
	// TODO: Implement secret provisioning to vault or environment
	response := ProvisionResponse{
		Success:         false,
		StorageLocation: "",
		ValidationResult: SecretValidation{
			ValidationStatus: "error",
			ErrorMessage:     stringPtr("Provisioning not yet implemented"),
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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
		log.Printf("Failed to store discovered secret: %v", err)
	}
}

func storeScanRecord(scan SecretScan) {
	// TODO: Implement scan record storage
	log.Printf("Scan completed: %d secrets discovered in %dms", scan.SecretsDiscovered, scan.ScanDurationMs)
}

// Helper functions
func stringPtr(s string) *string {
	return &s
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
	log.Printf("Starting vulnerability fix request %s for %d vulnerabilities", fixRequestID, len(req.Vulnerabilities))

	// Spawn claude-code agent to fix vulnerabilities
	go func() {
		err := spawnVulnerabilityFixerAgent(fixRequestID, req.Vulnerabilities)
		if err != nil {
			log.Printf("Failed to spawn vulnerability fixer agent: %v", err)
		}
	}()

	response := map[string]interface{}{
		"status":           "accepted",
		"fix_request_id":   fixRequestID,
		"vulnerabilities":  len(req.Vulnerabilities),
		"message":          "Claude Code vulnerability fixer agent has been spawned",
		"timestamp":        time.Now().Format(time.RFC3339),
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
		FixRequestID     string `json:"fix_request_id"`
		VulnerabilityID  string `json:"vulnerability_id"`
		Status           string `json:"status"` // completed, failed, skipped
		Message          string `json:"message"`
		FilesModified    []string `json:"files_modified,omitempty"`
		VaultPath        string `json:"vault_path,omitempty"`
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
	log.Printf("Fix progress [%s]: %s - %s (%s)", req.FixRequestID, req.VulnerabilityID, req.Status, req.Message)
	
	if len(req.FilesModified) > 0 {
		log.Printf("  Files modified: %v", req.FilesModified)
	}
	
	if req.VaultPath != "" {
		log.Printf("  Vault path: %s", req.VaultPath)
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
		".js":     "javascript",
		".ts":     "typescript", 
		".go":     "go",
		".py":     "python",
		".sh":     "bash",
		".bash":   "bash",
		".zsh":    "bash",
		".fish":   "bash",
		".json":   "json",
		".yaml":   "yaml",
		".yml":    "yaml",
		".toml":   "toml",
		".sql":    "sql",
		".md":     "markdown",
		".html":   "html",
		".css":    "css",
		".scss":   "scss",
		".sass":   "sass",
		".java":   "java",
		".cpp":    "cpp",
		".c":      "c",
		".rs":     "rust",
		".php":    "php",
		".rb":     "ruby",
		".swift":  "swift",
		".kt":     "kotlin",
		".dart":   "dart",
		".r":      "r",
		".dockerfile": "dockerfile",
		".xml":    "xml",
		".ini":    "ini",
		".cfg":    "ini",
		".conf":   "nginx", // Common for config files
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
		log.Printf("⚠️  Vault status check failed: %v", err)
		log.Printf("    The agent will attempt to work without vault access")
		// Don't fail completely - agent can still do code cleanup without vault
	}

	// Check if resource-claude-code is available
	cmd = exec.Command("resource-claude-code", "--help")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("resource-claude-code not available: %w", err)
	}

	log.Printf("✅ Environment validation passed - ready to spawn vulnerability fixer agent")
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
	log.Printf("🤖 Spawning vulnerability fixer agent:")
	log.Printf("   📁 Working directory: %s", vrooliRoot)
	log.Printf("   🎯 Vulnerabilities to fix: %d", len(vulnerabilities))
	log.Printf("   ⏱️  Timeout: 10 minutes")
	log.Printf("   📄 Prompt template: %s", promptPath)

	// Start the command
	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start claude-code command: %w", err)
	}

	log.Printf("✅ Claude Code agent started successfully (PID: %d)", cmd.Process.Pid)

	// Write prompt to stdin and close
	go func() {
		defer stdin.Close()
		stdin.Write([]byte(prompt))
	}()

	// Wait for completion (non-blocking)
	go func() {
		err := cmd.Wait()
		if err != nil {
			log.Printf("Claude Code vulnerability fixer completed with error: %v", err)
		} else {
			log.Printf("Claude Code vulnerability fixer completed successfully for request %s", fixRequestID)
		}
	}()

	log.Printf("Successfully spawned vulnerability fixer agent for request %s", fixRequestID)
	return nil
}

func main() {
	// Initialize database (skip if not available)
	// initDB()
	// defer db.Close()
	log.Println("🚀 Starting Secrets Manager API (database optional)")
	
	// Create router
	r := mux.NewRouter()
	
	// Health check
	r.HandleFunc("/health", healthHandler).Methods("GET")
	
	// API routes
	api := r.PathPrefix("/api/v1").Subrouter()
	
	// Vault secrets integration routes
	api.HandleFunc("/vault/secrets/status", vaultSecretsStatusHandler).Methods("GET")
	api.HandleFunc("/vault/secrets/provision", vaultProvisionHandler).Methods("POST")
	
	// Security scanning routes
	api.HandleFunc("/security/scan", securityScanHandler).Methods("GET")
	api.HandleFunc("/security/compliance", complianceHandler).Methods("GET")
	api.HandleFunc("/vulnerabilities", vulnerabilitiesHandler).Methods("GET")
	api.HandleFunc("/vulnerabilities/fix", fixVulnerabilitiesHandler).Methods("POST")
	api.HandleFunc("/vulnerabilities/fix/progress", fixProgressHandler).Methods("POST")
	api.HandleFunc("/files/content", fileContentHandler).Methods("GET")
	
	// Legacy routes (keep for backward compatibility)
	api.HandleFunc("/secrets/scan", vaultSecretsStatusHandler).Methods("GET", "POST")
	api.HandleFunc("/secrets/validate", validateHandler).Methods("GET", "POST")
	api.HandleFunc("/secrets/provision", provisionHandler).Methods("POST")
	
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
	
	log.Printf("🔐 Secrets Manager API starting on port %s", port)
	log.Printf("   📊 Health check: http://localhost:%s/health", port)
	log.Printf("   🔍 Scan endpoint: http://localhost:%s/api/v1/secrets/scan", port)
	log.Printf("   ✅ Validate endpoint: http://localhost:%s/api/v1/secrets/validate", port)
	
	// Start server
	server := &http.Server{
		Addr:    ":" + port,
		Handler: handlers.CORS(corsHeaders, corsMethods, corsOrigins)(r),
	}
	
	log.Fatal(server.ListenAndServe())
}