package main

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	repocontract "github.com/vrooli/repo-contract-go"
	facts "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
)

// LightScanner performs fast static analysis using Makefile integration
type LightScanner struct {
	scenarioPath string
	timeout      time.Duration
	excludes     []string
	codeFacts    ReductionFacts
}

// NewLightScanner creates a scanner for the specified scenario directory
func NewLightScanner(scenarioPath string, timeout time.Duration, excludes ...string) *LightScanner {
	if timeout == 0 {
		timeout = 120 * time.Second // Default 2 minutes
	}
	return &LightScanner{
		scenarioPath: scenarioPath,
		timeout:      timeout,
		excludes:     append([]string(nil), excludes...),
	}
}

func (ls *LightScanner) withCodeFacts(source ReductionFacts) *LightScanner {
	ls.codeFacts = source
	return ls
}

// ScanResult contains all outputs from a light scan
type ScanResult struct {
	Scenario        string                        `json:"scenario"`
	StartedAt       time.Time                     `json:"started_at"`
	CompletedAt     time.Time                     `json:"completed_at"`
	Duration        int64                         `json:"duration_ms"` // Duration in milliseconds
	LintOutput      *CommandRun                   `json:"lint_output,omitempty"`
	TypeOutput      *CommandRun                   `json:"type_output,omitempty"`
	FileMetrics     []FileMetric                  `json:"file_metrics"`
	LongFiles       []LongFile                    `json:"long_files"`
	TotalFiles      int                           `json:"total_files"`
	TotalLines      int                           `json:"total_lines"`
	Incremental     bool                          `json:"incremental"`
	HasMakefile     bool                          `json:"has_makefile"`
	LanguageMetrics map[Language]*LanguageMetrics `json:"language_metrics,omitempty"`
	// Convenience counts for CLI consumption
	LintIssuesCount int `json:"lint_issues"`
	TypeIssuesCount int `json:"type_issues"`
	LongFilesCount  int `json:"long_files_count"`
}

// LanguageMetrics contains comprehensive metrics for a detected language
type LanguageMetrics struct {
	Language    Language          `json:"language"`
	FileCount   int               `json:"file_count"`
	TotalLines  int               `json:"total_lines"`
	CodeMetrics *CodeMetrics      `json:"code_metrics,omitempty"`
	Complexity  *ComplexityResult `json:"complexity,omitempty"`
	Duplicates  *DuplicateResult  `json:"duplicates,omitempty"`
	facts       *facts.CodeFactsReport
}

// CommandRun captures execution details
type CommandRun struct {
	Command    string `json:"command"`
	ExitCode   int    `json:"exit_code"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	Duration   int64  `json:"duration_ms"`
	Success    bool   `json:"success"`
	Skipped    bool   `json:"skipped"`
	SkipReason string `json:"skip_reason,omitempty"`
}

// FileMetric holds per-file statistics
type FileMetric struct {
	Path                  string `json:"path"`
	Lines                 int    `json:"lines"`
	Extension             string `json:"extension"`
	Language              string `json:"language,omitempty"`
	CodeLines             int    `json:"code_lines,omitempty"`
	CommentLines          int    `json:"comment_lines,omitempty"`
	TodoCount             int    `json:"todo_count,omitempty"`
	FixmeCount            int    `json:"fixme_count,omitempty"`
	HackCount             int    `json:"hack_count,omitempty"`
	AsAnyCount            int    `json:"as_any_count,omitempty"`
	AsTypeAssertionCount  int    `json:"as_type_assertion_count,omitempty"`
	TsIgnoreCount         int    `json:"ts_ignore_count,omitempty"`
	NonNullAssertionCount int    `json:"non_null_assertion_count,omitempty"`
	FallbackImportCount   int    `json:"fallback_import_count,omitempty"`
	FallbackFunctionCount int    `json:"fallback_function_count,omitempty"`
}

// LongFile represents files exceeding threshold
type LongFile struct {
	Path      string `json:"path"`
	Lines     int    `json:"lines"`
	Threshold int    `json:"threshold"`
}

// ScanOptions configures scan behavior
type ScanOptions struct {
	Incremental bool    // Only scan files modified since last scan
	DB          *sql.DB // Database connection for incremental mode
}

// Scan runs the complete light scan pipeline
func (ls *LightScanner) Scan(ctx context.Context) (*ScanResult, error) {
	return ls.ScanWithOptions(ctx, ScanOptions{Incremental: false, DB: nil})
}

// ScanWithOptions runs the light scan with custom options
func (ls *LightScanner) ScanWithOptions(ctx context.Context, opts ScanOptions) (*ScanResult, error) {
	startTime := time.Now()

	result := &ScanResult{
		Scenario:    filepath.Base(ls.scenarioPath),
		StartedAt:   startTime,
		Incremental: opts.Incremental && opts.DB != nil,
	}

	// Check for Makefile
	makefilePath := filepath.Join(ls.scenarioPath, "Makefile")
	if _, err := os.Stat(makefilePath); err == nil {
		result.HasMakefile = true
	}

	// Run lint if available
	if result.HasMakefile {
		lintResult := ls.runMakeCommand(ctx, "lint")
		result.LintOutput = lintResult
	}

	// Run type check if available
	if result.HasMakefile {
		typeResult := ls.runMakeCommand(ctx, "type")
		result.TypeOutput = typeResult
	}

	// Collect one complete inventory. Incremental mode filters the returned file
	// metrics, while language analysis retains its whole-source semantics.
	inventory, err := ls.collectFileMetricsContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect file metrics: %w", err)
	}
	metrics := inventory
	if opts.Incremental && opts.DB != nil {
		metrics = ls.changedFileMetrics(ctx, opts.DB, inventory)
	}

	result.FileMetrics = metrics

	// Calculate totals
	totalLines := 0
	for _, m := range metrics {
		totalLines += m.Lines
	}
	result.TotalFiles = len(metrics)
	result.TotalLines = totalLines

	// Flag long files (default threshold 500 lines)
	threshold := 500
	longFiles := []LongFile{}
	for _, m := range metrics {
		if m.Lines > threshold {
			longFiles = append(longFiles, LongFile{
				Path:      m.Path,
				Lines:     m.Lines,
				Threshold: threshold,
			})
		}
	}
	result.LongFiles = longFiles

	// All metrics share this scan's filtered file inventory.
	result.LanguageMetrics = ls.collectLanguageMetrics(ctx, languagesFromFileMetrics(inventory), inventory)

	result.CompletedAt = time.Now()
	result.Duration = durationMilliseconds(result.CompletedAt.Sub(startTime))

	return result, nil
}

func durationMilliseconds(duration time.Duration) int64 {
	if duration <= 0 {
		return 0
	}
	whole := duration.Milliseconds()
	if duration%time.Millisecond != 0 {
		whole++
	}
	return whole
}

// collectLanguageMetrics detects languages and runs all available analyzers
func (ls *LightScanner) collectLanguageMetrics(ctx context.Context, languages map[Language]*LanguageInfo, inventory []FileMetric) map[Language]*LanguageMetrics {
	var report *facts.CodeFactsReport
	var factsErr error
	if needsSharedSyntaxFacts(languages) {
		factsCtx, cancel := context.WithTimeout(ctx, ls.timeout)
		report, factsErr = currentCodeFacts(factsCtx, ls.codeFacts, &facts.CodeTarget{
			Kind: facts.TargetKind_TARGET_KIND_PATH, Path: ls.scenarioPath,
			LanguageFilter: []string{"go", "typescript", "javascript"},
		})
		cancel()
	}
	return ls.collectLanguageMetricsFromFacts(languages, inventory, report, factsErr)
}

func (ls *LightScanner) collectLanguageMetricsFromFacts(languages map[Language]*LanguageInfo, inventory []FileMetric, report *facts.CodeFactsReport, factsErr error) map[Language]*LanguageMetrics {
	result := make(map[Language]*LanguageMetrics)
	graphMetrics := graphFileMetrics(report)
	filesByLanguage := make(map[string][]string, len(languages))
	for language, info := range languages {
		filesByLanguage[string(language)] = info.Files
	}
	duplicationDetector := NewDuplicationDetector(ls.scenarioPath, ls.timeout)
	var duplicatesByLanguage map[string]*DuplicateResult
	if factsErr != nil {
		duplicatesByLanguage = make(map[string]*DuplicateResult, len(filesByLanguage))
		for language := range filesByLanguage {
			duplicatesByLanguage[language] = &DuplicateResult{Skipped: true, SkipReason: factsErr.Error(), Tool: "code-facts"}
		}
	} else if report != nil {
		duplicatesByLanguage = duplicationDetector.DetectDuplicationForLanguagesFromFacts(filesByLanguage, report)
	} else {
		duplicatesByLanguage = make(map[string]*DuplicateResult, len(filesByLanguage))
		for language := range filesByLanguage {
			duplicatesByLanguage[language] = &DuplicateResult{Skipped: true, SkipReason: "shared syntax facts are not available", Tool: "code-facts"}
		}
	}

	for lang, langInfo := range languages {
		files := langInfo.Files
		if len(files) == 0 {
			continue
		}
		// One graph response supplies every structural metric across all languages.
		// Python and Rust still use lexical fallbacks; JavaScript graph gaps are
		// handled per file for config entrypoints outside TypeScript projects.
		metrics := &LanguageMetrics{
			Language: lang, FileCount: len(files), TotalLines: langInfo.TotalLines,
			facts: report,
		}

		// Run code metrics (TODOs, imports, functions) - always available
		languageInventory := make([]FileMetric, 0, len(files))
		for _, metric := range inventory {
			if Language(strings.ToLower(metric.Language)) == lang {
				languageInventory = append(languageInventory, metric)
			}
		}
		codeMetricsAnalyzer := NewCodeMetricsAnalyzer(ls.scenarioPath)
		codeMetrics, err := codeMetricsAnalyzer.AnalyzeInventoryWithGraphMetrics(languageInventory, lang, graphMetrics)
		if err == nil {
			metrics.CodeMetrics = codeMetrics
		}

		// Reuse current Code Facts syntax metrics.
		complexityAnalyzer := NewComplexityAnalyzer(ls.scenarioPath, ls.timeout)
		var complexity *ComplexityResult
		if factsErr != nil {
			complexity = &ComplexityResult{Skipped: true, SkipReason: factsErr.Error()}
		} else if lang != LanguageGo {
			complexity = &ComplexityResult{Skipped: true, SkipReason: fmt.Sprintf("shared complexity facts are not yet available for %s", lang)}
		} else if report != nil {
			complexity, err = complexityAnalyzer.AnalyzeComplexityFromFacts(files, report)
		} else {
			complexity = &ComplexityResult{Skipped: true, SkipReason: fmt.Sprintf("shared syntax facts are not available for %s", lang)}
		}
		if err == nil {
			metrics.Complexity = complexity
		}

		// Project all language duplicate results from the one shared graph traversal.
		metrics.Duplicates = duplicatesByLanguage[string(lang)]

		result[lang] = metrics
	}

	return result
}

func needsSharedSyntaxFacts(languages map[Language]*LanguageInfo) bool {
	for _, lang := range []Language{LanguageGo, LanguageTypeScript, LanguageJavaScript} {
		if info := languages[lang]; info != nil && len(info.Files) > 0 {
			return true
		}
	}
	return false
}

// runMakeCommand executes a make target and captures output
func (ls *LightScanner) runMakeCommand(ctx context.Context, target string) *CommandRun {
	command := fmt.Sprintf("make %s", target)
	if !isValidMakeTarget(target) {
		return skippedCommandRun(command, "invalid make target format", "target validation failed", 0)
	}

	cmd := exec.CommandContext(ctx, "make", target) // #nosec G204 G702 -- executable is fixed and target is validated by isValidMakeTarget.
	cmd.Dir = ls.scenarioPath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return skippedCommandRun(
			command,
			err.Error(),
			fmt.Sprintf("target '%s' not available or failed to execute", target),
			0,
		)
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	timer := time.NewTimer(ls.timeout)
	defer timer.Stop()

	startTime := time.Now()

	select {
	case err := <-done:
		return buildCommandRun(command, target, err, &stdout, &stderr, startTime)
	case <-ctx.Done():
		_ = cmd.Process.Kill()
		return skippedCommandRun(command, "command canceled", "context canceled", time.Since(startTime).Milliseconds())
	case <-timer.C:
		_ = cmd.Process.Kill()
		return skippedCommandRun(command, "command timed out", "timeout exceeded", time.Since(startTime).Milliseconds())
	}
}

func isValidMakeTarget(target string) bool {
	if len(target) == 0 || len(target) > 64 {
		return false
	}

	for _, c := range target {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

func skippedCommandRun(command, stderr, reason string, durationMs int64) *CommandRun {
	return &CommandRun{
		Command:    command,
		ExitCode:   -1,
		Stderr:     stderr,
		Duration:   durationMs,
		Success:    false,
		Skipped:    true,
		SkipReason: reason,
	}
}

func buildCommandRun(command, target string, err error, stdout, stderr *bytes.Buffer, startTime time.Time) *CommandRun {
	durationMs := time.Since(startTime).Milliseconds()

	if err == nil {
		return &CommandRun{
			Command:  command,
			ExitCode: 0,
			Stdout:   stdout.String(),
			Stderr:   stderr.String(),
			Duration: durationMs,
			Success:  true,
		}
	}

	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return skippedCommandRun(command, err.Error(), "timeout exceeded", durationMs)
	}

	if exitErr, ok := err.(*exec.ExitError); ok {
		return &CommandRun{
			Command:  command,
			ExitCode: exitErr.ExitCode(),
			Stdout:   stdout.String(),
			Stderr:   stderr.String(),
			Duration: durationMs,
			Success:  false,
		}
	}

	return skippedCommandRun(
		command,
		err.Error(),
		fmt.Sprintf("target '%s' not available or failed to execute", target),
		durationMs,
	)
}

func (ls *LightScanner) collectFileMetrics() ([]FileMetric, error) {
	ctx, cancel := context.WithTimeout(context.Background(), ls.timeout)
	defer cancel()
	return ls.collectFileMetricsContext(ctx)
}

func (ls *LightScanner) collectFileMetricsContext(ctx context.Context) ([]FileMetric, error) {
	if ls.codeFacts == nil {
		return nil, fmt.Errorf("Code Facts file metrics source is required")
	}
	target := &facts.CodeTarget{Kind: facts.TargetKind_TARGET_KIND_PATH, Path: ls.scenarioPath}
	request := &facts.DescribeCodeFactsRequest{Target: target, Include: []facts.FactFamily{facts.FactFamily_FACT_FAMILY_FILE_METRICS}, UseCache: true, PageSize: 1000}
	metrics := make([]FileMetric, 0)
	for page := 0; page < 100; page++ {
		response, err := ls.codeFacts.DescribeCodeFacts(ctx, connect.NewRequest(request))
		if err != nil {
			return nil, fmt.Errorf("Code Facts file metrics: %w", err)
		}
		if response == nil || response.Msg == nil {
			return nil, fmt.Errorf("Code Facts file metrics returned an empty report")
		}
		for _, warning := range response.Msg.GetWarnings() {
			code, message := warning.GetCode(), warning.GetMessage()
			fileMetricUnsupported := code == "phase_6_contract_only" && strings.Contains(message, "FILE_METRICS")
			if strings.HasPrefix(code, "file_metrics_") || fileMetricUnsupported {
				return nil, fmt.Errorf("Code Facts file metrics unavailable: %s", warning.GetMessage())
			}
		}
		for _, fact := range response.Msg.GetFacts() {
			attrs := fact.GetAttributes()
			path := filepath.ToSlash(strings.TrimSpace(attrs["file_path"]))
			if path == "" || ls.isExcluded(path) {
				continue
			}
			lines, parseErr := strconv.Atoi(attrs["line_count"])
			if parseErr != nil || lines < 0 {
				return nil, fmt.Errorf("Code Facts returned invalid line_count for %q: %q", path, attrs["line_count"])
			}
			if attrs["lexical_metrics_version"] != "1" {
				return nil, fmt.Errorf("Code Facts returned unsupported lexical metrics for %q: version %q", path, attrs["lexical_metrics_version"])
			}
			metric := FileMetric{Path: path, Lines: lines, Extension: attrs["file_extension"], Language: attrs["language"]}
			for key, destination := range map[string]*int{
				"code_lines": &metric.CodeLines, "comment_lines": &metric.CommentLines,
				"todo_count": &metric.TodoCount, "fixme_count": &metric.FixmeCount, "hack_count": &metric.HackCount,
				"as_any_count": &metric.AsAnyCount, "as_type_assertion_count": &metric.AsTypeAssertionCount,
				"ts_ignore_count": &metric.TsIgnoreCount, "non_null_assertion_count": &metric.NonNullAssertionCount,
				"fallback_import_count": &metric.FallbackImportCount, "fallback_function_count": &metric.FallbackFunctionCount,
			} {
				value, parseErr := strconv.Atoi(attrs[key])
				if parseErr != nil || value < 0 {
					return nil, fmt.Errorf("Code Facts returned invalid %s for %q: %q", key, path, attrs[key])
				}
				*destination = value
			}
			if metric.CodeLines+metric.CommentLines != lines {
				return nil, fmt.Errorf("Code Facts line metrics disagree for %q: non-empty=%d code=%d comment=%d", path, lines, metric.CodeLines, metric.CommentLines)
			}
			metrics = append(metrics, metric)
		}
		request.PageToken = response.Msg.GetNextPageToken()
		if request.PageToken == "" {
			return metrics, nil
		}
	}
	return nil, fmt.Errorf("Code Facts file metrics exceeded 100 pages")
}

// getPreviousScans retrieves the last scan times for all files from the database
func (ls *LightScanner) getPreviousScans(ctx context.Context, db *sql.DB) (map[string]time.Time, error) {
	scenario := filepath.Base(ls.scenarioPath)
	query := `
		SELECT file_path, updated_at
		FROM file_metrics
		WHERE scenario = $1
	`

	rows, err := db.QueryContext(ctx, query, scenario)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	previousScans := make(map[string]time.Time)
	for rows.Next() {
		var filePath string
		var updatedAt time.Time
		if err := rows.Scan(&filePath, &updatedAt); err != nil {
			continue
		}
		previousScans[filePath] = updatedAt
	}

	return previousScans, nil
}

func (ls *LightScanner) isExcluded(relPath string) bool {
	if ls == nil || len(ls.excludes) == 0 {
		return false
	}
	for _, pattern := range ls.excludes {
		candidate := filepath.ToSlash(relPath)
		for candidate != "" && candidate != "." {
			matched, err := repocontract.MatchRepoGlob(pattern, candidate)
			if err == nil && matched {
				return true
			}
			separator := strings.LastIndex(candidate, "/")
			if separator < 0 {
				break
			}
			candidate = candidate[:separator]
		}
	}
	return false
}

// changedFileMetrics preserves incremental output without a second source walk.
func (ls *LightScanner) changedFileMetrics(ctx context.Context, db *sql.DB, inventory []FileMetric) []FileMetric {
	previous, err := ls.getPreviousScans(ctx, db)
	if err != nil {
		return inventory
	}
	changed := make([]FileMetric, 0)
	for _, file := range inventory {
		info, err := os.Stat(filepath.Join(ls.scenarioPath, file.Path))
		if err != nil {
			continue
		}
		lastScan, exists := previous[file.Path]
		if !exists || info.ModTime().After(lastScan) {
			changed = append(changed, file)
		}
	}
	return changed
}
