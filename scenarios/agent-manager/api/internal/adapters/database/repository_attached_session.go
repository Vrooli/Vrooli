// Responsibility: resolve a live attached harness session to the run that owns
// its physical executor, for quota readings that name the session but not the
// run.
package database

import (
	"context"
	"strings"
)

// AttachedSessionRunResolver returns the earliest live attached run of one
// harness session, or "" when none is live. Re-attaching a session creates a
// new run of the same conversation while the earlier run stays live; the
// physical inventory probes only the earliest one (maintenance
// reattachedSessionDuplicates), so quota readings name that same run and
// their exposure matches the run the samples observe. Only non-terminal rows
// are considered, which keeps the read on the status index.
func AttachedSessionRunResolver(db *DB) func(ctx context.Context, harnessKind, harnessSessionID string) string {
	return func(ctx context.Context, harnessKind, harnessSessionID string) string {
		harnessKind, harnessSessionID = strings.TrimSpace(harnessKind), strings.TrimSpace(harnessSessionID)
		if db == nil || harnessKind == "" || harnessSessionID == "" {
			return ""
		}
		var id string
		err := db.GetContext(ctx, &id, `SELECT id FROM runs
			WHERE status IN ('pending','starting','running') AND execution_mode = 'attached'
			  AND harness_kind = ? AND harness_session_id = ? AND COALESCE(runner_pid,0) > 1
			ORDER BY julianday(started_at), id LIMIT 1`, harnessKind, harnessSessionID)
		if err != nil {
			// No live run (sql.ErrNoRows) and a failed read both leave the
			// reading without a source run; it still counts toward account burn.
			return ""
		}
		return id
	}
}
