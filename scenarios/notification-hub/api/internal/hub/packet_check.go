package hub

import (
	"fmt"
	"net"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

// The packet check: advice about how an ask reads to an operator who has none
// of the requester's context, and about explainer pages that will not work in
// the Hub's sealed frame. It is pure and never blocks an ask: every finding is
// a PacketWarning with severity "advice". The rules are documented in
// docs/concepts/DECISION_PACKETS.md ("Packet check", "HTML explainers"), and
// mirror the writing rules in skills/notification-hub/SKILL.md.
//
// The check prefers silence to a wrong accusation. Each pattern below is
// deliberately narrow, and the tests pin the ordinary text (years, times,
// money, versions, chip and quarter names) it must leave alone.

// Warning codes of the packet check.
const (
	WarnInternalIDInQuestion       = "internal_id_in_question"
	WarnPathInQuestion             = "path_in_question"
	WarnLongQuestion               = "long_question"
	WarnMissingBigPicture          = "missing_big_picture"
	WarnMissingRecommendReason     = "missing_recommendation_reason"
	WarnLocalContextURL            = "local_context_url"
	WarnUnattachedPath             = "unattached_path"
	WarnExplainerNotHTML           = "explainer_not_html"
	WarnExplainerTooLarge          = "explainer_too_large"
	WarnExplainerNetwork           = "explainer_network"
	WarnExplainerNoViewport        = "explainer_no_viewport"
	WarnExplainerForm              = "explainer_form"
	WarnExplainerUnknownAttachment = "explainer_unknown_attachment"
)

const (
	// longQuestionRunes is the length past which a question stops reading as
	// one sentence.
	longQuestionRunes = 200
	// MaxExplainerBytes is the largest explainer page the Hub shows.
	MaxExplainerBytes = 2 << 20
	// maxNamedInWarning bounds the names or addresses a message quotes.
	maxNamedInWarning = 3
)

// PacketAttachment is what the packet check needs to know about one attachment.
type PacketAttachment struct {
	Name string
	// SourcePath is the absolute path on the sender's host, "" when unknown.
	SourcePath string
}

// PacketExplainer is the explainer attachment and its page.
type PacketExplainer struct {
	// Field names the request field of the explainer, for example attachments[1].
	Field string
	Name  string
	// SourcePath and MediaType identify the page's format.
	SourcePath string
	MediaType  string
	// IsFolder is true when the explainer attachment is a folder.
	IsFolder bool
	// Size is the stored size in bytes.
	Size int64
	// HTML is the page, nil when the Hub could not read it. A page larger than
	// MaxExplainerBytes needs only its Size.
	HTML []byte
}

// CheckPacket returns the advice for a normalized ask: the cover text, the
// attachments and the explainer page (nil when the ask has none).
func CheckPacket(spec AskSpec, attachments []PacketAttachment, explainer *PacketExplainer) []PacketWarning {
	var warnings []PacketWarning
	warnings = append(warnings, checkCover(spec, attachments)...)
	if explainer != nil {
		attached := make(map[string]bool, len(attachments))
		for _, attachment := range attachments {
			attached[attachment.Name] = true
		}
		warnings = append(warnings, checkExplainer(*explainer, attached)...)
	}
	warnings = append(warnings, checkShape(spec, attachments, explainer != nil)...)
	return warnings
}

func advice(code, field, message string) PacketWarning {
	return PacketWarning{Code: code, Field: field, Message: message, Severity: WarningAdvice}
}

func checkCover(spec AskSpec, attachments []PacketAttachment) []PacketWarning {
	var warnings []PacketWarning
	if ids := internalIDs(spec.Question); len(ids) > 0 {
		warnings = append(warnings, advice(WarnInternalIDInQuestion, "question",
			fmt.Sprintf("the question uses internal labels the operator cannot know (%s); say what each one is in plain words", quoteList(ids))))
	} else if tag := bracketedTag(spec.Question); tag != "" {
		warnings = append(warnings, advice(WarnInternalIDInQuestion, "question",
			fmt.Sprintf("the question starts with a tag (%s); drop it and say what the decision is about in plain words", tag)))
	}
	for _, field := range []struct{ name, text string }{{"question", spec.Question}, {"big_picture", spec.BigPicture}} {
		if paths := localPaths(field.text); len(paths) > 0 {
			warnings = append(warnings, advice(WarnPathInQuestion, field.name,
				fmt.Sprintf("the %s names a file path (%s); attach the file and name it in words", strings.ReplaceAll(field.name, "_", " "), quoteList(paths))))
		}
	}
	if utf8.RuneCountInString(spec.Question) > longQuestionRunes {
		warnings = append(warnings, advice(WarnLongQuestion, "question",
			fmt.Sprintf("the question is %d characters; keep it to one sentence under %d and move the rest to big_picture", utf8.RuneCountInString(spec.Question), longQuestionRunes)))
	}
	if strings.TrimSpace(spec.BigPicture) == "" {
		warnings = append(warnings, advice(WarnMissingBigPicture, "big_picture",
			"there is no big picture; add the situation and what each answer causes, so the operator can decide without your context"))
	}
	if spec.Recommended != "" && strings.TrimSpace(spec.RecommendationReason) == "" {
		warnings = append(warnings, advice(WarnMissingRecommendReason, "recommendation_reason",
			"the recommendation has no reason; add one plain sentence about why it is the best choice"))
	}
	if isLocalContextURL(spec.ContextURL) {
		warnings = append(warnings, advice(WarnLocalContextURL, "context_url",
			"context_url points at this machine, which the operator's phone cannot open; attach the file instead"))
	}
	if paths := unattachedPaths(spec.TechnicalDetails, attachments); len(paths) > 0 {
		warnings = append(warnings, advice(WarnUnattachedPath, "technical_details",
			fmt.Sprintf("technical_details names a local file that is not attached (%s); the operator cannot open it from the Hub", quoteList(paths))))
	}
	return warnings
}

func quoteList(values []string) string {
	shown := values
	if len(shown) > maxNamedInWarning {
		shown = shown[:maxNamedInWarning]
	}
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

var (
	// A capital letter with one or two digits as a whole word (S7, O7, F49), or
	// with a hyphen and up to five digits (S-14, D-065).
	shortLabelPattern = regexp.MustCompile(`(?:^|[^A-Za-z0-9_-])([A-Z](?:\d{1,2}|-\d{1,5}))\b`)
	// Two to four capitals, a hyphen, digits: BAS-065. Standards such as
	// SHA-256 and UTF-8 are allowed prefixes below.
	prefixedLabelPattern = regexp.MustCompile(`(?:^|[^A-Za-z0-9_-])([A-Z]{2,4}-\d{1,5})\b`)
	// Three or more capital/digit segments: F49-MOR-CHECKOUT, BAS-D-065. One
	// segment must hold a digit, so SOME-CAPS-WORDS is left alone.
	segmentedLabelPattern = regexp.MustCompile(`(?:^|[^A-Za-z0-9_-])((?:[A-Z][A-Z0-9]*-){2,}[A-Z0-9]+)\b`)
	uuidPattern           = regexp.MustCompile(`(?i)\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b`)
	bracketTagPattern     = regexp.MustCompile(`^\s*(\[[^\]\n]{1,40}\])`)
	hasDigitPattern       = regexp.MustCompile(`\d`)
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

func labelPrefix(label string) string {
	if dash := strings.IndexByte(label, '-'); dash >= 0 {
		return label[:dash]
	}
	return label
}

// internalIDs returns the internal-looking labels in text, in order of
// appearance and without repeats: S-14, O7, F49-MOR-CHECKOUT, UUIDs. A label
// inside a longer one (F49 in F49-MOR-CHECKOUT) is reported once, as the
// longer.
func internalIDs(text string) []string {
	type span struct {
		start, end int
		label      string
	}
	var spans []span
	collect := func(pattern *regexp.Regexp, keep func(string) bool) {
		for _, match := range pattern.FindAllStringSubmatchIndex(text, -1) {
			label := text[match[2]:match[3]]
			if keep(label) && !continuesAsVersion(text[match[3]:]) {
				spans = append(spans, span{start: match[2], end: match[3], label: label})
			}
		}
	}
	collect(shortLabelPattern, func(label string) bool { return !wellKnownShortLabels[label] })
	collect(prefixedLabelPattern, func(label string) bool { return !wellKnownLabelPrefixes[labelPrefix(label)] })
	collect(segmentedLabelPattern, func(label string) bool {
		return hasDigitPattern.MatchString(label) && !wellKnownLabelPrefixes[labelPrefix(label)]
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

func bracketedTag(text string) string {
	match := bracketTagPattern.FindStringSubmatch(text)
	if match == nil {
		return ""
	}
	return match[1]
}

// --- file paths ------------------------------------------------------------

var (
	// A path starts a word: after whitespace, an opening bracket, or a quote.
	// URLs ("https://host/a/b") never match because their slash follows ":".
	pathStart = `(?:^|[\s("'` + "`" + `\[])`
	// Absolute paths under the roots a host's files live in.
	hostRootPathPattern = regexp.MustCompile(pathStart + `(/(?:home|tmp|var|etc|usr|opt|mnt|root|srv|run|media|dev|proc|Users|Volumes|private)/[^\s"'` + "`" + `)\]>,;]+)`)
	// Home-relative and explicitly relative paths: ~/notes.md, ./out/report.txt.
	relativePathPattern = regexp.MustCompile(pathStart + `((?:~|\.{1,2})/[^\s"'` + "`" + `)\]>,;]+)`)
	// Windows drive paths: C:\Users\me\file.txt.
	windowsPathPattern = regexp.MustCompile(pathStart + `([A-Za-z]:\\[^\s"'` + "`" + `)\]>,;]+)`)
	// Any other absolute path with at least two segments and a file extension:
	// /srv-less project paths such as /project/docs/plan.md. API routes such as
	// /api/v1/asks have no extension and stay unflagged.
	extensionPathPattern = regexp.MustCompile(pathStart + `(/(?:[\w.@+-]+/)+[\w@+-]+\.[A-Za-z][A-Za-z0-9]{0,7})(?:$|[\s"'` + "`" + `)\]>,;:.])`)
)

// localPaths returns the file paths text names, without repeats and without
// trailing sentence punctuation.
func localPaths(text string) []string {
	var found []string
	seen := map[string]bool{}
	for _, pattern := range []*regexp.Regexp{hostRootPathPattern, relativePathPattern, windowsPathPattern, extensionPathPattern} {
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

// unattachedPaths returns the paths in text that no attachment's source path
// matches: the path itself, or a file inside an attached folder.
func unattachedPaths(text string, attachments []PacketAttachment) []string {
	var missing []string
	for _, candidate := range localPaths(text) {
		if !pathIsAttached(candidate, attachments) {
			missing = append(missing, candidate)
		}
	}
	return missing
}

func pathIsAttached(candidate string, attachments []PacketAttachment) bool {
	cleaned := path.Clean(candidate)
	for _, attachment := range attachments {
		source := strings.TrimRight(attachment.SourcePath, "/")
		if source == "" {
			continue
		}
		if cleaned == path.Clean(source) || strings.HasPrefix(cleaned, path.Clean(source)+"/") {
			return true
		}
	}
	return false
}

// --- context url -----------------------------------------------------------

// isLocalContextURL reports a context_url the operator's phone cannot open: a
// path on the requester's machine or an address on its loopback interface.
func isLocalContextURL(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	lower := strings.ToLower(raw)
	if strings.HasPrefix(lower, "file:") || strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "~") || strings.HasPrefix(raw, ".") {
		return true
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Hostname() == "" {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "localhost" || strings.HasSuffix(host, ".localhost") || host == "0.0.0.0" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
