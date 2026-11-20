package search

import (
	"strings"
	"unicode"
)

// queryUnit is one AND-ed clause: either a single token (searched as a prefix)
// or a quoted phrase (searched as adjacent tokens). The planner never passes raw
// user text to FTS5, so quotes, parentheses, colons, dashes and the NEAR/OR
// operators in a query are data rather than syntax.
type queryUnit struct {
	words []string
}

type queryPlan struct {
	units      []queryUnit
	andQuery   string
	orQuery    string
	words      []string
	normalized string
}

func (p queryPlan) empty() bool { return len(p.units) == 0 }

// planQuery turns a free-form query into a safe FTS5 expression.
//
// The grammar is intentionally small: quoted runs become phrases, every other
// run of letters/numbers becomes one prefix token. That is enough for the
// workbench's queries and removes the entire class of FTS5 syntax injection,
// because the only characters that ever reach MATCH are our own quotes,
// asterisks and the AND/OR keywords.
func planQuery(query string) queryPlan {
	type segment struct {
		text   string
		quoted bool
	}
	var segments []segment
	var current strings.Builder
	quoted := false
	for _, r := range query {
		if r == '"' {
			segments = append(segments, segment{text: current.String(), quoted: quoted})
			current.Reset()
			quoted = !quoted
			continue
		}
		current.WriteRune(r)
	}
	segments = append(segments, segment{text: current.String(), quoted: quoted})

	var plan queryPlan
	for _, seg := range segments {
		words := tokenize(seg.text)
		if len(words) == 0 {
			continue
		}
		if seg.quoted {
			plan.units = append(plan.units, queryUnit{words: words})
			continue
		}
		for _, word := range words {
			plan.units = append(plan.units, queryUnit{words: []string{word}})
		}
	}
	if len(plan.units) == 0 {
		return plan
	}
	andParts := make([]string, 0, len(plan.units))
	orParts := make([]string, 0, len(plan.units))
	for _, unit := range plan.units {
		literal := ftsLiteral(unit.words)
		andParts = append(andParts, literal)
		orParts = append(orParts, literal)
		plan.words = append(plan.words, unit.words...)
	}
	plan.andQuery = strings.Join(andParts, " AND ")
	plan.orQuery = strings.Join(orParts, " OR ")
	plan.normalized = strings.Join(plan.words, " ")
	return plan
}

func tokenize(value string) []string {
	return strings.FieldsFunc(strings.ToLower(value), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r) && !unicode.IsMark(r)
	})
}

// ftsLiteral renders one planned unit as an FTS5 string literal, with a prefix
// marker only for single tokens. A phrase must stay one quoted string; quoting
// each word separately would leave FTS5 free to match them anywhere in the
// document instead of adjacent.
func ftsLiteral(words []string) string {
	escaped := make([]string, 0, len(words))
	for _, word := range words {
		escaped = append(escaped, strings.ReplaceAll(word, `"`, `""`))
	}
	literal := `"` + strings.Join(escaped, " ") + `"`
	if len(words) == 1 {
		return literal + "*"
	}
	return literal
}

// snippet returns a short body excerpt around the first matched term. It is
// byte-safe for multi-byte text because the window is measured in runes.
func snippet(body string, words []string, limit int) string {
	body = strings.Join(strings.Fields(body), " ")
	if body == "" {
		return ""
	}
	if limit <= 0 {
		limit = 160
	}
	runes := []rune(body)
	if len(runes) <= limit {
		return body
	}
	lower := strings.ToLower(body)
	start := 0
	for _, word := range words {
		if index := strings.Index(lower, word); index >= 0 {
			// Convert the byte offset to a rune offset before windowing.
			runesBefore := len([]rune(lower[:index]))
			start = runesBefore - limit/3
			if start < 0 {
				start = 0
			}
			break
		}
	}
	end := start + limit
	if end > len(runes) {
		end = len(runes)
		start = end - limit
		if start < 0 {
			start = 0
		}
	}
	text := string(runes[start:end])
	if start > 0 {
		text = "…" + text
	}
	if end < len(runes) {
		text += "…"
	}
	return text
}

// truncateBody bounds the FTS payload for records whose JSON spec is large.
// Search should index the page, not embed an unbounded document in the index.
func truncateBody(body string, limit int) string {
	if limit <= 0 {
		return body
	}
	runes := []rune(body)
	if len(runes) <= limit {
		return body
	}
	return string(runes[:limit]) + "…"
}
