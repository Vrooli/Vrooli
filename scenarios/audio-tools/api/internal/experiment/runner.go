//go:build audio_lab

package experiment

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/vrooli/api-core/schedule"
)

var (
	ErrQueueFull  = errors.New("experiment: queue full")
	ErrNotStarted = errors.New("experiment: manager not started")
)

const (
	queueCap          = 64
	closeWaitTimeout  = 5 * time.Second
	cancelWaitTimeout = 2 * time.Second
)

// Logger is the narrow logging seam the Manager uses to surface persistence
// failures it cannot return to a caller (they happen on the background worker).
type Logger interface {
	Printf(format string, args ...any)
}

type noopLogger struct{}

func (noopLogger) Printf(string, ...any) {}

// Manager owns server-lifetime experiment execution. Work survives client
// disconnects because Submit persists and workers run under baseCtx.
type Manager struct {
	service *Service
	runner  Runner
	clock   schedule.Clock
	logger  Logger

	baseCtx context.Context
	cancel  context.CancelFunc

	mu      sync.Mutex
	entries map[string]*entry
	queued  []*entry
	queue   chan *entry
	started bool
	wg      sync.WaitGroup
}

type subscriber struct {
	ch     chan ProgressEvent
	closed bool
}

type entry struct {
	id         string
	mu         sync.Mutex
	exp        Experiment
	cancel     context.CancelFunc
	maxRuntime time.Duration
	done       chan struct{}
	subs       []*subscriber
	last       *ProgressEvent
}

// Config configures an experiment Manager.
type Config struct {
	Service *Service
	Runner  Runner
	Clock   schedule.Clock
	Logger  Logger
}

func NewManager(cfg Config) *Manager {
	clk := cfg.Clock
	if clk == nil {
		clk = schedule.System()
	}
	lg := cfg.Logger
	if lg == nil {
		lg = noopLogger{}
	}
	return &Manager{
		service: cfg.Service,
		runner:  cfg.Runner,
		clock:   clk,
		logger:  lg,
		entries: make(map[string]*entry),
		queue:   make(chan *entry, queueCap),
	}
}

func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.started {
		m.mu.Unlock()
		return nil
	}
	if m.service == nil {
		m.mu.Unlock()
		return fmt.Errorf("experiment: service is required")
	}
	if m.runner == nil {
		m.mu.Unlock()
		return fmt.Errorf("experiment: runner is required")
	}
	m.baseCtx, m.cancel = context.WithCancel(ctx)
	m.mu.Unlock()

	if err := m.recover(); err != nil {
		m.cancel()
		return err
	}

	m.mu.Lock()
	m.started = true
	m.mu.Unlock()

	m.wg.Add(1)
	go m.worker()
	return nil
}

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
	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(closeWaitTimeout):
		m.logger.Printf("experiment manager close timed out after %s; worker still stopping", closeWaitTimeout)
	}
}

func (m *Manager) recover() error {
	orphans, err := m.service.repo.ListNonTerminal(m.baseCtx)
	if err != nil {
		return err
	}
	now := m.clock.Now().UTC()
	for _, exp := range orphans {
		if err := exp.Transition(StatusFailed); err != nil {
			return fmt.Errorf("experiment: recover %q status: %w", exp.ID, err)
		}
		exp.Error = "interrupted by server restart"
		exp.FinishedAt = &now
		if err := m.service.repo.UpdateExperiment(m.baseCtx, exp); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) Submit(ctx context.Context, spec SubmitSpec) (Experiment, error) {
	m.mu.Lock()
	if !m.started {
		m.mu.Unlock()
		return Experiment{}, ErrNotStarted
	}
	m.mu.Unlock()

	exp := Experiment{
		Name:        spec.Name,
		Status:      StatusQueued,
		RecipeJSON:  spec.RecipeJSON,
		MachineJSON: spec.MachineJSON,
	}
	saved, err := m.service.CreateExperiment(m.baseCtx, exp)
	if err != nil {
		return Experiment{}, err
	}

	e := &entry{id: saved.ID, exp: saved, maxRuntime: spec.MaxRuntime, done: make(chan struct{})}
	m.mu.Lock()
	m.entries[saved.ID] = e
	m.queued = append(m.queued, e)
	ahead := len(m.queued) - 1
	m.mu.Unlock()
	m.emitQueued(e, ahead)

	select {
	case m.queue <- e:
		return saved, nil
	default:
		m.removeQueued(e)
		now := m.clock.Now().UTC()
		if err := saved.Transition(StatusFailed); err != nil {
			m.logger.Printf("experiment %s: transition queue-full status: %v", saved.ID, err)
			return Experiment{}, err
		}
		saved.Error = "experiment queue is full"
		saved.FinishedAt = &now
		if err := m.service.repo.UpdateExperiment(m.baseCtx, saved); err != nil {
			m.logger.Printf("experiment %s: persist queue-full status: %v", saved.ID, err)
		}
		m.finalizeEntry(e, saved)
		return Experiment{}, ErrQueueFull
	}
}

func (m *Manager) worker() {
	defer m.wg.Done()
	for {
		select {
		case <-m.baseCtx.Done():
			return
		case e := <-m.queue:
			m.markDequeued(e)
			m.execute(e)
		}
	}
}

func (m *Manager) execute(e *entry) {
	e.mu.Lock()
	if e.exp.Status.Terminal() {
		e.mu.Unlock()
		return
	}
	var jobCtx context.Context
	var cancel context.CancelFunc
	if e.maxRuntime > 0 {
		jobCtx, cancel = context.WithTimeout(m.baseCtx, e.maxRuntime)
	} else {
		jobCtx, cancel = context.WithCancel(m.baseCtx)
	}
	e.cancel = cancel
	now := m.clock.Now().UTC()
	if err := e.exp.Transition(StatusRunning); err != nil {
		e.mu.Unlock()
		m.logger.Printf("experiment %s: transition to running: %v", e.id, err)
		return
	}
	e.exp.StartedAt = &now
	snapshot := e.exp
	e.mu.Unlock()
	defer cancel()

	result, err := m.runExecution(jobCtx, e, snapshot)

	e.mu.Lock()
	if e.exp.Status.Terminal() {
		e.mu.Unlock()
		return
	}
	finished := m.clock.Now().UTC()
	e.exp.FinishedAt = &finished
	terminal := StatusSucceeded
	switch {
	case errors.Is(jobCtx.Err(), context.DeadlineExceeded):
		terminal = StatusFailed
		e.exp.Error = fmt.Sprintf("experiment exceeded runtime budget of %s", e.maxRuntime)
	case err != nil && jobCtx.Err() != nil:
		terminal = StatusCanceled
	case err != nil:
		terminal = StatusFailed
		e.exp.Error = err.Error()
	case jobCtx.Err() != nil:
		terminal = StatusCanceled
	}
	base := e.exp
	final := base
	if transitionErr := final.Transition(terminal); transitionErr != nil {
		e.mu.Unlock()
		m.logger.Printf("experiment %s: transition to %s: %v", e.id, terminal, transitionErr)
		return
	}

	var persistOK bool
	final, persistOK = m.persistExecutionResult(base, final, result, finished)
	if !persistOK {
		e.mu.Unlock()
		return
	}
	e.exp = final
	e.mu.Unlock()
	m.finalizeEntry(e, final)
}

// runExecution publishes the running state and delegates progress/result
// production while execute owns the job cancellation lifetime.
func (m *Manager) runExecution(jobCtx context.Context, e *entry, snapshot Experiment) (RunResult, error) {
	if err := m.service.repo.UpdateExperiment(m.baseCtx, snapshot); err != nil {
		m.logger.Printf("experiment %s: persist running transition: %v", snapshot.ID, err)
	}
	startedMessage := "started"
	if e.maxRuntime > 0 {
		startedMessage = fmt.Sprintf("started; runtime budget %s", e.maxRuntime)
	}
	m.emit(e, StatusRunning, 0, startedMessage)
	emit := func(progress int, message string) {
		if progress < 0 {
			progress = 0
		}
		if progress > 100 {
			progress = 100
		}
		m.emit(e, StatusRunning, progress, message)
	}
	return m.runner(jobCtx, snapshot, emit)
}

func (m *Manager) persistQualificationEvidence(exp Experiment, items []QualificationEvidence) {
	for _, item := range items {
		if item.ArtifactRef == "" {
			item.ArtifactRef = "experiment:" + exp.ID
		}
		if _, err := m.service.RecordQualificationEvidence(m.baseCtx, item); err != nil {
			m.logger.Printf("experiment %s: persist qualification evidence %q: %v", exp.ID, item.Kind, err)
		}
	}
}

func (m *Manager) finalizeEntry(e *entry, final Experiment) {
	e.mu.Lock()
	e.exp = final
	select {
	case <-e.done:
		e.mu.Unlock()
		return
	default:
	}
	ev := ProgressEvent{ExperimentID: final.ID, Status: final.Status, Progress: terminalProgress(final.Status), Message: finalMessage(final), At: m.clock.Now().UTC()}
	e.last = &ev
	for _, sub := range e.subs {
		if !sub.closed {
			trySend(sub.ch, ev)
			close(sub.ch)
			sub.closed = true
		}
	}
	e.subs = nil
	// Remove the terminal entry before waking Wait callers. Wait's contract is
	// that once it returns, the terminal state is durable and no longer kept
	// resident in this process. Keep e.mu held so a concurrent finalizer cannot
	// observe an open done channel and repeat the eviction.
	m.mu.Lock()
	if m.entries[final.ID] == e {
		delete(m.entries, final.ID)
	}
	m.mu.Unlock()
	close(e.done)
	e.mu.Unlock()
}

func (m *Manager) emit(e *entry, status Status, progress int, message string) {
	ev := ProgressEvent{ExperimentID: e.id, Status: status, Progress: progress, Message: message, At: m.clock.Now().UTC()}
	e.mu.Lock()
	e.last = &ev
	for _, sub := range e.subs {
		if !sub.closed {
			trySend(sub.ch, ev)
		}
	}
	e.mu.Unlock()
}

func trySend(ch chan ProgressEvent, ev ProgressEvent) {
	select {
	case ch <- ev:
	default:
	}
}
