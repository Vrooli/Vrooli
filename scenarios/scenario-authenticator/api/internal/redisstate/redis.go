package redisstate

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	coreRedis "github.com/vrooli/api-core/redis"
)

// RedisStore is the production Store backed by go-redis/v9.
type RedisStore struct {
	client *redis.Client
}

// NewRedisStore builds a Store from the lifecycle-injected Redis environment
// (REDIS_URL, or REDIS_HOST/REDIS_PORT/REDIS_DB/REDIS_PASSWORD) and verifies
// connectivity with a Ping. Redis is a REQUIRED resource for this scenario
// (sessions, refresh-family revocation, blacklist are security controls), so a
// failed connection is a boot-fatal error for the caller — never a silent
// degrade that would re-admit revoked tokens.
func NewRedisStore(ctx context.Context) (*RedisStore, error) {
	opts, err := optionsFromEnv()
	if err != nil {
		return nil, err
	}
	client := redis.NewClient(opts)
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	return &RedisStore{client: client}, nil
}

// Close releases the underlying connection pool.
func (r *RedisStore) Close() error { return r.client.Close() }

// RedisConfigured reports whether the lifecycle injected Redis connection
// details. It is the selection input for hot state, and it deliberately does
// not probe the server: "configured but unreachable" must stay a boot-fatal
// error rather than a quiet fallback to a store that shares nothing across
// replicas.
func RedisConfigured() bool {
	_, err := coreRedis.Resolve(os.Getenv)
	return err == nil
}

func optionsFromEnv() (*redis.Options, error) {
	cfg, err := coreRedis.Resolve(os.Getenv)
	if err != nil {
		return nil, err
	}
	return &redis.Options{Addr: cfg.Addr, DB: cfg.DB, Password: cfg.Password}, nil
}

var _ Store = (*RedisStore)(nil)

func (r *RedisStore) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

func (r *RedisStore) Get(ctx context.Context, key string) (string, bool, error) {
	v, err := r.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return v, true, nil
}

func (r *RedisStore) Del(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return r.client.Del(ctx, keys...).Err()
}

func (r *RedisStore) Exists(ctx context.Context, key string) (bool, error) {
	n, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *RedisStore) SAdd(ctx context.Context, key string, members ...string) error {
	if len(members) == 0 {
		return nil
	}
	return r.client.SAdd(ctx, key, toAny(members)...).Err()
}

func (r *RedisStore) SRem(ctx context.Context, key string, members ...string) error {
	if len(members) == 0 {
		return nil
	}
	return r.client.SRem(ctx, key, toAny(members)...).Err()
}

func (r *RedisStore) SMembers(ctx context.Context, key string) ([]string, error) {
	return r.client.SMembers(ctx, key).Result()
}

func (r *RedisStore) Incr(ctx context.Context, key string) (int64, error) {
	return r.client.Incr(ctx, key).Result()
}

func (r *RedisStore) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return r.client.Expire(ctx, key, ttl).Err()
}

func toAny(in []string) []any {
	out := make([]any, len(in))
	for i, s := range in {
		out[i] = s
	}
	return out
}
