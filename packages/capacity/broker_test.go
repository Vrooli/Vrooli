package capacity

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	internalcapacity "github.com/vrooli/vrooli/internal/capacity"
	"github.com/vrooli/vrooli/internal/hostinventory"
)

// countingSource records how many host collections a run actually performed.
// The count is the whole point: on 2026-08-08 Test Genie made roughly 49 of
// these per suite, and each one shelled out to a probe set that a single wedged
// daemon had pushed to 8.2 seconds.
type countingSource struct {
	calls  int
	err    error
	cores  int
	memory hostinventory.Memory
}

func (s *countingSource) Snapshot(context.Context) (hostinventory.Snapshot, error) {
	s.calls++
	if s.err != nil {
		return hostinventory.Snapshot{}, s.err
	}
	return hostinventory.Snapshot{CPU: hostinventory.CPU{Cores: s.cores}, Memory: s.memory}, nil
}

// fakeClock advances only when a test says so, so the TTL is exercised without
// sleeping.
type fakeClock struct{ at time.Time }

func (c *fakeClock) now() time.Time          { return c.at }
func (c *fakeClock) Now() time.Time          { return c.at }
func (c *fakeClock) advance(d time.Duration) { c.at = c.at.Add(d) }

func newTestBroker(source *countingSource, clock *fakeClock) *Broker {
	return &Broker{source: source, now: clock.now}
}

func TestSnapshotReusedInsideTTL(t *testing.T) {
	source := &countingSource{cores: 32}
	clock := &fakeClock{at: time.Unix(1000, 0)}
	broker := newTestBroker(source, clock)

	for i := 0; i < 20; i++ {
		clock.advance(snapshotTTL / 10)
		if _, err := broker.snapshot(context.Background()); err != nil {
			t.Fatalf("snapshot: %v", err)
		}
	}
	// 20 admissions spanning two TTL windows must not cost 20 collections.
	if source.calls > 3 {
		t.Fatalf("collected the host %d times for 20 admissions; the cache is not holding", source.calls)
	}
}

func TestSnapshotRefreshedAfterTTL(t *testing.T) {
	source := &countingSource{cores: 32}
	clock := &fakeClock{at: time.Unix(1000, 0)}
	broker := newTestBroker(source, clock)

	if _, err := broker.snapshot(context.Background()); err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	clock.advance(snapshotTTL + time.Millisecond)
	if _, err := broker.snapshot(context.Background()); err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if source.calls != 2 {
		t.Fatalf("calls = %d, want 2 — a reading older than the TTL must be recollected", source.calls)
	}
}

// A cached reading must never outlive its window just because collection later
// failed. Granting capacity from a stale number on a host whose sensing has
// broken is worse than denying.
func TestSnapshotFailureAfterExpiryIsNotServedFromCache(t *testing.T) {
	source := &countingSource{cores: 32}
	clock := &fakeClock{at: time.Unix(1000, 0)}
	broker := newTestBroker(source, clock)

	if _, err := broker.snapshot(context.Background()); err != nil {
		t.Fatalf("seed snapshot: %v", err)
	}
	clock.advance(snapshotTTL + time.Millisecond)
	source.err = errors.New("sensing unavailable")

	if _, err := broker.snapshot(context.Background()); err == nil {
		t.Fatal("an expired cache served a reading after collection failed")
	}
}

// A failure must not evict a reading that is still inside its window either;
// the valid cached value is returned and no error surfaces.
func TestSnapshotFailureInsideTTLKeepsValidReading(t *testing.T) {
	source := &countingSource{cores: 32}
	clock := &fakeClock{at: time.Unix(1000, 0)}
	broker := newTestBroker(source, clock)

	if _, err := broker.snapshot(context.Background()); err != nil {
		t.Fatalf("seed snapshot: %v", err)
	}
	source.err = errors.New("sensing unavailable")
	clock.advance(snapshotTTL / 2)

	snapshot, err := broker.snapshot(context.Background())
	if err != nil {
		t.Fatalf("a valid cached reading was discarded on a failure that was never attempted: %v", err)
	}
	if snapshot.CPU.Cores != 32 {
		t.Fatalf("cached snapshot lost its content: %+v", snapshot)
	}
}

// The first call must always collect; an empty cache is not a valid reading.
func TestSnapshotCollectsOnFirstCall(t *testing.T) {
	source := &countingSource{cores: 8}
	clock := &fakeClock{at: time.Unix(1000, 0)}
	broker := newTestBroker(source, clock)

	snapshot, err := broker.snapshot(context.Background())
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if source.calls != 1 {
		t.Fatalf("calls = %d, want 1", source.calls)
	}
	if snapshot.CPU.Cores != 8 {
		t.Fatalf("cores = %d, want 8", snapshot.CPU.Cores)
	}
}

func TestAcquireExpiresStaleOperationClaimsBeforeAdmission(t *testing.T) {
	ctx := context.Background()
	clock := &fakeClock{at: time.Date(2026, 8, 12, 20, 0, 0, 0, time.UTC)}
	store, err := internalcapacity.NewSQLiteStore(ctx, internalcapacity.Config{
		DBPath: filepath.Join(t.TempDir(), "capacity.db"),
		Clock:  clock,
	})
	if err != nil {
		t.Fatalf("open capacity store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	stale, err := store.CreateClaim(ctx, internalcapacity.CapacityClaim{
		OwnerKind:      internalcapacity.OwnerKindOp,
		OwnerID:        "test-genie:crashed-run:phase",
		ResourceKind:   internalcapacity.ResourceKindCPU,
		AmountBytes:    4_000,
		PreferredBytes: 4_000,
		FloorBytes:     4_000,
		Priority:       internalcapacity.PriorityBatch,
	}, time.Second)
	if err != nil {
		t.Fatalf("create stale claim: %v", err)
	}
	clock.advance(2 * time.Second)
	broker := &Broker{
		store:  store,
		source: &countingSource{cores: 4, memory: hostinventory.Memory{TotalBytes: 1 << 30, AvailableBytes: 1 << 30}},
		now:    clock.now,
	}
	lease, verdict, err := broker.Acquire(ctx, "test-genie:new-run:phase", 1, 1_000)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	if lease == nil || verdict.Kind != internalcapacity.VerdictGrant {
		t.Fatalf("acquire verdict = %#v, lease=%v; stale claim should not block admission", verdict, lease != nil)
	}
	if err := lease.Release(ctx); err != nil {
		t.Fatalf("release new lease: %v", err)
	}
	got, err := store.GetClaim(ctx, stale.ClaimID)
	if err != nil {
		t.Fatalf("read stale claim: %v", err)
	}
	if got.Status != internalcapacity.StatusExpired {
		t.Fatalf("stale claim status = %q, want expired", got.Status)
	}
}
