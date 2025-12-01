package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

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

	activeScansMutex sync.RWMutex
	activeScans      = make(map[string]*ProgressiveScanResult)
)

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
