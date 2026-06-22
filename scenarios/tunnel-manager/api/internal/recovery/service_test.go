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

// fakePresence is the injected UnitPresence seam. present=true mirrors a
// host that has cloudflared.service (the default for every test that
// exercises recovery actuation); present=false drives the dormant self-gate.
type fakePresence struct{ present bool }

func (f fakePresence) CloudflaredUnitPresent(context.Context) bool { return f.present }

func newEngine(t *testing.T, health *fakeHealth, runner *mocks.FakeCmdRunner, repo *fakeRepo, cfg recovery.Config) recovery.Service {
	t.Helper()
	clk := mocks.NewFakeClock(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC))
	return recovery.NewService(repo, health, fakePresence{present: true}, runner.Run, clk, cfg, noopSleep)
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
	// reset-failed precedes restart so recovery survives systemd's
	// StartLimitBurst exhaustion (the case TM adds value over systemd).
	require.Equal(t, 2, runner.CallCount(), "reset-failed then restart")
	require.Equal(t, []string{"systemctl", "reset-failed", "cloudflared"}, runner.Calls[0].Args)
	require.Equal(t, []string{"systemctl", "restart", "cloudflared"}, runner.Calls[1].Args)

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

	// Force bypasses the breaker and restarts (reset-failed + restart = 2).
	_, _, err = svc.Recover(context.Background(), true)
	require.NoError(t, err)
	require.Equal(t, restartsBefore+2, runner.CallCount())
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
	svc := recovery.NewService(repo, health, fakePresence{present: true}, runner.Run, clk, recovery.Config{ReadyPollAttempts: 100}, noopSleep)

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
	require.Equal(t, 2, runner.CallCount(), "twice == once: a single recovery attempt (reset-failed + restart), not two")
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
	require.Equal(t, 2, runner.CallCount(), "reset-failed then restart")
}

func TestEvaluate_DormantWhenNoCloudflaredUnit(t *testing.T) {
	repo := &fakeRepo{}
	runner := &mocks.FakeCmdRunner{}
	// Health would report not-ready, which on a present unit would count a
	// failure and eventually restart. The presence gate must short-circuit
	// before any of that.
	health := &fakeHealth{ready: false}
	clk := mocks.NewFakeClock(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC))
	svc := recovery.NewService(repo, health, fakePresence{present: false}, runner.Run, clk, recovery.Config{ConsecutiveFailures: 1}, noopSleep)

	for i := 0; i < 5; i++ {
		evt, acted, err := svc.Evaluate(context.Background())
		require.NoError(t, err)
		require.False(t, acted, "dormant: never acts without a cloudflared unit")
		require.Equal(t, recovery.RecoveryEvent{}, evt)
	}

	require.Equal(t, 0, runner.CallCount(), "no restart while dormant")
	require.Empty(t, repo.persisted, "no recovery_events row while dormant")

	state, _ := svc.GetState(context.Background())
	require.Equal(t, recovery.StatusIdle, state.Status, "dormant maps to idle")
	require.Equal(t, 0, state.ConsecFailures, "dormant never counts a failure")
	require.False(t, state.CircuitOpen, "circuit never opens while dormant")
}

func TestEvaluate_ResumesWhenUnitAppears(t *testing.T) {
	repo := &fakeRepo{}
	runner := &mocks.FakeCmdRunner{}
	health := &fakeHealth{ready: true}
	clk := mocks.NewFakeClock(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC))
	// Presence flips to true after the first probe — a cloudflared installed
	// after the scenario started must be picked up on the next tick.
	presence := &togglePresence{}
	svc := recovery.NewService(repo, health, presence, runner.Run, clk, recovery.Config{}, noopSleep)

	_, acted, err := svc.Evaluate(context.Background())
	require.NoError(t, err)
	require.False(t, acted, "dormant on the first tick (no unit yet)")

	presence.present = true
	_, acted, err = svc.Evaluate(context.Background())
	require.NoError(t, err)
	require.False(t, acted, "healthy once the unit is present")

	state, _ := svc.GetState(context.Background())
	require.Equal(t, recovery.StatusIdle, state.Status)
}

func TestExecuteRecovery_ResetFailedIsNonFatal(t *testing.T) {
	repo := &fakeRepo{}
	// reset-failed errors (e.g. nothing to reset) but restart succeeds; the
	// attempt must still report success.
	runner := &mocks.FakeCmdRunner{ErrFn: func(_ string, args []string) error {
		if len(args) >= 2 && args[1] == "reset-failed" {
			return errors.New("cloudflared.service: no such unit to reset")
		}
		return nil
	}}
	health := &fakeHealth{ready: true}
	svc := newEngine(t, health, runner, repo, recovery.Config{})

	outcome, evt, err := svc.Recover(context.Background(), false)
	require.NoError(t, err)
	require.Equal(t, recovery.OutcomeSuccess, outcome, "failing reset-failed does not abort recovery")
	require.Equal(t, []string{"systemctl", "reset-failed", "cloudflared"}, runner.Calls[0].Args)
	require.Equal(t, []string{"systemctl", "restart", "cloudflared"}, runner.Calls[1].Args)
	require.Contains(t, evt.Details, "reset-failed non-fatal", "the non-fatal reset is recorded for forensics")
}

func TestExecuteRecovery_RestartFailureIsFatal(t *testing.T) {
	repo := &fakeRepo{}
	// reset-failed succeeds, restart fails — the recovery fails.
	runner := &mocks.FakeCmdRunner{ErrFn: func(_ string, args []string) error {
		if len(args) >= 2 && args[1] == "restart" {
			return errors.New("boom")
		}
		return nil
	}}
	health := &fakeHealth{ready: false}
	svc := newEngine(t, health, runner, repo, recovery.Config{})

	outcome, _, err := svc.Recover(context.Background(), false)
	require.NoError(t, err)
	require.Equal(t, recovery.OutcomeFailure, outcome)
	require.Equal(t, 2, runner.CallCount(), "reset-failed attempted before the failing restart")
}

// togglePresence reports its current `present` value; flipping the field
// between Evaluate calls models a cloudflared unit installed after start.
type togglePresence struct{ present bool }

func (p *togglePresence) CloudflaredUnitPresent(context.Context) bool { return p.present }

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
