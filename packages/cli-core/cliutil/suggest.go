// Package-level nearest-match suggestion helper. Backs the "did you mean X?"
// hints on unknown commands (cliapp) and unknown flags (scenario CLIs), so an
// agent's typo costs one corrected retry instead of a help-lookup roundtrip.
package cliutil

// NearestString returns the option closest to candidate by edit distance, or
// "" when nothing is within maxDist. Ties break to the lexicographically
// smaller option so the suggestion is deterministic.
func NearestString(candidate string, options []string, maxDist int) string {
	if candidate == "" {
		return ""
	}
	best := ""
	bestDist := -1
	for _, opt := range options {
		if opt == "" {
			continue
		}
		d := editDistance(candidate, opt)
		if bestDist == -1 || d < bestDist || (d == bestDist && opt < best) {
			best, bestDist = opt, d
		}
	}
	if bestDist == -1 || bestDist > maxDist {
		return ""
	}
	return best
}

// editDistance is a small two-row Levenshtein implementation; it only ever
// runs over short command/flag vocabularies.
func editDistance(a, b string) int {
	if a == b {
		return 0
	}
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	curr := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		curr[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			curr[j] = min(prev[j]+1, curr[j-1]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[len(rb)]
}
