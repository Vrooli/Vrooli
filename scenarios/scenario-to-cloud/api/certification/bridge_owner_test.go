package certification

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	bridgeartifacts "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/artifacts"
	bridgeruns "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/runs"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type bridgeAssertionOwnerFake struct {
	run      *bridgeruns.Run
	artifact *bridgeartifacts.GetRunArtifactResponse
	runID    string
	name     string
}

func (f *bridgeAssertionOwnerFake) GetRun(_ context.Context, runID string) (*bridgeruns.Run, error) {
	f.runID = runID
	return f.run, nil
}

func (f *bridgeAssertionOwnerFake) GetRunArtifact(_ context.Context, runID, name string) (*bridgeartifacts.GetRunArtifactResponse, error) {
	f.runID, f.name = runID, name
	return f.artifact, nil
}

func TestVerifyBridgeAssertionOwnerReceiptBindsTerminalNodeRunAndArtifact(t *testing.T) {
	content := []byte(`{"assertion":"green"}`)
	digest := sha256.Sum256(content)
	sha := hex.EncodeToString(digest[:])
	result := DriverResult{Cell: CellID{CaseID: "AUTH-01", Lane: LaneRelay, TargetGround: TargetGroundQEMU}, Target: Target{Ground: TargetGroundQEMU, MachineID: "local-qemu:guest-7", NodeID: "node-7", Generation: 3, Transport: "bridge", Architecture: "amd64", OS: "ubuntu-24.04"}}
	assertion := TestGenieAssertionProof{SHA256: sha, Envelope: TestGenieAssertionArtifact{AssertionOwner: "Vrooli Bridge", OwnerReceiptRef: "bridge-run://run-77/assertion.json"}}
	artifactRef := "device-sync-hub://artifact-77"
	owner := &bridgeAssertionOwnerFake{
		run:      &bridgeruns.Run{Id: "run-77", NodeId: "node-7", Scenario: "scenario-to-cloud", Verb: "run", Status: bridgeruns.RunStatus_RUN_STATUS_PASSED, ExitCode: 0, FinishedAt: timestamppb.New(time.Now().UTC()), ArtifactRefs: []string{artifactRef}},
		artifact: &bridgeartifacts.GetRunArtifactResponse{RunId: "run-77", Name: "assertion.json", Data: content, ArtifactRef: artifactRef},
	}
	proof, err := VerifyBridgeAssertionOwnerReceipt(context.Background(), owner, result, assertion)
	if err != nil {
		t.Fatal(err)
	}
	if owner.runID != "run-77" || owner.name != "assertion.json" || proof.Owner != "Vrooli Bridge" || proof.NodeID != "node-7" || proof.SHA256 != sha || proof.OwnerArtifactRef != artifactRef || proof.ArtifactRef != "bridge-run-artifact://run-77/assertion.json?sha256="+sha {
		t.Fatalf("Bridge proof lost owner-bound identity: %+v", proof)
	}
	if err := validateBridgeAssertionOwnerProof(&proof, result.Target, &assertion); err != nil {
		t.Fatalf("valid Bridge proof failed projection validation: %v", err)
	}
}

func TestVerifyBridgeAssertionOwnerReceiptRefusesWrongTargetOrArtifactBytes(t *testing.T) {
	content := []byte("assertion")
	digest := sha256.Sum256(content)
	sha := hex.EncodeToString(digest[:])
	result := DriverResult{Cell: CellID{CaseID: "AUTH-01", Lane: LaneRelay, TargetGround: TargetGroundQEMU}, Target: Target{Ground: TargetGroundQEMU, NodeID: "node-expected"}}
	assertion := TestGenieAssertionProof{SHA256: sha, Envelope: TestGenieAssertionArtifact{AssertionOwner: "Vrooli Bridge", OwnerReceiptRef: "bridge-run://run-9/a.json"}}
	base := &bridgeAssertionOwnerFake{run: &bridgeruns.Run{Id: "run-9", NodeId: "node-other", Status: bridgeruns.RunStatus_RUN_STATUS_PASSED, ExitCode: 0, FinishedAt: timestamppb.Now(), ArtifactRefs: []string{"ref"}}, artifact: &bridgeartifacts.GetRunArtifactResponse{RunId: "run-9", Name: "a.json", Data: content, ArtifactRef: "ref"}}
	if _, err := VerifyBridgeAssertionOwnerReceipt(context.Background(), base, result, assertion); err == nil || !strings.Contains(err.Error(), "target node") {
		t.Fatalf("wrong-node run accepted: %v", err)
	}
	base.run.NodeId = "node-expected"
	base.artifact.Data = []byte("different")
	if _, err := VerifyBridgeAssertionOwnerReceipt(context.Background(), base, result, assertion); err == nil || !strings.Contains(err.Error(), "do not match") {
		t.Fatalf("mismatched bytes accepted: %v", err)
	}
}
