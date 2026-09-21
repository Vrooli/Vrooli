package composition

import (
	"database/sql"
	"sync"
	"testing"

	_ "modernc.org/sqlite"
)

func TestPoolPersistsTakesAndPolicy(t *testing.T) {
	db, err := sql.Open("sqlite", "file:pool-persistence?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	first := NewPoolWithStore(db)
	if err := first.Configure("style", 8, 3); err != nil {
		t.Fatal(err)
	}
	first.SetLastReplenishJob("style", "replenish-job")
	takes, err := PlanBatch(BatchRequest{JobID: "job", StyleID: "style", Caption: "sound", Takes: 1, Seed: 4})
	if err != nil {
		t.Fatal(err)
	}
	first.Add(takes...)
	second := NewPoolWithStore(db)
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
	takes, err := PlanBatch(BatchRequest{JobID: "job", StyleID: "style", Caption: "sound", Takes: 1, Seed: 4})
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
	takes, err := PlanBatch(BatchRequest{JobID: "job", StyleID: "style", Caption: "sound", Takes: 1, Seed: 4})
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
	takes, err := PlanBatch(BatchRequest{JobID: "job", StyleID: "style", Caption: "sound", Takes: 2, Seed: 4})
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
	takes, err := PlanBatch(BatchRequest{JobID: "job", StyleID: "style", Caption: "sound", Takes: 3, Seed: 4})
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
	takes, _ := PlanBatch(BatchRequest{JobID: "job", StyleID: "style", Caption: "sound", Takes: 1, Seed: 4})
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
	takes, _ := PlanBatch(BatchRequest{JobID: "job", StyleID: "style", Caption: "sound", Takes: 1, Seed: 4})
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
