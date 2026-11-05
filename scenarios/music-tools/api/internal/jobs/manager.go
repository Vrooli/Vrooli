package jobs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"maps"
	"sync"
	"time"

	"github.com/vrooli/api-core/schedule"

	"github.com/google/uuid"
)

// Runner executes a job's actual work. It runs under a server-lifetime context
// (jobCtx) that is canceled only by Cancel or server shutdown — never by a
// client disconnect. emit reports progress (0..100) + a message; the manager
// persists and streams it.
type Runner func(jobCtx context.Context, job Job, emit func(progress int, message string)) (Result, error)

// Result is what a Runner produces.
//
// It is a struct rather than a bare ref string because a backend learns facts
// while running that nothing downstream can reconstruct afterwards: which model
// actually served the request, which tier it ran on, and what the route cost.
// Those were assembled by the AI backends and then dropped one stack frame
// later, so a caller could be billed for a cloud generation with no record of
// the charge anywhere in the system.
type Result struct {
	// Ref is the output reference (blob key / path).
	Ref string
	// Meta carries the backend's own record of the run, keyed by the backend's
	// vocabulary. It is persisted with the job and served on the wire, so a
	// caller reads what happened rather than what it asked for. Partial facts are
	// retained on failure/cancel only if this worker wins terminal persistence.
	Meta map[string]string
}

// Manager errors.
var (
	// ErrNotFound is returned for an unknown job id.
	ErrNotFound = errors.New("jobs: not found")
	// ErrQueueFull is returned when a lane's queue is saturated.
	ErrQueueFull = errors.New("jobs: queue full")
	// ErrNotStarted is returned when Submit is called before Start.
	ErrNotStarted  = errors.New("jobs: manager not started")
	ErrPersistence = errors.New("jobs: persistence unavailable; completed artifacts retained")
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
	// TerminalCacheLimit bounds persisted terminal entries; active and unpersisted
	// finalizations are retained. Defaults to 256. Durable rows are not deleted.
	TerminalCacheLimit int
	// NetworkWorkers is the concurrent network lane size for remote tiers.
	// Defaults to 4 when <= 0.
	NetworkWorkers int
	// OnComplete, when set, is called once per job as it reaches a terminal state
	// (succeeded/failed/canceled). The measures recorder uses it to capture op
	// latency + queue-wait without coupling the Manager to the measures package.
	// It must not block; it runs on the finalizing goroutine after subscribers
	// are notified.
	OnComplete func(job Job)
	// Admit runs immediately before a queued job starts. It may acquire a
	// resource lease for the whole job lifetime and returns a release callback
	// plus metadata that is merged into the terminal job record.
	Admit func(context.Context, context.CancelCauseFunc, Job) (release func(), meta map[string]string, err error)
}

// Manager owns durable jobs: it serializes GPU work, runs CPU work
// concurrently, and survives client disconnects (work runs under baseCtx).
type Manager struct {
	st         *store
	runner     Runner
	clock      schedule.Clock
	cpuN       int
	netN       int
	onComplete func(Job)
	admit      func(context.Context, context.CancelCauseFunc, Job) (func(), map[string]string, error)
	baseCtx    context.Context
	cancel     context.CancelFunc

	mu            sync.Mutex
	entries       map[string]*jobEntry
	terminalIDs   []string
	terminalLimit int
	gpuCh         chan *jobEntry
	cpuCh         chan *jobEntry
	netCh         chan *jobEntry
	started       bool
	starting      bool
	wg            sync.WaitGroup
}

type subscriber struct {
	ch     chan ProgressEvent
	closed bool
}

type jobEntry struct {
	persistenceErr error
	pending        *Job
	completed      bool
	mu             sync.Mutex
	job            Job
	cancel         context.CancelFunc
	done           chan struct{}
	subs           []*subscriber
	last           *ProgressEvent
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
	netN := cfg.NetworkWorkers
	if netN <= 0 {
		netN = 4
	}
	terminalLimit := cfg.TerminalCacheLimit
	if terminalLimit <= 0 {
		terminalLimit = 256
	}
	return &Manager{
		terminalLimit: terminalLimit,
		st:            newStore(db),
		runner:        cfg.Runner,
		clock:         clk,
		cpuN:          n,
		netN:          netN,
		onComplete:    cfg.OnComplete,
		admit:         cfg.Admit,
		entries:       make(map[string]*jobEntry),
		gpuCh:         make(chan *jobEntry, laneQueueCap),
		cpuCh:         make(chan *jobEntry, laneQueueCap),
		netCh:         make(chan *jobEntry, laneQueueCap),
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
	if m.starting {
		m.mu.Unlock()
		return fmt.Errorf("jobs: initialization already in progress")
	}
	m.baseCtx, m.cancel = context.WithCancel(ctx)
	m.starting = true
	m.mu.Unlock()

	if err := m.st.migrate(m.baseCtx); err != nil {
		m.mu.Lock()
		m.starting = false
		m.mu.Unlock()
		m.cancel()
		return err
	}
	if err := m.recover(); err != nil {
		m.mu.Lock()
		m.starting = false
		m.mu.Unlock()
		m.cancel()
		return err
	}

	m.mu.Lock()
	m.starting = false
	m.started = true
	m.mu.Unlock()

	// One GPU worker (serialization); N CPU workers (concurrency); a separate
	// bounded network pool so remote tiers never queue behind the GPU worker.
	m.wg.Add(1)
	go m.worker(m.gpuCh)
	for i := 0; i < m.cpuN; i++ {
		m.wg.Add(1)
		go m.worker(m.cpuCh)
	}
	for i := 0; i < m.netN; i++ {
		m.wg.Add(1)
		go m.worker(m.netCh)
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
			var winner *terminalWinner
			if errors.As(err, &winner) {
				continue
			}
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

	lane := normalizeLane(spec.Lane)
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
	ch := m.laneChannel(lane)
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
		m.persistFinal(entry, failed)
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
	lane := normalizeLane(spec.Lane)
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
	jobCtx, cancelCause := context.WithCancelCause(m.baseCtx)
	cancel := func() { cancelCause(context.Canceled) }
	entry.cancel = cancel
	now := m.clock.Now()
	entry.job.State = StateRunning
	entry.job.StartedAt = &now
	snapshot := entry.job
	entry.mu.Unlock()
	defer cancel()

	if err := m.st.update(m.baseCtx, snapshot); err != nil {
		if m.adoptTerminal(entry, err) {
			return
		}
		snapshot.State = StateFailed
		snapshot.Error = "persist running: " + err.Error()
		m.persistFinal(entry, snapshot)
		return
	}
	m.emit(entry, StateRunning, snapshot.Progress, "started")

	release, admissionMeta, admissionErr := func() (func(), map[string]string, error) {
		if m.admit == nil || snapshot.Lane != LaneGPU {
			return nil, nil, nil
		}
		return m.admit(jobCtx, cancelCause, snapshot)
	}()
	if admissionErr != nil {
		entry.mu.Lock()
		fin := m.clock.Now()
		final := entry.job
		final.State = StateFailed
		final.Error = admissionErr.Error()
		final.FinishedAt = &fin
		entry.mu.Unlock()
		m.persistFinal(entry, final)
		return
	}
	if release != nil {
		defer release()
	}
	if len(admissionMeta) > 0 {
		entry.mu.Lock()
		if entry.job.Meta == nil {
			entry.job.Meta = map[string]string{}
		}
		for key, value := range admissionMeta {
			entry.job.Meta[key] = value
		}
		snapshot = entry.job
		entry.mu.Unlock()
		if err := m.st.update(m.baseCtx, snapshot); err != nil {
			if m.adoptTerminal(entry, err) {
				return
			}
			snapshot.State = StateFailed
			snapshot.Error = "persist admission: " + err.Error()
			m.persistFinal(entry, snapshot)
			return
		}
	}

	emit := func(progress int, message string) {
		if progress < 0 {
			progress = 0
		}
		if progress > 100 {
			progress = 100
		}
		entry.mu.Lock()
		if entry.job.State.Terminal() {
			entry.mu.Unlock()
			return
		}
		entry.job.Progress = progress
		entry.job.Message = message
		snap := entry.job
		entry.mu.Unlock()
		if err := m.st.update(m.baseCtx, snap); err != nil {
			cancelCause(fmt.Errorf("persist progress: %w", err))
			m.adoptTerminal(entry, err)
			return
		}
		m.emit(entry, StateRunning, progress, message)
	}

	result, err := m.runner(jobCtx, snapshot, emit)

	entry.mu.Lock()
	final := entry.job
	final.Meta = maps.Clone(entry.job.Meta)
	fin := m.clock.Now()
	final.FinishedAt = &fin
	if len(result.Meta) > 0 {
		if final.Meta == nil {
			final.Meta = map[string]string{}
		}
		maps.Copy(final.Meta, result.Meta)
	}
	switch {
	case jobCtx.Err() != nil:
		if cause := context.Cause(jobCtx); cause != context.Canceled {
			final.State = StateFailed
			final.Error = cause.Error()
		} else {
			final.State = StateCanceled
			final.Message = "canceled"
		}
	case err != nil:
		final.State = StateFailed
		final.Error = err.Error()
	default:
		final.State = StateSucceeded
		final.Progress = 100
		final.ResultRef = result.Ref

	}
	entry.mu.Unlock()

	m.persistFinal(entry, final)
}

// finalizeEntry records the terminal state, emits a final event, and closes the
// done channel + subscribers exactly once.
// persistFinal saves before notification. Failed finalization is retained for
// idempotent Get/Wait retry; it never reruns the producer or invents success.
func (m *Manager) persistFinal(entry *jobEntry, final Job) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(m.baseCtx), 5*time.Second)
	defer cancel()
	if err := m.st.update(ctx, final); err != nil {
		if m.adoptTerminal(entry, err) {
			return
		}
		entry.mu.Lock()
		entry.pending = &final
		entry.persistenceErr = fmt.Errorf("%w: %v", ErrPersistence, err)
		select {
		case <-entry.done:
		default:
			close(entry.done)
		}
		for _, sub := range entry.subs {
			if !sub.closed {
				close(sub.ch)
				sub.closed = true
			}
		}
		entry.subs = nil
		entry.mu.Unlock()
		return
	}
	m.finalizeEntry(entry, final)
}

func (m *Manager) adoptTerminal(entry *jobEntry, err error) bool {
	var winner *terminalWinner
	if !errors.As(err, &winner) {
		return false
	}
	m.finalizeEntry(entry, winner.job)
	return true
}

func (m *Manager) finalizeEntry(entry *jobEntry, final Job) {
	entry.mu.Lock()
	entry.job = final
	entry.pending = nil
	entry.persistenceErr = nil
	select {
	case <-entry.done:
	default:
		ev := ProgressEvent{JobID: final.ID, State: final.State, Progress: final.Progress, Message: final.Message, At: m.clock.Now()}
		entry.last = &ev
		for _, sub := range entry.subs {
			if !sub.closed {
				sendTerminal(sub.ch, ev)
				close(sub.ch)
				sub.closed = true
			}
		}
		entry.subs = nil
		close(entry.done)
	}
	notify := !entry.completed && final.State.Terminal()
	if notify {
		entry.completed = true
	}
	entry.mu.Unlock()
	if notify {
		m.rememberTerminal(final.ID)
		if m.onComplete != nil {
			m.onComplete(final)
		}
	}
}

// Only persisted finalizations enter this bounded cache. Existing Wait callers
// retain their entry pointer; future observers recover the same durable row.
func (m *Manager) rememberTerminal(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.terminalIDs = append(m.terminalIDs, id)
	for len(m.terminalIDs) > m.terminalLimit {
		old := m.terminalIDs[0]
		m.terminalIDs = m.terminalIDs[1:]
		entry := m.entries[old]
		if entry == nil {
			continue
		}
		entry.mu.Lock()
		eligible := entry.pending == nil && entry.job.State.Terminal() && len(entry.subs) == 0
		entry.mu.Unlock()
		if eligible {
			delete(m.entries, old)
		}
	}
}

func (m *Manager) readEntry(ctx context.Context, entry *jobEntry) (Job, error) {
	entry.mu.Lock()
	if entry.pending == nil {
		id, err := entry.job.ID, entry.persistenceErr
		entry.mu.Unlock()
		if err != nil {
			return Job{}, err
		}
		return m.durableGet(ctx, id)
	}
	final := *entry.pending
	if err := m.st.update(ctx, final); err != nil {
		entry.mu.Unlock()
		var winner *terminalWinner
		if errors.As(err, &winner) {
			m.finalizeEntry(entry, winner.job)
			return winner.job, nil
		}
		if ctx.Err() != nil {
			return Job{}, ctx.Err()
		}
		return Job{}, fmt.Errorf("%w: %w", ErrPersistence, err)
	}
	// Keep competing readers from rewriting a terminal job after recovery.
	entry.pending = nil
	entry.persistenceErr = nil
	entry.job = final
	entry.mu.Unlock()
	m.finalizeEntry(entry, final)
	return final, nil
}

func (m *Manager) emit(entry *jobEntry, state State, progress int, message string) {
	entry.mu.Lock()
	if entry.job.State.Terminal() {
		entry.mu.Unlock()
		return
	}
	ev := ProgressEvent{JobID: entry.job.ID, State: state, Progress: progress, Message: message, At: m.clock.Now()}
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

// Terminal state takes precedence over discardable progress. The entry lock
// serializes senders/close, so dropping one buffered progress event guarantees
// space without waiting for a slow consumer.
func sendTerminal(ch chan ProgressEvent, ev ProgressEvent) {
	select {
	case ch <- ev:
		return
	default:
	}
	select {
	case <-ch:
	default:
	}
	ch <- ev
}

// Cancel aborts a job. A running job's context is canceled; a still-queued job
// is marked canceled immediately so Wait returns without executing it.
func (m *Manager) Cancel(id string) error {
	m.mu.Lock()
	entry, ok := m.entries[id]
	m.mu.Unlock()
	if !ok {
		job, err := m.durableGet(m.baseCtx, id)
		if err != nil {
			return err
		}
		if job.State.Terminal() {
			return nil
		}
		return fmt.Errorf("job %s is not active in this process", id)
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
	final := entry.job
	now := m.clock.Now()
	final.State = StateCanceled
	final.FinishedAt = &now
	final.Message = "canceled"
	if err := m.st.update(m.baseCtx, final); err != nil {
		entry.mu.Unlock()
		if m.adoptTerminal(entry, err) {
			return nil
		}
		return fmt.Errorf("%w: %v", ErrPersistence, err)
	}
	entry.job = final
	entry.mu.Unlock()
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
		return m.durableGet(ctx, id)
	}
	select {
	case <-entry.done:
		return m.readEntry(ctx, entry)
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
		return m.readEntry(ctx, entry)
	}
	return m.durableGet(ctx, id)
}

func (m *Manager) durableGet(ctx context.Context, id string) (Job, error) {
	job, err := m.st.get(ctx, id)
	if err == nil {
		return job, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return Job{}, fmt.Errorf("%w: %s", ErrNotFound, id)
	}
	if ctx.Err() != nil {
		return Job{}, ctx.Err()
	}
	return Job{}, fmt.Errorf("%w: read job %s: %w", ErrPersistence, id, err)
}

// List returns recent jobs, newest first.
func (m *Manager) List(ctx context.Context, limit int) ([]Job, error) {
	return m.st.list(ctx, limit)
}

// ListRetentionCandidates returns the oldest successful jobs with managed
// output blobs older than before. It is intentionally separate from List so a
// cleanup owner cannot accidentally treat arbitrary recent or caller-owned job
// references as reclaimable.
func (m *Manager) ListRetentionCandidates(ctx context.Context, before time.Time, limit int) ([]Job, error) {
	return m.st.listRetentionCandidates(ctx, before, limit)
}

// Subscribe returns a channel of progress events for the job and an unsubscribe
// function. The last known event (if any) is replayed immediately. The channel
// is closed when the job reaches a terminal state or unsubscribe is called.
func (m *Manager) Subscribe(id string) (<-chan ProgressEvent, func(), error) {
	m.mu.Lock()
	entry, ok := m.entries[id]
	m.mu.Unlock()
	if !ok {
		job, err := m.durableGet(m.baseCtx, id)
		if err != nil {
			return nil, nil, err
		}
		if !job.State.Terminal() {
			return nil, nil, fmt.Errorf("job %s is not active in this process", id)
		}
		ch := make(chan ProgressEvent, 1)
		ch <- ProgressEvent{JobID: job.ID, State: job.State, Progress: job.Progress, Message: job.Message, At: m.clock.Now()}
		close(ch)
		return ch, func() {}, nil
	}
	sub := &subscriber{ch: make(chan ProgressEvent, 64)}
	entry.mu.Lock()
	if entry.persistenceErr != nil {
		err := entry.persistenceErr
		entry.mu.Unlock()
		return nil, nil, err
	}
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

// normalizeLane folds an unknown or empty lane onto the CPU lane; the three
// declared lanes pass through.
func normalizeLane(lane Lane) Lane {
	switch lane {
	case LaneGPU, LaneCPU, LaneNetwork:
		return lane
	default:
		return LaneCPU
	}
}

// laneChannel returns the queue a lane's workers read from.
func (m *Manager) laneChannel(lane Lane) chan *jobEntry {
	switch lane {
	case LaneGPU:
		return m.gpuCh
	case LaneNetwork:
		return m.netCh
	default:
		return m.cpuCh
	}
}
