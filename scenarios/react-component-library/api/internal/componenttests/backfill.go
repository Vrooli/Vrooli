package componenttests

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
)

// BackfillRollups folds existing report payloads into the compact version
// rollup and then applies the same retention policy used by new writes.
// It is intentionally an explicit one-shot operation, not a startup side
// effect, so operators can verify the before/after counts around it.
func BackfillRollups(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	rows, err := tx.QueryContext(ctx, `SELECT id, component_id, root_library_id, root_version, created_at, verdict FROM component_test_reports ORDER BY created_at ASC, id ASC`)
	if err != nil {
		return err
	}
	type key struct{ component, library, version string }
	keys := map[key]struct{}{}
	for rows.Next() {
		var id, component, library, version, created, verdict string
		if err := rows.Scan(&id, &component, &library, &version, &created, &verdict); err != nil {
			rows.Close()
			return err
		}
		passed, failed, blocked := 0, 0, 0
		switch Verdict(verdict) {
		case VerdictPassed:
			passed = 1
		case VerdictFailed:
			failed = 1
		case VerdictBlocked:
			blocked = 1
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO component_version_test_rollup (library_id, version, runs_total, runs_passed, runs_failed, runs_blocked, first_pass_report_id, first_fail_report_id, last_run_at)
VALUES (?, ?, 1, ?, ?, ?, CASE WHEN ? = 1 THEN ? ELSE '' END, CASE WHEN ? = 1 THEN ? ELSE '' END, ?)
ON CONFLICT(library_id, version) DO UPDATE SET
 runs_total=runs_total+1, runs_passed=runs_passed+excluded.runs_passed, runs_failed=runs_failed+excluded.runs_failed, runs_blocked=runs_blocked+excluded.runs_blocked,
 first_pass_report_id=CASE WHEN component_version_test_rollup.first_pass_report_id='' AND excluded.first_pass_report_id<>'' THEN excluded.first_pass_report_id ELSE component_version_test_rollup.first_pass_report_id END,
 first_fail_report_id=CASE WHEN component_version_test_rollup.first_fail_report_id='' AND excluded.first_fail_report_id<>'' THEN excluded.first_fail_report_id ELSE component_version_test_rollup.first_fail_report_id END,
 last_run_at=CASE WHEN excluded.last_run_at>component_version_test_rollup.last_run_at THEN excluded.last_run_at ELSE component_version_test_rollup.last_run_at END`, library, version, passed, failed, blocked, passed, id, failed, id, created); err != nil {
			rows.Close()
			return err
		}
		keys[key{component, library, version}] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	ordered := make([]key, 0, len(keys))
	for k := range keys {
		ordered = append(ordered, k)
	}
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].component != ordered[j].component {
			return ordered[i].component < ordered[j].component
		}
		return ordered[i].version < ordered[j].version
	})
	for _, k := range ordered {
		if err := retainVersionReports(ctx, tx, k.component, k.library, k.version); err != nil {
			return fmt.Errorf("retain %s@%s: %w", k.library, k.version, err)
		}
	}
	return tx.Commit()
}
