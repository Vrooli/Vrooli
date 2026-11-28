package registry

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/api-core/schedule"

	"github.com/google/uuid"
)

// SQLExecutor is the narrow database surface sqliteRepository depends on. Both
// *sql.DB (repository unit tests via testutil/db.NewSQLite) and
// *database.RoutedDB (production via main.go) satisfy it, so production
// participates in per-request routing without the test fixture wrapping its
// handle.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type sqlBeginner interface {
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

type sqliteRepository struct {
	db          SQLExecutor
	clock       schedule.Clock
	migrateOnce sync.Once
	migrateErr  error
}

// NewSQLiteRepository constructs the production Repository.
func NewSQLiteRepository(db SQLExecutor, clk schedule.Clock) Repository {
	return &sqliteRepository{db: db, clock: clk}
}

// Compile-time guarantee.
var _ Repository = (*sqliteRepository)(nil)

const (
	// RFC3339Nano sorts lexicographically in time order for a fixed zone, so a
	// string range/order comparison is a correct filter — matching the wire
	// format and the notes domain convention.
	nodeTimeFormat = time.RFC3339Nano

	insertNodeSQL = `
	INSERT INTO nodes (id, name, kind, deployment_tier, enrollment_generation, owner_revision, os, arch, machine_arch, binary_arch, revision, endpoint, capabilities, scopes, pairing_correlation_id, created_at, updated_at, last_seen_at, revoked_at, capability_inventory, capability_probed_at, configuration_op_id, configuration_state, configuration_at, configuration_unmet)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

	selectNodeColumns = `
	SELECT id, name, kind, deployment_tier, enrollment_generation, owner_revision, os, arch, machine_arch, binary_arch, revision, endpoint, capabilities, scopes, pairing_correlation_id, created_at, updated_at, last_seen_at, revoked_at, capability_inventory, capability_probed_at, configuration_op_id, configuration_state, configuration_at, configuration_unmet
FROM nodes
`

	selectNodeByIDSQL = selectNodeColumns + `WHERE id = ?`

	listNodesSQL = selectNodeColumns + `ORDER BY created_at DESC, id DESC`

	updateNodeSQL = `
	UPDATE nodes
SET name = ?, kind = ?, deployment_tier = ?, endpoint = ?, capabilities = ?, scopes = ?, revision = ?, configuration_op_id = ?, configuration_state = ?, configuration_at = ?, configuration_unmet = ?, updated_at = ?, owner_revision=owner_revision+1
WHERE id = ? AND owner_revision = ?
`

	updateArchitectureSQL = `
UPDATE nodes
SET arch = CASE WHEN ? <> '' THEN ? ELSE arch END,
    machine_arch = ?, binary_arch = ?, updated_at = ?
WHERE id = ?
`

	revokeNodeSQL = `
UPDATE nodes
SET revoked_at = ?, updated_at = ?
WHERE id = ? AND revoked_at = ''
`
	removeNodeSQL = `DELETE FROM nodes WHERE id = ?`

	touchLastSeenSQL = `
UPDATE nodes
SET last_seen_at = ?
WHERE id = ?
`
)

func (s *sqliteRepository) Create(ctx context.Context, n Node) (Node, error) {
	if err := s.ensureColumns(ctx); err != nil {
		return Node{}, err
	}
	if n.ID == "" {
		n.ID = uuid.NewString()
	}
	if n.CreatedAt.IsZero() {
		n.CreatedAt = s.clock.Now().UTC()
	}
	if n.UpdatedAt.IsZero() {
		n.UpdatedAt = n.CreatedAt
	}

	caps, err := marshalStrings(n.Capabilities)
	if err != nil {
		return Node{}, fmt.Errorf("encode capabilities: %w", err)
	}
	scopes, err := marshalStrings(n.Scopes)
	if err != nil {
		return Node{}, fmt.Errorf("encode scopes: %w", err)
	}
	inventory, err := marshalCapabilityInventory(n.CapabilityInventory)
	if err != nil {
		return Node{}, fmt.Errorf("encode capability inventory: %w", err)
	}

	if n.Kind == "" {
		n.Kind = KindAgent
	}
	if n.OwnerRevision <= 0 {
		n.OwnerRevision = 1
	}
	_, err = s.db.ExecContext(ctx, insertNodeSQL,
		n.ID, n.Name, n.Kind, n.DeploymentTier, n.EnrollmentGeneration, n.OwnerRevision, n.OS, n.Arch, n.MachineArch, n.BinaryArch, n.Revision, n.Endpoint, caps, scopes,
		n.PairingCorrelationID,
		n.CreatedAt.Format(nodeTimeFormat), n.UpdatedAt.Format(nodeTimeFormat),
		formatNullableTime(n.LastSeenAt), formatNullableTime(n.RevokedAt),
		inventory, formatNullableTime(n.CapabilityProbedAt),
		n.ConfigurationOpID, n.ConfigurationState, formatNullableTime(n.ConfigurationAt), configurationUnmetJSON(n.ConfigurationUnmet),
	)
	if err != nil {
		return Node{}, fmt.Errorf("insert node %q: %w", n.ID, err)
	}
	return n, nil
}

func (s *sqliteRepository) GetByPairingCorrelation(ctx context.Context, correlationID string) (Node, error) {
	if err := s.ensureColumns(ctx); err != nil {
		return Node{}, err
	}
	row := s.db.QueryRowContext(ctx, selectNodeColumns+"WHERE pairing_correlation_id = ?", correlationID)
	n, err := scanNode(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Node{}, ErrNodeNotFound{ID: correlationID}
	}
	if err != nil {
		return Node{}, fmt.Errorf("get node by pairing correlation %q: %w", correlationID, err)
	}
	return n, nil
}

func (s *sqliteRepository) Get(ctx context.Context, id string) (Node, error) {
	if err := s.ensureColumns(ctx); err != nil {
		return Node{}, err
	}
	row := s.db.QueryRowContext(ctx, selectNodeByIDSQL, id)
	n, err := scanNode(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Node{}, ErrNodeNotFound{ID: id}
	}
	if err != nil {
		return Node{}, fmt.Errorf("get node %q: %w", id, err)
	}
	return n, nil
}

func (s *sqliteRepository) List(ctx context.Context) ([]Node, error) {
	if err := s.ensureColumns(ctx); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, listNodesSQL)
	if err != nil {
		return nil, fmt.Errorf("list nodes: %w", err)
	}
	defer rows.Close()

	var nodes []Node
	for rows.Next() {
		n, err := scanNode(rows)
		if err != nil {
			return nil, fmt.Errorf("scan node: %w", err)
		}
		nodes = append(nodes, n)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate nodes: %w", err)
	}
	return nodes, nil
}

func (s *sqliteRepository) Update(ctx context.Context, n Node) (Node, error) {
	if err := s.ensureColumns(ctx); err != nil {
		return Node{}, err
	}
	// Read-modify-write so the returned node carries the immutable fields
	// (os/arch/created_at) and so we return ErrNodeNotFound consistently.
	existing, err := s.Get(ctx, n.ID)
	if err != nil {
		return Node{}, err
	}

	existing.Name = n.Name
	existing.Endpoint = n.Endpoint
	existing.Capabilities = n.Capabilities
	existing.Scopes = n.Scopes
	existing.Revision = n.Revision
	if n.Kind != "" {
		existing.Kind = n.Kind
	}
	existing.UpdatedAt = s.clock.Now().UTC()
	existing.ConfigurationOpID = n.ConfigurationOpID
	existing.ConfigurationState = n.ConfigurationState
	existing.ConfigurationAt = n.ConfigurationAt
	existing.ConfigurationUnmet = n.ConfigurationUnmet

	caps, err := marshalStrings(existing.Capabilities)
	if err != nil {
		return Node{}, fmt.Errorf("encode capabilities: %w", err)
	}
	scopes, err := marshalStrings(existing.Scopes)
	if err != nil {
		return Node{}, fmt.Errorf("encode scopes: %w", err)
	}

	res, err := s.db.ExecContext(ctx, updateNodeSQL,
		existing.Name, existing.Kind, existing.DeploymentTier, existing.Endpoint, caps, scopes, existing.Revision,
		existing.ConfigurationOpID, existing.ConfigurationState, formatNullableTime(existing.ConfigurationAt), configurationUnmetJSON(existing.ConfigurationUnmet),
		existing.UpdatedAt.Format(nodeTimeFormat), existing.ID, existing.OwnerRevision,
	)
	if err != nil {
		return Node{}, fmt.Errorf("update node %q: %w", n.ID, err)
	}
	changed, err := res.RowsAffected()
	if err != nil {
		return Node{}, err
	}
	if changed == 0 {
		latest, getErr := s.Get(ctx, n.ID)
		if getErr != nil {
			return Node{}, getErr
		}
		return Node{}, ErrRevisionConflict{ID: n.ID, Expected: existing.OwnerRevision, Current: latest.OwnerRevision}
	}
	existing.OwnerRevision++
	return existing, nil
}

func (s *sqliteRepository) UpdateCAS(ctx context.Context, m CASMutation) (UpdateCASResult, error) {
	if err := s.ensureColumns(ctx); err != nil {
		return UpdateCASResult{}, err
	}
	b, ok := s.db.(sqlBeginner)
	if !ok {
		return UpdateCASResult{}, fmt.Errorf("registry database does not support atomic transactions")
	}
	tx, err := b.BeginTx(ctx, nil)
	if err != nil {
		return UpdateCASResult{}, err
	}
	defer tx.Rollback()
	now := s.clock.Now().UTC()
	_, err = tx.ExecContext(ctx, `INSERT INTO node_update_receipts(request_id,actor_id,target_id,subject,expected_revision,new_revision,payload_digest,created_at) VALUES(?,?,?,?,?,?,?,?)`, m.RequestID, m.ActorID, m.TargetID, m.Subject, m.ExpectedRevision, 0, m.PayloadDigest, now.Format(nodeTimeFormat))
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") || strings.Contains(strings.ToLower(err.Error()), "constraint") {
			prior, readErr := scanReceipt(tx.QueryRowContext(ctx, `SELECT request_id,actor_id,target_id,subject,expected_revision,new_revision,payload_digest,created_at FROM node_update_receipts WHERE request_id=?`, m.RequestID))
			if readErr == nil && prior.ActorID == m.ActorID && prior.TargetID == m.TargetID && prior.Subject == m.Subject && prior.ExpectedRevision == m.ExpectedRevision && prior.PayloadDigest == m.PayloadDigest {
				_ = tx.Rollback()
				node, getErr := s.Get(ctx, m.TargetID)
				if getErr != nil {
					return UpdateCASResult{}, getErr
				}
				return UpdateCASResult{Node: node, Receipt: prior}, nil
			}
			return UpdateCASResult{}, ErrRequestIdentityConflict{RequestID: m.RequestID}
		}
		return UpdateCASResult{}, err
	}
	caps, err := marshalStrings(m.Node.Capabilities)
	if err != nil {
		return UpdateCASResult{}, err
	}
	scopes, err := marshalStrings(m.Node.Scopes)
	if err != nil {
		return UpdateCASResult{}, err
	}
	res, err := tx.ExecContext(ctx, `UPDATE nodes SET name=?,kind=?,deployment_tier=?,endpoint=?,capabilities=?,scopes=?,revision=?,updated_at=?,owner_revision=owner_revision+1 WHERE id=? AND owner_revision=? AND revoked_at=''`, m.Node.Name, m.Node.Kind, m.Node.DeploymentTier, m.Node.Endpoint, caps, scopes, m.Node.Revision, now.Format(nodeTimeFormat), m.TargetID, m.ExpectedRevision)
	if err != nil {
		return UpdateCASResult{}, err
	}
	changed, err := res.RowsAffected()
	if err != nil {
		return UpdateCASResult{}, err
	}
	if changed == 0 {
		latest, getErr := scanNode(tx.QueryRowContext(ctx, selectNodeByIDSQL, m.TargetID))
		if getErr == sql.ErrNoRows {
			return UpdateCASResult{}, ErrNodeNotFound{ID: m.TargetID}
		}
		if getErr != nil {
			return UpdateCASResult{}, getErr
		}
		if !latest.RevokedAt.IsZero() {
			return UpdateCASResult{}, ErrNodeRevoked{ID: m.TargetID}
		}
		return UpdateCASResult{}, ErrRevisionConflict{ID: m.TargetID, Expected: m.ExpectedRevision, Current: latest.OwnerRevision}
	}
	newRevision := m.ExpectedRevision + 1
	if _, err = tx.ExecContext(ctx, `UPDATE node_update_receipts SET new_revision=? WHERE request_id=?`, newRevision, m.RequestID); err != nil {
		return UpdateCASResult{}, err
	}
	if err = tx.Commit(); err != nil {
		return UpdateCASResult{}, err
	}
	node, err := s.Get(ctx, m.TargetID)
	if err != nil {
		return UpdateCASResult{}, err
	}
	return UpdateCASResult{Node: node, Receipt: NodeUpdateReceipt{RequestID: m.RequestID, ActorID: m.ActorID, TargetID: m.TargetID, Subject: m.Subject, ExpectedRevision: m.ExpectedRevision, NewRevision: newRevision, PayloadDigest: m.PayloadDigest, CreatedAt: now}}, nil
}

func scanReceipt(s rowScanner) (NodeUpdateReceipt, error) {
	var r NodeUpdateReceipt
	var at string
	if err := s.Scan(&r.RequestID, &r.ActorID, &r.TargetID, &r.Subject, &r.ExpectedRevision, &r.NewRevision, &r.PayloadDigest, &at); err != nil {
		return r, err
	}
	t, err := time.Parse(nodeTimeFormat, at)
	r.CreatedAt = t
	return r, err
}

func (s *sqliteRepository) UpdateArchitecture(ctx context.Context, id, machineArch, binaryArch string) error {
	if err := s.ensureColumns(ctx); err != nil {
		return err
	}
	now := s.clock.Now().UTC().Format(nodeTimeFormat)
	res, err := s.db.ExecContext(ctx, updateArchitectureSQL, machineArch, machineArch, machineArch, binaryArch, now, id)
	if err != nil {
		return fmt.Errorf("update architecture for node %q: %w", id, err)
	}
	changed, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update architecture rows: %w", err)
	}
	if changed == 0 {
		return ErrNodeNotFound{ID: id}
	}
	return nil
}

func (s *sqliteRepository) SetEnrollmentGeneration(ctx context.Context, id string, generation int64) (Node, error) {
	if generation <= 0 {
		return Node{}, ErrInvalidNode{Field: "enrollment_generation", Reason: "must be positive"}
	}
	if err := s.ensureColumns(ctx); err != nil {
		return Node{}, err
	}
	result, err := s.db.ExecContext(ctx, `UPDATE nodes SET enrollment_generation=?,updated_at=? WHERE id=? AND enrollment_generation<?`, generation, s.clock.Now().UTC().Format(nodeTimeFormat), id, generation)
	if err != nil {
		return Node{}, err
	}
	if changed, _ := result.RowsAffected(); changed == 1 {
		return s.Get(ctx, id)
	}
	current, err := s.Get(ctx, id)
	if err != nil {
		return Node{}, err
	}
	if current.EnrollmentGeneration == generation {
		return current, nil
	}
	return Node{}, ErrEnrollmentGenerationConflict{ID: id, Current: current.EnrollmentGeneration, Attempted: generation}
}

func (s *sqliteRepository) Revoke(ctx context.Context, id string) (Node, error) {
	if err := s.ensureColumns(ctx); err != nil {
		return Node{}, err
	}
	existing, err := s.Get(ctx, id)
	if err != nil {
		return Node{}, err
	}
	if existing.Revoked() {
		// Idempotent: already revoked.
		return existing, nil
	}
	now := s.clock.Now().UTC()
	if _, err := s.db.ExecContext(ctx, revokeNodeSQL,
		now.Format(nodeTimeFormat), now.Format(nodeTimeFormat), id,
	); err != nil {
		return Node{}, fmt.Errorf("revoke node %q: %w", id, err)
	}
	existing.RevokedAt = now
	existing.UpdatedAt = now
	return existing, nil
}

func (s *sqliteRepository) Remove(ctx context.Context, id string) error {
	if err := s.ensureColumns(ctx); err != nil {
		return err
	}
	existing, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if !existing.Revoked() {
		return ErrNodeActive{ID: id}
	}
	if _, err := s.db.ExecContext(ctx, removeNodeSQL, id); err != nil {
		return fmt.Errorf("remove node %q: %w", id, err)
	}
	return nil
}

func (s *sqliteRepository) TouchLastSeen(ctx context.Context, id string, t time.Time) error {
	if err := s.ensureColumns(ctx); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, touchLastSeenSQL, t.UTC().Format(nodeTimeFormat), id); err != nil {
		return fmt.Errorf("touch last_seen for node %q: %w", id, err)
	}
	return nil
}

func (s *sqliteRepository) UpdateCapabilityInventory(ctx context.Context, id string, observations []CapabilityObservation, probedAt time.Time) error {
	if err := s.ensureColumns(ctx); err != nil {
		return err
	}
	inventory, err := marshalCapabilityInventory(observations)
	if err != nil {
		return fmt.Errorf("encode capability inventory: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `UPDATE nodes SET capability_inventory = ?, capability_probed_at = ?, updated_at = ? WHERE id = ?`, inventory, formatNullableTime(probedAt), s.clock.Now().UTC().Format(nodeTimeFormat), id)
	return err
}

// rowScanner unifies *sql.Row and *sql.Rows under their common Scan surface.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanNode(s rowScanner) (Node, error) {
	var (
		n                                                                                Node
		capsRaw                                                                          string
		scopesRaw                                                                        string
		createdRaw                                                                       string
		updatedRaw                                                                       string
		lastSeenRaw                                                                      string
		revokedRaw                                                                       string
		inventoryRaw                                                                     string
		probedRaw                                                                        string
		configurationOpID, configurationState, configurationAtRaw, configurationUnmetRaw string
	)
	if err := s.Scan(&n.ID, &n.Name, &n.Kind, &n.DeploymentTier, &n.EnrollmentGeneration, &n.OwnerRevision, &n.OS, &n.Arch, &n.MachineArch, &n.BinaryArch, &n.Revision, &n.Endpoint,
		&capsRaw, &scopesRaw, &n.PairingCorrelationID, &createdRaw, &updatedRaw, &lastSeenRaw, &revokedRaw, &inventoryRaw, &probedRaw, &configurationOpID, &configurationState, &configurationAtRaw, &configurationUnmetRaw); err != nil {
		return Node{}, err
	}

	caps, err := unmarshalStrings(capsRaw)
	if err != nil {
		return Node{}, fmt.Errorf("decode capabilities: %w", err)
	}
	scopes, err := unmarshalStrings(scopesRaw)
	if err != nil {
		return Node{}, fmt.Errorf("decode scopes: %w", err)
	}
	n.Capabilities = caps
	n.Scopes = scopes

	if n.CreatedAt, err = time.Parse(nodeTimeFormat, createdRaw); err != nil {
		return Node{}, fmt.Errorf("parse created_at %q: %w", createdRaw, err)
	}
	if n.UpdatedAt, err = time.Parse(nodeTimeFormat, updatedRaw); err != nil {
		return Node{}, fmt.Errorf("parse updated_at %q: %w", updatedRaw, err)
	}
	if n.LastSeenAt, err = parseNullableTime(lastSeenRaw); err != nil {
		return Node{}, fmt.Errorf("parse last_seen_at %q: %w", lastSeenRaw, err)
	}
	if n.RevokedAt, err = parseNullableTime(revokedRaw); err != nil {
		return Node{}, fmt.Errorf("parse revoked_at %q: %w", revokedRaw, err)
	}
	if inventoryRaw != "" {
		if err := json.Unmarshal([]byte(inventoryRaw), &n.CapabilityInventory); err != nil {
			return Node{}, fmt.Errorf("decode capability inventory: %w", err)
		}
	}
	if n.CapabilityProbedAt, err = parseNullableTime(probedRaw); err != nil {
		return Node{}, fmt.Errorf("parse capability_probed_at %q: %w", probedRaw, err)
	}
	n.ConfigurationOpID = configurationOpID
	n.ConfigurationState = configurationState
	if n.ConfigurationAt, err = parseNullableTime(configurationAtRaw); err != nil {
		return Node{}, fmt.Errorf("parse configuration_at %q: %w", configurationAtRaw, err)
	}
	if configurationUnmetRaw != "" && json.Unmarshal([]byte(configurationUnmetRaw), &n.ConfigurationUnmet) != nil {
		return Node{}, fmt.Errorf("decode configuration unmet items")
	}
	return n, nil
}

func (s *sqliteRepository) ensureColumns(ctx context.Context) error {
	s.migrateOnce.Do(func() {
		for _, stmt := range []string{
			`ALTER TABLE nodes ADD COLUMN kind TEXT NOT NULL DEFAULT 'agent'`,
			`ALTER TABLE nodes ADD COLUMN deployment_tier TEXT NOT NULL DEFAULT ''`,
			`ALTER TABLE nodes ADD COLUMN enrollment_generation INTEGER NOT NULL DEFAULT 0`,
			`ALTER TABLE nodes ADD COLUMN owner_revision INTEGER NOT NULL DEFAULT 1`,
			`ALTER TABLE nodes ADD COLUMN machine_arch TEXT NOT NULL DEFAULT ''`,
			`ALTER TABLE nodes ADD COLUMN binary_arch TEXT NOT NULL DEFAULT ''`,
			`ALTER TABLE nodes ADD COLUMN capability_inventory TEXT NOT NULL DEFAULT '[]'`,
			`ALTER TABLE nodes ADD COLUMN capability_probed_at TEXT NOT NULL DEFAULT ''`,
			`ALTER TABLE nodes ADD COLUMN configuration_op_id TEXT NOT NULL DEFAULT ''`,
			`ALTER TABLE nodes ADD COLUMN configuration_state TEXT NOT NULL DEFAULT ''`,
			`ALTER TABLE nodes ADD COLUMN configuration_at TEXT NOT NULL DEFAULT ''`,
			`ALTER TABLE nodes ADD COLUMN configuration_unmet TEXT NOT NULL DEFAULT '[]'`,
		} {
			_, s.migrateErr = s.db.ExecContext(ctx, stmt)
			if s.migrateErr != nil && !strings.Contains(s.migrateErr.Error(), "duplicate column") {
				return
			}
			s.migrateErr = nil
		}
		if s.migrateErr == nil {
			_, s.migrateErr = s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS node_update_receipts(request_id TEXT PRIMARY KEY,actor_id TEXT NOT NULL,target_id TEXT NOT NULL,subject TEXT NOT NULL,expected_revision INTEGER NOT NULL,new_revision INTEGER NOT NULL,payload_digest TEXT NOT NULL,created_at TEXT NOT NULL)`)
		}
	})
	return s.migrateErr
}

// marshalStrings encodes a string slice as a JSON array, normalising nil to
// "[]" so the column never holds NULL/"".
func marshalStrings(v []string) (string, error) {
	if v == nil {
		v = []string{}
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func configurationUnmetJSON(v []string) string {
	encoded, err := marshalStrings(v)
	if err != nil {
		return "[]"
	}
	return encoded
}

func marshalCapabilityInventory(v []CapabilityObservation) (string, error) {
	if v == nil {
		v = []CapabilityObservation{}
	}
	b, err := json.Marshal(v)
	return string(b), err
}

func unmarshalStrings(raw string) ([]string, error) {
	if raw == "" {
		return nil, nil
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

// formatNullableTime renders a zero time as "" (the column default) so absence
// is distinguishable from a real timestamp.
func formatNullableTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(nodeTimeFormat)
}

func parseNullableTime(raw string) (time.Time, error) {
	if raw == "" {
		return time.Time{}, nil
	}
	return time.Parse(nodeTimeFormat, raw)
}
