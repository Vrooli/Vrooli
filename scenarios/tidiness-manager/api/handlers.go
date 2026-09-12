package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/maturity-go/assessment"
	repocontract "github.com/vrooli/repo-contract-go"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
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

type BudgetAuditRequest struct {
	Scenario     string `json:"scenario"`
	ScenarioPath string `json:"scenario_path"`
	TimeoutSec   int    `json:"timeout_sec,omitempty"`
}

func (s *Server) handleBudgetAudit(w http.ResponseWriter, r *http.Request) {
	var req BudgetAuditRequest
	if !decodeAndValidateJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.ScenarioPath) == "" {
		respondError(w, http.StatusBadRequest, "scenario_path is required")
		return
	}
	scenario := strings.TrimSpace(req.Scenario)
	if scenario == "" {
		scenario = filepath.Base(filepath.Clean(req.ScenarioPath))
	}
	if audit, ok := s.loadBudgetAudit(req.ScenarioPath); ok {
		respondJSON(w, http.StatusOK, audit)
		return
	}
	respondError(w, http.StatusConflict, fmt.Sprintf("no completed tidiness validation is available for %s; run the aggregate tidiness gate before auditing budgets", scenario))
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

// TidinessScanResponse is the normalized maintainability contract consumed by
// the shared ScenarioValidationService native_detail. It intentionally excludes
// lint/type/static-quality policy findings, which are owned by quality-health.
type TidinessScanResponse struct {
	Scenario      string                       `json:"scenario"`
	Status        string                       `json:"status"`
	Findings      []TidinessFinding            `json:"findings"`
	Violations    []TidinessFinding            `json:"violations"` // compatibility alias for simple consumers
	Summary       TidinessScanSummary          `json:"summary"`
	Opportunities []DuplicationOpportunity     `json:"opportunities,omitempty"`
	Assessment    *commonv1.MaturityAssessment `json:"assessment"`
	SeamFiles     []string                     `json:"seam_files"`
}

type TidinessScanSummary struct {
	TotalFindings          int `json:"total_findings"`
	LongFiles              int `json:"long_files"`
	Complexity             int `json:"complexity"`
	Duplication            int `json:"duplication"`
	TechDebt               int `json:"tech_debt"`
	Coupling               int `json:"coupling"`
	BypassedSeams          int `json:"bypassed_seams"`
	DroppedDuplicateGroups int `json:"dropped_duplicate_groups,omitempty"`
	DroppedDuplicateLines  int `json:"dropped_duplicate_lines,omitempty"`
	DuplicationLineDebt    int `json:"duplication_line_debt,omitempty"`
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

func buildTidinessScan(ctx context.Context, scenarioName, scenarioPath string, timeout time.Duration, excludes ...string) (*TidinessScanResponse, error) {
	response, _, err := buildTidinessScanWithAudit(ctx, scenarioName, scenarioPath, timeout, excludes...)
	return response, err
}

func buildTidinessScanWithAudit(ctx context.Context, scenarioName, scenarioPath string, timeout time.Duration, excludes ...string) (*TidinessScanResponse, *BudgetAuditReport, error) {
	collector := metricsFrom(ctx)

	scanStage := collector.Stage("scan")
	scanner := NewLightScanner(scenarioPath, timeout, excludes...)
	fileMetrics, err := scanner.collectFileMetrics()
	if err != nil {
		scanStage.End()
		return nil, nil, err
	}
	languageMetrics, err := scanner.collectLanguageMetrics(ctx)
	if err != nil {
		languageMetrics = map[Language]*LanguageMetrics{}
	}
	scanStage.End()

	analysisStage := collector.Stage("analysis")

	roles := newRoleCache(scenarioPath)

	findings := make([]TidinessFinding, 0)
	targetKind := SeamTargetScenario
	if filepath.Base(filepath.Clean(scenarioPath)) == "internal" {
		targetKind = SeamTargetControlPlane
	}
	seams, seamResolution, err := ResolveSeams(scenarioPath, targetKind)
	if err != nil {
		analysisStage.End()
		return nil, nil, err
	}
	seamHits, err := ScanSeams(seamResolution.ScanRoot, seams)
	if err != nil {
		analysisStage.End()
		return nil, nil, err
	}
	findings = append(findings, seamFindings(scenarioName, seams, seamHits)...)
	droppedDuplicateGroups, droppedDuplicateLines := 0, 0
	const longFileThreshold = 500
	const longTestFileThreshold = 1250
	for _, metric := range fileMetrics {
		role := roles.role(metric.Path)
		// Generated code is fully excluded; declarative const registries are
		// long by design, so they do not gate on file length.
		if role == FileRoleGenerated || role == FileRoleDeclarativeWiring {
			continue
		}
		threshold := longFileThreshold
		if role == FileRoleTest || role == FileRoleTestSupport {
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
		langInfoFiles := filesForLanguageMetric(scenarioPath, lang, excludes...)
		if len(langInfoFiles) == 0 {
			continue
		}
		for _, relPath := range langInfoFiles {
			fileRole := roles.role(relPath)
			// Generated code is excluded from every finding family.
			if fileRole == FileRoleGenerated {
				continue
			}
			if codeMetrics, err := codeAnalyzer.analyzeFile(filepath.Join(scenarioPath, relPath), lang); err == nil {
				findings = append(findings, techDebtFindings(scenarioName, relPath, codeMetrics)...)
				// Composition roots aggregate many collaborators by design;
				// their high import count is structural, not coupling debt.
				if fileRole != FileRoleCompositionRoot && codeMetrics.ImportCount > DefaultIssueGeneratorConfig().HighImportThreshold {
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
				if roles.role(complexFile.Path) == FileRoleGenerated {
					continue
				}
				findings = append(findings, newTidinessFinding(scenarioName, "high-complexity", "complexity", severityForComplexity(complexFile.Complexity, metrics.Complexity.Threshold), complexFile.Path, complexFile.Function, complexFile.Line,
					fmt.Sprintf("%s has cyclomatic complexity %d", complexFile.Function, complexFile.Complexity),
					fmt.Sprintf("Function %s has cyclomatic complexity %d, exceeding the threshold of %d.", complexFile.Function, complexFile.Complexity, metrics.Complexity.Threshold),
					map[string]any{"complexity": complexFile.Complexity, "threshold": metrics.Complexity.Threshold, "tool": metrics.Complexity.Tool},
					"Highly branched code is harder to test, review, and safely modify.",
					"Extract decision branches into named helpers and add focused tests around each behavior path.",
					"complexity"))
			}
		}

		if metrics.Duplicates != nil {
			droppedDuplicateGroups += metrics.Duplicates.DroppedGroups
			droppedDuplicateLines += metrics.Duplicates.DroppedLines
		}
		findings = append(findings, duplicationFindings(scenarioName, scenarioPath, roles, metrics.Duplicates)...)
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
	summary.DroppedDuplicateGroups = droppedDuplicateGroups
	summary.DroppedDuplicateLines = droppedDuplicateLines
	audit := newBudgetAuditReport(scenarioName, scenarioPath, seams, seamHits, summary)
	if budgetFinding := tidinessBudgetFinding(scenarioName, scenarioPath, summary); budgetFinding != nil {
		findings = append(findings, *budgetFinding)
		summary = summarizeTidinessFindings(findings)
		summary.DroppedDuplicateGroups, summary.DroppedDuplicateLines = droppedDuplicateGroups, droppedDuplicateLines
	}
	status := "passed"
	if len(findings) > 0 {
		status = "issues_found"
	}
	analysisStage.Gauge("findings", float64(len(findings)))
	analysisStage.End()

	maturityStage := collector.Stage("maturity")
	spec, err := loadTidinessMaturitySpec()
	if err != nil {
		maturityStage.End()
		return nil, nil, err
	}
	maturityAssessment, err := buildTidinessMaturityAssessment(scenarioName, findings, spec)
	if err != nil {
		maturityStage.End()
		return nil, nil, err
	}
	opportunities := rankDuplicationOpportunities(findings)
	applyDuplicationOpportunityPresentation(maturityAssessment, opportunities)
	maturityStage.End()

	return &TidinessScanResponse{
		Scenario:      scenarioName,
		Status:        status,
		Findings:      findings,
		Violations:    findings,
		Summary:       summary,
		Opportunities: opportunities,
		Assessment:    maturityAssessment,
		SeamFiles:     seamResolution.Files,
	}, audit, nil
}

func loadTidinessMaturitySpec() (*assessment.Spec, error) {
	_, repoRoot, err := repocontract.LoadDefaultFromEnvOrCWD()
	if err != nil {
		return nil, fmt.Errorf("resolve repo root for tidiness maturity spec: %w", err)
	}
	return assessment.LoadSpecFromScenario(filepath.Join(repoRoot, "scenarios", "tidiness-manager"))
}

func buildTidinessMaturityAssessment(scenarioName string, findings []TidinessFinding, spec *assessment.Spec) (*commonv1.MaturityAssessment, error) {
	if spec == nil {
		return nil, fmt.Errorf("tidiness maturity spec is required")
	}
	assessed := make([]assessment.Finding, 0, len(findings))
	for _, finding := range findings {
		assessed = append(assessed, assessment.Finding{
			Code:        finding.RuleID,
			Severity:    tidinessSeverityToAssessment(finding.Severity),
			Title:       finding.Title,
			Message:     finding.Description,
			Location:    tidinessFindingLocation(finding),
			Remediation: finding.RecommendedRemediation,
			Source:      architecturev1.FindingSource_FINDING_SOURCE_TIDINESS,
			Phase:       spec.Phase,
		})
	}
	return assessment.BuildProtoAssessment(assessment.BuildInput{
		Scenario: scenarioName,
		Spec:     *spec,
		Findings: assessed,
	})
}

func tidinessSeverityToAssessment(severity string) string {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "critical", "high":
		return architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR.String()
	case "medium", "warning", "warn":
		return architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING.String()
	case "low", "info":
		return architecturev1.FindingSeverity_FINDING_SEVERITY_INFO.String()
	default:
		return severity
	}
}

func tidinessFindingLocation(finding TidinessFinding) string {
	location := strings.TrimSpace(finding.FilePath)
	if location != "" && finding.LineNumber > 0 {
		return fmt.Sprintf("%s:%d", location, finding.LineNumber)
	}
	return location
}

func filesForLanguageMetric(scenarioPath string, lang Language, excludes ...string) []string {
	detector := NewLanguageDetector(scenarioPath)
	languages, err := detector.DetectLanguages()
	if err != nil {
		return nil
	}
	if info, ok := languages[lang]; ok {
		return NewLightScanner(scenarioPath, 0, excludes...).filterExcludedFiles(info.Files)
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
			if finding.RuleID == "DUPLICATED_CODE" {
				if debt, ok := finding.Evidence["duplication_line_debt"].(int); ok {
					summary.DuplicationLineDebt += debt
				}
			}
		case "technical_debt":
			summary.TechDebt++
		case "coupling":
			summary.Coupling++
		case "seam":
			summary.BypassedSeams++
		}
	}
	return summary
}

func seamFindings(scenario string, seams []Seam, hits []SeamHit) []TidinessFinding {
	hitsBySeam := make(map[string][]SeamHit, len(seams))
	for _, hit := range hits {
		hitsBySeam[hit.SeamID] = append(hitsBySeam[hit.SeamID], hit)
	}
	findings := make([]TidinessFinding, 0, len(hits))
	for _, seam := range seams {
		seamHits := hitsBySeam[seam.ID]
		observed := len(seamHits)
		if seam.Budget > observed+seam.Reserve {
			slack := seam.Budget - observed - seam.Reserve
			findings = append(findings, newTidinessFinding(
				scenario, "seam-budget-slack", "seam", "high", "", seam.ID, 0,
				fmt.Sprintf("Canonical seam %s has unjustified budget slack", seam.ID),
				fmt.Sprintf("Seam %s declares budget %d with observation %d and reserve %d, leaving slack %d.", seam.ID, seam.Budget, observed, seam.Reserve, slack),
				map[string]any{"seam_id": seam.ID, "budget": seam.Budget, "observed": observed, "reserve": seam.Reserve, "slack": slack},
				"A seam budget must describe observed debt or an explicit reserve, not pre-approve unknown debt.", "Lower the budget or declare and justify the required reserve.", "canonical-seam-budget",
			))
		}
		if seam.Budget > 0 && observed == 0 {
			findings = append(findings, newTidinessFinding(
				scenario, "seam-rule-never-matches", "seam", "critical", "", seam.ID, 0,
				fmt.Sprintf("Canonical seam %s matched nothing", seam.ID),
				fmt.Sprintf("Seam %s has a non-zero budget of %d but matched nothing across every scanned path.", seam.ID, seam.Budget),
				map[string]any{"seam_id": seam.ID, "budget": seam.Budget, "observed": observed},
				"A rule that never matches reports coverage that does not exist.", "Repair or remove the rule, then re-derive its budget from a live observation.", "canonical-seam-budget",
			))
		}
		for index, hit := range seamHits {
			if index+1 <= seam.Budget {
				continue
			}
			findings = append(findings, newTidinessFinding(
				scenario, "bypassed-seam", "seam", hit.Severity, hit.Path, hit.Symbol, hit.Line,
				fmt.Sprintf("Canonical seam %s is bypassed", hit.SeamID),
				fmt.Sprintf("%s bypasses canonical seam %s (%s).", hit.Symbol, hit.SeamID, hit.Canonical),
				map[string]any{"seam_id": hit.SeamID, "canonical": hit.Canonical, "why": hit.Why, "budget": hit.Budget, "observed": index + 1},
				hit.Why, hit.Remediation, "canonical-seam",
			))
			if hit.Analyzer != "" {
				findings[len(findings)-1].Evidence["analyzer"] = hit.Analyzer
			}
		}
	}
	return findings
}

type tidinessTestingConfig struct {
	Phases struct {
		Tidiness struct {
			Budgets tidinessBudgets `json:"budgets"`
		} `json:"tidiness"`
	} `json:"phases"`
}

type tidinessBudgets struct {
	DuplicationLineDebt         int    `json:"duplication_line_debt"`
	BaselineDuplicationLineDebt int    `json:"baseline_duplication_line_debt"`
	LongFiles                   int    `json:"long_files"`
	BaselineLongFiles           int    `json:"baseline_long_files"`
	Complexity                  int    `json:"complexity_over_threshold"`
	BaselineComplexity          int    `json:"baseline_complexity_over_threshold"`
	Coupling                    int    `json:"coupling_over_threshold"`
	BaselineCoupling            int    `json:"baseline_coupling_over_threshold"`
	DebtMarkers                 int    `json:"debt_markers"`
	BaselineDebtMarkers         int    `json:"baseline_debt_markers"`
	Reserve                     int    `json:"reserve,omitempty"`
	ReserveReason               string `json:"reserve_reason,omitempty"`
	Ratchet                     bool   `json:"ratchet"`
	declared                    map[string]bool
}

func (budgets *tidinessBudgets) UnmarshalJSON(data []byte) error {
	type budgetValues tidinessBudgets
	var values budgetValues
	if err := json.Unmarshal(data, &values); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	*budgets = tidinessBudgets(values)
	budgets.declared = make(map[string]bool, len(fields))
	for name := range fields {
		budgets.declared[name] = true
	}
	return nil
}

type BudgetAuditReport struct {
	Scenario string              `json:"scenario"`
	Seams    []SeamBudgetAudit   `json:"seams"`
	Metrics  []MetricBudgetAudit `json:"metrics"`
	Blocking bool                `json:"blocking"`
}

type SeamBudgetAudit struct {
	ID               string   `json:"id"`
	DeclaredBudget   int      `json:"declared_budget"`
	DeclaredBaseline *int     `json:"declared_baseline"`
	Observed         int      `json:"observed"`
	Reserve          int      `json:"reserve"`
	ReserveReason    string   `json:"reserve_reason,omitempty"`
	Verdicts         []string `json:"verdicts"`
}

type MetricBudgetAudit struct {
	Name             string   `json:"name"`
	DeclaredBudget   *int     `json:"declared_budget"`
	DeclaredBaseline *int     `json:"declared_baseline"`
	Observed         int      `json:"observed"`
	Reserve          int      `json:"reserve"`
	ReserveReason    string   `json:"reserve_reason,omitempty"`
	Verdicts         []string `json:"verdicts"`
}

func newBudgetAuditReport(scenario, scenarioPath string, seams []Seam, hits []SeamHit, summary TidinessScanSummary) *BudgetAuditReport {
	hitCounts := make(map[string]int, len(seams))
	for _, hit := range hits {
		hitCounts[hit.SeamID]++
	}
	report := &BudgetAuditReport{Scenario: scenario, Seams: make([]SeamBudgetAudit, 0, len(seams))}
	for _, seam := range seams {
		observed := hitCounts[seam.ID]
		verdicts := make([]string, 0, 2)
		if seam.Budget > observed+seam.Reserve {
			verdicts = append(verdicts, "SEAM_BUDGET_SLACK")
			report.Blocking = true
		}
		if seam.Budget > 0 && observed == 0 {
			verdicts = append(verdicts, "SEAM_RULE_NEVER_MATCHES")
			report.Blocking = true
		}
		if len(verdicts) == 0 {
			verdicts = append(verdicts, "ok")
		}
		report.Seams = append(report.Seams, SeamBudgetAudit{ID: seam.ID, DeclaredBudget: seam.Budget, Observed: observed, Reserve: seam.Reserve, ReserveReason: seam.ReserveReason, Verdicts: verdicts})
	}

	budgets, ok := loadTidinessBudgets(scenarioPath)
	if !ok {
		budgets = tidinessBudgets{declared: map[string]bool{}}
	}
	for _, check := range budgets.metricChecks(summary) {
		verdicts := metricBudgetVerdicts(budgets, check)
		if len(verdicts) == 0 {
			verdicts = append(verdicts, "ok")
		} else {
			report.Blocking = true
		}
		var budget, baseline *int
		if budgets.declared[check.name] {
			value := check.budget
			budget = &value
		}
		if budgets.declared[check.baselineName] {
			value := check.baseline
			baseline = &value
		}
		report.Metrics = append(report.Metrics, MetricBudgetAudit{Name: check.name, DeclaredBudget: budget, DeclaredBaseline: baseline, Observed: check.observed, Reserve: budgets.Reserve, ReserveReason: budgets.ReserveReason, Verdicts: verdicts})
	}
	return report
}

func metricBudgetVerdicts(budgets tidinessBudgets, check tidinessMetricCheck) []string {
	budgetDeclared := budgets.declared[check.name]
	baselineDeclared := budgets.declared[check.baselineName]
	var verdicts []string
	if budgets.Reserve > 0 && len(strings.TrimSpace(budgets.ReserveReason)) < 40 {
		verdicts = append(verdicts, "reserve_reason_missing")
	}
	if budgets.Ratchet && budgetDeclared && baselineDeclared && check.budget > 0 && check.budget == check.baseline && check.observed == check.baseline {
		verdicts = append(verdicts, "frozen_budget")
	}
	if budgets.Ratchet && budgetDeclared && baselineDeclared && check.budget > check.baseline {
		verdicts = append(verdicts, "ratchet_loosened_budget")
	}
	if budgets.Ratchet && baselineDeclared && check.observed > check.baseline {
		verdicts = append(verdicts, "ratchet_worsened_debt")
	}
	if budgetDeclared && check.observed > check.budget+budgets.Reserve {
		verdicts = append(verdicts, "budget_exceeded")
	}
	return verdicts
}

func loadTidinessBudgets(scenarioPath string) (tidinessBudgets, bool) {
	data, err := os.ReadFile(filepath.Join(scenarioPath, ".vrooli", "testing.json"))
	if err != nil {
		return tidinessBudgets{}, false
	}
	var config tidinessTestingConfig
	if json.Unmarshal(data, &config) != nil {
		return tidinessBudgets{}, false
	}
	return config.Phases.Tidiness.Budgets, true
}

func tidinessBudgetFinding(scenario, scenarioPath string, summary TidinessScanSummary) *TidinessFinding {
	budgets, ok := loadTidinessBudgets(scenarioPath)
	if !ok {
		return nil
	}
	for _, check := range budgets.metricChecks(summary) {
		budgetDeclared := budgets.declared[check.name]
		baselineDeclared := budgets.declared[check.baselineName]
		if budgets.Ratchet && budgetDeclared && baselineDeclared && check.budget > 0 && check.budget == check.baseline && check.observed == check.baseline {
			return ptrFinding(newTidinessFinding(scenario, "tidiness-budget-exceeded", "budget", "high", "", "", 0, "Tidiness budget is frozen at its observation", fmt.Sprintf("%s sets %s budget, baseline, and observed value to %d; the budget cannot detect recurring debt.", filepath.Join(scenarioPath, ".vrooli", "testing.json"), check.name, check.observed), map[string]any{"metric": check.name, "budget": check.budget, "baseline": check.baseline, "observed": check.observed, "violation": "frozen_budget"}, "A ratcheted budget must be below its recorded observation.", "Reduce the debt and set the budget below the recorded baseline.", "tidiness-budget"))
		}
		if budgets.Ratchet && budgetDeclared && baselineDeclared && check.budget > check.baseline {
			return ptrFinding(newTidinessFinding(scenario, "tidiness-budget-exceeded", "budget", "high", "", "", 0, "Tidiness budget loosens recorded baseline", fmt.Sprintf("%s budget is %d; recorded baseline is %d (loosening +%d).", check.name, check.budget, check.baseline, check.budget-check.baseline), map[string]any{"metric": check.name, "budget": check.budget, "baseline": check.baseline, "delta": check.budget - check.baseline, "violation": "ratchet_loosened_budget"}, "A ratcheted maintainability budget may not be loosened.", "Tighten the configured budget to the recorded baseline or below.", "tidiness-budget"))
		}
		if budgets.Ratchet && baselineDeclared && check.observed > check.baseline {
			return ptrFinding(newTidinessFinding(scenario, "tidiness-budget-exceeded", "budget", "high", "", "", 0, "Tidiness debt worsened from recorded baseline", fmt.Sprintf("%s is %d; recorded baseline is %d (delta +%d).", check.name, check.observed, check.baseline, check.observed-check.baseline), map[string]any{"metric": check.name, "baseline": check.baseline, "observed": check.observed, "delta": check.observed - check.baseline, "violation": "ratchet_worsened_debt"}, "A ratcheted maintainability baseline must not worsen.", "Reduce debt to the recorded baseline or below.", "tidiness-budget"))
		}
		if budgetDeclared && check.observed > check.budget+budgets.Reserve {
			return ptrFinding(newTidinessFinding(scenario, "tidiness-budget-exceeded", "budget", "high", "", "", 0, "Tidiness budget exceeded", fmt.Sprintf("%s is %d; budget is %d (delta +%d).", check.name, check.observed, check.budget, check.observed-check.budget), map[string]any{"metric": check.name, "budget": check.budget, "observed": check.observed, "delta": check.observed - check.budget}, "A configured maintainability budget regressed.", "Reduce the metric or explicitly tighten a truthful budget.", "tidiness-budget"))
		}
	}
	return nil
}

type tidinessMetricCheck struct {
	name         string
	baselineName string
	budget       int
	baseline     int
	observed     int
}

func (budgets tidinessBudgets) metricChecks(summary TidinessScanSummary) []tidinessMetricCheck {
	return []tidinessMetricCheck{
		{name: "duplication_line_debt", baselineName: "baseline_duplication_line_debt", budget: budgets.DuplicationLineDebt, baseline: budgets.BaselineDuplicationLineDebt, observed: summary.DuplicationLineDebt},
		{name: "long_files", baselineName: "baseline_long_files", budget: budgets.LongFiles, baseline: budgets.BaselineLongFiles, observed: summary.LongFiles},
		{name: "complexity_over_threshold", baselineName: "baseline_complexity_over_threshold", budget: budgets.Complexity, baseline: budgets.BaselineComplexity, observed: summary.Complexity},
		{name: "coupling_over_threshold", baselineName: "baseline_coupling_over_threshold", budget: budgets.Coupling, baseline: budgets.BaselineCoupling, observed: summary.Coupling},
		{name: "debt_markers", baselineName: "baseline_debt_markers", budget: budgets.DebtMarkers, baseline: budgets.BaselineDebtMarkers, observed: summary.TechDebt},
	}
}

func ptrFinding(f TidinessFinding) *TidinessFinding { return &f }

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
