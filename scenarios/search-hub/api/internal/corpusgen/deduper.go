package corpusgen

import "strings"

// Deduper decides whether a candidate query is a near-duplicate of any query in
// `seen`. It is a SEAM: the production impl is a lexical token-Jaccard matcher
// (below) — a pragmatic stand-in for true semantic de-dup, since search-hub
// holds no embedder of its own (embeddings live provider-side). When an embedder
// becomes available to search-hub, a cosine-over-embeddings Deduper drops in
// here with no change to the Generator. Keeping it behind the seam is the point:
// the generator's contract is "don't propose a paraphrase of a case we already
// have", and how "paraphrase" is judged can sharpen later.
type Deduper interface {
	// IsDuplicate reports whether candidate is a near-duplicate of any seen query.
	IsDuplicate(candidate string, seen []string) bool
}

// DefaultSimilarityThreshold is the Jaccard token-overlap at or above which two
// queries are treated as the same question. 0.8 catches reorderings and minor
// wording changes ("restart the api" vs "restart api service") while leaving
// genuinely distinct queries that merely share vocabulary.
const DefaultSimilarityThreshold = 0.8

// JaccardDeduper flags a candidate as duplicate when its normalized token set
// overlaps any seen query's by >= Threshold (Jaccard), or matches one exactly.
type JaccardDeduper struct {
	// Threshold in (0,1]; <=0 falls back to DefaultSimilarityThreshold.
	Threshold float64
}

// IsDuplicate implements Deduper.
func (d JaccardDeduper) IsDuplicate(candidate string, seen []string) bool {
	threshold := d.Threshold
	if threshold <= 0 {
		threshold = DefaultSimilarityThreshold
	}
	cand := tokenSet(candidate)
	if len(cand) == 0 {
		// An empty candidate carries no signal — treat as duplicate so it is
		// never proposed.
		return true
	}
	for _, s := range seen {
		if jaccard(cand, tokenSet(s)) >= threshold {
			return true
		}
	}
	return false
}

// tokenSet lowercases, splits on whitespace, and dedupes into a set.
func tokenSet(s string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, tok := range strings.Fields(strings.ToLower(s)) {
		out[tok] = struct{}{}
	}
	return out
}

// jaccard is |a∩b| / |a∪b| (0 when both empty).
func jaccard(a, b map[string]struct{}) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 0
	}
	inter := 0
	for t := range a {
		if _, ok := b[t]; ok {
			inter++
		}
	}
	union := len(a) + len(b) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}
