package store

import (
	"database/sql"
	"fmt"
	"time"
)

// purgeByTimestamp deletes rows older than the retention period from the given table.
// timestampCol is the column name used for the cutoff comparison.
func purgeByTimestamp(db *sql.DB, table, timestampCol string, retention time.Duration) (int64, error) {
	cutoff := time.Now().Add(-retention)
	result, err := db.Exec(
		fmt.Sprintf(`DELETE FROM %s WHERE %s < $1`, table, timestampCol),
		cutoff,
	)
	if err != nil {
		return 0, fmt.Errorf("purge %s: %w", table, err)
	}
	return result.RowsAffected()
}
