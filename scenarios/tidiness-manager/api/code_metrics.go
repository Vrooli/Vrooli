package main

import (
	"path/filepath"
	"strings"

	"github.com/vrooli/api-core/factgraph"
	facts "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
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

// AnalyzeInventory aggregates the lexical metrics already supplied by Code
// Facts and structural counts from its language graph.
func (cma *CodeMetricsAnalyzer) AnalyzeInventory(inventory []FileMetric, lang Language, report *facts.CodeFactsReport) (*CodeMetrics, error) {
	return cma.AnalyzeInventoryWithGraphMetrics(inventory, lang, graphFileMetrics(report))
}

func (cma *CodeMetricsAnalyzer) AnalyzeInventoryWithGraphMetrics(inventory []FileMetric, lang Language, graphMetrics map[string]map[Language]graphFileMetric) (*CodeMetrics, error) {
	if len(inventory) == 0 {
		return &CodeMetrics{}, nil
	}

	var totalTodos, totalFixmes, totalHacks int
	var totalImports, totalFunctions int
	var maxImports, maxFunctions int
	var totalCodeLines, totalCommentLines int
	var filesWithTests, filesWithoutTests int
	var totalAsAny, totalAsType, totalTsIgnore, totalNonNull int

	files := make([]string, 0, len(inventory))
	for _, metric := range inventory {
		files = append(files, metric.Path)
	}
	testFiles, sourceFiles := cma.buildTestFileMaps(files, lang)
	metricsByPath := make(map[string]FileMetric, len(inventory))
	for _, metric := range inventory {
		metricsByPath[metric.Path] = metric
	}

	for _, relPath := range files {
		// Skip test files themselves when counting coverage
		if cma.isTestFile(relPath, lang) {
			continue
		}

		fileMetrics := codeMetricsFromFileMetric(metricsByPath[relPath])

		// Check if this source file has a test
		fileMetrics.HasTestFile = cma.hasTestFile(relPath, testFiles, sourceFiles, lang)
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
	}, nil
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

// graphFileMetrics projects each graph once into file-level counts. Previously
// AnalyzeInventory rebuilt module paths and rescanned every graph for every
// source file, making this path O(files × (nodes + edges)).
func graphFileMetrics(report *facts.CodeFactsReport) map[string]map[Language]graphFileMetric {
	result := make(map[string]map[Language]graphFileMetric)
	for _, file := range factgraph.NewIndex(report).Files() {
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

// buildTestFileMaps separates test paths from source paths for companion lookup.
func (cma *CodeMetricsAnalyzer) buildTestFileMaps(files []string, lang Language) (map[string]bool, map[string]bool) {
	testFiles := make(map[string]bool)
	sourceFiles := make(map[string]bool)
	for _, file := range files {
		if cma.isTestFile(file, lang) {
			testFiles[file] = true
		} else {
			sourceFiles[file] = true
		}
	}
	return testFiles, sourceFiles
}

// IsTestFilePath checks if a file path looks like a test file based on common
// naming conventions across languages. Unlike the method variant, this does not
// require a Language parameter — it checks all known patterns.
func IsTestFilePath(filePath string) bool {
	base := filepath.Base(filePath)
	return strings.HasSuffix(base, "_test.go") ||
		strings.HasSuffix(base, ".test.ts") ||
		strings.HasSuffix(base, ".test.tsx") ||
		strings.HasSuffix(base, ".spec.ts") ||
		strings.HasSuffix(base, ".spec.tsx") ||
		strings.HasSuffix(base, ".test.js") ||
		strings.HasSuffix(base, ".test.jsx") ||
		strings.HasSuffix(base, ".spec.js") ||
		strings.HasSuffix(base, ".spec.jsx") ||
		strings.HasPrefix(base, "test_") && strings.HasSuffix(base, ".py") ||
		strings.HasSuffix(base, "_test.py") ||
		strings.Contains(filePath, "__tests__") ||
		strings.Contains(filePath, "/tests/")
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

// hasTestFile matches by exact stem or same-directory longest source stem.
// This filename heuristic does not prove that tests cover the source behavior.
func (cma *CodeMetricsAnalyzer) hasTestFile(sourceFile string, testFiles, sourceFiles map[string]bool, lang Language) bool {
	dir := filepath.Dir(sourceFile)
	base := filepath.Base(sourceFile)
	ext := filepath.Ext(base)
	nameWithoutExt := strings.TrimSuffix(base, ext)

	switch lang {
	case LanguageGo:
		// Go: file.go -> file_test.go or file_behavior_test.go.
		testFile := filepath.Join(dir, nameWithoutExt+"_test.go")
		if testFiles[testFile] {
			return true
		}
		// A descriptive suffix is common for package tests that cover one
		// production file (for example, overlap_agree_behavior_test.go).
		for candidate := range testFiles {
			if filepath.Dir(candidate) != dir {
				continue
			}
			candidateBase := filepath.Base(candidate)
			if !strings.HasPrefix(candidateBase, nameWithoutExt+"_") || !strings.HasSuffix(candidateBase, "_test.go") {
				continue
			}
			candidateStem := strings.TrimSuffix(candidateBase, "_test.go")
			mostSpecificSource := ""
			if sourceFiles[filepath.Join(dir, candidateStem+".go")] {
				mostSpecificSource = filepath.Join(dir, candidateStem+".go")
			}
			for i, char := range candidateStem {
				if char != '_' {
					continue
				}
				possibleSource := filepath.Join(dir, candidateStem[:i]+".go")
				if sourceFiles[possibleSource] && len(candidateStem[:i]) > len(strings.TrimSuffix(filepath.Base(mostSpecificSource), ".go")) {
					mostSpecificSource = possibleSource
				}
			}
			if mostSpecificSource != "" && mostSpecificSource != sourceFile {
				continue
			}
			return true
		}
		return false

	case LanguageTypeScript, LanguageJavaScript:
		// TS/JS: multiple possible test file patterns
		// 1. Same directory: file.ts -> file.test.ts or file.spec.ts
		// 2. __tests__ directory: file.ts -> __tests__/file.test.ts
		possibleTests := []string{
			filepath.Join(dir, nameWithoutExt+".test"+ext),
			filepath.Join(dir, nameWithoutExt+".spec"+ext),
			filepath.Join(dir, "__tests__", nameWithoutExt+".test"+ext),
			filepath.Join(dir, "__tests__", nameWithoutExt+".spec"+ext),
			filepath.Join(dir, "__tests__", base),
		}
		for _, testFile := range possibleTests {
			if testFiles[testFile] {
				return true
			}
		}
		return false

	case LanguagePython:
		// Python: file.py -> test_file.py or file_test.py
		possibleTests := []string{
			filepath.Join(dir, "test_"+base),
			filepath.Join(dir, nameWithoutExt+"_test.py"),
		}
		for _, testFile := range possibleTests {
			if testFiles[testFile] {
				return true
			}
		}
		return false

	case LanguageRust:
		// Rust: typically tests are in the same file or in tests/ directory
		// For now, assume if tests/ directory exists, it has tests
		testsDir := filepath.Join(dir, "tests")
		for testFile := range testFiles {
			if strings.HasPrefix(testFile, testsDir) {
				return true
			}
		}
		return false

	default:
		return false
	}
}
