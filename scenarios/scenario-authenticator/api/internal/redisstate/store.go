// Package redisstate is the hot-state seam: sessions, refresh-token-family
// state, the access-token blacklist, and rate-limit counters. Redis is a
// REQUIRED resource (these are security controls, not a cache — a lost
// blacklist re-admits a revoked token), and it is the horizontal-scale enabler
// (shared session/rate-limit state across replicas).
//
// Store is the narrow interface the auth domains depend on. Production wires
// the go-redis-backed impl; tests wire the in-memory Memory fake so handler and
// service tests never need a live Redis.
package redisstate

import (
	"context"
	"fmt"
	"time"

	"github.com/vrooli/api-core/storage"
)

// Store is the hot-state surface. All methods are key-scoped; callers own the
// local key shapes (session:<id>, refresh:<sha256>, blacklist:<sha256>, etc).
type Store interface {
	// Set stores value at key with a TTL (zero = no expiry).
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	// Get returns the value at key; found is false when the key is absent.
	Get(ctx context.Context, key string) (value string, found bool, err error)
	// Del removes keys (absent keys are ignored).
	Del(ctx context.Context, keys ...string) error
	// Exists reports whether key is present.
	Exists(ctx context.Context, key string) (bool, error)
	// SAdd adds members to the set at key.
	SAdd(ctx context.Context, key string, members ...string) error
	// SRem removes members from the set at key.
	SRem(ctx context.Context, key string, members ...string) error
	// SMembers returns the members of the set at key (empty when absent).
	SMembers(ctx context.Context, key string) ([]string, error)
	// Incr atomically increments the integer counter at key and returns the new
	// value (creating it at 1 when absent).
	Incr(ctx context.Context, key string) (int64, error)
	// Expire sets a TTL on an existing key.
	Expire(ctx context.Context, key string, ttl time.Duration) error
}

// NamespacedStore scopes every Redis key to the lifecycle-selected storage
// namespace. The domain packages continue to own their local key shapes while
// the process-wide storage seam owns isolation between live and shadow
// instances.
type NamespacedStore struct {
	inner  Store
	prefix string
}

// NewNamespacedStore wraps store with a variant-aware Redis prefix. The
// namespace must come from storage.ResolveNamespace so production callers use
// the lifecycle-injected VROOLI_STORAGE_NAMESPACE value.
func NewNamespacedStore(store Store, namespace storage.Namespace, domain string) (*NamespacedStore, error) {
	if store == nil {
		return nil, fmt.Errorf("redis state store is required")
	}
	prefix, err := namespace.RedisPrefix(domain)
	if err != nil {
		return nil, fmt.Errorf("resolve redis state prefix: %w", err)
	}
	return &NamespacedStore{inner: store, prefix: prefix}, nil
}

func (s *NamespacedStore) scoped(key string) string { return s.prefix + key }

func (s *NamespacedStore) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return s.inner.Set(ctx, s.scoped(key), value, ttl)
}

func (s *NamespacedStore) Get(ctx context.Context, key string) (string, bool, error) {
	return s.inner.Get(ctx, s.scoped(key))
}

func (s *NamespacedStore) Del(ctx context.Context, keys ...string) error {
	scoped := make([]string, len(keys))
	for i, key := range keys {
		scoped[i] = s.scoped(key)
	}
	return s.inner.Del(ctx, scoped...)
}

func (s *NamespacedStore) Exists(ctx context.Context, key string) (bool, error) {
	return s.inner.Exists(ctx, s.scoped(key))
}

func (s *NamespacedStore) SAdd(ctx context.Context, key string, members ...string) error {
	return s.inner.SAdd(ctx, s.scoped(key), members...)
}

func (s *NamespacedStore) SRem(ctx context.Context, key string, members ...string) error {
	return s.inner.SRem(ctx, s.scoped(key), members...)
}

func (s *NamespacedStore) SMembers(ctx context.Context, key string) ([]string, error) {
	return s.inner.SMembers(ctx, s.scoped(key))
}

func (s *NamespacedStore) Incr(ctx context.Context, key string) (int64, error) {
	return s.inner.Incr(ctx, s.scoped(key))
}

func (s *NamespacedStore) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return s.inner.Expire(ctx, s.scoped(key), ttl)
}

var _ Store = (*NamespacedStore)(nil)
