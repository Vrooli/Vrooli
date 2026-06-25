package cmd

import "testing"

func TestMapSuggestionStatus(t *testing.T) {
	cases := map[string]string{
		"accepted":   "accepted",
		" Accepted ": "accepted",
		"REJECTED":   "rejected",
		"rejected":   "rejected",
		"":           "pending",
		"other":      "pending",
		"unknown":    "pending",
	}
	for in, want := range cases {
		if got := mapSuggestionStatus(in); got != want {
			t.Errorf("mapSuggestionStatus(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestModePrefix(t *testing.T) {
	if got := modePrefix(true); got != "[dry-run]" {
		t.Errorf("modePrefix(true) = %q", got)
	}
	if got := modePrefix(false); got != "[migrate]" {
		t.Errorf("modePrefix(false) = %q", got)
	}
}

func TestAnswerString(t *testing.T) {
	if got := answerString(nil); got != "" {
		t.Errorf("nil = %q, want empty", got)
	}
	if got := answerString("hello"); got != "hello" {
		t.Errorf("string = %q", got)
	}
	if got := answerString(42); got != "42" {
		t.Errorf("int = %q, want 42", got)
	}
	if got := answerString([]string{"a", "b"}); got != `["a","b"]` {
		t.Errorf("slice = %q", got)
	}
}

func TestComputeReadiness(t *testing.T) {
	// fully ready: all answered + suggest + enhance.
	full := computeReadiness(3, 3, true, true)
	if full.ProblemClarity != 2 || full.ScopeDefined != 2 || full.ApproachSolid != 2 ||
		full.Testable != 1 || full.RiskAwareness != 1 {
		t.Errorf("full readiness = %+v", full)
	}

	// all answered but no refinement.
	partial := computeReadiness(2, 2, false, false)
	if partial.ProblemClarity != 2 || partial.ScopeDefined != 1 || partial.ApproachSolid != 0 {
		t.Errorf("partial readiness = %+v", partial)
	}

	// not all answered -> zero value.
	none := computeReadiness(1, 3, true, true)
	if none != (workshopReadiness{}) {
		t.Errorf("incomplete readiness = %+v, want zero", none)
	}

	// zero questions -> not allAnswered -> zero value.
	zeroQ := computeReadiness(0, 0, true, true)
	if zeroQ != (workshopReadiness{}) {
		t.Errorf("zero-question readiness = %+v, want zero", zeroQ)
	}
}
