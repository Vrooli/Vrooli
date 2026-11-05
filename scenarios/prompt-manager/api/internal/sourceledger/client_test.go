package sourceledger

import (
	"testing"
)

func TestTeamScopeFacetsUsesSemanticVocabularyAndFitsBothBudgets(t *testing.T) {
	facets := TeamScopeFacets("team:director-swarm")
	if len(facets) != 6 {
		t.Fatalf("got %d facets, want 6", len(facets))
	}

	validRetention := map[string]bool{
		"retain": true, "compact": true, "expire-on-resolution": true, "pinned-or-review": true,
	}
	var residency int32
	bySuffix := make(map[string]bool, len(facets))
	for _, facet := range facets {
		if !validRetention[facet.GetRetentionPolicy()] {
			t.Errorf("facet %q has invalid retention policy %q", facet.GetId(), facet.GetRetentionPolicy())
		}
		if facet.GetGuidance() == "" {
			t.Errorf("facet %q has empty guidance", facet.GetId())
		}
		residency += facet.GetResidentBudget()
		bySuffix[facet.GetId()] = true
	}
	if residency != 28 {
		t.Errorf("residency sum = %d, want 28", residency)
	}
	if residency > int32(96/2) {
		t.Errorf("residency sum %d exceeds line capacity %d", residency, 96/2)
	}
	if residency > int32(12000/200) {
		t.Errorf("residency sum %d exceeds character capacity %d", residency, 12000/200)
	}
	for _, suffix := range []string{"standing-lesson", "decision", "episode", "handoff", "thread", "rehearsal"} {
		if !bySuffix["prompt-manager-director-swarm-"+suffix] {
			t.Errorf("missing facet suffix %q", suffix)
		}
	}

	for _, facet := range facets {
		switch facet.GetId() {
		case "prompt-manager-director-swarm-standing-lesson", "prompt-manager-director-swarm-decision":
			if facet.GetCompactionEligible() {
				t.Errorf("facet %q must not be compaction eligible", facet.GetId())
			}
		case "prompt-manager-director-swarm-rehearsal":
			if facet.GetResidentBudget() != 0 {
				t.Errorf("rehearsal resident budget = %d, want 0", facet.GetResidentBudget())
			}
		}
	}
}

func TestValidRetentionPolicyKeepsSecondVocabularyEnsureIdempotent(t *testing.T) {
	for _, value := range []string{"retain", "compact", "expire-on-resolution", "pinned-or-review"} {
		if !validRetentionPolicy(value) {
			t.Errorf("valid retention policy %q rejected", value)
		}
	}
	for _, value := range []string{"", "expire_on_resolution", "pin_or_review"} {
		if validRetentionPolicy(value) {
			t.Errorf("invalid retention policy %q accepted", value)
		}
	}
}
