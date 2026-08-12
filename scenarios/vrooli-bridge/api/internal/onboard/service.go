package onboard

import (
	"context"
	"fmt"
	"sync"
	"time"

	"vrooli-bridge/internal/clock"
	"vrooli-bridge/internal/onboarding"
)

// DefaultVerifyTimeout bounds the orchestrator's post-bootstrap ONLINE
// confirmation when a request passes verify_timeout_seconds <= 0.
const DefaultVerifyTimeout = 2 * time.Minute

// DefaultWaitTimeout bounds a Wait call that passes timeout <= 0, so a
// block-once wait can never hang forever even if the orchestrator vanishes.
const DefaultWaitTimeout = 30 * time.Minute

// subscriberBuffer bounds a live event subscriber's channel. A slow streaming
// client that fills its buffer drops the overflow rather than blocking ingest.
const subscriberBuffer = 256

// Service is the application-layer surface the onboard handler depends on. It
// owns the durable op lifecycle, the server-side orchestration (SSH → SCP →
// remote bootstrap → verify), the block-once Wait + live Subscribe coordination,
// and the startup reconciliation of ops orphaned by a control-plane restart.
type Service interface {
	// Start validates the request, and (unless a dry-run) creates a durable op
	// and launches the detached orchestration goroutine that drives it. Returns
	// the created op id; the caller then blocks on Wait.
	Start(ctx context.Context, in StartInput) (Decision, error)

	// StartMachineEnrollment creates immutable Machine intent before any SSH
	// contact, then launches its correlated orchestration operation.
	StartMachineEnrollment(ctx context.Context, machineID string, in StartInput) (MachineEnrollmentDecision, error)

	// RetryMachineEnrollment creates a new immutable attempt linked to a
	// terminal predecessor, then launches a fresh orchestration. It never
	// reopens or mutates the failed attempt.
	RetryMachineEnrollment(ctx context.Context, machineID, priorAttemptID string, in StartInput) (MachineEnrollmentDecision, error)

	// GetOp returns an op and its full persisted step-event history.
	GetOp(ctx context.Context, id string) (Op, []StepEvent, error)

	// ListOps returns ops newest-first, narrowed by filter.
	ListOps(ctx context.Context, filter ListFilter) ([]Op, error)

	// Wait blocks until the op reaches a terminal state or the timeout elapses,
	// returning exactly once. timedOut is true when the deadline elapsed first.
	Wait(ctx context.Context, id string, timeout time.Duration) (op Op, timedOut bool, err error)

	// Cancel requests cancellation of a non-terminal op; the orchestrator drives
	// it to CANCELLED at the next boundary. A terminal op is returned unchanged.
	Cancel(ctx context.Context, id string) (Op, error)

	// RemoveFailed permanently removes a failed attempt's local operation
	// history. It never contacts the target machine or removes a fleet node.
	RemoveFailed(ctx context.Context, id string) error

	// Subscribe registers a live step-event subscriber for the op, returning a
	// channel and an unsubscribe func the caller MUST invoke.
	Subscribe(id string) (<-chan StepEvent, func())

	// ResumeInterrupted reconciles ops left non-terminal by a control-plane
	// restart: each is marked FAILED with FailureInterrupted (safe to retry — the
	// flow is idempotent). Returns the count reconciled. Called once at boot.
	ResumeInterrupted(ctx context.Context) (int, error)
}

type MachineEnrollmentDecision struct {
	Attempt  EnrollmentAttempt
	Decision Decision
}

type service struct {
	repo      Repository
	driver    SSHDriver
	issuer    CodeIssuer
	resolver  EnrollmentResolver
	linker    MachineLinker
	confirmer OnlineConfirmer
	clock     clock.Clock
	coord     *coordinator

	defaultControlPlaneURL string
	endpointResolver       EndpointResolver
	defaultRevision        string
	defaultScopes          []string
	revResolver            RevisionResolver
	worktree               WorkingTreeSource
	artifacts              ArtifactBuilder
	nodeRev                NodeRevisionRecorder
	handoff                onboarding.HandoffClient
	firewallAdmitter       FirewallAdmitter

	wg sync.WaitGroup // tracks in-flight orchestration goroutines (for tests)
}

// Option customises the service.
type Option func(*service)

// WithDefaultControlPlaneURL sets the control-plane URL used when a request
// omits one (phase 6 wires the real @cp default; phase 5 leaves it empty).
func WithDefaultControlPlaneURL(url string) Option {
	return func(s *service) { s.defaultControlPlaneURL = url }
}

// WithEnrollmentResolver supplies typed pairing outcomes for the orchestration
// path. Production wires pairing.Service; unit tests may inject a deterministic
// fake without parsing bootstrap diagnostics.
func WithEnrollmentResolver(resolver EnrollmentResolver) Option {
	return func(s *service) { s.resolver = resolver }
}

func WithMachineLinker(linker MachineLinker) Option { return func(s *service) { s.linker = linker } }

// EndpointResolver supplies the persisted host-level endpoint selection for a
// request that did not explicitly name an endpoint. Its result is still
// validated below and copied into the immutable onboarding operation.
type EndpointResolver func(context.Context) (url, mode string, err error)

// WithEndpointResolver makes durable host configuration the default for new
// onboarding attempts. An explicit request URL or mode always wins.
func WithEndpointResolver(resolver EndpointResolver) Option {
	return func(s *service) { s.endpointResolver = resolver }
}

// WithDefaultRevision sets the target revision used when a request omits one and
// no RevisionResolver is wired (the legacy fallback).
func WithDefaultRevision(rev string) Option {
	return func(s *service) { s.defaultRevision = rev }
}

// WithDefaultScopes sets the posture-selected execution scopes carried by a
// newly issued onboarding pairing code. The slice is copied so the policy
// remains immutable after service construction; an explicit narrower grant is
// still owned by the pairing API and can replace these defaults.
func WithDefaultScopes(scopes []string) Option {
	return func(s *service) { s.defaultScopes = append([]string(nil), scopes...) }
}

// WithRevisionResolver wires the control-plane revision resolver. When set, the
// target revision is defaulted (empty/"@cp" → the control plane's commit),
// metacharacter-validated, and preflighted against the clone remote; when unset,
// the legacy WithDefaultRevision behaviour applies.
func WithRevisionResolver(r RevisionResolver) Option {
	return func(s *service) { s.revResolver = r }
}

// WithWorkingTreeSource wires the control-plane working-tree snapshotter used in
// working-tree source mode. Unset (the default), a working-tree StartOnboarding is
// refused at Start with a clear error rather than silently degrading to pinned.
func WithWorkingTreeSource(w WorkingTreeSource) Option {
	return func(s *service) { s.worktree = w }
}

// WithArtifactBuilder wires the control-plane cross-builder used by working-tree
// onboarding. It is deliberately separate from SSH transport and is never used
// by pinned-revision mode.
func WithArtifactBuilder(b ArtifactBuilder) Option {
	return func(s *service) { s.artifacts = b }
}

// WithNodeRevisionRecorder wires the seam that stamps a node's provenance revision
// after it verifies ONLINE. Unset, provenance is recorded on the op only (the node
// record keeps whatever revision pairing left).
func WithNodeRevisionRecorder(r NodeRevisionRecorder) Option {
	return func(s *service) { s.nodeRev = r }
}

// WithOnboardingHandoff enables the optional cross-scenario selection
// handoff. When omitted, onboarding remains fully Bridge-local and unchanged.
func WithOnboardingHandoff(client onboarding.HandoffClient) Option {
	return func(s *service) { s.handoff = client }
}

// WithFirewallAdmitter enables automatic, scoped UFW admission recovery for a
// LAN candidate whose first reachability probe identifies a valid source IP.
// It is intentionally opt-in at composition time: production wires the local
// setup-managed broker, while isolated tests and other deployments remain inert.
func WithFirewallAdmitter(admitter FirewallAdmitter) Option {
	return func(s *service) { s.firewallAdmitter = admitter }
}

// NewService constructs the production Service.
func NewService(repo Repository, driver SSHDriver, issuer CodeIssuer, confirmer OnlineConfirmer, clk clock.Clock, opts ...Option) Service {
	s := &service{
		repo:      repo,
		driver:    driver,
		issuer:    issuer,
		confirmer: confirmer,
		clock:     clk,
		coord:     newCoordinator(),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Compile-time guarantee.
var _ Service = (*service)(nil)

func (s *service) Start(ctx context.Context, in StartInput) (Decision, error) {
	host := trimField(in.Host)
	if host == "" {
		zeroBytes(in.Password)
		return Decision{}, ErrInvalid{Field: "host", Reason: "required"}
	}
	user := trimField(in.User)
	if user == "" {
		user = "root"
	}
	port := in.Port
	if port == 0 {
		port = 22
	}
	cpURL := trimField(in.ControlPlaneURL)
	configuredMode := ""
	if cpURL == "" {
		if s.endpointResolver != nil {
			var resolveErr error
			cpURL, configuredMode, resolveErr = s.endpointResolver(ctx)
			if resolveErr != nil {
				zeroBytes(in.Password)
				return Decision{}, ErrInvalid{Field: "control_plane_url", Reason: resolveErr.Error()}
			}
		} else {
			cpURL = s.defaultControlPlaneURL
		}
	}
	requestedMode := in.ReachabilityMode
	if trimField(requestedMode) == "" {
		requestedMode = configuredMode
	}
	mode, err := normalizeReachabilityMode(requestedMode)
	if err != nil {
		zeroBytes(in.Password)
		return Decision{}, ErrInvalid{Field: "reachability_mode", Reason: err.Error()}
	}
	cpURL, err = ValidateControlPlaneURL(cpURL, mode)
	if err != nil {
		zeroBytes(in.Password)
		return Decision{}, ErrInvalid{Field: "control_plane_url", Reason: err.Error()}
	}
	if cpURL == "" {
		zeroBytes(in.Password)
		return Decision{}, ErrInvalid{Field: "control_plane_url", Reason: "required"}
	}
	// Working-tree mode ships the control plane's local tree over SSH; it requires
	// the snapshotter to be wired (production always does). Refuse loudly rather
	// than silently falling back to a pinned clone the operator did not ask for.
	if in.WorkingTree() && s.worktree == nil {
		zeroBytes(in.Password)
		return Decision{}, ErrInvalid{Field: "source_mode", Reason: "working-tree onboarding is not available on this control plane (no working-tree source configured)"}
	}
	if in.WorkingTree() && s.artifacts == nil {
		zeroBytes(in.Password)
		return Decision{}, ErrInvalid{Field: "source_mode", Reason: "working-tree onboarding is not available on this control plane (no prebuilt artifact builder configured)"}
	}

	revision := trimField(in.TargetRevision)
	if s.revResolver != nil {
		// Default (empty/"@cp" → the control plane's commit) and validate. Pinned
		// mode ALSO preflights the commit against the clone remote (an unpushed
		// commit hard-fails here); working-tree mode skips that preflight because the
		// tree is shipped over SSH, not fetched — an unpushed base is expected.
		var resolved string
		var rerr error
		if in.WorkingTree() {
			resolved, rerr = s.revResolver.ResolveWorkingTree(ctx, revision)
		} else {
			resolved, rerr = s.revResolver.Resolve(ctx, revision)
		}
		if rerr != nil {
			zeroBytes(in.Password)
			return Decision{}, rerr
		}
		revision = resolved
	} else {
		if revision == "" {
			revision = s.defaultRevision
		}
		if revision == "" {
			zeroBytes(in.Password)
			return Decision{}, ErrInvalid{Field: "target_revision", Reason: "required"}
		}
	}

	// Validate the operator-chosen setup profile (metachar rejection + enum-ish
	// environment) before any host is touched, so a shell-injectable or nonsense
	// value fails here, loudly, rather than deep in the node-side script.
	if err := validateSetupProfile(in); err != nil {
		zeroBytes(in.Password)
		return Decision{}, err
	}

	// Normalise the resolved values back onto the input the goroutine consumes.
	// in.TargetRevision carries the BASE commit (the bootstrap `--revision`); in
	// working-tree mode BaseRevision holds it too so the op provenance is complete.
	in.Host, in.User, in.Port, in.ControlPlaneURL, in.TargetRevision, in.ReachabilityMode = host, user, port, cpURL, revision, string(mode)
	if in.WorkingTree() {
		in.BaseRevision = revision
	}

	// The op's persisted TargetRevision is what renders in `nodes`/`onboard`
	// output: the pinned commit, or a "<base>+dirty" marker for a working-tree op
	// so a dirty op is visibly not a pinned one. The content digest is filled in by
	// the orchestrator once the tree is shipped.
	opRevision := revision
	if in.WorkingTree() {
		opRevision = workingTreeRevision(revision)
	}

	// Dry-run: the request validated and would be dispatched, but we create no
	// op and touch no host. The transient password is not needed — zero it.
	if in.DryRun {
		zeroBytes(in.Password)
		return Decision{DryRun: true, Host: host, Port: port, User: user}, nil
	}

	op, err := s.repo.Create(ctx, Op{
		Host:             host,
		Port:             port,
		User:             user,
		NodeName:         trimField(in.NodeName),
		TargetRevision:   opRevision,
		RepoURL:          trimField(in.RepoURL),
		State:            StatePending,
		SourceMode:       in.SourceMode,
		BaseRevision:     in.BaseRevision,
		ControlPlaneURL:  cpURL,
		ReachabilityMode: string(mode),
		CorrelationID:    trimField(in.EnrollmentCorrelationID),
	})
	if err != nil {
		zeroBytes(in.Password)
		return Decision{}, err
	}

	// Launch the orchestration on a DETACHED context (not the request ctx) so it
	// survives the client disconnecting right after Start returns. The op-scoped
	// context is cancelable for CancelOnboarding.
	runCtx, cancel := context.WithCancel(context.Background())
	s.coord.registerOp(op.ID, cancel)
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.runOnboarding(runCtx, op.ID, in)
	}()

	return Decision{OpID: op.ID, Host: host, Port: port, User: user}, nil
}

func (s *service) StartMachineEnrollment(ctx context.Context, machineID string, in StartInput) (MachineEnrollmentDecision, error) {
	return s.startMachineEnrollment(ctx, machineID, "", in)
}

func (s *service) RetryMachineEnrollment(ctx context.Context, machineID, priorAttemptID string, in StartInput) (MachineEnrollmentDecision, error) {
	store, ok := s.repo.(AttemptStore)
	if !ok {
		zeroBytes(in.Password)
		return MachineEnrollmentDecision{}, fmt.Errorf("machine enrollment requires attempt-capable repository")
	}
	prior, err := store.GetAttempt(ctx, priorAttemptID)
	if err != nil {
		zeroBytes(in.Password)
		return MachineEnrollmentDecision{}, err
	}
	if prior.MachineID != machineID {
		zeroBytes(in.Password)
		return MachineEnrollmentDecision{}, ErrInvalid{Field: "prior_attempt_id", Reason: "does not belong to machine"}
	}
	if !prior.State.Terminal() {
		zeroBytes(in.Password)
		return MachineEnrollmentDecision{}, ErrInvalid{Field: "prior_attempt_id", Reason: "must be terminal before retry"}
	}
	return s.startMachineEnrollment(ctx, machineID, prior.ID, in)
}

func (s *service) startMachineEnrollment(ctx context.Context, machineID, retryOfAttemptID string, in StartInput) (MachineEnrollmentDecision, error) {
	store, ok := s.repo.(AttemptStore)
	if !ok {
		zeroBytes(in.Password)
		return MachineEnrollmentDecision{}, fmt.Errorf("machine enrollment requires attempt-capable repository")
	}
	attempt, err := NewAttempt(machineID, map[string]string{
		"host": in.Host, "port": fmt.Sprintf("%d", in.Port), "user": in.User,
		"node_name": in.NodeName, "target_revision": in.TargetRevision,
		"control_plane_url": in.ControlPlaneURL,
	})
	if err != nil {
		zeroBytes(in.Password)
		return MachineEnrollmentDecision{}, err
	}
	attempt.RetryOfAttemptID = retryOfAttemptID
	attempt, err = store.CreateAttempt(ctx, attempt)
	if err != nil {
		zeroBytes(in.Password)
		return MachineEnrollmentDecision{}, err
	}
	in.EnrollmentCorrelationID = attempt.CorrelationID
	in.SSHKeyName = "machine-" + machineID
	decision, err := s.Start(ctx, in)
	if err != nil {
		_, _ = store.CompleteAttempt(ctx, attempt.ID, AttemptFailed, "start_rejected", err.Error())
		return MachineEnrollmentDecision{Attempt: attempt}, err
	}
	return MachineEnrollmentDecision{Attempt: attempt, Decision: decision}, nil
}

func (s *service) GetOp(ctx context.Context, id string) (Op, []StepEvent, error) {
	op, err := s.repo.Get(ctx, id)
	if err != nil {
		return Op{}, nil, err
	}
	events, err := s.repo.ListEvents(ctx, id)
	if err != nil {
		return Op{}, nil, err
	}
	return op, events, nil
}

func (s *service) ListOps(ctx context.Context, filter ListFilter) ([]Op, error) {
	return s.repo.List(ctx, filter)
}

func (s *service) Wait(ctx context.Context, id string, timeout time.Duration) (Op, bool, error) {
	if timeout <= 0 {
		timeout = DefaultWaitTimeout
	}

	// Register the waiter BEFORE the terminal recheck so a terminal transition
	// racing this call cannot be missed.
	wait, cancel := s.coord.registerWaiter(id)
	defer cancel()

	op, err := s.repo.Get(ctx, id)
	if err != nil {
		return Op{}, false, err
	}
	if op.State.Terminal() {
		return op, false, nil
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return Op{}, false, ctx.Err()
	case <-timer.C:
		op, err = s.repo.Get(ctx, id)
		if err != nil {
			return Op{}, false, err
		}
		return op, !op.State.Terminal(), nil
	case <-wait:
		op, err = s.repo.Get(ctx, id)
		if err != nil {
			return Op{}, false, err
		}
		return op, false, nil
	}
}

func (s *service) Cancel(ctx context.Context, id string) (Op, error) {
	op, err := s.repo.Get(ctx, id)
	if err != nil {
		return Op{}, err
	}
	if op.State.Terminal() {
		return op, nil
	}
	// Signal the orchestration goroutine; it observes ctx cancellation at the
	// next phase boundary and drives the op to CANCELLED (single-writer, so no
	// racing terminal transition here). The caller then blocks on Wait for the
	// terminal CANCELLED state.
	s.coord.cancelOp(id)
	return op, nil
}

func (s *service) RemoveFailed(ctx context.Context, id string) error {
	if trimField(id) == "" {
		return ErrInvalid{Field: "id", Reason: "required"}
	}
	return s.repo.DeleteFailed(ctx, id)
}

func (s *service) Subscribe(id string) (<-chan StepEvent, func()) {
	return s.coord.subscribe(id)
}

func (s *service) ResumeInterrupted(ctx context.Context) (int, error) {
	ops, err := s.repo.ListNonTerminal(ctx)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, op := range ops {
		now := s.clock.Now().UTC()
		if op.StartedAt.IsZero() {
			op.StartedAt = now
		}
		op.State = StateFailed
		op.FailureReason = FailureInterrupted
		op.FinishedAt = now
		if _, uerr := s.repo.Update(ctx, op); uerr != nil {
			return n, uerr
		}
		s.appendEvent(ctx, op.ID, StepRun, StepStatusFailed, "control plane restarted during onboarding; op is safe to retry (idempotent)")
		s.coord.signalTerminal(op.ID)
		n++
	}
	return n, nil
}

// WaitIdle blocks until every in-flight orchestration goroutine has finished.
// Test-only convenience; production never needs it.
func (s *service) WaitIdle() { s.wg.Wait() }

// ----------------------------------------------------------------------------
// coordinator — the in-memory block-once waiter registry, live event fan-out,
// and per-op cancellation registry. Mirrors the provision domain's coordinator.
// ----------------------------------------------------------------------------

type coordinator struct {
	mu      sync.Mutex
	waiters map[string]map[chan struct{}]struct{}
	subs    map[string]map[chan StepEvent]struct{}
	cancels map[string]context.CancelFunc
}

func newCoordinator() *coordinator {
	return &coordinator{
		waiters: make(map[string]map[chan struct{}]struct{}),
		subs:    make(map[string]map[chan StepEvent]struct{}),
		cancels: make(map[string]context.CancelFunc),
	}
}

func (c *coordinator) registerOp(id string, cancel context.CancelFunc) {
	c.mu.Lock()
	c.cancels[id] = cancel
	c.mu.Unlock()
}

func (c *coordinator) cancelOp(id string) {
	c.mu.Lock()
	cancel := c.cancels[id]
	c.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (c *coordinator) releaseOp(id string) {
	c.mu.Lock()
	if cancel := c.cancels[id]; cancel != nil {
		cancel()
	}
	delete(c.cancels, id)
	c.mu.Unlock()
}

func (c *coordinator) registerWaiter(id string) (<-chan struct{}, func()) {
	ch := make(chan struct{})
	c.mu.Lock()
	set := c.waiters[id]
	if set == nil {
		set = make(map[chan struct{}]struct{})
		c.waiters[id] = set
	}
	set[ch] = struct{}{}
	c.mu.Unlock()

	var once sync.Once
	cancel := func() {
		once.Do(func() {
			c.mu.Lock()
			if s := c.waiters[id]; s != nil {
				delete(s, ch)
				if len(s) == 0 {
					delete(c.waiters, id)
				}
			}
			c.mu.Unlock()
		})
	}
	return ch, cancel
}

func (c *coordinator) signalTerminal(id string) {
	c.mu.Lock()
	set := c.waiters[id]
	delete(c.waiters, id)
	c.mu.Unlock()
	for ch := range set {
		close(ch)
	}
}

func (c *coordinator) subscribe(id string) (<-chan StepEvent, func()) {
	ch := make(chan StepEvent, subscriberBuffer)
	c.mu.Lock()
	set := c.subs[id]
	if set == nil {
		set = make(map[chan StepEvent]struct{})
		c.subs[id] = set
	}
	set[ch] = struct{}{}
	c.mu.Unlock()

	var once sync.Once
	cancel := func() {
		once.Do(func() {
			c.mu.Lock()
			if s := c.subs[id]; s != nil {
				delete(s, ch)
				if len(s) == 0 {
					delete(c.subs, id)
				}
			}
			c.mu.Unlock()
		})
	}
	return ch, cancel
}

func (c *coordinator) publish(ev StepEvent) {
	c.mu.Lock()
	set := c.subs[ev.OpID]
	chans := make([]chan StepEvent, 0, len(set))
	for ch := range set {
		chans = append(chans, ch)
	}
	c.mu.Unlock()
	for _, ch := range chans {
		select {
		case ch <- ev:
		default:
		}
	}
}

// zeroBytes overwrites b with zeros (best-effort credential wipe).
func zeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
