package main

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ComplexityResult contains cyclomatic complexity analysis results
type ComplexityResult struct {
	AverageComplexity   float64        `json:"average_complexity"`
	MaxComplexity       int            `json:"max_complexity"`
	HighComplexityFiles []ComplexFile  `json:"high_complexity_files"`
	HighComplexityCount int            `json:"high_complexity_count"`
	Threshold           int            `json:"threshold"` // Functions above this are flagged
	TotalFunctions      int            `json:"total_functions"`
	Skipped             bool           `json:"skipped"`
	SkipReason          string         `json:"skip_reason,omitempty"`
	Tool                string         `json:"tool,omitempty"`
}

// ComplexFile represents a function with high cyclomatic complexity
type ComplexFile struct {
	Path       string `json:"path"`
	Function   string `json:"function"`
	Complexity int    `json:"complexity"`
	Line       int    `json:"line"`
}

// ComplexityAnalyzer analyzes cyclomatic complexity using external tools
type ComplexityAnalyzer struct {
	scenarioPath string
	timeout      time.Duration
}

// NewComplexityAnalyzer creates an analyzer for the specified scenario
func NewComplexityAnalyzer(scenarioPath string, timeout time.Duration) *ComplexityAnalyzer {
	if timeout == 0 {
		timeout = 60 * time.Second
	}
	return &ComplexityAnalyzer{
		scenarioPath: scenarioPath,
		timeout:      timeout,
	}
}

// AnalyzeComplexity runs complexity analysis based on detected language
func (ca *ComplexityAnalyzer) AnalyzeComplexity(ctx context.Context, lang Language, files []string) (*ComplexityResult, error) {
	switch lang {
	case LanguageGo:
		return ca.analyzeGoComplexity(ctx, files)
	case LanguageTypeScript, LanguageJavaScript:
		// TypeScript/JavaScript complexity analysis would require eslint setup
		// Skip for now with a clear reason
		return &ComplexityResult{
			Skipped:    true,
			SkipReason: "TypeScript/JavaScript complexity analysis requires eslint configuration",
		}, nil
	default:
		return &ComplexityResult{
			Skipped:    true,
			SkipReason: fmt.Sprintf("complexity analysis not implemented for %s", lang),
		}, nil
	}
}

// analyzeGoComplexity uses gocyclo to analyze Go code complexity
func (ca *ComplexityAnalyzer) analyzeGoComplexity(ctx context.Context, files []string) (*ComplexityResult, error) {
	// Check if gocyclo is installed
	if !commandExists("gocyclo") {
		return &ComplexityResult{
			Skipped:    true,
			SkipReason: "gocyclo not installed (install with: go install github.com/fzipp/gocyclo/cmd/gocyclo@latest)",
		}, nil
	}

	if len(files) == 0 {
		return &ComplexityResult{
			Skipped:    true,
			SkipReason: "no files to analyze",
		}, nil
	}

	// Convert relative paths to absolute paths
	absPaths := make([]string, len(files))
	for i, relPath := range files {
		absPaths[i] = filepath.Join(ca.scenarioPath, relPath)
	}

	// Run gocyclo with threshold (default 10 for high complexity)
	threshold := 10
	cmdCtx, cancel := context.WithTimeout(ctx, ca.timeout)
	defer cancel()

	// gocyclo -over <threshold> <files...>
	args := append([]string{"-over", fmt.Sprintf("%d", threshold)}, absPaths...)
	cmd := exec.CommandContext(cmdCtx, "gocyclo", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// gocyclo returns non-zero if high complexity functions found, which is fine
	_ = cmd.Run()

	// Parse output
	output := stdout.String()
	complexFiles := ca.parseGoCycloOutput(output)

	// Run gocyclo without threshold to get all functions for statistics
	cmdCtx2, cancel2 := context.WithTimeout(ctx, ca.timeout)
	defer cancel2()

	cmd2 := exec.CommandContext(cmdCtx2, "gocyclo", absPaths...)
	var stdout2 bytes.Buffer
	cmd2.Stdout = &stdout2
	_ = cmd2.Run()

	allOutput := stdout2.String()
	allComplexity := ca.parseGoCycloOutputAll(allOutput)

	// Calculate statistics
	avgComplexity := 0.0
	maxComplexity := 0
	totalFunctions := len(allComplexity)

	if totalFunctions > 0 {
		sum := 0
		for _, c := range allComplexity {
			sum += c
			if c > maxComplexity {
				maxComplexity = c
			}
		}
		avgComplexity = float64(sum) / float64(totalFunctions)
	}

	return &ComplexityResult{
		AverageComplexity:   avgComplexity,
		MaxComplexity:       maxComplexity,
		HighComplexityFiles: complexFiles,
		HighComplexityCount: len(complexFiles),
		Threshold:           threshold,
		TotalFunctions:      totalFunctions,
		Skipped:             false,
		Tool:                "gocyclo",
	}, nil
}

// parseGoCycloOutput parses gocyclo output into structured ComplexFile entries
// Format: <complexity> <package> <function> <file>:<line>:<column>
func (ca *ComplexityAnalyzer) parseGoCycloOutput(output string) []ComplexFile {
	var results []ComplexFile

	// Example line: 15 main (*Server).handleLightScan /path/to/file.go:26:1
	re := regexp.MustCompile(`^(\d+)\s+\S+\s+(\S+)\s+(.+?):(\d+):`)

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		matches := re.FindStringSubmatch(line)
		if len(matches) < 5 {
			continue
		}

		complexity, _ := strconv.Atoi(matches[1])
		function := matches[2]
		filePath := matches[3]
		lineNum, _ := strconv.Atoi(matches[4])

		// Convert absolute path to relative path
		relPath, err := filepath.Rel(ca.scenarioPath, filePath)
		if err != nil {
			relPath = filePath // Use absolute if conversion fails
		}

		results = append(results, ComplexFile{
			Path:       relPath,
			Function:   function,
			Complexity: complexity,
			Line:       lineNum,
		})
	}

	return results
}

// parseGoCycloOutputAll parses all complexity values for statistics
func (ca *ComplexityAnalyzer) parseGoCycloOutputAll(output string) []int {
	var complexities []int

	re := regexp.MustCompile(`^(\d+)\s+`)
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		matches := re.FindStringSubmatch(line)
		if len(matches) < 2 {
			continue
		}

		complexity, _ := strconv.Atoi(matches[1])
		complexities = append(complexities, complexity)
	}

	return complexities
}

// commandExists checks if a command is available in PATH
func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}
