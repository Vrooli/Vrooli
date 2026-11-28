package validation

import (
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"unit-health/internal/discovery"
)

// passedCoverageRun records that each workspace's coverage command ran and
// passed in this run, the precondition for judging its coverage artifact.
func passedCoverageRun(workspaces ...Workspace) map[string]bool {
	runs := make(map[string]bool, len(workspaces))
	for _, ws := range workspaces {
		runs[ws.ID] = true
	}
	return runs
}

// uiCoverageFixture writes a react-vite policy whose UI class commits to the
// given floor, plus an Istanbul json-summary measuring the given totals.
func uiCoverageFixture(t *testing.T, committed map[string]float64, summary string) (string, Workspace) {
	t.Helper()
	root := t.TempDir()
	profile := reactViteUnitPolicyProfile()
	ui := profile.PolicyClasses["react_vite_ui"]
	ui.Coverage.Committed = committed
	profile.PolicyClasses["react_vite_ui"] = ui
	writeUnitPolicyProfile(t, root, profile)
	wsDir := filepath.Join(root, "ui")
	writeFile(t, filepath.Join(wsDir, "coverage", "coverage-summary.json"), summary)
	return root, Workspace{ID: "ui", Language: "typescript", RootPath: wsDir, CoverageCommand: "pnpm test:coverage"}
}

// summaryJSON builds an Istanbul json-summary whose totals measure each metric
// as covered/100.
func summaryJSON(lines, statements, functions, branches int) string {
	count := func(covered int) string {
		return `{"total":100,"covered":` + strconv.Itoa(covered) + `}`
	}
	return `{"total":{"lines":` + count(lines) + `,"statements":` + count(statements) + `,"functions":` + count(functions) + `,"branches":` + count(branches) + `},` +
		`"src/App.tsx":{"lines":` + count(lines) + `}}`
}

var baselineCommitment = map[string]float64{"lines": 78, "statements": 78, "functions": 73, "branches": 65}

func TestCoverageBelowCommittedFloorFailsWithMeasuredAndRequiredValues(t *testing.T) {
	root, ws := uiCoverageFixture(t, baselineCommitment, summaryJSON(70, 79, 73, 66))

	_, findings := analyzeCoverage("demo", root, []Workspace{ws}, passedCoverageRun(ws), fixedNowStr)

	gate, ok := findingByCode(findings, codeCoverageBelowCommitted)
	if !ok {
		t.Fatalf("a regression below the committed floor must fail, got %v", codes(findings))
	}
	if gate.Severity != "error" || gate.Category != "coverage" {
		t.Fatalf("gate must be a blocking coverage finding: %+v", gate)
	}
	for _, want := range []string{"lines 70% < committed 78%", "(70/100)"} {
		if !strings.Contains(gate.Message, want) {
			t.Fatalf("gate message must name measured and required values (%q): %s", want, gate.Message)
		}
	}
	if strings.Contains(gate.Observed, "statements") {
		t.Fatalf("metrics at or above their floor are not regressions: %s", gate.Observed)
	}
	if status := deriveStatus(discovery.Inventory{Surfaces: []discovery.Surface{{ID: "ui"}}}, findings); status != "failed" {
		t.Fatalf("a committed-coverage regression must fail the run, status=%s", status)
	}
}

func TestCoverageAtCommittedFloorPassesAndRecordsTargetGapAsDebt(t *testing.T) {
	root, ws := uiCoverageFixture(t, baselineCommitment, summaryJSON(78, 78, 73, 65))

	_, findings := analyzeCoverage("demo", root, []Workspace{ws}, passedCoverageRun(ws), fixedNowStr)

	if _, ok := findingByCode(findings, codeCoverageBelowCommitted); ok {
		t.Fatalf("coverage at the committed floor must pass, got %v", codes(findings))
	}
	var debt *Finding
	for i := range findings {
		if findings[i].Code == codeLowCoverage && findings[i].ID == codeLowCoverage+"-ui" {
			debt = &findings[i]
		}
	}
	if debt == nil {
		t.Fatalf("the gap to the policy target must stay visible as debt, got %v", codes(findings))
	}
	if debt.Severity != "warning" || !strings.Contains(debt.Message, "85% policy target") || !strings.Contains(debt.Observed, "branches 65%") {
		t.Fatalf("debt must report measured value and target: %+v", debt)
	}
	if status := deriveStatus(discovery.Inventory{Surfaces: []discovery.Surface{{ID: "ui"}}}, findings); status != "passed" {
		t.Fatalf("debt alone must not fail the run, status=%s findings=%v", status, codes(findings))
	}
}

func TestCoverageGateIgnoresFailedAndStaleRuns(t *testing.T) {
	root, ws := uiCoverageFixture(t, baselineCommitment, summaryJSON(10, 10, 10, 10))

	_, failed := analyzeCoverage("demo", root, []Workspace{ws}, map[string]bool{"ui": false}, fixedNowStr)
	if _, ok := findingByCode(failed, codeCoverageBelowCommitted); ok {
		t.Fatalf("a failing test run reports the test failure, not a coverage regression: %v", codes(failed))
	}

	targets, stale := analyzeCoverage("demo", root, []Workspace{ws}, map[string]bool{}, fixedNowStr)
	if len(stale) != 0 || len(targets) != 0 {
		t.Fatalf("an artifact this run did not write must not be judged: targets=%v findings=%v", targets, codes(stale))
	}
}

func TestCoverageGateFailsClosedWhenCommittedMetricIsNotMeasured(t *testing.T) {
	root, ws := uiCoverageFixture(t, map[string]float64{"lines": 50, "branches": 50},
		`{"total":{"lines":{"total":10,"covered":9}},"src/App.tsx":{"lines":{"total":10,"covered":9}}}`)

	_, findings := analyzeCoverage("demo", root, []Workspace{ws}, passedCoverageRun(ws), fixedNowStr)

	gate, ok := findingByCode(findings, codeCoverageBelowCommitted)
	if !ok || !strings.Contains(gate.Observed, "not reported by the coverage artifact: branches") {
		t.Fatalf("an unmeasured committed metric must fail closed, got %+v", findings)
	}
}

func TestUncommittedClassIsMeasuredButNotGated(t *testing.T) {
	root, ws := uiCoverageFixture(t, nil, summaryJSON(10, 10, 10, 10))

	_, findings := analyzeCoverage("demo", root, []Workspace{ws}, passedCoverageRun(ws), fixedNowStr)

	if _, ok := findingByCode(findings, codeCoverageBelowCommitted); ok {
		t.Fatalf("a class without coverage.committed is advisory only, got %v", codes(findings))
	}
	if _, ok := findingByCode(findings, codeLowCoverage); !ok {
		t.Fatalf("the gap to target must still be reported, got %v", codes(findings))
	}
	missing, ok := findingByCode(findings, codeCoverageCommitmentBehind)
	if !ok || missing.Severity != "info" || missing.Observed != "no coverage.committed" {
		t.Fatalf("a default-gated UI class without a floor must say so, got %+v", findings)
	}
}

func TestCoverageCommitmentBehindProposesRatchetUpToTarget(t *testing.T) {
	root, ws := uiCoverageFixture(t, baselineCommitment, summaryJSON(81, 78, 99, 65))

	_, findings := analyzeCoverage("demo", root, []Workspace{ws}, passedCoverageRun(ws), fixedNowStr)

	behind, ok := findingByCode(findings, codeCoverageCommitmentBehind)
	if !ok {
		t.Fatalf("an outgrown floor must be flagged for ratcheting, got %v", codes(findings))
	}
	if behind.Severity != "info" || behind.Observed != "lines 78% -> 81%, functions 73% -> 85%" {
		t.Fatalf("ratchet proposal must cap at the target and skip unchanged metrics: %+v", behind)
	}
}

func TestRatchetedCommitment(t *testing.T) {
	cases := []struct {
		name                      string
		current, measured, target float64
		want                      float64
	}{
		{name: "rounds measured down", current: 70, measured: 78.66, target: 85, want: 78},
		{name: "caps at target", current: 70, measured: 97.2, target: 85, want: 85},
		{name: "never lowers", current: 80, measured: 72.4, target: 85, want: 80},
		{name: "keeps a stricter floor", current: 90, measured: 95, target: 85, want: 90},
		{name: "no target means no cap", current: 0, measured: 64.9, target: 0, want: 64},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := RatchetedCommitment(tc.current, tc.measured, tc.target); got != tc.want {
				t.Fatalf("RatchetedCommitment(%v, %v, %v)=%v want %v", tc.current, tc.measured, tc.target, got, tc.want)
			}
		})
	}
}

func TestPolicyRejectsMalformedCoverageCommitment(t *testing.T) {
	profile := reactViteUnitPolicyProfile()
	ui := profile.PolicyClasses["react_vite_ui"]
	ui.Coverage.Committed = map[string]float64{"lines": 78, "mutation": 40, "branches": 140}
	profile.PolicyClasses["react_vite_ui"] = ui

	findings := validateUnitPolicyProfile("demo", "testing.json", profile, fixedNowStr)

	var invalid []string
	for _, f := range findings {
		if f.Code == codeUnitPolicyInvalid && strings.Contains(f.Evidence, "coverage.committed") {
			invalid = append(invalid, f.Evidence)
		}
	}
	if len(invalid) != 2 {
		t.Fatalf("unknown metric and out-of-range percent must both be invalid, got %v", invalid)
	}
}

func TestCoverageWithinNoiseMarginOfFloorPasses(t *testing.T) {
	root, ws := uiCoverageFixture(t, baselineCommitment, summaryJSON(78, 78, 73, 64))

	_, findings := analyzeCoverage("demo", root, []Workspace{ws}, passedCoverageRun(ws), fixedNowStr)

	if _, ok := findingByCode(findings, codeCoverageBelowCommitted); ok {
		t.Fatalf("a one-point wobble (branches 64 vs committed 65) is run-to-run noise, got %v", codes(findings))
	}
}
