package facts

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"code-facts/internal/catalog"
	"code-facts/internal/retrieval"
)

type jsToken struct {
	kind string
	text string
	line int
}

// javascriptImportDocuments indexes only literal module specifiers in static
// import/export-from declarations, literal dynamic imports, and literal
// CommonJS require calls. It does not resolve aliases, package exports,
// conditions, or runtime reachability.
func javascriptImportDocuments(source catalog.SourceFile, payload []byte) ([]retrieval.Document, bool, error) {
	ext := strings.ToLower(filepath.Ext(source.Path))
	if source.Language != "typescript" && source.Language != "javascript" {
		return nil, false, nil
	}
	switch ext {
	case ".ts", ".tsx", ".mts", ".cts", ".js", ".jsx", ".mjs", ".cjs":
	default:
		return nil, false, nil
	}
	if len(payload) >= maxIndexedFile {
		line := strings.Count(string(payload), "\n") + 1
		return []retrieval.Document{importIndexPartialDocument(source, line, "source reached the per-file indexed byte cap")}, true, nil
	}
	tokens, lexErr := lexJavaScriptModuleTokens(payload)
	if lexErr != nil {
		line := lexErr.line
		if line < 1 {
			line = 1
		}
		return []retrieval.Document{javascriptImportPartialDocument(source, line, lexErr.message)}, true, nil
	}
	documents := make([]retrieval.Document, 0)
	incomplete := false
	partialLine, partialReason := 0, ""
	braceDepth := 0
	for i := 0; i < len(tokens); i++ {
		kind, module, line, ok, partial := "", "", 0, false, false
		if tokens[i].text == "require" || braceDepth == 0 {
			kind, module, line, ok, partial = jsImportAt(tokens, i)
		}
		if tokens[i].kind == "template-interpolation" {
			partial = true
		}
		if partial {
			incomplete = true
			if partialLine == 0 {
				partialLine, partialReason = tokens[i].line, "nonliteral dynamic module specifier"
			}
		}
		if !ok {
			if tokens[i].text == "{" {
				braceDepth++
			} else if tokens[i].text == "}" && braceDepth > 0 {
				braceDepth--
			}
			continue
		}
		if len(documents) >= maxImportEdgesPerFile {
			documents = append(documents, javascriptImportPartialDocument(source, line, fmt.Sprintf("import indexing reached the per-file cap of %d edges", maxImportEdgesPerFile)))
			return documents, true, nil
		}
		title := fmt.Sprintf("%s %s %s", source.Path, kind, module)
		documents = append(documents, retrieval.Document{
			ID: fmt.Sprintf("code-facts:import:%s:%d:%d:%s:%s", source.Path, line, i, kind, module), SourceFileID: source.ID,
			SourceHash: source.Hash, Path: source.Path, Language: source.Language, Role: string(source.Role), Scope: source.Scope,
			Authority: source.Authority, Kind: "imports", Title: title, ExactText: module,
			Body: title, Aliases: []string{module, source.Path, filepath.Base(source.Path)}, StartLine: line, EndLine: line,
		})
		if tokens[i].text == "{" {
			braceDepth++
		} else if tokens[i].text == "}" && braceDepth > 0 {
			braceDepth--
		}
	}
	if incomplete {
		documents = append(documents, javascriptImportPartialDocument(source, partialLine, partialReason))
	}
	return documents, incomplete, nil
}

func javascriptImportPartialDocument(source catalog.SourceFile, line int, reason string) retrieval.Document {
	return retrieval.Document{
		ID: fmt.Sprintf("code-facts:imports-partial:%s:%d", source.Path, line), SourceFileID: source.ID,
		SourceHash: source.Hash, Path: source.Path, Language: source.Language, Role: string(source.Role), Scope: source.Scope,
		Authority: source.Authority, Kind: "imports_partial", Title: "JavaScript/TypeScript import evidence incomplete",
		ExactText: "JavaScript/TypeScript import evidence incomplete", Body: fmt.Sprintf("%s: %s; import results may be incomplete.", source.Path, reason),
		Aliases: []string{source.Path, filepath.Base(source.Path)}, StartLine: line, EndLine: line,
	}
}

func jsImportAt(tokens []jsToken, i int) (kind, module string, line int, ok, partial bool) {
	t := tokens[i]
	if t.kind != "identifier" {
		return "", "", 0, false, false
	}
	if t.text == "require" && (i == 0 || tokens[i-1].text != ".") && i+1 < len(tokens) && tokens[i+1].text == "(" {
		if i+3 < len(tokens) && tokens[i+2].kind == "string" && tokens[i+3].text == ")" {
			return "requires", tokens[i+2].text, tokens[i+2].line, true, false
		}
		return "", "", t.line, false, true
	}
	if t.text != "import" && t.text != "export" {
		return "", "", 0, false, false
	}
	if t.text == "import" && i+1 < len(tokens) && tokens[i+1].text == "." {
		return "", "", 0, false, false
	}
	if t.text == "import" && i+2 < len(tokens) && tokens[i+1].text == "(" {
		if tokens[i+2].kind == "string" && i+3 < len(tokens) && tokens[i+3].text == ")" {
			return "dynamically imports", tokens[i+2].text, tokens[i+2].line, true, false
		}
		return "", "", t.line, false, true
	}
	kind = "imports"
	if t.text == "export" {
		kind = "re-exports"
	}
	// Side-effect import: import "module".
	if t.text == "import" && i+1 < len(tokens) && tokens[i+1].kind == "string" {
		return kind, tokens[i+1].text, tokens[i+1].line, true, false
	}
	// The declaration grammar permits arbitrary bindings before `from`.
	// Stop at a semicolon or a new top-level import/export token so malformed
	// syntax cannot borrow a later statement's specifier.
	for j := i + 1; j < len(tokens) && j < i+256; j++ {
		if j > i+1 && tokens[j].line > tokens[j-1].line && (tokens[j].text == "import" || tokens[j].text == "export") {
			break
		}
		if tokens[j].text == ";" {
			break
		}
		if tokens[j].text == "from" && j+1 < len(tokens) && tokens[j+1].kind == "string" {
			return kind, tokens[j+1].text, tokens[j+1].line, true, false
		}
	}
	return "", "", 0, false, false
}

type jsLexError struct {
	line    int
	message string
}

func lexJavaScriptModuleTokens(src []byte) ([]jsToken, *jsLexError) {
	var tokens []jsToken
	line := 1
	for i := 0; i < len(src); {
		c := src[i]
		if c == '\n' {
			line++
			i++
			continue
		}
		if c == '\r' || c == ' ' || c == '\t' || c == '\f' {
			i++
			continue
		}
		if c == '/' && i+1 < len(src) && src[i+1] == '/' {
			i += 2
			for i < len(src) && src[i] != '\n' {
				i++
			}
			continue
		}
		if c == '/' && i+1 < len(src) && src[i+1] == '*' {
			start := line
			i += 2
			closed := false
			for i+1 < len(src) {
				if src[i] == '\n' {
					line++
				}
				if src[i] == '*' && src[i+1] == '/' {
					i += 2
					closed = true
					break
				}
				i++
			}
			if !closed {
				return nil, &jsLexError{start, "unterminated block comment"}
			}
			continue
		}
		if c == '/' && canStartJSRegex(tokens) {
			start := line
			i++
			escaped, inClass, closed := false, false, false
			for i < len(src) && src[i] != '\n' {
				if src[i] == '[' && !escaped {
					inClass = true
				}
				if src[i] == ']' && !escaped {
					inClass = false
				}
				if src[i] == '/' && !escaped && !inClass {
					i++
					closed = true
					break
				}
				if src[i] == '\\' && !escaped {
					escaped = true
				} else {
					escaped = false
				}
				i++
			}
			if !closed {
				return nil, &jsLexError{start, "unterminated regular expression literal"}
			}
			for i < len(src) && isJSIdentPart(src[i]) {
				i++
			}
			tokens = append(tokens, jsToken{kind: "regex", line: start})
			continue
		}
		if c == '\'' || c == '"' {
			quote, start, tokLine := c, i, line
			i++
			escaped := false
			for i < len(src) {
				if src[i] == '\n' && !escaped {
					return nil, &jsLexError{tokLine, "unterminated string literal"}
				}
				if src[i] == quote && !escaped {
					break
				}
				if src[i] == '\\' && !escaped {
					escaped = true
				} else {
					escaped = false
				}
				i++
			}
			if i >= len(src) {
				return nil, &jsLexError{tokLine, "unterminated string literal"}
			}
			// Keep only plainly literal content. Escaped specifiers are left
			// out and mark the file partial instead of guessing their value.
			content := src[start+1 : i]
			if strings.ContainsRune(string(content), '\\') {
				tokens = append(tokens, jsToken{kind: "escaped-string", line: tokLine})
			} else {
				tokens = append(tokens, jsToken{kind: "string", text: string(content), line: tokLine})
			}
			i++
			continue
		}
		if c == '`' { // Template contents can contain executable interpolation; skip and abstain.
			start := line
			i++
			escaped := false
			closed := false
			for i < len(src) {
				if src[i] == '\n' {
					line++
				}
				if src[i] == '`' && !escaped {
					i++
					closed = true
					break
				}
				if src[i] == '\\' && !escaped {
					escaped = true
				} else {
					escaped = false
				}
				i++
			}
			if !closed {
				return nil, &jsLexError{start, "unterminated template literal"}
			}
			kind := "template"
			if bytes.Contains(src[start:i], []byte("${")) {
				kind = "template-interpolation"
			}
			tokens = append(tokens, jsToken{kind: kind, line: start})
			continue
		}
		if isJSIdentStart(c) {
			start, tokLine := i, line
			for i < len(src) && isJSIdentPart(src[i]) {
				i++
			}
			tokens = append(tokens, jsToken{kind: "identifier", text: string(src[start:i]), line: tokLine})
			continue
		}
		if c >= utf8.RuneSelf {
			r, size := utf8.DecodeRune(src[i:])
			if unicode.IsLetter(r) {
				start, tokLine := i, line
				i += size
				for i < len(src) {
					rr, n := utf8.DecodeRune(src[i:])
					if !unicode.IsLetter(rr) && !unicode.IsDigit(rr) && rr != '_' && rr != '$' {
						break
					}
					i += n
				}
				tokens = append(tokens, jsToken{kind: "identifier", text: string(src[start:i]), line: tokLine})
				continue
			}
		}
		tokens = append(tokens, jsToken{kind: "punct", text: string(c), line: line})
		i++
	}
	for i, t := range tokens {
		if t.kind == "escaped-string" && i > 0 && (tokens[i-1].text == "from" || tokens[i-1].text == "(" || tokens[i-1].text == "require" || tokens[i-1].text == "import") {
			return nil, &jsLexError{t.line, "escaped module specifier is not normalized"}
		}
	}
	return tokens, nil
}

func isJSIdentStart(c byte) bool {
	return c == '_' || c == '$' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}
func isJSIdentPart(c byte) bool { return isJSIdentStart(c) || c >= '0' && c <= '9' }

func canStartJSRegex(tokens []jsToken) bool {
	if len(tokens) == 0 {
		return true
	}
	previous := tokens[len(tokens)-1]
	if previous.kind == "identifier" {
		switch previous.text {
		case "return", "throw", "case", "delete", "void", "typeof", "instanceof", "in", "of", "yield", "await":
			return true
		}
		return false
	}
	switch previous.text {
	case "(", "[", "{", "=", ":", ",", ";", "!", "?", "&", "|", "+", "-", "*", "%", "^", "~", "<", ">":
		return true
	default:
		return false
	}
}
