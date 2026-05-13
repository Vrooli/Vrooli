package feedback

import (
	"context"
	"testing"
	"time"

	"swarm-manager/internal/initiativelock"
	"swarm-manager/internal/proposals"
)

// staticLister implements InitiativeLister for tests by returning a fixed
// list of names without scanning disk.
type staticLister struct{ names []string }

func (l *staticLister) ListNames() ([]string, error) { return l.names, nil }

// newSweeperEnv builds a feedback service + sweeper backed by the same
// temp-dir storage so tests can drive the sweeper through real on-disk
// state. The clock is injected so we can fast-forward past MaxAge without
// sleeping.
type sweeperEnv struct {
	*serviceEnv
	canc    *fakeCanceller
	sweeper *Sweeper
	clock   func() time.Time
	nowPtr  *time.Time
}

func newSweeperEnv(t *testing.T, maxAge time.Duration) *sweeperEnv {
	t.Helper()
	env := newServiceEnv(t)
	canc := &fakeCanceller{}
	now := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
	clk := func() time.Time { return now }
	svc, err := NewService(Config{
		Store:     env.store,
		Lock:      env.lock,
		Spawner:   env.spawner,
		Canceller: canc,
		Apply:     env.applier,
		StateBuilder: func(name string) (proposals.CurrentState, error) {
			return proposals.CurrentState{InitiativeName: name}, nil
		},
		Clock: clk,
	})
	if err != nil {
		t.Fatal(err)
	}
	env.svc = svc
	env.lock.Clock = clk
	sw := &Sweeper{
		Service:     svc,
		Initiatives: &staticLister{names: []string{"ui-rewrite"}},
		MaxAge:      maxAge,
		Interval:    0,
		Clock:       clk,
	}
	return &sweeperEnv{serviceEnv: env, canc: canc, sweeper: sw, clock: clk, nowPtr: &now}
}

func TestSweeper_DismissesOldRound(t *testing.T) {
	t.Parallel()
	env := newSweeperEnv(t, 10*time.Minute)

	round, err := env.svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeFeedback,
		Text:           "stuck round",
	})
	if err != nil {
		t.Fatal(err)
	}
	if round.Status != RoundStatusAgentThinking {
		t.Fatalf("precondition: expected agent_thinking, got %s", round.Status)
	}

	// Force-update UpdatedAt to >MaxAge in the past.
	round.UpdatedAt = env.clock().Add(-30 * time.Minute).Format(time.RFC3339)
	if err := env.store.SaveRound(round); err != nil {
		t.Fatal(err)
	}

	dismissed, err := env.sweeper.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if dismissed != 1 {
		t.Fatalf("expected 1 dismissed, got %d", dismissed)
	}
	loaded, err := env.store.LoadRound("ui-rewrite", round.Number)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != RoundStatusDismissed {
		t.Fatalf("expected dismissed after sweep, got %s", loaded.Status)
	}
	if loaded.Decision == nil || loaded.Decision.DecidedBy != "swarm-manager-sweeper" {
		t.Fatalf("expected sweeper-attributed decision, got %#v", loaded.Decision)
	}
	if h, _ := env.lock.Inspect("ui-rewrite"); h != nil {
		t.Fatal("expected lock released after sweep")
	}
}

func TestSweeper_LeavesFreshRoundAlone(t *testing.T) {
	t.Parallel()
	env := newSweeperEnv(t, 30*time.Minute)

	round, err := env.svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeFeedback,
		Text:           "fresh round",
	})
	if err != nil {
		t.Fatal(err)
	}

	dismissed, err := env.sweeper.RunOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if dismissed != 0 {
		t.Fatalf("expected 0 dismissed for fresh round, got %d", dismissed)
	}
	loaded, err := env.store.LoadRound("ui-rewrite", round.Number)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != RoundStatusAgentThinking {
		t.Fatalf("expected agent_thinking preserved, got %s", loaded.Status)
	}
}

func TestSweeper_DismissesOrphanLockless(t *testing.T) {
	t.Parallel()
	env := newSweeperEnv(t, 10*time.Hour) // huge MaxAge so we test the lock path

	round, err := env.svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeFeedback,
		Text:           "lock-orphan",
	})
	if err != nil {
		t.Fatal(err)
	}
	// Manually release the lock — simulating a server restart that ran
	// SweepStale and cleared the file but left the round behind.
	if err := env.lock.Release("ui-rewrite", round.RunID); err != nil {
		t.Fatal(err)
	}

	dismissed, err := env.sweeper.RunOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if dismissed != 1 {
		t.Fatalf("expected 1 dismissed for orphan, got %d", dismissed)
	}
	loaded, err := env.store.LoadRound("ui-rewrite", round.Number)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != RoundStatusDismissed {
		t.Fatalf("expected dismissed, got %s", loaded.Status)
	}
}

func TestSweeper_RunOnceIdempotent(t *testing.T) {
	t.Parallel()
	env := newSweeperEnv(t, 30*time.Minute)

	// Fresh round — sweeper should leave it alone, repeatedly.
	if _, err := env.svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeFeedback,
		Text:           "fresh",
	}); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 3; i++ {
		dismissed, err := env.sweeper.RunOnce(context.Background())
		if err != nil {
			t.Fatalf("pass %d: %v", i, err)
		}
		if dismissed != 0 {
			t.Fatalf("pass %d: expected 0 dismissed, got %d", i, dismissed)
		}
	}
}

func TestSweeper_NoInitiatives_NoError(t *testing.T) {
	t.Parallel()
	env := newSweeperEnv(t, 30*time.Minute)
	env.sweeper.Initiatives = &staticLister{names: nil}

	dismissed, err := env.sweeper.RunOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if dismissed != 0 {
		t.Fatalf("expected 0, got %d", dismissed)
	}
}

func TestEnvDuration(t *testing.T) {
	t.Run("default when unset", func(t *testing.T) {
		t.Setenv("FOO_TEST_DUR", "")
		if got := envDuration("FOO_TEST_DUR", time.Minute); got != time.Minute {
			t.Fatalf("got %s want %s", got, time.Minute)
		}
	})
	t.Run("parsed duration", func(t *testing.T) {
		t.Setenv("FOO_TEST_DUR", "15m")
		if got := envDuration("FOO_TEST_DUR", time.Minute); got != 15*time.Minute {
			t.Fatalf("got %s want %s", got, 15*time.Minute)
		}
	})
	t.Run("bare integer is seconds", func(t *testing.T) {
		t.Setenv("FOO_TEST_DUR", "45")
		if got := envDuration("FOO_TEST_DUR", time.Minute); got != 45*time.Second {
			t.Fatalf("got %s want %s", got, 45*time.Second)
		}
	})
	t.Run("garbage falls back", func(t *testing.T) {
		t.Setenv("FOO_TEST_DUR", "not-a-duration")
		if got := envDuration("FOO_TEST_DUR", time.Minute); got != time.Minute {
			t.Fatalf("got %s want %s", got, time.Minute)
		}
	})
}

// Reachable from package scope so the import compiles.
var _ initiativelock.Holder
