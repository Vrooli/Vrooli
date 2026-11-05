package scenarioruntime

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const demandAuditEventType = "demand_transition"

// DemandAuditRetention bounds recent transition evidence per scenario across
// variants. This is operational history, not an indefinite compliance archive.
const DemandAuditRetention = 1000

// DemandTransition contains identity and outcome, never arbitrary consumer
// metadata. Rows are committed with the state transition they describe.
type DemandTransition struct {
	SchemaVersion int        `json:"schema_version"`
	EventID       string     `json:"event_id"`
	RecordedAt    time.Time  `json:"recorded_at"`
	Operation     string     `json:"operation"`
	Scenario      string     `json:"scenario"`
	Variant       string     `json:"variant"`
	InstanceID    string     `json:"instance_id,omitempty"`
	Generation    int64      `json:"generation,omitempty"`
	LeaseID       string     `json:"lease_id,omitempty"`
	ConsumerID    string     `json:"consumer_id,omitempty"`
	Kind          string     `json:"kind,omitempty"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
}

func (s *SQLiteStore) auditDemandTx(ctx context.Context, tx *sql.Tx, event DemandTransition) error {
	event.SchemaVersion = 1
	event.EventID, event.RecordedAt = newID("evt"), s.now()
	details, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if len(details) > 8*1024 {
		return fmt.Errorf("demand transition exceeds 8 KiB")
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO runtime_events
(event_id, instance_id, scenario, event_type, created_at, details_json) VALUES (?, ?, ?, ?, ?, ?)`,
		event.EventID, nullableString(event.InstanceID), event.Scenario, demandAuditEventType, formatTime(event.RecordedAt), string(details)); err != nil {
		return fmt.Errorf("record demand transition: %w", err)
	}
	// Insertion order avoids losing ordering when clocks move backwards or
	// several transitions share a timestamp. Other event families are untouched.
	_, err = tx.ExecContext(ctx, `DELETE FROM runtime_events
WHERE scenario = ? AND event_type = ? AND rowid IN (
 SELECT rowid FROM runtime_events WHERE scenario = ? AND event_type = ?
 ORDER BY rowid DESC LIMIT -1 OFFSET ?)`, event.Scenario, demandAuditEventType,
		event.Scenario, demandAuditEventType, DemandAuditRetention)
	return err
}

func (s *SQLiteStore) auditDemandLeaseTx(ctx context.Context, tx *sql.Tx, operation string, lease DemandLease) error {
	return s.auditDemandTx(ctx, tx, DemandTransition{Operation: operation, Scenario: lease.Scenario,
		Variant: lease.Variant, LeaseID: lease.LeaseID, ConsumerID: lease.ConsumerID, Kind: lease.Kind, ExpiresAt: &lease.ExpiresAt})
}

func (s *SQLiteStore) auditDemandInstanceTx(ctx context.Context, tx *sql.Tx, operation string, instance Instance) error {
	return s.auditDemandTx(ctx, tx, DemandTransition{Operation: operation, Scenario: instance.Scenario,
		Variant: instance.Variant, InstanceID: instance.InstanceID, Generation: instance.Generation})
}

// ListDemandTransitions returns newest-first recent committed transitions for
// one exact scenario/variant. An empty result does not prove no historic use:
// retention is bounded and installations predating auditing have no rows.
func (s *SQLiteStore) ListDemandTransitions(ctx context.Context, scenario, variant string, limit int) ([]DemandTransition, error) {
	scenario = strings.TrimSpace(scenario)
	variant = InstanceKey{Scenario: scenario, Variant: variant}.Normalize().Variant
	if scenario == "" {
		return nil, fmt.Errorf("demand transition history requires a scenario")
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > DemandAuditRetention {
		limit = DemandAuditRetention
	}
	rows, err := s.db.QueryContext(ctx, `SELECT details_json FROM runtime_events
WHERE scenario = ? AND event_type = ? AND json_extract(details_json, '$.variant') = ?
ORDER BY rowid DESC LIMIT ?`, scenario, demandAuditEventType, variant, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]DemandTransition, 0)
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var event DemandTransition
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			return nil, fmt.Errorf("decode demand transition: %w", err)
		}
		out = append(out, event)
	}
	return out, rows.Err()
}
