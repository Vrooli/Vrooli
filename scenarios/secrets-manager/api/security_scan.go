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

// -----------------------------------------------------------------------------
// Path Resolution
// -----------------------------------------------------------------------------

// vrooliPaths holds the resolved paths for Vrooli directories.
type vrooliPaths struct {
	root      string
	scenarios string
	resources string
}

// getVrooliPaths resolves the VROOLI_ROOT and derived paths.
func getVrooliPaths() (vrooliPaths, error) {
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return vrooliPaths{}, fmt.Errorf("failed to get user home directory: %w", err)
		}
		vrooliRoot = filepath.Join(home, "Vrooli")
	}
	return vrooliPaths{
		root:      vrooliRoot,
		scenarios: filepath.Join(vrooliRoot, "scenarios"),
		resources: filepath.Join(vrooliRoot, "resources"),
	}, nil
}

// -----------------------------------------------------------------------------
// Directory Walking Infrastructure
// -----------------------------------------------------------------------------

// scanWalkConfig configures a directory walk for vulnerability scanning.
type scanWalkConfig struct {
	basePath          string
	componentType     string // "scenario" or "resource"
	componentFilter   string
	severityFilter    string
	allowedExtensions []string
	maxFileSize       int64 // in bytes
	maxFiles          int
	maxFilesPerComp   int
	timeoutSeconds    int
}

// scanWalkResult captures the results of a directory walk.
type scanWalkResult struct {
	vulnerabilities []SecurityVulnerability
	componentsFound map[string]bool
	filesScanned    int
	filesSkipped    int
	largeSkipped    int
	scanTimeMs      int
	errors          []string
	timedOut        bool
}

// walkAndScan performs a directory walk with the given configuration.
// This consolidates the common walking logic used for both scenarios and resources.
func walkAndScan(ctx context.Context, cfg scanWalkConfig) scanWalkResult {
	startTime := time.Now()
	result := scanWalkResult{
		vulnerabilities: []SecurityVulnerability{},
		componentsFound: make(map[string]bool),
		errors:          []string{},
	}

	filesPerComponent := make(map[string]int)
	filesScanned := 0

	err := filepath.WalkDir(cfg.basePath, func(path string, d os.DirEntry, err error) error {
		// Check timeout
		select {
		case <-ctx.Done():
			result.timedOut = true
			result.errors = append(result.errors, fmt.Sprintf("%s scanning timeout after %ds", cfg.componentType, cfg.timeoutSeconds))
			return filepath.SkipAll
		default:
		}

		if err != nil {
			result.filesSkipped++
			return nil
		}

		if d.IsDir() {
			return nil
		}

		// Extract component name from path
		relPath, _ := filepath.Rel(cfg.basePath, path)
		pathParts := strings.Split(relPath, string(filepath.Separator))
		if len(pathParts) == 0 {
			return nil
		}
		componentName := pathParts[0]

		// Apply global file limit
		if filesScanned >= cfg.maxFiles {
			result.errors = append(result.errors, fmt.Sprintf("%s file limit reached (%d files)", cfg.componentType, cfg.maxFiles))
			return filepath.SkipAll
		}

		// Apply per-component file limit
		if filesPerComponent[componentName] >= cfg.maxFilesPerComp {
			return nil
		}

		// Check file size
		info, err := d.Info()
		if err == nil && info.Size() > cfg.maxFileSize {
			result.largeSkipped++
			return nil
		}

		// Check file extension
		ext := strings.ToLower(filepath.Ext(path))
		if !isAllowedExtension(ext, cfg.allowedExtensions) {
			return nil
		}

		// Apply component filter
		if cfg.componentFilter != "" && cfg.componentFilter != componentName {
			return nil
		}

		// Track unique components
		if !result.componentsFound[componentName] {
			result.componentsFound[componentName] = true
		}

		// Increment counters
		filesScanned++
		filesPerComponent[componentName]++
		result.filesScanned++

		// Scan file for vulnerabilities
		var fileVulns []SecurityVulnerability
		var scanErr error

		if cfg.componentType == "resource" {
			fileVulns, scanErr = scanResourceFileForVulnerabilities(ctx, path, cfg.componentType, componentName)
		} else {
			fileVulns, scanErr = scanFileForVulnerabilities(path, cfg.componentType, componentName)
		}

		if scanErr != nil {
			logger.Warning("failed to scan %s file %s: %v", cfg.componentType, path, scanErr)
			result.errors = append(result.errors, fmt.Sprintf("Failed to scan %s: %v", filepath.Base(path), scanErr))
			return nil
		}

		// Filter by severity
		for _, vuln := range fileVulns {
			if cfg.severityFilter == "" || vuln.Severity == cfg.severityFilter {
				result.vulnerabilities = append(result.vulnerabilities, vuln)
			}
		}

		return nil
	})

	if err != nil {
		logger.Warning("failed to walk %s directory: %v", cfg.componentType, err)
		result.errors = append(result.errors, fmt.Sprintf("Failed to walk %s directory: %v", cfg.componentType, err))
	}

	result.scanTimeMs = int(time.Since(startTime).Milliseconds())
	return result
}

// isAllowedExtension checks if an extension is in the allowed list.
func isAllowedExtension(ext string, allowed []string) bool {
	for _, a := range allowed {
		if ext == a {
			return true
		}
	}
	return false
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

	paths, err := getVrooliPaths()
	if err != nil {
		scan.Status = "failed"
		scan.ScanMetrics.ScanErrors = append(scan.ScanMetrics.ScanErrors, err.Error())
		return
	}

	// Estimate total files for progress tracking
	estimatedFiles := estimateFileCount(paths.scenarios, paths.resources, componentTypeFilter)
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

	paths, err := getVrooliPaths()
	if err != nil {
		return nil, err
	}

	var vulnerabilities []SecurityVulnerability
	metrics := ScanMetrics{ScanErrors: []string{}}
	var scenariosScanned, resourcesScanned int

	// Scan scenarios if not filtering for resources only
	if componentTypeFilter == "" || componentTypeFilter == "scenario" {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		scenarioResult := walkAndScan(ctx, scanWalkConfig{
			basePath:          paths.scenarios,
			componentType:     "scenario",
			componentFilter:   componentFilter,
			severityFilter:    severityFilter,
			allowedExtensions: []string{".go", ".js", ".ts", ".py", ".sh"},
			maxFileSize:       MaxScenarioWalkFileSize,
			maxFiles:          25000,
			maxFilesPerComp:   200,
			timeoutSeconds:    120,
		})
		cancel()

		vulnerabilities = append(vulnerabilities, scenarioResult.vulnerabilities...)
		scenariosScanned = len(scenarioResult.componentsFound)
		metrics.FilesScanned += scenarioResult.filesScanned
		metrics.FilesSkipped += scenarioResult.filesSkipped
		metrics.LargeFilesSkipped += scenarioResult.largeSkipped
		metrics.ScenarioScanTimeMs = scenarioResult.scanTimeMs
		metrics.ScanErrors = append(metrics.ScanErrors, scenarioResult.errors...)
		if scenarioResult.timedOut {
			metrics.TimeoutOccurred = true
		}
	}

	// Scan resources if not filtering for scenarios only
	if componentTypeFilter == "" || componentTypeFilter == "resource" {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		resourceResult := walkAndScan(ctx, scanWalkConfig{
			basePath:          paths.resources,
			componentType:     "resource",
			componentFilter:   componentFilter,
			severityFilter:    severityFilter,
			allowedExtensions: []string{".sh", ".yaml", ".yml", ".json", ".env"},
			maxFileSize:       MaxResourceWalkFileSize,
			maxFiles:          15000,
			maxFilesPerComp:   100,
			timeoutSeconds:    90,
		})
		cancel()

		vulnerabilities = append(vulnerabilities, resourceResult.vulnerabilities...)
		resourcesScanned = len(resourceResult.componentsFound)
		metrics.FilesScanned += resourceResult.filesScanned
		metrics.FilesSkipped += resourceResult.filesSkipped
		metrics.LargeFilesSkipped += resourceResult.largeSkipped
		metrics.ResourceScanTimeMs = resourceResult.scanTimeMs
		metrics.ScanErrors = append(metrics.ScanErrors, resourceResult.errors...)
		if resourceResult.timedOut {
			metrics.TimeoutOccurred = true
		}
	}

	riskScore := calculateRiskScore(vulnerabilities)
	recommendations := generateRemediationSuggestions(vulnerabilities)

	totalScanTime := int(time.Since(startTime).Milliseconds())
	metrics.TotalScanTimeMs = totalScanTime

	logScanSummary(metrics, len(vulnerabilities))

	completedAt := time.Now()
	if _, err := persistSecurityScan(context.Background(), scanID, componentFilter, componentTypeFilter, severityFilter, metrics, riskScore, vulnerabilities); err != nil {
		logger.Info("failed to persist scan run: %v", err)
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
		},
		ScanMetrics: metrics,
		GeneratedAt: completedAt,
	}, nil
}

// logScanSummary logs the scan results summary.
func logScanSummary(metrics ScanMetrics, vulnCount int) {
	logger.Info("Vulnerability scan completed:")
	logger.Info("  📊 Total scan time: %dms", metrics.TotalScanTimeMs)
	logger.Info("  📁 Files scanned: %d", metrics.FilesScanned)
	logger.Info("  ⏭️  Files skipped: %d", metrics.FilesSkipped)
	logger.Info("  🔍 Vulnerabilities found: %d", vulnCount)
	logger.Info("  ⚠️  Scan errors: %d", len(metrics.ScanErrors))
	if metrics.TimeoutOccurred {
		logger.Info("  ⏰ Timeout occurred during scanning")
	}
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

	// Read file content with size limit
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	if len(content) > MaxResourceFileScanSize {
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
			if IsScenarioSourceExtension(ext) {
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
			if IsResourceConfigExtension(ext) {
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

// persistedScanRun holds data loaded from a security_scan_runs row.
type persistedScanRun struct {
	runID       string
	scanID      string
	riskScore   int
	metrics     ScanMetrics
	completedAt time.Time
}

// loadScanRun loads the most recent scan run matching the filters.
func loadScanRun(ctx context.Context, componentFilter, componentTypeFilter, severityFilter string) (*persistedScanRun, error) {
	if db == nil {
		return nil, nil
	}
	query := `
		SELECT id, scan_id, risk_score, duration_ms, metadata, completed_at
		FROM security_scan_runs
		WHERE component_filter IS NOT DISTINCT FROM $1
		  AND component_type IS NOT DISTINCT FROM $2
		  AND severity_filter IS NOT DISTINCT FROM $3
		ORDER BY completed_at DESC
		LIMIT 1`

	var (
		runID         string
		scanID        string
		riskScore     sql.NullInt64
		durationMs    sql.NullInt64
		metadataBytes []byte
		completedAt   time.Time
	)
	row := db.QueryRowContext(ctx, query, nullString(componentFilter), nullString(componentTypeFilter), nullString(severityFilter))
	if err := row.Scan(&runID, &scanID, &riskScore, &durationMs, &metadataBytes, &completedAt); err != nil {
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

	return &persistedScanRun{
		runID:       runID,
		scanID:      scanID,
		riskScore:   int(riskScore.Int64),
		metrics:     metrics,
		completedAt: completedAt,
	}, nil
}

// vulnerabilitiesWithSummary holds vulnerabilities and component counts.
type vulnerabilitiesWithSummary struct {
	vulnerabilities  []SecurityVulnerability
	resourcesScanned int
	scenariosScanned int
}

// loadVulnerabilitiesForScan loads all vulnerabilities for a scan run.
func loadVulnerabilitiesForScan(ctx context.Context, runID string) (*vulnerabilitiesWithSummary, error) {
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

	var vulnerabilities []SecurityVulnerability
	resourceSet := map[string]struct{}{}
	scenarioSet := map[string]struct{}{}

	for rows.Next() {
		var (
			id, compType, compName, filePath, severityLevel string
			vulnType, title, status, fingerprint            string
			lineNumber                                      sql.NullInt64
			description, recommendation, codeSnippet        sql.NullString
			canAutoFix                                      bool
			firstObserved, lastObserved                     time.Time
		)
		if err := rows.Scan(&id, &compType, &compName, &filePath, &lineNumber, &severityLevel,
			&vulnType, &title, &description, &recommendation, &codeSnippet,
			&canAutoFix, &status, &fingerprint, &firstObserved, &lastObserved); err != nil {
			return nil, err
		}

		// Track unique components
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

	return &vulnerabilitiesWithSummary{
		vulnerabilities:  vulnerabilities,
		resourcesScanned: len(resourceSet),
		scenariosScanned: len(scenarioSet),
	}, nil
}

func loadPersistedSecurityScan(ctx context.Context, componentFilter, componentTypeFilter, severityFilter string) (*SecurityScanResult, error) {
	scanRun, err := loadScanRun(ctx, componentFilter, componentTypeFilter, severityFilter)
	if err != nil || scanRun == nil {
		return nil, err
	}

	vulnData, err := loadVulnerabilitiesForScan(ctx, scanRun.runID)
	if err != nil {
		return nil, err
	}

	return &SecurityScanResult{
		ScanID:          scanRun.scanID,
		ComponentFilter: componentFilter,
		ComponentType:   componentTypeFilter,
		Vulnerabilities: vulnData.vulnerabilities,
		RiskScore:       scanRun.riskScore,
		ScanDurationMs:  scanRun.metrics.TotalScanTimeMs,
		Recommendations: generateRemediationSuggestions(vulnData.vulnerabilities),
		ComponentsSummary: ComponentScanSummary{
			ResourcesScanned: vulnData.resourcesScanned,
			ScenariosScanned: vulnData.scenariosScanned,
			TotalComponents:  vulnData.resourcesScanned + vulnData.scenariosScanned,
		},
		ScanMetrics: scanRun.metrics,
		GeneratedAt: scanRun.completedAt,
	}, nil
}
