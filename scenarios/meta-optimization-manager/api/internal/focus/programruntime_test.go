package focus

import (
	"context"
	"errors"
	"testing"
)

type fakeProgramFrictionReader struct {
	report    ProgramFrictionReport
	condition ProgramConditionReport
	err       error
}

func (r fakeProgramFrictionReader) ReadFriction(context.Context) (ProgramFrictionReport, error) {
	return r.report, r.err
}

func (r fakeProgramFrictionReader) ReadCondition(context.Context) (ProgramConditionReport, error) {
	return r.condition, r.err
}

func TestProgramRuntimeGapSourceRanksDurableFrictionShapes(t *testing.T) {
	gaps, err := NewProgramRuntimeGapSource(fakeProgramFrictionReader{report: ProgramFrictionReport{
		Failures:   []ProgramFailureObservation{{Shape: "valueerror", Count: 3, SampleProgramID: "prog_new"}},
		Refusals:   []ProgramRefusalObservation{{BindingID: "demo/read/list", Reason: "grant missing", Count: 2}},
		Unresolved: []ProgramUnresolvedObservation{{AttemptedName: "demo/unknown/run", Count: 4}},
	}}).DerivedGaps(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(gaps) != 3 {
		t.Fatalf("gaps=%+v", gaps)
	}
	if gaps[0].Recurrence != 3 || gaps[0].EvidenceLocator != "program-runtime://programs/prog_new" {
		t.Fatalf("failure gap=%+v", gaps[0])
	}
	if gaps[1].EvidenceSource != "program-runtime" || gaps[2].Recurrence != 4 {
		t.Fatalf("friction gaps=%+v", gaps)
	}
}

func TestProgramRuntimeConditionGapSourceRanksDegradedAboveDormant(t *testing.T) {
	gaps, err := NewProgramRuntimeConditionGapSource(fakeProgramFrictionReader{condition: ProgramConditionReport{ReceiptExercise: ExerciseBasisInstrumentation{Basis: "fleet_receipt_aggregate"}, Conditions: []ProgramConditionObservation{
		{BindingID: "demo/read", Scenario: "demo", Status: "dormant", Reason: "no invocation"},
		{BindingID: "demo/write", Scenario: "demo", Status: "degraded", Reason: "failure majority"},
		{BindingID: "demo/list", Scenario: "demo", Status: "healthy"},
	}}}).DerivedGaps(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(gaps) != 2 || gaps[0].ConditionStatus != "dormant" || gaps[1].ConditionStatus != "degraded" {
		t.Fatalf("condition gaps=%+v", gaps)
	}
	if importanceWeight(gaps[1]) <= importanceWeight(gaps[0]) {
		t.Fatalf("degraded condition should outrank dormant: degraded=%v dormant=%v", importanceWeight(gaps[1]), importanceWeight(gaps[0]))
	}
	if len(gaps[0].Notes) < 2 || gaps[0].Notes[1] != "exercise_basis=fleet_receipt_aggregate" {
		t.Fatalf("condition basis notes=%v", gaps[0].Notes)
	}
}

func TestProgramRuntimeGapSourceDegradesIndependently(t *testing.T) {
	programs := NewProgramRuntimeGapSource(fakeProgramFrictionReader{err: errors.New("connection refused")})
	healthy := &fakeSource{gaps: []Gap{{ID: "trials/healthy", Axis: AxisEmpirical, Title: "trial evidence"}}}
	gaps, err := NewMultiGapSource([]NamedGapSource{{Name: "trials", Source: healthy}, {Name: "program-runtime", Source: programs}}).DerivedGaps(context.Background())
	if err == nil {
		t.Fatal("program-runtime source failure should propagate for an honest degraded focus response")
	}
	if len(gaps) != 2 || gaps[0].ID != "trials/healthy" || gaps[1].ID != "source/program-runtime/availability" {
		t.Fatalf("gaps=%+v", gaps)
	}
	if gaps[1].AvailabilityReason == "" {
		t.Fatal("availability gap has no reason")
	}
}
