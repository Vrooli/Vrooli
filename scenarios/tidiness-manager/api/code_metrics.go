package main

import (
	"path/filepath"
	"strings"

	"github.com/vrooli/api-core/factgraph"
)

// CodeMetrics contains language-agnostic code quality metrics
type CodeMetrics struct {
	TodoCount           int     `json:"todo_count"`
	FixmeCount          int     `json:"fixme_count"`
	HackCount           int     `json:"hack_count"`
	AvgImportsPerFile   float64 `json:"avg_imports_per_file"`
	AvgFunctionsPerFile float64 `json:"avg_functions_per_file"`
	MaxImportsInFile    int     `json:"max_imports_in_file"`
	MaxFunctionsInFile  int     `json:"max_functions_in_file"`
	// Comment density metrics
	TotalCodeLines     int     `json:"total_code_lines"`
	TotalCommentLines  int     `json:"total_comment_lines"`
	CommentToCodeRatio float64 `json:"comment_to_code_ratio"`
	// Test coverage indicators
	FilesWithTests    int     `json:"files_with_tests"`
	FilesWithoutTests int     `json:"files_without_tests"`
	TestCoverageRatio float64 `json:"test_coverage_ratio"` // FilesWithTests / TotalFiles
	// Type safety pattern counts (TS/JS only)
	AsAnyCount            int `json:"as_any_count"`
	AsTypeAssertionCount  int `json:"as_type_assertion_count"`
	TsIgnoreCount         int `json:"ts_ignore_count"`
	NonNullAssertionCount int `json:"non_null_assertion_count"`
}

// FileCodeMetrics contains metrics for a single file
type FileCodeMetrics struct {
	TodoCount             int     `json:"todo_count"`
	FixmeCount            int     `json:"fixme_count"`
	HackCount             int     `json:"hack_count"`
	ImportCount           int     `json:"import_count"`
	FunctionCount         int     `json:"function_count"`
	CodeLines             int     `json:"code_lines"`
	CommentLines          int     `json:"comment_lines"`
	CommentToCodeRatio    float64 `json:"comment_to_code_ratio"`
	HasTestFile           bool    `json:"has_test_file"` // Filename heuristic only; it does not prove behavioral coverage.
	AsAnyCount            int     `json:"as_any_count"`
	AsTypeAssertionCount  int     `json:"as_type_assertion_count"`
	TsIgnoreCount         int     `json:"ts_ignore_count"`
	NonNullAssertionCount int     `json:"non_null_assertion_count"`
}

// CodeMetricsAnalyzer computes language-agnostic code metrics
type CodeMetricsAnalyzer struct {
	scenarioPath string
}

// NewCodeMetricsAnalyzer creates an analyzer for the specified scenario
func NewCodeMetricsAnalyzer(scenarioPath string) *CodeMetricsAnalyzer {
	return &CodeMetricsAnalyzer{scenarioPath: scenarioPath}
}

// AnalyzeInventoryWithGraphMetrics combines a canonical file inventory with
// graph counts projected once from the shared Code Facts response.
func (cma *CodeMetricsAnalyzer) AnalyzeInventoryWithGraphMetrics(inventory []FileMetric, lang Language, graphMetrics map[string]map[Language]graphFileMetric) *CodeMetrics {
	if len(inventory) == 0 {
		return &CodeMetrics{}
	}

	var totalTodos, totalFixmes, totalHacks int
	var totalImports, totalFunctions int
	var maxImports, maxFunctions int
	var totalCodeLines, totalCommentLines int
	var filesWithTests, filesWithoutTests int
	var totalAsAny, totalAsType, totalTsIgnore, totalNonNull int

	for _, metric := range inventory {
		relPath := metric.Path
		// Skip test files themselves when counting coverage
		if metric.IsTestFile {
			continue
		}

		fileMetrics := codeMetricsFromFileMetric(metric)

		// Check if this source file has a test
		fileMetrics.HasTestFile = metric.HasTestFile
		if graphMetric, ok := graphMetrics[relPath][lang]; ok && graphMetric.graphBacked {
			fileMetrics.ImportCount, fileMetrics.FunctionCount = graphMetric.imports, graphMetric.functions
		}

		totalTodos += fileMetrics.TodoCount
		totalFixmes += fileMetrics.FixmeCount
		totalHacks += fileMetrics.HackCount
		totalAsAny += fileMetrics.AsAnyCount
		totalAsType += fileMetrics.AsTypeAssertionCount
		totalTsIgnore += fileMetrics.TsIgnoreCount
		totalNonNull += fileMetrics.NonNullAssertionCount
		totalImports += fileMetrics.ImportCount
		totalFunctions += fileMetrics.FunctionCount
		totalCodeLines += fileMetrics.CodeLines
		totalCommentLines += fileMetrics.CommentLines

		if fileMetrics.ImportCount > maxImports {
			maxImports = fileMetrics.ImportCount
		}
		if fileMetrics.FunctionCount > maxFunctions {
			maxFunctions = fileMetrics.FunctionCount
		}
		if fileMetrics.HasTestFile {
			filesWithTests++
		} else {
			filesWithoutTests++
		}
	}

	fileCount := float64(filesWithTests + filesWithoutTests) // Only count non-test files
	if fileCount == 0 {
		fileCount = 1 // Avoid division by zero
	}

	commentToCodeRatio := 0.0
	if totalCodeLines > 0 {
		commentToCodeRatio = float64(totalCommentLines) / float64(totalCodeLines)
	}

	testCoverageRatio := 0.0
	if filesWithTests+filesWithoutTests > 0 {
		testCoverageRatio = float64(filesWithTests) / float64(filesWithTests+filesWithoutTests)
	}

	return &CodeMetrics{
		TodoCount:             totalTodos,
		FixmeCount:            totalFixmes,
		HackCount:             totalHacks,
		AvgImportsPerFile:     float64(totalImports) / fileCount,
		AvgFunctionsPerFile:   float64(totalFunctions) / fileCount,
		MaxImportsInFile:      maxImports,
		MaxFunctionsInFile:    maxFunctions,
		TotalCodeLines:        totalCodeLines,
		TotalCommentLines:     totalCommentLines,
		CommentToCodeRatio:    commentToCodeRatio,
		FilesWithTests:        filesWithTests,
		FilesWithoutTests:     filesWithoutTests,
		TestCoverageRatio:     testCoverageRatio,
		AsAnyCount:            totalAsAny,
		AsTypeAssertionCount:  totalAsType,
		TsIgnoreCount:         totalTsIgnore,
		NonNullAssertionCount: totalNonNull,
	}
}

func codeMetricsFromFileMetric(metric FileMetric) *FileCodeMetrics {
	return &FileCodeMetrics{
		TodoCount: metric.TodoCount, FixmeCount: metric.FixmeCount, HackCount: metric.HackCount,
		ImportCount: metric.FallbackImportCount, FunctionCount: metric.FallbackFunctionCount,
		CodeLines: metric.CodeLines, CommentLines: metric.CommentLines,
		HasTestFile: false,
		AsAnyCount:  metric.AsAnyCount, AsTypeAssertionCount: metric.AsTypeAssertionCount,
		TsIgnoreCount: metric.TsIgnoreCount, NonNullAssertionCount: metric.NonNullAssertionCount,
		CommentToCodeRatio: commentRatio(metric.CommentLines, metric.CodeLines),
	}
}

func commentRatio(numerator, denominator int) float64 {
	if denominator == 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}

type graphFileMetric struct {
	imports, functions int
	graphBacked        bool
}

func graphFileMetricsFromIndex(index *factgraph.Index) map[string]map[Language]graphFileMetric {
	result := make(map[string]map[Language]graphFileMetric)
	if index == nil {
		return result
	}
	for _, file := range index.Files() {
		language := Language(file.Language())
		metric := graphFileMetric{imports: file.Imports(), graphBacked: true}
		for _, node := range file.Nodes() {
			kind := node.Attribute("kind")
			if language == LanguageGo && (kind == "go_func" || kind == "go_method") ||
				(language == LanguageTypeScript || language == LanguageJavaScript) && node.Attribute("body_hash") != "" {
				metric.functions++
			}
		}
		byLanguage := result[file.Path()]
		if byLanguage == nil {
			byLanguage = make(map[Language]graphFileMetric)
			result[file.Path()] = byLanguage
		}
		byLanguage[language] = metric
	}
	return result
}

// IsTestFilePath checks if a file path looks like a test file based on common
// naming conventions across languages. Unlike the method variant, this does not
// require a Language parameter — it checks all known patterns.
func IsTestFilePath(filePath string) bool {
	if strings.Contains(filePath, "/tests/") {
		return true
	}
	var analyzer CodeMetricsAnalyzer
	for _, language := range []Language{LanguageGo, LanguageTypeScript, LanguageJavaScript, LanguagePython, LanguageRust} {
		if analyzer.isTestFile(filePath, language) {
			return true
		}
	}
	return false
}

// isTestFile checks if a file is a test file based on language conventions
func (cma *CodeMetricsAnalyzer) isTestFile(filePath string, lang Language) bool {
	base := filepath.Base(filePath)

	switch lang {
	case LanguageGo:
		// Go: *_test.go
		return strings.HasSuffix(base, "_test.go")

	case LanguageTypeScript, LanguageJavaScript:
		// TS/JS: *.test.ts, *.spec.ts, *.test.js, *.spec.js
		// Also: __tests__/ directory
		return strings.Contains(filePath, "__tests__") ||
			strings.HasSuffix(base, ".test.ts") ||
			strings.HasSuffix(base, ".test.tsx") ||
			strings.HasSuffix(base, ".spec.ts") ||
			strings.HasSuffix(base, ".spec.tsx") ||
			strings.HasSuffix(base, ".test.js") ||
			strings.HasSuffix(base, ".test.jsx") ||
			strings.HasSuffix(base, ".spec.js") ||
			strings.HasSuffix(base, ".spec.jsx")

	case LanguagePython:
		// Python: test_*.py or *_test.py
		return strings.HasPrefix(base, "test_") || strings.HasSuffix(base, "_test.py")

	case LanguageRust:
		// Rust: tests in tests/ directory or #[cfg(test)] modules
		// For now, just check tests/ directory
		return strings.Contains(filePath, "/tests/")

	default:
		return false
	}
}
