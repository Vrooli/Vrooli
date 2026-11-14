package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// TestFileRetentionStore_FreshSeed — when no retention.json exists, Get
// returns the seed passed to NewFileRetentionStoreAtPath unmodified and
// the seed is NOT eagerly persisted (first-boot UX: env defaults stay
// in-memory until the operator commits a real change).
func TestFileRetentionStore_FreshSeed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "retention.json")
	seed := RetentionConfig{MaxArchiveAgeDays: 7, MaxArchiveSizeBytes: 1 << 30, MaxArchivesPerProject: 0}

	store := NewFileRetentionStoreAtPath(path, seed)
	got := store.Get()
	if got != seed {
		t.Fatalf("Get fresh = %+v, want %+v", got, seed)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected retention.json to NOT exist before first Set, stat err = %v", err)
	}
}

// TestFileRetentionStore_SetThenGet — Set persists the value to disk,
// and Get on a freshly constructed store at the same path returns it.
func TestFileRetentionStore_SetThenGet(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "retention.json")
	store := NewFileRetentionStoreAtPath(path, RetentionConfig{})

	want := RetentionConfig{MaxArchiveAgeDays: 30, MaxArchiveSizeBytes: 5 << 30, MaxArchivesPerProject: 200}
	if err := store.Set(want); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if got := store.Get(); got != want {
		t.Fatalf("Get after Set = %+v, want %+v", got, want)
	}

	// Reconstruct: the on-disk value must take over.
	store2 := NewFileRetentionStoreAtPath(path, RetentionConfig{MaxArchiveAgeDays: 999})
	if err := store2.ensureLoaded(); err != nil {
		t.Fatalf("ensureLoaded: %v", err)
	}
	if got := store2.Get(); got != want {
		t.Fatalf("reload Get = %+v, want %+v (file should override seed)", got, want)
	}
}

// A partial update must keep omitted fields in the durable value returned
// after a new store loads the file.
func TestFileRetentionStore_UpdatePersistsMergedConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "retention.json")
	seed := RetentionConfig{MaxArchiveAgeDays: 90, MaxArchiveSizeBytes: 10 << 30, MaxArchivesPerProject: 100}
	store := NewFileRetentionStoreAtPath(path, seed)
	age := 7
	size := int64(1 << 30)
	if _, err := store.Update(RetentionPatch{MaxArchiveAgeDays: &age}); err != nil {
		t.Fatal(err)
	}
	got, err := store.Update(RetentionPatch{MaxArchiveSizeBytes: &size})
	if err != nil {
		t.Fatal(err)
	}
	want := RetentionConfig{MaxArchiveAgeDays: age, MaxArchiveSizeBytes: size, MaxArchivesPerProject: 100}
	if got != want {
		t.Fatalf("updated config = %+v, want %+v", got, want)
	}
	reloaded := NewFileRetentionStoreAtPath(path, RetentionConfig{})
	if err := reloaded.ensureLoaded(); err != nil {
		t.Fatal(err)
	}
	if persisted := reloaded.Get(); persisted != want {
		t.Fatalf("reloaded config = %+v, want %+v", persisted, want)
	}
}

func TestFileRetentionStore_FailedPersistKeepsLastAcceptedConfig(t *testing.T) {
	root := t.TempDir()
	parent := filepath.Join(root, "config")
	path := filepath.Join(parent, "retention.json")
	accepted := RetentionConfig{MaxArchiveAgeDays: 30, MaxArchiveSizeBytes: 1 << 30}
	store := NewFileRetentionStoreAtPath(path, RetentionConfig{})
	if err := store.Set(accepted); err != nil {
		t.Fatal(err)
	}

	// Simulate a write failure without relying on chmod or process privileges.
	moved := filepath.Join(root, "saved-config")
	if err := os.Rename(parent, moved); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(parent, []byte("blocks directory recreation"), 0o600); err != nil {
		t.Fatal(err)
	}
	rejected := RetentionConfig{MaxArchiveAgeDays: 0, MaxArchiveSizeBytes: 0}
	if err := store.Set(rejected); err == nil {
		t.Fatal("Set must report the failed persistence")
	}
	if got := store.Get(); got != accepted {
		t.Fatalf("Get after rejected Set = %+v, want last accepted %+v", got, accepted)
	}
	data, err := os.ReadFile(filepath.Join(moved, "retention.json"))
	if err != nil {
		t.Fatal(err)
	}
	var durable RetentionConfig
	if err := json.Unmarshal(data, &durable); err != nil {
		t.Fatal(err)
	}
	if durable != accepted {
		t.Fatalf("durable config = %+v, want last accepted %+v", durable, accepted)
	}
}

// TestFileRetentionStore_RejectsNegative — Set rejects negative values
// for any of the three levers and does not mutate cache or disk.
func TestFileRetentionStore_RejectsNegative(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "retention.json")
	seed := RetentionConfig{MaxArchiveAgeDays: 7}
	store := NewFileRetentionStoreAtPath(path, seed)

	cases := []RetentionConfig{
		{MaxArchiveAgeDays: -1},
		{MaxArchiveSizeBytes: -1},
		{MaxArchivesPerProject: -1},
	}
	for _, bad := range cases {
		if err := store.Set(bad); err == nil {
			t.Fatalf("Set(%+v) accepted; want validation error", bad)
		}
	}
	if got := store.Get(); got != seed {
		t.Fatalf("seed mutated after rejected Sets: got %+v want %+v", got, seed)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("retention.json should not exist after rejected Sets, stat err = %v", err)
	}
}

// TestFileRetentionStore_AtomicWrite — Set uses tmp+rename so a
// pre-existing valid file is never visible in a partial state. Verify
// by writing twice and ensuring the second value fully replaced the
// first (no JSON corruption).
func TestFileRetentionStore_AtomicWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "retention.json")
	store := NewFileRetentionStoreAtPath(path, RetentionConfig{})

	if err := store.Set(RetentionConfig{MaxArchiveAgeDays: 1, MaxArchiveSizeBytes: 100, MaxArchivesPerProject: 1}); err != nil {
		t.Fatalf("Set 1: %v", err)
	}
	if err := store.Set(RetentionConfig{MaxArchiveAgeDays: 2, MaxArchiveSizeBytes: 200, MaxArchivesPerProject: 2}); err != nil {
		t.Fatalf("Set 2: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var got RetentionConfig
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("file is not valid JSON: %v", err)
	}
	want := RetentionConfig{MaxArchiveAgeDays: 2, MaxArchiveSizeBytes: 200, MaxArchivesPerProject: 2}
	if got != want {
		t.Fatalf("on-disk = %+v, want %+v", got, want)
	}
}

// TestFileRetentionStore_RejectsCorruptFile — a file with non-JSON
// contents fails ensureLoaded loud rather than silently falling back
// to the seed. Operators must be told their config is unreadable.
func TestFileRetentionStore_RejectsCorruptFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "retention.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatalf("seed corrupt file: %v", err)
	}

	store := NewFileRetentionStoreAtPath(path, RetentionConfig{MaxArchiveAgeDays: 7})
	if err := store.ensureLoaded(); err == nil {
		t.Fatalf("expected error from corrupt file, got nil")
	}
}

// TestFileRetentionStore_RejectsInvalidPersisted — a file that is
// syntactically valid JSON but semantically invalid (negative value)
// fails ensureLoaded so corrupt data doesn't propagate to the runtime.
func TestFileRetentionStore_RejectsInvalidPersisted(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "retention.json")
	if err := os.WriteFile(path, []byte(`{"maxArchiveAgeDays":-1}`), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	store := NewFileRetentionStoreAtPath(path, RetentionConfig{})
	if err := store.ensureLoaded(); err == nil {
		t.Fatalf("expected validation error, got nil")
	}
}

// TestMemoryRetentionStore — symmetric coverage with FileRetentionStore
// so callers using either implementation observe the same contract.
func TestMemoryRetentionStore(t *testing.T) {
	store := NewMemoryRetentionStore(RetentionConfig{MaxArchiveAgeDays: 7})
	if got := store.Get().MaxArchiveAgeDays; got != 7 {
		t.Fatalf("seed = %d, want 7", got)
	}
	if err := store.Set(RetentionConfig{MaxArchiveAgeDays: 30, MaxArchiveSizeBytes: 100}); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if got := store.Get(); got.MaxArchiveAgeDays != 30 || got.MaxArchiveSizeBytes != 100 {
		t.Fatalf("Get after Set = %+v", got)
	}
	if err := store.Set(RetentionConfig{MaxArchiveAgeDays: -5}); err == nil {
		t.Fatalf("Set negative accepted; want error")
	}
}

// TestFileRetentionStore_ConcurrentGetSet — Set and Get may race in
// production (PUT handler + reconciler tick). The store must serialize
// them correctly and never tear a partially-updated value.
func TestFileRetentionStore_ConcurrentGetSet(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "retention.json")
	store := NewFileRetentionStoreAtPath(path, RetentionConfig{})

	var wg sync.WaitGroup
	stop := make(chan struct{})
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; ; i++ {
			select {
			case <-stop:
				return
			default:
			}
			cfg := RetentionConfig{MaxArchiveAgeDays: i % 100}
			if err := store.Set(cfg); err != nil {
				t.Errorf("Set: %v", err)
				return
			}
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			select {
			case <-stop:
				return
			default:
			}
			cfg := store.Get()
			if cfg.MaxArchiveAgeDays < 0 {
				t.Errorf("torn read: %+v", cfg)
				return
			}
		}
	}()

	// Run for a brief window then stop.
	for i := 0; i < 1000; i++ {
		_ = store.Get()
	}
	close(stop)
	wg.Wait()
}
