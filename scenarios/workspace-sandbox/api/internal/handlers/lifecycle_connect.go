package handlers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"connectrpc.com/connect"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	commonconnect "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1/commonv1connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	"workspace-sandbox/internal/process"
)

type providerLifecycle struct {
	commonconnect.UnimplementedLifecycleMaintenanceServiceHandler
	tracker                      *process.Tracker
	provider, scenario, instance string
	mu                           sync.Mutex
	changed                      chan struct{}
	closed                       bool
	operation                    string
	revision                     int64
	admitting                    int
}

func newProviderLifecycle(tracker *process.Tracker, provider, scenario, instance string) *providerLifecycle {
	return &providerLifecycle{tracker: tracker, provider: provider, scenario: scenario, instance: instance, changed: make(chan struct{})}
}

func (g *providerLifecycle) beginProcess() (func(), error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.closed {
		return nil, fmt.Errorf("provider admission is closed for lifecycle maintenance")
	}
	g.admitting++
	var once sync.Once
	return func() {
		once.Do(func() { g.mu.Lock(); g.admitting--; close(g.changed); g.changed = make(chan struct{}); g.mu.Unlock() })
	}, nil
}

func (g *providerLifecycle) standingLocked(operation string) *commonv1.LifecycleStanding {
	running := 0
	if g.tracker != nil {
		running = g.tracker.GetAllStats().TotalRunning
	}
	drained := g.closed && running == 0 && g.admitting == 0
	phase := commonv1.LifecycleMaintenancePhase_LIFECYCLE_MAINTENANCE_PHASE_CLOSING
	if !g.closed {
		phase = commonv1.LifecycleMaintenancePhase_LIFECYCLE_MAINTENANCE_PHASE_OPEN
	}
	if drained {
		phase = commonv1.LifecycleMaintenancePhase_LIFECYCLE_MAINTENANCE_PHASE_DRAINED
	}
	return &commonv1.LifecycleStanding{
		Provider: g.provider, ProviderInstanceId: g.instance, Scenario: g.scenario,
		InstanceId: g.instance, OperationId: operation, FenceToken: fmt.Sprintf("%d", g.revision), Revision: g.revision,
		Phase: phase, AdmissionClosed: g.closed, Admitting: int32(g.admitting), Remaining: int32(running),
		InventoryComplete: true, Drained: drained, ObservedAt: timestamppb.Now(), Interlock: "scenario-lock-v1",
	}
}

func (g *providerLifecycle) Prepare(_ context.Context, req *connect.Request[commonv1.LifecyclePrepareRequest]) (*connect.Response[commonv1.LifecyclePrepareResponse], error) {
	if req.Msg.GetOperationId() == "" || req.Msg.GetReason() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("operation_id and reason are required"))
	}
	if req.Msg.GetEmergencyOverride() && req.Msg.GetOverrideReason() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("override_reason is required for emergency override"))
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.closed && g.operation != req.Msg.GetOperationId() {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("another lifecycle operation is fenced"))
	}
	g.closed, g.operation, g.revision = true, req.Msg.GetOperationId(), g.revision+1
	return connect.NewResponse(&commonv1.LifecyclePrepareResponse{Standing: g.standingLocked(g.operation)}), nil
}

func (g *providerLifecycle) Status(_ context.Context, req *connect.Request[commonv1.LifecycleStatusRequest]) (*connect.Response[commonv1.LifecycleStatusResponse], error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return connect.NewResponse(&commonv1.LifecycleStatusResponse{Standing: g.standingLocked(req.Msg.GetOperationId())}), nil
}

func (g *providerLifecycle) Drain(ctx context.Context, req *connect.Request[commonv1.LifecycleDrainRequest]) (*connect.Response[commonv1.LifecycleDrainResponse], error) {
	if req.Msg.GetOperationId() == "" || req.Msg.GetRevision() <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("operation_id and revision are required"))
	}
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		g.mu.Lock()
		if g.operation != req.Msg.GetOperationId() || g.revision != req.Msg.GetRevision() || !g.closed {
			g.mu.Unlock()
			return nil, connect.NewError(connect.CodeAborted, fmt.Errorf("lifecycle fence does not match"))
		}
		standing := g.standingLocked(req.Msg.GetOperationId())
		changed := g.changed
		g.mu.Unlock()
		if standing.GetDrained() {
			return connect.NewResponse(&commonv1.LifecycleDrainResponse{Standing: standing}), nil
		}
		select {
		case <-ctx.Done():
			return nil, connect.NewError(connect.CodeDeadlineExceeded, ctx.Err())
		case <-changed:
		case <-ticker.C:
		}
	}
}

func (g *providerLifecycle) Resume(_ context.Context, req *connect.Request[commonv1.LifecycleResumeRequest]) (*connect.Response[commonv1.LifecycleResumeResponse], error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.closed || g.operation != req.Msg.GetOperationId() || g.revision != req.Msg.GetRevision() {
		return nil, connect.NewError(connect.CodeAborted, fmt.Errorf("lifecycle fence does not match"))
	}
	g.closed, g.operation, g.revision = false, "", g.revision+1
	return connect.NewResponse(&commonv1.LifecycleResumeResponse{Standing: g.standingLocked(req.Msg.GetOperationId())}), nil
}

var _ commonconnect.LifecycleMaintenanceServiceHandler = (*providerLifecycle)(nil)
