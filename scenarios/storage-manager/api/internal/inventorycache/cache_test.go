package inventorycache

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	corestorage "github.com/vrooli/api-core/storage"
)

func TestLoadReturnsTheCachedInventoryInsideTheTTL(t *testing.T) {
	var loads atomic.Int32
	cache := New(5*time.Minute, func(opts corestorage.InventoryOptions) (corestorage.OwnerInventory, error) {
		loads.Add(1)
		return corestorage.OwnerInventory{RepoRoot: opts.RepoRoot, Owners: []corestorage.OwnerManifest{{Kind: corestorage.OwnerScenario, ID: "demo"}}}, nil
	})
	clock := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	cache.now = func() time.Time { return clock }

	first, err := cache.Load("/repo", corestorage.PlatformLinux)
	if err != nil {
		t.Fatal(err)
	}
	clock = clock.Add(4 * time.Minute)
	second, err := cache.Load("/repo", corestorage.PlatformLinux)
	if err != nil {
		t.Fatal(err)
	}
	if loads.Load() != 1 {
		t.Fatalf("loads = %d, want 1: the second call inside the TTL must not walk the repository", loads.Load())
	}
	if len(first.Owners) != 1 || len(second.Owners) != 1 || second.Owners[0].ID != "demo" {
		t.Fatalf("cached inventory = %+v / %+v", first, second)
	}

	clock = clock.Add(2 * time.Minute) // 6 minutes after the load
	if _, err := cache.Load("/repo", corestorage.PlatformLinux); err != nil {
		t.Fatal(err)
	}
	if loads.Load() != 2 {
		t.Fatalf("loads = %d, want 2: a call after the TTL must reload", loads.Load())
	}
}

func TestLoadKeysByRepoRootAndPlatform(t *testing.T) {
	var loads atomic.Int32
	cache := New(time.Hour, func(opts corestorage.InventoryOptions) (corestorage.OwnerInventory, error) {
		loads.Add(1)
		return corestorage.OwnerInventory{RepoRoot: opts.RepoRoot}, nil
	})
	for _, call := range []struct {
		root     string
		platform corestorage.Platform
	}{{"/a", corestorage.PlatformLinux}, {"/a", corestorage.PlatformLinux}, {"/b", corestorage.PlatformLinux}, {"/a", corestorage.PlatformWindows}} {
		if _, err := cache.Load(call.root, call.platform); err != nil {
			t.Fatal(err)
		}
	}
	if loads.Load() != 3 {
		t.Fatalf("loads = %d, want 3 distinct (root, platform) keys", loads.Load())
	}
}

func TestLoadDoesNotCacheErrorsAndInvalidateForcesAReload(t *testing.T) {
	var loads atomic.Int32
	fail := true
	cache := New(time.Hour, func(corestorage.InventoryOptions) (corestorage.OwnerInventory, error) {
		loads.Add(1)
		if fail {
			return corestorage.OwnerInventory{}, errors.New("transient")
		}
		return corestorage.OwnerInventory{RepoRoot: "/repo"}, nil
	})
	if _, err := cache.Load("/repo", corestorage.PlatformLinux); err == nil {
		t.Fatal("expected the transient error to surface")
	}
	fail = false
	if _, err := cache.Load("/repo", corestorage.PlatformLinux); err != nil {
		t.Fatalf("second load: %v", err)
	}
	if loads.Load() != 2 {
		t.Fatalf("loads = %d, want 2: an error must not be served from the cache", loads.Load())
	}
	cache.Invalidate()
	if _, err := cache.Load("/repo", corestorage.PlatformLinux); err != nil {
		t.Fatal(err)
	}
	if loads.Load() != 3 {
		t.Fatalf("loads = %d, want 3 after Invalidate", loads.Load())
	}
}

func TestConcurrentLoadsShareOneWalk(t *testing.T) {
	var loads atomic.Int32
	release := make(chan struct{})
	cache := New(time.Hour, func(corestorage.InventoryOptions) (corestorage.OwnerInventory, error) {
		loads.Add(1)
		<-release
		return corestorage.OwnerInventory{RepoRoot: "/repo"}, nil
	})
	const callers = 16
	var wg sync.WaitGroup
	started := make(chan struct{}, callers)
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			started <- struct{}{}
			if _, err := cache.Load("/repo", corestorage.PlatformLinux); err != nil {
				t.Error(err)
			}
		}()
	}
	for i := 0; i < callers; i++ {
		<-started
	}
	time.Sleep(20 * time.Millisecond)
	close(release)
	wg.Wait()
	if loads.Load() != 1 {
		t.Fatalf("loads = %d, want 1: a request storm must coalesce into one repository walk", loads.Load())
	}
}
