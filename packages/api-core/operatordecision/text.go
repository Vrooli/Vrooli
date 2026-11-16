package operatordecision

import (
	"encoding/json"
	"strconv"
	"strings"
)

// Clip cuts text to at most n bytes of valid UTF-8, the ellipsis included, at
// a word boundary when one is in the second half. Bytes bound both the Hub's
// byte limits (question, reason, labels) and its code-point limits, so one
// helper serves every decision field and page budget.
func Clip(text string, n int) string {
	text = strings.TrimSpace(text)
	if n <= 0 || len(text) <= n {
		return text
	}
	const mark = "…"
	if n <= len(mark) {
		return strings.ToValidUTF8(text[:n], "")
	}
	cut := strings.ToValidUTF8(text[:n-len(mark)], "")
	if i := strings.LastIndexAny(cut, " \n"); i > len(cut)/2 {
		cut = cut[:i]
	}
	return strings.TrimRight(cut, " \n,.;:") + mark
}

// Plural picks the word for n: Plural(n, "it", "them").
func Plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// Count writes n with its word: "1 file", "3 files".
func Count(n int, one, many string) string {
	return strconv.Itoa(n) + " " + Plural(n, one, many)
}

// EncodedLen is the size of s inside a JSON event payload, the measure
// MaxExplainerBytes bounds. An empty field is omitted, so it is 0.
func EncodedLen(s string) int {
	if s == "" {
		return 0
	}
	encoded, _ := json.Marshal(s)
	return len(encoded)
}

// Fit builds a page with a content budget (diff bytes, table rows) and, when
// the page would not fit in a decision event, builds it again with a quarter
// of the budget, then with none. A page that never fits is left out; the
// cover still carries the decision.
func Fit(budget int, build func(budget int) string) string {
	for _, b := range []int{budget, budget / 4, 0} {
		if page := build(b); EncodedLen(page) <= MaxExplainerBytes {
			return page
		}
	}
	return ""
}
