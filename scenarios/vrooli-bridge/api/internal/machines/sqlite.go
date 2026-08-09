package machines

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"vrooli-bridge/internal/clock"
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// transactionStarter is satisfied by both *sql.DB and api-core's RoutedDB.
// Linking a Node changes two rows and must never expose a Machine with no
// current lineage entry if the second write fails.
type transactionStarter interface {
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}
type sqliteRepository struct {
	db    SQLExecutor
	clock clock.Clock
}

func NewSQLiteRepository(db SQLExecutor, c clock.Clock) Repository { return &sqliteRepository{db, c} }

const machineTimeFormat = time.RFC3339Nano

func (s *sqliteRepository) Create(ctx context.Context, in CreateInput) (Machine, error) {
	if len(in.Locators) == 0 {
		return Machine{}, ErrInvalid{"locators", "at least one required"}
	}
	// Every Machine starts from an explicit, least-privilege policy rather than
	// carrying an empty implicit default into readiness evaluation.
	if in.DesiredProfileID == "" {
		in.DesiredProfileID = "managed-connection"
	}
	if in.DesiredProfileVersion == "" {
		in.DesiredProfileVersion = "v1"
	}
	type preparedLocator struct {
		Locator
		normalized string
	}
	locators := make([]preparedLocator, 0, len(in.Locators))
	for i, l := range in.Locators {
		normalized, e := normalizeLocator(l.Kind, l.Value)
		if e != nil {
			return Machine{}, e
		}
		if l.Ordinal == 0 && i > 0 {
			l.Ordinal = i
		}
		locators = append(locators, preparedLocator{Locator: l, normalized: normalized})
	}
	starter, ok := s.db.(transactionStarter)
	if !ok {
		return Machine{}, fmt.Errorf("machine creation requires transactional database")
	}
	id := in.ID
	if id == "" {
		id = uuid.NewString()
	}
	now := s.clock.Now().UTC()
	tx, e := starter.BeginTx(ctx, nil)
	if e != nil {
		return Machine{}, fmt.Errorf("begin machine creation transaction: %w", e)
	}
	defer tx.Rollback()
	if _, e := tx.ExecContext(ctx, "INSERT INTO machines (id,lifecycle,version,desired_profile_id,desired_profile_version,trust_ref,created_at,updated_at) VALUES (?, 'active',1,?,?,?,?,?)", id, in.DesiredProfileID, in.DesiredProfileVersion, in.TrustRef, now.Format(machineTimeFormat), now.Format(machineTimeFormat)); e != nil {
		return Machine{}, fmt.Errorf("insert machine: %w", e)
	}
	for _, l := range locators {
		if _, e = tx.ExecContext(ctx, "INSERT INTO machine_locators (machine_id,ordinal,kind,value,normalized_value,created_at) VALUES (?,?,?,?,?,?)", id, l.Ordinal, l.Kind, l.Value, l.normalized, now.Format(machineTimeFormat)); e != nil {
			return Machine{}, fmt.Errorf("insert locator: %w", e)
		}
	}
	if e = tx.Commit(); e != nil {
		return Machine{}, fmt.Errorf("commit machine creation: %w", e)
	}
	return s.Get(ctx, id)
}

func (s *sqliteRepository) Get(ctx context.Context, id string) (Machine, error) {
	var m Machine
	var created, updated, arch, removed string
	e := s.db.QueryRowContext(ctx, "SELECT id,lifecycle,version,desired_profile_id,desired_profile_version,trust_ref,created_at,updated_at,archived_at,removed_at FROM machines WHERE id=?", id).Scan(&m.ID, &m.Lifecycle, &m.Version, &m.DesiredProfileID, &m.DesiredProfileVersion, &m.TrustRef, &created, &updated, &arch, &removed)
	if errors.Is(e, sql.ErrNoRows) {
		return Machine{}, ErrNotFound{id}
	}
	if e != nil {
		return Machine{}, e
	}
	var er error
	if m.CreatedAt, er = time.Parse(machineTimeFormat, created); er != nil {
		return Machine{}, er
	}
	if m.UpdatedAt, er = time.Parse(machineTimeFormat, updated); er != nil {
		return Machine{}, er
	}
	if m.ArchivedAt, er = nullableTime(arch); er != nil {
		return Machine{}, er
	}
	if m.RemovedAt, er = nullableTime(removed); er != nil {
		return Machine{}, er
	}
	rows, e := s.db.QueryContext(ctx, "SELECT kind,value,ordinal FROM machine_locators WHERE machine_id=? ORDER BY ordinal", id)
	if e != nil {
		return Machine{}, e
	}
	defer rows.Close()
	for rows.Next() {
		var l Locator
		if e = rows.Scan(&l.Kind, &l.Value, &l.Ordinal); e != nil {
			return Machine{}, e
		}
		m.Locators = append(m.Locators, l)
	}
	if e = rows.Err(); e != nil {
		return Machine{}, e
	}
	lines, e := s.db.QueryContext(ctx, "SELECT node_id,is_current,linked_at,superseded_at,source_correlation_id FROM machine_node_lineage WHERE machine_id=? ORDER BY linked_at", id)
	if e != nil {
		return Machine{}, e
	}
	defer lines.Close()
	for lines.Next() {
		var n NodeLineage
		var cur int
		var linked, sup string
		if e = lines.Scan(&n.NodeID, &cur, &linked, &sup, &n.CorrelationID); e != nil {
			return Machine{}, e
		}
		n.Current = cur == 1
		if n.LinkedAt, e = time.Parse(machineTimeFormat, linked); e != nil {
			return Machine{}, e
		}
		if n.SupersededAt, e = nullableTime(sup); e != nil {
			return Machine{}, e
		}
		m.Lineage = append(m.Lineage, n)
	}
	return m, lines.Err()
}

func (s *sqliteRepository) List(ctx context.Context) ([]Machine, error) {
	rows, e := s.db.QueryContext(ctx, "SELECT id FROM machines ORDER BY created_at DESC")
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			return nil, e
		}
		ids = append(ids, id)
	}
	if e = rows.Err(); e != nil {
		return nil, e
	}
	if e = rows.Close(); e != nil {
		return nil, e
	}

	// The production pool is deliberately capped at one SQLite connection.
	// Materialize and close the id query before loading each aggregate; nesting
	// Get while rows is open would wait forever for that same connection.
	out := make([]Machine, 0, len(ids))
	for _, id := range ids {
		m, e := s.Get(ctx, id)
		if e != nil {
			return nil, e
		}
		out = append(out, m)
	}
	return out, nil
}

func (s *sqliteRepository) Archive(ctx context.Context, id string, version int64) (Machine, error) {
	m, e := s.Get(ctx, id)
	if e != nil {
		return Machine{}, e
	}
	if m.Version != version {
		return Machine{}, ErrConflict{id, m.Version}
	}
	if m.Lifecycle == LifecycleArchived {
		return m, nil
	}
	now := s.clock.Now().UTC()
	r, e := s.db.ExecContext(ctx, "UPDATE machines SET lifecycle='archived',version=version+1,archived_at=?,updated_at=? WHERE id=? AND version=?", now.Format(machineTimeFormat), now.Format(machineTimeFormat), id, version)
	if e != nil {
		return Machine{}, e
	}
	n, _ := r.RowsAffected()
	if n == 0 {
		return Machine{}, ErrConflict{id, version}
	}
	return s.Get(ctx, id)
}

// Remove is a local lifecycle effect only. It preserves Machine, attempt, Node
// lineage, and audit history; exceptional history purge is deliberately not a
// repository convenience method.
func (s *sqliteRepository) Remove(ctx context.Context, id string, version int64) (Machine, error) {
	machine, err := s.Get(ctx, id)
	if err != nil {
		return Machine{}, err
	}
	if machine.Version != version {
		return Machine{}, ErrConflict{id, machine.Version}
	}
	if machine.Lifecycle == LifecycleRemoved {
		return machine, nil
	}
	now := s.clock.Now().UTC()
	result, err := s.db.ExecContext(ctx, "UPDATE machines SET lifecycle='removed',version=version+1,removed_at=?,updated_at=? WHERE id=? AND version=?", now.Format(machineTimeFormat), now.Format(machineTimeFormat), id, version)
	if err != nil {
		return Machine{}, err
	}
	changed, _ := result.RowsAffected()
	if changed == 0 {
		return Machine{}, ErrConflict{id, version}
	}
	return s.Get(ctx, id)
}

func (s *sqliteRepository) LinkNode(ctx context.Context, id, node, correlation string) (Machine, error) {
	if _, e := s.Get(ctx, id); e != nil {
		return Machine{}, e
	}
	now := s.clock.Now().UTC()
	starter, ok := s.db.(transactionStarter)
	if !ok {
		return Machine{}, fmt.Errorf("machine lineage requires transactional database")
	}
	tx, e := starter.BeginTx(ctx, nil)
	if e != nil {
		return Machine{}, fmt.Errorf("begin machine lineage transaction: %w", e)
	}
	defer tx.Rollback()
	if _, e = tx.ExecContext(ctx, "UPDATE machine_node_lineage SET is_current=0,superseded_at=? WHERE machine_id=? AND is_current=1", now.Format(machineTimeFormat), id); e != nil {
		return Machine{}, fmt.Errorf("supersede current node: %w", e)
	}
	if _, e = tx.ExecContext(ctx, "INSERT INTO machine_node_lineage (id,machine_id,node_id,is_current,linked_at,source_correlation_id) VALUES (?,?,?,?,?,?) ON CONFLICT(machine_id,node_id) DO UPDATE SET is_current=1,superseded_at='',source_correlation_id=excluded.source_correlation_id", uuid.NewString(), id, node, 1, now.Format(machineTimeFormat), correlation); e != nil {
		return Machine{}, fmt.Errorf("upsert node lineage: %w", e)
	}
	if e = tx.Commit(); e != nil {
		return Machine{}, fmt.Errorf("commit machine lineage: %w", e)
	}
	return s.Get(ctx, id)
}

func (s *sqliteRepository) ListMigrationReviews(ctx context.Context) ([]MigrationReview, error) {
	rows, e := s.db.QueryContext(ctx, "SELECT id,legacy_source,legacy_id,status,confidence,reason,created_at,reviewed_at FROM machine_migration_reviews ORDER BY created_at,id")
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	var reviews []MigrationReview
	for rows.Next() {
		var review MigrationReview
		var created, reviewed string
		if e = rows.Scan(&review.ID, &review.LegacySource, &review.LegacyID, &review.Status, &review.Confidence, &review.Reason, &created, &reviewed); e != nil {
			return nil, e
		}
		if review.CreatedAt, e = time.Parse(machineTimeFormat, created); e != nil {
			return nil, e
		}
		if review.ReviewedAt, e = nullableTime(reviewed); e != nil {
			return nil, e
		}
		reviews = append(reviews, review)
	}
	return reviews, rows.Err()
}

// AcknowledgeMigrationReview records an explicit operator decision that the
// ambiguous historic record was inspected. It does not create or link a
// Machine: an association requires a future correlation-backed action.
func (s *sqliteRepository) AcknowledgeMigrationReview(ctx context.Context, id string) (MigrationReview, error) {
	now := s.clock.Now().UTC().Format(machineTimeFormat)
	result, e := s.db.ExecContext(ctx, "UPDATE machine_migration_reviews SET status='acknowledged',reviewed_at=? WHERE id=? AND status='needs_review'", now, id)
	if e != nil {
		return MigrationReview{}, e
	}
	if changed, _ := result.RowsAffected(); changed == 0 {
		var exists int
		e = s.db.QueryRowContext(ctx, "SELECT 1 FROM machine_migration_reviews WHERE id=?", id).Scan(&exists)
		if errors.Is(e, sql.ErrNoRows) {
			return MigrationReview{}, ErrNotFound{id}
		}
		if e != nil {
			return MigrationReview{}, e
		}
	}
	reviews, e := s.ListMigrationReviews(ctx)
	if e != nil {
		return MigrationReview{}, e
	}
	for _, review := range reviews {
		if review.ID == id {
			return review, nil
		}
	}
	return MigrationReview{}, ErrNotFound{id}
}

func (s *sqliteRepository) CreateCleanupTombstone(ctx context.Context, tombstone CleanupTombstone) (CleanupTombstone, error) {
	if tombstone.MachineID == "" || tombstone.Action == "" {
		return CleanupTombstone{}, ErrInvalid{"cleanup", "machine_id and action required"}
	}
	if tombstone.ID == "" {
		tombstone.ID = uuid.NewString()
	}
	if tombstone.Status == "" {
		tombstone.Status = CleanupPending
	}
	if tombstone.Status == CleanupPending {
		existing, found, err := s.pendingCleanupTombstone(ctx, tombstone.MachineID, tombstone.Action)
		if err != nil {
			return CleanupTombstone{}, err
		}
		if found {
			return existing, nil
		}
	}
	if tombstone.CreatedAt.IsZero() {
		tombstone.CreatedAt = s.clock.Now().UTC()
	}
	if tombstone.UpdatedAt.IsZero() {
		tombstone.UpdatedAt = tombstone.CreatedAt
	}
	_, err := s.db.ExecContext(ctx, "INSERT INTO machine_cleanup_tombstones (id,machine_id,action,status,detail,created_at,updated_at,acknowledged_at) VALUES (?,?,?,?,?,?,?,?)", tombstone.ID, tombstone.MachineID, tombstone.Action, string(tombstone.Status), tombstone.Detail, tombstone.CreatedAt.Format(machineTimeFormat), tombstone.UpdatedAt.Format(machineTimeFormat), formatMachineNullableTime(tombstone.AcknowledgedAt))
	if err != nil {
		if tombstone.Status == CleanupPending {
			existing, found, lookupErr := s.pendingCleanupTombstone(ctx, tombstone.MachineID, tombstone.Action)
			if lookupErr == nil && found {
				return existing, nil
			}
		}
		return CleanupTombstone{}, fmt.Errorf("create cleanup tombstone: %w", err)
	}
	return tombstone, nil
}

func (s *sqliteRepository) pendingCleanupTombstone(ctx context.Context, machineID, action string) (CleanupTombstone, bool, error) {
	var id string
	err := s.db.QueryRowContext(ctx, "SELECT id FROM machine_cleanup_tombstones WHERE machine_id=? AND action=? AND status='pending' ORDER BY created_at,id LIMIT 1", machineID, action).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return CleanupTombstone{}, false, nil
	}
	if err != nil {
		return CleanupTombstone{}, false, err
	}
	tombstone, err := s.getCleanupTombstone(ctx, id)
	return tombstone, true, err
}

// AppendAudit records an operator-visible effect after its durable local write.
// The handler intentionally uses this narrow seam so audit data cannot become a
// second source of Machine or Node state.
func (s *sqliteRepository) AppendAudit(ctx context.Context, event AuditEvent) error {
	if event.MachineID == "" || event.Action == "" {
		return ErrInvalid{"audit", "machine_id and action required"}
	}
	if event.ID == "" {
		event.ID = uuid.NewString()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = s.clock.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, "INSERT INTO machine_audit_events (id,machine_id,action,actor,detail,created_at) VALUES (?,?,?,?,?,?)", event.ID, event.MachineID, event.Action, event.Actor, event.Detail, event.CreatedAt.Format(machineTimeFormat))
	if err != nil {
		return fmt.Errorf("append machine audit: %w", err)
	}
	return nil
}

func (s *sqliteRepository) ListAudit(ctx context.Context, machineID string) ([]AuditEvent, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id,machine_id,action,actor,detail,created_at FROM machine_audit_events WHERE machine_id=? ORDER BY created_at DESC,id DESC", machineID)
	if err != nil {
		return nil, fmt.Errorf("list machine audit: %w", err)
	}
	defer rows.Close()
	events := make([]AuditEvent, 0)
	for rows.Next() {
		var event AuditEvent
		var created string
		if err := rows.Scan(&event.ID, &event.MachineID, &event.Action, &event.Actor, &event.Detail, &created); err != nil {
			return nil, err
		}
		if event.CreatedAt, err = time.Parse(machineTimeFormat, created); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate machine audit: %w", err)
	}
	return events, nil
}

func (s *sqliteRepository) UpdateCleanupTombstone(ctx context.Context, id string, status CleanupStatus, detail string) (CleanupTombstone, error) {
	switch status {
	case CleanupPending, CleanupConfirmed, CleanupNotApplicable, CleanupAbandoned:
	default:
		return CleanupTombstone{}, ErrInvalid{"cleanup.status", "invalid"}
	}
	now := s.clock.Now().UTC()
	acknowledged := ""
	if status == CleanupAbandoned {
		acknowledged = now.Format(machineTimeFormat)
	}
	result, err := s.db.ExecContext(ctx, "UPDATE machine_cleanup_tombstones SET status=?,detail=?,updated_at=?,acknowledged_at=CASE WHEN ?<>'' THEN ? ELSE acknowledged_at END WHERE id=?", string(status), detail, now.Format(machineTimeFormat), acknowledged, acknowledged, id)
	if err != nil {
		return CleanupTombstone{}, err
	}
	changed, _ := result.RowsAffected()
	if changed == 0 {
		return CleanupTombstone{}, ErrNotFound{id}
	}
	return s.getCleanupTombstone(ctx, id)
}

func (s *sqliteRepository) getCleanupTombstone(ctx context.Context, id string) (CleanupTombstone, error) {
	var t CleanupTombstone
	var status, created, updated, ack string
	err := s.db.QueryRowContext(ctx, "SELECT id,machine_id,action,status,detail,created_at,updated_at,acknowledged_at FROM machine_cleanup_tombstones WHERE id=?", id).Scan(&t.ID, &t.MachineID, &t.Action, &status, &t.Detail, &created, &updated, &ack)
	if errors.Is(err, sql.ErrNoRows) {
		return CleanupTombstone{}, ErrNotFound{id}
	}
	if err != nil {
		return CleanupTombstone{}, err
	}
	t.Status = CleanupStatus(status)
	var parseErr error
	if t.CreatedAt, parseErr = time.Parse(machineTimeFormat, created); parseErr != nil {
		return CleanupTombstone{}, parseErr
	}
	if t.UpdatedAt, parseErr = time.Parse(machineTimeFormat, updated); parseErr != nil {
		return CleanupTombstone{}, parseErr
	}
	if t.AcknowledgedAt, parseErr = nullableTime(ack); parseErr != nil {
		return CleanupTombstone{}, parseErr
	}
	return t, nil
}

func (s *sqliteRepository) ListCleanupTombstones(ctx context.Context, machineID string) ([]CleanupTombstone, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id,machine_id,action,status,detail,created_at,updated_at,acknowledged_at FROM machine_cleanup_tombstones WHERE machine_id=? ORDER BY created_at DESC,id DESC", machineID)
	if err != nil {
		return nil, fmt.Errorf("list cleanup tombstones: %w", err)
	}
	defer rows.Close()
	tombstones := make([]CleanupTombstone, 0)
	for rows.Next() {
		var tombstone CleanupTombstone
		var status, created, updated, acknowledged string
		if err := rows.Scan(&tombstone.ID, &tombstone.MachineID, &tombstone.Action, &status, &tombstone.Detail, &created, &updated, &acknowledged); err != nil {
			return nil, err
		}
		tombstone.Status = CleanupStatus(status)
		if tombstone.CreatedAt, err = time.Parse(machineTimeFormat, created); err != nil {
			return nil, err
		}
		if tombstone.UpdatedAt, err = time.Parse(machineTimeFormat, updated); err != nil {
			return nil, err
		}
		if tombstone.AcknowledgedAt, err = nullableTime(acknowledged); err != nil {
			return nil, err
		}
		tombstones = append(tombstones, tombstone)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate cleanup tombstones: %w", err)
	}
	return tombstones, nil
}

func (s *sqliteRepository) UpsertTrust(ctx context.Context, trust TrustRecord) (TrustRecord, error) {
	if trust.MachineID == "" {
		return TrustRecord{}, ErrInvalid{"trust.machine_id", "required"}
	}
	if trust.ClientKeyRef == "" {
		return TrustRecord{}, ErrInvalid{"trust.client_key_ref", "required"}
	}
	if trust.HostKeyState == "" {
		trust.HostKeyState = HostKeyUnverified
	}
	switch trust.HostKeyState {
	case HostKeyUnverified, HostKeyVerified, HostKeyReviewRequired:
	default:
		return TrustRecord{}, ErrInvalid{"trust.host_key_state", "invalid"}
	}
	// A changed server host key is never silently accepted. Keep the previous
	// verified fingerprint as the trust anchor and persist a review-required
	// state; an explicit review workflow must supply and approve the replacement.
	previous, previousErr := s.GetTrust(ctx, trust.MachineID)
	if previousErr == nil && previous.HostKeyFingerprint != "" && trust.HostKeyFingerprint != "" && previous.HostKeyFingerprint != trust.HostKeyFingerprint {
		trust.HostKeyFingerprint = previous.HostKeyFingerprint
		trust.HostKeyState = HostKeyReviewRequired
	} else if previousErr != nil {
		var notFound ErrNotFound
		if !errors.As(previousErr, &notFound) {
			return TrustRecord{}, previousErr
		}
	}
	trust.UpdatedAt = s.clock.Now().UTC()
	_, err := s.db.ExecContext(ctx, `INSERT INTO machine_trust (machine_id,client_key_ref,client_key_fingerprint,host_key_fingerprint,host_key_state,updated_at) VALUES (?,?,?,?,?,?) ON CONFLICT(machine_id) DO UPDATE SET client_key_ref=excluded.client_key_ref,client_key_fingerprint=excluded.client_key_fingerprint,host_key_fingerprint=excluded.host_key_fingerprint,host_key_state=excluded.host_key_state,updated_at=excluded.updated_at`, trust.MachineID, trust.ClientKeyRef, trust.ClientKeyFingerprint, trust.HostKeyFingerprint, string(trust.HostKeyState), trust.UpdatedAt.Format(machineTimeFormat))
	if err != nil {
		return TrustRecord{}, fmt.Errorf("upsert machine trust: %w", err)
	}
	return trust, nil
}

func (s *sqliteRepository) GetTrust(ctx context.Context, machineID string) (TrustRecord, error) {
	var trust TrustRecord
	var state, updated string
	err := s.db.QueryRowContext(ctx, "SELECT machine_id,client_key_ref,client_key_fingerprint,host_key_fingerprint,host_key_state,updated_at FROM machine_trust WHERE machine_id=?", machineID).Scan(&trust.MachineID, &trust.ClientKeyRef, &trust.ClientKeyFingerprint, &trust.HostKeyFingerprint, &state, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return TrustRecord{}, ErrNotFound{machineID}
	}
	if err != nil {
		return TrustRecord{}, err
	}
	trust.HostKeyState = HostKeyState(state)
	var parseErr error
	trust.UpdatedAt, parseErr = time.Parse(machineTimeFormat, updated)
	if parseErr != nil {
		return TrustRecord{}, parseErr
	}
	return trust, nil
}

// ReviewHostKey records an explicit owner approval for a replacement host key.
// It never receives a private key reference and is deliberately distinct from
// automatic enrollment writes, which remain fail-closed on mismatch.
func (s *sqliteRepository) ReviewHostKey(ctx context.Context, machineID, replacement string) (TrustRecord, error) {
	replacement = strings.TrimSpace(replacement)
	if replacement == "" {
		return TrustRecord{}, ErrInvalid{"replacement_host_key_fingerprint", "required"}
	}
	trust, err := s.GetTrust(ctx, machineID)
	if err != nil {
		return TrustRecord{}, err
	}
	if trust.HostKeyState != HostKeyReviewRequired {
		return TrustRecord{}, ErrConflict{ID: machineID}
	}
	now := s.clock.Now().UTC()
	result, err := s.db.ExecContext(ctx, "UPDATE machine_trust SET host_key_fingerprint=?,host_key_state=?,updated_at=? WHERE machine_id=? AND host_key_state=?", replacement, string(HostKeyVerified), now.Format(machineTimeFormat), machineID, string(HostKeyReviewRequired))
	if err != nil {
		return TrustRecord{}, fmt.Errorf("review machine host key: %w", err)
	}
	if changed, _ := result.RowsAffected(); changed == 0 {
		return TrustRecord{}, ErrConflict{ID: machineID}
	}
	return s.GetTrust(ctx, machineID)
}

func (s *sqliteRepository) SavePolicySnapshot(ctx context.Context, snapshot PolicySnapshot) (PolicySnapshot, error) {
	if snapshot.MachineID == "" || snapshot.ProfileID == "" || snapshot.ProfileVersion == "" || snapshot.JSON == "" {
		return PolicySnapshot{}, ErrInvalid{"policy_snapshot", "machine/profile/snapshot required"}
	}
	// A snapshot is immutable evidence. A second resolution for a Machine must
	// use an explicit policy-change workflow rather than overwrite this record.
	_, err := s.db.ExecContext(ctx, "INSERT INTO machine_policy_snapshots (machine_id,profile_id,profile_version,snapshot_json,created_at) VALUES (?,?,?,?,?)", snapshot.MachineID, snapshot.ProfileID, snapshot.ProfileVersion, snapshot.JSON, s.clock.Now().UTC().Format(machineTimeFormat))
	if err != nil {
		return PolicySnapshot{}, fmt.Errorf("save policy snapshot: %w", err)
	}
	return snapshot, nil
}

// ApplyPolicy resolves a built-in profile once, appends immutable evidence,
// and updates only the Machine's desired policy under optimistic concurrency.
// Registry-approved scopes are intentionally not present in this transaction.
func (s *sqliteRepository) ApplyPolicy(ctx context.Context, input PolicyChangeInput) (Machine, PolicySnapshot, error) {
	if input.MachineID == "" {
		return Machine{}, PolicySnapshot{}, ErrInvalid{"machine_id", "required"}
	}
	snapshot, err := ResolveProfile(input.MachineID, input.ProfileID, input.ProfileVersion, input.Overrides)
	if err != nil {
		return Machine{}, PolicySnapshot{}, err
	}
	overrides, err := json.Marshal(input.Overrides)
	if err != nil {
		return Machine{}, PolicySnapshot{}, fmt.Errorf("encode policy overrides: %w", err)
	}
	starter, ok := s.db.(transactionStarter)
	if !ok {
		return Machine{}, PolicySnapshot{}, fmt.Errorf("policy change requires transactional database")
	}
	tx, err := starter.BeginTx(ctx, nil)
	if err != nil {
		return Machine{}, PolicySnapshot{}, fmt.Errorf("begin policy change: %w", err)
	}
	defer tx.Rollback()
	var current int64
	if err = tx.QueryRowContext(ctx, "SELECT version FROM machines WHERE id=?", input.MachineID).Scan(&current); errors.Is(err, sql.ErrNoRows) {
		return Machine{}, PolicySnapshot{}, ErrNotFound{input.MachineID}
	} else if err != nil {
		return Machine{}, PolicySnapshot{}, err
	}
	if current != input.ExpectedVersion {
		return Machine{}, PolicySnapshot{}, ErrConflict{ID: input.MachineID, Version: current}
	}
	var previousJSON string
	err = tx.QueryRowContext(ctx, "SELECT snapshot_json FROM machine_policy_snapshot_history WHERE machine_id=? ORDER BY created_at DESC,id DESC LIMIT 1", input.MachineID).Scan(&previousJSON)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Machine{}, PolicySnapshot{}, fmt.Errorf("read prior policy snapshot: %w", err)
	}
	if previousJSON != "" && !input.ConfirmRemoval {
		var previous struct{ SuggestedScopes, RequiredCapabilities []string }
		if err := json.Unmarshal([]byte(previousJSON), &previous); err != nil {
			return Machine{}, PolicySnapshot{}, fmt.Errorf("decode prior policy snapshot: %w", err)
		}
		if policyRemoves(previous.SuggestedScopes, snapshot.SuggestedScopes) || policyRemoves(previous.RequiredCapabilities, snapshot.RequiredCapabilities) {
			return Machine{}, PolicySnapshot{}, ErrInvalid{"confirm_removal", "required when removing policy capability or scope intent"}
		}
	}
	now := s.clock.Now().UTC()
	if _, err = tx.ExecContext(ctx, "INSERT INTO machine_policy_snapshot_history (id,machine_id,profile_id,profile_version,overrides_json,snapshot_json,actor,reason,created_at) VALUES (?,?,?,?,?,?,?,?,?)", uuid.NewString(), input.MachineID, snapshot.ProfileID, snapshot.ProfileVersion, string(overrides), snapshot.JSON, input.Actor, input.Reason, now.Format(machineTimeFormat)); err != nil {
		return Machine{}, PolicySnapshot{}, fmt.Errorf("append policy snapshot: %w", err)
	}
	result, err := tx.ExecContext(ctx, "UPDATE machines SET desired_profile_id=?,desired_profile_version=?,version=version+1,updated_at=? WHERE id=? AND version=?", snapshot.ProfileID, snapshot.ProfileVersion, now.Format(machineTimeFormat), input.MachineID, input.ExpectedVersion)
	if err != nil {
		return Machine{}, PolicySnapshot{}, err
	}
	if changed, _ := result.RowsAffected(); changed == 0 {
		return Machine{}, PolicySnapshot{}, ErrConflict{ID: input.MachineID, Version: input.ExpectedVersion}
	}
	if err = tx.Commit(); err != nil {
		return Machine{}, PolicySnapshot{}, fmt.Errorf("commit policy change: %w", err)
	}
	machine, err := s.Get(ctx, input.MachineID)
	return machine, snapshot, err
}

func policyRemoves(previous, next []string) bool {
	present := make(map[string]struct{}, len(next))
	for _, value := range next {
		present[value] = struct{}{}
	}
	for _, value := range previous {
		if _, ok := present[value]; !ok {
			return true
		}
	}
	return false
}

func nullableTime(v string) (time.Time, error) {
	if v == "" {
		return time.Time{}, nil
	}
	return time.Parse(machineTimeFormat, v)
}

func formatMachineNullableTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(machineTimeFormat)
}
