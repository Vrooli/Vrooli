package facts

import (
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
)

// searchFieldWeight keeps source-backed identity and location fields more
// influential than a line of source text. Code Facts is authoritative about
// paths and identifiers, so natural-language search can find a source
// location without requiring every query word on the same line.
type searchFieldWeight struct {
	value  string
	weight float64
}

var searchStopWords = map[string]struct{}{
	"a": {}, "an": {}, "and": {}, "are": {}, "as": {}, "at": {},
	"be": {}, "by": {}, "do": {}, "does": {}, "for": {}, "from": {},
	"class": {}, "classes": {}, "code": {}, "file": {}, "files": {},
	"implementation": {}, "implement": {},
	"how": {}, "in": {}, "into": {}, "is": {}, "it": {}, "of": {},
	"location": {}, "locations": {}, "method": {}, "methods": {}, "module": {}, "modules": {},
	"on": {}, "or": {}, "that": {}, "the": {}, "these": {}, "this": {},
	"those": {}, "to": {}, "router": {}, "routers": {}, "source": {}, "was": {}, "were": {}, "what": {}, "which": {},
	"where": {}, "with": {},
}

func searchQueryTokens(query string) []string {
	return uniqueSearchTokens(tokenizeSearchText(query), true)
}

func tokenizeSearchText(text string) []string {
	var tokens []string
	var current []rune
	var previous rune
	for _, r := range text {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			if len(current) > 0 {
				tokens = append(tokens, normalizeSearchToken(string(current)))
				current = current[:0]
			}
			previous = 0
			continue
		}
		if unicode.IsUpper(r) && len(current) > 0 && (unicode.IsLower(previous) || unicode.IsDigit(previous)) {
			tokens = append(tokens, normalizeSearchToken(string(current)))
			current = current[:0]
		}
		current = append(current, unicode.ToLower(r))
		previous = r
	}
	if len(current) > 0 {
		tokens = append(tokens, normalizeSearchToken(string(current)))
	}
	return tokens
}

func normalizeSearchToken(token string) string {
	token = strings.ToLower(strings.TrimSpace(token))
	switch token {
	case "generated", "generation":
		return "gen"
	case "function", "functions":
		return "func"
	case "declared", "declaration", "declarations":
		return "declar"
	}
	if len(token) > 5 && strings.HasSuffix(token, "ies") {
		return strings.TrimSuffix(token, "ies") + "y"
	}
	if len(token) > 5 && strings.HasSuffix(token, "ing") {
		return strings.TrimSuffix(token, "ing")
	}
	if len(token) > 4 && strings.HasSuffix(token, "ed") {
		return strings.TrimSuffix(token, "ed")
	}
	if len(token) > 3 && strings.HasSuffix(token, "s") {
		return strings.TrimSuffix(token, "s")
	}
	return token
}

func uniqueSearchTokens(tokens []string, removeStopWords bool) []string {
	seen := make(map[string]struct{}, len(tokens))
	result := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if token == "" {
			continue
		}
		if removeStopWords {
			if _, ok := searchStopWords[token]; ok {
				continue
			}
		}
		if _, ok := seen[token]; ok {
			continue
		}
		seen[token] = struct{}{}
		result = append(result, token)
	}
	return result
}

func factSearchFields(fact *factsv1.GenericFact) []searchFieldWeight {
	attrs := fact.GetAttributes()
	id := fact.GetId()
	if strings.HasPrefix(id, "code-facts:lexical:") {
		// The lexical adapter's namespace is implementation metadata, not
		// source evidence. Its generic words (code/facts/lexical) must not make
		// every indexed line appear relevant to a query.
		id = ""
	}
	fields := []searchFieldWeight{
		{value: id, weight: 5},
		{value: attrs["path"], weight: 9},
		{value: attrs["name"], weight: 6},
		{value: attrs["qualified_name"], weight: 6},
		{value: attrs["route_path"], weight: 5},
		{value: attrs["import_path"], weight: 5},
		{value: fact.GetSubject(), weight: 3},
		{value: fact.GetKind(), weight: 2},
	}

	keys := make([]string, 0, len(attrs))
	for key := range attrs {
		switch key {
		case "path", "name", "qualified_name", "route_path", "import_path", "analyzer", "line":
			continue
		default:
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		fields = append(fields, searchFieldWeight{value: attrs[key], weight: 1})
	}
	return fields
}

// shouldIndexLexicalLine keeps the project index bounded to source-bearing
// lines. The repository contains millions of generated and fixture lines;
// indexing every brace, delimiter, and generated field makes the provider
// consume disproportionate memory without improving a natural-language code
// answer. File anchors, declarations, comments, and contract vocabulary are
// retained so source-backed locations remain discoverable and explainable.
func shouldIndexLexicalLine(path string, line int, subject string) bool {
	trimmed := strings.TrimSpace(subject)
	if trimmed == "" {
		return false
	}
	if line == 1 {
		return true
	}
	lower := strings.ToLower(trimmed)
	generated := strings.Contains(filepath.ToSlash(path), "/proto/gen/")
	if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*") {
		if generated {
			return strings.Contains(lower, "service") || strings.Contains(lower, "client") || strings.Contains(lower, "request") || strings.Contains(lower, "response") || strings.Contains(lower, "contract")
		}
		return len(trimmed) >= 24
	}
	markers := []string{
		"func ", "type ", "interface ", "class ", "service ", "rpc ", "message ", "enum ",
		"package ", "import ", "route", "handler", "provider", "search", "contract", "declar",
		"request", "response", "client", "server", "endpoint", "register", "demot", "adoption",
	}
	for _, marker := range markers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func scoreSearchFact(fact *factsv1.GenericFact, query string, queryTokens []string) float64 {
	if fact == nil || len(queryTokens) == 0 {
		return 0
	}
	fields := factSearchFields(fact)
	fieldTokens := make([]struct {
		tokens []string
		weight float64
	}, 0, len(fields))
	for _, field := range fields {
		if strings.TrimSpace(field.value) == "" {
			continue
		}
		fieldTokens = append(fieldTokens, struct {
			tokens []string
			weight float64
		}{tokens: uniqueSearchTokens(tokenizeSearchText(field.value), false), weight: field.weight})
	}

	matched := 0
	var score float64
	for _, queryToken := range queryTokens {
		bestWeight := 0.0
		for _, field := range fieldTokens {
			for _, fieldToken := range field.tokens {
				if fieldToken == queryToken && field.weight > bestWeight {
					bestWeight = field.weight
				}
			}
		}
		if bestWeight > 0 {
			matched++
			score += bestWeight
		}
	}
	if matched == 0 {
		return 0
	}

	score /= 10 * float64(len(queryTokens))
	score += searchScoreBonuses(fact, query, queryTokens)
	coverage := float64(matched) / float64(len(queryTokens))
	score *= coverage * coverage
	if score > 1 {
		return 1
	}
	return score
}

func searchScoreBonuses(fact *factsv1.GenericFact, query string, queryTokens []string) float64 {
	pathTokens := uniqueSearchTokens(tokenizeSearchText(fact.GetAttributes()["path"]), false)
	pathSet := make(map[string]struct{}, len(pathTokens))
	for _, token := range pathTokens {
		pathSet[token] = struct{}{}
	}
	pathMatches := 0
	subjectSet := make(map[string]struct{})
	for _, token := range uniqueSearchTokens(tokenizeSearchText(fact.GetSubject()), false) {
		subjectSet[token] = struct{}{}
	}
	pathOnlyMatches := 0
	for _, token := range queryTokens {
		if _, ok := pathSet[token]; ok {
			pathMatches++
			if _, inSubject := subjectSet[token]; !inSubject {
				pathOnlyMatches++
			}
		}
	}
	bonus := 0.2 * float64(pathMatches) / float64(len(queryTokens))
	bonus += 0.25 * float64(pathOnlyMatches) / float64(len(queryTokens))
	bonus += 0.3 * float64(searchLongestPhrase(fact, queryTokens)) / float64(len(queryTokens))
	if searchHasToken(queryTokens, "func") && strings.HasPrefix(strings.TrimSpace(strings.ToLower(fact.GetSubject())), "func ") {
		bonus += 0.5
	}
	if searchDeclarationIntent(queryTokens) && isDeclarationLine(fact.GetSubject()) {
		bonus += 0.15
	}
	if searchFileIntent(queryTokens) && fact.GetAttributes()["line"] == "1" && pathMatches > 0 {
		bonus += 0.1
	}
	phrase := strings.ToLower(strings.TrimSpace(query))
	if searchIdentifierShaped(phrase) {
		idAndPath := strings.ToLower(fact.GetId() + " " + fact.GetAttributes()["path"])
		if strings.Contains(idAndPath, phrase) {
			bonus += 0.25
		}
	}
	return bonus
}

// searchScoreBonusesIndexed mirrors the high-value parts of the complete
// scorer without re-tokenizing every candidate returned by the project index.
// Path and declaration bonuses are deliberately cheap because broad queries
// can produce thousands of indexed candidates.
func searchScoreBonusesIndexed(fact *factsv1.GenericFact, query string, queryTokens []string) float64 {
	path := strings.ToLower(fact.GetAttributes()["path"])
	subject := strings.ToLower(fact.GetSubject())
	pathMatches := 0
	pathOnlyMatches := 0
	for _, token := range queryTokens {
		if !strings.Contains(path, token) {
			continue
		}
		pathMatches++
		if !strings.Contains(subject, token) {
			pathOnlyMatches++
		}
	}
	bonus := 0.2 * float64(pathMatches) / float64(len(queryTokens))
	bonus += 0.25 * float64(pathOnlyMatches) / float64(len(queryTokens))
	bonus += searchCompoundIdentifierBonus(fact, query)
	if searchHasToken(queryTokens, "func") && strings.HasPrefix(strings.TrimSpace(subject), "func ") {
		bonus += 0.5
	}
	if searchDeclarationIntent(queryTokens) && isDeclarationLine(fact.GetSubject()) {
		bonus += 0.15
	}
	if searchFileIntent(queryTokens) && fact.GetAttributes()["line"] == "1" && pathMatches > 0 {
		bonus += 0.1
	}
	phrase := strings.ToLower(strings.TrimSpace(query))
	if searchIdentifierShaped(phrase) && strings.Contains(strings.ToLower(fact.GetId()+" "+path), phrase) {
		bonus += 0.25
	}
	return bonus
}

func searchCompoundIdentifierBonus(fact *factsv1.GenericFact, query string) float64 {
	for _, identifier := range searchCompoundIdentifiers(query) {
		if searchFactContainsCompoundIdentifier(fact, identifier) {
			return 0.35
		}
	}
	return 0
}

func searchCompoundIdentifiers(query string) []string {
	var identifiers []string
	for _, token := range strings.FieldsFunc(query, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}) {
		if len(token) >= 2 && token != strings.ToLower(token) {
			identifiers = append(identifiers, token)
		}
	}
	return identifiers
}

func searchFactContainsCompoundIdentifier(fact *factsv1.GenericFact, identifier string) bool {
	for _, field := range factSearchFields(fact) {
		if strings.Contains(field.value, identifier) {
			return true
		}
	}
	return false
}

func searchHasToken(tokens []string, wanted string) bool {
	for _, token := range tokens {
		if token == wanted {
			return true
		}
	}
	return false
}

func searchLongestPhrase(fact *factsv1.GenericFact, queryTokens []string) int {
	longest := 0
	for _, field := range factSearchFields(fact) {
		fieldTokens := uniqueSearchTokens(tokenizeSearchText(field.value), false)
		for start := 0; start < len(queryTokens); start++ {
			for fieldStart := 0; fieldStart < len(fieldTokens); fieldStart++ {
				length := 0
				for start+length < len(queryTokens) && fieldStart+length < len(fieldTokens) && queryTokens[start+length] == fieldTokens[fieldStart+length] {
					length++
				}
				if length > longest {
					longest = length
				}
			}
		}
	}
	return longest
}

func searchDeclarationIntent(tokens []string) bool {
	for _, token := range tokens {
		if token == "type" || token == "function" || token == "declar" || token == "define" || token == "rpc" || token == "message" {
			return true
		}
	}
	return false
}

func searchFileIntent(tokens []string) bool {
	for _, token := range tokens {
		if token == "file" || token == "proto" || token == "schema" || token == "package" || token == "located" || token == "location" {
			return true
		}
	}
	return false
}

func isDeclarationLine(subject string) bool {
	subject = strings.TrimSpace(strings.ToLower(subject))
	return strings.HasPrefix(subject, "type ") || strings.HasPrefix(subject, "func ") || strings.HasPrefix(subject, "message ") || strings.HasPrefix(subject, "service ") || strings.HasPrefix(subject, "rpc ")
}

func searchIdentifierShaped(query string) bool {
	return strings.ContainsAny(query, "/\\._:-")
}
