package recall

import (
	"math"
	"slices"
	"strings"
	"unicode"
)

// Favor coverage of the subject, adjacent query terms, and focused documents.
// A long canopy node that mentions two terms in unrelated paragraphs should
// not tie a note dedicated to those terms.

func lexicalScores(query string, nodes []Node) []float64 {
	terms := tokens(query)
	scores := make([]float64, len(nodes))
	if len(terms) == 0 || len(nodes) == 0 {
		return scores
	}
	// Count only terms the query asks about. Keeping every distinct term in
	// every note makes fallback spend the provider's deadline allocating.
	termIndex := make(map[string]int, len(terms))
	for i, term := range terms {
		termIndex[term] = i
	}
	frequency := make([]int, len(terms))
	lastSeen := make([]int, len(terms))
	lastPairSeen := make([]int, len(terms))
	matches := make([][]int, len(nodes))
	adjacent := make([]int, len(nodes))
	wordCounts := make([]int, len(nodes))
	for i, node := range nodes {
		previous := -2
		for word := range strings.FieldsFuncSeq(strings.ToLower(node.Text), lexicalSeparator) {
			wordCounts[i]++
			term, ok := termIndex[word]
			if !ok {
				previous = -2
				continue
			}
			if term == previous+1 && lastPairSeen[term] != i+1 {
				lastPairSeen[term] = i + 1
				adjacent[i]++
			}
			previous = term
			if lastSeen[term] != i+1 {
				lastSeen[term] = i + 1
				frequency[term]++
				matches[i] = append(matches[i], term)
			}
		}
	}
	var total float64
	weights := make([]float64, len(terms))
	for i, count := range frequency {
		weights[i] = math.Log(1 + float64(len(nodes))/float64(1+count))
		total += weights[i]
	}
	for i, matched := range matches {
		slices.Sort(matched)
		var got float64
		for _, term := range matched {
			got += weights[term]
		}
		coverage := got / total
		focus := 160 / (160 + float64(wordCounts[i]))
		proximity := 0.0
		if len(terms) > 1 {
			proximity = float64(adjacent[i]) / float64(len(terms)-1)
		}
		scores[i] = 0.99 * coverage * (0.62 + 0.24*focus + 0.14*proximity)
	}
	return scores
}

func lexicalSeparator(r rune) bool {
	return !unicode.IsLetter(r) && !unicode.IsDigit(r)
}

func lexicalWords(text string) []string {
	return strings.FieldsFunc(strings.ToLower(text), lexicalSeparator)
}

func tokens(text string) []string {
	fields := lexicalWords(text)
	out := fields[:0]
	seen := map[string]bool{}
	for _, f := range fields {
		if len(f) > 1 && !seen[f] && !queryStopWord(f) {
			seen[f] = true
			out = append(out, f)
		}
	}
	return out
}

func queryStopWord(word string) bool {
	switch word {
	case "an", "and", "are", "can", "did", "do", "does", "for", "how", "in", "is", "it",
		"me", "of", "on", "or", "the", "to", "was", "we", "what", "when", "where", "which", "with":
		return true
	default:
		return false
	}
}
