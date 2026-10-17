package composition

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/api-core/blobstore"
	testdb "github.com/vrooli/api-core/databasetest"
	musicstorage "music-tools/internal/storage"

	_ "modernc.org/sqlite"
)

func TestPoolPersistsTakesAndPolicy(t *testing.T) {
	db, err := sql.Open("sqlite", "file:pool-persistence?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	first, err := NewPoolWithStore(db)
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Configure("style", 8, 3); err != nil {
		t.Fatal(err)
	}
	first.SetLastReplenishJob("style", "replenish-job")
	takes, err := PlanBatch(BatchRequest{Duration: 45, JobID: "job", StyleID: "style", Caption: "sound", Takes: 1, Seed: 4})
	if err != nil {
		t.Fatal(err)
	}
	first.Add(takes...)
	second, err := NewPoolWithStore(db)
	if err != nil {
		t.Fatal(err)
	}
	available, _, target, threshold := second.Status("style")
	if available != 1 || target != 8 || threshold != 3 {
		t.Fatalf("pool did not restore persisted state: available=%d target=%d threshold=%d", available, target, threshold)
	}
	if got := second.LastReplenishJob("style"); got != "replenish-job" {
		t.Fatalf("last replenishment job did not persist: %q", got)
	}
}

type recordingBlobPolicy struct{ states map[string][2]any }

func (p *recordingBlobPolicy) SetProtected(key string, protected bool) {
	state := p.states[key]
	state[0] = protected
	p.states[key] = state
}

func (p *recordingBlobPolicy) SetEvictionPriority(key string, priority int) {
	state := p.states[key]
	state[1] = priority
	p.states[key] = state
}

func TestPoolTransitionsUpdateBlobPolicy(t *testing.T) {
	p := NewPool()
	policy := &recordingBlobPolicy{states: map[string][2]any{}}
	p.SetBlobPolicy(policy)
	takes, err := PlanBatch(BatchRequest{Duration: 45, JobID: "job", StyleID: "style", Caption: "sound", Takes: 1, Seed: 4})
	if err != nil {
		t.Fatal(err)
	}
	takes[0].BlobRef = "out/take.wav"
	p.Add(takes[0])
	if _, _, err := p.Draw("style", "holder"); err != nil {
		t.Fatal(err)
	}
	if state := policy.states["out/take.wav"]; state != [2]any{true, 1} {
		t.Fatalf("reserved policy=%v", state)
	}
	if err := p.Discard(takes[0].ID); err != nil {
		t.Fatal(err)
	}
	if state := policy.states["out/take.wav"]; state != [2]any{false, 0} {
		t.Fatalf("discarded policy=%v", state)
	}
}

func TestConcurrentDrawsAreExclusive(t *testing.T) {
	pool := NewPool()
	takes, err := PlanBatch(BatchRequest{Duration: 45, JobID: "job", StyleID: "style", Caption: "sound", Takes: 1, Seed: 4})
	if err != nil {
		t.Fatal(err)
	}
	pool.Add(takes...)
	var wg sync.WaitGroup
	results := make(chan Take, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			take, _, err := pool.Draw("style", "consumer")
			if err == nil {
				results <- take
			}
		}()
	}
	wg.Wait()
	close(results)
	count := 0
	for range results {
		count++
	}
	if count != 1 {
		t.Fatalf("two concurrent draws returned %d takes", count)
	}
}

func TestReserveTargetsRequestedTake(t *testing.T) {
	pool := NewPool()
	takes, err := PlanBatch(BatchRequest{Duration: 45, JobID: "job", StyleID: "style", Caption: "sound", Takes: 2, Seed: 4})
	if err != nil {
		t.Fatal(err)
	}
	pool.Add(takes...)

	reserved, _, err := pool.Reserve(takes[1].ID, "operator")
	if err != nil {
		t.Fatal(err)
	}
	if reserved.ID != takes[1].ID || reserved.PoolState != "reserved" || reserved.ReservedBy != "operator" {
		t.Fatalf("reserved=%+v, want exact requested take", reserved)
	}
	got, ok := pool.Get(takes[0].ID)
	if !ok || got.PoolState != "available" {
		t.Fatalf("unrequested take state=%q, want available", got.PoolState)
	}
}

func TestPartialBatchRetention(t *testing.T) {
	pool := NewPool()
	takes, err := PlanBatch(BatchRequest{Duration: 45, JobID: "job", StyleID: "style", Caption: "sound", Takes: 3, Seed: 4})
	if err != nil {
		t.Fatal(err)
	}
	pool.Add(takes[0], takes[1])
	available, _, _, _ := pool.Status("style")
	if available != 2 {
		t.Fatalf("partial batch discarded completed takes: %d", available)
	}
}

func TestReplenishmentIsOnConsumeAndDeduplicated(t *testing.T) {
	pool := NewPool()
	if err := pool.Configure("style", 10, 4); err != nil {
		t.Fatal(err)
	}
	takes, _ := PlanBatch(BatchRequest{Duration: 45, JobID: "job", StyleID: "style", Caption: "sound", Takes: 1, Seed: 4})
	pool.Add(takes...)
	take, replenish, err := pool.Draw("style", "consumer")
	if err != nil || !replenish {
		t.Fatalf("draw should trigger replenish: take=%#v replenish=%v err=%v", take, replenish, err)
	}
	ok, needed := pool.ConfigureReplenishment("style")
	if !ok || needed != 9 {
		t.Fatalf("expected one in-flight replenishment for 9 takes, got ok=%v needed=%d", ok, needed)
	}
	ok, _ = pool.ConfigureReplenishment("style")
	if ok {
		t.Fatal("duplicate replenishment was admitted")
	}
	pool.ReplenishmentFinished("style")
}

func TestConsumeTriggersOneReplenishmentBatch(t *testing.T) {
	p := NewPool()
	if err := p.Configure("style", 4, 2); err != nil {
		t.Fatal(err)
	}
	takes, _ := PlanBatch(BatchRequest{Duration: 45, JobID: "job", StyleID: "style", Caption: "sound", Takes: 1, Seed: 4})
	p.Add(takes...)
	var style string
	var needed int
	p.SetReplenishTrigger(func(gotStyle string, gotNeeded int) { style, needed = gotStyle, gotNeeded })
	if _, _, err := p.Draw("style", "holder"); err != nil {
		t.Fatal(err)
	}
	if _, err := p.MarkConsumed(takes[0].ID); err != nil {
		t.Fatal(err)
	}
	if style != "style" || needed != 3 {
		t.Fatalf("trigger style=%q needed=%d", style, needed)
	}
	// A burst cannot stack a second trigger while the first batch is in flight.
	if _, err := p.MarkConsumed(takes[0].ID); err == nil {
		t.Fatal("consumed take was offered twice")
	}
}

type failingPoolDB struct {
	*sql.DB
	fail bool
}

func (d *failingPoolDB) ExecContext(ctx context.Context, q string, args ...any) (sql.Result, error) {
	if d.fail {
		return nil, errors.New("injected pool outage")
	}
	return d.DB.ExecContext(ctx, q, args...)
}

func TestPoolFailedMutationPreservesObservableState(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	store := &failingPoolDB{DB: db}
	pool, err := NewPoolWithStore(store)
	if err != nil {
		t.Fatal(err)
	}
	takes, err := PlanBatch(BatchRequest{JobID: "test", StyleID: "s", Caption: "sound", Takes: 1, Duration: 45})
	if err != nil {
		t.Fatal(err)
	}
	store.fail = true
	if err = pool.Add(takes...); err == nil {
		t.Fatal("failed Add accepted")
	}
	if a, _, _, _ := pool.Status("s"); a != 0 {
		t.Fatal("failed take became observable")
	}
	store.fail = false
	if err = pool.Add(takes...); err != nil {
		t.Fatal(err)
	}
	store.fail = true
	if _, _, err = pool.Reserve(takes[0].ID, "reviewer"); err == nil {
		t.Fatal("failed reservation accepted")
	}
	take, _ := pool.Get(takes[0].ID)
	if take.PoolState != "available" {
		t.Fatal("failed reserve changed memory")
	}
	if err = pool.Configure("s", 10, 2); err == nil {
		t.Fatal("failed policy accepted")
	}
}

func (p *recordingBlobPolicy) ProtectAndCommit(key string, commit func() error) (bool, error) {
	if err := commit(); err != nil {
		return false, err
	}
	p.SetProtected(key, true)
	return true, nil
}

type reservationBarrierDB struct {
	*sql.DB
	entered, proceed chan struct{}
}

func (d *reservationBarrierDB) ExecContext(ctx context.Context, q string, args ...any) (sql.Result, error) {
	if strings.Contains(q, "INSERT INTO music_pool_takes") && len(args) > 3 && args[3] == "reserved" {
		close(d.entered)
		select {
		case <-d.proceed:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return d.DB.ExecContext(ctx, q, args...)
}

func TestDurableReservationFencesCleanupDuringPersistence(t *testing.T) {
	db := testdb.NewSQLite(t)
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	barrier := &reservationBarrierDB{DB: db, entered: make(chan struct{}), proceed: make(chan struct{})}
	pool, err := NewPoolWithStore(barrier)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	store := musicstorage.NewWithBlobStore(blobstore.NewFilesystemBlobStore(root), root)
	pool.SetBlobPolicy(store)
	take := Take{ID: "take", StyleID: "s", BlobRef: "out/take.wav", CreatedAt: time.Now().UTC()}
	if err = store.PutOwned(t.Context(), take.BlobRef, take.ID, strings.NewReader("audio"), "audio/wav", true); err != nil {
		t.Fatal(err)
	}
	if err = pool.Add(take); err != nil {
		t.Fatal(err)
	}
	snapshot := store.Entries()[0]
	reserved := make(chan error, 1)
	go func() { _, _, err := pool.Reserve(take.ID, "listener"); reserved <- err }()
	<-barrier.entered
	deleting := make(chan struct{})
	deleted := make(chan bool, 1)
	go func() { close(deleting); ok, _ := store.DeleteEligible(t.Context(), snapshot); deleted <- ok }()
	<-deleting
	close(barrier.proceed)
	if err = <-reserved; err != nil {
		t.Fatal(err)
	}
	if <-deleted {
		t.Fatal("cleanup deleted in-flight durable reservation")
	}
	reloaded, err := NewPoolWithStore(db)
	if err != nil {
		t.Fatal(err)
	}
	persisted, _ := reloaded.Get(take.ID)
	if persisted.PoolState != "reserved" {
		t.Fatalf("durable state %+v", persisted)
	}
	if _, _, err = store.Get(t.Context(), take.BlobRef); err != nil {
		t.Fatal("lost reserved audio", err)
	}
}

type selectionFailureBlobStore struct {
	blobstore.BlobStore
	receiptErr error
	getErr     error
}

func (s *selectionFailureBlobStore) Delete(ctx context.Context, key string) error {
	if strings.HasPrefix(key, "ownership/") && s.receiptErr != nil {
		return s.receiptErr
	}
	return s.BlobStore.Delete(ctx, key)
}
func (s *selectionFailureBlobStore) Get(ctx context.Context, key string) (io.ReadCloser, string, error) {
	if s.getErr != nil {
		return nil, "", s.getErr
	}
	return s.BlobStore.Get(ctx, key)
}

func TestMissingAvailableAudioIsRetiredDurablyAndRequestsRefill(t *testing.T) {
	for _, loss := range []string{"managed-delete", "physical-only", "receipt-delete-failure", "transient-read"} {
		for _, selection := range []string{"reserve", "draw"} {
			t.Run(loss+"/"+selection, func(t *testing.T) {
				db := testdb.NewSQLite(t)
				if _, err := db.Exec(Schema()); err != nil {
					t.Fatal(err)
				}
				pool, err := NewPoolWithStore(db)
				if err != nil {
					t.Fatal(err)
				}
				root := t.TempDir()
				injected := errors.New("injected storage I/O failure")
				substrate := &selectionFailureBlobStore{BlobStore: blobstore.NewFilesystemBlobStore(root)}
				store := musicstorage.NewWithBlobStore(substrate, root)
				pool.SetBlobPolicy(store)
				if err = pool.Configure("s", 2, 1); err != nil {
					t.Fatal(err)
				}
				take := Take{ID: "take", StyleID: "s", BlobRef: "out/take.wav", CreatedAt: time.Now().UTC()}
				if err = store.PutOwned(t.Context(), take.BlobRef, take.ID, strings.NewReader("audio"), "audio/wav", true); err != nil {
					t.Fatal(err)
				}
				if err = pool.Add(take); err != nil {
					t.Fatal(err)
				}
				switch loss {
				case "physical-only":
					if err = os.Remove(filepath.Join(root, take.BlobRef)); err != nil {
						t.Fatal(err)
					}
					if len(store.Entries()) != 1 {
						t.Fatal("test requires stale catalog")
					}
				case "managed-delete", "receipt-delete-failure":
					if loss == "receipt-delete-failure" {
						substrate.receiptErr = injected
					}
					removed, deleteErr := store.DeleteEligible(t.Context(), store.Entries()[0])
					if loss == "managed-delete" && (deleteErr != nil || !removed) {
						t.Fatalf("cleanup: %v %v", removed, deleteErr)
					}
					if loss == "receipt-delete-failure" && !errors.Is(deleteErr, injected) {
						t.Fatal("receipt error lost", deleteErr)
					}
				case "transient-read":
					substrate.getErr = injected
				}
				refill := 0
				pool.SetReplenishTrigger(func(style string, needed int) {
					if style != "s" {
						t.Error("wrong style")
					}
					refill = needed
				})
				selectTake := func(p *Pool) error {
					if selection == "reserve" {
						_, _, e := p.Reserve(take.ID, "listener")
						return e
					}
					_, _, e := p.Draw("s", "listener")
					return e
				}
				err = selectTake(pool)
				reloaded, loadErr := NewPoolWithStore(db)
				if loadErr != nil {
					t.Fatal(loadErr)
				}
				persisted, _ := reloaded.Get(take.ID)
				if loss == "transient-read" {
					if !errors.Is(err, injected) || errors.Is(err, ErrUnavailable) {
						t.Fatal("transient failure misclassified", err)
					}
					if persisted.PoolState != "available" || refill != 0 {
						t.Fatalf("transient failure retired/refilled: %+v %d", persisted, refill)
					}
					return
				}
				if !errors.Is(err, ErrUnavailable) || refill != 2 {
					t.Fatalf("absent audio counted/granted: %v refill=%d", err, refill)
				}
				if persisted.PoolState != "unavailable" {
					t.Fatalf("not durably retired: %+v", persisted)
				}
				for _, p := range []*Pool{pool, reloaded} {
					p.SetBlobPolicy(store)
					if available, _, _, _ := p.Status("s"); available != 0 {
						t.Fatal("missing audio counted ready")
					}
					if err = selectTake(p); !errors.Is(err, ErrUnavailable) {
						t.Fatal("repeat selected missing source", err)
					}
				}
			})
		}
	}
}
