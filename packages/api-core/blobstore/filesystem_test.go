package blobstore

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
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

func TestFilesystemBlobStoreRejectsInvalidInputs(t *testing.T) {
	t.Run("empty root", func(t *testing.T) {
		store := NewFilesystemBlobStore("")
		if err := store.Put(context.Background(), "notes/a.txt", strings.NewReader("x"), "text/plain"); err == nil {
			t.Fatal("expected empty root error")
		}
	})

	t.Run("nil reader", func(t *testing.T) {
		store := NewFilesystemBlobStore(t.TempDir())
		if err := store.Put(context.Background(), "notes/a.txt", nil, "text/plain"); err == nil {
			t.Fatal("expected nil reader error")
		}
	})

	t.Run("canceled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		store := NewFilesystemBlobStore(t.TempDir())
		if err := store.Put(ctx, "notes/a.txt", strings.NewReader("x"), "text/plain"); !errors.Is(err, context.Canceled) {
			t.Fatalf("Put error = %v, want context.Canceled", err)
		}
		if _, _, err := store.Get(ctx, "notes/a.txt"); !errors.Is(err, context.Canceled) {
			t.Fatalf("Get error = %v, want context.Canceled", err)
		}
		if err := store.Delete(ctx, "notes/a.txt"); !errors.Is(err, context.Canceled) {
			t.Fatalf("Delete error = %v, want context.Canceled", err)
		}
	})
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, fmt.Errorf("read failed")
}

func TestFilesystemBlobStorePutFailures(t *testing.T) {
	t.Run("parent path is file", func(t *testing.T) {
		root := t.TempDir()
		blocker := filepath.Join(root, "notes")
		if err := os.WriteFile(blocker, []byte("not a dir"), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		store := NewFilesystemBlobStore(root)
		if err := store.Put(context.Background(), "notes/a.txt", strings.NewReader("x"), "text/plain"); err == nil {
			t.Fatal("expected parent directory error")
		}
	})

	t.Run("reader fails", func(t *testing.T) {
		store := NewFilesystemBlobStore(t.TempDir())
		if err := store.Put(context.Background(), "notes/a.txt", failingReader{}, "text/plain"); err == nil {
			t.Fatal("expected reader error")
		}
	})

	t.Run("destination is directory", func(t *testing.T) {
		root := t.TempDir()
		if err := os.MkdirAll(filepath.Join(root, "notes", "a.txt"), 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		store := NewFilesystemBlobStore(root)
		if err := store.Put(context.Background(), "notes/a.txt", strings.NewReader("x"), "text/plain"); err == nil {
			t.Fatal("expected commit error")
		}
	})
}

func TestFilesystemBlobStoreGetWithoutMetadata(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "notes", "a.txt")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	store := NewFilesystemBlobStore(root)
	rc, mime, err := store.Get(context.Background(), "notes/a.txt")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(data) != "hello" || mime != "" {
		t.Fatalf("got data=%q mime=%q", data, mime)
	}
}

func TestFilesystemBlobStoreDeleteMissingIsOK(t *testing.T) {
	store := NewFilesystemBlobStore(t.TempDir())
	if err := store.Delete(context.Background(), "missing"); err != nil {
		t.Fatalf("Delete missing: %v", err)
	}
}

func TestFilesystemBlobStoreDeleteFailure(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "notes", "a.txt")
	if err := os.MkdirAll(filepath.Join(path, "child"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	store := NewFilesystemBlobStore(root)
	if err := store.Delete(context.Background(), "notes/a.txt"); err == nil {
		t.Fatal("expected delete error for non-empty directory")
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
