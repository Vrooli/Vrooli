package capacity

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

func operationRAMFixture(t *testing.T) OperationReservationOptions {
	t.Helper()
	open, clk := admitTestStore(t)
	return OperationReservationOptions{OwnerID: "update:connector:operation", InstanceID: "operation", AmountBytes: 30, TTL: 10 * time.Minute, Admission: AdmitOptions{OpenStore: open, Clock: clk, EnforceEnv: EnforceAdvisory, Source: StaticSource{Inventory: hostinventory.Snapshot{Memory: hostinventory.Memory{TotalBytes: 100, AvailableBytes: 40}}}}}
}

func TestOperationRAMReservationPreservesPendingHeadroom(t *testing.T) {
	ctx := context.Background()
	opts := operationRAMFixture(t)
	first, err := ReserveOperationRAM(ctx, opts)
	if err != nil {
		t.Fatal(err)
	}
	if first.Status != StatusReserved || !first.Protected || first.ActivityState != ActivityActive {
		t.Fatalf("unsafe reservation: %+v", first)
	}
	replayed, err := ReserveOperationRAM(ctx, opts)
	if err != nil || replayed.ClaimID != first.ClaimID {
		t.Fatalf("replay allocated a different reservation: %+v %v", replayed, err)
	}
	other := opts
	other.OwnerID, other.InstanceID = "update:other:operation", "other"
	if _, err := ReserveOperationRAM(ctx, other); err == nil {
		t.Fatal("second operation spent pending headroom already reserved above resident RAM")
	}
	store, err := opts.Admission.OpenStore(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	claims, err := store.ListClaims(ctx, ClaimFilter{Statuses: ActiveClaimStatuses()})
	if err != nil || len(claims) != 1 {
		t.Fatalf("denied reservation mutated ledger: %+v %v", claims, err)
	}
	if _, err := store.ReleaseClaim(ctx, first.ClaimID); err != nil {
		t.Fatal(err)
	}
	if _, err := ReserveOperationRAM(ctx, other); err != nil {
		t.Fatalf("verified release did not restore headroom: %v", err)
	}
}

func TestOperationRAMReservationConcurrentAdmission(t *testing.T) {
	opts := operationRAMFixture(t)
	start := make(chan struct{})
	var wg sync.WaitGroup
	results := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			local := opts
			local.OwnerID, local.InstanceID = fmt.Sprintf("concurrent:%d", i), fmt.Sprintf("op-%d", i)
			_, err := ReserveOperationRAM(context.Background(), local)
			results <- err
		}(i)
	}
	close(start)
	wg.Wait()
	close(results)
	grants := 0
	for err := range results {
		if err == nil {
			grants++
		}
	}
	if grants != 1 {
		t.Fatalf("concurrent grants=%d, want exactly one within 40 bytes of free RAM", grants)
	}
}

func TestOperationRAMReservationRejectsChangedOwnershipAndExpiredLease(t *testing.T) {
	opts := operationRAMFixture(t)
	original, err := ReserveOperationRAM(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*OperationReservationOptions){func(o *OperationReservationOptions) { o.InstanceID = "foreign" }, func(o *OperationReservationOptions) { o.AmountBytes++ }} {
		changed := opts
		change(&changed)
		if _, err := ReserveOperationRAM(context.Background(), changed); err == nil {
			t.Fatal("reservation accepted changed ownership/budget")
		}
	}
	now := opts.Admission.Clock().Add(opts.TTL)
	originalOpen := opts.Admission.OpenStore
	store, err := originalOpen(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	path := store.(*SQLiteStore).dbPath
	store.Close()
	opts.Admission.Clock = func() time.Time { return now }
	opts.Admission.OpenStore = func(ctx context.Context) (AdmitStore, error) {
		return NewSQLiteStore(ctx, Config{DBPath: path, Clock: clockAdapter(opts.Admission.Clock)})
	}
	replacement, err := ReserveOperationRAM(context.Background(), opts)
	if err != nil || replacement.ClaimID == original.ClaimID {
		t.Fatalf("expired operation must receive a fresh decision and row: %+v %v", replacement, err)
	}
}

func TestRAMAdmissionTreatsZeroAvailableAsNoHeadroom(t *testing.T) {
	v := Decide(CapacityRequest{ResourceKind: ResourceKindRAM, PreferredBytes: 1, FloorBytes: 1, NoReclaim: true}, hostinventory.Snapshot{Memory: hostinventory.Memory{TotalBytes: 100}}, nil, DefaultPolicy(), time.Now())
	if v.Granted() {
		t.Fatalf("unknown/zero available RAM granted capacity: %+v", v)
	}
}
