package validation

import (
	"regexp"
	"strings"

	"brand-manager/internal/brandsurface"
)

// htmledit holds the small, deterministic string transforms the self-contained
// fixers apply to ui/index.html. The scenarios are generated Vite templates with
// a predictable head, so targeted string edits (insert before </head>, rewrite a
// title, set a meta's content) are sufficient and keep the file's formatting.

var headCloseRe = regexp.MustCompile(`(?i)</head>`)

// headIndent returns the leading whitespace of the existing head children so an
// injected tag lines up. Defaults to four spaces.
func headIndent(content string) string {
	if m := regexp.MustCompile(`(?im)^(\s+)<(?:meta|title|link)\b`).FindStringSubmatch(content); m != nil {
		return m[1]
	}
	return "    "
}

// injectBeforeHeadClose inserts each line (already-rendered tags) immediately
// before </head>, indented to match the head. Returns content unchanged when
// there is no </head> or no lines to add.
func injectBeforeHeadClose(content string, lines []string) string {
	if len(lines) == 0 || !headCloseRe.MatchString(content) {
		return content
	}
	indent := headIndent(content)
	var b strings.Builder
	for _, l := range lines {
		b.WriteString(indent)
		b.WriteString(l)
		b.WriteString("\n")
	}
	loc := headCloseRe.FindStringIndex(content)
	// Trim back to the start of the </head> line so the close tag keeps its own
	// indentation.
	insertAt := loc[0]
	lineStart := strings.LastIndexByte(content[:insertAt], '\n') + 1
	return content[:lineStart] + b.String() + content[lineStart:]
}

// metaLine renders a Tag as an HTML <meta> element.
func metaLine(t brandsurface.Tag) string {
	return `<meta ` + string(t.Kind) + `="` + t.Key + `" content="` + htmlAttr(t.Content) + `" />`
}

// htmlAttr escapes the double-quote and ampersand that would break a quoted
// attribute value.
func htmlAttr(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}

// setTitle rewrites the <title> text, returning changed=false if absent.
func setTitle(content, title string) (string, bool) {
	if !titleRe.MatchString(content) {
		return content, false
	}
	return titleRe.ReplaceAllString(content, "<title>"+title+"</title>"), true
}

// setMetaContent rewrites the content attribute of the first matching meta tag,
// returning changed=false when no such tag exists.
func setMetaContent(content string, kind brandsurface.TagKind, key, value string) (string, bool) {
	re := regexp.MustCompile(`(?is)<meta\b[^>]*\b` + string(kind) + `\s*=\s*"` + regexp.QuoteMeta(key) + `"[^>]*>`)
	tag := re.FindString(content)
	if tag == "" {
		return content, false
	}
	updated := setContentAttr(tag, value)
	if updated == tag {
		return content, false
	}
	return strings.Replace(content, tag, updated, 1), true
}

var contentAttrRe = regexp.MustCompile(`(?is)content\s*=\s*"[^"]*"`)

func setContentAttr(tag, value string) string {
	repl := `content="` + htmlAttr(value) + `"`
	if contentAttrRe.MatchString(tag) {
		return contentAttrRe.ReplaceAllString(tag, repl)
	}
	// No content attr — add one before the closing '>'.
	return strings.TrimRight(strings.TrimRight(tag, ">"), "/ ") + " " + repl + " />"
}

// addViewportFitCover appends viewport-fit=cover to the viewport meta's content,
// returning changed=false when already present or no viewport meta exists.
func addViewportFitCover(content string) (string, bool) {
	re := regexp.MustCompile(`(?is)<meta\b[^>]*\bname\s*=\s*"viewport"[^>]*>`)
	tag := re.FindString(content)
	if tag == "" {
		return content, false
	}
	cur := parseAttrs(tag)["content"]
	if strings.Contains(strings.ReplaceAll(cur, " ", ""), viewportFitCover) {
		return content, false
	}
	next := strings.TrimSpace(cur)
	if next != "" {
		next += ", "
	}
	next += viewportFitCover
	updated := setContentAttr(tag, next)
	return strings.Replace(content, tag, updated, 1), true
}

// removeLinkMatching deletes whole lines whose <link> tag matches pred, returning
// changed=false when nothing matched.
func removeLinkMatching(content string, pred func(linkTag) bool) (string, bool) {
	lines := strings.Split(content, "\n")
	out := make([]string, 0, len(lines))
	changed := false
	for _, line := range lines {
		if tag := linkRe.FindString(line); tag != "" {
			attrs := parseAttrs(tag)
			if pred(linkTag{rel: attrs["rel"], href: attrs["href"], sizes: attrs["sizes"]}) {
				changed = true
				continue
			}
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n"), changed
}
