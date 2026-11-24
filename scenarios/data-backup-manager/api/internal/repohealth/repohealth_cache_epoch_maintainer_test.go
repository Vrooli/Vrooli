package repohealth

import (
	"context"
	"sync"
	"testing"

	"data-backup-manager/internal/engine"
)

type epochBlockingOwner struct {
	*fakeEngine
	started chan struct{}
	release chan struct{}
	mu      sync.Mutex
	calls   int
	first   engine.RepoDamageScan
	second  engine.RepoDamageScan
}

func (o *epochBlockingOwner) RepoDamageScan(context.Context, string) (engine.RepoDamageScan, error) {
	o.mu.Lock()
	o.calls++
	call := o.calls
	o.mu.Unlock()
	if call == 1 {
		close(o.started)
		<-o.release
		return o.first, nil
	}
	return o.second, nil
}

func (o *epochBlockingOwner) scanCalls() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.calls
}

func TestUsageEngineDoesNotCacheOwnerScanAfterConcurrentInvalidation(t *testing.T) {
	old := engine.RepoDamageScan{
		Root: root, StartedAt: fixedNow.Format("2006-01-02T15:04:05.999999999Z07:00"),
		Files: 2, Bytes: 200, TempFiles: 1,
	}
	refreshed := engine.RepoDamageScan{
		Root: root, StartedAt: fixedNow.Add(1).Format("2006-01-02T15:04:05.999999999Z07:00"),
		Files: 3, Bytes: 300, TempFiles: 2,
	}
	owner := &epochBlockingOwner{
		fakeEngine: &fakeEngine{}, started: make(chan struct{}), release: make(chan struct{}),
		first: old, second: refreshed,
	}
	u := usageEngineFor(newFakeFS(readable), owner.fakeEngine, &memExceptions{})
	u.KopiaEngine = owner

	type result struct {
		scan engine.RepoDamageScan
		err  error
	}
	finished := make(chan result, 1)
	go func() {
		scan, err := u.RepoDamageScan(context.Background(), "elements-local")
		finished <- result{scan: scan, err: err}
	}()

	<-owner.started // owner scan has begun, but has not returned to the cache helper
	u.InvalidateScan(root)
	close(owner.release)
	first := <-finished
	if first.err != nil || first.scan.Bytes != old.Bytes {
		t.Fatalf("in-flight owner scan = %+v, %v; want its independent 200-byte response", first.scan, first.err)
	}

	got, err := u.RepoDamageScan(context.Background(), "elements-local")
	if err != nil {
		t.Fatal(err)
	}
	if got.Bytes != refreshed.Bytes || got.Files != refreshed.Files || got.TempFiles != refreshed.TempFiles {
		t.Fatalf("scan after invalidation = %+v; want fresh owner response %+v", got, refreshed)
	}
	if calls := owner.scanCalls(); calls != 2 {
		t.Fatalf("owner scan calls = %d, want 2 because the in-flight pre-invalidation response must not be cached", calls)
	}
}
