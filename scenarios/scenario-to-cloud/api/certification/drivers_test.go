package certification

import (
	"strings"
	"testing"
)

func TestAssertionDriversCoverEveryGroundBoundMatrixCell(t *testing.T) {
	m := mustMatrix(t)
	drivers, err := AssertionDrivers(m)
	if err != nil {
		t.Fatal(err)
	}
	if len(drivers) != 532 || len(m.RequiredCells()) != 532 {
		t.Fatalf("driver inventory has %d entries for %d required cells; want 532", len(drivers), len(m.RequiredCells()))
	}
	seen := make(map[CellID]bool, len(drivers))
	for _, driver := range drivers {
		if seen[driver.Cell] {
			t.Fatalf("duplicate driver for %s", driver.Cell)
		}
		seen[driver.Cell] = true
		want, err := driverRefForMatrix(m, driver.Cell)
		if err != nil || driver.Ref != want {
			t.Fatalf("driver identity for %s = %q, want %q (err %v)", driver.Cell, driver.Ref, want, err)
		}
		caseRow, _ := m.Case(driver.Cell.CaseID)
		wantAdapterStatus := "missing"
		if laneOwnerProofSupported(driver.Cell.Lane) {
			wantAdapterStatus = "implemented"
		}
		if driver.Trigger != caseRow.Trigger || driver.Expected != caseRow.Assertion || driver.Status != "assertion_owner_required" || driver.OwnerReceiptScheme != laneOwnerReceiptScheme(driver.Cell.Lane) || driver.OwnerAdapterStatus != wantAdapterStatus {
			t.Fatalf("driver %s lost frozen assertion or truthful pending status: %+v", driver.Cell, driver)
		}
		if driver.Owner == "" || driver.EvidenceSource == "" {
			t.Fatalf("driver %s has no evidence owner contract", driver.Cell)
		}
	}
}

func TestEveryLaneHasExplicitOwnerAndEvidenceSource(t *testing.T) {
	want := map[Lane]string{
		LanePackage: "scenario-to-cloud", LaneAPI: "scenario-to-cloud", LaneWorkflow: "scenario-to-cloud",
		LaneRelay: "Vrooli Bridge", LaneQEMU: "scenario-to-cloud QEMU lane", LaneRealVPS: "scenario-to-cloud VPS lane",
		LaneBASUI: "Browser Automation Studio", LaneIndependentReview: "independent reviewer", LaneSoak: "scenario-to-cloud soak owner",
	}
	m := mustMatrix(t)
	for _, lane := range m.LaneVocabulary {
		owner, source := laneEvidenceOwner(lane)
		if owner == "unassigned" || source == "owner evidence required" || owner != want[lane] || strings.TrimSpace(source) == "" {
			t.Errorf("lane %q has incomplete owner contract: owner=%q source=%q want=%q", lane, owner, source, want[lane])
		}
		if scheme := laneOwnerReceiptScheme(lane); scheme == "" {
			t.Errorf("lane %q has no owner receipt scheme", lane)
		}
	}
}

func TestLaneOwnerReceiptSchemesMatchTypedOwnerBoundary(t *testing.T) {
	want := map[Lane]string{
		LanePackage: "test-genie", LaneAPI: "test-genie", LaneWorkflow: "test-genie", LaneRelay: "bridge-run",
		LaneQEMU: "target-receipt", LaneRealVPS: "target-receipt", LaneBASUI: "bas-execution",
		LaneIndependentReview: "reviewer-attestation", LaneSoak: "soak-run",
	}
	for lane, scheme := range want {
		if got := laneOwnerReceiptScheme(lane); got != scheme {
			t.Errorf("lane %q owner receipt scheme = %q, want %q", lane, got, scheme)
		}
		if err := validateOwnerReceiptScheme(lane, scheme+":fixture"); err != nil {
			t.Errorf("lane %q refused its typed receipt reference: %v", lane, err)
		}
	}
	for _, lane := range []Lane{LaneIndependentReview, LaneSoak} {
		if laneOwnerProofSupported(lane) {
			t.Errorf("lane %q claimed an unimplemented owner adapter", lane)
		}
	}
	for _, lane := range []Lane{LanePackage, LaneAPI, LaneWorkflow, LaneRelay, LaneQEMU, LaneRealVPS, LaneBASUI} {
		if !laneOwnerProofSupported(lane) {
			t.Errorf("implemented owner lane %q is marked unavailable", lane)
		}
	}
	if err := validateOwnerReceiptScheme(LaneRelay, "test-genie:forged"); err == nil {
		t.Fatal("relay accepted a Test Genie receipt without Bridge run ownership")
	}
}

func TestReceiptFromDriverResultBindsExactCellAndAssertion(t *testing.T) {
	m := mustMatrix(t)
	cell := CellID{CaseID: "RUN-01", Lane: LanePackage, TargetGround: TargetGroundQEMU}
	driverRef, err := driverRefForMatrix(m, cell)
	if err != nil {
		t.Fatal(err)
	}
	caseRow, _ := m.Case(cell.CaseID)
	result := DriverResult{
		Cell: cell, DriverRef: driverRef, Verdict: VerdictPassed,
		Candidate:            testCandidate(),
		Target:               Target{Ground: cell.TargetGround, MachineID: "qemu:guest-1", Generation: 1, Architecture: "amd64", OS: "ubuntu-24.04"},
		ValidationReceiptRef: "test-genie:run-203", OperationRefs: []string{"operation:op-4"},
		TargetReceiptRefs:  []string{"target-receipt://op-4/apply?fence=9"},
		AssertionArtifacts: []string{"assertion-artifact:" + driverRef + ":sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		Assertion:          caseRow.Assertion,
		ObservedAt:         testNow,
	}
	if _, err := ReceiptFromDriverResult(m, result); err == nil || !strings.Contains(err.Error(), "authenticated target receipt") {
		t.Fatalf("structural fields without owner proofs were accepted: %v", err)
	}

	result.Assertion = "a different test passed"
	if _, err := ReceiptFromDriverResult(m, result); err == nil || !strings.Contains(err.Error(), "frozen matrix assertion") {
		t.Fatalf("substituted assertion accepted: %v", err)
	}
	result.Assertion = caseRow.Assertion
	result.DriverRef = "certification-assertion:v1:RUN-02:package:qemu"
	if _, err := ReceiptFromDriverResult(m, result); err == nil || !strings.Contains(err.Error(), "driver_ref must be") {
		t.Fatalf("cross-cell driver accepted: %v", err)
	}
	result.DriverRef = driverRef
}

func TestSchemaV2ReceiptRejectsMissingCellDriverRef(t *testing.T) {
	r := receipt("no-driver", "RUN-01", LanePackage, VerdictPassed, candidateDigest)
	r.DriverRef = ""
	if err := r.Validate(mustMatrix(t)); err == nil || !strings.Contains(err.Error(), "driver_ref must be") {
		t.Fatalf("schema v2 receipt without its cell driver accepted: %v", err)
	}
}
