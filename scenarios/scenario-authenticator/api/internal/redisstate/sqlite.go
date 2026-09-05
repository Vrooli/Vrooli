package redisstate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// SQLiteStore is the durable single-node Store.
//
// It exists because the state behind this seam is a set of security controls
// rather than a cache. Memory cannot be the single-node answer: a process
// restart drops the revocation blacklist, and a dropped blacklist re-admits a
// revoked token. SQLiteStore keeps that guarantee across restart with no Redis
// server, which is what makes the scenario runnable on a host where no
// vendorable RESP server exists.
//
// What it deliberately does NOT provide is cross-replica sharing. A deployment
// running more than one replica still needs Redis, because a blacklist that is
// not shared does not revoke. That boundary is recorded per tier in
// .vrooli/service.json under tier_feasibility.
type SQLiteStore struct {
	db  Querier
	now func() time.Time
}

// Querier is the minimal context-routed database surface this store needs.
// *database.RoutedDB satisfies it, which keeps hot state on the same routed
// handle as the rest of the scenario — including an installed test pool. A
// plain *sql.DB satisfies it too, which is what tests use.
type Querier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Schema is the durable hot-state schema, declared beside the code that
// interprets it so it travels with this package rather than with a migration
// directory.
//
// expires_at is unix nanoseconds and NULL means no expiry. Expiry is enforced
// on every read rather than only by the sweep: a store that returned an expired
// blacklist entry as absent before the sweep ran would re-admit a revoked token
// for the length of the sweep interval.
func Schema() string {
	return `
CREATE TABLE IF NOT EXISTS hot_state_values (
    key        TEXT PRIMARY KEY,
    value      TEXT NOT NULL,
    expires_at INTEGER
);
CREATE INDEX IF NOT EXISTS hot_state_values_expires_at_idx
    ON hot_state_values (expires_at);
CREATE TABLE IF NOT EXISTS hot_state_set_members (
    key    TEXT NOT NULL,
    member TEXT NOT NULL,
    PRIMARY KEY (key, member)
);
`
}

// NewSQLiteStore builds a durable Store over an already-open database handle.
func NewSQLiteStore(db Querier) (*SQLiteStore, error) {
	if db == nil {
		return nil, errors.New("durable hot-state store requires a database handle")
	}
	return &SQLiteStore{db: db, now: time.Now}, nil
}

// NewSQLiteStoreWithClock is the test variant with an injectable clock so TTL
// expiry is deterministic.
func NewSQLiteStoreWithClock(db Querier, now func() time.Time) (*SQLiteStore, error) {
	store, err := NewSQLiteStore(db)
	if err != nil {
		return nil, err
	}
	if now != nil {
		store.now = now
	}
	return store, nil
}

var _ Store = (*SQLiteStore)(nil)

func (s *SQLiteStore) stamp() int64 { return s.now().UnixNano() }

// liveRow is the shared predicate for "present and not expired".
const liveRow = `(expires_at IS NULL OR expires_at > ?)`

func (s *SQLiteStore) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	var expires any
	if ttl > 0 {
		expires = s.now().Add(ttl).UnixNano()
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO hot_state_values (key, value, expires_at) VALUES (?, ?, ?)
ON CONFLICT(key) DO UPDATE SET value = excluded.value, expires_at = excluded.expires_at`,
		key, value, expires)
	if err != nil {
		return fmt.Errorf("hot state set %q: %w", key, err)
	}
	return nil
}

func (s *SQLiteStore) Get(ctx context.Context, key string) (string, bool, error) {
	var value string
	err := s.db.QueryRowContext(ctx,
		`SELECT value FROM hot_state_values WHERE key = ? AND `+liveRow, key, s.stamp()).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("hot state get %q: %w", key, err)
	}
	return value, true, nil
}

func (s *SQLiteStore) Del(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(keys)), ",")
	args := make([]any, 0, len(keys))
	for _, key := range keys {
		args = append(args, key)
	}
	// A key is either a value or a set, never both, so both tables are cleared.
	for _, table := range []string{"hot_state_values", "hot_state_set_members"} {
		if _, err := s.db.ExecContext(ctx,
			"DELETE FROM "+table+" WHERE key IN ("+placeholders+")", args...); err != nil {
			return fmt.Errorf("hot state delete from %s: %w", table, err)
		}
	}
	return nil
}

func (s *SQLiteStore) Exists(ctx context.Context, key string) (bool, error) {
	var present int
	err := s.db.QueryRowContext(ctx,
		`SELECT 1 FROM hot_state_values WHERE key = ? AND `+liveRow, key, s.stamp()).Scan(&present)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("hot state exists %q: %w", key, err)
	}
	return true, nil
}

func (s *SQLiteStore) SAdd(ctx context.Context, key string, members ...string) error {
	for _, member := range members {
		if _, err := s.db.ExecContext(ctx,
			`INSERT INTO hot_state_set_members (key, member) VALUES (?, ?)
ON CONFLICT(key, member) DO NOTHING`, key, member); err != nil {
			return fmt.Errorf("hot state sadd %q: %w", key, err)
		}
	}
	return nil
}

func (s *SQLiteStore) SRem(ctx context.Context, key string, members ...string) error {
	for _, member := range members {
		if _, err := s.db.ExecContext(ctx,
			`DELETE FROM hot_state_set_members WHERE key = ? AND member = ?`, key, member); err != nil {
			return fmt.Errorf("hot state srem %q: %w", key, err)
		}
	}
	return nil
}

func (s *SQLiteStore) SMembers(ctx context.Context, key string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT member FROM hot_state_set_members WHERE key = ?`, key)
	if err != nil {
		return nil, fmt.Errorf("hot state smembers %q: %w", key, err)
	}
	defer rows.Close()
	members := make([]string, 0)
	for rows.Next() {
		var member string
		if err := rows.Scan(&member); err != nil {
			return nil, fmt.Errorf("hot state smembers %q: %w", key, err)
		}
		members = append(members, member)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("hot state smembers %q: %w", key, err)
	}
	return members, nil
}

// Incr increments in a single statement rather than reading and then writing.
// A read-then-write implementation lets two concurrent callers observe the same
// counter value, which silently doubles every rate limit built on it.
//
// An expired counter is treated as absent, so it restarts at 1 and drops its
// stale expiry — the same behaviour the in-memory implementation has.
func (s *SQLiteStore) Incr(ctx context.Context, key string) (int64, error) {
	now := s.stamp()
	var next int64
	err := s.db.QueryRowContext(ctx, `
INSERT INTO hot_state_values (key, value, expires_at) VALUES (?, '1', NULL)
ON CONFLICT(key) DO UPDATE SET
    value = CAST(
        CASE WHEN hot_state_values.expires_at IS NOT NULL AND hot_state_values.expires_at <= ?
             THEN 0
             ELSE CAST(hot_state_values.value AS INTEGER)
        END + 1 AS TEXT),
    expires_at = CASE WHEN hot_state_values.expires_at IS NOT NULL AND hot_state_values.expires_at <= ?
                      THEN NULL
                      ELSE hot_state_values.expires_at
                 END
RETURNING CAST(value AS INTEGER)`, key, now, now).Scan(&next)
	if err != nil {
		return 0, fmt.Errorf("hot state incr %q: %w", key, err)
	}
	return next, nil
}

func (s *SQLiteStore) Expire(ctx context.Context, key string, ttl time.Duration) error {
	var expires any
	if ttl > 0 {
		expires = s.now().Add(ttl).UnixNano()
	}
	if _, err := s.db.ExecContext(ctx,
		`UPDATE hot_state_values SET expires_at = ? WHERE key = ? AND `+liveRow,
		expires, key, s.stamp()); err != nil {
		return fmt.Errorf("hot state expire %q: %w", key, err)
	}
	return nil
}

// Sweep deletes rows whose TTL has passed and reports how many it removed.
// Reads already treat an expired row as absent, so this reclaims space rather
// than enforcing correctness: without it an abandoned key grows the file
// forever.
func (s *SQLiteStore) Sweep(ctx context.Context) (int64, error) {
	result, err := s.db.ExecContext(ctx,
		`DELETE FROM hot_state_values WHERE expires_at IS NOT NULL AND expires_at <= ?`, s.stamp())
	if err != nil {
		return 0, fmt.Errorf("hot state sweep: %w", err)
	}
	removed, err := result.RowsAffected()
	if err != nil {
		return 0, nil
	}
	return removed, nil
}

// RunSweeper sweeps on an interval until ctx is cancelled. Sweep failures are
// not fatal: reclaiming space is best effort, and correctness does not depend
// on it.
func (s *SQLiteStore) RunSweeper(ctx context.Context, interval time.Duration, onError func(error)) {
	if interval <= 0 {
		interval = time.Hour
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := s.Sweep(ctx); err != nil && onError != nil {
				onError(err)
			}
		}
	}
}
