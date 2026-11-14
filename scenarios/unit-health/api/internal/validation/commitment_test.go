package validation

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

const indentedTestingJSON = `{
  "phases": {"unit": {"enabled": true}},
  "unit": {
    "policy_profile": {
      "policy_classes": {
        "react_vite_ui": {
          "language": "typescript",
          "coverage": {
            "minimum_percent": 85,
            "mode": "both"
          }
        }
      }
    }
  }
}
`

func TestSetCoverageCommitmentEditsOnlyTheCommittedMember(t *testing.T) {
	got, err := SetCoverageCommitment(indentedTestingJSON, "react_vite_ui", map[string]float64{"branches": 65, "lines": 78})
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(indentedTestingJSON, `            "mode": "both"
`, `            "mode": "both",
            "committed": {"lines": 78, "branches": 65}
`, 1)
	if got != want {
		t.Fatalf("insert changed more than the member:\n%s", got)
	}

	raised, err := SetCoverageCommitment(got, "react_vite_ui", map[string]float64{"branches": 70, "lines": 78})
	if err != nil {
		t.Fatal(err)
	}
	if raised != strings.Replace(want, `"branches": 65`, `"branches": 70`, 1) {
		t.Fatalf("replace changed more than the value:\n%s", raised)
	}
}

func TestSetCoverageCommitmentHandlesInlineAndMissingObjects(t *testing.T) {
	inline := `{"unit":{"policy_profile":{"policy_classes":{"ui":{"coverage":{"minimum_percent":85}}}}}}`
	got, err := SetCoverageCommitment(inline, "ui", map[string]float64{"lines": 50})
	if err != nil || got != `{"unit":{"policy_profile":{"policy_classes":{"ui":{"coverage":{"minimum_percent":85, "committed": {"lines": 50}}}}}}}` {
		t.Fatalf("inline insert = %s, %v", got, err)
	}
	if !json.Valid([]byte(got)) {
		t.Fatal("result is not valid JSON")
	}
	if _, err := SetCoverageCommitment(inline, "absent_class", map[string]float64{"lines": 50}); err == nil {
		t.Fatal("a class the profile does not declare must be refused, not invented")
	}
}

func TestPlanCoverageCommitmentsAdoptsNativeGateAtMeasuredBaseline(t *testing.T) {
	root, _ := uiCoverageFixture(t, nil, summaryJSON(78, 79, 92, 65))

	plan, err := PlanCoverageCommitments(root, CoverageCommitmentRequest{NativeGates: map[string]map[string]float64{
		"ui": {"lines": 85, "statements": 85, "functions": 85, "branches": 85},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Changes) != 1 || !plan.GateOwned["ui"] {
		t.Fatalf("expected one adopted commitment owning the ui gate, got %+v", plan)
	}
	want := map[string]float64{"lines": 78, "statements": 79, "functions": 85, "branches": 65}
	if !sameCommitment(plan.Changes[0].To, want) {
		t.Fatalf("commitment = %v, want measured floor capped at target %v", plan.Changes[0].To, want)
	}
	profile, _, ok, _ := loadUnitPolicyProfile("", writeAndReload(t, root, plan.After), "")
	if !ok || !sameCommitment(profile.PolicyClasses["react_vite_ui"].Coverage.Committed, want) {
		t.Fatalf("planned testing.json does not carry the commitment: %+v", profile.PolicyClasses["react_vite_ui"].Coverage)
	}
}

func TestPlanCoverageCommitmentsNeverAdoptsAZeroNativeGateAsZero(t *testing.T) {
	root, _ := uiCoverageFixture(t, nil, summaryJSON(40, 41, 42, 43))

	plan, err := PlanCoverageCommitments(root, CoverageCommitmentRequest{NativeGates: map[string]map[string]float64{
		"ui": {"lines": 0, "statements": 0, "functions": 0, "branches": 0},
	}})
	if err != nil || len(plan.Changes) != 1 {
		t.Fatalf("plan=%+v err=%v", plan, err)
	}
	if got := plan.Changes[0].To; got["lines"] != 40 || got["branches"] != 43 {
		t.Fatalf("a vacuous native gate must become the measured baseline, got %v", got)
	}
}

func TestPlanCoverageCommitmentsKeepsGateWithoutArtifact(t *testing.T) {
	root, ws := uiCoverageFixture(t, nil, "{}")
	writeFile(t, filepath.Join(ws.RootPath, "coverage", "coverage-summary.json"), "not json")

	plan, err := PlanCoverageCommitments(root, CoverageCommitmentRequest{NativeGates: map[string]map[string]float64{"ui": {"lines": 85}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Changes) != 0 || plan.GateOwned["ui"] || plan.After != plan.Before || len(plan.Messages) != 1 {
		t.Fatalf("without a measurement the native gate must stay and the plan must say why: %+v", plan)
	}
}

func TestPlanCoverageCommitmentsRatchetsOnlyUpward(t *testing.T) {
	root, _ := uiCoverageFixture(t, baselineCommitment, summaryJSON(81, 70, 99, 65))

	plan, err := PlanCoverageCommitments(root, CoverageCommitmentRequest{Ratchet: true})
	if err != nil || len(plan.Changes) != 1 {
		t.Fatalf("plan=%+v err=%v", plan, err)
	}
	want := map[string]float64{"lines": 81, "statements": 78, "functions": 85, "branches": 65}
	if !sameCommitment(plan.Changes[0].To, want) {
		t.Fatalf("ratchet = %v, want %v (raise, cap at target, never lower)", plan.Changes[0].To, want)
	}

	again, err := PlanCoverageCommitments(writeAndReload(t, root, plan.After), CoverageCommitmentRequest{Ratchet: true})
	if err != nil || len(again.Changes) != 0 || again.After != again.Before {
		t.Fatalf("a ratcheted commitment must be stable, got %+v err=%v", again, err)
	}
}

func writeAndReload(t *testing.T, root, testingJSON string) string {
	t.Helper()
	writeFile(t, filepath.Join(root, ".vrooli", "testing.json"), testingJSON)
	return root
}

func TestPlanCoverageCommitmentsRatchetAdoptsAMissingUIFloor(t *testing.T) {
	root, _ := uiCoverageFixture(t, nil, summaryJSON(72, 71, 90, 60))

	plan, err := PlanCoverageCommitments(root, CoverageCommitmentRequest{Ratchet: true})
	if err != nil || len(plan.Changes) != 1 || plan.Changes[0].PolicyClass != "react_vite_ui" {
		t.Fatalf("the ratchet must adopt the measured floor for an uncommitted UI class, got %+v err=%v", plan, err)
	}
	if want := (map[string]float64{"lines": 72, "statements": 71, "functions": 85, "branches": 60}); !sameCommitment(plan.Changes[0].To, want) {
		t.Fatalf("adopted floor = %v, want %v", plan.Changes[0].To, want)
	}
	profile, _, _, _ := loadUnitPolicyProfile("", writeAndReload(t, root, plan.After), "")
	if len(profile.PolicyClasses["go_service"].Coverage.Committed) != 0 {
		t.Fatal("Go classes stay advisory; the ratchet must not commit them")
	}
}
