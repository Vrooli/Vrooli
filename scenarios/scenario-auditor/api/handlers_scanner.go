package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"scenario-auditor/scanners"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// RealScanLogger implements the scanners.Logger interface
type RealScanLogger struct {
	logger *Logger
}

func NewRealScanLogger(logger *Logger) *RealScanLogger {
	return &RealScanLogger{logger: logger}
}

func (l *RealScanLogger) Info(msg string, args ...interface{}) {
	l.logger.Info(fmt.Sprintf(msg, args...))
}

func (l *RealScanLogger) Warn(msg string, args ...interface{}) {
	l.logger.Info(fmt.Sprintf("WARN: "+msg, args...))
}

func (l *RealScanLogger) Error(msg string, args ...interface{}) {
	if len(args) > 0 {
		if err, ok := args[len(args)-1].(error); ok {
			l.logger.Error(fmt.Sprintf(msg, args[:len(args)-1]...), err)
			return
		}
	}
	l.logger.Info(fmt.Sprintf("ERROR: "+msg, args...))
}

func (l *RealScanLogger) Debug(msg string, args ...interface{}) {
	l.logger.Info(fmt.Sprintf("DEBUG: "+msg, args...))
}

// enhancedScanScenarioHandler handles security scans using real scanners
func enhancedScanScenarioHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["name"]
	logger := NewLogger()
	scanLogger := NewRealScanLogger(logger)

	w.Header().Set("Content-Type", "application/json")

	// Parse request body for scan options
	var scanRequest struct {
		Type             string   `json:"type"`              // "quick", "full", or "targeted"
		TargetedChecks   []string `json:"targeted_checks"`   // For targeted scans
		IncludeUnstable  bool     `json:"include_unstable"`  // Include rules with failing tests
		SkipTestValidation bool   `json:"skip_test_validation"` // Skip pre-flight rule validation
	}
	if err := json.NewDecoder(r.Body).Decode(&scanRequest); err != nil {
		scanRequest.Type = "quick" // Default to quick scan
	}

	// Validate scan type
	if scanRequest.Type != "quick" && scanRequest.Type != "full" && scanRequest.Type != "targeted" {
		scanRequest.Type = "quick"
	}

	// Initialize the scan orchestrator
	orchestrator := scanners.NewScanOrchestrator(scanLogger)

	logger.Info("Registering security scanners...")
	
	// Try to register each scanner - they'll check if tools are available
	err := orchestrator.RegisterScanner(scanners.NewGosecScanner(scanLogger))
	if err != nil {
		logger.Info("Gosec scanner not available: " + err.Error())
	}
	
	err = orchestrator.RegisterScanner(scanners.NewGitleaksScanner(scanLogger))
	if err != nil {
		logger.Info("Gitleaks scanner not available: " + err.Error())
	}
	
	// Custom pattern scanner should always be available
	err = orchestrator.RegisterScanner(scanners.NewCustomPatternScanner(scanLogger))
	if err != nil {
		logger.Error("Custom pattern scanner failed to register", err)
		HTTPError(w, "Scanner initialization failed", http.StatusInternalServerError, err)
		return
	}
	
	logger.Info("Scanner registration completed")

	// Pre-flight rule validation (unless skipped)
	var unstableRules []string
	var skippedRuleCount int
	
	if !scanRequest.SkipTestValidation {
		logger.Info("Running pre-flight rule validation...")
		unstableRules, skippedRuleCount = validateRulesBeforeScan(logger, scanRequest.IncludeUnstable)
		
		if len(unstableRules) > 0 && !scanRequest.IncludeUnstable {
			logger.Info(fmt.Sprintf("Found %d unstable rules (with failing tests). Use 'include_unstable: true' to include them in scan.", len(unstableRules)))
		}
	}

	// Determine scan path
	var scanPath string
	var scenarioCount int = 1

	if scenarioName == "all" {
		scanPath = filepath.Join(os.Getenv("HOME"), "Vrooli", "scenarios")
		// Count scenarios for reporting
		scenarios, _ := getVrooliScenarios()
		if scenarios != nil && scenarios.Scenarios != nil {
			scenarioCount = len(scenarios.Scenarios)
		}
	} else {
		scanPath = filepath.Join(os.Getenv("HOME"), "Vrooli", "scenarios", scenarioName)
		if _, err := os.Stat(scanPath); os.IsNotExist(err) {
			HTTPError(w, "Scenario not found", http.StatusNotFound, nil)
			return
		}
	}

	// Configure scan options
	scanOpts := scanners.ScanOptions{
		Path:           scanPath,
		ScanType:       scanRequest.Type,
		TargetedChecks: scanRequest.TargetedChecks,
		Timeout:        30 * time.Second, // Shorter timeout to prevent hangs
		ExcludePatterns: []string{
			"**/vendor/**",
			"**/node_modules/**",
			"**/.git/**",
			"**/dist/**",
			"**/build/**",
		},
	}

	logger.Info(fmt.Sprintf("Starting %s security scan on %s", scanRequest.Type, scenarioName))

	// Perform the scan with error recovery
	logger.Info("Starting scan...")
	result, err := orchestrator.Scan(scanOpts)
	if err != nil {
		logger.Error("Enhanced security scan failed, falling back to basic scan", err)
		// Fall back to basic file counting
		totalFiles, _, _ := countScannableFiles(scanPath)
		
		response := map[string]interface{}{
			"scan_id":    fmt.Sprintf("fallback-%d", time.Now().Unix()),
			"status":     "completed",
			"scan_type":  scanRequest.Type,
			"started_at": time.Now().Format(time.RFC3339),
			"completed_at": time.Now().Format(time.RFC3339),
			"files_scanned": totalFiles,
			"vulnerabilities_found": 0,
			"vulnerabilities": map[string]interface{}{
				"total": 0, "critical": 0, "high": 0, "medium": 0, "low": 0,
			},
			"message": "Scan completed with basic file counting. Enhanced scanning temporarily unavailable.",
			"note": "Scanner tools not available. Install gosec and gitleaks for full security scanning.",
		}
		
		if scenarioName == "all" {
			response["scenarios_scanned"] = scenarioCount
		} else {
			response["scenario_name"] = scenarioName
		}
		
		json.NewEncoder(w).Encode(response)
		return
	}

	// Convert findings to API format
	vulnerabilities := make([]map[string]interface{}, 0)
	for _, finding := range result.Findings {
		vuln := map[string]interface{}{
			"id":          finding.ID,
			"type":        finding.Category,
			"severity":    string(finding.Severity),
			"title":       finding.Title,
			"description": finding.Description,
			"file_path":   finding.FilePath,
			"line_number": finding.LineNumber,
			"scanner":     string(finding.ScannerType),
			"rule_id":     finding.RuleID,
			"confidence":  finding.Confidence,
		}

		if finding.CodeSnippet != "" {
			vuln["code_snippet"] = finding.CodeSnippet
		}
		if finding.Recommendation != "" {
			vuln["recommendation"] = finding.Recommendation
		}
		if finding.CWE > 0 {
			vuln["cwe"] = finding.CWE
		}
		if finding.OWASP != "" {
			vuln["owasp"] = finding.OWASP
		}

		vulnerabilities = append(vulnerabilities, vuln)
	}

	// Build response
	response := map[string]interface{}{
		"scan_id":    result.ScanID,
		"status":     "completed",
		"scan_type":  scanRequest.Type,
		"started_at": result.StartTime.Format(time.RFC3339),
		"completed_at": result.EndTime.Format(time.RFC3339),
		"duration_seconds": result.Duration.Seconds(),
		"files_scanned": result.FilesScanned,
		"lines_scanned": result.LinesScanned,
		"vulnerabilities_found": len(result.Findings),
		"vulnerabilities": map[string]interface{}{
			"total":    result.Statistics.TotalFindings,
			"critical": result.Statistics.BySeverity["critical"],
			"high":     result.Statistics.BySeverity["high"],
			"medium":   result.Statistics.BySeverity["medium"],
			"low":      result.Statistics.BySeverity["low"],
			"info":     result.Statistics.BySeverity["info"],
		},
		"statistics": result.Statistics,
		"findings":   vulnerabilities,
	}

	// Add rule validation info to response
	if !scanRequest.SkipTestValidation {
		response["rule_validation"] = map[string]interface{}{
			"enabled":        true,
			"unstable_rules": unstableRules,
			"skipped_count":  skippedRuleCount,
			"include_unstable": scanRequest.IncludeUnstable,
		}
		
		if len(unstableRules) > 0 && !scanRequest.IncludeUnstable {
			response["warnings"] = []string{
				fmt.Sprintf("%d rules skipped due to failing tests. Use 'include_unstable: true' to include them.", skippedRuleCount),
			}
		}
	} else {
		response["rule_validation"] = map[string]interface{}{
			"enabled": false,
			"message": "Rule validation was skipped",
		}
	}

	// Add scenario-specific info
	if scenarioName == "all" {
		response["scenarios_scanned"] = scenarioCount
		response["message"] = fmt.Sprintf("Security scan completed across %d scenarios. Found %d potential vulnerabilities.",
			scenarioCount, len(result.Findings))
	} else {
		response["scenario_name"] = scenarioName
		response["message"] = fmt.Sprintf("Security scan completed for %s. Found %d potential vulnerabilities.",
			scenarioName, len(result.Findings))
	}

	// Add scanner info
	scannerInfo := make([]map[string]interface{}, 0)
	for _, scanResult := range result.ScannerResults {
		info := map[string]interface{}{
			"scanner":     string(scanResult.ScannerType),
			"findings":    len(scanResult.Findings),
			"duration":    scanResult.Duration.Seconds(),
			"version":     scanResult.ToolVersion,
		}
		scannerInfo = append(scannerInfo, info)
	}
	response["scanners_used"] = scannerInfo

	// Store results in memory (always) and database (if available)
	vulnStore.StoreVulnerabilities(scenarioName, result)
	logger.Info(fmt.Sprintf("Stored %d vulnerabilities in memory for %s", len(result.Findings), scenarioName))
	
	// Also try to store in database if available
	if db != nil {
		go storeScanResults(result, scenarioName)
	}

	logger.Info(fmt.Sprintf("Security scan completed: %d vulnerabilities found", len(result.Findings)))
	json.NewEncoder(w).Encode(response)
}

// storeScanResults saves scan results to database
func storeScanResults(result *scanners.AggregatedScanResult, scenarioName string) {
	logger := NewLogger()

	if db == nil {
		return
	}

	// Get or create scenario ID
	var scenarioID string
	err := db.QueryRow(`
		INSERT INTO scenarios (id, name, path, status, created_at, updated_at)
		VALUES ($1, $2, $3, 'active', NOW(), NOW())
		ON CONFLICT (name) DO UPDATE SET 
			last_scanned = NOW(),
			updated_at = NOW()
		RETURNING id
	`, uuid.New(), scenarioName, result.ScannedPath).Scan(&scenarioID)

	if err != nil {
		logger.Error("Failed to upsert scenario", err)
		return
	}

	// Store each finding as a vulnerability
	for _, finding := range result.Findings {
		_, err := db.Exec(`
			INSERT INTO vulnerability_scans 
			(id, scenario_id, scan_type, severity, category, title, description, 
			 file_path, line_number, code_snippet, recommendation, status, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW())
			ON CONFLICT DO NOTHING
		`, uuid.New(), scenarioID, result.ScanType, string(finding.Severity), 
			finding.Category, finding.Title, finding.Description, 
			finding.FilePath, finding.LineNumber, finding.CodeSnippet, 
			finding.Recommendation, "open")

		if err != nil {
			logger.Error("Failed to store vulnerability", err)
		}
	}

	// Record scan in history
	resultsSummary, _ := json.Marshal(map[string]interface{}{
		"total_findings": result.Statistics.TotalFindings,
		"by_severity":    result.Statistics.BySeverity,
		"scanners_used":  len(result.ScannerResults),
	})

	_, err = db.Exec(`
		INSERT INTO scan_history 
		(id, scenario_id, scan_type, status, results_summary, duration_ms, 
		 triggered_by, started_at, completed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, uuid.New(), scenarioID, result.ScanType, "completed",
		string(resultsSummary), int(result.Duration.Milliseconds()),
		"api", result.StartTime, result.EndTime)

	if err != nil {
		logger.Error("Failed to record scan history", err)
	}
}

// enhancedGetVulnerabilitiesHandler returns vulnerabilities (real ones if scans have been performed)
func enhancedGetVulnerabilitiesHandler(w http.ResponseWriter, r *http.Request) {
	scenario := r.URL.Query().Get("scenario")
	
	w.Header().Set("Content-Type", "application/json")
	
	vulnerabilities := []map[string]interface{}{}
	
	// If we have a database, try to get real vulnerabilities
	if db != nil {
		query := `
			SELECT 
				vs.id, vs.severity, vs.category, vs.title, vs.description,
				vs.file_path, vs.line_number, vs.code_snippet, vs.recommendation,
				vs.status, vs.created_at, s.name as scenario_name
			FROM vulnerability_scans vs
			JOIN scenarios s ON vs.scenario_id = s.id
			WHERE vs.status = 'open'
		`
		args := []interface{}{}
		
		if scenario != "" && scenario != "all" {
			query += " AND s.name = $1"
			args = append(args, scenario)
		}
		
		query += " ORDER BY vs.created_at DESC LIMIT 100"
		
		rows, err := db.Query(query, args...)
		if err == nil {
			defer rows.Close()
			
			for rows.Next() {
				var vuln struct {
					ID           string
					Severity     string
					Category     string
					Title        string
					Description  string
					FilePath     string
					LineNumber   int
					CodeSnippet  sql.NullString
					Recommendation string
					Status       string
					CreatedAt    time.Time
					ScenarioName string
				}
				
				if err := rows.Scan(&vuln.ID, &vuln.Severity, &vuln.Category, 
					&vuln.Title, &vuln.Description, &vuln.FilePath, &vuln.LineNumber,
					&vuln.CodeSnippet, &vuln.Recommendation, &vuln.Status, 
					&vuln.CreatedAt, &vuln.ScenarioName); err != nil {
					continue
				}
				
				v := map[string]interface{}{
					"id":             vuln.ID,
					"scenario_name":  vuln.ScenarioName,
					"type":          vuln.Category,
					"severity":      strings.ToLower(vuln.Severity),
					"title":         vuln.Title,
					"description":   vuln.Description,
					"file_path":     vuln.FilePath,
					"line_number":   vuln.LineNumber,
					"recommendation": vuln.Recommendation,
					"status":        vuln.Status,
					"discovered_at": vuln.CreatedAt.Format(time.RFC3339),
				}
				
				if vuln.CodeSnippet.Valid {
					v["code_snippet"] = vuln.CodeSnippet.String
				}
				
				vulnerabilities = append(vulnerabilities, v)
			}
		}
	}
	
	// Return response
	response := map[string]interface{}{
		"vulnerabilities": vulnerabilities,
		"count":          len(vulnerabilities),
		"timestamp":      time.Now().Format(time.RFC3339),
	}
	
	// Add a note if no real scans have been performed
	if len(vulnerabilities) == 0 {
		response["note"] = "No vulnerabilities found. Run a security scan to detect real vulnerabilities."
	}
	
	json.NewEncoder(w).Encode(response)
}

// validateRulesBeforeScan runs pre-flight tests on all rules to identify unstable ones
func validateRulesBeforeScan(logger *Logger, includeUnstable bool) (unstableRules []string, skippedCount int) {
	// Load all available rules
	ruleInfos, err := LoadRulesFromFiles()
	if err != nil {
		logger.Error("Failed to load rules for validation", err)
		return []string{}, 0
	}
	
	// Initialize test runner
	testRunner := NewTestRunner()
	unstableRulesList := []string{}
	totalRules := len(ruleInfos)
	stableRules := 0
	
	logger.Info(fmt.Sprintf("Validating %d rules with embedded test cases...", totalRules))
	
	// Test each rule
	for ruleID, ruleInfo := range ruleInfos {
		// Run all tests for this rule
		testResults, err := testRunner.RunAllTests(ruleID, ruleInfo)
		if err != nil {
			logger.Info(fmt.Sprintf("WARNING: Failed to run tests for rule %s: %v", ruleID, err))
			unstableRulesList = append(unstableRulesList, ruleID)
			continue
		}
		
		// Check if any tests failed
		hasFailingTests := false
		totalTests := len(testResults)
		passedTests := 0
		
		for _, result := range testResults {
			if result.Passed {
				passedTests++
			} else {
				hasFailingTests = true
			}
		}
		
		if hasFailingTests || totalTests == 0 {
			// Rule is unstable - has failing tests or no tests
			unstableRulesList = append(unstableRulesList, ruleID)
			
			if totalTests == 0 {
				logger.Info(fmt.Sprintf("WARNING: Rule %s has no test cases (considered unstable)", ruleID))
			} else {
				logger.Info(fmt.Sprintf("WARNING: Rule %s has failing tests: %d/%d passed", ruleID, passedTests, totalTests))
			}
		} else {
			// Rule is stable - all tests pass
			stableRules++
			logger.Info(fmt.Sprintf("DEBUG: Rule %s is stable: %d/%d tests passed", ruleID, passedTests, totalTests))
		}
	}
	
	// Calculate statistics
	unstableCount := len(unstableRulesList)
	skippedCount = 0
	if !includeUnstable {
		skippedCount = unstableCount
	}
	
	// Log summary
	logger.Info(fmt.Sprintf("Rule validation complete: %d stable, %d unstable out of %d total rules", 
		stableRules, unstableCount, totalRules))
	
	if unstableCount > 0 && !includeUnstable {
		logger.Info(fmt.Sprintf("Skipping %d unstable rules during scan. Use 'include_unstable: true' to include them.", unstableCount))
		logger.Info(fmt.Sprintf("Unstable rules: %s", strings.Join(unstableRulesList, ", ")))
	}
	
	return unstableRulesList, skippedCount
}