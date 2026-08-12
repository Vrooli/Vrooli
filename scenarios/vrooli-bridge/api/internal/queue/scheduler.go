package queue

import (
	"context"
	"sort"
	"sync"
	"time"

	"vrooli-bridge/internal/clock"
)

// Scheduler is the in-memory per-node job scheduler. It is safe for concurrent
// use and constructed once in main.go, shared between the dispatch handler
// (Submit, via dispatch's job-push seam) and the runs domain (Complete, via the
// run-terminal hook). The QueueService reads its Snapshot.
type Scheduler struct {
	pusher        Pusher
	aborter       Aborter
	clock         clock.Clock
	limit         int
	store         DurableStore
	deliveryLease time.Duration

	mu        sync.Mutex
	running   map[string]map[string]Entry // nodeID -> runID -> running entry
	queued    map[string][]Entry          // nodeID -> FIFO queue of queued entries
	promoting map[string]bool             // nodeID -> a Complete drain is in progress
}

type Option func(*Scheduler)

func WithDurableStore(store DurableStore) Option {
	return func(s *Scheduler) { s.store = store }
}

func WithDeliveryLease(lease time.Duration) Option {
	return func(s *Scheduler) {
		if lease > 0 {
			s.deliveryLease = lease
		}
	}
}

// NewScheduler constructs the scheduler with a per-node concurrency limit
// (limit <= 0 uses DefaultConcurrencyLimit).
func NewScheduler(pusher Pusher, aborter Aborter, clk clock.Clock, limit int, opts ...Option) *Scheduler {
	s := newScheduler(pusher, aborter, clk, limit, opts...)
	_ = s.load(context.Background())
	return s
}

// NewSchedulerWithStore is the boot path. It returns a migration/load error so
// the control plane cannot accept traffic with an incomplete queue projection.
func NewSchedulerWithStore(pusher Pusher, aborter Aborter, clk clock.Clock, limit int, store DurableStore) (*Scheduler, error) {
	s := newScheduler(pusher, aborter, clk, limit, WithDurableStore(store))
	if err := s.load(context.Background()); err != nil {
		return nil, err
	}
	return s, nil
}

func newScheduler(pusher Pusher, aborter Aborter, clk clock.Clock, limit int, opts ...Option) *Scheduler {
	if limit <= 0 {
		limit = DefaultConcurrencyLimit
	}
	s := &Scheduler{
		pusher:        pusher,
		aborter:       aborter,
		clock:         clk,
		limit:         limit,
		running:       make(map[string]map[string]Entry),
		queued:        make(map[string][]Entry),
		promoting:     make(map[string]bool),
		deliveryLease: 10 * time.Second,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *Scheduler) load(ctx context.Context) error {
	if s.store == nil {
		return nil
	}
	entries, err := s.store.Load(ctx)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, entry := range entries {
		if entry.State == StateQueued {
			s.queued[entry.Job.NodeID] = append(s.queued[entry.Job.NodeID], Entry{
				Job: entry.Job, State: StateQueued, EnqueuedAt: entry.EnqueuedAt,
			})
			continue
		}
		s.ensureRunning(entry.Job.NodeID)[entry.Job.RunID] = Entry{
			Job: entry.Job, State: StateRunning, Position: -1,
			EnqueuedAt: entry.EnqueuedAt, StartedAt: entry.StartedAt,
		}
	}
	return nil
}

// Submit schedules a job for its node. If the node has a free running slot the
// job is pushed immediately (Outcome Pushed) and counts toward the bound; if the
// node is at its bound the job is held queued (Outcome Queued). The return value
// also satisfies dispatch's job-push seam (delivered, err): a queued job reports
// delivered=1 (accepted), a pushed job reports the live-connection count, and a
// push that did not land reports delivered=0 so the caller aborts the run. A
// genuine pusher error is returned verbatim.
func (s *Scheduler) Submit(ctx context.Context, job Job) (Outcome, int, error) {
	if available, ok := s.pusher.(Availability); ok && !available.IsAvailable(job.NodeID) {
		entry := s.queuedEntry(job)
		s.mu.Lock()
		s.queued[job.NodeID] = append(s.queued[job.NodeID], entry)
		s.mu.Unlock()
		if s.store != nil {
			if err := s.store.MarkQueued(ctx, job.RunID, entry.EnqueuedAt); err != nil {
				s.mu.Lock()
				s.removeLocked(job.NodeID, job.RunID)
				s.mu.Unlock()
				return OutcomeQueued, 0, err
			}
		}
		return OutcomeQueued, 1, nil
	}
	s.mu.Lock()
	if len(s.running[job.NodeID]) < s.limit {
		// A slot is free: optimistically occupy it, then push outside the lock.
		entry := s.runningEntry(job)
		s.ensureRunning(job.NodeID)[job.RunID] = entry
		s.mu.Unlock()

		delivered, err := s.pusher.Push(ctx, job)
		if err != nil || delivered == 0 {
			// The node dropped between the slot check and the push: free the slot
			// so a later promotion can use it, and let the caller abort the run.
			s.mu.Lock()
			s.removeLocked(job.NodeID, job.RunID)
			s.mu.Unlock()
			return OutcomePushed, delivered, err
		}
		if s.store != nil {
			now := s.clock.Now().UTC()
			if err := s.store.MarkPushed(ctx, job.RunID, now, now.Add(s.deliveryLease)); err != nil {
				s.mu.Lock()
				s.removeLocked(job.NodeID, job.RunID)
				s.mu.Unlock()
				return OutcomePushed, delivered, err
			}
		}
		return OutcomePushed, delivered, nil
	}

	// The node is at its bound: queue the job in FIFO order.
	entry := s.queuedEntry(job)
	s.queued[job.NodeID] = append(s.queued[job.NodeID], entry)
	s.mu.Unlock()
	if s.store != nil {
		if err := s.store.MarkQueued(ctx, job.RunID, entry.EnqueuedAt); err != nil {
			s.mu.Lock()
			s.removeLocked(job.NodeID, job.RunID)
			s.mu.Unlock()
			return OutcomeQueued, 0, err
		}
	}
	return OutcomeQueued, 1, nil
}

// Promote pushes queued work for a node after its channel becomes available.
func (s *Scheduler) Promote(ctx context.Context, nodeID string) {
	s.promote(ctx, nodeID)
}

// Complete frees the slot a (now-terminal) run held on its node and promotes the
// next queued job(s) until the bound is met or the queue is empty. It is
// idempotent: calling it for a run the scheduler does not track simply drains
// any promotable jobs. Pushes/aborts happen OUTSIDE the lock; a promotion whose
// push does not land aborts that run and continues to the next queued job. The
// run-terminal hook (runs domain) calls this.
func (s *Scheduler) Complete(ctx context.Context, nodeID, runID string) {
	s.mu.Lock()
	s.removeLocked(nodeID, runID)
	s.mu.Unlock()

	s.promote(ctx, nodeID)
}

// Requeue returns a delivery attempt to FIFO waiting state after its lease
// expires without aborting the durable run.
func (s *Scheduler) Requeue(ctx context.Context, job Job) error {
	s.mu.Lock()
	s.removeLocked(job.NodeID, job.RunID)
	entry := s.queuedEntry(job)
	s.queued[job.NodeID] = append(s.queued[job.NodeID], entry)
	s.mu.Unlock()
	if s.store != nil {
		if err := s.store.MarkQueued(ctx, job.RunID, entry.EnqueuedAt, "delivery_lease_expired"); err != nil {
			return err
		}
	}
	if available, ok := s.pusher.(Availability); ok && available.IsAvailable(job.NodeID) {
		s.Promote(ctx, job.NodeID)
	}
	return nil
}

// Remove drops a run from the live projection without changing its durable
// status. It is used immediately before terminal watchdog transitions.
func (s *Scheduler) Remove(nodeID, runID string) {
	s.mu.Lock()
	s.removeLocked(nodeID, runID)
	s.mu.Unlock()
}

func (s *Scheduler) promote(ctx context.Context, nodeID string) {
	s.mu.Lock()
	if s.promoting[nodeID] {
		s.mu.Unlock()
		return
	}
	s.promoting[nodeID] = true
	s.mu.Unlock()

	var toAbort []string
	for {
		s.mu.Lock()
		if len(s.running[nodeID]) >= s.limit || len(s.queued[nodeID]) == 0 {
			s.mu.Unlock()
			break
		}
		next := s.queued[nodeID][0]
		s.queued[nodeID] = s.queued[nodeID][1:]
		if len(s.queued[nodeID]) == 0 {
			delete(s.queued, nodeID)
		}
		next.State = StateRunning
		next.Position = -1
		next.StartedAt = s.clock.Now().UTC()
		s.ensureRunning(nodeID)[next.Job.RunID] = next
		s.mu.Unlock()

		delivered, err := s.pusher.Push(ctx, next.Job)
		if err != nil || delivered == 0 {
			// The node is unreachable: free the slot and abort the run after the
			// drain completes (so the abort's terminal hook is a no-op while we
			// still hold the promoting guard).
			s.mu.Lock()
			s.removeLocked(nodeID, next.Job.RunID)
			s.mu.Unlock()
			toAbort = append(toAbort, next.Job.RunID)
		} else if s.store != nil {
			now := s.clock.Now().UTC()
			if markErr := s.store.MarkPushed(ctx, next.Job.RunID, now, now.Add(s.deliveryLease)); markErr != nil {
				s.mu.Lock()
				s.removeLocked(nodeID, next.Job.RunID)
				s.mu.Unlock()
				toAbort = append(toAbort, next.Job.RunID)
			}
		}
	}

	for _, rid := range toAbort {
		// The terminal hook will re-enter Complete; the promoting guard (still
		// held) makes that a no-op, and our loop already drained the queue.
		_ = s.aborter.Abort(ctx, rid, "node unreachable when promoting from queue")
		if s.store != nil {
			_ = s.store.MarkFailedDelivery(ctx, rid, "node unreachable when promoting from queue", s.clock.Now().UTC())
		}
	}

	s.mu.Lock()
	delete(s.promoting, nodeID)
	s.mu.Unlock()
}

// Snapshot returns the live queue view: one NodeQueue per node with any running
// or queued job, optionally narrowed to a single node. Running entries come
// first, then queued in FIFO order with their current positions.
func (s *Scheduler) Snapshot(nodeID string) []NodeQueue {
	s.mu.Lock()
	defer s.mu.Unlock()

	nodes := make(map[string]struct{})
	for id := range s.running {
		if len(s.running[id]) > 0 {
			nodes[id] = struct{}{}
		}
	}
	for id := range s.queued {
		if len(s.queued[id]) > 0 {
			nodes[id] = struct{}{}
		}
	}

	out := make([]NodeQueue, 0, len(nodes))
	for id := range nodes {
		if nodeID != "" && id != nodeID {
			continue
		}
		out = append(out, s.nodeQueueLocked(id))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].NodeID < out[j].NodeID })
	return out
}

// nodeQueueLocked builds one node's view; the caller holds the lock.
func (s *Scheduler) nodeQueueLocked(nodeID string) NodeQueue {
	running := make([]Entry, 0, len(s.running[nodeID]))
	for _, e := range s.running[nodeID] {
		running = append(running, e)
	}
	// Stable order for running entries (by start time then run id).
	sort.Slice(running, func(i, j int) bool {
		if !running[i].StartedAt.Equal(running[j].StartedAt) {
			return running[i].StartedAt.Before(running[j].StartedAt)
		}
		return running[i].Job.RunID < running[j].Job.RunID
	})

	entries := make([]Entry, 0, len(running)+len(s.queued[nodeID]))
	entries = append(entries, running...)
	for i, e := range s.queued[nodeID] {
		e.Position = i
		entries = append(entries, e)
	}

	return NodeQueue{
		NodeID:           nodeID,
		ConcurrencyLimit: s.limit,
		Running:          len(s.running[nodeID]),
		Queued:           len(s.queued[nodeID]),
		Entries:          entries,
	}
}

// removeLocked drops runID from both the running set and the queued list for
// nodeID; the caller holds the lock.
func (s *Scheduler) removeLocked(nodeID, runID string) {
	if set := s.running[nodeID]; set != nil {
		delete(set, runID)
		if len(set) == 0 {
			delete(s.running, nodeID)
		}
	}
	if q := s.queued[nodeID]; len(q) > 0 {
		filtered := q[:0]
		for _, e := range q {
			if e.Job.RunID != runID {
				filtered = append(filtered, e)
			}
		}
		if len(filtered) == 0 {
			delete(s.queued, nodeID)
		} else {
			s.queued[nodeID] = filtered
		}
	}
}

// ensureRunning returns (creating if needed) the running set for nodeID; the
// caller holds the lock.
func (s *Scheduler) ensureRunning(nodeID string) map[string]Entry {
	set := s.running[nodeID]
	if set == nil {
		set = make(map[string]Entry)
		s.running[nodeID] = set
	}
	return set
}

func (s *Scheduler) runningEntry(job Job) Entry {
	now := s.clock.Now().UTC()
	return Entry{Job: job, State: StateRunning, Position: -1, EnqueuedAt: now, StartedAt: now}
}

func (s *Scheduler) queuedEntry(job Job) Entry {
	return Entry{Job: job, State: StateQueued, EnqueuedAt: s.clock.Now().UTC()}
}
