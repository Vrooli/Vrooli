package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/lifecycle"
	common "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"web-console/internal/dbx"
)

const hostReasonPrefix = "coordinated host reboot on boot "
const hostInterlock = "web-console-host-reboot-v1"

const hostMaintenanceSchema = `
CREATE TABLE IF NOT EXISTS host_maintenance_operations (
	operation_id TEXT PRIMARY KEY,
	boot_id TEXT NOT NULL,
	token TEXT NOT NULL,
	phase TEXT NOT NULL CHECK (phase IN ('prepared', 'drained', 'released'))
);
CREATE UNIQUE INDEX IF NOT EXISTS host_maintenance_single_owner
	ON host_maintenance_operations ((1)) WHERE phase != 'released';`

// The fixed per-user host socket is reserved for the managed live runtime.
func hostMaintenanceInstance(goos string, getenv func(string) string) string {
	variant := strings.TrimSpace(getenv("VROOLI_VARIANT"))
	managed := strings.EqualFold(strings.TrimSpace(getenv("VROOLI_LIFECYCLE_MANAGED")), "true")
	if goos != "linux" || !managed || (variant != "" && variant != "live") {
		return ""
	}
	return strings.TrimSpace(getenv("VROOLI_RUNTIME_INSTANCE_ID"))
}

type hostReceipt struct {
	operation, boot, token, phase string
}

type hostSnapshotMode int

const (
	hostInventory hostSnapshotMode = iota + 1
	gateOnly
)

type hostSnapshotKey struct{}

// hostMaintenance owns persistent operation identity. The shared provider
// remains the single HTTP admission counter and websocket quiesce path.
type hostMaintenance struct {
	mu             sync.Mutex
	db             dbx.Handle
	boot, instance string
	shared         *lifecycle.Provider
	active         string
	local          *common.LifecycleStanding
}

func newHostMaintenance(ctx context.Context, db dbx.Handle, boot, instance string, hooks lifecycle.Hooks) (*hostMaintenance, error) {
	if _, err := uuid.Parse(boot); err != nil || strings.TrimSpace(instance) == "" {
		return nil, errors.New("host maintenance requires kernel boot ID and runtime instance ID")
	}
	snapshot := hooks.Snapshot
	hooks.Snapshot = func(ctx context.Context) (lifecycle.Inventory, error) {
		mode, _ := ctx.Value(hostSnapshotKey{}).(hostSnapshotMode)
		if mode == gateOnly || snapshot == nil {
			return lifecycle.Inventory{Complete: true}, nil
		}
		report, err := snapshot(ctx)
		if mode == hostInventory {
			report = hostWorkInventory(report)
		}
		return report, err
	}
	shared, err := lifecycle.NewProvider("web-console", instance, hooks)
	if err != nil {
		return nil, err
	}
	h := &hostMaintenance{db: db, boot: boot, instance: instance, shared: shared}
	var row hostReceipt
	err = db.QueryRowContext(ctx, `SELECT operation_id,boot_id,token,phase FROM host_maintenance_operations WHERE phase != 'released'`).
		Scan(&row.operation, &row.boot, &row.token, &row.phase)
	switch {
	case errors.Is(err, sql.ErrNoRows):
	case err != nil:
		return nil, err
	default:
		if err := h.closeGate(ctx, row.operation); err != nil {
			return nil, err
		}
		if row.phase == "drained" {
			if err := h.quiesce(ctx); err != nil {
				return nil, err
			}
		}
	}
	return h, nil
}

func hostWorkInventory(report lifecycle.Inventory) lifecycle.Inventory {
	work := make([]*common.LifecycleWorkItem, 0, len(report.Work))
	for _, item := range report.Work {
		switch item.Status {
		case "idle", "native_goal_continuation", "recovery_only":
			// Durable thread state allows recovery without replaying a turn.
		case "ordinary_turn_running":
			work = append(work, item)
		default:
			work = append(work, item)
			report.Blockers = append(report.Blockers, &common.LifecycleBlocker{
				Code: item.Status, Message: item.Id, Observed: true,
				Disposition: common.LifecycleBlockerDisposition_LIFECYCLE_BLOCKER_DISPOSITION_BLOCKING,
			})
		}
	}
	report.Work = work
	return report
}

func (h *hostMaintenance) Mount(app http.Handler) http.Handler {
	mounted := h.shared.Mount(app)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/common.v1.LifecycleMaintenanceService/") {
			if !h.mu.TryLock() {
				http.Error(w, "lifecycle transition in progress", http.StatusServiceUnavailable)
				return
			}
			defer h.mu.Unlock()
			if h.active != "" {
				http.Error(w, "host lifecycle owns admission", http.StatusServiceUnavailable)
				return
			}
		}
		mounted.ServeHTTP(w, r)
	})
}

func (h *hostMaintenance) closeGate(ctx context.Context, operation string) error {
	// A different local operation can never be replaced by host prepare.
	old, err := h.shared.Status(context.WithValue(ctx, hostSnapshotKey{}, gateOnly), connect.NewRequest(&common.LifecycleStatusRequest{}))
	if err != nil {
		return err
	}
	if old.Msg.GetStanding().GetAdmissionClosed() {
		return connect.NewError(connect.CodeFailedPrecondition, errors.New("ordinary lifecycle owns admission"))
	}
	reply, err := h.shared.Prepare(context.WithValue(ctx, hostSnapshotKey{}, gateOnly),
		connect.NewRequest(&common.LifecyclePrepareRequest{OperationId: operation, Reason: "host maintenance"}))
	if err != nil {
		return err
	}
	h.active, h.local = operation, reply.Msg.GetStanding()
	return nil
}

func (h *hostMaintenance) quiesce(ctx context.Context) error {
	// Request quiescence once and use Status to wait for actual owner work.
	reply, err := h.shared.Drain(context.WithValue(ctx, hostSnapshotKey{}, gateOnly),
		connect.NewRequest(&common.LifecycleDrainRequest{OperationId: h.active,
			FenceToken: h.local.GetFenceToken(), Revision: h.local.GetRevision(),
			QuiesceComponents: true, Timeout: durationpb.New(time.Millisecond)}))
	if err == nil && reply.Msg.GetStanding().GetQuiescencePending() {
		return errors.New("shared admission was not quiesced")
	}
	if connect.CodeOf(err) == connect.CodeDeadlineExceeded {
		// Existing admitted requests may still be draining.
		return nil
	}
	return err
}

func (h *hostMaintenance) openGate(ctx context.Context) error {
	_, err := h.shared.Resume(context.WithValue(ctx, hostSnapshotKey{}, gateOnly),
		connect.NewRequest(&common.LifecycleResumeRequest{OperationId: h.active,
			FenceToken: h.local.GetFenceToken(), Revision: h.local.GetRevision()}))
	if err == nil {
		h.active, h.local = "", nil
	}
	return err
}

func (h *hostMaintenance) receipt(ctx context.Context, operation string) (hostReceipt, error) {
	if strings.TrimSpace(operation) == "" {
		return hostReceipt{}, connect.NewError(connect.CodeInvalidArgument, errors.New("operation ID required"))
	}
	row := hostReceipt{operation: operation}
	err := h.db.QueryRowContext(ctx, `SELECT boot_id,token,phase FROM host_maintenance_operations WHERE operation_id=?`, operation).
		Scan(&row.boot, &row.token, &row.phase)
	if errors.Is(err, sql.ErrNoRows) {
		return row, connect.NewError(connect.CodeNotFound, errors.New("no host operation receipt"))
	}
	return row, err
}

func (h *hostMaintenance) identity(row hostReceipt) *common.LifecycleStanding {
	out := &common.LifecycleStanding{Provider: "web-console", Scenario: "web-console",
		ProviderInstanceId: h.instance, InstanceId: h.instance, OperationId: row.operation,
		FenceToken: row.token, Revision: 1, Interlock: hostInterlock, ObservedAt: timestamppb.Now(),
		FencePersistence: common.LifecycleFencePersistence_LIFECYCLE_FENCE_PERSISTENCE_OPERATION_DURABLE}
	if row.phase == "released" && h.active != row.operation {
		out.Revision = 2
		out.Phase = common.LifecycleMaintenancePhase_LIFECYCLE_MAINTENANCE_PHASE_OPEN
		return out
	}
	out.AdmissionClosed = true
	out.Phase = common.LifecycleMaintenancePhase_LIFECYCLE_MAINTENANCE_PHASE_CLOSING
	return out
}

func (h *hostMaintenance) standing(ctx context.Context, row hostReceipt) (*common.LifecycleStanding, error) {
	out := h.identity(row)
	if !out.AdmissionClosed || row.boot != h.boot || row.phase == "released" {
		return out, nil
	}
	reply, err := h.shared.Status(context.WithValue(ctx, hostSnapshotKey{}, hostInventory),
		connect.NewRequest(&common.LifecycleStatusRequest{OperationId: row.operation,
			FenceToken: h.local.GetFenceToken(), Revision: h.local.GetRevision()}))
	if err != nil {
		return nil, err
	}
	standing := reply.Msg.GetStanding()
	out.Admitting, out.Remaining = standing.Admitting, standing.Remaining
	out.InventoryComplete, out.QuiescencePending = standing.InventoryComplete, standing.QuiescencePending
	out.Work, out.Blockers = standing.Work, standing.Blockers
	out.ComponentStopReady = standing.Drained && len(out.Work) == 0
	if row.phase == "drained" && out.ComponentStopReady {
		out.Drained = true
		out.Phase = common.LifecycleMaintenancePhase_LIFECYCLE_MAINTENANCE_PHASE_DRAINED
	}
	return out, nil
}

func (h *hostMaintenance) Prepare(ctx context.Context, req *connect.Request[common.LifecyclePrepareRequest]) (*connect.Response[common.LifecyclePrepareResponse], error) {
	msg := req.Msg
	if msg.GetReason() != hostReasonPrefix+h.boot || msg.GetEmergencyOverride() {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("prepare requires current kernel boot identity"))
	}
	if !h.mu.TryLock() {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("lifecycle transition in progress"))
	}
	defer h.mu.Unlock()
	row, err := h.receipt(ctx, msg.GetOperationId())
	if connect.CodeOf(err) == connect.CodeNotFound {
		if h.active != "" {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("another host operation owns admission"))
		}
		if err := h.closeGate(ctx, row.operation); err != nil {
			return nil, err
		}
		row.boot, row.token, row.phase = h.boot, uuid.NewString(), "prepared"
		// Once admission has closed, finish recording ownership even if the
		// caller disconnects. The broker can retrieve the exact receipt later.
		save, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
		defer cancel()
		_, err = h.db.ExecContext(save, `INSERT INTO host_maintenance_operations VALUES (?,?,?,?)`, row.operation, row.boot, row.token, row.phase)
		if err != nil {
			return nil, errors.Join(err, h.openGate(save))
		}
	} else if err != nil {
		return nil, err
	}
	if row.boot != h.boot {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("operation belongs to a prior boot"))
	}
	return connect.NewResponse(&common.LifecyclePrepareResponse{Standing: h.identity(row)}), nil
}

func (h *hostMaintenance) Status(ctx context.Context, req *connect.Request[common.LifecycleStatusRequest]) (*connect.Response[common.LifecycleStatusResponse], error) {
	if !h.mu.TryLock() {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("lifecycle transition in progress"))
	}
	defer h.mu.Unlock()
	row, err := h.receipt(ctx, req.Msg.GetOperationId())
	if err != nil {
		return nil, err
	}
	id := h.identity(row)
	if (req.Msg.GetFenceToken() != "" && req.Msg.GetFenceToken() != id.FenceToken) ||
		(req.Msg.GetRevision() != 0 && req.Msg.GetRevision() != id.Revision) {
		return nil, connect.NewError(connect.CodeAborted, errors.New("host receipt identity mismatch"))
	}
	out, err := h.standing(ctx, row)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&common.LifecycleStatusResponse{Standing: out}), nil
}

func (h *hostMaintenance) Drain(ctx context.Context, req *connect.Request[common.LifecycleDrainRequest]) (*connect.Response[common.LifecycleDrainResponse], error) {
	msg := req.Msg
	timeout := lifecycle.DefaultDrainTimeout
	if d := msg.GetTimeout(); d != nil && d.IsValid() && d.AsDuration() > 0 {
		timeout = min(d.AsDuration(), 5*time.Minute)
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	timer := time.NewTicker(100 * time.Millisecond)
	defer timer.Stop()
	for {
		out, err := h.checkpoint(ctx, msg)
		if err != nil {
			return nil, err
		}
		if out.GetDrained() {
			return connect.NewResponse(&common.LifecycleDrainResponse{Standing: out}), nil
		}
		select {
		case <-ctx.Done():
			return nil, connect.NewError(connect.CodeDeadlineExceeded, ctx.Err())
		case <-timer.C:
		}
	}
}

func (h *hostMaintenance) checkpoint(ctx context.Context, msg *common.LifecycleDrainRequest) (*common.LifecycleStanding, error) {
	if !h.mu.TryLock() {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("lifecycle transition in progress"))
	}
	defer h.mu.Unlock()
	row, err := h.receipt(ctx, msg.GetOperationId())
	if err != nil {
		return nil, err
	}
	if row.boot != h.boot || row.phase == "released" || h.active != row.operation ||
		row.token != msg.GetFenceToken() || msg.GetRevision() != 1 || len(msg.GetComponentExits()) != 0 {
		return nil, connect.NewError(connect.CodeAborted, errors.New("host checkpoint identity mismatch"))
	}
	if err := h.quiesce(ctx); err != nil {
		return nil, err
	}
	out, err := h.standing(ctx, row)
	if err != nil {
		return nil, err
	}
	if len(out.Blockers) != 0 {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("host checkpoint blocked: %s", out.Blockers[0].GetCode()))
	}
	if out.ComponentStopReady && !out.Drained {
		if _, err := h.db.ExecContext(ctx, `UPDATE host_maintenance_operations SET phase='drained' WHERE operation_id=? AND phase='prepared'`, row.operation); err != nil {
			return nil, err
		}
		out.Drained = true
		out.Phase = common.LifecycleMaintenancePhase_LIFECYCLE_MAINTENANCE_PHASE_DRAINED
	}
	return out, nil
}

func (h *hostMaintenance) Resume(ctx context.Context, req *connect.Request[common.LifecycleResumeRequest]) (*connect.Response[common.LifecycleResumeResponse], error) {
	if !h.mu.TryLock() {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("lifecycle transition in progress"))
	}
	defer h.mu.Unlock()
	msg := req.Msg
	row, err := h.receipt(ctx, msg.GetOperationId())
	if err != nil {
		return nil, err
	}
	if msg.GetFenceToken() != row.token || (msg.GetRevision() != 1 && !(row.phase == "released" && msg.GetRevision() == 2)) {
		return nil, connect.NewError(connect.CodeAborted, errors.New("host release identity mismatch"))
	}
	if row.phase != "released" {
		if h.active != row.operation {
			return nil, connect.NewError(connect.CodeAborted, errors.New("host operation does not own admission"))
		}
		if _, err := h.db.ExecContext(ctx, `UPDATE host_maintenance_operations SET phase='released' WHERE operation_id=?`, row.operation); err != nil {
			return nil, err
		}
		row.phase = "released"
	}
	if h.active == row.operation {
		if err := h.openGate(ctx); err != nil {
			return nil, err
		}
	}
	return connect.NewResponse(&common.LifecycleResumeResponse{Standing: h.identity(row)}), nil
}
