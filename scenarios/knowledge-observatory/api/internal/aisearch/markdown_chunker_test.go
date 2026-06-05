package aisearch

import (
	"strings"
	"testing"

	pkg "github.com/vrooli/aisearch-go"
)

func docWith(body string) pkg.SourceDoc {
	return pkg.SourceDoc{
		ID:   "docs/EXAMPLE.md",
		Kind: DocKind,
		Body: body,
		Meta: map[string]any{
			MetaTitle:       "Example",
			MetaDescription: "An example document",
		},
	}
}

func TestChunkEmptyBodyYieldsNoChunks(t *testing.T) {
	chunks, err := NewMarkdownChunker().Chunk(docWith("   \n\n  \n"))
	if err != nil {
		t.Fatalf("Chunk: %v", err)
	}
	if len(chunks) != 0 {
		t.Fatalf("want 0 chunks for blank body, got %d", len(chunks))
	}
}

func TestChunkShortDocIsSingleChunk(t *testing.T) {
	chunks, err := NewMarkdownChunker().Chunk(docWith("# Title\n\nA short paragraph of prose."))
	if err != nil {
		t.Fatalf("Chunk: %v", err)
	}
	if len(chunks) != 1 {
		t.Fatalf("want 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Index != 0 {
		t.Fatalf("want index 0, got %d", chunks[0].Index)
	}
	if hp, _ := chunks[0].Meta[MetaHeadingPath].(string); hp != "Title" {
		t.Fatalf("heading path = %q, want Title", hp)
	}
	if !strings.Contains(chunks[0].Body, "short paragraph") {
		t.Fatalf("chunk body missing prose: %q", chunks[0].Body)
	}
}

func TestChunkHeadingPathTracksNesting(t *testing.T) {
	body := "# Top\n\nintro\n\n## Middle\n\nmid body\n\n### Deep\n\ndeep body"
	chunks, err := NewMarkdownChunker().Chunk(docWith(body))
	if err != nil {
		t.Fatalf("Chunk: %v", err)
	}
	// The deepest content should carry the full path "Top > Middle > Deep".
	var found bool
	for _, c := range chunks {
		if strings.Contains(c.Body, "deep body") {
			if hp, _ := c.Meta[MetaHeadingPath].(string); hp != "Top > Middle > Deep" {
				t.Fatalf("deep heading path = %q, want Top > Middle > Deep", hp)
			}
			found = true
		}
	}
	if !found {
		t.Fatal("deep body not found in any chunk")
	}
}

func TestChunkKeepsCodeFenceAtomic(t *testing.T) {
	// A fenced block containing blank lines and heading-looking lines must stay
	// in one chunk and never be split.
	code := "```go\nfunc main() {\n\n\t// # not a heading\n\tprintln(\"hi\")\n}\n```"
	body := "# API\n\nlead-in\n\n" + code + "\n\ntrailing"
	chunks, err := NewMarkdownChunker().Chunk(docWith(body))
	if err != nil {
		t.Fatalf("Chunk: %v", err)
	}
	var whole int
	for _, c := range chunks {
		if strings.Contains(c.Body, "func main()") {
			whole++
			if !strings.Contains(c.Body, "println(\"hi\")") || !strings.Contains(c.Body, "```go") {
				t.Fatalf("code fence was split across chunks: %q", c.Body)
			}
		}
	}
	if whole != 1 {
		t.Fatalf("code fence should appear whole in exactly one chunk, got %d", whole)
	}
}

func TestChunkKeepsTableAtomic(t *testing.T) {
	table := "| a | b |\n| - | - |\n| 1 | 2 |\n| 3 | 4 |"
	chunks, err := NewMarkdownChunker().Chunk(docWith("# T\n\n" + table))
	if err != nil {
		t.Fatalf("Chunk: %v", err)
	}
	var rows int
	for _, c := range chunks {
		if strings.Contains(c.Body, "| a | b |") {
			rows = strings.Count(c.Body, "|\n|") // crude: rows stay together
			if !strings.Contains(c.Body, "| 3 | 4 |") {
				t.Fatalf("table split across chunks: %q", c.Body)
			}
		}
	}
	if rows == 0 {
		t.Fatal("table header not found")
	}
}

func TestChunkLargeDocFansOutWithOverlap(t *testing.T) {
	// Build a doc well over the target so it must split into several chunks.
	var b strings.Builder
	b.WriteString("# Big Doc\n\n")
	for i := 0; i < 40; i++ {
		b.WriteString(strings.Repeat("word ", 60))
		b.WriteString("\n\n")
	}
	chunks, err := NewMarkdownChunker().Chunk(docWith(b.String()))
	if err != nil {
		t.Fatalf("Chunk: %v", err)
	}
	if len(chunks) < 2 {
		t.Fatalf("large doc should fan out, got %d chunks", len(chunks))
	}
	// Indices must be sequential from 0.
	for i, c := range chunks {
		if c.Index != i {
			t.Fatalf("chunk %d has index %d", i, c.Index)
		}
		if est := estimateTokens(c.Body); est > HardMaxTokens {
			t.Fatalf("chunk %d exceeds hard max: ~%d tokens", i, est)
		}
	}
	// Overlap: the start of chunk N should re-include the tail of chunk N-1's
	// final paragraph (continuity).
	prevTail := lastParagraph(chunks[0].Body)
	if prevTail != "" && !strings.Contains(chunks[1].Body, prevTail) {
		t.Fatalf("expected overlap of %q into next chunk", prevTail)
	}
}

func TestChunkOversizedCodeFenceStandsAlone(t *testing.T) {
	var code strings.Builder
	code.WriteString("```\n")
	code.WriteString(strings.Repeat("x x x x x x x x\n", 1200)) // far over hard max
	code.WriteString("```")
	chunks, err := NewMarkdownChunker().Chunk(docWith("# C\n\n" + code.String()))
	if err != nil {
		t.Fatalf("Chunk: %v", err)
	}
	var holding int
	for _, c := range chunks {
		if strings.Contains(c.Body, "```") && strings.Contains(c.Body, "x x x") {
			holding++
			// The fence must be intact (open and close present).
			if strings.Count(c.Body, "```") < 2 {
				t.Fatalf("oversized fence not intact in its chunk")
			}
		}
	}
	if holding != 1 {
		t.Fatalf("oversized fence should occupy exactly one chunk, got %d", holding)
	}
}

func lastParagraph(s string) string {
	parts := strings.Split(strings.TrimSpace(s), "\n\n")
	if len(parts) == 0 {
		return ""
	}
	return strings.TrimSpace(parts[len(parts)-1])
}

func TestComposerPrependsContext(t *testing.T) {
	chunk := pkg.Chunk{
		Body: "the body text",
		Meta: map[string]any{
			MetaTitle:       "Architecture",
			MetaDescription: "How it fits together",
			MetaHeadingPath: "Architecture > Components",
		},
	}
	got := NewContextualComposer().Compose(chunk)
	want := "Architecture — How it fits together\nArchitecture > Components\n\nthe body text"
	if got != want {
		t.Fatalf("Compose:\n got %q\nwant %q", got, want)
	}
}

func TestComposerOmitsEmptyPrefix(t *testing.T) {
	chunk := pkg.Chunk{Body: "just body", Meta: map[string]any{}}
	if got := NewContextualComposer().Compose(chunk); got != "just body" {
		t.Fatalf("Compose with no meta = %q, want body verbatim", got)
	}
}

func TestComposerTitleOnly(t *testing.T) {
	chunk := pkg.Chunk{Body: "body", Meta: map[string]any{MetaTitle: "T"}}
	if got := NewContextualComposer().Compose(chunk); got != "T\n\nbody" {
		t.Fatalf("Compose title-only = %q", got)
	}
}

func TestAtxHeadingParsing(t *testing.T) {
	cases := []struct {
		in    string
		level int
		text  string
	}{
		{"# Title", 1, "Title"},
		{"### Deep Section", 3, "Deep Section"},
		{"#NotAHeading", 0, ""},
		{"####### too deep", 0, ""},
		{"plain text", 0, ""},
	}
	for _, c := range cases {
		lvl, txt := atxHeading(c.in)
		if lvl != c.level || txt != c.text {
			t.Fatalf("atxHeading(%q) = (%d,%q), want (%d,%q)", c.in, lvl, txt, c.level, c.text)
		}
	}
}
