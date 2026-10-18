package conversations

import (
	"context"
	"database/sql"
	"fmt"
)

// AskIdentityDuplicateGroups counts all conflicting nonempty requester
// identities using the same (source, requester_ref) domain as the declared
// partial unique index. It does not return Ask rows or identity values.
//
// Notification Hub currently owns a SQLite database. Checking sqlite_master
// lets a first-run setup treat a database with no asks table as clear without
// creating or migrating anything; schema application at API startup remains
// the final authority.
func AskIdentityDuplicateGroups(ctx context.Context, db *sql.DB) (int64, error) {
	var tableExists bool
	if err := db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name='asks')`).Scan(&tableExists); err != nil {
		return 0, fmt.Errorf("inspect Ask table availability: %w", err)
	}
	if !tableExists {
		return 0, nil
	}

	var duplicateGroups int64
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM (
		SELECT 1 FROM asks
		WHERE requester_ref <> ''
		GROUP BY source, requester_ref
		HAVING COUNT(*) > 1
	)`).Scan(&duplicateGroups); err != nil {
		return 0, fmt.Errorf("count duplicate Ask requester identities: %w", err)
	}
	return duplicateGroups, nil
}
