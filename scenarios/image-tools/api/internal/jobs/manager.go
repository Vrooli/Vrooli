package jobs

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/vrooli/api-core/schedule"

	"github.com/google/uuid"
)

// Runner executes a job's actual work. It runs under a server-lifetime context
// (jobCtx) that is canceled only by Cancel or server shutdown — never by a
// client disconnect. emit reports progress (0..100) + a message; the manager
// persists and streams it. The returned ref is the output (blob key / path).
type Runner func(jobCtx context.Context, job Job, emit func(progress int, message string)) (ref string, err error)

// Manager errors.
var (
	// ErrNotFound is returned for an unknown job id.
	ErrNotFound = errors.New("jobs: not found")
	// ErrQueueFull is returned when a lane's queue is saturated.
	ErrQueueFull = errors.New("jobs: queue full")
	// ErrNotStarted is returned when Submit is called before Start.
	ErrNotStarted = errors.New("jobs: manager not started")
)

const laneQueueCap = 1024

// Config configures a Manager.
type Config struct {
	// Runner executes work. Required.
	Runner Runner
	// Clock supplies timestamps. Defaults to schedule.System().
	Clock schedule.Clock
	// CPUWorkers is the concurrent CPU lane size. Defaults to 4 when <= 0.
	CPUWorkers int
	// OnComplete, when set, is called once per job as it reaches a terminal state
	// (succeeded/failed/canceled). The measures recorder uses it to capture op
	// latency + queue-wait without coupling the Manager to the measures package.
	// It must not block; it runs on the finalizing goroutine after subscribers
	// are notified.
	OnComplete func(job Job)
}

// Manager owns durable jobs: it serializes GPU work, runs CPU work
// concurrently, and survives client disconnects (work runs under baseCtx).
type Manager struct {
	st         *store
	runner     Runner
	clock      schedule.Clock
	cpuN       int
	onComplete func(Job)
	baseCtx    context.Context
	cancel     context.CancelFunc

	mu      sync.Mutex
	entries map[string]*jobEntry
	gpuCh   chan *jobEntry
	cpuCh   chan *jobEntry
	started bool
	wg      sync.WaitGroup
}

type subscriber struct {
	ch     chan ProgressEvent
	closed bool
}

type jobEntry struct {
	mu     sync.Mutex
	job    Job
	cancel context.CancelFunc
	done   chan struct{}
	subs   []*subscriber
	last   *ProgressEvent
}

// New builds a Manager backed by db. Call Start before Submit.
func New(db SQLExecutor, cfg Config) *Manager {
	clk := cfg.Clock
	if clk == nil {
		clk = schedule.System()
	}
	n := cfg.CPUWorkers
	if n <= 0 {
		n = 4
	}
	return &Manager{
		st:         newStore(db),
		runner:     cfg.Runner,
		clock:      clk,
		cpuN:       n,
		onComplete: cfg.OnComplete,
		entries:    make(map[string]*jobEntry),
		gpuCh:      make(chan *jobEntry, laneQueueCap),
		cpuCh:      make(chan *jobEntry, laneQueueCap),
	}
}

// Start launches the lane workers and runs restart recovery. ctx is the
// server-lifetime context; canceling it (or calling Close) stops the manager.
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.started {
		m.mu.Unlock()
		return nil
	}
	if m.runner == nil {
		m.mu.Unlock()
		return fmt.Errorf("jobs: runner is required")
	}
	m.baseCtx, m.cancel = context.WithCancel(ctx)
	m.started = true
	m.mu.Unlock()

	if err := m.recover(); err != nil {
		return err
	}

	// One GPU worker (serialization); N CPU workers (concurrency).
	m.wg.Add(1)
	go m.worker(m.gpuCh)
	for i := 0; i < m.cpuN; i++ {
		m.wg.Add(1)
		go m.worker(m.cpuCh)
	}
	return nil
}

// Close stops workers and waits for in-flight jobs to observe cancellation.
func (m *Manager) Close() {
	m.mu.Lock()
	if !m.started {
		m.mu.Unlock()
		return
	}
	m.started = false
	cancel := m.cancel
	m.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	m.wg.Wait()
}

// recover marks jobs left non-terminal by a previous process as failed: generic
// backend work cannot be resumed mid-flight, so we fail-loud rather than lie
// that they are still running.
func (m *Manager) recover() error {
	orphans, err := m.st.listNonTerminal(m.baseCtx)
	if err != nil {
		return err
	}
	now := m.clock.Now()
	for _, j := range orphans {
		j.State = StateFailed
		j.Error = "interrupted by server restart"
		j.FinishedAt = &now
		if err := m.st.update(m.baseCtx, j); err != nil {
			return err
		}
	}
	return nil
}

// Submit accepts a unit of work, persists it as queued, enqueues it on its
// lane, and returns the job (with id + ETA) immediately.
func (m *Manager) Submit(ctx context.Context, spec Spec) (Job, error) {
	m.mu.Lock()
	if !m.started {
		m.mu.Unlock()
		return Job{}, ErrNotStarted
	}
	m.mu.Unlock()

	lane := spec.Lane
	if lane != LaneGPU {
		lane = LaneCPU
	}
	now := m.clock.Now()
	job := Job{
		ID:               uuid.NewString(),
		Operation:        spec.Operation,
		Lane:             lane,
		State:            StateQueued,
		Payload:          spec.Payload,
		EstimatedSeconds: spec.EstimatedSeconds,
		CreatedAt:        now,
	}
	// Persist under baseCtx so a caller disconnect can't lose the record.
	if err := m.st.insert(m.baseCtx, job); err != nil {
		return Job{}, err
	}

	entry := &jobEntry{job: job, done: make(chan struct{})}
	m.mu.Lock()
	m.entries[job.ID] = entry
	ch := m.cpuCh
	if lane == LaneGPU {
		ch = m.gpuCh
	}
	m.mu.Unlock()

	select {
	case ch <- entry:
		return job, nil
	default:
		// Lane saturated: fail the job rather than block Submit.
		failed := job
		failed.State = StateFailed
		failed.Error = "lane queue is full"
		failed.FinishedAt = &now
		_ = m.st.update(m.baseCtx, failed)
		m.finalizeEntry(entry, failed)
		return Job{}, ErrQueueFull
	}
}

// Record persists an already-completed unit of work as a terminal job. It is
// the path for SYNCHRONOUS operations that execute inline in the request (the
// deterministic image ops finish in milliseconds, so the durable queue's
// disconnect-survival buys nothing) yet must still appear in List/Get and the
// UI job monitor alongside queued async work. runErr nil records a succeeded
// job carrying resultRef; a non-nil runErr records a failed job. The record is
// persisted under baseCtx so it is durable regardless of the caller's context.
func (m *Manager) Record(spec Spec, resultRef string, runErr error) (Job, error) {
	m.mu.Lock()
	started := m.started
	m.mu.Unlock()
	if !started {
		return Job{}, ErrNotStarted
	}
	lane := spec.Lane
	if lane != LaneGPU {
		lane = LaneCPU
	}
	now := m.clock.Now()
	job := Job{
		ID:               uuid.NewString(),
		Operation:        spec.Operation,
		Lane:             lane,
		Payload:          spec.Payload,
		EstimatedSeconds: spec.EstimatedSeconds,
		CreatedAt:        now,
		StartedAt:        &now,
		FinishedAt:       &now,
		Progress:         100,
	}
	if runErr != nil {
		job.State = StateFailed
		job.Error = runErr.Error()
		job.Progress = 0
	} else {
		job.State = StateSucceeded
		job.ResultRef = resultRef
	}
	if err := m.st.insert(m.baseCtx, job); err != nil {
		return Job{}, err
	}
	if m.onComplete != nil {
		m.onComplete(job)
	}
	return job, nil
}

func (m *Manager) worker(ch chan *jobEntry) {
	defer m.wg.Done()
	for {
		select {
		case <-m.baseCtx.Done():
			return
		case entry := <-ch:
			m.execute(entry)
		}
	}
}

func (m *Manager) execute(entry *jobEntry) {
	entry.mu.Lock()
	if entry.job.State.Terminal() {
		// Canceled while queued — skip.
		entry.mu.Unlock()
		return
	}
	jobCtx, cancel := context.WithCancel(m.baseCtx)
	entry.cancel = cancel
	now := m.clock.Now()
	entry.job.State = StateRunning
	entry.job.StartedAt = &now
	snapshot := entry.job
	entry.mu.Unlock()
	defer cancel()

	_ = m.st.update(m.baseCtx, snapshot)
	m.emit(entry, StateRunning, snapshot.Progress, "started")

	emit := func(progress int, message string) {
		if progress < 0 {
			progress = 0
		}
		if progress > 100 {
			progress = 100
		}
		entry.mu.Lock()
		entry.job.Progress = progress
		entry.job.Message = message
		snap := entry.job
		entry.mu.Unlock()
		_ = m.st.update(m.baseCtx, snap)
		m.emit(entry, StateRunning, progress, message)
	}

	ref, err := m.runner(jobCtx, snapshot, emit)

	entry.mu.Lock()
	fin := m.clock.Now()
	entry.job.FinishedAt = &fin
	switch {
	case err != nil && jobCtx.Err() != nil:
		entry.job.State = StateCanceled
		entry.job.Message = "canceled"
	case err != nil:
		entry.job.State = StateFailed
		entry.job.Error = err.Error()
	default:
		entry.job.State = StateSucceeded
		entry.job.Progress = 100
		entry.job.ResultRef = ref
	}
	final := entry.job
	entry.mu.Unlock()

	_ = m.st.update(m.baseCtx, final)
	m.finalizeEntry(entry, final)
}

// finalizeEntry records the terminal state, emits a final event, and closes the
// done channel + subscribers exactly once.
func (m *Manager) finalizeEntry(entry *jobEntry, final Job) {
	entry.mu.Lock()
	entry.job = final
	select {
	case <-entry.done:
		// already finalized
		entry.mu.Unlock()
		return
	default:
	}
	ev := ProgressEvent{JobID: final.ID, State: final.State, Progress: final.Progress, Message: final.Message, At: m.clock.Now()}
	entry.last = &ev
	for _, s := range entry.subs {
		if !s.closed {
			trySend(s.ch, ev)
			close(s.ch)
			s.closed = true
		}
	}
	entry.subs = nil
	close(entry.done)
	entry.mu.Unlock()

	if m.onComplete != nil && final.State.Terminal() {
		m.onComplete(final)
	}
}

func (m *Manager) emit(entry *jobEntry, state State, progress int, message string) {
	ev := ProgressEvent{JobID: entry.job.ID, State: state, Progress: progress, Message: message, At: m.clock.Now()}
	entry.mu.Lock()
	entry.last = &ev
	for _, s := range entry.subs {
		if !s.closed {
			trySend(s.ch, ev)
		}
	}
	entry.mu.Unlock()
}

func trySend(ch chan ProgressEvent, ev ProgressEvent) {
	select {
	case ch <- ev:
	default:
		// Slow subscriber: drop this intermediate event. The terminal event
		// and Get/Wait still reflect the true final state.
	}
}

// Cancel aborts a job. A running job's context is canceled; a still-queued job
// is marked canceled immediately so Wait returns without executing it.
func (m *Manager) Cancel(id string) error {
	m.mu.Lock()
	entry, ok := m.entries[id]
	m.mu.Unlock()
	if !ok {
		return fmt.Errorf("%w: %s", ErrNotFound, id)
	}
	entry.mu.Lock()
	if entry.job.State.Terminal() {
		entry.mu.Unlock()
		return nil
	}
	if entry.job.State == StateRunning && entry.cancel != nil {
		c := entry.cancel
		entry.mu.Unlock()
		c() // execute() observes ctx cancellation and finalizes as canceled
		return nil
	}
	// Queued: finalize as canceled now.
	now := m.clock.Now()
	entry.job.State = StateCanceled
	entry.job.FinishedAt = &now
	entry.job.Message = "canceled"
	final := entry.job
	entry.mu.Unlock()
	_ = m.st.update(m.baseCtx, final)
	m.finalizeEntry(entry, final)
	return nil
}

// Wait blocks ONCE until the job is terminal, then returns it. If ctx is
// canceled first, Wait returns ctx.Err() WITHOUT affecting the job (the work
// continues server-side) — this is what makes a client disconnect survivable.
func (m *Manager) Wait(ctx context.Context, id string) (Job, error) {
	m.mu.Lock()
	entry, ok := m.entries[id]
	m.mu.Unlock()
	if !ok {
		// Not in memory — maybe a terminal job from a prior process.
		j, err := m.st.get(ctx, id)
		if err != nil {
			return Job{}, fmt.Errorf("%w: %s", ErrNotFound, id)
		}
		return j, nil
	}
	select {
	case <-entry.done:
		entry.mu.Lock()
		j := entry.job
		entry.mu.Unlock()
		return j, nil
	case <-ctx.Done():
		return Job{}, ctx.Err()
	}
}

// Get returns the current job record.
func (m *Manager) Get(ctx context.Context, id string) (Job, error) {
	m.mu.Lock()
	entry, ok := m.entries[id]
	m.mu.Unlock()
	if ok {
		entry.mu.Lock()
		j := entry.job
		entry.mu.Unlock()
		return j, nil
	}
	j, err := m.st.get(ctx, id)
	if err != nil {
		return Job{}, fmt.Errorf("%w: %s", ErrNotFound, id)
	}
	return j, nil
}

// List returns recent jobs, newest first.
func (m *Manager) List(ctx context.Context, limit int) ([]Job, error) {
	return m.st.list(ctx, limit)
}

// Subscribe returns a channel of progress events for the job and an unsubscribe
// function. The last known event (if any) is replayed immediately. The channel
// is closed when the job reaches a terminal state or unsubscribe is called.
func (m *Manager) Subscribe(id string) (<-chan ProgressEvent, func(), error) {
	m.mu.Lock()
	entry, ok := m.entries[id]
	m.mu.Unlock()
	if !ok {
		return nil, nil, fmt.Errorf("%w: %s", ErrNotFound, id)
	}
	sub := &subscriber{ch: make(chan ProgressEvent, 64)}
	entry.mu.Lock()
	if entry.job.State.Terminal() {
		// Already done: emit a final event and a closed channel.
		ev := ProgressEvent{JobID: entry.job.ID, State: entry.job.State, Progress: entry.job.Progress, Message: entry.job.Message, At: m.clock.Now()}
		entry.mu.Unlock()
		sub.ch <- ev
		close(sub.ch)
		sub.closed = true
		return sub.ch, func() {}, nil
	}
	if entry.last != nil {
		trySend(sub.ch, *entry.last)
	}
	entry.subs = append(entry.subs, sub)
	entry.mu.Unlock()

	unsub := func() {
		entry.mu.Lock()
		defer entry.mu.Unlock()
		for i, s := range entry.subs {
			if s == sub {
				entry.subs = append(entry.subs[:i], entry.subs[i+1:]...)
				if !s.closed {
					close(s.ch)
					s.closed = true
				}
				break
			}
		}
	}
	return sub.ch, unsub, nil
}
