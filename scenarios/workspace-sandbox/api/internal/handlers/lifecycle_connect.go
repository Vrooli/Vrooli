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
	"workspace-sandbox/internal/repository"
)

type providerLifecycle struct {
	commonconnect.UnimplementedLifecycleMaintenanceServiceHandler
	tracker                      *process.Tracker
	store                        lifecycleFenceStore
	provider, scenario, instance string
	mu                           sync.Mutex
	changed                      chan struct{}
	closed                       bool
	operation                    string
	revision                     int64
	admitting                    int
}

type lifecycleFenceStore interface {
	GetProviderLifecycleFence(context.Context) (repository.ProviderLifecycleFence, error)
	CompareAndSwapProviderLifecycleFence(context.Context, int64, repository.ProviderLifecycleFence) error
}

func newProviderLifecycle(ctx context.Context, tracker *process.Tracker, store lifecycleFenceStore, provider, scenario, instance string) (*providerLifecycle, error) {
	state, err := store.GetProviderLifecycleFence(ctx)
	if err != nil {
		return nil, fmt.Errorf("initialize provider lifecycle fence: %w", err)
	}
	if state.Revision < 0 || state.AdmissionClosed != (state.OperationID != "") || (state.AdmissionClosed && state.Revision == 0) {
		return nil, fmt.Errorf("initialize provider lifecycle fence: invalid durable state")
	}
	return &providerLifecycle{tracker: tracker, store: store, provider: provider, scenario: scenario, instance: instance, changed: make(chan struct{}), closed: state.AdmissionClosed, operation: state.OperationID, revision: state.Revision}, nil
}

func (g *providerLifecycle) beginProcess(ctx context.Context) (func(), error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	state, err := g.syncLocked(ctx)
	if err != nil {
		g.closed = true
		return nil, fmt.Errorf("provider admission is closed because lifecycle state is unavailable")
	}
	if state.AdmissionClosed {
		return nil, fmt.Errorf("provider admission is closed for lifecycle maintenance")
	}
	if g.closed {
		return nil, fmt.Errorf("provider admission is closed for lifecycle maintenance")
	}
	g.admitting++
	var once sync.Once
	return func() {
		once.Do(func() { g.mu.Lock(); g.admitting--; close(g.changed); g.changed = make(chan struct{}); g.mu.Unlock() })
	}, nil
}

func fenceToken(revision int64) string { return fmt.Sprintf("%d", revision) }

func (g *providerLifecycle) syncLocked(ctx context.Context) (repository.ProviderLifecycleFence, error) {
	state, err := g.store.GetProviderLifecycleFence(ctx)
	if err != nil {
		return repository.ProviderLifecycleFence{}, err
	}
	if state.Revision < 0 || state.AdmissionClosed != (state.OperationID != "") || (state.AdmissionClosed && state.Revision == 0) {
		return repository.ProviderLifecycleFence{}, fmt.Errorf("invalid durable lifecycle fence state")
	}
	g.closed, g.operation, g.revision = state.AdmissionClosed, state.OperationID, state.Revision
	return state, nil
}

func (g *providerLifecycle) standingLocked(state repository.ProviderLifecycleFence) *commonv1.LifecycleStanding {
	running := 0
	if g.tracker != nil {
		running = g.tracker.GetAllStats().TotalRunning
	}
	drained := state.AdmissionClosed && running == 0 && g.admitting == 0
	phase := commonv1.LifecycleMaintenancePhase_LIFECYCLE_MAINTENANCE_PHASE_OPEN
	if state.AdmissionClosed {
		phase = commonv1.LifecycleMaintenancePhase_LIFECYCLE_MAINTENANCE_PHASE_CLOSING
	}
	if drained {
		phase = commonv1.LifecycleMaintenancePhase_LIFECYCLE_MAINTENANCE_PHASE_DRAINED
	}
	return &commonv1.LifecycleStanding{
		Provider: g.provider, ProviderInstanceId: g.instance, Scenario: g.scenario,
		InstanceId: g.instance, OperationId: state.OperationID, FenceToken: fenceToken(state.Revision), Revision: state.Revision,
		Phase: phase, AdmissionClosed: state.AdmissionClosed, Admitting: int32(g.admitting), Remaining: int32(running),
		InventoryComplete: true, Drained: drained, ObservedAt: timestamppb.Now(), Interlock: "scenario-lock-v1",
	}
}

func (g *providerLifecycle) Prepare(ctx context.Context, req *connect.Request[commonv1.LifecyclePrepareRequest]) (*connect.Response[commonv1.LifecyclePrepareResponse], error) {
	if req.Msg.GetOperationId() == "" || req.Msg.GetReason() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("operation_id and reason are required"))
	}
	if req.Msg.GetEmergencyOverride() && req.Msg.GetOverrideReason() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("override_reason is required for emergency override"))
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	state, err := g.syncLocked(ctx)
	if err != nil {
		g.closed = true
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("read durable lifecycle fence: %w", err))
	}
	if state.AdmissionClosed {
		if state.OperationID != req.Msg.GetOperationId() {
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("another lifecycle operation is fenced"))
		}
		return connect.NewResponse(&commonv1.LifecyclePrepareResponse{Standing: g.standingLocked(state)}), nil
	}
	next := repository.ProviderLifecycleFence{AdmissionClosed: true, OperationID: req.Msg.GetOperationId(), Revision: state.Revision + 1}
	if err := g.store.CompareAndSwapProviderLifecycleFence(ctx, state.Revision, next); err != nil {
		g.closed = true
		return nil, connect.NewError(connect.CodeAborted, fmt.Errorf("persist lifecycle fence: %w", err))
	}
	g.closed, g.operation, g.revision = true, next.OperationID, next.Revision
	close(g.changed)
	g.changed = make(chan struct{})
	return connect.NewResponse(&commonv1.LifecyclePrepareResponse{Standing: g.standingLocked(next)}), nil
}

func (g *providerLifecycle) Status(ctx context.Context, _ *connect.Request[commonv1.LifecycleStatusRequest]) (*connect.Response[commonv1.LifecycleStatusResponse], error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	state, err := g.syncLocked(ctx)
	if err != nil {
		g.closed = true
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("read durable lifecycle fence: %w", err))
	}
	return connect.NewResponse(&commonv1.LifecycleStatusResponse{Standing: g.standingLocked(state)}), nil
}

func (g *providerLifecycle) Drain(ctx context.Context, req *connect.Request[commonv1.LifecycleDrainRequest]) (*connect.Response[commonv1.LifecycleDrainResponse], error) {
	if req.Msg.GetOperationId() == "" || req.Msg.GetRevision() <= 0 || req.Msg.GetFenceToken() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("operation_id, fence_token and revision are required"))
	}
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		g.mu.Lock()
		state, err := g.syncLocked(ctx)
		if err != nil {
			g.mu.Unlock()
			return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("read durable lifecycle fence: %w", err))
		}
		if state.OperationID != req.Msg.GetOperationId() || state.Revision != req.Msg.GetRevision() || fenceToken(state.Revision) != req.Msg.GetFenceToken() || !state.AdmissionClosed {
			g.mu.Unlock()
			return nil, connect.NewError(connect.CodeAborted, fmt.Errorf("lifecycle fence does not match"))
		}
		standing := g.standingLocked(state)
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

func (g *providerLifecycle) Resume(ctx context.Context, req *connect.Request[commonv1.LifecycleResumeRequest]) (*connect.Response[commonv1.LifecycleResumeResponse], error) {
	if req.Msg.GetOperationId() == "" || req.Msg.GetRevision() <= 0 || req.Msg.GetFenceToken() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("operation_id, fence_token and revision are required"))
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	state, err := g.syncLocked(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("read durable lifecycle fence: %w", err))
	}
	if !state.AdmissionClosed || state.OperationID != req.Msg.GetOperationId() || state.Revision != req.Msg.GetRevision() || fenceToken(state.Revision) != req.Msg.GetFenceToken() {
		return nil, connect.NewError(connect.CodeAborted, fmt.Errorf("lifecycle fence does not match"))
	}
	next := repository.ProviderLifecycleFence{Revision: state.Revision + 1}
	if err := g.store.CompareAndSwapProviderLifecycleFence(ctx, state.Revision, next); err != nil {
		return nil, connect.NewError(connect.CodeAborted, fmt.Errorf("persist lifecycle resume: %w", err))
	}
	g.closed, g.operation, g.revision = false, "", next.Revision
	close(g.changed)
	g.changed = make(chan struct{})
	return connect.NewResponse(&commonv1.LifecycleResumeResponse{Standing: g.standingLocked(next)}), nil
}

var _ commonconnect.LifecycleMaintenanceServiceHandler = (*providerLifecycle)(nil)
