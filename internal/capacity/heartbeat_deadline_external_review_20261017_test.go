package capacity

import (
	"context"
	"errors"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"
)

// externalReviewHeartbeatClock advances independently of the transaction wait.
type externalReviewHeartbeatClock struct {
	mu sync.Mutex
	at time.Time
}

func (c *externalReviewHeartbeatClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.at
}

func TestExternalReviewOperationHeartbeatCannotCrossDeadlineWhileWaitingForTransaction(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	clk := &externalReviewHeartbeatClock{at: time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)}
	store, err := NewSQLiteStore(ctx, Config{DBPath: filepath.Join(t.TempDir(), "capacity.db"), Clock: clk})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	claim, err := store.CreateClaim(ctx, CapacityClaim{
		OwnerKind: OwnerKindOp, OwnerID: "external-review:deadline",
		ResourceKind: ResourceKindRAM, AmountBytes: 1, PreferredBytes: 1,
		Status: StatusGranted,
	}, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	// The store has MaxOpenConns(1). Holding that connection makes BeginTx wait
	// until after the captured pre-transaction time has passed the lease deadline.
	connection, err := store.db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	waitsBefore := store.db.Stats().WaitCount
	done := make(chan error, 1)
	go func() {
		_, heartbeatErr := store.HeartbeatClaim(ctx, claim.ClaimID, claim.Generation, time.Minute)
		done <- heartbeatErr
	}()
	// Prove the caller is queued for the held SQL connection without requiring
	// its clock to be read before that queue. Correct implementations may read
	// the clock only after acquiring transaction/write authority.
	for store.db.Stats().WaitCount <= waitsBefore {
		select {
		case <-ctx.Done():
			t.Fatal("heartbeat did not queue for the held connection")
		default:
			runtime.Gosched()
		}
	}
	clk.mu.Lock()
	clk.at = clk.at.Add(2 * time.Second)
	clk.mu.Unlock()
	if err := connection.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case heartbeatErr := <-done:
		if !errors.Is(heartbeatErr, ErrStaleGeneration) {
			t.Fatalf("operation heartbeat committed after its lease deadline: got %v, want ErrStaleGeneration", heartbeatErr)
		}
	case <-ctx.Done():
		t.Fatal("heartbeat did not settle after the held connection was released")
	}
	got, err := store.GetClaim(ctx, claim.ClaimID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Generation != claim.Generation || got.HeartbeatDeadlineAt == nil || !got.HeartbeatDeadlineAt.Equal(*claim.HeartbeatDeadlineAt) || got.LastHeartbeatAt == nil || !got.LastHeartbeatAt.Equal(*claim.LastHeartbeatAt) {
		t.Fatalf("rejected renewal changed the authoritative claim: before=%+v after=%+v", claim, got)
	}
}
