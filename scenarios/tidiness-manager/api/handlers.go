package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// LightScanRequest defines the request body for light scanning
type LightScanRequest struct {
	ScenarioPath string `json:"scenario_path"`
	TimeoutSec   int    `json:"timeout_sec,omitempty"`
	Timeout      string `json:"timeout,omitempty"`     // backwards compatibility with tests sending string
	Incremental  bool   `json:"incremental,omitempty"` // Only scan modified files
}

// ParseRequest defines request for parsing lint/type output
type ParseRequest struct {
	Scenario string `json:"scenario"`
	Tool     string `json:"tool"`
	Output   string `json:"output"`
}

// handleLightScan performs a complete light scan on a scenario
func (s *Server) handleLightScan(w http.ResponseWriter, r *http.Request) {
	var req LightScanRequest
	if !decodeAndValidateJSON(w, r, &req) {
		return
	}

	if strings.TrimSpace(req.ScenarioPath) == "" {
		respondError(w, http.StatusBadRequest, "scenario_path is required")
		return
	}

	// Support legacy "timeout" field encoded as string
	if req.TimeoutSec == 0 && strings.TrimSpace(req.Timeout) != "" {
		if parsed, err := strconv.Atoi(req.Timeout); err == nil && parsed > 0 {
			req.TimeoutSec = parsed
		} else {
			respondError(w, http.StatusBadRequest, "timeout must be a positive integer")
			return
		}
	}

	if err := s.ensureScanCoordinator(); err != nil {
		respondError(w, http.StatusInternalServerError, "scan coordinator not initialized")
		return
	}

	result, scanErr := s.scanCoordinator.LightScan(r.Context(), req)
	if scanErr != nil {
		respondError(w, scanErr.Status, scanErr.Message)
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// handleParseLint parses lint output into structured issues
func (s *Server) handleParseLint(w http.ResponseWriter, r *http.Request) {
	var req ParseRequest
	if !decodeAndValidateJSON(w, r, &req) {
		return
	}

	if req.Scenario == "" || req.Tool == "" {
		respondError(w, http.StatusBadRequest, "scenario and tool are required")
		return
	}

	issues := ParseLintOutput(req.Scenario, req.Tool, req.Output)
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"issues": issues,
		"count":  len(issues),
	})
}

// handleParseType parses type checker output into structured issues
func (s *Server) handleParseType(w http.ResponseWriter, r *http.Request) {
	var req ParseRequest
	if !decodeAndValidateJSON(w, r, &req) {
		return
	}

	if req.Scenario == "" || req.Tool == "" {
		respondError(w, http.StatusBadRequest, "scenario and tool are required")
		return
	}

	issues := ParseTypeOutput(req.Scenario, req.Tool, req.Output)
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"issues": issues,
		"count":  len(issues),
	})
}

// handleRefactorRecommendations returns prioritized refactor candidates, optionally
// triggering a light scan to seed metrics when none exist.
func (s *Server) handleRefactorRecommendations(w http.ResponseWriter, r *http.Request) {
	scenario := r.URL.Query().Get("scenario")
	if scenario == "" {
		respondError(w, http.StatusBadRequest, "scenario parameter is required")
		return
	}

	if err := s.ensureScanCoordinator(); err != nil {
		respondError(w, http.StatusInternalServerError, "scan coordinator not initialized")
		return
	}

	scenarioName, err := s.scanCoordinator.NormalizeScenarioName(scenario)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	limit := parseIntParam(r, "limit", 10)
	sortBy := parseStringParam(r, "sort_by", "priority")
	minLines := parseIntParam(r, "min_lines", 0)
	maxVisits := parseIntParam(r, "max_visits", 0)
	autoScan := parseBoolParam(r, "auto_scan")

	if autoScan {
		if err := s.scanCoordinator.EnsureFileMetrics(r.Context(), scenarioName); err != nil {
			s.log("auto-scan failed", map[string]interface{}{
				"error":    err.Error(),
				"scenario": scenarioName,
			})
		}
	}

	recommender := NewRefactorRecommender(s.db, s.campaignMgr)
	recommendations, err := recommender.GetRecommendations(
		r.Context(),
		scenarioName,
		limit,
		sortBy,
		minLines,
		maxVisits,
	)
	if err != nil {
		s.log("failed to get refactor recommendations", map[string]interface{}{
			"error":    err.Error(),
			"scenario": scenarioName,
		})
		respondError(w, http.StatusInternalServerError, "failed to get recommendations")
		return
	}

	response := map[string]interface{}{
		"scenario":        scenarioName,
		"recommendations": recommendations,
		"count":           len(recommendations),
	}

	// Add warning if no data exists for scenario
	if len(recommendations) == 0 {
		hasMetrics, _ := s.scanCoordinator.HasMetricsForScenario(r.Context(), scenarioName)
		if !hasMetrics {
			response["warning"] = "no file metrics found for scenario - run 'tidiness-manager scan <scenario-path>' first"
		}
	}

	respondJSON(w, http.StatusOK, response)
}

// respondJSON writes a JSON response
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	setStandardSecurityHeaders(w)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func setStandardSecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Strict-Transport-Security", "max-age=31536000")
	w.Header().Set("Access-Control-Allow-Origin", "http://localhost")
}

// respondError writes an error response
func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{
		"error": message,
	})
}

// decodeAndValidateJSON decodes JSON request body and validates required fields
// Security: Limits request body size to prevent DoS attacks
func decodeAndValidateJSON(w http.ResponseWriter, r *http.Request, v interface{}) bool {
	// Limit request body to 10MB to prevent memory exhaustion
	const maxBodySize = 10 * 1024 * 1024
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

	decoder := json.NewDecoder(r.Body)
	// Security: DisallowUnknownFields prevents injection of unexpected fields
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(v); err != nil {
		// Don't leak parsing details to client
		respondError(w, http.StatusBadRequest, "invalid request body")
		return false
	}
	return true
}

// parseIntParam extracts and parses an integer query parameter with a default value
func parseIntParam(r *http.Request, key string, defaultValue int) int {
	if val := r.URL.Query().Get(key); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			return parsed
		}
	}
	return defaultValue
}

// parseStringParam extracts a string query parameter with a default value
func parseStringParam(r *http.Request, key, defaultValue string) string {
	if val := r.URL.Query().Get(key); val != "" {
		return val
	}
	return defaultValue
}

// parseBoolParam extracts and parses a boolean query parameter
func parseBoolParam(r *http.Request, key string) bool {
	return r.URL.Query().Get(key) == "true"
}

// handleSmartScan performs AI-powered smart scanning (TM-SS-001, TM-SS-002)
func (s *Server) handleSmartScan(w http.ResponseWriter, r *http.Request) {
	var req SmartScanRequest
	if !decodeAndValidateJSON(w, r, &req) {
		return
	}

	if req.Scenario == "" {
		respondError(w, http.StatusBadRequest, "scenario is required")
		return
	}

	if len(req.Files) == 0 {
		respondError(w, http.StatusBadRequest, "files list cannot be empty")
		return
	}

	if err := s.ensureScanCoordinator(); err != nil {
		respondError(w, http.StatusInternalServerError, "scan coordinator not initialized")
		return
	}

	result, scanErr := s.scanCoordinator.SmartScan(r.Context(), req)
	if scanErr != nil {
		respondError(w, scanErr.Status, scanErr.Message)
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// TidinessScanRequest defines the request body for maintainability scanning.
type TidinessScanRequest struct {
	ScenarioName string `json:"scenario_name"`
	TimeoutSec   int    `json:"timeout_sec,omitempty"`
}

// TidinessScanResponse is the normalized maintainability contract consumed by
// Test Genie and agents. It intentionally excludes lint/type/static-quality
// policy findings, which are owned by quality-health.
type TidinessScanResponse struct {
	Scenario   string              `json:"scenario"`
	Status     string              `json:"status"`
	Findings   []TidinessFinding   `json:"findings"`
	Violations []TidinessFinding   `json:"violations"` // compatibility alias for simple consumers
	Summary    TidinessScanSummary `json:"summary"`
}

type TidinessScanSummary struct {
	TotalFindings int `json:"total_findings"`
	LongFiles     int `json:"long_files"`
	Complexity    int `json:"complexity"`
	Duplication   int `json:"duplication"`
	TechDebt      int `json:"tech_debt"`
	Coupling      int `json:"coupling"`
}

type TidinessFinding struct {
	ID                     string         `json:"id"`
	RuleID                 string         `json:"rule_id"`
	Scenario               string         `json:"scenario"`
	FilePath               string         `json:"file_path,omitempty"`
	Symbol                 string         `json:"symbol,omitempty"`
	LineNumber             int            `json:"line_number,omitempty"`
	Category               string         `json:"category"`
	Severity               string         `json:"severity"`
	Title                  string         `json:"title"`
	Description            string         `json:"description"`
	Evidence               map[string]any `json:"evidence,omitempty"`
	WhyItMatters           string         `json:"why_it_matters"`
	RecommendedRemediation string         `json:"recommended_remediation"`
	Remediation            string         `json:"remediation"` // compatibility alias
	CampaignGroupHint      string         `json:"campaign_group_hint,omitempty"`
}

// handleTidinessScan scans a scenario for maintainability/tidiness debt.
func (s *Server) handleTidinessScan(w http.ResponseWriter, r *http.Request) {
	var req TidinessScanRequest
	if !decodeAndValidateJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.ScenarioName) == "" {
		respondError(w, http.StatusBadRequest, "scenario_name is required")
		return
	}
	if s.scenarioLocator == nil {
		respondError(w, http.StatusInternalServerError, "scenario locator not initialized")
		return
	}

	scenarioName, err := s.scenarioLocator.ValidateScenarioName(req.ScenarioName)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	scenarioPath, err := s.scenarioLocator.ScenarioPath(scenarioName)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	timeout := 120 * time.Second
	if req.TimeoutSec > 0 {
		if req.TimeoutSec > 600 {
			respondError(w, http.StatusBadRequest, "timeout_sec cannot exceed 600 seconds")
			return
		}
		timeout = time.Duration(req.TimeoutSec) * time.Second
	}

	result, err := buildTidinessScan(r.Context(), scenarioName, scenarioPath, timeout)
	if err != nil {
		s.log("tidiness scan failed", map[string]interface{}{
			"error":    err.Error(),
			"scenario": scenarioName,
		})
		respondError(w, http.StatusInternalServerError, "tidiness scan failed")
		return
	}
	respondJSON(w, http.StatusOK, result)
}

func buildTidinessScan(ctx context.Context, scenarioName, scenarioPath string, timeout time.Duration) (*TidinessScanResponse, error) {
	scanner := NewLightScanner(scenarioPath, timeout)
	fileMetrics, err := scanner.collectFileMetrics()
	if err != nil {
		return nil, err
	}
	languageMetrics, err := scanner.collectLanguageMetrics(ctx)
	if err != nil {
		languageMetrics = map[Language]*LanguageMetrics{}
	}

	findings := make([]TidinessFinding, 0)
	const longFileThreshold = 500
	const longTestFileThreshold = 1250
	for _, metric := range fileMetrics {
		threshold := longFileThreshold
		if IsTestFilePath(metric.Path) {
			threshold = longTestFileThreshold
		}
		if metric.Lines > threshold {
			findings = append(findings, newTidinessFinding(scenarioName, "long-file", "length", severityForLineCount(metric.Lines, threshold), metric.Path, "", 1,
				fmt.Sprintf("File has %d lines", metric.Lines),
				fmt.Sprintf("File has %d lines, exceeding the tidiness threshold of %d.", metric.Lines, threshold),
				map[string]any{"lines": metric.Lines, "threshold": threshold},
				"Large files slow review, hide ownership boundaries, and make agent edits riskier.",
				"Split the file around cohesive responsibilities and move tests/helpers with the code they support.",
				"file-size"))
		}
	}

	codeAnalyzer := NewCodeMetricsAnalyzer(scenarioPath)
	for lang, metrics := range languageMetrics {
		langInfoFiles := filesForLanguageMetric(scenarioPath, lang)
		if len(langInfoFiles) == 0 {
			continue
		}
		for _, relPath := range langInfoFiles {
			if codeMetrics, err := codeAnalyzer.analyzeFile(filepath.Join(scenarioPath, relPath), lang); err == nil {
				findings = append(findings, techDebtFindings(scenarioName, relPath, codeMetrics)...)
				if codeMetrics.ImportCount > DefaultIssueGeneratorConfig().HighImportThreshold {
					threshold := DefaultIssueGeneratorConfig().HighImportThreshold
					findings = append(findings, newTidinessFinding(scenarioName, "high-coupling", "coupling", severityForCoupling(codeMetrics.ImportCount, threshold), relPath, "", 1,
						fmt.Sprintf("File has %d imports", codeMetrics.ImportCount),
						fmt.Sprintf("File has %d imports, exceeding the coupling threshold of %d.", codeMetrics.ImportCount, threshold),
						map[string]any{"imports": codeMetrics.ImportCount, "threshold": threshold},
						"High import counts often signal broad responsibilities and brittle dependency surfaces.",
						"Extract focused collaborators or move unrelated responsibilities into narrower modules.",
						"coupling"))
				}
			}
		}

		if metrics.Complexity != nil && !metrics.Complexity.Skipped {
			for _, complexFile := range metrics.Complexity.HighComplexityFiles {
				findings = append(findings, newTidinessFinding(scenarioName, "high-complexity", "complexity", severityForComplexity(complexFile.Complexity, metrics.Complexity.Threshold), complexFile.Path, complexFile.Function, complexFile.Line,
					fmt.Sprintf("%s has cyclomatic complexity %d", complexFile.Function, complexFile.Complexity),
					fmt.Sprintf("Function %s has cyclomatic complexity %d, exceeding the threshold of %d.", complexFile.Function, complexFile.Complexity, metrics.Complexity.Threshold),
					map[string]any{"complexity": complexFile.Complexity, "threshold": metrics.Complexity.Threshold, "tool": metrics.Complexity.Tool},
					"Highly branched code is harder to test, review, and safely modify.",
					"Extract decision branches into named helpers and add focused tests around each behavior path.",
					"complexity"))
			}
		}

		if metrics.Duplicates != nil && !metrics.Duplicates.Skipped {
			for i, block := range metrics.Duplicates.DuplicateBlocks {
				primaryPath := ""
				line := 0
				if len(block.Files) > 0 {
					primaryPath = block.Files[0].Path
					line = block.Files[0].StartLine
				}
				findings = append(findings, newTidinessFinding(scenarioName, "duplicated-code", "duplication", severityForDuplication(float64(block.Lines), 10), primaryPath, "", line,
					fmt.Sprintf("Duplicated block spans %d lines", block.Lines),
					fmt.Sprintf("Duplicated code block #%d spans %d lines across %d locations.", i+1, block.Lines, len(block.Files)),
					map[string]any{"lines": block.Lines, "locations": block.Files, "tool": metrics.Duplicates.Tool},
					"Duplicated code multiplies future fixes and makes behavior drift likely.",
					"Extract the shared behavior or intentionally document why the copies must diverge.",
					"duplication"))
			}
		}
	}

	sort.SliceStable(findings, func(i, j int) bool {
		if supportRank(findings[i].Severity) != supportRank(findings[j].Severity) {
			return supportRank(findings[i].Severity) < supportRank(findings[j].Severity)
		}
		if findings[i].FilePath != findings[j].FilePath {
			return findings[i].FilePath < findings[j].FilePath
		}
		return findings[i].RuleID < findings[j].RuleID
	})

	summary := summarizeTidinessFindings(findings)
	status := "passed"
	if len(findings) > 0 {
		status = "issues_found"
	}
	return &TidinessScanResponse{
		Scenario:   scenarioName,
		Status:     status,
		Findings:   findings,
		Violations: findings,
		Summary:    summary,
	}, nil
}

func filesForLanguageMetric(scenarioPath string, lang Language) []string {
	detector := NewLanguageDetector(scenarioPath)
	languages, err := detector.DetectLanguages()
	if err != nil {
		return nil
	}
	if info, ok := languages[lang]; ok {
		return info.Files
	}
	return nil
}

func techDebtFindings(scenario, relPath string, metrics *FileCodeMetrics) []TidinessFinding {
	config := DefaultIssueGeneratorConfig()
	debt := metrics.TodoCount + metrics.FixmeCount + metrics.HackCount
	if debt <= config.HighTechDebtThreshold {
		return nil
	}
	return []TidinessFinding{
		newTidinessFinding(scenario, "tech-debt-markers", "technical_debt", severityForTechDebt(debt, config.HighTechDebtThreshold), relPath, "", 1,
			fmt.Sprintf("File has %d TODO/FIXME/HACK markers", debt),
			fmt.Sprintf("File has %d tracked debt markers: %d TODO, %d FIXME, %d HACK.", debt, metrics.TodoCount, metrics.FixmeCount, metrics.HackCount),
			map[string]any{"todo": metrics.TodoCount, "fixme": metrics.FixmeCount, "hack": metrics.HackCount, "threshold": config.HighTechDebtThreshold},
			"Dense local debt markers are a sign that cleanup has stopped being managed intentionally.",
			"Resolve stale markers, convert real work into tracked issues, and keep only comments that explain active constraints.",
			"technical-debt"),
	}
}

func newTidinessFinding(scenario, ruleID, category, severity, filePath, symbol string, line int, title, description string, evidence map[string]any, why, remediation, campaignGroup string) TidinessFinding {
	idParts := []string{scenario, ruleID, category, filePath, symbol, strconv.Itoa(line)}
	remediation = strings.TrimSpace(remediation)
	return TidinessFinding{
		ID:                     strings.ReplaceAll(strings.Join(idParts, ":"), " ", "-"),
		RuleID:                 strings.ToUpper(strings.ReplaceAll(ruleID, "-", "_")),
		Scenario:               scenario,
		FilePath:               filePath,
		Symbol:                 symbol,
		LineNumber:             line,
		Category:               category,
		Severity:               severity,
		Title:                  title,
		Description:            description,
		Evidence:               evidence,
		WhyItMatters:           why,
		RecommendedRemediation: remediation,
		Remediation:            remediation,
		CampaignGroupHint:      campaignGroup,
	}
}

func summarizeTidinessFindings(findings []TidinessFinding) TidinessScanSummary {
	summary := TidinessScanSummary{TotalFindings: len(findings)}
	for _, finding := range findings {
		switch finding.Category {
		case "length":
			summary.LongFiles++
		case "complexity":
			summary.Complexity++
		case "duplication":
			summary.Duplication++
		case "technical_debt":
			summary.TechDebt++
		case "coupling":
			summary.Coupling++
		}
	}
	return summary
}

func supportRank(severity string) int {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "critical":
		return 0
	case "high":
		return 1
	case "medium":
		return 2
	case "low":
		return 3
	default:
		return 4
	}
}

// storeAIIssue stores an AI-discovered issue in the database (TM-SS-002, TM-API-006)
func (s *Server) storeAIIssue(ctx context.Context, scenario string, issue AIIssue, sessionID string, campaignID *int) error {
	if s.store == nil {
		return fmt.Errorf("store not initialized")
	}
	return s.store.StoreAIIssue(ctx, scenario, issue, sessionID, campaignID)
}

// recordScanHistory records a scan in the audit trail
func (s *Server) recordScanHistory(ctx context.Context, scenario, scanType string, result *SmartScanResult, campaignID *int) error {
	if s.store == nil {
		return fmt.Errorf("store not initialized")
	}
	return s.store.RecordScanHistory(ctx, scenario, scanType, result, campaignID)
}

// persistFileMetrics stores file metrics from a light scan in the database (TM-FM-001, TM-FM-002)
func (s *Server) persistFileMetrics(ctx context.Context, scenario string, metrics []FileMetric) error {
	if len(metrics) == 0 {
		return nil
	}
	if s.store == nil {
		return fmt.Errorf("store not initialized")
	}
	return s.store.PersistFileMetrics(ctx, scenario, metrics)
}

// GenerateIssuesFromMetricsRequest defines request for generating issues from stored metrics
type GenerateIssuesFromMetricsRequest struct {
	Scenario string `json:"scenario"`
}

// handleGenerateIssuesFromMetrics generates issues from existing file metrics in the database
// This is useful when metrics exist but issues weren't generated (e.g., after incremental scans)
func (s *Server) handleGenerateIssuesFromMetrics(w http.ResponseWriter, r *http.Request) {
	var req GenerateIssuesFromMetricsRequest
	if !decodeAndValidateJSON(w, r, &req) {
		return
	}

	if req.Scenario == "" {
		respondError(w, http.StatusBadRequest, "scenario is required")
		return
	}

	if s.store == nil {
		respondError(w, http.StatusInternalServerError, "store not initialized")
		return
	}

	// Get existing file metrics from database
	metrics, err := s.store.GetDetailedFileMetrics(r.Context(), req.Scenario)
	if err != nil {
		s.log("failed to get file metrics", map[string]interface{}{
			"error":    err.Error(),
			"scenario": req.Scenario,
		})
		respondError(w, http.StatusInternalServerError, "failed to get file metrics")
		return
	}

	if len(metrics) == 0 {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"scenario":  req.Scenario,
			"generated": 0,
			"inserted":  0,
			"message":   "no file metrics found for scenario - run a scan first",
		})
		return
	}

	// Generate issues from metrics
	config := DefaultIssueGeneratorConfig()
	issues := GenerateIssuesFromMetrics(req.Scenario, metrics, config)

	if len(issues) == 0 {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"scenario":      req.Scenario,
			"metrics_count": len(metrics),
			"generated":     0,
			"inserted":      0,
			"message":       "no issues exceeded thresholds",
		})
		return
	}

	// Persist the generated issues
	inserted, persistErr := s.store.StoreLintTypeIssues(r.Context(), req.Scenario, issues)
	if persistErr != nil {
		s.log("failed to persist metric-based issues", map[string]interface{}{
			"error":    persistErr.Error(),
			"scenario": req.Scenario,
			"count":    len(issues),
		})
		respondError(w, http.StatusInternalServerError, "failed to persist issues")
		return
	}

	// Resolve stale metric issues that are no longer present in the fresh set
	resolved, resolveErr := s.store.ResolveStaleMetricIssues(r.Context(), req.Scenario, issues)
	if resolveErr != nil {
		s.log("failed to resolve stale metric issues", map[string]interface{}{
			"error": resolveErr.Error(), "scenario": req.Scenario,
		})
		// Don't fail the request - generation succeeded
	}

	s.log("generated issues from metrics", map[string]interface{}{
		"scenario":      req.Scenario,
		"metrics_count": len(metrics),
		"generated":     len(issues),
		"inserted":      inserted,
		"resolved":      resolved,
	})

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"scenario":      req.Scenario,
		"metrics_count": len(metrics),
		"generated":     len(issues),
		"inserted":      inserted,
		"resolved":      resolved,
	})
}
