package facts

import (
	"bufio"
	"context"
	"errors"
	"io"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

var (
	errInvalidUTF8Source = errors.New("file is not valid UTF-8 source text")
	todoMarkerRE         = regexp.MustCompile(`(?i)\bTODO\b`)
	fixmeMarkerRE        = regexp.MustCompile(`(?i)\bFIXME\b`)
	hackMarkerRE         = regexp.MustCompile(`(?i)\bHACK\b`)
	asAnyRE              = regexp.MustCompile(`\bas\s+any\b|\bas\s+unknown\s+as\b`)
	asTypeRE             = regexp.MustCompile(`\bas\s+[A-Z]\w+`)
	tsIgnoreRE           = regexp.MustCompile(`@ts-ignore|@ts-expect-error`)
	nonNullRE            = regexp.MustCompile(`[a-zA-Z0-9_)\]]\s*!\s*[.\[]`)
)

// lexicalFileMetrics are lightweight source metrics shared with consumers that
// need file-level maintainability signals without reopening source files.
type lexicalFileMetrics struct {
	NonEmptyLines         int
	CodeLines             int
	CommentLines          int
	TodoCount             int
	FixmeCount            int
	HackCount             int
	AsAnyCount            int
	AsTypeAssertionCount  int
	TsIgnoreCount         int
	NonNullAssertionCount int
	FallbackImportCount   int
	FallbackFunctionCount int
}

func (m lexicalFileMetrics) attributes() map[string]string {
	return map[string]string{
		"lexical_metrics_version": "1",
		"line_count":              strconv.Itoa(m.NonEmptyLines), "code_lines": strconv.Itoa(m.CodeLines),
		"comment_lines": strconv.Itoa(m.CommentLines), "todo_count": strconv.Itoa(m.TodoCount),
		"fixme_count": strconv.Itoa(m.FixmeCount), "hack_count": strconv.Itoa(m.HackCount),
		"as_any_count": strconv.Itoa(m.AsAnyCount), "as_type_assertion_count": strconv.Itoa(m.AsTypeAssertionCount),
		"ts_ignore_count": strconv.Itoa(m.TsIgnoreCount), "non_null_assertion_count": strconv.Itoa(m.NonNullAssertionCount),
		"fallback_import_count": strconv.Itoa(m.FallbackImportCount), "fallback_function_count": strconv.Itoa(m.FallbackFunctionCount),
	}
}

func inspectLexicalFile(ctx context.Context, reader io.Reader, language string) (lexicalFileMetrics, error) {
	var metrics lexicalFileMetrics
	if language == "typescript" || language == "javascript" {
		// Type-safety patterns are meaningful only for TypeScript/JavaScript.
	} else {
		language = strings.ToLower(language)
	}

	inMultiLineComment := false
	buffered := bufio.NewReader(reader)
	for {
		if err := ctx.Err(); err != nil {
			return lexicalFileMetrics{}, err
		}
		lineBytes, readErr := buffered.ReadBytes('\n')
		if len(lineBytes) == 0 && readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}
			return lexicalFileMetrics{}, readErr
		}
		if !utf8.Valid(lineBytes) {
			return lexicalFileMetrics{}, errInvalidUTF8Source
		}
		line := string(lineBytes)
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		metrics.NonEmptyLines++
		isComment, nextCommentState := sourceCommentLine(trimmed, language, inMultiLineComment)
		inMultiLineComment = nextCommentState
		if isComment {
			metrics.CommentLines++
		} else {
			metrics.CodeLines++
		}
		if todoMarkerRE.MatchString(line) {
			metrics.TodoCount++
		}
		if fixmeMarkerRE.MatchString(line) {
			metrics.FixmeCount++
		}
		if hackMarkerRE.MatchString(line) {
			metrics.HackCount++
		}
		isTypeScript := language == "typescript" || language == "javascript"
		if isTypeScript && tsIgnoreRE.MatchString(line) {
			metrics.TsIgnoreCount++
		}
		if !isComment && isTypeScript {
			if asAnyRE.MatchString(line) {
				metrics.AsAnyCount++
			}
			if asTypeRE.MatchString(line) {
				metrics.AsTypeAssertionCount++
			}
			if nonNullRE.MatchString(line) && !strings.Contains(line, "!==") && !strings.Contains(line, "!=") {
				metrics.NonNullAssertionCount++
			}
		}
		if !isComment && (language == "python" || language == "rust" || language == "javascript") {
			if sourceFallbackImport(trimmed, language) {
				metrics.FallbackImportCount++
			}
			if sourceFallbackFunction(trimmed, language) {
				metrics.FallbackFunctionCount++
			}
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}
			return lexicalFileMetrics{}, readErr
		}
	}
	return metrics, nil
}

func sourceFallbackImport(trimmed, language string) bool {
	return language == "python" && (strings.HasPrefix(trimmed, "import ") || strings.HasPrefix(trimmed, "from ")) ||
		language == "rust" && strings.HasPrefix(trimmed, "use ") ||
		language == "javascript" && (strings.HasPrefix(trimmed, "import ") || strings.HasPrefix(trimmed, "import{"))
}

func sourceFallbackFunction(trimmed, language string) bool {
	if language == "python" && strings.HasPrefix(trimmed, "def ") ||
		language == "rust" && (strings.HasPrefix(trimmed, "fn ") || strings.HasPrefix(trimmed, "pub fn ")) {
		return true
	}
	if language != "javascript" {
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

func sourceCommentLine(trimmed, language string, inMultiLineComment bool) (bool, bool) {
	switch language {
	case "go", "typescript", "javascript", "rust":
		if strings.HasPrefix(trimmed, "/*") {
			if strings.Contains(trimmed, "*/") {
				return true, false
			}
			return true, true
		}
		if inMultiLineComment {
			if strings.Contains(trimmed, "*/") {
				return true, false
			}
			return true, true
		}
		return strings.HasPrefix(trimmed, "//"), false
	case "python":
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
			if strings.Contains(trimmed, `"""`) || strings.Contains(trimmed, `'''`) {
				return true, false
			}
			return true, true
		}
		return strings.HasPrefix(trimmed, "#"), false
	default:
		return false, false
	}
}
