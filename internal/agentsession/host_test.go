package agentsession

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
)

func TestScopeCgroupPathCachesSliceLookupAcrossCalls(t *testing.T) {
	old := agentSlicePath
	t.Cleanup(func() { agentSlicePath = old })
	var calls atomic.Int32
	agentSlicePath = &scopeSlicePathCache{resolve: func() (string, error) {
		calls.Add(1)
		return "/user.slice/user-1000.slice/user@1000.service/vrooli.slice/vrooli-agents.slice", nil
	}}

	paths := make([]string, 2)
	for i, unit := range []string{"vrooli-agent-one.scope", "vrooli-agent-two.scope"} {
		path, err := ScopeCgroupPath(unit)
		if err != nil {
			t.Fatalf("ScopeCgroupPath(%q): %v", unit, err)
		}
		paths[i] = path
	}
	if calls.Load() != 1 {
		t.Fatalf("slice path lookups=%d want 1", calls.Load())
	}
	if paths[0] == paths[1] || paths[0] != "/user.slice/user-1000.slice/user@1000.service/vrooli.slice/vrooli-agents.slice/vrooli-agent-one.scope" {
		t.Fatalf("scope paths were not independently resolved under the shared slice: %v", paths)
	}
}

func TestScopeSlicePathCacheSerializesConcurrentFirstLookup(t *testing.T) {
	var calls atomic.Int32
	cache := &scopeSlicePathCache{resolve: func() (string, error) {
		calls.Add(1)
		return "/user.slice/user-1000.slice/vrooli.slice/vrooli-agents.slice", nil
	}}
	const readers = 32
	var wg sync.WaitGroup
	for range readers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			path, err := cache.get()
			if err != nil || path == "" {
				t.Errorf("cache.get() = %q, %v", path, err)
			}
		}()
	}
	wg.Wait()
	if calls.Load() != 1 {
		t.Fatalf("concurrent slice path lookups=%d want 1", calls.Load())
	}
}

func TestScopeSlicePathCacheRetriesLookupFailures(t *testing.T) {
	var calls atomic.Int32
	cache := &scopeSlicePathCache{resolve: func() (string, error) {
		if calls.Add(1) == 1 {
			return "", errors.New("user manager is not ready")
		}
		return "/user.slice/user-1000.slice/vrooli.slice/vrooli-agents.slice", nil
	}}
	if _, err := cache.get(); err == nil {
		t.Fatal("first lookup failure was hidden")
	}
	if path, err := cache.get(); err != nil || path == "" {
		t.Fatalf("retry lookup = %q, %v", path, err)
	}
	if calls.Load() != 2 {
		t.Fatalf("lookup calls=%d want failed lookup plus successful retry", calls.Load())
	}
}

func TestScopeCgroupPathRejectsInvalidUnitBeforeLookup(t *testing.T) {
	old := agentSlicePath
	t.Cleanup(func() { agentSlicePath = old })
	var calls atomic.Int32
	agentSlicePath = &scopeSlicePathCache{resolve: func() (string, error) {
		calls.Add(1)
		return "/user.slice/user-1000.slice/vrooli.slice/vrooli-agents.slice", nil
	}}
	if _, err := ScopeCgroupPath("vrooli-agent-nested/escape.scope"); err == nil {
		t.Fatal("invalid unit path was accepted")
	}
	if calls.Load() != 0 {
		t.Fatalf("invalid unit caused %d user-manager lookups", calls.Load())
	}
}
