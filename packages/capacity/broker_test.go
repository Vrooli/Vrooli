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

func TestOwnerIdentityHelpersRejectColonComponents(t *testing.T) {
	if got := OwnerIDFor("test-genie", "run-1", "unit"); got != "test-genie:run-1:unit" {
		t.Fatalf("OwnerIDFor = %q", got)
	}
	if got := RunOwnerPrefix("test-genie", "run-1"); got != "test-genie:run-1:" {
		t.Fatalf("RunOwnerPrefix = %q", got)
	}
	if got := OwnerIDFor("test:genie", "run-1", "unit"); got != "" {
		t.Fatalf("OwnerIDFor accepted colon component: %q", got)
	}
	if got := RunOwnerPrefix("test-genie", "run:1"); got != "" {
		t.Fatalf("RunOwnerPrefix accepted colon component: %q", got)
	}
}

func TestReleaseRunReleasesOnlyThatRun(t *testing.T) {
	ctx := context.Background()
	store, err := internalcapacity.NewSQLiteStore(ctx, internalcapacity.Config{DBPath: filepath.Join(t.TempDir(), "capacity.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	for _, owner := range []string{OwnerIDFor("test-genie", "run-1", "unit"), OwnerIDFor("test-genie", "run-1", "contracts"), OwnerIDFor("test-genie", "run-2", "unit")} {
		if _, err := store.CreateClaim(ctx, internalcapacity.CapacityClaim{OwnerKind: internalcapacity.OwnerKindOp, OwnerID: owner, ResourceKind: internalcapacity.ResourceKindCPU, AmountBytes: 1, PreferredBytes: 1, FloorBytes: 1, Priority: internalcapacity.PriorityBatch}, 0); err != nil {
			t.Fatal(err)
		}
	}
	broker := NewBrokerWithSource(store, &countingSource{cores: 4, memory: hostinventory.Memory{TotalBytes: 1 << 30, AvailableBytes: 1 << 30}})
	if err := broker.ReleaseRun(ctx, "test-genie", "run-1"); err != nil {
		t.Fatal(err)
	}
	claims, err := store.ListClaims(ctx, internalcapacity.ClaimFilter{OwnerKind: internalcapacity.OwnerKindOp, Statuses: internalcapacity.ActiveClaimStatuses()})
	if err != nil {
		t.Fatal(err)
	}
	if len(claims) != 1 || claims[0].OwnerID != OwnerIDFor("test-genie", "run-2", "unit") {
		t.Fatalf("active claims after release = %#v, want only run-2", claims)
	}
}

// countingSource records how many host collections a run actually performed.
// The count is the whole point: on 2026-08-08 Test Genie made roughly 49 of
// these per suite, and each one shelled out to a probe set that a single wedged
// daemon had pushed to 8.2 seconds.
type countingSource struct {
	calls  int
	err    error
	cores  int
	memory hostinventory.Memory
	load   hostinventory.Load
	swap   hostinventory.Swap
}

func (s *countingSource) Snapshot(context.Context) (hostinventory.Snapshot, error) {
	s.calls++
	if s.err != nil {
		return hostinventory.Snapshot{}, s.err
	}
	return hostinventory.Snapshot{CPU: hostinventory.CPU{Cores: s.cores}, Memory: s.memory, Load: s.load, Swap: s.swap}, nil
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

func TestObserveHostStateUsesCachedReadingAndReportsSwap(t *testing.T) {
	source := &countingSource{
		cores:  4,
		memory: hostinventory.Memory{AvailableBytes: 8 << 30},
		load:   hostinventory.Load{Load1: 1.5},
		swap:   hostinventory.Swap{TotalBytes: 100, FreeBytes: 25},
	}
	clock := &fakeClock{at: time.Unix(1000, 0)}
	broker := newTestBroker(source, clock)
	observation, err := broker.ObserveHostState(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if observation.AvailableRAMBytes != 8<<30 || observation.Load1 != 1.5 || observation.SwapUsedPercent != 75 {
		t.Fatalf("observation = %#v, want available RAM 8GiB, load 1.5, swap 75%%", observation)
	}
	if source.calls != 1 {
		t.Fatalf("host collections = %d, want cached single collection", source.calls)
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
