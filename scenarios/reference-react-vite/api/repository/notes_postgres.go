package repository

import (
	"context"
	"database/sql"
	"fmt"

	"reference-react-vite/api/domain/notes"
)

// PostgresNoteRepository implements NoteRepository using PostgreSQL.
type PostgresNoteRepository struct {
	db *sql.DB
}

// NewPostgresNoteRepository creates a new PostgreSQL note repository.
func NewPostgresNoteRepository(db *sql.DB) *PostgresNoteRepository {
	return &PostgresNoteRepository{db: db}
}

// Create inserts a new note into the database.
func (r *PostgresNoteRepository) Create(ctx context.Context, note *notes.Note) error {
	query := `
		INSERT INTO notes (id, task_id, content, author, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	var author *string
	if note.Author != "" {
		author = &note.Author
	}
	_, err := r.db.ExecContext(ctx, query,
		note.ID, note.TaskID, note.Content, author,
		note.CreatedAt, note.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create note: %w", err)
	}
	return nil
}

// FindByID retrieves a note by its ID.
func (r *PostgresNoteRepository) FindByID(ctx context.Context, id string) (*notes.Note, error) {
	query := `
		SELECT id, task_id, content, author, created_at, updated_at
		FROM notes
		WHERE id = $1
	`
	note := &notes.Note{}
	var author sql.NullString
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&note.ID, &note.TaskID, &note.Content, &author,
		&note.CreatedAt, &note.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find note by id: %w", err)
	}
	if author.Valid {
		note.Author = author.String
	}
	return note, nil
}

// ListByTask retrieves notes for a specific task with pagination.
func (r *PostgresNoteRepository) ListByTask(ctx context.Context, filter notes.ListFilter) ([]*notes.Note, int, error) {
	// Count total matching records
	var total int
	if err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM notes WHERE task_id = $1",
		filter.TaskID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count notes: %w", err)
	}

	// Apply pagination defaults
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT id, task_id, content, author, created_at, updated_at
		FROM notes
		WHERE task_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.QueryContext(ctx, query, filter.TaskID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list notes: %w", err)
	}
	defer rows.Close()

	var result []*notes.Note
	for rows.Next() {
		note := &notes.Note{}
		var author sql.NullString
		if err := rows.Scan(
			&note.ID, &note.TaskID, &note.Content, &author,
			&note.CreatedAt, &note.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan note: %w", err)
		}
		if author.Valid {
			note.Author = author.String
		}
		result = append(result, note)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate notes: %w", err)
	}

	return result, total, nil
}

// Update saves changes to an existing note.
func (r *PostgresNoteRepository) Update(ctx context.Context, note *notes.Note) error {
	query := `
		UPDATE notes
		SET content = $2, updated_at = $3
		WHERE id = $1
	`
	result, err := r.db.ExecContext(ctx, query,
		note.ID, note.Content, note.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update note: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check update result: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("note not found")
	}
	return nil
}

// Delete removes a note by ID.
func (r *PostgresNoteRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM notes WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete note: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check delete result: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("note not found")
	}
	return nil
}
