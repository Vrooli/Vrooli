package livesearch

import "strings"

// Result is the normalized domain view of a SearXNG hit. Score is the RAW
// SearXNG relevance score, carried through unchanged so downstream consumers
// (and the federation layer) can apply their own normalization.
type Result struct {
	URL      string
	Title    string
	Snippet  string
	Engine   string
	Score    float64
	Category string
}

// Synthesis is the optional L1 summary over the returned snippets. Citation
// grounds every claim back to a result; Abstained signals the snippets were
// insufficient or in conflict.
type Synthesis struct {
	Text      string
	Citations []Citation
	Abstained bool
}

// Citation links a synthesis claim to the Result that supports it.
type Citation struct {
	ResultIndex int
	URL         string
	Title       string
}

// SearchOutcome is the full service result: the L0 results plus the optional
// L1 synthesis and the cache/degraded provenance flags.
type SearchOutcome struct {
	Results        []Result
	Synthesis      *Synthesis
	Cached         bool
	Degraded       bool
	DegradedReason string
	// DegradedEngines lists upstream engines that did not answer this query
	// (per-query signal from SearXNG). Results may be partial when non-empty.
	// On cache hits this reflects the engines at fetch time — acceptable,
	// the TTL bounds the staleness.
	DegradedEngines []EngineIssue
}

// normalize maps a SearXNG RawResult onto the domain Result. The SearXNG
// "content" field is the snippet; the score is carried through raw.
func normalize(r RawResult) Result {
	return Result{
		URL:      strings.TrimSpace(r.URL),
		Title:    strings.TrimSpace(r.Title),
		Snippet:  strings.TrimSpace(r.Content),
		Engine:   strings.TrimSpace(r.Engine),
		Score:    r.Score,
		Category: strings.TrimSpace(r.Category),
	}
}

// normalizeAll maps a slice of raw results onto domain results in order,
// deduplicating by URL: SearXNG fans one query out to many engines, so the
// same page often comes back several times. The highest-scored hit for a URL
// wins (keeping its engine attribution); order follows each URL's first
// appearance so upstream relevance ordering is preserved.
func normalizeAll(raw []RawResult) []Result {
	out := make([]Result, 0, len(raw))
	seen := make(map[string]int, len(raw))
	for _, r := range raw {
		n := normalize(r)
		if i, dup := seen[n.URL]; dup {
			if n.Score > out[i].Score {
				out[i] = n
			}
			continue
		}
		seen[n.URL] = len(out)
		out = append(out, n)
	}
	return out
}
