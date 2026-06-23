package completeness

import (
	"testing"

	scoringv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-completeness-scoring/v1/scoring"
)

// otGroup builds a composite carrying a quality/target_pass_rate metric line.
func otGroup(observed string) *scoringv1.CompositeScore {
	return &scoringv1.CompositeScore{
		Score: 51,
		Groups: []*scoringv1.ScoreGroup{
			{
				Id: "quality",
				Metrics: []*scoringv1.MetricLine{
					{Id: "requirement_pass_rate", Observed: "38 total, 0 passing (0%)"},
					{Id: "target_pass_rate", Observed: observed},
				},
			},
		},
	}
}

func TestScoreFromProto_ParsesRungBuildAndComposite(t *testing.T) {
	resp := &scoringv1.GetScoreResponse{
		Maturity: &scoringv1.MaturityHeadline{
			WorkingRung:      "R1 Safe & standards-clean",
			SatisfiedThrough: "R0 Runnable & green",
			LadderClean:      false,
			BuildPassing:     true,
		},
		Composite: otGroup("12 total, 9 passing (75%)"),
	}
	s := scoreFromProto(resp)

	if s.WorkingRung != "R1 Safe & standards-clean" {
		t.Errorf("WorkingRung = %q", s.WorkingRung)
	}
	if s.SatisfiedThrough != "R0 Runnable & green" {
		t.Errorf("SatisfiedThrough = %q", s.SatisfiedThrough)
	}
	if !s.BuildPassing {
		t.Error("BuildPassing should be true")
	}
	if s.Composite != 51 {
		t.Errorf("Composite = %d, want 51", s.Composite)
	}
	if s.OTTotal != 12 || s.OTPassing != 9 || s.OTPercentage != 75 {
		t.Errorf("OT = %d/%d/%.0f, want 12/9/75", s.OTTotal, s.OTPassing, s.OTPercentage)
	}
	if !s.OTHasTargets {
		t.Error("OTHasTargets should be true when OTTotal>0")
	}
	if !s.OTKnown {
		t.Error("OTKnown should be true with no requirements degradation")
	}
}

func TestScoreFromProto_NoTargetsIsKnownButNotHasTargets(t *testing.T) {
	resp := &scoringv1.GetScoreResponse{
		Maturity:  &scoringv1.MaturityHeadline{LadderClean: true},
		Composite: otGroup("0 total, 0 passing (0%)"),
	}
	s := scoreFromProto(resp)
	if s.OTTotal != 0 || s.OTHasTargets {
		t.Errorf("expected no targets, got total=%d hasTargets=%v", s.OTTotal, s.OTHasTargets)
	}
	if !s.OTKnown {
		t.Error("OTKnown should be true: the requirements collector contributed (0 targets declared)")
	}
	if !s.LadderClean {
		t.Error("LadderClean should be true")
	}
}

func TestScoreFromProto_RequirementsDegradationMarksOTUnknown(t *testing.T) {
	resp := &scoringv1.GetScoreResponse{
		Composite: otGroup("0 total, 0 passing (0%)"),
		Degradations: []*scoringv1.CollectorDegradation{
			{Collector: "requirements", State: "failed", Reason: "boom"},
		},
	}
	s := scoreFromProto(resp)
	if s.OTKnown {
		t.Error("OTKnown should be false when the requirements collector degraded")
	}
}

func TestScoreFromProto_MissingTargetMetricLeavesOTZero(t *testing.T) {
	resp := &scoringv1.GetScoreResponse{
		Composite: &scoringv1.CompositeScore{
			Groups: []*scoringv1.ScoreGroup{
				{Id: "quality", Metrics: []*scoringv1.MetricLine{{Id: "phase_pass_rate", Observed: "18 recorded, 8 passing (44%)"}}},
			},
		},
	}
	s := scoreFromProto(resp)
	if s.OTTotal != 0 || s.OTPercentage != 0 {
		t.Errorf("missing target_pass_rate must leave OT zero, got %d/%.0f", s.OTTotal, s.OTPercentage)
	}
}

func TestScoreFromProto_NilSafe(t *testing.T) {
	if got := scoreFromProto(nil); got != (Score{}) {
		t.Errorf("nil response must yield zero Score, got %+v", got)
	}
}
