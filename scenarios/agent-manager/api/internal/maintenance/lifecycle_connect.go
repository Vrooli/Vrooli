package maintenance

import (
	"context"
	"fmt"
	"sync"
	"time"

	"connectrpc.com/connect"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	commonconnect "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1/commonv1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// LifecycleService is the scenario-owned maintenance-fence implementation.
// It reuses Agent Manager's admission gate and complete inventory observer;
// the control plane does not inspect Agent Manager's private run model.
type LifecycleService struct {
	commonconnect.UnimplementedLifecycleMaintenanceServiceHandler
	gate                         *Gate
	observe                      func(context.Context) (Inventory, error)
	provider, scenario, instance string
	mu                           sync.Mutex
	operation                    string
	revision                     int64
}

func NewLifecycleService(gate *Gate, observe func(context.Context) (Inventory, error), provider, scenario, instance string) (*LifecycleService, error) {
	if gate == nil || observe == nil {
		return nil, fmt.Errorf("lifecycle: gate and inventory observer are required")
	}
	if provider == "" || scenario == "" || instance == "" {
		return nil, fmt.Errorf("lifecycle: provider, scenario, and instance identity are required")
	}
	return &LifecycleService{gate: gate, observe: observe, provider: provider, scenario: scenario, instance: instance}, nil
}

func (s *LifecycleService) Prepare(ctx context.Context, req *connect.Request[commonv1.LifecyclePrepareRequest]) (*connect.Response[commonv1.LifecyclePrepareResponse], error) {
	if req.Msg.GetOperationId() == "" || req.Msg.GetReason() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("operation_id and reason are required"))
	}
	if req.Msg.GetEmergencyOverride() && req.Msg.GetOverrideReason() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("override_reason is required for emergency override"))
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.operation != "" && s.operation != req.Msg.GetOperationId() {
		if !req.Msg.GetEmergencyOverride() {
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("another lifecycle operation is fenced"))
		}
		// A prior lifecycle initiator may have disappeared after closing
		// admission but before its provider fence was resumed. An explicitly
		// audited emergency replacement may adopt that already-closed gate;
		// it must not reopen admission or pretend the inventory is complete.
		state, err := s.gate.Status(ctx)
		if err != nil {
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		}
		if !state.Closed {
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("stale lifecycle operation has no closed admission"))
		}
		standing, err := s.standingFrom(ctx, state, req.Msg.GetOperationId(), state.Revision)
		if err != nil {
			return nil, err
		}
		s.operation, s.revision = req.Msg.GetOperationId(), state.Revision
		return connect.NewResponse(&commonv1.LifecyclePrepareResponse{Standing: standing}), nil
	}
	state, err := s.gate.Enter(ctx, "control-plane", req.Msg.GetReason())
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	standing, err := s.standing(ctx, req.Msg.GetOperationId(), state.Revision)
	if err != nil {
		return nil, err
	}
	s.operation, s.revision = req.Msg.GetOperationId(), state.Revision
	return connect.NewResponse(&commonv1.LifecyclePrepareResponse{Standing: standing}), nil
}

func (s *LifecycleService) Status(ctx context.Context, req *connect.Request[commonv1.LifecycleStatusRequest]) (*connect.Response[commonv1.LifecycleStatusResponse], error) {
	standing, err := s.standing(ctx, req.Msg.GetOperationId(), req.Msg.GetRevision())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&commonv1.LifecycleStatusResponse{Standing: standing}), nil
}

func (s *LifecycleService) Drain(ctx context.Context, req *connect.Request[commonv1.LifecycleDrainRequest]) (*connect.Response[commonv1.LifecycleDrainResponse], error) {
	if req.Msg.GetRevision() <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("revision is required"))
	}
	timeout := 120 * time.Second
	if req.Msg.GetTimeout() != nil && req.Msg.GetTimeout().AsDuration() > 0 {
		timeout = req.Msg.GetTimeout().AsDuration()
	}
	drainCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	standing, err := s.gate.Wait(drainCtx, func(ctx context.Context) (int, error) { return s.gateRemaining(ctx) }, time.Second)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	converted, err := s.standingFrom(ctx, standing, req.Msg.GetOperationId(), req.Msg.GetRevision())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&commonv1.LifecycleDrainResponse{Standing: converted}), nil
}

func (s *LifecycleService) Resume(ctx context.Context, req *connect.Request[commonv1.LifecycleResumeRequest]) (*connect.Response[commonv1.LifecycleResumeResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.operation != req.Msg.GetOperationId() || s.revision != req.Msg.GetRevision() {
		return nil, connect.NewError(connect.CodeAborted, fmt.Errorf("lifecycle fence does not match"))
	}
	state, err := s.gate.Resume(ctx, "control-plane", req.Msg.GetRevision())
	if err != nil {
		return nil, connect.NewError(connect.CodeAborted, err)
	}
	s.operation, s.revision = "", 0
	standing, err := s.standing(ctx, req.Msg.GetOperationId(), state.Revision)
	if err != nil {
		// Resume already reopened admission. Return an explicit open/unknown
		// standing instead of reporting failure after the durable transition;
		// otherwise callers can retry a successful resume and observe a stale
		// fence while the provider is already open.
		standing = &commonv1.LifecycleStanding{
			Provider:           s.provider,
			ProviderInstanceId: s.instance,
			Scenario:           s.scenario,
			InstanceId:         s.instance,
			OperationId:        req.Msg.GetOperationId(),
			FenceToken:         fmt.Sprintf("%d", state.Revision),
			Revision:           state.Revision,
			Phase:              commonv1.LifecycleMaintenancePhase_LIFECYCLE_MAINTENANCE_PHASE_OPEN,
			AdmissionClosed:    false,
			Admitting:          0,
			Remaining:          -1,
			InventoryComplete:  false,
			Drained:            false,
			ObservedAt:         timestamppb.Now(),
			Interlock:          ScenarioLockV1,
			Blockers: []*commonv1.LifecycleBlocker{{
				Code:        "INVENTORY_INCOMPLETE",
				Message:     "durable or physical executor inventory is incomplete: " + err.Error(),
				Disposition: commonv1.LifecycleBlockerDisposition_LIFECYCLE_BLOCKER_DISPOSITION_UNKNOWN,
			}},
		}
	}
	return connect.NewResponse(&commonv1.LifecycleResumeResponse{Standing: standing}), nil
}

func (s *LifecycleService) gateRemaining(ctx context.Context) (int, error) {
	inv, err := s.observe(ctx)
	if err != nil || inv.Remaining == nil {
		if err == nil {
			err = fmt.Errorf("inventory is incomplete")
		}
		return -1, err
	}
	return *inv.Remaining, nil
}

func (s *LifecycleService) standing(ctx context.Context, operation string, revision int64) (*commonv1.LifecycleStanding, error) {
	v, err := s.gate.Status(ctx)
	if err != nil {
		return nil, err
	}
	return s.standingFrom(ctx, v, operation, revision)
}

func (s *LifecycleService) standingFrom(ctx context.Context, v Standing, operation string, revision int64) (*commonv1.LifecycleStanding, error) {
	inv, err := s.observe(ctx)
	if err != nil {
		return nil, err
	}
	complete := inv.Remaining != nil && len(inv.Unknown) == 0
	remaining := int32(-1)
	if inv.Remaining != nil {
		remaining = int32(*inv.Remaining)
	}
	phase := commonv1.LifecycleMaintenancePhase_LIFECYCLE_MAINTENANCE_PHASE_OPEN
	if v.Closed {
		phase = commonv1.LifecycleMaintenancePhase_LIFECYCLE_MAINTENANCE_PHASE_CLOSING
	}
	if v.Closed && complete && remaining == 0 && len(inv.Work) == 0 && len(inv.Executors) == 0 {
		phase = commonv1.LifecycleMaintenancePhase_LIFECYCLE_MAINTENANCE_PHASE_DRAINED
	}
	standing := &commonv1.LifecycleStanding{Provider: s.provider, ProviderInstanceId: s.instance, Scenario: s.scenario, InstanceId: s.instance, OperationId: operation, FenceToken: fmt.Sprintf("%d", revision), Revision: revision, Phase: phase, AdmissionClosed: v.Closed, Admitting: int32(v.Admitting), Remaining: remaining, InventoryComplete: complete, Drained: phase == commonv1.LifecycleMaintenancePhase_LIFECYCLE_MAINTENANCE_PHASE_DRAINED, ObservedAt: timestamppb.Now(), Interlock: ScenarioLockV1}
	for _, w := range inv.Work {
		standing.Work = append(standing.Work, &commonv1.LifecycleWorkItem{Id: w.ID, Kind: w.Kind, Status: w.Status})
	}
	if !complete {
		standing.Blockers = append(standing.Blockers, &commonv1.LifecycleBlocker{Code: "INVENTORY_INCOMPLETE", Message: "durable or physical executor inventory is incomplete", Disposition: commonv1.LifecycleBlockerDisposition_LIFECYCLE_BLOCKER_DISPOSITION_UNKNOWN})
	}
	return standing, nil
}

var _ commonconnect.LifecycleMaintenanceServiceHandler = (*LifecycleService)(nil)
