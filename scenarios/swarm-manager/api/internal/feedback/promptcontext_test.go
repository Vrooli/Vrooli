package feedback

import (
	"strings"
	"testing"

	"swarm-manager/internal/operatingmode/promptcatalog"
)

func TestBuildPromptVariables_PopulatesAllKeys(t *testing.T) {
	vars := BuildPromptVariables(PromptInputs{
		InitiativeName:        "ui-rewrite",
		InitiativeTitle:       "UI Rewrite",
		InitiativeDescription: "Cohesive design tokens across kiosk + command center",
		CurrentGraphJSON:      `{"initiative":"ui-rewrite","nodes":[]}`,
		ItemSummaries: []ItemSummary{
			{Ref: "execute/foo", Title: "Foo", Status: "backlog", Priority: 5, Effort: "M"},
		},
		PriorRounds: []Round{{
			Number: 1, Type: RoundTypeFeedback, Status: RoundStatusApplied,
			Submission: Submission{Text: "old feedback"},
			Decision:   &Decision{Kind: DecisionAccept, AcceptedMutationIDs: []string{"m1"}},
		}},
		PriorHandoffs: []HandoffSummary{
			{Ref: "execute/foo", Source: "handoff.md", Content: "did the thing"},
		},
		ItemFolderIndex: []ItemFolderEntry{
			{Ref: "execute/foo", Path: "/tmp/executes/foo"},
		},
		Attachments: []AttachmentSummary{
			{Filename: "screenshot.png", ContentType: "image/png", SizeBytes: 1024},
		},
		ThisFeedback: "the kiosk is ugly",
	})

	required := []string{
		"INITIATIVE_NAME", "INITIATIVE_TITLE", "INITIATIVE_DESCRIPTION",
		"CURRENT_GRAPH", "ITEM_SUMMARIES", "PRIOR_FEEDBACK", "PRIOR_HANDOFFS",
		"ITEM_FOLDER_INDEX", "THIS_FEEDBACK", "ATTACHMENT_IMAGES",
	}
	for _, k := range required {
		if _, ok := vars[k]; !ok {
			t.Fatalf("missing key %q in prompt variables", k)
		}
	}
	if !strings.Contains(vars["ITEM_SUMMARIES"], "execute/foo") {
		t.Fatalf("ITEM_SUMMARIES missing item ref: %s", vars["ITEM_SUMMARIES"])
	}
	if !strings.Contains(vars["PRIOR_FEEDBACK"], "Round 001") {
		t.Fatalf("PRIOR_FEEDBACK missing round label: %s", vars["PRIOR_FEEDBACK"])
	}
	if !strings.Contains(vars["PRIOR_HANDOFFS"], "did the thing") {
		t.Fatalf("PRIOR_HANDOFFS missing content: %s", vars["PRIOR_HANDOFFS"])
	}
	if !strings.Contains(vars["ATTACHMENT_IMAGES"], "screenshot.png") {
		t.Fatalf("ATTACHMENT_IMAGES missing filename: %s", vars["ATTACHMENT_IMAGES"])
	}
	if vars["THIS_FEEDBACK"] != "the kiosk is ugly" {
		t.Fatalf("THIS_FEEDBACK = %q", vars["THIS_FEEDBACK"])
	}
	if !strings.Contains(vars["CURRENT_GRAPH"], `"ui-rewrite"`) {
		t.Fatalf("CURRENT_GRAPH not preserved: %s", vars["CURRENT_GRAPH"])
	}
}

// TestBuildPromptVariables_RendersSharedSnippet pins the load-bearing
// invariant that the feedback prompt and the operating-mode reconcile
// prompts substitute the same proposal-format snippet under the same
// template variable. If this drifts, the agent sees two different
// proposal-envelope contracts depending on which surface ran the round —
// exactly the failure mode the snippet was extracted to prevent.
func TestBuildPromptVariables_RendersSharedSnippet(t *testing.T) {
	vars := BuildPromptVariables(PromptInputs{InitiativeName: "x"})
	got, ok := vars[promptcatalog.BacklogSyncProposalVariableKey]
	if !ok {
		t.Fatalf("feedback prompt vars missing %q (the shared reconcile snippet)", promptcatalog.BacklogSyncProposalVariableKey)
	}
	if got != promptcatalog.BacklogSyncProposalSnippet() {
		t.Fatalf("feedback snippet drift: got %d chars, want %d (must match operatingmode/promptcatalog)", len(got), len(promptcatalog.BacklogSyncProposalSnippet()))
	}
}

func TestBuildPromptVariables_EmptyInputsHavePlaceholders(t *testing.T) {
	vars := BuildPromptVariables(PromptInputs{InitiativeName: "x"})
	if vars["CURRENT_GRAPH"] != "{}" {
		t.Fatalf("expected empty graph to render as {}, got %q", vars["CURRENT_GRAPH"])
	}
	if !strings.Contains(vars["ITEM_SUMMARIES"], "no items") {
		t.Fatalf("ITEM_SUMMARIES placeholder missing: %q", vars["ITEM_SUMMARIES"])
	}
	if !strings.Contains(vars["PRIOR_FEEDBACK"], "no prior feedback") {
		t.Fatalf("PRIOR_FEEDBACK placeholder missing: %q", vars["PRIOR_FEEDBACK"])
	}
	if !strings.Contains(vars["ATTACHMENT_IMAGES"], "no attachments") {
		t.Fatalf("ATTACHMENT_IMAGES placeholder missing: %q", vars["ATTACHMENT_IMAGES"])
	}
}

func TestTruncatePromptString_TrimsLongInputs(t *testing.T) {
	out := truncatePromptString(strings.Repeat("a", 100), 10)
	if len([]rune(out)) != 11 { // 10 chars + ellipsis
		t.Fatalf("expected 10 chars + ellipsis, got %d (%q)", len([]rune(out)), out)
	}
	if !strings.HasSuffix(out, "…") {
		t.Fatalf("expected trailing ellipsis, got %q", out)
	}
	short := truncatePromptString("abc", 10)
	if short != "abc" {
		t.Fatalf("expected passthrough, got %q", short)
	}
}

func TestMarshalGraphForPrompt_FallsBackOnNil(t *testing.T) {
	if got := MarshalGraphForPrompt(nil); got != "{}" {
		t.Fatalf("expected {} for nil, got %q", got)
	}
	got := MarshalGraphForPrompt(map[string]any{"k": "v"})
	if !strings.Contains(got, `"k": "v"`) {
		t.Fatalf("expected pretty-printed JSON, got %q", got)
	}
}
