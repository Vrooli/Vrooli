package main

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/api-core/pathfilter"
	repocontract "github.com/vrooli/repo-contract-go"
)

// LightScanner performs fast static analysis using Makefile integration
type LightScanner struct {
	scenarioPath string
	timeout      time.Duration
	excludes     []string
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
	Path      string `json:"path"`
	Lines     int    `json:"lines"`
	Extension string `json:"extension"`
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
		Scenario:  filepath.Base(ls.scenarioPath),
		StartedAt: startTime,
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
	inventory, err := ls.collectFileMetrics()
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
	result.LanguageMetrics = ls.collectLanguageMetrics(ctx, languagesFromFileMetrics(inventory))

	result.CompletedAt = time.Now()
	result.Duration = result.CompletedAt.Sub(startTime).Milliseconds()

	return result, nil
}

// collectLanguageMetrics detects languages and runs all available analyzers
func (ls *LightScanner) collectLanguageMetrics(ctx context.Context, languages map[Language]*LanguageInfo) map[Language]*LanguageMetrics {
	result := make(map[Language]*LanguageMetrics)

	for lang, langInfo := range languages {
		files := langInfo.Files
		if len(files) == 0 {
			continue
		}
		metrics := &LanguageMetrics{
			Language: lang, FileCount: len(files), TotalLines: langInfo.TotalLines,
		}

		// Run code metrics (TODOs, imports, functions) - always available
		codeMetricsAnalyzer := NewCodeMetricsAnalyzer(ls.scenarioPath)
		codeMetrics, err := codeMetricsAnalyzer.AnalyzeFiles(files, lang)
		if err == nil {
			metrics.CodeMetrics = codeMetrics
		}

		// Run complexity analysis (requires external tools)
		complexityAnalyzer := NewComplexityAnalyzer(ls.scenarioPath, ls.timeout)
		complexity, err := complexityAnalyzer.AnalyzeComplexity(ctx, lang, files)
		if err == nil {
			metrics.Complexity = complexity
		}

		// Run duplication detection (requires external tools)
		duplicationDetector := NewDuplicationDetector(ls.scenarioPath, ls.timeout)
		duplicates, err := duplicationDetector.DetectDuplication(ctx, lang, files)
		if err == nil {
			metrics.Duplicates = duplicates
		}

		result[lang] = metrics
	}

	return result
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

// collectFileMetrics is the source inventory for every maintainability metric.
func (ls *LightScanner) collectFileMetrics() ([]FileMetric, error) {
	metrics := []FileMetric{}
	err := filepath.Walk(ls.scenarioPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		relPath, _ := filepath.Rel(ls.scenarioPath, path)
		if info.IsDir() {
			if pathfilter.SkipDir(info.Name()) || ls.isExcluded(relPath) {
				return filepath.SkipDir
			}
			return nil
		}
		ext := filepath.Ext(path)
		if ls.isExcluded(relPath) || !pathfilter.IsSourceExt(ext) {
			return nil
		}
		lines, err := countLines(path)
		if err != nil {
			return nil
		}
		metrics = append(metrics, FileMetric{Path: relPath, Lines: lines, Extension: ext})
		return nil
	})
	return metrics, err
}

// countLines counts non-empty lines in a file
func countLines(path string) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	count := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			count++
		}
	}

	return count, scanner.Err()
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
		matched, err := repocontract.MatchRepoGlob(pattern, filepath.ToSlash(relPath))
		if err == nil && matched {
			return true
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
