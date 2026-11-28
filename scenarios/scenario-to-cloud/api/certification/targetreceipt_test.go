package certification

import (
	"context"
	"strings"
	"testing"

	operationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/operations"
)

type targetReceiptOwnerFunc func(context.Context, string, string, string) (*operationsv1.TargetReceiptRead, error)

func (f targetReceiptOwnerFunc) GetTargetReceipt(ctx context.Context, deploymentID, operationID, step string) (*operationsv1.TargetReceiptRead, error) {
	return f(ctx, deploymentID, operationID, step)
}

func TestVerifyTargetOwnerReceiptBindsFreshReceiptToGroundOperationAndTarget(t *testing.T) {
	expected := TargetReceiptExpectation{
		Cell:      CellID{CaseID: "RUN-01", Lane: LanePackage, TargetGround: TargetGroundQEMU},
		DriverRef: "certification-assertion:v1:RUN-01:package:qemu", DeploymentID: "dep-1",
		OperationID: "op-7", PlanDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Step: "release.activate", Target: Target{Ground: TargetGroundQEMU, MachineID: "local-qemu:guest-7", NodeID: "node-qemu-7", Generation: 12, Transport: "bridge", Architecture: "amd64", OS: "ubuntu-24.04"},
	}
	owner := targetReceiptOwnerFunc(func(_ context.Context, deploymentID, operationID, step string) (*operationsv1.TargetReceiptRead, error) {
		if deploymentID != expected.DeploymentID || operationID != expected.OperationID || step != expected.Step {
			t.Fatalf("owner query = %s/%s/%s", deploymentID, operationID, step)
		}
		return &operationsv1.TargetReceiptRead{
			SchemaVersion: "1", DeploymentId: expected.DeploymentID, OperationId: expected.OperationID,
			PlanDigest: expected.PlanDigest, Step: expected.Step,
			Target:  &operationsv1.TargetReceiptTarget{MachineId: expected.Target.MachineID, NodeId: expected.Target.NodeID, EnrollmentGeneration: expected.Target.Generation, Transport: expected.Target.Transport},
			Receipt: &operationsv1.TargetOwnerReceipt{Found: true, Outcome: "succeeded", Fence: 19, Detail: "must not be persisted"},
		}, nil
	})
	proof, err := VerifyTargetOwnerReceipt(context.Background(), owner, expected)
	if err != nil {
		t.Fatal(err)
	}
	if proof.DeploymentID != expected.DeploymentID || proof.OperationID != expected.OperationID || proof.PlanDigest != expected.PlanDigest || proof.Step != expected.Step || proof.TargetGround != expected.Cell.TargetGround || proof.MachineID != expected.Target.MachineID || proof.NodeID != expected.Target.NodeID || proof.EnrollmentGeneration != expected.Target.Generation || proof.Transport != expected.Target.Transport || proof.Outcome != "succeeded" || proof.Fence != 19 {
		t.Fatalf("target proof lost owner identity: %+v", proof)
	}
	if !strings.Contains(proof.ArtifactRef, "target-receipt://op-7/release.activate?") || !strings.Contains(proof.ArtifactRef, "fence=19") || !strings.Contains(proof.ArtifactRef, "sha256="+proof.SHA256) || strings.Contains(proof.ArtifactRef, "must not be persisted") {
		t.Fatalf("target proof ref is not canonical or redacted: %+v", proof)
	}
}

func TestVerifyTargetOwnerReceiptRefusesMismatchedOrUnsuccessfulOwnerResults(t *testing.T) {
	expected := TargetReceiptExpectation{
		Cell:      CellID{CaseID: "RUN-01", Lane: LaneRealVPS, TargetGround: TargetGroundRealVPS},
		DriverRef: "certification-assertion:v1:RUN-01:real_vps:real_vps", DeploymentID: "dep-stage",
		OperationID: "op-9", PlanDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		Step: "secrets.rotate", Target: Target{Ground: TargetGroundRealVPS, MachineID: "disposable-vps:stage-9", NodeID: "node-9", Generation: 4, Transport: "bridge", Architecture: "amd64", OS: "ubuntu-24.04"},
	}
	valid := func() *operationsv1.TargetReceiptRead {
		return &operationsv1.TargetReceiptRead{
			SchemaVersion: "1", DeploymentId: expected.DeploymentID, OperationId: expected.OperationID,
			PlanDigest: expected.PlanDigest, Step: expected.Step,
			Target:  &operationsv1.TargetReceiptTarget{MachineId: expected.Target.MachineID, NodeId: expected.Target.NodeID, EnrollmentGeneration: expected.Target.Generation, Transport: expected.Target.Transport},
			Receipt: &operationsv1.TargetOwnerReceipt{Found: true, Outcome: "unchanged", Fence: 22},
		}
	}
	tests := []struct {
		name    string
		mutate  func(*operationsv1.TargetReceiptRead, *TargetReceiptExpectation)
		wantErr string
	}{
		{name: "wrong deployment", mutate: func(r *operationsv1.TargetReceiptRead, _ *TargetReceiptExpectation) { r.DeploymentId = "other" }, wantErr: "envelope does not match"},
		{name: "wrong plan", mutate: func(r *operationsv1.TargetReceiptRead, _ *TargetReceiptExpectation) { r.PlanDigest = "sha256:other" }, wantErr: "envelope does not match"},
		{name: "wrong node", mutate: func(r *operationsv1.TargetReceiptRead, _ *TargetReceiptExpectation) { r.Target.NodeId = "other-node" }, wantErr: "identity does not match"},
		{name: "missing receipt", mutate: func(r *operationsv1.TargetReceiptRead, _ *TargetReceiptExpectation) { r.Receipt.Found = false }, wantErr: "no successful, fenced receipt"},
		{name: "failed receipt", mutate: func(r *operationsv1.TargetReceiptRead, _ *TargetReceiptExpectation) { r.Receipt.Outcome = "failed" }, wantErr: "no successful, fenced receipt"},
		{name: "zero fence", mutate: func(r *operationsv1.TargetReceiptRead, _ *TargetReceiptExpectation) { r.Receipt.Fence = 0 }, wantErr: "no successful, fenced receipt"},
		{name: "ground relabel", mutate: func(_ *operationsv1.TargetReceiptRead, e *TargetReceiptExpectation) {
			e.Target.MachineID = "local-qemu:wrong-ground"
		}, wantErr: "not owned by target ground"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			expectation := expected
			read := valid()
			test.mutate(read, &expectation)
			owner := targetReceiptOwnerFunc(func(context.Context, string, string, string) (*operationsv1.TargetReceiptRead, error) {
				return read, nil
			})
			_, err := VerifyTargetOwnerReceipt(context.Background(), owner, expectation)
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("error = %v, want %q", err, test.wantErr)
			}
		})
	}
}
