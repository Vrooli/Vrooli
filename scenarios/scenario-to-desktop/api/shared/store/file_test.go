package store

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestJSONFileStoreSingleFilePersistsOptionsAndDelete(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "state", "items.json")
	store, err := NewJSONFileStoreString(path, SingleFile,
		WithFileStoreOptions(StoreOptions[string, testItem]{
			Validator: func(item testItem) error {
				if item.Name == "" {
					return errors.New("name is required")
				}
				return nil
			},
			BeforeSave: func(item testItem) testItem { item.Name += "-stored"; return item },
			AfterLoad:  func(item testItem) testItem { item.Name += "-loaded"; return item },
		}),
	)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	if err := store.Save(ctx, "one", testItem{ID: "one", Name: "alpha"}); err != nil {
		t.Fatalf("save item: %v", err)
	}
	if err := store.Save(ctx, "invalid", testItem{ID: "invalid"}); err == nil {
		t.Fatal("expected validator error")
	}
	got, err := store.Get(ctx, "one")
	if err != nil || got.Name != "alpha-stored-loaded" {
		t.Fatalf("get transformed item = %#v, %v", got, err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close store: %v", err)
	}

	reloaded, err := NewJSONFileStoreString[testItem](path, SingleFile)
	if err != nil {
		t.Fatalf("reload store: %v", err)
	}
	got, err = reloaded.Get(ctx, "one")
	if err != nil || got.Name != "alpha-stored" {
		t.Fatalf("reloaded item = %#v, %v", got, err)
	}
	if err := reloaded.Delete(ctx, "one"); err != nil {
		t.Fatalf("delete item: %v", err)
	}
	if count, _ := reloaded.Count(ctx); count != 0 {
		t.Fatalf("count after delete = %d, want 0", count)
	}
}

func TestJSONFileStorePerItemRoundTripsUnsafeKeysAndIgnoresTemporaryFiles(t *testing.T) {
	ctx := context.Background()
	path := t.TempDir()
	store, err := NewJSONFileStoreString[testItem](path, PerItem)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	for key, item := range map[string]testItem{
		"safe-key": {ID: "safe-key", Name: "safe"},
		"a/b":      {ID: "a/b", Name: "slash"},
		"a_b":      {ID: "a_b", Name: "underscore"},
	} {
		if err := store.Save(ctx, key, item); err != nil {
			t.Fatalf("save %q: %v", key, err)
		}
	}
	if err := os.WriteFile(filepath.Join(path, "orphan.tmp.json"), []byte(`{"ID":"bad"}`), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(path, "not-json.txt"), []byte("ignored"), 0o644); err != nil {
		t.Fatalf("write unrelated file: %v", err)
	}

	reloaded, err := NewJSONFileStoreString[testItem](path, PerItem)
	if err != nil {
		t.Fatalf("reload store: %v", err)
	}
	for _, key := range []string{"safe-key", "a/b", "a_b"} {
		got, err := reloaded.Get(ctx, key)
		if err != nil || got.ID != key {
			t.Fatalf("reloaded %q = %#v, %v", key, got, err)
		}
	}
	if err := reloaded.Delete(ctx, "a/b"); err != nil {
		t.Fatalf("delete unsafe key: %v", err)
	}
	if exists, _ := reloaded.Exists(ctx, "a/b"); exists {
		t.Fatal("deleted unsafe key still exists")
	}
}

func TestJSONFileStoreRejectsCorruptDataAndEmptyFileLoadsAsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "items.json")
	if err := os.WriteFile(path, []byte("not json"), 0o644); err != nil {
		t.Fatalf("write corrupt data: %v", err)
	}
	if _, err := NewJSONFileStoreString[testItem](path, SingleFile); err == nil {
		t.Fatal("expected corrupt data error")
	}
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatalf("write empty data: %v", err)
	}
	store, err := NewJSONFileStoreString[testItem](path, SingleFile)
	if err != nil {
		t.Fatalf("open empty store: %v", err)
	}
	if count, _ := store.Count(context.Background()); count != 0 {
		t.Fatalf("empty file count = %d, want 0", count)
	}
}

func TestJSONFileStoreListsFiltersAndHybridFlushesDeliberately(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "items.json")
	store, err := NewJSONFileStoreString[testItem](path, SingleFile,
		WithFileStoreOptions[string, testItem](StoreOptions[string, testItem]{
			AfterLoad: func(item testItem) testItem { item.Name += "-loaded"; return item },
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(ctx, "one", testItem{Name: "first"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(ctx, "two", testItem{Name: "second"}); err != nil {
		t.Fatal(err)
	}
	items, err := store.List(ctx)
	if err != nil || len(items) != 2 || items[0].Name == "first" || items[1].Name == "second" {
		t.Fatalf("List = %#v, %v", items, err)
	}
	keys, err := store.ListKeys(ctx)
	if err != nil || len(keys) != 2 {
		t.Fatalf("ListKeys = %#v, %v", keys, err)
	}
	filtered, err := store.Filter(ctx, func(item testItem) bool { return item.Name == "first-loaded" })
	if err != nil || len(filtered) != 1 || filtered[0].Name != "first-loaded" {
		t.Fatalf("Filter = %#v, %v", filtered, err)
	}

	hybridPath := filepath.Join(t.TempDir(), "hybrid.json")
	hybrid, err := NewHybridStore[string, testItem](hybridPath, SingleFile, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := hybrid.Save(ctx, "one", testItem{Name: "pending"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(hybridPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("manual-flush hybrid unexpectedly persisted: %v", err)
	}
	if err := hybrid.Flush(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(hybridPath); err != nil {
		t.Fatalf("manual Flush did not persist: %v", err)
	}
	if err := hybrid.Delete(ctx, "one"); err != nil {
		t.Fatal(err)
	}
	if err := hybrid.Delete(ctx, "one"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing HybridStore Delete = %v", err)
	}

	autoPath := filepath.Join(t.TempDir(), "auto.json")
	auto, err := NewHybridStore[string, testItem](autoPath, SingleFile, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := auto.Save(ctx, "one", testItem{Name: "saved"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(autoPath); err != nil {
		t.Fatalf("auto-flush Save did not persist: %v", err)
	}
	if err := auto.Delete(ctx, "one"); err != nil {
		t.Fatal(err)
	}
	reloaded, err := NewJSONFileStoreString[testItem](autoPath, SingleFile)
	if err != nil {
		t.Fatal(err)
	}
	if count, err := reloaded.Count(ctx); err != nil || count != 0 {
		t.Fatalf("auto-flush Delete count = %d, %v", count, err)
	}
}
