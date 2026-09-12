package measureindex

import (
	"context"
	"sort"
	"strings"

	measures "github.com/vrooli/measures-go"
)

// LexicalMatcher is a deterministic, infra-free implementation of the measures
// Matcher seam: it scores each indexed declaration by token overlap against the
// curated natural-language questions[] (the embedding key the aisearch index
// would vectorize), plus the intent / name / domain as weaker signals, and
// returns the best matches best-first. It needs no ollama or qdrant, so the
// federated measures provider answers (and is fully testable) even with no
// vector backend — exactly the role cli-health's text fallback plays. When the
// plan's aisearch hybrid index lands it implements this same interface and the
// provider swaps one for the other (or layers them) with no other change.
type LexicalMatcher struct {
	decls []measures.MeasureDeclaration
	// docs is the precomputed lowercased token set per declaration, aligned with
	// decls by index, so Match does not re-tokenize the corpus on every query.
	docs []scoredDoc
}

type scoredDoc struct {
	questionTokens map[string]struct{}
	groundTokens   map[string]struct{} // intent + name + domain (weaker signal)
}

// NewLexicalMatcher builds a matcher over the harvested corpus, precomputing the
// per-declaration token sets.
func NewLexicalMatcher(decls []measures.MeasureDeclaration) *LexicalMatcher {
	m := &LexicalMatcher{decls: decls, docs: make([]scoredDoc, len(decls))}
	for i, d := range decls {
		m.docs[i] = scoredDoc{
			questionTokens: tokenSet(strings.Join(d.Questions, " ")),
			groundTokens:   tokenSet(d.Intent + " " + d.Name + " " + d.Domain),
		}
	}
	return m
}

// Len reports the number of indexed measures (the Status indexed_count).
func (m *LexicalMatcher) Len() int { return len(m.decls) }

// Match scores the question against every indexed measure and returns up to
// `limit` matches above a small relevance floor, best-first. A question that
// shares no salient term with any measure returns no matches (an honest empty
// result, never a guess). Score is normalized to (0,1].
func (m *LexicalMatcher) Match(_ context.Context, question string, limit int) ([]measures.Match, error) {
	terms := tokenize(question)
	if len(terms) == 0 || len(m.decls) == 0 {
		return nil, nil
	}
	type cand struct {
		idx   int
		score float64
	}
	cands := make([]cand, 0, len(m.decls))
	for i := range m.decls {
		s := m.scoreDoc(i, terms)
		if s <= 0 {
			continue
		}
		cands = append(cands, cand{idx: i, score: s})
	}
	sort.Slice(cands, func(a, b int) bool {
		if cands[a].score != cands[b].score {
			return cands[a].score > cands[b].score
		}
		// Deterministic tie-break by measure name.
		return m.decls[cands[a].idx].Name < m.decls[cands[b].idx].Name
	})
	if limit <= 0 {
		limit = 1
	}
	if len(cands) > limit {
		cands = cands[:limit]
	}
	out := make([]measures.Match, 0, len(cands))
	for _, c := range cands {
		out = append(out, measures.Match{Decl: m.decls[c.idx], Score: c.score})
	}
	return out, nil
}

// scoreDoc scores one declaration against the query terms. A term hitting a
// curated question counts most (questions are the embedding key); a term hitting
// the intent/name/domain grounding counts less. The raw score is normalized by
// the maximum achievable (every term a question hit) so the result lands in
// (0,1], comparable across measures and clamped for the cosine score scale the
// provider declares to search-hub.
func (m *LexicalMatcher) scoreDoc(idx int, terms []string) float64 {
	doc := m.docs[idx]
	const (
		wQuestion = 1.0
		wGround   = 0.4
	)
	var raw float64
	for _, t := range terms {
		switch {
		case contains(doc.questionTokens, t):
			raw += wQuestion
		case contains(doc.groundTokens, t):
			raw += wGround
		}
	}
	if raw <= 0 {
		return 0
	}
	max := float64(len(terms)) * wQuestion
	n := raw / max
	if n > 1 {
		n = 1
	}
	return n
}

func contains(set map[string]struct{}, t string) bool {
	_, ok := set[t]
	return ok
}

// tokenSet returns the unique salient tokens of s as a set.
func tokenSet(s string) map[string]struct{} {
	toks := tokenize(s)
	set := make(map[string]struct{}, len(toks))
	for _, t := range toks {
		set[t] = struct{}{}
	}
	return set
}

// tokenize lowercases, splits on non-word runes, drops 1-char tokens and a small
// stoplist of analytical-question filler so "how many backlog items did we
// complete this week" keys on the salient terms (backlog, items, complete, week).
func tokenize(q string) []string {
	q = strings.ToLower(q)
	fields := strings.FieldsFunc(q, func(r rune) bool {
		switch r {
		case ' ', '\t', '\n', '\r', ',', ';', ':', '/', '\\', '-', '_', '.', '?', '!', '"', '\'', '(', ')', '[', ']':
			return true
		}
		return false
	})
	out := make([]string, 0, len(fields))
	seen := make(map[string]struct{}, len(fields))
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if len(f) < 2 {
			continue
		}
		if _, stop := stopwords[f]; stop {
			continue
		}
		if _, ok := seen[f]; ok {
			continue
		}
		seen[f] = struct{}{}
		out = append(out, f)
	}
	return out
}

// stopwords are high-frequency analytical-question filler that carry no measure-
// discriminating signal; dropping them keeps a short query's salient terms from
// being diluted in the normalized score.
var stopwords = map[string]struct{}{
	"how": {}, "many": {}, "much": {}, "what": {}, "whats": {}, "the": {}, "did": {},
	"do": {}, "we": {}, "is": {}, "are": {}, "of": {}, "in": {}, "on": {}, "for": {},
	"to": {}, "a": {}, "an": {}, "and": {}, "or": {}, "this": {}, "that": {}, "my": {},
	"me": {}, "our": {}, "us": {}, "get": {}, "show": {}, "list": {}, "count": {},
}
