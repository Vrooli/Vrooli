package lifecycle

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"connectrpc.com/connect"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	commonconnect "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1/commonv1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const servicePath = "/common.v1.LifecycleMaintenanceService/"

// Schema is the durable owner table for Device Control's provider fence.
func Schema() string {
	return `CREATE TABLE IF NOT EXISTS device_control_lifecycle_fence (
	 singleton INTEGER PRIMARY KEY CHECK (singleton = 1), provider_instance_id TEXT NOT NULL,
	 closed INTEGER NOT NULL, operation_id TEXT NOT NULL, fence_token TEXT NOT NULL,
	 revision INTEGER NOT NULL, updated_at TEXT NOT NULL
);`
}

type Provider struct {
	commonconnect.UnimplementedLifecycleMaintenanceServiceHandler
	db                 *sql.DB
	mu                 sync.Mutex
	instance           string
	closed             bool
	operation, token   string
	revision           int64
	active             int
	changed            chan struct{}
	owners             func(context.Context) ([]*commonv1.LifecycleWorkItem, []*commonv1.LifecycleBlocker, bool)
	components         func(context.Context, map[string]string) ([]*commonv1.LifecycleManagedComponent, []*commonv1.LifecycleWorkItem, []*commonv1.LifecycleBlocker, bool)
	quiesce            func(context.Context) error
	componentQuiesced  bool
	componentStopBegun bool
	componentExits     map[string]string
}

// SetComponentMaintenance installs the owner-routed two-stage drain. Enumerated
// components remain blocking until the local control plane waits their pinned
// incarnation; enumeration completeness alone never means replacement readiness.
func (p *Provider) SetComponentMaintenance(snapshot func(context.Context, map[string]string) ([]*commonv1.LifecycleManagedComponent, []*commonv1.LifecycleWorkItem, []*commonv1.LifecycleBlocker, bool), quiesce func(context.Context) error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.components, p.quiesce = snapshot, quiesce
}

// SetOwnerInventory connects lifecycle standing to the owning control service.
// A nil or failed inventory remains explicitly incomplete.
func (p *Provider) SetOwnerInventory(snapshot func(context.Context) ([]*commonv1.LifecycleWorkItem, []*commonv1.LifecycleBlocker, bool)) {
	p.mu.Lock()
	p.owners = snapshot
	p.mu.Unlock()
}

func New(ctx context.Context, db *sql.DB) (*Provider, error) {
	if db == nil {
		return nil, errors.New("lifecycle database is required")
	}
	if _, err := db.ExecContext(ctx, Schema()); err != nil {
		return nil, fmt.Errorf("ensure lifecycle fence schema: %w", err)
	}
	p := &Provider{db: db, changed: make(chan struct{})}
	var instance string
	err := db.QueryRowContext(ctx, `SELECT provider_instance_id FROM device_control_lifecycle_fence WHERE singleton=1`).Scan(&instance)
	if errors.Is(err, sql.ErrNoRows) {
		b := make([]byte, 16)
		if _, err = rand.Read(b); err != nil {
			return nil, err
		}
		instance = hex.EncodeToString(b)
		_, err = db.ExecContext(ctx, `INSERT INTO device_control_lifecycle_fence(singleton,provider_instance_id,closed,operation_id,fence_token,revision,updated_at) VALUES(1,?,0,'','',0,?)`, instance, time.Now().UTC().Format(time.RFC3339Nano))
	}
	if err != nil {
		return nil, fmt.Errorf("load lifecycle provider identity: %w", err)
	}
	p.instance = instance
	if err := p.refreshLocked(ctx); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *Provider) refreshLocked(ctx context.Context) error {
	var closed int
	err := p.db.QueryRowContext(ctx, `SELECT closed,operation_id,fence_token,revision FROM device_control_lifecycle_fence WHERE singleton=1`).Scan(&closed, &p.operation, &p.token, &p.revision)
	if err != nil {
		p.closed = true
		return err
	}
	p.closed = closed != 0
	if p.revision < 0 || p.closed != (p.operation != "") || p.closed != (p.token != "") {
		p.closed = true
		return errors.New("durable lifecycle fence is inconsistent")
	}
	return nil
}

func (p *Provider) admit(ctx context.Context) (func(), error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.refreshLocked(ctx); err != nil {
		return nil, errors.New("admission closed: durable lifecycle state unavailable")
	}
	if p.closed {
		return nil, errors.New("admission closed for lifecycle maintenance")
	}
	p.active++
	var once sync.Once
	return func() {
		once.Do(func() { p.mu.Lock(); p.active--; close(p.changed); p.changed = make(chan struct{}); p.mu.Unlock() })
	}, nil
}

func (p *Provider) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(r.URL.Path) >= len(servicePath) && r.URL.Path[:len(servicePath)] == servicePath {
			next.ServeHTTP(w, r)
			return
		}
		leave, err := p.admit(r.Context())
		if err != nil {
			http.Error(w, "service unavailable: lifecycle admission closed", http.StatusServiceUnavailable)
			return
		}
		defer leave()
		next.ServeHTTP(w, r)
	})
}

func (p *Provider) Prepare(ctx context.Context, req *connect.Request[commonv1.LifecyclePrepareRequest]) (*connect.Response[commonv1.LifecyclePrepareResponse], error) {
	m := req.Msg
	if m.GetOperationId() == "" || m.GetReason() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("operation_id and reason are required"))
	}
	if m.GetEmergencyOverride() && m.GetOverrideReason() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("override_reason is required for emergency override"))
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.refreshLocked(ctx); err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	if p.closed && p.operation != m.GetOperationId() && !m.GetEmergencyOverride() {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("another lifecycle operation holds the fence"))
	}
	if !p.closed || p.operation != m.GetOperationId() {
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		token := hex.EncodeToString(b)
		rev := p.revision + 1
		res, err := p.db.ExecContext(ctx, `UPDATE device_control_lifecycle_fence SET closed=1,operation_id=?,fence_token=?,revision=?,updated_at=? WHERE singleton=1 AND revision=?`, m.GetOperationId(), token, rev, time.Now().UTC().Format(time.RFC3339Nano), p.revision)
		if err != nil {
			return nil, connect.NewError(connect.CodeUnavailable, err)
		}
		n, _ := res.RowsAffected()
		if n != 1 {
			return nil, connect.NewError(connect.CodeAborted, errors.New("lifecycle fence changed"))
		}
		p.closed, p.operation, p.token, p.revision = true, m.GetOperationId(), token, rev
		// An override adopts a new fence, but cannot reconstruct an owner that
		// this API instance already stopped. Keep stop-begun sticky until exit.
		p.componentQuiesced, p.componentExits = false, nil
		close(p.changed)
		p.changed = make(chan struct{})
	}
	return connect.NewResponse(&commonv1.LifecyclePrepareResponse{Standing: p.standing()}), nil
}

func (p *Provider) Status(ctx context.Context, req *connect.Request[commonv1.LifecycleStatusRequest]) (*connect.Response[commonv1.LifecycleStatusResponse], error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.refreshLocked(ctx); err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	if req.Msg.GetRevision() > 0 && req.Msg.GetRevision() != p.revision {
		return nil, connect.NewError(connect.CodeAborted, errors.New("lifecycle revision mismatch"))
	}
	if req.Msg.GetOperationId() != "" && req.Msg.GetOperationId() != p.operation {
		return nil, connect.NewError(connect.CodeAborted, errors.New("lifecycle operation mismatch"))
	}
	if req.Msg.GetFenceToken() != "" && req.Msg.GetFenceToken() != p.token {
		return nil, connect.NewError(connect.CodeAborted, errors.New("lifecycle token mismatch"))
	}
	return connect.NewResponse(&commonv1.LifecycleStatusResponse{Standing: p.standing()}), nil
}

func (p *Provider) Drain(ctx context.Context, req *connect.Request[commonv1.LifecycleDrainRequest]) (*connect.Response[commonv1.LifecycleDrainResponse], error) {
	if err := p.validate(ctx, req.Msg.GetOperationId(), req.Msg.GetFenceToken(), req.Msg.GetRevision()); err != nil {
		return nil, err
	}
	if d := req.Msg.GetTimeout(); d != nil && d.AsDuration() > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, d.AsDuration())
		defer cancel()
	}
	if req.Msg.GetQuiesceComponents() {
		if len(req.Msg.GetComponentExits()) != 0 {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("quiescence cannot carry process-exit completion"))
		}
		p.mu.Lock()
		quiesce := p.quiesce
		if quiesce != nil {
			// Quiescing can release owner leases and stop owner admission. It is
			// irreversible for this API instance; only replacement may reopen.
			p.componentStopBegun = true
		}
		p.mu.Unlock()
		if quiesce == nil {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("component quiescence is unavailable"))
		}
		if err := quiesce(ctx); err != nil {
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		}
	}
	if len(req.Msg.GetComponentExits()) > 0 {
		p.mu.Lock()
		if !p.componentQuiesced || p.components == nil {
			p.mu.Unlock()
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("component work has not quiesced"))
		}
		components, _, _, complete := p.components(ctx, nil)
		expected := map[string]string{}
		for _, c := range components {
			expected[c.GetName()] = c.GetGeneration()
		}
		proof := map[string]string{}
		for _, c := range req.Msg.GetComponentExits() {
			if !complete || c.GetName() == "" || c.GetGeneration() == "" || expected[c.GetName()] != c.GetGeneration() || proof[c.GetName()] != "" {
				p.mu.Unlock()
				return nil, connect.NewError(connect.CodeAborted, errors.New("component generation mismatch"))
			}
			proof[c.GetName()] = c.GetGeneration()
		}
		// The final request is still validated below against the durable fence.
		if err := p.refreshLocked(ctx); err != nil || !p.closed || p.operation != req.Msg.GetOperationId() || p.token != req.Msg.GetFenceToken() || p.revision != req.Msg.GetRevision() {
			p.mu.Unlock()
			return nil, connect.NewError(connect.CodeAborted, errors.New("lifecycle fence changed before component completion"))
		}
		p.componentExits = proof
		p.mu.Unlock()
	}
	for {
		p.mu.Lock()
		if err := p.refreshLocked(ctx); err != nil {
			p.mu.Unlock()
			return nil, connect.NewError(connect.CodeUnavailable, err)
		}
		if !p.closed || p.operation != req.Msg.GetOperationId() || p.token != req.Msg.GetFenceToken() || p.revision != req.Msg.GetRevision() {
			p.mu.Unlock()
			return nil, connect.NewError(connect.CodeAborted, errors.New("lifecycle fence changed while draining"))
		}
		if req.Msg.GetQuiesceComponents() && p.owners != nil {
			_, blockers, complete := p.owners(ctx)
			if p.active == 0 && complete && len(blockers) == 0 {
				p.componentQuiesced = true
			}
		}
		s := p.standing()
		ch := p.changed
		p.mu.Unlock()
		if s.GetDrained() || (req.Msg.GetQuiesceComponents() && s.GetComponentStopReady()) {
			return connect.NewResponse(&commonv1.LifecycleDrainResponse{Standing: s}), nil
		}
		select {
		case <-ctx.Done():
			return nil, connect.NewError(connect.CodeDeadlineExceeded, ctx.Err())
		case <-ch:
		case <-time.After(250 * time.Millisecond):
			// Owner state can change without another HTTP admission transition
			// (for example, a recording stop receipt or lease expiry). Re-snapshot
			// boundedly so Drain observes that completion.
		}
	}
}

func (p *Provider) Resume(ctx context.Context, req *connect.Request[commonv1.LifecycleResumeRequest]) (*connect.Response[commonv1.LifecycleResumeResponse], error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.refreshLocked(ctx); err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	if !p.closed || req.Msg.GetOperationId() != p.operation || req.Msg.GetFenceToken() != p.token || req.Msg.GetRevision() != p.revision {
		return nil, connect.NewError(connect.CodeAborted, errors.New("lifecycle fence identity mismatch"))
	}
	if p.componentStopBegun {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("component quiescence requires replacement before admission may reopen"))
	}
	next := p.revision + 1
	res, err := p.db.ExecContext(ctx, `UPDATE device_control_lifecycle_fence SET closed=0,operation_id='',fence_token='',revision=?,updated_at=? WHERE singleton=1 AND revision=? AND fence_token=?`, next, time.Now().UTC().Format(time.RFC3339Nano), p.revision, p.token)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return nil, connect.NewError(connect.CodeAborted, errors.New("lifecycle fence changed"))
	}
	p.closed = false
	p.componentQuiesced, p.componentStopBegun, p.componentExits = false, false, nil
	p.operation = ""
	p.token = ""
	p.revision = next
	close(p.changed)
	p.changed = make(chan struct{})
	s := p.standing()
	s.Phase = commonv1.LifecycleMaintenancePhase_LIFECYCLE_MAINTENANCE_PHASE_RESUMED
	return connect.NewResponse(&commonv1.LifecycleResumeResponse{Standing: s}), nil
}

func (p *Provider) validate(ctx context.Context, op, token string, revision int64) error {
	if op == "" || token == "" || revision <= 0 {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("operation_id, fence_token and revision are required"))
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.refreshLocked(ctx); err != nil {
		return connect.NewError(connect.CodeUnavailable, err)
	}
	if !p.closed || p.operation != op || p.token != token || p.revision != revision {
		return connect.NewError(connect.CodeAborted, errors.New("lifecycle fence identity mismatch"))
	}
	return nil
}

func (p *Provider) standing() *commonv1.LifecycleStanding {
	// The control and desktop WebRTC owners report their live session state.
	// The desktop owner/helper pair remains unknown until those processes expose
	// lifecycle completion receipts.
	work := []*commonv1.LifecycleWorkItem{}
	blockers := []*commonv1.LifecycleBlocker{}
	complete := false
	// Owner hooks are installed once during composition, before the listener
	// starts, so standing can read this immutable callback while holding p.mu.
	owners := p.owners
	if owners != nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		ownerWork, ownerBlockers, ownerComplete := owners(ctx)
		cancel()
		work = append(work, ownerWork...)
		blockers = append(blockers, ownerBlockers...)
		complete = ownerComplete
	}
	var components []*commonv1.LifecycleManagedComponent
	workQuiesced := complete && len(blockers) == 0 && p.active == 0
	if p.components != nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		var componentWork []*commonv1.LifecycleWorkItem
		var componentBlockers []*commonv1.LifecycleBlocker
		var componentComplete bool
		components, componentWork, componentBlockers, componentComplete = p.components(ctx, p.componentExits)
		cancel()
		work, blockers = append(work, componentWork...), append(blockers, componentBlockers...)
		complete = complete && componentComplete
	} else {
		for _, owner := range []string{"desktop-owner-and-helper"} {
			complete = false
			work = append(work, &commonv1.LifecycleWorkItem{Id: "owner/" + owner, Kind: owner, Status: "unknown"})
			blockers = append(blockers, &commonv1.LifecycleBlocker{Code: "owner_inventory_unavailable", Message: "owner has not supplied lifecycle completion evidence: " + owner, Disposition: commonv1.LifecycleBlockerDisposition_LIFECYCLE_BLOCKER_DISPOSITION_UNKNOWN, Observed: true})
		}
	}
	drained := p.closed && p.active == 0 && complete && len(blockers) == 0
	phase := commonv1.LifecycleMaintenancePhase_LIFECYCLE_MAINTENANCE_PHASE_OPEN
	if p.closed {
		phase = commonv1.LifecycleMaintenancePhase_LIFECYCLE_MAINTENANCE_PHASE_CLOSING
	}
	if drained {
		phase = commonv1.LifecycleMaintenancePhase_LIFECYCLE_MAINTENANCE_PHASE_DRAINED
	}
	token := p.token
	if token == "" {
		token = strconv.FormatInt(p.revision, 10)
	}
	return &commonv1.LifecycleStanding{Provider: "device-control", ProviderInstanceId: p.instance, Scenario: "device-control", InstanceId: p.instance, OperationId: p.operation, FenceToken: token, Revision: p.revision, Phase: phase, AdmissionClosed: p.closed, Admitting: int32(p.active), Remaining: int32(len(blockers)), InventoryComplete: complete, Drained: drained, Work: work, Blockers: blockers, ObservedAt: timestamppb.Now(), Interlock: "device-control-owner-fence-v1", ManagedComponents: components, ComponentStopReady: p.closed && p.componentQuiesced && complete && workQuiesced}
}
