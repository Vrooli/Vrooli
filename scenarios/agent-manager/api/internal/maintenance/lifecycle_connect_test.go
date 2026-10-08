package maintenance

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	apilifecycle "github.com/vrooli/api-core/lifecycle"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	commonconnect "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1/commonv1connect"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestLifecycleConnectFenceDrainResumeAndStaleResume(t *testing.T) {
	gate, _ := openGate(t, t.TempDir()+"/lifecycle.db")
	release, err := gate.Admit(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	remaining := 1
	service, err := NewLifecycleService(gate, func(context.Context) (Inventory, error) {
		return Inventory{Remaining: &remaining}, nil
	}, "agent-manager", "agent-manager", "instance-1")
	if err != nil {
		t.Fatal(err)
	}
	path, handler := commonconnect.NewLifecycleMaintenanceServiceHandler(service)
	_ = path
	server := httptest.NewServer(handler)
	defer server.Close()
	client, err := apilifecycle.NewClient(server.Client(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	operation := "operation-1"
	prepared, err := client.Prepare(t.Context(), &commonv1.LifecyclePrepareRequest{OperationId: operation, Reason: "test replacement"})
	if err != nil {
		t.Fatal(err)
	}
	if !prepared.GetStanding().GetAdmissionClosed() || prepared.GetStanding().GetDrained() {
		t.Fatalf("prepare standing = %+v", prepared.GetStanding())
	}
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()
	if _, err := client.Drain(ctx, &commonv1.LifecycleDrainRequest{OperationId: operation, Revision: prepared.GetStanding().GetRevision(), Timeout: durationpb.New(20 * time.Millisecond)}); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("blocked drain error = %v", err)
	}
	remaining = 0
	release()
	drained, err := client.Drain(t.Context(), &commonv1.LifecycleDrainRequest{OperationId: operation, FenceToken: prepared.GetStanding().GetFenceToken(), Revision: prepared.GetStanding().GetRevision()})
	if err != nil || !drained.GetStanding().GetDrained() {
		t.Fatalf("drain standing = %+v err=%v", drained.GetStanding(), err)
	}
	resumed, err := client.Resume(t.Context(), &commonv1.LifecycleResumeRequest{OperationId: operation, FenceToken: drained.GetStanding().GetFenceToken(), Revision: drained.GetStanding().GetRevision()})
	if err != nil || resumed.GetStanding().GetAdmissionClosed() {
		t.Fatalf("resume standing = %+v err=%v", resumed.GetStanding(), err)
	}
	if _, err := client.Resume(t.Context(), &commonv1.LifecycleResumeRequest{OperationId: operation, FenceToken: drained.GetStanding().GetFenceToken(), Revision: drained.GetStanding().GetRevision()}); connect.CodeOf(err) != connect.CodeAborted {
		t.Fatalf("stale resume code = %v", connect.CodeOf(err))
	}
}

func TestLifecycleConnectEmergencyOverrideRequiresReasonAndDoesNotDrain(t *testing.T) {
	gate, _ := openGate(t, t.TempDir()+"/lifecycle-emergency.db")
	remaining := 1
	service, err := NewLifecycleService(gate, func(context.Context) (Inventory, error) {
		return Inventory{Remaining: &remaining}, nil
	}, "agent-manager", "agent-manager", "instance-1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Prepare(t.Context(), connect.NewRequest(&commonv1.LifecyclePrepareRequest{OperationId: "operation-2", Reason: "emergency", EmergencyOverride: true})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("missing override reason code = %v", connect.CodeOf(err))
	}
	prepared, err := service.Prepare(t.Context(), connect.NewRequest(&commonv1.LifecyclePrepareRequest{OperationId: "operation-2", Reason: "emergency: operator decision", EmergencyOverride: true, OverrideReason: "operator decision"}))
	if err != nil {
		t.Fatal(err)
	}
	if prepared.Msg.GetStanding().GetDrained() || !prepared.Msg.GetStanding().GetAdmissionClosed() {
		t.Fatalf("emergency prepare incorrectly drained: %+v", prepared.Msg.GetStanding())
	}
}

func TestLifecycleConnectEmergencyOverrideAdoptsStaleClosedFence(t *testing.T) {
	gate, _ := openGate(t, t.TempDir()+"/lifecycle-stale-fence.db")
	if _, err := gate.Enter(t.Context(), "control-plane", "abandoned operation"); err != nil {
		t.Fatal(err)
	}
	service, err := NewLifecycleService(gate, func(context.Context) (Inventory, error) {
		return Inventory{Unknown: []string{"startup recovery degraded"}}, nil
	}, "agent-manager", "agent-manager", "instance-1")
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := service.Prepare(t.Context(), connect.NewRequest(&commonv1.LifecyclePrepareRequest{
		OperationId: "replacement-operation", Reason: "recovery-first replacement", EmergencyOverride: true, OverrideReason: "recover abandoned lifecycle fence",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if prepared.Msg.GetStanding().GetRevision() != 1 || !prepared.Msg.GetStanding().GetAdmissionClosed() || prepared.Msg.GetStanding().GetInventoryComplete() {
		t.Fatalf("stale fence adoption standing = %+v", prepared.Msg.GetStanding())
	}
}

func TestLifecycleConnectFailedPrepareDoesNotLeaveFence(t *testing.T) {
	gate, _ := openGate(t, t.TempDir()+"/lifecycle-failed-prepare.db")
	observeErr := true
	service, err := NewLifecycleService(gate, func(context.Context) (Inventory, error) {
		if observeErr {
			return Inventory{}, errors.New("startup recovery is not ready")
		}
		remaining := 0
		return Inventory{Remaining: &remaining}, nil
	}, "agent-manager", "agent-manager", "instance-1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Prepare(t.Context(), connect.NewRequest(&commonv1.LifecyclePrepareRequest{OperationId: "failed-operation", Reason: "startup"})); err == nil {
		t.Fatal("failed inventory unexpectedly prepared")
	}
	observeErr = false
	prepared, err := service.Prepare(t.Context(), connect.NewRequest(&commonv1.LifecyclePrepareRequest{OperationId: "replacement-operation", Reason: "retry"}))
	if err != nil {
		t.Fatal(err)
	}
	if prepared.Msg.GetStanding().GetOperationId() != "replacement-operation" {
		t.Fatalf("replacement operation=%q", prepared.Msg.GetStanding().GetOperationId())
	}
}

func TestLifecycleConnectResumeKeepsSuccessfulOpenWhenObservationFails(t *testing.T) {
	gateway, _ := openGate(t, t.TempDir()+"/lifecycle-resume-observation.db")
	observations := 0
	service, err := NewLifecycleService(gateway, func(context.Context) (Inventory, error) {
		observations++
		if observations > 1 {
			return Inventory{}, errors.New("physical executor inventory is incomplete")
		}
		remaining := 0
		return Inventory{Remaining: &remaining}, nil
	}, "agent-manager", "agent-manager", "instance-1")
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := service.Prepare(t.Context(), connect.NewRequest(&commonv1.LifecyclePrepareRequest{OperationId: "resume-observation", Reason: "test"}))
	if err != nil {
		t.Fatal(err)
	}
	resumed, err := service.Resume(t.Context(), connect.NewRequest(&commonv1.LifecycleResumeRequest{OperationId: "resume-observation", Revision: prepared.Msg.GetStanding().GetRevision()}))
	if err != nil {
		t.Fatalf("resume returned error after durable reopen: %v", err)
	}
	if resumed.Msg.GetStanding().GetAdmissionClosed() || resumed.Msg.GetStanding().GetInventoryComplete() {
		t.Fatalf("resume falsely reported closed/complete standing: %+v", resumed.Msg.GetStanding())
	}
	state, err := gateway.Status(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if state.Closed {
		t.Fatal("resume observation failure left admission closed")
	}
}
