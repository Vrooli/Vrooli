package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"scenario-auditor/scanners"
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

var (
	errSecurityScanCancelled = errors.New("security scan cancelled")
	errSecurityScanNotFound  = errors.New("security scan not found")
	errSecurityScanFinished  = errors.New("security scan already finished")
)

type securityScanRequest struct {
	Type               string
	TargetedChecks     []string
	IncludeUnstable    bool
	SkipTestValidation bool
}

type securityScanTarget struct {
	Name string
	Path string
}

type SecurityScanResult struct {
	ScanID               string                   `json:"scan_id"`
	Status               string                   `json:"status"`
	ScanType             string                   `json:"scan_type"`
	StartedAt            string                   `json:"started_at"`
	CompletedAt          string                   `json:"completed_at"`
	DurationSeconds      float64                  `json:"duration_seconds"`
	FilesScanned         int                      `json:"files_scanned"`
	LinesScanned         int                      `json:"lines_scanned"`
	VulnerabilitiesFound int                      `json:"vulnerabilities_found"`
	Vulnerabilities      map[string]int           `json:"vulnerabilities"`
	Statistics           scanners.ScanStatistics  `json:"statistics"`
	Findings             []map[string]interface{} `json:"findings"`
	ScenarioName         string                   `json:"scenario_name,omitempty"`
	ScenariosScanned     int                      `json:"scenarios_scanned,omitempty"`
	Message              string                   `json:"message"`
	RuleValidation       map[string]interface{}   `json:"rule_validation,omitempty"`
	Warnings             []string                 `json:"warnings,omitempty"`
	ScannersUsed         []map[string]interface{} `json:"scanners_used"`
	ScanNotes            string                   `json:"scan_notes,omitempty"`
}

type SecurityScanStatus struct {
	ID                 string              `json:"id"`
	Scenario           string              `json:"scenario"`
	ScanType           string              `json:"scan_type"`
	Status             string              `json:"status"`
	StartedAt          time.Time           `json:"started_at"`
	CompletedAt        *time.Time          `json:"completed_at,omitempty"`
	ElapsedSeconds     float64             `json:"elapsed_seconds"`
	TotalScenarios     int                 `json:"total_scenarios"`
	ProcessedScenarios int                 `json:"processed_scenarios"`
	ProcessedFiles     int                 `json:"processed_files"`
	TotalFiles         int                 `json:"total_files"`
	CurrentScenario    string              `json:"current_scenario,omitempty"`
	CurrentScanner     string              `json:"current_scanner,omitempty"`
	Message            string              `json:"message,omitempty"`
	Error              string              `json:"error,omitempty"`
	Result             *SecurityScanResult `json:"result,omitempty"`
}

type SecurityScanJob struct {
	mu     sync.RWMutex
	status SecurityScanStatus
	cancel context.CancelFunc
}

type SecurityScanManager struct {
	mu   sync.RWMutex
	jobs map[string]*SecurityScanJob
}

var securityScanManager = newSecurityScanManager()

func newSecurityScanManager() *SecurityScanManager {
	return &SecurityScanManager{
		jobs: make(map[string]*SecurityScanJob),
	}
}

func (m *SecurityScanManager) StartScan(scenarioName string, req securityScanRequest) (SecurityScanStatus, error) {
	targets, err := buildSecurityScanTargets(scenarioName)
	if err != nil {
		return SecurityScanStatus{}, err
	}
	if len(targets) == 0 {
		return SecurityScanStatus{}, fmt.Errorf("no scenarios available to scan")
	}

	ctx, cancel := context.WithCancel(context.Background())
	jobID := fmt.Sprintf("security-%s", uuid.NewString())
	start := time.Now()

	job := &SecurityScanJob{
		cancel: cancel,
		status: SecurityScanStatus{
			ID:             jobID,
			Scenario:       scenarioName,
			ScanType:       req.Type,
			Status:         "running",
			StartedAt:      start,
			TotalScenarios: len(targets),
			Message:        "Security scan started",
		},
	}

	m.mu.Lock()
	m.jobs[jobID] = job
	m.mu.Unlock()

	go job.run(ctx, targets, scenarioName, req)

	return job.snapshot(), nil
}

func (m *SecurityScanManager) Get(jobID string) (*SecurityScanJob, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	job, ok := m.jobs[jobID]
	return job, ok
}

func (m *SecurityScanManager) Cancel(jobID string) (SecurityScanStatus, error) {
	job, ok := m.Get(jobID)
	if !ok {
		return SecurityScanStatus{}, errSecurityScanNotFound
	}

	err := job.requestCancel()
	return job.snapshot(), err
}

func (job *SecurityScanJob) update(fn func(*SecurityScanStatus)) {
	job.mu.Lock()
	defer job.mu.Unlock()
	fn(&job.status)
	if !job.status.StartedAt.IsZero() {
		job.status.ElapsedSeconds = time.Since(job.status.StartedAt).Seconds()
	}
}

func (job *SecurityScanJob) snapshot() SecurityScanStatus {
	job.mu.RLock()
	defer job.mu.RUnlock()
	copy := job.status
	if !copy.StartedAt.IsZero() && copy.Status != "pending" {
		copy.ElapsedSeconds = time.Since(copy.StartedAt).Seconds()
	}
	return copy
}

func (job *SecurityScanJob) requestCancel() error {
	job.mu.Lock()
	status := job.status.Status
	if status == "completed" || status == "failed" || status == "cancelled" {
		job.mu.Unlock()
		return errSecurityScanFinished
	}
	if status == "cancelling" {
		job.mu.Unlock()
		return nil
	}
	job.status.Status = "cancelling"
	job.status.Message = "Cancellation requested"
	if !job.status.StartedAt.IsZero() {
		job.status.ElapsedSeconds = time.Since(job.status.StartedAt).Seconds()
	}
	cancel := job.cancel
	job.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	return nil
}

func (job *SecurityScanJob) markCancelled() {
	completedAt := time.Now()
	job.update(func(status *SecurityScanStatus) {
		status.Status = "cancelled"
		status.CompletedAt = &completedAt
		status.Message = "Security scan cancelled"
		status.CurrentScenario = ""
		status.CurrentScanner = ""
	})
}

func (job *SecurityScanJob) markFailed(err error) {
	completedAt := time.Now()
	job.update(func(status *SecurityScanStatus) {
		status.Status = "failed"
		status.CompletedAt = &completedAt
		status.Error = err.Error()
		status.Message = "Security scan failed"
		status.CurrentScenario = ""
		status.CurrentScanner = ""
	})
}

func (job *SecurityScanJob) markCompleted(result SecurityScanResult, processedFiles int) {
	completedAt := time.Now()
	job.update(func(status *SecurityScanStatus) {
		status.Status = "completed"
		status.CompletedAt = &completedAt
		status.Message = result.Message
		status.Result = &result
		status.ProcessedFiles = processedFiles
		status.TotalFiles = processedFiles
		status.CurrentScenario = ""
		status.CurrentScanner = ""
	})
}

func (job *SecurityScanJob) handleScannerProgress(scenarioName string, scannerType scanners.ScannerType, result *scanners.ScanResult) {
	job.update(func(status *SecurityScanStatus) {
		status.CurrentScenario = scenarioName
		status.CurrentScanner = string(scannerType)
		if result != nil {
			status.ProcessedFiles += result.FilesScanned
		}
	})
}

func (job *SecurityScanJob) run(ctx context.Context, targets []securityScanTarget, scenarioName string, req securityScanRequest) {
	logger := NewLogger()
	scanLogger := NewRealScanLogger(logger)
	orchestrator := scanners.NewScanOrchestrator(scanLogger)

	logger.Info("Registering security scanners...")

	if err := orchestrator.RegisterScanner(scanners.NewGosecScanner(scanLogger)); err != nil {
		logger.Info("Gosec scanner not available: " + err.Error())
	}

	if err := orchestrator.RegisterScanner(scanners.NewGitleaksScanner(scanLogger)); err != nil {
		logger.Info("Gitleaks scanner not available: " + err.Error())
	}

	if err := orchestrator.RegisterScanner(scanners.NewCustomPatternScanner(scanLogger)); err != nil {
		logger.Error("Custom pattern scanner failed to register", err)
		job.markFailed(fmt.Errorf("failed to initialize required scanner: %w", err))
		return
	}

	logger.Info("Scanner registration completed")

	var unstableRules []string
	var skippedRuleCount int
	if !req.SkipTestValidation {
		logger.Info("Running pre-flight rule validation...")
		unstableRules, skippedRuleCount = validateRulesBeforeScan(logger, req.IncludeUnstable)
		if len(unstableRules) > 0 && !req.IncludeUnstable {
			logger.Info(fmt.Sprintf("Found %d unstable rules (with failing tests). Use 'include_unstable: true' to include them in scan.", len(unstableRules)))
		}
	}

	initialStatus := job.snapshot()
	start := initialStatus.StartedAt
	jobID := initialStatus.ID
	severityTotals := map[string]int{
		"critical": 0,
		"high":     0,
		"medium":   0,
		"low":      0,
		"info":     0,
	}
	categoryTotals := make(map[string]int)
	scannerTotals := make(map[string]int)
	fileStatsMap := make(map[string]scanners.FileStats)
	var scannersUsed []map[string]interface{}
	var allFindings []scanners.Finding
	var warnings []string

	totalFiles := 0
	totalLines := 0
	totalFindings := 0

	for _, target := range targets {
		select {
		case <-ctx.Done():
			logger.Info(fmt.Sprintf("Security scan cancelled before completing scenario %s", target.Name))
			job.markCancelled()
			return
		default:
		}

		job.update(func(status *SecurityScanStatus) {
			status.CurrentScenario = target.Name
			status.Message = fmt.Sprintf("Scanning %s", target.Name)
		})

		opts := scanners.ScanOptions{
			Path:           target.Path,
			ScanType:       req.Type,
			TargetedChecks: req.TargetedChecks,
			Context:        ctx,
			Progress: func(scannerType scanners.ScannerType, result *scanners.ScanResult) {
				job.handleScannerProgress(target.Name, scannerType, result)
			},
		}

		aggregatedResult, err := orchestrator.Scan(opts)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				logger.Info("Security scan cancelled during scanner execution")
				job.markCancelled()
				return
			}
			logger.Error("Security scan failed", err)
			job.markFailed(err)
			return
		}

		totalFiles += aggregatedResult.FilesScanned
		totalLines += aggregatedResult.LinesScanned
		totalFindings += len(aggregatedResult.Findings)
		allFindings = append(allFindings, aggregatedResult.Findings...)

		for severity, count := range aggregatedResult.Statistics.BySeverity {
			severityTotals[severity] += count
		}
		for category, count := range aggregatedResult.Statistics.ByCategory {
			categoryTotals[category] += count
		}
		for scannerName, count := range aggregatedResult.Statistics.ByScanner {
			scannerTotals[scannerName] += count
		}
		for _, fs := range aggregatedResult.Statistics.TopVulnerableFiles {
			existing := fileStatsMap[fs.FilePath]
			existing.FilePath = fs.FilePath
			existing.FindingCount += fs.FindingCount
			existing.CriticalCount += fs.CriticalCount
			existing.HighCount += fs.HighCount
			fileStatsMap[fs.FilePath] = existing
		}
		for _, scanResult := range aggregatedResult.ScannerResults {
			scannersUsed = append(scannersUsed, map[string]interface{}{
				"scanner":  string(scanResult.ScannerType),
				"findings": len(scanResult.Findings),
				"duration": scanResult.Duration.Seconds(),
				"version":  scanResult.ToolVersion,
			})
		}

		vulnStore.StoreVulnerabilities(target.Name, aggregatedResult)
		logger.Info(fmt.Sprintf("Stored %d vulnerabilities in memory for %s", len(aggregatedResult.Findings), target.Name))
		if db != nil {
			go storeScanResults(aggregatedResult, target.Name)
		}

		job.update(func(status *SecurityScanStatus) {
			status.ProcessedScenarios++
			status.CurrentScanner = ""
		})
	}

	if len(unstableRules) > 0 && !req.IncludeUnstable {
		warnings = append(warnings, fmt.Sprintf("%d rules skipped due to failing tests. Use 'include_unstable: true' to include them.", skippedRuleCount))
	}

	combinedStats := scanners.ScanStatistics{
		TotalFindings:      totalFindings,
		BySeverity:         make(map[string]int),
		ByCategory:         make(map[string]int),
		ByScanner:          make(map[string]int),
		TopVulnerableFiles: []scanners.FileStats{},
	}
	for severity, count := range severityTotals {
		combinedStats.BySeverity[severity] = count
	}
	for category, count := range categoryTotals {
		combinedStats.ByCategory[category] = count
	}
	for scannerName, count := range scannerTotals {
		combinedStats.ByScanner[scannerName] = count
	}
	fileStats := make([]scanners.FileStats, 0, len(fileStatsMap))
	for _, fs := range fileStatsMap {
		fileStats = append(fileStats, fs)
	}
	sort.Slice(fileStats, func(i, j int) bool {
		return fileStats[i].FindingCount > fileStats[j].FindingCount
	})
	if len(fileStats) > 10 {
		fileStats = fileStats[:10]
	}
	combinedStats.TopVulnerableFiles = fileStats

	end := time.Now()

	vulnerabilityMap := map[string]int{
		"total":    totalFindings,
		"critical": severityTotals["critical"],
		"high":     severityTotals["high"],
		"medium":   severityTotals["medium"],
		"low":      severityTotals["low"],
		"info":     severityTotals["info"],
	}

	ruleValidation := map[string]interface{}{
		"enabled": !req.SkipTestValidation,
	}
	if !req.SkipTestValidation {
		ruleValidation["unstable_rules"] = unstableRules
		ruleValidation["skipped_count"] = skippedRuleCount
		ruleValidation["include_unstable"] = req.IncludeUnstable
	}

	result := SecurityScanResult{
		ScanID:               jobID,
		Status:               "completed",
		ScanType:             req.Type,
		StartedAt:            start.Format(time.RFC3339),
		CompletedAt:          end.Format(time.RFC3339),
		DurationSeconds:      end.Sub(start).Seconds(),
		FilesScanned:         totalFiles,
		LinesScanned:         totalLines,
		VulnerabilitiesFound: totalFindings,
		Vulnerabilities:      vulnerabilityMap,
		Statistics:           combinedStats,
		Findings:             convertScanFindingsToResponse(allFindings),
		Message:              buildSecurityScanCompletionMessage(scenarioName, len(targets), totalFindings),
		RuleValidation:       ruleValidation,
		Warnings:             warnings,
		ScannersUsed:         scannersUsed,
	}

	if !strings.EqualFold(scenarioName, "all") {
		result.ScenarioName = scenarioName
	} else {
		result.ScenariosScanned = len(targets)
	}

	job.markCompleted(result, totalFiles)
	logger.Info(fmt.Sprintf("Security scan %s completed", jobID))
}

// enhancedScanScenarioHandler handles security scans using real scanners
func enhancedScanScenarioHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := strings.TrimSpace(vars["name"])
	if scenarioName == "" {
		scenarioName = "all"
	} else if strings.EqualFold(scenarioName, "all") {
		scenarioName = "all"
	}

	logger := NewLogger()

	requestBody := struct {
		Type               string   `json:"type"`
		TargetedChecks     []string `json:"targeted_checks"`
		IncludeUnstable    bool     `json:"include_unstable"`
		SkipTestValidation bool     `json:"skip_test_validation"`
	}{Type: "quick"}

	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil && err.Error() != "EOF" {
		logger.Info("Security scan request body could not be parsed; defaulting to quick scan")
	}

	scanType := strings.ToLower(strings.TrimSpace(requestBody.Type))
	if scanType != "quick" && scanType != "full" && scanType != "targeted" {
		scanType = "quick"
	}

	if scanType == "targeted" && strings.EqualFold(scenarioName, "all") {
		HTTPError(w, "Targeted scans require a specific scenario", http.StatusBadRequest, nil)
		return
	}

	req := securityScanRequest{
		Type:               scanType,
		TargetedChecks:     requestBody.TargetedChecks,
		IncludeUnstable:    requestBody.IncludeUnstable,
		SkipTestValidation: requestBody.SkipTestValidation,
	}

	status, err := securityScanManager.StartScan(scenarioName, req)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) || strings.Contains(strings.ToLower(err.Error()), "not found") {
			HTTPError(w, "Scenario not found", http.StatusNotFound, err)
			return
		}
		HTTPError(w, "Failed to start security scan", http.StatusInternalServerError, err)
		return
	}

	logger.Info(fmt.Sprintf("Started %s security scan %s for %s", scanType, status.ID, scenarioName))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"job_id": status.ID,
		"status": status,
	})
}

func buildSecurityScanTargets(scenarioName string) ([]securityScanTarget, error) {
	standardsTargets, err := buildStandardsScanTargets(scenarioName)
	if err != nil {
		return nil, err
	}
	targets := make([]securityScanTarget, len(standardsTargets))
	for i, t := range standardsTargets {
		targets[i] = securityScanTarget{Name: t.Name, Path: t.Path}
	}
	return targets, nil
}

func buildSecurityScanCompletionMessage(scenarioName string, scenarioCount, findings int) string {
	if strings.EqualFold(scenarioName, "all") {
		return fmt.Sprintf("Security scan completed across %d scenarios. Found %d potential vulnerabilities.", scenarioCount, findings)
	}
	return fmt.Sprintf("Security scan completed for %s. Found %d potential vulnerabilities.", scenarioName, findings)
}

func convertScanFindingsToResponse(findings []scanners.Finding) []map[string]interface{} {
	vulnerabilities := make([]map[string]interface{}, 0, len(findings))
	for _, finding := range findings {
		vuln := map[string]interface{}{
			"id":            finding.ID,
			"type":          finding.Category,
			"severity":      strings.ToLower(string(finding.Severity)),
			"title":         finding.Title,
			"description":   finding.Description,
			"file_path":     finding.FilePath,
			"line_number":   finding.LineNumber,
			"scanner":       string(finding.ScannerType),
			"rule_id":       finding.RuleID,
			"confidence":    finding.Confidence,
			"scenario_name": extractScenarioName(finding.FilePath),
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
		if len(finding.References) > 0 {
			vuln["references"] = finding.References
		}
		if len(finding.Metadata) > 0 {
			vuln["metadata"] = finding.Metadata
		}

		vulnerabilities = append(vulnerabilities, vuln)
	}
	return vulnerabilities
}

func getSecurityScanStatusHandler(w http.ResponseWriter, r *http.Request) {
	jobID := strings.TrimSpace(mux.Vars(r)["jobId"])
	if jobID == "" {
		HTTPError(w, "Job ID is required", http.StatusBadRequest, nil)
		return
	}

	job, ok := securityScanManager.Get(jobID)
	if !ok {
		HTTPError(w, "Security scan not found", http.StatusNotFound, errSecurityScanNotFound)
		return
	}

	status := job.snapshot()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func cancelSecurityScanHandler(w http.ResponseWriter, r *http.Request) {
	jobID := strings.TrimSpace(mux.Vars(r)["jobId"])
	if jobID == "" {
		HTTPError(w, "Job ID is required", http.StatusBadRequest, nil)
		return
	}

	status, err := securityScanManager.Cancel(jobID)
	if err != nil {
		if errors.Is(err, errSecurityScanNotFound) {
			HTTPError(w, "Security scan not found", http.StatusNotFound, err)
			return
		}
		if errors.Is(err, errSecurityScanFinished) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"status":  status,
				"message": fmt.Sprintf("Scan already %s", status.Status),
			})
			return
		}
		HTTPError(w, "Failed to cancel security scan", http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"status":  status,
	})
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
					ID             string
					Severity       string
					Category       string
					Title          string
					Description    string
					FilePath       string
					LineNumber     int
					CodeSnippet    sql.NullString
					Recommendation string
					Status         string
					CreatedAt      time.Time
					ScenarioName   string
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
					"type":           vuln.Category,
					"severity":       strings.ToLower(vuln.Severity),
					"title":          vuln.Title,
					"description":    vuln.Description,
					"file_path":      vuln.FilePath,
					"line_number":    vuln.LineNumber,
					"recommendation": vuln.Recommendation,
					"status":         vuln.Status,
					"discovered_at":  vuln.CreatedAt.Format(time.RFC3339),
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
		"count":           len(vulnerabilities),
		"timestamp":       time.Now().Format(time.RFC3339),
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
