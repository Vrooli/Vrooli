package scenarioruntime

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

const (
	EditorBindingActive          = "active"
	EditorBindingSuperseded      = "superseded"
	EditorBindingTerminal        = "terminal"
	EditorContainmentIntended    = "intended"
	EditorContainmentEstablished = "established"
	EditorContainmentFailed      = "failed"
	EditorContainmentUnavailable = "unavailable"
)

// EditorLeaseProvenance preserves one launch attempt independently of its session.
// Process identity and intent are immutable; terminal attempts retain evidence.
type EditorLeaseProvenance struct {
	BindingID, SessionID, State, ContainmentState, IntendedScope, Scope, Method, Failure string
	PID, UID                                                                             int
	HostBootID, ProcessStart                                                             string
	EvidencePID, EvidenceUID                                                             int
	EvidenceBootID, EvidenceProcessStart                                                 string
	CreatedAt, UpdatedAt                                                                 time.Time
	EndedAt                                                                              *time.Time
}

const editorLeaseProvenanceSchemaSQL = `
CREATE TABLE IF NOT EXISTS runtime_editor_lease_bindings (
 binding_id TEXT PRIMARY KEY, session_id TEXT NOT NULL, state TEXT NOT NULL,
 containment_state TEXT NOT NULL, intended_scope TEXT NOT NULL, scope TEXT NOT NULL,
 method TEXT NOT NULL, failure TEXT NOT NULL, pid INTEGER NOT NULL, uid INTEGER NOT NULL,
 host_boot_id TEXT NOT NULL, process_start TEXT NOT NULL,
 evidence_pid INTEGER NOT NULL, evidence_uid INTEGER NOT NULL,
 evidence_boot_id TEXT NOT NULL, evidence_process_start TEXT NOT NULL,
 created_at TEXT NOT NULL, updated_at TEXT NOT NULL, ended_at TEXT
);
CREATE INDEX IF NOT EXISTS idx_editor_bindings_session ON runtime_editor_lease_bindings(session_id, created_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_editor_binding_active ON runtime_editor_lease_bindings(session_id) WHERE state = 'active';
`

func addEditorLeaseProvenance(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `ALTER TABLE runtime_editor_leases ADD COLUMN binding_id TEXT NOT NULL DEFAULT ''`); err != nil && !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
		return err
	}
	_, err := db.ExecContext(ctx, editorLeaseProvenanceSchemaSQL)
	return err
}

func endEditorLeaseBindingTx(ctx context.Context, tx *sql.Tx, sessionID, state string, now time.Time) error {
	_, err := tx.ExecContext(ctx, `UPDATE runtime_editor_lease_bindings SET state = ?, updated_at = ?, ended_at = ? WHERE session_id = ? AND state = ?`, state, formatTime(now), formatTime(now), sessionID, EditorBindingActive)
	return err
}

// CreateEditorLeaseBinding atomically begins an attempt and supersedes its predecessor.
// Callers cannot manufacture success evidence at creation or reuse an attempt ID.
func (s *SQLiteStore) CreateEditorLeaseBinding(ctx context.Context, lease EditorLease, binding EditorLeaseProvenance, ttl time.Duration) (EditorLeaseProvenance, error) {
	lease.SessionID = strings.TrimSpace(lease.SessionID)
	if lease.SessionID == "" || strings.TrimSpace(binding.BindingID) == "" || (binding.SessionID != "" && binding.SessionID != lease.SessionID) || (lease.BindingID != "" && lease.BindingID != binding.BindingID) || binding.IntendedScope != lease.Scope || binding.PID <= 1 || binding.PID != lease.PID || strings.TrimSpace(binding.HostBootID) == "" || binding.HostBootID != lease.HostBootID || strings.TrimSpace(binding.ProcessStart) == "" || binding.UID < 0 {
		return EditorLeaseProvenance{}, fmt.Errorf("create editor binding: exact session and process identity required")
	}
	if (binding.State != "" && binding.State != EditorBindingActive) || (binding.ContainmentState != "" && binding.ContainmentState != EditorContainmentIntended) || binding.Scope != "" || binding.Method != "" || binding.Failure != "" || binding.EvidencePID != 0 || binding.EvidenceUID != 0 || binding.EvidenceBootID != "" || binding.EvidenceProcessStart != "" || binding.EndedAt != nil {
		return EditorLeaseProvenance{}, fmt.Errorf("create editor binding: only intended containment is permitted")
	}
	binding.SessionID = lease.SessionID
	binding.State = EditorBindingActive
	binding.ContainmentState = EditorContainmentIntended
	binding.CreatedAt = s.now()
	binding.UpdatedAt = binding.CreatedAt
	lease.BindingID = binding.BindingID
	lease.ContainmentMethod = ""
	err := s.withTx(ctx, func(tx *sql.Tx) error {
		if _, err := s.createEditorLeaseTx(ctx, tx, lease, ttl); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO runtime_editor_lease_bindings (binding_id,session_id,state,containment_state,intended_scope,scope,method,failure,pid,uid,host_boot_id,process_start,evidence_pid,evidence_uid,evidence_boot_id,evidence_process_start,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, binding.BindingID, binding.SessionID, binding.State, binding.ContainmentState, binding.IntendedScope, binding.Scope, binding.Method, binding.Failure, binding.PID, binding.UID, binding.HostBootID, binding.ProcessStart, binding.EvidencePID, binding.EvidenceUID, binding.EvidenceBootID, binding.EvidenceProcessStart, formatTime(binding.CreatedAt), formatTime(binding.UpdatedAt))
		return err
	})
	if err != nil {
		return EditorLeaseProvenance{}, err
	}
	return binding, nil
}

// RecordEditorLeaseContainment records exactly one outcome for the current attempt.
// Evidence identifies the actual contained process, which may be a child of the launcher.
func (s *SQLiteStore) RecordEditorLeaseContainment(ctx context.Context, sessionID, bindingID string, e EditorLeaseProvenance) (bool, error) {
	if strings.TrimSpace(sessionID) == "" || strings.TrimSpace(bindingID) == "" {
		return false, fmt.Errorf("record containment: session and binding required")
	}
	switch e.ContainmentState {
	case EditorContainmentEstablished:
		if strings.TrimSpace(e.Scope) == "" || strings.EqualFold(strings.TrimSpace(e.Scope), "none") || strings.TrimSpace(e.Method) == "" || strings.EqualFold(strings.TrimSpace(e.Method), "none") || e.EvidencePID <= 1 || e.EvidenceUID < 0 || strings.TrimSpace(e.EvidenceBootID) == "" || strings.TrimSpace(e.EvidenceProcessStart) == "" || e.Failure != "" {
			return false, fmt.Errorf("record containment: exact success evidence required")
		}
	case EditorContainmentFailed, EditorContainmentUnavailable:
		if e.Scope != "" || e.Method != "" || e.EvidencePID != 0 || e.EvidenceUID != 0 || e.EvidenceBootID != "" || e.EvidenceProcessStart != "" {
			return false, fmt.Errorf("record containment: unsuccessful outcome cannot carry success evidence")
		}
	default:
		return false, fmt.Errorf("record containment: terminal containment outcome required")
	}
	changed := false
	err := s.withTx(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `UPDATE runtime_editor_lease_bindings SET containment_state=?,scope=?,method=?,failure=?,evidence_pid=?,evidence_uid=?,evidence_boot_id=?,evidence_process_start=?,updated_at=? WHERE binding_id=? AND session_id=? AND state=? AND containment_state=? AND (? <> ? OR (host_boot_id=? AND uid=?)) AND EXISTS (SELECT 1 FROM runtime_editor_leases l WHERE l.session_id=? AND l.binding_id=? AND l.status=?)`, e.ContainmentState, e.Scope, e.Method, e.Failure, e.EvidencePID, e.EvidenceUID, e.EvidenceBootID, e.EvidenceProcessStart, formatTime(s.now()), bindingID, sessionID, EditorBindingActive, EditorContainmentIntended, e.ContainmentState, EditorContainmentEstablished, e.EvidenceBootID, e.EvidenceUID, sessionID, bindingID, EditorLeaseActive)
		if err != nil {
			return err
		}
		n, err := result.RowsAffected()
		if err != nil || n == 0 {
			return err
		}
		_, err = tx.ExecContext(ctx, `UPDATE runtime_editor_leases SET containment_method=? WHERE session_id=? AND binding_id=? AND status=?`, e.Method, sessionID, bindingID, EditorLeaseActive)
		changed = err == nil
		return err
	})
	return changed && err == nil, err
}

const editorProvenanceSelectSQL = `SELECT binding_id,session_id,state,containment_state,intended_scope,scope,method,failure,pid,uid,host_boot_id,process_start,evidence_pid,evidence_uid,evidence_boot_id,evidence_process_start,created_at,updated_at,ended_at FROM runtime_editor_lease_bindings`

func scanEditorProvenance(row editorLeaseScanner) (EditorLeaseProvenance, error) {
	var b EditorLeaseProvenance
	var created, updated string
	var ended sql.NullString
	if err := row.Scan(&b.BindingID, &b.SessionID, &b.State, &b.ContainmentState, &b.IntendedScope, &b.Scope, &b.Method, &b.Failure, &b.PID, &b.UID, &b.HostBootID, &b.ProcessStart, &b.EvidencePID, &b.EvidenceUID, &b.EvidenceBootID, &b.EvidenceProcessStart, &created, &updated, &ended); err != nil {
		return b, err
	}
	var err error
	if b.CreatedAt, err = parseRequiredTime(created); err != nil {
		return b, err
	}
	if b.UpdatedAt, err = parseRequiredTime(updated); err != nil {
		return b, err
	}
	b.EndedAt, err = parseOptionalTime(ended)
	return b, err
}
func (s *SQLiteStore) GetEditorLeaseProvenance(ctx context.Context, bindingID string) (EditorLeaseProvenance, error) {
	b, err := scanEditorProvenance(s.db.QueryRowContext(ctx, editorProvenanceSelectSQL+` WHERE binding_id=?`, bindingID))
	if err == sql.ErrNoRows {
		err = ErrNotFound
	}
	return b, err
}
func (s *SQLiteStore) ListEditorLeaseProvenance(ctx context.Context, sessionID string) ([]EditorLeaseProvenance, error) {
	rows, err := s.db.QueryContext(ctx, editorProvenanceSelectSQL+` WHERE session_id=? ORDER BY created_at,binding_id`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []EditorLeaseProvenance{}
	for rows.Next() {
		b, err := scanEditorProvenance(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}
func (s *SQLiteStore) HeartbeatEditorLeaseBinding(ctx context.Context, sessionID, bindingID string, ttl time.Duration) (bool, error) {
	if strings.TrimSpace(bindingID) == "" || strings.TrimSpace(sessionID) == "" {
		return false, fmt.Errorf("heartbeat binding: session and binding required")
	}
	now := s.now()
	result, err := s.db.ExecContext(ctx, `UPDATE runtime_editor_leases SET last_heartbeat_at=?,heartbeat_deadline_at=? WHERE session_id=? AND binding_id=? AND status=? AND EXISTS (SELECT 1 FROM runtime_editor_lease_bindings b WHERE b.binding_id=? AND b.session_id=? AND b.state=?)`, formatTime(now), formatTime(now.Add(NormalizeHeartbeatTTL(ttl))), sessionID, bindingID, EditorLeaseActive, bindingID, sessionID, EditorBindingActive)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n > 0, err
}
func (s *SQLiteStore) StopEditorLeaseBinding(ctx context.Context, sessionID, bindingID, reason string) (bool, error) {
	if strings.TrimSpace(bindingID) == "" || strings.TrimSpace(sessionID) == "" {
		return false, fmt.Errorf("stop binding: session and binding required")
	}
	changed := false
	now := s.now()
	err := s.withTx(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `UPDATE runtime_editor_leases SET status=?,stopped_at=?,stop_reason=? WHERE session_id=? AND binding_id=? AND status=? AND EXISTS (SELECT 1 FROM runtime_editor_lease_bindings b WHERE b.binding_id=? AND b.session_id=? AND b.state=?)`, EditorLeaseStopped, formatTime(now), strings.TrimSpace(reason), sessionID, bindingID, EditorLeaseActive, bindingID, sessionID, EditorBindingActive)
		if err != nil {
			return err
		}
		n, err := result.RowsAffected()
		if err != nil || n == 0 {
			return err
		}
		if err = endEditorLeaseBindingTx(ctx, tx, sessionID, EditorBindingTerminal, now); err != nil {
			return err
		}
		changed = true
		return nil
	})
	return changed && err == nil, err
}
