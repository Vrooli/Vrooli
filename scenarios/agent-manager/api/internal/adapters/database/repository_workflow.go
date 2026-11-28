package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/repository"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type workflowRepository struct {
	db  *DB
	log *logrus.Logger
}

var _ repository.WorkflowRepository = (*workflowRepository)(nil)

type workflowRevisionRow struct {
	ID              string     `db:"id"`
	Owner           string     `db:"owner"`
	Key             string     `db:"workflow_key"`
	SemanticVersion string     `db:"semantic_version"`
	Digest          string     `db:"digest"`
	DefinitionJSON  string     `db:"definition_json"`
	SourcePath      string     `db:"source_path"`
	SourceHash      string     `db:"source_hash"`
	SourceUpdatedAt SQLiteTime `db:"source_updated_at"`
	Active          bool       `db:"active"`
	CreatedAt       SQLiteTime `db:"created_at"`
}

const workflowRevisionColumns = `id, owner, workflow_key, semantic_version, digest,
	definition_json, source_path, source_hash, source_updated_at, active, created_at`

func (r workflowRevisionRow) domain() (*domain.WorkflowRevision, error) {
	id, err := uuid.Parse(r.ID)
	if err != nil {
		return nil, fmt.Errorf("parse workflow revision id: %w", err)
	}
	var definition domain.WorkflowDefinition
	if err := json.Unmarshal([]byte(r.DefinitionJSON), &definition); err != nil {
		return nil, fmt.Errorf("decode workflow revision: %w", err)
	}
	return &domain.WorkflowRevision{ID: id, Owner: r.Owner, Key: r.Key, SemanticVersion: r.SemanticVersion, Digest: r.Digest, Definition: definition, SourcePath: r.SourcePath, SourceHash: r.SourceHash, SourceUpdatedAt: r.SourceUpdatedAt.Time(), Active: r.Active, CreatedAt: r.CreatedAt.Time()}, nil
}

func (r *workflowRepository) ActivateBatch(ctx context.Context, revisions []*domain.WorkflowRevision) error {
	if len(revisions) == 0 {
		return nil
	}
	return r.db.WithTransaction(ctx, func(tx *Tx) error {
		seen := map[string]bool{}
		for _, revision := range revisions {
			if revision == nil {
				return errors.New("nil workflow revision")
			}
			identity := revision.Owner + "\x00" + revision.Key
			if seen[identity] {
				return fmt.Errorf("duplicate workflow activation %s", revision.Key)
			}
			seen[identity] = true
			if revision.ID == uuid.Nil {
				revision.ID = uuid.New()
			}
			if revision.CreatedAt.IsZero() {
				revision.CreatedAt = time.Now().UTC()
			}
			definition, err := json.Marshal(revision.Definition)
			if err != nil {
				return fmt.Errorf("encode workflow %s: %w", revision.Key, err)
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO workflow_revisions
				(id, owner, workflow_key, semantic_version, digest, definition_json, source_path, source_hash, source_updated_at, active, created_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 0, ?)
				ON CONFLICT(digest) DO NOTHING`, revision.ID, revision.Owner, revision.Key, revision.SemanticVersion, revision.Digest, string(definition), revision.SourcePath, revision.SourceHash, SQLiteTime(revision.SourceUpdatedAt), SQLiteTime(revision.CreatedAt)); err != nil {
				return fmt.Errorf("insert workflow revision: %w", err)
			}
		}
		for _, revision := range revisions {
			if _, err := tx.ExecContext(ctx, `UPDATE workflow_revisions SET active = 0 WHERE owner = ? AND workflow_key = ? AND active = 1`, revision.Owner, revision.Key); err != nil {
				return err
			}
			result, err := tx.ExecContext(ctx, `UPDATE workflow_revisions SET active = 1 WHERE digest = ? AND owner = ? AND workflow_key = ?`, revision.Digest, revision.Owner, revision.Key)
			if err != nil {
				return err
			}
			if count, _ := result.RowsAffected(); count != 1 {
				return fmt.Errorf("workflow revision %s was not stored", revision.Digest)
			}
		}
		return nil
	})
}

func (r *workflowRepository) GetActive(ctx context.Context, owner, key string) (*domain.WorkflowRevision, error) {
	return r.get(ctx, `SELECT `+workflowRevisionColumns+` FROM workflow_revisions WHERE owner = ? AND workflow_key = ? AND active = 1`, owner, key)
}

func (r *workflowRepository) GetByDigest(ctx context.Context, digest string) (*domain.WorkflowRevision, error) {
	return r.get(ctx, `SELECT `+workflowRevisionColumns+` FROM workflow_revisions WHERE digest = ?`, digest)
}

func (r *workflowRepository) get(ctx context.Context, query string, args ...any) (*domain.WorkflowRevision, error) {
	var row workflowRevisionRow
	if err := r.db.GetContext(ctx, &row, query, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return row.domain()
}

func (r *workflowRepository) List(ctx context.Context, owner, key string, filter repository.ListFilter) ([]*domain.WorkflowRevision, error) {
	query := `SELECT ` + workflowRevisionColumns + ` FROM workflow_revisions WHERE owner = ?`
	args := []any{owner}
	if key != "" {
		query += ` AND workflow_key = ?`
		args = append(args, key)
	}
	query += ` ORDER BY created_at DESC, digest ASC`
	query, pageArgs := appendLimitOffset(query, filter.Limit, filter.Offset)
	args = append(args, pageArgs...)
	var rows []workflowRevisionRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, err
	}
	out := make([]*domain.WorkflowRevision, 0, len(rows))
	for _, row := range rows {
		revision, err := row.domain()
		if err != nil {
			return nil, err
		}
		out = append(out, revision)
	}
	return out, nil
}
