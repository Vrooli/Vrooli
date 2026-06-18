package provision

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"vrooli-bridge/internal/clock"
)

// DefaultTimeoutSeconds is applied when a Sync passes timeout_seconds <= 0, so a
// provisioning op always carries a finite wall-clock budget the node aborts on.
// Provisioning (git fetch + full `vrooli setup`) is slower than a job, so the
// default is generous.
const DefaultTimeoutSeconds int64 = 3600 // 1 hour

// DefaultWaitTimeout bounds a Wait call that passes timeout_seconds <= 0, so a
// block-once wait can never hang forever even if the node vanishes mid-provision.
const DefaultWaitTimeout = 60 * time.Minute

// subscriberBuffer bounds a live event subscriber's channel. A slow streaming
// client that fills its buffer drops the overflow rather than blocking ingest.
const subscriberBuffer = 256

// Service is the application-layer surface the provision handler depends on. It
// owns the SyncToRevision orchestration (the privileged-tier safety sequence)
// AND the durable op lifecycle + node-event ingest + the in-memory coordination
// durability needs (block-once Wait, live Subscribe fan-out).
type Service interface {
	// Sync validates the node + revision, resolves the rollback target, creates
	// a durable op, audits the request, and pushes the privileged
	// ProvisionCommand to the node. On a dry-run it validates and short-circuits
	// before any side effect. A rejection is audited before the error returns.
	Sync(ctx context.Context, in SyncInput) (Decision, error)

	// GetOp returns an op and its full persisted event history.
	GetOp(ctx context.Context, id string) (ProvisioningOp, []ProvisionEvent, error)

	// ListOps returns ops newest-first, narrowed by filter.
	ListOps(ctx context.Context, filter ListFilter) ([]ProvisioningOp, error)

	// AppendEvent ingests one ProvisionEvent streamed from the node's privileged
	// helper. It persists the event, drives the op's status transition (running
	// on the first STATUS, version-recording on VERSION, terminal on EXIT),
	// wakes block-once waiters on a terminal transition, and fans the event out
	// to live subscribers. accepted is false (without error) for an event
	// targeting an unknown or already-terminal op.
	AppendEvent(ctx context.Context, ev ProvisionEvent) (accepted bool, err error)

	// Wait blocks until the op reaches a terminal status or the timeout elapses,
	// returning exactly once. timedOut is true when the deadline elapsed first.
	Wait(ctx context.Context, id string, timeout time.Duration) (op ProvisioningOp, timedOut bool, err error)

	// Subscribe registers a live event subscriber for the op, returning a
	// channel that receives subsequent AppendEvent events and an unsubscribe
	// func the caller MUST invoke.
	Subscribe(id string) (<-chan ProvisionEvent, func())

	// GetNodeVersion returns a node's current recorded version or
	// ErrNoNodeVersion{nodeID}.
	GetNodeVersion(ctx context.Context, nodeID string) (NodeVersion, error)
}

type service struct {
	repo           Repository
	nodes          NodeReader
	presence       Presence
	audit          AuditSink
	pusher         CommandPusher
	clock          clock.Clock
	coord          *coordinator
	defaultTimeout int64
}

// Option customises the service (default timeout).
type Option func(*service)

// WithDefaultTimeout overrides the timeout applied when a Sync passes <= 0.
func WithDefaultTimeout(seconds int64) Option {
	return func(s *service) {
		if seconds > 0 {
			s.defaultTimeout = seconds
		}
	}
}

// NewService constructs the production Service. A single instance is shared
// between the provision handler (operator verbs + node ingest); the in-memory
// waiter/subscriber state therefore stays coherent across both call sites.
func NewService(repo Repository, nodes NodeReader, presence Presence, sink AuditSink, pusher CommandPusher, clk clock.Clock, opts ...Option) Service {
	s := &service{
		repo:           repo,
		nodes:          nodes,
		presence:       presence,
		audit:          sink,
		pusher:         pusher,
		clock:          clk,
		coord:          newCoordinator(),
		defaultTimeout: DefaultTimeoutSeconds,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Compile-time guarantee.
var _ Service = (*service)(nil)

func (s *service) Sync(ctx context.Context, in SyncInput) (Decision, error) {
	nodeID := strings.TrimSpace(in.NodeID)
	target := trimRevision(in.TargetRevision)
	if nodeID == "" {
		return Decision{}, ErrInvalidOp{Field: "node_id", Reason: "required"}
	}
	if target == "" {
		return Decision{}, ErrInvalidOp{Field: "target_revision", Reason: "required"}
	}

	// 1. Resolve the target node.
	node, err := s.nodes.GetTarget(ctx, nodeID)
	if err != nil {
		var notFound ErrNodeNotFound
		if errors.As(err, &notFound) {
			return Decision{}, notFound
		}
		return Decision{}, err
	}

	// 2. A revoked node can be provisioned no further.
	if node.Revoked {
		s.auditReject(ctx, in, "node revoked")
		return Decision{}, ErrNodeRevoked{ID: nodeID}
	}

	// 3. Resolve the rollback revision: an explicit one wins; otherwise the
	//    node's last recorded version, so a failed setup returns the node to
	//    where it was. A never-provisioned node has no rollback target (first
	//    provision).
	rollback := trimRevision(in.RollbackRevision)
	if rollback == "" {
		if v, verr := s.repo.GetNodeVersion(ctx, nodeID); verr == nil {
			rollback = v.Revision
		}
	}

	// 4. Dry-run: the request validated and would be dispatched, but we create
	//    no op, write no audit, and push nothing.
	if in.DryRun {
		return Decision{DryRun: true, NodeID: nodeID, TargetRevision: target, RollbackRevision: rollback}, nil
	}

	// 5. The node must currently hold a channel to receive the privileged push.
	if !s.presence.IsOnline(nodeID) {
		s.auditReject(ctx, in, "node offline")
		return Decision{}, ErrNodeOffline{ID: nodeID}
	}

	// 6. Create the durable op.
	timeout := in.TimeoutSeconds
	if timeout <= 0 {
		timeout = s.defaultTimeout
	}
	op, err := s.repo.Create(ctx, ProvisioningOp{
		NodeID:           nodeID,
		TargetRevision:   target,
		RollbackRevision: rollback,
		Status:           StatusQueued,
		TimeoutSeconds:   timeout,
	})
	if err != nil {
		return Decision{}, err
	}

	// 7. Audit the accepted op. FAIL-CLOSED: privileged remote provisioning must
	//    be accountable, so if we cannot record it we mark the op failed and
	//    refuse rather than provision un-audited.
	if aerr := s.audit.Record(ctx, Entry{
		Actor: in.Actor, NodeID: nodeID, TargetRevision: target, RollbackRevision: rollback,
		Accepted: true, Detail: "provisioning dispatched", OpID: op.ID,
	}); aerr != nil {
		_ = s.markFailed(ctx, op.ID, "audit write failed")
		return Decision{}, aerr
	}

	// 8. Push the privileged ProvisionCommand to the node. If it does not land,
	//    mark the op failed and fail the request; Phase 5 adds a durable queue
	//    that redelivers instead.
	delivered, perr := s.pusher.PushProvision(ctx, nodeID, PushedCommand{
		OpID: op.ID, TargetRevision: target, RollbackRevision: rollback,
	})
	if perr != nil || delivered == 0 {
		_ = s.markFailed(ctx, op.ID, "provisioning command delivery failed")
		if perr != nil {
			return Decision{}, perr
		}
		return Decision{}, ErrDeliveryFailed{NodeID: nodeID}
	}

	return Decision{OpID: op.ID, NodeID: nodeID, TargetRevision: target, RollbackRevision: rollback}, nil
}

func (s *service) GetOp(ctx context.Context, id string) (ProvisioningOp, []ProvisionEvent, error) {
	op, err := s.repo.Get(ctx, id)
	if err != nil {
		return ProvisioningOp{}, nil, err
	}
	events, err := s.repo.ListEvents(ctx, id)
	if err != nil {
		return ProvisioningOp{}, nil, err
	}
	return op, events, nil
}

func (s *service) ListOps(ctx context.Context, filter ListFilter) ([]ProvisioningOp, error) {
	return s.repo.List(ctx, filter)
}

func (s *service) GetNodeVersion(ctx context.Context, nodeID string) (NodeVersion, error) {
	return s.repo.GetNodeVersion(ctx, strings.TrimSpace(nodeID))
}

func (s *service) AppendEvent(ctx context.Context, ev ProvisionEvent) (bool, error) {
	ev.OpID = strings.TrimSpace(ev.OpID)
	if ev.OpID == "" {
		return false, ErrInvalidOp{Field: "op_id", Reason: "required"}
	}

	op, err := s.repo.Get(ctx, ev.OpID)
	if err != nil {
		var notFound ErrOpNotFound
		if errors.As(err, &notFound) {
			// Unknown op: acknowledge without error so a confused node stops.
			return false, nil
		}
		return false, err
	}
	if op.Status.Terminal() {
		// Stale completion: persist the late event for the trail, no further
		// effect.
		_ = s.repo.AppendEvent(ctx, ev)
		return false, nil
	}

	if ev.EmittedAt.IsZero() {
		ev.EmittedAt = s.clock.Now().UTC()
	}
	if err := s.repo.AppendEvent(ctx, ev); err != nil {
		return false, err
	}

	now := s.clock.Now().UTC()
	changed := false
	switch ev.Kind {
	case EventStatus:
		if op.Status == StatusQueued {
			op.Status = StatusRunning
			op.StartedAt = now
			changed = true
		}
	case EventVersion:
		if rev := trimRevision(ev.Revision); rev != "" {
			op.ResultingRevision = rev
			changed = true
			// Record the node's current version immediately (a successful op
			// reports target; a rollback reports the prior revision).
			_ = s.repo.UpsertNodeVersion(ctx, NodeVersion{
				NodeID: op.NodeID, Revision: rev, OpID: op.ID, ReportedAt: now,
			})
		}
	case EventExit:
		if op.StartedAt.IsZero() {
			op.StartedAt = now
		}
		op.FinishedAt = now
		op.ExitCode = ev.ExitCode
		op.Status = terminalStatus(op, ev.ExitCode)
		changed = true
	case EventLog, EventUnspecified:
		// No lifecycle effect.
	}

	if changed {
		if op, err = s.repo.Update(ctx, op); err != nil {
			return false, err
		}
	}

	// Fan out to live subscribers first so a follower sees the event, then wake
	// block-once waiters if the op is now terminal.
	s.coord.publish(ev)
	if op.Status.Terminal() {
		s.coord.signalTerminal(op.ID)
	}
	return true, nil
}

// terminalStatus computes an op's terminal disposition from its exit code and
// the resulting revision the node reported (the VERSION event). A clean exit is
// COMPLETED; a failing exit that landed back on the rollback revision is the
// SAFE failure (ROLLED_BACK); anything else FAILED (degraded).
func terminalStatus(op ProvisioningOp, exitCode int32) ProvisioningStatus {
	if exitCode == 0 {
		return StatusCompleted
	}
	if rb := trimRevision(op.RollbackRevision); rb != "" && trimRevision(op.ResultingRevision) == rb && rb != trimRevision(op.TargetRevision) {
		return StatusRolledBack
	}
	return StatusFailed
}

func (s *service) Wait(ctx context.Context, id string, timeout time.Duration) (ProvisioningOp, bool, error) {
	if timeout <= 0 {
		timeout = DefaultWaitTimeout
	}

	// Register the waiter BEFORE the terminal recheck so a terminal transition
	// racing this call cannot be missed.
	wait, cancel := s.coord.registerWaiter(id)
	defer cancel()

	op, err := s.repo.Get(ctx, id)
	if err != nil {
		return ProvisioningOp{}, false, err
	}
	if op.Status.Terminal() {
		return op, false, nil
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ProvisioningOp{}, false, ctx.Err()
	case <-timer.C:
		op, err = s.repo.Get(ctx, id)
		if err != nil {
			return ProvisioningOp{}, false, err
		}
		return op, !op.Status.Terminal(), nil
	case <-wait:
		op, err = s.repo.Get(ctx, id)
		if err != nil {
			return ProvisioningOp{}, false, err
		}
		return op, false, nil
	}
}

func (s *service) Subscribe(id string) (<-chan ProvisionEvent, func()) {
	return s.coord.subscribe(id)
}

// markFailed transitions an op to FAILED with a recorded status event. Used on a
// control-plane-side failure (audit write / delivery) before the node ever runs.
func (s *service) markFailed(ctx context.Context, id, reason string) error {
	op, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if op.Status.Terminal() {
		return nil
	}
	now := s.clock.Now().UTC()
	op.Status = StatusFailed
	op.FinishedAt = now
	if op.StartedAt.IsZero() {
		op.StartedAt = now
	}
	if _, err := s.repo.Update(ctx, op); err != nil {
		return err
	}
	label := "failed"
	if r := strings.TrimSpace(reason); r != "" {
		label = "failed: " + r
	}
	ev := ProvisionEvent{OpID: id, Kind: EventStatus, Sequence: s.coord.nextSyntheticSeq(id), Status: label, EmittedAt: now}
	_ = s.repo.AppendEvent(ctx, ev)
	s.coord.publish(ev)
	s.coord.signalTerminal(id)
	return nil
}

// auditReject records a denied provisioning request. Best-effort: the rejection
// stands even if the audit write fails, but the record is the default.
func (s *service) auditReject(ctx context.Context, in SyncInput, reason string) {
	_ = s.audit.Record(ctx, Entry{
		Actor: in.Actor, NodeID: strings.TrimSpace(in.NodeID),
		TargetRevision: trimRevision(in.TargetRevision), RollbackRevision: trimRevision(in.RollbackRevision),
		Accepted: false, Detail: reason,
	})
}

// ----------------------------------------------------------------------------
// coordinator — the in-memory block-once waiter registry + live event fan-out.
// Mirrors the runs domain's coordinator (same durability discipline).
// ----------------------------------------------------------------------------

type coordinator struct {
	mu       sync.Mutex
	waiters  map[string]map[chan struct{}]struct{}
	subs     map[string]map[chan ProvisionEvent]struct{}
	synthSeq map[string]uint64
}

func newCoordinator() *coordinator {
	return &coordinator{
		waiters:  make(map[string]map[chan struct{}]struct{}),
		subs:     make(map[string]map[chan ProvisionEvent]struct{}),
		synthSeq: make(map[string]uint64),
	}
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

func (c *coordinator) subscribe(id string) (<-chan ProvisionEvent, func()) {
	ch := make(chan ProvisionEvent, subscriberBuffer)
	c.mu.Lock()
	set := c.subs[id]
	if set == nil {
		set = make(map[chan ProvisionEvent]struct{})
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

func (c *coordinator) publish(ev ProvisionEvent) {
	c.mu.Lock()
	set := c.subs[ev.OpID]
	chans := make([]chan ProvisionEvent, 0, len(set))
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

// nextSyntheticSeq returns a per-op sequence for a control-plane-synthesised
// event (a markFailed status), kept well above any node sequence space by
// offsetting high. It only needs to be unique per op for the primary key.
func (c *coordinator) nextSyntheticSeq(id string) uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	const base = 1 << 62
	if c.synthSeq[id] == 0 {
		c.synthSeq[id] = base
	}
	c.synthSeq[id]++
	return c.synthSeq[id]
}
