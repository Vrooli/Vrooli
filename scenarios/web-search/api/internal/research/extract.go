package research

import (
	"regexp"
	"strings"
)

// boilerplateBlocks are container elements whose entire content is navigation,
// chrome, or ads — never article body. They are removed wholesale (open tag,
// content, close tag) before tag-stripping.
var boilerplateBlocks = []string{"script", "style", "noscript", "nav", "header", "footer", "aside", "form"}

// blockBoilerplateRes holds one full <tag>...</tag> matcher per boilerplate
// block. Go's RE2 has no backreferences, so each tag gets its own compiled
// regexp rather than a single backreferenced alternation.
var blockBoilerplateRes = func() []*regexp.Regexp {
	out := make([]*regexp.Regexp, 0, len(boilerplateBlocks))
	for _, tag := range boilerplateBlocks {
		t := regexp.QuoteMeta(tag)
		out = append(out, regexp.MustCompile(`(?is)<`+t+`\b[^>]*>.*?</\s*`+t+`\s*>`))
	}
	return out
}()

// htmlCommentRe matches HTML comments.
var htmlCommentRe = regexp.MustCompile(`(?s)<!--.*?-->`)

// tagRe matches any remaining HTML tag.
var tagRe = regexp.MustCompile(`(?s)<[^>]*>`)

// wsRe collapses runs of whitespace to a single space.
var wsRe = regexp.MustCompile(`[ \t\f\r]+`)

// blankLinesRe collapses 3+ newlines to a paragraph break.
var blankLinesRe = regexp.MustCompile(`\n{3,}`)

// htmlEntities is a small replacer for the entities that survive tag stripping
// and would otherwise leak markup noise into the synthesis prompt.
var htmlEntities = strings.NewReplacer(
	"&nbsp;", " ",
	"&amp;", "&",
	"&lt;", "<",
	"&gt;", ">",
	"&quot;", `"`,
	"&#39;", "'",
	"&apos;", "'",
)

// ExtractReadableText turns a page's HTML into readable body text: it drops
// boilerplate blocks (script/style/nav/header/footer/aside/form), strips the
// remaining tags, decodes a handful of common entities, and normalizes
// whitespace. Block-level tags become newlines so paragraph structure survives.
// Plain (tag-free) input is returned trimmed unchanged, so non-HTML fetches
// degrade gracefully.
func ExtractReadableText(html string) string {
	if html == "" {
		return ""
	}
	cleaned := htmlCommentRe.ReplaceAllString(html, "")
	for _, re := range blockBoilerplateRes {
		cleaned = re.ReplaceAllString(cleaned, " ")
	}
	// Map block-closing tags to newlines so paragraphs don't run together.
	cleaned = blockBreakRe.ReplaceAllString(cleaned, "\n")
	cleaned = tagRe.ReplaceAllString(cleaned, " ")
	cleaned = htmlEntities.Replace(cleaned)

	// Normalize whitespace line-by-line, dropping empty lines.
	lines := strings.Split(cleaned, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = wsRe.ReplaceAllString(line, " ")
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	joined := strings.Join(out, "\n")
	joined = blankLinesRe.ReplaceAllString(joined, "\n\n")
	return strings.TrimSpace(joined)
}

// blockBreakRe matches the closing (or self-contained) form of common
// block-level tags so they become line breaks rather than spaces.
var blockBreakRe = regexp.MustCompile(`(?is)</?\s*(p|div|br|li|tr|h[1-6]|section|article|ul|ol|table)\b[^>]*>`)
