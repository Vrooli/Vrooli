package main

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	facts "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
	debt "tidiness-manager/internal/discovery"
)

// ComplexityResult contains cyclomatic complexity analysis results
type ComplexityResult struct {
	AverageComplexity   float64       `json:"average_complexity"`
	MaxComplexity       int           `json:"max_complexity"`
	HighComplexityFiles []ComplexFile `json:"high_complexity_files"`
	HighComplexityCount int           `json:"high_complexity_count"`
	Threshold           int           `json:"threshold"` // Functions above this are flagged
	TotalFunctions      int           `json:"total_functions"`
	Skipped             bool          `json:"skipped"`
	SkipReason          string        `json:"skip_reason,omitempty"`
	Tool                string        `json:"tool,omitempty"`
	// FileComplexity maps file paths to their aggregate complexity stats (avg, max)
	// This includes ALL files, not just those with high-complexity functions
	FileComplexity map[string]FileComplexityStats `json:"file_complexity,omitempty"`
}

// FileComplexityStats holds per-file aggregate complexity statistics
type FileComplexityStats struct {
	Avg           float64 `json:"avg"`
	Max           int     `json:"max"`
	FunctionCount int     `json:"function_count"`
}

// ComplexFile represents a function with high cyclomatic complexity
type ComplexFile struct {
	Path       string `json:"path"`
	Function   string `json:"function"`
	Complexity int    `json:"complexity"`
	Line       int    `json:"line"`
}

// ComplexityAnalyzer consumes the shared source graph. No local parser or
// external-tool installation is needed for Go complexity metrics.
type ComplexityAnalyzer struct {
	scenarioPath string
	timeout      time.Duration
	source       ReductionFacts
}

func NewComplexityAnalyzer(scenarioPath string, timeout time.Duration) *ComplexityAnalyzer {
	if timeout == 0 {
		timeout = 60 * time.Second
	}
	return &ComplexityAnalyzer{scenarioPath: scenarioPath, timeout: timeout}
}

func (ca *ComplexityAnalyzer) AnalyzeComplexity(ctx context.Context, lang Language, files []string) (*ComplexityResult, error) {
	skipped := func(reason string) (*ComplexityResult, error) {
		return &ComplexityResult{Skipped: true, SkipReason: reason}, nil
	}
	if lang != LanguageGo {
		return skipped(fmt.Sprintf("shared complexity facts are not yet available for %s", lang))
	}
	if len(files) == 0 {
		return skipped("no files to analyze")
	}
	if _, err := resolveScenarioFiles(ca.scenarioPath, files); err != nil {
		return skipped(err.Error())
	}
	ctx, cancel := context.WithTimeout(ctx, ca.timeout)
	defer cancel()
	report, err := currentCodeFacts(ctx, ca.source, &facts.CodeTarget{Kind: facts.TargetKind_TARGET_KIND_PATH, Path: ca.scenarioPath, LanguageFilter: []string{"go"}})
	if err != nil {
		return skipped(err.Error())
	}
	if len(report.GetGraphs()) == 0 {
		return skipped("Code Facts returned no Go graph evidence")
	}
	functions, warnings := debt.FunctionMetrics(report)
	if len(warnings) > 0 {
		return skipped(strings.Join(warnings, "; "))
	}
	allowed := map[string]bool{}
	for _, file := range files {
		allowed[filepath.ToSlash(filepath.Clean(file))] = true
	}
	result := &ComplexityResult{Threshold: 10, Tool: "code-facts", FileComplexity: map[string]FileComplexityStats{}, HighComplexityFiles: []ComplexFile{}}
	sum := 0
	for _, function := range functions {
		if !allowed[function.Path] {
			continue
		}
		sum += function.Complexity
		result.TotalFunctions++
		result.MaxComplexity = max(result.MaxComplexity, function.Complexity)
		metric := result.FileComplexity[function.Path]
		metric.Avg += float64(function.Complexity)
		metric.Max = max(metric.Max, function.Complexity)
		metric.FunctionCount++
		result.FileComplexity[function.Path] = metric
		if function.Complexity > result.Threshold {
			result.HighComplexityFiles = append(result.HighComplexityFiles, ComplexFile{Path: function.Path, Function: function.Symbol, Complexity: function.Complexity, Line: function.StartLine})
		}
	}
	if result.TotalFunctions > 0 {
		result.AverageComplexity = float64(sum) / float64(result.TotalFunctions)
	}
	for path, metric := range result.FileComplexity {
		metric.Avg /= float64(metric.FunctionCount)
		result.FileComplexity[path] = metric
	}
	sort.Slice(result.HighComplexityFiles, func(i, j int) bool {
		a, b := result.HighComplexityFiles[i], result.HighComplexityFiles[j]
		if a.Complexity != b.Complexity {
			return a.Complexity > b.Complexity
		}
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		return a.Line < b.Line
	})
	result.HighComplexityCount = len(result.HighComplexityFiles)
	return result, nil
}
