package scenarioruntime

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// MaxEditorLeaseBindingKeys bounds a maintenance inventory to the canonical and
// legacy keys of at most 65536 attached runs, before duplicate keys are removed.
const MaxEditorLeaseBindingKeys = 2 * 65536

const editorLeaseBindingChunkSize = 512

const editorLeaseBindingSelectSQL = `
SELECT session_id, binding_id, harness, agent, pid, host_boot_id, scope, status, created_at,
  owner_kind, owner_ref, owner_evidence_json, owner_source
FROM runtime_editor_leases`

// GetEditorLeaseBindings reads only the exact requested keys, across all lease
// statuses. A stopped or expired canonical lease must remain present so callers
// cannot mistake it for absence and borrow a legacy run-key lease.
//
// The result contains only fields compared by EditorLeaseBindingEqual. Missing
// keys are absent, duplicate keys produce one row, and keys are never normalized.
// Reads are bounded to MaxEditorLeaseBindingKeys inputs and 512 keys per query.
// Each chunk is fully read and closed before the next query. Callers must
// revalidate bindings before acting; this is not an atomic multi-chunk snapshot.
func (s *SQLiteStore) GetEditorLeaseBindings(ctx context.Context, sessionIDs []string) ([]EditorLease, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(sessionIDs) > MaxEditorLeaseBindingKeys {
		return nil, fmt.Errorf("get editor lease bindings: at most %d keys are permitted", MaxEditorLeaseBindingKeys)
	}
	keys := make([]string, 0, len(sessionIDs))
	seen := make(map[string]struct{}, len(sessionIDs))
	for _, id := range sessionIDs {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if strings.TrimSpace(id) == "" {
			return nil, fmt.Errorf("get editor lease bindings: session_id is required")
		}
		if _, exists := seen[id]; !exists {
			seen[id] = struct{}{}
			keys = append(keys, id)
		}
	}
	out := make([]EditorLease, 0)
	for _, chunk := range chunkIDs(keys, editorLeaseBindingChunkSize) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		query := editorLeaseBindingSelectSQL + ` WHERE session_id IN (` + placeholders(len(chunk)) + `)`
		rows, err := s.db.QueryContext(ctx, query, idArgs(chunk)...)
		if err != nil {
			return nil, fmt.Errorf("get editor lease bindings: %w", err)
		}
		bindings, scanErr := scanEditorLeaseBindings(ctx, rows)
		closeErr := rows.Close()
		if scanErr != nil {
			return nil, fmt.Errorf("scan editor lease bindings: %w", scanErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("close editor lease bindings: %w", closeErr)
		}
		out = append(out, bindings...)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func scanEditorLeaseBindings(ctx context.Context, rows *sql.Rows) ([]EditorLease, error) {
	var bindings []EditorLease
	for rows.Next() {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		var lease EditorLease
		var pid sql.NullInt64
		var createdAt string
		if err := rows.Scan(&lease.SessionID, &lease.BindingID, &lease.Harness, &lease.Agent, &pid,
			&lease.HostBootID, &lease.Scope, &lease.Status, &createdAt,
			&lease.Owner.Kind, &lease.Owner.Ref, &lease.Owner.EvidenceJSON, &lease.Owner.Source); err != nil {
			return nil, err
		}
		if pid.Valid {
			lease.PID = int(pid.Int64)
		}
		var err error
		if lease.CreatedAt, err = parseRequiredTime(createdAt); err != nil {
			return nil, err
		}
		bindings = append(bindings, lease)
	}
	return bindings, rows.Err()
}
