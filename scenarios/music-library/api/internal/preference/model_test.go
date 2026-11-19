package preference

import (
	"math"
	"testing"
)

func TestPairwiseFitKeepsChoiceAndLearnsSupportedDirection(t *testing.T) {
	a := Candidate{ID: "a", Features: []Feature{{Attribute: "hook", Value: 1, Confidence: 1, Source: "analysis:aa"}}}
	b := Candidate{ID: "b", Features: []Feature{{Attribute: "hook", Value: -1, Confidence: 1, Source: "analysis:bb"}}}
	p := NewProfile()
	p.Judgments = []Judgment{{LeftID: "a", RightID: "b", Choice: ChooseLeft, Context: "focus", Source: "operator", At: "2026-11-15T00:00:00Z"}}
	p = FitPairwise(p, []Candidate{a, b}, 30)
	if p.Version != ModelVersion || p.Weights["hook"].Mean <= 0 {
		t.Fatalf("expected learned positive hook preference, got %+v", p)
	}
	if p.Weights["hook"].Variance >= 1 {
		t.Fatalf("informative comparison should reduce prior variance: %+v", p.Weights["hook"])
	}
	if p.Judgments[0].Choice != ChooseLeft || p.Judgments[0].Source != "operator" {
		t.Fatal("fit altered source judgment")
	}
}

func TestContradictoryPairwiseEvidenceStaysBoundedAndRetainsUncertainty(t *testing.T) {
	a := Candidate{ID: "a", Features: []Feature{{Attribute: "drive", Value: 1, Confidence: 1, Source: "catalog:a"}}}
	b := Candidate{ID: "b", Features: []Feature{{Attribute: "drive", Value: -1, Confidence: 1, Source: "catalog:b"}}}
	p := NewProfile()
	p.Judgments = []Judgment{
		{ID: "left", LeftID: "a", RightID: "b", Choice: ChooseLeft},
		{ID: "right", LeftID: "a", RightID: "b", Choice: ChooseRight},
	}
	p = FitPairwise(p, []Candidate{a, b}, 40)
	weight := p.Weights["drive"]
	if math.Abs(weight.Mean) > 1e-6 || weight.Variance <= 0.05 || weight.Variance >= 1 {
		t.Fatalf("contradiction should retain a bounded neutral estimate with residual uncertainty: %+v", weight)
	}
}

func TestTieAndSkipDoNotBecomePositiveLabels(t *testing.T) {
	a := Candidate{ID: "a", Features: []Feature{{Attribute: "bass", Value: 1, Confidence: 1, Source: "catalog"}}}
	b := Candidate{ID: "b", Features: []Feature{{Attribute: "bass", Value: -1, Confidence: 1, Source: "catalog"}}}
	p := NewProfile()
	p.Judgments = []Judgment{{LeftID: "a", RightID: "b", Choice: ChooseTie}, {LeftID: "a", RightID: "b", Choice: ChooseSkip}}
	p = FitPairwise(p, []Candidate{a, b}, 10)
	if math.Abs(p.Weights["bass"].Mean) > 1e-9 {
		t.Fatalf("tie/skip should not create directional preference: %+v", p.Weights["bass"])
	}
	if p.Weights["bass"].Variance >= 1 {
		t.Fatalf("tie should reduce uncertainty while skip contributes none: %+v", p.Weights["bass"])
	}
}

func TestNeitherDoesNotClaimFeatureIndifference(t *testing.T) {
	a := Candidate{ID: "a", Features: []Feature{{Attribute: "timbre", Value: 1, Confidence: 1, Source: "catalog:a"}}}
	b := Candidate{ID: "b", Features: []Feature{{Attribute: "timbre", Value: -1, Confidence: 1, Source: "catalog:b"}}}
	p := NewProfile()
	p.Judgments = []Judgment{{LeftID: "a", RightID: "b", Choice: ChooseNeither}}
	p = FitPairwise(p, []Candidate{a, b}, 40)
	if p.Weights["timbre"].Variance != 1 || p.Weights["timbre"].Mean != 0 {
		t.Fatalf("neither must not be coerced to tie or preference evidence: %+v", p.Weights["timbre"])
	}
}

func TestScoreExplanationReconstructsAndUnknownFeaturesStayUnknown(t *testing.T) {
	p := NewProfile()
	p.Weights["rhythm"] = Weight{Mean: 0.8, Variance: 0.4}
	p.Controls = []Control{{Attribute: "rhythm", Mode: ControlBoost, Value: 0.2}}
	s := ScoreCandidate(p, Candidate{ID: "x", Features: []Feature{{Attribute: "rhythm", Value: 0.5, Confidence: 0.5, Source: "decomp:1"}, {Attribute: "vocal", Value: 1, Confidence: 1}}})
	var reconstructed float64
	for _, c := range s.Contributions {
		reconstructed += c.Contribution
		if c.EvidenceSource == "" {
			t.Fatal("explanation lacks provenance")
		}
	}
	if math.Abs(reconstructed-s.Mean) > 1e-9 || len(s.Contributions) != 1 || s.Uncertainty <= 0 {
		t.Fatalf("score/explanation mismatch: %+v", s)
	}
}

func TestScoreRejectsCandidatesWithoutSourcedFeatures(t *testing.T) {
	for _, candidate := range []Candidate{
		{ID: "empty"},
		{ID: "unsourced", Features: []Feature{{Attribute: "tempo", Value: 1, Confidence: 1}}},
	} {
		if score := ScoreCandidate(NewProfile(), candidate); score.Eligible || len(score.Contributions) != 0 {
			t.Fatalf("candidate without sourced evidence was eligible: %+v", score)
		}
	}
}

func TestPairSelectorAvoidsAlreadySeenAndPreservesUncertainChoices(t *testing.T) {
	p := NewProfile()
	a := Candidate{ID: "a", Features: []Feature{{Attribute: "hook", Value: 1, Confidence: 1, Source: "catalog"}}}
	b := Candidate{ID: "b", Features: []Feature{{Attribute: "hook", Value: -1, Confidence: 1, Source: "catalog"}}}
	c := Candidate{ID: "c", Features: []Feature{{Attribute: "timbre", Value: 1, Confidence: 1, Source: "catalog"}}}
	x, y, ok := SelectPair(p, []Candidate{a, b, c}, map[string]bool{pairKey(a.ID, b.ID): true})
	if !ok || pairKey(x.ID, y.ID) == pairKey(a.ID, b.ID) {
		t.Fatalf("selector failed to offer a fresh pair: %s %s", x.ID, y.ID)
	}
}

func TestPairSelectorSkipsDuplicateOrUnattributedCatalogEvidence(t *testing.T) {
	p := NewProfile()
	a := Candidate{ID: "a", Features: []Feature{{Attribute: "hook", Value: 1, Confidence: 1, Source: "catalog:a"}}}
	duplicate := Candidate{ID: "b", Features: []Feature{{Attribute: "hook", Value: 1, Confidence: 1, Source: "catalog:b"}}}
	if _, _, ok := SelectPair(p, []Candidate{a, duplicate}, nil); ok {
		t.Fatal("selector offered a pair with identical evidence")
	}
	unknown := Candidate{ID: "c"}
	if _, _, ok := SelectPair(p, []Candidate{a, unknown}, nil); ok {
		t.Fatal("selector offered a pair with no evidence to compare")
	}
}

func TestProfileRejectsInvalidControlsAndChoices(t *testing.T) {
	p := NewProfile()
	p.Controls = []Control{{Attribute: "hook", Mode: "mystery"}}
	if p.Validate() == nil {
		t.Fatal("accepted unsupported control")
	}
	p = NewProfile()
	p.Judgments = []Judgment{{LeftID: "a", RightID: "b", Choice: "maybe"}}
	if p.Validate() == nil {
		t.Fatal("accepted unsupported choice")
	}
}
