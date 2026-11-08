package analysisoperations

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/operationcoord"
)

type Config struct {
	ExecutionTimeout time.Duration
	Now              func() time.Time
	Logf             func(string, ...any)
}

type Manager struct {
	repo      Repository
	runner    Runner
	cfg       Config
	coord     *operationcoord.Coordinator[struct{}]
	wake      chan struct{}
	mu        sync.Mutex
	cancel    context.CancelFunc
	done      chan struct{}
	accepting bool
}

func NewManager(repo Repository, runner Runner, cfg Config) *Manager {
	if cfg.ExecutionTimeout <= 0 {
		cfg.ExecutionTimeout = 15 * time.Minute
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.Logf == nil {
		cfg.Logf = log.Printf
	}
	return &Manager{repo: repo, runner: runner, cfg: cfg, coord: operationcoord.New[struct{}](), wake: make(chan struct{}, 1)}
}

func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cancel != nil {
		if m.accepting {
			return nil
		}
		return ErrStopped
	}
	if m.repo == nil || m.runner == nil {
		return ErrStopped
	}
	interrupted, err := m.repo.Recover(ctx, m.cfg.Now())
	if err != nil {
		return err
	}
	for _, id := range interrupted {
		m.coord.SignalTerminal(id)
	}
	if err := m.repo.Prune(ctx); err != nil {
		return err
	}
	workerCtx, cancel := context.WithCancel(ctx)
	m.cancel, m.done = cancel, make(chan struct{})
	m.accepting = true
	go m.work(workerCtx, m.done)
	return nil
}

// Close cancels execution, never the observer. It joins the worker before a
// lifecycle resume may recover persisted work with a new execution context.
func (m *Manager) Close(ctx context.Context) error {
	m.mu.Lock()
	cancel, done := m.cancel, m.done
	m.accepting = false
	m.mu.Unlock()
	if cancel == nil {
		return nil
	}
	cancel()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		m.mu.Lock()
		if m.done == done {
			m.cancel = nil
		}
		m.mu.Unlock()
		return nil
	}
}

func (m *Manager) Submit(ctx context.Context, callerID string, spec Spec) (Operation, bool, error) {
	if database.IsTestMode(ctx) {
		return Operation{}, false, fmt.Errorf("analyzer execution has not qualified routed database and report-file isolation: %w", ErrInvalid)
	}
	callerID = strings.TrimSpace(callerID)
	if callerID == "" || len(callerID) > 200 || strings.IndexFunc(callerID, unicode.IsControl) >= 0 {
		return Operation{}, false, ErrInvalid
	}
	if spec.Kind == "" {
		spec.Kind = FleetAnalysis
	}
	if (spec.Kind != FleetAnalysis && spec.Kind != GraphRebuild) || (spec.ApplyGraph && spec.Kind != GraphRebuild) {
		return Operation{}, false, ErrInvalid
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.accepting {
		return Operation{}, false, ErrStopped
	}
	// Wake even when the response read fails after a successful admission commit.
	defer func() {
		select {
		case m.wake <- struct{}{}:
		default:
		}
	}()
	// First recover an existing binding without depending on source availability.
	op, reused, err := m.repo.Submit(ctx, callerID, spec, nil, m.cfg.Now())
	if errors.Is(err, ErrNotFound) {
		names, listErr := m.runner.Scenarios(ctx)
		if listErr != nil {
			return Operation{}, false, listErr
		}
		names, listErr = scenarioSnapshot(names)
		if listErr != nil {
			return Operation{}, false, listErr
		}
		op, reused, err = m.repo.Submit(ctx, callerID, spec, names, m.cfg.Now())
	}

	return op, reused, err
}

func scenarioSnapshot(names []string) ([]string, error) {
	unique := make(map[string]struct{}, len(names))
	for _, name := range names {
		if name == "" || len(name) > 200 || name == "." || name == ".." || strings.ContainsAny(name, "/\\\x00") {
			return nil, ErrInvalid
		}
		unique[name] = struct{}{}
	}
	if len(unique) == 0 || len(unique) > MaxScenarios {
		return nil, ErrCapacity
	}
	names = make([]string, 0, len(unique))
	for name := range unique {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

func (m *Manager) Get(ctx context.Context, id string) (Operation, error) {
	if strings.TrimSpace(id) == "" || len(id) > 200 {
		return Operation{}, ErrInvalid
	}
	return m.repo.Get(ctx, id)
}

func (m *Manager) Wait(ctx context.Context, id string) (Operation, error) {
	ready, release := m.coord.RegisterWaiter(id)
	defer release()
	op, err := m.Get(ctx, id)
	if err != nil || op.State.Terminal() {
		return op, err
	}
	select {
	case <-ctx.Done():
		return Operation{}, ctx.Err()
	case <-ready:
		return m.Get(ctx, id)
	}
}

func (m *Manager) work(ctx context.Context, done chan struct{}) {
	defer func() {
		m.mu.Lock()
		if m.done == done {
			m.accepting = false
		}
		m.mu.Unlock()
		close(done)
	}()
	for ctx.Err() == nil {
		op, err := m.repo.ClaimNext(ctx, m.cfg.Now())
		if err == nil {
			m.execute(ctx, op)
			continue
		}
		if !errors.Is(err, ErrNotFound) && ctx.Err() == nil {
			m.cfg.Logf("analysis operations: cannot claim queued work: %v", err)
			select {
			case <-ctx.Done():
			case <-time.After(time.Second):
			}
			continue
		}
		select {
		case <-ctx.Done():
		case <-m.wake:
		}
	}
}

func (m *Manager) execute(serverCtx context.Context, op Operation) {
	ctx, cancel := context.WithTimeout(serverCtx, m.cfg.ExecutionTimeout)
	defer cancel()
	state, message := Failed, "execution ended before completion"
	var graph *GraphOutcome
	defer func() {
		if value := recover(); value != nil {
			state, message = Failed, fmt.Sprintf("analysis runner panic: %v", value)
		}
		if serverCtx.Err() != nil {
			state, message = Interrupted, "server stopped before operation completion"
		} else if ctx.Err() != nil {
			state, message = Failed, ctx.Err().Error()
		}
		m.finalize(serverCtx, op.ID, state, message, graph)
	}()
	for _, item := range op.Scenarios {
		if err := ctx.Err(); err != nil {
			message = err.Error()
			return
		}
		if err := m.repo.StartScenario(ctx, op.ID, item.Scenario); err != nil {
			message = err.Error()
			return
		}
		started := m.cfg.Now()
		outcome, err := m.runner.Analyze(ctx, item.Scenario)
		outcome.Scenario, outcome.Duration = item.Scenario, m.cfg.Now().Sub(started)
		if outcome.Duration < 0 {
			outcome.Duration = 0
		}
		outcome.State, outcome.Error = Succeeded, ""
		if err != nil {
			outcome.State, outcome.Error = Failed, bounded(err.Error())
			op.Failures++
		}
		if err := m.repo.CompleteScenario(ctx, op.ID, outcome); err != nil {
			message = err.Error()
			return
		}
	}
	if op.Failures > 0 {
		state, message = Partial, fmt.Sprintf("%d scenario analyses failed; graph publication was not attempted", op.Failures)
		return
	}
	if op.Spec.Kind == GraphRebuild {
		result, err := m.runner.RebuildGraph(ctx, op.Spec.ApplyGraph)
		if len(result.DegradedSources) > 8 {
			result.DegradedSources = result.DegradedSources[:8]
		}
		result.DegradedSources = append([]string(nil), result.DegradedSources...)
		for i := range result.DegradedSources {
			result.DegradedSources[i] = bounded(result.DegradedSources[i])
		}
		graph = &result
		if err != nil {
			message = err.Error()
			return
		}
		if result.Applied != op.Spec.ApplyGraph {
			message = "graph publication outcome does not match the requested mode"
			return
		}
		if len(result.DegradedSources) > 0 {
			state, message = Partial, "graph sources were degraded"
			return
		}
	}
	state, message = Succeeded, ""
}

func (m *Manager) finalize(serverCtx context.Context, id string, state State, message string, graph *GraphOutcome) {
	for attempt := 1; ; attempt++ {
		finalCtx, release := context.WithTimeout(context.Background(), 5*time.Second)
		err := m.repo.Finish(finalCtx, id, state, message, graph, m.cfg.Now())
		if err == nil {
			m.coord.SignalTerminal(id)
			if pruneErr := m.repo.Prune(finalCtx); pruneErr != nil {
				m.cfg.Logf("analysis operations: detail retention deferred: %v", pruneErr)
			}
			release()
			return
		}
		if errors.Is(err, ErrState) {
			current, readErr := m.repo.Get(finalCtx, id)
			if readErr == nil && current.State.Terminal() {
				m.coord.SignalTerminal(id)
				release()
				return
			}
			state, message = Failed, "terminal publication rejected an inconsistent operation result"
		}
		release()
		if attempt == 1 || attempt%10 == 0 {
			m.cfg.Logf("analysis operations: finalization deferred id=%s attempt=%d: %v", id, attempt, err)
		}
		// Retry only terminal publication, never the already executed work. A
		// shutdown leaves a durable receipt for startup's interrupted recovery.
		select {
		case <-serverCtx.Done():
			return
		case <-time.After(time.Second):
		}
	}
}

func bounded(value string) string {
	value = strings.ToValidUTF8(value, "?")
	if len(value) <= MaxDiagnosticBytes {
		return value
	}
	value = value[:MaxDiagnosticBytes]
	for !utf8.ValidString(value) {
		value = value[:len(value)-1]
	}
	return value
}
