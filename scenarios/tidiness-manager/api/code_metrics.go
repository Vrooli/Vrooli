package main

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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
}

// FileCodeMetrics contains metrics for a single file
type FileCodeMetrics struct {
	FilePath           string  `json:"file_path"`
	TodoCount          int     `json:"todo_count"`
	FixmeCount         int     `json:"fixme_count"`
	HackCount          int     `json:"hack_count"`
	ImportCount        int     `json:"import_count"`
	FunctionCount      int     `json:"function_count"`
	CodeLines          int     `json:"code_lines"`
	CommentLines       int     `json:"comment_lines"`
	CommentToCodeRatio float64 `json:"comment_to_code_ratio"`
	HasTestFile        bool    `json:"has_test_file"`
}

// CodeMetricsAnalyzer computes language-agnostic code metrics
type CodeMetricsAnalyzer struct {
	scenarioPath string
}

// NewCodeMetricsAnalyzer creates an analyzer for the specified scenario
func NewCodeMetricsAnalyzer(scenarioPath string) *CodeMetricsAnalyzer {
	return &CodeMetricsAnalyzer{
		scenarioPath: scenarioPath,
	}
}

// AnalyzeFiles computes metrics for a list of files
func (cma *CodeMetricsAnalyzer) AnalyzeFiles(files []string, lang Language) (*CodeMetrics, error) {
	if len(files) == 0 {
		return &CodeMetrics{}, nil
	}

	var totalTodos, totalFixmes, totalHacks int
	var totalImports, totalFunctions int
	var maxImports, maxFunctions int
	var totalCodeLines, totalCommentLines int
	var filesWithTests, filesWithoutTests int

	// Build test file map for this language
	testFiles := cma.buildTestFileMap(files, lang)

	for _, relPath := range files {
		// Skip test files themselves when counting coverage
		if cma.isTestFile(relPath, lang) {
			continue
		}

		absPath := filepath.Join(cma.scenarioPath, relPath)
		fileMetrics, err := cma.analyzeFile(absPath, lang)
		if err != nil {
			continue // Skip files we can't analyze
		}

		// Check if this source file has a test
		fileMetrics.HasTestFile = cma.hasTestFile(relPath, testFiles, lang)

		totalTodos += fileMetrics.TodoCount
		totalFixmes += fileMetrics.FixmeCount
		totalHacks += fileMetrics.HackCount
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
		TodoCount:           totalTodos,
		FixmeCount:          totalFixmes,
		HackCount:           totalHacks,
		AvgImportsPerFile:   float64(totalImports) / fileCount,
		AvgFunctionsPerFile: float64(totalFunctions) / fileCount,
		MaxImportsInFile:    maxImports,
		MaxFunctionsInFile:  maxFunctions,
		TotalCodeLines:      totalCodeLines,
		TotalCommentLines:   totalCommentLines,
		CommentToCodeRatio:  commentToCodeRatio,
		FilesWithTests:      filesWithTests,
		FilesWithoutTests:   filesWithoutTests,
		TestCoverageRatio:   testCoverageRatio,
	}, nil
}

// analyzeFile computes metrics for a single file
func (cma *CodeMetricsAnalyzer) analyzeFile(path string, lang Language) (*FileCodeMetrics, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	metrics := &FileCodeMetrics{
		FilePath: path,
	}

	// Compile regex patterns
	todoPattern := regexp.MustCompile(`(?i)\bTODO\b`)
	fixmePattern := regexp.MustCompile(`(?i)\bFIXME\b`)
	hackPattern := regexp.MustCompile(`(?i)\bHACK\b`)

	inMultiLineComment := false
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Skip blank lines
		if trimmed == "" {
			continue
		}

		// Detect if this is a comment line
		isComment, newMultiLineState := cma.isCommentLine(line, lang, inMultiLineComment)
		inMultiLineComment = newMultiLineState

		if isComment {
			metrics.CommentLines++
		} else {
			metrics.CodeLines++
		}

		// Count technical debt markers (can be in comments or code)
		if todoPattern.MatchString(line) {
			metrics.TodoCount++
		}
		if fixmePattern.MatchString(line) {
			metrics.FixmeCount++
		}
		if hackPattern.MatchString(line) {
			metrics.HackCount++
		}

		// Only count imports/functions in code lines
		if !isComment {
			// Count imports (language-specific)
			if cma.isImportLine(line, lang) {
				metrics.ImportCount++
			}

			// Count function definitions (language-specific)
			if cma.isFunctionDefinition(line, lang) {
				metrics.FunctionCount++
			}
		}
	}

	// Calculate comment-to-code ratio
	if metrics.CodeLines > 0 {
		metrics.CommentToCodeRatio = float64(metrics.CommentLines) / float64(metrics.CodeLines)
	}

	return metrics, scanner.Err()
}

// isImportLine detects import statements based on language
func (cma *CodeMetricsAnalyzer) isImportLine(line string, lang Language) bool {
	trimmed := strings.TrimSpace(line)

	switch lang {
	case LanguageGo:
		// Go imports: import "..." or import ( ... )
		return strings.HasPrefix(trimmed, "import ") && !strings.Contains(trimmed, "//")

	case LanguageTypeScript, LanguageJavaScript:
		// TS/JS imports: import ... from '...' or import '...'
		// Exclude type imports inside blocks
		return (strings.HasPrefix(trimmed, "import ") || strings.HasPrefix(trimmed, "import{")) &&
			!strings.HasPrefix(trimmed, "//")

	case LanguagePython:
		// Python imports: import ... or from ... import ...
		return (strings.HasPrefix(trimmed, "import ") || strings.HasPrefix(trimmed, "from ")) &&
			!strings.HasPrefix(trimmed, "#")

	case LanguageRust:
		// Rust imports: use ...
		return strings.HasPrefix(trimmed, "use ") && !strings.HasPrefix(trimmed, "//")

	default:
		return false
	}
}

// isFunctionDefinition detects function/method definitions based on language
func (cma *CodeMetricsAnalyzer) isFunctionDefinition(line string, lang Language) bool {
	trimmed := strings.TrimSpace(line)

	switch lang {
	case LanguageGo:
		// Go functions: func name(...) or func (receiver) name(...)
		return strings.HasPrefix(trimmed, "func ") && !strings.Contains(trimmed, "//")

	case LanguageTypeScript, LanguageJavaScript:
		// TS/JS functions: function name(...), const name = (...) =>, name(...) {, async function
		patterns := []string{
			"function ",
			"async function ",
			"const ",
			"let ",
			"var ",
		}
		for _, pattern := range patterns {
			if strings.HasPrefix(trimmed, pattern) {
				// Simple heuristic: contains ( and either => or {
				if strings.Contains(trimmed, "(") && (strings.Contains(trimmed, "=>") || strings.Contains(trimmed, "{")) {
					return !strings.HasPrefix(trimmed, "//")
				}
			}
		}
		// Method definitions: methodName(...) {
		if strings.Contains(trimmed, "(") && strings.Contains(trimmed, "{") && !strings.HasPrefix(trimmed, "//") {
			// Exclude control flow and other non-function lines
			excludes := []string{"if ", "while ", "for ", "switch ", "catch "}
			for _, exclude := range excludes {
				if strings.HasPrefix(trimmed, exclude) {
					return false
				}
			}
			return true
		}
		return false

	case LanguagePython:
		// Python functions: def name(...):
		return strings.HasPrefix(trimmed, "def ") && !strings.HasPrefix(trimmed, "#")

	case LanguageRust:
		// Rust functions: fn name(...) or pub fn name(...)
		return (strings.HasPrefix(trimmed, "fn ") || strings.HasPrefix(trimmed, "pub fn ")) &&
			!strings.HasPrefix(trimmed, "//")

	default:
		return false
	}
}

// isCommentLine detects if a line is a comment, tracking multi-line comment state
func (cma *CodeMetricsAnalyzer) isCommentLine(line string, lang Language, inMultiLineComment bool) (isComment bool, newMultiLineState bool) {
	trimmed := strings.TrimSpace(line)

	switch lang {
	case LanguageGo, LanguageTypeScript, LanguageJavaScript, LanguageRust:
		// Check for multi-line comment start
		if strings.HasPrefix(trimmed, "/*") {
			// Check if it also ends on the same line
			if strings.Contains(trimmed, "*/") {
				// Single-line /* */ comment
				return true, false
			}
			return true, true
		}

		// If currently in multi-line comment
		if inMultiLineComment {
			// Check for multi-line comment end
			if strings.Contains(trimmed, "*/") {
				return true, false
			}
			return true, true
		}

		// Check for single-line comment
		return strings.HasPrefix(trimmed, "//"), false

	case LanguagePython:
		// Check for multi-line string literals used as comments (docstrings)
		if strings.HasPrefix(trimmed, `"""`) || strings.HasPrefix(trimmed, `'''`) {
			// Check if it ends on the same line
			quote := `"""`
			if strings.HasPrefix(trimmed, `'''`) {
				quote = `'''`
			}
			// Count occurrences to see if it closes on same line
			count := strings.Count(trimmed, quote)
			if count >= 2 {
				// Starts and ends on same line
				return true, false
			}
			// Toggle multi-line state
			return true, !inMultiLineComment
		}

		// If in multi-line string
		if inMultiLineComment {
			// Check for end of multi-line string
			if strings.Contains(trimmed, `"""`) || strings.Contains(trimmed, `'''`) {
				return true, false
			}
			return true, true
		}

		// Single-line comment
		return strings.HasPrefix(trimmed, "#"), false

	default:
		return false, false
	}
}

// buildTestFileMap creates a map of test files for quick lookup
func (cma *CodeMetricsAnalyzer) buildTestFileMap(files []string, lang Language) map[string]bool {
	testFiles := make(map[string]bool)
	for _, file := range files {
		if cma.isTestFile(file, lang) {
			testFiles[file] = true
		}
	}
	return testFiles
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

// hasTestFile checks if a source file has a corresponding test file
func (cma *CodeMetricsAnalyzer) hasTestFile(sourceFile string, testFiles map[string]bool, lang Language) bool {
	dir := filepath.Dir(sourceFile)
	base := filepath.Base(sourceFile)
	ext := filepath.Ext(base)
	nameWithoutExt := strings.TrimSuffix(base, ext)

	switch lang {
	case LanguageGo:
		// Go: file.go -> file_test.go
		testFile := filepath.Join(dir, nameWithoutExt+"_test.go")
		return testFiles[testFile]

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
