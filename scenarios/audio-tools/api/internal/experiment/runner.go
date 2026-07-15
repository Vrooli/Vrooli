package experiment

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"audio-tools/internal/clock"
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
	clock   clock.Clock
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
	Clock   clock.Clock
	Logger  Logger
}

func NewManager(cfg Config) *Manager {
	clk := cfg.Clock
	if clk == nil {
		clk = clock.System{}
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
		exp.Status = StatusFailed
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
		saved.Status = StatusFailed
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
	jobCtx := m.baseCtx
	cancel := func() {}
	if e.maxRuntime > 0 {
		jobCtx, cancel = context.WithTimeout(m.baseCtx, e.maxRuntime)
	} else {
		jobCtx, cancel = context.WithCancel(m.baseCtx)
	}
	e.cancel = cancel
	now := m.clock.Now().UTC()
	e.exp.Status = StatusRunning
	e.exp.StartedAt = &now
	snapshot := e.exp
	e.mu.Unlock()
	defer cancel()

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

	result, err := m.runner(jobCtx, snapshot, emit)

	e.mu.Lock()
	finished := m.clock.Now().UTC()
	e.exp.FinishedAt = &finished
	switch {
	case errors.Is(jobCtx.Err(), context.DeadlineExceeded):
		e.exp.Status = StatusFailed
		e.exp.Error = fmt.Sprintf("experiment exceeded runtime budget of %s", e.maxRuntime)
	case err != nil && jobCtx.Err() != nil:
		e.exp.Status = StatusCanceled
	case err != nil:
		e.exp.Status = StatusFailed
		e.exp.Error = err.Error()
	case jobCtx.Err() != nil:
		e.exp.Status = StatusCanceled
	default:
		e.exp.Status = StatusSucceeded
	}
	final := e.exp
	e.mu.Unlock()

	if final.Status == StatusSucceeded {
		if len(result.RecipeJSON) > 0 {
			final.RecipeJSON = result.RecipeJSON
		}
		if len(result.Report) > 0 {
			mime := result.ReportMIME
			if mime == "" {
				mime = "application/json"
			}
			key, storeErr := m.service.storeReportBlob(m.baseCtx, final, result.Report, mime, finished)
			if storeErr != nil {
				m.logger.Printf("experiment %s: store report: %v", final.ID, storeErr)
				final.Status = StatusFailed
				final.Error = storeErr.Error()
				final.ResultRef = ""
				if err := m.service.repo.UpdateExperiment(m.baseCtx, final); err != nil {
					m.logger.Printf("experiment %s: persist report-failure status: %v", final.ID, err)
				}
			} else {
				final.ResultRef = key
				if err := m.service.repo.CompleteSucceeded(m.baseCtx, final, result.Runs); err != nil {
					_ = m.service.blobs.Delete(m.baseCtx, key)
					m.logger.Printf("experiment %s: persist successful result: %v", final.ID, err)
					final.Status = StatusFailed
					final.Error = fmt.Sprintf("persist successful result: %v", err)
					final.ResultRef = ""
					if updateErr := m.service.repo.UpdateExperiment(m.baseCtx, final); updateErr != nil {
						m.logger.Printf("experiment %s: persist result-failure status: %v", final.ID, updateErr)
					}
				}
			}
		} else {
			if err := m.service.repo.CompleteSucceeded(m.baseCtx, final, result.Runs); err != nil {
				m.logger.Printf("experiment %s: persist successful result: %v", final.ID, err)
				final.Status = StatusFailed
				final.Error = fmt.Sprintf("persist successful result: %v", err)
				if updateErr := m.service.repo.UpdateExperiment(m.baseCtx, final); updateErr != nil {
					m.logger.Printf("experiment %s: persist result-failure status: %v", final.ID, updateErr)
				}
			}
		}
	} else {
		if err := m.service.repo.UpdateExperiment(m.baseCtx, final); err != nil {
			m.logger.Printf("experiment %s: persist terminal status: %v", final.ID, err)
		}
	}
	m.finalizeEntry(e, final)
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
	close(e.done)
	e.mu.Unlock()

	m.mu.Lock()
	if m.entries[final.ID] == e {
		delete(m.entries, final.ID)
	}
	m.mu.Unlock()
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

func (m *Manager) emitQueued(e *entry, ahead int) {
	if ahead < 0 {
		ahead = 0
	}
	message := "queued"
	if ahead == 1 {
		message = "queued; 1 experiment ahead"
	} else if ahead > 1 {
		message = fmt.Sprintf("queued; %d experiments ahead", ahead)
	}
	m.emit(e, StatusQueued, 0, message)
}

func (m *Manager) markDequeued(e *entry) {
	m.removeQueued(e)
	m.emitQueuePositions()
}

func (m *Manager) removeQueued(target *entry) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, e := range m.queued {
		if e == target {
			m.queued = append(m.queued[:i], m.queued[i+1:]...)
			return
		}
	}
}

func (m *Manager) emitQueuePositions() {
	m.mu.Lock()
	queued := append([]*entry(nil), m.queued...)
	m.mu.Unlock()
	for i, e := range queued {
		m.emitQueued(e, i)
	}
}

func trySend(ch chan ProgressEvent, ev ProgressEvent) {
	select {
	case ch <- ev:
	default:
	}
}

func (m *Manager) Get(ctx context.Context, id string) (Experiment, error) {
	m.mu.Lock()
	e, ok := m.entries[id]
	m.mu.Unlock()
	if ok {
		e.mu.Lock()
		exp := e.exp
		e.mu.Unlock()
		return exp, nil
	}
	return m.service.GetExperiment(ctx, id)
}

func (m *Manager) Wait(ctx context.Context, id string) (Experiment, error) {
	m.mu.Lock()
	e, ok := m.entries[id]
	m.mu.Unlock()
	if !ok {
		return m.service.GetExperiment(ctx, id)
	}
	select {
	case <-e.done:
		e.mu.Lock()
		exp := e.exp
		e.mu.Unlock()
		return exp, nil
	case <-ctx.Done():
		return Experiment{}, ctx.Err()
	}
}

func (m *Manager) List(ctx context.Context, filter ListFilter) ([]Experiment, error) {
	return m.service.ListExperiments(ctx, filter)
}

func (m *Manager) Cancel(id string) error {
	m.mu.Lock()
	e, ok := m.entries[id]
	m.mu.Unlock()
	if !ok {
		exp, err := m.service.GetExperiment(context.Background(), id)
		if err != nil {
			return err
		}
		if exp.Status.Terminal() {
			return nil
		}
		now := m.clock.Now().UTC()
		exp.Status = StatusCanceled
		exp.FinishedAt = &now
		return m.service.repo.UpdateExperiment(m.baseCtx, exp)
	}
	e.mu.Lock()
	if e.exp.Status.Terminal() {
		e.mu.Unlock()
		return nil
	}
	if e.exp.Status == StatusRunning && e.cancel != nil {
		cancel := e.cancel
		e.mu.Unlock()
		cancel()
		select {
		case <-e.done:
			return nil
		case <-time.After(cancelWaitTimeout):
			now := m.clock.Now().UTC()
			e.mu.Lock()
			if e.exp.Status.Terminal() {
				e.mu.Unlock()
				return nil
			}
			e.exp.Status = StatusCanceled
			e.exp.FinishedAt = &now
			final := e.exp
			e.mu.Unlock()
			if err := m.service.repo.UpdateExperiment(m.baseCtx, final); err != nil {
				m.logger.Printf("experiment %s: persist cancel-timeout status: %v", final.ID, err)
			}
			m.finalizeEntry(e, final)
		}
		return nil
	}
	now := m.clock.Now().UTC()
	e.exp.Status = StatusCanceled
	e.exp.FinishedAt = &now
	final := e.exp
	e.mu.Unlock()
	if err := m.service.repo.UpdateExperiment(m.baseCtx, final); err != nil {
		m.logger.Printf("experiment %s: persist cancel status: %v", final.ID, err)
	}
	m.finalizeEntry(e, final)
	m.removeQueued(e)
	m.emitQueuePositions()
	return nil
}

func (m *Manager) Subscribe(id string) (<-chan ProgressEvent, func(), error) {
	m.mu.Lock()
	e, ok := m.entries[id]
	m.mu.Unlock()
	if !ok {
		exp, err := m.service.GetExperiment(context.Background(), id)
		if err != nil {
			return nil, nil, err
		}
		ch := make(chan ProgressEvent, 1)
		ch <- ProgressEvent{ExperimentID: exp.ID, Status: exp.Status, Progress: terminalProgress(exp.Status), Message: finalMessage(exp), At: m.clock.Now().UTC()}
		close(ch)
		return ch, func() {}, nil
	}
	sub := &subscriber{ch: make(chan ProgressEvent, 64)}
	e.mu.Lock()
	if e.exp.Status.Terminal() {
		ev := ProgressEvent{ExperimentID: e.exp.ID, Status: e.exp.Status, Progress: terminalProgress(e.exp.Status), Message: finalMessage(e.exp), At: m.clock.Now().UTC()}
		e.mu.Unlock()
		sub.ch <- ev
		close(sub.ch)
		sub.closed = true
		return sub.ch, func() {}, nil
	}
	if e.last != nil {
		trySend(sub.ch, *e.last)
	}
	e.subs = append(e.subs, sub)
	e.mu.Unlock()

	unsub := func() {
		e.mu.Lock()
		defer e.mu.Unlock()
		for i, s := range e.subs {
			if s == sub {
				e.subs = append(e.subs[:i], e.subs[i+1:]...)
				if !s.closed {
					close(s.ch)
					s.closed = true
				}
				return
			}
		}
	}
	return sub.ch, unsub, nil
}

func terminalProgress(s Status) int {
	if s == StatusSucceeded {
		return 100
	}
	return 0
}

func finalMessage(exp Experiment) string {
	switch exp.Status {
	case StatusSucceeded:
		return "succeeded"
	case StatusCanceled:
		return "canceled"
	case StatusFailed:
		return exp.Error
	default:
		return string(exp.Status)
	}
}
