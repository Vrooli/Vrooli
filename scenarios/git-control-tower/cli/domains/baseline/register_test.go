package baseline

import "testing"

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
