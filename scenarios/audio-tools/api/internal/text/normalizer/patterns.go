package normalizer

import "regexp"

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
