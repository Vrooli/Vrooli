package metrics

import (
	"context"
	"database/sql"
	"time"
)

// WaitlistServicer is the transport-facing contract for waitlist operations.
type WaitlistServicer interface {
	Create(ctx context.Context, email, source string) (*WaitlistEmail, error)
	List(ctx context.Context) ([]WaitlistEmail, error)
	Delete(ctx context.Context, id int64) error
	Count(ctx context.Context) (int64, error)
}

type WaitlistEmail struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	Source    string    `json:"source"`
	CreatedAt time.Time `json:"created_at"`
}

// WaitlistStore is the context-aware persistence contract used by waitlist
// operations. Both *sql.DB and database.RoutedDB implement it; the latter
// selects the lease-owned test pool when the request context is test-mode.
//
// seam: WaitlistStore keeps this domain independent of a concrete database
// handle and preserves request-scoped test isolation.
type WaitlistStore interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

type WaitlistService struct{ db WaitlistStore }

var _ WaitlistServicer = (*WaitlistService)(nil)

func NewWaitlistService(db WaitlistStore) *WaitlistService { return &WaitlistService{db: db} }

func (s *WaitlistService) Create(ctx context.Context, email, source string) (*WaitlistEmail, error) {
	if source == "" {
		source = "coming_soon"
	}
	var entry WaitlistEmail
	err := s.db.QueryRowContext(ctx, `INSERT INTO waitlist_emails (email, source)
		VALUES ($1, $2) ON CONFLICT (email) DO UPDATE SET source = EXCLUDED.source
		RETURNING id, email, source, created_at`, email, source).Scan(&entry.ID, &entry.Email, &entry.Source, &entry.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

func (s *WaitlistService) List(ctx context.Context) ([]WaitlistEmail, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, email, source, created_at FROM waitlist_emails ORDER BY created_at DESC`)
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

func (s *WaitlistService) Delete(ctx context.Context, id int64) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM waitlist_emails WHERE id = $1`, id)
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

func (s *WaitlistService) Count(ctx context.Context) (int64, error) {
	var count int64
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM waitlist_emails`).Scan(&count)
	return count, err
}
