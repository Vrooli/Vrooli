package storage

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/api-core/blobstore"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	corestorage "github.com/vrooli/api-core/storage"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	return NewWithBlobStore(blobstore.NewMemoryBlobStore(), t.TempDir())
}

func TestPutGetDelete(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	want := []byte("PNGDATA")
	if err := s.Put(ctx, "inputs/a.png", bytes.NewReader(want), "image/png"); err != nil {
		t.Fatal(err)
	}
	rc, mime, err := s.Get(ctx, "inputs/a.png")
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(rc)
	_ = rc.Close()
	if !bytes.Equal(got, want) || mime != "image/png" {
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
		if err := s.Put(ctx, k, bytes.NewReader([]byte("x")), "image/png"); err == nil {
			t.Errorf("key %q should be rejected", k)
		}
	}
}

func TestWriteToBlobKey(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	ref, err := s.Write(ctx, OutputTarget{BlobKey: "outputs/x.png"}, bytes.NewReader([]byte("data")), "image/png")
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
	ref, err := s.Write(ctx, OutputTarget{LocalPath: dest}, bytes.NewReader([]byte("local-owned")), "image/png")
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

func TestRoutedStoreWritesOnlyToLeasedRoot(t *testing.T) {
	primary := corestorage.Paths{ConfigDir: t.TempDir(), DataDir: t.TempDir(), CacheDir: t.TempDir(), LogsDir: t.TempDir(), StateDir: t.TempDir()}
	routes := filerouting.New(primary)
	leaseRoots, err := routes.InstallLeasedTestRoots("lease-image-blobs", time.Minute, true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = routes.ClearTestRoots("lease-image-blobs") })
	store, err := NewWithRoutedRoots(primary, routes)
	if err != nil {
		t.Fatal(err)
	}
	ctx := database.WithTestMode(context.Background())
	if err := store.Put(ctx, "out/test.png", bytes.NewReader([]byte("test-pixels")), "image/png"); err != nil {
		t.Fatal(err)
	}
	leased := filepath.Join(leaseRoots.CacheDir, "blobs", "out", "test.png")
	if data, err := os.ReadFile(leased); err != nil || string(data) != "test-pixels" {
		t.Fatalf("leased blob=%q err=%v", data, err)
	}
	if _, err := os.Stat(filepath.Join(primary.DataDir, "blobs", "out", "test.png")); !os.IsNotExist(err) {
		t.Fatalf("test blob reached primary storage: err=%v", err)
	}
	if got := routes.LeaseStats().TestRootWrites; got != 1 {
		t.Fatalf("recorded test writes=%d want 1", got)
	}
	rc, _, err := store.Get(ctx, "out/test.png")
	if err != nil {
		t.Fatal(err)
	}
	_ = rc.Close()
	if err := store.Delete(ctx, "out/test.png"); err != nil {
		t.Fatal(err)
	}
	if got := routes.LeaseStats().TestRootWrites; got != 2 {
		t.Fatalf("recorded test writes after delete=%d want 2", got)
	}
}

func TestRoutedStoreFailsClosedWithoutLease(t *testing.T) {
	primary := corestorage.Paths{ConfigDir: t.TempDir(), DataDir: t.TempDir(), CacheDir: t.TempDir(), LogsDir: t.TempDir(), StateDir: t.TempDir()}
	routes := filerouting.New(primary)
	store, err := NewWithRoutedRoots(primary, routes)
	if err != nil {
		t.Fatal(err)
	}
	ctx := database.WithTestMode(context.Background())
	if err := store.Put(ctx, "out/no-leak.png", bytes.NewReader([]byte("pixels")), "image/png"); err == nil {
		t.Fatal("expected test-mode write without a lease to fail")
	}
	if _, err := os.Stat(filepath.Join(primary.DataDir, "blobs", "out", "no-leak.png")); !os.IsNotExist(err) {
		t.Fatalf("missing-lease write reached primary storage: err=%v", err)
	}
}

func TestDerivedOutputsUseCacheAndRetainedInputsUseData(t *testing.T) {
	paths := corestorage.Paths{DataDir: t.TempDir(), CacheDir: t.TempDir()}
	s, err := NewWithRoutedRoots(paths, nil)
	if err != nil {
		t.Fatal(err)
	}
	for key, root := range map[string]string{
		"out/result.png":                paths.CacheDir,
		"input/source.png":              paths.DataDir,
		"mask/edit.png":                 paths.DataDir,
		"scene-preparations/pinned.zip": paths.DataDir,
	} {
		if err := s.Put(context.Background(), key, bytes.NewBufferString(key), "application/octet-stream"); err != nil {
			t.Fatal(err)
		}
		want := filepath.Join(root, "blobs", key)
		if data, err := os.ReadFile(want); err != nil || string(data) != key {
			t.Fatalf("%s: %q, %v", want, data, err)
		}
		if got, err := s.PathForKey(key); err != nil || got != want {
			t.Fatalf("path %q, %v; want %q", got, err, want)
		}
		r, _, err := s.Get(context.Background(), key)
		if err != nil {
			t.Fatal(err)
		}
		r.Close()
		if err := s.Delete(context.Background(), key); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(want); !os.IsNotExist(err) {
			t.Fatalf("delete: %v", err)
		}
	}
}

func TestLegacyOutputsRequireQuiescedMigration(t *testing.T) {
	paths := corestorage.Paths{DataDir: t.TempDir(), CacheDir: t.TempDir()}
	legacy := filepath.Join(paths.DataDir, "blobs", "out")
	if err := os.MkdirAll(legacy, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(legacy, "existing.png")
	if err := os.WriteFile(file, []byte("user result"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewWithRoutedRoots(paths, nil); err == nil {
		t.Fatal("legacy output references would be stranded")
	}
	if got, err := os.ReadFile(file); err != nil || string(got) != "user result" {
		t.Fatalf("source altered: %q %v", got, err)
	}
}
