package summarize

import (
	"regexp"
	"strings"
	"unicode"
)

// assignmentCue recognizes a deliberately small set of explicit assignment
// forms. This is a source-span rescue, not a general semantic fact extractor.
var assignmentCue = regexp.MustCompile(`(?i)\b(owns|is responsible for|are responsible for|will handle|handles)\b`)
var assignmentNegation = regexp.MustCompile(`(?i)\b(not|never|isn't|aren't|wasn't|weren't|no longer)\b`)

// SourceAssignment retains the verbatim sentence and byte offsets that support
// a likely named assignment. Offsets refer to the exact source passed to
// ExtractSourceAssignments; callers must not apply them to normalized text.
type SourceAssignment struct {
	Sentence  string
	StartByte int
	EndByte   int
	Actor     string
	Action    string
}

// ExtractSourceAssignments conservatively selects source sentences with a
// responsibility cue and a plausible named actor. It intentionally misses
// assignments outside this lexical pattern rather than inventing facts.
func ExtractSourceAssignments(source string) []SourceAssignment {
	var facts []SourceAssignment
	for _, span := range splitSourceSentenceSpans(source) {
		sentence := strings.TrimSpace(span.text)
		loc := assignmentCue.FindStringIndex(sentence)
		if loc == nil || assignmentNegation.MatchString(sentence[:loc[0]]) {
			continue
		}
		actor := lastLexicalToken(sentence[:loc[0]])
		action := meaningfulPhrase(sentence[loc[1]:])
		if !plausibleNamedActor(actor) || action == "" {
			continue
		}
		facts = append(facts, SourceAssignment{
			Sentence: sentence, StartByte: span.start, EndByte: span.end,
			Actor: actor, Action: action,
		})
	}
	return facts
}

// PreserveExplicitAssignments rescues omitted candidate assignments by
// appending their exact source sentence. It never rewrites model text and is
// idempotent. A longer result is intentional when preserving a source fact;
// this bounded behavior may exceed the model's concise-summary budget.
func PreserveExplicitAssignments(source, summary string) string {
	return PreserveSourceAssignments(ExtractSourceAssignments(source), summary)
}

// PreserveSourceAssignments verifies candidate facts carried from the source
// boundary and appends omitted exact spans in source order.
func PreserveSourceAssignments(facts []SourceAssignment, summary string) string {
	result := strings.TrimSpace(summary)
	for _, fact := range facts {
		if assignmentCovered(result, fact) {
			continue
		}
		if result != "" {
			result += "\n\n"
		}
		result += fact.Sentence
	}
	return result
}

func assignmentCovered(summary string, fact SourceAssignment) bool {
	actor := strings.ToLower(fact.Actor)
	action := strings.ToLower(fact.Action)
	for _, sentence := range splitSourceSentences(summary) {
		lowerSentence := strings.ToLower(sentence)
		if strings.Contains(lowerSentence, actor) && strings.Contains(lowerSentence, action) {
			return true
		}
	}
	return false
}

func plausibleNamedActor(actor string) bool {
	if actor == "" {
		return false
	}
	// Common sentence starters and pronouns are not named owners. Requiring an
	// initial capital also avoids treating generic lowercase clauses as names.
	if strings.Contains("|i|we|you|he|she|they|it|the|this|that|there|who|someone|everyone|", "|"+strings.ToLower(actor)+"|") {
		return false
	}
	for _, r := range actor {
		return unicode.IsUpper(r)
	}
	return false
}

func lastLexicalToken(text string) string {
	fields := strings.FieldsFunc(text, func(r rune) bool {
		return !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '\'' || r == '-')
	})
	if len(fields) == 0 {
		return ""
	}
	return strings.Trim(fields[len(fields)-1], "'-")
}

func meaningfulPhrase(text string) string {
	text = strings.TrimSpace(strings.TrimRight(text, ".!?;:"))
	return strings.Join(strings.Fields(text), " ")
}

type sourceSentenceSpan struct {
	text       string
	start, end int
}

func splitSourceSentenceSpans(source string) []sourceSentenceSpan {
	var spans []sourceSentenceSpan
	start := 0
	for i, r := range source {
		if r != '.' && r != '!' && r != '?' {
			continue
		}
		end := i + len(string(r))
		if end < len(source) && source[end] != ' ' && source[end] != '\n' && source[end] != '\t' {
			continue
		}
		if sentence := strings.TrimSpace(source[start:end]); sentence != "" {
			left := strings.Index(source[start:end], sentence)
			spans = append(spans, sourceSentenceSpan{text: sentence, start: start + left, end: start + left + len(sentence)})
		}
		start = end
	}
	if sentence := strings.TrimSpace(source[start:]); sentence != "" {
		left := strings.Index(source[start:], sentence)
		spans = append(spans, sourceSentenceSpan{text: sentence, start: start + left, end: start + left + len(sentence)})
	}
	return spans
}

func splitSourceSentences(source string) []string {
	spans := splitSourceSentenceSpans(source)
	sentences := make([]string, 0, len(spans))
	for _, span := range spans {
		sentences = append(sentences, span.text)
	}
	return sentences
}
