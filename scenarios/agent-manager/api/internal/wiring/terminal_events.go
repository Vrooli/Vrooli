package wiring

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"agent-manager/internal/adapters/database"
	"agent-manager/internal/runmodel"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/api-core/schedule"
)

// TerminalDispatchBoundary is trusted owner composition, not a boolean or data
// from the retained event. It must hold current publication authority for the
// exact installation/epoch, serializing revocation and replacement with the
// entire synchronous callback. It must call the callback at most once, only
// while authorized, and never retain it. Capture is local prospective evidence;
// revoked publication holds pending receipts rather than changing their bytes.
type TerminalDispatchBoundary interface {
	WithTerminalDispatch(context.Context, runmodel.TerminalReceiptScope, func() error) error
}

// TerminalEventsConfig has no environment/config-file loader or default enabled
// value. Only an explicitly authorized composition can supply these owners.
// Acceptor must be the acknowledged Events adapter, never PublishDomainEvent.
// No constructor starts workers, resolves endpoints, or provisions credentials.
type TerminalEventsConfig struct {
	Scope     runmodel.TerminalReceiptScope
	Acceptor  database.TerminalEventAcceptor
	Boundary  TerminalDispatchBoundary
	BatchSize int
	Interval  time.Duration
}

// TerminalEvents owns lifecycle only. Durable state stays in the existing Run
// receipt and acknowledgment ledgers, and Events owns its acceptance receipt.
type TerminalEvents struct {
	outbox   *database.TerminalOutbox
	scope    runmodel.TerminalReceiptScope
	boundary TerminalDispatchBoundary
	batch    int
	interval time.Duration
	clock    schedule.Clock
	mu       sync.Mutex
	cancel   context.CancelFunc
	done     chan struct{}
	stopped  bool
	status   TerminalEventsStatus
}

// TerminalEventsStatus is a bounded health observation, not event or Work
// acceptance. No source data, credentials, or provider error text are exposed.
type TerminalEventsStatus struct {
	Running      bool
	Attempts     uint64
	Acknowledged int
	State        string
}

// NewTerminalPersistence is shared by the production graph and isolated owner tests.
// Nil config preserves the original repository path and allocates no lifecycle,
// timer, worker, transport, or terminal source binding.
func NewTerminalPersistence(db *database.DB, log *logrus.Logger, cfg *TerminalEventsConfig) (*database.Repositories, *TerminalEvents, error) {
	if cfg == nil {
		return database.NewRepositories(db, log), nil, nil
	}
	if db == nil || log == nil || cfg.Acceptor == nil || cfg.Boundary == nil {
		return nil, nil, errors.New("terminal events require explicit database, logger, acceptor and authority")
	}
	if err := cfg.Scope.Validate(); err != nil {
		return nil, nil, err
	}
	if cfg.BatchSize < 1 || cfg.BatchSize > 100 || cfg.Interval < time.Second || cfg.Interval > time.Hour {
		return nil, nil, errors.New("terminal events require bounded batch and retry interval")
	}
	repos, err := database.NewRepositoriesWithTerminalReceipts(db, log, cfg.Scope)
	if err != nil {
		return nil, nil, err
	}
	outbox, err := database.NewTerminalOutbox(repos, cfg.Acceptor)
	if err != nil {
		return nil, nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := outbox.ValidateSourceBinding(ctx); err != nil {
		return nil, nil, err
	}
	return repos, &TerminalEvents{outbox: outbox, scope: cfg.Scope, boundary: cfg.Boundary, batch: cfg.BatchSize, interval: cfg.Interval, clock: schedule.System(), status: TerminalEventsStatus{State: "not_started"}}, nil
}

// Start is idempotent during a server lifetime and inert on a nil/disabled
// receiver. Restart after Stop requires a newly constructed owner graph.
func (w *TerminalEvents) Start(ctx context.Context) error {
	if w == nil {
		return nil
	}
	if ctx == nil {
		return errors.New("terminal events require lifecycle context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.stopped {
		return errors.New("terminal events lifecycle is stopped")
	}
	if w.done != nil {
		return nil
	}
	ctx, w.cancel = context.WithCancel(ctx)
	w.done = make(chan struct{})
	w.status.Running = true
	go w.run(ctx, w.done)
	return nil
}

func (w *TerminalEvents) run(ctx context.Context, done chan struct{}) {
	defer func() {
		w.mu.Lock()
		w.status.Running = false
		w.status.State = "stopped"
		w.mu.Unlock()
		close(done)
	}()
	ticker := w.clock.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		if ctx.Err() != nil {
			return
		}
		w.dispatch(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C():
		}
	}
}

func (w *TerminalEvents) dispatch(parent context.Context) {
	// Every pass is finite, including authority checks and transport waits.
	ctx, cancel := context.WithTimeout(parent, 15*time.Second)
	defer cancel()
	count, calls := 0, 0
	var callbackMu sync.Mutex
	var returned atomic.Bool
	var callbackErr error
	err := w.boundary.WithTerminalDispatch(ctx, w.scope, func() error {
		callbackMu.Lock()
		defer callbackMu.Unlock()
		calls++
		if calls != 1 || returned.Load() {
			callbackErr = errors.New("terminal authority repeated or retained its callback")
			return callbackErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		var err error
		count, err = w.outbox.Dispatch(ctx, w.batch)
		callbackErr = err
		return err
	})
	returned.Store(true)
	contextErr := ctx.Err()
	cancel()
	callbackMu.Lock()
	defer callbackMu.Unlock()
	state := "idle"
	if err != nil || callbackErr != nil || contextErr != nil || calls != 1 {
		state = "held"
	}
	w.mu.Lock()
	w.status.Attempts++
	w.status.Acknowledged = count
	w.status.State = state
	w.mu.Unlock()
}

// Stop cancels in-flight acceptance and joins before the owning DB may close.
// An uncertain response leaves the original receipt pending for exact replay.
func (w *TerminalEvents) Stop() {
	if w == nil {
		return
	}
	w.mu.Lock()
	w.stopped = true
	cancel, done := w.cancel, w.done
	w.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
}

func (w *TerminalEvents) Status() TerminalEventsStatus {
	if w == nil {
		return TerminalEventsStatus{State: "disabled"}
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.status
}
