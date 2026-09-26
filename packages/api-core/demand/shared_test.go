package demand

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// countingClient records how many control-plane round trips a caller causes.
// Each acquire and release is a process spawn and a write to the runtime
// registry, so the counts are the thing under test, not an implementation
// detail.
type countingClient struct {
	acquires atomic.Int64
	releases atomic.Int64

	mu         sync.Mutex
	acquireErr error
	block      chan struct{}
}

func (c *countingClient) Acquire(ctx context.Context, req AcquireRequest) (Lease, error) {
	c.mu.Lock()
	block, err := c.block, c.acquireErr
	c.mu.Unlock()
	if block != nil {
		select {
		case <-block:
		case <-ctx.Done():
			return Lease{}, ctx.Err()
		}
	}
	if err != nil {
		return Lease{}, err
	}
	n := c.acquires.Add(1)
	ttl := req.TTL
	if ttl <= 0 {
		ttl = time.Minute
	}
	return Lease{LeaseID: fmt.Sprintf("demand_%d", n), Scenario: req.Scenario, ConsumerID: req.ConsumerID, Kind: req.Kind, Status: "active", ExpiresAt: time.Now().Add(ttl)}, nil
}

func (c *countingClient) Renew(context.Context, string, time.Duration) (Lease, error) {
	return Lease{Status: "active", ExpiresAt: time.Now().Add(time.Minute)}, nil
}

func (c *countingClient) Release(_ context.Context, leaseID, _ string) (Lease, error) {
	c.releases.Add(1)
	return Lease{LeaseID: leaseID, Status: "released"}, nil
}

func TestConcurrentCallersAtOneScenarioShareOneLease(t *testing.T) {
	client := &countingClient{}
	client.block = make(chan struct{})
	holder := NewSharedHolder(client, "test")

	const callers = 8
	holds := make([]*SharedHold, callers)
	errs := make([]error, callers)
	var wg sync.WaitGroup
	for i := range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			holds[i], errs[i] = holder.Acquire(context.Background(), AcquireRequest{
				Scenario: "source-ledger", ConsumerID: "sess-1", Kind: KindProgram, TTL: time.Minute,
			})
		}()
	}
	// Hold the first acquire open so every caller is genuinely overlapping.
	time.Sleep(50 * time.Millisecond)
	close(client.block)
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("caller %d: %v", i, err)
		}
	}
	if got := client.acquires.Load(); got != 1 {
		t.Fatalf("8 concurrent callers at one scenario caused %d acquires, want 1", got)
	}
	if holder.Len() != 1 {
		t.Fatalf("holder tracks %d leases, want 1", holder.Len())
	}

	// The lease survives every caller but the last.
	for i, hold := range holds[:callers-1] {
		if err := hold.Close(); err != nil {
			t.Fatalf("close %d: %v", i, err)
		}
		if got := client.releases.Load(); got != 0 {
			t.Fatalf("lease released while %d callers still hold it", callers-1-i)
		}
	}
	if err := holds[callers-1].Close(); err != nil {
		t.Fatalf("final close: %v", err)
	}
	if got := client.releases.Load(); got != 1 {
		t.Fatalf("releases = %d, want 1", got)
	}
	if holder.Len() != 0 {
		t.Fatalf("holder retained %d leases after the last close", holder.Len())
	}
}

func TestDifferentScenariosDoNotShareALease(t *testing.T) {
	client := &countingClient{}
	holder := NewSharedHolder(client, "test")
	for _, scenario := range []string{"source-ledger", "swarm-manager", "command-center"} {
		hold, err := holder.Acquire(context.Background(), AcquireRequest{Scenario: scenario, ConsumerID: "sess-1", Kind: KindProgram})
		if err != nil {
			t.Fatalf("%s: %v", scenario, err)
		}
		defer hold.Close()
	}
	if got := client.acquires.Load(); got != 3 {
		t.Fatalf("acquires = %d, want 3", got)
	}
	if holder.Len() != 3 {
		t.Fatalf("holder tracks %d leases, want 3", holder.Len())
	}
}

func TestSequentialCallersDoNotOutliveTheirLease(t *testing.T) {
	client := &countingClient{}
	holder := NewSharedHolder(client, "test")
	for range 3 {
		hold, err := holder.Acquire(context.Background(), AcquireRequest{Scenario: "source-ledger", ConsumerID: "sess-1", Kind: KindProgram})
		if err != nil {
			t.Fatal(err)
		}
		if err := hold.Close(); err != nil {
			t.Fatal(err)
		}
	}
	// Nothing lingers: three sequential units of work, three matched pairs.
	if a, r := client.acquires.Load(), client.releases.Load(); a != 3 || r != 3 {
		t.Fatalf("acquires=%d releases=%d, want 3/3", a, r)
	}
}

func TestOneCallerClosingDoesNotCancelItsSiblings(t *testing.T) {
	client := &countingClient{}
	holder := NewSharedHolder(client, "test")
	first, err := holder.Acquire(context.Background(), AcquireRequest{Scenario: "source-ledger", ConsumerID: "sess-1", Kind: KindProgram})
	if err != nil {
		t.Fatal(err)
	}
	second, err := holder.Acquire(context.Background(), AcquireRequest{Scenario: "source-ledger", ConsumerID: "sess-1", Kind: KindProgram})
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	if err := second.Context().Err(); err != nil {
		t.Fatalf("sibling context cancelled by an unrelated caller: %v", err)
	}
	if got := client.releases.Load(); got != 0 {
		t.Fatal("lease released while a sibling still held it")
	}
	if err := second.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestCallerCancellationDoesNotCancelTheSharedLease(t *testing.T) {
	client := &countingClient{}
	holder := NewSharedHolder(client, "test")
	cancellable, cancel := context.WithCancel(context.Background())
	first, err := holder.Acquire(cancellable, AcquireRequest{Scenario: "source-ledger", ConsumerID: "sess-1", Kind: KindProgram})
	if err != nil {
		t.Fatal(err)
	}
	second, err := holder.Acquire(context.Background(), AcquireRequest{Scenario: "source-ledger", ConsumerID: "sess-1", Kind: KindProgram})
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	if err := first.Context().Err(); err == nil {
		t.Fatal("the cancelled caller kept a live context")
	}
	if err := second.Context().Err(); err != nil {
		t.Fatalf("cancelling the lease creator cancelled a sibling: %v", err)
	}
	_ = first.Close()
	if err := second.Context().Err(); err != nil {
		t.Fatalf("sibling lost its context after the creator closed: %v", err)
	}
	_ = second.Close()
}

func TestAcquireFailureIsNotCachedAsALiveLease(t *testing.T) {
	client := &countingClient{}
	client.acquireErr = errors.New("control plane unavailable")
	holder := NewSharedHolder(client, "test")
	if _, err := holder.Acquire(context.Background(), AcquireRequest{Scenario: "source-ledger", ConsumerID: "sess-1", Kind: KindProgram}); err == nil {
		t.Fatal("expected acquire failure")
	}
	if holder.Len() != 0 {
		t.Fatalf("failed acquire left %d entries behind", holder.Len())
	}
	client.mu.Lock()
	client.acquireErr = nil
	client.mu.Unlock()
	hold, err := holder.Acquire(context.Background(), AcquireRequest{Scenario: "source-ledger", ConsumerID: "sess-1", Kind: KindProgram})
	if err != nil {
		t.Fatalf("recovery acquire: %v", err)
	}
	defer hold.Close()
}

func TestNilHolderKeepsLeaseFreeCallersWorking(t *testing.T) {
	var holder *SharedHolder
	hold, err := holder.Acquire(context.Background(), AcquireRequest{Scenario: "x"})
	if err != nil || hold != nil {
		t.Fatalf("nil holder must be inert: hold=%v err=%v", hold, err)
	}
	if err := hold.Close(); err != nil {
		t.Fatalf("nil hold Close: %v", err)
	}
	if hold.Context() == nil {
		t.Fatal("nil hold must still yield a usable context")
	}
}

func TestIsContentionSeparatesRegistryLockFromScenarioOutage(t *testing.T) {
	contended := []string{
		"acquire demand lease for source-ledger: demand control-plane command: exit status 1: create demand lease: database is locked (517)",
		"create demand lease: database is locked (5) (SQLITE_BUSY)",
		"table in the database is locked",
	}
	for _, msg := range contended {
		if !IsContention(errors.New(msg)) {
			t.Fatalf("registry lock not recognised: %s", msg)
		}
	}
	other := []string{
		"binding source-ledger/journal/list is unreachable: no running runtime ports found",
		"remote status 404 Not Found",
		"",
	}
	for _, msg := range other {
		if msg != "" && IsContention(errors.New(msg)) {
			t.Fatalf("scenario outage misread as contention: %s", msg)
		}
	}
	if IsContention(nil) {
		t.Fatal("nil is not contention")
	}
}

// registryBusyWindow mirrors busy_timeout(10000) in the runtime registry DSN
// (buildDSN, internal/scenarioruntime). The two modules cannot import each
// other, so the coupling is pinned from both sides instead.
const registryBusyWindow = 10 * time.Second

func TestControlPlaneTimeoutOutlastsRegistryBusyWindow(t *testing.T) {
	if ControlPlaneTimeout <= registryBusyWindow {
		t.Fatalf("ControlPlaneTimeout %s does not outlast the registry busy window %s: a writer waiting its turn would be killed by its own caller", ControlPlaneTimeout, registryBusyWindow)
	}
	if ControlPlaneReleaseTimeout >= ControlPlaneTimeout {
		t.Fatalf("release timeout %s should stay below the acquisition timeout %s", ControlPlaneReleaseTimeout, ControlPlaneTimeout)
	}
}

// deadLeaseClient fails renewal after the first lease so the shared hold dies
// while a caller still holds a reference to it.
type deadLeaseClient struct {
	countingClient
	renewFails atomic.Bool
}

func (c *deadLeaseClient) Renew(ctx context.Context, id string, ttl time.Duration) (Lease, error) {
	if c.renewFails.Load() {
		return Lease{}, errors.New("control plane unavailable")
	}
	return c.countingClient.Renew(ctx, id, ttl)
}

func TestADeadLeaseDoesNotStallLaterCallers(t *testing.T) {
	client := &deadLeaseClient{}
	holder := NewSharedHolder(client, "test")

	// A short TTL makes the renewal loop tick quickly.
	stuck, err := holder.Acquire(context.Background(), AcquireRequest{Scenario: "source-ledger", ConsumerID: "sess-1", Kind: KindProgram, TTL: 30 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	client.renewFails.Store(true)

	deadline := time.Now().Add(2 * time.Second)
	for stuck.Context().Err() == nil {
		if time.Now().After(deadline) {
			t.Fatal("shared lease never noticed its renewal failure")
		}
		time.Sleep(5 * time.Millisecond)
	}
	client.renewFails.Store(false)

	// The dead lease still has a holder. A new caller must get a fresh lease
	// rather than joining the dead one or failing.
	fresh, err := holder.Acquire(context.Background(), AcquireRequest{Scenario: "source-ledger", ConsumerID: "sess-1", Kind: KindProgram, TTL: time.Minute})
	if err != nil {
		t.Fatalf("a dead lease blocked a later caller: %v", err)
	}
	if fresh.Context().Err() != nil {
		t.Fatal("new caller received a dead context")
	}
	if got := client.acquires.Load(); got != 2 {
		t.Fatalf("acquires = %d, want 2 (the dead lease plus its replacement)", got)
	}
	_ = stuck.Close()
	if fresh.Context().Err() != nil {
		t.Fatal("releasing the dead lease disturbed the replacement")
	}
	_ = fresh.Close()
}
