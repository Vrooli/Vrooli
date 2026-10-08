package proposals

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"git-control-tower/internal/dbschema"
)

// Store keeps anchors and proposals in GCT's database. Each record is one
// JSON document with indexed lookup columns; writes compare the stored
// revision and state so concurrent edits and applies cannot both win.
type Store struct{ db dbschema.DB }

func NewStore(db dbschema.DB) (*Store, error) {
	if db == nil {
		return nil, fmt.Errorf("proposal store requires a database")
	}
	s := &Store{db: db}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s, s.ensureSchema(ctx)
}

func (s *Store) ensureSchema(ctx context.Context) error {
	for _, statement := range []string{
		`CREATE TABLE IF NOT EXISTS gct_proposal_anchors (
			id TEXT PRIMARY KEY, repository_id TEXT NOT NULL, effort_ref TEXT NOT NULL DEFAULT '',
			epoch TEXT NOT NULL DEFAULT '', record_json TEXT NOT NULL, created_at TEXT NOT NULL)`,
		`CREATE INDEX IF NOT EXISTS idx_gct_proposal_anchors_work ON gct_proposal_anchors(repository_id, effort_ref, epoch, created_at)`,
		`CREATE TABLE IF NOT EXISTS gct_proposals (
			id TEXT PRIMARY KEY, repository_id TEXT NOT NULL, state TEXT NOT NULL, revision INTEGER NOT NULL,
			effort_ref TEXT NOT NULL DEFAULT '', epoch TEXT NOT NULL DEFAULT '', record_json TEXT NOT NULL,
			created_at TEXT NOT NULL, updated_at TEXT NOT NULL)`,
		`CREATE INDEX IF NOT EXISTS idx_gct_proposals_queue ON gct_proposals(repository_id, state, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_gct_proposals_work ON gct_proposals(repository_id, effort_ref, epoch, state)`,
	} {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("create proposal schema: %w", err)
		}
	}
	return nil
}

func (s *Store) SaveAnchor(ctx context.Context, anchor Anchor) error {
	data, err := json.Marshal(anchor)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO gct_proposal_anchors (id, repository_id, effort_ref, epoch, record_json, created_at) VALUES (?,?,?,?,?,?)`,
		anchor.ID, anchor.RepositoryID, anchor.EffortRef, anchor.Epoch, string(data), anchor.CreatedAt.UTC().Format(time.RFC3339Nano))
	return err
}

func (s *Store) GetAnchor(ctx context.Context, id string) (Anchor, error) {
	var raw string
	err := s.db.QueryRowContext(ctx, `SELECT record_json FROM gct_proposal_anchors WHERE id = ?`, id).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return Anchor{}, fmt.Errorf("anchor %s: %w", id, ErrNotFound)
	}
	if err != nil {
		return Anchor{}, err
	}
	var anchor Anchor
	return anchor, json.Unmarshal([]byte(raw), &anchor)
}

// LatestAnchor returns the newest anchor for an effort and epoch, or false.
func (s *Store) LatestAnchor(ctx context.Context, repositoryID, effortRef, epoch string) (Anchor, bool, error) {
	var raw string
	err := s.db.QueryRowContext(ctx, `SELECT record_json FROM gct_proposal_anchors WHERE repository_id = ? AND effort_ref = ? AND epoch = ? ORDER BY created_at DESC LIMIT 1`,
		repositoryID, effortRef, epoch).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return Anchor{}, false, nil
	}
	if err != nil {
		return Anchor{}, false, err
	}
	var anchor Anchor
	return anchor, true, json.Unmarshal([]byte(raw), &anchor)
}

func (s *Store) Insert(ctx context.Context, proposal Proposal) error {
	data, err := json.Marshal(proposal)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO gct_proposals (id, repository_id, state, revision, effort_ref, epoch, record_json, created_at, updated_at) VALUES (?,?,?,?,?,?,?,?,?)`,
		proposal.ID, proposal.RepositoryID, string(proposal.State), proposal.Revision, proposal.Work.EffortRef, proposal.Work.Epoch,
		string(data), proposal.CreatedAt.UTC().Format(time.RFC3339Nano), proposal.UpdatedAt.UTC().Format(time.RFC3339Nano))
	return err
}

// Update replaces a proposal only when the stored revision and state still
// match what the caller read.
func (s *Store) Update(ctx context.Context, proposal Proposal, expectedRevision int, expectedState State) error {
	data, err := json.Marshal(proposal)
	if err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, `UPDATE gct_proposals SET state = ?, revision = ?, record_json = ?, updated_at = ? WHERE id = ? AND revision = ? AND state = ?`,
		string(proposal.State), proposal.Revision, string(data), proposal.UpdatedAt.UTC().Format(time.RFC3339Nano), proposal.ID, expectedRevision, string(expectedState))
	if err != nil {
		return err
	}
	if rows, rowsErr := result.RowsAffected(); rowsErr != nil || rows != 1 {
		return ErrConflict
	}
	return nil
}

func (s *Store) Get(ctx context.Context, id string) (Proposal, error) {
	var raw string
	err := s.db.QueryRowContext(ctx, `SELECT record_json FROM gct_proposals WHERE id = ?`, strings.TrimSpace(id)).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return Proposal{}, fmt.Errorf("proposal %s: %w", id, ErrNotFound)
	}
	if err != nil {
		return Proposal{}, err
	}
	var proposal Proposal
	return proposal, json.Unmarshal([]byte(raw), &proposal)
}

// ListFilter selects proposals. Empty States means open only.
type ListFilter struct {
	RepositoryID string
	States       []State
	EffortRef    string
	Epoch        string
	Limit        int
}

// List returns proposals oldest first, which is queue order.
func (s *Store) List(ctx context.Context, filter ListFilter) ([]Proposal, error) {
	states := filter.States
	if len(states) == 0 {
		states = []State{StateOpen}
	}
	query := `SELECT record_json FROM gct_proposals WHERE repository_id = ? AND state IN (` + strings.TrimSuffix(strings.Repeat("?,", len(states)), ",") + `)`
	args := []any{filter.RepositoryID}
	for _, state := range states {
		args = append(args, string(state))
	}
	if filter.EffortRef != "" {
		query += ` AND effort_ref = ?`
		args = append(args, filter.EffortRef)
	}
	if filter.Epoch != "" {
		query += ` AND epoch = ?`
		args = append(args, filter.Epoch)
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	query += ` ORDER BY created_at ASC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	var raws []string
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			_ = rows.Close()
			return nil, err
		}
		raws = append(raws, raw)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	proposals := make([]Proposal, 0, len(raws))
	for _, raw := range raws {
		var proposal Proposal
		if err := json.Unmarshal([]byte(raw), &proposal); err != nil {
			return nil, fmt.Errorf("decode proposal: %w", err)
		}
		proposals = append(proposals, proposal)
	}
	return proposals, nil
}

// CountOpen returns the number of open proposals for a repository.
func (s *Store) CountOpen(ctx context.Context, repositoryID string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM gct_proposals WHERE repository_id = ? AND state = ?`, repositoryID, string(StateOpen)).Scan(&count)
	return count, err
}
