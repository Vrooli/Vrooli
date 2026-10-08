package trailers

import (
	"fmt"
	"regexp"
	"strings"
)

// MaxValueLength bounds one trailer value.
const MaxValueLength = 256

// Spec is one supported trailer key. Each key carries one serialized Agent
// Manager work reference (kind, id, optional revision); the key encodes the
// kind, so adding a kind is one row.
type Spec struct {
	Key  string
	Kind string
	// Max is the largest number of entries allowed; zero means unbounded.
	Max     int
	Grammar *regexp.Regexp
	// Example is a valid value, used in validation messages.
	Example string
}

var (
	effortRefPattern = `effort:[A-Za-z0-9][A-Za-z0-9._:/+-]*`
	revisionPattern  = `[A-Za-z0-9][A-Za-z0-9._-]{0,63}`
	workIDPattern    = `[A-Za-z0-9][A-Za-z0-9._:/#+-]*`
)

// Registry lists the supported keys in generated render order.
var Registry = []Spec{
	{Key: "Vrooli-Effort", Kind: "effort", Grammar: regexp.MustCompile(`^` + effortRefPattern + `(@` + revisionPattern + `)?$`), Example: "effort:browser-automation-studio-rehabilitation@9"},
	{Key: "Vrooli-Epoch", Kind: "epoch", Grammar: regexp.MustCompile(`^` + effortRefPattern + `#E[0-9]{1,6}$`), Example: "effort:browser-automation-studio-rehabilitation#E26"},
	{Key: "Vrooli-Run", Kind: "run", Max: 20, Grammar: regexp.MustCompile(`^(?i:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})$`), Example: "e8c322ca-72e7-434c-9d37-1d2cf853961f"},
	{Key: "Vrooli-Plan", Kind: "plan", Grammar: regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]*(#phase-[0-9]{1,4})?$`), Example: "git-control-tower-advisory-maturity-20260905#phase-6"},
	{Key: "Vrooli-Backlog", Kind: "backlog", Grammar: regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*/[A-Za-z0-9][A-Za-z0-9._:/-]*$`), Example: "bug-inbox/gct-commit-timeout"},
	{Key: "Vrooli-Continues", Kind: "commit", Max: 1, Grammar: regexp.MustCompile(`^[0-9a-f]{7,40}$`), Example: "0e1f1210438"},
	{Key: "Vrooli-Proposal", Kind: "proposal", Max: 1, Grammar: regexp.MustCompile(`^gctp-[0-9a-f]{8,32}$`), Example: "gctp-3f9a1c2b7d4e"},
	{Key: "Vrooli-Work", Kind: "", Grammar: regexp.MustCompile(`^[a-z][a-z0-9_-]{0,31} ` + workIDPattern + `(@` + revisionPattern + `)?$`), Example: "incident inc-2026-10-07@2"},
}

// SpecFor returns the registry row for a canonical or case-variant key.
func SpecFor(key string) (Spec, bool) {
	for _, spec := range Registry {
		if strings.EqualFold(spec.Key, strings.TrimSpace(key)) {
			return spec, true
		}
	}
	return Spec{}, false
}

// KindOf returns the work-reference kind a trailer carries. Vrooli-Work takes
// its kind from the value; unknown Vrooli keys use their suffix; other keys
// have no kind.
func KindOf(key, value string) string {
	spec, ok := SpecFor(key)
	if ok && spec.Kind != "" {
		return spec.Kind
	}
	if ok && spec.Key == "Vrooli-Work" {
		kind, _, _ := strings.Cut(strings.TrimSpace(value), " ")
		return kind
	}
	if IsVrooliKey(key) {
		return strings.ToLower(strings.TrimSpace(key)[len("Vrooli-"):])
	}
	return ""
}

// Severity of a validation issue. Errors block agent-created proposals;
// operator commits only see them as warnings.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// Issue is one validation finding. Index is the trailer index, or -1 for the
// subject or body.
type Issue struct {
	Index    int      `json:"index"`
	Key      string   `json:"key"`
	Code     string   `json:"code"`
	Message  string   `json:"message"`
	Severity Severity `json:"severity"`
}

// ValidateOptions selects caller-dependent rules.
type ValidateOptions struct {
	// AgentAuthored rejects agent co-author trailers: agents appear only
	// through Vrooli-Run, and the operator is always the commit author.
	AgentAuthored bool
}

// Validate checks a subject, body and trailer list. It never resolves
// references; an unresolvable reference is not a validation issue.
func Validate(subject, body string, entries []Entry, opts ValidateOptions) []Issue {
	var issues []Issue
	switch {
	case strings.TrimSpace(subject) == "":
		issues = append(issues, Issue{Index: -1, Key: "subject", Code: "subject_required", Message: "commit subject is required", Severity: SeverityError})
	case strings.ContainsAny(subject, "\r\n"):
		issues = append(issues, Issue{Index: -1, Key: "subject", Code: "subject_multiline", Message: "commit subject must be one line", Severity: SeverityError})
	}
	if len(entries) == 0 && bodyEndsInTrailerBlock(body) {
		issues = append(issues, Issue{Index: -1, Key: "body", Code: "syntax", Message: "the body's last paragraph reads as a trailer block; add a trailer or reword it", Severity: SeverityWarning})
	}
	counts := map[string]int{}
	for i, entry := range entries {
		if entry.Key == "" {
			continue
		}
		if _, _, ok := splitTrailerLine(entry.Key + ": x"); !ok {
			issues = append(issues, Issue{Index: i, Key: entry.Key, Code: "syntax", Message: fmt.Sprintf("trailer key %q must be letters, digits and '-'", entry.Key), Severity: SeverityError})
			continue
		}
		value := entry.Value
		if strings.ContainsAny(value, "\r\n") {
			issues = append(issues, Issue{Index: i, Key: entry.Key, Code: "syntax", Message: "trailer value must be one line", Severity: SeverityError})
		}
		if len(value) > MaxValueLength {
			issues = append(issues, Issue{Index: i, Key: entry.Key, Code: "length", Message: fmt.Sprintf("trailer value exceeds %d characters", MaxValueLength), Severity: SeverityError})
		}
		if strings.EqualFold(entry.Key, "Co-Authored-By") && opts.AgentAuthored {
			issues = append(issues, Issue{Index: i, Key: entry.Key, Code: "forbidden_key", Message: "agents appear only through Vrooli-Run; the operator is the commit author", Severity: SeverityError})
		}
		spec, known := SpecFor(entry.Key)
		if !known {
			if IsVrooliKey(entry.Key) {
				issues = append(issues, Issue{Index: i, Key: entry.Key, Code: "legacy_key", Message: "unknown Vrooli trailer is kept as written and never resolved", Severity: SeverityWarning})
			}
			continue
		}
		counts[spec.Key]++
		if spec.Max > 0 && counts[spec.Key] == spec.Max+1 {
			issues = append(issues, Issue{Index: i, Key: spec.Key, Code: "multiplicity", Message: fmt.Sprintf("%s allows at most %d", spec.Key, spec.Max), Severity: SeverityError})
		}
		if !spec.Grammar.MatchString(strings.TrimSpace(value)) {
			issues = append(issues, Issue{Index: i, Key: spec.Key, Code: "value_grammar", Message: fmt.Sprintf("%s value %q does not match the grammar; example: %s", spec.Key, value, spec.Example), Severity: SeverityError})
		}
	}
	return issues
}

// bodyEndsInTrailerBlock reports whether git would read the body's final
// paragraph as trailers when no trailer block follows it.
func bodyEndsInTrailerBlock(body string) bool {
	paragraphs := splitParagraphs(strings.Split(strings.ReplaceAll(strings.TrimSpace(body), "\r\n", "\n"), "\n"))
	if len(paragraphs) == 0 {
		return false
	}
	_, ok := parseTrailerBlock(paragraphs[len(paragraphs)-1])
	return ok
}

// HasErrors reports whether any issue is an error.
func HasErrors(issues []Issue) bool {
	for _, issue := range issues {
		if issue.Severity == SeverityError {
			return true
		}
	}
	return false
}
