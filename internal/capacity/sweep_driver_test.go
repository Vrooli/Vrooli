package capacity

import (
	"context"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/internal/testenv"
)

// countingSource records how many times Snapshot is called so a test can prove
// the debounce avoids re-collecting a GPU snapshot on every read.
type countingSource struct {
	snap  hostinventory.Snapshot
	err   error
	calls int
}

func (s *countingSource) Snapshot(context.Context) (hostinventory.Snapshot, error) {
	s.calls++
	return s.snap, s.err
}

func observedWhisper() (hostinventory.Snapshot, Attributor) {
	snap := snapshotWithProcs(hostinventory.GPUProcess{GPUIndex: 0, PID: 1000, ProcessName: "whisper", UsedBytes: 3 * uint64(gib)})
	attr := fakeAttributor{1000: {ContainerName: "/vrooli-whisper-1", OwnerID: "whisper"}}
	return snap, attr
}

// SweepIfDue runs the first time, then debounces within policy.SweepInterval, and
// runs again once the interval elapses.
func TestSweepIfDueDebouncesWithinInterval(t *testing.T) {
	ctx := context.Background()
	clk := testenv.NewClock(time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC))
	store := testenv.NewSQLiteStore(t, "capacity.db", func(path string) (*SQLiteStore, error) {
		return NewSQLiteStore(context.Background(), Config{DBPath: path, Clock: clk})
	})
	snap, attr := observedWhisper()
	policy := DefaultPolicy() // SweepInterval == 15s

	if _, err := store.CreateClaim(ctx, residentClaim("whisper"), DefaultHeartbeatTTL); err != nil {
		t.Fatalf("CreateClaim() error = %v", err)
	}

	// First call: due (no prior cursor), records the cursor at T0.
	if _, due, err := SweepIfDue(ctx, store, snap, attr, policy, clk.Now()); err != nil || !due {
		t.Fatalf("first SweepIfDue() due=%v err=%v, want due", due, err)
	}

	// 5s later (< 15s): debounced. Prove it did NOT run by passing an EMPTY
	// snapshot at a moment past the deadline — were it to run, whisper (now
	// unobserved) would be expired.
	clk.Advance(2 * DefaultHeartbeatTTL)
	if _, due, err := SweepIfDue(ctx, store, snapshotWithProcs(), fakeAttributor{}, policy, clk.Now().Add(-2*DefaultHeartbeatTTL+5*time.Second)); err != nil {
		t.Fatalf("debounced SweepIfDue() error = %v", err)
	} else if due {
		t.Fatalf("SweepIfDue() ran within interval, want debounced")
	}

	// Past the interval since the last sweep: due again.
	if _, due, err := SweepIfDue(ctx, store, snap, attr, policy, clk.Now()); err != nil || !due {
		t.Fatalf("post-interval SweepIfDue() due=%v err=%v, want due", due, err)
	}
}

// MaybeSweep must NOT collect a snapshot while debounced (the whole point: rapid
// reads do not shell out to nvidia-smi on every call).
func TestMaybeSweepSkipsSnapshotWhenDebounced(t *testing.T) {
	ctx := context.Background()
	clk := testenv.NewClock(time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC))
	store := testenv.NewSQLiteStore(t, "capacity.db", func(path string) (*SQLiteStore, error) {
		return NewSQLiteStore(context.Background(), Config{DBPath: path, Clock: clk})
	})
	snap, attr := observedWhisper()
	src := &countingSource{snap: snap}
	policy := DefaultPolicy()

	if _, err := store.CreateClaim(ctx, residentClaim("whisper"), DefaultHeartbeatTTL); err != nil {
		t.Fatalf("CreateClaim() error = %v", err)
	}

	if _, due, err := MaybeSweep(ctx, store, src, attr, policy, clk.Now()); err != nil || !due {
		t.Fatalf("first MaybeSweep() due=%v err=%v, want due", due, err)
	}
	if src.calls != 1 {
		t.Fatalf("snapshot calls = %d after first sweep, want 1", src.calls)
	}
	// Three debounced calls within the interval: none should sense.
	for i := 0; i < 3; i++ {
		clk.Advance(time.Second)
		if _, due, err := MaybeSweep(ctx, store, src, attr, policy, clk.Now()); err != nil || due {
			t.Fatalf("debounced MaybeSweep() due=%v err=%v, want not-due", due, err)
		}
	}
	if src.calls != 1 {
		t.Errorf("snapshot calls = %d, want 1 (debounced calls must not sense)", src.calls)
	}
}

// A sensing failure must NEVER expire a claim: MaybeSweep skips the sweep
// entirely so a transient nvidia-smi hiccup cannot strand a live resident.
func TestMaybeSweepSensingDownNeverExpires(t *testing.T) {
	ctx := context.Background()
	clk := testenv.NewClock(time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC))
	store := testenv.NewSQLiteStore(t, "capacity.db", func(path string) (*SQLiteStore, error) {
		return NewSQLiteStore(context.Background(), Config{DBPath: path, Clock: clk})
	})
	created, err := store.CreateClaim(ctx, residentClaim("whisper"), DefaultHeartbeatTTL)
	if err != nil {
		t.Fatalf("CreateClaim() error = %v", err)
	}
	clk.Advance(4 * DefaultHeartbeatTTL) // well past the deadline

	src := &countingSource{err: errSensing}
	if _, due, err := MaybeSweep(ctx, store, src, fakeAttributor{}, DefaultPolicy(), clk.Now()); err != nil || due {
		t.Fatalf("MaybeSweep() due=%v err=%v, want skipped no-op", due, err)
	}
	got, err := store.GetClaim(ctx, created.ClaimID)
	if err != nil {
		t.Fatalf("GetClaim() error = %v", err)
	}
	if got.Status != StatusGranted {
		t.Errorf("status = %q, want granted (sensing-down must not expire)", got.Status)
	}
}

// The DoD test: an observed resident claim is heartbeat-refreshed across many
// ticks and never expires, though total elapsed time is far past its original
// 30s deadline.
func TestSweepKeepsResidentAliveAcrossManyTicks(t *testing.T) {
	ctx := context.Background()
	clk := testenv.NewClock(time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC))
	store := testenv.NewSQLiteStore(t, "capacity.db", func(path string) (*SQLiteStore, error) {
		return NewSQLiteStore(context.Background(), Config{DBPath: path, Clock: clk})
	})
	snap, attr := observedWhisper()

	created, err := store.CreateClaim(ctx, residentClaim("whisper"), DefaultHeartbeatTTL)
	if err != nil {
		t.Fatalf("CreateClaim() error = %v", err)
	}

	const ticks = 6
	tickEvery := 20 * time.Second // < 30s TTL, so a refresh each tick keeps it alive
	for i := 0; i < ticks; i++ {
		clk.Advance(tickEvery)
		result, err := Sweep(ctx, store, snap, attr, DefaultPolicy(), clk.Now())
		if err != nil {
			t.Fatalf("Sweep() tick %d error = %v", i, err)
		}
		if len(result.Expired) != 0 {
			t.Fatalf("tick %d expired a live resident: %+v", i, result.Expired)
		}
		if len(result.Refreshed) != 1 {
			t.Fatalf("tick %d refreshed = %+v, want the resident claim", i, result.Refreshed)
		}
	}

	// 6 * 20s = 120s elapsed, 4x the original 30s deadline — still alive.
	got, err := store.GetClaim(ctx, created.ClaimID)
	if err != nil {
		t.Fatalf("GetClaim() error = %v", err)
	}
	if got.Status != StatusGranted {
		t.Errorf("status = %q after %v, want granted", got.Status, ticks*int(tickEvery))
	}
	// Once the owner disappears and a tick passes the (now lapsed) deadline, it
	// expires — liveness is real, not permanent.
	clk.Advance(2 * DefaultHeartbeatTTL)
	result, err := Sweep(ctx, store, snapshotWithProcs(), fakeAttributor{}, DefaultPolicy(), clk.Now())
	if err != nil {
		t.Fatalf("final Sweep() error = %v", err)
	}
	if len(result.Expired) != 1 || result.Expired[0].ClaimID != created.ClaimID {
		t.Fatalf("expired = %+v, want the resident claim once unobserved", result.Expired)
	}
}

var errSensing = sensingError("nvidia-smi unavailable")

type sensingError string

func (e sensingError) Error() string { return string(e) }
