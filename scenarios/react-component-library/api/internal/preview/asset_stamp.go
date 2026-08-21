package preview

import (
	"path/filepath"
	"regexp"
	"strings"
)

var (
	previewMarkerAttribute      = regexp.MustCompile(`\sdata-rcl-(?:asset|version|stamp)(?:\s*=\s*(?:"[^"]*"|'[^']*'|\{[^}]*\}))?`)
	previewComponentDeclaration = regexp.MustCompile(`(?m)(?:export\s+)?(?:function\s+([A-Za-z_$][\w$]*)\b|(?:const|let|var)\s+([A-Za-z_$][\w$]*)\s*=)`)
	previewCreateElement        = regexp.MustCompile(`\bcreateElement\(\s*[A-Za-z_$][\w$]*\s*,\s*\{`)
)

// stampPreviewSource is the API-preview counterpart of the Vite stamp
// plugin. Preview bundles are compiled directly by esbuild rather than by
// the UI's Vite graph, so leaving this path unstamped would make the rendered
// story disagree with the production harness oracle.
func stampPreviewSource(source, sourcePath, asset, version string) string {
	asset = strings.TrimSpace(asset)
	version = strings.TrimSpace(version)
	if asset == "" || version == "" || !strings.HasSuffix(strings.ToLower(sourcePath), ".tsx") {
		return source
	}
	cleaned := previewMarkerAttribute.ReplaceAllString(source, "")
	componentName := strings.TrimSuffix(filepath.Base(sourcePath), filepath.Ext(sourcePath))
	start := 0
	for _, match := range previewComponentDeclaration.FindAllStringSubmatchIndex(cleaned, -1) {
		nameStart, nameEnd := match[2], match[3]
		if nameStart < 0 || nameEnd < 0 {
			nameStart, nameEnd = match[4], match[5]
		}
		if nameStart < 0 || nameEnd < 0 || cleaned[nameStart:nameEnd] != componentName {
			continue
		}
		start = match[1]
		break
	}

	if rootStart, rootEnd := previewOwnedOpening(cleaned, start); rootStart >= 0 {
		attrs := ` data-rcl-asset="` + escapeAttribute(asset) + `" data-rcl-version="` + escapeAttribute(version) + `" data-rcl-stamp="vite"`
		insertAt := rootEnd - 1
		if insertAt > rootStart && cleaned[insertAt-1] == '/' {
			insertAt--
		}
		return cleaned[:insertAt] + attrs + cleaned[insertAt:]
	}
	if match := previewCreateElement.FindStringIndex(cleaned[start:]); match != nil {
		insertAt := start + match[1]
		// The regexp ends at the opening brace. Keep the object valid by
		// placing the generated properties immediately after that brace.
		attrs := `"data-rcl-asset": "` + escapeAttribute(asset) + `", "data-rcl-version": "` + escapeAttribute(version) + `", "data-rcl-stamp": "vite", `
		brace := strings.LastIndex(cleaned[:insertAt], "{")
		if brace >= 0 {
			return cleaned[:brace+1] + attrs + cleaned[brace+1:]
		}
	}
	return cleaned
}

func escapeAttribute(value string) string {
	return strings.NewReplacer("&", "&amp;", "\"", "&quot;", "<", "&lt;", ">", "&gt;").Replace(value)
}

func previewOwnedOpening(source string, start int) (int, int) {
	for position := start; position < len(source); {
		relative := strings.IndexByte(source[position:], '<')
		if relative < 0 {
			return -1, -1
		}
		open := position + relative
		if open+1 >= len(source) || source[open+1] == '/' || source[open+1] == '!' || source[open+1] == '>' {
			position = open + 1
			continue
		}
		// A JSX tag name must begin with an identifier character. This rejects
		// ordinary TypeScript comparisons such as `raw < 1` and `value <= 0`
		// before the scanner can mistake the right-hand side for a tag name.
		if !isJSXNameStart(source[open+1]) {
			position = open + 1
			continue
		}
		// A JSX opening cannot immediately continue an identifier, while a
		// TypeScript type-argument list commonly does (`forwardRef<Props>`).
		// Treating that generic as JSX injects marker attributes into the type
		// list and leaves esbuild with invalid syntax.
		if open > 0 && isIdentifierContinuation(source[open-1]) {
			position = open + 1
			continue
		}
		nameEnd := open + 1
		for nameEnd < len(source) && (source[nameEnd] == '.' || source[nameEnd] == ':' || source[nameEnd] == '_' || source[nameEnd] == '-' || source[nameEnd] >= 'A' && source[nameEnd] <= 'Z' || source[nameEnd] >= 'a' && source[nameEnd] <= 'z' || source[nameEnd] >= '0' && source[nameEnd] <= '9') {
			nameEnd++
		}
		name := source[open+1 : nameEnd]
		end := jsxOpeningEnd(source, nameEnd)
		if end < 0 {
			return -1, -1
		}
		if name == "style" || name == "Fragment" || name == "React.Fragment" || strings.HasSuffix(name, ".Provider") {
			position = end
			continue
		}
		return open, end
	}
	return -1, -1
}

func isIdentifierContinuation(value byte) bool {
	return value == '_' || value == '$' || value == '.' || value >= 'A' && value <= 'Z' || value >= 'a' && value <= 'z' || value >= '0' && value <= '9'
}

func isJSXNameStart(value byte) bool {
	return value == '_' || value == '$' || value >= 'A' && value <= 'Z' || value >= 'a' && value <= 'z'
}

func jsxOpeningEnd(source string, start int) int {
	depth := 0
	var quote byte
	for position := start; position < len(source); position++ {
		current := source[position]
		if quote != 0 {
			if current == quote && source[position-1] != '\\' {
				quote = 0
			}
			continue
		}
		if current == '\'' || current == '"' || current == '`' {
			quote = current
			continue
		}
		switch current {
		case '{':
			depth++
		case '}':
			if depth > 0 {
				depth--
			}
		case '>':
			if depth == 0 {
				return position + 1
			}
		}
	}
	return -1
}
