package backlogstatus

import "testing"

// validPhases is written out independently of the table so that inventing a
// new phase requires updating this list too — the guard cannot be satisfied by
// a typo'd phase string.
var validPhases = map[Phase]bool{
	PhaseIntake:   true,
	PhasePlanning: true,
	PhaseInFlight: true,
	PhaseReview:   true,
	PhaseTerminal: true,
}

// A status that reaches the table without a complete classification would fall
// into some consumer's `default` branch and behave as whatever that branch
// happens to do. This is the guard that makes adding a status a decision
// rather than an oversight.
func TestEveryStatusIsClassified(t *testing.T) {
	if len(definitions) == 0 {
		t.Fatal("the status table is empty")
	}
	seen := make(map[string]bool, len(definitions))
	for _, d := range definitions {
		if d.Value == "" {
			t.Error("a definition has an empty Value")
			continue
		}
		if seen[d.Value] {
			t.Errorf("%s: duplicate entry in the table", d.Value)
		}
		seen[d.Value] = true

		if d.Label == "" {
			t.Errorf("%s: missing Label — CLI and UI render this", d.Value)
		}
		if d.Doc == "" {
			t.Errorf("%s: missing Doc — say what it means and why it is classified this way", d.Value)
		}
		if !validPhases[d.Phase] {
			t.Errorf("%s: phase %q is not one of the declared phases", d.Value, d.Phase)
		}
	}
}

// Every status must be reachable through the phase partition, so that code
// switching on phase covers the whole vocabulary.
func TestPhasesPartitionTheVocabulary(t *testing.T) {
	covered := make(map[string]bool, len(definitions))
	for phase := range validPhases {
		for _, s := range InPhase(phase) {
			if covered[s] {
				t.Errorf("%s appears in more than one phase", s)
			}
			covered[s] = true
		}
	}
	for _, s := range All() {
		if !covered[s] {
			t.Errorf("%s belongs to no phase", s)
		}
	}
	if len(covered) != len(All()) {
		t.Errorf("phase partition covers %d statuses, vocabulary has %d", len(covered), len(All()))
	}
}

// Resolved is the dependency-gate axis, so a wrong answer here silently either
// strands dependents forever or releases them onto a prerequisite that never
// landed. Pinning the full expected set makes that a deliberate edit.
func TestResolvedSetIsPinned(t *testing.T) {
	want := map[string]bool{
		Completed: true,
		Dropped:   true,
	}
	for _, s := range All() {
		if got := IsResolved(s); got != want[s] {
			t.Errorf("IsResolved(%s) = %v, want %v — this axis decides whether dependents stay blocked",
				s, got, want[s])
		}
	}
}

// A status that is both operator-writable and owned by the execution system
// would let a PATCH fabricate run state.
func TestExecutionOwnedStatusesAreNotUserSettable(t *testing.T) {
	for _, s := range All() {
		if IsInFlight(s) && IsUserSettable(s) {
			t.Errorf("%s is execution-owned but user-settable — a PATCH could fake a run", s)
		}
		if IsReview(s) && IsUserSettable(s) {
			t.Errorf("%s is review-gated but user-settable — it must exit via review-decide", s)
		}
	}
}

// Unknown statuses must not read as classified. Map lookups return zero values,
// so a missing guard would quietly report an unknown status as "not terminal,
// not resolved" rather than as invalid.
func TestUnknownStatusIsNotClassified(t *testing.T) {
	const bogus = "banana"
	if IsValid(bogus) {
		t.Fatal("bogus status reported valid")
	}
	if _, ok := Lookup(bogus); ok {
		t.Error("Lookup reported a bogus status as known")
	}
	for name, got := range map[string]bool{
		"IsResolved":     IsResolved(bogus),
		"IsTerminal":     IsTerminal(bogus),
		"IsUserSettable": IsUserSettable(bogus),
		"IsReview":       IsReview(bogus),
		"IsInFlight":     IsInFlight(bogus),
		"IsPlanning":     IsPlanning(bogus),
	} {
		if got {
			t.Errorf("%s(%q) = true, want false", name, bogus)
		}
	}
}
