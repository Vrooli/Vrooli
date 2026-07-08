package promptcatalog

import (
	"strings"
	"testing"
)

// TestBacklogSyncProposalSnippet_NotEmpty pins the basic load-bearing
// invariants on the shared snippet: it is non-empty, mentions the
// mutation_list form (the one the apply path actually accepts), and shows
// the required `id`, `rationale`, and fenced JSON block conventions every
// downstream prompt depends on.
func TestBacklogSyncProposalSnippet_NotEmpty(t *testing.T) {
	snippet := BacklogSyncProposalSnippet()
	if strings.TrimSpace(snippet) == "" {
		t.Fatal("BacklogSyncProposalSnippet() returned empty")
	}
	mustContain := []string{
		"mutation_list",
		"```json",
		"\"id\":",
		"\"op\":",
		"\"rationale\":",
		"BacklogSyncPolicy",
	}
	for _, want := range mustContain {
		if !strings.Contains(snippet, want) {
			t.Errorf("BacklogSyncProposalSnippet missing required content %q", want)
		}
	}
}

// TestBacklogSyncProposalVariableKey_Stable pins the template variable name.
// Renaming this key requires coordinated edits in every consuming SKILL.md;
// the test is a tripwire so the rename is intentional, not accidental.
func TestBacklogSyncProposalVariableKey_Stable(t *testing.T) {
	if got, want := BacklogSyncProposalVariableKey, "BACKLOG_SYNC_PROPOSAL_SNIPPET"; got != want {
		t.Fatalf("BacklogSyncProposalVariableKey = %q, want %q (consumers reference this string in SKILL.md templates)", got, want)
	}
}

// TestElasticSliceSnippet_Contract pins the load-bearing content of the
// elastic-slice contract: it names the comprehensively-completable unit, the
// frontier field the handoff carries, and the "remainder of one" case that is
// how a too-large phase is handled without failure.
func TestElasticSliceSnippet_Contract(t *testing.T) {
	snippet := ElasticSliceSnippet()
	if strings.TrimSpace(snippet) == "" {
		t.Fatal("ElasticSliceSnippet() returned empty")
	}
	mustContain := []string{
		"comprehensively-completable",
		"frontier",
		"handoff.frontier",
		"remainder",
	}
	for _, want := range mustContain {
		if !strings.Contains(snippet, want) {
			t.Errorf("ElasticSliceSnippet missing required content %q", want)
		}
	}
}

// TestElasticSliceVariableKey_Stable pins the template variable name the two
// execute SKILL.md templates reference. Renaming it silently drops the slice
// contract from the rendered prompt, so the rename must be intentional.
func TestElasticSliceVariableKey_Stable(t *testing.T) {
	if got, want := ElasticSliceVariableKey, "ELASTIC_SLICE_SNIPPET"; got != want {
		t.Fatalf("ElasticSliceVariableKey = %q, want %q (execute SKILL.md templates reference this string)", got, want)
	}
}
