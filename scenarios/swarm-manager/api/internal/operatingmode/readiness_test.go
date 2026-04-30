package operatingmode

import "testing"

func TestScoreHolisticReadiness(t *testing.T) {
	report, err := ScoreHolisticReadiness([]ReadinessDimension{
		{Key: "problem_clarity", Score: 1},
		{Key: "scope_defined", Score: 1},
		{Key: "approach_solid", Score: 0.8},
		{Key: "testable", Score: 0.8},
		{Key: "risk_awareness", Score: 0.8},
		{Key: "coupling_understood", Score: 0.8},
		{Key: "system_acceptance_defined", Score: 0.8},
	})
	if err != nil {
		t.Fatalf("ScoreHolisticReadiness: %v", err)
	}
	if !report.Ready {
		t.Fatal("expected readiness report to be ready")
	}
	if report.Dimensions[0].Label == "" {
		t.Fatal("expected labels to be defaulted")
	}
}

func TestScoreHolisticReadinessRejectsUnknownDimension(t *testing.T) {
	_, err := ScoreHolisticReadiness([]ReadinessDimension{{Key: "vibes", Score: 1}})
	if err == nil {
		t.Fatal("expected unknown dimension error")
	}
}

func TestParseProgressState(t *testing.T) {
	state, err := ParseProgressState([]byte(`{"decision":"continue","completed_phases":["phase-1"]}`))
	if err != nil {
		t.Fatalf("ParseProgressState: %v", err)
	}
	if state.Decision != ProgressContinue {
		t.Fatalf("decision = %q", state.Decision)
	}
}

func TestParseProgressStateRejectsUnknownDecision(t *testing.T) {
	_, err := ParseProgressState([]byte(`{"decision":"maybe"}`))
	if err == nil {
		t.Fatal("expected invalid decision error")
	}
}
