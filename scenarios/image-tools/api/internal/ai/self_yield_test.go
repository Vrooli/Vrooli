package ai

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"image-tools/internal/backends"
	"image-tools/internal/capacity"
	"image-tools/internal/technique"
)

const testGPU = "GPU-test"

// ledgerBroker grants a claim only while enough VRAM is unclaimed, like the host
// ledger: every claim counts, whoever holds it and whether or not it is in use.
type ledgerBroker struct {
	mu      sync.Mutex
	total   int64
	claims  int
	granted map[string]int64
}

func (b *ledgerBroker) free() int64 {
	free := b.total
	for _, bytes := range b.granted {
		free -= bytes
	}
	return free
}

func (b *ledgerBroker) Claim(_ context.Context, owner string, bytes int64, _ capacity.ClaimOptions) (capacity.Lease, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.claims++
	if bytes > b.free() {
		return capacity.Lease{DegradeToCPU: true, Warnings: []string{"insufficient vram"}}, nil
	}
	if b.granted == nil {
		b.granted = map[string]int64{}
	}
	b.granted[owner] = bytes
	return capacity.Lease{ClaimID: owner}, nil
}
func (b *ledgerBroker) Release(_ context.Context, lease capacity.Lease) error {
	b.give(lease.ClaimID)
	return nil
}
func (b *ledgerBroker) give(owner string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.granted, owner)
}
func (b *ledgerBroker) Heartbeat(_ context.Context, lease capacity.Lease) (capacity.Lease, error) {
	return lease, nil
}

// residentFixture registers a warm provider whose worker holds bytes of VRAM for
// model on gpu, claimed from the ledger, and reports whether it is still loaded.
type residentFixture struct {
	runner *warmSidecarRunner
}

func (f residentFixture) loaded() bool {
	f.runner.mu.Lock()
	defer f.runner.mu.Unlock()
	return f.runner.resident != nil
}

func addResident(t *testing.T, reg *backends.Registry, broker *ledgerBroker, name, model, gpu string, bytes int64, lastUsed time.Time) residentFixture {
	t.Helper()
	runner := newWarmSidecarRunner()
	provider := &execProvider{
		backendProcess: backendProcess{name: name, program: "python3", warm: runner},
		techniques:     technique.Single("caption", nil),
	}
	if err := reg.Register(provider); err != nil {
		t.Fatal(err)
	}
	if _, err := broker.Claim(context.Background(), name, bytes, capacity.ClaimOptions{}); err != nil {
		t.Fatal(err)
	}
	group := newLeaseGroup(context.Background())
	group.renew("GPU residency", &memoryTestLease{onRelease: func(context.Context) { broker.give(name) }})
	runner.resident, runner.residentModel = group, model+"|gpu:"+gpu
	runner.residentGPUUUID, runner.residentGPUBytes, runner.lastUsed = gpu, bytes, lastUsed
	runner.residencyID = "residency:" + name
	t.Cleanup(func() { _ = runner.Close() })
	return residentFixture{runner: runner}
}

func holdForAnotherOwner(t *testing.T, broker *ledgerBroker, bytes int64) {
	t.Helper()
	if lease, err := broker.Claim(context.Background(), "resource/ollama", bytes, capacity.ClaimOptions{}); err != nil || lease.DegradeToCPU {
		t.Fatalf("setup claim: %#v %v", lease, err)
	}
}

func newYieldEngine(broker *ledgerBroker) (*Engine, *backends.Registry) {
	reg := backends.New()
	return &Engine{deps: Deps{Capacity: broker, Backends: reg}}, reg
}

func TestAGPUDenialUnloadsIdleResidentsOfThisServiceThenAdmits(t *testing.T) {
	broker := &ledgerBroker{total: 10 << 30}
	eng, reg := newYieldEngine(broker)
	old := addResident(t, reg, broker, "sam2", "sam2-large", testGPU, 2<<30, time.Now().Add(-time.Hour))
	recent := addResident(t, reg, broker, "florence", "florence-2", testGPU, 4<<30, time.Now().Add(-time.Minute))
	holdForAnotherOwner(t, broker, 2<<30)
	// 2 GiB is free and 6 GiB is wanted: the oldest resident alone leaves 4 GiB,
	// both leave 8 GiB.
	meta := map[string]string{}
	lease, err := eng.claimGPU(context.Background(), "image-tools:job", 6<<30, capacity.ClaimOptions{GPUUUID: testGPU}, meta)
	if err != nil {
		t.Fatal(err)
	}
	if lease.DegradeToCPU {
		t.Fatalf("claim still denied after yielding idle residents: %#v", lease)
	}
	if got := meta[evictedResidentsMetadataKey]; got != "sam2-large,florence-2" {
		t.Fatalf("evicted_residents = %q, want the least recently used first", got)
	}
	if old.loaded() || recent.loaded() {
		t.Fatal("yielded residents are still loaded")
	}
}

func TestAGPUDenialStopsUnloadingOnceTheClaimIsGranted(t *testing.T) {
	broker := &ledgerBroker{total: 8 << 30}
	eng, reg := newYieldEngine(broker)
	old := addResident(t, reg, broker, "sam2", "sam2-large", testGPU, 2<<30, time.Now().Add(-time.Hour))
	recent := addResident(t, reg, broker, "florence", "florence-2", testGPU, 2<<30, time.Now().Add(-time.Minute))
	holdForAnotherOwner(t, broker, 2<<30)
	meta := map[string]string{}
	lease, err := eng.claimGPU(context.Background(), "image-tools:job", 4<<30, capacity.ClaimOptions{GPUUUID: testGPU}, meta)
	if err != nil || lease.DegradeToCPU {
		t.Fatalf("claim = %#v, %v", lease, err)
	}
	// 2 GiB is free and 4 GiB is wanted: the oldest resident alone covers it.
	if meta[evictedResidentsMetadataKey] != "sam2-large" || old.loaded() || !recent.loaded() {
		t.Fatalf("evicted=%q old loaded=%v recent loaded=%v", meta[evictedResidentsMetadataKey], old.loaded(), recent.loaded())
	}
}

func TestAResidentServingAJobIsNeverUnloaded(t *testing.T) {
	broker := &ledgerBroker{total: 5 << 30}
	eng, reg := newYieldEngine(broker)
	busy := addResident(t, reg, broker, "florence", "florence-2", testGPU, 4<<30, time.Now().Add(-time.Hour))
	busy.runner.gate <- struct{}{} // a job holds the worker from reserve to release
	meta := map[string]string{}
	lease, err := eng.claimGPU(context.Background(), "image-tools:job", 6<<30, capacity.ClaimOptions{GPUUUID: testGPU}, meta)
	if err != nil {
		t.Fatal(err)
	}
	if !lease.DegradeToCPU {
		t.Fatal("a busy resident's memory was handed out")
	}
	if !busy.loaded() || meta[evictedResidentsMetadataKey] != "" {
		t.Fatalf("busy resident unloaded: loaded=%v evicted=%q", busy.loaded(), meta[evictedResidentsMetadataKey])
	}
	<-busy.runner.gate
}

func TestYieldingNeverReachesAnotherGPUOrAnotherOwner(t *testing.T) {
	broker := &ledgerBroker{total: 9 << 30}
	eng, reg := newYieldEngine(broker)
	elsewhere := addResident(t, reg, broker, "sam2", "sam2-large", "GPU-other", 4<<30, time.Now().Add(-time.Hour))
	// Another owner's claim (Ollama, another scenario) is plain ledger state: it
	// is not an image-tools worker, so nothing here can unload it.
	holdForAnotherOwner(t, broker, 4<<30)
	before := broker.claims
	meta := map[string]string{}
	lease, err := eng.claimGPU(context.Background(), "image-tools:job", 6<<30, capacity.ClaimOptions{GPUUUID: testGPU}, meta)
	if err != nil || !lease.DegradeToCPU {
		t.Fatalf("lease=%#v err=%v", lease, err)
	}
	if !elsewhere.loaded() || broker.granted["resource/ollama"] == 0 {
		t.Fatal("a claim on another GPU or another owner was touched")
	}
	if broker.claims-before != 1 {
		t.Fatalf("a denial with nothing to yield must not retry: %d claims", broker.claims-before)
	}
}

func TestAGrantedClaimNeverTouchesResidents(t *testing.T) {
	broker := &ledgerBroker{total: 10 << 30}
	eng, reg := newYieldEngine(broker)
	idle := addResident(t, reg, broker, "sam2", "sam2-large", testGPU, 2<<30, time.Now())
	meta := map[string]string{}
	lease, err := eng.claimGPU(context.Background(), "image-tools:job", 4<<30, capacity.ClaimOptions{GPUUUID: testGPU}, meta)
	if err != nil || lease.DegradeToCPU || !idle.loaded() || len(meta) != 0 {
		t.Fatalf("lease=%#v err=%v loaded=%v meta=%v", lease, err, idle.loaded(), meta)
	}
}

func TestResidentGPUBytesRoundTripThroughReserveMeta(t *testing.T) {
	runner := newWarmSidecarRunner()
	t.Cleanup(func() { _ = runner.Close() })
	meta := map[string]string{}
	admitted, err := runner.reserve(context.Background(), "model|gpu:"+testGPU, "job", func(ctx context.Context) (*leaseGroup, error) {
		meta["gpu_uuid"] = testGPU
		meta[gpuCapacityClaimBytesMetadataKey] = strconv.FormatInt(6<<30, 10)
		return newLeaseGroup(ctx), nil
	}, meta)
	if err != nil {
		t.Fatal(err)
	}
	if _, idle := runner.idleGPUResident(testGPU); idle {
		t.Fatal("a reserved worker reported itself idle")
	}
	_ = admitted
	if runner.residentGPUBytes != 6<<30 || runner.residentGPUUUID != testGPU {
		t.Fatalf("resident GPU claim not remembered: %d on %q", runner.residentGPUBytes, runner.residentGPUUUID)
	}
}

// laggingBroker refuses a claim for the first refusals calls after any release,
// as the host broker does while its snapshot still shows the unloaded worker.
type laggingBroker struct {
	*ledgerBroker
	pending int
}

func (b *laggingBroker) Claim(ctx context.Context, owner string, bytes int64, opts capacity.ClaimOptions) (capacity.Lease, error) {
	if b.pending > 0 {
		b.pending--
		b.mu.Lock()
		b.claims++
		b.mu.Unlock()
		return capacity.Lease{DegradeToCPU: true, Warnings: []string{"stale snapshot"}}, nil
	}
	return b.ledgerBroker.Claim(ctx, owner, bytes, opts)
}

func TestTheRetryOutlastsTheBrokersSnapshotLagAfterAnUnload(t *testing.T) {
	ledger := &ledgerBroker{total: 6 << 30}
	broker := &laggingBroker{ledgerBroker: ledger}
	eng, reg := newYieldEngine(ledger)
	eng.deps.Capacity = broker
	eng.yield = yieldPacing{settle: time.Second, poll: time.Millisecond}
	idle := addResident(t, reg, ledger, "florence", "florence-2", testGPU, 3<<30, time.Now())
	holdForAnotherOwner(t, ledger, 3<<30)
	meta := map[string]string{}
	broker.pending = 3
	lease, err := eng.claimGPU(context.Background(), "image-tools:job", 3<<30, capacity.ClaimOptions{GPUUUID: testGPU}, meta)
	if err != nil || lease.DegradeToCPU {
		t.Fatalf("lease=%#v err=%v", lease, err)
	}
	if idle.loaded() || meta[evictedResidentsMetadataKey] != "florence-2" {
		t.Fatalf("loaded=%v evicted=%q", idle.loaded(), meta[evictedResidentsMetadataKey])
	}
}

func TestTheRetryGivesUpWhenTheSettleWindowEnds(t *testing.T) {
	ledger := &ledgerBroker{total: 6 << 30}
	broker := &laggingBroker{ledgerBroker: ledger, pending: 1 << 30}
	eng, reg := newYieldEngine(ledger)
	eng.deps.Capacity = broker
	eng.yield = yieldPacing{settle: 20 * time.Millisecond, poll: 5 * time.Millisecond}
	addResident(t, reg, ledger, "florence", "florence-2", testGPU, 3<<30, time.Now())
	lease, err := eng.claimGPU(context.Background(), "image-tools:job", 3<<30, capacity.ClaimOptions{GPUUUID: testGPU}, map[string]string{})
	if err != nil || !lease.DegradeToCPU {
		t.Fatalf("a claim the broker keeps refusing must degrade: lease=%#v err=%v", lease, err)
	}
}
