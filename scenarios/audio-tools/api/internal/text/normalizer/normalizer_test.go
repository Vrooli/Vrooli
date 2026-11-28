package normalizer

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Fenced code blocks
// ---------------------------------------------------------------------------

func TestNormalize_FencedCodeBlocks(t *testing.T) {
	tests := []struct {
		name, input, want string
	}{
		{"language", "Here is some code:\n```go\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n```\nDone.", "Here is some code:\nCode block: 3 lines of go.\nDone."},
		{"no language", "```\nline1\nline2\n```", "Code block: 2 lines."},
		{"json", "Response:\n```json\n{\n  \"key\": \"value\",\n  \"count\": 42\n}\n```", "Response:\nCode block: 4 lines of json."},
		{"multiple", "First:\n```py\nprint(1)\n```\nSecond:\n```js\nconsole.log(2)\n```", "First:\nCode block: 1 lines of py.\nSecond:\nCode block: 1 lines of js."},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := NormalizeTextForSpeech(tc.input); got != tc.want {
				t.Errorf("NormalizeTextForSpeech() = %q, want %q", got, tc.want)
			}
		})
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

func TestNormalize_MarkdownEmphasis(t *testing.T) {
	tests := []struct {
		name, input, want string
	}{
		{"bold asterisks", "This is **important** text.", "This is important text."},
		{"bold underscores", "This is __important__ text.", "This is important text."},
		{"italic", "This is *emphasized* text.", "This is emphasized text."},
		{"strikethrough", "This is ~~deleted~~ text.", "This is deleted text."},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := NormalizeTextForSpeech(tc.input); got != tc.want {
				t.Errorf("NormalizeTextForSpeech() = %q, want %q", got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Links and images
// ---------------------------------------------------------------------------

func TestNormalize_ImageDescriptions(t *testing.T) {
	tests := []struct {
		name, input, want string
	}{
		{"with alt text", "Here: ![screenshot of dashboard](http://img.png)", "Here: Image: screenshot of dashboard."},
		{"without alt text", "![](http://img.png)", "Image."},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := NormalizeTextForSpeech(tc.input); got != tc.want {
				t.Errorf("NormalizeTextForSpeech() = %q, want %q", got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Horizontal rules
// ---------------------------------------------------------------------------

func TestNormalize_MarkdownContainers(t *testing.T) {
	tests := []struct {
		name, input string
		want, avoid []string
	}{
		{"link text", "See [the docs](https://example.com/docs) for more.", []string{"the docs"}, []string{"https://", "[", "]"}},
		{"horizontal rule", "Above\n\n---\n\nBelow", []string{"Above", "Below"}, []string{"---"}},
		{"HTML tags", "Line one.<br>Line two.<hr/>Done.", []string{"Line one.", "Done."}, []string{"<br>", "<hr/>"}},
		{"blockquote", "> This is a quote.\n> Second line.", []string{"This is a quote."}, []string{">"}},
		{"unordered list markers", "- First item\n- Second item\n* Third item", []string{"First item.", "Third item."}, []string{"- First", "* Third"}},
		{"unordered punctuation", "- Already has a period.\n- Has exclamation!\n- Has question?\n- Has colon:\n- No punctuation", []string{"No punctuation."}, []string{"period..", "exclamation!.", "question?.", "colon:."}},
		{"indented unordered items", "- Top level\n  - Indented item\n    - Deep item", []string{"Top level.", "Indented item.", "Deep item."}, nil},
		{"plus list markers", "+ Item one\n+ Item two", []string{"Item one.", "Item two."}, []string{"+"}},
		{"ordered items", "1. Alpha\n2. Beta\n3. Gamma", []string{"1. Alpha.", "2. Beta.", "3. Gamma."}, nil},
		{"ordered punctuation", "1. Already done.\n2. Also done!\n3. Really?\n4. No punctuation", []string{"4. No punctuation."}, []string{"done..", "done!."}},
		{"ordered numbering", "1. First step\n2. Second step\n3. Third step", []string{"1.", "2.", "3."}, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := NormalizeTextForSpeech(tc.input)
			for _, text := range tc.want {
				if !strings.Contains(got, text) {
					t.Errorf("missing %q in:\n%s", text, got)
				}
			}
			for _, text := range tc.avoid {
				if strings.Contains(got, text) {
					t.Errorf("unexpected %q in:\n%s", text, got)
				}
			}
		})
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

func TestDiagramLineScore(t *testing.T) {
	tests := []struct {
		name string
		line string
		min  float64
		max  float64
	}{
		{"empty", "", 0, 0}, {"whitespace", "   ", 0, 0},
		{"plain prose", "This is just a regular sentence.", 0, diagramScoreThreshold},
		{"plain sentence", "The quick brown fox jumps over the lazy dog.", 0, diagramScoreThreshold},
		{"numbers in prose", "Here are some numbers: 42, 100, 256.", 0, diagramScoreThreshold},
		{"unicode top edge", "┌─────────────────┐", diagramScoreThreshold, 2},
		{"unicode bottom edge", "└────────┴────────┘", diagramScoreThreshold, 2},
		{"unicode divider", "├────────┬────────┤", diagramScoreThreshold, 2},
		{"unicode content row", "│ Sidebar│Content │", diagramScoreThreshold, 2},
		{"ASCII box", "+-------------------+", .9, 2}, {"ASCII equals", "+===+===+===+", .9, 2}, {"ASCII short", "+--+--+", .9, 2},
		{"tree source", "├── src/", diagramScoreThreshold, 2}, {"tree components", "│   ├── components/", diagramScoreThreshold, 2},
		{"tree leaf", "│   │   └── Button.tsx", diagramScoreThreshold, 2}, {"tree root", "└── package.json", diagramScoreThreshold, 2},
		{"graph TD", "graph TD", .9, 2}, {"graph LR", "graph LR", .9, 2}, {"flowchart", "flowchart LR", .9, 2},
		{"sequence", "sequenceDiagram", .9, 2}, {"class", "classDiagram", .9, 2}, {"state", "stateDiagram", .9, 2},
		{"ER", "erDiagram", .9, 2}, {"gantt", "gantt", .9, 2}, {"pie", "pie", .9, 2}, {"gitgraph", "gitgraph", .9, 2}, {"journey", "journey", .9, 2},
		{"arrow chain", "A --> B --> C", diagramScoreThreshold, 2}, {"double arrow", "Start ==> Middle ==> End", diagramScoreThreshold, 2}, {"plain arrow", "Input -> Process -> Output", diagramScoreThreshold, 2},
		{"pipe sidebar", "|  Sidebar | Content|", diagramScoreThreshold, 2}, {"pipe blank", "|          |        |", diagramScoreThreshold, 2}, {"pipe header", "| Header              |", diagramScoreThreshold, 2},
		{"mermaid node", "    A[Start] --> B{Decision}", diagramScoreThreshold, 2}, {"mermaid label", "    B -->|Yes| C[Action]", diagramScoreThreshold, 2}, {"mermaid client", "    Client[Browser] -> Server[API]", diagramScoreThreshold, 2},
		{"structural box", "+----------+--------+", diagramScoreThreshold, 2}, {"structural rule", "==========", diagramScoreThreshold, 2}, {"structural arrow", "<-------->", diagramScoreThreshold, 2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := diagramLineScore(tc.line)
			if (tc.max == 0 && got != 0) || (tc.max > 0 && (got < tc.min || got >= tc.max)) {
				t.Errorf("diagramLineScore(%q) = %.2f, want in [%.2f, %.2f)", tc.line, got, tc.min, tc.max)
			}
		})
	}
}

// --- replaceUnfencedDiagrams unit tests ---

func TestReplaceDiagrams(t *testing.T) {
	tests := []struct {
		name, input string
		normalize   bool
		want        []string
		avoid       []string
		unchanged   bool
	}{
		{"unicode wireframe", "Here's the layout:\n\n┌─────────────────┐\n│     Header      │\n├────────┬────────┤\n│ Sidebar│Content │\n└────────┴────────┘\n\nThat's the wireframe.", false, []string{"Wireframe diagram.", "Here's the layout:", "That's the wireframe."}, []string{"┌", "│", "└"}, false},
		{"ASCII wireframe", "The page layout:\n\n+-------------------+\n|     Header        |\n+-------------------+\n|  Sidebar | Content|\n|          |        |\n+-------------------+\n\nEnd of wireframe.", false, []string{"Wireframe diagram.", "The page layout:", "End of wireframe."}, []string{"+---"}, false},
		{"file tree", "Project structure:\n\n├── src/\n│   ├── components/\n│   │   ├── Button.tsx\n│   │   └── Header.tsx\n│   └── utils/\n└── package.json\n\nThat's the tree.", false, []string{"File tree with 6 entries.", "Project structure:", "That's the tree."}, []string{"├", "└"}, false},
		{"mermaid flowchart", "Here is the flow:\n\ngraph TD\n    A[Start] --> B{Decision}\n    B -->|Yes| C[Action]\n    B -->|No| D[End]\n\nThat's the diagram.", false, []string{"Diagram: graph with", "Here is the flow:", "That's the diagram."}, []string{"graph TD", "-->"}, false},
		{"mermaid sequence", "sequenceDiagram\n    Alice->>John: Hello John\n    John-->>Alice: Hi Alice\n    Alice->>John: How are you?\n", false, []string{"Diagram: sequence diagram with"}, []string{"Alice->>John"}, false},
		{"arrow flow", "The data flow:\n\nRequest --> Router --> Handler\nHandler --> Service --> DB\nDB --> Service --> Handler\nHandler --> Response\n\nThat's the flow.", false, []string{"Flow diagram with", "lines."}, []string{"-->"}, false},
		{"two-line run", "┌──────┐\n└──────┘", false, nil, nil, true},
		{"isolated arrow prose", "The user said \"use A -> B notation\" in the discussion.\nWe agreed that this was the right approach.\nThe team will implement it next sprint.", false, nil, nil, true},
		{"markdown table uses earlier pipeline pass", "| Name | Age | City |\n| --- | --- | --- |\n| Alice | 30 | NYC |\n| Bob | 25 | LA |", true, []string{"Table with"}, []string{"Diagram", "Wireframe"}, false},
		{"multiple regions preserve middle prose", "First:\n\n┌───┐\n│ A │\n└───┘\n\nMiddle prose.\n\n├── file1.go\n│   └── nested.go\n└── file2.go\n\nDone.", false, []string{"Wireframe diagram.", "File tree with 3 entries.", "Middle prose.", "Done."}, nil, false},
		{"bridge line", "┌───────┐\n│ Header│\nA label here\n│ Body  │\n└───────┘", false, []string{"Wireframe diagram."}, nil, false},
		{"plain text", "Just some regular text.\nNothing to see here.\nMove along.", false, nil, nil, true},
		{"empty", "", false, nil, nil, true},
		{"diagram only", "┌───┐\n│ X │\n├───┤\n│ Y │\n└───┘", false, []string{"Wireframe diagram."}, nil, false},
		{"mixed unicode and ASCII", "+--------+\n│ Header │\n+--------+\n│ Body   │\n+--------+", false, []string{"Wireframe diagram."}, nil, false},
		{"unicode arrows", "Request → Router → Handler\nHandler → Service → DB\nDB → Response → Client", false, []string{"diagram"}, []string{"→"}, false},
		{"sequence-like diagram", "Client          Server          DB\n  |--- request --->|              |\n  |                |--- query --->|\n  |                |<-- result ---|\n  |<-- response ---|              |", false, nil, []string{"--->", "<--"}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			transform := replaceUnfencedDiagrams
			if tc.normalize {
				transform = NormalizeTextForSpeech
			}
			got := transform(tc.input)
			if tc.unchanged && got != tc.input {
				t.Errorf("unexpected change:\n%s", got)
			}
			for _, text := range tc.want {
				if !strings.Contains(got, text) {
					t.Errorf("missing %q in:\n%s", text, got)
				}
			}
			for _, text := range tc.avoid {
				if strings.Contains(got, text) {
					t.Errorf("unexpected %q in:\n%s", text, got)
				}
			}
		})
	}
}

// --- classifyDiagramRegion unit tests ---

func TestClassifyDiagramRegion(t *testing.T) {
	tests := []struct {
		name, want, contains string
		lines                []string
		generic              bool
	}{
		{"graph TD", "Diagram: graph with 3 lines.", "", []string{"graph TD", "A --> B", "B --> C"}, false},
		{"flowchart LR", "Diagram: flowchart with 3 lines.", "", []string{"flowchart LR", "Start --> End", "End --> Done"}, false},
		{"sequenceDiagram", "Diagram: sequence diagram with 4 lines.", "", []string{"sequenceDiagram", "A->>B: Hello", "B-->>A: Reply", "A->>B: Thanks"}, false},
		{"gantt", "Diagram: gantt chart with 4 lines.", "", []string{"gantt", "title Schedule", "section Phase1", "Task1: done, 2024-01-01, 30d"}, false},
		{"file tree", "File tree with 4 entries.", "", []string{"├── src/", "│   ├── main.go", "│   └── util.go", "└── README.md"}, false},
		{"unicode wireframe", "Wireframe diagram.", "", []string{"┌──────────┐", "│  Header  │", "├──────────┤", "│  Content │", "└──────────┘"}, false},
		{"ASCII wireframe", "Wireframe diagram.", "", []string{"+----------+", "|  Header  |", "+----------+", "|  Content |", "+----------+"}, false},
		{"flow diagram", "", "Flow diagram with", []string{"Input --> Process --> Output", "Output --> Cache --> Client", "Client --> Response --> Done"}, false},
		{"generic structural diagram", "", "", []string{"==========", "||      ||", "=========="}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyDiagramRegion(tc.lines)
			switch {
			case tc.generic:
				if !strings.Contains(got, "diagram") && !strings.Contains(got, "Diagram") && !strings.Contains(got, "Wireframe") {
					t.Errorf("expected diagram classification, got %q", got)
				}
			case tc.contains != "":
				if !strings.Contains(got, tc.contains) {
					t.Errorf("classifyDiagramRegion() = %q, want %q", got, tc.contains)
				}
			case got != tc.want:
				t.Errorf("classifyDiagramRegion() = %q, want %q", got, tc.want)
			}
		})
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
