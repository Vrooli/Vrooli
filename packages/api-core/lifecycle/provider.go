package lifecycle

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	commonconnect "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1/commonv1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const maintenanceServicePath = "/common.v1.LifecycleMaintenanceService/"

// Hooks represent scenario-owned work that continues outside an HTTP request.
// The shared provider tracks request admission itself. A scenario with
// background goroutines must supply Quiesce and Resume so those workers stop
// before replacement and restart if a prepared replacement is abandoned.
type Hooks struct {
	// Snapshot must return a complete, owner-authored view of detached work.
	// Its returned Remaining count is added to shared HTTP admission. A nil hook
	// asserts that all work belongs to admitted requests or the registered
	// quiesce hook.
	Snapshot func(context.Context) (Inventory, error)
	Quiesce  func(context.Context) error
	Resume   func(context.Context) error
}

// Inventory describes owner work that can outlive its initiating HTTP request.
type Inventory struct {
	Remaining int32
	Work      []*commonv1.LifecycleWorkItem
	Blockers  []*commonv1.LifecycleBlocker
	Complete  bool
}

// Provider is the shared ingress half of the maintenance-fence protocol.
// Owners with durable jobs or external processes should use their
// domain-specific inventory/provider and install its HTTP admission middleware
// instead of this zero-detached-work implementation.
type Provider struct {
	commonconnect.UnimplementedLifecycleMaintenanceServiceHandler

	scenario string
	instance string
	hooks    Hooks

	mu          sync.Mutex
	quiesceMu   sync.Mutex
	changed     chan struct{}
	active      int32
	closed      bool
	quiesced    bool
	operationID string
	token       string
	revision    int64
}

// NewProvider creates a process-generation-bound provider. instanceID should
// be the control plane's runtime instance ID; an empty value receives a random
// process identity.
func NewProvider(scenario, instanceID string, hooks Hooks) (*Provider, error) {
	scenario = strings.TrimSpace(scenario)
	instanceID = strings.TrimSpace(instanceID)
	if scenario == "" {
		return nil, errors.New("lifecycle: scenario identity is required")
	}
	if (hooks.Quiesce == nil) != (hooks.Resume == nil) {
		return nil, errors.New("lifecycle: background work requires both quiesce and resume hooks")
	}
	if instanceID == "" {
		var err error
		instanceID, err = randomToken(16)
		if err != nil {
			return nil, fmt.Errorf("lifecycle: create provider instance identity: %w", err)
		}
	}
	return &Provider{
		scenario: scenario,
		instance: instanceID,
		hooks:    hooks,
		changed:  make(chan struct{}),
		quiesced: hooks.Quiesce == nil,
		revision: 1,
	}, nil
}

// NewProviderFromEnv constructs the standard provider from the lifecycle
// identity environment installed by the control plane.
func NewProviderFromEnv(getenv func(string) string, hooks Hooks) (*Provider, error) {
	if getenv == nil {
		getenv = os.Getenv
	}
	scenario := strings.TrimSpace(getenv("VROOLI_SCENARIO"))
	if scenario == "" {
		scenario = strings.TrimSpace(getenv("SCENARIO_NAME"))
	}
	return NewProvider(scenario, getenv("VROOLI_RUNTIME_INSTANCE_ID"), hooks)
}

// Mount adds the loopback-only lifecycle RPC service before the scenario's
// catch-all handler. It returns a handler that also fences application ingress.
func (p *Provider) Mount(application http.Handler) http.Handler {
	if application == nil {
		application = http.NotFoundHandler()
	}
	mux := http.NewServeMux()
	path, handler := commonconnect.NewLifecycleMaintenanceServiceHandler(p)
	mux.Handle(path, LoopbackOnly(handler))
	mux.Handle("/", application)
	return p.Middleware(mux)
}

// Middleware closes scenario application admission atomically with Prepare
// and accounts for every admitted request until its handler returns. Health
// probes remain readable while maintenance is closed and are not counted.
func (p *Provider) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, maintenanceServicePath) || isHealthPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		leave, err := p.admit()
		if err != nil {
			http.Error(w, "service unavailable: lifecycle admission closed", http.StatusServiceUnavailable)
			return
		}
		defer leave()
		next.ServeHTTP(w, r)
	})
}

func isHealthPath(path string) bool {
	return path == "/health" || path == "/api/v1/health"
}

func (p *Provider) admit() (func(), error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return nil, errors.New("lifecycle admission closed")
	}
	p.active++
	var once sync.Once
	return func() {
		once.Do(func() {
			p.mu.Lock()
			p.active--
			p.signalLocked()
			p.mu.Unlock()
		})
	}, nil
}

func (p *Provider) Prepare(ctx context.Context, req *connect.Request[commonv1.LifecyclePrepareRequest]) (*connect.Response[commonv1.LifecyclePrepareResponse], error) {
	message := req.Msg
	if strings.TrimSpace(message.GetOperationId()) == "" || strings.TrimSpace(message.GetReason()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("operation_id and reason are required"))
	}
	if message.GetEmergencyOverride() && strings.TrimSpace(message.GetOverrideReason()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("override_reason is required for emergency override"))
	}
	p.mu.Lock()
	if p.closed && p.operationID != message.GetOperationId() && !message.GetEmergencyOverride() {
		p.mu.Unlock()
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("another lifecycle operation holds the fence"))
	}
	if !p.closed || p.operationID != message.GetOperationId() {
		token, err := randomToken(32)
		if err != nil {
			p.mu.Unlock()
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		p.closed = true
		p.operationID = message.GetOperationId()
		p.token = token
		p.revision++
		p.quiesced = p.hooks.Quiesce == nil
		p.signalLocked()
	}
	p.mu.Unlock()

	standing, err := p.standing(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	return connect.NewResponse(&commonv1.LifecyclePrepareResponse{Standing: standing}), nil
}

func (p *Provider) Status(ctx context.Context, req *connect.Request[commonv1.LifecycleStatusRequest]) (*connect.Response[commonv1.LifecycleStatusResponse], error) {
	p.mu.Lock()
	if (req.Msg.GetOperationId() != "" && req.Msg.GetOperationId() != p.operationID) ||
		(req.Msg.GetRevision() > 0 && req.Msg.GetRevision() != p.revision) ||
		(req.Msg.GetFenceToken() != "" && req.Msg.GetFenceToken() != p.token) {
		p.mu.Unlock()
		return nil, connect.NewError(connect.CodeAborted, errors.New("lifecycle fence identity mismatch"))
	}
	p.mu.Unlock()
	standing, err := p.standing(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	return connect.NewResponse(&commonv1.LifecycleStatusResponse{Standing: standing}), nil
}

func (p *Provider) Drain(ctx context.Context, req *connect.Request[commonv1.LifecycleDrainRequest]) (*connect.Response[commonv1.LifecycleDrainResponse], error) {
	if err := p.validateFence(req.Msg.GetOperationId(), req.Msg.GetFenceToken(), req.Msg.GetRevision()); err != nil {
		return nil, err
	}
	if len(req.Msg.GetComponentExits()) != 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("shared ingress provider has no external managed components"))
	}
	if timeout := req.Msg.GetTimeout(); timeout != nil && timeout.AsDuration() > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout.AsDuration())
		defer cancel()
	}
	if req.Msg.GetQuiesceComponents() {
		if err := p.quiesce(ctx); err != nil {
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("quiesce owner work: %w", err))
		}
	}
	for {
		standing, err := p.standing(ctx)
		if err != nil {
			return nil, connect.NewError(connect.CodeUnavailable, err)
		}
		if (req.Msg.GetQuiesceComponents() && standing.GetComponentStopReady()) || standing.GetDrained() {
			return connect.NewResponse(&commonv1.LifecycleDrainResponse{Standing: standing}), nil
		}
		p.mu.Lock()
		changed := p.changed
		p.mu.Unlock()
		select {
		case <-ctx.Done():
			return nil, connect.NewError(connect.CodeDeadlineExceeded, fmt.Errorf("lifecycle drain timed out: %w", ctx.Err()))
		case <-changed:
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func (p *Provider) Resume(ctx context.Context, req *connect.Request[commonv1.LifecycleResumeRequest]) (*connect.Response[commonv1.LifecycleResumeResponse], error) {
	if err := p.validateFence(req.Msg.GetOperationId(), req.Msg.GetFenceToken(), req.Msg.GetRevision()); err != nil {
		return nil, err
	}
	p.quiesceMu.Lock()
	defer p.quiesceMu.Unlock()
	if p.hooks.Resume != nil {
		if err := p.hooks.Resume(ctx); err != nil {
			return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("resume owner work: %w", err))
		}
	}
	p.mu.Lock()
	if !p.closed || p.operationID != req.Msg.GetOperationId() || p.token != req.Msg.GetFenceToken() || p.revision != req.Msg.GetRevision() {
		p.mu.Unlock()
		return nil, connect.NewError(connect.CodeAborted, errors.New("lifecycle fence changed during resume"))
	}
	p.closed = false
	p.quiesced = true
	p.operationID = ""
	p.token = ""
	p.revision++
	p.signalLocked()
	p.mu.Unlock()
	standing, err := p.standing(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	standing.Phase = commonv1.LifecycleMaintenancePhase_LIFECYCLE_MAINTENANCE_PHASE_RESUMED
	standing.OperationId = req.Msg.GetOperationId()
	return connect.NewResponse(&commonv1.LifecycleResumeResponse{Standing: standing}), nil
}

func (p *Provider) quiesce(ctx context.Context) error {
	p.quiesceMu.Lock()
	defer p.quiesceMu.Unlock()
	p.mu.Lock()
	if !p.closed {
		p.mu.Unlock()
		return errors.New("admission is open")
	}
	if p.quiesced {
		p.mu.Unlock()
		return nil
	}
	p.mu.Unlock()
	if p.hooks.Quiesce != nil {
		if err := p.hooks.Quiesce(ctx); err != nil {
			return err
		}
	}
	p.mu.Lock()
	p.quiesced = true
	p.signalLocked()
	p.mu.Unlock()
	return nil
}

func (p *Provider) validateFence(operation, token string, revision int64) error {
	if strings.TrimSpace(operation) == "" || strings.TrimSpace(token) == "" || revision <= 0 {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("operation_id, fence_token and revision are required"))
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.closed || p.operationID != operation || p.token != token || p.revision != revision {
		return connect.NewError(connect.CodeAborted, errors.New("lifecycle fence identity mismatch"))
	}
	return nil
}

func (p *Provider) standing(ctx context.Context) (*commonv1.LifecycleStanding, error) {
	p.mu.Lock()
	closed, operation, token, revision, active, quiesced := p.closed, p.operationID, p.token, p.revision, p.active, p.quiesced
	p.mu.Unlock()
	if ctx == nil {
		ctx = context.Background()
	}
	owner := Inventory{Complete: true}
	var ownerErr error
	if p.hooks.Snapshot != nil {
		owner, ownerErr = p.hooks.Snapshot(ctx)
	}
	if ownerErr != nil {
		owner.Complete = false
		owner.Remaining = -1
		owner.Blockers = append(owner.Blockers, &commonv1.LifecycleBlocker{
			Code: "OWNER_INVENTORY_UNAVAILABLE", Message: ownerErr.Error(),
			Disposition: commonv1.LifecycleBlockerDisposition_LIFECYCLE_BLOCKER_DISPOSITION_UNKNOWN,
		})
	}
	if owner.Remaining < 0 {
		owner.Complete = false
	}
	remaining := active + owner.Remaining
	if !owner.Complete {
		remaining = -1
	}
	drained := closed && active == 0 && owner.Complete && owner.Remaining == 0 && len(owner.Blockers) == 0 && quiesced
	phase := commonv1.LifecycleMaintenancePhase_LIFECYCLE_MAINTENANCE_PHASE_OPEN
	if closed {
		phase = commonv1.LifecycleMaintenancePhase_LIFECYCLE_MAINTENANCE_PHASE_CLOSING
		if drained {
			phase = commonv1.LifecycleMaintenancePhase_LIFECYCLE_MAINTENANCE_PHASE_DRAINED
		}
	}
	standing := &commonv1.LifecycleStanding{
		Provider:           p.scenario,
		ProviderInstanceId: p.instance,
		Scenario:           p.scenario,
		InstanceId:         p.instance,
		OperationId:        operation,
		FenceToken:         token,
		Revision:           revision,
		Phase:              phase,
		AdmissionClosed:    closed,
		Admitting:          active,
		Remaining:          remaining,
		InventoryComplete:  owner.Complete,
		Drained:            drained,
		ObservedAt:         timestamppb.Now(),
		Interlock:          ProtocolVersion,
		FencePersistence:   commonv1.LifecycleFencePersistence_LIFECYCLE_FENCE_PERSISTENCE_PROCESS_LOCAL,
		ComponentStopReady: closed && quiesced && active == 0 && owner.Complete && owner.Remaining == 0 && len(owner.Blockers) == 0,
		QuiescencePending:  closed && p.hooks.Quiesce != nil && !quiesced,
		Work:               append([]*commonv1.LifecycleWorkItem(nil), owner.Work...),
		Blockers:           append([]*commonv1.LifecycleBlocker(nil), owner.Blockers...),
	}
	return standing, nil
}

func (p *Provider) signalLocked() {
	close(p.changed)
	p.changed = make(chan struct{})
}

func randomToken(size int) (string, error) {
	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

var _ commonconnect.LifecycleMaintenanceServiceHandler = (*Provider)(nil)
