package conversationpolicy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Named sample values shared across the policy tests. Hoisting these makes the
// intent explicit ("this is the model the fixture seeds") and a rename a single
// edit rather than a scatter of bare literals.
const (
	sampleOllama     = "ollama"
	sampleGrok       = "grok"
	sampleModel      = "model-actual"
	sampleClaudeFull = "claude-opus-4-8[1m]"
)

// Rule builders remove the repeated Action:/EnforceAt: boilerplate. Each takes
// the one match field the test cares about; compose the returned rule for
// multi-field cases.
func excludeImport(harness string) Rule {
	return Rule{Action: ActionExclude, EnforceAt: EnforceImport, Harness: harness}
}

func excludeSearch(model string) Rule {
	return Rule{Action: ActionExclude, EnforceAt: EnforceSearch, Model: model}
}

func redactImport(harness string) Rule {
	return Rule{Action: ActionRedact, EnforceAt: EnforceImport, Harness: harness}
}

// mustPolicy compiles rules and fails the test on a validation error.
func mustPolicy(t *testing.T, rules ...Rule) *Policy {
	t.Helper()
	policy, err := New(rules)
	if err != nil {
		t.Fatalf("New(%+v): %v", rules, err)
	}
	return policy
}

func TestRuleValidate(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		rule    Rule
		wantErr bool
	}{
		{"valid exclude import", excludeImport(""), false},
		{"valid redact search", Rule{Action: ActionRedact, EnforceAt: EnforceSearch, Model: "x"}, false},
		{"missing action", Rule{EnforceAt: EnforceImport}, true},
		{"bad action", Rule{Action: Action("drop"), EnforceAt: EnforceImport}, true},
		{"missing enforce", Rule{Action: ActionExclude}, true},
		{"bad enforce", Rule{Action: ActionExclude, EnforceAt: EnforcePoint("both")}, true},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if gotErr := tc.rule.Validate() != nil; gotErr != tc.wantErr {
				t.Fatalf("Validate() err?=%v, wantErr=%v", gotErr, tc.wantErr)
			}
		})
	}
}

// TestMatchFieldSemantics pins the exact-vs-glob matching contract: an empty
// rule is a wildcard, a rule without `*` is exact (case-insensitive), and a
// rule with `*` is a prefix/substring/suffix glob. This is the contract the
// headline "exclude this model" lever depends on — a bare `claude` must NOT
// match `claude-opus-4-8`, but `claude*` must.
func TestMatchFieldSemantics(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		rule  string
		value string
		want  bool
	}{
		{"empty rule is wildcard", "", "anything", true},
		{"exact match case-insensitive", "Ollama", "ollama", true},
		{"exact no partial", "claude", sampleClaudeFull, false},
		{"prefix glob", "claude*", sampleClaudeFull, true},
		{"prefix glob miss", "sonnet*", sampleClaudeFull, false},
		{"suffix glob", "*[1m]", sampleClaudeFull, true},
		{"substring glob", "*opus*", sampleClaudeFull, true},
		{"substring glob miss", "*haiku*", sampleClaudeFull, false},
		{"bare star matches all", "*", sampleClaudeFull, true},
		{"interior globs in order", "claude*opus*[1m]", sampleClaudeFull, true},
		{"interior globs out of order", "opus*claude*", sampleClaudeFull, false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := matchField(tc.rule, tc.value); got != tc.want {
				t.Fatalf("matchField(%q, %q) = %v, want %v", tc.rule, tc.value, got, tc.want)
			}
		})
	}
}

// TestModelGlobExcludesInProduction is the end-to-end form of the glob contract
// across both match surfaces: a search-seam rule `model:"claude*"` hides a run
// whose resolved model carries a provider suffix, and a harness glob matches a
// family of harnesses.
func TestModelGlobExcludesInProduction(t *testing.T) {
	t.Parallel()
	modelPolicy := mustPolicy(t, excludeSearch("claude*"))
	if !modelPolicy.Evaluate(EnforceSearch, Fields{Model: sampleClaudeFull}).Hidden() {
		t.Fatalf("model glob should hide %q", sampleClaudeFull)
	}
	if modelPolicy.Evaluate(EnforceSearch, Fields{Model: "gpt-4o"}).Matched {
		t.Fatalf("model glob must not match an unrelated model")
	}

	harnessPolicy := mustPolicy(t, excludeImport("resource:codex/*"))
	if !harnessPolicy.Evaluate(EnforceImport, Fields{Harness: "resource:codex/private"}).Hidden() {
		t.Fatalf("harness glob should hide a matching harness")
	}
	if harnessPolicy.Evaluate(EnforceImport, Fields{Harness: "resource:ollama/sessions"}).Matched {
		t.Fatalf("harness glob must not match a different resource family")
	}
}

func TestEvaluateWildcardAndExactMatch(t *testing.T) {
	t.Parallel()
	// A rule with no match fields is a wildcard (every conversation).
	wildcard := Rule{Action: ActionExclude, EnforceAt: EnforceSearch}
	if !wildcard.IsWildcard() {
		t.Fatalf("expected rule with no match fields to be a wildcard")
	}
	policy := mustPolicy(t, wildcard)
	if !policy.Evaluate(EnforceSearch, Fields{Harness: "anything"}).Hidden() {
		t.Fatalf("wildcard search-exclude should hide any conversation")
	}
	// A wildcard rule declared for search must not affect the import seam.
	if policy.Evaluate(EnforceImport, Fields{Harness: "anything"}).Matched {
		t.Fatalf("search rule must not match at the import seam")
	}

	// Exact, case-insensitive single-field match.
	exact := mustPolicy(t, excludeImport("Ollama"))
	if !exact.Evaluate(EnforceImport, Fields{Harness: sampleOllama}).Hidden() {
		t.Fatalf("harness match should be case-insensitive")
	}
	if exact.Evaluate(EnforceImport, Fields{Harness: "codex"}).Matched {
		t.Fatalf("non-matching harness must not match")
	}
}

func TestEvaluateMultiFieldAndSpecificity(t *testing.T) {
	t.Parallel()
	// All set fields must match (AND semantics).
	rule := Rule{Action: ActionExclude, EnforceAt: EnforceSearch, Harness: sampleOllama, Model: "llama3"}
	if rule.Specificity() != 2 {
		t.Fatalf("expected specificity 2, got %d", rule.Specificity())
	}
	policy := mustPolicy(t, rule)
	if !policy.Evaluate(EnforceSearch, Fields{Harness: sampleOllama, Model: "llama3"}).Hidden() {
		t.Fatalf("both fields match should hide")
	}
	if policy.Evaluate(EnforceSearch, Fields{Harness: sampleOllama, Model: "qwen"}).Matched {
		t.Fatalf("one field mismatch must not match (AND semantics)")
	}
}

// TestImportUnobservableFields pins which import-enforced match fields can
// never fire at the import seam (model/project/cwd), so the CLI warning in
// A-P1-7 stays honest.
func TestImportUnobservableFields(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		rule Rule
		want []string
	}{
		{"harness import is observable", excludeImport(sampleOllama), nil},
		{"model import is unobservable", excludeSearch(sampleModel).withEnforce(EnforceImport), []string{"model"}},
		{"search rule reports nothing", excludeSearch(sampleModel), nil},
		{"project+cwd import unobservable", Rule{Action: ActionExclude, EnforceAt: EnforceImport, ProjectScope: "/p", CwdScope: "/c"}, []string{"project", "cwd"}},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := tc.rule.ImportUnobservableFields()
			if len(got) != len(tc.want) {
				t.Fatalf("ImportUnobservableFields() = %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("ImportUnobservableFields()[%d] = %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func (r Rule) withEnforce(point EnforcePoint) Rule {
	r.EnforceAt = point
	return r
}

func TestEvaluatePrecedenceFirstMatchWins(t *testing.T) {
	t.Parallel()
	// A narrow allow (index) rule declared before a broad exclude keeps a
	// specific conversation even when a later wildcard would exclude it.
	policy := mustPolicy(t,
		Rule{Name: "keep-mine", Action: ActionIndex, EnforceAt: EnforceSearch, Model: "keep-me"},
		Rule{Name: "hide-rest", Action: ActionExclude, EnforceAt: EnforceSearch},
	)
	keep := policy.Evaluate(EnforceSearch, Fields{Model: "keep-me"})
	if keep.Hidden() || keep.RuleIndex != 0 {
		t.Fatalf("first matching rule (allow) must win: %+v", keep)
	}
	hide := policy.Evaluate(EnforceSearch, Fields{Model: "other"})
	if !hide.Hidden() || hide.RuleIndex != 1 {
		t.Fatalf("second rule should exclude the rest: %+v", hide)
	}
}

func TestRedactIsNotHidden(t *testing.T) {
	t.Parallel()
	// Until W5, redact is a recognized, observable no-op that keeps the
	// conversation (it must never silently exclude).
	policy := mustPolicy(t, redactImport("codex"))
	decision := policy.RecordImport(Fields{Harness: "codex"})
	if !decision.Matched {
		t.Fatalf("redact rule should match")
	}
	if decision.Hidden() {
		t.Fatalf("redact must not hide the conversation (W5 deferred)")
	}
	if got := policy.Snapshot().RedactMarked; got != 1 {
		t.Fatalf("redact match should be counted, got %d", got)
	}
	if got := policy.Snapshot().ImportExcluded; got != 0 {
		t.Fatalf("redact must not increment the exclusion count, got %d", got)
	}
}

// TestDescribeRuleMarksRedactMasking pins the surface now that W5 masking is
// live: a redact rule at the import seam advertises that it masks PII before
// indexing (the run stays searchable), and a redact rule at the search seam is
// marked inert (masking runs at import). Neither carries the old "no-op" marker.
func TestDescribeRuleMarksRedactMasking(t *testing.T) {
	t.Parallel()
	importLine := DescribeRule(redactImport("codex"))
	if want := "masks PII"; !strings.Contains(importLine, want) {
		t.Fatalf("redact @ import rule should surface %q, got %q", want, importLine)
	}
	if strings.Contains(importLine, "no-op") {
		t.Fatalf("redact rule must no longer be marked a no-op, got %q", importLine)
	}
	searchLine := DescribeRule(Rule{Action: ActionRedact, EnforceAt: EnforceSearch, Model: "claude*"})
	if want := "no effect at search seam"; !strings.Contains(searchLine, want) {
		t.Fatalf("redact @ search rule should surface %q, got %q", want, searchLine)
	}
	if excludeLine := DescribeRule(excludeImport("codex")); strings.Contains(excludeLine, "no-op") || strings.Contains(excludeLine, "masks PII") {
		t.Fatalf("exclude rule must carry no redact marker, got %q", excludeLine)
	}
}

func TestDisabledPolicyKeepsEverything(t *testing.T) {
	t.Parallel()
	policy := mustPolicy(t)
	if policy.Enabled() {
		t.Fatalf("empty policy should be disabled")
	}
	if policy.Evaluate(EnforceImport, Fields{Harness: "x"}).Hidden() {
		t.Fatalf("disabled policy must keep everything")
	}
	if snap := policy.Snapshot(); snap.ImportExcluded != 0 || snap.SearchHidden != 0 || snap.RedactMarked != 0 || len(snap.ByRule) != 0 {
		t.Fatalf("disabled policy snapshot should be empty, got %+v", snap)
	}
}

func TestSnapshotCountsByRule(t *testing.T) {
	t.Parallel()
	policy := mustPolicy(t,
		Rule{Name: "no-ollama", Action: ActionExclude, EnforceAt: EnforceImport, Harness: sampleOllama},
		Rule{Name: "hide-grok", Action: ActionExclude, EnforceAt: EnforceSearch, Harness: sampleGrok},
	)
	policy.RecordImport(Fields{Harness: sampleOllama})
	policy.RecordImport(Fields{Harness: sampleOllama})
	policy.RecordSearchHide(Fields{Harness: sampleGrok})
	snap := policy.Snapshot()
	if snap.ImportExcluded != 2 || snap.SearchHidden != 1 {
		t.Fatalf("unexpected counts: %+v", snap)
	}
	if snap.ByRule["no-ollama"] != 2 || snap.ByRule["hide-grok"] != 1 {
		t.Fatalf("unexpected per-rule counts: %+v", snap.ByRule)
	}
}

func TestFileStoreRoundTrip(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "conversation-policy.json")

	// Missing file is not an error (no policy).
	doc, err := LoadDocument(path)
	if err != nil || len(doc.Rules) != 0 {
		t.Fatalf("missing file should load empty: doc=%+v err=%v", doc, err)
	}

	write := Document{Rules: []Rule{
		{Name: "local-private", Action: ActionExclude, EnforceAt: EnforceImport, Harness: sampleOllama},
	}}
	if err := SaveDocument(path, write); err != nil {
		t.Fatal(err)
	}
	read, err := LoadDocument(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(read.Rules) != 1 || read.Rules[0].Harness != sampleOllama {
		t.Fatalf("round-trip mismatch: %+v", read)
	}

	// An invalid rule must be rejected on save, leaving no partial write.
	if err := SaveDocument(path, Document{Rules: []Rule{{Harness: "x"}}}); err == nil {
		t.Fatalf("expected save of invalid rule to fail")
	}

	// A malformed file must be a load error, not a silent empty policy.
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadDocument(path); err == nil {
		t.Fatalf("expected malformed policy file to error")
	}
}

// TestMutateDocumentSerializesWrites proves the add/remove TOCTOU fix: a
// concurrent read-modify-write under MutateDocument does not lose a rule. Each
// goroutine appends a uniquely-named rule; all must survive.
func TestMutateDocumentSerializesWrites(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "conversation-policy.json")
	const writers = 8
	done := make(chan error, writers)
	for i := 0; i < writers; i++ {
		i := i
		go func() {
			done <- MutateDocument(path, func(doc Document) (Document, error) {
				doc.Rules = append(doc.Rules, Rule{Name: ruleLabel(Rule{}, i), Action: ActionExclude, EnforceAt: EnforceImport, Harness: sampleOllama})
				return doc, nil
			})
		}()
	}
	for i := 0; i < writers; i++ {
		if err := <-done; err != nil {
			t.Fatalf("concurrent MutateDocument: %v", err)
		}
	}
	doc, err := LoadDocument(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Rules) != writers {
		t.Fatalf("expected %d rules after concurrent writes, got %d (lost update)", writers, len(doc.Rules))
	}
}

func TestResolveFilePathEnvOverride(t *testing.T) {
	t.Setenv(FileEnvVar, "/tmp/custom-policy.json")
	if got := ResolveFilePath("/ignored"); got != "/tmp/custom-policy.json" {
		t.Fatalf("env override should win, got %q", got)
	}
}
