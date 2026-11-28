package store

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestJSONFileStorePerItemReloadsTemporarySuffixKeyAndIgnoresWriterArtifact(t *testing.T) {
	ctx := context.Background()
	path := t.TempDir()
	store, err := NewJSONFileStoreString[testItem](path, PerItem)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	want := testItem{ID: "orphan.tmp", Name: "valid suffix key"}
	if err := store.Save(ctx, "orphan.tmp", want); err != nil {
		t.Fatalf("save temporary-suffix key: %v", err)
	}
	if _, err := os.Stat(filepath.Join(path, "orphan.tmp.json")); err != nil {
		t.Fatalf("stat saved key file: %v", err)
	}

	if err := os.WriteFile(filepath.Join(path, "unfinished.json.tmp"), []byte(`{"ID":"unfinished","Name":"partial"}`), 0o644); err != nil {
		t.Fatalf("write temporary artifact: %v", err)
	}

	reloaded, err := NewJSONFileStoreString[testItem](path, PerItem)
	if err != nil {
		t.Fatalf("reload store: %v", err)
	}
	got, err := reloaded.Get(ctx, "orphan.tmp")
	if err != nil {
		t.Fatalf("get reloaded temporary-suffix key: %v", err)
	}
	if got != want {
		t.Fatalf("reloaded item = %#v, want %#v", got, want)
	}
	if _, err := reloaded.Get(ctx, "unfinished"); err != ErrNotFound {
		t.Fatalf("get temporary artifact key error = %v, want %v", err, ErrNotFound)
	}
}
