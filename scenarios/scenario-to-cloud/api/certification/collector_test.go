package certification

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	operationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/operations"
	bridgeartifacts "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/artifacts"
	bridgeruns "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/runs"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestCollectFullyAuthenticatedDriverResultsWritesOnlyComposedOwnerProof(t *testing.T) {
	result, receipts, artifactOwner, envelope := certificationAssertionFixture(t)
	result.Target = Target{Ground: result.Cell.TargetGround, MachineID: "local-qemu:guest-7", NodeID: "node-7", Generation: 3, Transport: "bridge", Architecture: "amd64", OS: "ubuntu-24.04"}
	envelope.Target = result.Target
	body, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	artifactOwner.body = body
	artifactOwner.meta.SizeBytes = int64(len(body))
	result.OperationRefs = []string{"operation:op-7"}
	result.ObservedAt = testNow
	result.TargetDeploymentID, result.TargetPlanDigest, result.TargetReceiptStep = "dep-1", "sha256:"+strings.Repeat("e", 64), "apply"
	owner := targetReceiptOwnerFunc(func(_ context.Context, deploymentID, operationID, step string) (*operationsv1.TargetReceiptRead, error) {
		if deploymentID != result.TargetDeploymentID || operationID != "op-7" || step != result.TargetReceiptStep {
			t.Fatalf("unexpected owner query %s/%s/%s", deploymentID, operationID, step)
		}
		return &operationsv1.TargetReceiptRead{SchemaVersion: "1", DeploymentId: deploymentID, OperationId: operationID, PlanDigest: result.TargetPlanDigest, Step: step,
			Target:  &operationsv1.TargetReceiptTarget{MachineId: result.Target.MachineID, NodeId: result.Target.NodeID, EnrollmentGeneration: result.Target.Generation, Transport: result.Target.Transport},
			Receipt: &operationsv1.TargetOwnerReceipt{Found: true, Outcome: "succeeded", Fence: 11}}, nil
	})
	report, err := CollectFullyAuthenticatedDriverResults(context.Background(), t.TempDir(), []DriverResult{result}, receipts, artifactOwner, owner)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Recorded) != 1 || report.Recorded[0].Verdict != VerdictPassed {
		t.Fatalf("authenticated result not recorded: %+v", report)
	}
	if len(report.Pending) != 531 {
		t.Fatalf("unexpected pending cell count %d", len(report.Pending))
	}
}

func TestCollectFullyAuthenticatedDriverResultsKeepsCellPendingWhenTargetOwnerRefuses(t *testing.T) {
	result, receipts, artifactOwner, envelope := certificationAssertionFixture(t)
	result.Target = Target{Ground: result.Cell.TargetGround, MachineID: "local-qemu:guest-8", NodeID: "node-8", Generation: 4, Transport: "bridge", Architecture: "amd64", OS: "ubuntu-24.04"}
	envelope.Target = result.Target
	result.OperationRefs = []string{"operation:op-8"}
	result.TargetDeploymentID, result.TargetPlanDigest, result.TargetReceiptStep, result.ObservedAt = "dep-8", "sha256:"+strings.Repeat("f", 64), "apply", testNow
	body, _ := json.Marshal(envelope)
	artifactOwner.body = body
	artifactOwner.meta.SizeBytes = int64(len(body))
	owner := targetReceiptOwnerFunc(func(_ context.Context, deploymentID, operationID, step string) (*operationsv1.TargetReceiptRead, error) {
		return &operationsv1.TargetReceiptRead{SchemaVersion: "1", DeploymentId: deploymentID, OperationId: operationID, PlanDigest: result.TargetPlanDigest, Step: step,
			Target:  &operationsv1.TargetReceiptTarget{MachineId: result.Target.MachineID, NodeId: result.Target.NodeID, EnrollmentGeneration: result.Target.Generation, Transport: result.Target.Transport},
			Receipt: &operationsv1.TargetOwnerReceipt{Found: false}}, nil
	})
	evidenceDir := t.TempDir()
	report, err := CollectFullyAuthenticatedDriverResults(context.Background(), evidenceDir, []DriverResult{result}, receipts, artifactOwner, owner)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Recorded) != 0 || len(report.Pending) != 532 || len(report.Issues) != 1 || !strings.Contains(report.Issues[0].Error, "no successful, fenced receipt") {
		t.Fatalf("target-owner refusal did not stay pending: recorded=%d pending=%d issues=%+v", len(report.Recorded), len(report.Pending), report.Issues)
	}
}

func TestCollectFullyAuthenticatedDriverResultsRejectsDuplicateBeforeOwnerReads(t *testing.T) {
	result, receipts, artifacts, _ := certificationAssertionFixture(t)
	called := false
	targets := targetReceiptOwnerFunc(func(context.Context, string, string, string) (*operationsv1.TargetReceiptRead, error) {
		called = true
		return nil, nil
	})
	report, err := CollectFullyAuthenticatedDriverResults(context.Background(), t.TempDir(), []DriverResult{result, result}, receipts, artifacts, targets)
	if err != nil {
		t.Fatal(err)
	}
	if called || artifacts.listed != "" || receipts.reads != 0 {
		t.Fatalf("duplicate input reached evidence owners: target=%t artifact=%q receipt_reads=%d", called, artifacts.listed, receipts.reads)
	}
	if len(report.Recorded) != 0 || len(report.Pending) != 532 || len(report.Issues) != 1 || !strings.Contains(report.Issues[0].Error, "2 owner results") {
		t.Fatalf("duplicate result was not refused: recorded=%d pending=%d issues=%+v", len(report.Recorded), len(report.Pending), report.Issues)
	}
}

func TestCollectFullyAuthenticatedDriverResultsRequiresBridgeRunForRelayLane(t *testing.T) {
	matrix := mustMatrix(t)
	var relayCell CellID
	for _, cell := range matrix.RequiredCells() {
		if cell.Lane == LaneRelay && cell.TargetGround == TargetGroundQEMU {
			relayCell = cell
			break
		}
	}
	if relayCell.CaseID == "" {
		t.Fatal("matrix has no QEMU relay obligation")
	}
	result, receipts, artifacts, envelope := certificationAssertionFixtureForCell(t, relayCell)
	result.Target = Target{Ground: relayCell.TargetGround, MachineID: "local-qemu:guest-relay", NodeID: "node-relay", Generation: 7, Transport: "bridge", Architecture: "amd64", OS: "ubuntu-24.04"}
	envelope.Target = result.Target
	envelope.OwnerReceiptRef = "bridge-run://bridge-run-77/assertion.json"
	body, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	artifacts.body = body
	artifacts.meta.SizeBytes = int64(len(body))
	artifacts.meta.Metadata["owner_receipt_ref"] = envelope.OwnerReceiptRef
	result.OperationRefs = []string{"operation:relay-op"}
	result.ObservedAt = testNow
	result.TargetDeploymentID, result.TargetPlanDigest, result.TargetReceiptStep = "relay-dep", "sha256:"+strings.Repeat("e", 64), "apply"
	bridgeArtifactRef := "device-sync-hub://relay-assertion"
	bridgeOwner := &bridgeAssertionOwnerFake{
		run:      &bridgeruns.Run{Id: "bridge-run-77", NodeId: result.Target.NodeID, Scenario: "scenario-to-cloud", Verb: "run", Status: bridgeruns.RunStatus_RUN_STATUS_PASSED, ExitCode: 0, FinishedAt: timestamppb.Now(), ArtifactRefs: []string{bridgeArtifactRef}},
		artifact: &bridgeartifacts.GetRunArtifactResponse{RunId: "bridge-run-77", Name: "assertion.json", Data: body, ArtifactRef: bridgeArtifactRef},
	}
	targetOwner := targetReceiptOwnerFunc(func(_ context.Context, deploymentID, operationID, step string) (*operationsv1.TargetReceiptRead, error) {
		return &operationsv1.TargetReceiptRead{SchemaVersion: "1", DeploymentId: deploymentID, OperationId: operationID, PlanDigest: result.TargetPlanDigest, Step: step,
			Target:  &operationsv1.TargetReceiptTarget{MachineId: result.Target.MachineID, NodeId: result.Target.NodeID, EnrollmentGeneration: result.Target.Generation, Transport: result.Target.Transport},
			Receipt: &operationsv1.TargetOwnerReceipt{Found: true, Outcome: "succeeded", Fence: 9}}, nil
	})
	report, err := CollectFullyAuthenticatedDriverResults(context.Background(), t.TempDir(), []DriverResult{result}, receipts, artifacts, targetOwner, AssertionOwnerReaders{Bridge: bridgeOwner})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Recorded) != 1 || report.Recorded[0].Cell != relayCell || report.Recorded[0].Verdict != VerdictPassed {
		var issue string
		if len(report.Issues) > 0 {
			issue = report.Issues[0].Error
		}
		t.Fatalf("valid Bridge-owned relay assertion not collected: recorded=%+v issue=%q", report.Recorded, issue)
	}
}

func TestCollectFullyAuthenticatedDriverResultsComposesBASExecutionAndTargetOwner(t *testing.T) {
	matrix := mustMatrix(t)
	var cell CellID
	for _, required := range matrix.RequiredCells() {
		if required.Lane == LaneBASUI && required.TargetGround == TargetGroundQEMU {
			cell = required
			break
		}
	}
	if cell.CaseID == "" {
		t.Fatal("matrix has no BAS UI QEMU obligation")
	}
	result, receipts, artifacts, envelope := certificationAssertionFixtureForCell(t, cell)
	result.Target = Target{Ground: cell.TargetGround, MachineID: "local-qemu:guest-bas", NodeID: "node-bas", Generation: 12, Transport: "bridge", Architecture: "amd64", OS: "ubuntu-24.04"}
	envelope.Target = result.Target
	body, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	artifacts.body = body
	artifacts.meta.SizeBytes = int64(len(body))
	result.OperationRefs = []string{"operation:bas-op"}
	result.ObservedAt = testNow
	result.TargetDeploymentID, result.TargetPlanDigest, result.TargetReceiptStep = "bas-dep", "sha256:"+strings.Repeat("e", 64), "apply"
	_, _, basOwner := basOwnerFixture(t)
	// The execution fixture is deterministic and the envelope names its node.
	basOwner.execution.Parameters.Env[basTargetMachineParam] = &commonv1.JsonValue{Kind: &commonv1.JsonValue_StringValue{StringValue: result.Target.MachineID}}
	basOwner.execution.Parameters.Env[basTargetNodeParam] = &commonv1.JsonValue{Kind: &commonv1.JsonValue_StringValue{StringValue: result.Target.NodeID}}
	basOwner.execution.Parameters.Env[basTargetGenerationParam] = &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: int64(result.Target.Generation)}}
	targetOwner := targetReceiptOwnerFunc(func(_ context.Context, deploymentID, operationID, step string) (*operationsv1.TargetReceiptRead, error) {
		return &operationsv1.TargetReceiptRead{SchemaVersion: "1", DeploymentId: deploymentID, OperationId: operationID, PlanDigest: result.TargetPlanDigest, Step: step,
			Target:  &operationsv1.TargetReceiptTarget{MachineId: result.Target.MachineID, NodeId: result.Target.NodeID, EnrollmentGeneration: result.Target.Generation, Transport: result.Target.Transport},
			Receipt: &operationsv1.TargetOwnerReceipt{Found: true, Outcome: "succeeded", Fence: 12}}, nil
	})
	report, err := CollectFullyAuthenticatedDriverResults(context.Background(), t.TempDir(), []DriverResult{result}, receipts, artifacts, targetOwner, AssertionOwnerReaders{BAS: basOwner})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Recorded) != 1 || report.Recorded[0].Verdict != VerdictPassed || len(report.Pending) != 531 {
		t.Fatalf("authenticated BAS result not recorded: recorded=%d pending=%d issues=%+v", len(report.Recorded), len(report.Pending), report.Issues)
	}
}

func TestCollectFullyAuthenticatedDriverResultsRequiresBASReaderBeforeOwnerReads(t *testing.T) {
	matrix := mustMatrix(t)
	var cell CellID
	for _, required := range matrix.RequiredCells() {
		if required.Lane == LaneBASUI && required.TargetGround == TargetGroundQEMU {
			cell = required
			break
		}
	}
	result, receipts, artifacts, _ := certificationAssertionFixtureForCell(t, cell)
	targetCalled := false
	targets := targetReceiptOwnerFunc(func(context.Context, string, string, string) (*operationsv1.TargetReceiptRead, error) {
		targetCalled = true
		return nil, nil
	})
	report, err := CollectFullyAuthenticatedDriverResults(context.Background(), t.TempDir(), []DriverResult{result}, receipts, artifacts, targets)
	if err != nil {
		t.Fatal(err)
	}
	if receipts.reads != 0 || artifacts.listed != "" || targetCalled || len(report.Recorded) != 0 || len(report.Pending) != 532 || len(report.Issues) != 1 {
		t.Fatalf("missing BAS reader did not fail closed before owner reads: reads=%d artifact=%q target=%t report=%+v", receipts.reads, artifacts.listed, targetCalled, report)
	}
}

func TestCollectFullyAuthenticatedDriverResultsFailsClosedForUnimplementedLaneOwner(t *testing.T) {
	matrix := mustMatrix(t)
	var cell CellID
	for _, required := range matrix.RequiredCells() {
		if required.Lane == LaneIndependentReview && required.TargetGround == TargetGroundQEMU {
			cell = required
			break
		}
	}
	if cell.CaseID == "" {
		t.Fatal("matrix has no reviewer QEMU obligation")
	}
	result, receipts, artifacts, _ := certificationAssertionFixtureForCell(t, cell)
	targetRead := false
	targets := targetReceiptOwnerFunc(func(context.Context, string, string, string) (*operationsv1.TargetReceiptRead, error) {
		targetRead = true
		return nil, nil
	})
	report, err := CollectFullyAuthenticatedDriverResults(context.Background(), t.TempDir(), []DriverResult{result}, receipts, artifacts, targets)
	if err != nil {
		t.Fatal(err)
	}
	if receipts.reads != 0 || artifacts.listed != "" || targetRead {
		t.Fatalf("unimplemented lane reached an evidence owner: receipt reads=%d artifact=%q target=%t", receipts.reads, artifacts.listed, targetRead)
	}
	if len(report.Recorded) != 0 || len(report.Pending) != 532 || len(report.Issues) != 1 || !strings.Contains(report.Issues[0].Error, "adapter is not implemented") {
		t.Fatalf("unimplemented owner lane did not remain unavailable: %+v", report.Issues)
	}
}

func syntheticDriverResults(t *testing.T, cells []CellID) []DriverResult {
	t.Helper()
	m := mustMatrix(t)
	results := make([]DriverResult, 0, len(cells))
	for i, cell := range cells {
		caseRow, ok := m.Case(cell.CaseID)
		if !ok {
			t.Fatalf("unknown test case %s", cell.CaseID)
		}
		ref, err := driverRefForMatrix(m, cell)
		if err != nil {
			t.Fatalf("driver ref for %s: %v", cell, err)
		}
		results = append(results, DriverResult{
			Cell: cell, DriverRef: ref, Verdict: VerdictUnavailable,
			Candidate: testCandidate(),
			Target: Target{
				Ground: cell.TargetGround, MachineID: fmt.Sprintf("fixture-%s-%d", cell.TargetGround, i),
				Generation: 1, Architecture: "amd64", OS: "ubuntu-24.04",
			},
			ValidationReceiptRef: "test-genie:synthetic-collector-unit-test",
			OperationRefs:        []string{fmt.Sprintf("operation:synthetic-%d", i)},
			TargetReceiptRefs:    []string{fmt.Sprintf("target-receipt://synthetic-%d/apply?fence=9", i)},
			AssertionArtifacts:   []string{fmt.Sprintf("assertion-artifact:%s:sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ref)},
			Assertion:            caseRow.Assertion,
			ObservedAt:           testNow,
		})
	}
	return results
}

func TestCollectorPersistsAll532UnavailableResultsWithoutImplicitPasses(t *testing.T) {
	dir := t.TempDir()
	cells := mustMatrix(t).RequiredCells()
	if len(cells) != 532 {
		t.Fatalf("required cells = %d, want 532", len(cells))
	}
	report, err := CollectDriverResults(dir, syntheticDriverResults(t, cells))
	if err != nil {
		t.Fatal(err)
	}
	if report.Required != 532 || len(report.Recorded) != 532 || len(report.Pending) != 0 || len(report.Issues) != 0 {
		t.Fatalf("collector report does not account for all 532 results: required=%d recorded=%d pending=%d issues=%d", report.Required, len(report.Recorded), len(report.Pending), len(report.Issues))
	}

	receipts, err := LoadEvidenceDir(dir, mustMatrix(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(receipts) != 532 {
		t.Fatalf("stored receipts = %d, want 532", len(receipts))
	}
	readiness := Evaluate(mustMatrix(t), receipts, testCandidate(), testNow)
	if readiness.Ready || len(readiness.SatisfiedCells) != 0 {
		t.Fatalf("unavailable synthetic receipts must not satisfy cells: ready=%t satisfied=%d", readiness.Ready, len(readiness.SatisfiedCells))
	}
}

func TestCollectorCannotPersistPassedResultWithoutOwnerProofReaders(t *testing.T) {
	cell := mustMatrix(t).RequiredCells()[0]
	result := syntheticDriverResults(t, []CellID{cell})[0]
	result.Verdict = VerdictPassed
	report, err := CollectDriverResults(t.TempDir(), []DriverResult{result})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Recorded) != 0 || len(report.Pending) != 532 || len(report.Issues) != 1 || !strings.Contains(report.Issues[0].Error, "authenticated target-owner receipt and assertion-artifact bytes") {
		t.Fatalf("unverified pass was accepted: recorded=%d pending=%d issues=%+v", len(report.Recorded), len(report.Pending), report.Issues)
	}
}

func TestCollectorLeavesMissingDuplicateAndInvalidResultsPending(t *testing.T) {
	cells := mustMatrix(t).RequiredCells()
	results := syntheticDriverResults(t, cells[:1])
	results = append(results, results[0])
	invalid := syntheticDriverResults(t, cells[1:2])[0]
	invalid.Assertion = "substituted assertion"
	results = append(results, invalid)

	report, err := CollectDriverResults(t.TempDir(), results)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Recorded) != 0 || len(report.Pending) != 532 || len(report.Issues) != 2 {
		t.Fatalf("invalid/duplicate results changed cell standing: recorded=%d pending=%d issues=%d", len(report.Recorded), len(report.Pending), len(report.Issues))
	}
	issueText := report.Issues[0].Error + report.Issues[1].Error
	if !strings.Contains(issueText, "received 2 owner results") || !strings.Contains(issueText, "frozen matrix assertion") {
		t.Fatalf("expected duplicate and assertion mismatch issues, got %+v", report.Issues)
	}
}
