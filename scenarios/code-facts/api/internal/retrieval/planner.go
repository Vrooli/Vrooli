package retrieval

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

var relationshipSignals = regexp.MustCompile(`(?i)\b(calls?|callers?|callees?|references?|imports?|depends?|uses?)\b`)

type QueryPlanner struct{}

func (QueryPlanner) Plan(query Query) Plan {
	normalized := strings.TrimSpace(query.Text)
	regime := RegimeNatural
	switch {
	case relationshipSignals.MatchString(normalized):
		regime = RegimeRelationship
	case containsFold(query.Families, "contract") || strings.Contains(strings.ToLower(normalized), "rpc "):
		regime = RegimeContract
	case isExactSignal(normalized):
		regime = RegimeExact
	}
	return Plan{Regime: regime, UseLexical: true, UseSemantic: regime != RegimeExact, UseReranker: regime == RegimeNatural}
}

func normalizeDocumentText(value string) string {
	return strings.Join(splitIdentifier(value), " ")
}

func normalizeExact(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func splitIdentifier(value string) []string {
	runes := []rune(value)
	var tokens []string
	var current []rune
	flush := func() {
		if len(current) == 0 {
			return
		}
		tokens = append(tokens, strings.ToLower(string(current)))
		current = current[:0]
	}
	var previous rune
	for index, next := range runes {
		if !unicode.IsLetter(next) && !unicode.IsDigit(next) {
			flush()
			previous = 0
			continue
		}
		nextIsLower := index+1 < len(runes) && unicode.IsLower(runes[index+1])
		if unicode.IsUpper(next) && len(current) > 0 && (unicode.IsLower(previous) || unicode.IsDigit(previous) || (unicode.IsUpper(previous) && nextIsLower)) {
			flush()
		}
		current = append(current, unicode.ToLower(next))
		previous = next
	}
	flush()
	return tokens
}

// NormalizeQuery parses explicit filters once. Unknown prefixes stay in the
// search text so future syntax cannot silently broaden a request.
func NormalizeQuery(query Query) (Query, error) {
	words := strings.Fields(strings.TrimSpace(query.Text))
	text := make([]string, 0, len(words))
	for _, word := range words {
		key, value, found := strings.Cut(word, ":")
		if !found || value == "" {
			text = append(text, word)
			continue
		}
		switch strings.ToLower(key) {
		case "target", "path":
			query.Target = value
		case "scope":
			query.Scope = value
		case "role":
			query.Roles = append(query.Roles, value)
		case "family", "kind":
			query.Families = append(query.Families, value)
		case "language", "lang":
			query.Languages = append(query.Languages, value)
		case "limit":
			limit, err := strconv.Atoi(value)
			if err != nil || limit <= 0 || limit > 100 {
				return Query{}, fmt.Errorf("query limit must be between 1 and 100")
			}
			query.Limit = limit
		default:
			text = append(text, word)
		}
	}
	query.Text = strings.Join(text, " ")
	if query.Text == "" {
		return Query{}, fmt.Errorf("query text is required after filters")
	}
	return query, nil
}

func isExactSignal(value string) bool {
	if value == "" || strings.ContainsAny(value, "\n\t") {
		return false
	}
	if strings.Contains(value, "/") || filepath.Ext(value) != "" {
		return true
	}
	return len(strings.Fields(value)) == 1
}

func containsFold(values []string, needle string) bool {
	for _, value := range values {
		if strings.EqualFold(value, needle) {
			return true
		}
	}
	return false
}
