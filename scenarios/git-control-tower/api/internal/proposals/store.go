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
	coredb "github.com/vrooli/api-core/database"
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
	return coredb.EnsureSchemas(ctx, s.db, coredb.SchemaProviderFunc(Schema))
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
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO gct_proposals (id, repository_id, state, revision, effort_ref, epoch, record_json, created_at, updated_at) VALUES (?,?,?,?,?,?,?,?,?)`,
		proposal.ID, proposal.RepositoryID, string(proposal.State), proposal.Revision, proposal.Work.EffortRef, proposal.Work.Epoch,
		string(data), proposal.CreatedAt.UTC().Format(time.RFC3339Nano), proposal.UpdatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO gct_proposal_revisions (proposal_id, revision, record_json) VALUES (?,?,?)`, proposal.ID, proposal.Revision, string(data)); err != nil {
		return err
	}
	return tx.Commit()
}

// Update replaces a proposal only when the stored revision and state still
// match what the caller read.
func (s *Store) Update(ctx context.Context, proposal Proposal, expectedRevision int, expectedState State) error {
	data, err := json.Marshal(proposal)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Preserve the old payload before a legacy record advances, and never
	// rewrite immutable content revisions for lifecycle/activity-only updates.
	if _, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO gct_proposal_revisions (proposal_id, revision, record_json) SELECT id, revision, record_json FROM gct_proposals WHERE id = ? AND revision = ? AND state = ?`, proposal.ID, expectedRevision, expectedState); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `UPDATE gct_proposals SET state = ?, revision = ?, record_json = ?, updated_at = ? WHERE id = ? AND revision = ? AND state = ?`,
		string(proposal.State), proposal.Revision, string(data), proposal.UpdatedAt.UTC().Format(time.RFC3339Nano), proposal.ID, expectedRevision, string(expectedState))
	if err != nil {
		return err
	}
	if rows, rowsErr := result.RowsAffected(); rowsErr != nil || rows != 1 {
		return ErrConflict
	}
	if proposal.Revision != expectedRevision {
		if _, err = tx.ExecContext(ctx, `INSERT INTO gct_proposal_revisions (proposal_id, revision, record_json) VALUES (?,?,?)`, proposal.ID, proposal.Revision, string(data)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) Revision(ctx context.Context, id string, revision int) (Proposal, error) {
	var raw string
	err := s.db.QueryRowContext(ctx, `SELECT record_json FROM gct_proposal_revisions WHERE proposal_id = ? AND revision = ?`, id, revision).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		current, currentErr := s.Get(ctx, id)
		if currentErr == nil && current.Revision == revision {
			return current, nil
		}
		return Proposal{}, ErrNotFound
	}
	if err != nil {
		return Proposal{}, err
	}
	var p Proposal
	return p, json.Unmarshal([]byte(raw), &p)
}

func (s *Store) SaveBlobs(ctx context.Context, blobs map[string][]byte) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for id, data := range blobs {
		if data == nil {
			data = []byte{}
		}
		if _, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO gct_proposal_blobs (blob_id, content) VALUES (?,?)`, id, data); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) Blob(ctx context.Context, id string) ([]byte, error) {
	if id == "" || id == DeletedBlob {
		return nil, nil
	}
	var data []byte
	err := s.db.QueryRowContext(ctx, `SELECT content FROM gct_proposal_blobs WHERE blob_id = ?`, id).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: captured content unavailable", ErrNotFound)
	}
	return data, err
}

// Combine creates the package and closes all exact source revisions in one
// transaction. A concurrent edit/apply cannot leave a partially combined inbox.
func (s *Store) Combine(ctx context.Context, p Proposal, sources []Proposal, actor Actor) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, old := range sources {
		original, err := json.Marshal(old)
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO gct_proposal_revisions (proposal_id, revision, record_json) VALUES (?,?,?)`, old.ID, old.Revision, string(original)); err != nil {
			return err
		}
		old.State, old.SupersededBy, old.UpdatedAt = StateSuperseded, p.ID, p.UpdatedAt
		old.Events = append(old.Events, Event{Action: "superseded", Actor: actor, Revision: old.Revision, Detail: "explicitly combined into " + p.ID, Timestamp: p.UpdatedAt})
		data, err := json.Marshal(old)
		if err != nil {
			return err
		}
		result, err := tx.ExecContext(ctx, `UPDATE gct_proposals SET state = ?, record_json = ?, updated_at = ? WHERE id = ? AND revision = ? AND state = ?`, string(old.State), string(data), old.UpdatedAt.UTC().Format(time.RFC3339Nano), old.ID, old.Revision, string(StateOpen))
		if err != nil {
			return err
		}
		if n, err := result.RowsAffected(); err != nil || n != 1 {
			return ErrConflict
		}
	}
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO gct_proposals (id, repository_id, state, revision, effort_ref, epoch, record_json, created_at, updated_at) VALUES (?,?,?,?,?,?,?,?,?)`, p.ID, p.RepositoryID, string(p.State), p.Revision, p.Work.EffortRef, p.Work.Epoch, string(data), p.CreatedAt.UTC().Format(time.RFC3339Nano), p.UpdatedAt.UTC().Format(time.RFC3339Nano)); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO gct_proposal_revisions (proposal_id, revision, record_json) VALUES (?,?,?)`, p.ID, p.Revision, string(data)); err != nil {
		return err
	}
	return tx.Commit()
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
	NewestFirst  bool
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
	if filter.NewestFirst {
		query += ` ORDER BY created_at DESC,id DESC LIMIT ?`
	} else {
		query += ` ORDER BY created_at ASC LIMIT ?`
	}
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
// Publish takes a short write lease across ref publication so concurrent edits
// cannot change the reviewed revision. Nonempty OID on error means publication
// succeeded but durable recording failed; never restore the old index then.
func (s *Store) Publish(ctx context.Context, reviewed Proposal, actor Actor, publish func(context.Context) (string, error), now time.Time) (string, Proposal, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", Proposal{}, err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `UPDATE gct_proposals SET revision = revision WHERE id = ? AND revision = ? AND state = ?`, reviewed.ID, reviewed.Revision, string(StateOpen))
	if err != nil {
		return "", Proposal{}, err
	}
	if n, err := result.RowsAffected(); err != nil || n != 1 {
		return "", Proposal{}, ErrConflict
	}
	var raw string
	if err = tx.QueryRowContext(ctx, `SELECT record_json FROM gct_proposals WHERE id = ?`, reviewed.ID).Scan(&raw); err != nil {
		return "", Proposal{}, err
	}
	var p Proposal
	if err = json.Unmarshal([]byte(raw), &p); err != nil {
		return "", Proposal{}, err
	}
	oid, err := publish(ctx)
	if err != nil {
		return oid, p, err
	}
	if oid == "" {
		return "", p, fmt.Errorf("publisher returned no commit identity")
	}
	p.State, p.CommitOID, p.UpdatedAt, p.CommittedAt = StateCommitted, oid, now, &now
	p.Events = append(p.Events, Event{Action: "committed", Actor: actor, Revision: p.Revision, Detail: "published verified candidate " + oid, Timestamp: now})
	data, err := json.Marshal(p)
	if err != nil {
		return oid, p, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE gct_proposals SET state = ?, record_json = ?, updated_at = ? WHERE id = ?`, string(p.State), string(data), now.UTC().Format(time.RFC3339Nano), p.ID); err != nil {
		return oid, p, err
	}
	return oid, p, tx.Commit()
}

// CountOpen returns the number of open proposals for a repository.
func (s *Store) CountOpen(ctx context.Context, repositoryID string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM gct_proposals WHERE repository_id = ? AND state = ?`, repositoryID, string(StateOpen)).Scan(&count)
	return count, err
}

// AvailableRevisions reports retained records, never guesses missing legacy history.
func (s *Store) AvailableRevisions(ctx context.Context, id string) ([]int, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT revision FROM gct_proposal_revisions WHERE proposal_id = ? ORDER BY revision`, id)
	if err != nil {
		return nil, err
	}
	values := []int{}
	for rows.Next() {
		var value int
		if err = rows.Scan(&value); err != nil {
			rows.Close()
			return nil, err
		}
		values = append(values, value)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	current, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if len(values) == 0 || values[len(values)-1] != current.Revision {
		values = append(values, current.Revision)
	}
	return values, nil
}

func (s *Store) captureSize(ctx context.Context, files []File) (int64, error) {
	seen := map[string]bool{}
	var total int64
	for _, file := range files {
		for _, id := range []string{file.BaseBlobID, file.BlobID} {
			if id == "" || seen[id] {
				continue
			}
			seen[id] = true
			var size int64
			err := s.db.QueryRowContext(ctx, `SELECT length(content) FROM gct_proposal_blobs WHERE blob_id = ?`, id).Scan(&size)
			if err != nil {
				return 0, err
			}
			total += size
			if total > MaxCaptureBytes {
				return total, nil
			}
		}
	}
	return total, nil
}
