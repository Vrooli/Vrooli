package main

import (
	"bufio"
	"bytes"
	"context"
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
	Scenario    string        `json:"scenario"`
	StartedAt   time.Time     `json:"started_at"`
	CompletedAt time.Time     `json:"completed_at"`
	Duration    time.Duration `json:"duration_ms"`
	LintOutput  *CommandRun   `json:"lint_output,omitempty"`
	TypeOutput  *CommandRun   `json:"type_output,omitempty"`
	FileMetrics []FileMetric  `json:"file_metrics"`
	LongFiles   []LongFile    `json:"long_files"`
	TotalFiles  int           `json:"total_files"`
	TotalLines  int           `json:"total_lines"`
	HasMakefile bool          `json:"has_makefile"`
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

// Scan runs the complete light scan pipeline
func (ls *LightScanner) Scan(ctx context.Context) (*ScanResult, error) {
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

	// Collect file metrics
	metrics, err := ls.collectFileMetrics()
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

	result.CompletedAt = time.Now()
	result.Duration = result.CompletedAt.Sub(startTime)

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
