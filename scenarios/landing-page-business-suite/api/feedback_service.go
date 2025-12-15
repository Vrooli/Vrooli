package main

import (
	"database/sql"
	"strconv"
	"strings"
	"time"
)

// FeedbackRequest represents a user feedback submission
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

// FeedbackService handles feedback operations
type FeedbackService struct {
	db *sql.DB
}

// NewFeedbackService creates a new feedback service
func NewFeedbackService(db *sql.DB) *FeedbackService {
	return &FeedbackService{db: db}
}

// CreateFeedbackInput represents the input for creating feedback
type CreateFeedbackInput struct {
	Type    string  `json:"type"`
	Email   string  `json:"email"`
	Subject string  `json:"subject"`
	Message string  `json:"message"`
	OrderID *string `json:"order_id,omitempty"`
}

// Create stores a new feedback request
func (s *FeedbackService) Create(input *CreateFeedbackInput) (*FeedbackRequest, error) {
	query := `
		INSERT INTO feedback_requests (type, email, subject, message, order_id, status)
		VALUES ($1, $2, $3, $4, $5, 'pending')
		RETURNING id, type, email, subject, message, order_id, status, created_at, updated_at
	`

	var f FeedbackRequest
	err := s.db.QueryRow(query,
		input.Type, input.Email, input.Subject, input.Message, input.OrderID,
	).Scan(
		&f.ID, &f.Type, &f.Email, &f.Subject, &f.Message, &f.OrderID, &f.Status, &f.CreatedAt, &f.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &f, nil
}

// List retrieves all feedback requests with optional status filter
func (s *FeedbackService) List(status string) ([]FeedbackRequest, error) {
	var query string
	var args []interface{}

	if status != "" {
		query = `
			SELECT id, type, email, subject, message, order_id, status, created_at, updated_at
			FROM feedback_requests
			WHERE status = $1
			ORDER BY created_at DESC
		`
		args = append(args, status)
	} else {
		query = `
			SELECT id, type, email, subject, message, order_id, status, created_at, updated_at
			FROM feedback_requests
			ORDER BY created_at DESC
		`
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []FeedbackRequest
	for rows.Next() {
		var f FeedbackRequest
		if err := rows.Scan(
			&f.ID, &f.Type, &f.Email, &f.Subject, &f.Message, &f.OrderID, &f.Status, &f.CreatedAt, &f.UpdatedAt,
		); err != nil {
			return nil, err
		}
		requests = append(requests, f)
	}

	return requests, rows.Err()
}

// UpdateStatus changes the status of a feedback request
func (s *FeedbackService) UpdateStatus(id int, status string) (*FeedbackRequest, error) {
	query := `
		UPDATE feedback_requests
		SET status = $1, updated_at = NOW()
		WHERE id = $2
		RETURNING id, type, email, subject, message, order_id, status, created_at, updated_at
	`

	var f FeedbackRequest
	err := s.db.QueryRow(query, status, id).Scan(
		&f.ID, &f.Type, &f.Email, &f.Subject, &f.Message, &f.OrderID, &f.Status, &f.CreatedAt, &f.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &f, nil
}

// GetByID retrieves a single feedback request by ID
func (s *FeedbackService) GetByID(id int) (*FeedbackRequest, error) {
	query := `
		SELECT id, type, email, subject, message, order_id, status, created_at, updated_at
		FROM feedback_requests
		WHERE id = $1
	`

	var f FeedbackRequest
	err := s.db.QueryRow(query, id).Scan(
		&f.ID, &f.Type, &f.Email, &f.Subject, &f.Message, &f.OrderID, &f.Status, &f.CreatedAt, &f.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &f, nil
}

// Delete removes a feedback request by ID
func (s *FeedbackService) Delete(id int) error {
	query := `DELETE FROM feedback_requests WHERE id = $1`
	result, err := s.db.Exec(query, id)
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

// DeleteBulk removes multiple feedback requests by IDs
func (s *FeedbackService) DeleteBulk(ids []int) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	// Build placeholders for IN clause
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "$" + strconv.Itoa(i+1)
		args[i] = id
	}

	query := "DELETE FROM feedback_requests WHERE id IN (" + strings.Join(placeholders, ",") + ")"
	result, err := s.db.Exec(query, args...)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}
