package aisearch

import (
	"context"
	"errors"
	"prompt-manager/internal/testutil/assertx"
	"sync/atomic"
	"testing"
	"time"
)

func TestSyncLoop_RunOnce_NoWork_AppliesEmptyPlan(t *testing.T) {
	store := newFakeStore()
	desc, _ := newTestDescriptor(KindSkill, store)
	r := NewReconciler(&fakeEmbedder{}, []CollectionDescriptor{desc}, 1)
	s := &SyncLoop{Reconciler: r, Interval: time.Hour}

	plan, apply, err := s.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if plan.HasWork() {
		t.Errorf("expected no work, got %+v", plan)
	}
	if apply == nil {
		t.Fatal("expected non-nil apply")
	}
}

func TestSyncLoop_RunOnce_SwallowsBusyError(t *testing.T) {
	store := newFakeStore()
	desc, items := newTestDescriptor(KindSkill, store)
	*items = []testItem{{ID: "a", Text: "foo"}}
	emb := &fakeEmbedder{delay: 100 * time.Millisecond}
	r := NewReconciler(emb, []CollectionDescriptor{desc}, 1)
	s := &SyncLoop{Reconciler: r, Interval: time.Hour}

	go func() {
		_, _, _ = r.RunOnce(context.Background())
	}()
	// Give the first run a chance to enter.
	time.Sleep(20 * time.Millisecond)

	// The SyncLoop should swallow ErrReconcileBusy as nil.
	_, _, err := s.RunOnce(context.Background())
	if err != nil {
		t.Errorf("expected nil on busy, got %v", err)
	}
}

func TestSyncLoop_Start_TickerCallsRunOnce(t *testing.T) {
	store := newFakeStore()
	desc, _ := newTestDescriptor(KindSkill, store)
	emb := &fakeEmbedder{}
	r := NewReconciler(emb, []CollectionDescriptor{desc}, 1)

	var ticks int32
	clock := time.Now
	wrapped := &Reconciler{
		Embedder:    r.Embedder,
		Descriptors: r.Descriptors,
		Parallelism: r.Parallelism,
		Clock:       clock,
	}
	// Wrap by counting calls in a goroutine that watches Status.
	s := &SyncLoop{Reconciler: wrapped, Interval: 10 * time.Millisecond}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go s.Start(ctx)

	assertx.Eventually(t, 2*time.Second, 20*time.Millisecond, func() error {
		st := wrapped.Status()
		if st.LastPlan != nil {
			atomic.StoreInt32(&ticks, 1)
			return nil
		}
		return errors.New("no tick yet")
	})

	if atomic.LoadInt32(&ticks) == 0 {
		t.Fatalf("expected at least one tick")
	}
}

func TestSyncLoop_Start_CancelStopsGoroutine(t *testing.T) {
	store := newFakeStore()
	desc, _ := newTestDescriptor(KindSkill, store)
	r := NewReconciler(&fakeEmbedder{}, []CollectionDescriptor{desc}, 1)
	s := &SyncLoop{Reconciler: r, Interval: 10 * time.Millisecond}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		s.Start(ctx)
		close(done)
	}()
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("Start did not exit after cancel")
	}
}

func TestSyncLoop_Start_DisabledIsNoop(t *testing.T) {
	store := newFakeStore()
	desc, _ := newTestDescriptor(KindSkill, store)
	r := NewReconciler(&fakeEmbedder{}, []CollectionDescriptor{desc}, 1)
	s := &SyncLoop{Reconciler: r, Interval: 1 * time.Millisecond, Disabled: true}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { s.Start(ctx); close(done) }()
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Start should return immediately when disabled")
	}
}

func TestSyncLoop_Start_ZeroIntervalIsNoop(t *testing.T) {
	r := NewReconciler(&fakeEmbedder{}, nil, 1)
	s := &SyncLoop{Reconciler: r, Interval: 0}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { s.Start(ctx); close(done) }()
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Start should return immediately on zero interval")
	}
}

func TestSyncLoop_NewSyncLoop_ReadsEnv(t *testing.T) {
	t.Setenv(EnvAISearchSyncInterval, "13m")
	t.Setenv(EnvAISearchSyncDisabled, "1")
	r := NewReconciler(&fakeEmbedder{}, nil, 1)
	s := NewSyncLoop(r)
	if s.Interval != 13*time.Minute {
		t.Errorf("expected 13m interval, got %s", s.Interval)
	}
	if !s.Disabled {
		t.Errorf("expected Disabled true")
	}
}

func TestSyncLoop_NewSyncLoop_DefaultsWhenUnset(t *testing.T) {
	t.Setenv(EnvAISearchSyncInterval, "")
	t.Setenv(EnvAISearchSyncDisabled, "")
	r := NewReconciler(&fakeEmbedder{}, nil, 1)
	s := NewSyncLoop(r)
	if s.Interval != DefaultSyncInterval {
		t.Errorf("expected default interval %s, got %s", DefaultSyncInterval, s.Interval)
	}
	if s.Disabled {
		t.Errorf("expected Disabled false by default")
	}
}
