package importance

import "testing"

func TestComposeNormalizesAndAppliesSystemRequiredFloor(t *testing.T) { // [REQ:EM-FLEET-001]
	report := Compose(
		[]ScenarioFact{
			{Name: "core", SystemRequired: true},
			{Name: "leaf"},
			{Name: "shared"},
		},
		[]CentralityMetric{
			{Scenario: "core", RequiredEdgeWeightedScore: 1, DistanceToCoreSeed: 0},
			{Scenario: "shared", RequiredEdgeWeightedScore: 10, DistanceToCoreSeed: 1},
			{Scenario: "leaf", RequiredEdgeWeightedScore: 0, DistanceToCoreSeed: -1},
		},
		map[string]int{"shared": 5, "leaf": 1},
		DefaultConfig(),
		nil,
	)

	if got := len(report.Scores); got != 3 {
		t.Fatalf("expected 3 scores, got %d", got)
	}
	byScenario := map[string]Score{}
	for _, score := range report.Scores {
		byScenario[score.Scenario] = score
	}
	if byScenario["core"].Score < DefaultConfig().SystemRequiredFloor {
		t.Fatalf("system required score = %v, want floor >= %v", byScenario["core"].Score, DefaultConfig().SystemRequiredFloor)
	}
	if byScenario["shared"].Components.Centrality != 1 {
		t.Fatalf("shared centrality = %v, want 1", byScenario["shared"].Components.Centrality)
	}
	if byScenario["leaf"].Components.CoreProximity != DefaultConfig().NeutralScore {
		t.Fatalf("leaf core proximity = %v, want neutral", byScenario["leaf"].Components.CoreProximity)
	}
}

func TestComposeMarksMissingInputsDegraded(t *testing.T) { // [REQ:EM-FLEET-002]
	report := Compose(
		[]ScenarioFact{{Name: "demo"}},
		nil,
		nil,
		DefaultConfig(),
		[]string{"centrality:not_configured"},
	)

	if got := len(report.Scores); got != 1 {
		t.Fatalf("expected one score, got %d", got)
	}
	score := report.Scores[0]
	if len(score.Degraded) == 0 {
		t.Fatal("expected degraded reasons")
	}
	if score.Components.Centrality != DefaultConfig().NeutralScore {
		t.Fatalf("centrality = %v, want neutral", score.Components.Centrality)
	}
	if score.Components.Recency != DefaultConfig().NeutralScore {
		t.Fatalf("recency = %v, want neutral", score.Components.Recency)
	}
}
