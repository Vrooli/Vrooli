package recall

import (
	"math"
	"strings"
	"unicode"
)

// The lexical leg keeps the journal searchable when the embedding route is
// down: recall ranks by term overlap and says so instead of failing.

// lexicalScores ranks nodes by IDF-weighted query-term overlap, scaled to
// [0,1) so the fallback never claims cosine-level confidence.
func lexicalScores(query string, nodes []Node) []float64 {
	terms := tokens(query)
	scores := make([]float64, len(nodes))
	if len(terms) == 0 || len(nodes) == 0 {
		return scores
	}
	docs := make([]map[string]bool, len(nodes))
	df := map[string]int{}
	for i, n := range nodes {
		docs[i] = map[string]bool{}
		for _, t := range tokens(n.Text) {
			if !docs[i][t] {
				docs[i][t] = true
				df[t]++
			}
		}
	}
	var total float64
	idf := make(map[string]float64, len(terms))
	for _, t := range terms {
		idf[t] = math.Log(1 + float64(len(nodes))/float64(1+df[t]))
		total += idf[t]
	}
	for i := range nodes {
		var got float64
		for _, t := range terms {
			if docs[i][t] {
				got += idf[t]
			}
		}
		scores[i] = 0.99 * got / total
	}
	return scores
}

func tokens(text string) []string {
	fields := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	out := fields[:0]
	seen := map[string]bool{}
	for _, f := range fields {
		if len(f) > 1 && !seen[f] {
			seen[f] = true
			out = append(out, f)
		}
	}
	return out
}
