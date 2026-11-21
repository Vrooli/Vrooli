package measures

import (
	"testing"

	"agent-manager/internal/runannotations"
)

func TestProtoInferredAnnotationsKeepsUnprovenLabelsRateFree(t *testing.T) {
	rate := 0.42
	threshold := 0.6
	report := runannotations.CoverageResponse{
		Pack:         runannotations.PackRef{Name: "default", Version: "1"},
		EligibleRuns: 10, ObservedRuns: 4, Coverage: 0.4,
		Labels: []runannotations.CoverageLabel{
			{ID: "agent.spinning", Group: "agent", State: "observed", Observed: 2, Eligible: 4, Rate: &rate, MinProbability: &threshold, Friction: runannotations.Friction{Enabled: true}, Gate: runannotations.GateStatus{State: "proven", Gold: 40, Precision: 0.8, Recall: 0.7, TargetPrecision: 0.8, TargetRecall: 0.7}},
			{ID: "operator.valence", Group: "operator", State: "observed", Observed: 3, Eligible: 4, Friction: runannotations.Friction{Enabled: true}, Gate: runannotations.GateStatus{State: "unmeasured", Gold: 0, TargetPrecision: 0.85, TargetRecall: 0.7}},
		},
	}
	section := protoInferredAnnotations(report)
	if section.GetCoverage() != 0.4 || section.GetEligibleRuns() != 10 || section.GetObservedRuns() != 4 {
		t.Fatalf("coverage = %+v", section)
	}
	if len(section.GetLabels()) != 2 {
		t.Fatalf("labels = %d", len(section.GetLabels()))
	}
	proven := section.GetLabels()[0]
	if proven.GetRate() != 0.42 || proven.GetCoverage() != 0.5 || proven.GetMinProbability() != 0.6 || !proven.GetFriction() {
		t.Fatalf("proven row = %+v", proven)
	}
	unproven := section.GetLabels()[1]
	if unproven.Rate != nil {
		t.Fatalf("an unmeasured gate carried a rate: %+v", unproven)
	}
	if unproven.GetGateState() != "unmeasured" || unproven.GetTargetPrecision() != 0.85 {
		t.Fatalf("unproven row = %+v", unproven)
	}
}

func TestProtoInferredAnnotationsStatesAnEmptyWindowAsUnknown(t *testing.T) {
	section := protoInferredAnnotations(runannotations.CoverageResponse{})
	if section.GetNote() == "" {
		t.Fatal("an empty coverage report had no note")
	}
	if len(section.GetLabels()) != 0 {
		t.Fatalf("empty report carried labels: %+v", section.GetLabels())
	}
}
