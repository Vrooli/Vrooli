package operatordecision

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

// The cover rules: how a decision reads to an operator who has none of the
// requester's context. Notification Hub's packet check gives them as advice on
// every ask, and each producer's tests assert CoverProblems is empty for the
// Facts it builds, so both read one rule set. The rules are documented in
// docs/concepts/DECISION_PACKETS.md ("Packet check").
//
// The rules prefer silence to a wrong accusation. Each pattern is deliberately
// narrow, and the tests pin the ordinary text (years, times, money, versions,
// chip and quarter names, URLs) they must leave alone.

// Codes of the cover findings; the Hub reports them as its warning codes.
const (
	CoverInternalLabel  = "internal_id_in_question"
	CoverPath           = "path_in_question"
	CoverLongQuestion   = "long_question"
	CoverNoBigPicture   = "missing_big_picture"
	CoverNoReason       = "missing_recommendation_reason"
	MaxCoverQuestionLen = 200 // runes past which a question stops reading as one sentence
)

// CoverFinding is one way a cover fails a cold reader. Field names the ask
// field ("question", "options[1].label").
type CoverFinding struct{ Code, Field, Message string }

// CheckCover applies the cover rules to what the operator reads cold (the push
// shows the question; the decision shows the big picture, the recommendation
// reason and the option labels): internal labels in any of them, a leading tag
// on the question, file paths in the question or big picture, a question longer
// than one sentence, and a missing big picture or recommendation reason.
func CheckCover(f Facts) []CoverFinding {
	texts := []struct{ field, name, text string }{
		{"question", "question", f.Question},
		{"big_picture", "big picture", f.BigPicture},
		{"recommendation_reason", "recommendation reason", f.RecommendationReason},
	}
	for i, option := range f.Options {
		texts = append(texts, struct{ field, name, text string }{fmt.Sprintf("options[%d].label", i), "label of option " + option.Key, option.Label})
	}
	var found []CoverFinding
	add := func(code, field, message string) { found = append(found, CoverFinding{code, field, message}) }
	for _, t := range texts {
		if ids := InternalLabels(t.text); len(ids) > 0 {
			add(CoverInternalLabel, t.field, labelMessage(t.name, ids))
		} else if tag := LeadingTag(t.text); t.field == "question" && tag != "" {
			add(CoverInternalLabel, t.field, fmt.Sprintf("the question starts with a tag (%s); drop it and say what the decision is about in plain words", tag))
		}
	}
	for _, t := range texts[:2] {
		if paths := findPaths(t.text, true); len(paths) > 0 {
			add(CoverPath, t.field, pathMessage(t.name, paths))
		}
	}
	if n := utf8.RuneCountInString(f.Question); n > MaxCoverQuestionLen {
		add(CoverLongQuestion, "question", fmt.Sprintf("the question is %d characters; keep it to one sentence under %d and move the rest to big_picture", n, MaxCoverQuestionLen))
	}
	if strings.TrimSpace(f.BigPicture) == "" {
		add(CoverNoBigPicture, "big_picture", "there is no big picture; add the situation and what each answer causes, so the operator can decide without your context")
	}
	if f.Recommended != "" && strings.TrimSpace(f.RecommendationReason) == "" {
		add(CoverNoReason, "recommendation_reason", "the recommendation has no reason; add one plain sentence about why it is the best choice")
	}
	return found
}

// CoverProblems is what a producer's tests assert is empty for the Facts it
// builds: the cover rules, plus internal labels and paths in the subject, the
// push title. Internal names belong in TechnicalDetails.
func CoverProblems(f Facts) []string {
	var problems []string
	if ids := InternalLabels(f.Subject); len(ids) > 0 {
		problems = append(problems, labelMessage("subject", ids))
	}
	if paths := findPaths(f.Subject, true); len(paths) > 0 {
		problems = append(problems, pathMessage("subject", paths))
	}
	for _, finding := range CheckCover(f) {
		problems = append(problems, finding.Message)
	}
	return problems
}

// PayloadCoverProblems is CoverProblems for a decision event's payload
// (Facts.Payload), for producer tests that see only the published event.
func PayloadCoverProblems(payload map[string]any) []string {
	text := func(m map[string]any, key string) string { s, _ := m[key].(string); return s }
	f := Facts{
		Subject: text(payload, "subject"), Question: text(payload, "question"), BigPicture: text(payload, "big_picture"),
		Recommended: text(payload, "recommended"), RecommendationReason: text(payload, "recommendation_reason"),
	}
	options, _ := payload["options"].([]any)
	for _, option := range options {
		m, _ := option.(map[string]any)
		f.Options = append(f.Options, Option{Key: text(m, "key"), Label: text(m, "label")})
	}
	return CoverProblems(f)
}

func labelMessage(name string, ids []string) string {
	return fmt.Sprintf("the %s uses internal labels the operator cannot know (%s); say what each one is in plain words, and keep the labels in technical_details", name, QuoteList(ids))
}

func pathMessage(name string, paths []string) string {
	return fmt.Sprintf("the %s names a file path (%s); attach the file and name it in words", name, QuoteList(paths))
}

// QuoteList quotes at most three values for a message, then says how many more.
func QuoteList(values []string) string {
	shown := values[:min(len(values), 3)]
	quoted := make([]string, 0, len(shown)+1)
	for _, value := range shown {
		quoted = append(quoted, fmt.Sprintf("%q", value))
	}
	if extra := len(values) - len(shown); extra > 0 {
		quoted = append(quoted, fmt.Sprintf("and %d more", extra))
	}
	return strings.Join(quoted, ", ")
}

// --- internal labels -------------------------------------------------------

// labelStart is what may precede a label: the start, or a character that
// cannot be part of a longer word or id. codeStart also excludes '.' and '/',
// so a dated slug or hash inside a host or path is left to the path rule.
const (
	labelStart = `(?:^|[^A-Za-z0-9_-])`
	codeStart  = `(?:^|[^A-Za-z0-9_./-])`
	digits     = "0123456789"
)

var (
	// A capital letter with one or two digits as a whole word (S7, O7, F49), or
	// with a hyphen and up to five digits (S-14, D-065).
	shortLabelPattern = regexp.MustCompile(labelStart + `([A-Z](?:\d{1,2}|-\d{1,5}))\b`)
	// Two to four capitals, a hyphen, digits: BAS-065. Standards such as
	// SHA-256 and UTF-8 are allowed prefixes below.
	prefixedLabelPattern = regexp.MustCompile(labelStart + `([A-Z]{2,4}-\d{1,5})\b`)
	// Three or more capital/digit segments: F49-MOR-CHECKOUT, BAS-D-065. One
	// segment must hold a digit, so SOME-CAPS-WORDS is left alone.
	segmentedLabelPattern = regexp.MustCompile(labelStart + `((?:[A-Z][A-Z0-9]*-){2,}[A-Z0-9]+)\b`)
	// Slugs ending in a date stamp: research-run-20261114, um-20261109-57fc87.
	datedLabelPattern = regexp.MustCompile(codeStart + `([a-z][a-z0-9]*(?:-[a-z0-9]+)*-\d{8}(?:-[a-z0-9]+)*)\b`)
	// Hashes: twelve or more lowercase hex digits holding a digit and a letter.
	hashPattern       = regexp.MustCompile(codeStart + `([0-9a-f]{12,})\b`)
	uuidPattern       = regexp.MustCompile(`(?i)\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b`)
	bracketTagPattern = regexp.MustCompile(`^\s*(\[[^\]\n]{1,40}\])`)
)

// wellKnownShortLabels are one-letter-and-digits words ordinary text uses:
// quarters, halves, Apple chips, percentile latencies, and a few products.
var wellKnownShortLabels = map[string]bool{
	"Q1": true, "Q2": true, "Q3": true, "Q4": true, "H1": true, "H2": true,
	"M1": true, "M2": true, "M3": true, "M4": true, "A4": true, "S3": true, "R2": true,
	"D1": true, "B2": true, "V8": true, "G7": true, "G20": true,
	"P50": true, "P75": true, "P90": true, "P95": true, "P99": true,
}

// wellKnownLabelPrefixes are capital prefixes of hyphenated standards and
// products that are not internal labels (UTF-8, SHA-256, GPT-4, CVE-2026).
var wellKnownLabelPrefixes = map[string]bool{
	"UTF": true, "SHA": true, "AES": true, "GPT": true, "ISO": true, "RFC": true, "CVE": true,
	"RSA": true, "ECC": true, "DES": true, "MD": true, "USB": true, "HTTP": true, "WPA": true, "TLS": true,
}

// continuesAsVersion reports whether the text after a label continues as a
// version number (V2.1, S-14.5), which is not an internal label.
func continuesAsVersion(rest string) bool {
	return len(rest) >= 2 && rest[0] == '.' && rest[1] >= '0' && rest[1] <= '9'
}

// continuesAsName reports whether the text after a dated slug or hash shows it
// is part of a host, path or email address.
func continuesAsName(rest string) bool {
	return rest != "" && rest[0] == '@' || len(rest) >= 2 && (rest[0] == '.' || rest[0] == '/') && isAlnum(rest[1])
}

func isAlnum(b byte) bool {
	return b >= '0' && b <= '9' || b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z'
}

func labelPrefix(label string) string {
	prefix, _, _ := strings.Cut(label, "-")
	return prefix
}

// InternalLabels returns the internal-looking labels in text, in order of
// appearance and without repeats: S-14, O7, F49-MOR-CHECKOUT, run-20261114,
// hashes, UUIDs. Snake-case words are left alone: folder names such as
// node_modules are ordinary to the operator. A label inside a longer one (F49 in
// F49-MOR-CHECKOUT) is reported once, as the longer.
func InternalLabels(text string) []string {
	type span struct {
		start, end int
		label      string
	}
	var spans []span
	collect := func(pattern *regexp.Regexp, keep func(label, rest string) bool) {
		for _, match := range pattern.FindAllStringSubmatchIndex(text, -1) {
			if label := text[match[2]:match[3]]; keep(label, text[match[3]:]) {
				spans = append(spans, span{start: match[2], end: match[3], label: label})
			}
		}
	}
	code := func(_, rest string) bool { return !continuesAsName(rest) }
	collect(shortLabelPattern, func(label, rest string) bool {
		return !wellKnownShortLabels[label] && !continuesAsVersion(rest)
	})
	collect(prefixedLabelPattern, func(label, rest string) bool {
		return !wellKnownLabelPrefixes[labelPrefix(label)] && !continuesAsVersion(rest)
	})
	collect(segmentedLabelPattern, func(label, rest string) bool {
		return strings.ContainsAny(label, digits) && !wellKnownLabelPrefixes[labelPrefix(label)] && !continuesAsVersion(rest)
	})
	collect(datedLabelPattern, code)
	collect(hashPattern, func(label, rest string) bool {
		return strings.ContainsAny(label, digits) && strings.ContainsAny(label, "abcdef") && code(label, rest)
	})
	for _, match := range uuidPattern.FindAllStringIndex(text, -1) {
		spans = append(spans, span{start: match[0], end: match[1], label: text[match[0]:match[1]]})
	}
	// Longest first at a given start, then by position: drop spans inside one
	// already kept.
	sort.SliceStable(spans, func(i, j int) bool {
		if spans[i].start != spans[j].start {
			return spans[i].start < spans[j].start
		}
		return spans[i].end > spans[j].end
	})
	var found []string
	seen := map[string]bool{}
	lastEnd := -1
	for _, item := range spans {
		if item.start < lastEnd {
			continue
		}
		lastEnd = item.end
		if !seen[item.label] {
			seen[item.label] = true
			found = append(found, item.label)
		}
	}
	return found
}

// LeadingTag returns a bracketed tag that starts text ("[BAS]"), or "".
func LeadingTag(text string) string {
	if match := bracketTagPattern.FindStringSubmatch(text); match != nil {
		return match[1]
	}
	return ""
}

// --- file paths ------------------------------------------------------------

var (
	// A path starts a word: after whitespace, an opening bracket, or a quote.
	// URLs ("https://host/a/b") never match because their slash follows ":".
	pathStart = `(?:^|[\s("'` + "`" + `\[])`
	pathEnd   = `(?:$|[\s"'` + "`" + `)\]>,;:.])`
	pathRest  = `[^\s"'` + "`" + `)\]>,;]+`
	// A file name with an extension, the end of a path that names a file.
	fileName = `[\w@+-]+\.[A-Za-z][A-Za-z0-9]{0,7}`
	// Absolute paths under the roots a host's files live in.
	hostRootPathPattern = regexp.MustCompile(pathStart + `(/(?:home|tmp|var|etc|usr|opt|mnt|root|srv|run|media|dev|proc|Users|Volumes|private)/` + pathRest + `)`)
	// Home-relative and explicitly relative paths: ~/notes.md, ./out/report.txt.
	relativePathPattern = regexp.MustCompile(pathStart + `((?:~|\.{1,2})/` + pathRest + `)`)
	// Windows drive paths: C:\Users\me\file.txt.
	windowsPathPattern = regexp.MustCompile(pathStart + `([A-Za-z]:\\` + pathRest + `)`)
	// Any other absolute path with at least two segments and a file extension:
	// /project/docs/plan.md. API routes such as /api/v1/asks have no extension
	// and stay unflagged.
	extensionPathPattern = regexp.MustCompile(pathStart + `(/(?:[\w.@+-]+/)+` + fileName + `)` + pathEnd)
	// A repository-relative file: docs/plan.md, api/main.go. Directory names
	// hold no dot, so a host with a path (example.com/plan.md) is left alone.
	repoPathPattern = regexp.MustCompile(pathStart + `((?:\.?[\w@+-]+/)+` + fileName + `)` + pathEnd)
)

// LocalPaths returns the files on this host that text names, in order,
// without repeats and without trailing sentence punctuation.
func LocalPaths(text string) []string { return findPaths(text, false) }

// findPaths is LocalPaths, and with repo also repository-relative files: in
// the cover those are paths too, but they are not files the Hub could attach.
func findPaths(text string, repo bool) []string {
	patterns := []*regexp.Regexp{hostRootPathPattern, relativePathPattern, windowsPathPattern, extensionPathPattern}
	if repo {
		patterns = append(patterns, repoPathPattern)
	}
	var found []string
	seen := map[string]bool{}
	for _, pattern := range patterns {
		for _, match := range pattern.FindAllStringSubmatch(text, -1) {
			candidate := strings.TrimRight(match[1], ".:")
			if candidate != "" && !seen[candidate] {
				seen[candidate] = true
				found = append(found, candidate)
			}
		}
	}
	sort.SliceStable(found, func(i, j int) bool { return strings.Index(text, found[i]) < strings.Index(text, found[j]) })
	return found
}
