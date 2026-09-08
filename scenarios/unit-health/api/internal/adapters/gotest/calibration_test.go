package gotest

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"unit-health/internal/testquality"
	"unit-health/internal/testquality/calibration"
)

func TestRetainedGoAssertionCalibration(t *testing.T) {
	root := filepath.Join("..", "..", "testquality", "testdata")
	cases, err := calibration.LoadCases(root, "development.json", "development")
	if err != nil {
		t.Fatal(err)
	}
	var selected []calibration.Case
	for _, c := range cases {
		// Skip declarations have their own rule-scoped calibration below.
		if c.Input.Profile == "go-syntax-v1" && c.Input.ID != "C013" && c.Input.ID != "C014" {
			selected = append(selected, c)
		}
	}
	if len(selected) != 19 {
		t.Fatalf("unexpected retained inventory: %d", len(selected))
	}
	report, err := calibration.Run(context.Background(), selected, "development", func(_ context.Context, in calibration.Input) ([]testquality.Result, error) {
		return Analyze(Input{Root: filepath.Join(root, in.Root), Workspace: "api", Files: in.Files, GOOS: in.GOOS, GOARCH: in.GOARCH, SelectedTests: in.SelectedTests, RegisteredHelpers: in.RegisteredHelpers, TestKinds: in.TestKinds})
	})
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if path := os.Getenv("UNIT_HEALTH_GO_CALIBRATION_OUTPUT"); path != "" {
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if report.FailedCases != 0 {
		t.Fatalf("retained Go calibration:\n%s", data)
	}
}

func TestRetainedGoSkipDeclarationCalibration(t *testing.T) {
	root := filepath.Join("..", "..", "testquality", "testdata")
	cases, err := calibration.LoadCases(root, "development.json", "development")
	if err != nil {
		t.Fatal(err)
	}
	var selected []calibration.Case
	for _, c := range cases {
		if c.Input.ID == "C013" || c.Input.ID == "C014" {
			selected = append(selected, c)
		}
	}
	if len(selected) != 2 {
		t.Fatalf("skip corpus inventory: %d", len(selected))
	}
	report, err := calibration.Run(context.Background(), selected, "development", func(_ context.Context, in calibration.Input) ([]testquality.Result, error) {
		rows, err := Analyze(Input{Root: filepath.Join(root, in.Root), Workspace: "api", Files: in.Files, SelectedTests: in.SelectedTests})
		var declarations []testquality.Result
		for _, row := range rows {
			if row.RuleID == "skip-declaration" {
				declarations = append(declarations, row)
			}
		}
		return declarations, err
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.FailedCases != 0 {
		t.Fatalf("retained skip calibration: %+v", report)
	}
}
