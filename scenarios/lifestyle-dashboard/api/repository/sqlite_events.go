package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"lifestyle-dashboard/config"
	"lifestyle-dashboard/domain"
)

// SQLiteEventRepository implements EventRepository for SQLite.
type SQLiteEventRepository struct {
	db *sql.DB
}

// NewSQLiteEventRepository creates a new SQLite event repository.
func NewSQLiteEventRepository(db *sql.DB) *SQLiteEventRepository {
	return &SQLiteEventRepository{db: db}
}

// Create persists a new event with generated ID and timestamps.
// [REQ:LD-EVENT-STORAGE] Stores events in SQLite with JSON payload.
func (r *SQLiteEventRepository) Create(ctx context.Context, event *domain.Event) error {
	// Generate ID if not set
	if event.ID == "" {
		event.ID = uuid.New().String()
	}

	// Set timestamps
	now := time.Now().UTC().Format(time.RFC3339)
	event.CreatedAt = now
	if event.Timestamp == "" {
		event.Timestamp = now
	}

	// Ensure payload is valid JSON
	payload := []byte("{}")
	if event.Payload != nil {
		payload = event.Payload
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO events (id, timestamp, domain, event_type, payload, is_intervention, hypothesis_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, event.ID, event.Timestamp, event.Domain, event.EventType, string(payload), event.IsIntervention, event.HypothesisID, event.CreatedAt)

	return err
}

// GetByID retrieves a single event by ID.
// [REQ:LD-EVENT-STORAGE] Fetches event with JSON payload parsing.
func (r *SQLiteEventRepository) GetByID(ctx context.Context, id string) (*domain.Event, error) {
	var e domain.Event
	var payload string
	var hypothesisID sql.NullString

	err := r.db.QueryRowContext(ctx, `
		SELECT id, timestamp, domain, event_type, payload, is_intervention, hypothesis_id, created_at
		FROM events WHERE id = ?
	`, id).Scan(&e.ID, &e.Timestamp, &e.Domain, &e.EventType, &payload, &e.IsIntervention, &hypothesisID, &e.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound{Entity: "event", ID: id}
	}
	if err != nil {
		return nil, err
	}

	e.Payload = json.RawMessage(payload)
	if hypothesisID.Valid {
		e.HypothesisID = &hypothesisID.String
	}

	return &e, nil
}

// List retrieves events matching the given filter.
// [REQ:LD-QUERY-FILTER] Supports filtering by domain, type, and time range.
func (r *SQLiteEventRepository) List(ctx context.Context, filter EventFilter) ([]domain.Event, error) {
	query := `SELECT id, timestamp, domain, event_type, payload, is_intervention, hypothesis_id, created_at FROM events WHERE 1=1`
	args := []interface{}{}

	if filter.Domain != "" {
		query += " AND domain = ?"
		args = append(args, filter.Domain)
	}
	if filter.EventType != "" {
		query += " AND event_type = ?"
		args = append(args, filter.EventType)
	}
	if filter.StartTime != "" {
		query += " AND timestamp >= ?"
		args = append(args, filter.StartTime)
	}
	if filter.EndTime != "" {
		query += " AND timestamp <= ?"
		args = append(args, filter.EndTime)
	}

	query += " ORDER BY timestamp DESC"

	// Apply limit from filter, falling back to config default
	cfg := config.DefaultQueryConfig()
	limit := filter.Limit
	if limit <= 0 {
		limit = cfg.DefaultEventLimit
	}
	if limit > cfg.MaxEventLimit {
		limit = cfg.MaxEventLimit
	}
	query += " LIMIT ?"
	args = append(args, limit)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := []domain.Event{}
	for rows.Next() {
		var e domain.Event
		var payload string
		var hypothesisID sql.NullString
		if err := rows.Scan(&e.ID, &e.Timestamp, &e.Domain, &e.EventType, &payload, &e.IsIntervention, &hypothesisID, &e.CreatedAt); err != nil {
			continue
		}
		e.Payload = json.RawMessage(payload)
		if hypothesisID.Valid {
			e.HypothesisID = &hypothesisID.String
		}
		events = append(events, e)
	}

	return events, rows.Err()
}
