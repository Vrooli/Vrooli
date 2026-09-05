package normalizer

import "strings"

// NormalizeTextForSpeech transforms markdown-formatted text into a form that
// sounds natural when read aloud by a TTS engine. The original text is left
// untouched for display; this function produces a parallel "speech" version.
//
// Processing order matters — later rules assume earlier ones have already run:
//  1. Fenced code blocks    → short summary
//  2. Markdown tables       → prose description
//     2.5. Unfenced diagrams   → classified summary (after tables are gone)
//  3. Inline code paths     → basename only
//  4. Inline code (other)   → unwrap backticks
//  5. Images                → "Image: alt"
//  6. Links                 → link text only
//  7. Heading markers       → stripped
//  8. Bold / italic         → inner text
//  9. Strikethrough         → inner text
//  10. Horizontal rules     → removed
//  11. HTML tags             → removed
//  12. Blockquote markers    → stripped
//  13. List markers          → stripped
//  14. Bare file paths       → basename only
//  15. Collapse whitespace
func NormalizeTextForSpeech(text string) string {
	s := text

	// 1. Fenced code blocks: ```lang\n...\n``` → summary.
	s = reFencedCodeBlock.ReplaceAllStringFunc(s, summarizeCodeBlock)

	// 2. Markdown tables → prose description.
	s = reMarkdownTable.ReplaceAllStringFunc(s, describeTable)

	// 2.5. Unfenced diagrams/wireframes → classified summary.
	// Runs after tables so pipe-heavy table rows are already replaced.
	s = replaceUnfencedDiagrams(s)

	// 3. Inline code containing file paths → basename.
	s = reInlineCodePath.ReplaceAllStringFunc(s, func(m string) string {
		inner := m[1 : len(m)-1] // strip backticks
		return extractBasename(inner)
	})

	// 4. Remaining inline code → unwrap backticks.
	s = reInlineCode.ReplaceAllString(s, "$1")

	// 5. Images ![alt](url) → "Image: alt" or just "Image".
	s = reImage.ReplaceAllStringFunc(s, func(m string) string {
		alt := reImage.FindStringSubmatch(m)[1]
		if strings.TrimSpace(alt) == "" {
			return "Image."
		}
		return "Image: " + strings.TrimSpace(alt) + "."
	})

	// 6. Links [text](url) → text only.
	s = reLink.ReplaceAllString(s, "$1")

	// 7. Heading markers: ## Heading → Heading.
	s = reHeading.ReplaceAllString(s, "$1")

	// 8. Bold / italic: **text**, *text*, __text__, _text_.
	// Process bold before italic so ** is matched before *.
	s = reBoldAsterisks.ReplaceAllString(s, "$1")
	s = reBoldUnderscores.ReplaceAllString(s, "$1")
	s = reItalicAsterisks.ReplaceAllString(s, "$1")
	s = reItalicUnderscores.ReplaceAllString(s, "$1")

	// 9. Strikethrough: ~~text~~ → text.
	s = reStrikethrough.ReplaceAllString(s, "$1")

	// 10. Horizontal rules.
	s = reHorizontalRule.ReplaceAllString(s, "")

	// 11. HTML tags: <br>, <hr/>, <div class="x">, etc.
	s = reHTMLTag.ReplaceAllString(s, " ")

	// 12. Blockquote markers: > text → text.
	s = reBlockquote.ReplaceAllString(s, "$1")

	// 13. List markers: strip bullets (add period), keep numbers (add period).
	s = reUnorderedList.ReplaceAllStringFunc(s, stripBulletAddPeriod)
	s = reOrderedList.ReplaceAllStringFunc(s, keepNumberAddPeriod)

	// 14. Bare file paths outside of inline code (e.g. /home/user/project/file.go).
	s = reBareFilePath.ReplaceAllStringFunc(s, extractBasename)

	// 15. Collapse whitespace: multiple blank lines → single, trailing spaces → trimmed.
	s = reMultipleBlankLines.ReplaceAllString(s, "\n\n")
	s = strings.TrimSpace(s)

	return s
}
