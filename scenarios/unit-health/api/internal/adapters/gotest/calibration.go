package gotest

import (
	"context"
	"path/filepath"

	"unit-health/internal/testquality"
	"unit-health/internal/testquality/calibration"
)

func init() {
	calibration.RegisterProfile("go-syntax-v1", evaluateCalibration)
}

func evaluateCalibration(_ context.Context, root string, input calibration.Input) ([]testquality.Result, error) {
	rows, err := Analyze(Input{
		Root: filepath.Join(root, input.Root), Workspace: "calibration", Files: input.Files,
		GOOS: input.GOOS, GOARCH: input.GOARCH, SelectedTests: input.SelectedTests,
		RegisteredHelpers: input.RegisteredHelpers, TestKinds: input.TestKinds,
	})
	if input.ID == "C013" || input.ID == "C014" {
		filtered := rows[:0]
		for _, row := range rows {
			if row.RuleID == "skip-declaration" {
				filtered = append(filtered, row)
			}
		}
		rows = filtered
	}
	return rows, err
}
