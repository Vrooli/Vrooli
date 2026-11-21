// Package inventorycache memoizes the owner inventory for a short window.
//
// storage.LoadOwnerInventory walks scenarios/, resources/, internal/tools and
// internal/safeguards to find native manifests. The census tick, the
// retention tick, and every operator read model used to call it
// independently, so one request storm meant one repository walk per request.
// A single shared cache with a short TTL bounds that to one walk per window
// while keeping a manifest edit visible within minutes.
package inventorycache

import (
	"os"
	"strings"
	"sync"
	"time"

	corestorage "github.com/vrooli/api-core/storage"
)

// DefaultTTL is how long a loaded inventory is reused. STORAGE_INVENTORY_CACHE_TTL
// overrides it (any duration of at least one second).
const DefaultTTL = 5 * time.Minute

// Loader is the function that produces a fresh inventory. Tests substitute it.
type Loader func(corestorage.InventoryOptions) (corestorage.OwnerInventory, error)

type entry struct {
	inventory corestorage.OwnerInventory
	err       error
	loadedAt  time.Time
}

// Cache memoizes inventories per (repo root, platform). Concurrent callers
// for the same key share one in-flight load rather than racing.
type Cache struct {
	ttl     time.Duration
	load    Loader
	now     func() time.Time
	mu      sync.Mutex
	entries map[string]*entry
	pending map[string]*sync.WaitGroup
}

// New builds a cache with the given TTL and loader. A zero TTL uses
// DefaultTTL; a nil loader uses storage.LoadOwnerInventory.
func New(ttl time.Duration, load Loader) *Cache {
	if ttl <= 0 {
		ttl = DefaultTTL
	}
	if load == nil {
		load = corestorage.LoadOwnerInventory
	}
	return &Cache{ttl: ttl, load: load, now: time.Now, entries: map[string]*entry{}, pending: map[string]*sync.WaitGroup{}}
}

// TTL reports the cache window.
func (c *Cache) TTL() time.Duration { return c.ttl }

// Load returns the inventory for repoRoot on platform, reusing a value loaded
// within the TTL. Errors are not cached: a failed load is retried on the next
// call so a transient read error cannot be served for five minutes.
func (c *Cache) Load(repoRoot string, platform corestorage.Platform) (corestorage.OwnerInventory, error) {
	key := strings.TrimSpace(repoRoot) + "\x00" + string(platform)
	for {
		c.mu.Lock()
		if cached, ok := c.entries[key]; ok && cached.err == nil && c.now().Sub(cached.loadedAt) < c.ttl {
			c.mu.Unlock()
			return cached.inventory, nil
		}
		if wait, inFlight := c.pending[key]; inFlight {
			c.mu.Unlock()
			wait.Wait()
			continue
		}
		wait := &sync.WaitGroup{}
		wait.Add(1)
		c.pending[key] = wait
		c.mu.Unlock()

		inventory, err := c.load(corestorage.InventoryOptions{RepoRoot: repoRoot, Platform: platform})

		c.mu.Lock()
		if err == nil {
			c.entries[key] = &entry{inventory: inventory, loadedAt: c.now()}
		}
		delete(c.pending, key)
		c.mu.Unlock()
		wait.Done()
		return inventory, err
	}
}

// Invalidate drops every cached inventory so the next Load walks again.
func (c *Cache) Invalidate() {
	c.mu.Lock()
	c.entries = map[string]*entry{}
	c.mu.Unlock()
}

var (
	defaultOnce  sync.Once
	defaultCache *Cache
)

// Default is the process-wide cache shared by the census tick, the retention
// tick, and the HTTP read models.
func Default() *Cache {
	defaultOnce.Do(func() { defaultCache = New(ttlFromEnv(), nil) })
	return defaultCache
}

// Load is shorthand for Default().Load.
func Load(repoRoot string, platform corestorage.Platform) (corestorage.OwnerInventory, error) {
	return Default().Load(repoRoot, platform)
}

// Invalidate is shorthand for Default().Invalidate.
func Invalidate() { Default().Invalidate() }

func ttlFromEnv() time.Duration {
	raw := strings.TrimSpace(os.Getenv("STORAGE_INVENTORY_CACHE_TTL"))
	if raw == "" {
		return DefaultTTL
	}
	ttl, err := time.ParseDuration(raw)
	if err != nil || ttl < time.Second {
		return DefaultTTL
	}
	return ttl
}
