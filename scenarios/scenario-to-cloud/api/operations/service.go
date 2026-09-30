package operations

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/operationcoord"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/execplan"
	"scenario-to-cloud/faultinject"
	"scenario-to-cloud/persistence"
)

// Config bounds every timeout separately. Each bound has one meaning:
//
//   - QueueTimeout: how long a submitted operation may sit in the in-process
//     queue before the worker pool gives it back to the durable record (it
//     stays admitted; the reconciler picks it up). Bounded so a full pool
//     never holds an admission hostage in memory.
//   - ExecutionTimeout: the deadline of one worker attempt. Reaching it is a
//     lost lease, never an inferred failure: the operation stays running with
//     its receipts and the reconciler resumes it.
//   - TransportTimeout: one target call (receipt read). A timeout is an
//     unknown effect, recorded as such.
//   - ObserverTimeout: the maximum server-side block of GET …/wait. Observers
//     never change operation state.
//   - LeaseTTL / HeartbeatInterval / ReconcileInterval: ownership lease,
//     how often the owner renews it, how often unowned or expired records
//     are scanned.
type Config struct {
	WorkerID          string
	Workers           int
	QueueTimeout      time.Duration
	ExecutionTimeout  time.Duration
	TransportTimeout  time.Duration
	ObserverTimeout   time.Duration
	LeaseTTL          time.Duration
	HeartbeatInterval time.Duration
	ReconcileInterval time.Duration
	Logger            func(msg string, fields map[string]interface{})
}

// DefaultConfig is the production bound set.
func DefaultConfig() Config {
	host, _ := os.Hostname()
	return Config{
		WorkerID:          fmt.Sprintf("%s-%d-%s", host, os.Getpid(), uuid.New().String()[:8]),
		Workers:           2,
		QueueTimeout:      2 * time.Minute,
		ExecutionTimeout:  45 * time.Minute,
		TransportTimeout:  60 * time.Second,
		ObserverTimeout:   5 * time.Minute,
		LeaseTTL:          90 * time.Second,
		HeartbeatInterval: 30 * time.Second,
		ReconcileInterval: 90 * time.Second,
	}
}

func (c Config) withDefaults() Config {
	d := DefaultConfig()
	if c.WorkerID == "" {
		c.WorkerID = d.WorkerID
	}
	if c.Workers <= 0 {
		c.Workers = d.Workers
	}
	if c.QueueTimeout <= 0 {
		c.QueueTimeout = d.QueueTimeout
	}
	if c.ExecutionTimeout <= 0 {
		c.ExecutionTimeout = d.ExecutionTimeout
	}
	if c.TransportTimeout <= 0 {
		c.TransportTimeout = d.TransportTimeout
	}
	if c.ObserverTimeout <= 0 {
		c.ObserverTimeout = d.ObserverTimeout
	}
	if c.LeaseTTL <= 0 {
		c.LeaseTTL = d.LeaseTTL
	}
	if c.HeartbeatInterval <= 0 || c.HeartbeatInterval >= c.LeaseTTL {
		c.HeartbeatInterval = c.LeaseTTL / 3
	}
	if c.ReconcileInterval <= 0 {
		c.ReconcileInterval = c.LeaseTTL
	}
	return c
}

// ExecuteOptions are the non-durable execution inputs a submission carries.
// Provided secrets are never persisted: after an owner restart they are gone
// and a plan that still needs them moves to waiting_input.
type ExecuteOptions struct {
	RunPreflight     bool
	ForceBundleBuild bool
	ProvidedSecrets  map[string]string
}

// ExecutionContext is what a Runner receives for one attempt.
type ExecutionContext struct {
	Operation *domain.CloudOperation
	Plan      *execplan.Plan
	Fence     uint64
	Steps     StepSink
	Options   ExecuteOptions
	// Resumed is true when a previous attempt already committed receipts.
	Resumed bool
}

// Runner executes an admitted plan. The adapter over vps.ExecutePlan lives
// in the deployment package; this package owns the durable bookkeeping.
type Runner interface {
	Execute(ctx context.Context, ec *ExecutionContext) error
}

// RunnerFunc adapts a function to Runner.
type RunnerFunc func(ctx context.Context, ec *ExecutionContext) error

// Execute implements Runner.
func (f RunnerFunc) Execute(ctx context.Context, ec *ExecutionContext) error { return f(ctx, ec) }

// Decision is what StepSink.Begin tells the runner to do with an action.
type Decision int

// Decisions.
const (
	// DecisionRun executes the action.
	DecisionRun Decision = iota
	// DecisionSkip does not execute: a receipt already proves the outcome.
	DecisionSkip
)

// StepSink is the per-action seam between the runner and the durable record.
// Begin is called before an action; Commit after it (with the observed
// outcome). Both are fenced: a stale worker fails here.
type StepSink interface {
	Begin(ctx context.Context, action execplan.Action) (Decision, error)
	Commit(ctx context.Context, action execplan.Action, outcome StepOutcome, detail string, execErr error) error
}

// TargetReceipt is what the target owner recorded for (operation, step).
type TargetReceipt struct {
	Found   bool
	Outcome StepOutcome
	Fence   uint64
	Detail  string
	Error   string
}

// TargetReceipts reads receipts from the target owner. ErrNativeCLIAbsent
// says the target cannot answer (no native binary): the caller records an
// unknown effect instead of replaying.
type TargetReceipts interface {
	Read(ctx context.Context, op *domain.CloudOperation, step string) (TargetReceipt, error)
}

// ErrNativeCLIAbsent is returned by a TargetReceipts implementation when the
// target has no receipt owner to ask.
var ErrNativeCLIAbsent = errors.New("native vrooli binary is absent on the target; receipts cannot be read")

// Sentinel control errors a runner returns through StepSink.
var (
	// ErrCancelled: a cancel point was reached with cancellation requested.
	ErrCancelled = errors.New("operation cancelled at a safe boundary")
	// ErrUnknownEffect: a step's outcome is unknown and could not be observed.
	ErrUnknownEffect = errors.New("step outcome unknown; reconciliation required")
	// ErrWaitingInput: execution needs input that is not durable (secrets).
	ErrWaitingInput = errors.New("operation needs input")
	// ErrRecoveryRequired: a consistency-critical step failed hard.
	ErrRecoveryRequired = errors.New("recovery required")
	// ErrReplayStep is returned by Commit when a step with an unknown outcome
	// may be replayed under its retry contract; the runner re-runs it once.
	ErrReplayStep = errors.New("replay step")
)

// Event is the progress event published to subscribers and waiters.
type Event struct {
	OperationID string `json:"operation_id"`
	State       State  `json:"state"`
	Step        string `json:"step,omitempty"`
	Message     string `json:"message,omitempty"`
	Timestamp   string `json:"timestamp"`
}

// Projection lets the worker mirror operation state onto the legacy
// deployment record (status/progress) and the SSE hub. It is a write-only
// projection; nothing reads it back for decisions.
type Projection interface {
	OperationStarted(ctx context.Context, op *domain.CloudOperation, plan *execplan.Plan)
	OperationFinished(ctx context.Context, op *domain.CloudOperation, plan *execplan.Plan, to State, result *Result, failure *apierrors.Error)
}

// Service owns the in-process worker pool. The durable record is the owner
// of every operation; the pool is compute. Losing the pool (restart) loses
// nothing but the lease.
type Service struct {
	cfg      Config
	repo     *persistence.Repository
	runner   Runner
	receipts TargetReceipts
	proj     Projection
	coord    *operationcoord.Coordinator[Event]

	queue chan submission
	wg    sync.WaitGroup
	stop  chan struct{}
	once  sync.Once

	mu      sync.Mutex
	inhand  map[string]struct{}
	started bool

	// heartbeatsPaused simulates a stalled owner (GC pause, partition) in
	// tests: the worker keeps running but stops renewing its lease.
	heartbeatsPaused atomic.Bool
}

// PauseHeartbeats stops (or resumes) lease renewal for every worker of this
// service. It exists to prove the stale-fence contract; production never
// calls it.
func (s *Service) PauseHeartbeats(paused bool) { s.heartbeatsPaused.Store(paused) }

type submission struct {
	ctx     context.Context
	opID    string
	options ExecuteOptions
	// enqueued lets the pool drop a submission that exceeded QueueTimeout
	// (the record stays admitted for the reconciler).
	enqueued time.Time
}

// NewService builds the service. runner and repo are required; receipts may
// be nil (every unknown outcome then becomes an unknown effect); proj may be nil.
func NewService(cfg Config, repo *persistence.Repository, runner Runner, receipts TargetReceipts, proj Projection) *Service {
	cfg = cfg.withDefaults()
	return &Service{
		cfg:      cfg,
		repo:     repo,
		runner:   runner,
		receipts: receipts,
		proj:     proj,
		coord:    operationcoord.New[Event](),
		queue:    make(chan submission, 64),
		stop:     make(chan struct{}),
		inhand:   map[string]struct{}{},
	}
}

// Config returns the effective bounds.
func (s *Service) Config() Config { return s.cfg }

// WorkerID identifies this owner incarnation.
func (s *Service) WorkerID() string { return s.cfg.WorkerID }

// Start launches the pool and the reconciler. It is idempotent.
func (s *Service) Start() {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return
	}
	s.started = true
	s.mu.Unlock()
	for i := 0; i < s.cfg.Workers; i++ {
		s.wg.Add(1)
		go s.workerLoop()
	}
	s.wg.Add(1)
	go s.reconcileLoop()
}

// Stop halts the pool. Running attempts are cancelled (their lease expires
// and a successor resumes from receipts); nothing terminal is written.
func (s *Service) Stop() {
	s.once.Do(func() { close(s.stop) })
	s.wg.Wait()
}

// Submit hands an admitted operation to the pool. It returns immediately;
// the caller already holds the durable operation id as its wait reference.
// If the queue is full the operation simply stays admitted and the
// reconciler will pick it up: submission is a hint, not ownership.
func (s *Service) Submit(ctx context.Context, opID string, options ExecuteOptions) {
	sub := submission{ctx: detach(ctx), opID: opID, options: options, enqueued: time.Now()}
	select {
	case s.queue <- sub:
	default:
		s.log("operation queue full; leaving operation admitted for reconciliation", map[string]interface{}{"operation_id": opID})
	}
}

// detach builds the worker context: no request cancellation (the client may
// disconnect; the work is server-owned), but the routed test-mode mark and
// the fault-injection registry are preserved so routed runs stay isolated.
func detach(ctx context.Context) context.Context {
	out := context.Background()
	if database.IsTestMode(ctx) {
		out = database.WithTestMode(out)
	}
	if reg, ok := faultinject.FromContext(ctx); ok {
		out = faultinject.WithRegistry(out, reg)
	}
	return out
}

func (s *Service) workerLoop() {
	defer s.wg.Done()
	for {
		select {
		case <-s.stop:
			return
		case sub := <-s.queue:
			if time.Since(sub.enqueued) > s.cfg.QueueTimeout {
				s.log("operation exceeded queue timeout; left admitted for reconciliation", map[string]interface{}{"operation_id": sub.opID})
				continue
			}
			s.runAttempt(sub.ctx, sub.opID, sub.options)
		}
	}
}

func (s *Service) reconcileLoop() {
	defer s.wg.Done()
	ticker := time.NewTicker(s.cfg.ReconcileInterval)
	defer ticker.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), s.cfg.ReconcileInterval)
			_, _ = s.Reconcile(ctx)
			cancel()
		}
	}
}

// claim marks an operation as in hand in this process so two pool workers
// never race on one id (the durable lease is the real guard).
func (s *Service) claim(opID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, busy := s.inhand[opID]; busy {
		return false
	}
	s.inhand[opID] = struct{}{}
	return true
}

func (s *Service) release(opID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.inhand, opID)
}

// runAttempt is one ownership attempt. Every exit path leaves the durable
// record truthful: either a terminal state, a waiting/reconciling state with
// a next action, or a running record whose lease will expire.
func (s *Service) runAttempt(parent context.Context, opID string, options ExecuteOptions) {
	if !s.claim(opID) {
		return
	}
	defer s.release(opID)
	ctx, cancel := context.WithTimeout(parent, s.cfg.ExecutionTimeout)
	defer cancel()

	fence, err := s.repo.AcquireWorker(ctx, opID, s.cfg.WorkerID, s.cfg.LeaseTTL)
	if err != nil {
		s.log("operation not acquired", map[string]interface{}{"operation_id": opID, "error": err.Error()})
		return
	}
	op, err := s.repo.GetOperation(ctx, opID)
	if err != nil || op == nil {
		s.log("operation vanished after acquisition", map[string]interface{}{"operation_id": opID})
		return
	}
	w := &worker{svc: s, op: op, fence: fence, options: options}
	w.run(ctx)
}

// Reconcile performs one pass over every non-terminal operation: expired or
// absent leases are reacquired with a new fence and resumed from receipts;
// live leases held by another worker are left alone (a lease is trusted
// until it expires — a slow owner is not a dead owner). It returns the ids
// it took ownership of.
func (s *Service) Reconcile(ctx context.Context) ([]string, error) {
	ops, err := s.repo.ListNonTerminal(ctx)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	var taken []string
	for _, op := range ops {
		switch op.State {
		case WaitingInput, Recovering:
			// Needs an operator (input) or a recovery owner; nothing to run.
			continue
		}
		if leaseLive(op, now) && op.WorkerID != s.cfg.WorkerID {
			continue
		}
		if !s.claim(op.ID) {
			continue
		}
		taken = append(taken, op.ID)
		s.log("reconciling operation", map[string]interface{}{
			"operation_id": op.ID, "deployment_id": op.DeploymentID, "state": string(op.State),
			"previous_worker": op.WorkerID, "previous_fence": op.Fence, "active_step": activeStepID(op),
		})
		id := op.ID
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			defer s.release(id)
			s.runReconciled(detach(ctx), id)
		}()
	}
	return taken, nil
}

func (s *Service) runReconciled(parent context.Context, opID string) {
	ctx, cancel := context.WithTimeout(parent, s.cfg.ExecutionTimeout)
	defer cancel()
	fence, err := s.repo.AcquireWorker(ctx, opID, s.cfg.WorkerID, s.cfg.LeaseTTL)
	if err != nil {
		s.log("reconcile: operation not acquired", map[string]interface{}{"operation_id": opID, "error": err.Error()})
		return
	}
	op, err := s.repo.GetOperation(ctx, opID)
	if err != nil || op == nil {
		return
	}
	w := &worker{svc: s, op: op, fence: fence, resumed: true}
	w.run(ctx)
}

func leaseLive(op *domain.CloudOperation, now time.Time) bool {
	return op.WorkerID != "" && op.LeaseExpiresAt != nil && op.LeaseExpiresAt.After(now)
}

func activeStepID(op *domain.CloudOperation) string {
	if marker := op.ActiveStepMarker(); marker != nil {
		return marker.Step
	}
	return ""
}

// Get returns the durable record.
func (s *Service) Get(ctx context.Context, opID string) (*domain.CloudOperation, error) {
	op, err := s.repo.GetOperation(ctx, opID)
	if err != nil {
		return nil, err
	}
	if op == nil {
		return nil, apierrors.New(apierrors.CodeOperationNotFound, "Operation not found").WithDetail("operation_id", opID)
	}
	return op, nil
}

// Cancel records the cancellation intent. The worker honours it at the next
// declared cancel point; an operation without a worker is cancelled at once.
func (s *Service) Cancel(ctx context.Context, opID string) (*domain.CloudOperation, error) {
	op, err := s.repo.RequestCancel(ctx, opID)
	if err != nil {
		return nil, err
	}
	s.publish(op, "", "cancellation requested")
	if op.State.IsTerminal() {
		s.coord.SignalTerminal(op.ID)
	}
	return op, nil
}

// Wait blocks until the operation is terminal or timeout elapses. It
// registers the waiter before the durable recheck so a terminal transition
// racing the recheck is observed. The observer's timeout never touches the
// record: the second return value is true when the wait ended pending.
func (s *Service) Wait(ctx context.Context, opID string, timeout time.Duration) (*domain.CloudOperation, bool, error) {
	if timeout <= 0 || timeout > s.cfg.ObserverTimeout {
		timeout = s.cfg.ObserverTimeout
	}
	done, cancel := s.coord.RegisterWaiter(opID)
	defer cancel()
	op, err := s.Get(ctx, opID)
	if err != nil {
		return nil, false, err
	}
	if op.State.IsTerminal() {
		return op, false, nil
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-done:
		op, err = s.Get(ctx, opID)
		return op, false, err
	case <-timer.C:
		op, err = s.Get(ctx, opID)
		if err != nil {
			return nil, false, err
		}
		return op, !op.State.IsTerminal(), nil
	case <-ctx.Done():
		op, err = s.Get(context.WithoutCancel(ctx), opID)
		if err != nil {
			return nil, false, err
		}
		return op, !op.State.IsTerminal(), nil
	}
}

// RegisterWaiter exposes the coordinator's terminal waiter (register before
// the durable recheck).
func (s *Service) RegisterWaiter(opID string) (<-chan struct{}, func()) {
	return s.coord.RegisterWaiter(opID)
}

// Subscribe streams events for an operation.
func (s *Service) Subscribe(opID string) (<-chan Event, func()) {
	return s.coord.Subscribe(opID)
}

func (s *Service) publish(op *domain.CloudOperation, step, message string) {
	s.coord.Publish(op.ID, Event{OperationID: op.ID, State: op.State, Step: step, Message: message, Timestamp: time.Now().UTC().Format(time.RFC3339Nano)})
}

func (s *Service) log(msg string, fields map[string]interface{}) {
	if s.cfg.Logger != nil {
		s.cfg.Logger(msg, fields)
	}
}

// DecodePlan reads the admitted plan JSON.
func DecodePlan(op *domain.CloudOperation) (*execplan.Plan, error) {
	if op == nil || len(op.Plan) == 0 {
		return nil, apierrors.New(apierrors.CodeInvalidRequest, "operation carries no plan").WithDetail("operation_id", opID(op))
	}
	var plan execplan.Plan
	if err := json.Unmarshal(op.Plan, &plan); err != nil {
		return nil, apierrors.New(apierrors.CodeInvalidRequest, "operation plan is not decodable").WithDetail("operation_id", op.ID).WithDetail("cause", err.Error())
	}
	if strings.TrimSpace(plan.SchemaVersion) != execplan.SchemaVersion {
		// RUN-10: an operation stored by an unknown schema is an explicit
		// incompatibility, never a guessed success.
		return nil, apierrors.New(apierrors.CodeUnsupportedSchemaVersion, "operation plan schema is not supported by this owner").
			WithDetail("operation_id", op.ID).WithDetail("stored_schema_version", plan.SchemaVersion).WithDetail("supported", execplan.SchemaVersion)
	}
	return &plan, nil
}

func opID(op *domain.CloudOperation) string {
	if op == nil {
		return ""
	}
	return op.ID
}
