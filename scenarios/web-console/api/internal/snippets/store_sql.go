// DOC: docs/internal/SECURITY-POSTURE.md#sql-injection-prevention
package snippets

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"web-console/internal/dbx"
)

type SQLStore struct{ db dbx.Handle }

func NewSQLStore(db dbx.Handle) *SQLStore { return &SQLStore{db: db} }

const snippetColumns = `id, name, body, color, pinned, use_count, last_used_at, sort_order, created_at, updated_at`

type rowScanner interface{ Scan(dest ...any) error }

func scanSnippet(row rowScanner) (Snippet, error) {
	var snippet Snippet
	var pinned int
	if err := row.Scan(
		&snippet.ID, &snippet.Name, &snippet.Body, &snippet.Color, &pinned,
		&snippet.UseCount, &snippet.LastUsedAt, &snippet.SortOrder,
		&snippet.CreatedAt, &snippet.UpdatedAt,
	); err != nil {
		return Snippet{}, err
	}
	snippet.Pinned = pinned != 0
	return snippet, nil
}

func (s *SQLStore) List(ctx context.Context) ([]Snippet, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+snippetColumns+` FROM message_snippets ORDER BY pinned DESC, last_used_at DESC, use_count DESC, id`)
	if err != nil {
		return nil, fmt.Errorf("list snippets: %w", err)
	}
	defer rows.Close()
	out := make([]Snippet, 0)
	for rows.Next() {
		snippet, err := scanSnippet(rows)
		if err != nil {
			return nil, fmt.Errorf("scan snippet: %w", err)
		}
		out = append(out, snippet)
	}
	return out, rows.Err()
}

func (s *SQLStore) Upsert(ctx context.Context, req UpsertRequest) (Snippet, error) {
	if err := req.Validate(); err != nil {
		return Snippet{}, err
	}
	id := req.ID
	if id == "" {
		id = uuid.NewString()
	}
	now := FormatTime(time.Now())
	pinnedExpr := "message_snippets.pinned"
	if req.HasPinned {
		pinnedExpr = "excluded.pinned"
	}
	row := s.db.QueryRowContext(ctx, `
		INSERT INTO message_snippets
			(id, name, body, color, pinned, use_count, last_used_at, sort_order, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 0, '', ?, ?, ?)
		ON CONFLICT (id) DO UPDATE SET
			name = excluded.name,
			body = excluded.body,
			color = excluded.color,
			pinned = `+pinnedExpr+`,
			sort_order = excluded.sort_order,
			updated_at = excluded.updated_at
		RETURNING `+snippetColumns,
		id, req.Name, req.Body, req.Color, req.Pinned, req.SortOrder, now, now,
	)
	snippet, err := scanSnippet(row)
	if err != nil {
		return Snippet{}, fmt.Errorf("upsert snippet: %w", err)
	}
	return snippet, nil
}

func (s *SQLStore) Delete(ctx context.Context, id string) (bool, error) {
	result, err := s.db.ExecContext(ctx, `DELETE FROM message_snippets WHERE id = ?`, id)
	if err != nil {
		return false, fmt.Errorf("delete snippet %s: %w", id, err)
	}
	n, _ := result.RowsAffected()
	return n > 0, nil
}

func (s *SQLStore) Touch(ctx context.Context, id string, now time.Time) (Snippet, error) {
	formatted := FormatTime(now)
	row := s.db.QueryRowContext(ctx, `
		UPDATE message_snippets
		SET use_count = use_count + 1, last_used_at = ?, updated_at = ?
		WHERE id = ?
		RETURNING `+snippetColumns,
		formatted, formatted, id,
	)
	snippet, err := scanSnippet(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Snippet{}, ErrSnippetNotFound
	}
	if err != nil {
		return Snippet{}, fmt.Errorf("touch snippet %s: %w", id, err)
	}
	return snippet, nil
}
