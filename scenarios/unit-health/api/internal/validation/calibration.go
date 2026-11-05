package validation

import (
	"context"
	"path/filepath"

	"unit-health/internal/testquality"
	"unit-health/internal/testquality/calibration"
)

func init() {
	calibration.RegisterEmittedCodes(EmittedCodes)
	calibration.RegisterProfile("go-architecture-v1", evaluateArchitectureCalibration)
}

func evaluateArchitectureCalibration(_ context.Context, root string, input calibration.Input) ([]testquality.Result, error) {
	findings := analyzeArchitecture("calibration", []Workspace{{
		ID:       "calibration",
		Language: "go",
		RootPath: filepath.Join(root, input.Root),
	}}, "")
	status := testquality.CheckedClean
	for _, finding := range findings {
		if finding.Code == codeTestUtilMissing {
			status = testquality.Violation
			break
		}
	}
	return []testquality.Result{{
		RuleID:         "test-util-root",
		RuleVersion:    "1",
		Target:         testquality.Target{Workspace: "calibration", File: input.Files[0], TestID: "TestUtilRoot"},
		SupportProfile: input.Profile,
		Status:         status,
		Reason:         testquality.ReasonNone,
		EvidenceKind:   testquality.Static,
		Severity:       testquality.Warning,
		Enforcement:    testquality.Advisory,
		Location:       testquality.Location{Line: 1, Column: 1},
	}}, nil
}
