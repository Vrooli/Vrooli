package handlers

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/schedule"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"workspace-sandbox/internal/process"
)

func TestProviderLifecycleClosesAdmissionAndDrainsTrackedProcess(t *testing.T) {
	tracker := process.NewTracker(schedule.System())
	provider := newProviderLifecycle(tracker, "workspace-sandbox", "workspace-sandbox", "instance-1")
	release, err := provider.beginProcess()
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := provider.Prepare(t.Context(), connect.NewRequest(&commonv1.LifecyclePrepareRequest{OperationId: "operation-1", Reason: "test replacement"}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.beginProcess(); err == nil {
		t.Fatal("new admission was accepted after prepare")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()
	if _, err := provider.Drain(ctx, connect.NewRequest(&commonv1.LifecycleDrainRequest{OperationId: "operation-1", Revision: prepared.Msg.GetStanding().GetRevision()})); connect.CodeOf(err) != connect.CodeDeadlineExceeded {
		t.Fatalf("blocked drain code = %v err=%v", connect.CodeOf(err), err)
	}
	release()
	drained, err := provider.Drain(t.Context(), connect.NewRequest(&commonv1.LifecycleDrainRequest{OperationId: "operation-1", Revision: prepared.Msg.GetStanding().GetRevision()}))
	if err != nil || !drained.Msg.GetStanding().GetDrained() {
		t.Fatalf("drain standing = %+v err=%v", drained.Msg.GetStanding(), err)
	}
	if _, err := provider.Resume(t.Context(), connect.NewRequest(&commonv1.LifecycleResumeRequest{OperationId: "operation-1", Revision: prepared.Msg.GetStanding().GetRevision() - 1})); connect.CodeOf(err) != connect.CodeAborted {
		t.Fatalf("stale resume code = %v", connect.CodeOf(err))
	}
	if _, err := provider.Resume(t.Context(), connect.NewRequest(&commonv1.LifecycleResumeRequest{OperationId: "operation-1", Revision: prepared.Msg.GetStanding().GetRevision()})); err != nil {
		t.Fatal(err)
	}
}

func TestProviderLifecycleEmergencyOverrideRequiresReason(t *testing.T) {
	provider := newProviderLifecycle(nil, "workspace-sandbox", "workspace-sandbox", "instance-1")
	if _, err := provider.Prepare(t.Context(), connect.NewRequest(&commonv1.LifecyclePrepareRequest{OperationId: "operation-2", Reason: "emergency", EmergencyOverride: true})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("missing override reason code = %v", connect.CodeOf(err))
	}
}
