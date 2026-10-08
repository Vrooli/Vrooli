package gates

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type Store struct {
	db  SQLExecutor
	now func() time.Time
}

func NewStore(db SQLExecutor, now func() time.Time) *Store {
	if now == nil {
		now = time.Now
	}
	return &Store{db: db, now: now}
}

func (s *Store) Raise(ctx context.Context, threadID, ownerID, scope, withheld, unblock string, ttl time.Duration) (Gate, error) {
	if threadID == "" || ownerID == "" || scope == "" || ttl <= 0 {
		return Gate{}, fmt.Errorf("thread, owner, scope, and positive ttl are required")
	}
	now := s.now().UTC()
	g := Gate{ID: uuid.NewString(), ThreadID: threadID, OwnerID: ownerID, Scope: scope, Withheld: withheld, Unblock: unblock, CreatedAt: now, ExpiresAt: now.Add(ttl), Status: Pending, GrantOnce: true}
	_, err := s.db.ExecContext(ctx, `INSERT INTO switchboard_capability_gates(id,thread_id,owner_id,scope,withheld,unblock,created_at,expires_at,status,grant_once) VALUES(?,?,?,?,?,?,?,?,?,1)`, g.ID, g.ThreadID, g.OwnerID, g.Scope, g.Withheld, g.Unblock, g.CreatedAt.Format(time.RFC3339Nano), g.ExpiresAt.Format(time.RFC3339Nano), g.Status)
	if err != nil {
		return Gate{}, fmt.Errorf("persist gate: %w", err)
	}
	return g, nil
}

func (s *Store) Get(ctx context.Context, id string) (Gate, bool, error) {
	var g Gate
	var created, expires string
	err := s.db.QueryRowContext(ctx, `SELECT id,thread_id,owner_id,scope,withheld,unblock,created_at,expires_at,status,grant_once FROM switchboard_capability_gates WHERE id=?`, id).Scan(&g.ID, &g.ThreadID, &g.OwnerID, &g.Scope, &g.Withheld, &g.Unblock, &created, &expires, &g.Status, &g.GrantOnce)
	if err == sql.ErrNoRows {
		return Gate{}, false, nil
	}
	if err != nil {
		return Gate{}, false, fmt.Errorf("read gate: %w", err)
	}
	var parseErr error
	g.CreatedAt, parseErr = time.Parse(time.RFC3339Nano, created)
	if parseErr != nil {
		return Gate{}, false, fmt.Errorf("parse gate created_at: %w", parseErr)
	}
	g.ExpiresAt, parseErr = time.Parse(time.RFC3339Nano, expires)
	if parseErr != nil {
		return Gate{}, false, fmt.Errorf("parse gate expires_at: %w", parseErr)
	}
	if g.Status == Pending && !s.now().Before(g.ExpiresAt) {
		_, err = s.db.ExecContext(ctx, `UPDATE switchboard_capability_gates SET status=? WHERE id=? AND status=?`, Expired, id, Pending)
		if err != nil {
			return Gate{}, false, fmt.Errorf("expire gate: %w", err)
		}
		g.Status = Expired
	}
	return g, true, nil
}

func (s *Store) Answer(ctx context.Context, id, actorID string, grant bool) (Gate, error) {
	g, ok, err := s.Get(ctx, id)
	if err != nil {
		return Gate{}, err
	}
	if !ok {
		return Gate{}, fmt.Errorf("gate %q not found", id)
	}
	if actorID != g.OwnerID {
		return Gate{}, ErrNotOwner
	}
	if g.Status != Pending {
		return Gate{}, ErrNotPending
	}
	status := Denied
	if grant {
		status = Granted
	}
	result, err := s.db.ExecContext(ctx, `UPDATE switchboard_capability_gates SET status=? WHERE id=? AND status=?`, status, id, Pending)
	if err != nil {
		return Gate{}, fmt.Errorf("answer gate: %w", err)
	}
	if n, _ := result.RowsAffected(); n != 1 {
		return Gate{}, ErrNotPending
	}
	g.Status = status
	return g, nil
}
