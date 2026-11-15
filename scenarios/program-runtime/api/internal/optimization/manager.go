package optimization

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

const (
	maxActiveRuns       = 1000
	maxAttemptsPerStage = 3
)

// Manager owns the unattended queue. Runs are persisted before being queued;
// startup recovers every nonterminal record and resumes the same run/stage ID.
type Manager struct {
	engine   *Engine
	submitMu sync.Mutex
	mu       sync.Mutex
	queue    chan string
	queued   map[string]bool
	inflight map[string]bool
	waiters  map[string][]chan struct{}
	ctx      context.Context
	cancel   context.CancelFunc
	started  bool
	wg       sync.WaitGroup
	maxWait  time.Duration
	active   map[string]struct{}
	limit    int
}

func NewManager(engine *Engine) (*Manager, error) {
	if engine == nil || engine.Store == nil {
		return nil, fmt.Errorf("optimization engine and durable store are required")
	}
	return &Manager{engine: engine, queue: make(chan string, maxActiveRuns), queued: map[string]bool{}, inflight: map[string]bool{}, waiters: map[string][]chan struct{}{}, maxWait: time.Minute, active: map[string]struct{}{}, limit: maxActiveRuns}, nil
}

func (m *Manager) Start(parent context.Context) error {
	m.mu.Lock()
	if m.started {
		m.mu.Unlock()
		return nil
	}
	m.ctx, m.cancel = context.WithCancel(parent)
	m.started = true
	m.wg.Add(1)
	go m.worker()
	m.mu.Unlock()
	active, err := m.engine.Store.ListActive(m.ctx, m.limit+1)
	if err != nil {
		m.Stop()
		return fmt.Errorf("recover optimization queue: %w", err)
	}
	if len(active) > m.limit {
		m.Stop()
		return fmt.Errorf("optimizer recovery found more than %d active runs; refusing to truncate the durable queue", m.limit)
	}
	m.mu.Lock()
	for _, record := range active {
		m.active[record.Request.RunID] = struct{}{}
	}
	m.mu.Unlock()
	for _, record := range active {
		m.schedule(record)
	}
	return nil
}

func (m *Manager) Stop() {
	// Prevent a concurrent Submit from scheduling retry timers after Wait has
	// observed the worker/timer WaitGroup reach zero.
	m.submitMu.Lock()
	defer m.submitMu.Unlock()

	m.mu.Lock()
	cancel := m.cancel
	m.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	m.wg.Wait()
}

func (m *Manager) Submit(ctx context.Context, request Request) (Record, error) {
	// Keep active-slot reservation and durable Start atomic relative to other
	// submissions. Otherwise a failing same-run caller could release a slot
	// already used by another caller that successfully created the shared run.
	m.submitMu.Lock()
	defer m.submitMu.Unlock()

	m.mu.Lock()
	started := m.started
	_, alreadyActive := m.active[request.RunID]
	reserved := false
	if started && !alreadyActive {
		if len(m.active) >= m.limit {
			m.mu.Unlock()
			return Record{}, ErrQueueFull
		}
		m.active[request.RunID] = struct{}{}
		reserved = true
	}
	m.mu.Unlock()
	if !started {
		return Record{}, fmt.Errorf("optimization manager is not running")
	}
	record, err := m.engine.Start(ctx, request)
	if err != nil {
		if reserved {
			m.mu.Lock()
			delete(m.active, request.RunID)
			m.mu.Unlock()
		}
		return Record{}, err
	}
	if record.Stage == StageComplete || record.Stage == StageRefused {
		m.mu.Lock()
		delete(m.active, request.RunID)
		m.mu.Unlock()
	}
	if record.Stage != StageComplete && record.Stage != StageRefused {
		m.schedule(record)
	}
	return record, nil
}

// Wait blocks server-side until a stage changes, the run becomes terminal, or
// the caller's bounded wait elapses. It makes no client-side poll loop.
func (m *Manager) Wait(ctx context.Context, runID string, maxWait time.Duration) (Record, bool, error) {
	if maxWait < 0 {
		return Record{}, false, fmt.Errorf("wait duration cannot be negative")
	}
	if maxWait > m.maxWait {
		maxWait = m.maxWait
	}
	deadline := time.NewTimer(maxWait)
	defer deadline.Stop()
	for {
		wake := make(chan struct{}, 1)
		m.mu.Lock()
		m.waiters[runID] = append(m.waiters[runID], wake)
		m.mu.Unlock()

		record, err := m.engine.Store.Load(ctx, runID)
		if err != nil {
			m.removeWaiter(runID, wake)
			return Record{}, false, err
		}
		if record.Stage == StageComplete || record.Stage == StageRefused {
			m.removeWaiter(runID, wake)
			return record, true, nil
		}
		select {
		case <-ctx.Done():
			m.removeWaiter(runID, wake)
			return record, false, ctx.Err()
		case <-deadline.C:
			m.removeWaiter(runID, wake)
			return record, false, nil
		case <-wake:
		}
	}
}

func (m *Manager) removeWaiter(runID string, target chan struct{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	waiters := m.waiters[runID]
	kept := waiters[:0]
	for _, waiter := range waiters {
		if waiter != target {
			kept = append(kept, waiter)
		}
	}
	if len(kept) == 0 {
		delete(m.waiters, runID)
	} else {
		m.waiters[runID] = kept
	}
}

func (m *Manager) schedule(record Record) {
	if record.Stage == StageComplete || record.Stage == StageRefused {
		return
	}
	if !record.RetryAt.IsZero() && time.Now().Before(record.RetryAt) {
		delay := time.Until(record.RetryAt)
		m.wg.Add(1)
		go func(id string) {
			defer m.wg.Done()
			timer := time.NewTimer(delay)
			defer timer.Stop()
			select {
			case <-m.ctx.Done():
				return
			case <-timer.C:
				m.enqueue(id)
			}
		}(record.Request.RunID)
		return
	}
	m.enqueue(record.Request.RunID)
}

func (m *Manager) enqueue(runID string) {
	m.mu.Lock()
	if m.queued[runID] || m.inflight[runID] || m.ctx == nil {
		m.mu.Unlock()
		return
	}
	m.queued[runID] = true
	ctx := m.ctx
	m.mu.Unlock()
	select {
	case m.queue <- runID:
	case <-ctx.Done():
		m.mu.Lock()
		delete(m.queued, runID)
		m.mu.Unlock()
	}
}

func (m *Manager) worker() {
	defer m.wg.Done()
	for {
		select {
		case <-m.ctx.Done():
			return
		case runID := <-m.queue:
			m.mu.Lock()
			delete(m.queued, runID)
			if m.inflight[runID] {
				m.mu.Unlock()
				continue
			}
			m.inflight[runID] = true
			m.mu.Unlock()
			record, err := m.engine.Advance(m.ctx, runID)
			if errors.Is(err, ErrRetryNotReady) {
				m.clearInflight(runID)
				m.schedule(record)
				continue
			}
			if errors.Is(err, ErrNoAdapter) || errors.Is(err, ErrOwnerUnavailable) {
				reason := "target owner adapter became unavailable before the operation could run"
				if errors.Is(err, ErrOwnerUnavailable) {
					reason = "target owner operation is not configured; refusing without retry"
				}
				terminal, failErr := m.engine.Fail(m.ctx, runID, reason)
				m.clearInflight(runID)
				if failErr == nil {
					m.notify(terminal)
				}
				continue
			}
			if err != nil {
				record, loadErr := m.engine.Store.Load(m.ctx, runID)
				if loadErr != nil {
					m.clearInflight(runID)
					continue
				}
				attempts := record.StageAttempts[record.Stage]
				if attempts >= maxAttemptsPerStage {
					terminal, failErr := m.engine.Fail(m.ctx, runID, fmt.Sprintf("stage %s failed after %d attempts: %s", record.Stage, attempts, record.LastError))
					m.clearInflight(runID)
					if failErr == nil {
						m.notify(terminal)
					}
					continue
				}
				m.notify(record)
				m.clearInflight(runID)
				m.schedule(record)
				continue
			}
			m.notify(record)
			m.clearInflight(runID)
			m.schedule(record)
		}
	}
}

func (m *Manager) clearInflight(runID string) {
	m.mu.Lock()
	delete(m.inflight, runID)
	m.mu.Unlock()
}

func (m *Manager) notify(record Record) {
	m.mu.Lock()
	if record.Stage == StageComplete || record.Stage == StageRefused {
		delete(m.active, record.Request.RunID)
	}
	waiters := m.waiters[record.Request.RunID]
	delete(m.waiters, record.Request.RunID)
	m.mu.Unlock()
	for _, wake := range waiters {
		select {
		case wake <- struct{}{}:
		default:
		}
	}
}
