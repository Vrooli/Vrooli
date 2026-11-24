package soak

import "testing"

func TestFailedAssertionsRequiresEveryOutcomeToBeMeasuredAndPassed(t *testing.T) {
	tests := []struct {
		name string
		doc  runDocument
		want int
	}{
		{name: "all passed", doc: runDocument{Assertions: []struct {
			Name    string `json:"name"`
			Outcome string `json:"outcome"`
			Detail  string `json:"detail"`
		}{{Name: "latency", Outcome: "passed"}, {Name: "recovery", Outcome: "passed"}}}, want: 0},
		{name: "failed and unknown do not qualify", doc: runDocument{Assertions: []struct {
			Name    string `json:"name"`
			Outcome string `json:"outcome"`
			Detail  string `json:"detail"`
		}{{Name: "latency", Outcome: "failed"}, {Name: "recovery", Outcome: "unmeasured"}}}, want: 2},
		{name: "empty evidence is not a pass", doc: runDocument{}, want: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := failedAssertions(tt.doc); got != tt.want {
				t.Fatalf("failedAssertions() = %d, want %d", got, tt.want)
			}
		})
	}
}
