// Package audit records Plan Manager's authoritative, run-scoped mutation
// facts. It never decides ownership or pushes into Swarm Manager; the evidence
// reconciler pulls these immutable facts by verified Agent Manager run id.
package audit

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/api-core/provenance"
)

type Fact struct {
	EventID       string
	RunID         string
	TaskID        string
	Action        string
	PlanID        string
	ContentDigest string
	OccurredAt    time.Time
}

func (s *Store) ListByRun(ctx context.Context, runID string) ([]Fact, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("plan audit store is not configured")
	}
	queryer, ok := s.db.(interface {
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	})
	if !ok {
		return nil, fmt.Errorf("plan audit store does not support queries")
	}
	rows, err := queryer.QueryContext(ctx, `SELECT event_id, run_id, task_id, action, plan_id, content_digest, occurred_at FROM plan_audit_facts WHERE run_id=? ORDER BY occurred_at, event_id`, strings.TrimSpace(runID))
	if err != nil {
		return nil, fmt.Errorf("list plan audit facts: %w", err)
	}
	defer rows.Close()
	facts := []Fact{}
	for rows.Next() {
		var fact Fact
		var occurred string
		if err := rows.Scan(&fact.EventID, &fact.RunID, &fact.TaskID, &fact.Action, &fact.PlanID, &fact.ContentDigest, &occurred); err != nil {
			return nil, fmt.Errorf("scan plan audit fact: %w", err)
		}
		fact.OccurredAt, err = time.Parse(time.RFC3339Nano, occurred)
		if err != nil {
			return nil, fmt.Errorf("parse plan audit fact timestamp: %w", err)
		}
		facts = append(facts, fact)
	}
	return facts, rows.Err()
}

type Recorder interface {
	Record(ctx context.Context, fact Fact) error
}

type Store struct{ db executor }

type executor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func NewStore(db executor) *Store { return &Store{db: db} }

// RecordVerifiedFact derives source identity from the verified provenance in
// ctx. Missing, invalid, or unavailable provenance creates no audit fact.
func (s *Store) RecordVerifiedFact(ctx context.Context, action, planID, digest, eventID string, occurredAt time.Time) error {
	provenance := provenance.FromContext(ctx)
	if !provenance.IsVerifiedAgent() {
		return nil
	}
	return s.Record(ctx, Fact{
		EventID:       eventID,
		RunID:         provenance.RunID,
		TaskID:        provenance.TaskID,
		Action:        action,
		PlanID:        planID,
		ContentDigest: digest,
		OccurredAt:    occurredAt,
	})
}

func (s *Store) Record(ctx context.Context, fact Fact) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("plan audit store is not configured")
	}
	if strings.TrimSpace(fact.EventID) == "" || strings.TrimSpace(fact.RunID) == "" || strings.TrimSpace(fact.Action) == "" || strings.TrimSpace(fact.PlanID) == "" {
		return fmt.Errorf("plan audit event id, run id, action, and plan id are required")
	}
	if fact.OccurredAt.IsZero() {
		fact.OccurredAt = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO plan_audit_facts (event_id, run_id, task_id, action, plan_id, content_digest, occurred_at) VALUES (?, ?, ?, ?, ?, ?, ?) ON CONFLICT(event_id) DO NOTHING`, fact.EventID, fact.RunID, fact.TaskID, fact.Action, fact.PlanID, fact.ContentDigest, fact.OccurredAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("record plan audit fact: %w", err)
	}
	return nil
}
