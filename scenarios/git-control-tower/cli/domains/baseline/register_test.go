package baseline

import (
	"context"
	"errors"
	"testing"
	"time"
)

// A snapshot interrupted (SIGINT) or whose bounded client deadline elapses must
// DETACH (exit 0, the durable run continues), not surface as a hard error.
// A genuine RPC error must propagate.
func TestIsDetach(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	live := context.Background()

	cases := []struct {
		name string
		ctx  context.Context
		err  error
		want bool
	}{
		{"interrupt cancels ctx", canceled, errors.New("rpc aborted"), true},
		{"deadline exceeded", live, context.DeadlineExceeded, true},
		{"explicit canceled error", live, context.Canceled, true},
		{"real backend error", live, errors.New("internal: pin failed"), false},
	}
	for _, tc := range cases {
		if got := isDetach(tc.ctx, tc.err); got != tc.want {
			t.Errorf("%s: isDetach = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestHumanDuration(t *testing.T) {
	if got := humanDuration(30 * time.Minute); got != "30m0s" {
		t.Errorf("humanDuration = %q", got)
	}
}

func TestBaselineClientTimeoutIsBounded(t *testing.T) {
	// The snapshot/diff deadline must be a finite ceiling, never zero (which
	// http.Client treats as "no timeout" — the bare-Background hang this fixes).
	if baselineClientTimeout <= 0 {
		t.Fatalf("baselineClientTimeout must be a positive ceiling, got %v", baselineClientTimeout)
	}
}

func TestExitCodeForVerdict(t *testing.T) {
	cases := map[string]int{
		"clean":          exitOK,
		"new-failure":    exitOK, // added by the change, not a regression — safe to proceed
		"preexisting":    exitOK, // inherited, not caused by the change
		"regression":     exitRegression,
		"not-comparable": exitNotComparable,
		"":               exitOK,
	}
	for verdict, want := range cases {
		if got := exitCodeForVerdict(verdict); got != want {
			t.Errorf("exitCodeForVerdict(%q) = %d, want %d", verdict, got, want)
		}
	}
}

func TestSplitCSV(t *testing.T) {
	if got := splitCSV(""); got != nil {
		t.Errorf("empty = %v, want nil", got)
	}
	got := splitCSV("workflows, tests ,structure")
	want := []string{"workflows", "tests", "structure"}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestVerdictMark(t *testing.T) {
	cases := map[string]string{
		"clean":          "✓",
		"regression":     "✗",
		"new-failure":    "✗",
		"preexisting":    "•",
		"not-comparable": "?",
	}
	for v, want := range cases {
		if got := verdictMark(v); got != want {
			t.Errorf("verdictMark(%q) = %q want %q", v, got, want)
		}
	}
}

func TestSummaryText(t *testing.T) {
	if got := summaryText(`{"passed":3,"failed":0}`); got != "passed:3,failed:0" {
		t.Errorf("summaryText = %q", got)
	}
	if got := summaryText(""); got != "" {
		t.Errorf("empty summary = %q", got)
	}
}
