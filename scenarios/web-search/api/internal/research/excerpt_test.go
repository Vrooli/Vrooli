package research

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// keywordEmbedder is a deterministic fake: the vector is 1.0 on axis 0 when
// the text contains the keyword, else 1.0 on axis 1. Chunks mentioning the
// keyword therefore score cosine=1 against the query and others score 0.
type keywordEmbedder struct {
	keyword string
	fail    bool
	calls   int
}

func (k *keywordEmbedder) Embed(_ context.Context, text string) ([]float64, error) {
	k.calls++
	if k.fail {
		return nil, errors.New("embedder down")
	}
	if strings.Contains(text, k.keyword) {
		return []float64{1, 0}, nil
	}
	return []float64{0, 1}, nil
}

func (k *keywordEmbedder) Available(context.Context) bool { return !k.fail }

// section builds a markdown section whose body repeats filler to a size.
func section(heading, body string, repeat int) string {
	return "## " + heading + "\n\n" + strings.Repeat(body+" ", repeat) + "\n\n"
}

// TestRelevantExcerpterPicksQueryRelevantSection: a long page whose relevant
// section sits PAST the positional budget must still contribute that section
// (the exact failure mode of first-N-chars truncation).
func TestRelevantExcerpterPicksQueryRelevantSection(t *testing.T) {
	// ~12k chars of irrelevant prose followed by the relevant section.
	page := section("Intro", "unrelated filler prose about nothing in particular", 60) +
		section("History", "more unrelated filler prose that pads the page", 60) +
		section("Padding", "yet more filler to push the answer past any prefix", 60) +
		section("Answer", "zebra migration patterns are seasonal", 30)
	if len(page) < 8000 {
		t.Fatalf("fixture page too short (%d chars) to exercise the budget", len(page))
	}

	emb := &keywordEmbedder{keyword: "zebra"}
	ex := RelevantExcerpter{Embedder: emb, Budget: 2000}
	docs := ex.Select(context.Background(), "zebra migration", []Document{{URL: "https://a", Title: "A", Text: page}})

	if len(docs) != 1 {
		t.Fatalf("got %d docs, want 1", len(docs))
	}
	if !strings.Contains(docs[0].Text, "zebra migration patterns") {
		t.Fatalf("excerpt lost the query-relevant section:\n%s", docs[0].Text[:200])
	}
	if len(docs[0].Text) > 2000 {
		t.Fatalf("excerpt %d chars exceeds the 2000-char budget", len(docs[0].Text))
	}
}

// TestRelevantExcerpterIsDeterministic: same inputs, same embedder → byte-
// identical excerpts across runs.
func TestRelevantExcerpterIsDeterministic(t *testing.T) {
	page := section("One", "alpha filler text for the first section", 40) +
		section("Two", "zebra relevant text for the query", 40) +
		section("Three", "gamma filler text for the third section", 40)

	run := func() string {
		ex := RelevantExcerpter{Embedder: &keywordEmbedder{keyword: "zebra"}, Budget: 1500}
		docs := ex.Select(context.Background(), "zebra", []Document{{URL: "https://a", Title: "A", Text: page}})
		return docs[0].Text
	}
	first := run()
	for i := 0; i < 3; i++ {
		if got := run(); got != first {
			t.Fatalf("selection not deterministic on run %d", i+2)
		}
	}
}

// TestRelevantExcerpterPreservesDocumentOrder: citation indices point into the
// document slice, so the excerpter must never reorder or drop documents.
func TestRelevantExcerpterPreservesDocumentOrder(t *testing.T) {
	long := section("Pad", "filler prose to exceed the budget by a lot", 80) +
		section("Hit", "zebra text", 20)
	in := []Document{
		{URL: "https://first", Title: "F", Text: "short doc"},
		{URL: "https://second", Title: "S", Text: long},
		{URL: "https://third", Title: "T", Text: "another short doc"},
	}
	ex := RelevantExcerpter{Embedder: &keywordEmbedder{keyword: "zebra"}, Budget: 1000}
	out := ex.Select(context.Background(), "zebra", in)

	if len(out) != len(in) {
		t.Fatalf("doc count changed: %d -> %d", len(in), len(out))
	}
	for i := range in {
		if out[i].URL != in[i].URL {
			t.Fatalf("doc %d reordered: %s -> %s (citation indices would break)", i, in[i].URL, out[i].URL)
		}
	}
	// Short docs pass through whole.
	if out[0].Text != "short doc" || out[2].Text != "another short doc" {
		t.Fatalf("under-budget docs must pass through unmodified")
	}
}

// TestRelevantExcerpterFallsBackPositionally: embedder failure must degrade to
// first-N-chars truncation — never an error, never an unbounded excerpt.
func TestRelevantExcerpterFallsBackPositionally(t *testing.T) {
	page := strings.Repeat("word ", 3000) // 15k chars
	ex := RelevantExcerpter{Embedder: &keywordEmbedder{fail: true}, Budget: 1000}
	out := ex.Select(context.Background(), "anything", []Document{{URL: "https://a", Title: "A", Text: page}})

	if len(out) != 1 {
		t.Fatalf("got %d docs, want 1", len(out))
	}
	if out[0].Text != page[:1000] {
		t.Fatalf("fallback must be exact positional truncation (got %d chars)", len(out[0].Text))
	}
}

// TestRelevantExcerpterSkipsEmbedsWhenAllFit: documents within budget pass
// through without a single embed call (no needless ollama traffic on short
// pages).
func TestRelevantExcerpterSkipsEmbedsWhenAllFit(t *testing.T) {
	emb := &keywordEmbedder{keyword: "x"}
	ex := RelevantExcerpter{Embedder: emb, Budget: 1000}
	out := ex.Select(context.Background(), "q", []Document{{URL: "https://a", Title: "A", Text: "tiny"}})
	if out[0].Text != "tiny" {
		t.Fatalf("under-budget doc must pass through")
	}
	if emb.calls != 0 {
		t.Fatalf("embedder called %d times for under-budget docs, want 0", emb.calls)
	}
}

// TestPositionalExcerpterTruncates pins the escape-hatch behavior.
func TestPositionalExcerpterTruncates(t *testing.T) {
	page := strings.Repeat("a", 9000)
	out := PositionalExcerpter{}.Select(context.Background(), "q", []Document{{Text: page}})
	if len(out[0].Text) != DefaultExcerptChars {
		t.Fatalf("default budget = %d chars, got %d", DefaultExcerptChars, len(out[0].Text))
	}
}
