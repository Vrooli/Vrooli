package feedback

import (
	"testing"

	"swarm-manager/internal/proposals"
)

// TestExtractProposal_LenientFormats verifies the parser tolerates the
// shapes agents actually produce: lowercase/uppercase fences, no fence,
// PROPOSAL: sentinel, and embedded JSON in surrounding prose.
func TestExtractProposal_LenientFormats(t *testing.T) {
	t.Parallel()

	wantOp := proposals.OpChangePriority
	const wantTarget = "execute/foo"

	cases := []struct {
		name string
		body string
	}{
		{
			name: "fenced lowercase json",
			body: "Done.\n```json\n{\"form\":\"mutation_list\",\"mutations\":[{\"id\":\"m1\",\"op\":\"change_priority\",\"target\":\"execute/foo\",\"priority\":9}]}\n```\n",
		},
		{
			name: "fenced uppercase JSON",
			body: "Done.\n```JSON\n{\"form\":\"mutation_list\",\"mutations\":[{\"id\":\"m1\",\"op\":\"change_priority\",\"target\":\"execute/foo\",\"priority\":9}]}\n```\n",
		},
		{
			name: "fenced with trailing language attributes",
			body: "```json title=\"proposal\"\n{\"form\":\"mutation_list\",\"mutations\":[{\"id\":\"m1\",\"op\":\"change_priority\",\"target\":\"execute/foo\",\"priority\":9}]}\n```\n",
		},
		{
			name: "fenced no language tag",
			body: "Plan:\n```\n{\"form\":\"mutation_list\",\"mutations\":[{\"id\":\"m1\",\"op\":\"change_priority\",\"target\":\"execute/foo\",\"priority\":9}]}\n```\n",
		},
		{
			name: "PROPOSAL sentinel",
			body: "I propose the following.\nPROPOSAL: {\"form\":\"mutation_list\",\"mutations\":[{\"id\":\"m1\",\"op\":\"change_priority\",\"target\":\"execute/foo\",\"priority\":9}]}\nThanks.",
		},
		{
			name: "PROPOSAL sentinel lowercase with newline",
			body: "proposal:\n{\"form\":\"mutation_list\",\"mutations\":[{\"id\":\"m1\",\"op\":\"change_priority\",\"target\":\"execute/foo\",\"priority\":9}]}",
		},
		{
			name: "raw json no fence no sentinel",
			body: "{\"form\":\"mutation_list\",\"mutations\":[{\"id\":\"m1\",\"op\":\"change_priority\",\"target\":\"execute/foo\",\"priority\":9}]}",
		},
		{
			name: "raw json after prose",
			body: "Here is the change I want to make:\n{\"form\":\"mutation_list\",\"mutations\":[{\"id\":\"m1\",\"op\":\"change_priority\",\"target\":\"execute/foo\",\"priority\":9}]}\nLet me know.",
		},
		{
			name: "first fenced block bogus, second valid",
			body: "Earlier draft:\n```json\n{not json}\n```\nFinal:\n```json\n{\"form\":\"mutation_list\",\"mutations\":[{\"id\":\"m1\",\"op\":\"change_priority\",\"target\":\"execute/foo\",\"priority\":9}]}\n```\n",
		},
		{
			name: "json string contains braces (no false-stop)",
			body: "```json\n{\"form\":\"mutation_list\",\"rationale\":\"the value {x} should change\",\"mutations\":[{\"id\":\"m1\",\"op\":\"change_priority\",\"target\":\"execute/foo\",\"priority\":9}]}\n```\n",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p, _ := extractProposal(tc.body)
			if p == nil {
				t.Fatalf("expected proposal, got nil for body=%q", tc.body)
			}
			if len(p.Mutations) != 1 {
				t.Fatalf("expected 1 mutation, got %d", len(p.Mutations))
			}
			if p.Mutations[0].Op != wantOp {
				t.Fatalf("op: got %q, want %q", p.Mutations[0].Op, wantOp)
			}
			if p.Mutations[0].Target != wantTarget {
				t.Fatalf("target: got %q, want %q", p.Mutations[0].Target, wantTarget)
			}
		})
	}
}

// TestExtractProposal_NoSilentFailures asserts that genuinely-broken
// inputs return nil + warnings (so the round still records the turn) and
// never panic.
func TestExtractProposal_NoSilentFailures(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		body        string
		wantWarning bool
	}{
		{
			name: "empty body",
			body: "",
		},
		{
			name: "no JSON anywhere",
			body: "Hi, no proposal this turn.",
		},
		{
			name:        "fenced block with invalid JSON",
			body:        "```json\n{not really json}\n```",
			wantWarning: true,
		},
		{
			name: "unbalanced braces",
			body: "Plan: {\"form\":\"mutation_list\",",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p, warns := extractProposal(tc.body)
			if p != nil {
				t.Fatalf("expected nil proposal, got %+v", p)
			}
			if tc.wantWarning && len(warns) == 0 {
				t.Fatal("expected at least one warning for malformed input")
			}
		})
	}
}

// TestExtractFirstBalancedJSON_RespectsStrings ensures the brace counter
// doesn't get confused by braces inside JSON string literals or escapes.
func TestExtractFirstBalancedJSON_RespectsStrings(t *testing.T) {
	t.Parallel()
	in := `prefix {"a":"}{ literal", "b":{"nested":1}, "c":"\"escaped quote\""} trailing`
	want := `{"a":"}{ literal", "b":{"nested":1}, "c":"\"escaped quote\""}`
	got := extractFirstBalancedJSON(in)
	if got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}
