package summarize

import (
	"strings"
	"testing"
)

func TestPreserveExplicitAssignmentsRescuesMissingOwnerAndAction(t *testing.T) {
	source := "Project Atlas opens on April 12. Priya owns caption follow-up. Marcus owns legal approval."
	summary := "Project Atlas opens on April 12. Priya manages caption follow-up."

	got := PreserveExplicitAssignments(source, summary)
	if !strings.Contains(got, "Marcus owns legal approval.") {
		t.Fatalf("missing source-grounded owner assignment: %q", got)
	}
	if strings.Count(got, "Priya owns caption follow-up.") != 0 {
		t.Fatalf("already covered assignment should not be duplicated: %q", got)
	}
	if got != PreserveExplicitAssignments(source, got) {
		t.Fatalf("preservation is not idempotent: first %q, second %q", got, PreserveExplicitAssignments(source, got))
	}
}

func TestPreserveExplicitAssignmentsSupportsResponsibilityCue(t *testing.T) {
	source := "Jordan is responsible for release approval."
	got := PreserveExplicitAssignments(source, "Release approval is pending.")
	if !strings.Contains(got, source) {
		t.Fatalf("expected exact source assignment to be appended, got %q", got)
	}
}

func TestPreserveExplicitAssignmentsDoesNotCombineSeparateSummarySentences(t *testing.T) {
	const source = "Marcus owns legal approval."
	summary := "Marcus leads the review. Legal approval is pending."
	got := PreserveExplicitAssignments(source, summary)
	if !strings.Contains(got, source) {
		t.Fatalf("owner and action in separate sentences must not count as covered: %q", got)
	}
}

func TestExtractSourceAssignmentsCarriesExactOrderedSpans(t *testing.T) {
	source := "Opening.\nPriya is responsible for captions. Marcus will handle legal approval. José owns localization."
	facts := ExtractSourceAssignments(source)
	if len(facts) != 3 {
		t.Fatalf("facts = %#v, want three", facts)
	}
	for _, fact := range facts {
		if source[fact.StartByte:fact.EndByte] != fact.Sentence {
			t.Fatalf("source span %d:%d = %q, want %q", fact.StartByte, fact.EndByte, source[fact.StartByte:fact.EndByte], fact.Sentence)
		}
	}
	if facts[0].Actor != "Priya" || facts[1].Actor != "Marcus" || facts[2].Actor != "José" {
		t.Fatalf("actors = %q, %q, %q", facts[0].Actor, facts[1].Actor, facts[2].Actor)
	}
	got := PreserveSourceAssignments(facts, "Summary.")
	if strings.Index(got, facts[0].Sentence) > strings.Index(got, facts[1].Sentence) {
		t.Fatalf("rescued assignments lost source order: %q", got)
	}
}

func TestPreserveExplicitAssignmentsDeduplicatesAndIgnoresNegation(t *testing.T) {
	source := "Marcus owns legal approval. Marcus owns legal approval. Jordan is not responsible for the budget."
	got := PreserveExplicitAssignments(source, "A concise summary.")
	if strings.Count(got, "Marcus owns legal approval.") != 1 {
		t.Fatalf("duplicate assignment rescue: %q", got)
	}
	if strings.Contains(got, "Jordan is not responsible") {
		t.Fatalf("negated responsibility was treated as an assignment: %q", got)
	}
}

func TestPreserveExplicitAssignmentsIgnoresNonAssignmentSentences(t *testing.T) {
	source := "The host is Denver. The deadline is March 29. The budget is $18,400."
	const summary = "Denver hosts the event, with a March 29 deadline and $18,400 budget."
	if got := PreserveExplicitAssignments(source, summary); got != summary {
		t.Fatalf("unexpected change for source without assignment cues: %q", got)
	}
}

func TestSplitSourceSentencesKeepsPunctuation(t *testing.T) {
	got := splitSourceSentences("One. Two!\nThree? Four")
	want := []string{"One.", "Two!", "Three?", "Four"}
	if len(got) != len(want) {
		t.Fatalf("sentences = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("sentences = %#v, want %#v", got, want)
		}
	}
}
