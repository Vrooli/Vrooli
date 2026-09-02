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

	"github.com/vrooli/api-core/schedule"
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
	clock schedule.Clock
}

func NewSQLiteRepository(db SQLExecutor, c schedule.Clock) Repository {
	return &sqliteRepository{db, c}
}

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
	defer func() { _ = tx.Rollback() }()

	// Check the claims inside the same transaction as insertion. The primary
	// key on machine_locator_claims is the final race-safe guard; this check
	// makes normal retries return the existing Machine instead of surfacing a
	// low-level SQLite uniqueness error.
	matched := make(map[string]string)
	for _, locator := range locators {
		var existingID string
		err := tx.QueryRowContext(ctx, `SELECT c.machine_id FROM machine_locator_claims c JOIN machines m ON m.id=c.machine_id WHERE m.lifecycle='active' AND c.kind=? AND c.normalized_value=?`, strings.ToLower(strings.TrimSpace(locator.Kind)), locator.normalized).Scan(&existingID)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return Machine{}, fmt.Errorf("resolve active machine claim: %w", err)
		}
		matched[existingID] = existingID
	}
	if len(matched) == 1 {
		for _, existingID := range matched {
			_ = tx.Rollback()
			return s.Get(ctx, existingID)
		}
	}
	if len(matched) > 1 {
		return Machine{}, ErrAmbiguous{Evidence: "submitted locators"}
	}
	if _, e := tx.ExecContext(ctx, "INSERT INTO machines (id,lifecycle,version,desired_profile_id,desired_profile_version,trust_ref,created_at,updated_at) VALUES (?, 'active',1,?,?,?,?,?)", id, in.DesiredProfileID, in.DesiredProfileVersion, in.TrustRef, now.Format(machineTimeFormat), now.Format(machineTimeFormat)); e != nil {
		return Machine{}, fmt.Errorf("insert machine: %w", e)
	}
	for _, l := range locators {
		if _, e = tx.ExecContext(ctx, "INSERT INTO machine_locators (machine_id,ordinal,kind,value,normalized_value,created_at) VALUES (?,?,?,?,?,?)", id, l.Ordinal, l.Kind, l.Value, l.normalized, now.Format(machineTimeFormat)); e != nil {
			return Machine{}, fmt.Errorf("insert locator: %w", e)
		}
		if _, e = tx.ExecContext(ctx, "INSERT INTO machine_locator_claims (kind,normalized_value,machine_id,created_at) VALUES (?,?,?,?)", strings.ToLower(strings.TrimSpace(l.Kind)), l.normalized, id, now.Format(machineTimeFormat)); e != nil {
			if isUniqueConstraint(e) {
				return Machine{}, ErrAmbiguous{Evidence: l.Kind + "=" + l.normalized}
			}
			return Machine{}, fmt.Errorf("claim locator: %w", e)
		}
	}
	if e = tx.Commit(); e != nil {
		return Machine{}, fmt.Errorf("commit machine creation: %w", e)
	}
	return s.Get(ctx, id)
}

func (s *sqliteRepository) Resolve(ctx context.Context, query IdentityQuery) (Machine, error) {
	if strings.TrimSpace(query.MachineID) != "" {
		return s.Get(ctx, strings.TrimSpace(query.MachineID))
	}
	if nodeID := strings.TrimSpace(query.NodeID); nodeID != "" {
		return s.machineForEvidence(ctx, "node_id", nodeID, `SELECT l.machine_id FROM machine_node_lineage l JOIN machines m ON m.id=l.machine_id WHERE l.node_id=? AND l.is_current=1 AND m.lifecycle='active'`)
	}
	if fingerprint := strings.TrimSpace(query.SSHHostKeyFingerprint); fingerprint != "" {
		return s.machineForEvidence(ctx, "ssh host-key fingerprint", fingerprint, `SELECT l.machine_id FROM machine_locators l JOIN machines m ON m.id=l.machine_id WHERE l.kind='ssh-host-key' AND l.normalized_value=? AND m.lifecycle='active'`)
	}
	if hostname := strings.TrimSpace(query.Hostname); hostname != "" {
		normalized, err := normalizeLocator("hostname", hostname)
		if err != nil {
			return Machine{}, err
		}
		return s.machineForEvidence(ctx, "hostname "+normalized, normalized, `SELECT l.machine_id FROM machine_locators l JOIN machines m ON m.id=l.machine_id WHERE l.kind='hostname' AND l.normalized_value=? AND m.lifecycle='active'`)
	}
	return Machine{}, ErrInvalid{"identity", "at least one identity fact is required"}
}

func isUniqueConstraint(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "unique constraint")
}

func (s *sqliteRepository) machineForEvidence(ctx context.Context, evidence, value, query string) (Machine, error) {
	rows, err := s.db.QueryContext(ctx, query, value)
	if err != nil {
		return Machine{}, fmt.Errorf("resolve machine by %s: %w", evidence, err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return Machine{}, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return Machine{}, err
	}
	if len(ids) == 0 {
		return Machine{}, ErrNotFound{ID: value}
	}
	if len(ids) > 1 {
		return Machine{}, ErrAmbiguous{Evidence: evidence + "=" + value}
	}
	return s.Get(ctx, ids[0])
}

func (s *sqliteRepository) Get(ctx context.Context, id string) (Machine, error) {
	var m Machine
	var created, updated, arch, removed, appliedAt string
	e := s.db.QueryRowContext(ctx, "SELECT id,lifecycle,version,desired_profile_id,desired_profile_version,desired_selection_json,applied_profile_id,applied_profile_version,applied_selection_json,applied_at,trust_ref,created_at,updated_at,archived_at,removed_at FROM machines WHERE id=?", id).Scan(&m.ID, &m.Lifecycle, &m.Version, &m.DesiredProfileID, &m.DesiredProfileVersion, &m.DesiredSelectionJSON, &m.AppliedProfileID, &m.AppliedProfileVersion, &m.AppliedSelectionJSON, &appliedAt, &m.TrustRef, &created, &updated, &arch, &removed)
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
	if m.AppliedAt, er = nullableTime(appliedAt); er != nil {
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
	if _, err := s.db.ExecContext(ctx, "DELETE FROM machine_locator_claims WHERE machine_id=?", id); err != nil {
		return Machine{}, fmt.Errorf("release archived machine locator claims: %w", err)
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
	if _, err := s.db.ExecContext(ctx, "DELETE FROM machine_locator_claims WHERE machine_id=?", id); err != nil {
		return Machine{}, fmt.Errorf("release removed machine locator claims: %w", err)
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
	defer func() { _ = tx.Rollback() }()
	// A retry may encounter a stale current lineage left on an archived or
	// removed Machine by an older enrollment attempt. It is safe to supersede
	// that historical row while linking the node to the active target. Never
	// steal a node from another active Machine: that remains an explicit merge
	// or operator-repair decision and must fail closed.
	var conflictingMachineID, conflictingLifecycle string
	conflictErr := tx.QueryRowContext(ctx, `
		SELECT l.machine_id,m.lifecycle
		FROM machine_node_lineage l
		JOIN machines m ON m.id=l.machine_id
		WHERE l.node_id=? AND l.is_current=1 AND l.machine_id<>?
		LIMIT 1`, node, id).Scan(&conflictingMachineID, &conflictingLifecycle)
	if conflictErr == nil {
		if conflictingLifecycle == string(LifecycleActive) {
			return Machine{}, fmt.Errorf("node %q is currently linked to active machine %q", node, conflictingMachineID)
		}
		if _, e = tx.ExecContext(ctx, "UPDATE machine_node_lineage SET is_current=0,superseded_at=? WHERE machine_id=? AND node_id=? AND is_current=1", now.Format(machineTimeFormat), conflictingMachineID, node); e != nil {
			return Machine{}, fmt.Errorf("supersede stale node lineage: %w", e)
		}
	} else if !errors.Is(conflictErr, sql.ErrNoRows) {
		return Machine{}, fmt.Errorf("check existing node lineage owner: %w", conflictErr)
	}
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

// Merge folds duplicate durable identities into the explicitly selected
// target. It keeps all historical attempts/lineage on the target, archives the
// source, and writes exactly one audit record for the operator-visible effect.
func (s *sqliteRepository) Merge(ctx context.Context, input MergeInput) (Machine, error) {
	fromID, intoID := strings.TrimSpace(input.FromMachineID), strings.TrimSpace(input.IntoMachineID)
	if fromID == "" || intoID == "" {
		return Machine{}, ErrInvalid{"merge", "from and into machine ids are required"}
	}
	if fromID == intoID {
		return Machine{}, ErrInvalid{"merge", "source and target must differ"}
	}
	from, err := s.Get(ctx, fromID)
	if err != nil {
		return Machine{}, err
	}
	into, err := s.Get(ctx, intoID)
	if err != nil {
		return Machine{}, err
	}
	if from.Lifecycle == LifecycleRemoved {
		return Machine{}, ErrInvalid{"from_machine_id", "removed machines cannot be merged"}
	}
	if into.Lifecycle != LifecycleActive {
		return Machine{}, ErrInvalid{"into_machine_id", "target machine must be active"}
	}
	starter, ok := s.db.(transactionStarter)
	if !ok {
		return Machine{}, fmt.Errorf("machine merge requires transactional database")
	}
	tx, err := starter.BeginTx(ctx, nil)
	if err != nil {
		return Machine{}, fmt.Errorf("begin machine merge: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	now := s.clock.Now().UTC().Format(machineTimeFormat)
	// Fold locators while preserving the target's stable ordinals and avoiding
	// a duplicate locator on replay.
	var nextOrdinal int
	if err := tx.QueryRowContext(ctx, "SELECT COALESCE(MAX(ordinal)+1,0) FROM machine_locators WHERE machine_id=?", intoID).Scan(&nextOrdinal); err != nil {
		return Machine{}, err
	}
	locRows, err := tx.QueryContext(ctx, "SELECT kind,value,normalized_value FROM machine_locators WHERE machine_id=? ORDER BY ordinal", fromID)
	if err != nil {
		return Machine{}, err
	}
	defer locRows.Close()
	for locRows.Next() {
		var kind, value, normalized string
		if err := locRows.Scan(&kind, &value, &normalized); err != nil {
			locRows.Close()
			return Machine{}, err
		}
		var exists int
		if err := tx.QueryRowContext(ctx, "SELECT 1 FROM machine_locators WHERE machine_id=? AND kind=? AND normalized_value=?", intoID, kind, normalized).Scan(&exists); errors.Is(err, sql.ErrNoRows) {
			if _, err := tx.ExecContext(ctx, "INSERT INTO machine_locators (machine_id,ordinal,kind,value,normalized_value,created_at) VALUES (?,?,?,?,?,?)", intoID, nextOrdinal, kind, value, normalized, now); err != nil {
				locRows.Close()
				return Machine{}, err
			}
			nextOrdinal++
		} else if err != nil {
			locRows.Close()
			return Machine{}, err
		}
	}
	if err := locRows.Close(); err != nil {
		return Machine{}, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO machine_locator_claims (kind,normalized_value,machine_id,created_at) SELECT kind,normalized_value,?,? FROM machine_locator_claims WHERE machine_id=?`, intoID, now, fromID); err != nil {
		return Machine{}, fmt.Errorf("transfer source locator claims: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM machine_locator_claims WHERE machine_id=?", fromID); err != nil {
		return Machine{}, fmt.Errorf("release source locator claims: %w", err)
	}
	// A current source lineage wins only when the target has no current node.
	var targetCurrent string
	_ = tx.QueryRowContext(ctx, "SELECT node_id FROM machine_node_lineage WHERE machine_id=? AND is_current=1 LIMIT 1", intoID).Scan(&targetCurrent)
	lineRows, err := tx.QueryContext(ctx, "SELECT node_id,is_current,linked_at,superseded_at,source_correlation_id FROM machine_node_lineage WHERE machine_id=? ORDER BY linked_at", fromID)
	if err != nil {
		return Machine{}, err
	}
	defer lineRows.Close()
	type lineageCopy struct {
		nodeID, linked, superseded, correlation string
		current                                 int
	}
	var sourceLineage []lineageCopy
	for lineRows.Next() {
		var nodeID, linked, superseded, correlation string
		var current int
		if err := lineRows.Scan(&nodeID, &current, &linked, &superseded, &correlation); err != nil {
			lineRows.Close()
			return Machine{}, err
		}
		sourceLineage = append(sourceLineage, lineageCopy{nodeID: nodeID, current: current, linked: linked, superseded: superseded, correlation: correlation})
	}
	if err := lineRows.Close(); err != nil {
		return Machine{}, err
	}
	// Release source current flags before inserting target copies; the global
	// partial index otherwise (correctly) rejects the transient duplicate.
	if _, err := tx.ExecContext(ctx, "UPDATE machine_node_lineage SET is_current=0,superseded_at=? WHERE machine_id=? AND is_current=1", now, fromID); err != nil {
		return Machine{}, err
	}
	for _, lineage := range sourceLineage {
		nodeID, current, linked, superseded, correlation := lineage.nodeID, lineage.current, lineage.linked, lineage.superseded, lineage.correlation
		var existingCurrent int
		existingErr := tx.QueryRowContext(ctx, "SELECT is_current FROM machine_node_lineage WHERE machine_id=? AND node_id=?", intoID, nodeID).Scan(&existingCurrent)
		if errors.Is(existingErr, sql.ErrNoRows) {
			useCurrent := current == 1 && targetCurrent == ""
			if useCurrent {
				targetCurrent = nodeID
			}
			if _, err := tx.ExecContext(ctx, "INSERT INTO machine_node_lineage (id,machine_id,node_id,is_current,linked_at,superseded_at,source_correlation_id) VALUES (?,?,?,?,?,?,?)", uuid.NewString(), intoID, nodeID, boolInt(useCurrent), linked, superseded, correlation); err != nil {
				lineRows.Close()
				return Machine{}, err
			}
		} else if existingErr != nil {
			lineRows.Close()
			return Machine{}, existingErr
		}
	}
	if _, err := tx.ExecContext(ctx, "UPDATE machines SET lifecycle='archived',version=version+1,archived_at=?,updated_at=? WHERE id=?", now, now, fromID); err != nil {
		return Machine{}, err
	}
	if attemptsTable, err := migrationTableExists(ctx, tx, "enrollment_attempts"); err != nil {
		return Machine{}, err
	} else if attemptsTable {
		if _, err := tx.ExecContext(ctx, "UPDATE enrollment_attempts SET machine_id=? WHERE machine_id=?", intoID, fromID); err != nil {
			return Machine{}, fmt.Errorf("move enrollment attempt lineage: %w", err)
		}
	}
	detail := fmt.Sprintf("source=%s target=%s", fromID, intoID)
	if _, err := tx.ExecContext(ctx, "INSERT INTO machine_audit_events (id,machine_id,action,actor,detail,created_at) VALUES (?,?,?,?,?,?)", uuid.NewString(), intoID, "merge", input.Actor, detail, now); err != nil {
		return Machine{}, err
	}
	if err := tx.Commit(); err != nil {
		return Machine{}, fmt.Errorf("commit machine merge: %w", err)
	}
	return s.Get(ctx, intoID)
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
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
	if trust.SSHPort == 0 {
		trust.SSHPort = 22
	}
	if trust.ConnectionState == "" {
		trust.ConnectionState = ConnectionUntrusted
	}
	switch trust.ConnectionState {
	case ConnectionUntrusted, ConnectionTrusted, ConnectionRecovery:
	default:
		return TrustRecord{}, ErrInvalid{"trust.connection_state", "invalid"}
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
	_, err := s.db.ExecContext(ctx, `INSERT INTO machine_trust (machine_id,client_key_ref,client_key_fingerprint,host_key_fingerprint,host_key_state,ssh_user,ssh_port,connection_state,updated_at) VALUES (?,?,?,?,?,?,?,?,?) ON CONFLICT(machine_id) DO UPDATE SET client_key_ref=excluded.client_key_ref,client_key_fingerprint=excluded.client_key_fingerprint,host_key_fingerprint=excluded.host_key_fingerprint,host_key_state=excluded.host_key_state,ssh_user=excluded.ssh_user,ssh_port=excluded.ssh_port,connection_state=excluded.connection_state,updated_at=excluded.updated_at`, trust.MachineID, trust.ClientKeyRef, trust.ClientKeyFingerprint, trust.HostKeyFingerprint, string(trust.HostKeyState), trust.SSHUser, trust.SSHPort, string(trust.ConnectionState), trust.UpdatedAt.Format(machineTimeFormat))
	if err != nil {
		return TrustRecord{}, fmt.Errorf("upsert machine trust: %w", err)
	}
	// Host-key evidence is a durable machine locator, not merely trust display
	// data. It lets a re-pair after hostname/DHCP changes resolve the original
	// Machine before any new record can be created.
	if trust.HostKeyFingerprint != "" {
		normalized, normalizeErr := normalizeLocator("ssh-host-key", trust.HostKeyFingerprint)
		if normalizeErr != nil {
			return TrustRecord{}, normalizeErr
		}
		var ordinal int
		if err := s.db.QueryRowContext(ctx, "SELECT COALESCE(MAX(ordinal)+1,0) FROM machine_locators WHERE machine_id=?", trust.MachineID).Scan(&ordinal); err != nil {
			return TrustRecord{}, err
		}
		if _, err := s.db.ExecContext(ctx, "INSERT OR IGNORE INTO machine_locators (machine_id,ordinal,kind,value,normalized_value,created_at) VALUES (?,?,?,?,?,?)", trust.MachineID, ordinal, "ssh-host-key", trust.HostKeyFingerprint, normalized, trust.UpdatedAt.Format(machineTimeFormat)); err != nil {
			return TrustRecord{}, fmt.Errorf("store machine host-key locator: %w", err)
		}
	}
	return trust, nil
}

func (s *sqliteRepository) GetTrust(ctx context.Context, machineID string) (TrustRecord, error) {
	var trust TrustRecord
	var state, connectionState, updated string
	err := s.db.QueryRowContext(ctx, "SELECT machine_id,client_key_ref,client_key_fingerprint,host_key_fingerprint,host_key_state,ssh_user,ssh_port,connection_state,updated_at FROM machine_trust WHERE machine_id=?", machineID).Scan(&trust.MachineID, &trust.ClientKeyRef, &trust.ClientKeyFingerprint, &trust.HostKeyFingerprint, &state, &trust.SSHUser, &trust.SSHPort, &connectionState, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return TrustRecord{}, ErrNotFound{machineID}
	}
	if err != nil {
		return TrustRecord{}, err
	}
	trust.HostKeyState = HostKeyState(state)
	trust.ConnectionState = ConnectionState(connectionState)
	if trust.SSHPort == 0 {
		trust.SSHPort = 22
	}
	if trust.ConnectionState == "" {
		trust.ConnectionState = ConnectionUntrusted
	}
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

// LatestPolicySnapshot returns the immutable policy decision most recently
// recorded for a machine. History is authoritative for policy changes; the
// original single-snapshot table remains a compatibility fallback for older
// installations.
func (s *sqliteRepository) LatestPolicySnapshot(ctx context.Context, machineID string) (PolicySnapshot, error) {
	var profileID, profileVersion, snapshotJSON string
	err := s.db.QueryRowContext(ctx, "SELECT profile_id, profile_version, snapshot_json FROM machine_policy_snapshot_history WHERE machine_id=? ORDER BY created_at DESC,id DESC LIMIT 1", machineID).Scan(&profileID, &profileVersion, &snapshotJSON)
	if errors.Is(err, sql.ErrNoRows) {
		err = s.db.QueryRowContext(ctx, "SELECT profile_id, profile_version, snapshot_json FROM machine_policy_snapshots WHERE machine_id=?", machineID).Scan(&profileID, &profileVersion, &snapshotJSON)
	}
	if errors.Is(err, sql.ErrNoRows) {
		return PolicySnapshot{}, ErrNotFound{machineID}
	}
	if err != nil {
		return PolicySnapshot{}, fmt.Errorf("read latest policy snapshot: %w", err)
	}
	return decodePolicySnapshot(machineID, profileID, profileVersion, snapshotJSON)
}

func decodePolicySnapshot(machineID, profileID, profileVersion, snapshotJSON string) (PolicySnapshot, error) {
	var payload struct {
		Preset               string
		Scenarios            []string
		OptionalResources    []string
		SuggestedScopes      []string
		RequiredCapabilities []string
	}
	if err := json.Unmarshal([]byte(snapshotJSON), &payload); err != nil {
		return PolicySnapshot{}, fmt.Errorf("decode policy snapshot: %w", err)
	}
	return PolicySnapshot{MachineID: machineID, ProfileID: profileID, ProfileVersion: profileVersion, Preset: payload.Preset, Scenarios: payload.Scenarios, OptionalResources: payload.OptionalResources, SuggestedScopes: payload.SuggestedScopes, RequiredCapabilities: payload.RequiredCapabilities, JSON: snapshotJSON}, nil
}

// MarkProfileApplied records the profile that remote onboarding applied after
// its readiness check succeeded. Repeating the same result is idempotent.
func (s *sqliteRepository) MarkProfileApplied(ctx context.Context, machineID, profileID, profileVersion string, appliedAt time.Time) (Machine, error) {
	if machineID == "" || profileID == "" || profileVersion == "" || appliedAt.IsZero() {
		return Machine{}, ErrInvalid{"applied_profile", "machine, profile, version, and time are required"}
	}
	stamp := appliedAt.UTC().Format(machineTimeFormat)
	var desiredSelection string
	if err := s.db.QueryRowContext(ctx, "SELECT desired_selection_json FROM machines WHERE id=?", machineID).Scan(&desiredSelection); err != nil {
		return Machine{}, err
	}
	result, err := s.db.ExecContext(ctx, "UPDATE machines SET applied_profile_id=?,applied_profile_version=?,applied_selection_json=?,applied_at=?,updated_at=? WHERE id=?", profileID, profileVersion, desiredSelection, stamp, stamp, machineID)
	if err != nil {
		return Machine{}, fmt.Errorf("mark profile applied: %w", err)
	}
	changed, _ := result.RowsAffected()
	if changed == 0 {
		return Machine{}, ErrNotFound{machineID}
	}
	return s.Get(ctx, machineID)
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
	defer func() { _ = tx.Rollback() }()
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
	result, err := tx.ExecContext(ctx, "UPDATE machines SET desired_profile_id=?,desired_profile_version=?,desired_selection_json=?,version=version+1,updated_at=? WHERE id=? AND version=?", snapshot.ProfileID, snapshot.ProfileVersion, snapshot.SelectionJSON, now.Format(machineTimeFormat), input.MachineID, input.ExpectedVersion)
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
