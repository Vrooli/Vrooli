package main

import (
	"context"
	"database/sql"
	"time"
)

// WaitlistServicer defines the interface for waitlist operations.
// This interface allows for easy mocking in tests.
type WaitlistServicer interface {
	Create(ctx context.Context, email, source string) (*WaitlistEmail, error)
	List(ctx context.Context) ([]WaitlistEmail, error)
	Delete(ctx context.Context, id int64) error
	Count(ctx context.Context) (int64, error)
}

// Compile-time check that WaitlistService implements WaitlistServicer
var _ WaitlistServicer = (*WaitlistService)(nil)

// WaitlistEmail represents an email collected from the waitlist/coming soon form
type WaitlistEmail struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	Source    string    `json:"source"`
	CreatedAt time.Time `json:"created_at"`
}

// WaitlistService handles waitlist email operations
type WaitlistService struct {
	db *sql.DB
}

// NewWaitlistService creates a new WaitlistService
func NewWaitlistService(db *sql.DB) *WaitlistService {
	return &WaitlistService{db: db}
}

// Create adds a new email to the waitlist
func (s *WaitlistService) Create(ctx context.Context, email, source string) (*WaitlistEmail, error) {
	if source == "" {
		source = "coming_soon"
	}

	var entry WaitlistEmail
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO waitlist_emails (email, source)
		 VALUES ($1, $2)
		 ON CONFLICT (email) DO UPDATE SET source = EXCLUDED.source
		 RETURNING id, email, source, created_at`,
		email, source,
	).Scan(&entry.ID, &entry.Email, &entry.Source, &entry.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

// List returns all waitlist emails ordered by created_at desc
func (s *WaitlistService) List(ctx context.Context) ([]WaitlistEmail, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, email, source, created_at
		 FROM waitlist_emails
		 ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var emails []WaitlistEmail
	for rows.Next() {
		var entry WaitlistEmail
		if err := rows.Scan(&entry.ID, &entry.Email, &entry.Source, &entry.CreatedAt); err != nil {
			return nil, err
		}
		emails = append(emails, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return emails, nil
}

// Delete removes an email from the waitlist by ID
func (s *WaitlistService) Delete(ctx context.Context, id int64) error {
	result, err := s.db.ExecContext(ctx,
		`DELETE FROM waitlist_emails WHERE id = $1`,
		id,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// Count returns the total number of waitlist emails
func (s *WaitlistService) Count(ctx context.Context) (int64, error) {
	var count int64
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM waitlist_emails`,
	).Scan(&count)
	return count, err
}
