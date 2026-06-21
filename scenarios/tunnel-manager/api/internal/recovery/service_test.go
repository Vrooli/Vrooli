package recovery_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"tunnel-manager/internal/recovery"
	"tunnel-manager/internal/testutil/mocks"

	"github.com/stretchr/testify/require"
)

// fakeRepo records persisted events for assertions.
type fakeRepo struct {
	mu         sync.Mutex
	persisted  []recovery.RecoveryEvent
	persistErr error
}

func (f *fakeRepo) PersistEvent(_ context.Context, e recovery.RecoveryEvent) (recovery.RecoveryEvent, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.persistErr != nil {
		return recovery.RecoveryEvent{}, f.persistErr
	}
	if e.ID == "" {
		e.ID = "evt-" + string(e.Outcome)
	}
	f.persisted = append(f.persisted, e)
	return e, nil
}

func (f *fakeRepo) ListEvents(_ context.Context, limit int) ([]recovery.RecoveryEvent, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if limit > 0 && limit < len(f.persisted) {
		return f.persisted[:limit], nil
	}
	return f.persisted, nil
}

// fakeHealth returns a scripted readiness value. When readyAfterCall is
// >0, Ready returns false for the first readyAfterCall calls and true
// thereafter — modelling a tunnel that only comes back once the restart's
// readiness poll runs.
type fakeHealth struct {
	mu             sync.Mutex
	ready          bool
	readyAfterCall int
	calls          int
}

func (h *fakeHealth) Ready(context.Context) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.calls++
	if h.readyAfterCall > 0 {
		return h.calls > h.readyAfterCall
	}
	return h.ready
}

func noopSleep(time.Duration) {}

func newEngine(t *testing.T, health *fakeHealth, runner *mocks.FakeCmdRunner, repo *fakeRepo, cfg recovery.Config) recovery.Service {
	t.Helper()
	clk := mocks.NewFakeClock(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC))
	return recovery.NewService(repo, health, runner.Run, clk, cfg, noopSleep)
}

func TestRecover_SuccessRestartsAndResets(t *testing.T) {
	repo := &fakeRepo{}
	runner := &mocks.FakeCmdRunner{}
	health := &fakeHealth{ready: true}
	svc := newEngine(t, health, runner, repo, recovery.Config{})

	outcome, evt, err := svc.Recover(context.Background(), false)
	require.NoError(t, err)
	require.Equal(t, recovery.OutcomeSuccess, outcome)
	require.Equal(t, recovery.TriggerManual, evt.Trigger)
	require.Equal(t, recovery.ActionRestart, evt.Action)
	require.Equal(t, 1, runner.CallCount(), "exactly one restart")
	require.Equal(t, []string{"systemctl", "restart", "cloudflared"}, runner.Calls[0].Args)

	state, _ := svc.GetState(context.Background())
	require.Equal(t, recovery.StatusIdle, state.Status)
	require.False(t, state.CircuitOpen)
}

func TestRecover_RestartFailureBacksOff(t *testing.T) {
	repo := &fakeRepo{}
	runner := &mocks.FakeCmdRunner{Err: errors.New("boom")}
	health := &fakeHealth{ready: false}
	svc := newEngine(t, health, runner, repo, recovery.Config{})

	outcome, _, err := svc.Recover(context.Background(), false)
	require.NoError(t, err)
	require.Equal(t, recovery.OutcomeFailure, outcome)

	state, _ := svc.GetState(context.Background())
	require.Equal(t, 1, state.FailedRecovery)
	require.Equal(t, recovery.StatusMonitoring, state.Status)
	require.False(t, state.NextRetryAfter.IsZero(), "backoff window armed")
}

func TestRecover_CircuitOpensAfterMaxFailures(t *testing.T) {
	repo := &fakeRepo{}
	runner := &mocks.FakeCmdRunner{Err: errors.New("boom")}
	health := &fakeHealth{ready: false}
	svc := newEngine(t, health, runner, repo, recovery.Config{MaxBackoffRetries: 2})

	_, _, _ = svc.Recover(context.Background(), false)
	_, outcome2, _ := svc.Recover(context.Background(), false)
	_ = outcome2

	state, _ := svc.GetState(context.Background())
	require.True(t, state.CircuitOpen)
	require.Equal(t, recovery.StatusCircuitOpen, state.Status)

	// While open, a non-forced recover is skipped (no further restart).
	restartsBefore := runner.CallCount()
	outcome, _, err := svc.Recover(context.Background(), false)
	require.NoError(t, err)
	require.Equal(t, recovery.OutcomeSkipped, outcome)
	require.Equal(t, restartsBefore, runner.CallCount(), "no restart while circuit open")

	// Force bypasses the breaker and restarts.
	_, _, err = svc.Recover(context.Background(), true)
	require.NoError(t, err)
	require.Equal(t, restartsBefore+1, runner.CallCount())
}

func TestRecover_IdempotentWhileInFlight(t *testing.T) {
	// A health checker that blocks the first poll lets a second Recover
	// observe the in-flight guard. We model "in flight" by making the
	// first restart's readiness poll spin until released.
	repo := &fakeRepo{}
	runner := &mocks.FakeCmdRunner{}
	release := make(chan struct{})
	health := &gatedHealth{gate: release}
	clk := mocks.NewFakeClock(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC))
	svc := recovery.NewService(repo, health, runner.Run, clk, recovery.Config{ReadyPollAttempts: 100}, noopSleep)

	var firstOutcome recovery.EventOutcome
	done := make(chan struct{})
	go func() {
		firstOutcome, _, _ = svc.Recover(context.Background(), false)
		close(done)
	}()

	// Wait until the first attempt is mid-poll, then fire a concurrent one.
	health.waitUntilPolling()
	outcome, _, err := svc.Recover(context.Background(), false)
	require.NoError(t, err)
	require.Equal(t, recovery.OutcomeSkipped, outcome, "second call skipped while in flight")

	close(release)
	<-done
	require.Equal(t, recovery.OutcomeSuccess, firstOutcome)
	require.Equal(t, 1, runner.CallCount(), "twice == once: only one restart")
}

func TestListEvents_RejectsNegativeLimit(t *testing.T) {
	svc := newEngine(t, &fakeHealth{}, &mocks.FakeCmdRunner{}, &fakeRepo{}, recovery.Config{})
	_, err := svc.ListEvents(context.Background(), -1)
	var invalid recovery.ErrInvalidRecovery
	require.ErrorAs(t, err, &invalid)
}

func TestEvaluate_TriggersAfterThreshold(t *testing.T) {
	repo := &fakeRepo{}
	runner := &mocks.FakeCmdRunner{}
	// False for the three pre-check evaluations, then true so the
	// post-restart readiness poll on the third eval succeeds.
	health := &fakeHealth{readyAfterCall: 3}
	svc := newEngine(t, health, runner, repo, recovery.Config{ConsecutiveFailures: 3})

	for i := 0; i < 2; i++ {
		_, acted, err := svc.Evaluate(context.Background())
		require.NoError(t, err)
		require.False(t, acted, "no action before threshold")
	}
	require.Equal(t, 0, runner.CallCount())

	_, acted, err := svc.Evaluate(context.Background())
	require.NoError(t, err)
	require.True(t, acted)
	require.Equal(t, 1, runner.CallCount())
}

// gatedHealth blocks Ready until the gate channel closes, modelling a
// long-running readiness poll so a concurrent Recover hits the in-flight
// guard deterministically.
type gatedHealth struct {
	gate     chan struct{}
	polledMu sync.Mutex
	polled   chan struct{}
	once     sync.Once
}

func (g *gatedHealth) Ready(context.Context) bool {
	g.once.Do(func() {
		g.polledMu.Lock()
		if g.polled == nil {
			g.polled = make(chan struct{})
		}
		close(g.polled)
		g.polledMu.Unlock()
	})
	<-g.gate
	return true
}

func (g *gatedHealth) waitUntilPolling() {
	g.polledMu.Lock()
	if g.polled == nil {
		g.polled = make(chan struct{})
	}
	ch := g.polled
	g.polledMu.Unlock()
	<-ch
}
