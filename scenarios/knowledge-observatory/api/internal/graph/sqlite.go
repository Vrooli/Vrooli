package graph

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
	apidb "github.com/vrooli/api-core/database"
)

// SQLite implements Repository against SQLite.
type SQLite struct {
	DB *apidb.RoutedDB
}

var _ Repository = (*SQLite)(nil)

// NewSQLite returns a Repository backed by db.
func NewSQLite(db *apidb.RoutedDB) *SQLite { return &SQLite{DB: db} }

// UpsertEdges writes every edge in one transaction. Self-edges and edges with a
// missing endpoint are skipped rather than rejected, matching the previous
// behaviour: edge discovery is best-effort and a bad pair must not fail a batch.
func (s *SQLite) UpsertEdges(ctx context.Context, edges []Edge) error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("graph repository not configured")
	}
	if len(edges) == 0 {
		return nil
	}

	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx, `
INSERT INTO knowledge_relationships
  (id, source_id, target_id, relationship_type, weight, discovered_at)
VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(source_id, target_id, relationship_type) DO UPDATE SET
  weight = excluded.weight,
  discovered_at = CURRENT_TIMESTAMP
`)
	if err != nil {
		return fmt.Errorf("prepare edge upsert: %w", err)
	}
	defer stmt.Close()

	for _, e := range edges {
		e.SourceID = strings.TrimSpace(e.SourceID)
		e.TargetID = strings.TrimSpace(e.TargetID)
		e.RelationshipType = strings.TrimSpace(e.RelationshipType)
		if e.SourceID == "" || e.TargetID == "" || e.SourceID == e.TargetID {
			continue
		}
		if e.RelationshipType == "" {
			e.RelationshipType = "semantic_similarity"
		}
		if e.ID == "" {
			e.ID = uuid.NewString()
		}
		if _, err := stmt.ExecContext(ctx, e.ID, e.SourceID, e.TargetID, e.RelationshipType, e.Weight); err != nil {
			return fmt.Errorf("upsert edge: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit edges: %w", err)
	}
	return nil
}

func (s *SQLite) ListEdges(ctx context.Context, limit int) ([]Edge, error) {
	if s == nil || s.DB == nil {
		return nil, fmt.Errorf("graph repository not configured")
	}
	if limit <= 0 {
		limit = 500
	}
	rows, err := s.DB.QueryContext(ctx, `
SELECT id, source_id, target_id, relationship_type, weight, discovered_at
FROM knowledge_relationships
ORDER BY discovered_at DESC, id DESC
LIMIT ?
`, limit)
	if err != nil {
		return nil, fmt.Errorf("query edges: %w", err)
	}
	defer rows.Close()

	var out []Edge
	for rows.Next() {
		var e Edge
		if err := rows.Scan(&e.ID, &e.SourceID, &e.TargetID, &e.RelationshipType,
			&e.Weight, &e.DiscoveredAt); err != nil {
			return nil, fmt.Errorf("scan edge: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *SQLite) CountEdges(ctx context.Context) (int64, error) {
	if s == nil || s.DB == nil {
		return 0, fmt.Errorf("graph repository not configured")
	}
	var n int64
	if err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM knowledge_relationships`).Scan(&n); err != nil {
		return 0, fmt.Errorf("count edges: %w", err)
	}
	return n, nil
}
