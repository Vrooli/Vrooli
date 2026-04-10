package store

import (
	"database/sql"
	"fmt"

	"tunnel-manager/domain"
)

// RecoveryStore persists recovery events.
type RecoveryStore struct {
	db *sql.DB
}

func NewRecoveryStore(db *sql.DB) *RecoveryStore {
	return &RecoveryStore{db: db}
}

// PersistEvent writes a recovery event to the database.
func (rs *RecoveryStore) PersistEvent(evt *domain.RecoveryEvent) error {
	if rs.db == nil {
		return nil
	}
	var details *string
	if evt.Details != "" {
		details = &evt.Details
	}
	_, err := rs.db.Exec(
		`INSERT INTO recovery_events (trigger_type, action, outcome, details) VALUES ($1, $2, $3, $4)`,
		evt.TriggerType, evt.Action, evt.Outcome, details,
	)
	if err != nil {
		return fmt.Errorf("persist recovery event: %w", err)
	}
	return nil
}

// ListEvents returns recent recovery events from the database.
func (rs *RecoveryStore) ListEvents(limit int) ([]domain.RecoveryEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := rs.db.Query(
		`SELECT id, trigger_type, action, outcome, COALESCE(details, ''), created_at FROM recovery_events ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	defer rows.Close()

	var events []domain.RecoveryEvent
	for rows.Next() {
		var e domain.RecoveryEvent
		if err := rows.Scan(&e.ID, &e.TriggerType, &e.Action, &e.Outcome, &e.Details, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}
