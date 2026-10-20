// Package spawn owns the runner-startup serialization seam.
//
// agent-manager's CreateRun + ResumeRun used to spawn `go executeRun(...)`
// inline. With heartbeat-driven multi-agent callers (prompt-manager team
// agents firing on the same tick), N codex processes entered the fragile
// "register session writer / open rollout file / acquire SQLite WAL lock"
// window simultaneously and burst-failed silently. There was no startup
// concurrency cap, no minimum spacing, no caller-visible queue depth.
//
// [Dispatcher] is the single entry point for starting a run. Behaviour:
//
//   - At most [Config.MaxStartingConcurrency] runs occupy the startup
//     window at once. Default 1 — strict serialization until codex's
//     bootstrap-window contention is proven to tolerate parallelism.
//   - Successive startups are spaced by at least [Config.MinSpacing].
//     Default 0 (no artificial delay).
//   - The queue depth is capped by [Config.QueueCapacity]. When full,
//     [Dispatcher.Enqueue] returns a [domain.CapacityExceededError] —
//     the same typed error the existing capacity ceiling produces.
//   - The starting slot is released either when the executor calls the
//     injected [StartedFn] (signalling the run reached RunStatusRunning)
//     or when the executor goroutine returns, whichever comes first.
//     This is a `defer started()` safety net: a panic, an early-exit
//     terminal failure, or a launcher error all release the slot.
//
// Stats are exposed via [Dispatcher.Stats] so HTTP responses include
// observable backpressure (queueDepth/activeCount/startingCount in the
// CreateRunResponse proto).
//
// CONTRACT (see SEAMS.md, Decision: "Dispatcher.Enqueue is the only
// spawn entry point"):
//   - The dispatcher must never call [ExecuteFn] while holding any
//     internal lock. Locks are released before [ExecuteFn] is invoked.
//   - Lifecycle events (spawn-enqueued, spawn-started) are emitted
//     through [obs.EmitSpawnEnqueued] / [obs.EmitSpawnStarted]. The
//     dispatcher does not invent a parallel event taxonomy.
//   - Closing the dispatcher waits for in-flight workers but does NOT
//     wait for ExecuteFn invocations to return — those are owned by
//     the caller's lifecycle (RunExecutor + finalize seam).
//
// DOC: scenarios/agent-manager/docs/internal/SEAMS.md
// DOC: scenarios/agent-manager/docs/reference/configuration.md
// (Spawn levers).
package spawn

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/obs"

	"github.com/google/uuid"
)

// =============================================================================
// PUBLIC TYPES
// =============================================================================

// StartedFn is the callback the dispatcher injects into [ExecuteFn].
// Calling it signals "the run has reached RunStatusRunning; the codex
// startup window is over; release my starting slot so the next queued
// run can proceed." Safe to call from any goroutine; safe to call
// multiple times (subsequent calls are no-ops).
//
// Even when [ExecuteFn] never calls it (panic, early-exit failure),
// the dispatcher releases the slot when [ExecuteFn] returns. This is
// the `defer started()` safety net — a terminal failure must not
// permanently exhaust the starting-slot semaphore.
type StartedFn func()

// ExecuteFn is the per-run executor body. Implementations call
// `started()` when the run reaches RunStatusRunning. They MAY return
// without calling started — the dispatcher releases the slot
// defensively when ExecuteFn returns.
type ExecuteFn func(started StartedFn)

// Job is one queued unit of work.
type Job struct {
	// RunID identifies the run for log + lifecycle event correlation.
	RunID uuid.UUID

	// RunMode is "sandboxed" / "in-place" — published in spawn-enqueued
	// for operator visibility into what's queued.
	RunMode domain.RunMode

	// RunnerType is "codex" / "claude" / "opencode" — published in
	// spawn-enqueued for operator visibility into the bootstrap surface.
	RunnerType domain.RunnerType

	// Sink is the per-run [emit.Gate] (or any other [obs.Sink]) for
	// emitting lifecycle events to the run timeline. Nil is safe — the
	// helpers fall through to logging only.
	Sink obs.Sink

	// Fn is the executor body. Required.
	Fn ExecuteFn

	// OnPanic owns the durable failure transition when Fn panics. The
	// dispatcher deliberately knows nothing about persistence, so callers wire
	// their run-specific failure path here. Nil is safe for dispatcher-only
	// callers, which still get a recovered, structured log entry.
	OnPanic func(obs.PanicFailure)

	// Admission, when non-nil, is the durable Task-attempt launch fence. The
	// dispatcher claims it immediately before invoking Fn; cancellation that
	// wins this gate prevents the executor from starting.
	Admission *Admission

	enqueuedAt time.Time
}

// Stats is a snapshot of the dispatcher's queue/slot occupancy. Fields
// are independently sampled and may not sum exactly across a single
// snapshot; use them for backpressure surfacing, not for invariants.
type Stats struct {
	QueueDepth    int
	ActiveCount   int
	StartingCount int
}

// Config holds dispatcher parameters. Production wiring fills these
// from [config.SpawnLevers]; tests construct Config directly.
type Config struct {
	// MaxStartingConcurrency caps how many runs can be in the codex-
	// startup window simultaneously. Must be >= 1.
	MaxStartingConcurrency int

	// MinSpacing is the minimum delay between successive
	// slot-acquisition events. Zero disables spacing.
	MinSpacing time.Duration

	// QueueCapacity is the maximum number of queued (not-yet-started)
	// runs. When full, Enqueue returns CapacityExceededError. Must be
	// >= MaxStartingConcurrency so a single burst of size N can be
	// accepted.
	QueueCapacity int

	// Clock provides enqueue and startup timestamps. Nil uses the system clock.
	Clock func() time.Time
}

// ErrDispatcherClosed is returned from Enqueue when Close has already
// been called. Callers should treat it as transient: the dispatcher is
// being shut down with the rest of the orchestrator.
var ErrDispatcherClosed = errors.New("spawn: dispatcher closed")

// =============================================================================
// DISPATCHER
// =============================================================================

// Dispatcher is the single startup-serialization choke point for runs.
// It is safe for concurrent use.
type Dispatcher struct {
	cfg   Config
	clock func() time.Time

	// starting is a buffered semaphore: cap = MaxStartingConcurrency.
	// A worker writes a struct{} to acquire a slot, and runJob reads
	// to release. Buffered so acquire is non-blocking when capacity
	// is available.
	starting chan struct{}

	// queue is the FIFO of pending jobs. Capacity = QueueCapacity.
	// A separate worker goroutine drains the queue serially, applies
	// spacing, then acquires a slot before launching the job.
	queue chan *Job

	// Counters (atomic) for Stats(). Updated lock-free so Stats() is
	// cheap (called inline in the HTTP response path).
	queueDepth    atomic.Int32 // jobs in queue, not yet picked
	activeCount   atomic.Int32 // queued OR slot-held OR running ExecuteFn
	startingCount atomic.Int32 // jobs holding a starting slot right now

	// lifecycle
	closed      atomic.Bool
	workerStop  chan struct{}
	workerDone  chan struct{}
	admissionMu sync.Mutex
	admissions  map[uuid.UUID]*Admission
}

// Admission is the in-process half of a durable Task-attempt claim. Durable
// Task/Run state remains authoritative; this object only closes the race
// between cancellation and dispatch within this owner process. It never holds
// its mutex while calling a provider or executor.
type Admission struct {
	mu              sync.Mutex
	ctx             context.Context
	cancel          context.CancelFunc
	cancelled       bool
	cancelPending   bool
	executorClaimed bool
	externalStarts  int
}

// Context carries cooperative cancellation into provider startup. Cancellation
// during Start is reported as an uncertain external effect by the caller; a
// provider error does not prove that no remote start occurred.
func (a *Admission) Context() context.Context {
	if a == nil {
		return context.Background()
	}
	return a.ctx
}

// BeginExternalStart returns false if cancellation already won. The caller
// must call EndExternalStart after the external call returns.
func (a *Admission) BeginExternalStart() bool {
	if a == nil {
		return true
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cancelled || a.cancelPending {
		return false
	}
	a.externalStarts++
	return true
}

func (a *Admission) EndExternalStart() {
	if a == nil {
		return
	}
	a.mu.Lock()
	if a.externalStarts > 0 {
		a.externalStarts--
	}
	a.mu.Unlock()
}

func (a *Admission) Cancelled() bool {
	if a == nil {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.cancelled
}

// Cancel prevents a later dispatcher claim and cancels in-flight cooperative
// provider work. It reports whether execution or an external start had already
// crossed its admission gate, so the owner can reconcile/stop that exact Run.
func (a *Admission) Cancel() (effectMayHaveStarted bool) {
	if a == nil {
		return false
	}
	a.mu.Lock()
	a.cancelled = true
	started := a.executorClaimed || a.externalStarts > 0
	cancel := a.cancel
	a.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	return started
}

// PrepareCancel closes the executor claim gate while the caller atomically
// persists Task/Run cancellation. Call CommitCancel after persistence succeeds
// or AbortCancel if it fails; no external work is called while this lock is held.
func (a *Admission) PrepareCancel() (effectMayHaveStarted bool) {
	if a == nil {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.cancelPending = true
	return a.executorClaimed || a.externalStarts > 0
}

func (a *Admission) CommitCancel() bool {
	if a == nil {
		return false
	}
	a.mu.Lock()
	a.cancelPending = false
	a.cancelled = true
	started := a.executorClaimed || a.externalStarts > 0
	cancel := a.cancel
	a.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	return started
}

func (a *Admission) AbortCancel() {
	if a == nil {
		return
	}
	a.mu.Lock()
	if !a.cancelled {
		a.cancelPending = false
	}
	a.mu.Unlock()
}

func (a *Admission) claimExecutor() bool {
	if a == nil {
		return true
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cancelled || a.cancelPending {
		return false
	}
	a.executorClaimed = true
	return true
}

// New constructs a Dispatcher with the supplied Config and starts its
// background worker. The worker drains the queue, applies spacing,
// acquires starting slots, and spawns runJob goroutines.
//
// New panics on cfg.MaxStartingConcurrency < 1 or cfg.QueueCapacity <
// MaxStartingConcurrency — both are programmer errors that should be
// caught by [config.SpawnLevers.Validate] long before this point.
func New(cfg Config) *Dispatcher {
	if cfg.MaxStartingConcurrency < 1 {
		panic("spawn.New: MaxStartingConcurrency must be >= 1")
	}
	if cfg.QueueCapacity < cfg.MaxStartingConcurrency {
		panic("spawn.New: QueueCapacity must be >= MaxStartingConcurrency")
	}
	clock := cfg.Clock
	if clock == nil {
		clock = time.Now
	}
	d := &Dispatcher{
		cfg:        cfg,
		clock:      clock,
		starting:   make(chan struct{}, cfg.MaxStartingConcurrency),
		queue:      make(chan *Job, cfg.QueueCapacity),
		workerStop: make(chan struct{}),
		workerDone: make(chan struct{}),
		admissions: make(map[uuid.UUID]*Admission),
	}
	go d.worker()
	return d
}

// ReserveAdmission installs the local cancellation fence before the durable
// Task/Run claim is committed. Call ReleaseAdmission if durable admission
// fails. The returned context is detached from the request lifetime.
func (d *Dispatcher) ReserveAdmission(runID uuid.UUID) (*Admission, error) {
	if runID == uuid.Nil {
		return nil, errors.New("spawn: admission requires a run ID")
	}
	ctx, cancel := context.WithCancel(context.Background())
	a := &Admission{ctx: ctx, cancel: cancel}
	d.admissionMu.Lock()
	defer d.admissionMu.Unlock()
	if d.closed.Load() {
		cancel()
		return nil, ErrDispatcherClosed
	}
	if _, exists := d.admissions[runID]; exists {
		cancel()
		return nil, errors.New("spawn: run admission already reserved")
	}
	d.admissions[runID] = a
	return a, nil
}

// CancelAdmission closes the exact run's local launch gate. The caller must
// persist its Task/Run cancellation fence before invoking this method.
func (d *Dispatcher) CancelAdmission(runID uuid.UUID) bool {
	d.admissionMu.Lock()
	a := d.admissions[runID]
	d.admissionMu.Unlock()
	return a.Cancel()
}

func (d *Dispatcher) PrepareCancelAdmission(runID uuid.UUID) (*Admission, bool) {
	d.admissionMu.Lock()
	a := d.admissions[runID]
	d.admissionMu.Unlock()
	if a == nil {
		return nil, false
	}
	return a, a.PrepareCancel()
}

// ReleaseAdmission forgets a reservation after the attempt has settled or
// durable admission failed. It is safe to call more than once.
func (d *Dispatcher) ReleaseAdmission(runID uuid.UUID, admission *Admission) {
	d.admissionMu.Lock()
	if d.admissions[runID] == admission {
		delete(d.admissions, runID)
	}
	d.admissionMu.Unlock()
	if admission != nil && admission.cancel != nil {
		admission.cancel()
	}
}

// Enqueue submits a job for execution. Returns immediately with either
// a nil error (job is queued) or a typed error:
//
//   - [ErrDispatcherClosed] when Close has been called.
//   - [*domain.CapacityExceededError] when the queue is full.
//   - validation errors when job is malformed.
//
// On success the caller's goroutine returns to its HTTP handler with
// the job-pre-snapshot Stats embedded in the response — the executor
// runs entirely in the dispatcher's lifecycle from this point.
func (d *Dispatcher) Enqueue(job *Job) error {
	return d.enqueue(job)
}

// EnqueueAdmitted submits a job under a previously reserved Task-attempt
// admission. Legacy callers continue using Enqueue without a reservation.
func (d *Dispatcher) EnqueueAdmitted(job *Job, admission *Admission) error {
	if admission == nil {
		return errors.New("spawn.EnqueueAdmitted: admission is nil")
	}
	if job == nil {
		return errors.New("spawn.EnqueueAdmitted: job is nil")
	}
	job.Admission = admission
	return d.enqueue(job)
}

func (d *Dispatcher) enqueue(job *Job) error {
	if job == nil {
		return errors.New("spawn.Enqueue: job is nil")
	}
	if job.Fn == nil {
		return errors.New("spawn.Enqueue: job.Fn is nil")
	}
	d.admissionMu.Lock()
	if d.closed.Load() {
		d.admissionMu.Unlock()
		return ErrDispatcherClosed
	}

	job.enqueuedAt = d.clock()

	// Reserve a queue slot atomically. queueDepth is the authoritative
	// rejection gate — relying on the channel's `default` branch would
	// race the worker (which pulls from the channel before incrementing
	// startingCount, briefly opening false capacity that lets a job in
	// past the cap).
	for {
		cur := d.queueDepth.Load()
		if cur >= int32(d.cfg.QueueCapacity) {
			d.admissionMu.Unlock()
			return &domain.CapacityExceededError{
				Resource: "spawn_queue",
				Current:  int(cur),
				Maximum:  d.cfg.QueueCapacity,
			}
		}
		if d.queueDepth.CompareAndSwap(cur, cur+1) {
			break
		}
	}
	d.activeCount.Add(1)

	// Send to the queue. The CAS above guarantees queueDepth ==
	// channel-resident count (worker decrements queueDepth on pickup),
	// so this send never blocks past channel capacity.
	d.queue <- job
	d.admissionMu.Unlock()

	obs.EmitSpawnEnqueued(job.Sink, job.RunID, obs.SpawnEnqueuedFields{
		RunMode:     job.RunMode,
		RunnerType:  job.RunnerType,
		QueueDepth:  int(d.queueDepth.Load()),
		ActiveCount: int(d.activeCount.Load()),
	})
	return nil
}

// Stats returns the current snapshot. Cheap; safe to call from the
// HTTP response path.
func (d *Dispatcher) Stats() Stats {
	return Stats{
		QueueDepth:    int(d.queueDepth.Load()),
		ActiveCount:   int(d.activeCount.Load()),
		StartingCount: int(d.startingCount.Load()),
	}
}

// Close stops the worker goroutine. In-flight ExecuteFn invocations
// are NOT interrupted — they own their own ctx-driven lifecycle. Close
// is idempotent and safe to call concurrently with Enqueue/Stats; once
// closed, Enqueue returns ErrDispatcherClosed.
func (d *Dispatcher) Close() {
	d.admissionMu.Lock()
	if !d.closed.CompareAndSwap(false, true) {
		d.admissionMu.Unlock()
		return
	}
	close(d.workerStop)
	d.admissionMu.Unlock()
	<-d.workerDone
}

// =============================================================================
// INTERNAL: WORKER
// =============================================================================

// worker drains the queue serially. For each job it applies spacing,
// blocks for a starting slot, then spawns runJob in its own goroutine
// so the worker can keep pulling — runJob holds the slot until either
// StartedFn is called or ExecuteFn returns.
func (d *Dispatcher) worker() {
	defer close(d.workerDone)
	for {
		if d.workerLoop() {
			return
		}
	}
}

// workerLoop is one supervised incarnation of the drain loop. It returns true
// on clean shutdown; a recovered panic returns false so worker() restarts it,
// keeping the only spawn path in the process alive.
func (d *Dispatcher) workerLoop() (clean bool) {
	defer obs.RecoverToFailure("spawn dispatcher worker", nil)

	var lastSpawn time.Time
	for {
		var job *Job
		select {
		case <-d.workerStop:
			d.drainQueueOnClose()
			return true
		case job = <-d.queue:
		}

		// Apply minimum spacing between starts. Re-check workerStop
		// during the sleep so Close() doesn't deadlock waiting for
		// MinSpacing on a closed dispatcher.
		if d.cfg.MinSpacing > 0 && !lastSpawn.IsZero() {
			wait := d.cfg.MinSpacing - d.clock().Sub(lastSpawn)
			if wait > 0 {
				timer := time.NewTimer(wait)
				select {
				case <-timer.C:
				case <-d.workerStop:
					timer.Stop()
					d.discardPickedJob(job)
					d.drainQueueOnClose()
					return true
				}
			}
		}

		// Acquire a starting slot. Blocks if MaxStartingConcurrency
		// is exhausted. Bail on workerStop so Close() returns promptly.
		select {
		case d.starting <- struct{}{}:
		case <-d.workerStop:
			d.discardPickedJob(job)
			d.drainQueueOnClose()
			return true
		}

		// Slot acquired: counter crosses from "queued" to "starting".
		// queueDepth stays elevated through the slot-wait (jobs pulled
		// by the worker but blocked on slot acquisition still count
		// against QueueCapacity from the operator's perspective).
		d.queueDepth.Add(-1)
		d.startingCount.Add(1)
		lastSpawn = d.clock()

		// Contained separately: a panicking sink here would otherwise unwind
		// the loop between slot acquisition and runJob, leaking the starting
		// slot and wedging the only spawn path.
		func() {
			defer obs.RecoverToFailure("spawn started emit", nil)
			obs.EmitSpawnStarted(job.Sink, job.RunID, obs.SpawnStartedFields{
				RunMode:     job.RunMode,
				RunnerType:  job.RunnerType,
				QueuedFor:   lastSpawn.Sub(job.enqueuedAt),
				ActiveCount: int(d.activeCount.Load()),
			})
		}()

		if job.Admission != nil && !job.Admission.claimExecutor() {
			// No ExecuteFn defer will run for a job discarded at this gate. Drop
			// its reservation here so a canceled queued attempt does not retain
			// its context and admission entry for the life of the dispatcher.
			d.ReleaseAdmission(job.RunID, job.Admission)
			<-d.starting
			d.startingCount.Add(-1)
			d.activeCount.Add(-1)
			continue
		}
		go d.runJob(job)
	}
}

// runJob owns the lifecycle of a single accepted job. It guarantees
// the starting slot is released (via sync.Once-wrapped startedFn)
// regardless of how ExecuteFn returns: normal, panic, or never-calls-
// startedFn.
//
// The slot is held from acquisition until whichever happens first:
//   - ExecuteFn calls startedFn (signal: run reached RunStatusRunning)
//   - ExecuteFn returns (defensive release; covers panic + early-exit)
//
// activeCount is decremented only after ExecuteFn returns — Stats()
// reflects "this run is still under dispatcher accounting."
func (d *Dispatcher) runJob(job *Job) {
	var once sync.Once
	release := func() {
		once.Do(func() {
			<-d.starting
			d.startingCount.Add(-1)
		})
	}

	defer func() {
		// Defensive release: even if ExecuteFn panics or never invokes
		// startedFn, the slot is freed exactly once.
		release()
		d.activeCount.Add(-1)
	}()
	defer obs.RecoverToFailure("spawn dispatcher job", job.OnPanic)

	job.Fn(release)
}

// discardPickedJob undoes the counters for a job the worker pulled
// from the channel but won't execute (Close() interrupted before slot
// acquisition). queueDepth is decremented because we hold the
// pulled-but-not-acquired job against the queue cap; activeCount
// matches the original Enqueue increment.
func (d *Dispatcher) discardPickedJob(job *Job) {
	if job == nil {
		return
	}
	if job.Admission != nil {
		d.ReleaseAdmission(job.RunID, job.Admission)
	}
	d.queueDepth.Add(-1)
	d.activeCount.Add(-1)
}

// drainQueueOnClose decrements counters for every job still in the
// channel at shutdown. queueDepth and activeCount must both be undone
// — these are jobs Enqueue accepted but the worker never picked.
func (d *Dispatcher) drainQueueOnClose() {
	for {
		select {
		case job := <-d.queue:
			if job == nil {
				continue
			}
			if job.Admission != nil {
				d.ReleaseAdmission(job.RunID, job.Admission)
			}
			d.queueDepth.Add(-1)
			d.activeCount.Add(-1)
		default:
			return
		}
	}
}
