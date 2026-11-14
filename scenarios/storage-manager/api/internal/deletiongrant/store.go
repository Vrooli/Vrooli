// Package deletiongrant persists operator decisions about whether an agent may
// delete a set of filesystem paths, and projects the approved, unexpired ones
// as agent-policy "allow" path rules. It reuses the operatordecision ask/answer
// rail (the Raiser writes pending asks here; the resolution webhook marks them
// approved) rather than inventing a new approval mechanism. An approved,
// unexpired ask IS the grant; there is no second grant concept.
package deletiongrant

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"time"

	"github.com/vrooli/api-core/operatordecision"
)

//go:embed schema.sql
var schemaSQL string

// Schema returns the table this package owns, registered in modules.AllSchemas.
func Schema() string { return schemaSQL }

const (
	// stateResolved marks a decided ask. operatordecision's own pending state is
	// used while the ask is outstanding (so it counts toward the flood cap);
	// once resolved it never counts again.
	stateResolved = "resolved"

	// Extra keys carried on the ask and read on resolution / projection.
	extraPaths          = "paths"            // JSON array of requested paths
	extraAnswer         = "answer"           // "approve" | "deny"
	extraGrantExpiresAt = "grant_expires_at" // RFC3339Nano; when the deletion authority lapses
	extraRequestedBy    = "requested_by"     // the agent subject that asked
	extraReason         = "reason"           // the agent's stated reason

	ts = time.RFC3339Nano
)

// DB is the subset of a database handle this store needs. Declaring it as an
// interface (not *sql.DB) keeps the scenario eligible for per-request routing.
type DB interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Store persists deletion asks and satisfies operatordecision.PendingStore.
type Store struct{ db DB }

// NewStore builds a store over db.
func NewStore(db DB) *Store { return &Store{db: db} }

// GrantRule is one approved, unexpired deletion authority for one path.
type GrantRule struct {
	Root      string
	GrantID   string
	Approver  string
	ExpiresAt time.Time
}

// Get returns the ask for a correlation id, or nil when none exists.
func (s *Store) Get(ctx context.Context, id string) (*operatordecision.PendingAsk, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, subject, state, raised_at, expires_at, event_id, published, resolved_at, resolved_by, extra FROM deletion_asks WHERE id = ?`, id)
	var (
		ask        operatordecision.PendingAsk
		raised     string
		expires    string
		published  int
		resolvedAt sql.NullString
		extraJSON  string
	)
	switch err := row.Scan(&ask.ID, &ask.Subject, &ask.State, &raised, &expires, &ask.EventID, &published, &resolvedAt, &ask.ResolvedBy, &extraJSON); err {
	case nil:
	case sql.ErrNoRows:
		return nil, nil
	default:
		return nil, err
	}
	ask.RaisedAt, _ = time.Parse(ts, raised)
	ask.ExpiresAt, _ = time.Parse(ts, expires)
	ask.Published = published != 0
	if resolvedAt.Valid && resolvedAt.String != "" {
		if t, err := time.Parse(ts, resolvedAt.String); err == nil {
			ask.ResolvedAt = &t
		}
	}
	if extraJSON != "" {
		_ = json.Unmarshal([]byte(extraJSON), &ask.Extra)
	}
	return &ask, nil
}

// Put inserts or replaces an ask by id.
func (s *Store) Put(ctx context.Context, ask *operatordecision.PendingAsk) error {
	extraJSON := []byte("{}")
	if len(ask.Extra) > 0 {
		if encoded, err := json.Marshal(ask.Extra); err == nil {
			extraJSON = encoded
		}
	}
	var resolvedAt any
	if ask.ResolvedAt != nil {
		resolvedAt = ask.ResolvedAt.UTC().Format(ts)
	}
	published := 0
	if ask.Published {
		published = 1
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO deletion_asks (id, subject, state, raised_at, expires_at, event_id, published, resolved_at, resolved_by, extra)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET subject=excluded.subject, state=excluded.state, raised_at=excluded.raised_at, expires_at=excluded.expires_at, event_id=excluded.event_id, published=excluded.published, resolved_at=excluded.resolved_at, resolved_by=excluded.resolved_by, extra=excluded.extra`,
		ask.ID, ask.Subject, ask.State, ask.RaisedAt.UTC().Format(ts), ask.ExpiresAt.UTC().Format(ts), ask.EventID, published, resolvedAt, ask.ResolvedBy, string(extraJSON))
	return err
}

// CountOutstandingForSubject counts un-decided, un-expired asks for a subject.
func (s *Store) CountOutstandingForSubject(ctx context.Context, subject string, now time.Time) (int, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT expires_at FROM deletion_asks WHERE subject = ? AND state = ?`, subject, operatordecision.PendingStatePending)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var expires string
		if err := rows.Scan(&expires); err != nil {
			return 0, err
		}
		if at, err := time.Parse(ts, expires); err == nil && now.Before(at) {
			count++
		}
	}
	return count, rows.Err()
}

// ActiveGrantRoots returns one GrantRule per path of every approved ask whose
// deletion authority has not yet lapsed. A denied, pending, or expired ask
// yields nothing. The policy snapshot projects these as "allow" rules.
func (s *Store) ActiveGrantRoots(ctx context.Context, now time.Time) ([]GrantRule, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, resolved_by, extra FROM deletion_asks WHERE state = ?`, stateResolved)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []GrantRule
	for rows.Next() {
		var id, resolvedBy, extraJSON string
		if err := rows.Scan(&id, &resolvedBy, &extraJSON); err != nil {
			return nil, err
		}
		extra := map[string]string{}
		if extraJSON != "" {
			_ = json.Unmarshal([]byte(extraJSON), &extra)
		}
		if extra[extraAnswer] != "approve" {
			continue
		}
		expiresAt, err := time.Parse(ts, extra[extraGrantExpiresAt])
		if err != nil || !expiresAt.After(now) {
			continue
		}
		var paths []string
		if err := json.Unmarshal([]byte(extra[extraPaths]), &paths); err != nil {
			continue
		}
		for _, path := range paths {
			if path == "" {
				continue
			}
			out = append(out, GrantRule{Root: path, GrantID: id, Approver: resolvedBy, ExpiresAt: expiresAt})
		}
	}
	return out, rows.Err()
}
