package main

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	facts "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
)

var (
	testTodoPattern     = regexp.MustCompile(`(?i)\bTODO\b`)
	testFixmePattern    = regexp.MustCompile(`(?i)\bFIXME\b`)
	testHackPattern     = regexp.MustCompile(`(?i)\bHACK\b`)
	testAsAnyPattern    = regexp.MustCompile(`\bas\s+any\b|\bas\s+unknown\s+as\b`)
	testAsTypePattern   = regexp.MustCompile(`\bas\s+[A-Z]\w+`)
	testTsIgnorePattern = regexp.MustCompile(`@ts-ignore|@ts-expect-error`)
	testNonNullPattern  = regexp.MustCompile(`[a-zA-Z0-9_)\]]\s*!\s*[.\[]`)
)

func languageForTestExtension(extension string) (Language, bool) {
	switch strings.ToLower(extension) {
	case ".go":
		return LanguageGo, true
	case ".ts", ".tsx":
		return LanguageTypeScript, true
	case ".js", ".jsx":
		return LanguageJavaScript, true
	case ".py":
		return LanguagePython, true
	case ".rs":
		return LanguageRust, true
	default:
		return "", false
	}
}

// AnalyzeFilesWithFacts adapts real fixture files to Code Facts-owned metrics.
// Production requests use AnalyzeInventory with the catalog facts response.
func (cma *CodeMetricsAnalyzer) AnalyzeFilesWithFacts(files []string, lang Language, report *facts.CodeFactsReport) (*CodeMetrics, error) {
	inventory := make([]FileMetric, 0, len(files))
	for _, relPath := range files {
		metric, err := fileMetricForTest(filepath.Join(cma.scenarioPath, relPath), relPath, lang)
		if err != nil {
			return nil, err
		}
		inventory = append(inventory, metric)
	}
	return cma.AnalyzeInventory(inventory, lang, report)
}

func fileMetricForTest(absPath, relPath string, lang Language) (FileMetric, error) {
	metrics, err := analyzeFileForTest(absPath, lang)
	if err != nil {
		return FileMetric{}, err
	}
	return FileMetric{
		Path: relPath, Lines: metrics.CodeLines + metrics.CommentLines, Extension: filepath.Ext(relPath), Language: string(lang),
		CodeLines: metrics.CodeLines, CommentLines: metrics.CommentLines,
		TodoCount: metrics.TodoCount, FixmeCount: metrics.FixmeCount, HackCount: metrics.HackCount,
		AsAnyCount: metrics.AsAnyCount, AsTypeAssertionCount: metrics.AsTypeAssertionCount,
		TsIgnoreCount: metrics.TsIgnoreCount, NonNullAssertionCount: metrics.NonNullAssertionCount,
		FallbackImportCount: metrics.ImportCount, FallbackFunctionCount: metrics.FunctionCount,
	}, nil
}

func fileMetricAttributesForTest(metric FileMetric) map[string]string {
	return map[string]string{
		"lexical_metrics_version": "1", "file_path": metric.Path, "line_count": strconv.Itoa(metric.Lines),
		"file_extension": metric.Extension, "language": metric.Language,
		"code_lines": strconv.Itoa(metric.CodeLines), "comment_lines": strconv.Itoa(metric.CommentLines),
		"todo_count": strconv.Itoa(metric.TodoCount), "fixme_count": strconv.Itoa(metric.FixmeCount), "hack_count": strconv.Itoa(metric.HackCount),
		"as_any_count": strconv.Itoa(metric.AsAnyCount), "as_type_assertion_count": strconv.Itoa(metric.AsTypeAssertionCount),
		"ts_ignore_count": strconv.Itoa(metric.TsIgnoreCount), "non_null_assertion_count": strconv.Itoa(metric.NonNullAssertionCount),
		"fallback_import_count": strconv.Itoa(metric.FallbackImportCount), "fallback_function_count": strconv.Itoa(metric.FallbackFunctionCount),
	}
}

func analyzeFileForTest(path string, lang Language) (*FileCodeMetrics, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	metrics := &FileCodeMetrics{}
	isTS := lang == LanguageTypeScript || lang == LanguageJavaScript
	inMultiLineComment := false
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		isComment, nextCommentState := isCommentLineForTest(trimmed, lang, inMultiLineComment)
		inMultiLineComment = nextCommentState
		if isComment {
			metrics.CommentLines++
		} else {
			metrics.CodeLines++
		}
		if testTodoPattern.MatchString(line) {
			metrics.TodoCount++
		}
		if testFixmePattern.MatchString(line) {
			metrics.FixmeCount++
		}
		if testHackPattern.MatchString(line) {
			metrics.HackCount++
		}
		if isTS && testTsIgnorePattern.MatchString(line) {
			metrics.TsIgnoreCount++
		}
		if !isComment && (lang == LanguagePython || lang == LanguageRust || lang == LanguageJavaScript) {
			if fallbackImportForTest(trimmed, lang) {
				metrics.ImportCount++
			}
			if fallbackFunctionForTest(trimmed, lang) {
				metrics.FunctionCount++
			}
		}
		if !isComment && isTS {
			if testAsAnyPattern.MatchString(line) {
				metrics.AsAnyCount++
			}
			if testAsTypePattern.MatchString(line) {
				metrics.AsTypeAssertionCount++
			}
			if testNonNullPattern.MatchString(line) && !strings.Contains(line, "!==") && !strings.Contains(line, "!=") {
				metrics.NonNullAssertionCount++
			}
		}
	}
	if metrics.CodeLines > 0 {
		metrics.CommentToCodeRatio = float64(metrics.CommentLines) / float64(metrics.CodeLines)
	}
	return metrics, scanner.Err()
}

func isCommentLineForTest(trimmed string, lang Language, inMultiLineComment bool) (bool, bool) {
	switch lang {
	case LanguageGo, LanguageTypeScript, LanguageJavaScript, LanguageRust:
		if strings.HasPrefix(trimmed, "/*") {
			return true, !strings.Contains(trimmed, "*/")
		}
		if inMultiLineComment {
			return true, !strings.Contains(trimmed, "*/")
		}
		return strings.HasPrefix(trimmed, "//"), false
	case LanguagePython:
		if strings.HasPrefix(trimmed, `"""`) || strings.HasPrefix(trimmed, `'''`) {
			quote := `"""`
			if strings.HasPrefix(trimmed, `'''`) {
				quote = `'''`
			}
			if strings.Count(trimmed, quote) >= 2 {
				return true, false
			}
			return true, !inMultiLineComment
		}
		if inMultiLineComment {
			return true, !strings.Contains(trimmed, `"""`) && !strings.Contains(trimmed, `'''`)
		}
		return strings.HasPrefix(trimmed, "#"), false
	default:
		return false, false
	}
}

func fallbackImportForTest(trimmed string, lang Language) bool {
	return lang == LanguagePython && (strings.HasPrefix(trimmed, "import ") || strings.HasPrefix(trimmed, "from ")) ||
		lang == LanguageRust && strings.HasPrefix(trimmed, "use ") ||
		lang == LanguageJavaScript && (strings.HasPrefix(trimmed, "import ") || strings.HasPrefix(trimmed, "import{"))
}

func fallbackFunctionForTest(trimmed string, lang Language) bool {
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
