package manifest_test

import (
	"testing"

	"architecture-cartographer/internal/manifest"
)

func TestDefaultSignalWeights_MatchesLadder(t *testing.T) {
	w := manifest.DefaultSignalWeights()
	if w["path-token"] != 1.5 {
		t.Fatalf("path-token: %v", w["path-token"])
	}
	if w["import-cluster"] != 1.0 {
		t.Fatalf("import-cluster: %v", w["import-cluster"])
	}
	if w["symbol-glossary"] != 0.9 {
		t.Fatalf("symbol-glossary: %v", w["symbol-glossary"])
	}
	if w["importer-voting"] != 0.8 {
		t.Fatalf("importer-voting: %v", w["importer-voting"])
	}
	if w["test-coupling"] != 0.7 {
		t.Fatalf("test-coupling: %v", w["test-coupling"])
	}
	if w["git-co-edit"] != 0.6 {
		t.Fatalf("git-co-edit: %v", w["git-co-edit"])
	}
}

func TestDefaultSignalWeights_FreshMapPerCall(t *testing.T) {
	a := manifest.DefaultSignalWeights()
	a["path-token"] = 99
	b := manifest.DefaultSignalWeights()
	if b["path-token"] != 1.5 {
		t.Fatalf("mutation leaked across calls: %v", b["path-token"])
	}
}

func TestEffectiveWeights_LayersInOrder(t *testing.T) {
	m := manifest.ManifestDefinition{
		SignalWeights: manifest.SignalWeights{Weights: map[string]float64{
			"path-token": 1.7, // manifest-level
		}},
		Domains: []manifest.DomainSpec{
			{Name: "graph", SignalWeightOverrides: manifest.SignalWeights{Weights: map[string]float64{
				"path-token":     2.5, // domain-level wins
				"import-cluster": 0.0, // disable
			}}},
		},
	}
	wAll := manifest.EffectiveWeights(m, "")
	if wAll["path-token"] != 1.7 {
		t.Fatalf("manifest overlay missing: %v", wAll["path-token"])
	}
	if wAll["import-cluster"] != 1.0 {
		t.Fatalf("non-overridden default should remain: %v", wAll["import-cluster"])
	}

	wGraph := manifest.EffectiveWeights(m, "graph")
	if wGraph["path-token"] != 2.5 {
		t.Fatalf("domain overlay missing: %v", wGraph["path-token"])
	}
	if wGraph["import-cluster"] != 0 {
		t.Fatalf("domain disable not applied: %v", wGraph["import-cluster"])
	}
	if wGraph["test-coupling"] != 0.7 {
		t.Fatalf("non-overridden inherits default: %v", wGraph["test-coupling"])
	}
}

func TestEffectiveThresholds_MergesAndSorts(t *testing.T) {
	m := manifest.ManifestDefinition{Thresholds: []manifest.Threshold{
		{Tier: "auto_place", MinValue: 0.9}, // override
		{Tier: "custom", MinValue: 0.3},
	}}
	got := manifest.EffectiveThresholds(m)
	if len(got) != 3 {
		t.Fatalf("expected 3 tiers, got %+v", got)
	}
	if got[0].Tier != "auto_place" || got[0].MinValue != 0.9 {
		t.Fatalf("expected auto_place first with 0.9, got %+v", got[0])
	}
	if got[1].Tier != "suggest" || got[1].MinValue != 0.55 {
		t.Fatalf("expected suggest second with 0.55, got %+v", got[1])
	}
	if got[2].Tier != "custom" {
		t.Fatalf("expected custom third, got %+v", got[2])
	}
}
