// Package conversationpolicy is the declarative visibility/exclusion policy for
// imported and indexed conversations. It gives an operator a data-driven way to
// keep certain conversations out of the import corpus and/or out of search —
// matched by harness, model, project scope, cwd scope, or tag.
//
// The package is deliberately dependency-free (no config, no orchestration, no
// conversationsearch) so both the import seam (orchestration.ImportTranscript)
// and the search seam (conversationsearch.SQLiteSource) can consume the same
// evaluator without an import cycle. config.Levers builds a Policy from its
// ConversationPolicy lever section.
//
// Observability is a first-class goal: a policy must never silently disappear a
// conversation. Every enforcement increments a visible counter, and the active
// rules plus hide/exclude counts are surfaced through `conversation policy
// list` and the import summary. This is the explicit counter-measure to the
// "silently hidden" failure mode W1 just fixed for codec parse drops.
package conversationpolicy

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
)

// Action is what a matched rule does to a conversation.
type Action string

const (
	// ActionExclude keeps the conversation out entirely at the enforcement
	// point. At the import seam an excluded run writes zero run_events; at the
	// search seam its documents are filtered from every result.
	ActionExclude Action = "exclude"
	// ActionRedact masks PII in the conversation's content before it is indexed.
	// A matched run is still imported and searchable; its content fields are
	// scrubbed of PII (via the ai-gateway DETECT_PII route) at the import seam,
	// before the event store write. The masking is fail-closed: if the detector
	// is unavailable the content is withheld (replaced with a loud sentinel),
	// never indexed in the clear. Enforcement is at the import seam only — a
	// redact rule declared at the search seam is inert. It never excludes.
	ActionRedact Action = "redact"
	// ActionIndex is the explicit "keep this" action. It exists so a narrow
	// allow rule can precede a broad exclude rule (first-match-wins precedence).
	ActionIndex Action = "index"
)

// EnforcePoint selects where a rule takes effect. A single rule chooses exactly
// one point so enforcement is never ambiguous.
type EnforcePoint string

const (
	// EnforceImport blocks at ingest — the strongest privacy guarantee, for
	// truly-private data (e.g. private local-model chats). An import-excluded
	// run never produces run_events.
	EnforceImport EnforcePoint = "import"
	// EnforceSearch keeps the data indexed but hides it from search results.
	// Reversible and observable; good for per-viewer scoping.
	EnforceSearch EnforcePoint = "search"
)

// Fields is the match surface of a conversation. Every field is matched
// case-insensitively (after trimming). An empty rule field is a wildcard that
// matches any value, including empty. A rule field may use a trailing/leading
// `*` glob (see matchField) so a model prefix such as `claude*` matches
// `claude-opus-4-8[1m]` — exact equality remains the default when no `*`
// appears.
type Fields struct {
	Harness      string
	Model        string
	ProjectScope string
	CwdScope     string
	Tag          string
}

// Rule is one declarative match → action mapping. An unset match field is a
// wildcard. Every set field must match for the rule to apply (AND semantics).
type Rule struct {
	Name         string       `json:"name,omitempty"`
	Harness      string       `json:"harness,omitempty"`
	Model        string       `json:"model,omitempty"`
	ProjectScope string       `json:"projectScope,omitempty"`
	CwdScope     string       `json:"cwdScope,omitempty"`
	Tag          string       `json:"tag,omitempty"`
	Action       Action       `json:"action"`
	EnforceAt    EnforcePoint `json:"enforceAt"`
}

// IsWildcard reports whether the rule matches every conversation (all match
// fields unset). A wildcard exclude is a blunt instrument, so callers may warn.
func (r Rule) IsWildcard() bool {
	return strings.TrimSpace(r.Harness) == "" &&
		strings.TrimSpace(r.Model) == "" &&
		strings.TrimSpace(r.ProjectScope) == "" &&
		strings.TrimSpace(r.CwdScope) == "" &&
		strings.TrimSpace(r.Tag) == ""
}

// Specificity counts the set match fields. More specific rules are not
// auto-reordered — precedence is declaration order (first match wins) — but the
// count is surfaced so an operator can see why a rule did or did not fire.
func (r Rule) Specificity() int {
	count := 0
	for _, field := range []string{r.Harness, r.Model, r.ProjectScope, r.CwdScope, r.Tag} {
		if strings.TrimSpace(field) != "" {
			count++
		}
	}
	return count
}

// ImportUnobservableFields lists the match fields this rule sets that the
// import seam cannot evaluate. Only harness and tag are known when a transcript
// is adopted (see orchestration.importPolicyFields); model, project, and cwd
// are resolved later, so an import-enforced rule that sets any of them can
// never fire at import and would silently match nothing. Returns nil when the
// rule is not import-enforced or sets only observable fields.
func (r Rule) ImportUnobservableFields() []string {
	if r.EnforceAt != EnforceImport {
		return nil
	}
	var unobservable []string
	if strings.TrimSpace(r.Model) != "" {
		unobservable = append(unobservable, "model")
	}
	if strings.TrimSpace(r.ProjectScope) != "" {
		unobservable = append(unobservable, "project")
	}
	if strings.TrimSpace(r.CwdScope) != "" {
		unobservable = append(unobservable, "cwd")
	}
	return unobservable
}

func (r Rule) matches(f Fields) bool {
	return matchField(r.Harness, f.Harness) &&
		matchField(r.Model, f.Model) &&
		matchField(r.ProjectScope, f.ProjectScope) &&
		matchField(r.CwdScope, f.CwdScope) &&
		matchField(r.Tag, f.Tag)
}

// matchField reports whether a rule field matches a conversation field value.
//
// Matching is case-insensitive after trimming. An empty rule field is a
// whole-field wildcard (matches any value). A rule field containing `*` is a
// glob so operators can express the common "this model family" case without
// knowing the exact suffix a provider emits:
//
//	"claude*"  matches "claude-opus-4-8[1m]" (prefix)
//	"*opus*"   matches "claude-opus-4-8"     (substring)
//	"*[1m]"    matches "claude-opus-4-8[1m]" (suffix)
//	"claude"   matches only "claude"          (exact — no `*`, so literal)
//
// Without a `*` the match stays exact equality, so existing exact rules keep
// their precise semantics. The glob is literal everywhere except `*` (no
// `?`/`[` classes) to keep the operator contract small and predictable.
func matchField(rule, value string) bool {
	rule = strings.TrimSpace(rule)
	if rule == "" {
		return true // wildcard
	}
	value = strings.TrimSpace(value)
	if !strings.Contains(rule, "*") {
		return strings.EqualFold(rule, value)
	}
	return globMatch(strings.ToLower(rule), strings.ToLower(value))
}

// globMatch matches value against a lowercased glob pattern whose only
// metacharacter is `*` (matches any run of characters, including empty). Both
// arguments must already be lowercased by the caller.
func globMatch(pattern, value string) bool {
	segments := strings.Split(pattern, "*")
	// No `*`: exact (caller normally routes these through EqualFold, but keep
	// globMatch total so it is safe to call directly from tests).
	if len(segments) == 1 {
		return pattern == value
	}
	// Leading segment must be a prefix; trailing segment must be a suffix.
	if first := segments[0]; first != "" {
		if !strings.HasPrefix(value, first) {
			return false
		}
		value = value[len(first):]
	}
	last := segments[len(segments)-1]
	if last != "" {
		if !strings.HasSuffix(value, last) {
			return false
		}
		value = value[:len(value)-len(last)]
	}
	// Interior segments must appear in order.
	for _, segment := range segments[1 : len(segments)-1] {
		if segment == "" {
			continue
		}
		index := strings.Index(value, segment)
		if index < 0 {
			return false
		}
		value = value[index+len(segment):]
	}
	return true
}

// Validate rejects a structurally invalid rule before it can mask a
// conversation for the wrong reason.
func (r Rule) Validate() error {
	switch r.Action {
	case ActionExclude, ActionRedact, ActionIndex:
	case "":
		return fmt.Errorf("action is required (exclude, redact, or index)")
	default:
		return fmt.Errorf("invalid action %q (expected exclude, redact, or index)", r.Action)
	}
	switch r.EnforceAt {
	case EnforceImport, EnforceSearch:
	case "":
		return fmt.Errorf("enforceAt is required (import or search)")
	default:
		return fmt.Errorf("invalid enforceAt %q (expected import or search)", r.EnforceAt)
	}
	return nil
}

// Decision is the resolved outcome for a conversation at one enforcement point.
type Decision struct {
	// Matched is false when no rule applied (default: keep/index).
	Matched bool
	// Rule is the first matching rule (zero value when Matched is false).
	Rule Rule
	// RuleIndex is the position of the matching rule in declaration order.
	RuleIndex int
	Action    Action
}

// Hidden reports whether this decision keeps the conversation out at its
// enforcement point. ActionExclude hides; ActionRedact and ActionIndex (and the
// no-match default) keep it.
func (d Decision) Hidden() bool { return d.Matched && d.Action == ActionExclude }

// Policy is an ordered, validated rule set plus observability counters. It is
// safe for concurrent use: evaluation and counter reads happen on hot paths
// (every imported run, every indexed page).
type Policy struct {
	rules []Rule

	importExcluded atomic.Int64
	searchHidden   atomic.Int64
	redactMarked   atomic.Int64

	mu      sync.Mutex
	byRule  map[string]int64 // rule name/key -> enforcement count
	enabled bool
}

// New validates and compiles a rule set into a Policy. A nil/empty rule set
// yields a disabled policy whose Evaluate always returns the keep default, so
// callers never need a nil check.
func New(rules []Rule) (*Policy, error) {
	compiled := make([]Rule, 0, len(rules))
	for i, rule := range rules {
		if err := rule.Validate(); err != nil {
			return nil, fmt.Errorf("conversation policy rule %d (%s): %w", i, ruleLabel(rule, i), err)
		}
		compiled = append(compiled, rule)
	}
	return &Policy{rules: compiled, byRule: map[string]int64{}, enabled: len(compiled) > 0}, nil
}

func ruleLabel(r Rule, index int) string {
	if name := strings.TrimSpace(r.Name); name != "" {
		return name
	}
	return fmt.Sprintf("rule-%d", index)
}

// Enabled reports whether the policy has any rules. A disabled policy adds no
// per-conversation cost beyond a single comparison.
func (p *Policy) Enabled() bool { return p != nil && p.enabled }

// Rules returns a copy of the compiled rules in declaration order.
func (p *Policy) Rules() []Rule {
	if p == nil {
		return nil
	}
	out := make([]Rule, len(p.rules))
	copy(out, p.rules)
	return out
}

// Evaluate returns the decision for a conversation at the given enforcement
// point using first-match-wins precedence over rules declared for that point.
// Rules for the other enforcement point are ignored, so an import rule never
// hides a search result and vice versa.
func (p *Policy) Evaluate(point EnforcePoint, f Fields) Decision {
	if !p.Enabled() {
		return Decision{Action: ActionIndex}
	}
	for i, rule := range p.rules {
		if rule.EnforceAt != point {
			continue
		}
		if rule.matches(f) {
			return Decision{Matched: true, Rule: rule, RuleIndex: i, Action: rule.Action}
		}
	}
	return Decision{Action: ActionIndex}
}

// RecordImport evaluates at the import seam and records the outcome. It returns
// true when the conversation must be excluded (no events written). A redact
// match is recorded (intent) but does not exclude.
func (p *Policy) RecordImport(f Fields) Decision {
	decision := p.Evaluate(EnforceImport, f)
	if !decision.Matched {
		return decision
	}
	switch decision.Action {
	case ActionExclude:
		p.importExcluded.Add(1)
	case ActionRedact:
		p.redactMarked.Add(1)
	}
	p.bump(decision)
	return decision
}

// RecordSearchHide evaluates at the search seam and records the outcome. It
// returns true when the document must be hidden from results.
func (p *Policy) RecordSearchHide(f Fields) Decision {
	decision := p.Evaluate(EnforceSearch, f)
	if !decision.Matched {
		return decision
	}
	switch decision.Action {
	case ActionExclude:
		p.searchHidden.Add(1)
	case ActionRedact:
		p.redactMarked.Add(1)
	}
	p.bump(decision)
	return decision
}

func (p *Policy) bump(decision Decision) {
	key := ruleLabel(decision.Rule, decision.RuleIndex)
	p.mu.Lock()
	p.byRule[key]++
	p.mu.Unlock()
}

// Counts is a point-in-time observability snapshot.
type Counts struct {
	// ImportExcluded is how many imported runs were kept out of the corpus.
	ImportExcluded int64 `json:"importExcluded"`
	// SearchHidden is how many documents/runs were filtered from search.
	SearchHidden int64 `json:"searchHidden"`
	// RedactMarked is how many conversations matched a redact rule (their content
	// is masked for PII before indexing; see piimask for mask/fail-closed counts).
	RedactMarked int64 `json:"redactMarked"`
	// ByRule maps each rule's label to its enforcement count.
	ByRule map[string]int64 `json:"byRule,omitempty"`
}

// Snapshot returns the current enforcement counts. Safe for concurrent use.
func (p *Policy) Snapshot() Counts {
	if !p.Enabled() {
		return Counts{}
	}
	p.mu.Lock()
	byRule := make(map[string]int64, len(p.byRule))
	for key, count := range p.byRule {
		byRule[key] = count
	}
	p.mu.Unlock()
	return Counts{
		ImportExcluded: p.importExcluded.Load(),
		SearchHidden:   p.searchHidden.Load(),
		RedactMarked:   p.redactMarked.Load(),
		ByRule:         byRule,
	}
}

// DescribeRules renders the compiled rules in declaration order for the CLI.
func (p *Policy) DescribeRules() []string {
	rules := p.Rules()
	lines := make([]string, 0, len(rules))
	for i, rule := range rules {
		lines = append(lines, fmt.Sprintf("%d. %s", i, DescribeRule(rule)))
	}
	return lines
}

// DescribeRule renders one rule as a stable, operator-readable line. Match
// fields render in the operator's mental field order (harness, model, project,
// cwd, tag) rather than alphabetically. A redact rule carries an explicit
// marker so `conversation policy list` makes its effect visible: a redact rule
// at the import seam masks PII before indexing (run stays searchable); one at
// the search seam is inert (masking runs at import).
func DescribeRule(r Rule) string {
	fields := []struct{ label, value string }{
		{"harness", r.Harness},
		{"model", r.Model},
		{"project", r.ProjectScope},
		{"cwd", r.CwdScope},
		{"tag", r.Tag},
	}
	matchers := make([]string, 0, len(fields))
	for _, field := range fields {
		if strings.TrimSpace(field.value) != "" {
			matchers = append(matchers, fmt.Sprintf("%s=%s", field.label, field.value))
		}
	}
	match := strings.Join(matchers, " ")
	if match == "" {
		match = "*"
	}
	name := strings.TrimSpace(r.Name)
	if name != "" {
		name = " (" + name + ")"
	}
	suffix := ""
	if r.Action == ActionRedact {
		// Redact masks PII in content before it is indexed (W5). It is enforced
		// at the import seam: a matched run's content is scrubbed via the PII
		// detector before the event store write, so the run stays searchable but
		// its PII does not. A redact rule at the search seam is inert (masking
		// happens on ingest), which the marker makes visible.
		if r.EnforceAt == EnforceImport {
			suffix = " [masks PII before index; run stays searchable]"
		} else {
			suffix = " [no effect at search seam: PII masking runs at import]"
		}
	}
	return fmt.Sprintf("[%s @ %s] %s%s%s", r.Action, r.EnforceAt, match, name, suffix)
}
