package main

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

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

// ---------------------------------------------------------------------------
// Compiled regular expressions
// ---------------------------------------------------------------------------

var (
	// Fenced code block: ```lang\n...\n``` (multiline, non-greedy).
	reFencedCodeBlock = regexp.MustCompile("(?s)```(\\w*)\\n(.*?)\\n```")

	// Markdown table: two or more lines where every line starts & ends with |.
	// The second line must be a separator row (|---|---|).
	reMarkdownTable = regexp.MustCompile(`(?m)^(\|[^\n]+\|\n\|[-| :]+\|\n(?:\|[^\n]+\|\n?)*)`)

	// Inline code containing a path with at least one / separator.
	// [^`\n] prevents matching across line boundaries.
	reInlineCodePath = regexp.MustCompile("`([^`\n]+/[^`\n]+)`")

	// Generic inline code. Must not span lines.
	reInlineCode = regexp.MustCompile("`([^`\n]+)`")

	// Image: ![alt](url)
	reImage = regexp.MustCompile(`!\[([^\]]*)\]\([^)]+\)`)

	// Link: [text](url)
	reLink = regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`)

	// Heading marker: # through ######
	reHeading = regexp.MustCompile(`(?m)^#{1,6}\s+(.*)`)

	// Bold with asterisks: **text**
	reBoldAsterisks = regexp.MustCompile(`\*\*([^*]+)\*\*`)

	// Bold with underscores: __text__
	reBoldUnderscores = regexp.MustCompile(`__([^_]+)__`)

	// Italic with asterisks: *text* (word-bounded to avoid matching **)
	reItalicAsterisks = regexp.MustCompile(`\*([^*]+)\*`)

	// Italic with underscores: _text_
	reItalicUnderscores = regexp.MustCompile(`_([^_]+)_`)

	// Strikethrough: ~~text~~
	reStrikethrough = regexp.MustCompile(`~~([^~]+)~~`)

	// Horizontal rule: ---, ***, ___ (with optional spaces)
	reHorizontalRule = regexp.MustCompile(`(?m)^[\s]*[-*_]{3,}\s*$`)

	// HTML tags (opening, closing, self-closing).
	reHTMLTag = regexp.MustCompile(`<[^>]+>`)

	// Blockquote marker: > at line start.
	reBlockquote = regexp.MustCompile(`(?m)^>\s?(.*)`)

	// Unordered list marker: -, *, + at line start with optional indentation.
	reUnorderedList = regexp.MustCompile(`(?m)^[\t ]*[-*+]\s+(.*)`)

	// Ordered list marker: 1., 2., etc.
	reOrderedList = regexp.MustCompile(`(?m)^[\t ]*\d+\.\s+(.*)`)

	// Ordered list number prefix: captures "1." from "  1. item".
	reOrderedListNumber = regexp.MustCompile(`\d+\.`)

	// Bare file path: starts with / or ./ or ~/ and has at least 2 segments.
	reBareFilePath = regexp.MustCompile(`(?:^|[\s(])([/~](?:\w[\w.-]*/)+[\w.-]+)`)

	// Multiple blank lines → collapse to one.
	reMultipleBlankLines = regexp.MustCompile(`\n{3,}`)
)

// ---------------------------------------------------------------------------
// Helper functions
// ---------------------------------------------------------------------------

// summarizeCodeBlock replaces a fenced code block with a short description.
func summarizeCodeBlock(match string) string {
	parts := reFencedCodeBlock.FindStringSubmatch(match)
	if len(parts) < 3 {
		return "Code block."
	}
	lang := parts[1]
	body := parts[2]
	lineCount := strings.Count(body, "\n") + 1

	if lang != "" {
		return fmt.Sprintf("Code block: %d lines of %s.", lineCount, lang)
	}
	return fmt.Sprintf("Code block: %d lines.", lineCount)
}

// describeTable converts a markdown table into a prose summary.
// Example output: "Table with 3 columns: Name, Age, City. 2 data rows."
func describeTable(match string) string {
	lines := strings.Split(strings.TrimRight(match, "\n"), "\n")
	if len(lines) < 2 {
		return match
	}

	// Parse header row.
	headers := parseTableRow(lines[0])
	if len(headers) == 0 {
		return match
	}

	// Count data rows (skip header and separator).
	dataRows := 0
	for i := 2; i < len(lines); i++ {
		row := strings.TrimSpace(lines[i])
		if row != "" && strings.Contains(row, "|") {
			dataRows++
		}
	}

	colList := strings.Join(headers, ", ")
	rowWord := "row"
	if dataRows != 1 {
		rowWord = "rows"
	}

	result := fmt.Sprintf("Table with %d columns: %s. %d data %s.",
		len(headers), colList, dataRows, rowWord)

	// Optionally include first few data rows as a preview.
	if dataRows > 0 && dataRows <= 5 {
		var previews []string
		for i := 2; i < len(lines) && len(previews) < 3; i++ {
			cells := parseTableRow(lines[i])
			if len(cells) > 0 {
				previews = append(previews, strings.Join(cells, ", "))
			}
		}
		if len(previews) > 0 {
			result += " Items include: " + strings.Join(previews, "; ") + "."
		}
	}

	return result + "\n"
}

// parseTableRow splits a markdown table row into trimmed cell values.
// Example: "| foo | bar |" → ["foo", "bar"].
func parseTableRow(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.Trim(line, "|")
	parts := strings.Split(line, "|")

	var cells []string
	for _, p := range parts {
		cell := strings.TrimSpace(p)
		// Skip separator cells (---, :--:, etc.)
		if cell != "" && !isSeparatorCell(cell) {
			cells = append(cells, cell)
		}
	}
	return cells
}

// isSeparatorCell returns true if a cell is a table separator like "---" or ":--:".
func isSeparatorCell(cell string) bool {
	for _, ch := range cell {
		if ch != '-' && ch != ':' && ch != ' ' {
			return false
		}
	}
	return true
}

// ---------------------------------------------------------------------------
// Unfenced diagram / wireframe detection
// ---------------------------------------------------------------------------

// minDiagramRunLength is the minimum number of consecutive diagram-scored lines
// required to trigger replacement. Shorter runs are left alone to avoid
// false-positives on isolated decorative lines.
const minDiagramRunLength = 3

// diagramScoreThreshold is the minimum per-line score (0–1) to consider a line
// part of a diagram region.
const diagramScoreThreshold = 0.5

// replaceUnfencedDiagrams finds runs of diagram/wireframe lines not inside
// fenced code blocks and replaces each run with a speech-friendly summary.
func replaceUnfencedDiagrams(text string) string {
	lines := strings.Split(text, "\n")
	scores := make([]float64, len(lines))
	for i, line := range lines {
		scores[i] = diagramLineScore(line)
	}

	// Find runs of consecutive high-scoring lines, allowing a single
	// low-scoring "bridge" line between two high-scoring lines.
	type region struct{ start, end int } // [start, end)
	var regions []region

	i := 0
	for i < len(lines) {
		if scores[i] < diagramScoreThreshold {
			i++
			continue
		}
		// Start of a potential run.
		start := i
		i++
		for i < len(lines) {
			if scores[i] >= diagramScoreThreshold {
				i++
				continue
			}
			// Allow a 1-line bridge if the next line is high-scoring.
			if i+1 < len(lines) && scores[i+1] >= diagramScoreThreshold {
				i += 2
				continue
			}
			break
		}
		if i-start >= minDiagramRunLength {
			regions = append(regions, region{start, i})
		}
	}

	if len(regions) == 0 {
		return text
	}

	// Replace regions back-to-front to preserve line indices.
	for ri := len(regions) - 1; ri >= 0; ri-- {
		r := regions[ri]
		summary := classifyDiagramRegion(lines[r.start:r.end])
		newLines := make([]string, 0, len(lines)-(r.end-r.start)+1)
		newLines = append(newLines, lines[:r.start]...)
		newLines = append(newLines, summary)
		newLines = append(newLines, lines[r.end:]...)
		lines = newLines
	}

	return strings.Join(lines, "\n")
}

// diagramLineScore returns 0.0–1.0 indicating how "diagrammatic" a line is.
func diagramLineScore(line string) float64 {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return 0
	}

	score := 0.0

	// Box-drawing Unicode characters (high signal).
	boxCount := 0
	for _, r := range trimmed {
		if isBoxDrawingRune(r) {
			boxCount++
		}
	}
	if boxCount >= 2 {
		score = maxFloat(score, 0.9)
	}

	// Tree connectors: ├──, └──, │ (high signal).
	if reTreeConnector.MatchString(trimmed) {
		score = maxFloat(score, 0.85)
	}

	// Mermaid keyword anchor (high signal).
	if reMermaidKeyword.MatchString(trimmed) {
		score = maxFloat(score, 0.9)
	}

	// Mermaid body line: brackets/braces combined with an arrow (high signal).
	// e.g. "A[Start] --> B{Decision}"
	hasBrackets := strings.ContainsAny(trimmed, "[]{}")
	hasArrow := reDiagramArrow.MatchString(trimmed)
	if hasBrackets && hasArrow {
		score = maxFloat(score, 0.8)
	}

	// Arrow patterns: -->, ==>, ->, => (medium/high signal).
	arrowCount := len(reDiagramArrow.FindAllString(trimmed, -1))
	if arrowCount >= 2 {
		score = maxFloat(score, 0.8)
	} else if arrowCount == 1 {
		score = maxFloat(score, 0.5)
	}

	// Pipe-bordered line: starts and ends with | (common in wireframes).
	// Tables are already replaced before diagram detection runs.
	if len(trimmed) >= 3 && trimmed[0] == '|' && trimmed[len(trimmed)-1] == '|' {
		score = maxFloat(score, 0.7)
	}

	// Structural character ratio — lines dominated by +|-=<>/\^_ and similar.
	structural, total := 0, 0
	for _, r := range trimmed {
		if !unicode.IsSpace(r) {
			total++
			if isDiagramStructuralChar(r) {
				structural++
			}
		}
	}
	if total > 0 {
		ratio := float64(structural) / float64(total)
		if ratio > 0.6 {
			score = maxFloat(score, 0.8)
		} else if ratio > 0.4 {
			score = maxFloat(score, 0.5)
		}
	}

	// ASCII box junction pattern: +---+ or +===+ (high signal).
	if reASCIIBoxJunction.MatchString(trimmed) {
		score = maxFloat(score, 0.9)
	}

	return score
}

// classifyDiagramRegion examines the lines in a diagram run and returns an
// appropriate speech-friendly summary string.
func classifyDiagramRegion(lines []string) string {
	block := strings.Join(lines, "\n")
	lineCount := len(lines)

	// Count signal types across the region.
	boxDrawingLines := 0
	treeLines := 0
	mermaidAnchor := ""
	arrowLines := 0
	asciiBoxLines := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Box drawing.
		boxCount := 0
		for _, r := range trimmed {
			if isBoxDrawingRune(r) {
				boxCount++
			}
		}
		if boxCount >= 2 {
			boxDrawingLines++
		}

		// ASCII box junctions.
		if reASCIIBoxJunction.MatchString(trimmed) {
			asciiBoxLines++
		}

		// Tree connectors.
		if reTreeConnector.MatchString(trimmed) {
			treeLines++
		}

		// Mermaid keyword.
		if mermaidAnchor == "" {
			if m := reMermaidKeyword.FindStringSubmatch(trimmed); len(m) > 1 {
				mermaidAnchor = m[1]
			}
		}

		// Arrows.
		if reDiagramArrow.MatchString(trimmed) {
			arrowLines++
		}
	}

	// Classify based on dominant signal.
	switch {
	case mermaidAnchor != "":
		return fmt.Sprintf("Diagram: %s with %d lines.", normalizeMermaidType(mermaidAnchor), lineCount)
	case treeLines > lineCount/2:
		entries := countTreeEntries(block)
		return fmt.Sprintf("File tree with %d entries.", entries)
	case boxDrawingLines > lineCount/3 || asciiBoxLines > lineCount/3:
		return "Wireframe diagram."
	case arrowLines > lineCount/3:
		return fmt.Sprintf("Flow diagram with %d lines.", lineCount)
	default:
		return "Diagram."
	}
}

// normalizeMermaidType maps mermaid keyword prefixes to human-readable names.
func normalizeMermaidType(keyword string) string {
	kw := strings.ToLower(strings.TrimSpace(keyword))
	switch {
	case strings.HasPrefix(kw, "graph"):
		return "graph"
	case strings.HasPrefix(kw, "flowchart"):
		return "flowchart"
	case strings.HasPrefix(kw, "sequencediagram"):
		return "sequence diagram"
	case strings.HasPrefix(kw, "classdiagram"):
		return "class diagram"
	case strings.HasPrefix(kw, "statediagram"):
		return "state diagram"
	case strings.HasPrefix(kw, "erdiagram"):
		return "entity relationship diagram"
	case strings.HasPrefix(kw, "gantt"):
		return "gantt chart"
	case strings.HasPrefix(kw, "pie"):
		return "pie chart"
	case strings.HasPrefix(kw, "gitgraph"):
		return "git graph"
	case strings.HasPrefix(kw, "journey"):
		return "user journey"
	default:
		return kw + " diagram"
	}
}

// countTreeEntries counts non-empty, non-blank lines in a tree diagram,
// which roughly corresponds to the number of files/directories listed.
func countTreeEntries(block string) int {
	count := 0
	for _, line := range strings.Split(block, "\n") {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}

// isBoxDrawingRune returns true for Unicode box-drawing characters
// (U+2500–U+257F) and related line-drawing symbols.
func isBoxDrawingRune(r rune) bool {
	return r >= 0x2500 && r <= 0x257F
}

// isDiagramStructuralChar returns true for characters commonly used to draw
// ASCII diagrams: box corners, lines, arrows, pipes.
func isDiagramStructuralChar(r rune) bool {
	switch r {
	case '+', '-', '|', '=', '<', '>', '/', '\\', '^', '_', '~',
		'[', ']', '{', '}', '(', ')',
		'→', '←', '↑', '↓', '⇒', '⇐':
		return true
	}
	return isBoxDrawingRune(r)
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// ---------------------------------------------------------------------------
// Diagram-related compiled regular expressions
// ---------------------------------------------------------------------------

var (
	// Tree connectors: ├──, └──, │ (with optional leading whitespace).
	reTreeConnector = regexp.MustCompile(`^[\s│|]*[├└┬┤┼][─]+|^[\s]*[|][\s]*[` + "`" + `|+\\]?[\s]*[-─]+`)

	// Mermaid keywords that anchor a diagram.
	reMermaidKeyword = regexp.MustCompile(`(?i)^\s*(graph\s+[A-Z]{2}|flowchart\s+[A-Z]{2}|sequenceDiagram|classDiagram|stateDiagram|erDiagram|gantt|pie|gitgraph|journey)\b`)

	// Arrow patterns common in diagrams.
	reDiagramArrow = regexp.MustCompile(`-->>|--?>|==>|<-->|<=>|<->|<--|->|=>|->>|→|←|↔|⇒|⇐`)

	// ASCII box junction: lines like +---+, +===+, +---+---+.
	reASCIIBoxJunction = regexp.MustCompile(`\+[-=]+\+`)
)

// endsWithSentencePunctuation reports whether s ends with . ! ? or :
func endsWithSentencePunctuation(s string) bool {
	if s == "" {
		return false
	}
	switch s[len(s)-1] {
	case '.', '!', '?', ':':
		return true
	}
	return false
}

// stripBulletAddPeriod removes a bullet marker (-, *, +) and appends a period
// if the item doesn't already end with sentence punctuation.
// "- Install the package" → "Install the package."
func stripBulletAddPeriod(match string) string {
	item := strings.TrimSpace(reUnorderedList.FindStringSubmatch(match)[1])
	if item == "" {
		return ""
	}
	if !endsWithSentencePunctuation(item) {
		item += "."
	}
	return item
}

// keepNumberAddPeriod preserves the number prefix and appends a period if the
// item doesn't already end with sentence punctuation.
// "1. Install the package" → "1. Install the package."
func keepNumberAddPeriod(match string) string {
	sub := reOrderedList.FindStringSubmatch(match)
	item := strings.TrimSpace(sub[1])
	if item == "" {
		return ""
	}
	// Extract the number from the original match.
	numStr := strings.TrimSpace(reOrderedListNumber.FindString(match))
	if !endsWithSentencePunctuation(item) {
		item += "."
	}
	return numStr + " " + item
}

// extractBasename returns the last path segment of a file path.
// "/home/user/project/src/main.go" → "main.go"
func extractBasename(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return path
	}
	// Find last slash.
	idx := strings.LastIndex(path, "/")
	if idx < 0 || idx == len(path)-1 {
		return path
	}
	return path[idx+1:]
}
