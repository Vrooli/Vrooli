package main

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestEnsureSearchCollectionBoundsUnavailableDependency(t *testing.T) {
	start := time.Now()
	err := ensureSearchCollection(context.Background(), 25*time.Millisecond, func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("ensure error = %v, want bounded deadline", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("unavailable search dependency blocked startup for %s", elapsed)
	}
}

func TestEnsureSearchCollectionPreservesSuccessfulEnsure(t *testing.T) {
	want := errors.New("unexpected qdrant schema")
	err := ensureSearchCollection(context.Background(), time.Second, func(ctx context.Context) error {
		if _, ok := ctx.Deadline(); !ok {
			t.Fatal("ensure context has no deadline")
		}
		return want
	})
	if !errors.Is(err, want) {
		t.Fatalf("ensure error = %v, want %v", err, want)
	}
}
