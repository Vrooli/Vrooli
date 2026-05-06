package main

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Fenced code blocks
// ---------------------------------------------------------------------------

func TestNormalize_FencedCodeBlock_WithLanguage(t *testing.T) {
	input := "Here is some code:\n```go\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n```\nDone."
	got := NormalizeTextForSpeech(input)
	if !strings.Contains(got, "Code block: 3 lines of go.") {
		t.Errorf("expected code block summary with language, got:\n%s", got)
	}
	if strings.Contains(got, "func main") {
		t.Error("code block body should be removed")
	}
	if !strings.Contains(got, "Done.") {
		t.Error("prose after code block should be preserved")
	}
}

func TestNormalize_FencedCodeBlock_NoLanguage(t *testing.T) {
	input := "```\nline1\nline2\n```"
	got := NormalizeTextForSpeech(input)
	if !strings.Contains(got, "Code block: 2 lines.") {
		t.Errorf("expected code block summary without language, got:\n%s", got)
	}
}

func TestNormalize_FencedCodeBlock_JSON(t *testing.T) {
	input := "Response:\n```json\n{\n  \"key\": \"value\",\n  \"count\": 42\n}\n```"
	got := NormalizeTextForSpeech(input)
	if !strings.Contains(got, "Code block: 4 lines of json.") {
		t.Errorf("expected JSON code block summary, got:\n%s", got)
	}
	if strings.Contains(got, `"key"`) {
		t.Error("JSON content should be removed")
	}
}

func TestNormalize_MultipleFencedCodeBlocks(t *testing.T) {
	input := "First:\n```py\nprint(1)\n```\nSecond:\n```js\nconsole.log(2)\n```"
	got := NormalizeTextForSpeech(input)
	if strings.Count(got, "Code block:") != 2 {
		t.Errorf("expected 2 code block summaries, got:\n%s", got)
	}
}

// ---------------------------------------------------------------------------
// Markdown tables
// ---------------------------------------------------------------------------

func TestNormalize_SimpleTable(t *testing.T) {
	input := `| Name | Age | City |
| --- | --- | --- |
| Alice | 30 | NYC |
| Bob | 25 | LA |`
	got := NormalizeTextForSpeech(input)
	if !strings.Contains(got, "Table with 3 columns: Name, Age, City.") {
		t.Errorf("expected table header summary, got:\n%s", got)
	}
	if !strings.Contains(got, "2 data rows.") {
		t.Errorf("expected 2 data rows, got:\n%s", got)
	}
	if strings.Contains(got, "|") {
		t.Error("pipe characters should be removed from table output")
	}
}

func TestNormalize_TableWithPreview(t *testing.T) {
	input := `| Col1 | Col2 |
| --- | --- |
| A | B |`
	got := NormalizeTextForSpeech(input)
	if !strings.Contains(got, "Items include:") {
		t.Errorf("expected row preview for small tables, got:\n%s", got)
	}
	if !strings.Contains(got, "A, B") {
		t.Errorf("expected cell values in preview, got:\n%s", got)
	}
}

// ---------------------------------------------------------------------------
// File paths
// ---------------------------------------------------------------------------

func TestNormalize_InlineCodeFilePath(t *testing.T) {
	input := "Edit the file `/home/user/project/src/components/Header.tsx` to fix the bug."
	got := NormalizeTextForSpeech(input)
	if !strings.Contains(got, "Header.tsx") {
		t.Errorf("expected basename Header.tsx, got:\n%s", got)
	}
	if strings.Contains(got, "/home/user") {
		t.Error("full path should be replaced with basename")
	}
}

func TestNormalize_RelativeInlineCodePath(t *testing.T) {
	input := "See `src/lib/utils/helpers.ts` for details."
	got := NormalizeTextForSpeech(input)
	if !strings.Contains(got, "helpers.ts") {
		t.Errorf("expected basename, got:\n%s", got)
	}
}

func TestNormalize_InlineCodeNotAPath(t *testing.T) {
	input := "Use the `useState` hook."
	got := NormalizeTextForSpeech(input)
	if !strings.Contains(got, "useState") {
		t.Errorf("expected inline code content preserved, got:\n%s", got)
	}
	if strings.Contains(got, "`") {
		t.Error("backticks should be removed")
	}
}

func TestNormalize_InlineCodeDoesNotSpanLines(t *testing.T) {
	// Regression: reInlineCodePath matched from `service.json` across many
	// lines to `status` because [^`] allows newlines. This swallowed entire
	// sections of the document.
	input := "It generates a `service.json` config file.\n\n" +
		"## Tech Stack\n\n" +
		"- **API**: Go, gorilla/mux, PostgreSQL\n" +
		"- **CLI**: Go CLI for `status` and `configure` commands"
	got := NormalizeTextForSpeech(input)
	if strings.Contains(got, "jsonmux") || strings.Contains(got, "service.jsonmux") {
		t.Errorf("inline code regex must not span lines, got:\n%s", got)
	}
	if !strings.Contains(got, "service.json") {
		t.Errorf("service.json should be preserved, got:\n%s", got)
	}
	if !strings.Contains(got, "gorilla") {
		t.Errorf("gorilla/mux text should be preserved, got:\n%s", got)
	}
	if !strings.Contains(got, "status") {
		t.Errorf("status should be preserved, got:\n%s", got)
	}
}

func TestNormalize_BareFilePath(t *testing.T) {
	input := "The config is at /etc/nginx/sites-available/default for reference."
	got := NormalizeTextForSpeech(input)
	if !strings.Contains(got, "default") {
		t.Errorf("expected basename 'default', got:\n%s", got)
	}
}

// ---------------------------------------------------------------------------
// Heading markers
// ---------------------------------------------------------------------------

func TestNormalize_HeadingMarkers(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"# Title", "Title"},
		{"## Section", "Section"},
		{"###### Deep heading", "Deep heading"},
	}
	for _, tc := range tests {
		got := NormalizeTextForSpeech(tc.input)
		if strings.TrimSpace(got) != tc.want {
			t.Errorf("NormalizeTextForSpeech(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Bold / italic / strikethrough
// ---------------------------------------------------------------------------

func TestNormalize_Bold(t *testing.T) {
	got := NormalizeTextForSpeech("This is **important** text.")
	if strings.Contains(got, "**") {
		t.Error("bold markers should be removed")
	}
	if !strings.Contains(got, "important") {
		t.Error("bold text content should be preserved")
	}
}

func TestNormalize_BoldUnderscores(t *testing.T) {
	got := NormalizeTextForSpeech("This is __important__ text.")
	if strings.Contains(got, "__") {
		t.Error("bold underscore markers should be removed")
	}
	if !strings.Contains(got, "important") {
		t.Error("bold text content should be preserved")
	}
}

func TestNormalize_Italic(t *testing.T) {
	got := NormalizeTextForSpeech("This is *emphasized* text.")
	if got != "This is emphasized text." {
		t.Errorf("expected italic unwrapped, got: %q", got)
	}
}

func TestNormalize_Strikethrough(t *testing.T) {
	got := NormalizeTextForSpeech("This is ~~deleted~~ text.")
	if strings.Contains(got, "~~") {
		t.Error("strikethrough markers should be removed")
	}
	if !strings.Contains(got, "deleted") {
		t.Error("strikethrough content should be preserved")
	}
}

// ---------------------------------------------------------------------------
// Links and images
// ---------------------------------------------------------------------------

func TestNormalize_Link(t *testing.T) {
	got := NormalizeTextForSpeech("See [the docs](https://example.com/docs) for more.")
	if !strings.Contains(got, "the docs") {
		t.Error("link text should be preserved")
	}
	if strings.Contains(got, "https://") {
		t.Error("URL should be removed")
	}
	if strings.Contains(got, "[") || strings.Contains(got, "]") {
		t.Error("bracket characters should be removed")
	}
}

func TestNormalize_Image(t *testing.T) {
	got := NormalizeTextForSpeech("Here: ![screenshot of dashboard](http://img.png)")
	if !strings.Contains(got, "Image: screenshot of dashboard.") {
		t.Errorf("expected image description, got:\n%s", got)
	}
}

func TestNormalize_ImageNoAlt(t *testing.T) {
	got := NormalizeTextForSpeech("![](http://img.png)")
	if !strings.Contains(got, "Image.") {
		t.Errorf("expected 'Image.' for no-alt image, got:\n%s", got)
	}
}

// ---------------------------------------------------------------------------
// Horizontal rules
// ---------------------------------------------------------------------------

func TestNormalize_HorizontalRule(t *testing.T) {
	input := "Above\n\n---\n\nBelow"
	got := NormalizeTextForSpeech(input)
	if strings.Contains(got, "---") {
		t.Error("horizontal rule should be removed")
	}
	if !strings.Contains(got, "Above") || !strings.Contains(got, "Below") {
		t.Error("surrounding text should be preserved")
	}
}

// ---------------------------------------------------------------------------
// HTML tags
// ---------------------------------------------------------------------------

func TestNormalize_HTMLTags(t *testing.T) {
	input := "Line one.<br>Line two.<hr/>Done."
	got := NormalizeTextForSpeech(input)
	if strings.Contains(got, "<br>") || strings.Contains(got, "<hr/>") {
		t.Error("HTML tags should be removed")
	}
	if !strings.Contains(got, "Line one.") || !strings.Contains(got, "Done.") {
		t.Error("surrounding text should be preserved")
	}
}

// ---------------------------------------------------------------------------
// Blockquotes
// ---------------------------------------------------------------------------

func TestNormalize_Blockquote(t *testing.T) {
	input := "> This is a quote.\n> Second line."
	got := NormalizeTextForSpeech(input)
	if strings.Contains(got, ">") {
		t.Error("blockquote markers should be removed")
	}
	if !strings.Contains(got, "This is a quote.") {
		t.Error("blockquote content should be preserved")
	}
}

// ---------------------------------------------------------------------------
// List markers
// ---------------------------------------------------------------------------

func TestNormalize_UnorderedList(t *testing.T) {
	input := "- First item\n- Second item\n* Third item"
	got := NormalizeTextForSpeech(input)
	if strings.Contains(got, "- First") || strings.Contains(got, "* Third") {
		t.Errorf("list markers should be removed, got:\n%s", got)
	}
	if !strings.Contains(got, "First item.") || !strings.Contains(got, "Third item.") {
		t.Errorf("list items should have periods appended, got:\n%s", got)
	}
}

func TestNormalize_UnorderedList_ExistingPunctuation(t *testing.T) {
	input := "- Already has a period.\n- Has exclamation!\n- Has question?\n- Has colon:\n- No punctuation"
	got := NormalizeTextForSpeech(input)
	if strings.Contains(got, "period..") {
		t.Errorf("should not double-up periods, got:\n%s", got)
	}
	if strings.Contains(got, "exclamation!.") {
		t.Errorf("should not add period after !, got:\n%s", got)
	}
	if strings.Contains(got, "question?.") {
		t.Errorf("should not add period after ?, got:\n%s", got)
	}
	if strings.Contains(got, "colon:.") {
		t.Errorf("should not add period after :, got:\n%s", got)
	}
	if !strings.Contains(got, "No punctuation.") {
		t.Errorf("should add period when missing, got:\n%s", got)
	}
}

func TestNormalize_UnorderedList_IndentedItems(t *testing.T) {
	input := "- Top level\n  - Indented item\n    - Deep item"
	got := NormalizeTextForSpeech(input)
	if !strings.Contains(got, "Top level.") {
		t.Errorf("top-level item should get period, got:\n%s", got)
	}
	if !strings.Contains(got, "Indented item.") {
		t.Errorf("indented item should get period, got:\n%s", got)
	}
	if !strings.Contains(got, "Deep item.") {
		t.Errorf("deeply indented item should get period, got:\n%s", got)
	}
}

func TestNormalize_UnorderedList_PlusMarker(t *testing.T) {
	input := "+ Item one\n+ Item two"
	got := NormalizeTextForSpeech(input)
	if strings.Contains(got, "+") {
		t.Errorf("+ markers should be removed, got:\n%s", got)
	}
	if !strings.Contains(got, "Item one.") || !strings.Contains(got, "Item two.") {
		t.Errorf("items should have periods, got:\n%s", got)
	}
}

func TestNormalize_OrderedList(t *testing.T) {
	input := "1. Alpha\n2. Beta\n3. Gamma"
	got := NormalizeTextForSpeech(input)
	if !strings.Contains(got, "1. Alpha.") {
		t.Errorf("expected '1. Alpha.', got:\n%s", got)
	}
	if !strings.Contains(got, "2. Beta.") {
		t.Errorf("expected '2. Beta.', got:\n%s", got)
	}
	if !strings.Contains(got, "3. Gamma.") {
		t.Errorf("expected '3. Gamma.', got:\n%s", got)
	}
}

func TestNormalize_OrderedList_ExistingPunctuation(t *testing.T) {
	input := "1. Already done.\n2. Also done!\n3. Really?\n4. No punctuation"
	got := NormalizeTextForSpeech(input)
	if strings.Contains(got, "done..") {
		t.Errorf("should not double-up periods, got:\n%s", got)
	}
	if strings.Contains(got, "done!.") {
		t.Errorf("should not add period after !, got:\n%s", got)
	}
	if !strings.Contains(got, "4. No punctuation.") {
		t.Errorf("should add period when missing, got:\n%s", got)
	}
}

func TestNormalize_OrderedList_PreservesNumbers(t *testing.T) {
	input := "1. First step\n2. Second step\n3. Third step"
	got := NormalizeTextForSpeech(input)
	// Numbers should be preserved for natural reading
	if !strings.Contains(got, "1.") || !strings.Contains(got, "2.") || !strings.Contains(got, "3.") {
		t.Errorf("ordered list numbers should be preserved, got:\n%s", got)
	}
}

// ---------------------------------------------------------------------------
// Whitespace collapse
// ---------------------------------------------------------------------------

func TestNormalize_CollapseBlankLines(t *testing.T) {
	input := "First.\n\n\n\n\nSecond."
	got := NormalizeTextForSpeech(input)
	if strings.Contains(got, "\n\n\n") {
		t.Error("should collapse multiple blank lines")
	}
	if !strings.Contains(got, "First.") || !strings.Contains(got, "Second.") {
		t.Error("content should be preserved")
	}
}

// ---------------------------------------------------------------------------
// Combined / integration tests
// ---------------------------------------------------------------------------

func TestNormalize_RealWorldAssistantMessage(t *testing.T) {
	input := `## Summary

I've updated the configuration file at ` + "`/tmp/vrooli-test/scenarios/web-console/api/config.go`" + ` to fix the issue.

Here are the changes:

` + "```go" + `
func NewConfig() *Config {
	return &Config{
		Port: 8080,
	}
}
` + "```" + `

### Key changes:

- **Port** is now configurable
- ~~Hard-coded values~~ have been removed
- See [the PR](https://github.com/example/pr/123) for details

| Setting | Before | After |
| --- | --- | --- |
| Port | 3000 | 8080 |
| Debug | true | false |`

	got := NormalizeTextForSpeech(input)

	// Heading markers stripped.
	if strings.Contains(got, "##") {
		t.Error("heading markers should be stripped")
	}
	if !strings.Contains(got, "Summary") {
		t.Error("heading text should be preserved")
	}

	// File path reduced to basename.
	if strings.Contains(got, "/tmp/vrooli-test") {
		t.Error("full file path should be replaced with basename")
	}
	if !strings.Contains(got, "config.go") {
		t.Error("basename should be present")
	}

	// Code block replaced with summary.
	if strings.Contains(got, "func NewConfig") {
		t.Error("code block content should be removed")
	}
	if !strings.Contains(got, "Code block:") {
		t.Error("code block summary should be present")
	}

	// Bold markers removed.
	if strings.Contains(got, "**") {
		t.Error("bold markers should be removed")
	}

	// Strikethrough removed.
	if strings.Contains(got, "~~") {
		t.Error("strikethrough markers should be removed")
	}

	// Link text preserved, URL removed.
	if !strings.Contains(got, "the PR") {
		t.Error("link text should be preserved")
	}
	if strings.Contains(got, "https://github.com") {
		t.Error("URL should be removed")
	}

	// Table converted to prose.
	if strings.Contains(got, "|") {
		t.Error("table pipe characters should be removed")
	}
	if !strings.Contains(got, "Table with") {
		t.Error("table summary should be present")
	}

	// List markers removed.
	if strings.HasPrefix(strings.TrimSpace(got), "-") {
		t.Error("list markers should be removed")
	}
}

func TestNormalize_EmptyInput(t *testing.T) {
	got := NormalizeTextForSpeech("")
	if got != "" {
		t.Errorf("expected empty output for empty input, got: %q", got)
	}
}

func TestNormalize_PlainText(t *testing.T) {
	input := "Just some plain text with no markdown."
	got := NormalizeTextForSpeech(input)
	if got != input {
		t.Errorf("plain text should pass through unchanged, got: %q", got)
	}
}

// ---------------------------------------------------------------------------
// Unfenced diagram / wireframe detection
// ---------------------------------------------------------------------------

// --- diagramLineScore unit tests ---

func TestDiagramLineScore_EmptyLine(t *testing.T) {
	if s := diagramLineScore(""); s != 0 {
		t.Errorf("empty line should score 0, got %.2f", s)
	}
	if s := diagramLineScore("   "); s != 0 {
		t.Errorf("whitespace-only line should score 0, got %.2f", s)
	}
}

func TestDiagramLineScore_PlainProse(t *testing.T) {
	lines := []string{
		"This is just a regular sentence.",
		"The quick brown fox jumps over the lazy dog.",
		"Here are some numbers: 42, 100, 256.",
	}
	for _, line := range lines {
		s := diagramLineScore(line)
		if s >= diagramScoreThreshold {
			t.Errorf("prose line should score below threshold, got %.2f for %q", s, line)
		}
	}
}

func TestDiagramLineScore_UnicodeBoxDrawing(t *testing.T) {
	tests := []struct {
		name string
		line string
	}{
		{"top edge", "┌─────────────────┐"},
		{"bottom edge", "└────────┴────────┘"},
		{"divider", "├────────┬────────┤"},
		{"content row", "│ Sidebar│Content │"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := diagramLineScore(tc.line)
			if s < diagramScoreThreshold {
				t.Errorf("box-drawing line should score high, got %.2f for %q", s, tc.line)
			}
		})
	}
}

func TestDiagramLineScore_ASCIIBoxJunctions(t *testing.T) {
	tests := []string{
		"+-------------------+",
		"+===+===+===+",
		"+--+--+",
	}
	for _, line := range tests {
		s := diagramLineScore(line)
		if s < 0.9 {
			t.Errorf("ASCII box junction should score >= 0.9, got %.2f for %q", s, line)
		}
	}
}

func TestDiagramLineScore_TreeConnectors(t *testing.T) {
	tests := []string{
		"├── src/",
		"│   ├── components/",
		"│   │   └── Button.tsx",
		"└── package.json",
	}
	for _, line := range tests {
		s := diagramLineScore(line)
		if s < diagramScoreThreshold {
			t.Errorf("tree connector line should score high, got %.2f for %q", s, line)
		}
	}
}

func TestDiagramLineScore_MermaidKeywords(t *testing.T) {
	tests := []string{
		"graph TD",
		"graph LR",
		"flowchart LR",
		"sequenceDiagram",
		"classDiagram",
		"stateDiagram",
		"erDiagram",
		"gantt",
		"pie",
		"gitgraph",
		"journey",
	}
	for _, line := range tests {
		s := diagramLineScore(line)
		if s < 0.9 {
			t.Errorf("mermaid keyword should score >= 0.9, got %.2f for %q", s, line)
		}
	}
}

func TestDiagramLineScore_ArrowPatterns(t *testing.T) {
	tests := []struct {
		line    string
		minReqs float64
	}{
		{"A --> B --> C", diagramScoreThreshold},
		{"Start ==> Middle ==> End", diagramScoreThreshold},
		{"Input -> Process -> Output", diagramScoreThreshold},
	}
	for _, tc := range tests {
		s := diagramLineScore(tc.line)
		if s < tc.minReqs {
			t.Errorf("arrow line should score >= %.2f, got %.2f for %q", tc.minReqs, s, tc.line)
		}
	}
}

func TestDiagramLineScore_PipeBorderedLine(t *testing.T) {
	// After table normalization, remaining pipe-bordered lines are wireframe content.
	tests := []string{
		"|  Sidebar | Content|",
		"|          |        |",
		"| Header              |",
	}
	for _, line := range tests {
		s := diagramLineScore(line)
		if s < diagramScoreThreshold {
			t.Errorf("pipe-bordered line should score above threshold, got %.2f for %q", s, line)
		}
	}
}

func TestDiagramLineScore_MermaidBodyLine(t *testing.T) {
	tests := []string{
		"    A[Start] --> B{Decision}",
		"    B -->|Yes| C[Action]",
		"    Client[Browser] -> Server[API]",
	}
	for _, line := range tests {
		s := diagramLineScore(line)
		if s < diagramScoreThreshold {
			t.Errorf("mermaid body line should score above threshold, got %.2f for %q", s, line)
		}
	}
}

func TestDiagramLineScore_HighStructuralRatio(t *testing.T) {
	tests := []string{
		"+----------+--------+",
		"==========",
		"<-------->",
	}
	for _, line := range tests {
		s := diagramLineScore(line)
		if s < diagramScoreThreshold {
			t.Errorf("high-structural line should score above threshold, got %.2f for %q", s, line)
		}
	}
}

// --- replaceUnfencedDiagrams unit tests ---

func TestReplaceDiagrams_UnicodeBoxWireframe(t *testing.T) {
	input := "Here's the layout:\n\n" +
		"┌─────────────────┐\n" +
		"│     Header      │\n" +
		"├────────┬────────┤\n" +
		"│ Sidebar│Content │\n" +
		"└────────┴────────┘\n\n" +
		"That's the wireframe."

	got := replaceUnfencedDiagrams(input)

	if strings.Contains(got, "┌") || strings.Contains(got, "│") || strings.Contains(got, "└") {
		t.Errorf("box-drawing characters should be removed, got:\n%s", got)
	}
	if !strings.Contains(got, "Wireframe diagram.") {
		t.Errorf("expected 'Wireframe diagram.', got:\n%s", got)
	}
	if !strings.Contains(got, "Here's the layout:") || !strings.Contains(got, "That's the wireframe.") {
		t.Error("surrounding prose should be preserved")
	}
}

func TestReplaceDiagrams_ASCIIBoxWireframe(t *testing.T) {
	input := "The page layout:\n\n" +
		"+-------------------+\n" +
		"|     Header        |\n" +
		"+-------------------+\n" +
		"|  Sidebar | Content|\n" +
		"|          |        |\n" +
		"+-------------------+\n\n" +
		"End of wireframe."

	got := replaceUnfencedDiagrams(input)

	if strings.Contains(got, "+---") {
		t.Errorf("ASCII box should be removed, got:\n%s", got)
	}
	if !strings.Contains(got, "Wireframe diagram.") {
		t.Errorf("expected 'Wireframe diagram.', got:\n%s", got)
	}
	if !strings.Contains(got, "The page layout:") || !strings.Contains(got, "End of wireframe.") {
		t.Error("surrounding prose should be preserved")
	}
}

func TestReplaceDiagrams_FileTree(t *testing.T) {
	input := "Project structure:\n\n" +
		"├── src/\n" +
		"│   ├── components/\n" +
		"│   │   ├── Button.tsx\n" +
		"│   │   └── Header.tsx\n" +
		"│   └── utils/\n" +
		"└── package.json\n\n" +
		"That's the tree."

	got := replaceUnfencedDiagrams(input)

	if strings.Contains(got, "├") || strings.Contains(got, "└") {
		t.Errorf("tree connectors should be removed, got:\n%s", got)
	}
	if !strings.Contains(got, "File tree with 6 entries.") {
		t.Errorf("expected file tree summary with 6 entries, got:\n%s", got)
	}
	if !strings.Contains(got, "Project structure:") || !strings.Contains(got, "That's the tree.") {
		t.Error("surrounding prose should be preserved")
	}
}

func TestReplaceDiagrams_MermaidFlowchart(t *testing.T) {
	input := "Here is the flow:\n\n" +
		"graph TD\n" +
		"    A[Start] --> B{Decision}\n" +
		"    B -->|Yes| C[Action]\n" +
		"    B -->|No| D[End]\n\n" +
		"That's the diagram."

	got := replaceUnfencedDiagrams(input)

	if strings.Contains(got, "graph TD") || strings.Contains(got, "-->") {
		t.Errorf("mermaid content should be removed, got:\n%s", got)
	}
	if !strings.Contains(got, "Diagram: graph with") {
		t.Errorf("expected mermaid graph summary, got:\n%s", got)
	}
	if !strings.Contains(got, "Here is the flow:") || !strings.Contains(got, "That's the diagram.") {
		t.Error("surrounding prose should be preserved")
	}
}

func TestReplaceDiagrams_MermaidSequenceDiagram(t *testing.T) {
	input := "sequenceDiagram\n" +
		"    Alice->>John: Hello John\n" +
		"    John-->>Alice: Hi Alice\n" +
		"    Alice->>John: How are you?\n"

	got := replaceUnfencedDiagrams(input)

	if !strings.Contains(got, "Diagram: sequence diagram with") {
		t.Errorf("expected sequence diagram summary, got:\n%s", got)
	}
	if strings.Contains(got, "Alice->>John") {
		t.Error("mermaid content should be removed")
	}
}

func TestReplaceDiagrams_FlowDiagramWithArrows(t *testing.T) {
	input := "The data flow:\n\n" +
		"Request --> Router --> Handler\n" +
		"Handler --> Service --> DB\n" +
		"DB --> Service --> Handler\n" +
		"Handler --> Response\n\n" +
		"That's the flow."

	got := replaceUnfencedDiagrams(input)

	if strings.Contains(got, "-->") {
		t.Errorf("arrows should be removed, got:\n%s", got)
	}
	if !strings.Contains(got, "Flow diagram with") || !strings.Contains(got, "lines.") {
		t.Errorf("expected flow diagram summary, got:\n%s", got)
	}
}

func TestReplaceDiagrams_TwoLineRunIgnored(t *testing.T) {
	// Only 2 lines — below the minDiagramRunLength threshold.
	input := "┌──────┐\n└──────┘"
	got := replaceUnfencedDiagrams(input)
	if got != input {
		t.Errorf("2-line diagram should be left alone, got:\n%s", got)
	}
}

func TestReplaceDiagrams_NoFalsePositiveOnProse(t *testing.T) {
	input := "The user said \"use A -> B notation\" in the discussion.\n" +
		"We agreed that this was the right approach.\n" +
		"The team will implement it next sprint."

	got := replaceUnfencedDiagrams(input)
	if got != input {
		t.Errorf("prose with isolated arrow should not be modified, got:\n%s", got)
	}
}

func TestReplaceDiagrams_NoFalsePositiveOnMarkdownTable(t *testing.T) {
	// In the real pipeline, tables are replaced before diagram detection runs.
	// But replaceUnfencedDiagrams itself may see pipe-bordered lines.
	// Test through the full NormalizeTextForSpeech pipeline instead.
	input := "| Name | Age | City |\n| --- | --- | --- |\n| Alice | 30 | NYC |\n| Bob | 25 | LA |"

	got := NormalizeTextForSpeech(input)
	// Should be handled as table, not diagram.
	if strings.Contains(got, "Diagram") || strings.Contains(got, "Wireframe") {
		t.Errorf("markdown table should not be treated as diagram, got:\n%s", got)
	}
	if !strings.Contains(got, "Table with") {
		t.Errorf("should be handled as table, got:\n%s", got)
	}
}

func TestReplaceDiagrams_MultipleDiagramRegions(t *testing.T) {
	input := "First:\n\n" +
		"┌───┐\n│ A │\n└───┘\n\n" +
		"Middle prose.\n\n" +
		"├── file1.go\n│   └── nested.go\n└── file2.go\n\n" +
		"Done."

	got := replaceUnfencedDiagrams(input)

	if !strings.Contains(got, "Wireframe diagram.") {
		t.Errorf("expected wireframe summary, got:\n%s", got)
	}
	if !strings.Contains(got, "File tree with 3 entries.") {
		t.Errorf("expected file tree summary, got:\n%s", got)
	}
	if !strings.Contains(got, "Middle prose.") || !strings.Contains(got, "Done.") {
		t.Error("prose between diagrams should be preserved")
	}
}

func TestReplaceDiagrams_BridgeLineAllowed(t *testing.T) {
	// A single non-diagram line between diagram lines should be "bridged".
	input := "┌───────┐\n" +
		"│ Header│\n" +
		"A label here\n" + // bridge line (prose, but between diagram lines)
		"│ Body  │\n" +
		"└───────┘"

	got := replaceUnfencedDiagrams(input)
	if !strings.Contains(got, "Wireframe diagram.") {
		t.Errorf("diagram with 1-line bridge should still be detected, got:\n%s", got)
	}
}

func TestReplaceDiagrams_PlainTextUnchanged(t *testing.T) {
	input := "Just some regular text.\nNothing to see here.\nMove along."
	got := replaceUnfencedDiagrams(input)
	if got != input {
		t.Errorf("plain text should be unchanged, got:\n%s", got)
	}
}

func TestReplaceDiagrams_EmptyInput(t *testing.T) {
	got := replaceUnfencedDiagrams("")
	if got != "" {
		t.Errorf("empty input should return empty, got: %q", got)
	}
}

// --- classifyDiagramRegion unit tests ---

func TestClassifyDiagramRegion_Mermaid(t *testing.T) {
	tests := []struct {
		name   string
		lines  []string
		expect string
	}{
		{
			"graph TD",
			[]string{"graph TD", "A --> B", "B --> C"},
			"Diagram: graph with 3 lines.",
		},
		{
			"flowchart LR",
			[]string{"flowchart LR", "Start --> End", "End --> Done"},
			"Diagram: flowchart with 3 lines.",
		},
		{
			"sequenceDiagram",
			[]string{"sequenceDiagram", "A->>B: Hello", "B-->>A: Reply", "A->>B: Thanks"},
			"Diagram: sequence diagram with 4 lines.",
		},
		{
			"gantt",
			[]string{"gantt", "title Schedule", "section Phase1", "Task1: done, 2024-01-01, 30d"},
			"Diagram: gantt chart with 4 lines.",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyDiagramRegion(tc.lines)
			if got != tc.expect {
				t.Errorf("classifyDiagramRegion() = %q, want %q", got, tc.expect)
			}
		})
	}
}

func TestClassifyDiagramRegion_FileTree(t *testing.T) {
	lines := []string{
		"├── src/",
		"│   ├── main.go",
		"│   └── util.go",
		"└── README.md",
	}
	got := classifyDiagramRegion(lines)
	if got != "File tree with 4 entries." {
		t.Errorf("got %q, want %q", got, "File tree with 4 entries.")
	}
}

func TestClassifyDiagramRegion_Wireframe(t *testing.T) {
	lines := []string{
		"┌──────────┐",
		"│  Header  │",
		"├──────────┤",
		"│  Content │",
		"└──────────┘",
	}
	got := classifyDiagramRegion(lines)
	if got != "Wireframe diagram." {
		t.Errorf("got %q, want %q", got, "Wireframe diagram.")
	}
}

func TestClassifyDiagramRegion_ASCIIWireframe(t *testing.T) {
	lines := []string{
		"+----------+",
		"|  Header  |",
		"+----------+",
		"|  Content |",
		"+----------+",
	}
	got := classifyDiagramRegion(lines)
	if got != "Wireframe diagram." {
		t.Errorf("got %q, want %q", got, "Wireframe diagram.")
	}
}

func TestClassifyDiagramRegion_FlowDiagram(t *testing.T) {
	lines := []string{
		"Input --> Process --> Output",
		"Output --> Cache --> Client",
		"Client --> Response --> Done",
	}
	got := classifyDiagramRegion(lines)
	if !strings.Contains(got, "Flow diagram with") {
		t.Errorf("expected flow diagram classification, got: %q", got)
	}
}

func TestClassifyDiagramRegion_GenericDiagram(t *testing.T) {
	// High structural content but no dominant signal.
	lines := []string{
		"==========",
		"||      ||",
		"==========",
	}
	got := classifyDiagramRegion(lines)
	// Could be wireframe or generic — either is acceptable.
	if !strings.Contains(got, "diagram") && !strings.Contains(got, "Diagram") && !strings.Contains(got, "Wireframe") {
		t.Errorf("expected some diagram classification, got: %q", got)
	}
}

// --- normalizeMermaidType unit tests ---

func TestNormalizeMermaidType(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"graph TD", "graph"},
		{"graph LR", "graph"},
		{"flowchart LR", "flowchart"},
		{"sequenceDiagram", "sequence diagram"},
		{"classDiagram", "class diagram"},
		{"stateDiagram", "state diagram"},
		{"erDiagram", "entity relationship diagram"},
		{"gantt", "gantt chart"},
		{"pie", "pie chart"},
		{"gitgraph", "git graph"},
		{"journey", "user journey"},
		{"unknownType", "unknowntype diagram"},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := normalizeMermaidType(tc.input)
			if got != tc.want {
				t.Errorf("normalizeMermaidType(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// --- NormalizeTextForSpeech integration with diagram detection ---

func TestNormalize_UnicodeWireframeInFullMessage(t *testing.T) {
	input := "## Layout\n\n" +
		"Here's the proposed UI:\n\n" +
		"┌─────────────────┐\n" +
		"│     Header      │\n" +
		"├────────┬────────┤\n" +
		"│ Sidebar│Content │\n" +
		"└────────┴────────┘\n\n" +
		"What do you think?"

	got := NormalizeTextForSpeech(input)

	// Heading markers removed.
	if strings.Contains(got, "##") {
		t.Error("heading markers should be removed")
	}
	if !strings.Contains(got, "Layout") {
		t.Error("heading text should be preserved")
	}
	// Wireframe replaced.
	if strings.Contains(got, "┌") {
		t.Error("box-drawing characters should be replaced")
	}
	if !strings.Contains(got, "Wireframe diagram.") {
		t.Errorf("expected wireframe summary, got:\n%s", got)
	}
	// Surrounding prose preserved.
	if !strings.Contains(got, "proposed UI") || !strings.Contains(got, "What do you think?") {
		t.Error("surrounding prose should be preserved")
	}
}

func TestNormalize_FileTreeInFullMessage(t *testing.T) {
	input := "The project looks like this:\n\n" +
		"├── api/\n" +
		"│   ├── main.go\n" +
		"│   └── handlers/\n" +
		"├── ui/\n" +
		"│   └── src/\n" +
		"└── README.md\n\n" +
		"Let me know if you want changes."

	got := NormalizeTextForSpeech(input)

	if strings.Contains(got, "├") || strings.Contains(got, "└") {
		t.Error("tree characters should be replaced")
	}
	if !strings.Contains(got, "File tree with") {
		t.Errorf("expected file tree summary, got:\n%s", got)
	}
	if !strings.Contains(got, "The project looks like this:") {
		t.Error("preceding prose should be preserved")
	}
}

func TestNormalize_MermaidInFullMessage(t *testing.T) {
	input := "Here's the **architecture**:\n\n" +
		"graph TD\n" +
		"    A[Frontend] --> B[API]\n" +
		"    B --> C[Database]\n" +
		"    B --> D[Cache]\n\n" +
		"See [the docs](https://example.com) for more."

	got := NormalizeTextForSpeech(input)

	// Bold removed.
	if strings.Contains(got, "**") {
		t.Error("bold markers should be removed")
	}
	// Mermaid replaced.
	if strings.Contains(got, "graph TD") {
		t.Error("mermaid content should be replaced")
	}
	if !strings.Contains(got, "Diagram: graph with") {
		t.Errorf("expected mermaid graph summary, got:\n%s", got)
	}
	// Link normalized.
	if strings.Contains(got, "https://") {
		t.Error("URL should be removed")
	}
	if !strings.Contains(got, "the docs") {
		t.Error("link text should be preserved")
	}
}

func TestNormalize_DiagramDoesNotAffectCodeBlocks(t *testing.T) {
	// Fenced code blocks with diagram-like content should be handled by the
	// code block normalizer, not the diagram detector.
	input := "Example:\n```\n" +
		"┌───┐\n│ A │\n└───┘\n" +
		"```\nDone."

	got := NormalizeTextForSpeech(input)

	// Should be handled as code block, not wireframe.
	if strings.Contains(got, "Wireframe") {
		t.Error("fenced diagram content should be handled as code block, not wireframe")
	}
	if !strings.Contains(got, "Code block:") {
		t.Errorf("should contain code block summary, got:\n%s", got)
	}
}

func TestNormalize_DiagramDoesNotAffectTables(t *testing.T) {
	input := "Results:\n\n" +
		"| Col A | Col B |\n" +
		"| --- | --- |\n" +
		"| 1 | 2 |\n" +
		"| 3 | 4 |\n\n" +
		"End."

	got := NormalizeTextForSpeech(input)

	if strings.Contains(got, "Diagram") || strings.Contains(got, "Wireframe") {
		t.Errorf("markdown table should not be classified as diagram, got:\n%s", got)
	}
	if !strings.Contains(got, "Table with") {
		t.Errorf("table should be handled by table normalizer, got:\n%s", got)
	}
}

// --- Edge cases ---

func TestReplaceDiagrams_OnlyDiagram(t *testing.T) {
	// Input is nothing but a diagram.
	input := "┌───┐\n│ X │\n├───┤\n│ Y │\n└───┘"
	got := replaceUnfencedDiagrams(input)
	if !strings.Contains(got, "Wireframe diagram.") {
		t.Errorf("diagram-only input should produce wireframe summary, got:\n%s", got)
	}
}

func TestReplaceDiagrams_MixedUnicodeAndASCII(t *testing.T) {
	// Sometimes people mix Unicode and ASCII in diagrams.
	input := "+--------+\n" +
		"│ Header │\n" +
		"+--------+\n" +
		"│ Body   │\n" +
		"+--------+"

	got := replaceUnfencedDiagrams(input)
	if !strings.Contains(got, "Wireframe diagram.") {
		t.Errorf("mixed Unicode/ASCII diagram should be detected, got:\n%s", got)
	}
}

func TestReplaceDiagrams_UnicodeArrows(t *testing.T) {
	input := "Request → Router → Handler\n" +
		"Handler → Service → DB\n" +
		"DB → Response → Client"

	got := replaceUnfencedDiagrams(input)
	if strings.Contains(got, "→") {
		t.Errorf("Unicode arrows should be replaced, got:\n%s", got)
	}
	if !strings.Contains(got, "diagram") && !strings.Contains(got, "Diagram") {
		t.Errorf("expected diagram summary, got:\n%s", got)
	}
}

func TestReplaceDiagrams_SequenceLikeDiagram(t *testing.T) {
	input := "Client          Server          DB\n" +
		"  |--- request --->|              |\n" +
		"  |                |--- query --->|\n" +
		"  |                |<-- result ---|\n" +
		"  |<-- response ---|              |"

	got := replaceUnfencedDiagrams(input)
	// The first line is prose-like, but the arrow lines should form a run.
	if strings.Contains(got, "--->") || strings.Contains(got, "<--") {
		t.Errorf("sequence arrows should be removed, got:\n%s", got)
	}
}

// ---------------------------------------------------------------------------
// Helper unit tests
// ---------------------------------------------------------------------------

func TestExtractBasename(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/home/user/file.go", "file.go"},
		{"src/lib/utils.ts", "utils.ts"},
		{"./relative/path/index.js", "index.js"},
		{"~/documents/notes.md", "notes.md"},
		{"nopath", "nopath"},
		{"trailing/", "trailing/"},
		{"", ""},
	}
	for _, tc := range tests {
		got := extractBasename(tc.input)
		if got != tc.want {
			t.Errorf("extractBasename(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestParseTableRow(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"| foo | bar | baz |", []string{"foo", "bar", "baz"}},
		{"| --- | --- | --- |", nil}, // separator row
		{"| a |", []string{"a"}},
	}
	for _, tc := range tests {
		got := parseTableRow(tc.input)
		if len(got) != len(tc.want) {
			t.Errorf("parseTableRow(%q): got %d cells, want %d", tc.input, len(got), len(tc.want))
			continue
		}
		for i, cell := range got {
			if cell != tc.want[i] {
				t.Errorf("parseTableRow(%q)[%d] = %q, want %q", tc.input, i, cell, tc.want[i])
			}
		}
	}
}
