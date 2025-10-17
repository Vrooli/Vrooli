package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

type TidinessProcessor struct {
	db *sql.DB
}

// Code Scanner types (replaces code-scanner workflow)
type CodeScanRequest struct {
	Paths           []string `json:"paths,omitempty"`
	Types           []string `json:"types,omitempty"`
	DeepScan        bool     `json:"deep_scan,omitempty"`
	Limit           int      `json:"limit,omitempty"`
	ExcludePatterns []string `json:"exclude_patterns,omitempty"`
}

type CodeScanResponse struct {
	Success     bool             `json:"success"`
	ScanID      string           `json:"scan_id"`
	ScanConfig  CodeScanRequest  `json:"scan_config"`
	StartedAt   string           `json:"started_at"`
	CompletedAt string           `json:"completed_at"`
	Statistics  ScanStatistics   `json:"statistics"`
	ResultsByType map[string]TypeResult `json:"results_by_type"`
	Issues      []ScanIssue      `json:"issues"`
	TotalIssues int              `json:"total_issues"`
	Error       *TidinessError   `json:"error,omitempty"`
}

type ScanStatistics struct {
	Total       int         `json:"total"`
	BySeverity  map[string]int `json:"by_severity"`
	Automatable int         `json:"automatable"`
	ManualReview int        `json:"manual_review"`
}

type TypeResult struct {
	Count    int      `json:"count"`
	Patterns []string `json:"patterns"`
}

type ScanIssue struct {
	FilePath        string `json:"file_path"`
	IssueType       string `json:"issue_type"`
	Severity        string `json:"severity"`
	ConfidenceScore float64 `json:"confidence_score"`
	SafeToAutomate  bool   `json:"safe_to_automate"`
	CleanupScript   string `json:"cleanup_script,omitempty"`
	Description     string `json:"description"`
}

// Pattern Analyzer types (replaces pattern-analyzer workflow)
type PatternAnalysisRequest struct {
	AnalysisType string   `json:"analysis_type"`
	Paths        []string `json:"paths,omitempty"`
	DeepAnalysis bool     `json:"deep_analysis,omitempty"`
}

type PatternAnalysisResponse struct {
	Success                bool                     `json:"success"`
	AnalysisID            string                   `json:"analysis_id"`
	AnalysisType          string                   `json:"analysis_type"`
	StartedAt             string                   `json:"started_at"`
	CompletedAt           string                   `json:"completed_at"`
	Statistics            PatternStatistics        `json:"statistics"`
	IssuesByType          map[string][]PatternIssue `json:"issues_by_type"`
	Recommendations       []Recommendation         `json:"recommendations"`
	DeepAnalysisPerformed bool                     `json:"deep_analysis_performed"`
	Error                 *TidinessError           `json:"error,omitempty"`
}

type PatternStatistics struct {
	TotalPatterns    int            `json:"total_patterns"`
	TotalIssues      int            `json:"total_issues"`
	IssuesBySeverity map[string]int `json:"issues_by_severity"`
	IssueTypes       int            `json:"issue_types"`
}

type PatternIssue struct {
	Type                 string  `json:"type"`
	Severity             string  `json:"severity"`
	File                 string  `json:"file,omitempty"`
	Line                 string  `json:"line,omitempty"`
	Description          string  `json:"description"`
	RequiresHumanReview  bool    `json:"requires_human_review"`
	Confidence           float64 `json:"confidence"`
}

type Recommendation struct {
	Priority string `json:"priority"`
	Action   string `json:"action"`
	Impact   string `json:"impact"`
}

// Cleanup Executor types (replaces cleanup-executor workflow)
type CleanupExecutionRequest struct {
	SuggestionIDs  []string `json:"suggestion_ids,omitempty"`
	CleanupScripts []string `json:"cleanup_scripts,omitempty"`
	DryRun         bool     `json:"dry_run,omitempty"`
	Force          bool     `json:"force,omitempty"`
}

type CleanupExecutionResponse struct {
	Success     bool               `json:"success"`
	ExecutionID string             `json:"execution_id"`
	DryRun      bool               `json:"dry_run"`
	StartedAt   string             `json:"started_at"`
	CompletedAt string             `json:"completed_at"`
	Statistics  CleanupStatistics  `json:"statistics"`
	Results     []CleanupResult    `json:"results"`
	Errors      []CleanupError     `json:"errors"`
	Error       *TidinessError     `json:"error,omitempty"`
}

type CleanupStatistics struct {
	TotalExecuted  int `json:"total_executed"`
	Successful     int `json:"successful"`
	Failed         int `json:"failed"`
	FilesDeleted   int `json:"files_deleted"`
	FilesModified  int `json:"files_modified"`
	BytesFreed     int `json:"bytes_freed"`
}

type CleanupResult struct {
	Script        string   `json:"script"`
	Success       bool     `json:"success"`
	Output        string   `json:"output"`
	FilesAffected []string `json:"files_affected"`
}

type CleanupError struct {
	Script string `json:"script"`
	Error  string `json:"error"`
}

type TidinessError struct {
	Message   string `json:"message"`
	Type      string `json:"type"`
	Timestamp string `json:"timestamp"`
}

func NewTidinessProcessor(db *sql.DB) *TidinessProcessor {
	return &TidinessProcessor{
		db: db,
	}
}

// ScanCode performs comprehensive code scanning (replaces code-scanner workflow)
func (tp *TidinessProcessor) ScanCode(ctx context.Context, req CodeScanRequest) (*CodeScanResponse, error) {
	startTime := time.Now()
	scanID := fmt.Sprintf("scan_%d_%s", startTime.Unix(), generateRandomID(6))

	// Set defaults
	req = tp.setScanDefaults(req)

	// Validate request
	if err := tp.validateScanRequest(req); err != nil {
		return &CodeScanResponse{
			Success: false,
			ScanID:  scanID,
			Error: &TidinessError{
				Message:   err.Error(),
				Type:      "validation_error",
				Timestamp: time.Now().Format(time.RFC3339),
			},
		}, nil
	}

	// Build scan commands
	commands := tp.buildScanCommands(req)

	// Execute scans
	allIssues := []ScanIssue{}
	resultsByType := make(map[string]TypeResult)

	for _, cmd := range commands {
		issues, err := tp.executeScanning(ctx, cmd, req)
		if err != nil {
			log.Printf("Warning: Scan command failed: %v", err)
			continue
		}

		allIssues = append(allIssues, issues...)
		resultsByType[cmd.Type] = TypeResult{
			Count:    len(issues),
			Patterns: cmd.Patterns,
		}
	}

	// Calculate statistics
	stats := tp.calculateScanStatistics(allIssues)

	// Store results in database
	err := tp.storeScanResults(ctx, scanID, req, allIssues)
	if err != nil {
		log.Printf("Warning: Failed to store scan results: %v", err)
	}

	endTime := time.Now()

	return &CodeScanResponse{
		Success:       true,
		ScanID:        scanID,
		ScanConfig:    req,
		StartedAt:     startTime.Format(time.RFC3339),
		CompletedAt:   endTime.Format(time.RFC3339),
		Statistics:    stats,
		ResultsByType: resultsByType,
		Issues:        limitIssues(allIssues, 100), // Limit for response
		TotalIssues:   len(allIssues),
	}, nil
}

// AnalyzePatterns performs pattern analysis on codebase (replaces pattern-analyzer workflow)
func (tp *TidinessProcessor) AnalyzePatterns(ctx context.Context, req PatternAnalysisRequest) (*PatternAnalysisResponse, error) {
	startTime := time.Now()
	analysisID := fmt.Sprintf("analysis_%d_%s", startTime.Unix(), generateRandomID(6))

	// Set defaults
	req = tp.setPatternDefaults(req)

	// Build analysis commands
	commands, err := tp.buildPatternCommands(req)
	if err != nil {
		return &PatternAnalysisResponse{
			Success: false,
			Error: &TidinessError{
				Message:   err.Error(),
				Type:      "command_error",
				Timestamp: time.Now().Format(time.RFC3339),
			},
		}, nil
	}

	// Execute pattern analysis
	allIssues := []PatternIssue{}
	issuesByType := make(map[string][]PatternIssue)

	for _, cmd := range commands {
		issues, err := tp.executePatternAnalysis(ctx, cmd, req)
		if err != nil {
			log.Printf("Warning: Pattern analysis command failed: %v", err)
			continue
		}

		allIssues = append(allIssues, issues...)
		for _, issue := range issues {
			if _, exists := issuesByType[issue.Type]; !exists {
				issuesByType[issue.Type] = []PatternIssue{}
			}
			issuesByType[issue.Type] = append(issuesByType[issue.Type], issue)
		}
	}

	// Calculate statistics
	stats := tp.calculatePatternStatistics(allIssues)

	// Generate recommendations
	recommendations := tp.generateRecommendations(req.AnalysisType, stats)

	// Store results
	err = tp.storePatternResults(ctx, analysisID, req, allIssues)
	if err != nil {
		log.Printf("Warning: Failed to store pattern results: %v", err)
	}

	endTime := time.Now()

	return &PatternAnalysisResponse{
		Success:               true,
		AnalysisID:           analysisID,
		AnalysisType:         req.AnalysisType,
		StartedAt:            startTime.Format(time.RFC3339),
		CompletedAt:          endTime.Format(time.RFC3339),
		Statistics:           stats,
		IssuesByType:         issuesByType,
		Recommendations:      recommendations,
		DeepAnalysisPerformed: req.DeepAnalysis,
	}, nil
}

// ExecuteCleanup safely executes cleanup operations (replaces cleanup-executor workflow)
func (tp *TidinessProcessor) ExecuteCleanup(ctx context.Context, req CleanupExecutionRequest) (*CleanupExecutionResponse, error) {
	startTime := time.Now()
	executionID := fmt.Sprintf("cleanup_%d_%s", startTime.Unix(), generateRandomID(6))

	// Validate request
	if len(req.SuggestionIDs) == 0 && len(req.CleanupScripts) == 0 {
		return &CleanupExecutionResponse{
			Success: false,
			Error: &TidinessError{
				Message:   "No cleanup scripts or suggestion IDs provided",
				Type:      "validation_error",
				Timestamp: time.Now().Format(time.RFC3339),
			},
		}, nil
	}

	// Prepare safe scripts
	safeScripts, err := tp.prepareSafeScripts(req.CleanupScripts, req.DryRun)
	if err != nil {
		return &CleanupExecutionResponse{
			Success: false,
			Error: &TidinessError{
				Message:   err.Error(),
				Type:      "safety_error",
				Timestamp: time.Now().Format(time.RFC3339),
			},
		}, nil
	}

	// Execute cleanup scripts
	results := []CleanupResult{}
	errors := []CleanupError{}
	stats := CleanupStatistics{}

	for _, script := range safeScripts {
		result, err := tp.executeCleanupScript(ctx, script, req.DryRun)
		stats.TotalExecuted++

		if err != nil {
			stats.Failed++
			errors = append(errors, CleanupError{
				Script: script.Original,
				Error:  err.Error(),
			})
		} else {
			if result.Success {
				stats.Successful++
			} else {
				stats.Failed++
			}
			stats.FilesDeleted += len(result.FilesAffected)
			results = append(results, *result)
		}
	}

	// Store execution results
	err = tp.storeCleanupResults(ctx, executionID, req, results)
	if err != nil {
		log.Printf("Warning: Failed to store cleanup results: %v", err)
	}

	endTime := time.Now()

	return &CleanupExecutionResponse{
		Success:     stats.Failed == 0,
		ExecutionID: executionID,
		DryRun:      req.DryRun,
		StartedAt:   startTime.Format(time.RFC3339),
		CompletedAt: endTime.Format(time.RFC3339),
		Statistics:  stats,
		Results:     results,
		Errors:      errors,
	}, nil
}

// Helper methods

func (tp *TidinessProcessor) setScanDefaults(req CodeScanRequest) CodeScanRequest {
	if len(req.Paths) == 0 {
		req.Paths = []string{"/home/matthalloran8/Vrooli/"}
	}
	if len(req.Types) == 0 {
		req.Types = []string{"backup_files", "temp_files"}
	}
	if req.Limit == 0 {
		req.Limit = 1000
	}
	if len(req.ExcludePatterns) == 0 {
		req.ExcludePatterns = []string{"node_modules", ".git", "dist", "build"}
	}
	return req
}

func (tp *TidinessProcessor) validateScanRequest(req CodeScanRequest) error {
	for _, path := range req.Paths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return fmt.Errorf("path does not exist: %s", path)
		}
	}
	
	validTypes := []string{"backup_files", "temp_files", "empty_dirs", "large_files"}
	for _, scanType := range req.Types {
		if !contains(validTypes, scanType) {
			return fmt.Errorf("invalid scan type: %s", scanType)
		}
	}
	
	return nil
}

type ScanCommand struct {
	Type     string
	Patterns []string
	Command  string
}

func (tp *TidinessProcessor) buildScanCommands(req CodeScanRequest) []ScanCommand {
	commands := []ScanCommand{}
	pathsStr := strings.Join(req.Paths, " ")

	for _, scanType := range req.Types {
		switch scanType {
		case "backup_files":
			commands = append(commands, ScanCommand{
				Type:     "backup_files",
				Patterns: []string{"*.bak", "*~", "*.orig", "*.old"},
				Command:  fmt.Sprintf(`find %s -type f \( -name "*.bak" -o -name "*~" -o -name "*.orig" -o -name "*.old" \) 2>/dev/null | head -%d`, pathsStr, req.Limit),
			})
		case "temp_files":
			commands = append(commands, ScanCommand{
				Type:     "temp_files",
				Patterns: []string{".*.swp", ".DS_Store", "Thumbs.db", "npm-debug.log*"},
				Command:  fmt.Sprintf(`find %s -type f \( -name ".*.swp" -o -name ".DS_Store" -o -name "Thumbs.db" -o -name "npm-debug.log*" \) 2>/dev/null | head -%d`, pathsStr, req.Limit),
			})
		case "empty_dirs":
			commands = append(commands, ScanCommand{
				Type:     "empty_dirs",
				Patterns: []string{"empty directories"},
				Command:  fmt.Sprintf(`find %s -type d -empty 2>/dev/null | head -%d`, pathsStr, req.Limit),
			})
		case "large_files":
			commands = append(commands, ScanCommand{
				Type:     "large_files",
				Patterns: []string{"files > 10MB"},
				Command:  fmt.Sprintf(`find %s -type f -size +10M 2>/dev/null | head -%d`, pathsStr, req.Limit),
			})
		}
	}

	return commands
}

func (tp *TidinessProcessor) executeScanning(ctx context.Context, cmd ScanCommand, req CodeScanRequest) ([]ScanIssue, error) {
	// Execute the find command
	output, err := tp.executeCommand(ctx, cmd.Command)
	if err != nil {
		return nil, err
	}

	files := strings.Split(strings.TrimSpace(output), "\n")
	issues := []ScanIssue{}

	for _, filePath := range files {
		if filePath == "" {
			continue
		}

		// Skip excluded patterns
		if tp.shouldExclude(filePath, req.ExcludePatterns) {
			continue
		}

		issue := tp.createScanIssue(filePath, cmd.Type)
		issues = append(issues, issue)
	}

	return issues, nil
}

func (tp *TidinessProcessor) createScanIssue(filePath, issueType string) ScanIssue {
	severity := "low"
	confidence := 0.95
	safeToAutomate := true
	cleanupScript := ""

	// Determine safety and severity based on path and type
	if strings.Contains(filePath, ".git") || strings.Contains(filePath, "node_modules") {
		severity = "low"
		confidence = 0.80
	} else if strings.HasSuffix(filePath, ".swp") || strings.HasSuffix(filePath, ".DS_Store") {
		severity = "low"
		confidence = 0.99
		safeToAutomate = true
		cleanupScript = fmt.Sprintf(`rm -f "%s"`, filePath)
	} else if strings.HasSuffix(filePath, ".bak") || strings.HasSuffix(filePath, "~") {
		severity = "medium"
		confidence = 0.90
		cleanupScript = fmt.Sprintf(`rm -f "%s"`, filePath)
	} else if issueType == "large_files" {
		severity = "high"
		confidence = 0.70
		safeToAutomate = false
	}

	return ScanIssue{
		FilePath:        filePath,
		IssueType:       issueType,
		Severity:        severity,
		ConfidenceScore: confidence,
		SafeToAutomate:  safeToAutomate,
		CleanupScript:   cleanupScript,
		Description:     fmt.Sprintf("Found %s: %s", strings.ReplaceAll(issueType, "_", " "), filePath),
	}
}

func (tp *TidinessProcessor) calculateScanStatistics(issues []ScanIssue) ScanStatistics {
	stats := ScanStatistics{
		Total:      len(issues),
		BySeverity: make(map[string]int),
	}

	for _, issue := range issues {
		stats.BySeverity[issue.Severity]++
		if issue.SafeToAutomate {
			stats.Automatable++
		} else {
			stats.ManualReview++
		}
	}

	return stats
}

func (tp *TidinessProcessor) setPatternDefaults(req PatternAnalysisRequest) PatternAnalysisRequest {
	if req.AnalysisType == "" {
		req.AnalysisType = "duplicate_detection"
	}
	if len(req.Paths) == 0 {
		req.Paths = []string{"/home/matthalloran8/Vrooli/"}
	}
	return req
}

type PatternCommand struct {
	Name    string
	Command string
}

func (tp *TidinessProcessor) buildPatternCommands(req PatternAnalysisRequest) ([]PatternCommand, error) {
	commands := []PatternCommand{}
	pathsStr := req.Paths[0]

	switch req.AnalysisType {
	case "duplicate_detection":
		commands = append(commands, PatternCommand{
			Name:    "find_scenario_structures",
			Command: fmt.Sprintf(`find %s -maxdepth 2 -name "PRD.md" -o -name "service.json" | head -100`, pathsStr),
		})
	case "unused_imports":
		commands = append(commands, PatternCommand{
			Name:    "find_js_imports",
			Command: fmt.Sprintf(`grep -r "^import\\|^const.*require" %s --include="*.js" --include="*.ts" | head -200`, pathsStr),
		})
	case "dead_code":
		commands = append(commands, PatternCommand{
			Name:    "find_functions",
			Command: fmt.Sprintf(`grep -r "function\\|const.*=>\\|export" %s --include="*.js" --include="*.ts" | head -200`, pathsStr),
		})
	case "hardcoded_values":
		commands = append(commands, PatternCommand{
			Name:    "find_hardcoded",
			Command: fmt.Sprintf(`grep -r "localhost:[0-9]\\|127.0.0.1\\|password.*=\\|api[_-]\\?key.*=" %s --include="*.js" --include="*.ts" --include="*.json" | head -100`, pathsStr),
		})
	case "todo_comments":
		commands = append(commands, PatternCommand{
			Name:    "find_todos",
			Command: fmt.Sprintf(`grep -r "TODO\\|FIXME\\|HACK\\|XXX" %s --include="*.js" --include="*.ts" --include="*.go" | head -200`, pathsStr),
		})
	default:
		return nil, fmt.Errorf("unknown analysis type: %s", req.AnalysisType)
	}

	return commands, nil
}

func (tp *TidinessProcessor) executePatternAnalysis(ctx context.Context, cmd PatternCommand, req PatternAnalysisRequest) ([]PatternIssue, error) {
	output, err := tp.executeCommand(ctx, cmd.Command)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	issues := []PatternIssue{}

	switch req.AnalysisType {
	case "duplicate_detection":
		issues = tp.analyzeDuplicates(lines)
	case "hardcoded_values":
		issues = tp.analyzeHardcodedValues(lines)
	case "todo_comments":
		issues = tp.analyzeTodoComments(lines)
	case "unused_imports":
		issues = tp.analyzeUnusedImports(lines)
	}

	return issues, nil
}

func (tp *TidinessProcessor) analyzeDuplicates(lines []string) []PatternIssue {
	scenarios := make(map[string][]string)
	issues := []PatternIssue{}

	// Group scenarios
	for _, line := range lines {
		if line == "" {
			continue
		}
		scenarioMatch := regexp.MustCompile(`scenarios/([^/]+)`).FindStringSubmatch(line)
		if len(scenarioMatch) > 1 {
			scenario := scenarioMatch[1]
			if _, exists := scenarios[scenario]; !exists {
				scenarios[scenario] = []string{}
			}
			scenarios[scenario] = append(scenarios[scenario], line)
		}
	}

	// Find similar scenarios
	scenarioKeys := make([]string, 0, len(scenarios))
	for k := range scenarios {
		scenarioKeys = append(scenarioKeys, k)
	}

	for i := 0; i < len(scenarioKeys)-1; i++ {
		for j := i + 1; j < len(scenarioKeys); j++ {
			similarity := calculateSimilarity(scenarioKeys[i], scenarioKeys[j])
			if similarity > 0.7 {
				issues = append(issues, PatternIssue{
					Type:                "duplicate_scenario",
					Severity:            "medium",
					Description:         fmt.Sprintf("Potential duplicate functionality: %s and %s", scenarioKeys[i], scenarioKeys[j]),
					RequiresHumanReview: true,
					Confidence:          similarity,
				})
			}
		}
	}

	return issues
}

func (tp *TidinessProcessor) analyzeHardcodedValues(lines []string) []PatternIssue {
	issues := []PatternIssue{}

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) < 2 {
			continue
		}

		file := parts[0]
		content := parts[1]

		issueType := "hardcoded_value"
		severity := "low"

		if strings.Contains(content, "password") || strings.Contains(content, "api_key") || strings.Contains(content, "secret") {
			issueType = "potential_credential"
			severity = "critical"
		} else if strings.Contains(content, "localhost") || strings.Contains(content, "127.0.0.1") {
			issueType = "hardcoded_url"
			severity = "medium"
		}

		issues = append(issues, PatternIssue{
			Type:                issueType,
			Severity:            severity,
			File:                file,
			Description:         fmt.Sprintf("Hardcoded value detected: %s", truncateString(content, 100)),
			RequiresHumanReview: true,
			Confidence:          0.8,
		})
	}

	return issues
}

func (tp *TidinessProcessor) analyzeTodoComments(lines []string) []PatternIssue {
	issues := []PatternIssue{}

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 3)
		if len(parts) < 3 {
			continue
		}

		file := parts[0]
		lineNum := parts[1]
		comment := strings.TrimSpace(parts[2])

		severity := "low"
		if strings.Contains(comment, "FIXME") || strings.Contains(comment, "HACK") {
			severity = "medium"
		}

		issues = append(issues, PatternIssue{
			Type:                "todo_comment",
			Severity:            severity,
			File:                file,
			Line:                lineNum,
			Description:         truncateString(comment, 200),
			RequiresHumanReview: false,
			Confidence:          1.0,
		})
	}

	return issues
}

func (tp *TidinessProcessor) analyzeUnusedImports(lines []string) []PatternIssue {
	issues := []PatternIssue{}

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) < 2 {
			continue
		}

		file := parts[0]
		importLine := parts[1]

		importMatch := regexp.MustCompile(`import\s+([^\s]+)`).FindStringSubmatch(importLine)
		if len(importMatch) > 1 {
			issues = append(issues, PatternIssue{
				Type:                "potential_unused_import",
				Severity:            "low",
				File:                file,
				Description:         fmt.Sprintf("Potential unused import: %s", importMatch[1]),
				RequiresHumanReview: true,
				Confidence:          0.6,
			})
		}
	}

	return issues
}

func (tp *TidinessProcessor) calculatePatternStatistics(issues []PatternIssue) PatternStatistics {
	stats := PatternStatistics{
		TotalIssues:      len(issues),
		IssuesBySeverity: make(map[string]int),
	}

	issueTypes := make(map[string]bool)
	for _, issue := range issues {
		stats.IssuesBySeverity[issue.Severity]++
		issueTypes[issue.Type] = true
	}

	stats.IssueTypes = len(issueTypes)
	return stats
}

func (tp *TidinessProcessor) generateRecommendations(analysisType string, stats PatternStatistics) []Recommendation {
	recommendations := []Recommendation{}

	if stats.IssuesBySeverity["critical"] > 0 {
		recommendations = append(recommendations, Recommendation{
			Priority: "urgent",
			Action:   "Review critical issues immediately, especially potential credentials",
			Impact:   "Security risk mitigation",
		})
	}

	if analysisType == "duplicate_detection" && stats.TotalIssues > 0 {
		recommendations = append(recommendations, Recommendation{
			Priority: "high",
			Action:   "Consider merging duplicate scenarios or extracting shared functionality",
			Impact:   "Reduced maintenance burden and improved consistency",
		})
	}

	if analysisType == "todo_comments" && stats.TotalIssues > 10 {
		recommendations = append(recommendations, Recommendation{
			Priority: "medium",
			Action:   "Schedule technical debt sprint to address TODO items",
			Impact:   "Improved code quality and reduced future bugs",
		})
	}

	return recommendations
}

type SafeScript struct {
	Original string
	Safe     string
	DryRun   bool
}

func (tp *TidinessProcessor) prepareSafeScripts(scripts []string, dryRun bool) ([]SafeScript, error) {
	safeScripts := []SafeScript{}

	for _, script := range scripts {
		// Prevent dangerous operations
		if strings.Contains(script, "rm -rf /") || strings.Contains(script, "rm -fr /") {
			return nil, fmt.Errorf("dangerous cleanup script detected: %s", script)
		}

		safeScript := script
		if dryRun {
			safeScript = fmt.Sprintf(`echo "[DRY RUN] Would execute: %s"`, strings.ReplaceAll(script, `"`, `\"`))
		}

		safeScripts = append(safeScripts, SafeScript{
			Original: script,
			Safe:     safeScript,
			DryRun:   dryRun,
		})
	}

	return safeScripts, nil
}

func (tp *TidinessProcessor) executeCleanupScript(ctx context.Context, script SafeScript, dryRun bool) (*CleanupResult, error) {
	output, err := tp.executeCommand(ctx, script.Safe)
	
	filesAffected := []string{}
	if !dryRun && err == nil {
		// Try to extract file information from output
		fileMatches := regexp.MustCompile(`removed?[:\s]+['"]?([^'"]+)['"]?`).FindAllStringSubmatch(output, -1)
		for _, match := range fileMatches {
			if len(match) > 1 {
				filesAffected = append(filesAffected, match[1])
			}
		}
	}

	return &CleanupResult{
		Script:        script.Original,
		Success:       err == nil,
		Output:        output,
		FilesAffected: filesAffected,
	}, err
}

func (tp *TidinessProcessor) executeCommand(ctx context.Context, command string) (string, error) {
	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	output, err := cmd.Output()
	return string(output), err
}

// Database storage methods

func (tp *TidinessProcessor) storeScanResults(ctx context.Context, scanID string, req CodeScanRequest, issues []ScanIssue) error {
	if tp.db == nil {
		return nil
	}

	configJSON, _ := json.Marshal(req)
	issuesJSON, _ := json.Marshal(issues)

	query := `
		INSERT INTO tidiness_scans (
			id, scan_type, configuration, results, created_at
		) VALUES ($1, $2, $3, $4, NOW())`

	_, err := tp.db.ExecContext(ctx, query, scanID, "code_scan", string(configJSON), string(issuesJSON))
	return err
}

func (tp *TidinessProcessor) storePatternResults(ctx context.Context, analysisID string, req PatternAnalysisRequest, issues []PatternIssue) error {
	if tp.db == nil {
		return nil
	}

	configJSON, _ := json.Marshal(req)
	issuesJSON, _ := json.Marshal(issues)

	query := `
		INSERT INTO tidiness_scans (
			id, scan_type, configuration, results, created_at
		) VALUES ($1, $2, $3, $4, NOW())`

	_, err := tp.db.ExecContext(ctx, query, analysisID, "pattern_analysis", string(configJSON), string(issuesJSON))
	return err
}

func (tp *TidinessProcessor) storeCleanupResults(ctx context.Context, executionID string, req CleanupExecutionRequest, results []CleanupResult) error {
	if tp.db == nil {
		return nil
	}

	configJSON, _ := json.Marshal(req)
	resultsJSON, _ := json.Marshal(results)

	query := `
		INSERT INTO tidiness_scans (
			id, scan_type, configuration, results, created_at
		) VALUES ($1, $2, $3, $4, NOW())`

	_, err := tp.db.ExecContext(ctx, query, executionID, "cleanup_execution", string(configJSON), string(resultsJSON))
	return err
}

// Utility functions

func limitIssues(issues []ScanIssue, limit int) []ScanIssue {
	if len(issues) <= limit {
		return issues
	}
	return issues[:limit]
}

func (tp *TidinessProcessor) shouldExclude(filePath string, excludePatterns []string) bool {
	for _, pattern := range excludePatterns {
		if strings.Contains(filePath, pattern) {
			return true
		}
	}
	return false
}

func calculateSimilarity(str1, str2 string) float64 {
	longer := str1
	shorter := str2
	if len(str2) > len(str1) {
		longer = str2
		shorter = str1
	}

	if len(longer) == 0 {
		return 1.0
	}

	editDistance := getEditDistance(longer, shorter)
	return float64(len(longer)-editDistance) / float64(len(longer))
}

func getEditDistance(s1, s2 string) int {
	costs := make([]int, len(s2)+1)
	for j := 0; j <= len(s2); j++ {
		costs[j] = j
	}

	for i := 1; i <= len(s1); i++ {
		costs[0] = i
		nw := i - 1
		for j := 1; j <= len(s2); j++ {
			cj := min(
				1+min(costs[j], costs[j-1]),
				nw+boolToInt(s1[i-1] != s2[j-1]))
			nw = costs[j]
			costs[j] = cj
		}
	}
	return costs[len(s2)]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func generateRandomID(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(result)
}