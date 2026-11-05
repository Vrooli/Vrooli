package capacity

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/artifactlock"
	internalcapacity "github.com/vrooli/vrooli/internal/capacity"
	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/internal/hostpressure"
	"github.com/vrooli/vrooli/internal/setpoint"
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

func TestAcquireGPURequiresExactUUIDAndCreatesBoundActiveLease(t *testing.T) {
	ctx := context.Background()
	store, err := internalcapacity.NewSQLiteStore(ctx, internalcapacity.Config{DBPath: filepath.Join(t.TempDir(), "capacity.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	source := &gpuSnapshotSource{snapshot: hostinventory.Snapshot{
		GPUs: []hostinventory.GPU{{Index: 2, UUID: "GPU-exact", Name: "test", VRAMBytes: 8 << 30, VRAMUsedBytes: 1 << 30}},
	}}
	broker := NewBrokerWithSource(store, source)
	if _, verdict, err := broker.AcquireGPU(ctx, "device-studio:shot:render", "GPU-other", 1<<30); err != nil || verdict.Kind != "deny" {
		t.Fatalf("unknown UUID acquire = verdict %+v, err %v", verdict, err)
	}
	lease, verdict, err := broker.AcquireGPU(ctx, "device-studio:shot:render", "GPU-exact", 1<<30)
	if err != nil || lease == nil || verdict.Kind != "grant" {
		t.Fatalf("exact UUID acquire = lease %v, verdict %+v, err %v", lease != nil, verdict, err)
	}
	if got := lease.Identity(); got.Index != 2 || got.UUID != "GPU-exact" || got.Name != "test" {
		t.Fatalf("lease identity = %+v", got)
	}
	claims, err := store.ListClaims(ctx, internalcapacity.ClaimFilter{OwnerID: "device-studio:shot:render", ResourceKind: internalcapacity.ResourceKindVRAM, Statuses: internalcapacity.ActiveClaimStatuses()})
	if err != nil || len(claims) != 1 || claims[0].GPUIndex == nil || *claims[0].GPUIndex != 2 || claims[0].ActivityState != internalcapacity.ActivityActive {
		t.Fatalf("GPU claim = %+v, err %v", claims, err)
	}
	if err := lease.Heartbeat(ctx); err != nil {
		t.Fatalf("heartbeat: %v", err)
	}
	if err := lease.Release(ctx); err != nil {
		t.Fatalf("release: %v", err)
	}
}

type gpuSnapshotSource struct{ snapshot hostinventory.Snapshot }

func (s *gpuSnapshotSource) Snapshot(context.Context) (hostinventory.Snapshot, error) {
	return s.snapshot, nil
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

func TestBrokerUsesFreshMemoryPSIToClassifySwapOccupancy(t *testing.T) {
	const threshold = 10.0
	tests := []struct {
		name       string
		swapFree   uint64
		pressure   hostpressure.Reading
		wantKind   string
		wantReason string
	}{
		{
			name:     "high swap with measured low PSI and sufficient RAM is admitted",
			swapFree: 20,
			pressure: hostpressure.NewRead(threshold-1, "fixture"),
			wantKind: internalcapacity.VerdictGrant,
		},
		{
			name:       "high PSI denies even when swap occupancy is low",
			swapFree:   90,
			pressure:   hostpressure.NewRead(threshold, "fixture"),
			wantKind:   internalcapacity.VerdictDeny,
			wantReason: "memory PSI",
		},
		{
			name:       "unknown PSI retains the high swap fallback denial",
			swapFree:   20,
			pressure:   hostpressure.NewUnread("fixture", "sensor unavailable"),
			wantKind:   internalcapacity.VerdictDeny,
			wantReason: "swap pressure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			broker := newMemoryPressureTestBroker(t, hostinventory.Memory{TotalBytes: 100, AvailableBytes: 90}, hostinventory.Swap{TotalBytes: 100, FreeBytes: tt.swapFree})
			broker.memoryPressure = func() hostpressure.Reading { return tt.pressure }
			lease, verdict, err := broker.AcquireNoReclaim(context.Background(), "test:psi-no-reclaim", 10, 0)
			if err != nil {
				t.Fatal(err)
			}
			if verdict.Kind != tt.wantKind {
				t.Fatalf("verdict = %+v, want kind %s", verdict, tt.wantKind)
			}
			if tt.wantReason != "" && !strings.Contains(verdict.Reason, tt.wantReason) {
				t.Fatalf("reason = %q, want substring %q", verdict.Reason, tt.wantReason)
			}
			if lease != nil {
				if err := lease.Release(context.Background()); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestAcquireNoReclaimDoesNotYieldAnIdleRAMClaim(t *testing.T) {
	ctx := context.Background()
	store, err := internalcapacity.NewSQLiteStore(ctx, internalcapacity.Config{DBPath: filepath.Join(t.TempDir(), "capacity.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if _, err := store.SetPolicyKey(ctx, "idle_grace", "0s"); err != nil {
		t.Fatal(err)
	}
	claim, err := store.CreateClaim(ctx, internalcapacity.CapacityClaim{
		OwnerKind: internalcapacity.OwnerKindResource, OwnerID: "idle-ram-service",
		ResourceKind: internalcapacity.ResourceKindRAM, AmountBytes: 90,
		PreferredBytes: 90, FloorBytes: 90, Priority: internalcapacity.PriorityService,
		YieldWhenIdle: true, Status: internalcapacity.StatusGranted,
		ActivityState: internalcapacity.ActivityIdle,
	}, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	broker := NewBrokerWithSource(store, &countingSource{cores: 4, memory: hostinventory.Memory{TotalBytes: 100, AvailableBytes: 100}})

	lease, verdict, err := broker.AcquireNoReclaim(ctx, "test:no-reclaim", 90, 0)
	if err != nil {
		t.Fatal(err)
	}
	if lease != nil || verdict.Kind != internalcapacity.VerdictDeny {
		t.Fatalf("no-reclaim admission must refuse a claim that only fits by yielding another owner's RAM: lease=%v verdict=%+v", lease != nil, verdict)
	}
	updated, err := store.GetClaim(ctx, claim.ClaimID)
	if err != nil || updated.Status != internalcapacity.StatusGranted {
		t.Fatalf("other owner's claim changed after no-reclaim refusal: claim=%+v err=%v", updated, err)
	}
}

func TestBrokerLowPSIDoesNotOverrideInsufficientRAM(t *testing.T) {
	broker := newMemoryPressureTestBroker(t, hostinventory.Memory{TotalBytes: 100, AvailableBytes: 5}, hostinventory.Swap{TotalBytes: 100, FreeBytes: 10})
	broker.memoryPressure = func() hostpressure.Reading { return hostpressure.NewRead(1, "fixture") }
	lease, verdict, err := broker.Acquire(context.Background(), "test:psi-capacity", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if lease != nil || verdict.Kind != internalcapacity.VerdictDeny || !strings.Contains(verdict.Reason, "available") {
		t.Fatalf("low PSI overrode insufficient RAM: lease=%v verdict=%+v", lease != nil, verdict)
	}
}

func TestBrokerUsesActiveMemoryPSISetpointAndFailsClosedOnInvalidBar(t *testing.T) {
	setpointPath := filepath.Join(t.TempDir(), "setpoint.json")
	valid := `{"schema_version":"1.0.0","bars":[{"id":"memory-pressure","cell_ref":"substrate/SB20","decision_ref":"test","unit":"percent","max":5,"gradeable":true}]}`
	if err := os.WriteFile(setpointPath, []byte(valid), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(setpoint.PathEnv, setpointPath)
	broker := newMemoryPressureTestBroker(t, hostinventory.Memory{TotalBytes: 100, AvailableBytes: 90}, hostinventory.Swap{TotalBytes: 100, FreeBytes: 90})
	broker.memoryPressure = func() hostpressure.Reading { return hostpressure.NewRead(7, "fixture") }
	lease, verdict, err := broker.Acquire(context.Background(), "test:custom-bar", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if lease != nil || verdict.Kind != internalcapacity.VerdictDeny || !strings.Contains(verdict.Reason, "5.00%") {
		t.Fatalf("active setpoint threshold was not applied: lease=%v verdict=%+v", lease != nil, verdict)
	}

	if err := os.WriteFile(setpointPath, []byte(`{"bad":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	lease, verdict, err = broker.Acquire(context.Background(), "test:invalid-bar", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if lease != nil || verdict.Kind != internalcapacity.VerdictDeny || !strings.Contains(verdict.Reason, "threshold unavailable") {
		t.Fatalf("invalid active setpoint did not fail closed: lease=%v verdict=%+v", lease != nil, verdict)
	}
}

func newMemoryPressureTestBroker(t *testing.T, memory hostinventory.Memory, swap hostinventory.Swap) *Broker {
	t.Helper()
	store, err := internalcapacity.NewSQLiteStore(context.Background(), internalcapacity.Config{DBPath: filepath.Join(t.TempDir(), "capacity.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return NewBrokerWithSource(store, &countingSource{cores: 4, memory: memory, swap: swap})
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

func TestAcquirePreservesUnverifiedGPUResidentAndExpiresOperation(t *testing.T) {
	ctx := context.Background()
	clock := &fakeClock{at: time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)}
	store, err := internalcapacity.NewSQLiteStore(ctx, internalcapacity.Config{DBPath: filepath.Join(t.TempDir(), "capacity.db"), Clock: clock})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	resident, err := store.CreateClaim(ctx, internalcapacity.CapacityClaim{OwnerID: "whisper", OwnerKind: internalcapacity.OwnerKindResource, ResourceKind: internalcapacity.ResourceKindVRAM, AmountBytes: 3 << 30}, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	operation, err := store.CreateClaim(ctx, internalcapacity.CapacityClaim{OwnerID: "old-op", OwnerKind: internalcapacity.OwnerKindOp, ResourceKind: internalcapacity.ResourceKindVRAM, AmountBytes: 3 << 30}, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	clock.advance(2 * time.Second)
	broker := &Broker{store: store, source: &countingSource{cores: 4, memory: hostinventory.Memory{TotalBytes: 1 << 30, AvailableBytes: 1 << 30}}, now: clock.now}
	lease, verdict, err := broker.Acquire(ctx, "new-op", 1, 1000)
	if err != nil || lease == nil {
		t.Fatalf("Acquire=%+v/%v", verdict, err)
	}
	defer lease.Release(ctx)
	got, err := store.GetClaim(ctx, resident.ClaimID)
	if err != nil || got.Status != internalcapacity.StatusGranted {
		t.Fatalf("resident was expired without GPU evidence: %+v/%v", got, err)
	}
	old, err := store.GetClaim(ctx, operation.ClaimID)
	if err != nil || old.Status != internalcapacity.StatusExpired {
		t.Fatalf("operation TTL was coupled to GPU presence: %+v/%v", old, err)
	}
}

func TestCapacityAdmissionLockIsLedgerSpecific(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "capacity.db")
	store, err := internalcapacity.NewSQLiteStore(context.Background(), internalcapacity.Config{DBPath: dbPath})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	aliasPath := filepath.Join(dir, "capacity-alias.db")
	if err := os.Symlink(dbPath, aliasPath); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	first := capacityAdmissionLockKey(dbPath)
	alias := capacityAdmissionLockKey(aliasPath)
	if first != alias {
		t.Fatalf("ledger path alias got a different lock key: %q != %q", first, alias)
	}
	second := capacityAdmissionLockKey(filepath.Join(t.TempDir(), "two", "capacity.db"))
	if first == "" || second == "" || first == second {
		t.Fatalf("ledger lock keys must be nonempty and distinct: first=%q second=%q", first, second)
	}
	if capacityAdmissionLockKey(":memory:") != "" {
		t.Fatal("separate in-memory stores must not share a cross-process lock")
	}

	// Before a ledger exists, alternate paths through a symlinked parent must
	// still select the same constructor lock.
	canonicalParent := filepath.Join(t.TempDir(), "state")
	if err := os.Mkdir(canonicalParent, 0o700); err != nil {
		t.Fatal(err)
	}
	linkedParent := filepath.Join(t.TempDir(), "state-link")
	if err := os.Symlink(canonicalParent, linkedParent); err != nil {
		t.Skipf("parent symlink unavailable: %v", err)
	}
	canonicalNewLedger := filepath.Join(canonicalParent, "new.db")
	linkedNewLedger := filepath.Join(linkedParent, "new.db")
	if _, err := os.Stat(canonicalNewLedger); !os.IsNotExist(err) {
		t.Fatalf("fixture ledger unexpectedly exists: %v", err)
	}
	if got, want := capacityAdmissionLockKey(canonicalNewLedger), capacityAdmissionLockKey(linkedNewLedger); got != want {
		t.Fatalf("new ledger through parent symlink got different lock keys: %q != %q", got, want)
	}
}

func TestAcquireCleansCreatedClaimAfterCallerCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	store, err := internalcapacity.NewSQLiteStore(context.Background(), internalcapacity.Config{DBPath: filepath.Join(t.TempDir(), "capacity.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	broker := NewBrokerWithSource(store, &countingSource{cores: 4, memory: hostinventory.Memory{TotalBytes: 1 << 30, AvailableBytes: 1 << 30}})
	broker.afterClaimCreated = func(internalcapacity.CapacityClaim) { cancel() }
	lease, verdict, err := broker.Acquire(ctx, "test:cancelled", 1024, 1)
	if lease != nil || verdict.Kind != "" {
		t.Fatalf("Acquire after caller cancellation = (%v, %+v, %v)", lease != nil, verdict, err)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Acquire error = %v, want context.Canceled", err)
	}
	claims, err := store.ListClaims(context.Background(), internalcapacity.ClaimFilter{OwnerKind: internalcapacity.OwnerKindOp, OwnerID: "test:cancelled", Statuses: internalcapacity.ActiveClaimStatuses()})
	if err != nil {
		t.Fatal(err)
	}
	if len(claims) != 0 {
		t.Fatalf("canceled admission left created claims active: %+v", claims)
	}
}

func TestAcquireReportsClaimRollbackErrors(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "capacity.db")
	store, err := internalcapacity.NewSQLiteStore(ctx, internalcapacity.Config{DBPath: dbPath})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	triggerDB, err := sql.Open("sqlite", "file:"+url.PathEscape(dbPath)+"?_pragma=busy_timeout(1000)")
	if err != nil {
		t.Fatal(err)
	}
	defer triggerDB.Close()
	_, err = triggerDB.ExecContext(ctx, `CREATE TRIGGER reject_capacity_release
BEFORE UPDATE OF status ON capacity_claims
WHEN NEW.status = 'released'
BEGIN SELECT RAISE(ABORT, 'forced release failure'); END`)
	if err != nil {
		t.Fatal(err)
	}
	broker := NewBrokerWithSource(store, &countingSource{cores: 4, memory: hostinventory.Memory{TotalBytes: 100, AvailableBytes: 100}})
	lease, verdict, err := broker.Acquire(ctx, "test:rollback-error", 60, 5_000)
	if lease != nil || verdict.Kind != internalcapacity.VerdictDeny {
		t.Fatalf("Acquire = (%v, %+v), want denial after RAM claim", lease != nil, verdict)
	}
	if err == nil || !strings.Contains(err.Error(), "forced release failure") {
		t.Fatalf("rollback failure was not returned: %v", err)
	}
	claims, err := store.ListClaims(ctx, internalcapacity.ClaimFilter{OwnerKind: internalcapacity.OwnerKindOp, OwnerID: "test:rollback-error", Statuses: internalcapacity.ActiveClaimStatuses()})
	if err != nil || len(claims) != 1 {
		t.Fatalf("claim state after forced rollback error = %+v, %v; want retained active claim and surfaced error", claims, err)
	}
}

func TestBrokerAdmissionSerializesAcrossProcesses(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "capacity.db")
	ctx := context.Background()
	store, err := internalcapacity.NewSQLiteStore(ctx, internalcapacity.Config{DBPath: dbPath})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	startPath := filepath.Join(dir, "start")
	finishPath := filepath.Join(dir, "finish")
	cmds := make([]*exec.Cmd, 2)
	waited := make([]bool, len(cmds))
	for i := range cmds {
		cmd := exec.Command(os.Args[0], "-test.run=^TestBrokerAdmissionSubprocessHelper$")
		cmd.Env = append(os.Environ(),
			"CAPACITY_ADMISSION_HELPER=1",
			"CAPACITY_ADMISSION_DB="+dbPath,
			"CAPACITY_ADMISSION_DIR="+dir,
			fmt.Sprintf("CAPACITY_ADMISSION_ID=%d", i),
			"CAPACITY_ADMISSION_START="+startPath,
			"CAPACITY_ADMISSION_FINISH="+finishPath,
		)
		if err := cmd.Start(); err != nil {
			t.Fatalf("start helper %d: %v", i, err)
		}
		cmds[i] = cmd
	}
	defer func() {
		_ = os.WriteFile(finishPath, []byte("done"), 0o600)
		for i, cmd := range cmds {
			if cmd.Process != nil && !waited[i] {
				_ = cmd.Process.Kill()
				_, _ = cmd.Process.Wait()
			}
		}
	}()
	for i := range cmds {
		waitForFile(t, filepath.Join(dir, fmt.Sprintf("ready-%d", i)))
	}
	if err := os.WriteFile(startPath, []byte("go"), 0o600); err != nil {
		t.Fatal(err)
	}
	for i := range cmds {
		waitForFile(t, filepath.Join(dir, fmt.Sprintf("result-%d", i)))
	}
	results := []string{}
	for i := range cmds {
		result, err := os.ReadFile(filepath.Join(dir, fmt.Sprintf("result-%d", i)))
		if err != nil {
			t.Fatal(err)
		}
		results = append(results, strings.TrimSpace(string(result)))
	}
	grants, denies := 0, 0
	for _, result := range results {
		switch result {
		case "grant":
			grants++
		case "deny":
			denies++
		default:
			t.Fatalf("unexpected child admission result %q", result)
		}
	}
	if grants != 1 || denies != 1 {
		t.Fatalf("independent brokers admitted overlapping claims: results=%v", results)
	}
	if err := os.WriteFile(finishPath, []byte("done"), 0o600); err != nil {
		t.Fatal(err)
	}
	for i, cmd := range cmds {
		err := cmd.Wait()
		waited[i] = true
		if err != nil {
			t.Fatalf("helper %d: %v", i, err)
		}
	}
}

func TestBrokerAdmissionWaitsForCrossProcessLedgerLock(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "capacity.db")
	store, err := internalcapacity.NewSQLiteStore(context.Background(), internalcapacity.Config{DBPath: dbPath})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	unlock, err := artifactlock.AcquireContext(context.Background(), capacityAdmissionLockKey(dbPath))
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	startPath := filepath.Join(dir, "start")
	cmd := exec.Command(os.Args[0], "-test.run=^TestBrokerAdmissionSubprocessHelper$")
	cmd.Env = append(os.Environ(),
		"CAPACITY_ADMISSION_HELPER=1",
		"CAPACITY_ADMISSION_DB="+dbPath,
		"CAPACITY_ADMISSION_DIR="+dir,
		"CAPACITY_ADMISSION_ID=lock-held",
		"CAPACITY_ADMISSION_START="+startPath,
		"CAPACITY_ADMISSION_FINISH="+filepath.Join(dir, "finish"),
		"CAPACITY_ADMISSION_TIMEOUT=250ms",
	)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	waited := false
	defer func() {
		if cmd.Process != nil && !waited {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
		}
	}()
	waitForFile(t, filepath.Join(dir, "ready-lock-held"))
	if err := os.WriteFile(startPath, []byte("go"), 0o600); err != nil {
		t.Fatal(err)
	}
	waitForFile(t, filepath.Join(dir, "result-lock-held"))
	result, err := os.ReadFile(filepath.Join(dir, "result-lock-held"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "finish"), []byte("done"), 0o600); err != nil {
		t.Fatal(err)
	}
	err = cmd.Wait()
	waited = true
	if err != nil {
		t.Fatalf("helper process: %v", err)
	}
	if got := strings.TrimSpace(string(result)); got != "deny" {
		t.Fatalf("admission while another process holds the ledger lock = %q, want deny", got)
	}
}

func TestBrokerConstructorWaitsForLedgerLockBeforeOpeningSQLite(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "capacity.db")
	lockKey := capacityAdmissionLockKey(dbPath)
	unlock, err := artifactlock.AcquireContext(context.Background(), lockKey)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	readyPath := filepath.Join(dir, "ready")
	resultPath := filepath.Join(dir, "result")
	cmd := exec.Command(os.Args[0], "-test.run=^TestBrokerConstructorSubprocessHelper$")
	cmd.Env = append(os.Environ(),
		"CAPACITY_CONSTRUCTOR_HELPER=1",
		"CAPACITY_CONSTRUCTOR_MODE=held-lock",
		"CAPACITY_CONSTRUCTOR_DB="+dbPath,
		"CAPACITY_CONSTRUCTOR_READY="+readyPath,
		"CAPACITY_CONSTRUCTOR_START="+filepath.Join(dir, "start"),
		"CAPACITY_CONSTRUCTOR_RESULT="+resultPath,
	)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	waited := false
	defer func() {
		if cmd.Process != nil && !waited {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
		}
	}()
	waitForFile(t, readyPath)
	if err := os.WriteFile(filepath.Join(dir, "start"), []byte("go"), 0o600); err != nil {
		t.Fatal(err)
	}
	waitForFile(t, resultPath)
	result, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("constructor helper: %v", err)
	}
	waited = true
	if got := strings.TrimSpace(string(result)); got != "lock-timeout" {
		t.Fatalf("constructor under held ledger lock = %q, want lock-timeout before SQLite open", got)
	}
	if _, err := os.Stat(dbPath); !os.IsNotExist(err) {
		t.Fatalf("constructor touched ledger before admission lock; stat error = %v", err)
	}
}

func TestBrokerConstructorsSerializeEmptyLedgerAcrossProcesses(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "capacity.db")
	startPath := filepath.Join(dir, "start")
	cmds := make([]*exec.Cmd, 2)
	waited := make([]bool, len(cmds))
	for i := range cmds {
		cmd := exec.Command(os.Args[0], "-test.run=^TestBrokerConstructorSubprocessHelper$")
		cmd.Env = append(os.Environ(),
			"CAPACITY_CONSTRUCTOR_HELPER=1",
			"CAPACITY_CONSTRUCTOR_MODE=parallel",
			"CAPACITY_CONSTRUCTOR_DB="+dbPath,
			"CAPACITY_CONSTRUCTOR_READY="+filepath.Join(dir, fmt.Sprintf("ready-%d", i)),
			"CAPACITY_CONSTRUCTOR_START="+startPath,
			"CAPACITY_CONSTRUCTOR_RESULT="+filepath.Join(dir, fmt.Sprintf("result-%d", i)),
			fmt.Sprintf("CAPACITY_CONSTRUCTOR_ID=%d", i),
		)
		if err := cmd.Start(); err != nil {
			t.Fatalf("start constructor helper %d: %v", i, err)
		}
		cmds[i] = cmd
	}
	defer func() {
		for i, cmd := range cmds {
			if cmd.Process != nil && !waited[i] {
				_ = cmd.Process.Kill()
				_, _ = cmd.Process.Wait()
			}
		}
	}()
	for i := range cmds {
		waitForFile(t, filepath.Join(dir, fmt.Sprintf("ready-%d", i)))
	}
	if err := os.WriteFile(startPath, []byte("go"), 0o600); err != nil {
		t.Fatal(err)
	}
	for i := range cmds {
		waitForFile(t, filepath.Join(dir, fmt.Sprintf("result-%d", i)))
	}
	for i, cmd := range cmds {
		result, err := os.ReadFile(filepath.Join(dir, fmt.Sprintf("result-%d", i)))
		if err != nil {
			t.Fatal(err)
		}
		if got := strings.TrimSpace(string(result)); got != "opened" {
			t.Errorf("constructor %d result = %q, want opened", i, got)
		}
		if err := cmd.Wait(); err != nil {
			t.Errorf("constructor helper %d: %v", i, err)
		}
		waited[i] = true
	}
}

func TestBrokerConstructorSubprocessHelper(t *testing.T) {
	if os.Getenv("CAPACITY_CONSTRUCTOR_HELPER") != "1" {
		return
	}
	ready := os.Getenv("CAPACITY_CONSTRUCTOR_READY")
	start := os.Getenv("CAPACITY_CONSTRUCTOR_START")
	result := os.Getenv("CAPACITY_CONSTRUCTOR_RESULT")
	if err := os.WriteFile(ready, []byte("ready"), 0o600); err != nil {
		t.Fatalf("write constructor readiness: %v", err)
	}
	waitForFile(t, start)
	var ctx context.Context
	var cancel context.CancelFunc
	if os.Getenv("CAPACITY_CONSTRUCTOR_MODE") == "held-lock" {
		ctx, cancel = context.WithTimeout(context.Background(), 300*time.Millisecond)
	} else {
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	}
	defer cancel()
	broker, err := NewBroker(ctx, os.Getenv("CAPACITY_CONSTRUCTOR_DB"))
	if err != nil {
		status := "error"
		if strings.Contains(err.Error(), "initialization lock unavailable") {
			status = "lock-timeout"
		}
		if writeErr := os.WriteFile(result, []byte(status), 0o600); writeErr != nil {
			t.Fatalf("write constructor result: %v", writeErr)
		}
		return
	}
	if closeErr := broker.Close(); closeErr != nil {
		t.Fatalf("close constructed broker: %v", closeErr)
	}
	if err := os.WriteFile(result, []byte("opened"), 0o600); err != nil {
		t.Fatalf("write constructor result: %v", err)
	}
}

// TestBrokerAdmissionSubprocessHelper is the child entry point for the
// cross-process admission test above. Both processes use the same ledger path
// and deterministic host reading, while holding the first successful lease
// until the parent has observed both verdicts.
func TestBrokerAdmissionSubprocessHelper(t *testing.T) {
	if os.Getenv("CAPACITY_ADMISSION_HELPER") != "1" {
		return
	}
	dir := os.Getenv("CAPACITY_ADMISSION_DIR")
	id := os.Getenv("CAPACITY_ADMISSION_ID")
	dbPath := os.Getenv("CAPACITY_ADMISSION_DB")
	write := func(name, value string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(value), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	write("ready-"+id, "ready")
	waitForFile(t, os.Getenv("CAPACITY_ADMISSION_START"))
	timeout := 10 * time.Second
	if raw := os.Getenv("CAPACITY_ADMISSION_TIMEOUT"); raw != "" {
		parsed, parseErr := time.ParseDuration(raw)
		if parseErr != nil {
			t.Fatal(parseErr)
		}
		timeout = parsed
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	store, err := internalcapacity.NewSQLiteStore(ctx, internalcapacity.Config{DBPath: dbPath})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	broker := NewBrokerWithSource(store, &countingSource{cores: 4, memory: hostinventory.Memory{TotalBytes: 100, AvailableBytes: 100}})
	lease, verdict, err := broker.Acquire(ctx, "child-"+id, 60, 0)
	if err != nil {
		t.Fatal(err)
	}
	if lease != nil && verdict.Kind == internalcapacity.VerdictGrant {
		write("result-"+id, "grant")
		waitForFile(t, os.Getenv("CAPACITY_ADMISSION_FINISH"))
		if err := lease.Release(ctx); err != nil {
			t.Fatal(err)
		}
		return
	}
	if verdict.Kind != internalcapacity.VerdictDeny {
		t.Fatalf("unexpected verdict: %+v", verdict)
	}
	write("result-"+id, "deny")
}

func TestBrokerLeaseHeartbeatRenewalAndFences(t *testing.T) {
	ctx := context.Background()
	clock := &fakeClock{at: time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)}
	store, err := internalcapacity.NewSQLiteStore(ctx, internalcapacity.Config{DBPath: filepath.Join(t.TempDir(), "capacity.db"), Clock: clock})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	broker := NewBrokerWithSource(store, &countingSource{cores: 4, memory: hostinventory.Memory{TotalBytes: 1 << 30, AvailableBytes: 1 << 30}})
	base, verdict, err := broker.Acquire(ctx, "test:renew", 1024, 0)
	if err != nil || base == nil || verdict.Kind != internalcapacity.VerdictGrant {
		t.Fatalf("Acquire = (%v, %+v, %v)", base != nil, verdict, err)
	}
	renewable, ok := base.(RenewableLease)
	if !ok {
		t.Fatal("broker lease does not expose optional renewal")
	}
	claims, err := store.ListClaims(ctx, internalcapacity.ClaimFilter{OwnerKind: internalcapacity.OwnerKindOp, OwnerID: "test:renew", Statuses: internalcapacity.ActiveClaimStatuses()})
	if err != nil || len(claims) != 1 {
		t.Fatalf("claims before renewal = %d, %v", len(claims), err)
	}
	oldDeadline := *claims[0].HeartbeatDeadlineAt
	clock.advance(10 * time.Minute)
	if err := renewable.Heartbeat(ctx); err != nil {
		t.Fatalf("Heartbeat: %v", err)
	}
	renewed, err := store.GetClaim(ctx, claims[0].ClaimID)
	if err != nil {
		t.Fatal(err)
	}
	if renewed.Generation != claims[0].Generation || renewed.HeartbeatDeadlineAt == nil || !renewed.HeartbeatDeadlineAt.After(oldDeadline) {
		t.Fatalf("renewal changed generation or failed to move deadline: before=%+v after=%+v", claims[0], renewed)
	}
	if _, err := store.ReportActivity(ctx, renewed.ClaimID, renewed.Generation, internalcapacity.ActivityActive); err != nil {
		t.Fatal(err)
	}
	if err := renewable.Heartbeat(ctx); !errors.Is(err, internalcapacity.ErrStaleGeneration) {
		t.Fatalf("heartbeat after generation change = %v, want ErrStaleGeneration", err)
	}
	if err := base.Release(ctx); err != nil {
		t.Fatalf("Release: %v", err)
	}
	if err := renewable.Heartbeat(ctx); err == nil {
		t.Fatal("released lease was renewed")
	}
}

func TestBrokerLeaseHeartbeatCannotResurrectExpiredClaim(t *testing.T) {
	ctx := context.Background()
	clock := &fakeClock{at: time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)}
	store, err := internalcapacity.NewSQLiteStore(ctx, internalcapacity.Config{DBPath: filepath.Join(t.TempDir(), "capacity.db"), Clock: clock})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	broker := NewBrokerWithSource(store, &countingSource{cores: 4, memory: hostinventory.Memory{TotalBytes: 1 << 30, AvailableBytes: 1 << 30}})
	base, verdict, err := broker.Acquire(ctx, "test:expired", 1024, 0)
	if err != nil || base == nil || verdict.Kind != internalcapacity.VerdictGrant {
		t.Fatalf("Acquire = (%v, %+v, %v)", base != nil, verdict, err)
	}
	renewable := base.(RenewableLease)
	clock.advance(capacityHeartbeatTTL + time.Second)
	if err := renewable.Heartbeat(ctx); !errors.Is(err, internalcapacity.ErrStaleGeneration) {
		t.Fatalf("heartbeat after deadline without sweep = %v, want ErrStaleGeneration", err)
	}
	beforeSweep, err := store.ListClaims(ctx, internalcapacity.ClaimFilter{OwnerKind: internalcapacity.OwnerKindOp, OwnerID: "test:expired"})
	if err != nil || len(beforeSweep) != 1 || beforeSweep[0].HeartbeatDeadlineAt == nil || !beforeSweep[0].HeartbeatDeadlineAt.Before(clock.Now()) {
		t.Fatalf("unswept expired claim changed after refused renewal: %+v, %v", beforeSweep, err)
	}
	if _, err := store.ExpireStaleClaims(ctx, clock.Now(), nil); err != nil {
		t.Fatal(err)
	}
	if err := renewable.Heartbeat(ctx); !errors.Is(err, internalcapacity.ErrStaleGeneration) {
		t.Fatalf("heartbeat after expiry sweep = %v, want ErrStaleGeneration", err)
	}
	claims, err := store.ListClaims(ctx, internalcapacity.ClaimFilter{OwnerKind: internalcapacity.OwnerKindOp, OwnerID: "test:expired"})
	if err != nil || len(claims) != 1 || claims[0].Status != internalcapacity.StatusExpired {
		t.Fatalf("expired claim after failed heartbeat = %+v, %v", claims, err)
	}
}

func TestAcquireRollsBackFirstClaimWhenSecondResourceIsDenied(t *testing.T) {
	ctx := context.Background()
	store, err := internalcapacity.NewSQLiteStore(ctx, internalcapacity.Config{DBPath: filepath.Join(t.TempDir(), "capacity.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	broker := NewBrokerWithSource(store, &countingSource{cores: 4, memory: hostinventory.Memory{TotalBytes: 100, AvailableBytes: 100}})
	lease, verdict, err := broker.Acquire(ctx, "test:rollback", 60, 5_000)
	if err != nil {
		t.Fatal(err)
	}
	if lease != nil || verdict.Kind != internalcapacity.VerdictDeny {
		t.Fatalf("Acquire = (%v, %+v), want denial after RAM claim", lease != nil, verdict)
	}
	claims, err := store.ListClaims(ctx, internalcapacity.ClaimFilter{OwnerKind: internalcapacity.OwnerKindOp, OwnerID: "test:rollback", Statuses: internalcapacity.ActiveClaimStatuses()})
	if err != nil {
		t.Fatal(err)
	}
	if len(claims) != 0 {
		t.Fatalf("partial admission left active claims: %+v", claims)
	}
}

func waitForFile(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		} else if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("stat %s: %v", path, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", path)
}
