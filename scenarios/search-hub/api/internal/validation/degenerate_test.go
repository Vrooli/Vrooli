package validation

import (
	"strings"
	"testing"

	aisearch "github.com/vrooli/ai-go/search"
)

func TestDegenerateCorpusDetectionIsScenarioAgnostic(t *testing.T) {
	suite := &aisearch.TestSuite{
		Cases: []aisearch.TestCase{
			{ID: "token", Query: "retry", ExpectIDs: []string{"pkg/retry"}},
			{ID: "question", Query: "where is retry handled", ExpectIDs: []string{"pkg/retry"}},
			{ID: "candidate", Query: "api", Status: aisearch.CaseStatusCandidate, ExpectIDs: []string{"pkg/api"}},
		},
	}
	got := degenerateCorpusCases(suite.Cases)
	if len(got) != 1 || got[0] != "token" {
		t.Fatalf("degenerate cases = %#v, want only token", got)
	}

	groups := &aisearch.TestSuite{
		Cases: []aisearch.TestCase{
			{ID: "one", Query: "one", Tags: []string{"location"}, ExpectIDs: []string{"one"}},
		},
		Coverage: aisearch.CoverageConfig{RequiredTagGroups: []aisearch.CoverageTagGroup{
			{ID: "location", Tags: []string{"location"}},
			{ID: "ownership", Tags: []string{"ownership"}},
		}},
	}
	if got := declaredFamilyCount(groups); got != 1 {
		t.Fatalf("declared family count = %d, want 1", got)
	}
	if strings.Contains("SEARCH_EVAL_CORPUS_DEGENERATE", "code-facts") {
		t.Fatal("degenerate rule must not contain a scenario-specific identifier")
	}
}
