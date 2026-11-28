package metrics

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"time"
)

type FeedbackServicer interface {
	Create(context.Context, *CreateFeedbackInput) (*FeedbackRequest, error)
	List(context.Context, string) ([]FeedbackRequest, error)
	GetByID(context.Context, int) (*FeedbackRequest, error)
	UpdateStatus(context.Context, int, string) (*FeedbackRequest, error)
	Delete(context.Context, int) error
	DeleteBulk(context.Context, []int) (int64, error)
}

// FeedbackStore supports request-scoped routed database access. Both *sql.DB
// and database.RoutedDB implement it; the latter selects Test Genie's
// lease-owned pool when a request has test-mode context.
//
// seam: FeedbackStore keeps feedback persistence independent of a concrete
// database handle and preserves request-scoped test isolation.
type FeedbackStore interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

type FeedbackRequest struct {
	ID        int       `json:"id"`
	Type      string    `json:"type"`
	Email     string    `json:"email"`
	Subject   string    `json:"subject"`
	Message   string    `json:"message"`
	OrderID   *string   `json:"order_id,omitempty"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateFeedbackInput struct {
	Type    string  `json:"type"`
	Email   string  `json:"email"`
	Subject string  `json:"subject"`
	Message string  `json:"message"`
	OrderID *string `json:"order_id,omitempty"`
}

type FeedbackService struct{ db FeedbackStore }

var _ FeedbackServicer = (*FeedbackService)(nil)

func NewFeedbackService(db FeedbackStore) *FeedbackService { return &FeedbackService{db: db} }

func (s *FeedbackService) Create(ctx context.Context, input *CreateFeedbackInput) (*FeedbackRequest, error) {
	var feedback FeedbackRequest
	err := s.db.QueryRowContext(ctx, `INSERT INTO feedback_requests (type, email, subject, message, order_id, status)
		VALUES ($1, $2, $3, $4, $5, 'pending')
		RETURNING id, type, email, subject, message, order_id, status, created_at, updated_at`,
		input.Type, input.Email, input.Subject, input.Message, input.OrderID,
	).Scan(&feedback.ID, &feedback.Type, &feedback.Email, &feedback.Subject, &feedback.Message, &feedback.OrderID, &feedback.Status, &feedback.CreatedAt, &feedback.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &feedback, nil
}

func (s *FeedbackService) List(ctx context.Context, status string) ([]FeedbackRequest, error) {
	query := `SELECT id, type, email, subject, message, order_id, status, created_at, updated_at FROM feedback_requests`
	var args []interface{}
	if status != "" {
		query += " WHERE status = $1"
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC"
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var requests []FeedbackRequest
	for rows.Next() {
		var feedback FeedbackRequest
		if err := rows.Scan(&feedback.ID, &feedback.Type, &feedback.Email, &feedback.Subject, &feedback.Message, &feedback.OrderID, &feedback.Status, &feedback.CreatedAt, &feedback.UpdatedAt); err != nil {
			return nil, err
		}
		requests = append(requests, feedback)
	}
	return requests, rows.Err()
}

func (s *FeedbackService) UpdateStatus(ctx context.Context, id int, status string) (*FeedbackRequest, error) {
	var feedback FeedbackRequest
	err := s.db.QueryRowContext(ctx, `UPDATE feedback_requests SET status = $1, updated_at = NOW() WHERE id = $2
		RETURNING id, type, email, subject, message, order_id, status, created_at, updated_at`, status, id,
	).Scan(&feedback.ID, &feedback.Type, &feedback.Email, &feedback.Subject, &feedback.Message, &feedback.OrderID, &feedback.Status, &feedback.CreatedAt, &feedback.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &feedback, nil
}

func (s *FeedbackService) GetByID(ctx context.Context, id int) (*FeedbackRequest, error) {
	var feedback FeedbackRequest
	err := s.db.QueryRowContext(ctx, `SELECT id, type, email, subject, message, order_id, status, created_at, updated_at FROM feedback_requests WHERE id = $1`, id).Scan(&feedback.ID, &feedback.Type, &feedback.Email, &feedback.Subject, &feedback.Message, &feedback.OrderID, &feedback.Status, &feedback.CreatedAt, &feedback.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &feedback, nil
}

func (s *FeedbackService) Delete(ctx context.Context, id int) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM feedback_requests WHERE id = $1`, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *FeedbackService) DeleteBulk(ctx context.Context, ids []int) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	result, err := s.db.ExecContext(ctx, `DELETE FROM feedback_requests WHERE id = ANY($1::bigint[])`, postgresIntArray(ids))
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// postgresIntArray serializes already-validated integer IDs for a single
// parameterized PostgreSQL array value. No caller-controlled SQL is composed.
func postgresIntArray(ids []int) string {
	values := make([]string, len(ids))
	for i, id := range ids {
		values[i] = strconv.Itoa(id)
	}
	return "{" + strings.Join(values, ",") + "}"
}
