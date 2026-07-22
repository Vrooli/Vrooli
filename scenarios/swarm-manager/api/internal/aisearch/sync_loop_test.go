package aisearch

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/testutil"
)

// makeSyncLoop wires a SyncLoop over the test fakes, with RunOnce defaulting
// to "no work, no error" unless the caller seeds work into the backlog reader.
func makeSyncLoop(t *testing.T, interval time.Duration, disabled bool) (*SyncLoop, *fakeEmbedder, *fakeVectorStore, *fakeBacklogReader, *fakeGoalReader) {
	t.Helper()
	r, emb, bs, _, br, ir := newReconcilerForTest(t)
	loop := &SyncLoop{
		Reconciler: r,
		Interval:   interval,
		Disabled:   disabled,
		Clock:      time.Now,
	}
	return loop, emb, bs, br, ir
}

func TestSyncLoop_RunOnce_NoWork_NoApply(t *testing.T) {
	loop, emb, bs, _, _ := makeSyncLoop(t, 0, false)
	plan, _, err := loop.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.HasWork() {
		t.Errorf("expected no work, got %+v", plan)
	}
	if emb.callCount() != 0 {
		t.Errorf("expected no embed calls, got %d", emb.callCount())
	}
	if bs.upsertCalls != 0 {
		t.Errorf("expected no upsert calls, got %d", bs.upsertCalls)
	}
}

func TestSyncLoop_RunOnce_AppliesPlan(t *testing.T) {
	loop, emb, bs, br, _ := makeSyncLoop(t, 0, false)
	br.items = []backlog.BacklogItem{
		{Kind: backlog.KindFix, Name: "a"},
		{Kind: backlog.KindFix, Name: "b"},
	}
	_, res, err := loop.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.UpsertedBacklog != 2 {
		t.Errorf("expected 2 upserts, got %d", res.UpsertedBacklog)
	}
	if emb.callCount() != 2 || bs.upsertCalls != 2 {
		t.Errorf("expected 2 embeds + 2 upserts, got embeds=%d upserts=%d", emb.callCount(), bs.upsertCalls)
	}
}

func TestSyncLoop_RunOnce_PropagatesPlanError(t *testing.T) {
	loop, emb, bs, _, _ := makeSyncLoop(t, 0, false)
	bs.scrollErr = errors.New("qdrant down")
	loop.Reconciler.BacklogReader.(*fakeBacklogReader).items = []backlog.BacklogItem{
		{Kind: backlog.KindFix, Name: "x"},
	}
	_, _, err := loop.RunOnce(context.Background())
	if err == nil {
		t.Fatal("expected error to propagate")
	}
	if emb.callCount() != 0 {
		t.Errorf("expected no embeds when plan failed, got %d", emb.callCount())
	}
}

func TestSyncLoop_RunOnce_SwallowsBusyError(t *testing.T) {
	// Simulate concurrent ticks: the second RunOnce hits ErrReconcileBusy and
	// the SyncLoop must treat that as success (the in-flight pass is doing
	// the work the new tick would have).
	loop, emb, _, br, _ := makeSyncLoop(t, 0, false)
	emb.delay = 100 * time.Millisecond
	br.items = []backlog.BacklogItem{{Kind: backlog.KindFix, Name: "slow"}}

	go func() { _, _, _ = loop.RunOnce(context.Background()) }()
	// Give the first call a moment to acquire the singleton.
	time.Sleep(20 * time.Millisecond)

	_, _, err := loop.RunOnce(context.Background())
	if err != nil {
		t.Errorf("expected ErrReconcileBusy to be swallowed, got %v", err)
	}
	// Drain the slow first call so the test doesn't leak a goroutine.
	testutil.Eventually(t, 2*time.Second, "first runonce drains", func() bool {
		return loop.Reconciler.Status().Running == false
	})
}

func TestSyncLoop_Start_TickerCallsRunOnce(t *testing.T) {
	// Count Plan-time LoadAll calls on the backlog reader, NOT upserts.
	// After the first tick converges the index, subsequent ticks have nothing
	// to upsert — so upsertCalls maxes at 1. LoadAll fires every tick.
	loop, _, _, br, _ := makeSyncLoop(t, 10*time.Millisecond, false)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go loop.Start(ctx)

	testutil.Eventually(t, 500*time.Millisecond, "ticker fires twice (LoadAll calls)", func() bool {
		return atomic.LoadInt32(&br.loaded) >= 2
	})
}

func TestSyncLoop_Start_CancelStopsGoroutine(t *testing.T) {
	loop, _, _, _, _ := makeSyncLoop(t, 50*time.Millisecond, false)
	ctx, cancel := context.WithCancel(context.Background())

	var done int32
	go func() {
		loop.Start(ctx)
		atomic.StoreInt32(&done, 1)
	}()
	cancel()
	testutil.Eventually(t, 500*time.Millisecond, "loop goroutine exits", func() bool {
		return atomic.LoadInt32(&done) == 1
	})
}

func TestSyncLoop_Start_DisabledIsNoop(t *testing.T) {
	loop, _, bs, br, _ := makeSyncLoop(t, 10*time.Millisecond, true)
	br.items = []backlog.BacklogItem{{Kind: backlog.KindFix, Name: "x"}}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go loop.Start(ctx)
	time.Sleep(80 * time.Millisecond) // would have ticked ~8 times if enabled
	bs.mu.Lock()
	calls := bs.upsertCalls
	bs.mu.Unlock()
	if calls != 0 {
		t.Errorf("expected no upsert calls when Disabled, got %d", calls)
	}
}

func TestSyncLoop_Start_ZeroIntervalIsNoop(t *testing.T) {
	loop, _, bs, br, _ := makeSyncLoop(t, 0, false)
	br.items = []backlog.BacklogItem{{Kind: backlog.KindFix, Name: "x"}}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go loop.Start(ctx)
	time.Sleep(50 * time.Millisecond)
	bs.mu.Lock()
	calls := bs.upsertCalls
	bs.mu.Unlock()
	if calls != 0 {
		t.Errorf("expected no upsert calls when Interval=0, got %d", calls)
	}
}

func TestSyncLoop_Start_PanicRecovered(t *testing.T) {
	// Wire a Reconciler whose Embedder panics on every Embed call. The loop
	// must keep ticking despite repeated panics — pin it by counting the
	// panicking calls. If panic recovery were missing, the first panic would
	// bring down the goroutine and the counter would freeze at 1.
	loop, _, _, br, _ := makeSyncLoop(t, 10*time.Millisecond, false)
	pe := &panickingEmbedder{}
	loop.Reconciler.Embedder = pe
	br.items = []backlog.BacklogItem{{Kind: backlog.KindFix, Name: "boom"}}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go loop.Start(ctx)

	testutil.Eventually(t, time.Second, "loop keeps ticking through panics", func() bool {
		return atomic.LoadInt32(&pe.calls) >= 3
	})
}

func TestSyncLoop_NewSyncLoop_ReadsEnv(t *testing.T) {
	t.Setenv(EnvAISearchSyncInterval, "30s")
	t.Setenv(EnvAISearchSyncDisabled, "1")

	r, _, _, _, _, _ := newReconcilerForTest(t)
	loop := NewSyncLoop(r)
	if loop.Interval != 30*time.Second {
		t.Errorf("expected Interval=30s, got %v", loop.Interval)
	}
	if !loop.Disabled {
		t.Error("expected Disabled=true with env=1")
	}
}

func TestSyncLoop_NewSyncLoop_DefaultsWhenUnset(t *testing.T) {
	t.Setenv(EnvAISearchSyncInterval, "")
	t.Setenv(EnvAISearchSyncDisabled, "")

	r, _, _, _, _, _ := newReconcilerForTest(t)
	loop := NewSyncLoop(r)
	if loop.Interval != DefaultSyncInterval {
		t.Errorf("expected Interval=DefaultSyncInterval(%v), got %v", DefaultSyncInterval, loop.Interval)
	}
	if loop.Disabled {
		t.Error("expected Disabled=false when env unset")
	}
}

// ---- helpers ----

type panickingEmbedder struct {
	calls int32
}

func (p *panickingEmbedder) Embed(_ context.Context, _ string) ([]float64, error) {
	atomic.AddInt32(&p.calls, 1)
	panic("kaboom")
}
func (p *panickingEmbedder) Available(_ context.Context) bool { return true }
