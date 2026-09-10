package gotest

import (
	"context"
	"testing"

	"unit-health/internal/testquality/calibration"
)

func TestEvaluateCalibrationUsesAdapterObservationsAndCaseFilter(t *testing.T) {
	cases, err := calibration.LoadCases("../../testquality/testdata", "development.json", "development")
	if err != nil {
		t.Fatal(err)
	}
	for _, input := range []calibration.Input{cases[0].Input, {ID: "C013", Profile: "go-syntax-v1", Kind: "parser", Root: "snippets/go", Files: []string{"skip_test.go"}, SelectedTests: []string{"TestConditionalSkip"}}} {
		rows, err := evaluateCalibration(context.Background(), "../../testquality/testdata", input)
		if err != nil {
			t.Fatal(err)
		}
		if input.ID == "C013" {
			for _, row := range rows {
				if row.RuleID != "skip-declaration" {
					t.Fatalf("filtered calibration row = %+v", row)
				}
			}
		} else if len(rows) == 0 {
			t.Fatal("calibration returned no observed rows")
		}
	}
}
