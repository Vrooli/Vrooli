package blobstore

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestMemoryBlobStoreRoundTrip(t *testing.T) {
	store := NewMemoryBlobStore()
	ctx := context.Background()
	if err := store.Put(ctx, "k", strings.NewReader("body"), "text/plain"); err != nil {
		t.Fatalf("Put: %v", err)
	}
	rc, mime, err := store.Get(ctx, "k")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer rc.Close()
	data, _ := io.ReadAll(rc)
	if string(data) != "body" || mime != "text/plain" {
		t.Fatalf("got data=%q mime=%q", data, mime)
	}
	if err := store.Delete(ctx, "k"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, _, err := store.Get(ctx, "k"); err == nil {
		t.Fatal("expected missing blob")
	}
}

func TestMemoryBlobStoreRejectsEmptyKeyAndNilReader(t *testing.T) {
	store := NewMemoryBlobStore()
	if err := store.Put(context.Background(), "", strings.NewReader("x"), "text/plain"); err == nil {
		t.Fatal("expected empty key error")
	}
	if err := store.Put(context.Background(), "k", nil, "text/plain"); err == nil {
		t.Fatal("expected nil reader error")
	}
}

func TestMemoryBlobStoreCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	store := NewMemoryBlobStore()
	if err := store.Put(ctx, "k", strings.NewReader("x"), "text/plain"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Put error = %v, want context.Canceled", err)
	}
	if _, _, err := store.Get(ctx, "k"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Get error = %v, want context.Canceled", err)
	}
	if err := store.Delete(ctx, "k"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Delete error = %v, want context.Canceled", err)
	}
}

func TestMemoryBlobStoreZeroValue(t *testing.T) {
	var store MemoryBlobStore
	if err := store.Put(context.Background(), "k", strings.NewReader("body"), "text/plain"); err != nil {
		t.Fatalf("Put: %v", err)
	}
	rc, mime, err := store.Get(context.Background(), " k ")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer rc.Close()
	data, _ := io.ReadAll(rc)
	if string(data) != "body" || mime != "text/plain" {
		t.Fatalf("got data=%q mime=%q", data, mime)
	}
	if err := store.Delete(context.Background(), " k "); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}
