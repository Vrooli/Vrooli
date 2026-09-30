package storage

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/api-core/blobstore"
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
