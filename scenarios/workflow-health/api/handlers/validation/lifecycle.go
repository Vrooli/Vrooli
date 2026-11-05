package validation

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/database"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	commonconnect "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1/commonv1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
	workflowrun "workflow-health/internal/validationrun"
)

var errValidationAdmissionClosed = errors.New("workflow-health admission is closed for lifecycle maintenance")

const validationLifecycleInterlock = "scenario-lock-v1"

const lifecyclePrepareRollbackTimeout = 5 * time.Second

var _ commonconnect.LifecycleMaintenanceServiceHandler = (*connectHandler)(nil)

func (h *connectHandler) admitValidation(ctx context.Context) (func(), error) {
	h.lifecycleMu.Lock()
	state, err := h.primaryLedger().GetLifecycleFence(ctx)
	if err != nil {
		h.lifecycleMu.Unlock()
		return nil, err
	}
	if state.Closed {
		h.lifecycleMu.Unlock()
		return nil, errValidationAdmissionClosed
	}
	h.lifecycleAdmitting++
	h.lifecycleMu.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() {
			h.lifecycleMu.Lock()
			h.lifecycleAdmitting--
			h.signalLifecycleChangeLocked()
			h.lifecycleMu.Unlock()
		})
	}, nil
}

func (h *connectHandler) Prepare(ctx context.Context, req *connect.Request[commonv1.LifecyclePrepareRequest]) (*connect.Response[commonv1.LifecyclePrepareResponse], error) {
	message := req.Msg
	if message.GetOperationId() == "" || message.GetReason() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("operation_id and reason are required"))
	}
	if message.GetEmergencyOverride() && message.GetOverrideReason() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("override_reason is required for emergency override"))
	}

	h.lifecyclePrepareMu.Lock()
	defer h.lifecyclePrepareMu.Unlock()

	h.lifecycleMu.Lock()
	fence, err := h.primaryLedger().GetLifecycleFence(ctx)
	if err != nil {
		h.lifecycleMu.Unlock()
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("load lifecycle fence: %w", err))
	}
	changed := false
	newlyClosed := false
	switch {
	case !fence.Closed:
		next := workflowrun.LifecycleFence{Closed: true, Operation: message.GetOperationId(), Revision: fence.Revision + 1}
		if err := h.primaryLedger().UpdateLifecycleFence(ctx, fence.Revision, next); err != nil {
			h.lifecycleMu.Unlock()
			return nil, connect.NewError(connect.CodeAborted, err)
		}
		fence, changed, newlyClosed = next, true, true
	case fence.Operation == message.GetOperationId():
		// A retry for this exact operation is idempotent.
	case !message.GetEmergencyOverride():
		h.lifecycleMu.Unlock()
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("another lifecycle operation already holds the closed admission fence"))
	default:
		// An explicit emergency replacement may adopt an already-closed fence,
		// but it never reopens admission or claims that work has drained.
		next := workflowrun.LifecycleFence{Closed: true, Operation: message.GetOperationId(), Revision: fence.Revision}
		if err := h.primaryLedger().UpdateLifecycleFence(ctx, fence.Revision, next); err != nil {
			h.lifecycleMu.Unlock()
			return nil, connect.NewError(connect.CodeAborted, err)
		}
		fence, changed = next, true
	}
	if changed {
		h.signalLifecycleChangeLocked()
	}
	h.lifecycleMu.Unlock()

	standing, _, err := h.lifecycleStanding(ctx, fence.Operation, fence.Revision)
	if err != nil {
		if newlyClosed {
			cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), lifecyclePrepareRollbackTimeout)
			defer cancel()
			if rollbackErr := h.rollbackLifecyclePrepare(cleanupCtx, fence); rollbackErr != nil {
				err = errors.Join(err, fmt.Errorf("rollback failed lifecycle prepare: %w", rollbackErr))
			}
		}
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("observe validation work before lifecycle replacement: %w", err))
	}
	return connect.NewResponse(&commonv1.LifecyclePrepareResponse{Standing: standing}), nil
}

func (h *connectHandler) Status(ctx context.Context, req *connect.Request[commonv1.LifecycleStatusRequest]) (*connect.Response[commonv1.LifecycleStatusResponse], error) {
	standing, _, err := h.lifecycleStanding(ctx, req.Msg.GetOperationId(), req.Msg.GetRevision())
	if err != nil {
		return nil, lifecycleError(err)
	}
	if req.Msg.GetFenceToken() != "" && req.Msg.GetFenceToken() != standing.GetFenceToken() {
		return nil, connect.NewError(connect.CodeAborted, errors.New("lifecycle fence token does not match"))
	}
	return connect.NewResponse(&commonv1.LifecycleStatusResponse{Standing: standing}), nil
}

func (h *connectHandler) Drain(ctx context.Context, req *connect.Request[commonv1.LifecycleDrainRequest]) (*connect.Response[commonv1.LifecycleDrainResponse], error) {
	message := req.Msg
	if err := h.validateLifecycleIdentity(ctx, message.GetOperationId(), message.GetFenceToken(), message.GetRevision()); err != nil {
		return nil, lifecycleError(err)
	}
	if timeout := message.GetTimeout(); timeout != nil && timeout.AsDuration() > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout.AsDuration())
		defer cancel()
	}
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		standing, changed, err := h.lifecycleStanding(ctx, message.GetOperationId(), message.GetRevision())
		if err != nil {
			return nil, lifecycleError(err)
		}
		if message.GetFenceToken() != standing.GetFenceToken() {
			return nil, connect.NewError(connect.CodeAborted, errors.New("lifecycle fence token does not match"))
		}
		if standing.GetDrained() {
			return connect.NewResponse(&commonv1.LifecycleDrainResponse{Standing: standing}), nil
		}
		select {
		case <-ctx.Done():
			return nil, connect.NewError(connect.CodeDeadlineExceeded, fmt.Errorf("validation drain timed out; lifecycle admission remains closed: %w", ctx.Err()))
		case <-changed:
		case <-ticker.C:
		}
	}
}

func (h *connectHandler) Resume(ctx context.Context, req *connect.Request[commonv1.LifecycleResumeRequest]) (*connect.Response[commonv1.LifecycleResumeResponse], error) {
	message := req.Msg
	if message.GetOperationId() == "" || message.GetRevision() <= 0 || message.GetFenceToken() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("operation_id, fence_token, and revision are required"))
	}
	h.lifecycleMu.Lock()
	fence, err := h.primaryLedger().GetLifecycleFence(ctx)
	if err != nil {
		h.lifecycleMu.Unlock()
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("load lifecycle fence: %w", err))
	}
	if !fence.Closed || fence.Operation != message.GetOperationId() || fence.Revision != message.GetRevision() || strconv.FormatInt(fence.Revision, 10) != message.GetFenceToken() {
		h.lifecycleMu.Unlock()
		return nil, connect.NewError(connect.CodeAborted, errors.New("lifecycle fence does not match"))
	}
	next := workflowrun.LifecycleFence{Revision: fence.Revision + 1}
	if err := h.primaryLedger().UpdateLifecycleFence(ctx, fence.Revision, next); err != nil {
		h.lifecycleMu.Unlock()
		return nil, connect.NewError(connect.CodeAborted, err)
	}
	h.signalLifecycleChangeLocked()
	h.lifecycleMu.Unlock()
	standing, _, err := h.lifecycleStanding(ctx, message.GetOperationId(), next.Revision)
	if err != nil {
		return nil, lifecycleError(err)
	}
	standing.Phase = commonv1.LifecycleMaintenancePhase_LIFECYCLE_MAINTENANCE_PHASE_RESUMED
	return connect.NewResponse(&commonv1.LifecycleResumeResponse{Standing: standing}), nil
}

func (h *connectHandler) validateLifecycleIdentity(ctx context.Context, operation, token string, revision int64) error {
	if operation == "" || token == "" || revision <= 0 {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("operation_id, fence_token, and revision are required"))
	}
	h.lifecycleMu.Lock()
	fence, err := h.primaryLedger().GetLifecycleFence(ctx)
	h.lifecycleMu.Unlock()
	if err != nil {
		return connect.NewError(connect.CodeUnavailable, fmt.Errorf("load lifecycle fence: %w", err))
	}
	if !fence.Closed || fence.Operation != operation || fence.Revision != revision || strconv.FormatInt(fence.Revision, 10) != token {
		return connect.NewError(connect.CodeAborted, errors.New("lifecycle fence does not match"))
	}
	return nil
}

func (h *connectHandler) lifecycleStanding(ctx context.Context, requestedOperation string, expectedRevision int64) (*commonv1.LifecycleStanding, <-chan struct{}, error) {
	h.lifecycleMu.Lock()
	fence, err := h.primaryLedger().GetLifecycleFence(ctx)
	if err != nil {
		h.lifecycleMu.Unlock()
		return nil, nil, fmt.Errorf("load lifecycle fence: %w", err)
	}
	if expectedRevision > 0 && fence.Revision != expectedRevision {
		h.lifecycleMu.Unlock()
		return nil, nil, connect.NewError(connect.CodeAborted, errors.New("lifecycle fence revision does not match"))
	}
	if fence.Closed && requestedOperation != "" && fence.Operation != requestedOperation {
		h.lifecycleMu.Unlock()
		return nil, nil, connect.NewError(connect.CodeAborted, errors.New("lifecycle operation does not own the closed fence"))
	}
	admitting := h.lifecycleAdmitting
	changed := h.lifecycleChanged
	h.lifecycleMu.Unlock()

	active, err := h.activeValidationInventory(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("read active validation inventory: %w", err)
	}

	h.lifecycleMu.Lock()
	after, err := h.primaryLedger().GetLifecycleFence(ctx)
	if err != nil {
		h.lifecycleMu.Unlock()
		return nil, nil, fmt.Errorf("recheck lifecycle fence: %w", err)
	}
	admittingAfter := h.lifecycleAdmitting
	h.lifecycleMu.Unlock()
	if after.Revision != fence.Revision || after.Closed != fence.Closed || after.Operation != fence.Operation {
		return nil, nil, connect.NewError(connect.CodeAborted, errors.New("lifecycle fence changed while inventory was observed"))
	}

	operation := after.Operation
	if operation == "" {
		operation = requestedOperation
	}
	if operation == "" {
		operation = "workflow-health-status"
	}
	standing := &commonv1.LifecycleStanding{
		Provider: "workflow-health", ProviderInstanceId: h.providerInstanceID,
		Scenario: "workflow-health", InstanceId: h.providerInstanceID,
		OperationId: operation, FenceToken: strconv.FormatInt(after.Revision, 10), Revision: after.Revision,
		Phase:           commonv1.LifecycleMaintenancePhase_LIFECYCLE_MAINTENANCE_PHASE_OPEN,
		AdmissionClosed: after.Closed, Admitting: int32(admittingAfter), Remaining: int32(len(active)),
		InventoryComplete: true, ObservedAt: timestamppb.Now(), Interlock: validationLifecycleInterlock,
	}
	for _, item := range active {
		standing.Work = append(standing.Work, &commonv1.LifecycleWorkItem{Id: item.ID, Kind: item.Kind, Status: item.State, PhysicalExecutorPresent: item.Executor})
	}
	if after.Closed {
		standing.Phase = commonv1.LifecycleMaintenancePhase_LIFECYCLE_MAINTENANCE_PHASE_CLOSING
		if admitting == 0 && admittingAfter == 0 && len(active) == 0 {
			standing.Phase = commonv1.LifecycleMaintenancePhase_LIFECYCLE_MAINTENANCE_PHASE_DRAINED
			standing.Drained = true
		}
	}
	_ = admitting // The first value is retained to make the two-sided snapshot explicit.
	return standing, changed, nil
}

type activeValidationItem struct {
	ID       string
	Kind     string
	State    string
	Executor bool
}

func (h *connectHandler) activeValidationInventory(ctx context.Context) ([]activeValidationItem, error) {
	primary := h.primaryLedger()
	active, err := primary.ListActive(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]activeValidationItem, 0, len(active))
	for _, item := range active {
		out = append(out, activeValidationItem{ID: "primary/" + item.ID, Kind: "validation-run", State: string(item.State), Executor: h.hasExecutor(item.ID)})
	}
	if routed, ok := h.deps.Ledger.DB.(interface{ HasTestPool() bool }); ok && routed.HasTestPool() {
		testActive, err := h.deps.Ledger.ListActive(database.WithTestMode(ctx))
		if err != nil {
			return nil, err
		}
		for _, item := range testActive {
			out = append(out, activeValidationItem{ID: "test-pool/" + item.ID, Kind: "validation-run", State: string(item.State), Executor: h.hasExecutor(item.ID)})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (h *connectHandler) hasExecutor(id string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.cancels[id] != nil
}

func (h *connectHandler) primaryLedger() workflowrun.Repository {
	if primary, ok := h.deps.Ledger.DB.(interface{ Primary() *sql.DB }); ok {
		return workflowrun.Repository{DB: primary.Primary()}
	}
	return h.deps.Ledger
}

func (h *connectHandler) rollbackLifecyclePrepare(ctx context.Context, prepared workflowrun.LifecycleFence) error {
	h.lifecycleMu.Lock()
	defer h.lifecycleMu.Unlock()
	current, err := h.primaryLedger().GetLifecycleFence(ctx)
	if err != nil {
		return err
	}
	if !current.Closed || current.Operation != prepared.Operation || current.Revision != prepared.Revision {
		return errors.New("lifecycle fence changed; refusing stale prepare rollback")
	}
	open := workflowrun.LifecycleFence{Revision: current.Revision + 1}
	if err := h.primaryLedger().UpdateLifecycleFence(ctx, current.Revision, open); err != nil {
		return err
	}
	h.signalLifecycleChangeLocked()
	return nil
}

func (h *connectHandler) signalLifecycleChange() {
	h.lifecycleMu.Lock()
	h.signalLifecycleChangeLocked()
	h.lifecycleMu.Unlock()
}

func (h *connectHandler) signalLifecycleChangeLocked() {
	close(h.lifecycleChanged)
	h.lifecycleChanged = make(chan struct{})
}

func lifecycleError(err error) error {
	var connectErr *connect.Error
	if errors.As(err, &connectErr) {
		return err
	}
	return connect.NewError(connect.CodeFailedPrecondition, err)
}
