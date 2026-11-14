package agentcatalog

import "testing"

func TestTokenWeightPolicyPreservesThresholdsAndUnknowns(t *testing.T) {
	weights := DefaultTokenWeights()
	for model, want := range map[string]float64{"gpt-6.1-sol": 10, "gpt-6.1-terra": 5, "gpt-6-luna": 1, "gpt-5.6-sol": 10} {
		got, known := TokenWeight(model, weights)
		if !known || got != want {
			t.Fatalf("%s: %g %v", model, got, known)
		}
	}
	if _, known := TokenWeight("unrecognized", weights); known {
		t.Fatal("unknown model was treated as Luna")
	}
	zero, err := ParseTokenWeights("luna=0", weights)
	if err != nil {
		t.Fatal(err)
	}
	if got, known := TokenWeight("gpt-6-luna", zero); !known || got != 0 {
		t.Fatal("explicit zero weight became unknown")
	}
	for _, raw := range []string{"sol=NaN", "luna=+Inf", "terra=-1", "bad"} {
		if _, err := ParseTokenWeights(raw, weights); err == nil {
			t.Fatalf("invalid weight accepted: %s", raw)
		}
	}
	replaced, err := ParseTokenWeights("sol=12", nil)
	if err != nil || len(replaced) != 1 {
		t.Fatal("cap replacement-map semantics changed", replaced, err)
	}
}
