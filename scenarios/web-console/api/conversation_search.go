package main

import (
	"context"
	"fmt"
	"regexp"
	"regexp/syntax"
	"slices"
	"strings"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"
)

// DOC: docs/internal/MESSAGES-VIEW-PROJECTION-UX.md#one-search

// ConversationSearchMode chooses how a query is read. Every mode searches the
// whole history through the full-text mirror, then matches exactly in Go.
type ConversationSearchMode string

const (
	ConversationSearchText  ConversationSearchMode = "text"
	ConversationSearchRegex ConversationSearchMode = "regex"
	ConversationSearchFuzzy ConversationSearchMode = "fuzzy"
)

const (
	defaultConversationSearchLimit = 500
	maxSearchPatternBytes          = 512
	searchExcerptLead              = 60
	searchExcerptBytes             = 160
	fuzzyNearDistance              = 10
)

// searchScanCap bounds a scan the index cannot narrow — a regex with no usable
// literal, or a query that is only punctuation. The newest rows are scanned and
// the result is marked truncated. A variable so tests can shrink it.
var searchScanCap = 20_000

// ConversationSearchQuery is one search, in any scope.
type ConversationSearchQuery struct {
	Query         string
	Mode          ConversationSearchMode // "" reads as text
	CaseSensitive bool
	WholeWord     bool
	Role          string // "user" | "assistant" | "" for both
	Limit         int
}

// TextRange is a match inside an excerpt, in UTF-16 code units (what browser
// strings index by), end exclusive.
type TextRange struct {
	Start int
	End   int
}

// ConversationSearchResult is a session search's answer. Error carries a
// query that cannot run as asked (an invalid regex); it is not a failure.
type ConversationSearchResult struct {
	Matches   []ConversationSearchMatch
	Truncated bool
	Total     int64
	Error     string
}

// ArchivedConversationSearchResult is an archive search's answer.
type ArchivedConversationSearchResult struct {
	Matches          []ArchivedConversationSearchMatch
	Truncated        bool
	Total            int64
	DistinctSessions int64
	Error            string
}

// searchMatcher finds a query's byte ranges in a text; nil ranges mean no match.
type searchMatcher func(text string) [][2]int

// searchPlan is how a query runs: the FTS query that narrows candidates (empty
// when the index cannot narrow it) and the exact matcher applied to them.
type searchPlan struct {
	fts   string
	match searchMatcher
}

func planSearch(q ConversationSearchQuery) (searchPlan, string) {
	query := strings.TrimSpace(q.Query)
	if query == "" {
		return searchPlan{match: func(string) [][2]int { return nil }}, ""
	}
	switch q.Mode {
	case ConversationSearchRegex:
		if len(query) > maxSearchPatternBytes {
			return searchPlan{}, fmt.Sprintf("regular expression is longer than %d bytes", maxSearchPatternBytes)
		}
		flags := "(?m)"
		if !q.CaseSensitive {
			flags = "(?im)"
		}
		re, err := regexp.Compile(flags + query)
		if err != nil {
			// RE2 quotes the pattern it compiled; show the user's, not our flags.
			reason := strings.TrimPrefix(err.Error(), "error parsing regexp: ")
			return searchPlan{}, "invalid regular expression: " + strings.Replace(reason, "`"+flags, "`", 1)
		}
		return searchPlan{fts: prefixQuery(regexAnchoredTokens(query)), match: wordFiltered(regexMatcher(re), q.WholeWord)}, ""
	case ConversationSearchFuzzy:
		tokens := searchTokens(query)
		if len(tokens) == 0 {
			return searchPlan{match: func(string) [][2]int { return nil }}, ""
		}
		return searchPlan{fts: fuzzyQuery(tokens), match: fuzzyMatcher(tokens)}, ""
	default:
		flags := ""
		if !q.CaseSensitive {
			flags = "(?i)"
		}
		re := regexp.MustCompile(flags + regexp.QuoteMeta(query))
		// Words narrow the index as prefixes; the literal (punctuation and
		// all) decides. A query with no words scans a bounded window.
		return searchPlan{fts: prefixQuery(searchTokens(query)), match: wordFiltered(regexMatcher(re), q.WholeWord)}, ""
	}
}

func regexMatcher(re *regexp.Regexp) searchMatcher {
	return func(text string) [][2]int {
		found := re.FindAllStringIndex(text, -1)
		out := make([][2]int, 0, len(found))
		for _, r := range found {
			if r[1] > r[0] {
				out = append(out, [2]int{r[0], r[1]})
			}
		}
		return out
	}
}

// wordFiltered keeps only matches that start and end at word boundaries.
func wordFiltered(match searchMatcher, wholeWord bool) searchMatcher {
	if !wholeWord {
		return match
	}
	return func(text string) [][2]int {
		kept := [][2]int{}
		for _, r := range match(text) {
			if isWordStart(text, r[0]) && isWordEnd(text, r[1]) {
				kept = append(kept, r)
			}
		}
		return kept
	}
}

// fuzzyMatcher requires every token to begin a word, in any case.
func fuzzyMatcher(tokens []string) searchMatcher {
	patterns := make([]*regexp.Regexp, 0, len(tokens))
	for _, token := range tokens {
		patterns = append(patterns, regexp.MustCompile("(?i)"+regexp.QuoteMeta(token)))
	}
	return func(text string) [][2]int {
		var ranges [][2]int
		for _, re := range patterns {
			found := false
			for _, r := range re.FindAllStringIndex(text, -1) {
				if isWordStart(text, r[0]) {
					ranges = append(ranges, [2]int{r[0], r[1]})
					found = true
				}
			}
			if !found {
				return nil
			}
		}
		slices.SortFunc(ranges, func(a, b [2]int) int { return a[0] - b[0] })
		return ranges
	}
}

func isWordRune(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' }

func isWordStart(text string, at int) bool {
	if at == 0 {
		return true
	}
	r, _ := utf8.DecodeLastRuneInString(text[:at])
	return !isWordRune(r)
}

func isWordEnd(text string, at int) bool {
	if at >= len(text) {
		return true
	}
	r, _ := utf8.DecodeRuneInString(text[at:])
	return !isWordRune(r)
}

// searchTokens splits text the way the FTS unicode61 tokenizer does.
func searchTokens(text string) []string {
	return strings.FieldsFunc(text, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
}

// prefixQuery ANDs each token as a prefix term: "settle"* "loop"*.
func prefixQuery(tokens []string) string {
	terms := make([]string, 0, len(tokens))
	for _, token := range tokens {
		terms = append(terms, `"`+token+`"*`)
	}
	return strings.Join(terms, " ")
}

func fuzzyQuery(tokens []string) string {
	if len(tokens) == 2 {
		return fmt.Sprintf("NEAR(%s, %d)", prefixQuery(tokens), fuzzyNearDistance)
	}
	return prefixQuery(tokens)
}

// regexAnchoredTokens returns the words a regex's matches must begin with: the
// tokens of required literal runs that start at a word boundary (inside the
// literal, or after whitespace, a boundary, or a line start). A word that may
// continue an earlier word cannot narrow a prefix index, so it is skipped.
func regexAnchoredTokens(pattern string) []string {
	parsed, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		return nil
	}
	parsed = parsed.Simplify()
	for parsed.Op == syntax.OpCapture && len(parsed.Sub) == 1 {
		parsed = parsed.Sub[0]
	}
	subs := []*syntax.Regexp{parsed}
	if parsed.Op == syntax.OpConcat {
		subs = parsed.Sub
	}
	var tokens []string
	for index, sub := range subs {
		if sub.Op != syntax.OpLiteral {
			continue
		}
		leftBounded := index > 0 && boundsWord(subs[index-1])
		for _, token := range literalTokens(string(sub.Rune)) {
			if len(token.text) < 3 || (token.start == 0 && !leftBounded) {
				continue
			}
			tokens = append(tokens, token.text)
		}
	}
	return tokens
}

type literalToken struct {
	text  string
	start int
}

func literalTokens(literal string) []literalToken {
	var out []literalToken
	start := -1
	for index, r := range literal {
		word := unicode.IsLetter(r) || unicode.IsDigit(r)
		switch {
		case word && start < 0:
			start = index
		case !word && start >= 0:
			out = append(out, literalToken{text: literal[start:index], start: start})
			start = -1
		}
	}
	if start >= 0 {
		out = append(out, literalToken{text: literal[start:], start: start})
	}
	return out
}

// boundsWord reports whether a regex node always ends at a non-word position:
// whitespace or non-word classes, word boundaries, and line or text starts.
func boundsWord(node *syntax.Regexp) bool {
	switch node.Op {
	case syntax.OpWordBoundary, syntax.OpBeginLine, syntax.OpBeginText:
		return true
	case syntax.OpLiteral:
		if len(node.Rune) == 0 {
			return false
		}
		return !isWordRune(node.Rune[len(node.Rune)-1])
	case syntax.OpCharClass:
		for index := 0; index+1 < len(node.Rune); index += 2 {
			for r := node.Rune[index]; r <= node.Rune[index+1]; r++ {
				if isWordRune(r) {
					return false
				}
				if r-node.Rune[index] > 256 {
					break
				}
			}
		}
		return true
	case syntax.OpPlus, syntax.OpRepeat:
		return len(node.Sub) == 1 && node.Min > 0 && boundsWord(node.Sub[0])
	case syntax.OpCapture:
		return len(node.Sub) == 1 && boundsWord(node.Sub[0])
	default:
		return false
	}
}

// searchExcerpt cuts the text around the first match and returns the ranges
// that fall inside it, in UTF-16 code units relative to the excerpt.
func searchExcerpt(text string, ranges [][2]int) (string, []TextRange) {
	if len(ranges) == 0 {
		end := min(len(text), searchExcerptBytes)
		for end < len(text) && !utf8.RuneStart(text[end]) {
			end++
		}
		return text[:end], nil
	}
	from := max(0, ranges[0][0]-searchExcerptLead)
	for from > 0 && !utf8.RuneStart(text[from]) {
		from--
	}
	to := min(len(text), max(from+searchExcerptBytes, ranges[0][1]))
	for to < len(text) && !utf8.RuneStart(text[to]) {
		to++
	}
	excerpt := text[from:to]
	out := make([]TextRange, 0, len(ranges))
	for _, r := range ranges {
		if r[0] < from || r[1] > to {
			continue
		}
		start := utf16Len(text[from:r[0]])
		out = append(out, TextRange{Start: start, End: start + utf16Len(text[r[0]:r[1]])})
	}
	return excerpt, out
}

func utf16Len(s string) int { return len(utf16.Encode([]rune(s))) }

// searchScope is where a search looks: the SQL that selects its events.
type searchScope struct {
	joins     string
	where     []string
	args      []any
	ftsOrder  string
	scanOrder string
}

type searchHit struct {
	id        string
	sessionID string
	sequence  int64
	role      string
	createdAt string
	excerpt   string
	ranges    []TextRange
}

// searchEvents is the one search path: the FTS mirror narrows candidates when
// it can, a bounded scan of the newest rows stands in when it cannot, and the
// exact matcher decides. capped is true when rows beyond the cap went unread.
func (r *SQLConversationRepository) searchEvents(ctx context.Context, scope searchScope, q ConversationSearchQuery) (hits []searchHit, capped bool, queryError string, err error) {
	plan, queryError := planSearch(q)
	if queryError != "" {
		return nil, false, queryError, nil
	}
	if strings.TrimSpace(q.Query) == "" {
		return nil, false, "", nil
	}
	where := append([]string(nil), scope.where...)
	args := append([]any(nil), scope.args...)
	if q.Role != "" {
		where = append(where, "e.role = ?")
		args = append(args, q.Role)
	}
	var query string
	if plan.fts != "" {
		query = `SELECT e.id, e.session_id, e.sequence, e.role, e.created_at, e.text
			FROM conversation_events_fts JOIN conversation_events e ON e.rowid = conversation_events_fts.rowid ` + scope.joins + `
			WHERE conversation_events_fts MATCH ?` + andWhere(where) + `
			ORDER BY ` + scope.ftsOrder + ` LIMIT ?`
		args = append([]any{plan.fts}, args...)
	} else {
		query = `SELECT e.id, e.session_id, e.sequence, e.role, e.created_at, e.text
			FROM conversation_events e ` + scope.joins + `
			WHERE 1 = 1` + andWhere(where) + `
			ORDER BY ` + scope.scanOrder + ` LIMIT ?`
	}
	args = append(args, searchScanCap+1)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, false, "", fmt.Errorf("search conversation events: %w", err)
	}
	defer rows.Close()
	read := 0
	for rows.Next() {
		read++
		if read > searchScanCap {
			capped = true
			break
		}
		var hit searchHit
		var text string
		if err := rows.Scan(&hit.id, &hit.sessionID, &hit.sequence, &hit.role, &hit.createdAt, &text); err != nil {
			return nil, false, "", err
		}
		ranges := plan.match(text)
		if len(ranges) == 0 {
			continue
		}
		hit.excerpt, hit.ranges = searchExcerpt(text, ranges)
		hits = append(hits, hit)
	}
	if err := rows.Err(); err != nil {
		return nil, false, "", err
	}
	return hits, capped, "", nil
}

func andWhere(clauses []string) string {
	if len(clauses) == 0 {
		return ""
	}
	return " AND " + strings.Join(clauses, " AND ")
}

func searchLimit(limit int) int {
	if limit <= 0 {
		return defaultConversationSearchLimit
	}
	return limit
}
