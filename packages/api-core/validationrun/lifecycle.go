package validationrun

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/vrooli/api-core/schedule"
)

// Transition is pure and exhaustively restricts legal lifecycle moves.
func Transition(run Run, event Event, now time.Time) (Run, error) {
	if run.State.Terminal() {
		return Run{}, invalid(event, run.State, "terminal runs cannot transition")
	}
	next := run
	switch event {
	case EventClaim:
		if run.State != StateQueued {
			return Run{}, invalid(event, run.State, "only queued runs can be claimed")
		}
		next.State, next.StartedAt = StateRunning, now
	case EventSucceed:
		if run.State != StateRunning {
			return Run{}, invalid(event, run.State, "only running runs can succeed")
		}
		if run.CancellationRequested {
			return Run{}, invalid(event, run.State, "cancellation was requested; stale completion cannot succeed")
		}
		next.State, next.CompletedAt = StateSucceeded, now
	case EventFail:
		if run.State != StateQueued && run.State != StateRunning {
			return Run{}, invalid(event, run.State, "only queued or running runs can fail")
		}
		next.State, next.CompletedAt = StateFailed, now
	case EventCancel:
		if run.State != StateQueued && run.State != StateRunning {
			return Run{}, invalid(event, run.State, "only queued or running runs can cancel")
		}
		next.State, next.CompletedAt, next.CancellationRequested = StateCanceled, now, true
	case EventRecoveryFailed:
		if run.State != StateQueued && run.State != StateRunning {
			return Run{}, invalid(event, run.State, "only interrupted work can fail recovery")
		}
		next.State, next.CompletedAt = StateRecoveryFailed, now
	case EventRequestAbort:
		if run.State == StateQueued {
			next.State, next.CompletedAt = StateCanceled, now
		}
		next.CancellationRequested = true
	default:
		return Run{}, invalid(event, run.State, "unknown lifecycle event")
	}
	return next, nil
}

func invalid(event Event, state State, detail string) error {
	return &LifecycleError{Code: ErrorInvalidTransition, Operation: string(event), State: state, Cause: errors.New(detail)}
}

type (
	Clock     = schedule.Clock
	RealClock struct{}
)

func (RealClock) Now() time.Time                            { return schedule.System().Now().UTC() }
func (RealClock) NewTimer(d time.Duration) schedule.Timer   { return schedule.System().NewTimer(d) }
func (RealClock) NewTicker(d time.Duration) schedule.Ticker { return schedule.System().NewTicker(d) }
func (RealClock) Sleep(d time.Duration)                     { schedule.System().Sleep(d) }

type IDSource interface{ NewID() string }

type Coordinator struct {
	Repository Repository
	Notifier   Notifier
	Executor   Executor
	Clock      Clock
	IDs        IDSource
}

func (c Coordinator) StartOrGet(ctx context.Context, target Target, key, parentRunID string) (Run, bool, error) {
	if err := target.Validate(); err != nil {
		return Run{}, false, err
	}
	if key == "" {
		return Run{}, false, &LifecycleError{Code: ErrorInvalidTransition, Operation: "start", Cause: errors.New("idempotency key is required")}
	}
	if c.Repository == nil || c.IDs == nil {
		return Run{}, false, errors.New("validation run repository and id source are required")
	}
	if existing, err := c.Repository.FindByIdempotency(ctx, key); err == nil {
		if !existing.Target.Equal(target) {
			return Run{}, false, &LifecycleError{Code: ErrorIdempotencyConflict, Operation: "start", Cause: errors.New("idempotency key belongs to a different target")}
		}
		return existing, false, nil
	} else if !IsCode(err, ErrorNotFound) {
		return Run{}, false, fmt.Errorf("find validation run by idempotency key: %w", err)
	}
	now := c.now()
	run := Run{ID: c.IDs.NewID(), Target: target, IdempotencyKey: key, ParentRunID: parentRunID, State: StateQueued, CreatedAt: now, Version: 1}
	if err := run.Validate(); err != nil {
		return Run{}, false, err
	}
	if err := c.Repository.Create(ctx, run); err != nil {
		if existing, lookupErr := c.Repository.FindByIdempotency(ctx, key); lookupErr == nil {
			if !existing.Target.Equal(target) {
				return Run{}, false, &LifecycleError{Code: ErrorIdempotencyConflict, Operation: "start", Cause: errors.New("idempotency key belongs to a different target")}
			}
			return existing, false, nil
		}
		return Run{}, false, fmt.Errorf("create validation run: %w", err)
	}
	if c.Executor != nil {
		go c.Executor.Run(context.WithoutCancel(ctx), run.ID)
	}
	return run, true, nil
}

// CommitTerminal applies a provider worker's terminal outcome only if it is
// still legal. A stale completion after cancellation is rejected explicitly.
func (c Coordinator) CommitTerminal(ctx context.Context, runID string, event Event) (Run, error) {
	if event != EventSucceed && event != EventFail && event != EventCancel && event != EventRecoveryFailed {
		return Run{}, invalid(event, "", "event is not terminal")
	}
	run, err := c.get(ctx, runID)
	if err != nil {
		return Run{}, err
	}
	next, err := Transition(run, event, c.now())
	if err != nil {
		return Run{}, err
	}
	next.Version = run.Version + 1
	if err := c.Repository.Update(ctx, next, run.Version); err != nil {
		return Run{}, fmt.Errorf("commit validation run terminal state: %w", err)
	}
	return next, nil
}

func (c Coordinator) Abort(ctx context.Context, runID string) (Run, error) {
	run, err := c.get(ctx, runID)
	if err != nil {
		return Run{}, err
	}
	if run.State.Terminal() {
		return run, nil
	}
	next, err := Transition(run, EventRequestAbort, c.now())
	if err != nil {
		return Run{}, err
	}
	next.Version = run.Version + 1
	if err := c.Repository.Update(ctx, next, run.Version); err != nil {
		return Run{}, fmt.Errorf("record validation run abort request: %w", err)
	}
	if c.Executor != nil && next.State != StateCanceled {
		if err := c.Executor.Abort(ctx, runID); err != nil {
			return next, fmt.Errorf("request provider abort: %w", err)
		}
	}
	return next, nil
}

// Wait observes durable provider state without cancelling its worker when the
// caller times out or disconnects.
func (c Coordinator) Wait(ctx context.Context, runID string, timeout time.Duration) (Run, error) {
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	for {
		run, err := c.get(ctx, runID)
		if err != nil {
			return Run{}, err
		}
		if run.State.Terminal() {
			return run, nil
		}
		if c.Notifier == nil {
			return Run{}, errors.New("validation run notifier is required for wait")
		}
		if err := c.Notifier.WaitForChange(ctx, runID, run.Version); err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return Run{}, &LifecycleError{Code: ErrorWaitTimeout, Operation: "wait", State: run.State, Cause: err}
			}
			return Run{}, err
		}
	}
}

func (c Coordinator) get(ctx context.Context, runID string) (Run, error) {
	if c.Repository == nil {
		return Run{}, errors.New("validation run repository is required")
	}
	run, err := c.Repository.Get(ctx, runID)
	if err != nil {
		return Run{}, err
	}
	return run, nil
}

func (c Coordinator) now() time.Time {
	if c.Clock != nil {
		return c.Clock.Now().UTC()
	}
	return schedule.System().Now().UTC()
}
