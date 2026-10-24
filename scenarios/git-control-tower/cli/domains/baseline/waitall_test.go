package baseline

import (
	"errors"
	"testing"
)

func TestParseDiffRef(t *testing.T) {
	ok, err := parseDiffRef("web:pre-launch:20260616-120000-abcd")
	if err != nil {
		t.Fatalf("valid ref: %v", err)
	}
	if ok.scenario != "web" || ok.name != "pre-launch" || ok.run != "20260616-120000-abcd" {
		t.Fatalf("parsed = %+v", ok)
	}
	for _, bad := range []string{"", "web", "web:pre-launch", "web::run", ":n:r", "web:n:"} {
		if _, err := parseDiffRef(bad); err == nil {
			t.Errorf("parseDiffRef(%q) should error", bad)
		}
	}
}

// TestAggregateDiffCodePrecedence pins worst-wins: regression(1) > not-ready(3) >
// not-comparable(2) > clean(0).
func TestAggregateDiffCodePrecedence(t *testing.T) {
	clean := diffWaitResult{verdict: "clean"}
	regression := diffWaitResult{verdict: "regression"}
	notComparable := diffWaitResult{verdict: "not-comparable"}
	notReady := diffWaitResult{inProgress: true}
	errored := diffWaitResult{err: errors.New("unreachable")}

	cases := []struct {
		name    string
		results []diffWaitResult
		want    int
	}{
		{"all clean", []diffWaitResult{clean, clean}, exitOK},
		{"regression dominates", []diffWaitResult{clean, notReady, notComparable, regression}, exitRegression},
		{"not-ready beats not-comparable", []diffWaitResult{clean, notComparable, notReady}, exitNotReady},
		{"error counts as not-comparable", []diffWaitResult{clean, errored}, exitNotComparable},
		{"changed verdict is clean exit", []diffWaitResult{clean, {verdict: "changed"}}, exitOK},
	}
	for _, tc := range cases {
		if got := aggregateDiffCode(tc.results); got != tc.want {
			t.Errorf("%s: aggregateDiffCode = %d, want %d", tc.name, got, tc.want)
		}
	}
}
