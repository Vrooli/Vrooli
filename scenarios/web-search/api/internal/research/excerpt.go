package research

import (
	"context"
	"log"
	"math"
	"sort"
	"strings"

	aisearch "github.com/vrooli/ai-go/search"
)

// DefaultExcerptChars is the per-document character budget for what is sent to
// the synthesis model (lever WEB_SEARCH_SYNTH_EXCERPT_CHARS). It matches the
// historical positional truncation cap so the lever's zero value preserves
// behavior.
const DefaultExcerptChars = maxDocChars

// excerptJoin separates non-adjacent selected chunks so the model (and an
// operator reading the excerpt) can see where text was elided.
const excerptJoin = "\n[…]\n"

// Excerpter decides WHAT part of each fetched document is sent to the
// synthesis model, replacing blind positional truncation. Implementations
// never fail: a degraded excerpter falls back to first-N-chars rather than
// blocking the L2 pipeline. Document order (and therefore citation indices)
// is preserved.
type Excerpter interface {
	Select(ctx context.Context, query string, docs []Document) []Document
}

// PositionalExcerpter is the legacy behavior: the first Budget characters of
// each document. It is both the escape hatch
// (WEB_SEARCH_SYNTH_RELEVANT_EXCERPTS=off) and the fallback when the embedder
// is unreachable.
type PositionalExcerpter struct {
	Budget int // per-doc char cap; <=0 means DefaultExcerptChars
}

// Select truncates each document to the budget.
func (p PositionalExcerpter) Select(_ context.Context, _ string, docs []Document) []Document {
	budget := p.Budget
	if budget <= 0 {
		budget = DefaultExcerptChars
	}
	out := make([]Document, len(docs))
	for i, d := range docs {
		if len(d.Text) > budget {
			d.Text = d.Text[:budget]
		}
		out[i] = d
	}
	return out
}

// RelevantExcerpter selects the chunks of each document most relevant to the
// query (by embedding cosine similarity) into the per-doc budget, so a long
// page whose answer lives past the first N characters still contributes its
// relevant section. Selected chunks are reassembled in document order with an
// elision marker. Degradation is graceful by construction: a document that
// fits the budget whole is passed through unembedded, and any embed failure
// falls back to positional truncation for the remaining documents (logged,
// never an error).
type RelevantExcerpter struct {
	Embedder aisearch.Embedder
	Budget   int // per-doc char cap; <=0 means DefaultExcerptChars
	Logger   *log.Logger
}

func (r RelevantExcerpter) logf(format string, args ...any) {
	if r.Logger != nil {
		r.Logger.Printf(format, args...)
	}
}

// Select picks the most query-relevant chunks of each document into the
// budget. Deterministic for a fixed embedder: chunks are scored by cosine
// similarity, ties broken by document position.
func (r RelevantExcerpter) Select(ctx context.Context, query string, docs []Document) []Document {
	budget := r.Budget
	if budget <= 0 {
		budget = DefaultExcerptChars
	}
	positional := PositionalExcerpter{Budget: budget}
	if r.Embedder == nil {
		return positional.Select(ctx, query, docs)
	}

	// Only documents that exceed the budget need relevance selection; if none
	// do, skip the query embed entirely.
	anyOver := false
	for _, d := range docs {
		if len(d.Text) > budget {
			anyOver = true
			break
		}
	}
	if !anyOver {
		return positional.Select(ctx, query, docs)
	}

	queryVec, err := embedQuery(ctx, r.Embedder, query)
	if err != nil {
		r.logf("research: excerpt query embed failed (falling back to positional truncation): %v", err)
		return positional.Select(ctx, query, docs)
	}

	out := make([]Document, len(docs))
	for i, d := range docs {
		if len(d.Text) <= budget {
			out[i] = d
			continue
		}
		excerpt, err := r.selectDoc(ctx, queryVec, d, budget)
		if err != nil {
			r.logf("research: excerpt selection for %q failed (positional fallback): %v", d.URL, err)
			d.Text = d.Text[:budget]
			out[i] = d
			continue
		}
		d.Text = excerpt
		out[i] = d
	}
	return out
}

// selectDoc chunks one over-budget document, scores each chunk against the
// query vector, and reassembles the highest-scoring chunks (in document order)
// within the budget.
func (r RelevantExcerpter) selectDoc(ctx context.Context, queryVec []float64, d Document, budget int) (string, error) {
	chunks, err := aisearch.NewMarkdownChunker().Chunk(aisearch.SourceDoc{ID: d.URL, Body: d.Text})
	if err != nil || len(chunks) == 0 {
		return d.Text[:budget], err
	}
	if len(chunks) == 1 {
		return d.Text[:budget], nil
	}

	type scored struct {
		index int
		body  string
		score float64
	}
	scoredChunks := make([]scored, 0, len(chunks))
	for _, c := range chunks {
		vec, err := embedDocument(ctx, r.Embedder, c.Body)
		if err != nil {
			return "", err
		}
		scoredChunks = append(scoredChunks, scored{index: c.Index, body: c.Body, score: cosine(queryVec, vec)})
	}

	// Highest similarity first; document position breaks ties deterministically.
	sort.Slice(scoredChunks, func(i, j int) bool {
		if scoredChunks[i].score != scoredChunks[j].score {
			return scoredChunks[i].score > scoredChunks[j].score
		}
		return scoredChunks[i].index < scoredChunks[j].index
	})

	selected := make([]scored, 0, len(scoredChunks))
	used := 0
	for _, c := range scoredChunks {
		cost := len(c.body)
		if used > 0 {
			cost += len(excerptJoin)
		}
		if used+cost > budget {
			continue
		}
		selected = append(selected, c)
		used += cost
	}
	if len(selected) == 0 {
		// Even the best chunk exceeds the budget — take its head.
		return scoredChunks[0].body[:budget], nil
	}

	// Reassemble in document order so the excerpt reads top-to-bottom.
	sort.Slice(selected, func(i, j int) bool { return selected[i].index < selected[j].index })
	parts := make([]string, len(selected))
	for i, c := range selected {
		parts[i] = c.body
	}
	return strings.Join(parts, excerptJoin), nil
}

// embedQuery / embedDocument use the asymmetric task prefixes when the
// embedder supports them (mirrors aisearch-go's read/index split).
func embedQuery(ctx context.Context, e aisearch.Embedder, text string) ([]float64, error) {
	if te, ok := e.(aisearch.TaskEmbedder); ok {
		return te.EmbedQuery(ctx, text)
	}
	return e.Embed(ctx, text)
}

func embedDocument(ctx context.Context, e aisearch.Embedder, text string) ([]float64, error) {
	if te, ok := e.(aisearch.TaskEmbedder); ok {
		return te.EmbedDocument(ctx, text)
	}
	return e.Embed(ctx, text)
}

func cosine(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		dot += a[i] * b[i]
		na += a[i] * a[i]
		nb += b[i] * b[i]
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

var (
	_ Excerpter = PositionalExcerpter{}
	_ Excerpter = RelevantExcerpter{}
)
