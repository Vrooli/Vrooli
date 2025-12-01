package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

// Package-level logger
var logger *Logger

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil && logger != nil {
		logger.Error("failed to write JSON response: %v", err)
	}
}

// Database connection
var db *sql.DB

func initDB() *sql.DB {
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

	return db
}

// getEnvOrDefault removed to prevent hardcoded defaults

// HTTP Handlers
func (s *APIServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	// Check database connectivity
	dbConnected := false
	var dbLatency float64
	var dbError map[string]interface{}

	if s.db != nil {
		start := time.Now()
		err := s.db.Ping()
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
func (s *APIServer) vaultSecretsStatusHandler(w http.ResponseWriter, r *http.Request) {
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

func (s *APIServer) orientationSummaryHandler(w http.ResponseWriter, r *http.Request) {
	if s.orientationBuilder == nil {
		http.Error(w, "orientation summary unavailable: database not initialized", http.StatusServiceUnavailable)
		return
	}

	summary, err := s.orientationBuilder.Build(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to build orientation summary: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// Security scan handler
func (s *APIServer) securityScanHandler(w http.ResponseWriter, r *http.Request) {
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
func (s *APIServer) vaultProvisionHandler(w http.ResponseWriter, r *http.Request) {
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
	updates := s.normalizeProvisionSecrets(req.Secrets)
	if len(updates) == 0 {
		http.Error(w, "no secrets provided", http.StatusBadRequest)
		return
	}

	result, err := s.performSecretProvision(r.Context(), req.ResourceName, updates)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *APIServer) normalizeProvisionSecrets(raw map[string]string) map[string]string {
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

func (s *APIServer) performSecretProvision(ctx context.Context, resource string, secrets map[string]string) (*ProvisionResponse, error) {
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
func (s *APIServer) complianceHandler(w http.ResponseWriter, r *http.Request) {
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

	compliance := calculateComplianceMetrics(vaultStatus, securityResults)
	response := buildComplianceResponse(compliance, vaultStatus, securityResults)

	writeJSON(w, http.StatusOK, response)
}

type vulnerabilitySeverityCounts struct {
	critical int
	high     int
	medium   int
	low      int
}

func calculateComplianceMetrics(vaultStatus *VaultSecretsStatus, securityResults *SecurityScanResult) ComplianceMetrics {
	vault := VaultSecretsStatus{}
	if vaultStatus != nil {
		vault = *vaultStatus
	}

	summary := ComponentScanSummary{}
	vulnerabilities := []SecurityVulnerability{}
	riskScore := 0

	if securityResults != nil {
		summary = securityResults.ComponentsSummary
		vulnerabilities = securityResults.Vulnerabilities
		riskScore = securityResults.RiskScore
	}

	vaultHealth := calculatePercentage(vault.ConfiguredResources, vault.TotalResources)
	securityScore := 100 - riskScore
	if securityScore < 0 {
		securityScore = 0
	}
	overallCompliance := (vaultHealth + securityScore) / 2

	severityCounts := tallyVulnerabilitySeverities(vulnerabilities)
	configuredComponents := vault.ConfiguredResources
	if summary.TotalComponents > 0 {
		configuredComponents += summary.ConfiguredCount
	}

	return ComplianceMetrics{
		VaultSecretsHealth:   vaultHealth,
		SecurityScore:        securityScore,
		OverallCompliance:    overallCompliance,
		ConfiguredComponents: configuredComponents,
		CriticalIssues:       severityCounts.critical,
		HighIssues:           severityCounts.high,
		MediumIssues:         severityCounts.medium,
		LowIssues:            severityCounts.low,
	}
}

func calculatePercentage(part, total int) int {
	if total == 0 {
		return 0
	}
	return (part * 100) / total
}

func tallyVulnerabilitySeverities(vulnerabilities []SecurityVulnerability) vulnerabilitySeverityCounts {
	counts := vulnerabilitySeverityCounts{}
	for _, vuln := range vulnerabilities {
		switch vuln.Severity {
		case "critical":
			counts.critical++
		case "high":
			counts.high++
		case "medium":
			counts.medium++
		case "low":
			counts.low++
		}
	}
	return counts
}

func buildComplianceResponse(metrics ComplianceMetrics, vaultStatus *VaultSecretsStatus, securityResults *SecurityScanResult) map[string]interface{} {
	vault := VaultSecretsStatus{}
	if vaultStatus != nil {
		vault = *vaultStatus
	}

	componentsSummary := ComponentScanSummary{}
	totalVulnerabilities := 0
	if securityResults != nil {
		componentsSummary = securityResults.ComponentsSummary
		totalVulnerabilities = len(securityResults.Vulnerabilities)
	}

	return map[string]interface{}{
		"overall_score":        metrics.OverallCompliance,
		"vault_secrets_health": metrics.VaultSecretsHealth,
		"vulnerability_summary": map[string]int{
			"critical": metrics.CriticalIssues,
			"high":     metrics.HighIssues,
			"medium":   metrics.MediumIssues,
			"low":      metrics.LowIssues,
		},
		"remediation_progress":  metrics,
		"total_resources":       vault.TotalResources,
		"configured_resources":  vault.ConfiguredResources,
		"configured_components": metrics.ConfiguredComponents,
		"total_components":      componentsSummary.TotalComponents,
		"components_summary":    componentsSummary,
		"total_vulnerabilities": totalVulnerabilities,
		"last_updated":          time.Now(),
	}
}

func (s *APIServer) vulnerabilitiesHandler(w http.ResponseWriter, r *http.Request) {
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
		quickResults := buildQuickSecurityScanResult()
		response := buildVulnerabilityPayload(quickResults, component, severity, "quick")
		writeJSON(w, http.StatusOK, response)
		return
	}

	// Get security scan results from real filesystem scan
	securityResults, err := scanComponentsForVulnerabilities(component, componentType, severity)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to scan for vulnerabilities: %v", err), http.StatusInternalServerError)
		return
	}

	// [REQ:SEC-SCAN-002] Vulnerability listing endpoint
	response := buildVulnerabilityPayload(securityResults, component, severity, "")
	writeJSON(w, http.StatusOK, response)
}

func buildQuickSecurityScanResult() *SecurityScanResult {
	return &SecurityScanResult{
		ScanID:          uuid.New().String(),
		Vulnerabilities: []SecurityVulnerability{},
		RiskScore:       0,
		ScanDurationMs:  1,
		Recommendations: generateRemediationSuggestions([]SecurityVulnerability{}),
	}
}

func buildVulnerabilityPayload(result *SecurityScanResult, component, severity, mode string) map[string]interface{} {
	if result == nil {
		result = &SecurityScanResult{}
	}

	return map[string]interface{}{
		"vulnerabilities": result.Vulnerabilities,
		"total_count":     len(result.Vulnerabilities),
		"scan_metadata":   buildScanMetadata(result, component, severity, mode),
		"recommendations": result.Recommendations,
	}
}

func buildScanMetadata(result *SecurityScanResult, component, severity, mode string) map[string]interface{} {
	if result == nil {
		result = &SecurityScanResult{}
	}

	metadata := map[string]interface{}{
		"scan_id":       result.ScanID,
		"scan_duration": result.ScanDurationMs,
		"risk_score":    result.RiskScore,
		"component":     component,
		"severity":      severity,
		"timestamp":     time.Now().Format(time.RFC3339),
	}

	if mode != "" {
		metadata["mode"] = mode
	}

	return metadata
}

func (s *APIServer) validateHandler(w http.ResponseWriter, r *http.Request) {
	var req ValidationRequest
	if r.Method == "POST" {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
	}
	if s.validator == nil {
		http.Error(w, "validator not ready (database unavailable)", http.StatusServiceUnavailable)
		return
	}

	response, err := s.validator.ValidateSecrets(req.Resource)
	if err != nil {
		http.Error(w, fmt.Sprintf("Validation failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *APIServer) provisionHandler(w http.ResponseWriter, r *http.Request) {
	var req ProvisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	secrets := s.normalizeProvisionSecrets(req.Secrets)
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

	result, err := s.performSecretProvision(r.Context(), strings.TrimSpace(req.Resource), secrets)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// deploymentSecretsHandler generates deployment manifests for specific tiers
func (s *APIServer) deploymentSecretsHandler(w http.ResponseWriter, r *http.Request) {
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
func (s *APIServer) deploymentSecretsGetHandler(w http.ResponseWriter, r *http.Request) {
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

func (s *APIServer) resourceDetailHandler(w http.ResponseWriter, r *http.Request) {
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

func (s *APIServer) resourceSecretUpdateHandler(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
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
	if err := s.db.QueryRowContext(r.Context(), query, args...).Scan(&secretID); err != nil {
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

func (s *APIServer) secretStrategyHandler(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
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

func (s *APIServer) vulnerabilityStatusHandler(w http.ResponseWriter, r *http.Request) {
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
		db = initDB()
		defer db.Close()
		warmSecurityScanCache()
	}
	logger.Info("🚀 Starting Secrets Manager API (database optional)")

	apiServer := newAPIServer(db, logger)
	r := apiServer.routes()

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
