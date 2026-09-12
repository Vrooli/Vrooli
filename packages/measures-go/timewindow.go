package measures

import (
	"fmt"
	"strings"
	"time"

	measuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/measures/v1"
)

// TimeWindowToken is the string form of the canonical relative-range tokens
// (the lowercased manifest/param-value spelling of measuresv1.TimeWindowToken).
// These are what a measure's `default` and a resolved param value carry.
type TimeWindowToken string

const (
	TokenThisWeek    TimeWindowToken = "this_week"
	TokenLast7d      TimeWindowToken = "last_7d"
	TokenLast30d     TimeWindowToken = "last_30d"
	TokenThisMonth   TimeWindowToken = "this_month"
	TokenLastMonth   TimeWindowToken = "last_month"
	TokenThisQuarter TimeWindowToken = "this_quarter"
)

// Valid reports whether t is one of the six canonical tokens.
func (t TimeWindowToken) Valid() bool {
	switch t {
	case TokenThisWeek, TokenLast7d, TokenLast30d, TokenThisMonth, TokenLastMonth, TokenThisQuarter:
		return true
	default:
		return false
	}
}

// Proto returns the measuresv1 enum value for this token.
func (t TimeWindowToken) Proto() measuresv1.TimeWindowToken {
	switch t {
	case TokenThisWeek:
		return measuresv1.TimeWindowToken_TIME_WINDOW_TOKEN_THIS_WEEK
	case TokenLast7d:
		return measuresv1.TimeWindowToken_TIME_WINDOW_TOKEN_LAST_7D
	case TokenLast30d:
		return measuresv1.TimeWindowToken_TIME_WINDOW_TOKEN_LAST_30D
	case TokenThisMonth:
		return measuresv1.TimeWindowToken_TIME_WINDOW_TOKEN_THIS_MONTH
	case TokenLastMonth:
		return measuresv1.TimeWindowToken_TIME_WINDOW_TOKEN_LAST_MONTH
	case TokenThisQuarter:
		return measuresv1.TimeWindowToken_TIME_WINDOW_TOKEN_THIS_QUARTER
	default:
		return measuresv1.TimeWindowToken_TIME_WINDOW_TOKEN_UNSPECIFIED
	}
}

// TokenFromProto converts a measuresv1 enum value to its string token. The zero
// (UNSPECIFIED) value yields ("", false).
func TokenFromProto(t measuresv1.TimeWindowToken) (TimeWindowToken, bool) {
	switch t {
	case measuresv1.TimeWindowToken_TIME_WINDOW_TOKEN_THIS_WEEK:
		return TokenThisWeek, true
	case measuresv1.TimeWindowToken_TIME_WINDOW_TOKEN_LAST_7D:
		return TokenLast7d, true
	case measuresv1.TimeWindowToken_TIME_WINDOW_TOKEN_LAST_30D:
		return TokenLast30d, true
	case measuresv1.TimeWindowToken_TIME_WINDOW_TOKEN_THIS_MONTH:
		return TokenThisMonth, true
	case measuresv1.TimeWindowToken_TIME_WINDOW_TOKEN_LAST_MONTH:
		return TokenLastMonth, true
	case measuresv1.TimeWindowToken_TIME_WINDOW_TOKEN_THIS_QUARTER:
		return TokenThisQuarter, true
	default:
		return "", false
	}
}

// Range is a resolved [From, To) time range: From inclusive, To exclusive.
type Range struct {
	From time.Time
	To   time.Time
}

// ResolveToken deterministically maps a canonical token to a concrete [from, to)
// range anchored to `now` in `loc`. No LLM, no wall-clock read — `now` and `loc`
// are explicit inputs so the resolution is fully reproducible and testable.
//
// Semantics (from measures.proto):
//   - this_week    : Monday 00:00 (in loc) of the current week .. now
//   - last_7d      : rolling now-7d .. now
//   - last_30d     : rolling now-30d .. now
//   - this_month   : 1st 00:00 of the current month .. now
//   - last_month   : 1st 00:00 of the previous month .. 1st 00:00 of this month
//   - this_quarter : first day 00:00 of the current quarter .. now
//
// From is inclusive, To is exclusive.
func ResolveToken(token TimeWindowToken, now time.Time, loc *time.Location) (Range, error) {
	if loc == nil {
		loc = time.UTC
	}
	n := now.In(loc)
	switch token {
	case TokenThisWeek:
		return Range{From: startOfWeek(n, loc), To: n}, nil
	case TokenLast7d:
		return Range{From: n.AddDate(0, 0, -7), To: n}, nil
	case TokenLast30d:
		return Range{From: n.AddDate(0, 0, -30), To: n}, nil
	case TokenThisMonth:
		return Range{From: startOfMonth(n, loc), To: n}, nil
	case TokenLastMonth:
		thisMonth := startOfMonth(n, loc)
		return Range{From: thisMonth.AddDate(0, -1, 0), To: thisMonth}, nil
	case TokenThisQuarter:
		return Range{From: startOfQuarter(n, loc), To: n}, nil
	default:
		return Range{}, fmt.Errorf("measures: unknown time-window token %q", token)
	}
}

// ResolveTimeWindow resolves a proto TimeWindow (a relative token OR an explicit
// custom range) into a concrete [from, to) range. `now`/`loc` anchor relative
// tokens; a custom range is returned verbatim (from/to are taken as authored).
func ResolveTimeWindow(tw *measuresv1.TimeWindow, now time.Time, loc *time.Location) (Range, error) {
	if tw == nil {
		return Range{}, fmt.Errorf("measures: nil TimeWindow")
	}
	switch w := tw.GetWindow().(type) {
	case *measuresv1.TimeWindow_Token:
		token, ok := TokenFromProto(w.Token)
		if !ok {
			return Range{}, fmt.Errorf("measures: unspecified TimeWindow token")
		}
		return ResolveToken(token, now, loc)
	case *measuresv1.TimeWindow_Custom:
		c := w.Custom
		if c == nil || c.GetFrom() == nil || c.GetTo() == nil {
			return Range{}, fmt.Errorf("measures: custom range missing from/to")
		}
		from := c.GetFrom().AsTime()
		to := c.GetTo().AsTime()
		if !to.After(from) {
			return Range{}, fmt.Errorf("measures: custom range to (%s) must be after from (%s)", to, from)
		}
		return Range{From: from, To: to}, nil
	default:
		return Range{}, fmt.Errorf("measures: empty TimeWindow oneof")
	}
}

// startOfWeek returns Monday 00:00 (in loc) of the week containing n.
func startOfWeek(n time.Time, loc *time.Location) time.Time {
	// Go: Sunday=0 .. Saturday=6. Days since Monday: (weekday+6)%7.
	daysSinceMonday := (int(n.Weekday()) + 6) % 7
	d := n.AddDate(0, 0, -daysSinceMonday)
	return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, loc)
}

// startOfMonth returns the 1st 00:00 (in loc) of n's month.
func startOfMonth(n time.Time, loc *time.Location) time.Time {
	return time.Date(n.Year(), n.Month(), 1, 0, 0, 0, 0, loc)
}

// startOfQuarter returns the first day 00:00 (in loc) of n's calendar quarter.
func startOfQuarter(n time.Time, loc *time.Location) time.Time {
	// Months 1-3 -> Jan, 4-6 -> Apr, 7-9 -> Jul, 10-12 -> Oct.
	firstMonth := ((int(n.Month())-1)/3)*3 + 1
	return time.Date(n.Year(), time.Month(firstMonth), 1, 0, 0, 0, 0, loc)
}

// timeWindowPhrases maps natural-language phrasings to canonical tokens. Order
// matters: more specific phrases must precede the substrings they contain
// (e.g. "this week" before "week") so MatchTimeWindowToken takes the longest
// deterministic match.
var timeWindowPhrases = []struct {
	phrase string
	token  TimeWindowToken
}{
	{"last 7 days", TokenLast7d},
	{"past 7 days", TokenLast7d},
	{"last seven days", TokenLast7d},
	{"past seven days", TokenLast7d},
	{"last 30 days", TokenLast30d},
	{"past 30 days", TokenLast30d},
	{"last thirty days", TokenLast30d},
	{"this quarter", TokenThisQuarter},
	{"current quarter", TokenThisQuarter},
	{"this week", TokenThisWeek},
	{"current week", TokenThisWeek},
	{"this month", TokenThisMonth},
	{"current month", TokenThisMonth},
	{"last month", TokenLastMonth},
	{"previous month", TokenLastMonth},
}

// MatchTimeWindowToken deterministically extracts a canonical time-window token
// from a natural-language question, with no LLM. It returns ("", false) when no
// phrase matches, leaving the caller to fall back to a default or to needs[].
func MatchTimeWindowToken(question string) (TimeWindowToken, bool) {
	q := strings.ToLower(question)
	for _, p := range timeWindowPhrases {
		if strings.Contains(q, p.phrase) {
			return p.token, true
		}
	}
	return "", false
}
