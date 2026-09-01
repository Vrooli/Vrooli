package main

import "testing"

func TestHasRegressionDetectsWorseRunnerErrors(t *testing.T) {
	baseline := []measure{
		{Name: "runner_errors", Value: 2},
		{Name: "findings", Value: 10},
		{Name: "inspected_files", Value: 20},
	}
	current := map[string]int{"runner_errors": 3, "findings": 10, "inspected_files": 20}
	if !hasRegression(baseline, current) {
		t.Fatal("runner error increase must be reported as a regression")
	}
	current["runner_errors"] = 2
	if hasRegression(baseline, current) {
		t.Fatal("unchanged measures must not be reported as a regression")
	}
}
