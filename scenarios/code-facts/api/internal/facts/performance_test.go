package facts

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
)

func TestWeightedAdmissionCancellationReleasesQueueCapacity(t *testing.T) {
	admission := NewWeightedAdmission(4, 1, time.Second)
	release, err := admission.Acquire(context.Background(), "embedding", 4)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := admission.Acquire(ctx, "query", 1)
		done <- err
	}()
	for deadline := time.Now().Add(time.Second); admission.Snapshot().Queued != 1 && time.Now().Before(deadline); {
		time.Sleep(time.Millisecond)
	}
	cancel()
	if err := <-done; err == nil {
		t.Fatal("cancelled admission wait succeeded")
	}
	release()
	snapshot := admission.Snapshot()
	if snapshot.InUse != 0 || snapshot.Queued != 0 || snapshot.Cancelled != 1 || snapshot.HighWater != 4 {
		t.Fatalf("unexpected admission snapshot after cancellation: %+v", snapshot)
	}
}

func TestGraphFlightCoalescesIdenticalWork(t *testing.T) {
	group := newGraphFlightGroup()
	start := make(chan struct{})
	release := make(chan struct{})
	var calls atomic.Int32
	fn := func(context.Context) (*GraphResult, error) {
		calls.Add(1)
		close(start)
		<-release
		return cacheTestGraph("coalesced"), nil
	}
	var wg sync.WaitGroup
	wg.Add(2)
	results := make(chan *GraphResult, 2)
	go func() {
		defer wg.Done()
		result, _ := group.Do(context.Background(), "same", fn)
		results <- result
	}()
	<-start
	go func() {
		defer wg.Done()
		result, _ := group.Do(context.Background(), "same", fn)
		results <- result
	}()
	for deadline := time.Now().Add(time.Second); group.waiterCount("same") != 1 && time.Now().Before(deadline); {
		time.Sleep(time.Millisecond)
	}
	close(release)
	wg.Wait()
	close(results)
	if calls.Load() != 1 {
		t.Fatalf("coalesced function calls = %d, want 1", calls.Load())
	}
	for result := range results {
		if result == nil || result.GraphHash != "coalesced" {
			t.Fatalf("coalesced result = %#v", result)
		}
	}
}

func TestAssertFamilyCostFailsWhenSyntheticBudgetIsLowered(t *testing.T) {
	cost := FamilyCost{Family: "file_domain", Target: "scenario:search-hub", ColdMS: 420, WarmMS: 95}
	if err := AssertFamilyCost(cost, 1000, 200); err != nil {
		t.Fatalf("measured cost rejected by normal budget: %v", err)
	}
	if err := AssertFamilyCost(cost, 100, 200); err == nil {
		t.Fatal("lowered synthetic cold budget did not fail")
	}
}

func TestUnitFingerprintInvalidatesOnlyTouchedParseUnit(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "first")
	second := filepath.Join(root, "second")
	if err := os.MkdirAll(first, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(second, 0o755); err != nil {
		t.Fatal(err)
	}
	firstFile := filepath.Join(first, "a.go")
	secondFile := filepath.Join(second, "b.go")
	if err := os.WriteFile(firstFile, []byte("package first\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(secondFile, []byte("package second\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	firstUnit := &factsv1.ParseUnit{RootPath: first}
	secondUnit := &factsv1.ParseUnit{RootPath: second}
	firstBefore, _ := sourceFingerprintForUnit(firstUnit)
	secondBefore, _ := sourceFingerprintForUnit(secondUnit)
	if err := os.WriteFile(firstFile, []byte("package first\n\nvar changed = true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	firstAfter, _ := sourceFingerprintForUnit(firstUnit)
	secondAfter, _ := sourceFingerprintForUnit(secondUnit)
	if firstBefore == firstAfter {
		t.Fatal("touched parse unit retained its source fingerprint")
	}
	if secondBefore != secondAfter {
		t.Fatal("untouched parse unit fingerprint changed")
	}
}
