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
}

// FileCodeMetrics contains metrics for a single file
type FileCodeMetrics struct {
	FilePath      string `json:"file_path"`
	TodoCount     int    `json:"todo_count"`
	FixmeCount    int    `json:"fixme_count"`
	HackCount     int    `json:"hack_count"`
	ImportCount   int    `json:"import_count"`
	FunctionCount int    `json:"function_count"`
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

	for _, relPath := range files {
		absPath := filepath.Join(cma.scenarioPath, relPath)
		fileMetrics, err := cma.analyzeFile(absPath, lang)
		if err != nil {
			continue // Skip files we can't analyze
		}

		totalTodos += fileMetrics.TodoCount
		totalFixmes += fileMetrics.FixmeCount
		totalHacks += fileMetrics.HackCount
		totalImports += fileMetrics.ImportCount
		totalFunctions += fileMetrics.FunctionCount

		if fileMetrics.ImportCount > maxImports {
			maxImports = fileMetrics.ImportCount
		}
		if fileMetrics.FunctionCount > maxFunctions {
			maxFunctions = fileMetrics.FunctionCount
		}
	}

	fileCount := float64(len(files))
	return &CodeMetrics{
		TodoCount:           totalTodos,
		FixmeCount:          totalFixmes,
		HackCount:           totalHacks,
		AvgImportsPerFile:   float64(totalImports) / fileCount,
		AvgFunctionsPerFile: float64(totalFunctions) / fileCount,
		MaxImportsInFile:    maxImports,
		MaxFunctionsInFile:  maxFunctions,
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

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// Count technical debt markers
		if todoPattern.MatchString(line) {
			metrics.TodoCount++
		}
		if fixmePattern.MatchString(line) {
			metrics.FixmeCount++
		}
		if hackPattern.MatchString(line) {
			metrics.HackCount++
		}

		// Count imports (language-specific)
		if cma.isImportLine(line, lang) {
			metrics.ImportCount++
		}

		// Count function definitions (language-specific)
		if cma.isFunctionDefinition(line, lang) {
			metrics.FunctionCount++
		}
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
