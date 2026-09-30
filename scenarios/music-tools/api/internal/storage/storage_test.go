package storage

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/api-core/blobstore"
)

func TestOwnedBudgetEvictsOldestEligibleAndProtectsReserved(t *testing.T) {
	root := t.TempDir()
	store := NewWithBlobStore(blobstore.NewFilesystemBlobStore(root), root)
	store.SetBudgetBytes(3)
	if err := store.PutOwned(context.Background(), "out/discarded.wav", "take-1", strings.NewReader("aa"), "audio/wav", true); err != nil {
		t.Fatal(err)
	}
	if err := store.PutOwned(context.Background(), "out/reserved.wav", "take-2", strings.NewReader("bb"), "audio/wav", true); err != nil {
		t.Fatal(err)
	}
	store.SetProtected("out/reserved.wav", true)
	removed, err := store.Evict(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(removed) != 1 || removed[0] != "out/discarded.wav" {
		t.Fatalf("removed=%v", removed)
	}
}

func TestEvictionPriorityPrecedesAge(t *testing.T) {
	store := newTestStore(t)
	store.SetBudgetBytes(2)
	if err := store.PutOwned(context.Background(), "out/available.wav", "take-a", strings.NewReader("aa"), "audio/wav", true); err != nil {
		t.Fatal(err)
	}
	if err := store.PutOwned(context.Background(), "out/discarded.wav", "take-d", strings.NewReader("bb"), "audio/wav", true); err != nil {
		t.Fatal(err)
	}
	store.SetEvictionPriority("out/available.wav", 1)
	store.SetEvictionPriority("out/discarded.wav", 0)
	removed, err := store.Evict(context.Background())
	if err != nil || len(removed) != 1 || removed[0] != "out/discarded.wav" {
		t.Fatalf("removed=%v err=%v", removed, err)
	}
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	return NewWithBlobStore(blobstore.NewMemoryBlobStore(), t.TempDir())
}

func TestPutGetDelete(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	want := []byte("PNGDATA")
	if err := s.Put(ctx, "inputs/a.png", bytes.NewReader(want), "audio/png"); err != nil {
		t.Fatal(err)
	}
	rc, mime, err := s.Get(ctx, "inputs/a.png")
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(rc)
	_ = rc.Close()
	if !bytes.Equal(got, want) || mime != "audio/png" {
		t.Fatalf("got %q/%q", got, mime)
	}
	if err := s.Delete(ctx, "inputs/a.png"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Get(ctx, "inputs/a.png"); err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestInvalidKeys(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	for _, k := range []string{"", "  ", "/abs/path", "../escape", "a/../../b"} {
		if err := s.Put(ctx, k, bytes.NewReader([]byte("x")), "audio/png"); err == nil {
			t.Errorf("key %q should be rejected", k)
		}
	}
}

func TestWriteToBlobKey(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	ref, err := s.Write(ctx, OutputTarget{BlobKey: "outputs/x.png"}, bytes.NewReader([]byte("data")), "audio/png")
	if err != nil {
		t.Fatal(err)
	}
	if ref != "outputs/x.png" {
		t.Fatalf("ref=%q", ref)
	}
	if _, _, err := s.Get(ctx, "outputs/x.png"); err != nil {
		t.Fatalf("blob not stored: %v", err)
	}
}

func TestWriteToLocalPath(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	dest := filepath.Join(t.TempDir(), "nested", "result.png")
	ref, err := s.Write(ctx, OutputTarget{LocalPath: dest}, bytes.NewReader([]byte("local-owned")), "audio/png")
	if err != nil {
		t.Fatal(err)
	}
	if ref != dest {
		t.Fatalf("ref=%q want %q", ref, dest)
	}
	got, err := os.ReadFile(dest)
	if err != nil || string(got) != "local-owned" {
		t.Fatalf("local file wrong: %q err=%v", got, err)
	}
}

func TestOutputTargetValidate(t *testing.T) {
	cases := []struct {
		name   string
		target OutputTarget
		ok     bool
	}{
		{"blob", OutputTarget{BlobKey: "a/b.png"}, true},
		{"local-abs", OutputTarget{LocalPath: "/tmp/x.png"}, true},
		{"both", OutputTarget{BlobKey: "a", LocalPath: "/b"}, false},
		{"empty", OutputTarget{}, false},
		{"local-rel", OutputTarget{LocalPath: "rel/x.png"}, false},
		{"bad-blob", OutputTarget{BlobKey: "../escape"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.target.Validate(); (err == nil) != tc.ok {
				t.Fatalf("Validate ok=%v err=%v", tc.ok, err)
			}
		})
	}
}
