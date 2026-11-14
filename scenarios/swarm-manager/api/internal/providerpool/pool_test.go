package providerpool

import (
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func newTestPool(t *testing.T, pool string) (*Pool, *time.Time) {
	t.Helper()
	p := New(pool, filepath.Join(t.TempDir(), "pool.json"))
	clock := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)
	now := clock
	p.SetClock(func() time.Time { return now })
	return p, &now
}

func TestKnownResetPausesThenReleasesPool(t *testing.T) {
	p, now := newTestPool(t, "opencode-go")
	reset := now.Add(30 * time.Minute).UTC().Format(time.RFC3339)

	state, err := p.Observe(Observation{Class: ClassSubscriptionSession, ResetAt: reset, Source: "runner-adapter"})
	if err != nil {
		t.Fatalf("observe: %v", err)
	}
	if !state.Blocked || state.Eligible || !state.ResetKnown || state.RequiresObservation {
		t.Fatalf("expected a known-reset pause, got %+v", state)
	}

	*now = now.Add(29 * time.Minute)
	if state, _ := p.State(); !state.Blocked {
		t.Fatalf("pool must stay paused before the observed reset: %+v", state)
	}
	*now = now.Add(2 * time.Minute)
	state, err = p.State()
	if err != nil {
		t.Fatalf("state: %v", err)
	}
	if !state.Eligible || state.Blocked {
		t.Fatalf("pool must be eligible after the observed reset: %+v", state)
	}
}

func TestUnknownResetWaitsForRecoveryObservation(t *testing.T) {
	p, _ := newTestPool(t, "codex")

	obs := Observation{Class: ClassSubscriptionWeekly, Source: "account-status", Detail: "weekly cap reached"}
	state, err := p.Observe(obs)
	if err != nil {
		t.Fatalf("observe: %v", err)
	}
	if state.ResetKnown || !state.RequiresObservation || state.Eligible {
		t.Fatalf("unknown reset must require a new observation: %+v", state)
	}
	// A repeated observation with unchanged evidence must not clear the pause.
	if state, _ = p.Observe(obs); state.Eligible {
		t.Fatalf("repeating an unknown-reset observation must not clear it: %+v", state)
	}
	if _, err := p.Reserve(Limit{MaxConcurrent: 1}, Reservation{AttemptID: "a1", Known: true}); !errors.Is(err, ErrPoolBlocked) {
		t.Fatalf("expected blocked admission, got %v", err)
	}

	state, err = p.Observe(Observation{Class: ClassRecovered, Source: "owner-wake", Evidence: []string{"account reset observed"}})
	if err != nil {
		t.Fatalf("recover: %v", err)
	}
	if !state.Eligible || state.Blocked {
		t.Fatalf("explicit recovery must clear an unknown reset: %+v", state)
	}
	if _, err := p.Reserve(Limit{MaxConcurrent: 1}, Reservation{AttemptID: "a1", Known: true}); err != nil {
		t.Fatalf("reserve after recovery: %v", err)
	}
}

func TestContextCapacityDoesNotPausePool(t *testing.T) {
	p, _ := newTestPool(t, "opencode-go")
	state, err := p.Observe(Observation{Class: ClassContextCapacity, Source: "runner", CheckpointRef: "ckpt-1"})
	if err != nil {
		t.Fatalf("observe: %v", err)
	}
	if !state.Eligible || state.Blocked {
		t.Fatalf("context capacity is per-session and must not pause the pool: %+v", state)
	}
	if state, _ := p.Observe(Observation{Class: ClassDispatchUncertain}); !state.Eligible {
		t.Fatalf("dispatch uncertainty is per-operation and must not pause the pool")
	}
}

func TestRateLimitUsesRetryAfterWindow(t *testing.T) {
	p, now := newTestPool(t, "openrouter")
	reset := now.Add(5 * time.Second)
	state, err := p.Observe(Observation{Class: ClassRateLimit, RetryAfter: "5", ResetAt: reset.UTC().Format(time.RFC3339), Source: "provider-429"})
	if err != nil {
		t.Fatalf("observe: %v", err)
	}
	if !state.Blocked {
		t.Fatalf("rate limit must pause admission: %+v", state)
	}
	*now = now.Add(6 * time.Second)
	if state, _ := p.State(); !state.Eligible {
		t.Fatalf("retry-after window must release the pause: %+v", state)
	}
}

func TestReserveIsIdempotentAndRejectsIdentityReuse(t *testing.T) {
	p, _ := newTestPool(t, "opencode-go")
	limit := Limit{MaxConcurrent: 2, MaxUnits: 100, UnknownChargeUnits: 5}
	first, err := p.Reserve(limit, Reservation{AttemptID: "wake146-1", EffortID: "aquila", Units: 10, Known: true})
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	replay, err := p.Reserve(limit, Reservation{AttemptID: "wake146-1", EffortID: "aquila", Units: 10, Known: true})
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if replay.State != ReservationHeld || replay.Units != first.Units {
		t.Fatalf("replay must be idempotent: %+v", replay)
	}
	if _, err := p.Reserve(limit, Reservation{AttemptID: "wake146-1", EffortID: "aquila", Units: 20, Known: true}); !errors.Is(err, ErrReservationConflict) {
		t.Fatalf("expected identity conflict, got %v", err)
	}
}

func TestReserveEnforcesConcurrencyAndSharedAllowance(t *testing.T) {
	p, _ := newTestPool(t, "opencode-go")
	limit := Limit{MaxConcurrent: 1, MaxUnits: 10}
	if _, err := p.Reserve(limit, Reservation{AttemptID: "a1", Units: 6, Known: true}); err != nil {
		t.Fatalf("reserve a1: %v", err)
	}
	if _, err := p.Reserve(limit, Reservation{AttemptID: "a2", Units: 1, Known: true}); !errors.Is(err, ErrPoolSaturated) {
		t.Fatalf("expected saturation, got %v", err)
	}
	if _, err := p.Settle("a1", 6, true); err != nil {
		t.Fatalf("settle a1: %v", err)
	}
	if _, err := p.Reserve(limit, Reservation{AttemptID: "a2", Units: 4, Known: true}); err != nil {
		t.Fatalf("reserve a2 after settle: %v", err)
	}
	if _, err := p.Reserve(limit, Reservation{AttemptID: "a3", Units: 1, Known: true}); !errors.Is(err, ErrPoolSaturated) {
		t.Fatalf("expected saturation on a3, got %v", err)
	}
	totals, err := p.Totals()
	if err != nil {
		t.Fatalf("totals: %v", err)
	}
	if totals.Held != 1 || totals.Settled != 1 {
		t.Fatalf("unexpected totals: %+v", totals)
	}
}

func TestUnknownChargeStaysReservedUntilKnownSettle(t *testing.T) {
	p, _ := newTestPool(t, "opencode-go")
	limit := Limit{MaxConcurrent: 2, MaxUnits: 5, UnknownChargeUnits: 5}
	if _, err := p.Reserve(limit, Reservation{AttemptID: "u1", Known: false}); err != nil {
		t.Fatalf("reserve unknown: %v", err)
	}
	held, err := p.Settle("u1", 0, false)
	if err != nil {
		t.Fatalf("settle unknown: %v", err)
	}
	if !held.UsageUnresolved || held.State != ReservationHeld {
		t.Fatalf("unknown usage must stay conservatively held: %+v", held)
	}
	// The conservative reservation still consumes the shared allowance.
	if _, err := p.Reserve(limit, Reservation{AttemptID: "u2", Known: true, Units: 1}); !errors.Is(err, ErrAllowanceExhausted) {
		t.Fatalf("expected allowance exhaustion, got %v", err)
	}
	if _, err := p.Release("u1"); !errors.Is(err, ErrReservationActive) {
		t.Fatalf("unresolved reservation must not be released as zero: %v", err)
	}
	resolved, err := p.Settle("u1", 2, true)
	if err != nil {
		t.Fatalf("resolve settlement: %v", err)
	}
	if resolved.State != ReservationSettled || resolved.UsageUnresolved {
		t.Fatalf("known settle must release the reservation: %+v", resolved)
	}
	if _, err := p.Reserve(limit, Reservation{AttemptID: "u2", Known: true, Units: 1}); err != nil {
		t.Fatalf("reserve after resolution: %v", err)
	}
}

func TestConcurrentReserveNeverOversubscribes(t *testing.T) {
	p, _ := newTestPool(t, "opencode-go")
	limit := Limit{MaxConcurrent: 2, MaxUnits: 100}
	var wg sync.WaitGroup
	var mu sync.Mutex
	admitted := 0
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := p.Reserve(limit, Reservation{AttemptID: "c" + string(rune('a'+i)), Units: 1, Known: true})
			if err == nil {
				mu.Lock()
				admitted++
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()
	if admitted != 2 {
		t.Fatalf("concurrent reservations admitted %d, want exactly 2", admitted)
	}
}

func TestPoolPersistsAcrossReload(t *testing.T) {
	p, _ := newTestPool(t, "opencode-go")
	if _, err := p.Reserve(Limit{MaxConcurrent: 2}, Reservation{AttemptID: "persist-1", Units: 1, Known: true}); err != nil {
		t.Fatalf("reserve: %v", err)
	}
	reloaded := New(p.pool, p.path)
	reservations, err := reloaded.Reservations()
	if err != nil {
		t.Fatalf("reload reservations: %v", err)
	}
	if len(reservations) != 1 || reservations[0].AttemptID != "persist-1" {
		t.Fatalf("reservation did not persist: %+v", reservations)
	}
}

func TestInvalidObservationIsRefused(t *testing.T) {
	p, _ := newTestPool(t, "opencode-go")
	if _, err := p.Observe(Observation{Class: "not-a-class"}); !errors.Is(err, ErrUnknownClass) {
		t.Fatalf("expected unknown-class refusal, got %v", err)
	}
	if _, err := p.Observe(Observation{Pool: "other", Class: ClassRateLimit}); !errors.Is(err, ErrPoolMismatch) {
		t.Fatalf("expected pool mismatch, got %v", err)
	}
	if _, err := p.Observe(Observation{Class: ClassRateLimit, ResetAt: "tomorrow"}); err == nil {
		t.Fatalf("expected invalid reset_at refusal")
	}
}

func TestSetLimitPersistsEffectiveBound(t *testing.T) {
	p, _ := newTestPool(t, "opencode-go")
	initial, err := p.Limit()
	if err != nil {
		t.Fatalf("initial limit: %v", err)
	}
	if initial.MaxConcurrent != 0 || initial.MaxUnits != 0 {
		t.Fatalf("a fresh pool has no persisted bound: %+v", initial)
	}

	want := Limit{MaxConcurrent: 2, MaxUnits: 40, UnknownChargeUnits: 5}
	state, err := p.SetLimit(want)
	if err != nil {
		t.Fatalf("set limit: %v", err)
	}
	if state.Limit != want {
		t.Fatalf("state must report the amended bound: %+v", state.Limit)
	}
	got, err := p.Limit()
	if err != nil {
		t.Fatalf("read limit: %v", err)
	}
	if got != want {
		t.Fatalf("persisted limit mismatch: got %+v want %+v", got, want)
	}
	// A repeated set is idempotent and survives reload.
	if _, err := p.SetLimit(want); err != nil {
		t.Fatalf("repeat set limit: %v", err)
	}
	if reloaded, err := New(p.pool, p.path).Limit(); err != nil || reloaded != want {
		t.Fatalf("limit did not persist across reload: %+v err=%v", reloaded, err)
	}

	// The amended bound is enforced even though it was never supplied to a
	// reservation call.
	if _, err := p.Reserve(want, Reservation{AttemptID: "s1", Units: 1, Known: true}); err != nil {
		t.Fatalf("reserve within amended bound: %v", err)
	}
	if _, err := p.Reserve(want, Reservation{AttemptID: "s2", Units: 1, Known: true}); err != nil {
		t.Fatalf("reserve second within amended bound: %v", err)
	}
	if _, err := p.Reserve(want, Reservation{AttemptID: "s3", Units: 1, Known: true}); !errors.Is(err, ErrPoolSaturated) {
		t.Fatalf("expected the amended concurrency bound to govern admission, got %v", err)
	}
}

func TestSetLimitRefusesNegative(t *testing.T) {
	p, _ := newTestPool(t, "opencode-go")
	if _, err := p.SetLimit(Limit{MaxConcurrent: -1}); err == nil {
		t.Fatalf("expected negative concurrency refusal")
	}
	if _, err := p.SetLimit(Limit{UnknownChargeUnits: -1}); err == nil {
		t.Fatalf("expected negative unknown-charge refusal")
	}
}

func TestAttributedToAnchorsRefreshToTheOwningEffort(t *testing.T) {
	p, _ := newTestPool(t, "opencode-go")
	if ok, err := p.AttributedTo("aquila"); err != nil || ok {
		t.Fatalf("an unused pool is not attributed: ok=%v err=%v", ok, err)
	}
	if _, err := p.Reserve(Limit{MaxConcurrent: 3}, Reservation{AttemptID: "a1", EffortID: "aquila", Units: 1, Known: true}); err != nil {
		t.Fatalf("reserve: %v", err)
	}
	if ok, err := p.AttributedTo("aquila"); err != nil || !ok {
		t.Fatalf("expected attribution to aquila: ok=%v err=%v", ok, err)
	}
	if ok, err := p.AttributedTo("other"); err != nil || ok {
		t.Fatalf("a pool used only by aquila must not attribute to other: ok=%v err=%v", ok, err)
	}
	if _, err := p.Reserve(Limit{MaxConcurrent: 3}, Reservation{AttemptID: "a2", EffortID: "other", Units: 1, Known: true}); err != nil {
		t.Fatalf("reserve other: %v", err)
	}
	if ok, err := p.AttributedTo("aquila"); err != nil || ok {
		t.Fatalf("a shared pool must not be attributed to either effort: ok=%v err=%v", ok, err)
	}
}

func TestListForRootReturnsSortedPoolDocuments(t *testing.T) {
	root := t.TempDir()
	if pools, err := ListForRoot(root); err != nil || len(pools) != 0 {
		t.Fatalf("empty root must list no pools: %v err=%v", pools, err)
	}
	first := New("zeta", filepath.Join(root, "zeta.json"))
	if _, err := first.SetLimit(Limit{MaxConcurrent: 1}); err != nil {
		t.Fatalf("seed zeta: %v", err)
	}
	second := New("alpha", filepath.Join(root, "alpha.json"))
	if _, err := second.SetLimit(Limit{MaxConcurrent: 1}); err != nil {
		t.Fatalf("seed alpha: %v", err)
	}
	pools, err := ListForRoot(root)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(pools) != 2 || pools[0].Pool() != "alpha" || pools[1].Pool() != "zeta" {
		t.Fatalf("expected sorted pool references, got %v", []string{pools[0].Pool(), pools[1].Pool()})
	}
}
