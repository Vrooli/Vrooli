package trailers

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

func keyed(entries []Entry) []string {
	var out []string
	for _, entry := range entries {
		if entry.Key != "" {
			out = append(out, entry.Key+": "+entry.Value)
		}
	}
	return out
}

func TestParseKeepsWholeFinalParagraphAndUnfoldsContinuations(t *testing.T) {
	msg := Parse("Draft summary\n\nVrooli-Plan: plan-1\nVrooli-Unknown: keep\n continuation\nReviewed-by: human")
	if msg.Subject != "Draft summary" || msg.Body != "" {
		t.Fatalf("message = %#v", msg)
	}
	want := []string{"Vrooli-Plan: plan-1", "Vrooli-Unknown: keep continuation", "Reviewed-by: human"}
	if got := keyed(msg.Entries); !reflect.DeepEqual(got, want) {
		t.Fatalf("entries = %q, want %q", got, want)
	}
	if !msg.Entries[0].Supported || msg.Entries[1].Supported || msg.Entries[2].Supported {
		t.Fatalf("supported flags = %#v", msg.Entries)
	}
}

func TestParseDoesNotTreatMalformedSeparatorAsReference(t *testing.T) {
	msg := Parse("Subject\n\nVrooli-Plan without separator")
	if len(msg.Entries) != 0 || msg.Body != "Vrooli-Plan without separator" {
		t.Fatalf("message = %#v", msg)
	}
}

// GCT-027: unknown namespaced keys and foreign trailers round-trip in order.
func TestUnknownAndForeignTrailersRoundTripInOrder(t *testing.T) {
	text := "bas: one renderer (E26)\n\nBody paragraph.\n\nVrooli-Custom-Thing: keep me\nCo-Authored-By: Someone <someone@example.com>\nVrooli-Plan: plan-1\nSigned-off-by: Operator <op@example.com>"
	msg := Parse(text)
	want := []string{"Vrooli-Custom-Thing: keep me", "Co-Authored-By: Someone <someone@example.com>", "Vrooli-Plan: plan-1", "Signed-off-by: Operator <op@example.com>"}
	if got := keyed(msg.Entries); !reflect.DeepEqual(got, want) {
		t.Fatalf("entries = %q, want %q", got, want)
	}
	if rendered := Render(msg.Subject, msg.Body, Normalize(msg.Entries)); rendered != text {
		t.Fatalf("round trip changed text:\n%s\n---\n%s", rendered, text)
	}
}

type recordingResolver struct {
	calls  []string
	status ResolutionStatus
	detail string
}

func (r *recordingResolver) Resolve(_ context.Context, key, value string) (string, ResolutionStatus, string, error) {
	r.calls = append(r.calls, key+"="+value)
	return "owner-" + value, r.status, r.detail, nil
}

// GCT-028: a legacy initiative key stays legacy and never reaches a resolver.
func TestLegacyInitiativeIsNeverMappedWithoutOwnerEvidence(t *testing.T) {
	msg := Parse("subject\n\nVrooli-Initiative: gct-commit-initiative-linking\nVrooli-Plan: plan-42")
	resolver := &recordingResolver{status: Resolved}
	refs := ResolveEntries(context.Background(), msg.Entries, resolver)
	if len(refs) != 2 || refs[0].Status != Legacy || refs[0].OwnerID != "" || refs[0].Kind != "initiative" {
		t.Fatalf("legacy ref = %#v", refs)
	}
	if !reflect.DeepEqual(resolver.calls, []string{"Vrooli-Plan=plan-42"}) {
		t.Fatalf("resolver calls = %q", resolver.calls)
	}
	issues := Validate(msg.Subject, msg.Body, msg.Entries, ValidateOptions{AgentAuthored: true})
	if HasErrors(issues) || len(issues) != 1 || issues[0].Code != "legacy_key" {
		t.Fatalf("issues = %#v", issues)
	}
}

// GCT-029: duplicates parse as written, exact duplicates collapse on
// normalization, folded values unfold to one space and keep their folding on
// re-render, and multiplicity limits are validation errors.
func TestDuplicateAndMultilineTrailerBehavior(t *testing.T) {
	run := "e8c322ca-72e7-434c-9d37-1d2cf853961f"
	text := "subject\n\nVrooli-Run: " + run + "\nReviewed-by: Alice\n  <alice@example.com>\nVrooli-Run: " + run + "\nVrooli-Continues: abcdef1\nVrooli-Continues: 1234567"
	msg := Parse(text)
	if len(msg.Entries) != 5 || msg.Entries[1].Value != "Alice <alice@example.com>" {
		t.Fatalf("entries = %#v", msg.Entries)
	}
	if rendered := Render(msg.Subject, msg.Body, msg.Entries); rendered != text {
		t.Fatalf("folding lost:\n%s", rendered)
	}
	normalized := Normalize(msg.Entries)
	want := []string{"Vrooli-Run: " + run, "Reviewed-by: Alice <alice@example.com>", "Vrooli-Continues: abcdef1", "Vrooli-Continues: 1234567"}
	if got := keyed(normalized); !reflect.DeepEqual(got, want) {
		t.Fatalf("normalized = %q, want %q", got, want)
	}
	issues := Validate(msg.Subject, msg.Body, normalized, ValidateOptions{})
	if len(issues) != 1 || issues[0].Code != "multiplicity" || issues[0].Index != 3 {
		t.Fatalf("issues = %#v", issues)
	}
}

// GCT-030: private or deleted items stay explicit and never carry an owner
// title.
func TestPrivateOrDeletedReferencesWithholdOwnerText(t *testing.T) {
	msg := Parse("subject\n\nVrooli-Plan: private-plan")
	for _, status := range []ResolutionStatus{Inaccessible, Deleted} {
		refs := ResolveEntries(context.Background(), msg.Entries, &recordingResolver{status: status, detail: "Secret Plan Title"})
		if refs[0].Status != status || strings.Contains(refs[0].Detail, "Secret") || refs[0].OwnerID != "" || refs[0].Confidence != "asserted-metadata" {
			t.Fatalf("%s ref = %#v", status, refs[0])
		}
	}
}

func TestMidBodyVrooliLineIsBodyText(t *testing.T) {
	msg := Parse("subject\n\nVrooli-Plan: not-a-trailer\nprose continues here\n\nFinal prose paragraph.")
	if len(msg.Entries) != 0 || !strings.Contains(msg.Body, "Vrooli-Plan: not-a-trailer") {
		t.Fatalf("message = %#v", msg)
	}
	msg = Parse("subject\n\nVrooli-Plan: in-body\n\nprose\n\nVrooli-Plan: plan-1")
	if got := keyed(msg.Entries); !reflect.DeepEqual(got, []string{"Vrooli-Plan: plan-1"}) || !strings.Contains(msg.Body, "in-body") {
		t.Fatalf("message = %#v", msg)
	}
}

// Parity fixtures: the expected lists are what `git interpret-trailers
// --parse` prints for each input (authored from git's trailer rules, stored
// as static data; tests never run git).
func TestParseMatchesGitInterpretTrailersParse(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  []string
	}{
		{"plain block", "subject\n\nbody\n\nKey: value\nOther-Key:value2\n", []string{"Key: value", "Other-Key: value2"}},
		{"space before separator", "subject\n\nKey : spaced", []string{"Key: spaced"}},
		{"title is never trailers", "Key: only a title", nil},
		{"mixed block without git prefix", "subject\n\nNot a trailer line\nKey: v", nil},
		{"mixed block with git prefix", "subject\n\nSigned-off-by: A <a@x>\n(cherry picked from commit abc)\nfree text", []string{"Signed-off-by: A <a@x>"}},
		{"continuations unfold", "subject\n\nKey: line one\n line two\n\tline three", []string{"Key: line one line two line three"}},
		{"token with inner space is not a trailer", "subject\n\nTwo Words: no", nil},
		{"crlf", "subject\r\n\r\nKey: crlf\r\n", []string{"Key: crlf"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := keyed(Parse(tc.input).Entries); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestKeysMatchCaseInsensitivelyAndRenderCanonical(t *testing.T) {
	msg := Parse("subject\n\nvrooli-epoch: effort:bas#E27")
	if msg.Entries[0].Key != "Vrooli-Epoch" || !msg.Entries[0].Supported {
		t.Fatalf("entry = %#v", msg.Entries[0])
	}
	if rendered := Render(msg.Subject, msg.Body, msg.Entries); !strings.HasSuffix(rendered, "\nVrooli-Epoch: effort:bas#E27") {
		t.Fatalf("rendered = %q", rendered)
	}
}

func TestRenderedWorkTrailersParseBackIdentically(t *testing.T) {
	work := Work{
		EffortRef: "effort:browser-automation-studio-rehabilitation", EffortRevision: "9", Epoch: "e26",
		RunIDs:     []string{"E8C322CA-72E7-434C-9D37-1D2CF853961F", "70dd81be-0c15-4ef3-9b05-e18aa6ca7ccb"},
		Plans:      []string{"git-control-tower-advisory-maturity-20260905#phase-6"},
		ProposalID: "gctp-3f9a1c2b7d4e",
	}
	entries := Normalize(append(WorkEntries(work), Entry{Key: "Signed-off-by", Value: "Operator <op@example.com>"}))
	body := "Standalone replay archives use one renderer.\n\nGates: api pass; ui pass."
	text := Render("bas: one ReplaySpec renderer (E26)", body, entries)
	msg := Parse(text)
	if msg.Subject != "bas: one ReplaySpec renderer (E26)" || msg.Body != body {
		t.Fatalf("message = %#v", msg)
	}
	want := []string{
		"Vrooli-Effort: effort:browser-automation-studio-rehabilitation@9",
		"Vrooli-Epoch: effort:browser-automation-studio-rehabilitation#E26",
		"Vrooli-Run: e8c322ca-72e7-434c-9d37-1d2cf853961f",
		"Vrooli-Run: 70dd81be-0c15-4ef3-9b05-e18aa6ca7ccb",
		"Vrooli-Plan: git-control-tower-advisory-maturity-20260905#phase-6",
		"Vrooli-Proposal: gctp-3f9a1c2b7d4e",
		"Signed-off-by: Operator <op@example.com>",
	}
	if got := keyed(msg.Entries); !reflect.DeepEqual(got, want) {
		t.Fatalf("entries = %q", got)
	}
	if issues := Validate(msg.Subject, msg.Body, msg.Entries, ValidateOptions{AgentAuthored: true}); len(issues) != 0 {
		t.Fatalf("issues = %#v", issues)
	}
}

func TestValidateValueGrammarPerKey(t *testing.T) {
	cases := []struct {
		key, value string
		valid      bool
	}{
		{"Vrooli-Effort", "effort:bas", true},
		{"Vrooli-Effort", "effort:bas@12", true},
		{"Vrooli-Effort", "bas", false},
		{"Vrooli-Epoch", "effort:bas#E27", true},
		{"Vrooli-Epoch", "E27", false},
		{"Vrooli-Epoch", "#E27", false},
		{"Vrooli-Run", "70dd81be-0c15-4ef3-9b05-e18aa6ca7ccb", true},
		{"Vrooli-Run", "70dd81be", false},
		{"Vrooli-Plan", "my-plan#phase-6", true},
		{"Vrooli-Plan", "my plan", false},
		{"Vrooli-Backlog", "bug-inbox/gct-timeout", true},
		{"Vrooli-Backlog", "gct-timeout", false},
		{"Vrooli-Continues", "0e1f1210438", true},
		{"Vrooli-Continues", "HEAD~1", false},
		{"Vrooli-Proposal", "gctp-3f9a1c2b7d4e", true},
		{"Vrooli-Proposal", "proposal-1", false},
		{"Vrooli-Work", "incident inc-7@2", true},
		{"Vrooli-Work", "effort:bas", false},
		{"Vrooli-Work", "Incident inc-7", false},
	}
	for _, tc := range cases {
		issues := Validate("subject", "", []Entry{{Key: tc.key, Value: tc.value, Supported: true}}, ValidateOptions{})
		if HasErrors(issues) == tc.valid {
			t.Errorf("%s: %q valid=%v issues=%#v", tc.key, tc.value, tc.valid, issues)
		}
	}
	long := Validate("subject", "", []Entry{{Key: "Reviewed-by", Value: strings.Repeat("x", MaxValueLength+1)}}, ValidateOptions{})
	if len(long) != 1 || long[0].Code != "length" {
		t.Fatalf("length issues = %#v", long)
	}
	if kind := KindOf("Vrooli-Work", "incident inc-7@2"); kind != "incident" {
		t.Fatalf("work kind = %q", kind)
	}
}

func TestValidateSubjectAndAgentCoAuthor(t *testing.T) {
	coAuthor := []Entry{{Key: "Co-Authored-By", Value: "Agent <agent@example.com>"}}
	agentIssues := Validate("", "", coAuthor, ValidateOptions{AgentAuthored: true})
	codes := map[string]bool{}
	for _, issue := range agentIssues {
		codes[issue.Code] = true
	}
	if !codes["subject_required"] || !codes["forbidden_key"] || !HasErrors(agentIssues) {
		t.Fatalf("agent issues = %#v", agentIssues)
	}
	if operator := Validate("subject", "", coAuthor, ValidateOptions{}); len(operator) != 0 {
		t.Fatalf("operator issues = %#v", operator)
	}
	if multi := Validate("one\ntwo", "", nil, ValidateOptions{}); len(multi) != 1 || multi[0].Code != "subject_multiline" {
		t.Fatalf("multiline subject issues = %#v", multi)
	}
	if body := Validate("subject", "Gates: all pass", nil, ValidateOptions{}); len(body) != 1 || body[0].Severity != SeverityWarning {
		t.Fatalf("trailer-like body issues = %#v", body)
	}
}

type stubResolver struct{ status ResolutionStatus }

func (r stubResolver) Resolve(context.Context, string, string) (string, ResolutionStatus, string, error) {
	return "plan-42", r.status, "owner result", nil
}

func TestResolveNeverPromotesMetadataToAuthorship(t *testing.T) {
	message := Parse("subject\n\nVrooli-Plan: plan-42\nVrooli-Unknown: legacy\nSigned-off-by: Operator <op@example.com>")
	refs, err := Resolve(context.Background(), message, stubResolver{status: Resolved})
	if err != nil || len(refs) != 3 {
		t.Fatalf("refs=%#v err=%v", refs, err)
	}
	if refs[0].Status != Resolved || refs[0].Confidence != "owner-resolved-reference" {
		t.Fatalf("resolved ref=%#v", refs[0])
	}
	if refs[1].Status != Legacy || refs[1].Confidence != "asserted-metadata" {
		t.Fatalf("legacy ref=%#v", refs[1])
	}
	if refs[2].Status != NotApplicable {
		t.Fatalf("foreign ref=%#v", refs[2])
	}
}
