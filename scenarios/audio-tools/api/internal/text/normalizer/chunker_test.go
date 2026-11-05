package normalizer

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// SplitIntoSpeechParagraphs (integration: normalize + chunk)
// ---------------------------------------------------------------------------

func TestSplitSpeech_ShortText(t *testing.T) {
	got := SplitIntoSpeechParagraphs("Hello world")
	if len(got) != 1 || got[0] != "Hello world" {
		t.Errorf("short text should return single chunk, got: %v", got)
	}
}

func TestSplitSpeech_SplitsOnDoubleNewlines(t *testing.T) {
	text := "Paragraph one.\n\nParagraph two.\n\nParagraph three."
	got := SplitIntoSpeechParagraphs(text)
	if len(got) != 3 {
		t.Errorf("expected 3 paragraphs, got %d: %v", len(got), got)
	}
}

func TestSplitSpeech_FiltersEmptyParagraphs(t *testing.T) {
	got := SplitIntoSpeechParagraphs("a\n\n\n\nb")
	if len(got) != 2 {
		t.Errorf("expected 2 paragraphs, got %d: %v", len(got), got)
	}
}

func TestSplitSpeech_LongLineChunked(t *testing.T) {
	longText := strings.Repeat("word ", 1200)
	got := SplitIntoSpeechParagraphs(longText)
	for _, chunk := range got {
		if len(chunk) > TTSMaxChunkLength {
			t.Errorf("chunk exceeds limit: %d chars", len(chunk))
		}
	}
	if len(got) <= 1 {
		t.Error("expected multiple chunks for long text")
	}
}

func TestSplitSpeech_SentenceBoundarySplit(t *testing.T) {
	sentence := "This is a moderately long sentence that helps us test the chunker. "
	longText := strings.TrimSpace(strings.Repeat(sentence, 100))
	got := SplitIntoSpeechParagraphs(longText)
	for _, chunk := range got {
		if len(chunk) > TTSMaxChunkLength {
			t.Errorf("chunk exceeds limit: %d chars", len(chunk))
		}
	}
	if len(got) <= 1 {
		t.Error("expected multiple chunks")
	}
}

func TestSplitSpeech_HardSplitNoSpaces(t *testing.T) {
	longText := strings.Repeat("x", 10000)
	got := SplitIntoSpeechParagraphs(longText)
	for _, chunk := range got {
		if len(chunk) > TTSMaxChunkLength {
			t.Errorf("chunk exceeds limit: %d chars", len(chunk))
		}
	}
	if strings.Join(got, "") != longText {
		t.Error("hard split should preserve all content")
	}
}

func TestSplitSpeech_ExactlyAtLimit(t *testing.T) {
	text := strings.Repeat("a", TTSMaxChunkLength)
	got := SplitIntoSpeechParagraphs(text)
	if len(got) != 1 || got[0] != text {
		t.Errorf("text at exact limit should be single chunk, got %d chunks", len(got))
	}
}

func TestSplitSpeech_OneOverLimit(t *testing.T) {
	text := strings.Repeat("a", TTSMaxChunkLength+1)
	got := SplitIntoSpeechParagraphs(text)
	for _, chunk := range got {
		if len(chunk) > TTSMaxChunkLength {
			t.Errorf("chunk exceeds limit: %d chars", len(chunk))
		}
	}
	if strings.Join(got, "") != text {
		t.Error("content should be preserved")
	}
}

func TestSplitSpeech_MixedShortAndLong(t *testing.T) {
	shortP := "Short paragraph."
	longP := strings.TrimSpace(strings.Repeat("Long sentence here. ", 300))
	text := shortP + "\n\n" + longP + "\n\n" + shortP
	got := SplitIntoSpeechParagraphs(text)
	if got[0] != shortP {
		t.Errorf("first chunk should be short paragraph, got: %q", got[0])
	}
	if got[len(got)-1] != shortP {
		t.Errorf("last chunk should be short paragraph, got: %q", got[len(got)-1])
	}
	for _, chunk := range got {
		if len(chunk) > TTSMaxChunkLength {
			t.Errorf("chunk exceeds limit: %d chars", len(chunk))
		}
	}
}

// ---------------------------------------------------------------------------
// SplitIntoSpeechParagraphs: normalization integration
// ---------------------------------------------------------------------------

func TestSplitSpeech_CodeBlocksNormalized(t *testing.T) {
	input := "Before.\n\n```json\n{\"key\": \"value\"}\n```\n\nAfter."
	got := SplitIntoSpeechParagraphs(input)
	for _, chunk := range got {
		if strings.Contains(chunk, `"key"`) {
			t.Error("code block content should be replaced by summary")
		}
	}
	// Should contain the summary and the surrounding prose.
	joined := strings.Join(got, " ")
	if !strings.Contains(joined, "Before.") {
		t.Error("prose before code block should be preserved")
	}
	if !strings.Contains(joined, "After.") {
		t.Error("prose after code block should be preserved")
	}
	if !strings.Contains(joined, "Code block:") {
		t.Error("code block summary should be present")
	}
}

func TestSplitSpeech_FilePathsNormalized(t *testing.T) {
	input := "Edit `/home/user/project/src/main.go` to fix it."
	got := SplitIntoSpeechParagraphs(input)
	joined := strings.Join(got, " ")
	if strings.Contains(joined, "/home/user") {
		t.Error("full path should be normalized to basename")
	}
	if !strings.Contains(joined, "main.go") {
		t.Error("basename should be present")
	}
}

func TestSplitSpeech_HeadingsNormalized(t *testing.T) {
	input := "## Section Title\n\nSome content here."
	got := SplitIntoSpeechParagraphs(input)
	joined := strings.Join(got, " ")
	if strings.Contains(joined, "##") {
		t.Error("heading markers should be stripped")
	}
	if !strings.Contains(joined, "Section Title") {
		t.Error("heading text should be preserved")
	}
}

func TestSplitSpeech_TablesNormalized(t *testing.T) {
	input := "Results:\n\n| A | B |\n| --- | --- |\n| 1 | 2 |\n\nDone."
	got := SplitIntoSpeechParagraphs(input)
	joined := strings.Join(got, " ")
	if strings.Contains(joined, "|") {
		t.Error("table pipe characters should be removed")
	}
	if !strings.Contains(joined, "Table with") {
		t.Error("table summary should be present")
	}
}

// ---------------------------------------------------------------------------
// IsSpeakable
// ---------------------------------------------------------------------------

func TestIsSpeakable(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"Hello world", true},
		{"# Heading", true},
		{"- Item text", true},
		{"Some code: x = 1", true},
		{"---", false},
		{"***", false},
		{"___", false},
		{"```", false},
		{"```typescript", false},
		{"* ", false},
		{"-", false},
		{"+", false},
		{"> ", false},
		{".", false},
		{"...", false},
	}
	for _, tc := range tests {
		got := IsSpeakable(tc.input)
		if got != tc.want {
			t.Errorf("IsSpeakable(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Filter non-speakable chunks
// ---------------------------------------------------------------------------

func TestSplitSpeech_FiltersHorizontalRules(t *testing.T) {
	got := SplitIntoSpeechParagraphs("Para one.\n\n---\n\nPara two.")
	for _, chunk := range got {
		if strings.TrimSpace(chunk) == "---" {
			t.Error("horizontal rules should be filtered")
		}
	}
}

func TestSplitSpeech_FiltersCodeFences(t *testing.T) {
	got := SplitIntoSpeechParagraphs("Before.\n\n```typescript\n\nAfter.")
	for _, chunk := range got {
		if strings.TrimSpace(chunk) == "```typescript" {
			t.Error("lone code fence lines should be filtered")
		}
	}
}

func TestSplitSpeech_KeepsHeadingsWithText(t *testing.T) {
	// After normalization, "# My Heading" becomes "My Heading".
	got := SplitIntoSpeechParagraphs("# My Heading\n\nParagraph.")
	joined := strings.Join(got, " ")
	if !strings.Contains(joined, "My Heading") {
		t.Error("heading text should be preserved after normalization")
	}
}

func TestSplitSpeech_FallbackOnAllFiltered(t *testing.T) {
	got := SplitIntoSpeechParagraphs("---")
	if len(got) != 1 || got[0] != "---" {
		t.Errorf("when everything is filtered, should fall back to original text, got: %v", got)
	}
}

// ---------------------------------------------------------------------------
// SplitIntoSpeechParagraphs: diagram normalization integration
// ---------------------------------------------------------------------------

func TestSplitSpeech_WireframeNormalized(t *testing.T) {
	input := "Here's the layout:\n\n" +
		"┌─────────────────┐\n" +
		"│     Header      │\n" +
		"├────────┬────────┤\n" +
		"│ Sidebar│Content │\n" +
		"└────────┴────────┘\n\n" +
		"Done."

	got := SplitIntoSpeechParagraphs(input)
	joined := strings.Join(got, " ")

	if strings.Contains(joined, "┌") || strings.Contains(joined, "│") {
		t.Error("wireframe characters should be replaced")
	}
	if !strings.Contains(joined, "Wireframe diagram.") {
		t.Errorf("expected wireframe summary, got:\n%s", joined)
	}
	if !strings.Contains(joined, "Here's the layout:") || !strings.Contains(joined, "Done.") {
		t.Error("surrounding prose should be preserved")
	}
}

func TestSplitSpeech_FileTreeNormalized(t *testing.T) {
	input := "Structure:\n\n" +
		"├── api/\n" +
		"│   ├── main.go\n" +
		"│   └── handlers/\n" +
		"└── README.md\n\n" +
		"End."

	got := SplitIntoSpeechParagraphs(input)
	joined := strings.Join(got, " ")

	if strings.Contains(joined, "├") || strings.Contains(joined, "└") {
		t.Error("tree connectors should be replaced")
	}
	if !strings.Contains(joined, "File tree with") {
		t.Errorf("expected file tree summary, got:\n%s", joined)
	}
}

func TestSplitSpeech_MermaidNormalized(t *testing.T) {
	input := "Architecture:\n\n" +
		"graph TD\n" +
		"    A[Frontend] --> B[API]\n" +
		"    B --> C[Database]\n" +
		"    B --> D[Cache]\n\n" +
		"End."

	got := SplitIntoSpeechParagraphs(input)
	joined := strings.Join(got, " ")

	if strings.Contains(joined, "graph TD") {
		t.Error("mermaid content should be replaced")
	}
	if !strings.Contains(joined, "Diagram: graph with") {
		t.Errorf("expected mermaid summary, got:\n%s", joined)
	}
}

func TestSplitSpeech_DiagramDoesNotAffectTables(t *testing.T) {
	input := "Data:\n\n| A | B |\n| --- | --- |\n| 1 | 2 |\n\nEnd."
	got := SplitIntoSpeechParagraphs(input)
	joined := strings.Join(got, " ")

	if strings.Contains(joined, "Wireframe") || strings.Contains(joined, "Diagram:") {
		t.Error("tables should not be classified as diagrams")
	}
	if !strings.Contains(joined, "Table with") {
		t.Errorf("table should use table summary, got:\n%s", joined)
	}
}

func TestSplitSpeech_DiagramChunksRespectLimit(t *testing.T) {
	// Even after diagram normalization, chunks must stay under the limit.
	var lines []string
	for i := 0; i < 100; i++ {
		lines = append(lines, "├── "+strings.Repeat("filename", 20)+".go")
	}
	input := strings.Join(lines, "\n")

	got := SplitIntoSpeechParagraphs(input)
	for _, chunk := range got {
		if len(chunk) > TTSMaxChunkLength {
			t.Errorf("chunk exceeds limit: %d chars", len(chunk))
		}
	}
}

// ---------------------------------------------------------------------------
// splitNonEmpty
// ---------------------------------------------------------------------------

func TestSplitNonEmpty(t *testing.T) {
	got := splitNonEmpty("a\n\nb\n\n\n\nc", "\n\n")
	if len(got) != 3 {
		t.Errorf("expected 3 parts, got %d: %v", len(got), got)
	}
}
