package orchestrator

// DOC: docs/reference/providers.md#policy-profiles

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

//go:embed schema.sql
var schemaSQL string

// Schema returns the tables this package owns. Registered in
// modules.AllSchemas. It lives beside the code that reads and writes these
// tables so a domain stays a one-folder unit.
func Schema() string { return schemaSQL }

// DB is the subset of a database handle this store needs.
//
// Declaring the dependency as an interface rather than a *sql.DB is what keeps
// the scenario eligible for per-request routing: production passes
// *database.RoutedDB, which routes each request to the right pool, while tests
// pass a plain *sql.DB. Capturing the concrete pool type here would pin every
// caller to a single pool.
type DB interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// SQLiteStore persists the operator's durable decisions and keeps transient
// request state in memory.
//
// The split is deliberate rather than an unfinished migration:
//
//   - The active POLICY is an operator decision. It must survive a restart.
//     While it lived only in memory, every profile change silently reverted to
//     the shipped default on the next restart — indistinguishable from cleanup
//     never having been enabled at all, which is the exact class of bug that
//     let a disk fill while three safeguards reported healthy.
//   - The AUDIT trail is evidence. A record of what was deleted is worthless if
//     it disappears with the process that deleted it.
//   - PLANS and APPLY reports are transient. A plan is a measurement of the
//     filesystem at one instant, consumed within seconds by the apply that
//     follows it, and meaningless after a restart because the filesystem has
//     moved on. They are also large: one plan on this project's development
//     host serialised to 6.9 MB. Persisting those in a scenario whose purpose
//     is reclaiming disk space would be a self-defeating design.
type SQLiteStore struct {
	db DB

	// Transient state, same semantics as MemoryStore.
	mu      sync.Mutex
	plans   map[string]Plan
	applies map[string]ApplyReport
}

// NewSQLiteStore builds a store backed by db for policy and audit.
func NewSQLiteStore(db DB) *SQLiteStore {
	return &SQLiteStore{
		db:      db,
		plans:   map[string]Plan{},
		applies: map[string]ApplyReport{},
	}
}

// Compile-time guarantee that SQLiteStore satisfies the orchestrator contract.
var _ Store = (*SQLiteStore)(nil)

// SavePolicy replaces the single active policy row.
//
// There is exactly one active policy, so the row is keyed by a constant. Using
// a fixed key rather than appending versions means a reader can never pick the
// wrong one, and the policy's own Version field still records which profile
// generation is live.
func (s *SQLiteStore) SavePolicy(ctx context.Context, policy Policy) error {
	payload, err := json.Marshal(policy)
	if err != nil {
		return fmt.Errorf("marshal cleanup policy: %w", err)
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO cleanup_policy (id, payload, updated_at) VALUES (1, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET payload = excluded.payload, updated_at = excluded.updated_at`,
		string(payload), time.Now().UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("save cleanup policy: %w", err)
	}
	return nil
}

// CurrentPolicy returns the stored policy, reporting false when none has been
// saved yet so the caller applies its default profile.
func (s *SQLiteStore) CurrentPolicy(ctx context.Context) (Policy, bool, error) {
	var payload string
	err := s.db.QueryRowContext(ctx, `SELECT payload FROM cleanup_policy WHERE id = 1`).Scan(&payload)
	if err == sql.ErrNoRows {
		return Policy{}, false, nil
	}
	if err != nil {
		return Policy{}, false, fmt.Errorf("read cleanup policy: %w", err)
	}

	var policy Policy
	if err := json.Unmarshal([]byte(payload), &policy); err != nil {
		// A policy row that cannot be decoded must not be treated as "no
		// policy": silently falling back to the default would quietly change
		// what gets deleted. Report it and let the caller decide.
		return Policy{}, false, fmt.Errorf("decode stored cleanup policy: %w", err)
	}
	return policy, true, nil
}

func (s *SQLiteStore) SavePlan(_ context.Context, plan Plan) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.plans[plan.ID] = plan
	return nil
}

func (s *SQLiteStore) GetPlan(_ context.Context, id string) (Plan, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	plan, ok := s.plans[id]
	return plan, ok, nil
}

func (s *SQLiteStore) SaveApply(_ context.Context, report ApplyReport) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.applies[report.IdempotencyKey] = report
	return nil
}

func (s *SQLiteStore) ApplyByKey(_ context.Context, key string) (ApplyReport, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	report, ok := s.applies[key]
	return report, ok, nil
}

// AddAudit appends an audit event.
func (s *SQLiteStore) AddAudit(ctx context.Context, event AuditEvent) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO cleanup_audit (id, occurred_at, type, plan_id, provider_id, idempotency_key, message, redacted)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO NOTHING`,
		event.ID, event.Time.UTC().Format(time.RFC3339Nano), event.Type,
		event.PlanID, event.ProviderID, event.IdempotencyKey, event.Message, event.Redacted,
	)
	if err != nil {
		return fmt.Errorf("append cleanup audit event: %w", err)
	}
	return nil
}

// ListAudit returns audit events oldest first, matching MemoryStore's order.
func (s *SQLiteStore) ListAudit(ctx context.Context) ([]AuditEvent, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, occurred_at, type, plan_id, provider_id, idempotency_key, message, redacted
		 FROM cleanup_audit ORDER BY occurred_at, id`)
	if err != nil {
		return nil, fmt.Errorf("list cleanup audit events: %w", err)
	}
	defer rows.Close()

	var out []AuditEvent
	for rows.Next() {
		var (
			event    AuditEvent
			occurred string
			redacted int
			planID   sql.NullString
			provider sql.NullString
			idemKey  sql.NullString
			message  sql.NullString
		)
		if err := rows.Scan(&event.ID, &occurred, &event.Type, &planID, &provider, &idemKey, &message, &redacted); err != nil {
			return nil, fmt.Errorf("scan cleanup audit event: %w", err)
		}
		event.Time, _ = time.Parse(time.RFC3339Nano, occurred)
		event.PlanID = planID.String
		event.ProviderID = provider.String
		event.IdempotencyKey = idemKey.String
		event.Message = message.String
		event.Redacted = redacted != 0
		out = append(out, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate cleanup audit events: %w", err)
	}
	return out, nil
}
