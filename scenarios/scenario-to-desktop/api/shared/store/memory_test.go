package store

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type testItem struct {
	ID   string
	Name string
}

func TestInMemoryStore_Get(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryStore[string, testItem]()

	t.Run("not found", func(t *testing.T) {
		_, err := store.Get(ctx, "nonexistent")
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("found", func(t *testing.T) {
		item := testItem{ID: "1", Name: "test"}
		_ = store.Save(ctx, "1", item)

		got, err := store.Get(ctx, "1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID != "1" || got.Name != "test" {
			t.Errorf("expected item to match")
		}
	})
}

func TestInMemoryStore_Save(t *testing.T) {
	ctx := context.Background()

	t.Run("basic save", func(t *testing.T) {
		store := NewInMemoryStore[string, testItem]()
		item := testItem{ID: "1", Name: "test"}
		err := store.Save(ctx, "1", item)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, _ := store.Get(ctx, "1")
		if got.Name != "test" {
			t.Errorf("expected saved item")
		}
	})

	t.Run("with validator success", func(t *testing.T) {
		store := NewInMemoryStore[string, testItem](
			WithValidator[string, testItem](func(item testItem) error {
				if item.Name == "" {
					return errors.New("name required")
				}
				return nil
			}),
		)
		err := store.Save(ctx, "1", testItem{ID: "1", Name: "test"})
		if err != nil {
			t.Errorf("expected success, got %v", err)
		}
	})

	t.Run("with validator failure", func(t *testing.T) {
		store := NewInMemoryStore[string, testItem](
			WithValidator[string, testItem](func(item testItem) error {
				if item.Name == "" {
					return errors.New("name required")
				}
				return nil
			}),
		)
		err := store.Save(ctx, "1", testItem{ID: "1", Name: ""})
		if err == nil {
			t.Errorf("expected validation error")
		}
	})

	t.Run("overwrite existing", func(t *testing.T) {
		store := NewInMemoryStore[string, testItem]()
		_ = store.Save(ctx, "1", testItem{ID: "1", Name: "original"})
		_ = store.Save(ctx, "1", testItem{ID: "1", Name: "updated"})

		got, _ := store.Get(ctx, "1")
		if got.Name != "updated" {
			t.Errorf("expected updated item")
		}
	})
}

func TestInMemoryStore_SaveWithExtractedKey(t *testing.T) {
	ctx := context.Background()

	t.Run("with key extractor", func(t *testing.T) {
		store := NewInMemoryStore[string, testItem](
			WithKeyExtractor[string, testItem](func(item testItem) string {
				return item.ID
			}),
		)
		item := testItem{ID: "auto-id", Name: "test"}
		err := store.SaveWithExtractedKey(ctx, item)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, _ := store.Get(ctx, "auto-id")
		if got.Name != "test" {
			t.Errorf("expected item to be saved with extracted key")
		}
	})

	t.Run("without key extractor panics", func(t *testing.T) {
		store := NewInMemoryStore[string, testItem]()
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected panic")
			}
		}()
		_ = store.SaveWithExtractedKey(ctx, testItem{ID: "1"})
	})
}

func TestInMemoryStore_Delete(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryStore[string, testItem]()

	t.Run("delete existing", func(t *testing.T) {
		_ = store.Save(ctx, "1", testItem{ID: "1"})
		err := store.Delete(ctx, "1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		_, err = store.Get(ctx, "1")
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected item to be deleted")
		}
	})

	t.Run("delete nonexistent", func(t *testing.T) {
		err := store.Delete(ctx, "nonexistent")
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})
}

func TestInMemoryStore_Exists(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryStore[string, testItem]()

	exists, _ := store.Exists(ctx, "1")
	if exists {
		t.Errorf("expected not exists")
	}

	_ = store.Save(ctx, "1", testItem{ID: "1"})
	exists, _ = store.Exists(ctx, "1")
	if !exists {
		t.Errorf("expected exists")
	}
}

func TestInMemoryStore_List(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryStore[string, testItem]()

	t.Run("empty store", func(t *testing.T) {
		items, err := store.List(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(items) != 0 {
			t.Errorf("expected empty list")
		}
	})

	t.Run("with items", func(t *testing.T) {
		_ = store.Save(ctx, "1", testItem{ID: "1"})
		_ = store.Save(ctx, "2", testItem{ID: "2"})

		items, _ := store.List(ctx)
		if len(items) != 2 {
			t.Errorf("expected 2 items, got %d", len(items))
		}
	})
}

func TestInMemoryStore_ListKeys(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryStore[string, testItem]()

	_ = store.Save(ctx, "a", testItem{ID: "a"})
	_ = store.Save(ctx, "b", testItem{ID: "b"})

	keys, err := store.ListKeys(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(keys) != 2 {
		t.Errorf("expected 2 keys, got %d", len(keys))
	}
}

func TestInMemoryStore_Count(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryStore[string, testItem]()

	count, _ := store.Count(ctx)
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}

	_ = store.Save(ctx, "1", testItem{ID: "1"})
	_ = store.Save(ctx, "2", testItem{ID: "2"})

	count, _ = store.Count(ctx)
	if count != 2 {
		t.Errorf("expected 2, got %d", count)
	}
}

func TestInMemoryStore_Filter(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryStore[string, testItem]()

	_ = store.Save(ctx, "1", testItem{ID: "1", Name: "alpha"})
	_ = store.Save(ctx, "2", testItem{ID: "2", Name: "beta"})
	_ = store.Save(ctx, "3", testItem{ID: "3", Name: "alpha-2"})

	items, err := store.Filter(ctx, func(item testItem) bool {
		return len(item.Name) > 4
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 filtered items, got %d", len(items))
	}
}

func TestInMemoryStore_FindFirst(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryStore[string, testItem]()

	_ = store.Save(ctx, "1", testItem{ID: "1", Name: "alpha"})
	_ = store.Save(ctx, "2", testItem{ID: "2", Name: "beta"})

	t.Run("found", func(t *testing.T) {
		item, err := store.FindFirst(ctx, func(item testItem) bool {
			return item.Name == "beta"
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if item.ID != "2" {
			t.Errorf("expected item 2")
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := store.FindFirst(ctx, func(item testItem) bool {
			return item.Name == "gamma"
		})
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})
}

func TestInMemoryStore_Cleanup(t *testing.T) {
	ctx := context.Background()

	t.Run("without cleanup func", func(t *testing.T) {
		store := NewInMemoryStore[string, testItem]()
		_ = store.Save(ctx, "1", testItem{ID: "1"})

		err := store.Cleanup(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		count, _ := store.Count(ctx)
		if count != 1 {
			t.Errorf("expected items to remain")
		}
	})

	t.Run("with cleanup func", func(t *testing.T) {
		store := NewInMemoryStore[string, testItem](
			WithCleanupFunc[string, testItem](func(key string, item testItem) bool {
				return item.Name == "expired"
			}),
		)
		_ = store.Save(ctx, "1", testItem{ID: "1", Name: "keep"})
		_ = store.Save(ctx, "2", testItem{ID: "2", Name: "expired"})
		_ = store.Save(ctx, "3", testItem{ID: "3", Name: "expired"})

		err := store.Cleanup(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		count, _ := store.Count(ctx)
		if count != 1 {
			t.Errorf("expected 1 item remaining, got %d", count)
		}
	})
}

func TestInMemoryStore_CleanupWithCallback(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryStore[string, testItem]()

	_ = store.Save(ctx, "1", testItem{ID: "1", Name: "keep"})
	_ = store.Save(ctx, "2", testItem{ID: "2", Name: "remove"})

	var removedKeys []string
	removed, err := store.CleanupWithCallback(ctx,
		func(key string, item testItem) bool {
			return item.Name == "remove"
		},
		func(key string, item testItem) {
			removedKeys = append(removedKeys, key)
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if removed != 1 {
		t.Errorf("expected 1 removed, got %d", removed)
	}
	if len(removedKeys) != 1 || removedKeys[0] != "2" {
		t.Errorf("expected callback for key '2'")
	}
}

func TestInMemoryStore_GetAndDelete(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryStore[string, testItem]()

	_ = store.Save(ctx, "1", testItem{ID: "1", Name: "test"})

	item, err := store.GetAndDelete(ctx, "1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.Name != "test" {
		t.Errorf("expected item to be returned")
	}

	_, err = store.Get(ctx, "1")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected item to be deleted")
	}

	// GetAndDelete on nonexistent
	_, err = store.GetAndDelete(ctx, "1")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound")
	}
}

func TestInMemoryStore_Update(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryStore[string, testItem]()

	t.Run("update existing", func(t *testing.T) {
		_ = store.Save(ctx, "1", testItem{ID: "1", Name: "original"})

		err := store.Update(ctx, "1", func(item testItem) testItem {
			item.Name = "updated"
			return item
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, _ := store.Get(ctx, "1")
		if got.Name != "updated" {
			t.Errorf("expected updated name")
		}
	})

	t.Run("update nonexistent", func(t *testing.T) {
		err := store.Update(ctx, "nonexistent", func(item testItem) testItem {
			return item
		})
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})
}

func TestInMemoryStore_GetOrCreate(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryStore[string, testItem]()

	t.Run("create new", func(t *testing.T) {
		item, created, err := store.GetOrCreate(ctx, "1", func() testItem {
			return testItem{ID: "1", Name: "new"}
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !created {
			t.Errorf("expected created=true")
		}
		if item.Name != "new" {
			t.Errorf("expected new item")
		}
	})

	t.Run("get existing", func(t *testing.T) {
		item, created, err := store.GetOrCreate(ctx, "1", func() testItem {
			return testItem{ID: "1", Name: "should not be used"}
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if created {
			t.Errorf("expected created=false")
		}
		if item.Name != "new" {
			t.Errorf("expected existing item")
		}
	})
}

func TestInMemoryStore_Clear(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryStore[string, testItem]()

	_ = store.Save(ctx, "1", testItem{ID: "1"})
	_ = store.Save(ctx, "2", testItem{ID: "2"})

	err := store.Clear(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	count, _ := store.Count(ctx)
	if count != 0 {
		t.Errorf("expected empty store")
	}
}

func TestInMemoryStore_Snapshot(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryStore[string, testItem]()

	_ = store.Save(ctx, "1", testItem{ID: "1"})
	_ = store.Save(ctx, "2", testItem{ID: "2"})

	snapshot, err := store.Snapshot(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(snapshot) != 2 {
		t.Errorf("expected 2 items in snapshot")
	}

	// Modifying snapshot shouldn't affect store
	delete(snapshot, "1")
	count, _ := store.Count(ctx)
	if count != 2 {
		t.Errorf("store should be unaffected by snapshot modification")
	}
}

func TestInMemoryStore_Concurrency(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryStore[string, testItem]()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := string(rune('a' + i%26))
			_ = store.Save(ctx, key, testItem{ID: key})
			_, _ = store.Get(ctx, key)
			_, _ = store.List(ctx)
		}(i)
	}
	wg.Wait()

	// Should not panic or corrupt data
	_, err := store.List(ctx)
	if err != nil {
		t.Errorf("store corrupted after concurrent access")
	}
}

func TestTTLStore(t *testing.T) {
	ctx := context.Background()
	defaultTTL := 100 * time.Millisecond
	store := NewTTLStore[string, testItem](defaultTTL)

	t.Run("get before expiry", func(t *testing.T) {
		_ = store.Save(ctx, "1", testItem{ID: "1"})
		item, err := store.Get(ctx, "1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if item.ID != "1" {
			t.Errorf("expected item")
		}
	})

	t.Run("get after expiry", func(t *testing.T) {
		_ = store.Save(ctx, "2", testItem{ID: "2"})
		time.Sleep(150 * time.Millisecond)
		_, err := store.Get(ctx, "2")
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected ErrNotFound after expiry, got %v", err)
		}
	})

	t.Run("save with custom TTL", func(t *testing.T) {
		_ = store.SaveWithTTL(ctx, "3", testItem{ID: "3"}, 500*time.Millisecond)
		time.Sleep(150 * time.Millisecond)
		item, err := store.Get(ctx, "3")
		if err != nil {
			t.Errorf("expected item to still be valid")
		}
		if item.ID != "3" {
			t.Errorf("expected correct item")
		}
	})

	t.Run("refresh extends TTL", func(t *testing.T) {
		_ = store.Save(ctx, "4", testItem{ID: "4"})
		time.Sleep(50 * time.Millisecond)
		_ = store.Refresh(ctx, "4", 500*time.Millisecond)
		time.Sleep(100 * time.Millisecond)
		item, err := store.Get(ctx, "4")
		if err != nil {
			t.Errorf("expected item after refresh, got %v", err)
		}
		if item.ID != "4" {
			t.Errorf("expected correct item")
		}
	})

	t.Run("list excludes expired", func(t *testing.T) {
		freshStore := NewTTLStore[string, testItem](50 * time.Millisecond)
		_ = freshStore.Save(ctx, "a", testItem{ID: "a"})
		_ = freshStore.SaveWithTTL(ctx, "b", testItem{ID: "b"}, 500*time.Millisecond)

		time.Sleep(100 * time.Millisecond)

		items, err := freshStore.List(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(items) != 1 {
			t.Errorf("expected 1 non-expired item, got %d", len(items))
		}
		if len(items) > 0 && items[0].ID != "b" {
			t.Errorf("expected item 'b' to remain")
		}
	})
}

func TestCompositeKey(t *testing.T) {
	tests := []struct {
		parts    []string
		expected string
	}{
		{[]string{}, ""},
		{[]string{"a"}, "a"},
		{[]string{"a", "b"}, "a/b"},
		{[]string{"a", "b", "c"}, "a/b/c"},
	}

	for _, tt := range tests {
		result := CompositeKey(tt.parts...)
		if result != tt.expected {
			t.Errorf("CompositeKey(%v) = %q, want %q", tt.parts, result, tt.expected)
		}
	}
}

func TestWithPrefix(t *testing.T) {
	tests := []struct {
		prefix   string
		key      string
		expected string
	}{
		{"", "key", "key"},
		{"prefix", "key", "prefix/key"},
		{"a/b", "c", "a/b/c"},
	}

	for _, tt := range tests {
		result := WithPrefix(tt.prefix, tt.key)
		if result != tt.expected {
			t.Errorf("WithPrefix(%q, %q) = %q, want %q", tt.prefix, tt.key, result, tt.expected)
		}
	}
}
