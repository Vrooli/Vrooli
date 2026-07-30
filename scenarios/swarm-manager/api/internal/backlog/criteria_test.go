package backlog

import "testing"

func TestNormalizeCriteriaPreservesIdentityAndNeverReusesRemovedIDs(t *testing.T) {
	previous := []Criterion{{ID: "criterion-1", Gherkin: "Given a plan When reviewed Then it is visible"}, {ID: "criterion-3", Gherkin: "Given evidence When opened Then it renders"}}
	next := NormalizeCriteria(previous, []Criterion{{Gherkin: "Given evidence When opened Then it renders"}, {Gherkin: "Given a check When it passes Then evidence settles"}})
	if len(next) != 2 || next[0].ID != "criterion-3" || next[1].ID != "criterion-4" {
		t.Fatalf("criteria = %#v", next)
	}
}

func TestNormalizeCriteriaRejectsInvalidConditionsAndChecks(t *testing.T) {
	criteria := NormalizeCriteria(nil, []Criterion{{Gherkin: "not gherkin"}, {Gherkin: "Given a test When it runs Then it passes", Check: &Check{Kind: CheckKindTestGeniePhase, Scenario: "swarm-manager", Phase: "unit"}}, {Gherkin: "Given bad When it runs Then it fails", Check: &Check{Kind: "made-up"}}})
	if len(criteria) != 2 || criteria[0].Check == nil || criteria[1].Check != nil {
		t.Fatalf("criteria = %#v", criteria)
	}
}
