package components

import (
	"path/filepath"
	"regexp"
	"strings"
)

// A version's stylesheet is keyed page-globally by its library id and version
// (`<StyleSheet libraryId="…" version="…">` or
// `useLibraryStyleSheet("…", "…", css)`). The version is a string literal in
// source, so a draft copied from the previous release carries that release's
// version: FullPageDrawer 1.4.5 through 1.5.0 all registered key 1.4.3 with
// different CSS, and mounting two of them on one page collided. The literal is
// part of the version's identity, like its @version header, so the store
// rewrites it whenever it writes a version.

var (
	styleSheetElementRE   = regexp.MustCompile(`(?s)<StyleSheet\b.*?/>`)
	styleSheetLibraryIDRE = regexp.MustCompile(`\slibraryId\s*=\s*\{?\s*["']([^"']+)["']`)
	styleSheetVersionRE   = regexp.MustCompile(`(\sversion\s*=\s*\{?\s*["'])([^"']*)(["'])`)
	styleSheetHookRE      = regexp.MustCompile(`(useLibraryStyleSheet\s*\(\s*["']([^"']+)["']\s*,\s*["'])([^"']*)(["'])`)
	styleSheetCSSAttrRE   = regexp.MustCompile(`\scss\s*=\s*\{\s*([A-Za-z_$][\w$]*)\s*\}`)
	styleSheetHookCSSRE   = regexp.MustCompile(`^\s*,\s*([A-Za-z_$][\w$]*)\s*[,)]`)
)

// StyleSheetVersionLiteral is one literal version a source passes for its own
// stylesheet identity, with its byte offset in the source.
type StyleSheetVersionLiteral struct {
	Version string
	Offset  int
	// CSSName is the identifier the injection passes as its CSS, or "" when
	// the CSS is not a plain identifier.
	CSSName string
}

// SelfStyleSheetVersions lists the literal versions source passes when it
// injects the stylesheet of libraryID itself. Injections for other assets, and
// versions passed through a variable, are not listed.
func SelfStyleSheetVersions(source, libraryID string) []StyleSheetVersionLiteral {
	var out []StyleSheetVersionLiteral
	for _, element := range styleSheetElementRE.FindAllStringIndex(source, -1) {
		tag := source[element[0]:element[1]]
		id := styleSheetLibraryIDRE.FindStringSubmatch(tag)
		if len(id) < 2 || id[1] != libraryID {
			continue
		}
		if version := styleSheetVersionRE.FindStringSubmatchIndex(tag); version != nil {
			literal := StyleSheetVersionLiteral{Version: tag[version[4]:version[5]], Offset: element[0] + version[4]}
			if css := styleSheetCSSAttrRE.FindStringSubmatch(tag); len(css) > 1 {
				literal.CSSName = css[1]
			}
			out = append(out, literal)
		}
	}
	for _, hook := range styleSheetHookRE.FindAllStringSubmatchIndex(source, -1) {
		if source[hook[4]:hook[5]] == libraryID {
			literal := StyleSheetVersionLiteral{Version: source[hook[6]:hook[7]], Offset: hook[6]}
			if css := styleSheetHookCSSRE.FindStringSubmatch(source[hook[1]:]); len(css) > 1 {
				literal.CSSName = css[1]
			}
			out = append(out, literal)
		}
	}
	return out
}

// SyncStyleSheetVersion rewrites every literal version source passes for its
// own stylesheet identity to version.
func SyncStyleSheetVersion(source, libraryID, version string) string {
	if strings.TrimSpace(libraryID) == "" || strings.TrimSpace(version) == "" {
		return source
	}
	source = styleSheetElementRE.ReplaceAllStringFunc(source, func(tag string) string {
		id := styleSheetLibraryIDRE.FindStringSubmatch(tag)
		if len(id) < 2 || id[1] != libraryID {
			return tag
		}
		return styleSheetVersionRE.ReplaceAllString(tag, "${1}"+version+"${3}")
	})
	return styleSheetHookRE.ReplaceAllStringFunc(source, func(call string) string {
		parts := styleSheetHookRE.FindStringSubmatch(call)
		if parts[2] != libraryID {
			return call
		}
		return parts[1] + version + parts[4]
	})
}

func isSourceFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".ts" || ext == ".tsx"
}

// StyleSheetConstText returns the text of the string or template literal bound
// to `const name` in any of sources, and whether it was found. It is how a
// gate compares the CSS two versions inject under one key without evaluating
// the module.
func StyleSheetConstText(name string, sources ...string) (string, bool) {
	if name == "" {
		return "", false
	}
	binding := regexp.MustCompile(`(?:^|[\s;])(?:export\s+)?const\s+` + regexp.QuoteMeta(name) + `\s*(?::[^=]+)?=\s*`)
	for _, source := range sources {
		match := binding.FindStringIndex(source)
		if match == nil {
			continue
		}
		if text, ok := leadingStringLiteral(source[match[1]:]); ok {
			return text, true
		}
	}
	return "", false
}

// leadingStringLiteral reads the quoted or backtick literal at the start of
// rest, skipping over `${...}` interpolations inside a template.
func leadingStringLiteral(rest string) (string, bool) {
	if rest == "" {
		return "", false
	}
	quote := rest[0]
	if quote != '`' && quote != '"' && quote != '\'' {
		return "", false
	}
	depth := 0
	for index := 1; index < len(rest); index++ {
		switch {
		case rest[index] == '\\':
			index++
		case quote == '`' && depth == 0 && rest[index] == '$' && index+1 < len(rest) && rest[index+1] == '{':
			depth = 1
			index++
		case depth > 0 && rest[index] == '{':
			depth++
		case depth > 0 && rest[index] == '}':
			depth--
		case depth == 0 && rest[index] == quote:
			return rest[1:index], true
		}
	}
	return "", false
}

var fileLibraryIDRE = regexp.MustCompile(`@libraryId\s+([^\s*]+)`)

// StampStyleSheetVersions stamps version into every stylesheet identity a
// source file owns: the asset's own library id, and the file's @libraryId
// header when a companion module declares its own identity (markdown-renderer's
// CodeBlock is react-component-library:code-block).
func StampStyleSheetVersions(source, assetLibraryID, version string) string {
	source = SyncStyleSheetVersion(source, assetLibraryID, version)
	if header := fileLibraryIDRE.FindStringSubmatch(source); len(header) > 1 && header[1] != assetLibraryID {
		source = SyncStyleSheetVersion(source, header[1], version)
	}
	return source
}
