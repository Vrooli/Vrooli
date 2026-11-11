package main

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	facts "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
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

// AnalyzeFilesWithFacts keeps lexical markers and comment metrics local while
// projecting structural counts from the already-fetched Code Facts graph.
func (cma *CodeMetricsAnalyzer) AnalyzeFilesWithFacts(files []string, lang Language, report *facts.CodeFactsReport) (*CodeMetrics, error) {
	if len(files) == 0 {
		return &CodeMetrics{}, nil
	}

	var totalTodos, totalFixmes, totalHacks int
	var totalImports, totalFunctions int
	var maxImports, maxFunctions int
	var totalCodeLines, totalCommentLines int
	var filesWithTests, filesWithoutTests int
	var totalAsAny, totalAsType, totalTsIgnore, totalNonNull int

	// Build test file map for this language
	testFiles, sourceFiles := cma.buildTestFileMaps(files, lang)

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
		fileMetrics.HasTestFile = cma.hasTestFile(relPath, testFiles, sourceFiles, lang)
		if imports, functions, graphBacked := graphFileCounts(report, relPath, lang); graphBacked {
			fileMetrics.ImportCount, fileMetrics.FunctionCount = imports, functions
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

// analyzeFile computes metrics for a single file
func (cma *CodeMetricsAnalyzer) analyzeFile(path string, lang Language) (*FileCodeMetrics, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	metrics := &FileCodeMetrics{}

	// Compile regex patterns
	todoPattern := regexp.MustCompile(`(?i)\bTODO\b`)
	fixmePattern := regexp.MustCompile(`(?i)\bFIXME\b`)
	hackPattern := regexp.MustCompile(`(?i)\bHACK\b`)

	// Type safety patterns (TS/JS only)
	isTS := lang == LanguageTypeScript || lang == LanguageJavaScript
	var asAnyPattern, asTypePattern, tsIgnorePattern, nonNullPattern *regexp.Regexp
	if isTS {
		asAnyPattern = regexp.MustCompile(`\bas\s+any\b|\bas\s+unknown\s+as\b`)
		asTypePattern = regexp.MustCompile(`\bas\s+[A-Z]\w+`)
		tsIgnorePattern = regexp.MustCompile(`@ts-ignore|@ts-expect-error`)
		nonNullPattern = regexp.MustCompile(`[a-zA-Z0-9_)\]]\s*!\s*[.\[]`)
	}

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

		// Type safety pattern detection (TS/JS only)
		if isTS {
			// @ts-ignore/@ts-expect-error can appear in comments
			if tsIgnorePattern.MatchString(line) {
				metrics.TsIgnoreCount++
			}
		}

		// Python and Rust have no Code Facts graph provider. JavaScript keeps a
		// lightweight fallback for configuration files outside TS project graphs.
		if !isComment && (lang == LanguagePython || lang == LanguageRust || lang == LanguageJavaScript) {
			if cma.isFallbackImportLine(trimmed, lang) {
				metrics.ImportCount++
			}
			if cma.isFallbackFunctionDefinition(trimmed, lang) {
				metrics.FunctionCount++
			}
		}
		// Type safety patterns (non-comment lines only)
		if !isComment && isTS {
			if asAnyPattern.MatchString(line) {
				metrics.AsAnyCount++
			}
			if asTypePattern.MatchString(line) {
				metrics.AsTypeAssertionCount++
			}
			if nonNullPattern.MatchString(line) && !strings.Contains(line, "!==") && !strings.Contains(line, "!=") {
				metrics.NonNullAssertionCount++
			}
		}
	}

	// Calculate comment-to-code ratio
	if metrics.CodeLines > 0 {
		metrics.CommentToCodeRatio = float64(metrics.CommentLines) / float64(metrics.CodeLines)
	}

	return metrics, scanner.Err()
}

func (cma *CodeMetricsAnalyzer) isFallbackImportLine(trimmed string, lang Language) bool {
	return lang == LanguagePython && (strings.HasPrefix(trimmed, "import ") || strings.HasPrefix(trimmed, "from ")) ||
		lang == LanguageRust && strings.HasPrefix(trimmed, "use ") ||
		lang == LanguageJavaScript && (strings.HasPrefix(trimmed, "import ") || strings.HasPrefix(trimmed, "import{"))
}

func (cma *CodeMetricsAnalyzer) isFallbackFunctionDefinition(trimmed string, lang Language) bool {
	if lang == LanguagePython && strings.HasPrefix(trimmed, "def ") || lang == LanguageRust && (strings.HasPrefix(trimmed, "fn ") || strings.HasPrefix(trimmed, "pub fn ")) {
		return true
	}
	if lang != LanguageJavaScript {
		return false
	}
	if strings.Contains(trimmed, "(") && (strings.HasPrefix(trimmed, "function ") || strings.HasPrefix(trimmed, "async function ") ||
		((strings.HasPrefix(trimmed, "const ") || strings.HasPrefix(trimmed, "let ") || strings.HasPrefix(trimmed, "var ")) && (strings.Contains(trimmed, "=>") || strings.Contains(trimmed, "function")))) {
		return true
	}
	for _, control := range []string{"if ", "while ", "for ", "switch ", "catch "} {
		if strings.HasPrefix(trimmed, control) {
			return false
		}
	}
	return strings.Contains(trimmed, "(") && strings.Contains(trimmed, "{")
}

func graphFileCounts(report *facts.CodeFactsReport, file string, lang Language) (imports, functions int, graphBacked bool) {
	allowed := filepath.ToSlash(filepath.Clean(file))
	for _, envelope := range report.GetGraphs() {
		unit := envelope.GetParseUnit()
		prefix, err := filepath.Rel(report.GetTarget().GetRootPath(), unit.GetRootPath())
		if err != nil || prefix == ".." || strings.HasPrefix(prefix, "../") {
			continue
		}
		modulePaths := map[string]string{}
		for _, node := range envelope.GetGraph().GetNodes() {
			path := filepath.ToSlash(filepath.Join(prefix, node.GetPath()))
			attrs := node.GetAttributes()
			kind := attrs["kind"]
			if kind == "TS_NODE_KIND_MODULE" {
				modulePaths[node.GetId()] = path
			}
			if path != allowed {
				continue
			}
			graphBacked = graphBacked || attrs["language"] == string(lang)
			if lang == LanguageGo && kind == "go_import_spec" {
				imports++
			}
			if lang == LanguageGo && (kind == "go_func" || kind == "go_method") ||
				(lang == LanguageTypeScript || lang == LanguageJavaScript) && attrs["body_hash"] != "" {
				functions++
			}
		}
		if lang == LanguageTypeScript || lang == LanguageJavaScript {
			for _, edge := range envelope.GetGraph().GetEdges() {
				if edge.GetKind() == commonv1.EdgeKind_EDGE_KIND_IMPORT && modulePaths[edge.GetFromNodeId()] == allowed {
					imports++
				}
			}
		}
	}
	return imports, functions, graphBacked
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
