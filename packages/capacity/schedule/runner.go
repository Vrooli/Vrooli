package schedule

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Result is what an executor reports back: actual cost and resumable progress.
type Result struct {
	ActualSeconds float64
	Items         int
	Tokens        int
	Checkpoint    string
}

// Executor runs one job. It must honour ctx cancellation and must not assume it
// is the only job running in the process; the scheduler serializes batches, but
// nothing stops an operator from running two schedulers against one database.
type Executor interface {
	Execute(ctx context.Context, job Job) (Result, error)
}

// ExecutorFunc adapts a function to Executor.
type ExecutorFunc func(ctx context.Context, job Job) (Result, error)

// Execute implements Executor.
func (f ExecutorFunc) Execute(ctx context.Context, job Job) (Result, error) { return f(ctx, job) }

// Checkpointer persists progress for a running job. Saving is durable: a crash
// immediately after Save leaves that checkpoint on the job.
type Checkpointer interface {
	Save(ctx context.Context, checkpoint string) error
}

// CheckpointExecutor is implemented by executors that checkpoint mid-run. The
// scheduler detects it by type assertion; plain Executors still work, they just
// checkpoint only at completion.
type CheckpointExecutor interface {
	ExecuteWithCheckpoint(ctx context.Context, job Job, checkpoint Checkpointer) (Result, error)
}

// SchedulerConfig configures a Scheduler.
type SchedulerConfig struct {
	// Store is required.
	Store *Store
	// Policy is normalized; zero values become documented defaults.
	Policy Policy
	// Admission is required. Production use wraps packages/capacity.Broker with
	// NewBrokerAdmission.
	Admission Admission
	// Executor is required.
	Executor Executor
	// Observer is required for idle mode and for RequireIdle. In production use
	// BrokerObserver over the capacity broker.
	Observer HostObserver
	// LeaseOwner identifies this runner on job leases. Defaults to a random id.
	LeaseOwner string
	// LeaseTTL overrides Policy.LeaseTTL.
	LeaseTTL time.Duration
	// Interval overrides Policy.Interval for Run.
	Interval time.Duration
}

// Scheduler selects, admits, executes and records deferred work.
type Scheduler struct {
	store      *Store
	policy     Policy
	admission  Admission
	executor   Executor
	observer   HostObserver
	leaseOwner string
	leaseTTL   time.Duration
	interval   time.Duration

	mu sync.Mutex
}

// New builds a scheduler. The store, admission and executor are required; a
// scheduler that cannot admit or cannot execute is a configuration bug, not a
// degraded mode.
func New(cfg SchedulerConfig) (*Scheduler, error) {
	if cfg.Store == nil {
		return nil, errors.New("schedule: scheduler requires a store")
	}
	if cfg.Admission == nil {
		return nil, errors.New("schedule: scheduler requires an admission")
	}
	if cfg.Executor == nil {
		return nil, errors.New("schedule: scheduler requires an executor")
	}
	policy, err := cfg.Policy.Normalized()
	if err != nil {
		return nil, err
	}
	if cfg.LeaseTTL > 0 {
		policy.LeaseTTL = cfg.LeaseTTL
	}
	if cfg.Interval > 0 {
		policy.Interval = cfg.Interval
	}
	if policy.Mode == ModeIdle && cfg.Observer == nil {
		return nil, errors.New("schedule: idle mode requires a host observer")
	}
	if policy.RequireIdle && cfg.Observer == nil {
		return nil, errors.New("schedule: RequireIdle requires a host observer")
	}
	owner := strings.TrimSpace(cfg.LeaseOwner)
	if owner == "" {
		owner = NewID("scheduler")
	}
	return &Scheduler{
		store:      cfg.Store,
		policy:     policy,
		admission:  cfg.Admission,
		executor:   cfg.Executor,
		observer:   cfg.Observer,
		leaseOwner: owner,
		leaseTTL:   policy.LeaseTTL,
		interval:   policy.Interval,
	}, nil
}

// Policy returns the normalized policy in use.
func (s *Scheduler) Policy() Policy { return s.policy }

// LeaseOwner returns the identity written on this runner's leases.
func (s *Scheduler) LeaseOwner() string { return s.leaseOwner }

// TickResult explains what one tick did. Gate is set whenever the tick stopped
// early or had nothing to do; it is the answer to "why is my queue not moving".
type TickResult struct {
	At time.Time
	// Gate explains why the tick stopped or had nothing to do. It is the
	// answer to "why is my queue not moving".
	Gate string
	// Warnings records non-terminal notes: a degraded admission, a group
	// skipped for the model-load budget, a lease race lost to another runner.
	Warnings  []string
	Recovered int
	Expired   int
	Ran       int
	Deferred  int
	// Failed counts failed execution attempts. A retry increments both Failed
	// and Deferred.
	Failed   int
	Executed []string
}

// Tick runs one selection pass. It is safe to call concurrently with itself
// (calls serialize) and with Status.
func (s *Scheduler) Tick(ctx context.Context) (TickResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.tick(ctx)
}

func (s *Scheduler) tick(ctx context.Context) (TickResult, error) {
	result := TickResult{At: s.store.now()}
	if s.policy.Mode == ModeOff {
		result.Gate = "mode: off"
		return result, nil
	}

	recovered, err := s.store.ReapExpiredLeases(ctx)
	if err != nil {
		return result, err
	}
	result.Recovered = len(recovered)
	expired, err := s.store.ExpireOverdue(ctx)
	if err != nil {
		return result, err
	}
	result.Expired = expired

	gate, err := s.gate(ctx)
	if err != nil {
		return result, err
	}
	if gate != "" {
		result.Gate = gate
		return result, nil
	}

	now := s.store.now()
	ready, err := s.store.Ready(ctx, 0)
	if err != nil {
		return result, err
	}
	budget, err := s.store.budgetSnapshot(ctx, now)
	if err != nil {
		return result, err
	}
	ready = applyShareGate(ready, s.policy, budget)
	ready = readyForMode(ready, s.policy.Mode)
	if len(ready) == 0 {
		result.Gate = "queue: no ready jobs"
		return result, nil
	}

	groups := groupCandidates(ready)
	hot, err := s.store.HotModels(ctx, s.policy.ModelDwell)
	if err != nil {
		return result, err
	}
	markHot(groups, hot)
	rankCandidates(groups, budget)

	for index, group := range groups {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		budget, err = s.store.budgetSnapshot(ctx, s.store.now())
		if err != nil {
			return result, err
		}
		if reason := s.budgetGate(budget); reason != "" {
			result.Gate = reason
			break
		}
		if shareBlocksGroup(group, groups[index:], s.policy, budget) {
			result.Warnings = append(result.Warnings, "fair-share: owner "+group.Key.Owner+" is at its share; yielding the tick")
			continue
		}
		if s.policy.MaxModelLoadsPerHour > 0 && !group.Hot && budget.ModelLoads >= s.policy.MaxModelLoadsPerHour {
			result.Warnings = append(result.Warnings, "budget: model loads per hour exhausted ("+group.Key.Model+")")
			continue
		}

		limit := s.policy.maxBatchFor(group.Jobs)
		if s.policy.MaxJobsPerHour > 0 {
			remainingJobs := s.policy.MaxJobsPerHour - budget.Jobs
			if remainingJobs <= 0 {
				result.Gate = "budget: jobs per hour exhausted"
				break
			}
			if remainingJobs < limit {
				limit = remainingJobs
			}
		}
		batch := trimBatch(group.Jobs, limit, secondsRemaining(s.policy, budget))
		if len(batch) == 0 {
			result.Gate = "budget: seconds per hour exhausted"
			break
		}

		note, stop, err := s.dispatch(ctx, batch, &result)
		if err != nil {
			return result, err
		}
		if note != "" {
			result.Warnings = append(result.Warnings, note)
		}
		if stop {
			result.Gate = note
			break
		}
	}
	if result.Gate == "" && result.Ran == 0 && result.Deferred == 0 {
		result.Gate = "queue: nothing runnable"
	}
	return result, nil
}

// gate evaluates mode, idle and window gating. It returns a reason when the tick
// must not select, and "" when selection may proceed. A gating error (an
// unreadable budget, a failed observation) is returned, never turned into a
// silent "nothing to do".
func (s *Scheduler) gate(ctx context.Context) (string, error) {
	now := s.store.now()
	checksIdle := s.policy.Mode == ModeIdle || s.policy.RequireIdle
	if checksIdle {
		state, err := s.observer.Observe(ctx)
		if err != nil {
			return "idle: host observation unavailable: " + err.Error(), nil
		}
		if !s.policy.idleObservation(&state) {
			return fmt.Sprintf("idle: host busy (load1=%.2f availableRam=%d)", state.Load1, state.AvailableRAMBytes), nil
		}
	}
	checksWindows := s.policy.Mode == ModeIdle || s.policy.Mode == ModeBalanced
	if checksWindows && !s.policy.windowsOpen(now) {
		return "window: closed", nil
	}
	if s.policy.Mode == ModePassive {
		// Explicit operator work bypasses budgets: the operator asked for it now.
		return "", nil
	}
	budget, err := s.store.budgetSnapshot(ctx, now)
	if err != nil {
		return "", fmt.Errorf("read budget for gating: %w", err)
	}
	return s.budgetGate(budget), nil
}

// budgetGate reports the first exhausted hard budget, if any. Model loads are
// deliberately absent: a batch on an already-resident model costs no load, so the
// per-group check decides that budget instead of blocking the whole queue.
func (s *Scheduler) budgetGate(budget budgetSnapshot) string {
	if s.policy.MaxSecondsPerHour > 0 && budget.Seconds >= s.policy.MaxSecondsPerHour {
		return "budget: seconds per hour exhausted"
	}
	if s.policy.MaxJobsPerHour > 0 && budget.Jobs >= s.policy.MaxJobsPerHour {
		return "budget: jobs per hour exhausted"
	}
	return ""
}

// dispatch admits and runs one batch. stop reports that the tick must end
// (capacity was refused, so lower-value groups will be refused too).
func (s *Scheduler) dispatch(ctx context.Context, batch []Job, result *TickResult) (gate string, stop bool, err error) {
	ids := make([]string, len(batch))
	for i, job := range batch {
		ids[i] = job.ID
	}
	request := AdmissionRequest{
		JobID:         strings.Join(ids, ","),
		Kind:          batch[0].Kind,
		Owner:         batch[0].Owner,
		ResourceClass: batch[0].ResourceClass,
		Estimate:      aggregateEstimate(batch),
		Priority:      batch[0].Priority,
		NoReclaim:     true,
	}
	lease, verdict, err := s.admission.Acquire(ctx, request)
	if err != nil {
		if lease != nil {
			releaseLease(lease)
		}
		return "", true, fmt.Errorf("admit batch: %w", err)
	}
	if !verdict.Granted() {
		if lease != nil {
			releaseLease(lease)
		}
		s.deferBatch(ctx, batch, verdictReason(verdict), result)
		return "admission: " + verdict.Kind, true, nil
	}
	if lease == nil {
		lease = releasedLease{}
	}
	defer releaseLease(lease)

	leased, err := s.store.Lease(ctx, LeaseOptions{
		JobIDs:     ids,
		Owner:      s.leaseOwner,
		TTL:        s.leaseTTL,
		ModelDwell: s.policy.ModelDwell,
	})
	if err != nil {
		if errors.Is(err, ErrNotLeasable) || errors.Is(err, ErrJobNotFound) {
			// Another runner leased the batch between selection and this call.
			// Losing that race is expected, not an infrastructure failure.
			return "lease: contended", true, nil
		}
		return "", true, err
	}
	for _, item := range leased {
		if err := s.runJob(ctx, item, lease, result); err != nil {
			return "", true, err
		}
	}
	if verdict.Kind == VerdictDegrade {
		return "admission: degraded", false, nil
	}
	return "", false, nil
}

func (s *Scheduler) runJob(ctx context.Context, item LeasedJob, admissionLease Lease, result *TickResult) error {
	execCtx, cancelExec := context.WithCancel(ctx)
	claim := item.Job.LeaseClaim()
	stop := s.startRenewals(execCtx, claim, admissionLease, cancelExec)
	defer func() {
		stop()
		cancelExec()
	}()

	// Bookkeeping must outlive caller cancellation: once a job has been admitted
	// and executed, its outcome and its released capacity cannot depend on
	// whether the process was asked to stop mid-run.
	bookkeeping := func() (context.Context, context.CancelFunc) {
		return context.WithTimeout(context.WithoutCancel(execCtx), bookkeepingTimeout)
	}

	outcome, execErr := s.execute(execCtx, item.Job)
	if execErr != nil {
		result.Failed++
		if item.Job.Attempts >= item.Job.MaxAttempts {
			recordCtx, cancel := bookkeeping()
			defer cancel()
			_, err := s.store.Complete(recordCtx, Completion{
				JobID:         item.Job.ID,
				Lease:         claim,
				Outcome:       StateFailed,
				ActualSeconds: outcome.ActualSeconds,
				Items:         outcome.Items,
				Tokens:        outcome.Tokens,
				Checkpoint:    outcome.Checkpoint,
				Reason:        "execution failed: " + execErr.Error(),
			})
			if err != nil {
				return err
			}
			return nil
		}
		recordCtx, cancel := bookkeeping()
		defer cancel()
		_, err := s.store.Defer(recordCtx, DeferOptions{
			JobID:         item.Job.ID,
			Lease:         claim,
			Reason:        "execution failed: " + execErr.Error(),
			Backoff:       s.policy.backoff,
			ActualSeconds: outcome.ActualSeconds,
			Items:         outcome.Items,
			Tokens:        outcome.Tokens,
		})
		if err != nil {
			return err
		}
		result.Deferred++
		return nil
	}
	recordCtx, cancel := bookkeeping()
	defer cancel()
	_, err := s.store.Complete(recordCtx, Completion{
		JobID:         item.Job.ID,
		Lease:         claim,
		Outcome:       StateCompleted,
		ActualSeconds: outcome.ActualSeconds,
		Items:         outcome.Items,
		Tokens:        outcome.Tokens,
		Checkpoint:    outcome.Checkpoint,
	})
	if err != nil {
		return err
	}
	result.Ran++
	result.Executed = append(result.Executed, item.Job.ID)
	return nil
}

func (s *Scheduler) deferBatch(ctx context.Context, batch []Job, reason string, result *TickResult) {
	for _, job := range batch {
		if _, err := s.store.Defer(ctx, DeferOptions{
			JobID:   job.ID,
			Reason:  reason,
			Backoff: s.policy.backoff,
		}); err != nil {
			result.Gate = "defer failed: " + err.Error()
			return
		}
		result.Deferred++
	}
}

func verdictReason(verdict Verdict) string {
	if strings.TrimSpace(verdict.Reason) == "" {
		return "admission " + verdict.Kind
	}
	return "admission " + verdict.Kind + ": " + verdict.Reason
}

// execute runs the executor with panic containment: a panicking batch job must
// not take down the scheduler loop and must still release its lease.
func (s *Scheduler) execute(ctx context.Context, job Job) (outcome Result, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("executor panicked: %v", recovered)
		}
	}()
	if checkpointer, ok := s.executor.(CheckpointExecutor); ok {
		return checkpointer.ExecuteWithCheckpoint(ctx, job, storeCheckpointer{store: s.store, claim: job.LeaseClaim()})
	}
	return s.executor.Execute(ctx, job)
}

type storeCheckpointer struct {
	store *Store
	claim LeaseClaim
}

func (c storeCheckpointer) Save(ctx context.Context, checkpoint string) error {
	return c.store.CheckpointLeased(ctx, c.claim, checkpoint)
}

// startRenewals keeps the job lease and (when renewable) the admission lease
// alive while execution runs. Losing either lease cancels execution: continuing
// to run would consume capacity the broker no longer attributes to this job. It
// returns an idempotent stop function.
func (s *Scheduler) startRenewals(ctx context.Context, claim LeaseClaim, admissionLease Lease, lost context.CancelFunc) func() {
	if s.leaseTTL <= 0 {
		return func() {}
	}
	interval := s.leaseTTL / 3
	if interval <= 0 {
		interval = time.Second
	}
	renewable, _ := admissionLease.(RenewableLease)
	done := make(chan struct{})
	var once sync.Once
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				renewCtx := context.WithoutCancel(ctx)
				if _, err := s.store.HeartbeatLease(renewCtx, claim, s.leaseTTL); err != nil {
					lost()
					return
				}
				if renewable != nil {
					if err := renewable.Heartbeat(renewCtx); err != nil {
						lost()
						return
					}
				}
			}
		}
	}()
	return func() {
		once.Do(func() { close(done) })
		wg.Wait()
	}
}

// releaseLease releases an admission lease with a bounded, cancellation-proof
// context: releasing capacity must not be skipped because the caller's context
// ended.
func releaseLease(lease Lease) {
	if lease == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(context.Background()), 5*time.Second)
	defer cancel()
	_ = lease.Release(ctx)
}

// Run ticks until ctx is canceled. It returns ctx.Err() on shutdown and any
// infrastructure error that makes further ticks meaningless.
func (s *Scheduler) Run(ctx context.Context) error {
	interval := s.interval
	if interval <= 0 {
		interval = DefaultInterval
	}
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
		if _, err := s.Tick(ctx); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return err
		}
		timer.Reset(interval)
	}
}

// Status reports the current queue, budget, gate and last-run view.
func (s *Scheduler) Status(ctx context.Context) (Status, error) {
	status, err := s.store.Status(ctx, StatusOptions{Policy: s.policy})
	if err != nil {
		return Status{}, err
	}
	if s.observer != nil {
		if state, observeErr := s.observer.Observe(ctx); observeErr == nil {
			status.Idle = s.idleStatus(&state)
		} else {
			status.Idle = IdleStatus{Required: s.policy.Mode == ModeIdle || s.policy.RequireIdle, Reason: observeErr.Error()}
		}
	} else {
		status.Idle = s.idleStatus(nil)
	}
	return status, nil
}

func (s *Scheduler) idleStatus(state *HostState) IdleStatus {
	required := s.policy.Mode == ModeIdle || s.policy.RequireIdle
	status := IdleStatus{Required: required}
	if state != nil {
		status.Observed = true
		status.Load1 = state.Load1
		status.AvailableRAMBytes = state.AvailableRAMBytes
	}
	if state == nil {
		status.Reason = "no host observation"
		return status
	}
	status.Idle = s.policy.idleObservation(state)
	if !status.Idle {
		status.Reason = fmt.Sprintf("load1=%.2f availableRam=%d", state.Load1, state.AvailableRAMBytes)
	}
	return status
}
