package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ScenarioLocator centralizes scenario path resolution and discovery so handlers
// don't reinvent path validation or caching logic.
type ScenarioLocator struct {
	vrooliRoot   string
	scenariosDir string
	cache        []string
	cacheTime    time.Time
	cacheTTL     time.Duration
}

// NewScenarioLocator builds a locator with sensible defaults derived from env.
func NewScenarioLocator(cacheTTL time.Duration) *ScenarioLocator {
	root := os.Getenv("VROOLI_ROOT")
	if root == "" {
		root = filepath.Join(os.Getenv("HOME"), "Vrooli")
	}

	return &ScenarioLocator{
		vrooliRoot:   root,
		scenariosDir: filepath.Join(root, "scenarios"),
		cacheTTL:     cacheTTL,
	}
}

// ScenarioPath returns the absolute path for a scenario name within Vrooli.
func (sl *ScenarioLocator) ScenarioPath(name string) string {
	return filepath.Join(sl.scenariosDir, name)
}

// ValidateScenarioName ensures the scenario name cannot escape the scenarios directory.
func (sl *ScenarioLocator) ValidateScenarioName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", errors.New("scenario is required")
	}

	if strings.Contains(trimmed, "..") || strings.ContainsAny(trimmed, `/\`) {
		return "", fmt.Errorf("invalid scenario name")
	}

	return trimmed, nil
}

// defaultScenarioPath provides a fallback scenario path based on environment variables.
func defaultScenarioPath(scenario string) string {
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		vrooliRoot = filepath.Join(os.Getenv("HOME"), "Vrooli")
	}
	return filepath.Join(vrooliRoot, "scenarios", scenario)
}

// ResolveRequestedPath validates and normalizes a requested scenario path.
func (sl *ScenarioLocator) ResolveRequestedPath(requested string) (string, string, error) {
	if strings.TrimSpace(requested) == "" {
		return "", "", errors.New("scenario_path is required")
	}

	absPath, err := filepath.Abs(requested)
	if err != nil {
		return "", "", fmt.Errorf("invalid scenario_path")
	}

	// Prefer paths within the scenarios directory, but allow temp directories used in tests.
	return absPath, filepath.Base(absPath), nil
}

// List returns available scenarios, caching results briefly to avoid repeated CLI calls.
func (sl *ScenarioLocator) List(ctx context.Context) ([]string, error) {
	if len(sl.cache) > 0 && time.Since(sl.cacheTime) < sl.cacheTTL {
		return sl.cache, nil
	}

	callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(callCtx, "vrooli", "scenario", "status", "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var resp struct {
		Scenarios []map[string]interface{} `json:"scenarios"`
	}
	if err := json.Unmarshal(output, &resp); err != nil {
		return nil, err
	}

	scenarios := []string{}
	for _, scenario := range resp.Scenarios {
		if name, ok := scenario["name"].(string); ok {
			scenarios = append(scenarios, name)
		}
	}

	sl.cache = scenarios
	sl.cacheTime = time.Now()
	return scenarios, nil
}

// ScanError carries HTTP-friendly status and message for scan orchestration errors.
type ScanError struct {
	Status  int
	Message string
	Err     error
}

func (e *ScanError) Error() string {
	return e.Message
}

// ScanCoordinator encapsulates light and smart scan orchestration so HTTP handlers
// only translate requests and responses.
type ScanCoordinator struct {
	db                    *sql.DB
	scenarios             *ScenarioLocator
	logFn                 func(string, map[string]interface{})
	persistDetailed       func(context.Context, string, []DetailedFileMetrics) error
	persistBasic          func(context.Context, string, []FileMetric) error
	persistLintTypeIssues func(context.Context, string, []Issue) (int, error)
	storeIssue            func(context.Context, string, AIIssue, string, *int) error
	recordHistory         func(context.Context, string, string, *SmartScanResult, *int) error
}

// NewScanCoordinator wires dependencies required for scan flows.
func NewScanCoordinator(
	db *sql.DB,
	scenarios *ScenarioLocator,
	logFn func(string, map[string]interface{}),
	persistDetailed func(context.Context, string, []DetailedFileMetrics) error,
	persistBasic func(context.Context, string, []FileMetric) error,
	persistLintTypeIssues func(context.Context, string, []Issue) (int, error),
	storeIssue func(context.Context, string, AIIssue, string, *int) error,
	recordHistory func(context.Context, string, string, *SmartScanResult, *int) error,
) *ScanCoordinator {
	return &ScanCoordinator{
		db:                    db,
		scenarios:             scenarios,
		logFn:                 logFn,
		persistDetailed:       persistDetailed,
		persistBasic:          persistBasic,
		persistLintTypeIssues: persistLintTypeIssues,
		storeIssue:            storeIssue,
		recordHistory:         recordHistory,
	}
}

// LightScan runs a full light scan, including persistence of metrics.
func (sc *ScanCoordinator) LightScan(ctx context.Context, req LightScanRequest) (*ScanResult, *ScanError) {
	absPath, scenarioName, err := sc.scenarios.ResolveRequestedPath(req.ScenarioPath)
	if err != nil {
		return nil, &ScanError{Status: http.StatusBadRequest, Message: err.Error(), Err: err}
	}

	if _, statErr := os.Stat(absPath); os.IsNotExist(statErr) {
		return nil, &ScanError{Status: http.StatusNotFound, Message: "scenario path does not exist", Err: statErr}
	}

	// Security: Limit timeout to prevent resource exhaustion (max 10 minutes)
	const maxTimeout = 600
	timeout := 120 * time.Second
	if req.TimeoutSec > 0 {
		if req.TimeoutSec > maxTimeout {
			return nil, &ScanError{
				Status:  http.StatusBadRequest,
				Message: fmt.Sprintf("timeout_sec cannot exceed %d seconds", maxTimeout),
				Err:     fmt.Errorf("timeout exceeded"),
			}
		}
		timeout = time.Duration(req.TimeoutSec) * time.Second
	}

	// Auto-enable incremental scans when recent results exist
	incremental := req.Incremental
	if !incremental {
		hasRecent, _ := sc.hasRecentScan(ctx, scenarioName, 24*time.Hour)
		if hasRecent {
			incremental = true
			sc.log("auto-enabled incremental scan", map[string]interface{}{"scenario": scenarioName})
		}
	}

	scanner := NewLightScanner(absPath, timeout)
	scanOpts := ScanOptions{
		Incremental: incremental,
		DB:          sc.db,
	}

	result, err := scanner.ScanWithOptions(ctx, scanOpts)
	if err != nil {
		sc.log("light scan failed", map[string]interface{}{
			"error":       err.Error(),
			"scenario":    absPath,
			"incremental": req.Incremental,
		})
		return nil, &ScanError{
			Status:  http.StatusInternalServerError,
			Message: "scan failed - check server logs for details",
			Err:     err,
		}
	}

	// Parse and persist lint/type issues (OT-P0-001, OT-P0-007, OT-P0-008)
	var lintIssueCount, typeIssueCount int

	if sc.persistLintTypeIssues != nil {
		var allIssues []Issue

		// Parse lint output if present
		if result.LintOutput != nil && result.LintOutput.Stdout != "" {
			lintIssues := ParseLintOutput(result.Scenario, "lint", result.LintOutput.Stdout)
			lintIssueCount = len(lintIssues)
			allIssues = append(allIssues, lintIssues...)
		}

		// Parse type output if present
		if result.TypeOutput != nil && result.TypeOutput.Stdout != "" {
			typeIssues := ParseTypeOutput(result.Scenario, "type", result.TypeOutput.Stdout)
			typeIssueCount = len(typeIssues)
			allIssues = append(allIssues, typeIssues...)
		}

		if len(allIssues) > 0 {
			inserted, persistErr := sc.persistLintTypeIssues(ctx, result.Scenario, allIssues)
			if persistErr != nil {
				sc.log("failed to persist lint/type issues", map[string]interface{}{
					"error":    persistErr.Error(),
					"scenario": result.Scenario,
					"count":    len(allIssues),
				})
			} else {
				sc.log("persisted lint/type issues", map[string]interface{}{
					"scenario": result.Scenario,
					"inserted": inserted,
					"total":    len(allIssues),
				})
			}
		}
	}

	// Populate convenience counts for CLI consumption
	result.LintIssuesCount = lintIssueCount
	result.TypeIssuesCount = typeIssueCount
	result.LongFilesCount = len(result.LongFiles)

	// Collect and persist detailed file metrics to database (TM-FM-001, TM-FM-002)
	scenarioPath := sc.scenarios.ScenarioPath(result.Scenario)

	filePaths := make([]string, len(result.FileMetrics))
	for i, fm := range result.FileMetrics {
		filePaths[i] = fm.Path
	}

	var detailedMetrics []DetailedFileMetrics

	if len(filePaths) > 0 {
		// Pass language metrics for complexity/duplication data
		var metricsErr error
		detailedMetrics, metricsErr = CollectDetailedFileMetricsWithLangMetrics(scenarioPath, filePaths, result.LanguageMetrics)
		if metricsErr != nil {
			sc.log("failed to collect detailed metrics", map[string]interface{}{
				"error":    metricsErr.Error(),
				"scenario": result.Scenario,
			})
			if sc.persistBasic != nil {
				if err := sc.persistBasic(ctx, result.Scenario, result.FileMetrics); err != nil {
					sc.log("failed to persist basic file metrics", map[string]interface{}{
						"error":    err.Error(),
						"scenario": result.Scenario,
					})
				}
			}
			// Don't return early - try to generate issues from existing DB metrics
		} else if sc.persistDetailed != nil {
			if err := sc.persistDetailed(ctx, result.Scenario, detailedMetrics); err != nil {
				sc.log("failed to persist detailed file metrics", map[string]interface{}{
					"error":    err.Error(),
					"scenario": result.Scenario,
				})
			}
		}
	}

	// Generate and persist issues from metrics (length, complexity, duplication, technical_debt, coupling)
	// If no new files were scanned (incremental scan skipped all), fetch metrics from DB
	if sc.persistLintTypeIssues != nil {
		metricsForIssueGen := detailedMetrics

		// If we don't have freshly collected metrics, try to get them from the database
		if len(metricsForIssueGen) == 0 && sc.db != nil {
			dbMetrics, dbErr := sc.getMetricsFromDB(ctx, scenarioName)
			if dbErr != nil {
				sc.log("failed to get metrics from DB for issue generation", map[string]interface{}{
					"error":    dbErr.Error(),
					"scenario": result.Scenario,
				})
			} else {
				metricsForIssueGen = dbMetrics
			}
		}

		if len(metricsForIssueGen) > 0 {
			config := DefaultIssueGeneratorConfig()
			metricIssues := GenerateIssuesFromMetrics(result.Scenario, metricsForIssueGen, config)
			if len(metricIssues) > 0 {
				inserted, persistErr := sc.persistLintTypeIssues(ctx, result.Scenario, metricIssues)
				if persistErr != nil {
					sc.log("failed to persist metric-based issues", map[string]interface{}{
						"error":    persistErr.Error(),
						"scenario": result.Scenario,
						"count":    len(metricIssues),
					})
				} else {
					sc.log("persisted metric-based issues", map[string]interface{}{
						"scenario": result.Scenario,
						"inserted": inserted,
						"total":    len(metricIssues),
					})
				}
			}
		}
	}

	return result, nil
}

// SmartScan orchestrates the AI-powered smart scan workflow plus persistence.
func (sc *ScanCoordinator) SmartScan(ctx context.Context, req SmartScanRequest) (*SmartScanResult, *ScanError) {
	scenarioName, err := sc.validateScenarioName(req.Scenario)
	if err != nil {
		return nil, &ScanError{Status: http.StatusBadRequest, Message: err.Error(), Err: err}
	}

	if len(req.Files) == 0 {
		return nil, &ScanError{Status: http.StatusBadRequest, Message: "files list cannot be empty", Err: fmt.Errorf("missing files")}
	}

	scanner, scanErr := sc.createSmartScanner()
	if scanErr != nil {
		return nil, scanErr
	}

	normalizedReq := req
	normalizedReq.Scenario = scenarioName

	result, err := scanner.ScanScenario(ctx, normalizedReq)
	if err != nil {
		sc.log("smart scan failed", map[string]interface{}{
			"error":    err.Error(),
			"scenario": scenarioName,
		})
		return nil, &ScanError{
			Status:  http.StatusInternalServerError,
			Message: "scan failed - check server logs for details",
			Err:     err,
		}
	}

	sc.persistSmartScanResults(ctx, scenarioName, result, normalizedReq.CampaignID)
	return result, nil
}

// NormalizeScenarioName validates and returns a trimmed scenario name so callers
// don't replicate validation logic.
func (sc *ScanCoordinator) NormalizeScenarioName(raw string) (string, error) {
	return sc.validateScenarioName(raw)
}

// EnsureFileMetrics guarantees file metrics exist for a scenario, triggering a
// light scan and persistence when none are stored yet. This keeps HTTP handlers
// focused on transport concerns while centralizing scan orchestration here.
func (sc *ScanCoordinator) EnsureFileMetrics(ctx context.Context, scenario string) error {
	scenarioName, err := sc.validateScenarioName(scenario)
	if err != nil {
		return err
	}

	if sc.db == nil {
		return fmt.Errorf("database not configured")
	}

	hasMetrics, err := sc.HasMetricsForScenario(ctx, scenarioName)
	if err != nil {
		return fmt.Errorf("failed to check metrics existence: %w", err)
	}
	if hasMetrics {
		return nil
	}

	if sc.scenarios == nil {
		return fmt.Errorf("scenario locator not configured")
	}

	scenarioPath := sc.scenarios.ScenarioPath(scenarioName)
	if _, err := os.Stat(scenarioPath); err != nil {
		return fmt.Errorf("scenario path does not exist: %w", err)
	}

	sc.log("auto-scan triggered for scenario with no metrics", map[string]interface{}{
		"scenario": scenarioName,
	})

	scanner := NewLightScanner(scenarioPath, 120*time.Second)
	result, err := scanner.Scan(ctx)
	if err != nil {
		return fmt.Errorf("light scan failed: %w", err)
	}

	filePaths := make([]string, len(result.FileMetrics))
	for i, fm := range result.FileMetrics {
		filePaths[i] = fm.Path
	}

	if len(filePaths) == 0 {
		return nil
	}

	// Pass language metrics for complexity/duplication data
	detailedMetrics, metricsErr := CollectDetailedFileMetricsWithLangMetrics(scenarioPath, filePaths, result.LanguageMetrics)
	if metricsErr != nil {
		sc.log("failed to collect detailed metrics", map[string]interface{}{
			"error":    metricsErr.Error(),
			"scenario": scenarioName,
		})
		if sc.persistBasic != nil {
			return sc.persistBasic(ctx, scenarioName, result.FileMetrics)
		}
		return nil
	}

	if sc.persistDetailed != nil {
		if err := sc.persistDetailed(ctx, scenarioName, detailedMetrics); err != nil {
			return err
		}
	}

	// Generate and persist issues from metrics
	if sc.persistLintTypeIssues != nil && len(detailedMetrics) > 0 {
		config := DefaultIssueGeneratorConfig()
		metricIssues := GenerateIssuesFromMetrics(scenarioName, detailedMetrics, config)
		if len(metricIssues) > 0 {
			_, _ = sc.persistLintTypeIssues(ctx, scenarioName, metricIssues)
		}
	}

	return nil
}

func (sc *ScanCoordinator) createSmartScanner() (*SmartScanner, *ScanError) {
	config := GetDefaultSmartScanConfig()
	scanner, err := NewSmartScanner(config)
	if err != nil {
		return nil, &ScanError{Status: http.StatusInternalServerError, Message: "failed to create scanner: " + err.Error(), Err: err}
	}

	if sc.scenarios != nil {
		scanner.WithScenarioLocator(sc.scenarios)
	}

	return scanner, nil
}

func (sc *ScanCoordinator) persistSmartScanResults(ctx context.Context, scenario string, result *SmartScanResult, campaignID *int) {
	if sc.storeIssue != nil {
		for _, batchResult := range result.BatchResults {
			for _, issue := range batchResult.Issues {
				if err := sc.storeIssue(ctx, scenario, issue, result.SessionID, campaignID); err != nil {
					sc.log("failed to store issue", map[string]interface{}{
						"error":    err.Error(),
						"scenario": scenario,
						"file":     issue.FilePath,
					})
				}
			}
		}
	}

	if sc.recordHistory != nil {
		if err := sc.recordHistory(ctx, scenario, "smart", result, campaignID); err != nil {
			sc.log("failed to record scan history", map[string]interface{}{
				"error":    err.Error(),
				"scenario": scenario,
			})
		}
	}
}

func (sc *ScanCoordinator) hasRecentScan(ctx context.Context, scenario string, within time.Duration) (bool, error) {
	var lastUpdate time.Time
	query := `
		SELECT MAX(updated_at)
		FROM file_metrics
		WHERE scenario = $1
	`
	err := sc.db.QueryRowContext(ctx, query, scenario).Scan(&lastUpdate)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	age := time.Since(lastUpdate)
	return age <= within, nil
}

// HasMetricsForScenario checks if file metrics exist in the database for a scenario
func (sc *ScanCoordinator) HasMetricsForScenario(ctx context.Context, scenario string) (bool, error) {
	if sc.db == nil {
		return false, fmt.Errorf("database not configured")
	}

	var count int
	query := `SELECT COUNT(*) FROM file_metrics WHERE scenario = $1`
	err := sc.db.QueryRowContext(ctx, query, scenario).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// getMetricsFromDB retrieves file metrics from the database for issue generation
func (sc *ScanCoordinator) getMetricsFromDB(ctx context.Context, scenario string) ([]DetailedFileMetrics, error) {
	if sc.db == nil {
		return nil, fmt.Errorf("database not configured")
	}

	query := `
		SELECT
			file_path, language, file_extension, line_count,
			todo_count, fixme_count, hack_count,
			import_count, function_count, code_lines, comment_lines,
			comment_to_code_ratio, has_test_file,
			complexity_avg, complexity_max, duplication_pct
		FROM file_metrics
		WHERE scenario = $1
	`

	rows, err := sc.db.QueryContext(ctx, query, scenario)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []DetailedFileMetrics
	for rows.Next() {
		var m DetailedFileMetrics
		var lang, ext sql.NullString
		var complexityAvg sql.NullFloat64
		var complexityMax sql.NullInt64
		var duplicationPct sql.NullFloat64

		err := rows.Scan(
			&m.FilePath, &lang, &ext, &m.LineCount,
			&m.TodoCount, &m.FixmeCount, &m.HackCount,
			&m.ImportCount, &m.FunctionCount, &m.CodeLines, &m.CommentLines,
			&m.CommentRatio, &m.HasTestFile,
			&complexityAvg, &complexityMax, &duplicationPct,
		)
		if err != nil {
			continue
		}

		if lang.Valid {
			m.Language = lang.String
		}
		if ext.Valid {
			m.FileExtension = ext.String
		}
		if complexityAvg.Valid {
			val := complexityAvg.Float64
			m.ComplexityAvg = &val
		}
		if complexityMax.Valid {
			val := int(complexityMax.Int64)
			m.ComplexityMax = &val
		}
		if duplicationPct.Valid {
			val := duplicationPct.Float64
			m.DuplicationPct = &val
		}

		metrics = append(metrics, m)
	}

	return metrics, nil
}

func (sc *ScanCoordinator) log(msg string, fields map[string]interface{}) {
	if sc.logFn == nil {
		return
	}
	sc.logFn(msg, fields)
}

func (sc *ScanCoordinator) validateScenarioName(raw string) (string, error) {
	if sc.scenarios != nil {
		return sc.scenarios.ValidateScenarioName(raw)
	}

	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("scenario is required")
	}
	if strings.Contains(trimmed, "..") || strings.ContainsAny(trimmed, `/\`) {
		return "", fmt.Errorf("invalid scenario name")
	}
	return trimmed, nil
}
