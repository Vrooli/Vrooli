package dochealth

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/vrooli/api-core/markedrefs"
)

// DOC: docs/internal/SEAMS.md#dochealth
// Number / derived-count lint. A generic content check: it flags drift-prone
// hardcoded counts in prose ("four teams", "30+ resources", "seven audit
// lenses") and demands a per-occurrence decision — reword the number out, or
// tag it as an owner-backed value via the `num[<category>]` marker.
//
// Detection is deliberately CONSERVATIVE (OQ3): a count token (digit form or a
// curated word-number) must be adjacent to a plural "enumerable" noun, with
// generous excludes (dates, versions, units, identifiers, code spans). The lint
// emits WARNING-severity findings only, so it never breaks a build (the
// test-genie docs phase gates on FAILURE).

const (
	findingUnmarkedNumber       = "unmarked_number"
	findingNumberMarkerNoReason = "number_marker_without_reason"
)

var (
	inlineCodeSpanPattern = regexp.MustCompile("`[^`]*`")
	leadingListMarker     = regexp.MustCompile(`^\s*\d+[.)]\s+`)
	bareNumberToken       = regexp.MustCompile(`^(\d{1,6})(\+)?$`)
	plainWordPattern      = regexp.MustCompile(`^[a-z][a-z-]*$`)
)

// wordNumbers is the curated word-form count list. Singular ordinal-prone words
// (e.g. "second", "first") are intentionally excluded.
var wordNumbers = map[string]struct{}{
	"one": {}, "two": {}, "three": {}, "four": {}, "five": {}, "six": {},
	"seven": {}, "eight": {}, "nine": {}, "ten": {}, "eleven": {}, "twelve": {},
	"dozen": {}, "dozens": {}, "hundreds": {}, "thousands": {},
}

// nounStopwords are tokens that look plural (end in "s") but are not the kind of
// enumerable collection noun the lint targets — time units, adjectives, and
// common non-noun words. Conservative by design; expand only with evidence.
var nounStopwords = map[string]struct{}{
	// time / duration units
	"days": {}, "weeks": {}, "hours": {}, "years": {}, "mins": {}, "secs": {},
	"seconds": {}, "minutes": {}, "months": {}, "milliseconds": {}, "nanoseconds": {},
	// adjectives & common non-nouns ending in s
	"this": {}, "thus": {}, "plus": {}, "less": {}, "perhaps": {}, "across": {},
	"unless": {}, "whereas": {}, "towards": {}, "always": {}, "sometimes": {},
	"otherwise": {}, "various": {}, "previous": {}, "numerous": {}, "continuous": {},
	"obvious": {}, "serious": {}, "anonymous": {}, "synchronous": {}, "asynchronous": {},
	"miscellaneous": {}, "status": {}, "focus": {}, "versus": {}, "minus": {},
	"bonus": {}, "corpus": {}, "its": {}, "hers": {}, "ours": {}, "yours": {}, "theirs": {},
	// common third-person-singular verbs ending in s. After a number these
	// almost always mean the number was an identifier ("criterion 2 shows"),
	// not a count of a collection — so they are not enumerable nouns.
	"shows": {}, "feeds": {}, "runs": {}, "makes": {}, "gets": {}, "needs": {},
	"uses": {}, "helps": {}, "keeps": {}, "holds": {}, "gives": {}, "takes": {},
	"means": {}, "works": {}, "sets": {}, "lets": {}, "says": {}, "goes": {},
	"does": {}, "comes": {}, "becomes": {}, "includes": {}, "requires": {},
	"provides": {}, "returns": {}, "ensures": {}, "allows": {}, "tells": {},
	"ships": {}, "drifts": {}, "exists": {}, "applies": {}, "covers": {},
	"sends": {}, "seems": {}, "leads": {}, "knows": {}, "flows": {}, "grows": {},
	"moves": {}, "serves": {}, "solves": {}, "saves": {}, "loses": {}, "raises": {},
	"wants": {}, "looks": {}, "remains": {}, "maps": {},
}

// precedingExcludes: when the word immediately before a number is one of these,
// the number is an identifier / ordinal (version 2, phase 3, port 5432, PR 12)
// rather than a derived count of a collection.
var precedingExcludes = map[string]struct{}{
	"version": {}, "v": {}, "rev": {}, "revision": {}, "phase": {}, "step": {},
	"figure": {}, "fig": {}, "table": {}, "section": {}, "chapter": {}, "part": {},
	"tier": {}, "level": {}, "port": {}, "pr": {}, "issue": {}, "rfc": {},
	"page": {}, "line": {}, "item": {}, "no": {}, "num": {}, "stage": {}, "round": {},
}

// scanNumbersContent applies the number lint to a single document's content and
// returns the findings plus the count flagged. Exported (package-internal) and
// content-based so tests can exercise it without touching the filesystem.
func scanNumbersContent(path, content string) ([]Finding, int) {
	var findings []Finding

	scanner := bufio.NewScanner(strings.NewReader(content))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	lineNum := 0
	inFence := false
	fenceMarker := ""

	// Front-matter handling: a leading "---" fence (YAML) is skipped wholesale.
	frontMatter := false
	seenContent := false

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trim := strings.TrimSpace(line)

		if !seenContent && !frontMatter && trim == "---" {
			frontMatter = true
			seenContent = true
			continue
		}
		if frontMatter {
			if trim == "---" {
				frontMatter = false
			}
			continue
		}
		if trim != "" {
			seenContent = true
		}

		if matches := codeFencePattern.FindStringSubmatch(trim); len(matches) > 0 {
			marker := matches[1]
			if inFence && marker == fenceMarker {
				inFence = false
				fenceMarker = ""
			} else if !inFence {
				inFence = true
				fenceMarker = marker
			}
			continue
		}
		if inFence {
			continue
		}

		findings = append(findings, scanNumbersLine(path, line, lineNum)...)
	}

	return findings, len(findings)
}

// scanNumbersFile reads path and applies the number lint. Read errors are
// ignored here because the content validator already reports unreadable files.
func scanNumbersFile(path string) ([]Finding, int) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, 0
	}
	return scanNumbersContent(path, string(data))
}

func scanNumbersLine(path, line string, lineNum int) []Finding {
	var findings []Finding

	// 1. `num` markers: a recognized category is silent; a bare/unknown-category
	//    `num` marker is itself a finding.
	for _, ref := range markedrefs.ParseInlineCode(line, lineNum) {
		if ref.Marker != markedrefs.MarkerNum {
			continue
		}
		if _, ok := markedrefs.NumberCategory(ref); !ok {
			findings = append(findings, Finding{
				Code:     findingNumberMarkerNoReason,
				Severity: SeverityWarning,
				Message:  numberMarkerNoReasonGuidance(path, lineNum, ref.Raw),
				Path:     path,
				Line:     lineNum,
			})
		}
	}

	// 2. Unmarked counts in prose. Inline-code spans (which is where valid
	//    `num[...]` markers live) are stripped first, so tagged numbers and any
	//    other code-fenced values never reach the prose scan.
	prose := inlineCodeSpanPattern.ReplaceAllString(line, " ")
	prose = stripLeadingMarkdown(prose)
	tokens := strings.Fields(prose)
	for i, tok := range tokens {
		// Parenthesized / bracketed numbers are citations, footnotes, or
		// references ("(3)", "[12]"), not derived counts.
		if strings.HasPrefix(tok, "(") || strings.HasPrefix(tok, "[") {
			continue
		}
		number, ok := classifyCountToken(tok)
		if !ok {
			continue
		}
		// A number at a clause/sentence boundary ("...adding one.") has no noun
		// after it in the same clause.
		if endsWithHardStop(tok) {
			continue
		}
		if i > 0 {
			if _, skip := precedingExcludes[strings.ToLower(cleanWord(tokens[i-1]))]; skip {
				continue
			}
		}
		if noun, matched := findEnumerableNoun(tokens, i+1); matched {
			findings = append(findings, Finding{
				Code:     findingUnmarkedNumber,
				Severity: SeverityWarning,
				Message:  unmarkedNumberGuidance(path, lineNum, number, noun),
				Path:     path,
				Line:     lineNum,
			})
		}
	}

	return findings
}

// stripLeadingMarkdown removes leading heading hashes and an ordered-list /
// section-number marker so "## 3. Problem" and "1. four teams" are scanned as
// "Problem" and "four teams" — the "3." / "1." are structure, not counts.
func stripLeadingMarkdown(line string) string {
	out := strings.TrimLeft(line, "#")
	out = leadingListMarker.ReplaceAllString(out, "")
	return out
}

// classifyCountToken reports whether tok is a count token and returns the
// normalized number text. Digit forms that are decimals, versions, percentages,
// identifiers, or 4-digit years are rejected.
func classifyCountToken(tok string) (string, bool) {
	w := cleanWord(tok)
	if w == "" {
		return "", false
	}
	lw := strings.ToLower(w)
	if _, ok := wordNumbers[lw]; ok {
		return lw, true
	}
	m := bareNumberToken.FindStringSubmatch(w)
	if m == nil {
		return "", false
	}
	if m[2] == "" && len(m[1]) == 4 {
		if y, err := strconv.Atoi(m[1]); err == nil && y >= 1900 && y <= 2099 {
			return "", false // standalone 4-digit year
		}
	}
	return w, true
}

// findEnumerableNoun looks for a plural enumerable noun starting at index start,
// allowing up to two intervening modifier words (adjectives). The search stops
// at a clause/sentence boundary so it never matches a noun in the next sentence.
func findEnumerableNoun(tokens []string, start int) (string, bool) {
	intervening := 0
	for j := start; j < len(tokens) && intervening <= 2; j++ {
		w := cleanWord(tokens[j])
		if w == "" {
			return "", false
		}
		lw := strings.ToLower(w)
		if isEnumerableNoun(lw) {
			return lw, true
		}
		if !plainWordPattern.MatchString(lw) {
			return "", false
		}
		// A modifier that ends the clause stops the search before the next one.
		if endsWithHardStop(tokens[j]) {
			return "", false
		}
		intervening++
	}
	return "", false
}

// endsWithHardStop reports whether a raw token ends with clause/sentence
// terminating punctuation (after trimming wrapping quotes/brackets).
func endsWithHardStop(tok string) bool {
	tok = strings.TrimRight(tok, ")]}\"'*_`")
	if tok == "" {
		return false
	}
	switch tok[len(tok)-1] {
	case '.', '!', '?', ':', ';':
		return true
	default:
		return false
	}
}

func isEnumerableNoun(w string) bool {
	if len(w) < 4 {
		return false
	}
	if !plainWordPattern.MatchString(w) {
		return false
	}
	if !strings.HasSuffix(w, "s") || strings.HasSuffix(w, "ss") {
		return false
	}
	if _, stop := nounStopwords[w]; stop {
		return false
	}
	return true
}

// cleanWord strips surrounding markdown/punctuation from a whitespace token,
// preserving a meaningful trailing "+" (e.g. "30+").
func cleanWord(tok string) string {
	tok = strings.TrimLeft(tok, "([{\"'*_`#")
	tok = strings.TrimRight(tok, ")]}\"'*_`.,:;!?")
	return tok
}

func unmarkedNumberGuidance(path string, line int, number, noun string) string {
	return fmt.Sprintf(
		"%s:%d unmarked count %q before %q — counts in prose drift. Reword it out, point at the source of truth, or if it is owner-backed tag it as `num[<category>]:%s` (category ∈ target/threshold/price/version/decision/sot).",
		path, line, number, noun, number,
	)
}

func numberMarkerNoReasonGuidance(path string, line int, raw string) string {
	return fmt.Sprintf(
		"%s:%d intentional-number marker %s has no recognized justification — use `num[<category>]:<value>` (category ∈ target/threshold/price/version/decision/sot) or reword the number out.",
		path, line, raw,
	)
}
