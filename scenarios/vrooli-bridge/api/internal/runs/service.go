package runs

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"vrooli-bridge/internal/clock"
)

// DefaultWaitTimeout bounds a WaitRun call that passes timeout_seconds <= 0, so
// a block-once wait can never hang forever even if the node vanishes without a
// terminal event. The CLI surfaces a timed-out wait as exit 124 (like
// test-genie), and the operator re-issues the wait.
const DefaultWaitTimeout = 30 * time.Minute

// subscriberBuffer bounds a live event subscriber's channel. A slow streaming
// client that fills its buffer drops the overflow rather than blocking the
// ingest path; it can always GetRun to read the full persisted history.
const subscriberBuffer = 256

// Service is the application-layer surface the runs handlers depend on. It owns
// the durable lifecycle (create/get/list/abort), the node-event ingest that
// drives status transitions, and the in-memory coordination durability needs:
// block-once Wait and live Subscribe fan-out.
type Service interface {
	// Create persists a new QUEUED run for an already-validated job.
	Create(ctx context.Context, in CreateInput) (Run, error)

	// Get returns a run and its full persisted event history.
	Get(ctx context.Context, id string) (Run, []RunEvent, error)

	// List returns runs newest-first, narrowed by filter.
	List(ctx context.Context, filter ListFilter) ([]Run, error)

	// AppendEvent ingests one RunEvent streamed from the node-agent. It persists
	// the event, drives the run's status transition (running on the first STATUS,
	// terminal on EXIT), wakes block-once waiters on a terminal transition, and
	// fans the event out to live subscribers. accepted is false (without error)
	// for an event targeting an unknown or already-terminal run, so a re-sending
	// node never spins.
	AppendEvent(ctx context.Context, ev RunEvent) (accepted bool, err error)

	// Wait blocks until the run reaches a terminal status or the timeout
	// elapses, returning exactly once. timedOut is true when the deadline
	// elapsed first (the returned Run is the latest non-terminal snapshot).
	Wait(ctx context.Context, id string, timeout time.Duration) (run Run, timedOut bool, err error)

	// Abort marks a non-terminal run ABORTED, wakes waiters, and returns the
	// run. Idempotent on an already-terminal run (returns it unchanged).
	Abort(ctx context.Context, id, reason string) (Run, error)

	// Subscribe registers a live event subscriber for the run, returning a
	// channel that receives subsequent AppendEvent events and an unsubscribe
	// func the caller MUST invoke. Use with Get to replay history then tail.
	Subscribe(id string) (<-chan RunEvent, func())
}

type service struct {
	repo  Repository
	clock clock.Clock
	coord *coordinator
}

// NewService constructs the production Service with an empty coordinator. A
// single instance is shared between the runs handler (operator + node ingest)
// and the dispatch handler (Create) so the in-memory waiter/subscriber state is
// coherent across both call sites.
func NewService(repo Repository, clk clock.Clock) Service {
	return &service{repo: repo, clock: clk, coord: newCoordinator()}
}

// Compile-time guarantee.
var _ Service = (*service)(nil)

func (s *service) Create(ctx context.Context, in CreateInput) (Run, error) {
	nodeID := strings.TrimSpace(in.NodeID)
	if nodeID == "" {
		return Run{}, ErrInvalidRun{Field: "node_id", Reason: "required"}
	}
	verb := strings.TrimSpace(in.Verb)
	if verb == "" {
		return Run{}, ErrInvalidRun{Field: "verb", Reason: "required"}
	}
	return s.repo.Create(ctx, Run{
		NodeID:         nodeID,
		Scenario:       strings.TrimSpace(in.Scenario),
		Verb:           verb,
		Args:           trimArgs(in.Args),
		Status:         StatusQueued,
		TimeoutSeconds: in.TimeoutSeconds,
	})
}

func (s *service) Get(ctx context.Context, id string) (Run, []RunEvent, error) {
	run, err := s.repo.Get(ctx, id)
	if err != nil {
		return Run{}, nil, err
	}
	events, err := s.repo.ListEvents(ctx, id)
	if err != nil {
		return Run{}, nil, err
	}
	return run, events, nil
}

func (s *service) List(ctx context.Context, filter ListFilter) ([]Run, error) {
	return s.repo.List(ctx, filter)
}

func (s *service) AppendEvent(ctx context.Context, ev RunEvent) (bool, error) {
	ev.RunID = strings.TrimSpace(ev.RunID)
	if ev.RunID == "" {
		return false, ErrInvalidRun{Field: "run_id", Reason: "required"}
	}

	run, err := s.repo.Get(ctx, ev.RunID)
	if err != nil {
		var notFound ErrRunNotFound
		if errors.As(err, &notFound) {
			// Unknown run: acknowledge without error so a confused node stops.
			return false, nil
		}
		return false, err
	}
	if run.Status.Terminal() {
		// Stale completion: the run already reached a terminal status (e.g. an
		// operator abort or a prior EXIT). Ignore the late event but persist it
		// for the audit/log trail, then acknowledge no further effect.
		_ = s.repo.AppendEvent(ctx, ev)
		return false, nil
	}

	if ev.EmittedAt.IsZero() {
		ev.EmittedAt = s.clock.Now().UTC()
	}
	if err := s.repo.AppendEvent(ctx, ev); err != nil {
		return false, err
	}

	// Drive the lifecycle. The control plane stamps started/finished with its
	// own clock rather than trusting the node's, so ordering is authoritative.
	now := s.clock.Now().UTC()
	changed := false
	switch ev.Kind {
	case EventStatus:
		if run.Status == StatusQueued && strings.EqualFold(strings.TrimSpace(ev.Status), "running") {
			run.Status = StatusRunning
			run.StartedAt = now
			changed = true
		}
	case EventExit:
		if run.StartedAt.IsZero() {
			run.StartedAt = now
		}
		run.FinishedAt = now
		run.ExitCode = ev.ExitCode
		if ev.ExitCode == 0 {
			run.Status = StatusPassed
		} else {
			run.Status = StatusFailed
		}
		changed = true
	case EventArtifactRef:
		if ref := strings.TrimSpace(ev.ArtifactRef); ref != "" {
			run.ArtifactRefs = append(run.ArtifactRefs, ref)
			changed = true
		}
	case EventLog, EventUnspecified:
		// No lifecycle effect.
	}

	if changed {
		if run, err = s.repo.Update(ctx, run); err != nil {
			return false, err
		}
	}

	// Fan out to live subscribers first so a follower sees the event, then wake
	// block-once waiters if the run is now terminal.
	s.coord.publish(ev)
	if run.Status.Terminal() {
		s.coord.signalTerminal(run.ID)
	}
	return true, nil
}

func (s *service) Wait(ctx context.Context, id string, timeout time.Duration) (Run, bool, error) {
	if timeout <= 0 {
		timeout = DefaultWaitTimeout
	}

	// Register the waiter BEFORE the terminal recheck so a terminal transition
	// racing this call cannot be missed (the signal would find a registered
	// waiter; the recheck catches an already-terminal run).
	wait, cancel := s.coord.registerWaiter(id)
	defer cancel()

	run, err := s.repo.Get(ctx, id)
	if err != nil {
		return Run{}, false, err
	}
	if run.Status.Terminal() {
		return run, false, nil
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return Run{}, false, ctx.Err()
	case <-timer.C:
		run, err = s.repo.Get(ctx, id)
		if err != nil {
			return Run{}, false, err
		}
		return run, !run.Status.Terminal(), nil
	case <-wait:
		run, err = s.repo.Get(ctx, id)
		if err != nil {
			return Run{}, false, err
		}
		return run, false, nil
	}
}

func (s *service) Abort(ctx context.Context, id, reason string) (Run, error) {
	run, err := s.repo.Get(ctx, id)
	if err != nil {
		return Run{}, err
	}
	if run.Status.Terminal() {
		return run, nil
	}
	now := s.clock.Now().UTC()
	run.Status = StatusAborted
	run.FinishedAt = now
	if run.StartedAt.IsZero() {
		run.StartedAt = now
	}
	run, err = s.repo.Update(ctx, run)
	if err != nil {
		return Run{}, err
	}

	// Record the abort as a terminal status event so GetRun/follow show why the
	// run ended, then wake waiters.
	label := "aborted"
	if r := strings.TrimSpace(reason); r != "" {
		label = "aborted: " + r
	}
	ev := RunEvent{RunID: id, Kind: EventStatus, Sequence: s.coord.nextAbortSeq(id), Status: label, EmittedAt: now}
	_ = s.repo.AppendEvent(ctx, ev)
	s.coord.publish(ev)
	s.coord.signalTerminal(id)
	return run, nil
}

func (s *service) Subscribe(id string) (<-chan RunEvent, func()) {
	return s.coord.subscribe(id)
}

// trimArgs trims each arg and drops empties.
func trimArgs(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	for _, v := range in {
		if t := strings.TrimSpace(v); t != "" {
			out = append(out, t)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// ----------------------------------------------------------------------------
// coordinator — the in-memory block-once waiter registry + live event fan-out.
// ----------------------------------------------------------------------------

type coordinator struct {
	mu       sync.Mutex
	waiters  map[string]map[chan struct{}]struct{}
	subs     map[string]map[chan RunEvent]struct{}
	abortSeq map[string]uint64
}

func newCoordinator() *coordinator {
	return &coordinator{
		waiters:  make(map[string]map[chan struct{}]struct{}),
		subs:     make(map[string]map[chan RunEvent]struct{}),
		abortSeq: make(map[string]uint64),
	}
}

// registerWaiter adds a waiter for id and returns its signal channel plus an
// unregister func. The channel is closed (broadcast) when the run goes terminal.
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

// signalTerminal closes every registered waiter channel for id (a broadcast)
// and clears them. A waiter that already cancelled is simply absent.
func (c *coordinator) signalTerminal(id string) {
	c.mu.Lock()
	set := c.waiters[id]
	delete(c.waiters, id)
	c.mu.Unlock()
	for ch := range set {
		close(ch)
	}
}

// subscribe registers a live subscriber for id and returns its channel plus an
// unsubscribe func the caller MUST invoke.
func (c *coordinator) subscribe(id string) (<-chan RunEvent, func()) {
	ch := make(chan RunEvent, subscriberBuffer)
	c.mu.Lock()
	set := c.subs[id]
	if set == nil {
		set = make(map[chan RunEvent]struct{})
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

// publish delivers ev to every live subscriber for its run, non-blocking: a
// full subscriber buffer drops the event rather than stalling ingest.
func (c *coordinator) publish(ev RunEvent) {
	c.mu.Lock()
	set := c.subs[ev.RunID]
	chans := make([]chan RunEvent, 0, len(set))
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

// nextAbortSeq returns a per-run sequence for a control-plane-synthesised event
// (abort), kept well above any node sequence space by offsetting high. It only
// needs to be unique per run for the (run_id, sequence) primary key.
func (c *coordinator) nextAbortSeq(id string) uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	const abortSeqBase = 1 << 62
	if c.abortSeq[id] == 0 {
		c.abortSeq[id] = abortSeqBase
	}
	c.abortSeq[id]++
	return c.abortSeq[id]
}
