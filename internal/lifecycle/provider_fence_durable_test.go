package lifecycle

import (
	"context"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"connectrpc.com/connect"
	apilifecycle "github.com/vrooli/api-core/lifecycle"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	commonconnect "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1/commonv1connect"
	"google.golang.org/protobuf/proto"
)

type durableFenceService struct {
	commonconnect.UnimplementedLifecycleMaintenanceServiceHandler
	standing *commonv1.LifecycleStanding
	resumes  atomic.Int32
}

func (s *durableFenceService) Status(context.Context, *connect.Request[commonv1.LifecycleStatusRequest]) (*connect.Response[commonv1.LifecycleStatusResponse], error) {
	return connect.NewResponse(&commonv1.LifecycleStatusResponse{Standing: proto.Clone(s.standing).(*commonv1.LifecycleStanding)}), nil
}
func (s *durableFenceService) Resume(_ context.Context, r *connect.Request[commonv1.LifecycleResumeRequest]) (*connect.Response[commonv1.LifecycleResumeResponse], error) {
	s.resumes.Add(1)
	next := proto.Clone(s.standing).(*commonv1.LifecycleStanding)
	if r.Msg.GetOperationId() != next.GetOperationId() || r.Msg.GetFenceToken() != next.GetFenceToken() || r.Msg.GetRevision() != next.GetRevision() {
		return nil, connect.NewError(connect.CodeAborted, nil)
	}
	next.AdmissionClosed = false
	next.Phase = commonv1.LifecycleMaintenancePhase_LIFECYCLE_MAINTENANCE_PHASE_RESUMED
	next.Revision++
	return connect.NewResponse(&commonv1.LifecycleResumeResponse{Standing: next}), nil
}
func TestProviderFenceResumesOnlyExactDurableOperationAfterReplacement(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(*commonv1.LifecycleStanding, *providerFence)
		want bool
	}{
		{name: "same durable operation", want: true},
		{name: "legacy predecessor with durable successor", want: true, edit: func(_ *commonv1.LifecycleStanding, f *providerFence) {
			f.persistence = commonv1.LifecycleFencePersistence_LIFECYCLE_FENCE_PERSISTENCE_UNSPECIFIED
		}},
		{name: "new process local fence", edit: func(s *commonv1.LifecycleStanding, _ *providerFence) {
			s.FencePersistence = commonv1.LifecycleFencePersistence_LIFECYCLE_FENCE_PERSISTENCE_PROCESS_LOCAL
		}},
		{name: "known process local predecessor", edit: func(_ *commonv1.LifecycleStanding, f *providerFence) {
			f.persistence = commonv1.LifecycleFencePersistence_LIFECYCLE_FENCE_PERSISTENCE_PROCESS_LOCAL
		}},
		{name: "unknown predecessor persistence", edit: func(_ *commonv1.LifecycleStanding, f *providerFence) {
			f.persistence = commonv1.LifecycleFencePersistence(99)
		}},
		{name: "different operation", edit: func(s *commonv1.LifecycleStanding, _ *providerFence) { s.OperationId = "foreign" }},
		{name: "different token", edit: func(s *commonv1.LifecycleStanding, _ *providerFence) { s.FenceToken = "foreign" }},
		{name: "different revision", edit: func(s *commonv1.LifecycleStanding, _ *providerFence) { s.Revision++ }},
		{name: "different interlock", edit: func(s *commonv1.LifecycleStanding, _ *providerFence) { s.Interlock = "foreign" }},
		{name: "missing interlock", edit: func(_ *commonv1.LifecycleStanding, f *providerFence) { f.interlock = "" }},
		{name: "incomplete inventory", edit: func(s *commonv1.LifecycleStanding, _ *providerFence) { s.InventoryComplete = false }},
		{name: "not drained", edit: func(s *commonv1.LifecycleStanding, _ *providerFence) { s.Drained = false }},
		{name: "active work", edit: func(s *commonv1.LifecycleStanding, _ *providerFence) { s.Remaining = 1 }},
		{name: "in flight admission", edit: func(s *commonv1.LifecycleStanding, _ *providerFence) { s.Admitting = 1 }},
		{name: "pending quiescence", edit: func(s *commonv1.LifecycleStanding, _ *providerFence) { s.QuiescencePending = true }},
		{name: "blocker", edit: func(s *commonv1.LifecycleStanding, _ *providerFence) {
			s.Blockers = []*commonv1.LifecycleBlocker{{Code: "unresolved"}}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			standing := &commonv1.LifecycleStanding{Provider: "workflow-health", Scenario: "workflow-health", ProviderInstanceId: "replacement", OperationId: "owned-operation", FenceToken: "7", Revision: 7, Interlock: "scenario-lock-v1", AdmissionClosed: true, InventoryComplete: true, Drained: true, FencePersistence: commonv1.LifecycleFencePersistence_LIFECYCLE_FENCE_PERSISTENCE_OPERATION_DURABLE}
			fence := &providerFence{scenario: "workflow-health", providerInstanceID: "prior", operationID: "owned-operation", token: "7", revision: 7, interlock: "scenario-lock-v1", persistence: commonv1.LifecycleFencePersistence_LIFECYCLE_FENCE_PERSISTENCE_OPERATION_DURABLE, active: true}
			if tc.edit != nil {
				tc.edit(standing, fence)
			}
			service := &durableFenceService{standing: standing}
			_, handler := commonconnect.NewLifecycleMaintenanceServiceHandler(service)
			server := httptest.NewServer(handler)
			defer server.Close()
			client, err := apilifecycle.NewClient(server.Client(), server.URL)
			if err != nil {
				t.Fatal(err)
			}
			fence.client = client
			err = fence.resume(t.Context())
			if (err == nil) != tc.want {
				t.Fatalf("resume err=%v want success=%t", err, tc.want)
			}
			if tc.want {
				if fence.active || service.resumes.Load() != 1 {
					t.Fatal("durable fence was not resumed exactly once")
				}
			} else if !fence.active || service.resumes.Load() != 0 {
				t.Fatal("refused proof replayed a token or relinquished ownership")
			}
		})
	}
}
