package gates

// Calibration is the anti-fabrication boundary for catalog gates. A fixture is
// a small, real catalog asset with one deliberately planted defect. The gate
// is run against an isolated overlay containing that asset; a clean result is
// therefore a failed calibration, not evidence of quality.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type CalibrationFixture struct {
	Gate                string `json:"gate"`
	AssetID             string `json:"assetId"`
	Kind                string `json:"kind"`
	Runner              string `json:"runner"`
	RequiredFailureCode string `json:"requiredFailureCode"`
	Description         string `json:"description"`
	Source              string `json:"source"`
	Story               string `json:"story"`
	CatalogAsset        string `json:"catalogAsset"`
	Mutation            string `json:"mutation"`
	Test                string `json:"test"`
}

type CalibrationResult struct {
	Gate                string
	Fixture             string
	RequiredFailureCode string
	ObservedFailureCode string
	Status              string
	Message             string
}

type CalibrationReport struct {
	Results           []CalibrationResult
	NonDiscriminating bool
	Delegated         bool
}

type GateRunner = Runner

// GateRunnerFor returns the production runner for a declared gate. A nil
// runner is intentional for external gates: their calibration is delegated to
// the owning browser/toolchain suite rather than fabricated in this process.
func GateRunnerFor(gate string) GateRunner {
	definition, ok := Lookup(gate)
	if !ok {
		return nil
	}
	return definition.Run
}

// Calibrate evaluates every fixture owned by gate. A missing fixture is a
// quarantine condition for a blocking gate, not an empty pass.
func Calibrate(root, gate string, runner GateRunner) (CalibrationReport, error) {
	fixtures, err := loadCalibrationFixtures(root, gate)
	if err != nil {
		return CalibrationReport{}, err
	}
	if len(fixtures) == 0 {
		if gate == "dist-resolution" {
			return CalibrationReport{
				Results:   []CalibrationResult{{Gate: gate, Status: "delegated", Message: "bundle-resolution calibration is supplied by the published-package validation suite"}},
				Delegated: true,
			}, nil
		}
		return CalibrationReport{
			Results:           []CalibrationResult{{Gate: gate, Fixture: "", Status: "missing-fixture", Message: "blocking gate owns no calibration fixture"}},
			NonDiscriminating: true,
		}, nil
	}

	report := CalibrationReport{Results: make([]CalibrationResult, 0, len(fixtures))}
	for _, fixture := range fixtures {
		result := CalibrationResult{
			Gate:                gate,
			Fixture:             filepath.ToSlash(filepath.Join("catalog", "calibration", gate, filepath.Base(fixturePath(root, gate, fixture)))),
			RequiredFailureCode: fixture.RequiredFailureCode,
		}
		if fixture.Runner != "static" {
			result.Status = "delegated"
			result.Message = "the declared runner is external to the deterministic catalog gate process; the owning scenario suite supplies the calibration"
			report.Delegated = true
			report.Results = append(report.Results, result)
			continue
		}
		overlay, cleanup, materializeErr := materializeFixture(root, gate, fixture)
		if materializeErr != nil {
			return CalibrationReport{}, materializeErr
		}
		observed, runErr := runner(Scope{Root: overlay})
		cleanup()
		if runErr != nil {
			result.Status = "runner-error"
			result.Message = runErr.Error()
			report.NonDiscriminating = true
			report.Results = append(report.Results, result)
			continue
		}
		for _, finding := range append(append([]Finding(nil), observed.Findings...), observed.RunnerError...) {
			if finding.Code != fixture.RequiredFailureCode {
				continue
			}
			if fixture.AssetID == "" || fixture.Mutation != "" || finding.AssetID == fixture.AssetID || finding.AssetID == "" || strings.Contains(strings.ToLower(finding.AssetID), strings.ToLower(fixture.AssetID)) {
				result.ObservedFailureCode = finding.Code
				break
			}
		}
		if result.ObservedFailureCode != "" {
			result.Status = "failed"
			result.Message = "fixture defect was detected"
		} else {
			result.Status = "non-discriminating"
			observedCodes := make([]string, 0, len(observed.Findings)+len(observed.RunnerError))
			for _, finding := range append(observed.Findings, observed.RunnerError...) {
				if finding.Code != "" {
					observedCodes = append(observedCodes, finding.Code)
				}
			}
			result.Message = fmt.Sprintf("fixture completed without its required failure code; observed %v", observedCodes)
			report.NonDiscriminating = true
		}
		report.Results = append(report.Results, result)
	}
	return report, nil
}

func loadCalibrationFixtures(root, gate string) ([]CalibrationFixture, error) {
	dir := filepath.Join(root, "scenarios", "react-component-library", "catalog", "calibration", gate)
	paths, err := filepath.Glob(filepath.Join(dir, "fixture.json"))
	if err != nil {
		return nil, err
	}
	var fixtures []CalibrationFixture
	for _, path := range paths {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, readErr
		}
		var fixture CalibrationFixture
		if err := json.Unmarshal(data, &fixture); err != nil {
			return nil, fmt.Errorf("parse calibration fixture %s: %w", path, err)
		}
		if fixture.Gate == "" {
			fixture.Gate = gate
		}
		if fixture.Gate != gate {
			return nil, fmt.Errorf("calibration fixture %s declares gate %q, want %q", path, fixture.Gate, gate)
		}
		if fixture.RequiredFailureCode == "" {
			return nil, fmt.Errorf("calibration fixture %s has no requiredFailureCode", path)
		}
		fixtures = append(fixtures, fixture)
	}
	return fixtures, nil
}
