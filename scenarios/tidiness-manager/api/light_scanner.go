package main

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// LightScanner performs fast static analysis using Makefile integration
type LightScanner struct {
	scenarioPath string
	timeout      time.Duration
}

// NewLightScanner creates a scanner for the specified scenario directory
func NewLightScanner(scenarioPath string, timeout time.Duration) *LightScanner {
	if timeout == 0 {
		timeout = 120 * time.Second // Default 2 minutes
	}
	return &LightScanner{
		scenarioPath: scenarioPath,
		timeout:      timeout,
	}
}

// ScanResult contains all outputs from a light scan
type ScanResult struct {
	Scenario        string                        `json:"scenario"`
	StartedAt       time.Time                     `json:"started_at"`
	CompletedAt     time.Time                     `json:"completed_at"`
	Duration        time.Duration                 `json:"duration_ms"`
	LintOutput      *CommandRun                   `json:"lint_output,omitempty"`
	TypeOutput      *CommandRun                   `json:"type_output,omitempty"`
	FileMetrics     []FileMetric                  `json:"file_metrics"`
	LongFiles       []LongFile                    `json:"long_files"`
	TotalFiles      int                           `json:"total_files"`
	TotalLines      int                           `json:"total_lines"`
	HasMakefile     bool                          `json:"has_makefile"`
	LanguageMetrics map[Language]*LanguageMetrics `json:"language_metrics,omitempty"`
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

	// Collect file metrics (with incremental support)
	var metrics []FileMetric
	var err error

	if opts.Incremental && opts.DB != nil {
		metrics, err = ls.collectFileMetricsIncremental(ctx, opts.DB)
	} else {
		metrics, err = ls.collectFileMetrics()
	}

	if err != nil {
		return nil, fmt.Errorf("failed to collect file metrics: %w", err)
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

	// Detect languages and run advanced metrics
	langMetrics, err := ls.collectLanguageMetrics(ctx)
	if err != nil {
		// Don't fail the entire scan if language metrics fail
		// Just log and continue
		fmt.Printf("Warning: failed to collect language metrics: %v\n", err)
	} else {
		result.LanguageMetrics = langMetrics
	}

	result.CompletedAt = time.Now()
	result.Duration = result.CompletedAt.Sub(startTime)

	return result, nil
}

// collectLanguageMetrics detects languages and runs all available analyzers
func (ls *LightScanner) collectLanguageMetrics(ctx context.Context) (map[Language]*LanguageMetrics, error) {
	detector := NewLanguageDetector(ls.scenarioPath)
	languages, err := detector.DetectLanguages()
	if err != nil {
		return nil, err
	}

	result := make(map[Language]*LanguageMetrics)

	for lang, langInfo := range languages {
		metrics := &LanguageMetrics{
			Language:   lang,
			FileCount:  langInfo.FileCount,
			TotalLines: langInfo.TotalLines,
		}

		// Run code metrics (TODOs, imports, functions) - always available
		codeMetricsAnalyzer := NewCodeMetricsAnalyzer(ls.scenarioPath)
		codeMetrics, err := codeMetricsAnalyzer.AnalyzeFiles(langInfo.Files, lang)
		if err == nil {
			metrics.CodeMetrics = codeMetrics
		}

		// Run complexity analysis (requires external tools)
		complexityAnalyzer := NewComplexityAnalyzer(ls.scenarioPath, ls.timeout)
		complexity, err := complexityAnalyzer.AnalyzeComplexity(ctx, lang, langInfo.Files)
		if err == nil {
			metrics.Complexity = complexity
		}

		// Run duplication detection (requires external tools)
		duplicationDetector := NewDuplicationDetector(ls.scenarioPath, ls.timeout)
		duplicates, err := duplicationDetector.DetectDuplication(ctx, lang, langInfo.Files)
		if err == nil {
			metrics.Duplicates = duplicates
		}

		result[lang] = metrics
	}

	return result, nil
}

// runMakeCommand executes a make target and captures output
func (ls *LightScanner) runMakeCommand(ctx context.Context, target string) *CommandRun {
	cmdCtx, cancel := context.WithTimeout(ctx, ls.timeout)
	defer cancel()

	startTime := time.Now()
	cmd := exec.CommandContext(cmdCtx, "make", target)
	cmd.Dir = ls.scenarioPath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	duration := time.Since(startTime)

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			// Command not found or other errors
			return &CommandRun{
				Command:    fmt.Sprintf("make %s", target),
				ExitCode:   -1,
				Stderr:     err.Error(),
				Duration:   duration.Milliseconds(),
				Success:    false,
				Skipped:    true,
				SkipReason: fmt.Sprintf("target '%s' not available or failed to execute", target),
			}
		}
	}

	return &CommandRun{
		Command:  fmt.Sprintf("make %s", target),
		ExitCode: exitCode,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Duration: duration.Milliseconds(),
		Success:  exitCode == 0,
		Skipped:  false,
	}
}

// collectFileMetrics walks source directories and counts lines
func (ls *LightScanner) collectFileMetrics() ([]FileMetric, error) {
	metrics := []FileMetric{}

	// Source directories to scan
	sourceDirs := []string{
		filepath.Join(ls.scenarioPath, "api"),
		filepath.Join(ls.scenarioPath, "ui", "src"),
		filepath.Join(ls.scenarioPath, "cli"),
	}

	extensions := map[string]bool{
		".go":  true,
		".ts":  true,
		".tsx": true,
		".js":  true,
		".jsx": true,
	}

	for _, dir := range sourceDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue // Skip if directory doesn't exist
		}

		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip files we can't access
			}

			if info.IsDir() {
				// Skip node_modules and hidden directories
				if info.Name() == "node_modules" || strings.HasPrefix(info.Name(), ".") {
					return filepath.SkipDir
				}
				return nil
			}

			ext := filepath.Ext(path)
			if !extensions[ext] {
				return nil
			}

			lines, err := countLines(path)
			if err != nil {
				return nil // Skip files we can't read
			}

			relPath, _ := filepath.Rel(ls.scenarioPath, path)
			metrics = append(metrics, FileMetric{
				Path:      relPath,
				Lines:     lines,
				Extension: ext,
			})

			return nil
		})

		if err != nil {
			return nil, err
		}
	}

	return metrics, nil
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

// collectFileMetricsIncremental only scans files modified since last scan
func (ls *LightScanner) collectFileMetricsIncremental(ctx context.Context, db *sql.DB) ([]FileMetric, error) {
	// Get previously scanned files and their last scan times
	scenario := filepath.Base(ls.scenarioPath)
	query := `
		SELECT file_path, updated_at
		FROM file_metrics
		WHERE scenario = $1
	`

	rows, err := db.QueryContext(ctx, query, scenario)
	if err != nil {
		// If query fails, fall back to full scan
		return ls.collectFileMetrics()
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

	// Scan all source directories
	sourceDirs := []string{
		filepath.Join(ls.scenarioPath, "api"),
		filepath.Join(ls.scenarioPath, "ui", "src"),
		filepath.Join(ls.scenarioPath, "cli"),
	}

	extensions := map[string]bool{
		".go":  true,
		".ts":  true,
		".tsx": true,
		".js":  true,
		".jsx": true,
	}

	var changedFiles []FileMetric
	unchangedCount := 0

	for _, dir := range sourceDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}

		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}

			if info.IsDir() {
				if info.Name() == "node_modules" || strings.HasPrefix(info.Name(), ".") {
					return filepath.SkipDir
				}
				return nil
			}

			ext := filepath.Ext(path)
			if !extensions[ext] {
				return nil
			}

			relPath, _ := filepath.Rel(ls.scenarioPath, path)

			// Check if file needs rescanning
			lastScan, exists := previousScans[relPath]
			if exists && !info.ModTime().After(lastScan) {
				// File unchanged since last scan - skip it
				unchangedCount++
				return nil
			}

			// File is new or modified - scan it
			lines, err := countLines(path)
			if err != nil {
				return nil
			}

			changedFiles = append(changedFiles, FileMetric{
				Path:      relPath,
				Lines:     lines,
				Extension: ext,
			})

			return nil
		})

		if err != nil {
			return nil, err
		}
	}

	fmt.Printf("Incremental scan: %d files changed, %d files unchanged (skipped)\n",
		len(changedFiles), unchangedCount)

	// Note: Caller needs to merge changed files with cached metrics from DB
	// For now, we just return changed files - full merge happens in persistence layer
	return changedFiles, nil
}
