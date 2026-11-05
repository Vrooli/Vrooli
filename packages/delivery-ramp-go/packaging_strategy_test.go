package deliveryramp

import "testing"

func TestPackagingStrategyEligibilityAndSelection(t *testing.T) {
	registry, err := NewPackagingStrategyRegistry([]PackagingStrategy{
		{ID: "desktop/electron-sidecar", Version: "1", Ramp: "desktop", Default: true, Platforms: []string{"linux-x64"}},
		{ID: "desktop/go-inprocess", Version: "1", Ramp: "desktop", Predicates: []EligibilityPredicate{{Kind: "build.kind", Role: "api", Value: "go_module"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	facts := StackFacts{Components: []StackComponent{{Role: "api", BuildKind: "go_module"}}}
	eligible := registry.Evaluate(facts, "linux-x64")
	if len(eligible) != 2 || !eligible[1].Eligible {
		t.Fatalf("eligibility=%+v", eligible)
	}
	selection, err := EvaluateSelection(PackagingSelectionRequest{Policy: SelectionPolicy{Version: "selection-policy.v1", DefaultStrategy: "desktop/electron-sidecar"}, Eligible: eligible, Override: "desktop/go-inprocess"})
	if err != nil {
		t.Fatal(err)
	}
	if selection.StrategyID != "desktop/go-inprocess" || !selection.Override {
		t.Fatalf("selection=%+v", selection)
	}
}

func TestGoOnlyPredicateDoesNotExcludeOtherStrategies(t *testing.T) {
	registry, err := NewPackagingStrategyRegistry([]PackagingStrategy{
		{ID: "go", Version: "1", Ramp: "desktop", Predicates: []EligibilityPredicate{{Kind: "build.kind", Role: "api", Value: "go_module"}}},
		{ID: "generic", Version: "1", Ramp: "desktop"},
	})
	if err != nil {
		t.Fatal(err)
	}
	eligible := registry.Evaluate(StackFacts{Components: []StackComponent{{Role: "api", BuildKind: "node"}}}, "")
	for _, result := range eligible {
		if result.Strategy.ID == "go" && result.Eligible {
			t.Fatalf("Go strategy unexpectedly eligible: %+v", result)
		}
		if result.Strategy.ID == "generic" && !result.Eligible {
			t.Fatalf("generic strategy unexpectedly ineligible: %+v", result)
		}
	}
}

func TestSelectionUsesParityReliabilityAndVersionedWeightedEvidence(t *testing.T) {
	registry, err := NewPackagingStrategyRegistry([]PackagingStrategy{{ID: "baseline", Version: "1", Ramp: "generic", Default: true}, {ID: "alternative", Version: "1", Ramp: "generic"}})
	if err != nil {
		t.Fatal(err)
	}
	eligible := registry.Evaluate(StackFacts{}, "")
	request := PackagingSelectionRequest{Policy: SelectionPolicy{Version: "selection-policy.v1", DefaultStrategy: "baseline", Weights: map[string]float64{"launch_ms": 1}, MinimumLaunchReliability: 0.99}, Eligible: eligible, Evidence: []StrategyEvidence{
		{StrategyID: "alternative", Scope: "scenario", JourneyParity: "pass", LaunchReliability: 0.98, Metrics: map[string]float64{"launch_ms": 1}},
		{StrategyID: "alternative", Scope: "scenario", JourneyParity: "fail", LaunchReliability: 1, Metrics: map[string]float64{"launch_ms": 1}},
		{StrategyID: "alternative", Scope: "stack_class", JourneyParity: "pass", LaunchReliability: 1, Metrics: map[string]float64{"launch_ms": 0.5}, EvidenceIDs: []string{"sample-1"}},
	}}
	selection, err := EvaluateSelection(request)
	if err != nil {
		t.Fatal(err)
	}
	if selection.StrategyID != "alternative" || selection.EvidenceScope != "stack_class" || len(selection.EvidenceIDs) != 1 {
		t.Fatalf("selection=%+v", selection)
	}
	request.Override = "baseline"
	selection, err = EvaluateSelection(request)
	if err != nil || selection.StrategyID != "baseline" || !selection.Override {
		t.Fatalf("override=%+v err=%v", selection, err)
	}
}
