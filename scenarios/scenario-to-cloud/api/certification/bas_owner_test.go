package certification

import (
	"context"
	"strings"
	"testing"

	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type basAssertionOwnerFake struct {
	execution *basexecution.Execution
	timeline  *bastimeline.ExecutionTimeline
	gotExec   string
	gotTime   string
}

func (f *basAssertionOwnerFake) GetExecution(_ context.Context, id string) (*basexecution.Execution, error) {
	f.gotExec = id
	return f.execution, nil
}

func (f *basAssertionOwnerFake) GetExecutionTimeline(_ context.Context, id string) (*bastimeline.ExecutionTimeline, error) {
	f.gotTime = id
	return f.timeline, nil
}

func TestVerifyBASAssertionOwnerReceiptBindsExecutionTargetAndAssertionEvent(t *testing.T) {
	result, assertion, owner := basOwnerFixture(t)
	const executionID = "00000000-0000-4000-8000-000000000077"
	const nodeID = "00000000-0000-4000-8000-000000000078"
	proof, err := VerifyBASAssertionOwnerReceipt(context.Background(), owner, result, assertion)
	if err != nil {
		t.Fatal(err)
	}
	if owner.gotExec != executionID || owner.gotTime != executionID || proof.NodeID != nodeID || proof.TargetMachine != result.Target.MachineID || proof.TargetNode != result.Target.NodeID || proof.TargetGeneration != result.Target.Generation {
		t.Fatalf("BAS owner proof lost execution or target identity: %+v (reads %q/%q)", proof, owner.gotExec, owner.gotTime)
	}
	if err := validateBASAssertionOwnerProof(&proof, result.Target, &assertion); err != nil {
		t.Fatalf("valid BAS owner proof failed projection validation: %v", err)
	}
}

func TestVerifyBASAssertionOwnerReceiptRefusesTargetDriftDuplicateAndFailedAssertion(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(DriverResult, *basAssertionOwnerFake)
		want   string
	}{
		{"target drift", func(_ DriverResult, owner *basAssertionOwnerFake) {
			owner.execution.Parameters.Env[basTargetNodeParam] = &commonv1.JsonValue{Kind: &commonv1.JsonValue_StringValue{StringValue: "other-node"}}
		}, "target environment"},
		{"failed assertion", func(_ DriverResult, owner *basAssertionOwnerFake) {
			owner.timeline.Entries[0].Context.Assertion.Success = false
		}, "no successful assertion"},
		{"duplicate event", func(_ DriverResult, owner *basAssertionOwnerFake) {
			duplicate := *owner.timeline.Entries[0]
			owner.timeline.Entries = append(owner.timeline.Entries, &duplicate)
		}, "duplicate timeline"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, assertion, owner := basOwnerFixture(t)
			tc.mutate(result, owner)
			if _, err := VerifyBASAssertionOwnerReceipt(context.Background(), owner, result, assertion); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("owner evidence should fail with %q, got %v", tc.want, err)
			}
		})
	}
}

func basOwnerFixture(t *testing.T) (DriverResult, TestGenieAssertionProof, *basAssertionOwnerFake) {
	t.Helper()
	const executionID = "00000000-0000-4000-8000-000000000077"
	const workflowID = "00000000-0000-4000-8000-000000000079"
	const nodeID = "00000000-0000-4000-8000-000000000078"
	result := DriverResult{Cell: CellID{CaseID: "BAS-01", Lane: LaneBASUI, TargetGround: TargetGroundQEMU}, Target: Target{Ground: TargetGroundQEMU, MachineID: "local-qemu:guest-7", NodeID: "node-7", Generation: 3, Transport: "bridge", Architecture: "amd64", OS: "ubuntu-24.04"}}
	assertion := TestGenieAssertionProof{Envelope: TestGenieAssertionArtifact{AssertionOwner: "Browser Automation Studio", OwnerReceiptRef: "bas-execution://" + executionID + "/" + nodeID}}
	completed := timestamppb.Now()
	env := map[string]*commonv1.JsonValue{
		basTargetMachineParam:    {Kind: &commonv1.JsonValue_StringValue{StringValue: result.Target.MachineID}},
		basTargetNodeParam:       {Kind: &commonv1.JsonValue_StringValue{StringValue: result.Target.NodeID}},
		basTargetGenerationParam: {Kind: &commonv1.JsonValue_IntValue{IntValue: int64(result.Target.Generation)}},
	}
	execution := &basexecution.Execution{ExecutionId: executionID, WorkflowId: workflowID, Status: basbase.ExecutionStatus_EXECUTION_STATUS_COMPLETED, CompletedAt: completed, Parameters: &basexecution.ExecutionParameters{Env: env}, Result: &basexecution.ExecutionResult{Success: true}}
	node := nodeID
	entry := &bastimeline.TimelineEntry{Id: "event-1", NodeId: &node, Context: &basbase.EventContext{Origin: &basbase.EventContext_ExecutionId{ExecutionId: executionID}, Success: new(true), Assertion: &basbase.AssertionResult{Mode: basbase.AssertionMode_ASSERTION_MODE_VISIBLE, Selector: "[data-testid=ready]", Success: true}}}
	owner := &basAssertionOwnerFake{execution: execution, timeline: &bastimeline.ExecutionTimeline{ExecutionId: executionID, WorkflowId: workflowID, Status: basbase.ExecutionStatus_EXECUTION_STATUS_COMPLETED, CompletedAt: completed, Entries: []*bastimeline.TimelineEntry{entry}}}
	return result, assertion, owner
}
