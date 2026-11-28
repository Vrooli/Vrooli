package condition

import "testing"

func TestEvaluateTrustConservativeVocabulary(t *testing.T) {
	cases := []struct {
		name  string
		input TrustInput
		want  TrustVerdict
	}{
		{"valid", TrustInput{Available: true, UnitMatches: true}, TrustValid},
		{"ghost", TrustInput{Available: true, Ghost: true, UnitMatches: true}, TrustGhost},
		{"saturated", TrustInput{Available: true, Saturated: true, UnitMatches: true}, TrustSaturated},
		{"shelved", TrustInput{Available: true, Shelved: true, UnitMatches: true}, TrustShelved},
		{"unit mismatch", TrustInput{Available: true, UnitMatches: false}, TrustUnitMismatch},
		{"unavailable", TrustInput{Available: false, UnitMatches: true}, TrustUnavailable},
		{"unknown", TrustInput{Available: true, UnitMatches: true, VerdictToken: "future-token"}, TrustUntrusted},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := EvaluateTrust(tc.input); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestEvaluateBandNeverBandsUntrustedReadings(t *testing.T) {
	min := 10.0
	if got := EvaluateBand(12, TrustGhost, Band{Min: &min, SustainSatisfied: true}); got != BandNotEvaluated {
		t.Fatalf("ghost got %q", got)
	}
	if got := EvaluateBand(8, TrustValid, Band{Min: &min, SustainSatisfied: false}); got != BandPendingSustain {
		t.Fatalf("pending got %q", got)
	}
	if got := EvaluateBand(8, TrustValid, Band{Min: &min, SustainSatisfied: true}); got != BandOutOfBand {
		t.Fatalf("out of band got %q", got)
	}
	if got := EvaluateBand(12, TrustValid, Band{Min: &min, SustainSatisfied: true}); got != BandInBand {
		t.Fatalf("in band got %q", got)
	}
	if got := EvaluateBand(12, TrustValid, Band{Min: &min, NeedsBaseline: true}); got != BandNeedsBaseline {
		t.Fatalf("baseline got %q", got)
	}
}
