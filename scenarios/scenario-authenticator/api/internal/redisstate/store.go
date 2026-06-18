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
	"time"
)

// Store is the hot-state surface. All methods are key-scoped; callers own the
// key shapes (session:<id>, refresh:<sha256>, blacklist:<sha256>, etc).
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
