package lifecycle

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	commonconnect "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1/commonv1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const maintenanceServicePath = "/common.v1.LifecycleMaintenanceService/"

const (
	// DefaultDrainTimeout bounds a Drain request that carries no timeout of its
	// own. A handler that never returns would otherwise hold the fence closed
	// forever: every non-health request is refused while the replacement waits.
	// The bound exceeds the longest request budget any scenario sets today, so
	// a legitimate slow request still drains, and the control plane's
	// force-lifecycle path applies a much shorter bound of its own.
	DefaultDrainTimeout = 15 * time.Minute

	// admittedRequestKind labels the standing's work items that describe an
	// admitted HTTP request still running.
	admittedRequestKind = "http_request"

	// maxReportedRequests caps how many admitted requests one standing or
	// timeout message names, longest-running first. The count stays exact.
	maxReportedRequests = 16
)

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

	mu        sync.Mutex
	quiesceMu sync.Mutex
	changed   chan struct{}
	// inflight holds every admitted request until its handler returns, so a
	// hung handler can be named instead of only counted.
	inflight    map[uint64]admittedRequest
	nextRequest uint64
	now         func() time.Time
	closed      bool
	closing     chan struct{}
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
		closing:  make(chan struct{}),
		inflight: make(map[uint64]admittedRequest),
		now:      time.Now,
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

// admittedRequest identifies one request the provider let in.
type admittedRequest struct {
	method  string
	path    string
	started time.Time
}

func (r admittedRequest) describe(now time.Time) string {
	return fmt.Sprintf("%s %s (running %s since %s)", r.method, r.path, now.Sub(r.started).Round(time.Second), r.started.UTC().Format(time.RFC3339))
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
		leave, closing, err := p.admit(r)
		if err != nil {
			http.Error(w, "service unavailable: lifecycle admission closed", http.StatusServiceUnavailable)
			return
		}
		defer leave()
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), closingKey{}, closing)))
	})
}

type closingKey struct{}

// Closing returns a signal closed when maintenance closes admission for the
// request's generation. Resumable long polls can return a normal partial result
// on this signal; ordinary requests must continue to completion. It is nil
// outside provider middleware.
func Closing(ctx context.Context) <-chan struct{} {
	closing, _ := ctx.Value(closingKey{}).(<-chan struct{})
	return closing
}

func isHealthPath(path string) bool {
	return path == "/health" || path == "/api/v1/health"
}

func (p *Provider) admit(r *http.Request) (func(), <-chan struct{}, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return nil, nil, errors.New("lifecycle admission closed")
	}
	p.nextRequest++
	id := p.nextRequest
	p.inflight[id] = admittedRequest{method: r.Method, path: r.URL.Path, started: p.now()}
	var once sync.Once
	return func() {
		once.Do(func() {
			p.mu.Lock()
			delete(p.inflight, id)
			p.signalLocked()
			p.mu.Unlock()
		})
	}, p.closing, nil
}

// runningRequestsLocked returns the admitted requests, longest-running first.
func (p *Provider) runningRequestsLocked() []admittedRequest {
	running := make([]admittedRequest, 0, len(p.inflight))
	for _, request := range p.inflight {
		running = append(running, request)
	}
	sort.Slice(running, func(i, j int) bool { return running[i].started.Before(running[j].started) })
	return running
}

// describeRunning names the longest-running admitted requests for an error
// message, so a drain that cannot finish says which handlers are holding it.
func (p *Provider) describeRunning() string {
	p.mu.Lock()
	running := p.runningRequestsLocked()
	now := p.now()
	p.mu.Unlock()
	if len(running) == 0 {
		return "no admitted requests"
	}
	parts := make([]string, 0, maxReportedRequests)
	for i, request := range running {
		if i == maxReportedRequests {
			parts = append(parts, fmt.Sprintf("and %d more", len(running)-maxReportedRequests))
			break
		}
		parts = append(parts, request.describe(now))
	}
	return fmt.Sprintf("%d admitted request(s) still running: %s", len(running), strings.Join(parts, "; "))
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
		if !p.closed {
			close(p.closing)
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
	timeout := DefaultDrainTimeout
	if requested := req.Msg.GetTimeout(); requested != nil && requested.AsDuration() > 0 {
		timeout = requested.AsDuration()
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
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
			return nil, connect.NewError(connect.CodeDeadlineExceeded, fmt.Errorf("lifecycle drain timed out after %s: %s: %w", timeout, p.describeRunning(), ctx.Err()))
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
	p.closing = make(chan struct{})
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
	closed, operation, token, revision, quiesced := p.closed, p.operationID, p.token, p.revision, p.quiesced
	active := int32(len(p.inflight))
	running := p.runningRequestsLocked()
	now := p.now()
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
	// Remaining counts detached owner work only. Admitted requests are
	// reported separately in Admitting; every provider reports the two counters
	// apart so a control plane never has to guess which kind of work is left.
	remaining := owner.Remaining
	if !owner.Complete {
		remaining = -1
	}
	work := append([]*commonv1.LifecycleWorkItem(nil), owner.Work...)
	for i, request := range running {
		if i == maxReportedRequests {
			break
		}
		work = append(work, &commonv1.LifecycleWorkItem{
			Id:     request.method + " " + request.path,
			Kind:   admittedRequestKind,
			Status: fmt.Sprintf("running %s since %s", now.Sub(request.started).Round(time.Second), request.started.UTC().Format(time.RFC3339)),
		})
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
		Work:               work,
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
