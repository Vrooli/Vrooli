package testquality

import "testing"

func TestRequirementLinksSeparateIdentityExecutionAndResponsibility(t *testing.T) {
	registry := RequirementRegistry{SchemaVersion: "requirement-registry/v1", Requirements: []RequirementDeclaration{
		{ID: "UH-CORE-001", Validations: []RequirementResponsibility{{Phase: "unit"}}},
		{ID: "UH-CORE-010", Validations: []RequirementResponsibility{{Phase: "integration"}}},
		{ID: "UH-BUSINESS-001", Validations: []RequirementResponsibility{{Phase: "business"}}},
	}}
	makeTest := func(id string, state ExecutionState) TestLinks {
		return TestLinks{Target: Target{Workspace: "ui", File: "test.ts", TestID: id}, IDs: []string{"UH-CORE-001"}, EvidenceKind: Runtime, RunID: "current", ExpectedRunID: "current", Execution: state}
	}
	pass, skip, stale, comment := makeTest("parameter-1", ExecutionPassed), makeTest("parameter-2", ExecutionSkipped), makeTest("old", ExecutionPassed), makeTest("comment", ExecutionPassed)
	stale.RunID = "old-run"
	comment.EvidenceKind = Static
	comment.IDs = []string{"UH-CORE-01"} // Prefix resemblance is not membership.
	report := ReconcileRequirements(registry, ReasonNone, "unit", []TestLinks{pass, skip, stale, comment})
	if report.UnavailableReason != ReasonNone || len(report.Links) != 4 {
		t.Fatalf("report: %+v", report)
	}
	if report.Links[0].Registration != "registered" || report.Links[0].Execution != ExecutionPassed || report.Links[1].Execution != ExecutionSkipped || report.Links[1].Target.TestID != "parameter-2" {
		t.Fatalf("execution identities collapsed: %+v", report.Links)
	}
	if report.Links[2].Execution != ExecutionUnknown || report.Links[2].Reason != StaleEvidence {
		t.Fatalf("stale run became current pass: %+v", report.Links[2])
	}
	if report.Links[3].Registration != "stale" || report.Links[3].Execution != ExecutionNotRun {
		t.Fatalf("comment/prefix became proof: %+v", report.Links[3])
	}
	for _, scope := range report.Requirements {
		want := "not_applicable"
		if scope.ID == "UH-CORE-001" {
			want = "applicable"
		}
		if scope.Applicability != want {
			t.Fatalf("validation ownership: %+v", scope)
		}
	}
}

func TestUnavailableRegistryDoesNotDeclareLinksStale(t *testing.T) {
	report := ReconcileRequirements(RequirementRegistry{}, OwnerUnavailable, "unit", []TestLinks{{Target: Target{TestID: "test"}, IDs: []string{"UH-CORE-001"}, EvidenceKind: Static}})
	if report.UnavailableReason != OwnerUnavailable || len(report.Requirements) != 0 || len(report.Links) != 1 || report.Links[0].Registration != "unknown" || report.Links[0].Execution != ExecutionNotRun {
		t.Fatalf("unavailable owner: %+v", report)
	}
}

func TestDuplicateRegistryCannotEstablishMembership(t *testing.T) {
	report := ReconcileRequirements(RequirementRegistry{SchemaVersion: "requirement-registry/v1", Requirements: []RequirementDeclaration{{ID: "UH-CORE-001"}, {ID: "UH-CORE-001"}}}, ReasonNone, "unit", []TestLinks{{IDs: []string{"UH-CORE-001"}, EvidenceKind: Static}})
	if report.UnavailableReason != ParseFailure || report.Links[0].Registration != "unknown" {
		t.Fatalf("duplicate registry: %+v", report)
	}
}
