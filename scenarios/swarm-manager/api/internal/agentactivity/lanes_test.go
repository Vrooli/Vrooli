package agentactivity

import (
	"errors"
	"testing"
)

func TestLaneOf_PhaseKindWins(t *testing.T) {
	t.Parallel()

	// PurposeProcess defaults to LaneExecute, but an explicit phaseKind
	// (e.g. an operating-mode reconcile phase that happens to reuse the
	// process Purpose constant) overrides.
	cases := []struct {
		name      string
		purpose   Purpose
		phaseKind string
		want      Lane
	}{
		{"process+investigate phase kind wins", PurposeProcess, "investigate", LaneInvestigate},
		{"process+execute phase kind wins", PurposeProcess, "execute", LaneExecute},
		{"clarify+review phase kind wins", PurposeClarify, "review", LaneReview},
		{"workshop+reconcile phase kind wins", PurposeWorkshop, "reconcile", LaneReconcile},
		{"phase kind trims+lowercases", PurposeProcess, "  RECONCILE ", LaneReconcile},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := LaneOf(tc.purpose, tc.phaseKind)
			if err != nil {
				t.Fatalf("LaneOf(%q, %q): unexpected error: %v", tc.purpose, tc.phaseKind, err)
			}
			if got != tc.want {
				t.Fatalf("LaneOf(%q, %q) = %q, want %q", tc.purpose, tc.phaseKind, got, tc.want)
			}
		})
	}
}

func TestLaneOf_FallsBackToPurposeDefault(t *testing.T) {
	t.Parallel()

	cases := []struct {
		purpose Purpose
		want    Lane
	}{
		{PurposeProcess, LaneExecute},
		{PurposeFixup, LaneExecute},
		{PurposeFollowUp, LaneExecute},
		{PurposeSpecSync, LaneExecute},
		{PurposeClarify, LaneInvestigate},
		{PurposeWorkshop, LaneInvestigate},
		{PurposeClassify, LaneInvestigate},
		{PurposeResearch, LaneInvestigate},
		{PurposeFeedback, LaneInvestigate},
		{PurposeFeedbackContinue, LaneInvestigate},
		{PurposeMetaOrchestration, LaneInvestigate},
		{PurposeSwarmOperations, LaneInvestigate},
		{PurposeInitialize, LaneInvestigate},
		{PurposeFinalize, LaneReview},
		{PurposeReview, LaneReview},
		{PurposeMilestoneReview, LaneReview},
	}
	for _, tc := range cases {
		t.Run(string(tc.purpose), func(t *testing.T) {
			got, err := LaneOf(tc.purpose, "")
			if err != nil {
				t.Fatalf("LaneOf(%q, \"\"): unexpected error: %v", tc.purpose, err)
			}
			if got != tc.want {
				t.Fatalf("LaneOf(%q, \"\") = %q, want %q", tc.purpose, got, tc.want)
			}
		})
	}
}

func TestLaneOf_RejectsUnknownPhaseKind(t *testing.T) {
	t.Parallel()

	_, err := LaneOf(PurposeProcess, "deploy")
	if err == nil {
		t.Fatal("expected error for unknown phase_kind, got nil")
	}
}

func TestLaneOf_RejectsUnregisteredDynamicPurposeWithoutPhaseKind(t *testing.T) {
	t.Parallel()

	// Dynamic operating-mode purposes (e.g. "holistic_loop_investigate")
	// have no lane in purposeLane and rely on phaseKind. Without it,
	// LaneOf must error so the caller sees the misconfiguration.
	_, err := LaneOf(Purpose("holistic_loop_investigate"), "")
	if err == nil {
		t.Fatal("expected error for unregistered purpose without phase_kind, got nil")
	}
}

func TestIsKnownPurpose_DelegatesToLaneRegistry(t *testing.T) {
	t.Parallel()

	for _, p := range allRegisteredPurposes {
		if !isKnownPurpose(p) {
			t.Errorf("purpose %q registered but isKnownPurpose returned false", p)
		}
	}
	if isKnownPurpose(Purpose("not_a_purpose")) {
		t.Error("isKnownPurpose returned true for unregistered purpose")
	}
	if isKnownPurpose(Purpose("holistic_loop_investigate")) {
		t.Error("isKnownPurpose returned true for dynamic operating-mode purpose; only constants belong in the lane registry")
	}
}

func TestAllRegisteredPurposes_HaveLaneAssignments(t *testing.T) {
	t.Parallel()

	// Mirror of the init() panic check, kept as a regular test so a
	// missing lane is a clear test failure during development rather
	// than a panic-on-load that also breaks unrelated tests.
	for _, p := range allRegisteredPurposes {
		if _, ok := purposeLane[p]; !ok {
			t.Errorf("purpose %q listed in allRegisteredPurposes but not in purposeLane", p)
		}
	}
}

func TestLanes_ReturnsCanonicalOrdering(t *testing.T) {
	t.Parallel()

	got := Lanes()
	want := []Lane{LaneInvestigate, LaneExecute, LaneReview, LaneReconcile}
	if len(got) != len(want) {
		t.Fatalf("Lanes(): len=%d, want=%d", len(got), len(want))
	}
	for i, lane := range want {
		if got[i] != lane {
			t.Fatalf("Lanes()[%d] = %q, want %q", i, got[i], lane)
		}
	}
}

func TestLaneActiveCount_FiltersByLaneAndStatus(t *testing.T) {
	t.Parallel()

	records := []Record{
		{Purpose: PurposeProcess, Status: StatusRunning},
		{Purpose: PurposeProcess, Status: StatusStarting},
		{Purpose: PurposeProcess, Status: StatusComplete}, // inactive — excluded
		{Purpose: PurposeWorkshop, Status: StatusRunning}, // investigate lane
		{Purpose: PurposeReview, Status: StatusRunning},   // review lane
		{Purpose: Purpose("holistic_loop_reconcile"), PhaseKind: "reconcile", Status: StatusRunning},
		{Purpose: Purpose("dynamic_no_kind"), Status: StatusRunning}, // skipped — undecidable
	}

	if got := LaneActiveCount(records, LaneExecute); got != 2 {
		t.Errorf("Execute lane active = %d, want 2", got)
	}
	if got := LaneActiveCount(records, LaneInvestigate); got != 1 {
		t.Errorf("Investigate lane active = %d, want 1", got)
	}
	if got := LaneActiveCount(records, LaneReview); got != 1 {
		t.Errorf("Review lane active = %d, want 1", got)
	}
	if got := LaneActiveCount(records, LaneReconcile); got != 1 {
		t.Errorf("Reconcile lane active = %d, want 1", got)
	}
}

func TestLaneActiveCounts_BulkAggregate(t *testing.T) {
	t.Parallel()

	records := []Record{
		{Purpose: PurposeProcess, Status: StatusRunning},
		{Purpose: PurposeWorkshop, Status: StatusRunning},
		{Purpose: PurposeReview, Status: StatusRunning},
		{Purpose: PurposeProcess, Status: StatusComplete},
	}
	got := LaneActiveCounts(records)

	want := map[Lane]int{
		LaneInvestigate: 1,
		LaneExecute:     1,
		LaneReview:      1,
		LaneReconcile:   0,
	}
	for lane, expected := range want {
		if got[lane] != expected {
			t.Errorf("lane %q active = %d, want %d", lane, got[lane], expected)
		}
	}
	// Ensure every canonical lane has an entry, even when zero.
	if _, ok := got[LaneReconcile]; !ok {
		t.Error("LaneActiveCounts did not include LaneReconcile (must be present even when zero)")
	}
}

func TestErrLaneSaturated_IsTyped(t *testing.T) {
	t.Parallel()

	if !errors.Is(ErrLaneSaturated, ErrLaneSaturated) {
		t.Fatal("ErrLaneSaturated must be comparable via errors.Is")
	}
}
