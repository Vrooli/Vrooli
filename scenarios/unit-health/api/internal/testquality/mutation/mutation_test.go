package mutation

import (
	"testing"
	"unit-health/internal/testquality"
)

func TestClassifyMutationOutcomes(t *testing.T) {
	base := Experiment{MutationID: "m", SourceIdentity: "s", TestIdentity: "t", InContract: true, CompileOK: true, BehaviorChanged: true, EvidenceAvailable: true}
	cases := []struct {
		name string
		edit func(*Experiment)
		want testquality.MutationDisposition
	}{
		{"killed", func(e *Experiment) { e.TestFailed = true }, testquality.MutationKilled},
		{"survived", func(*Experiment) {}, testquality.MutationSurvived},
		{"invalid", func(e *Experiment) { e.CompileOK = false }, testquality.MutationInvalid},
		{"equivalent", func(e *Experiment) { e.BehaviorChanged = false }, testquality.MutationEquivalent},
		{"out of contract", func(e *Experiment) { e.InContract = false }, testquality.MutationOutOfContract},
		{"infrastructure", func(e *Experiment) { e.InfrastructureError = true }, testquality.MutationInfrastructureFailure},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := base
			tc.edit(&e)
			got, err := Classify(e)
			if err != nil || got != tc.want {
				t.Fatalf("got %q err=%v want %q", got, err, tc.want)
			}
		})
	}
}

func TestClassifyMutationPreservesUnknownEvidence(t *testing.T) {
	got, err := Classify(Experiment{MutationID: "m", SourceIdentity: "s", TestIdentity: "t"})
	if err != nil || got != testquality.MutationUnknown {
		t.Fatalf("got %q err=%v", got, err)
	}
}
