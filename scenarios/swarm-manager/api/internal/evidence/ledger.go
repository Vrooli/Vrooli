// Package evidence persists owner-neutral workflow observations in the shared
// event database. Attempt files remain readable recovery inputs; this ledger
// is the queryable authority for producer and trust facts.
package evidence

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/api-core/database"
)

type Observation struct {
	ID, Producer, SourceSystem, RunID string
	SubjectKind, SubjectID, Action    string
	Confidence, Title, Description    string
	Actor, Reason                     string
	ObservedAt                        time.Time
}

// MigrationAudit records a bounded file-to-ledger import parity result.
// A projection must not become authoritative until parity is explicitly true.
type MigrationAudit struct {
	Key, SourceDigest, ProjectionDigest string
	SourceCount, ProjectionCount        int
	AuditedAt                           time.Time
}

type Ledger struct{ db *database.RoutedDB }

func NewLedger(db *database.RoutedDB) *Ledger { return &Ledger{db: db} }

func (l *Ledger) Record(ctx context.Context, observation Observation, attemptRef string) error {
	if l == nil || l.db == nil {
		return nil
	}
	for _, value := range []string{observation.ID, observation.Producer, observation.SourceSystem, observation.RunID, observation.SubjectKind, observation.SubjectID, observation.Action, observation.Confidence, observation.Title, attemptRef} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("evidence observation requires complete identity")
		}
	}
	if observation.ObservedAt.IsZero() {
		observation.ObservedAt = time.Now().UTC()
	}
	if observation.Confidence == "operator_verified" && (strings.TrimSpace(observation.Actor) == "" || strings.TrimSpace(observation.Reason) == "") {
		return fmt.Errorf("operator_verified evidence requires actor and reason")
	}
	if _, err := l.db.ExecContext(ctx, `INSERT OR IGNORE INTO evidence_observations (id, producer, source_system, run_id, subject_kind, subject_id, action, confidence, title, description, actor, reason, observed_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, observation.ID, observation.Producer, observation.SourceSystem, observation.RunID, observation.SubjectKind, observation.SubjectID, observation.Action, observation.Confidence, observation.Title, observation.Description, observation.Actor, observation.Reason, observation.ObservedAt.Format(time.RFC3339Nano)); err != nil {
		return fmt.Errorf("insert evidence observation: %w", err)
	}
	if _, err := l.db.ExecContext(ctx, `INSERT OR IGNORE INTO evidence_links (observation_id, attempt_ref) VALUES (?, ?)`, observation.ID, attemptRef); err != nil {
		return fmt.Errorf("link evidence observation: %w", err)
	}
	if _, err := l.db.ExecContext(ctx, `INSERT OR REPLACE INTO evidence_checkpoints (producer, run_id, fact_kind, checkpoint_at) VALUES (?, ?, ?, ?)`, observation.Producer, observation.RunID, "review_evidence", observation.ObservedAt.Format(time.RFC3339Nano)); err != nil {
		return fmt.Errorf("write evidence checkpoint: %w", err)
	}
	_, err := l.db.ExecContext(ctx, `INSERT OR REPLACE INTO evidence_watermarks (producer, run_id, fact_kind, terminal_at) VALUES (?, ?, ?, ?)`, observation.Producer, observation.RunID, "review_evidence", observation.ObservedAt.Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("write evidence watermark: %w", err)
	}
	return nil
}

// RecordMigrationAudit persists a source/projection parity decision for a
// one-shot importer. It is idempotent for identical audit data.
func (l *Ledger) RecordMigrationAudit(ctx context.Context, audit MigrationAudit) error {
	if l == nil || l.db == nil {
		return nil
	}
	if strings.TrimSpace(audit.Key) == "" || strings.TrimSpace(audit.SourceDigest) == "" || strings.TrimSpace(audit.ProjectionDigest) == "" || audit.SourceCount < 0 || audit.ProjectionCount < 0 {
		return fmt.Errorf("evidence migration audit requires identity, digests, and non-negative counts")
	}
	if audit.AuditedAt.IsZero() {
		audit.AuditedAt = time.Now().UTC()
	}
	parity := audit.SourceCount == audit.ProjectionCount
	_, err := l.db.ExecContext(ctx, `INSERT OR REPLACE INTO evidence_migration_audits (migration_key, source_digest, projection_digest, source_count, projection_count, parity_proven, audited_at) VALUES (?, ?, ?, ?, ?, ?, ?)`, audit.Key, audit.SourceDigest, audit.ProjectionDigest, audit.SourceCount, audit.ProjectionCount, parity, audit.AuditedAt.Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("write evidence migration audit: %w", err)
	}
	return nil
}

// ParityProven reports whether the most recent durable audit permits a ledger
// projection to replace its legacy recovery input.
func (l *Ledger) ParityProven(ctx context.Context, key string) (bool, error) {
	if l == nil || l.db == nil {
		return false, nil
	}
	var parity bool
	err := l.db.QueryRowContext(ctx, `SELECT parity_proven FROM evidence_migration_audits WHERE migration_key = ?`, strings.TrimSpace(key)).Scan(&parity)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read evidence migration audit: %w", err)
	}
	return parity, nil
}

// OperatorVerifiedEvidenceIDs returns the current verification projection from
// append-only operator events linked to one attempt. Consumers must gate this
// projection on a successful migration parity audit before replacing a legacy
// file field.
func (l *Ledger) OperatorVerifiedEvidenceIDs(ctx context.Context, attemptRef string) (map[string]bool, error) {
	result := map[string]bool{}
	if l == nil || l.db == nil {
		return result, nil
	}
	rows, err := l.db.QueryContext(ctx, `SELECT observation_id, action FROM evidence_links JOIN evidence_observations ON evidence_observations.id = evidence_links.observation_id WHERE attempt_ref = ? AND confidence = 'operator_verified' AND action IN ('operator_verified', 'operator_unverified') ORDER BY observed_at, observation_id`, strings.TrimSpace(attemptRef))
	if err != nil {
		return nil, fmt.Errorf("query operator verification observations: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id, action string
		if err := rows.Scan(&id, &action); err != nil {
			return nil, err
		}
		if evidenceID, ok := verificationEvidenceID(strings.TrimSpace(attemptRef), id); ok {
			result[evidenceID] = action == "operator_verified"
		}
	}
	return result, rows.Err()
}

func verificationEvidenceID(attemptRef, observationID string) (string, bool) {
	// Legacy imports used the evidence id as an independent observation id;
	// retain that readable projection until their bounded migration has parity.
	if strings.HasSuffix(observationID, "/operator_verified") {
		return strings.TrimSuffix(observationID, "/operator_verified"), true
	}
	prefix := strings.TrimSpace(attemptRef) + "/"
	if !strings.HasPrefix(observationID, prefix) {
		return "", false
	}
	rest := strings.TrimPrefix(observationID, prefix)
	marker := "/operator-verification/"
	index := strings.LastIndex(rest, marker)
	if index <= 0 || strings.TrimSpace(rest[index+len(marker):]) == "" {
		return "", false
	}
	return rest[:index], true
}
