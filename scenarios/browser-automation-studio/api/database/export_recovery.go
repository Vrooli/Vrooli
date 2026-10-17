package database

import (
	"context"
	"fmt"
)

// RecoverInterruptedExports runs once at the process boundary, before serving.
// No encoding is replayed. Complete files/recovery sidecars remain on disk.
func RecoverInterruptedExports(ctx context.Context, db *DB) (int64, error) {
	result, err := db.ExecContext(ctx, "UPDATE exports SET status = 'failed', error = 'Export interrupted by API restart; retained output recovery receipts may be inspected', updated_at = CURRENT_TIMESTAMP WHERE status IN ('pending', 'processing')")
	if err != nil {
		return 0, fmt.Errorf("recover interrupted exports: %w", err)
	}
	return result.RowsAffected()
}
