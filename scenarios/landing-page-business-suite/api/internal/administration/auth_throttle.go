package administration

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// ThrottleRule bounds how many events one bucket may record inside a window.
type ThrottleRule struct {
	Limit  int
	Window time.Duration
}

// Authentication throttles. Buckets are keyed by purpose plus a normalized
// subject so one noisy subject never consumes another subject's allowance.
var (
	SignInPerEmail       = ThrottleRule{Limit: 5, Window: 15 * time.Minute}
	SignInPerIP          = ThrottleRule{Limit: 20, Window: time.Hour}
	AdminFailuresPerUser = ThrottleRule{Limit: 5, Window: 15 * time.Minute}
	AdminFailuresPerIP   = ThrottleRule{Limit: 20, Window: 15 * time.Minute}
)

// ErrThrottled reports that a bucket has used its allowance for the window.
var ErrThrottled = errors.New("too many attempts")

// ThrottleStore is the persistence boundary for AuthThrottle.
type ThrottleStore interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

// AuthThrottle persists authentication attempts so limits hold across API
// restarts and replicas. When the database is unavailable it degrades to a
// process-local window instead of failing open.
type AuthThrottle struct {
	store ThrottleStore
	now   func() time.Time

	mu        sync.Mutex
	fallback  map[string][]time.Time
	lastPrune time.Time
}

// NewAuthThrottle creates a throttle over the auth_rate_events table.
func NewAuthThrottle(store ThrottleStore) *AuthThrottle {
	return &AuthThrottle{store: store, now: time.Now, fallback: map[string][]time.Time{}}
}

// UseClock replaces the fallback clock in tests.
func (t *AuthThrottle) UseClock(now func() time.Time) { t.now = now }

// ThrottleBucket builds a normalized bucket key.
func ThrottleBucket(purpose, subject string) string {
	return purpose + ":" + strings.ToLower(strings.TrimSpace(subject))
}

// Allow records one attempt and reports whether it fits the rule. A rejected
// attempt is not recorded, so a caller that waits out the window recovers.
func (t *AuthThrottle) Allow(ctx context.Context, bucket string, rule ThrottleRule) (bool, error) {
	if t == nil || bucket == "" || rule.Limit <= 0 {
		return true, nil
	}
	if t.store != nil {
		var id int64
		err := t.store.QueryRowContext(ctx, `
			WITH recent AS (
				SELECT COUNT(*) AS used FROM auth_rate_events
				WHERE bucket = $1 AND created_at > NOW() - ($2 * INTERVAL '1 second')
			)
			INSERT INTO auth_rate_events (bucket)
			SELECT $1 FROM recent WHERE used < $3
			RETURNING id`, bucket, int64(rule.Window/time.Second), rule.Limit).Scan(&id)
		switch {
		case err == nil:
			t.maybePrune(ctx)
			return true, nil
		case errors.Is(err, sql.ErrNoRows):
			return false, nil
		}
		// Fall through to the local window; report the degradation.
		return t.allowLocal(bucket, rule, true), fmt.Errorf("persist auth throttle: %w", err)
	}
	return t.allowLocal(bucket, rule, true), nil
}

// Exceeded reports whether a bucket has already used its allowance, without
// recording an attempt. Pair it with Record for failure-only counting.
func (t *AuthThrottle) Exceeded(ctx context.Context, bucket string, rule ThrottleRule) (bool, error) {
	if t == nil || bucket == "" || rule.Limit <= 0 {
		return false, nil
	}
	if t.store != nil {
		var used int
		err := t.store.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM auth_rate_events
			WHERE bucket = $1 AND created_at > NOW() - ($2 * INTERVAL '1 second')`,
			bucket, int64(rule.Window/time.Second)).Scan(&used)
		if err == nil {
			return used >= rule.Limit, nil
		}
		return !t.allowLocal(bucket, rule, false), fmt.Errorf("read auth throttle: %w", err)
	}
	return !t.allowLocal(bucket, rule, false), nil
}

// Record stores one attempt unconditionally.
func (t *AuthThrottle) Record(ctx context.Context, bucket string) error {
	if t == nil || bucket == "" {
		return nil
	}
	if t.store != nil {
		if _, err := t.store.ExecContext(ctx, `INSERT INTO auth_rate_events (bucket) VALUES ($1)`, bucket); err == nil {
			return nil
		} else {
			t.recordLocal(bucket)
			return fmt.Errorf("record auth throttle: %w", err)
		}
	}
	t.recordLocal(bucket)
	return nil
}

// Reset clears a bucket, for example after a successful sign-in.
func (t *AuthThrottle) Reset(ctx context.Context, bucket string) error {
	if t == nil || bucket == "" {
		return nil
	}
	t.mu.Lock()
	delete(t.fallback, bucket)
	t.mu.Unlock()
	if t.store == nil {
		return nil
	}
	_, err := t.store.ExecContext(ctx, `DELETE FROM auth_rate_events WHERE bucket = $1`, bucket)
	return err
}

func (t *AuthThrottle) allowLocal(bucket string, rule ThrottleRule, record bool) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := t.now()
	cutoff := now.Add(-rule.Window)
	kept := t.fallback[bucket][:0]
	for _, at := range t.fallback[bucket] {
		if at.After(cutoff) {
			kept = append(kept, at)
		}
	}
	if len(kept) >= rule.Limit {
		t.fallback[bucket] = kept
		return false
	}
	if record {
		kept = append(kept, now)
	}
	t.fallback[bucket] = kept
	return true
}

func (t *AuthThrottle) recordLocal(bucket string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.fallback[bucket] = append(t.fallback[bucket], t.now())
}

// maybePrune removes events older than the longest window at most hourly.
func (t *AuthThrottle) maybePrune(ctx context.Context) {
	t.mu.Lock()
	due := t.now().Sub(t.lastPrune) > time.Hour
	if due {
		t.lastPrune = t.now()
	}
	t.mu.Unlock()
	if due {
		_, _ = t.store.ExecContext(ctx, `DELETE FROM auth_rate_events WHERE created_at < NOW() - INTERVAL '1 day'`)
	}
}
