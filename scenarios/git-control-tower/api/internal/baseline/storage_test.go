package baseline

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/api-core/storage"
)

func newTestStorage(t *testing.T) *Storage {
	t.Helper()
	resolver, err := storage.NewResolver(storage.ResolverConfig{AppID: "vrooli"})
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}
	return NewStorageAt(resolver, t.TempDir())
}

func sampleManifest(name, branch string) BaselineManifest {
	return BaselineManifest{
		Name:          name,
		Scenario:      "foo",
		Branch:        branch,
		CreatedAt:     time.Now().UTC(),
		Surfaces:      map[string]SurfacePointer{},
		SchemaVersion: SchemaVersion,
	}
}

func TestStorageSaveLoadRoundTrip(t *testing.T) {
	s := newTestStorage(t)
	m := sampleManifest("plan-1", "agi")
	m.Surfaces[SurfaceTests] = SurfacePointer{SurfaceID: SurfaceTests, Kind: KindTestGenieRun, Ref: "run-1"}
	if err := s.Save(1, m, CreateOnly); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := s.Load(1, "foo", "agi", "plan-1")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Name != "plan-1" || got.Surfaces[SurfaceTests].Ref != "run-1" {
		t.Fatalf("unexpected loaded manifest: %+v", got)
	}
}

func TestStorageBranchScopingIsolation(t *testing.T) {
	s := newTestStorage(t)
	if err := s.Save(1, sampleManifest("plan", "agi"), CreateOnly); err != nil {
		t.Fatalf("save agi: %v", err)
	}
	if err := s.Save(1, sampleManifest("plan", "master"), CreateOnly); err != nil {
		t.Fatalf("save master: %v", err)
	}
	if _, err := s.Load(1, "foo", "agi", "plan"); err != nil {
		t.Fatalf("load agi: %v", err)
	}
	if _, err := s.Load(1, "foo", "master", "plan"); err != nil {
		t.Fatalf("load master: %v", err)
	}
	all, err := s.List(1, "foo", "")
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 baselines across branches, got %d", len(all))
	}
}

func TestStorageSlashBranchSanitized(t *testing.T) {
	s := newTestStorage(t)
	m := sampleManifest("plan", "feature/login")
	if err := s.Save(1, m, CreateOnly); err != nil {
		t.Fatalf("save: %v", err)
	}
	if _, err := s.Load(1, "foo", "feature/login", "plan"); err != nil {
		t.Fatalf("load with slash branch: %v", err)
	}
}

func TestStorageCreateOnlyRejectsDuplicate(t *testing.T) {
	s := newTestStorage(t)
	if err := s.Save(1, sampleManifest("plan", "agi"), CreateOnly); err != nil {
		t.Fatalf("first save: %v", err)
	}
	err := s.Save(1, sampleManifest("plan", "agi"), CreateOnly)
	if !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}
	// Overwrite mode replaces it.
	if err := s.Save(1, sampleManifest("plan", "agi"), Overwrite); err != nil {
		t.Fatalf("overwrite: %v", err)
	}
}

func TestStorageConcurrentSameNameOneWinner(t *testing.T) {
	s := newTestStorage(t)
	const n = 50
	var wg sync.WaitGroup
	var mu sync.Mutex
	wins, exists := 0, 0
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := s.Save(1, sampleManifest("plan-7c3", "agi"), CreateOnly)
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				wins++
			case errors.Is(err, ErrAlreadyExists):
				exists++
			default:
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()
	if wins != 1 {
		t.Fatalf("expected exactly 1 winner, got %d (exists=%d)", wins, exists)
	}
	if exists != n-1 {
		t.Fatalf("expected %d ErrAlreadyExists, got %d", n-1, exists)
	}
}

func TestStorageConcurrentDistinctNames(t *testing.T) {
	s := newTestStorage(t)
	const n = 50
	var wg sync.WaitGroup
	errCh := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errCh <- s.Save(1, sampleManifest(fmt.Sprintf("plan-%d", i), "agi"), CreateOnly)
		}(i)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("distinct save failed: %v", err)
		}
	}
	all, err := s.List(1, "foo", "agi")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(all) != n {
		t.Fatalf("expected %d baselines, got %d", n, len(all))
	}
}

func TestStorageDeleteAndNotFound(t *testing.T) {
	s := newTestStorage(t)
	if err := s.Delete(1, "foo", "agi", "ghost"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("delete missing: expected ErrNotFound, got %v", err)
	}
	if _, err := s.Load(1, "foo", "agi", "ghost"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("load missing: expected ErrNotFound, got %v", err)
	}
	if err := s.Save(1, sampleManifest("plan", "agi"), CreateOnly); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := s.Delete(1, "foo", "agi", "plan"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := s.Load(1, "foo", "agi", "plan"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("load after delete: expected ErrNotFound, got %v", err)
	}
}
