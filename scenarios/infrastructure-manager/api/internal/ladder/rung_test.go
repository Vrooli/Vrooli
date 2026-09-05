package ladder

import (
	"strings"
	"testing"
)

func TestRungsAreOrderedByDependency(t *testing.T) {
	want := []Rung{RungIdentity, RungTelemetry, RungEvidence, RungControl, RungAnticipation}
	if len(Rungs) != len(want) {
		t.Fatalf("the ladder has %d rungs, want %d", len(Rungs), len(want))
	}
	for index, rung := range want {
		if Rungs[index] != rung {
			t.Fatalf("rung %d is %s, want %s", index, Rungs[index], rung)
		}
		if rung.Index() != index {
			t.Errorf("%s reports index %d, want %d", rung, rung.Index(), index)
		}
	}
	if below, ok := RungIdentity.Below(); ok {
		t.Errorf("identity reports %s beneath it; it is the bottom rung", below)
	}
	below, ok := RungAnticipation.Below()
	if !ok || below != RungControl {
		t.Errorf("anticipation reports %q beneath it, want control", below)
	}
}

func TestParseRungRejectsATokenOutsideTheVocabulary(t *testing.T) {
	if _, err := ParseRung("identity"); err != nil {
		t.Fatalf("identity was rejected: %v", err)
	}
	// An unrecognised rung must not be placed at the bottom of the ladder by
	// default: the bottom is the rung everything else depends on, so a wrong
	// guess there silently validates every rung above it.
	rung, err := ParseRung("durability")
	if err == nil {
		t.Fatalf("an unknown rung was accepted as %s", rung)
	}
	if rung.Index() != -1 {
		t.Errorf("the rejected rung reports index %d, want -1", rung.Index())
	}
}

func TestObservationSupportsOnlyMeasuredAndNotApplicable(t *testing.T) {
	supporting := map[Observation]bool{
		ObservationMeasured:      true,
		ObservationNotApplicable: true,
		ObservationUnmeasurable:  false,
		ObservationUnavailable:   false,
		ObservationBlocked:       false,
		ObservationUnread:        false,
	}
	for observation, want := range supporting {
		if got := observation.Supports(); got != want {
			t.Errorf("%s.Supports() = %v, want %v", observation, got, want)
		}
	}
}

// TestApplyDependencyBlocksEveryRungAboveABlindOne is the rung-ordering
// invariant. A device that emits a real temperature but whose identity could
// not be resolved must not report a covered telemetry rung: the reading is
// real, but the claim built on it is about a device nobody can name.
func TestApplyDependencyBlocksEveryRungAboveABlindOne(t *testing.T) {
	ordered := ApplyDependency(map[Rung]RungReading{
		RungIdentity:     {Observation: ObservationUnmeasurable, Reason: "the pci id database is absent"},
		RungTelemetry:    {Observation: ObservationMeasured},
		RungEvidence:     {Observation: ObservationMeasured},
		RungControl:      {Observation: ObservationMeasured},
		RungAnticipation: {Observation: ObservationMeasured},
	})
	if len(ordered) != len(Rungs) {
		t.Fatalf("got %d rungs, want the full ladder of %d", len(ordered), len(Rungs))
	}
	if ordered[0].Observation != ObservationUnmeasurable {
		t.Fatalf("the identity rung reports %s, want unmeasurable", ordered[0].Observation)
	}
	for _, reading := range ordered[1:] {
		if reading.Observation != ObservationBlocked {
			t.Errorf("%s reports %s above a blind identity rung, want blocked", reading.Rung, reading.Observation)
		}
		if reading.BlockedBy != RungIdentity {
			t.Errorf("%s reports blocked by %q, want identity", reading.Rung, reading.BlockedBy)
		}
		if !strings.Contains(reading.Reason, "unmeasurable") {
			t.Errorf("%s blocked reason %q does not name the blinding state", reading.Rung, reading.Reason)
		}
	}
}

// A rung that is meaningless for the device class is not blindness and must
// not block the rungs above it.
func TestApplyDependencyDoesNotBlockOnNotApplicable(t *testing.T) {
	ordered := ApplyDependency(map[Rung]RungReading{
		RungIdentity:     {Observation: ObservationMeasured},
		RungTelemetry:    {Observation: ObservationNotApplicable, Reason: "a pci bridge exposes no telemetry"},
		RungEvidence:     {Observation: ObservationMeasured},
		RungControl:      {Observation: ObservationMeasured},
		RungAnticipation: {Observation: ObservationMeasured},
	})
	for _, reading := range ordered {
		if reading.Observation == ObservationBlocked {
			t.Errorf("%s was blocked by a not-applicable rung", reading.Rung)
		}
	}
}

// A rung nobody graded is reported as unread rather than omitted, so the
// ladder is always five rungs and a missing grade is visible.
func TestApplyDependencyReportsUngradedRungsAsUnread(t *testing.T) {
	ordered := ApplyDependency(map[Rung]RungReading{
		RungIdentity: {Observation: ObservationMeasured},
	})
	if len(ordered) != len(Rungs) {
		t.Fatalf("got %d rungs, want %d", len(ordered), len(Rungs))
	}
	if ordered[1].Observation != ObservationUnread {
		t.Fatalf("the ungraded telemetry rung reports %s, want unread", ordered[1].Observation)
	}
	// Unread does not support the rungs above it, so they are blocked rather
	// than left looking gradeable.
	for _, reading := range ordered[2:] {
		if reading.Observation != ObservationBlocked {
			t.Errorf("%s reports %s above an unread rung, want blocked", reading.Rung, reading.Observation)
		}
	}
	if ordered[1].Reason == "" {
		t.Error("the unread rung carries no reason")
	}
}

// TestEveryLadderObservationIsReachable pins that the vocabulary has no dead
// token: a state nothing can ever produce is a claim the readout can never
// make, and it would quietly become the default the first time a token is
// mistyped.
func TestEveryLadderObservationIsReachable(t *testing.T) {
	seen := map[Observation]bool{}
	for _, reading := range ApplyDependency(map[Rung]RungReading{
		RungIdentity:  {Observation: ObservationMeasured},
		RungTelemetry: {Observation: ObservationNotApplicable},
		RungEvidence:  {Observation: ObservationUnavailable},
	}) {
		seen[reading.Observation] = true
	}
	for _, reading := range ApplyDependency(map[Rung]RungReading{
		RungIdentity: {Observation: ObservationUnmeasurable},
	}) {
		seen[reading.Observation] = true
	}
	for _, reading := range ApplyDependency(map[Rung]RungReading{
		RungIdentity:  {Observation: ObservationMeasured},
		RungTelemetry: {Observation: ObservationMeasured},
	}) {
		seen[reading.Observation] = true
	}
	for _, observation := range []Observation{
		ObservationMeasured, ObservationUnmeasurable, ObservationUnavailable,
		ObservationNotApplicable, ObservationBlocked, ObservationUnread,
	} {
		if !seen[observation] {
			t.Errorf("observation %s was never produced; it is a dead token", observation)
		}
	}
}
