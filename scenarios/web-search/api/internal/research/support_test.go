package research_test

import (
	"testing"

	research "web-search/internal/research"
)

func TestClaimSupportRequiresRetainedPassage(t *testing.T) {
	if got := research.AssessClaimSupport("release date is 2026", nil); got != research.AssessmentUnknown {
		t.Fatalf("got %q", got)
	}
	if got := research.AssessClaimSupport("release date is 2026", []string{"The release date is 2026."}); got != research.AssessmentSupported {
		t.Fatalf("got %q", got)
	}
}

func TestClaimSupportRejectsUnsupportedSentenceInMixedSummary(t *testing.T) {
	got := research.AssessClaimSupport("release date is 2026. The moon is made of cheese.", []string{"The release date is 2026."})
	if got != research.AssessmentUnresolved {
		t.Fatalf("got %q, want unresolved", got)
	}
}

func TestIndependentSourcesUsePublisherIdentity(t *testing.T) {
	urls := []string{"https://www.example.org/a", "https://news.example.org/b", "https://other.net/c"}
	if got := research.IndependentPublisherCount(urls); got != 2 {
		t.Fatalf("got %d, want 2", got)
	}
}
