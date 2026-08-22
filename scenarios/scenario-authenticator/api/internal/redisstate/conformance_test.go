package redisstate

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand/v2"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"
)

// The three Store implementations are interchangeable by contract, so they are
// held to one suite rather than three. A behaviour that only the fake has is a
// test that proves nothing about production; a behaviour only production has is
// a gap the fake hides.

// storeFactory builds a fresh Store bound to a clock the test controls, so TTL
// behaviour is deterministic rather than timing-dependent.
type storeFactory struct {
	name string
	// realClockOnly marks an implementation that reads its own clock, so cases
	// that advance a controlled clock cannot apply to it.
	realClockOnly bool
	// open returns a Store reading time through now, plus a reopen function
	// that rebinds a new handle to the same durable backing state. A store that
	// keeps nothing across handles returns a nil reopen.
	open func(t *testing.T, now func() time.Time) (Store, func(now func() time.Time) Store)
}

// build is the common case: a fresh store whose durability is not under test.
func (f storeFactory) build(t *testing.T, now func() time.Time) Store {
	t.Helper()
	store, _ := f.open(t, now)
	return store
}

func openTestDatabase(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	for _, statement := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
		Schema(),
	} {
		if _, err := db.ExecContext(context.Background(), statement); err != nil {
			t.Fatalf("prepare sqlite (%s): %v", statement, err)
		}
	}
	return db
}

func storeFactories() []storeFactory {
	return []storeFactory{
		{
			name: "memory",
			open: func(_ *testing.T, now func() time.Time) (Store, func(func() time.Time) Store) {
				return NewMemoryWithClock(now), nil
			},
		},
		{
			// Redis is the production implementation, so it is held to the same
			// suite. It skips rather than fails when no server is reachable:
			// the suite must stay runnable on a host with no Redis, which is
			// the whole point of the durable local store beside it.
			name: "redis",
			open: func(t *testing.T, now func() time.Time) (Store, func(func() time.Time) Store) {
				store, err := NewRedisStore(context.Background())
				if err != nil {
					t.Skipf("no reachable Redis server: %v", err)
				}
				t.Cleanup(func() { _ = store.Close() })
				// Redis reads its own clock, so a controlled clock cannot be
				// injected. Cases that move the clock are skipped below.
				_ = now
				prefix := fmt.Sprintf("conformance:%d:%d:", time.Now().UnixNano(), rand.Int64())
				scoped := &prefixedStore{inner: store, prefix: prefix}
				t.Cleanup(func() { scoped.purge(context.Background()) })
				return scoped, nil
			},
			realClockOnly: true,
		},
		{
			name: "sqlite",
			open: func(t *testing.T, now func() time.Time) (Store, func(func() time.Time) Store) {
				path := filepath.Join(t.TempDir(), "hot-state.db")
				build := func(clock func() time.Time) Store {
					store, err := NewSQLiteStoreWithClock(openTestDatabase(t, path), clock)
					if err != nil {
						t.Fatalf("build sqlite store: %v", err)
					}
					return store
				}
				return build(now), build
			},
		},
	}
}

// prefixedStore isolates one test run inside a shared Redis server and can
// remove exactly what it wrote, so the suite leaves no keys behind.
type prefixedStore struct {
	inner  Store
	prefix string
	mu     sync.Mutex
	seen   map[string]struct{}
}

func (p *prefixedStore) scoped(key string) string {
	p.mu.Lock()
	if p.seen == nil {
		p.seen = map[string]struct{}{}
	}
	p.seen[p.prefix+key] = struct{}{}
	p.mu.Unlock()
	return p.prefix + key
}

func (p *prefixedStore) purge(ctx context.Context) {
	p.mu.Lock()
	keys := make([]string, 0, len(p.seen))
	for key := range p.seen {
		keys = append(keys, key)
	}
	p.mu.Unlock()
	if len(keys) > 0 {
		_ = p.inner.Del(ctx, keys...)
	}
}

func (p *prefixedStore) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return p.inner.Set(ctx, p.scoped(key), value, ttl)
}

func (p *prefixedStore) Get(ctx context.Context, key string) (string, bool, error) {
	return p.inner.Get(ctx, p.scoped(key))
}

func (p *prefixedStore) Del(ctx context.Context, keys ...string) error {
	scoped := make([]string, len(keys))
	for index, key := range keys {
		scoped[index] = p.scoped(key)
	}
	return p.inner.Del(ctx, scoped...)
}

func (p *prefixedStore) Exists(ctx context.Context, key string) (bool, error) {
	return p.inner.Exists(ctx, p.scoped(key))
}

func (p *prefixedStore) SAdd(ctx context.Context, key string, members ...string) error {
	return p.inner.SAdd(ctx, p.scoped(key), members...)
}

func (p *prefixedStore) SRem(ctx context.Context, key string, members ...string) error {
	return p.inner.SRem(ctx, p.scoped(key), members...)
}

func (p *prefixedStore) SMembers(ctx context.Context, key string) ([]string, error) {
	return p.inner.SMembers(ctx, p.scoped(key))
}

func (p *prefixedStore) Incr(ctx context.Context, key string) (int64, error) {
	return p.inner.Incr(ctx, p.scoped(key))
}

func (p *prefixedStore) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return p.inner.Expire(ctx, p.scoped(key), ttl)
}

var _ Store = (*prefixedStore)(nil)

// mustNamespace wraps a store in the variant-aware prefix production uses, so
// the suite exercises the same key shapes the scenario writes.
func mustNamespace(t *testing.T, store Store, root, variant string) *NamespacedStore {
	t.Helper()
	namespace, err := storage.ResolveNamespace(storage.NamespaceConfig{Root: root, Variant: variant})
	if err != nil {
		t.Fatalf("resolve namespace %q: %v", root, err)
	}
	scoped, err := NewNamespacedStore(store, namespace, "auth")
	if err != nil {
		t.Fatalf("scope namespace %q: %v", root, err)
	}
	return scoped
}

func TestStoreConformance(t *testing.T) {
	for _, factory := range storeFactories() {
		t.Run(factory.name, func(t *testing.T) {
			runStoreConformance(t, factory)
		})
	}
}

func runStoreConformance(t *testing.T, factory storeFactory) {
	ctx := context.Background()

	t.Run("set get exists del", func(t *testing.T) {
		store := factory.build(t, time.Now)
		if err := store.Set(ctx, "k", "v", 0); err != nil {
			t.Fatalf("set: %v", err)
		}
		value, found, err := store.Get(ctx, "k")
		if err != nil || !found || value != "v" {
			t.Fatalf("get = %q, found=%v, err=%v", value, found, err)
		}
		present, err := store.Exists(ctx, "k")
		if err != nil || !present {
			t.Fatalf("exists = %v, err=%v", present, err)
		}
		if err := store.Del(ctx, "k"); err != nil {
			t.Fatalf("del: %v", err)
		}
		if _, found, _ := store.Get(ctx, "k"); found {
			t.Fatal("key survived deletion")
		}
	})

	t.Run("absent key is not an error", func(t *testing.T) {
		store := factory.build(t, time.Now)
		value, found, err := store.Get(ctx, "missing")
		if err != nil || found || value != "" {
			t.Fatalf("get missing = %q, found=%v, err=%v", value, found, err)
		}
		if err := store.Del(ctx, "missing"); err != nil {
			t.Fatalf("del missing: %v", err)
		}
	})

	t.Run("overwrite replaces value and ttl", func(t *testing.T) {
		if factory.realClockOnly {
			t.Skip("implementation reads its own clock; TTL is covered deterministically by the others")
		}
		clock := time.Unix(1000, 0)
		store := factory.build(t, func() time.Time { return clock })
		if err := store.Set(ctx, "k", "first", time.Minute); err != nil {
			t.Fatalf("set: %v", err)
		}
		if err := store.Set(ctx, "k", "second", 0); err != nil {
			t.Fatalf("overwrite: %v", err)
		}
		clock = clock.Add(2 * time.Minute)
		value, found, _ := store.Get(ctx, "k")
		if !found || value != "second" {
			t.Fatalf("overwrite did not clear the earlier ttl: %q found=%v", value, found)
		}
	})

	// The security-critical case: an expired entry must read as absent
	// immediately, not once a sweep happens to run.
	t.Run("ttl is enforced on read", func(t *testing.T) {
		if factory.realClockOnly {
			t.Skip("implementation reads its own clock; TTL is covered deterministically by the others")
		}
		clock := time.Unix(1000, 0)
		store := factory.build(t, func() time.Time { return clock })
		if err := store.Set(ctx, "k", "v", time.Minute); err != nil {
			t.Fatalf("set: %v", err)
		}
		if _, found, _ := store.Get(ctx, "k"); !found {
			t.Fatal("key absent before expiry")
		}
		clock = clock.Add(2 * time.Minute)
		if _, found, _ := store.Get(ctx, "k"); found {
			t.Fatal("expired key was returned")
		}
		if present, _ := store.Exists(ctx, "k"); present {
			t.Fatal("expired key still reports as existing")
		}
	})

	t.Run("expire sets and clears a ttl", func(t *testing.T) {
		if factory.realClockOnly {
			t.Skip("implementation reads its own clock; TTL is covered deterministically by the others")
		}
		clock := time.Unix(1000, 0)
		store := factory.build(t, func() time.Time { return clock })
		if err := store.Set(ctx, "k", "v", 0); err != nil {
			t.Fatalf("set: %v", err)
		}
		if err := store.Expire(ctx, "k", time.Minute); err != nil {
			t.Fatalf("expire: %v", err)
		}
		clock = clock.Add(2 * time.Minute)
		if _, found, _ := store.Get(ctx, "k"); found {
			t.Fatal("expire did not apply")
		}
		if err := store.Expire(ctx, "missing", time.Minute); err != nil {
			t.Fatalf("expire on absent key: %v", err)
		}
	})

	t.Run("sets add remove and list", func(t *testing.T) {
		store := factory.build(t, time.Now)
		if err := store.SAdd(ctx, "s", "a", "b", "c"); err != nil {
			t.Fatalf("sadd: %v", err)
		}
		if err := store.SAdd(ctx, "s", "a"); err != nil {
			t.Fatalf("sadd duplicate: %v", err)
		}
		members, err := store.SMembers(ctx, "s")
		if err != nil || len(members) != 3 {
			t.Fatalf("members = %v, err=%v", members, err)
		}
		if err := store.SRem(ctx, "s", "b"); err != nil {
			t.Fatalf("srem: %v", err)
		}
		members, _ = store.SMembers(ctx, "s")
		if len(members) != 2 {
			t.Fatalf("after srem = %v", members)
		}
		empty, err := store.SMembers(ctx, "absent-set")
		if err != nil || len(empty) != 0 {
			t.Fatalf("absent set = %v, err=%v", empty, err)
		}
		if err := store.SRem(ctx, "absent-set", "x"); err != nil {
			t.Fatalf("srem on absent set: %v", err)
		}
	})

	t.Run("del clears set membership", func(t *testing.T) {
		store := factory.build(t, time.Now)
		if err := store.SAdd(ctx, "s", "a"); err != nil {
			t.Fatalf("sadd: %v", err)
		}
		if err := store.Del(ctx, "s"); err != nil {
			t.Fatalf("del: %v", err)
		}
		members, _ := store.SMembers(ctx, "s")
		if len(members) != 0 {
			t.Fatalf("set survived deletion: %v", members)
		}
	})

	t.Run("incr counts from absent", func(t *testing.T) {
		store := factory.build(t, time.Now)
		for want := int64(1); want <= 3; want++ {
			got, err := store.Incr(ctx, "c")
			if err != nil || got != want {
				t.Fatalf("incr = %d, want %d, err=%v", got, want, err)
			}
		}
	})

	t.Run("incr restarts after the counter expires", func(t *testing.T) {
		if factory.realClockOnly {
			t.Skip("implementation reads its own clock; TTL is covered deterministically by the others")
		}
		clock := time.Unix(1000, 0)
		store := factory.build(t, func() time.Time { return clock })
		if _, err := store.Incr(ctx, "c"); err != nil {
			t.Fatalf("incr: %v", err)
		}
		if err := store.Expire(ctx, "c", time.Minute); err != nil {
			t.Fatalf("expire: %v", err)
		}
		clock = clock.Add(2 * time.Minute)
		got, err := store.Incr(ctx, "c")
		if err != nil || got != 1 {
			t.Fatalf("incr after expiry = %d, want 1, err=%v", got, err)
		}
	})

	// A rate limit built on a counter that hands the same value to two callers
	// admits twice the traffic it promises.
	t.Run("incr is atomic under concurrency", func(t *testing.T) {
		store := factory.build(t, time.Now)
		const callers = 8
		const perCaller = 25
		results := make(chan int64, callers*perCaller)
		var group sync.WaitGroup
		for range callers {
			group.Add(1)
			go func() {
				defer group.Done()
				for range perCaller {
					value, err := store.Incr(ctx, "c")
					if err != nil {
						t.Errorf("incr: %v", err)
						return
					}
					results <- value
				}
			}()
		}
		group.Wait()
		close(results)
		seen := make(map[int64]bool, callers*perCaller)
		for value := range results {
			if seen[value] {
				t.Fatalf("two callers observed counter value %d", value)
			}
			seen[value] = true
		}
		if len(seen) != callers*perCaller {
			t.Fatalf("observed %d distinct values, want %d", len(seen), callers*perCaller)
		}
	})

	t.Run("namespaced wrapper isolates variants", func(t *testing.T) {
		store := factory.build(t, time.Now)
		live := mustNamespace(t, store, "scenario-authenticator", "")
		shadow := mustNamespace(t, store, "scenario-authenticator_shadow", "shadow")
		if err := live.Set(ctx, "session:same", "live", time.Minute); err != nil {
			t.Fatalf("live set: %v", err)
		}
		if err := shadow.Set(ctx, "session:same", "shadow", time.Minute); err != nil {
			t.Fatalf("shadow set: %v", err)
		}
		value, found, _ := live.Get(ctx, "session:same")
		if !found || value != "live" {
			t.Fatalf("live value = %q found=%v", value, found)
		}
		if err := live.SAdd(ctx, "usersessions:user", "session:same"); err != nil {
			t.Fatalf("live sadd: %v", err)
		}
		members, _ := shadow.SMembers(ctx, "usersessions:user")
		if len(members) != 0 {
			t.Fatalf("shadow set leaked live members: %v", members)
		}
	})

	_, reopen := factory.open(t, time.Now)
	if reopen == nil {
		return
	}

	// Durability is the whole reason the SQLite implementation exists: a
	// restart that forgets the blacklist re-admits a revoked token.
	t.Run("state survives a new handle", func(t *testing.T) {
		store, reopen := factory.open(t, time.Now)
		if err := store.Set(ctx, "blacklist:token", "revoked", 0); err != nil {
			t.Fatalf("set: %v", err)
		}
		if err := store.SAdd(ctx, "family:user", "token"); err != nil {
			t.Fatalf("sadd: %v", err)
		}
		if _, err := store.Incr(ctx, "ratelimit:user"); err != nil {
			t.Fatalf("incr: %v", err)
		}

		reopened := reopen(time.Now)
		value, found, err := reopened.Get(ctx, "blacklist:token")
		if err != nil || !found || value != "revoked" {
			t.Fatalf("blacklist entry did not survive restart: %q found=%v err=%v", value, found, err)
		}
		members, _ := reopened.SMembers(ctx, "family:user")
		if len(members) != 1 {
			t.Fatalf("set did not survive restart: %v", members)
		}
		next, err := reopened.Incr(ctx, "ratelimit:user")
		if err != nil || next != 2 {
			t.Fatalf("counter did not survive restart: %d err=%v", next, err)
		}
	})

	t.Run("sweep reclaims expired rows without changing reads", func(t *testing.T) {
		clock := time.Unix(1000, 0)
		store := factory.build(t, func() time.Time { return clock })
		sweeper, ok := store.(interface {
			Sweep(context.Context) (int64, error)
		})
		if !ok {
			t.Skip("implementation does not expose a sweep")
		}
		for index := range 3 {
			if err := store.Set(ctx, fmt.Sprintf("k%d", index), "v", time.Minute); err != nil {
				t.Fatalf("set: %v", err)
			}
		}
		if err := store.Set(ctx, "keep", "v", 0); err != nil {
			t.Fatalf("set durable: %v", err)
		}
		clock = clock.Add(2 * time.Minute)
		removed, err := sweeper.Sweep(ctx)
		if err != nil {
			t.Fatalf("sweep: %v", err)
		}
		if removed != 3 {
			t.Fatalf("sweep removed %d rows, want 3", removed)
		}
		if _, found, _ := store.Get(ctx, "keep"); !found {
			t.Fatal("sweep removed a row with no expiry")
		}
	})
}
