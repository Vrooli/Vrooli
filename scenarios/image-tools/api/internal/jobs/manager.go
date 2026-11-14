package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/api-core/database"
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
	// caller reads what happened rather than what it asked for.
	Meta map[string]string
}

// Manager errors.
var (
	// ErrNotFound is returned for an unknown job id.
	ErrNotFound = errors.New("jobs: not found")
	// ErrQueueFull is returned when the scheduler queue is saturated.
	ErrQueueFull = errors.New("jobs: queue full")
	// ErrNotStarted is returned when Submit is called before Start.
	ErrNotStarted          = errors.New("jobs: manager not started")
	ErrIdempotencyConflict = errors.New("jobs: client operation id was reused with different input")
)

const queueCapacity = 1024

// Config configures a Manager.
type Config struct {
	// Runner executes work. Required.
	Runner Runner
	// Admit obtains all leases before a job runs. A capacity refusal requeues
	// without executing; Release runs before any terminal event is published.
	Admit Admitter
	// Clock supplies timestamps. Defaults to schedule.System().
	Clock schedule.Clock
	// CPUWorkers bounds all local work, including CPU-backed model inference.
	// Platform capacity determines the safe concurrency below this ceiling.
	CPUWorkers int
	// NetworkWorkers is the concurrent network lane size for remote tiers.
	// Defaults to 4 when <= 0.
	NetworkWorkers int
	// OnComplete, when set, is called once per job as it reaches a terminal state
	// (succeeded/failed/canceled). The measures recorder uses it to capture op
	// latency + queue-wait without coupling the Manager to the measures package.
	// It must not block; it runs on the finalizing goroutine after subscribers
	// are notified.
	OnComplete func(job Job)
}

// Manager owns durable jobs, their one priority queue, and admission lifetime.
// Work survives client disconnects and is terminal only after cleanup returns.
type Manager struct {
	st         *store
	runner     Runner
	admit      Admitter
	clock      schedule.Clock
	cpuN       int
	netN       int
	onComplete func(Job)
	baseCtx    context.Context
	cancel     context.CancelFunc

	mu            sync.Mutex
	entries       map[string]*jobEntry
	pending       []*jobEntry
	wake          chan struct{}
	localActive   int
	networkActive int
	started       bool
	wg            sync.WaitGroup
}

type subscriber struct {
	ch     chan ProgressEvent
	closed bool
}

type jobEntry struct {
	mu          sync.Mutex
	job         Job
	routeValues context.Context
	cancel      context.CancelFunc
	done        chan struct{}
	subs        []*subscriber
	last        *ProgressEvent
	retryAt     time.Time
	persistErr  error
	runner      Runner
}

type testPoolPinner interface {
	ActiveLeaseID() string
	PinTestPool(context.Context, string) (context.Context, error)
}

type valueOverlayContext struct {
	context.Context
	values context.Context
}

func (c valueOverlayContext) Value(key any) any {
	if value := c.values.Value(key); value != nil {
		return value
	}
	return c.Context.Value(key)
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
	return &Manager{
		st:         newStore(db),
		runner:     cfg.Runner,
		admit:      cfg.Admit,
		clock:      clk,
		cpuN:       n,
		netN:       netN,
		onComplete: cfg.OnComplete,
		entries:    make(map[string]*jobEntry),
		wake:       make(chan struct{}, 1),
	}
}

// Start performs restart recovery before admitting work. Failed startup never
// exposes a manager that accepts jobs without a running scheduler.
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.started {
		return nil
	}
	if m.runner == nil {
		return fmt.Errorf("jobs: runner is required")
	}
	m.baseCtx, m.cancel = context.WithCancel(ctx)
	if err := m.st.migrate(m.baseCtx); err != nil {
		m.cancel()
		return err
	}
	if err := m.recover(); err != nil {
		m.cancel()
		return err
	}
	m.started = true
	m.wg.Add(1)
	go m.schedule()
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
		if err := j.transition(StateFailed, now, "interrupted by server restart"); err != nil {
			return err
		}
		j.Error = j.Message
		if err := m.st.update(m.baseCtx, j); err != nil {
			return err
		}
	}
	return nil
}

// Submit persists a job and places it in the shared priority queue.
func (m *Manager) Submit(ctx context.Context, spec Spec) (Job, error) {
	return m.submit(ctx, spec, nil)
}

// SubmitTask routes an inline domain operation through the same scheduler.
// The closure is server-owned until completion, just like the configured runner.
func (m *Manager) SubmitTask(ctx context.Context, spec Spec, run Runner) (Job, error) {
	if run == nil {
		return Job{}, errors.New("jobs: task runner is required")
	}
	return m.submit(ctx, spec, run)
}

func (m *Manager) submit(ctx context.Context, spec Spec, run Runner) (Job, error) {
	m.mu.Lock()
	if !m.started {
		m.mu.Unlock()
		return Job{}, ErrNotStarted
	}
	m.mu.Unlock()
	testMode := database.IsTestMode(ctx)
	routeValues := context.Background()
	if testMode {
		routeValues = database.WithTestMode(routeValues)
		if pinner, ok := m.st.db.(testPoolPinner); ok {
			leaseID := pinner.ActiveLeaseID()
			if leaseID == "" {
				return Job{}, fmt.Errorf("jobs: test-mode submission requires an active database lease")
			}
			var err error
			routeValues, err = pinner.PinTestPool(routeValues, leaseID)
			if err != nil {
				return Job{}, fmt.Errorf("jobs: pin test database lease: %w", err)
			}
		}
	}
	storeCtx := valueOverlayContext{Context: m.baseCtx, values: routeValues}
	if spec.ClientOperationID != "" {
		if spec.RequestFingerprint == "" {
			return Job{}, ErrIdempotencyConflict
		}
		if prior, err := m.st.getByClientOperation(storeCtx, spec.ClientOperationID); err == nil {
			if prior.RequestFingerprint != spec.RequestFingerprint {
				return Job{}, ErrIdempotencyConflict
			}
			return prior, nil
		}
	}

	lane := normalizeLane(spec.Lane)
	now := m.clock.Now()
	job := Job{
		ID:                      uuid.NewString(),
		Operation:               spec.Operation,
		Lane:                    lane,
		Priority:                normalizePriority(spec.Priority),
		State:                   StateQueued,
		Payload:                 spec.Payload,
		ClientOperationID:       spec.ClientOperationID,
		RequestFingerprint:      spec.RequestFingerprint,
		TestMode:                testMode,
		ExecutionTimeoutSeconds: spec.ExecutionTimeoutSeconds,
		EstimatedSeconds:        spec.EstimatedSeconds,
		CreatedAt:               now,
	}
	// Persist on the server-lifetime context while retaining only the pinned
	// routing values, so disconnects do not cancel or redirect the job.
	if err := m.st.insert(storeCtx, job); err != nil {
		if spec.ClientOperationID != "" {
			if prior, lookupErr := m.st.getByClientOperation(storeCtx, spec.ClientOperationID); lookupErr == nil {
				if prior.RequestFingerprint != spec.RequestFingerprint {
					return Job{}, ErrIdempotencyConflict
				}
				return prior, nil
			}
		}
		return Job{}, err
	}

	entry := &jobEntry{job: job, routeValues: routeValues, done: make(chan struct{}), runner: run}
	m.mu.Lock()
	m.entries[job.ID] = entry
	if !m.started || m.baseCtx.Err() != nil {
		m.mu.Unlock()
		_ = m.transition(entry, StateCanceled, "server shutting down")
		m.finish(entry)
		return job, nil
	}
	if len(m.pending) >= queueCapacity {
		m.mu.Unlock()
		entry.job.Error = "scheduler queue is full"
		_ = m.transition(entry, StateFailed, entry.job.Error)
		m.finish(entry)
		return Job{}, ErrQueueFull
	}
	m.pending = append(m.pending, entry)
	m.mu.Unlock()
	m.signal()
	return job, nil
}

// GetByClientOperation returns a durable job by its caller-owned idempotency key.
func (m *Manager) GetByClientOperation(ctx context.Context, operationID string) (Job, error) {
	if operationID == "" {
		return Job{}, ErrNotFound
	}
	job, err := m.st.getByClientOperation(ctx, operationID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Job{}, fmt.Errorf("%w: %s", ErrNotFound, operationID)
		}
		return Job{}, err
	}
	return m.Get(ctx, job.ID)
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
		ID:                      uuid.NewString(),
		Operation:               spec.Operation,
		Lane:                    lane,
		Payload:                 spec.Payload,
		EstimatedSeconds:        spec.EstimatedSeconds,
		ExecutionTimeoutSeconds: spec.ExecutionTimeoutSeconds,
		CreatedAt:               now,
		StartedAt:               &now,
		FinishedAt:              &now,
		Progress:                100,
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

func (m *Manager) execute(entry *jobEntry) {
	defer m.wg.Done()
	defer m.finishedSlot(entry)
	entry.mu.Lock()
	if entry.job.State.Terminal() {
		entry.mu.Unlock()
		return
	}
	jobCtx, cancel := context.WithCancel(m.baseCtx)
	entry.cancel = cancel
	transitionErr := entry.job.transition(StateAdmitting, m.clock.Now(), "waiting for capacity")
	if transitionErr == nil {
		transitionErr = m.persistLocked(entry)
	}
	snapshot := entry.job
	entry.mu.Unlock()
	defer cancel()
	jobCtx = valueOverlayContext{Context: jobCtx, values: entry.routeValues}
	if transitionErr != nil {
		m.complete(entry, Result{}, transitionErr, jobCtx.Err() != nil)
		return
	}
	requestCtx := jobCtx
	admission, err := m.acquire(jobCtx, snapshot)
	if admission.Context != nil {
		jobCtx = admission.Context
	}
	// Cleanup is cancellation-independent and precedes completion even when the
	// runner panics, admission fails partway, or cancellation wins a result race.
	if err == nil && jobCtx.Err() == nil {
		err = m.transition(entry, StateRunning, "started")
	}
	var result Result
	if err == nil && jobCtx.Err() == nil {
		runCtx := jobCtx
		if snapshot.ExecutionTimeoutSeconds > 0 {
			var stop context.CancelFunc
			runCtx, stop = context.WithTimeout(jobCtx, time.Duration(snapshot.ExecutionTimeoutSeconds)*time.Second)
			defer stop()
		}
		run := m.runner
		if entry.runner != nil {
			run = entry.runner
		}
		result, err = runSafely(runCtx, snapshot, func(progress int, message string) { m.progress(entry, progress, message) }, run)
		if runCtx.Err() != nil {
			err = errors.Join(err, runCtx.Err())
		}
	}
	wasCanceled := requestCtx.Err() != nil
	if jobCtx.Err() != nil {
		err = errors.Join(err, context.Cause(jobCtx))
	}
	retry := errors.Is(err, ErrCapacityBusy)
	if admission.Release != nil {
		cleanup, stop := context.WithTimeout(context.WithoutCancel(jobCtx), 10*time.Second)
		releaseErr := admission.Release(cleanup)
		stop()
		if releaseErr != nil {
			retry = false
			err = errors.Join(err, ErrReleaseFailed, fmt.Errorf("release job capacity: %w", releaseErr))
		}
	}
	if retry && !wasCanceled {
		if m.requeue(entry, err.Error()) {
			return
		}
	}
	if len(admission.Meta) > 0 {
		if result.Meta == nil {
			result.Meta = map[string]string{}
		}
		for key, value := range admission.Meta {
			result.Meta[key] = value
		}
	}
	m.complete(entry, result, err, wasCanceled)
}

func runSafely(ctx context.Context, job Job, emit func(int, string), run Runner) (result Result, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("job runner panicked: %v", recovered)
		}
	}()
	return run(ctx, job, emit)
}

// transition serializes the state change and its durable write against progress
// and cancellation. Active work never runs after a failed transition write.
func (m *Manager) transition(entry *jobEntry, next State, message string) error {
	entry.mu.Lock()
	defer entry.mu.Unlock()
	if err := entry.job.transition(next, m.clock.Now(), message); err != nil {
		return err
	}
	return m.persistLocked(entry)
}

func (m *Manager) persistLocked(entry *jobEntry) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(m.baseCtx), 5*time.Second)
	defer cancel()
	err := m.st.update(valueOverlayContext{Context: ctx, values: entry.routeValues}, entry.job)
	entry.persistErr = err
	ev := ProgressEvent{JobID: entry.job.ID, State: entry.job.State, Progress: entry.job.Progress, Message: entry.job.Message, At: m.clock.Now()}
	entry.last = &ev
	for _, sub := range entry.subs {
		if !sub.closed {
			trySend(sub.ch, ev)
		}
	}
	return err
}

func (m *Manager) progress(entry *jobEntry, progress int, message string) {
	entry.mu.Lock()
	defer entry.mu.Unlock()
	if entry.job.State != StateRunning {
		return
	}
	entry.job.Progress = max(0, min(100, progress))
	entry.job.Message = message
	_ = m.persistLocked(entry)
}

func (m *Manager) complete(entry *jobEntry, result Result, runErr error, canceled bool) {
	entry.mu.Lock()
	if entry.job.State.Terminal() {
		entry.mu.Unlock()
		return
	}
	state, message := StateSucceeded, "completed"
	switch {
	case errors.Is(runErr, ErrReleaseFailed):
		state, message = StateFailed, runErr.Error()
		entry.job.Error = message
	case errors.Is(runErr, context.DeadlineExceeded):
		state, message = StateFailed, "execution time limit exceeded"
		entry.job.Error = message
	case canceled || entry.job.State == StateCanceling:
		state, message = StateCanceled, "canceled"
	case runErr != nil:
		state, message = StateFailed, runErr.Error()
		entry.job.Error = message
	}
	if state == StateSucceeded {
		entry.job.ResultRef, entry.job.Meta, entry.job.Progress = result.Ref, result.Meta, 100
	} else {
		entry.job.Meta, entry.job.ResultRef = failedExecutionEvidence(result.Meta), ""
	}
	if err := entry.job.transition(state, m.clock.Now(), message); err != nil {
		entry.persistErr = errors.Join(entry.persistErr, err)
	} else {
		_ = m.persistLocked(entry)
	}
	entry.mu.Unlock()
	m.finish(entry)
}

func (m *Manager) finish(entry *jobEntry) {
	entry.mu.Lock()
	final := entry.job
	entry.mu.Unlock()
	m.finalizeEntry(entry, final)
}

// failedExecutionEvidence narrows failed/canceled runner metadata to the one
// additive D33 contract key. Validate its envelope and reject success-only
// fields so backend cost or output metadata cannot leak into an incomplete job.
func failedExecutionEvidence(meta map[string]string) map[string]string {
	out := gatewayFailureEvidence(meta)
	raw := strings.TrimSpace(meta[MetaProvisioningEvidence])
	if raw != "" {
		var report struct {
			SchemaVersion int   `json:"schema_version"`
			Ready         *bool `json:"ready"`
		}
		if json.Unmarshal([]byte(raw), &report) == nil && report.SchemaVersion == 1 && report.Ready != nil && !*report.Ready {
			if out == nil {
				out = map[string]string{}
			}
			out[MetaProvisioningEvidence] = raw
		}
	}
	return out
}

func gatewayFailureEvidence(meta map[string]string) map[string]string {
	raw := strings.TrimSpace(meta[MetaGatewayExecutionEvidence])
	if raw == "" {
		return nil
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
		return nil
	}
	for _, forbidden := range []string{"cost_usd", "cost_provenance", "result_ref", "result_refs"} {
		if _, exists := envelope[forbidden]; exists {
			return nil
		}
	}
	var header struct {
		SchemaVersion int  `json:"schema_version"`
		Complete      bool `json:"complete"`
	}
	if err := json.Unmarshal([]byte(raw), &header); err != nil || header.SchemaVersion != 1 || header.Complete {
		return nil
	}
	return map[string]string{MetaGatewayExecutionEvidence: raw}
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

func trySend(ch chan ProgressEvent, ev ProgressEvent) {
	select {
	case ch <- ev:
	default:
		// Slow subscriber: drop this intermediate event. The terminal event
		// and Get/Wait still reflect the true final state.
	}
}

// Cancel marks active jobs canceling and signals their execution context.
// Wait remains attached until resource cleanup finishes. Queued jobs own no
// claims and can become canceled immediately.
func (m *Manager) Cancel(id string) error {
	m.mu.Lock()
	entry, ok := m.entries[id]
	m.mu.Unlock()
	if !ok {
		return fmt.Errorf("%w: %s", ErrNotFound, id)
	}
	entry.mu.Lock()
	if entry.job.State.Terminal() || entry.job.State == StateCanceling {
		entry.mu.Unlock()
		return nil
	}
	active := entry.cancel != nil
	state := StateCanceled
	if active {
		state = StateCanceling
	}
	err := entry.job.transition(state, m.clock.Now(), "canceled")
	if err == nil {
		err = m.persistLocked(entry)
	}
	cancel := entry.cancel
	entry.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if !active {
		m.finish(entry)
		m.signal()
	}
	return err
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
		j, err := entry.job, entry.persistErr
		entry.mu.Unlock()
		return j, err
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
		if errors.Is(err, sql.ErrNoRows) {
			return Job{}, fmt.Errorf("%w: %s", ErrNotFound, id)
		}
		return Job{}, err
	}
	return j, nil
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

// ListManagedOutputs returns every successful job with a managed out/ blob,
// oldest first, bounded by limit. It feeds the owner retention loop, which must
// see every reference to a key before deciding the key is reclaimable.
func (m *Manager) ListManagedOutputs(ctx context.Context, limit int) ([]Job, error) {
	return m.st.listManagedOutputs(ctx, limit)
}

// ListActive returns queued and running jobs. Retention uses it to protect
// blobs an in-flight job consumes as input.
func (m *Manager) ListActive(ctx context.Context) ([]Job, error) {
	return m.st.listNonTerminal(ctx)
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
