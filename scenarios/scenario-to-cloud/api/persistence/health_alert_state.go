package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

// HealthAlertStateStore persists the last acknowledged open alert set. The
// event publisher remains the Vrooli Events/Notification Hub path; this store
// only preserves transition context across API restarts.
type HealthAlertStateStore struct{ db DB }

func NewHealthAlertStateStore(db DB) *HealthAlertStateStore {
	return &HealthAlertStateStore{db: db}
}

func (s *HealthAlertStateStore) Load(ctx context.Context, deploymentID string) (json.RawMessage, error) {
	var raw []byte
	err := s.db.QueryRowContext(ctx, `SELECT open_alerts FROM deployment_health_alert_state WHERE deployment_id = $1`, deploymentID).Scan(&raw)
	if err == sql.ErrNoRows {
		return json.RawMessage("[]"), nil
	}
	if err != nil {
		return nil, fmt.Errorf("load open deployment alerts: %w", err)
	}
	if !json.Valid(raw) {
		return nil, fmt.Errorf("decode open deployment alerts: invalid JSON")
	}
	return json.RawMessage(raw), nil
}

func (s *HealthAlertStateStore) Save(ctx context.Context, deploymentID string, raw json.RawMessage) error {
	if !json.Valid(raw) {
		return fmt.Errorf("encode open deployment alerts: invalid JSON")
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO deployment_health_alert_state (deployment_id, open_alerts, updated_at) VALUES ($1, $2, CURRENT_TIMESTAMP) ON CONFLICT (deployment_id) DO UPDATE SET open_alerts = EXCLUDED.open_alerts, updated_at = CURRENT_TIMESTAMP`, deploymentID, string(raw))
	if err != nil {
		return fmt.Errorf("save open deployment alerts: %w", err)
	}
	return nil
}
