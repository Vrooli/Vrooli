package blobstore

import (
	"context"
	"io"
	"strings"
	"sync"
	"testing"
)

func TestFilesystemBlobStoreRoundTrip(t *testing.T) {
	store := NewFilesystemBlobStore(t.TempDir())
	ctx := context.Background()
	if err := store.Put(ctx, "notes/a.txt", strings.NewReader("hello"), "text/plain"); err != nil {
		t.Fatalf("Put: %v", err)
	}
	rc, mime, err := store.Get(ctx, "notes/a.txt")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(data) != "hello" || mime != "text/plain" {
		t.Fatalf("got data=%q mime=%q", data, mime)
	}
	if err := store.Delete(ctx, "notes/a.txt"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, _, err := store.Get(ctx, "notes/a.txt"); err == nil {
		t.Fatal("expected missing blob after delete")
	}
}

func TestFilesystemBlobStoreRejectsEscapingKeys(t *testing.T) {
	store := NewFilesystemBlobStore(t.TempDir())
	for _, key := range []string{"", "../x", "/absolute", "a/../../x"} {
		if err := store.Put(context.Background(), key, strings.NewReader("x"), "text/plain"); err == nil {
			t.Fatalf("expected key %q to fail", key)
		}
	}
}

func TestFilesystemBlobStoreConcurrentPut(t *testing.T) {
	store := NewFilesystemBlobStore(t.TempDir())
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := store.Put(context.Background(), "same-key", strings.NewReader("value"), "text/plain"); err != nil {
				t.Errorf("Put: %v", err)
			}
		}()
	}
	wg.Wait()
	rc, _, err := store.Get(context.Background(), "same-key")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer rc.Close()
	data, _ := io.ReadAll(rc)
	if string(data) != "value" {
		t.Fatalf("data = %q", data)
	}
}
