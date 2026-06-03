package aisearch

import (
	"context"
	"testing"
	"time"
)

func TestSyncLoopRunOnceReconciles(t *testing.T) {
	src := &sliceSource{docs: []SourceDoc{doc("README.md", "alpha\nbeta")}}
	store, emb := newMemStore(), &countingEmbedder{}
	rec := newDocReconciler(src, store, emb)
	loop := NewSyncLoop("test", rec, Config{SyncInterval: time.Minute})

	plan, apply, err := loop.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if plan == nil || apply == nil {
		t.Fatalf("plan/apply nil: %v / %v", plan, apply)
	}
	if store.upserts != 2 {
		t.Errorf("upserts = %d, want 2 (one per line)", store.upserts)
	}
}

func TestSyncLoopRunOnceSwallowsBusy(t *testing.T) {
	// A reconciler already marked running returns ErrReconcileBusy from RunOnce;
	// the loop treats that as a no-op success (nil error).
	src := &sliceSource{docs: []SourceDoc{doc("a.md", "x")}}
	store, emb := newMemStore(), &countingEmbedder{}
	rec := newDocReconciler(src, store, emb)
	rec.running = true // simulate an in-flight reconcile
	loop := NewSyncLoop("test", rec, Config{SyncInterval: time.Minute})

	_, _, err := loop.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce should swallow ErrReconcileBusy, got %v", err)
	}
}

func TestSyncLoopStartDisabledReturns(t *testing.T) {
	// Disabled / non-positive interval / nil reconciler must return promptly
	// instead of spinning a ticker.
	loop := NewSyncLoop("test", nil, Config{})
	loop.Start(context.Background()) // nil reconciler → returns immediately

	rec := NewReconciler(&countingEmbedder{}, nil, 1)
	disabled := NewSyncLoop("test", rec, Config{SyncInterval: time.Minute, SyncDisabled: true})
	disabled.Start(context.Background()) // disabled → returns immediately
}
